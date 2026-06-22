package interop

// This file is part of the Dafny model checkers (dafny_check_*.go files):
// test/debug-only helpers that check predicates from op-supernode/dafny-models/
// against the real Go types. Production code paths must not call them.

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

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
//	model checksum   ↔ Checksum (suptypes.MessageChecksum == common.Hash)
//
// Binding to suptypes.ExecutingMessage: Chain↔ChainID, BlockNum↔BlockNum,
// LogIdx↔LogIdx, Timestamp↔Timestamp, Checksum↔Checksum.
type ExecMsg struct {
	Chain     eth.ChainID
	BlockNum  uint64
	LogIdx    uint32
	Timestamp uint64
	Checksum  suptypes.MessageChecksum
}

// execMsgFromSup converts a suptypes.ExecutingMessage to ExecMsg.
func execMsgFromSup(m *suptypes.ExecutingMessage) ExecMsg {
	return ExecMsg{
		Chain:     m.ChainID,
		BlockNum:  m.BlockNum,
		LogIdx:    m.LogIdx,
		Timestamp: m.Timestamp,
		Checksum:  m.Checksum,
	}
}

// CheckValidExecutingMessage mirrors ValidExecutingMessage(execTimestamp,
// execChain, execMsg) in op-supernode/dafny-models/Interop.dfy.
// Uses oracle.BlockTime for the per-chain block times; if the oracle is nil
// or a chain's BlockTime is unavailable, those arithmetic conjuncts are
// skipped (R5). Preserves the additive (non-subtractive) form from the model.
// Conjuncts:
//
//	(0) execChain in CHAIN_IDS and m.Chain in CHAIN_IDS (mapping requirement)
//	(1) activationTimestamp + execBlockTime <= execTimestamp
//	    (skipped when oracle is nil or BlockTime unavailable for execChain)
//	(2) activationTimestamp + initBlockTime <= m.Timestamp
//	    (skipped when oracle is nil or BlockTime unavailable for m.Chain)
//	(3) m.Timestamp <= execTimestamp (initTimestamp <= execTimestamp)
//	(4) execTimestamp <= m.Timestamp + messageExpiryWindow
func CheckValidExecutingMessage(p ModelParams, oracle ChainBlockOracle, execTS uint64, execChain eth.ChainID, m ExecMsg) error {
	const pred = "Interop.dfy ValidExecutingMessage"

	if _, ok := p.ChainIDs[execChain]; !ok {
		return violation(pred, "0", "execChain %s not in CHAIN_IDS", execChain)
	}
	if _, ok := p.ChainIDs[m.Chain]; !ok {
		return violation(pred, "0", "initChain %s not in CHAIN_IDS", m.Chain)
	}

	var errs []error

	// Conjuncts (1) and (2) require oracle.BlockTime; skip per-chain when
	// oracle is nil or the chain's block time is not available in the oracle
	// (R5: per-chain BlockTime-missing skip is intentional, not a violation).
	if oracle != nil {
		if execBT, ok := oracle.BlockTime(execChain); ok {
			if p.ActivationTimestamp+execBT > execTS {
				errs = append(errs, violation(pred, "1",
					"activationTimestamp %d + execBlockTime %d > execTimestamp %d",
					p.ActivationTimestamp, execBT, execTS))
			}
		}
		// else: execChain BlockTime unavailable — skip conjunct (1)
		if initBT, ok := oracle.BlockTime(m.Chain); ok {
			if p.ActivationTimestamp+initBT > m.Timestamp {
				errs = append(errs, violation(pred, "2",
					"activationTimestamp %d + initBlockTime %d > initTimestamp %d",
					p.ActivationTimestamp, initBT, m.Timestamp))
			}
		}
		// else: m.Chain BlockTime unavailable — skip conjunct (2)
	}

	// Conjunct (3): initTimestamp <= execTimestamp.
	if m.Timestamp > execTS {
		errs = append(errs, violation(pred, "3",
			"initTimestamp %d > execTimestamp %d", m.Timestamp, execTS))
	}

	// Conjunct (4): execTimestamp <= initTimestamp + messageExpiryWindow.
	if execTS > m.Timestamp+p.MessageExpiryWindow {
		errs = append(errs, violation(pred, "4",
			"execTimestamp %d > initTimestamp %d + messageExpiryWindow %d",
			execTS, m.Timestamp, p.MessageExpiryWindow))
	}

	return errors.Join(errs...)
}

