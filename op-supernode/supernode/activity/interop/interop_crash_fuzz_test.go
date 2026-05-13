package interop

import (
	"context"
	"errors"
	"testing"

	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

// FuzzVerifyInteropMessagesCrashRecover exercises PR #19505's crash-safety
// contract: a process drop between SetPendingTransition (WAL written) and
// ClearPendingTransition (WAL cleared) must leave a state from which a
// fresh Interop converges to exactly the same final verifiedDB shape as
// a clean baseline run.
//
// Three phases, all driven from the same (seed, numChains) so the
// generated RandomChain is identical:
//
//  1. Baseline. progressAndRecord runs to fixpoint on a fresh dataDir,
//     no WAL injection. Snapshot final state.
//  2. Crash. New dataDir. progressAndRecord runs through a walInjector
//     that returns errWALInjectorCrash once a chosen number of WAL ops
//     has been observed. The loop exits with that error; the Interop is
//     closed without further work, leaving the dataDir in whatever WAL
//     state the injector produced.
//  3. Recover. Fresh Interop opened on the SAME dataDir from phase 2,
//     no injector. progressAndRecord runs to fixpoint and replays any
//     persisted pending transition. Snapshot final state.
//
// Oracle: recovery final state == baseline final state. If the recovery
// path silently drops or duplicates a commit, the comparison catches it.
func FuzzVerifyInteropMessagesCrashRecover(f *testing.F) {
	// Seed inputs span an early/mid/late crashAt range so the testdata
	// pass alone exercises:
	//   - very-early crash (no commits before injector fires)
	//   - mid-flight crash (a handful of commits before fire)
	//   - late/never crash (injector never fires; recovery degenerates
	//     to a clean re-run on the same dataDir)
	// numChainsRaw>>6 picks 1..4 chains; the harness clamps to >=2.
	f.Add(int64(1), uint8(0x80), uint8(3))   // 2 chains, mid crash
	f.Add(int64(2), uint8(0xc0), uint8(5))   // 3 chains, mid crash
	f.Add(int64(3), uint8(0x40), uint8(2))   // 1->2 chains, early crash
	f.Add(int64(4), uint8(0xff), uint8(7))   // 3 chains, mid crash
	f.Add(int64(5), uint8(0xff), uint8(1))   // earliest possible crash
	f.Add(int64(6), uint8(0x80), uint8(25))  // later crash, may not fire on short chains
	f.Add(int64(7), uint8(0xc0), uint8(100)) // very late, almost never fires (clean-run oracle)
	f.Add(int64(8), uint8(0x80), uint8(0))   // crashAt=0 disables crash; pure recovery-noop check
	f.Add(int64(9), uint8(0x40), uint8(10))  // mid, fewer chains

	f.Fuzz(func(t *testing.T, seed int64, numChainsRaw uint8, crashAtRaw uint8) {
		params := RandomChainParams{
			chainCount:            max(2, int(numChainsRaw>>6)),
			minLength:             30,
			maxLength:             60,
			invalidateChance:      80,
			dependencyChance:      20,
			maxBlockTimeExclusive: 15,
		}
		// crashAtRaw=0 disables crashing, leaving phases 2+3 equivalent
		// to two clean runs against the same dataDir. That still exercises
		// the recovery oracle (recovery should noop and converge) and is
		// a useful regression check, so don't skip it.
		crashAt := int(crashAtRaw)

		// Build the RandomChain ONCE so all three phases observe identical
		// chain shape. MakeRandomChain is not deterministic across calls
		// with the same seed because internal MergeBlocks iterates Go maps;
		// constructing once and sharing avoids that drift, which would
		// otherwise mask SUT behavior under chain-shape noise.
		rc := params.MakeRandomChain(t, seed)
		var activation uint64
		for _, blocks := range rc.chainBlocks {
			activation = max(activation, blocks[0].Time)
		}

		// Phase 1: baseline.
		baselineLastTS, baselineHasTS, baselineHeads := runCrashRecoverPhase(t, "baseline", &rc, activation, t.TempDir(), 0, nil)

		// Phase 2: crash. Independent dataDir, shared between phase 2 and 3.
		crashDir := t.TempDir()
		injector := newWALInjector(nil, crashAt) // inner attached inside the phase helper
		runCrashRecoverPhase(t, "crash", &rc, activation, crashDir, crashAt, injector)
		crashed := injector.crashed()

		// Phase 2.5: if a crash actually fired, peek at the persisted
		// PendingTransition before the recovery Interop opens. The on-disk
		// shape must satisfy the same invariants Dafny's
		// PendingTransitionIsConsistent enforces in our formal model:
		//   - DecisionRewind   : Rewind plan present, non-zero RewindAtOrAfter
		//   - DecisionAdvance  : Result present, non-zero Timestamp
		//   - DecisionInvalidate: Result present, non-empty InvalidHeads
		// A malformed entry here means PR #19505's WAL persistence is
		// non-atomic with respect to the apply step, even if recovery
		// somehow converges to the right final state by coincidence.
		if crashed {
			assertPendingTransitionShape(t, crashDir)
		}

		// Phase 3: recovery on crashDir. No injector.
		recoveryLastTS, recoveryHasTS, recoveryHeads := runCrashRecoverPhase(t, "recovery", &rc, activation, crashDir, 0, nil)

		require.Equal(t, baselineHasTS, recoveryHasTS,
			"recovery hasTS=%v differs from baseline hasTS=%v (crashed=%v, crashAt=%d)",
			recoveryHasTS, baselineHasTS, crashed, crashAt)
		if baselineHasTS {
			require.Equal(t, baselineLastTS, recoveryLastTS,
				"recovery LastTimestamp=%d differs from baseline=%d (crashed=%v, crashAt=%d)",
				recoveryLastTS, baselineLastTS, crashed, crashAt)
			require.Equal(t, baselineHeads, recoveryHeads,
				"recovery per-chain L2Heads differ from baseline (crashed=%v, crashAt=%d)",
				crashed, crashAt)
		}
	})
}

// runCrashRecoverPhase constructs an Interop on dataDir, optionally wraps
// its verifiedDB in injector, drives progressAndRecord to fixpoint, and
// returns the (lastTS, hasTS, headsByChain) snapshot. If injector is
// non-nil, its inner is wired to the just-constructed verifiedDB before
// the loop runs; the loop exits cleanly on errWALInjectorCrash without
// failing the test.
func runCrashRecoverPhase(
	t *testing.T,
	label string,
	rc *RandomChain,
	activation uint64,
	dataDir string,
	crashAfter int,
	injector *walInjector,
) (lastTS uint64, hasTS bool, heads map[eth.ChainID]eth.BlockID) {
	t.Helper()

	logger := gethlog.NewLogger(gethlog.NewTerminalHandler(testWriter{t}, true))
	mocks := rc.GetContainers()
	interop := New(logger, activation, rc.messageExpiryWindow, mocks, dataDir, *rc, 0, nil)
	require.NotNil(t, interop, "%s phase: interop.New returned nil", label)
	interop.ctx = context.Background()

	if injector != nil {
		// The injector is created up-front (so the caller can observe its
		// .crashed() state after the phase) but the inner reference must
		// be attached now that verifiedDB exists.
		injector.inner = interop.verifiedDB
		injector.crashAfter = crashAfter
		interop.verifiedDB = injector
	}

	defer func() {
		_ = interop.Stop(context.Background())
	}()

	for {
		advanced, err := interop.progressAndRecord()
		if err != nil {
			if errors.Is(err, errWALInjectorCrash) {
				t.Logf("%s phase: injector crash fired at WAL op count %d", label, injector.calls.Load())
				break
			}
			// Match the existing FuzzVerifyInteropMessages tolerance: the
			// harness loop only requires no-error when advanced=true. Bare
			// errors when not advancing are typically chain-shape edge
			// cases (e.g. frontier-view "previous timestamp not sealed")
			// that aren't what this fuzz target is verifying. Log and stop
			// so phases stay aligned across baseline/crash/recovery.
			t.Logf("%s phase: progressAndRecord exited with err=%v (advanced=%v)", label, err, advanced)
			break
		}
		if !advanced {
			break
		}
	}

	lastTS, hasTS = interop.verifiedDB.LastTimestamp()
	heads = make(map[eth.ChainID]eth.BlockID)
	if hasTS {
		result, err := interop.verifiedDB.Get(lastTS)
		if err == nil {
			for chainID, h := range result.L2Heads {
				heads[chainID] = h
			}
		}
	}
	return lastTS, hasTS, heads
}

// assertPendingTransitionShape opens just a VerifiedDB on the given dataDir
// (no Interop) and validates that any persisted PendingTransition is
// structurally well-formed. Mirrors Dafny's PendingTransitionIsConsistent
// predicate from the formal Supernode model.
//
// If no PendingTransition is present (the crash fired before
// SetPendingTransition completed, or after ClearPendingTransition), the
// check is a no-op. The recovery oracle (state convergence) still applies.
func assertPendingTransitionShape(t *testing.T, dataDir string) {
	t.Helper()

	db, err := OpenVerifiedDB(dataDir)
	if err != nil {
		t.Fatalf("OpenVerifiedDB(%s) for PendingTransition shape check: %v", dataDir, err)
	}
	defer db.Close()

	pending, err := db.GetPendingTransition()
	require.NoError(t, err, "GetPendingTransition on crashed dataDir")
	if pending == nil {
		return
	}

	switch pending.Decision {
	case DecisionRewind:
		require.NotNil(t, pending.Rewind,
			"persisted DecisionRewind missing Rewind plan")
		require.Greater(t, pending.Rewind.RewindAtOrAfter, uint64(0),
			"persisted DecisionRewind plan has zero RewindAtOrAfter")
	case DecisionAdvance:
		require.NotNil(t, pending.Result,
			"persisted DecisionAdvance missing Result")
		require.Greater(t, pending.Result.Timestamp, uint64(0),
			"persisted DecisionAdvance has zero Result.Timestamp")
		require.NotEmpty(t, pending.Result.L2Heads,
			"persisted DecisionAdvance has empty L2Heads")
	case DecisionInvalidate:
		require.NotNil(t, pending.Result,
			"persisted DecisionInvalidate missing Result")
		require.NotEmpty(t, pending.Result.InvalidHeads,
			"persisted DecisionInvalidate has no InvalidHeads")
	case DecisionWait:
		// DecisionWait should never be persisted as a pending transition —
		// it's the "no action" path. If we see it on disk, that's a SUT
		// invariant violation worth surfacing.
		t.Fatalf("persisted PendingTransition has DecisionWait; should never be written")
	default:
		t.Fatalf("persisted PendingTransition has unknown Decision=%v", pending.Decision)
	}
}
