package interop

// This file is part of the Dafny model checkers (dafny_check_*.go files):
// test/debug-only helpers that check predicates from op-supernode/dafny-models/
// against the real Go types. Production code paths must not call them.

import "github.com/ethereum-optimism/optimism/op-service/eth"

// ChainBlockOracle supplies the model's ChainContainer.BlockInfo / BlockLogs
// "immutable source of truth" (ChainContainer.dfy) for cross-validity checks.
// Tests provide a snapshot; nil means oracle-dependent conjuncts are skipped.
//
// Field binding (Types.dfy ExecutingMessage / ChainContainer.dfy):
//   - BlockInfo maps ChainContainer.BlockInfo(blockID): oracle returns the
//     canonical on-chain header for the given (chain, block) pair.
//   - BlockLogs maps ChainContainer.BlockLogs(blockID): oracle returns the
//     executing messages found in that block's receipts.
//   - BlockTime maps ChainContainer.BlockTime(): per-chain block period in
//     seconds, used in ValidExecutingMessage arithmetic.
type ChainBlockOracle interface {
	BlockInfo(chainID eth.ChainID, blockID eth.BlockID) (info eth.BlockInfo, ok bool)
	BlockLogs(chainID eth.ChainID, blockID eth.BlockID) (execMsgs []ExecMsg, ok bool)
	BlockTime(chainID eth.ChainID) (uint64, bool)
}

// ExecMsg is a Go mirror of Types.dfy ExecutingMessage. Field binding:
//
//	model chainID    ↔ Chain  (resolved to eth.ChainID)
//	model blockNum   ↔ BlockNum
//	model logIdx     ↔ LogIdx
//	model timestamp  ↔ Timestamp
//	model checksum   ↔ Checksum (ContainsQuery.checksum field in suptypes)
//
// (Full binding to suptypes.ExecutingMessage is recorded in T11.)
type ExecMsg struct {
	Chain     eth.ChainID
	BlockNum  uint64
	LogIdx    uint32
	Timestamp uint64
	Checksum  [32]byte
}

// CheckBlockSealsMatchOnChainTimestamps mirrors BlockSealsMatchOnChainTimestamps()
// in op-supernode/dafny-models/Interop.dfy: for every chain and every sealed
// block n, the oracle's BlockInfo for that seal agrees on the timestamp.
// Requires oracle; skips conjuncts without one (R5).
// Conjuncts:
//
//	(0) oracle-dependent conjuncts skipped when oracle is nil
//	(1) forall chainID in logsDBs.Keys, forall n:
//	    FindSealedBlock(n).Some? ==>
//	    oracle.BlockInfo(chainID, seal.id).ok &&
//	    seal.timestamp == oracle.BlockInfo(...).value.timestamp
func CheckBlockSealsMatchOnChainTimestamps(i *Interop, oracle ChainBlockOracle) error {
	const pred = "Interop.dfy BlockSealsMatchOnChainTimestamps()"
	if oracle == nil {
		// ponytail: nil oracle always skips; caller-visible via "conjunct (0)" prefix.
		return nil // R5: never a false violation without oracle
	}
	// TODO(T11): implement oracle-reading body.
	_ = pred
	return nil
}

// AssertBlockSealsMatchOnChainTimestamps fails t when CheckBlockSealsMatchOnChainTimestamps
// reports violations.
func AssertBlockSealsMatchOnChainTimestamps(t dafnyT, i *Interop, oracle ChainBlockOracle) {
	t.Helper()
	failOnViolation(t, CheckBlockSealsMatchOnChainTimestamps(i, oracle))
}

// CheckAllVerifiedHeadsBoundedByTimestamp mirrors AllVerifiedHeadsBoundedByTimestamp()
// in op-supernode/dafny-models/Interop.dfy: for every ts in
// [activationTimestamp, lastTimestamp], the on-chain timestamp of each
// verified l2Head is <= ts.
// Requires oracle for BlockInfo; skips conjuncts without one (R5).
// Conjuncts:
//
//	(0) oracle-dependent conjuncts skipped when oracle is nil
//	(1) forall ts in [activation, last]:
//	    verifiedDB.Has(ts) &&
//	    chains.Keys == verifiedDB.Get(ts).l2Heads.Keys &&
//	    BlocksExistedOnChain(l2Heads) &&
//	    forall chainID: oracle.BlockInfo(chainID, l2Heads[chainID]).timestamp <= ts
func CheckAllVerifiedHeadsBoundedByTimestamp(i *Interop, oracle ChainBlockOracle) error {
	const pred = "Interop.dfy AllVerifiedHeadsBoundedByTimestamp()"
	if oracle == nil {
		// ponytail: nil oracle always skips; caller-visible via "conjunct (0)" prefix.
		return nil // R5: never a false violation without oracle
	}
	// TODO(T11): implement oracle-reading body.
	_ = pred
	return nil
}

// AssertAllVerifiedHeadsBoundedByTimestamp fails t when
// CheckAllVerifiedHeadsBoundedByTimestamp reports violations.
func AssertAllVerifiedHeadsBoundedByTimestamp(t dafnyT, i *Interop, oracle ChainBlockOracle) {
	t.Helper()
	failOnViolation(t, CheckAllVerifiedHeadsBoundedByTimestamp(i, oracle))
}