// AssertValidExecutingMessage fails t when CheckValidExecutingMessage reports
// violations.
func AssertValidExecutingMessage(t dafnyT, p ModelParams, oracle ChainBlockOracle, execTS uint64, execChain eth.ChainID, m ExecMsg) {
	t.Helper()
	failOnViolation(t, CheckValidExecutingMessage(p, oracle, execTS, execChain, m))
}

// containsModel maps LogsDB.Contains to the model's boolean: true ↔ (_, nil),
// false ↔ ErrConflict/ErrFuture/ErrSkipped.
func containsModel(db LogsDB, q suptypes.ContainsQuery) (bool, error) {
	_, err := db.Contains(q)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, suptypes.ErrConflict) ||
		errors.Is(err, suptypes.ErrFuture) ||
		errors.Is(err, suptypes.ErrSkipped) {
		return false, nil
	}
	return false, err
}

// CheckInitMsgInLogsDB mirrors InitMsgInLogsDB(execMsg) in
// op-supernode/dafny-models/Interop.dfy: the initiating message identified by
// m is present in the logsDB for m.Chain. Model true ↔ Contains returns
// (seal, nil); ErrConflict/ErrFuture/ErrSkipped map to model false. Conjuncts:
//
//	(0) i is non-nil and m.Chain in logsDBs.Keys (mapping requirement and the
//	    model's requires clause); unexpected Contains errors also reported here
//	(1) logsDBs[m.Chain].Contains(query) is true (model Contains = true)
func CheckInitMsgInLogsDB(i *Interop, m ExecMsg) error {
	const pred = "Interop.dfy InitMsgInLogsDB"
	if i == nil {
		return violation(pred, "0", "Interop is nil")
	}
	db, ok := i.logsDBs[m.Chain]
	if !ok || db == nil {
		return violation(pred, "0", "chain %s has no logsDB", m.Chain)
	}
	q := suptypes.ContainsQuery{
		BlockNum:  m.BlockNum,
		LogIdx:    m.LogIdx,
		Timestamp: m.Timestamp,
		Checksum:  m.Checksum,
	}
	found, err := containsModel(db, q)
	if err != nil {
		return violation(pred, "0", "Contains failed: %v", err)
	}
	if !found {
		return violation(pred, "1",
			"logsDB for chain %s does not contain message block=%d logIdx=%d ts=%d",
			m.Chain, m.BlockNum, m.LogIdx, m.Timestamp)
	}
	return nil
}

// AssertInitMsgInLogsDB fails t when CheckInitMsgInLogsDB reports violations.
func AssertInitMsgInLogsDB(t dafnyT, i *Interop, m ExecMsg) {
	t.Helper()
	failOnViolation(t, CheckInitMsgInLogsDB(i, m))
}

