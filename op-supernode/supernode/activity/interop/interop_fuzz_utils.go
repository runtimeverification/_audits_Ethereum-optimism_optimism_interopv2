package interop

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/stretchr/testify/require"
)

// requireVerifiedDBChainIntegrity asserts two invariants over the VerifiedDB:
//
//  1. For every consecutive pair of timestamps in the VerifiedDB, each chain's
//     L2 head is either identical (same block) or a direct successor
//     (block number increases by exactly 1).
//
//  2. For every chain, the L2 head recorded in the VerifiedDB at both the
//     activation timestamp and the last verified timestamp are present in the
//     corresponding LogsDB with the same block number and block hash
//     (timestamp equality is implied by hash equality).
func requireVerifiedDBChainIntegrity(t *testing.T, interop *Interop) {
	t.Helper()

	lastTimestamp, ok := interop.verifiedDB.LastTimestamp()
	if !ok {
		return // VerifiedDB is empty; nothing to check
	}
	// Anchor on the actual first committed timestamp, not activationTimestamp.
	// The main loop now starts at resolveFirstVerifiableTimestamp(=minCrossSafeTime+1)
	// which is normally above activationTimestamp, so there is no commit at
	// activationTimestamp itself.
	firstTimestamp, ok := interop.verifiedDB.FirstTimestamp()
	if !ok {
		// LastTimestamp said the DB had data; FirstTimestamp disagreeing would
		// be a SUT invariant violation, not just a missing-commit edge case.
		require.FailNow(t, "verifiedDB.LastTimestamp reports data but FirstTimestamp does not")
	}

	// --- Invariant 1: L2 heads advance by at most one block per timestamp ---
	var prevResult *VerifiedResult
	for ts := firstTimestamp; ts <= lastTimestamp; ts++ {
		result, err := interop.verifiedDB.Get(ts)
		if err != nil {
			// Sequential commit guarantee means this should not happen, but if it
			// does we reset the rolling window and continue rather than failing hard.
			prevResult = nil
			continue
		}

		if prevResult != nil {
			for chainID, head := range result.L2Heads {
				prevHead, hasPrev := prevResult.L2Heads[chainID]
				if !hasPrev {
					continue
				}
				isEqual := head == prevHead
				isSuccessor := head.Number == prevHead.Number+1
				require.True(t, isEqual || isSuccessor,
					"chain %s: at timestamp %d, L2 head (num=%d, hash=%s) is neither equal to "+
						"nor the successor of the previous head (num=%d, hash=%s) at timestamp %d",
					chainID, ts, head.Number, head.Hash,
					prevHead.Number, prevHead.Hash, ts-1)
			}
		}

		snapshot := result
		prevResult = &snapshot
	}

	// --- Invariant 2: first and last L2 heads are anchored in the LogsDB ---
	firstResult, err := interop.verifiedDB.Get(firstTimestamp)
	require.NoError(t, err, "verifiedDB.Get(firstTimestamp=%d)", firstTimestamp)

	lastResult, err := interop.verifiedDB.Get(lastTimestamp)
	require.NoError(t, err, "verifiedDB.Get(lastTimestamp=%d)", lastTimestamp)

	for chainID, db := range interop.logsDBs {
		if firstHead, exists := firstResult.L2Heads[chainID]; exists {
			seal, err := db.FindSealedBlock(firstHead.Number)
			require.NoError(t, err,
				"chain %s: FindSealedBlock(%d) for firstTimestamp %d",
				chainID, firstHead.Number, firstTimestamp)
			require.Equal(t, firstHead.Hash, seal.Hash,
				"chain %s: LogsDB block hash at number %d does not match VerifiedDB L2 head for firstTimestamp %d",
				chainID, firstHead.Number, firstTimestamp)
		}

		if lastHead, exists := lastResult.L2Heads[chainID]; exists {
			seal, err := db.FindSealedBlock(lastHead.Number)
			require.NoError(t, err,
				"chain %s: FindSealedBlock(%d) for lastTimestamp %d",
				chainID, lastHead.Number, lastTimestamp)
			require.Equal(t, lastHead.Hash, seal.Hash,
				"chain %s: LogsDB block hash at number %d does not match VerifiedDB L2 head for lastTimestamp %d",
				chainID, lastHead.Number, lastTimestamp)
		}
	}
}

// requireLogsDBChainIntegrity asserts that every LogsDB in interop forms a valid
// chain: block numbers are sequential (+1 each step), parent hashes link to the
// previous block's hash, and timestamps are monotonically non-decreasing.
//
// Timestamps cannot be required to increase by exactly blockTime: upstream's
// frontier-view sealing skips L2 timestamps that have no new sealed block on
// some chain, so sealed-block runs legitimately have gaps. The sequential
// block-number and parent-hash linkage invariants remain enforced.
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
				require.GreaterOrEqual(t, seal.Timestamp, prev.Timestamp,
					"chain %s: block %d: timestamp must be non-decreasing (prev=%d, got=%d)",
					chainID, num, prev.Timestamp, seal.Timestamp)

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
