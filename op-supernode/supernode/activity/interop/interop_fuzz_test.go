package interop

import (
	"context"
	"testing"

	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/stretchr/testify/require"
	"maps"
)

func FuzzVerifyInteropMessages(f *testing.F) {
	// In-code seed corpus. testdata/fuzz/FuzzVerifyInteropMessages/ is
	// gitignored at the repo root, so a fresh checkout has no on-disk
	// seeds. These f.Add inputs give every checkout a deterministic
	// starting corpus that exercises a spread of the six invalidation
	// kinds (None, Cycle, SelfDependency, InvalidIdentifier,
	// FutureDependency, ExpiredMessage, L1Reorg). The numChainsRaw>>6
	// dance picks 1..4 chains; the harness clamps to >=2.
	f.Add(int64(0), uint8(0x80))
	f.Add(int64(1), uint8(0xff))
	f.Add(int64(2), uint8(0x40))
	f.Add(int64(42), uint8(0xa0))
	f.Add(int64(100), uint8(0xc0))
	f.Add(int64(2147483647), uint8(0x80)) // large positive seed
	f.Add(int64(-1), uint8(0x80))         // negative seed exercises rand seeding
	f.Add(int64(7777), uint8(0xc0))
	f.Fuzz(func(t *testing.T, seed int64, numChainsRaw uint8) {
		params := RandomChainParams {
			chainCount:             max(2, int(numChainsRaw>>6)),
			minLength:              30,
			maxLength:              60,
			invalidateChance:       80,
			dependencyChance:       20,
			maxBlockTimeExclusive:  15,
		}

		fuzzInterop := newInteropFuzzHarness(t).WithParams(params).WithSeed(seed)

		fuzzInterop.Build()

		interop := fuzzInterop.interop

		// Update the LogDBs for the chains
		i := uint64(0)
		for {
			advanced, err := interop.progressAndRecord()
			if !advanced {
				break
			}
			require.NoError(t, err)
			i++
		}

		randomChain := fuzzInterop.randomChain

		safeTimestamp := i

		blocksAtTimestamp := make(map[eth.ChainID]eth.BlockID)
		l1HeadsAtTimestamp := make(map[eth.ChainID]eth.BlockID)
		for chain, container := range fuzzInterop.mocks {
			block, l1, err := container.OptimisticAt(interop.ctx, safeTimestamp)
			require.NoError(t, err)
			blocksAtTimestamp[chain] = block
			l1HeadsAtTimestamp[chain] = l1
		}

		// verifyInteropMessages upstream takes (ts, blocks, l1Heads, view). The
		// frontier view is built once per round in verify(); we replicate that
		// shape so the re-verify call exercises the same SUT path.
		view, viewErr := interop.resolveFrontierVerificationView(blocksAtTimestamp)
		require.NoError(t, viewErr)
		result, err := interop.verifyInteropMessages(safeTimestamp, blocksAtTimestamp, l1HeadsAtTimestamp, view)

		requireLogsDBChainIntegrity(t, interop)
		requireVerifiedDBChainIntegrity(t, interop)

		// Item #5: probe the public read API and confirm it agrees with the
		// committed verifiedDB state. Runs regardless of validity.
		assertVerifiedDBReadback(t, interop)

		// Item #1a: the SUT must never commit past the injected invalidation.
		// This is the high-frequency check; it works from verifiedDB state
		// regardless of whether the harness's re-verify call can complete.
		randomChain.assertProgressStoppedBeforeBug(t, interop)

		// Item #1b: strict structured assertion against the re-verify result.
		// Replaces the prior loose check `err != nil || !result.IsValid()`
		// which silently accepted regressions returning the wrong error or
		// invalidating the wrong chain. Fires only on the subset of seeds
		// where verifyInteropMessages completes cleanly at safeTimestamp.
		randomChain.assertExpectedResult(t, safeTimestamp, result, err)

		// When the chain was not invalidated, the head we observe must be
		// the tip we generated — but only for chains whose final block's
		// time is at or before safeTimestamp. Chains advance at different
		// block times, so safeTimestamp may land before some chain's tip
		// and the SUT correctly returns an earlier block in that case.
		if randomChain.invalidationKind == KindNone {
			for chainID, block := range result.L2Heads {
				rcBlocks := randomChain.chainBlocks[chainID]
				lastBlock := rcBlocks[len(rcBlocks)-1]
				if safeTimestamp >= lastBlock.Time {
					require.Equal(t, lastBlock.Hash, block.Hash,
						"chain %s: L2Heads[%s].Hash at safeTimestamp=%d should match the chain's last block (Number=%d, Time=%d)",
						chainID, chainID, safeTimestamp, lastBlock.Number, lastBlock.Time)
				}
			}
		}

		if err != nil {
			t.Logf("%s", err)
		}
	})
}