// CheckLogsDBConsistentWithChainData mirrors LogsDBConsistentWithChainData(chainID)
// in op-supernode/dafny-models/Interop.dfy (opaque predicate): for every sealed
// block in the logsDB, the oracle's BlockInfo agrees on the timestamp, and the
// oracle's BlockLogs agrees with the logsDB recorded exec msgs. Skips without
// oracle (R5).
// Conjuncts:
//
//	(0) oracle-dependent conjuncts skipped when oracle is nil; i non-nil,
//	    chainID in logsDBs.Keys (mapping requirements)
//	(1) forall sealed block n in logsDB:
//	    oracle.BlockInfo(chainID, seal.id).ok &&
//	    seal.timestamp == oracle.BlockInfo(...).value.timestamp
//	(2) forall sealed block n in logsDB:
//	    oracle.BlockLogs(chainID, seal.id).ok &&
//	    len(logsDB.OpenBlock(n).execMsgs) == len(oracle.BlockLogs(...))
//	    (model: db.BlockLogs(n).execMsgs == oracle.BlockLogs; Go exposes exec
//	    msgs via OpenBlock; ErrSkipped on the anchor block maps to 0 exec msgs)
func CheckLogsDBConsistentWithChainData(i *Interop, oracle ChainBlockOracle, chainID eth.ChainID) error {
	const pred = "Interop.dfy LogsDBConsistentWithChainData"
	if i == nil {
		return violation(pred, "0", "Interop is nil")
	}
	db, ok := i.logsDBs[chainID]
	if !ok || db == nil {
		return violation(pred, "0", "chain %s has no logsDB", chainID)
	}
	if oracle == nil {
		return nil // R5: skip oracle-dependent conjuncts
	}

	// Enumerate all sealed blocks by scanning from first to latest.
	first, err := db.FirstSealedBlock()
	if errors.Is(err, suptypes.ErrFuture) || errors.Is(err, suptypes.ErrSkipped) {
		return nil // empty db: predicate holds vacuously
	}
	if err != nil {
		return violation(pred, "0", "FirstSealedBlock: %v", err)
	}
	latest, hasLatest := db.LatestSealedBlock()
	if !hasLatest {
		return nil // empty
	}

	var errs []error
	for n := first.Number; n <= latest.Number; n++ {
		seal, found, ferr := findSealedOption(db, n)
		if ferr != nil {
			errs = append(errs, violation(pred, "0",
				"chain %s FindSealedBlock(%d) failed: %v", chainID, n, ferr))
			if n == latest.Number {
				break
			}
			continue
		}
		if !found {
			if n == latest.Number {
				break
			}
			continue // gap: vacuously consistent
		}
		info, infoOK := oracle.BlockInfo(chainID, seal.ID())
		if !infoOK {
			errs = append(errs, violation(pred, "1",
				"chain %s block %d: oracle has no BlockInfo for %s", chainID, n, seal.ID()))
		} else if seal.Timestamp != info.Time() {
			errs = append(errs, violation(pred, "1",
				"chain %s block %d: seal.timestamp %d != oracle.BlockInfo.timestamp %d",
				chainID, n, seal.Timestamp, info.Time()))
		}
		oracleLogs, logsOK := oracle.BlockLogs(chainID, seal.ID())
		if !logsOK {
			errs = append(errs, violation(pred, "2",
				"chain %s block %d: oracle has no BlockLogs for %s", chainID, n, seal.ID()))
		} else {
			// Compare exec msg count from logsDB vs oracle.
			_, _, dbExecMsgs, openErr := db.OpenBlock(n)
			var dbCount int
			switch {
			case errors.Is(openErr, suptypes.ErrSkipped):
				dbCount = 0 // anchor block: ErrSkipped maps to 0 exec msgs
			case openErr == nil:
				dbCount = len(dbExecMsgs)
			default:
				errs = append(errs, violation(pred, "2",
					"chain %s block %d: OpenBlock failed: %v", chainID, n, openErr))
				if n == latest.Number {
					break
				}
				continue
			}
			if dbCount != len(oracleLogs) {
				errs = append(errs, violation(pred, "2",
					"chain %s block %d: logsDB exec msg count %d != oracle count %d",
					chainID, n, dbCount, len(oracleLogs)))
			}
		}
		if n == latest.Number {
			break
		}
	}
	return errors.Join(errs...)
}

// AssertLogsDBConsistentWithChainData fails t when
// CheckLogsDBConsistentWithChainData reports violations.
func AssertLogsDBConsistentWithChainData(t dafnyT, i *Interop, oracle ChainBlockOracle, chainID eth.ChainID) {
	t.Helper()
	failOnViolation(t, CheckLogsDBConsistentWithChainData(i, oracle, chainID))
}

