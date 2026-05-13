package interop

import (
	"sort"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

// assertExpectedResult verifies that result.InvalidHeads matches the prediction
// the RandomChain made when its invalidation was injected. It is one-shot
// against the result of a single verifyInteropMessages call at the timestamp
// where the harness's progressAndRecord loop stopped.
//
// Contract enforced (only when err == nil; a non-nil err means the SUT
// bailed before reaching a verdict — common at the harness's safeTimestamp
// boundary when a chain's logsDB hasn't been sealed up to that height yet):
//
//   - For an unmodified chain (KindNone): result.InvalidHeads is empty.
//   - For any invalidation kind, when tsVerified >= invalidationTimestamp:
//     at least one chain must be in InvalidHeads, the set of predicted
//     invalid chains is a subset of the actual set (catches
//     under-invalidation), and the actual set is a subset of the predicted
//     set (catches over-invalidation — e.g. bystander chains incorrectly
//     swept into cycle detection).
//   - When tsVerified < invalidationTimestamp the loop did not reach the
//     injected bug; the assertion is a no-op.
//
// The bidirectional set check is the real win over the old
// `err != nil || !result.IsValid()` assertion: a regression that marks the
// wrong chain invalid would silently pass before and fails now.
func (rc *RandomChain) assertExpectedResult(t *testing.T, tsVerified uint64, result Result, err error) {
	t.Helper()

	if err != nil {
		// The SUT was unable to fully process this timestamp (typically
		// ErrFuture from logsDB.OpenBlock when the harness's safeTimestamp
		// outruns the per-chain seal frontier). No claim is possible about
		// InvalidHeads; preserve the previous harness's permissive behavior.
		return
	}

	if rc.invalidationKind == KindNone {
		require.Empty(t, result.InvalidHeads,
			"kind=None: expected no invalid heads, got %v", sortedChainIDs(result.InvalidHeads))
		require.True(t, result.IsValid(), "kind=None: result.IsValid() must be true")
		return
	}

	// The injection sits at rc.invalidationTimestamp. If verification ran on
	// a timestamp strictly before that, the SUT had no chance to observe the
	// bug — silently accept.
	if tsVerified < rc.invalidationTimestamp {
		return
	}

	require.NotEmpty(t, result.InvalidHeads,
		"kind=%s @ ts=%d (verified=%d): expected at least one InvalidHead but result is valid (predicted=%v, heads=%v)",
		rc.invalidationKind, rc.invalidationTimestamp, tsVerified,
		sortedChainIDSet(rc.expectedInvalidChains),
		sortedChainIDs(result.L2Heads))

	// Predicted ⊆ Actual: every predicted invalid chain that has a block at
	// this timestamp must be reported invalid.
	for chainID := range rc.expectedInvalidChains {
		if _, hasBlock := result.L2Heads[chainID]; !hasBlock {
			continue
		}
		_, marked := result.InvalidHeads[chainID]
		require.True(t, marked,
			"kind=%s @ ts=%d: predicted chain %s should be in InvalidHeads, got %v",
			rc.invalidationKind, rc.invalidationTimestamp, chainID,
			sortedChainIDs(result.InvalidHeads))
	}

	// Actual ⊆ Predicted: any chain the SUT reports invalid must have been
	// predicted. This is where over-invalidation regressions get caught.
	for chainID := range result.InvalidHeads {
		require.True(t, rc.expectedInvalidChains[chainID],
			"kind=%s @ ts=%d: SUT marked chain %s invalid but harness did not predict it (predicted=%v, actual=%v)",
			rc.invalidationKind, rc.invalidationTimestamp, chainID,
			sortedChainIDSet(rc.expectedInvalidChains),
			sortedChainIDs(result.InvalidHeads))
	}
}

// assertProgressStoppedBeforeBug catches the highest-value regression class
// the new harness can detect from existing fuzz behavior: the SUT must
// never commit a verified result at or beyond the timestamp where the
// invalidation lives. If it did, the bug was missed — either by message
// validation or by cycle detection — and the verified frontier is wrong.
//
// This is one-sided (it doesn't prove the SUT detected the bug, only that
// it didn't blindly commit past it), but it works from the verifiedDB
// state directly, bypassing the ErrFuture / OpenBlock issues that prevent
// the re-verify path in assertExpectedResult from firing often.
func (rc *RandomChain) assertProgressStoppedBeforeBug(t *testing.T, interop *Interop) {
	t.Helper()
	if rc.invalidationKind == KindNone {
		return
	}
	lastTS, ok := interop.verifiedDB.LastTimestamp()
	if !ok {
		return
	}
	require.Less(t, lastTS, rc.invalidationTimestamp,
		"kind=%s: SUT committed verified result at ts=%d but invalidation @ ts=%d should have blocked it (predicted invalid chains: %v)",
		rc.invalidationKind, lastTS, rc.invalidationTimestamp,
		sortedChainIDSet(rc.expectedInvalidChains))
}

// assertVerifiedDBReadback probes the public read API after the
// progressAndRecord loop terminates and confirms it agrees with the
// committed VerifiedDB state. Currently covers VerifiedAtTimestamp, Has,
// LatestVerifiedL2Block, and VerifiedBlockAtL1.
func assertVerifiedDBReadback(t *testing.T, interop *Interop) {
	t.Helper()

	lastTS, ok := interop.verifiedDB.LastTimestamp()
	if !ok {
		// No timestamps were ever committed — nothing to read back, but
		// VerifiedAtTimestamp should still behave sanely on a pre-activation ts.
		if interop.activationTimestamp > 0 {
			pre := interop.activationTimestamp - 1
			verified, err := interop.VerifiedAtTimestamp(pre)
			require.NoError(t, err, "VerifiedAtTimestamp(%d) (pre-activation)", pre)
			require.True(t, verified, "VerifiedAtTimestamp(%d) (pre-activation) should be true", pre)
		}
		return
	}

	// Every committed timestamp should report as verified, and Has should agree.
	for ts := interop.activationTimestamp; ts <= lastTS; ts++ {
		verified, err := interop.VerifiedAtTimestamp(ts)
		require.NoError(t, err, "VerifiedAtTimestamp(%d)", ts)
		require.True(t, verified, "VerifiedAtTimestamp(%d) should be true (lastTS=%d)", ts, lastTS)

		has, err := interop.verifiedDB.Has(ts)
		require.NoError(t, err, "verifiedDB.Has(%d)", ts)
		require.True(t, has, "verifiedDB.Has(%d) should be true (lastTS=%d)", ts, lastTS)
	}

	// Pre-activation timestamps are considered verified by contract
	// (interop.go:719-725).
	if interop.activationTimestamp > 0 {
		pre := interop.activationTimestamp - 1
		verified, err := interop.VerifiedAtTimestamp(pre)
		require.NoError(t, err, "VerifiedAtTimestamp(%d) (pre-activation)", pre)
		require.True(t, verified, "VerifiedAtTimestamp(pre-activation=%d) should be true", pre)
	}

	// LatestVerifiedL2Block per chain should agree with the last committed
	// VerifiedResult.
	lastResult, err := interop.verifiedDB.Get(lastTS)
	require.NoError(t, err, "verifiedDB.Get(lastTS=%d)", lastTS)

	for chainID, expectedHead := range lastResult.L2Heads {
		gotHead, gotTS := interop.LatestVerifiedL2Block(chainID)
		require.Equal(t, expectedHead, gotHead,
			"LatestVerifiedL2Block(%s).head mismatch (lastTS=%d)", chainID, lastTS)
		require.Equal(t, lastTS, gotTS,
			"LatestVerifiedL2Block(%s).ts mismatch", chainID)
	}

	// VerifiedBlockAtL1 with the recorded L1Inclusion should resolve to the
	// last committed L2 heads. Skip when L1Inclusion is zero — that path is
	// already covered by the early-return in interop.go:751-754.
	if lastResult.L1Inclusion != (eth.BlockID{}) {
		l1Ref := eth.L1BlockRef{
			Hash:   lastResult.L1Inclusion.Hash,
			Number: lastResult.L1Inclusion.Number,
		}
		for chainID, expectedHead := range lastResult.L2Heads {
			gotHead, gotTS := interop.VerifiedBlockAtL1(chainID, l1Ref)
			require.Equal(t, expectedHead, gotHead,
				"VerifiedBlockAtL1(%s, L1=%d) mismatch", chainID, l1Ref.Number)
			require.Equal(t, lastTS, gotTS,
				"VerifiedBlockAtL1(%s).ts mismatch", chainID)
		}
	}
}

// sortedChainIDs renders a map keyed by ChainID for stable test output.
func sortedChainIDs(m map[eth.ChainID]eth.BlockID) []string {
	out := make([]string, 0, len(m))
	for id := range m {
		out = append(out, id.String())
	}
	sort.Strings(out)
	return out
}

// sortedChainIDSet renders a set of ChainID for stable test output.
func sortedChainIDSet(m map[eth.ChainID]bool) []string {
	out := make([]string, 0, len(m))
	for id, v := range m {
		if v {
			out = append(out, id.String())
		}
	}
	sort.Strings(out)
	return out
}