// =============================================================================
// Test Harness
// =============================================================================

type interopFuzzHarness struct {
	t              *testing.T
	interop        *Interop
	params         RandomChainParams
	seed           int64
	randomChain    RandomChain
	mocks          map[eth.ChainID]cc.InteropChain
	activationTime uint64
	dataDir        string
	skipBuild      bool // for tests that need custom construction
}

// newInteropFuzzHarness creates a new test harness with sensible defaults.
func newInteropFuzzHarness(t *testing.T) *interopFuzzHarness {
	t.Helper()
	t.Parallel()
	return &interopFuzzHarness{
		t:              t,
		mocks:          make(map[eth.ChainID]cc.InteropChain),
		dataDir:        t.TempDir(),
	}
}

// WithParams sets the parameters for random L2 chain generation.
func (h *interopFuzzHarness) WithParams(params RandomChainParams) *interopFuzzHarness {
	h.params = params
	return h
}

// WithSeed sets the seed for random generation and then generates the random
// L2 chains with it.
func (h *interopFuzzHarness) WithSeed(seed int64) *interopFuzzHarness {
	h.seed = seed
	return h
}

// WithActivation sets the interop activation timestamp.
func (h *interopFuzzHarness) WithActivation(ts uint64) *interopFuzzHarness {
	h.activationTime = ts
	return h
}

// WithDataDir sets a custom data directory (useful for error testing).
func (h *interopFuzzHarness) WithDataDir(dir string) *interopFuzzHarness {
	h.dataDir = dir
	return h
}

// SkipBuild marks that Build() should not create an Interop instance.
// Useful for tests that need to test New() directly.
func (h *interopFuzzHarness) SkipBuild() *interopFuzzHarness {
	h.skipBuild = true
	return h
}

type testWriter struct{ t *testing.T }

func (tw testWriter) Write(p []byte) (n int, err error) {
	tw.t.Logf("%s", string(p))
	return len(p), nil
}

// Build creates the Interop instance from configured mocks.
// Sets up context and registers cleanup.
func (h *interopFuzzHarness) Build() *interopFuzzHarness {
	if h.skipBuild {
		return h
	}
	h.randomChain = h.params.MakeRandomChain(h.t, h.seed)

	// Find an activationTime that all chains can satisfy
	for _, blocks := range h.randomChain.chainBlocks {
		h.activationTime = max(h.activationTime, blocks[0].Time)
	}

	h.mocks = h.randomChain.GetContainers()
	logger := gethlog.NewLogger(gethlog.NewTerminalHandler(testWriter{h.t}, true))
	// messageExpiryWindow comes from the RandomChain: 0 falls back to the
	// SUT's defaultMessageExpiryWindow (604800s) for all kinds except
	// KindExpiredMessage, which sets a small value targeted at the gap of
	// the injected expired dep so the SUT trips ErrMessageExpired.
	// logBackfillDepth=0 and metrics=nil match upstream's defaults; New()
	// substitutes a noop metrics impl when metrics is nil.
	h.interop = New(logger, h.activationTime, h.randomChain.messageExpiryWindow, h.mocks, h.dataDir, h.randomChain, 0, nil)
	if h.interop != nil {
		h.interop.ctx = context.Background()
		h.t.Cleanup(func() { _ = h.interop.Stop(context.Background()) })
	}
	return h
}

// Chains returns the map of chain containers for use with New().
func (h *interopFuzzHarness) Chains() map[eth.ChainID]cc.InteropChain {
	chains := make(map[eth.ChainID]cc.InteropChain)
	maps.Copy(chains, h.mocks)
	return chains
}

// Mock returns the mock for a given chain ID.
func (h *interopFuzzHarness) Mock(id uint64) cc.InteropChain {
	return h.mocks[eth.ChainIDFromUInt64(id)]
}