// CheckAllLogsDBsConsistentWithChainData mirrors AllLogsDBsConsistentWithChainData()
// in op-supernode/dafny-models/Interop.dfy:
// `forall chainID :: chainID in logsDBs.Keys ==> LogsDBConsistentWithChainData(chainID)`.
// Skips oracle-dependent conjuncts without an oracle (R5). Violations carry
// the failing chain's ID; per-chain conjunct labels are those of
// CheckLogsDBConsistentWithChainData.
func CheckAllLogsDBsConsistentWithChainData(i *Interop, oracle ChainBlockOracle) error {
	const pred = "Interop.dfy AllLogsDBsConsistentWithChainData"
	if i == nil {
		return violation(pred, "0", "Interop is nil")
	}
	var errs []error
	for _, k := range sortedLogsDBChainIDs(i) {
		if err := CheckLogsDBConsistentWithChainData(i, oracle, k); err != nil {
			errs = append(errs, fmt.Errorf("%s: chain %s: %w", pred, k, err))
		}
	}
	return errors.Join(errs...)
}

// AssertAllLogsDBsConsistentWithChainData fails t when
// CheckAllLogsDBsConsistentWithChainData reports violations.
func AssertAllLogsDBsConsistentWithChainData(t dafnyT, i *Interop, oracle ChainBlockOracle) {
	t.Helper()
	failOnViolation(t, CheckAllLogsDBsConsistentWithChainData(i, oracle))
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
		return nil // R5: never a false violation without oracle
	}
	if i == nil {
		return violation(pred, "0", "Interop is nil")
	}

	var errs []error
	for _, chainID := range sortedLogsDBChainIDs(i) {
		db := i.logsDBs[chainID]
		first, err := db.FirstSealedBlock()
		if errors.Is(err, suptypes.ErrFuture) || errors.Is(err, suptypes.ErrSkipped) {
			continue // empty logsDB: vacuously true
		}
		if err != nil {
			errs = append(errs, violation(pred, "0",
				"chain %s FirstSealedBlock: %v", chainID, err))
			continue
		}
		latest, hasLatest := db.LatestSealedBlock()
		if !hasLatest {
			continue
		}
		for n := first.Number; n <= latest.Number; n++ {
			seal, found, ferr := findSealedOption(db, n)
			if ferr != nil {
				errs = append(errs, violation(pred, "0",
					"chain %s FindSealedBlock(%d): %v", chainID, n, ferr))
				if n == latest.Number {
					break
				}
				continue
			}
			if !found {
				if n == latest.Number {
					break
				}
				continue
			}
			info, ok := oracle.BlockInfo(chainID, seal.ID())
			if !ok {
				errs = append(errs, violation(pred, "1",
					"chain %s block %d: oracle has no BlockInfo for %s", chainID, n, seal.ID()))
			} else if seal.Timestamp != info.Time() {
				errs = append(errs, violation(pred, "1",
					"chain %s block %d: seal.timestamp %d != oracle.BlockInfo.timestamp %d",
					chainID, n, seal.Timestamp, info.Time()))
			}
			if n == latest.Number {
				break
			}
		}
	}
	return errors.Join(errs...)
}

// AssertBlockSealsMatchOnChainTimestamps fails t when CheckBlockSealsMatchOnChainTimestamps
// reports violations.
func AssertBlockSealsMatchOnChainTimestamps(t dafnyT, i *Interop, oracle ChainBlockOracle) {
	t.Helper()
	failOnViolation(t, CheckBlockSealsMatchOnChainTimestamps(i, oracle))
}

