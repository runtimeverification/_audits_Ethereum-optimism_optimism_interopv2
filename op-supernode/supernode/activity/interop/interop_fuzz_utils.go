package interop

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/stretchr/testify/require"
)

// requireLogsDBChainIntegrity asserts that every LogsDB in interop forms a valid
// chain: block numbers are sequential (+1 each step), parent hashes link to the
// previous block's hash, and timestamps strictly increase by blockTime.
func requireLogsDBChainIntegrity(t *testing.T, interop *Interop) {
	t.Helper()
	for chainID, db := range interop.logsDBs {
		latestID, ok := db.LatestSealedBlock()
		if !ok {
			continue
		}

		first, err := db.FirstSealedBlock()
		require.NoError(t, err, "chain %s: FirstSealedBlock", chainID)

		var prev types.BlockSeal
		for num := first.Number; num <= latestID.Number; num++ {
			seal, err := db.FindSealedBlock(num)
			require.NoError(t, err, "chain %s: FindSealedBlock(%d)", chainID, num)

			if num > first.Number {
				blockTime := interop.chains[chainID].BlockTime()
				require.Equal(t, seal.Timestamp, prev.Timestamp+blockTime, "chain %s: block %d: timestamp must be %d", chainID, num, prev.Timestamp+blockTime)

				require.Equal(t, prev.Number+1, seal.Number,
					"chain %s: block %d: expected sequential block number", chainID, num)

				ref, _, _, err := db.OpenBlock(num)
				require.NoError(t, err, "chain %s: OpenBlock(%d)", chainID, num)
				require.Equal(t, prev.Hash, ref.ParentHash,
					"chain %s: block %d: parent hash must match previous block hash", chainID, num)
			}
			prev = seal
		}
	}
}
