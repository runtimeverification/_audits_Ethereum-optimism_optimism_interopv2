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
	f.Add(int64(69), byte('\x00')) // Failing test (gap > blockTime, because the chain randomizer isn't respecting blockTime yet)

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
		for {
			advanced, err := interop.progressAndRecord()
			require.NoError(t, err)
			if !advanced {
				break
			}
		}

		randomChain := fuzzInterop.randomChain

		safeBlock := randomChain.allBlocks[len(randomChain.allBlocks)-1]
		safeTimestamp := safeBlock.block.Time

		blocksAtTimestamp, err := interop.checkChainsReady(safeTimestamp)
		require.NoError(t, err)

		result, err := interop.verifyInteropMessages(safeTimestamp, blocksAtTimestamp)
		if !randomChain.isInvalid {
			require.NoError(t, err)
			for chain, block := range result.L2Heads {
				rcBlocks := randomChain.chainBlocks[chain]
				lastBlock := rcBlocks[len(rcBlocks)-1]
				require.Equal(t, block.Hash, lastBlock.Hash)
			}
		}

		// P1: Valid messages never produce InvalidHeads
		require.True(t, result.IsValid(), "P1: valid messages should produce valid result, got InvalidHeads: %v", result.InvalidHeads)

		// P3: IsValid() ↔ len(InvalidHeads) == 0
		require.Empty(t, result.InvalidHeads, "P3: InvalidHeads should be empty for valid result")
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
	mocks          map[eth.ChainID]cc.ChainContainer
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
		mocks:          make(map[eth.ChainID]cc.ChainContainer),
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
	h.interop = New(logger, h.activationTime, h.mocks, h.dataDir)
	if h.interop != nil {
		h.interop.ctx = context.Background()
		h.t.Cleanup(func() { _ = h.interop.Stop(context.Background()) })
	}
	return h
}

// Chains returns the map of chain containers for use with New().
func (h *interopFuzzHarness) Chains() map[eth.ChainID]cc.ChainContainer {
	chains := make(map[eth.ChainID]cc.ChainContainer)
	maps.Copy(chains, h.mocks)
	return chains
}

// Mock returns the mock for a given chain ID.
func (h *interopFuzzHarness) Mock(id uint64) cc.ChainContainer {
	return h.mocks[eth.ChainIDFromUInt64(id)]
}