// CheckAllVerifiedHeadsBoundedByTimestamp enforces the timestamp-bound portion
// of AllVerifiedHeadsBoundedByTimestamp() in
// op-supernode/dafny-models/Interop.dfy: for every entry present in the
// verifiedDB, the on-chain timestamp of each verified l2Head is <= ts.
// Requires oracle for BlockInfo; skips conjuncts without one (R5).
//
// Note: the completeness conjunct (verifiedDB.Has(ts) for every ts in
// [activation, last]) is covered by CheckVerifiedDBValid (checkSequential).
// The keys-equality conjunct (chains.Keys == l2Heads.Keys) is covered by
// CheckVerifiedHeadsAreHighestBlocksUpToTimestamp (conjunct A). This checker
// enforces only the oracle-dependent bound for entries that are present.
// Conjuncts:
//
//	(0) oracle-dependent conjuncts skipped when oracle is nil
//	(1) forall ts in verifiedDB (within [activation, last]):
//	    forall chainID in l2Heads:
//	    oracle.BlockInfo(chainID, l2Heads[chainID]).ok &&
//	    oracle.BlockInfo(...).timestamp <= ts
func CheckAllVerifiedHeadsBoundedByTimestamp(i *Interop, oracle ChainBlockOracle) error {
	const pred = "Interop.dfy AllVerifiedHeadsBoundedByTimestamp()"
	if oracle == nil {
		return nil // R5: never a false violation without oracle
	}
	if i == nil {
		return violation(pred, "0", "Interop is nil")
	}
	if i.verifiedDB == nil || i.verifiedDB.db == nil {
		return violation(pred, "0", "VerifiedDB has no underlying store")
	}

	lastTS, initialized := i.verifiedDB.LastTimestamp()
	if !initialized {
		return nil // empty db: predicate holds vacuously
	}

	p := modelParamsFromInterop(i)
	snapshot, err := i.verifiedDB.allVerified()
	if err != nil {
		return violation(pred, "0", "enumerate verified bucket: %v", err)
	}

	var errs []error
	for _, ts := range slices.Sorted(maps.Keys(snapshot)) {
		if ts < p.ActivationTimestamp || ts > lastTS {
			continue
		}
		verifiedHeads := snapshot[ts].L2Heads
		for _, chainID := range sortedChainIDs(verifiedHeads) {
			head := verifiedHeads[chainID]
			info, ok := oracle.BlockInfo(chainID, head)
			if !ok {
				errs = append(errs, violation(pred, "1",
					"ts %d chain %s: oracle has no BlockInfo for %s", ts, chainID, head))
				continue
			}
			if info.Time() > ts {
				errs = append(errs, violation(pred, "1",
					"ts %d chain %s: on-chain timestamp %d > verified ts",
					ts, chainID, info.Time()))
			}
		}
	}
	return errors.Join(errs...)
}

// AssertAllVerifiedHeadsBoundedByTimestamp fails t when
// CheckAllVerifiedHeadsBoundedByTimestamp reports violations.
func AssertAllVerifiedHeadsBoundedByTimestamp(t dafnyT, i *Interop, oracle ChainBlockOracle) {
	t.Helper()
	failOnViolation(t, CheckAllVerifiedHeadsBoundedByTimestamp(i, oracle))
}

// CheckBlocksExistedOnChain mirrors BlockExistedOnChain / BlocksExistedOnChain
// in op-supernode/dafny-models/Interop.dfy: for every (chainID, blockID) pair
// in blocks, the oracle must have a BlockInfo entry (chains[c].BlockInfo(b).Some?).
// Requires oracle; skips without one (R5).
// Conjuncts:
//
//	(0) oracle-dependent conjuncts skipped when oracle is nil;
//	    blocks.Keys must be non-empty (mapping requirement)
//	(1) forall chainID in blocks.Keys:
//	    oracle.BlockInfo(chainID, blocks[chainID]).ok
func CheckBlocksExistedOnChain(i *Interop, oracle ChainBlockOracle, blocks map[eth.ChainID]eth.BlockID) error {
	const pred = "Interop.dfy BlocksExistedOnChain"
	if oracle == nil {
		return nil // R5: skip oracle-dependent conjuncts
	}
	var errs []error
	for _, chainID := range sortedChainIDs(blocks) {
		blockID := blocks[chainID]
		if _, ok := oracle.BlockInfo(chainID, blockID); !ok {
			errs = append(errs, violation(pred, "1",
				"chain %s block %s not found in oracle", chainID, blockID))
		}
	}
	return errors.Join(errs...)
}

// AssertBlocksExistedOnChain fails t when CheckBlocksExistedOnChain reports
// violations.
func AssertBlocksExistedOnChain(t dafnyT, i *Interop, oracle ChainBlockOracle, blocks map[eth.ChainID]eth.BlockID) {
	t.Helper()
	failOnViolation(t, CheckBlocksExistedOnChain(i, oracle, blocks))
}

// CheckFrontierBlocksConsistentWithTimestamp mirrors
// FrontierBlocksConsistentWithTimestamp(ts, blocksAtTS) in
// op-supernode/dafny-models/Interop.dfy: for every chain,
// the oracle's BlockInfo timestamp is <= ts.
// Requires oracle; skips without one (R5).
// Conjuncts:
//
//	(0) oracle-dependent conjuncts skipped when oracle is nil
//	(1) forall chainID in blocks.Keys:
//	    oracle.BlockInfo(chainID, blocks[chainID]).ok &&
//	    oracle.BlockInfo(...).timestamp <= ts
func CheckFrontierBlocksConsistentWithTimestamp(i *Interop, oracle ChainBlockOracle, ts uint64, blocks map[eth.ChainID]eth.BlockID) error {
	const pred = "Interop.dfy FrontierBlocksConsistentWithTimestamp"
	if oracle == nil {
		return nil // R5: skip oracle-dependent conjuncts
	}
	var errs []error
	for _, chainID := range sortedChainIDs(blocks) {
		blockID := blocks[chainID]
		info, ok := oracle.BlockInfo(chainID, blockID)
		if !ok {
			errs = append(errs, violation(pred, "1",
				"chain %s block %s not found in oracle", chainID, blockID))
			continue
		}
		if info.Time() > ts {
			errs = append(errs, violation(pred, "1",
				"chain %s block %s: on-chain timestamp %d > ts %d",
				chainID, blockID, info.Time(), ts))
		}
	}
	return errors.Join(errs...)
}

// AssertFrontierBlocksConsistentWithTimestamp fails t when
// CheckFrontierBlocksConsistentWithTimestamp reports violations.
func AssertFrontierBlocksConsistentWithTimestamp(t dafnyT, i *Interop, oracle ChainBlockOracle, ts uint64, blocks map[eth.ChainID]eth.BlockID) {
	t.Helper()
	failOnViolation(t, CheckFrontierBlocksConsistentWithTimestamp(i, oracle, ts, blocks))
}

// CheckInitMsgInFrontier mirrors InitMsgInFrontier(execMsg, blocksAtTS) in
// op-supernode/dafny-models/Interop.dfy: the initiating message m is present
// in the frontier block for m.Chain. Requires oracle; skips without one (R5).
//
// Model mapping:
//
//	initBlock := blocksAtTS[m.Chain]
//	initBlock.number == m.BlockNum
//	oracle.BlockInfo(m.Chain, initBlock).ok && .timestamp == m.Timestamp
//	oracle.BlockLogs(m.Chain, initBlock).ok &&
//	  m.LogIdx < len(fullLogs) && fullLogs[m.LogIdx].Checksum == m.Checksum
//
// Conjuncts:
//
//	(0) oracle-dependent conjuncts skipped when oracle is nil; m.Chain must
//	    be present in blocks
//	(1) initBlock.number == m.BlockNum
//	(2) oracle.BlockInfo(m.Chain, initBlock).ok && .timestamp == m.Timestamp
//	(3) oracle.BlockLogs(m.Chain, initBlock).ok && m.LogIdx in range &&
//	    fullLogs[m.LogIdx].Checksum == m.Checksum
func CheckInitMsgInFrontier(i *Interop, oracle ChainBlockOracle, m ExecMsg, blocks map[eth.ChainID]eth.BlockID) error {
	const pred = "Interop.dfy InitMsgInFrontier"
	if oracle == nil {
		return nil // R5: skip oracle-dependent conjuncts
	}
	initBlock, ok := blocks[m.Chain]
	if !ok {
		return violation(pred, "0", "chain %s not in frontier blocks", m.Chain)
	}
	// Conjunct (1): block number matches.
	if initBlock.Number != m.BlockNum {
		return violation(pred, "1",
			"chain %s: frontier block number %d != message blockNum %d",
			m.Chain, initBlock.Number, m.BlockNum)
	}
	// Conjunct (2): on-chain timestamp matches.
	info, infoOK := oracle.BlockInfo(m.Chain, initBlock)
	if !infoOK {
		return violation(pred, "2",
			"chain %s block %s: oracle has no BlockInfo", m.Chain, initBlock)
	}
	if info.Time() != m.Timestamp {
		return violation(pred, "2",
			"chain %s block %s: oracle timestamp %d != message timestamp %d",
			m.Chain, initBlock, info.Time(), m.Timestamp)
	}
	// Conjunct (3): logIdx in range and checksum matches.
	logs, logsOK := oracle.BlockLogs(m.Chain, initBlock)
	if !logsOK {
		return violation(pred, "3",
			"chain %s block %s: oracle has no BlockLogs", m.Chain, initBlock)
	}
	if uint32(len(logs)) <= m.LogIdx {
		return violation(pred, "3",
			"chain %s block %s: logIdx %d out of range (len %d)",
			m.Chain, initBlock, m.LogIdx, len(logs))
	}
	if logs[m.LogIdx].Checksum != m.Checksum {
		return violation(pred, "3",
			"chain %s block %s logIdx %d: checksum mismatch",
			m.Chain, initBlock, m.LogIdx)
	}
	return nil
}

// AssertInitMsgInFrontier fails t when CheckInitMsgInFrontier reports
// violations.
func AssertInitMsgInFrontier(t dafnyT, i *Interop, oracle ChainBlockOracle, m ExecMsg, blocks map[eth.ChainID]eth.BlockID) {
	t.Helper()
	failOnViolation(t, CheckInitMsgInFrontier(i, oracle, m, blocks))
}

// CheckAllInitMsgsInLogsDB mirrors AllInitMsgsInLogsDB(chainID, blockID) in
// op-supernode/dafny-models/Interop.dfy: every executing message in the oracle's
// BlockLogs for (chainID, blockID) must satisfy InitMsgInLogsDB. Requires oracle;
// skips without one (R5).
// Conjuncts:
//
//	(0) oracle-dependent conjuncts skipped when oracle is nil; i non-nil
//	(1) oracle.BlockLogs(chainID, blockID).ok
//	(2) forall execMsg in BlockLogs.execMsgs:
//	    execMsg.chainID in logsDBs.Keys && InitMsgInLogsDB(execMsg)
func CheckAllInitMsgsInLogsDB(i *Interop, oracle ChainBlockOracle, chainID eth.ChainID, blockID eth.BlockID) error {
	const pred = "Interop.dfy AllInitMsgsInLogsDB"
	if oracle == nil {
		return nil // R5: skip oracle-dependent conjuncts
	}
	if i == nil {
		return violation(pred, "0", "Interop is nil")
	}
	logs, ok := oracle.BlockLogs(chainID, blockID)
	if !ok {
		return violation(pred, "1",
			"chain %s block %s: oracle has no BlockLogs", chainID, blockID)
	}
	var errs []error
	for idx, msg := range logs {
		if err := CheckInitMsgInLogsDB(i, msg); err != nil {
			errs = append(errs, fmt.Errorf("%s conjunct (2): execMsg[%d]: %w", pred, idx, err))
		}
	}
	return errors.Join(errs...)
}

// AssertAllInitMsgsInLogsDB fails t when CheckAllInitMsgsInLogsDB reports
// violations.
func AssertAllInitMsgsInLogsDB(t dafnyT, i *Interop, oracle ChainBlockOracle, chainID eth.ChainID, blockID eth.BlockID) {
	t.Helper()
	failOnViolation(t, CheckAllInitMsgsInLogsDB(i, oracle, chainID, blockID))
}

// CheckAllInitMsgsPresent mirrors AllInitMsgsPresent(chainID, blockID, blocksAtTS)
// in op-supernode/dafny-models/Interop.dfy: every executing message in the
// oracle's BlockLogs for (chainID, blockID) must satisfy
// InitMsgInFrontier(execMsg, blocks) || InitMsgInLogsDB(execMsg). Requires
// oracle; skips without one (R5).
// Conjuncts:
//
//	(0) oracle-dependent conjuncts skipped when oracle is nil; i non-nil
//	(1) oracle.BlockLogs(chainID, blockID).ok
//	(2) forall execMsg in BlockLogs.execMsgs:
//	    execMsg.chainID in CHAIN_IDS &&
//	    (InitMsgInFrontier(execMsg, blocks) || InitMsgInLogsDB(execMsg))
func CheckAllInitMsgsPresent(i *Interop, oracle ChainBlockOracle, chainID eth.ChainID, blockID eth.BlockID, blocks map[eth.ChainID]eth.BlockID) error {
	const pred = "Interop.dfy AllInitMsgsPresent"
	if oracle == nil {
		return nil // R5: skip oracle-dependent conjuncts
	}
	if i == nil {
		return violation(pred, "0", "Interop is nil")
	}
	logs, ok := oracle.BlockLogs(chainID, blockID)
	if !ok {
		return violation(pred, "1",
			"chain %s block %s: oracle has no BlockLogs", chainID, blockID)
	}
	var errs []error
	for idx, msg := range logs {
		frontierErr := CheckInitMsgInFrontier(i, oracle, msg, blocks)
		logsdbErr := CheckInitMsgInLogsDB(i, msg)
		if frontierErr != nil && logsdbErr != nil {
			errs = append(errs, violation(pred, "2",
				"execMsg[%d] chain %s block %d logIdx %d: not in frontier (%v) and not in logsDB (%v)",
				idx, msg.Chain, msg.BlockNum, msg.LogIdx, frontierErr, logsdbErr))
		}
	}
	return errors.Join(errs...)
}

// AssertAllInitMsgsPresent fails t when CheckAllInitMsgsPresent reports
// violations.
func AssertAllInitMsgsPresent(t dafnyT, i *Interop, oracle ChainBlockOracle, chainID eth.ChainID, blockID eth.BlockID, blocks map[eth.ChainID]eth.BlockID) {
	t.Helper()
	failOnViolation(t, CheckAllInitMsgsPresent(i, oracle, chainID, blockID, blocks))
}

// CheckBlockIsCrossValid mirrors BlockIsCrossValid(ts, chainID, blockID) in
// op-supernode/dafny-models/Interop.dfy: every executing message in the
// oracle's BlockLogs for (chainID, blockID) satisfies
// ValidExecutingMessage(ts, chainID, execMsg). Requires oracle; skips without
// one (R5).
// Conjuncts:
//
//	(0) oracle-dependent conjuncts skipped when oracle is nil; i non-nil
//	(1) oracle.BlockLogs(chainID, blockID).ok
//	(2) forall execMsg in BlockLogs.execMsgs:
//	    ValidExecutingMessage(ts, chainID, execMsg)
func CheckBlockIsCrossValid(i *Interop, oracle ChainBlockOracle, ts uint64, chainID eth.ChainID, blockID eth.BlockID) error {
	const pred = "Interop.dfy BlockIsCrossValid"
	if oracle == nil {
		return nil // R5: skip oracle-dependent conjuncts
	}
	if i == nil {
		return violation(pred, "0", "Interop is nil")
	}
	logs, ok := oracle.BlockLogs(chainID, blockID)
	if !ok {
		return violation(pred, "1",
			"chain %s block %s: oracle has no BlockLogs", chainID, blockID)
	}
	p := modelParamsFromInterop(i)
	var errs []error
	for idx, msg := range logs {
		if err := CheckValidExecutingMessage(p, oracle, ts, chainID, msg); err != nil {
			errs = append(errs, fmt.Errorf("%s conjunct (2): execMsg[%d]: %w", pred, idx, err))
		}
	}
	return errors.Join(errs...)
}

// AssertBlockIsCrossValid fails t when CheckBlockIsCrossValid reports
// violations.
func AssertBlockIsCrossValid(t dafnyT, i *Interop, oracle ChainBlockOracle, ts uint64, chainID eth.ChainID, blockID eth.BlockID) {
	t.Helper()
	failOnViolation(t, CheckBlockIsCrossValid(i, oracle, ts, chainID, blockID))
}
