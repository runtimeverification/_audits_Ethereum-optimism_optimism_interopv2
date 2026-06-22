package interop

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// stubOracle is a simple ChainBlockOracle for tests.
// blockInfos and blockLogs are keyed by chainID+blockID.
type stubOracle struct {
	blockInfos map[eth.ChainID]map[eth.BlockID]eth.BlockInfo
	blockLogs  map[eth.ChainID]map[eth.BlockID][]ExecMsg
	blockTimes map[eth.ChainID]uint64
}

func newStubOracle() *stubOracle {
	return &stubOracle{
		blockInfos: make(map[eth.ChainID]map[eth.BlockID]eth.BlockInfo),
		blockLogs:  make(map[eth.ChainID]map[eth.BlockID][]ExecMsg),
		blockTimes: make(map[eth.ChainID]uint64),
	}
}

func (o *stubOracle) BlockInfo(chainID eth.ChainID, blockID eth.BlockID) (eth.BlockInfo, bool) {
	m, ok := o.blockInfos[chainID]
	if !ok {
		return nil, false
	}
	info, ok := m[blockID]
	return info, ok
}

func (o *stubOracle) BlockLogs(chainID eth.ChainID, blockID eth.BlockID) ([]ExecMsg, bool) {
	m, ok := o.blockLogs[chainID]
	if !ok {
		return nil, false
	}
	logs, ok := m[blockID]
	return logs, ok
}

func (o *stubOracle) BlockTime(chainID eth.ChainID) (uint64, bool) {
	bt, ok := o.blockTimes[chainID]
	return bt, ok
}

func (o *stubOracle) setBlockInfo(chainID eth.ChainID, blockID eth.BlockID, info eth.BlockInfo) {
	if o.blockInfos[chainID] == nil {
		o.blockInfos[chainID] = make(map[eth.BlockID]eth.BlockInfo)
	}
	o.blockInfos[chainID][blockID] = info
}

func (o *stubOracle) setBlockLogs(chainID eth.ChainID, blockID eth.BlockID, msgs []ExecMsg) {
	if o.blockLogs[chainID] == nil {
		o.blockLogs[chainID] = make(map[eth.BlockID][]ExecMsg)
	}
	o.blockLogs[chainID][blockID] = msgs
}

// oracleBlockInfo builds a *mockBlockInfo for oracle use.
func oracleBlockInfo(id eth.BlockID, timestamp uint64) *mockBlockInfo {
	return &mockBlockInfo{hash: id.Hash, number: id.Number, timestamp: timestamp}
}

// TestCheckValidExecutingMessage covers all conjuncts.
func TestCheckValidExecutingMessage(t *testing.T) {
	t.Parallel()

	p := dafnyTestParams() // activation 1000, expiry defaultMessageExpiryWindow
	oracle := newStubOracle()
	oracle.blockTimes[dafnyChainID(1)] = 2
	oracle.blockTimes[dafnyChainID(2)] = 2

	execChain := dafnyChainID(1)
	initChain := dafnyChainID(2)
	// execTS=1100, initTS=1050: well within constraints
	m := ExecMsg{Chain: initChain, BlockNum: 10, LogIdx: 0, Timestamp: 1050}

	t.Run("pass", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckValidExecutingMessage(p, oracle, 1100, execChain, m))
	})

	t.Run("nil oracle skips block-time conjuncts", func(t *testing.T) {
		t.Parallel()
		// Without oracle, conjuncts (1) and (2) are skipped; (3)+(4) still checked.
		require.NoError(t, CheckValidExecutingMessage(p, nil, 1100, execChain, m))
	})

	t.Run("per-chain BlockTime missing skips that conjunct (R5)", func(t *testing.T) {
		t.Parallel()
		// Oracle present but BlockTime unavailable for execChain and initChain:
		// conjuncts (1) and (2) are skipped without error; (3)+(4) still enforced.
		noBlockTime := newStubOracle() // blockTimes map is empty
		require.NoError(t, CheckValidExecutingMessage(p, noBlockTime, 1100, execChain, m))
	})

	t.Run("conjunct 0: execChain not in CHAIN_IDS", func(t *testing.T) {
		t.Parallel()
		err := CheckValidExecutingMessage(p, oracle, 1100, dafnyChainID(99), m)
		require.ErrorContains(t, err, "conjunct (0)")
	})

	t.Run("conjunct 0: initChain not in CHAIN_IDS", func(t *testing.T) {
		t.Parallel()
		bad := ExecMsg{Chain: dafnyChainID(99), BlockNum: 10, LogIdx: 0, Timestamp: 1050}
		err := CheckValidExecutingMessage(p, oracle, 1100, execChain, bad)
		require.ErrorContains(t, err, "conjunct (0)")
	})

	t.Run("conjunct 1: execTimestamp too early", func(t *testing.T) {
		t.Parallel()
		// activationTimestamp(1000) + blockTime(2) = 1002; execTS=1001 violates
		err := CheckValidExecutingMessage(p, oracle, 1001, execChain, m)
		require.ErrorContains(t, err, "conjunct (1)")
	})

	t.Run("conjunct 2: initTimestamp too early", func(t *testing.T) {
		t.Parallel()
		// initTS=1001 is fine for initBT=2 (1000+2=1002... actually 1001 < 1002: violation)
		early := ExecMsg{Chain: initChain, BlockNum: 10, LogIdx: 0, Timestamp: 1001}
		err := CheckValidExecutingMessage(p, oracle, 1100, execChain, early)
		require.ErrorContains(t, err, "conjunct (2)")
	})

	t.Run("conjunct 3: initTimestamp > execTimestamp", func(t *testing.T) {
		t.Parallel()
		future := ExecMsg{Chain: initChain, BlockNum: 10, LogIdx: 0, Timestamp: 1200}
		err := CheckValidExecutingMessage(p, oracle, 1100, execChain, future)
		require.ErrorContains(t, err, "conjunct (3)")
	})

	t.Run("conjunct 4: message expired", func(t *testing.T) {
		t.Parallel()
		// initTS=1050, expiry=604800; execTS = 1050+604800+1 = 605851
		expired := ExecMsg{Chain: initChain, BlockNum: 10, LogIdx: 0, Timestamp: 1050}
		err := CheckValidExecutingMessage(p, oracle, 1050+defaultMessageExpiryWindow+1, execChain, expired)
		require.ErrorContains(t, err, "conjunct (4)")
	})
}

// TestCheckInitMsgInLogsDB covers pass and violation.
func TestCheckInitMsgInLogsDB(t *testing.T) {
	t.Parallel()

	chain := dafnyChainID(1)
	seal := dafnySeal(10, 1050)
	msg := ExecMsg{Chain: chain, BlockNum: 10, LogIdx: 0, Timestamp: 1050, Checksum: suptypes.MessageChecksum(seal.Hash)}

	// Build a mock that returns the seal for the ContainsQuery matching msg.
	// containsModel returns true iff Contains returns (_, nil).
	// We use a custom mock that satisfies Contains for exactly this query.
	makeContainsMock := func(found bool) LogsDB {
		base := dafnySealedMock(seal)
		base.openExecMsg[10] = map[uint32]*suptypes.ExecutingMessage{}
		return &containsMockLogsDB{sealsMockLogsDB: base, found: found, seal: seal}
	}

	t.Run("pass: message present", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		i.logsDBs[chain] = makeContainsMock(true)
		require.NoError(t, CheckInitMsgInLogsDB(i, msg))
	})

	t.Run("conjunct 1: message absent", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		i.logsDBs[chain] = makeContainsMock(false)
		err := CheckInitMsgInLogsDB(i, msg)
		require.ErrorContains(t, err, "conjunct (1)")
	})

	t.Run("conjunct 0: nil Interop", func(t *testing.T) {
		t.Parallel()
		err := CheckInitMsgInLogsDB(nil, msg)
		require.ErrorContains(t, err, "conjunct (0)")
	})

	t.Run("conjunct 0: unknown chain", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		bad := ExecMsg{Chain: dafnyChainID(99)}
		err := CheckInitMsgInLogsDB(i, bad)
		require.ErrorContains(t, err, "conjunct (0)")
	})
}

// containsMockLogsDB wraps sealsMockLogsDB and overrides Contains.
type containsMockLogsDB struct {
	*sealsMockLogsDB
	found bool
	seal  suptypes.BlockSeal
}

func (m *containsMockLogsDB) Contains(_ suptypes.ContainsQuery) (suptypes.BlockSeal, error) {
	if m.found {
		return m.seal, nil
	}
	return suptypes.BlockSeal{}, suptypes.ErrConflict
}

// TestCheckLogsDBConsistentWithChainData covers nil-oracle skip, pass, and violations.
func TestCheckLogsDBConsistentWithChainData(t *testing.T) {
	t.Parallel()

	chain := dafnyChainID(1)
	blockID := eth.BlockID{Hash: common.Hash{0xd0, 5}, Number: 5}
	seal := suptypes.BlockSeal{Hash: blockID.Hash, Number: 5, Timestamp: 1000}

	t.Run("nil oracle skips (R5)", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		i.logsDBs[chain] = dafnySealedMock(seal)
		require.NoError(t, CheckLogsDBConsistentWithChainData(i, nil, chain))
	})

	t.Run("pass: empty logsDB", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		i := dafnyTestInterop(t)
		require.NoError(t, CheckLogsDBConsistentWithChainData(i, oracle, chain))
	})

	t.Run("pass: seal timestamp matches oracle", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		oracle.setBlockInfo(chain, blockID, oracleBlockInfo(blockID, 1000))
		oracle.setBlockLogs(chain, blockID, nil)
		i := dafnyTestInterop(t)
		i.logsDBs[chain] = dafnySealedMock(seal)
		require.NoError(t, CheckLogsDBConsistentWithChainData(i, oracle, chain))
	})

	t.Run("conjunct 1: oracle has no BlockInfo", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		// No BlockInfo for this block
		oracle.setBlockLogs(chain, blockID, nil)
		i := dafnyTestInterop(t)
		i.logsDBs[chain] = dafnySealedMock(seal)
		err := CheckLogsDBConsistentWithChainData(i, oracle, chain)
		require.ErrorContains(t, err, "conjunct (1)")
	})

	t.Run("conjunct 1: timestamp mismatch", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		oracle.setBlockInfo(chain, blockID, oracleBlockInfo(blockID, 9999))
		oracle.setBlockLogs(chain, blockID, nil)
		i := dafnyTestInterop(t)
		i.logsDBs[chain] = dafnySealedMock(seal)
		err := CheckLogsDBConsistentWithChainData(i, oracle, chain)
		require.ErrorContains(t, err, "conjunct (1)")
		require.ErrorContains(t, err, "9999")
	})

	t.Run("conjunct 2: oracle has no BlockLogs", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		oracle.setBlockInfo(chain, blockID, oracleBlockInfo(blockID, 1000))
		// No BlockLogs
		i := dafnyTestInterop(t)
		i.logsDBs[chain] = dafnySealedMock(seal)
		err := CheckLogsDBConsistentWithChainData(i, oracle, chain)
		require.ErrorContains(t, err, "conjunct (2)")
	})

	t.Run("conjunct 2: exec msg count mismatch", func(t *testing.T) {
		t.Parallel()
		// logsDB.OpenBlock returns 1 exec msg; oracle has 0 — divergence.
		oracle := newStubOracle()
		oracle.setBlockInfo(chain, blockID, oracleBlockInfo(blockID, 1000))
		oracle.setBlockLogs(chain, blockID, nil) // 0 oracle msgs
		m := dafnySealedMock(seal)
		execMsg := &suptypes.ExecutingMessage{BlockNum: 5}
		m.openExecMsg[seal.Number] = map[uint32]*suptypes.ExecutingMessage{0: execMsg} // 1 logsDB msg
		i := dafnyTestInterop(t)
		i.logsDBs[chain] = m
		err := CheckLogsDBConsistentWithChainData(i, oracle, chain)
		require.ErrorContains(t, err, "conjunct (2)")
	})

	t.Run("conjunct 0: nil Interop", func(t *testing.T) {
		t.Parallel()
		err := CheckLogsDBConsistentWithChainData(nil, newStubOracle(), chain)
		require.ErrorContains(t, err, "conjunct (0)")
	})

	t.Run("conjunct 0: unknown chain", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		err := CheckLogsDBConsistentWithChainData(i, newStubOracle(), dafnyChainID(99))
		require.ErrorContains(t, err, "conjunct (0)")
	})
}

// TestCheckAllLogsDBsConsistentWithChainData covers the forall wrapper.
func TestCheckAllLogsDBsConsistentWithChainData(t *testing.T) {
	t.Parallel()

	t.Run("nil oracle skips (R5)", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		require.NoError(t, CheckAllLogsDBsConsistentWithChainData(i, nil))
	})

	t.Run("pass: all chains consistent", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		i := dafnyTestInterop(t)
		require.NoError(t, CheckAllLogsDBsConsistentWithChainData(i, oracle))
	})

	t.Run("violation: one chain fails", func(t *testing.T) {
		t.Parallel()
		chain := dafnyChainID(1)
		blockID := eth.BlockID{Hash: common.Hash{0xd0, 5}, Number: 5}
		seal := suptypes.BlockSeal{Hash: blockID.Hash, Number: 5, Timestamp: 1000}

		oracle := newStubOracle()
		// BlockInfo timestamp mismatch for chain 1
		oracle.setBlockInfo(chain, blockID, oracleBlockInfo(blockID, 9999))
		oracle.setBlockLogs(chain, blockID, nil)

		i := dafnyTestInterop(t)
		i.logsDBs[chain] = dafnySealedMock(seal)

		err := CheckAllLogsDBsConsistentWithChainData(i, oracle)
		require.Error(t, err)
		require.ErrorContains(t, err, "conjunct (1)")
	})
}

// TestCheckBlockSealsMatchOnChainTimestamps covers the nil-oracle skip, pass, and violation.
func TestCheckBlockSealsMatchOnChainTimestamps(t *testing.T) {
	t.Parallel()

	chain := dafnyChainID(1)
	blockID := eth.BlockID{Hash: common.Hash{0xd0, 5}, Number: 5}
	seal := suptypes.BlockSeal{Hash: blockID.Hash, Number: 5, Timestamp: 1000}

	t.Run("nil oracle skips (R5)", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		i.logsDBs[chain] = dafnySealedMock(seal)
		require.NoError(t, CheckBlockSealsMatchOnChainTimestamps(i, nil))
	})

	t.Run("pass: empty logsDBs", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		i := dafnyTestInterop(t)
		require.NoError(t, CheckBlockSealsMatchOnChainTimestamps(i, oracle))
	})

	t.Run("pass: timestamp matches oracle", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		oracle.setBlockInfo(chain, blockID, oracleBlockInfo(blockID, 1000))
		i := dafnyTestInterop(t)
		i.logsDBs[chain] = dafnySealedMock(seal)
		require.NoError(t, CheckBlockSealsMatchOnChainTimestamps(i, oracle))
	})

	t.Run("conjunct 1: timestamp mismatch", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		oracle.setBlockInfo(chain, blockID, oracleBlockInfo(blockID, 9999))
		i := dafnyTestInterop(t)
		i.logsDBs[chain] = dafnySealedMock(seal)
		err := CheckBlockSealsMatchOnChainTimestamps(i, oracle)
		require.ErrorContains(t, err, "conjunct (1)")
		require.ErrorContains(t, err, "9999")
	})

	t.Run("conjunct 1: oracle has no BlockInfo", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle() // empty oracle
		i := dafnyTestInterop(t)
		i.logsDBs[chain] = dafnySealedMock(seal)
		err := CheckBlockSealsMatchOnChainTimestamps(i, oracle)
		require.ErrorContains(t, err, "conjunct (1)")
	})
}

// TestCheckAllVerifiedHeadsBoundedByTimestamp covers nil-oracle skip, pass, and violation.
func TestCheckAllVerifiedHeadsBoundedByTimestamp(t *testing.T) {
	t.Parallel()

	chain1 := dafnyChainID(1)
	chain2 := dafnyChainID(2)
	blockID1 := eth.BlockID{Hash: common.HexToHash("0xaa"), Number: 100}

	t.Run("nil oracle skips (R5)", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		require.NoError(t, CheckAllVerifiedHeadsBoundedByTimestamp(i, nil))
	})

	t.Run("pass: empty verifiedDB", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		i := dafnyTestInterop(t)
		require.NoError(t, CheckAllVerifiedHeadsBoundedByTimestamp(i, oracle))
	})

	t.Run("pass: heads bounded by their timestamp", func(t *testing.T) {
		t.Parallel()
		// dafnySyncedInterop commits ts 1000..1002 with chain1 heads 100..102
		// and chain2 heads 200..202; populate oracle for all six blockIDs.
		oracle := newStubOracle()
		hash := blockID1.Hash // all heads share the same hash in dafnySyncedInterop
		for off := uint64(0); off <= 2; off++ {
			id1 := eth.BlockID{Hash: hash, Number: 100 + off}
			id2 := eth.BlockID{Hash: hash, Number: 200 + off}
			oracle.setBlockInfo(chain1, id1, oracleBlockInfo(id1, 1000+off))
			oracle.setBlockInfo(chain2, id2, oracleBlockInfo(id2, 1000+off))
		}
		i := dafnySyncedInterop(t)
		require.NoError(t, CheckAllVerifiedHeadsBoundedByTimestamp(i, oracle))
	})

	t.Run("conjunct 1: on-chain timestamp > verified ts", func(t *testing.T) {
		t.Parallel()
		// Populate all entries for chain2 (valid) and a single ts 1000 entry for
		// chain1 with an on-chain timestamp of 2000 (> 1000: violation).
		// Remaining chain1 entries are missing from oracle, so they also report
		// conjunct (1) "no BlockInfo" — but the 2000 violation is present too.
		oracle := newStubOracle()
		hash := blockID1.Hash
		oracle.setBlockInfo(chain1, blockID1, oracleBlockInfo(blockID1, 2000))
		for off := uint64(0); off <= 2; off++ {
			id2 := eth.BlockID{Hash: hash, Number: 200 + off}
			oracle.setBlockInfo(chain2, id2, oracleBlockInfo(id2, 1000+off))
		}
		i := dafnySyncedInterop(t)
		err := CheckAllVerifiedHeadsBoundedByTimestamp(i, oracle)
		require.ErrorContains(t, err, "conjunct (1)")
		require.ErrorContains(t, err, "2000")
	})

	t.Run("conjunct 1: oracle missing BlockInfo", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle() // empty oracle
		i := dafnySyncedInterop(t)
		err := CheckAllVerifiedHeadsBoundedByTimestamp(i, oracle)
		require.ErrorContains(t, err, "conjunct (1)")
	})
}

// TestCrossValidityAssertWrappers verifies the Assert* wrappers pass/fail correctly.
func TestCrossValidityAssertWrappers(t *testing.T) {
	t.Parallel()

	p := dafnyTestParams()
	execChain := dafnyChainID(1)
	m := ExecMsg{Chain: dafnyChainID(2), BlockNum: 10, LogIdx: 0, Timestamp: 1050}

	t.Run("AssertValidExecutingMessage pass", func(t *testing.T) {
		t.Parallel()
		ft := &fakeDafnyT{}
		AssertValidExecutingMessage(ft, p, nil, 1100, execChain, m)
		require.True(t, ft.helperCalled)
		require.False(t, ft.failNowCalled)
	})

	t.Run("AssertValidExecutingMessage fail", func(t *testing.T) {
		t.Parallel()
		ft := &fakeDafnyT{}
		// initTS > execTS: conjunct (3) violation
		bad := ExecMsg{Chain: dafnyChainID(2), BlockNum: 10, LogIdx: 0, Timestamp: 2000}
		AssertValidExecutingMessage(ft, p, nil, 1100, execChain, bad)
		require.True(t, ft.errorfCalled)
		require.True(t, ft.failNowCalled)
	})

	t.Run("AssertBlockSealsMatchOnChainTimestamps nil-oracle pass", func(t *testing.T) {
		t.Parallel()
		ft := &fakeDafnyT{}
		i := dafnyTestInterop(t)
		AssertBlockSealsMatchOnChainTimestamps(ft, i, nil)
		require.False(t, ft.failNowCalled)
	})

	t.Run("AssertAllVerifiedHeadsBoundedByTimestamp nil-oracle pass", func(t *testing.T) {
		t.Parallel()
		ft := &fakeDafnyT{}
		i := dafnyTestInterop(t)
		AssertAllVerifiedHeadsBoundedByTimestamp(ft, i, nil)
		require.False(t, ft.failNowCalled)
	})
}

// ---- T12: helpers shared by BlocksExistedOnChain / FrontierBlocks / InitMsgInFrontier tests ----

// frontierBlockID returns a test blockID for a given chain index and block number.
func frontierBlockID(chainIdx, number uint64) eth.BlockID {
	return eth.BlockID{Hash: common.Hash{byte(chainIdx), byte(number)}, Number: number}
}

// frontierBlocks builds a blocks map for chains 1 and 2 with the given block numbers.
func frontierBlocks(num1, num2 uint64) map[eth.ChainID]eth.BlockID {
	return map[eth.ChainID]eth.BlockID{
		dafnyChainID(1): frontierBlockID(1, num1),
		dafnyChainID(2): frontierBlockID(2, num2),
	}
}

// populateFrontierOracle fills the oracle with BlockInfo and BlockLogs entries
// for both chains at the given block numbers and timestamp.
func populateFrontierOracle(oracle *stubOracle, num1, num2, ts uint64) {
	id1 := frontierBlockID(1, num1)
	id2 := frontierBlockID(2, num2)
	oracle.setBlockInfo(dafnyChainID(1), id1, oracleBlockInfo(id1, ts))
	oracle.setBlockInfo(dafnyChainID(2), id2, oracleBlockInfo(id2, ts))
	oracle.setBlockLogs(dafnyChainID(1), id1, nil) // no exec msgs by default
	oracle.setBlockLogs(dafnyChainID(2), id2, nil)
}

// TestCheckBlocksExistedOnChain covers nil-oracle skip, pass, and violation.
func TestCheckBlocksExistedOnChain(t *testing.T) {
	t.Parallel()

	blocks := frontierBlocks(10, 20)

	t.Run("nil oracle skips (R5)", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		require.NoError(t, CheckBlocksExistedOnChain(i, nil, blocks))
	})

	t.Run("pass: all blocks in oracle", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		populateFrontierOracle(oracle, 10, 20, 1050)
		i := dafnyTestInterop(t)
		require.NoError(t, CheckBlocksExistedOnChain(i, oracle, blocks))
	})

	t.Run("conjunct 1: block missing from oracle", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		// Only chain 2 has a BlockInfo entry; chain 1 is missing.
		id2 := frontierBlockID(2, 20)
		oracle.setBlockInfo(dafnyChainID(2), id2, oracleBlockInfo(id2, 1050))
		i := dafnyTestInterop(t)
		err := CheckBlocksExistedOnChain(i, oracle, blocks)
		require.ErrorContains(t, err, "conjunct (1)")
		require.ErrorContains(t, err, dafnyChainID(1).String())
	})

	t.Run("pass: empty blocks map", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		require.NoError(t, CheckBlocksExistedOnChain(i, newStubOracle(), nil))
	})
}

// TestCheckFrontierBlocksConsistentWithTimestamp covers nil-oracle skip, pass, and violations.
func TestCheckFrontierBlocksConsistentWithTimestamp(t *testing.T) {
	t.Parallel()

	blocks := frontierBlocks(10, 20)
	ts := uint64(1050)

	t.Run("nil oracle skips (R5)", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		require.NoError(t, CheckFrontierBlocksConsistentWithTimestamp(i, nil, ts, blocks))
	})

	t.Run("pass: all timestamps <= ts", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		populateFrontierOracle(oracle, 10, 20, ts)
		i := dafnyTestInterop(t)
		require.NoError(t, CheckFrontierBlocksConsistentWithTimestamp(i, oracle, ts, blocks))
	})

	t.Run("conjunct 1: timestamp > ts", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		id1 := frontierBlockID(1, 10)
		id2 := frontierBlockID(2, 20)
		oracle.setBlockInfo(dafnyChainID(1), id1, oracleBlockInfo(id1, 2000)) // > ts
		oracle.setBlockInfo(dafnyChainID(2), id2, oracleBlockInfo(id2, ts))
		i := dafnyTestInterop(t)
		err := CheckFrontierBlocksConsistentWithTimestamp(i, oracle, ts, blocks)
		require.ErrorContains(t, err, "conjunct (1)")
		require.ErrorContains(t, err, "2000")
	})

	t.Run("conjunct 1: block missing from oracle", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle() // empty oracle
		i := dafnyTestInterop(t)
		err := CheckFrontierBlocksConsistentWithTimestamp(i, oracle, ts, blocks)
		require.ErrorContains(t, err, "conjunct (1)")
	})
}

// TestCheckInitMsgInFrontier covers nil-oracle skip, pass, and per-conjunct violations.
func TestCheckInitMsgInFrontier(t *testing.T) {
	t.Parallel()

	chain := dafnyChainID(1)
	blockNum := uint64(10)
	blockTS := uint64(1050)
	blockID := frontierBlockID(1, blockNum)
	checksum := suptypes.MessageChecksum(common.Hash{0xaa})

	// frontierMsg builds an ExecMsg for chain 1 that matches the frontier block.
	frontierMsg := ExecMsg{Chain: chain, BlockNum: blockNum, LogIdx: 0, Timestamp: blockTS, Checksum: checksum}

	// blocks maps chain 1 to blockID, chain 2 to a different block.
	blocks := map[eth.ChainID]eth.BlockID{
		chain:           blockID,
		dafnyChainID(2): frontierBlockID(2, 20),
	}

	// makeOracle with the frontier block's log entry at logIdx 0.
	makeOracle := func() *stubOracle {
		oracle := newStubOracle()
		oracle.setBlockInfo(chain, blockID, oracleBlockInfo(blockID, blockTS))
		oracle.setBlockLogs(chain, blockID, []ExecMsg{frontierMsg})
		return oracle
	}

	t.Run("nil oracle skips (R5)", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		require.NoError(t, CheckInitMsgInFrontier(i, nil, frontierMsg, blocks))
	})

	t.Run("pass", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		require.NoError(t, CheckInitMsgInFrontier(i, makeOracle(), frontierMsg, blocks))
	})

	t.Run("conjunct 0: chain not in blocks", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		bad := ExecMsg{Chain: dafnyChainID(99), BlockNum: blockNum, LogIdx: 0, Timestamp: blockTS}
		err := CheckInitMsgInFrontier(i, makeOracle(), bad, blocks)
		require.ErrorContains(t, err, "conjunct (0)")
	})

	t.Run("conjunct 1: block number mismatch", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		bad := ExecMsg{Chain: chain, BlockNum: blockNum + 1, LogIdx: 0, Timestamp: blockTS}
		err := CheckInitMsgInFrontier(i, makeOracle(), bad, blocks)
		require.ErrorContains(t, err, "conjunct (1)")
	})

	t.Run("conjunct 2: oracle has no BlockInfo", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle() // no BlockInfo
		oracle.setBlockLogs(chain, blockID, []ExecMsg{frontierMsg})
		i := dafnyTestInterop(t)
		err := CheckInitMsgInFrontier(i, oracle, frontierMsg, blocks)
		require.ErrorContains(t, err, "conjunct (2)")
	})

	t.Run("conjunct 2: timestamp mismatch", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		oracle.setBlockInfo(chain, blockID, oracleBlockInfo(blockID, 9999)) // wrong timestamp
		oracle.setBlockLogs(chain, blockID, []ExecMsg{frontierMsg})
		i := dafnyTestInterop(t)
		err := CheckInitMsgInFrontier(i, oracle, frontierMsg, blocks)
		require.ErrorContains(t, err, "conjunct (2)")
		require.ErrorContains(t, err, "9999")
	})

	t.Run("conjunct 3: oracle has no BlockLogs", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		oracle.setBlockInfo(chain, blockID, oracleBlockInfo(blockID, blockTS))
		// no BlockLogs set
		i := dafnyTestInterop(t)
		err := CheckInitMsgInFrontier(i, oracle, frontierMsg, blocks)
		require.ErrorContains(t, err, "conjunct (3)")
	})

	t.Run("conjunct 3: logIdx out of range", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		oracle.setBlockInfo(chain, blockID, oracleBlockInfo(blockID, blockTS))
		oracle.setBlockLogs(chain, blockID, nil) // empty logs
		i := dafnyTestInterop(t)
		err := CheckInitMsgInFrontier(i, oracle, frontierMsg, blocks)
		require.ErrorContains(t, err, "conjunct (3)")
		require.ErrorContains(t, err, "out of range")
	})

	t.Run("conjunct 3: checksum mismatch", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		oracle.setBlockInfo(chain, blockID, oracleBlockInfo(blockID, blockTS))
		// Log at index 0 has a different checksum.
		wrongMsg := ExecMsg{Chain: chain, BlockNum: blockNum, LogIdx: 0, Timestamp: blockTS, Checksum: suptypes.MessageChecksum(common.Hash{0xbb})}
		oracle.setBlockLogs(chain, blockID, []ExecMsg{wrongMsg})
		i := dafnyTestInterop(t)
		err := CheckInitMsgInFrontier(i, oracle, frontierMsg, blocks)
		require.ErrorContains(t, err, "conjunct (3)")
		require.ErrorContains(t, err, "checksum mismatch")
	})
}

// TestCheckAllInitMsgsInLogsDB covers nil-oracle skip, pass, and violation.
func TestCheckAllInitMsgsInLogsDB(t *testing.T) {
	t.Parallel()

	chain := dafnyChainID(1)
	blockID := frontierBlockID(1, 10)
	seal := suptypes.BlockSeal{Hash: blockID.Hash, Number: 10, Timestamp: 1050}

	// execMsg that will be returned by the oracle and is present in the logsDB.
	execMsg := ExecMsg{Chain: chain, BlockNum: 10, LogIdx: 0, Timestamp: 1050, Checksum: suptypes.MessageChecksum(seal.Hash)}

	makeContainsMock := func(found bool) LogsDB {
		base := dafnySealedMock(seal)
		base.openExecMsg[10] = map[uint32]*suptypes.ExecutingMessage{}
		return &containsMockLogsDB{sealsMockLogsDB: base, found: found, seal: seal}
	}

	t.Run("nil oracle skips (R5)", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		require.NoError(t, CheckAllInitMsgsInLogsDB(i, nil, chain, blockID))
	})

	t.Run("pass: no exec msgs in block", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		oracle.setBlockLogs(chain, blockID, nil) // no exec msgs
		i := dafnyTestInterop(t)
		require.NoError(t, CheckAllInitMsgsInLogsDB(i, oracle, chain, blockID))
	})

	t.Run("pass: exec msg present in logsDB", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		oracle.setBlockLogs(chain, blockID, []ExecMsg{execMsg})
		i := dafnyTestInterop(t)
		i.logsDBs[chain] = makeContainsMock(true)
		require.NoError(t, CheckAllInitMsgsInLogsDB(i, oracle, chain, blockID))
	})

	t.Run("conjunct 1: oracle has no BlockLogs", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle() // no BlockLogs
		i := dafnyTestInterop(t)
		err := CheckAllInitMsgsInLogsDB(i, oracle, chain, blockID)
		require.ErrorContains(t, err, "conjunct (1)")
	})

	t.Run("conjunct 2: exec msg not in logsDB", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		oracle.setBlockLogs(chain, blockID, []ExecMsg{execMsg})
		i := dafnyTestInterop(t)
		i.logsDBs[chain] = makeContainsMock(false)
		err := CheckAllInitMsgsInLogsDB(i, oracle, chain, blockID)
		require.ErrorContains(t, err, "conjunct (2)")
	})

	t.Run("conjunct 0: nil Interop", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		oracle.setBlockLogs(chain, blockID, []ExecMsg{execMsg})
		err := CheckAllInitMsgsInLogsDB(nil, oracle, chain, blockID)
		require.ErrorContains(t, err, "conjunct (0)")
	})
}

// TestCheckAllInitMsgsPresent covers nil-oracle skip, pass, and violation.
func TestCheckAllInitMsgsPresent(t *testing.T) {
	t.Parallel()

	chain := dafnyChainID(1)
	blockNum := uint64(10)
	blockTS := uint64(1050)
	blockID := frontierBlockID(1, blockNum)
	checksum := suptypes.MessageChecksum(common.Hash{0xcc})

	// execMsg: initiating message on chain 1 at blockNum 10.
	execMsg := ExecMsg{Chain: chain, BlockNum: blockNum, LogIdx: 0, Timestamp: blockTS, Checksum: checksum}
	blocks := frontierBlocks(blockNum, 20)
	seal := suptypes.BlockSeal{Hash: blockID.Hash, Number: blockNum, Timestamp: blockTS}

	makeContainsMock := func(found bool) LogsDB {
		base := dafnySealedMock(seal)
		base.openExecMsg[blockNum] = map[uint32]*suptypes.ExecutingMessage{}
		return &containsMockLogsDB{sealsMockLogsDB: base, found: found, seal: seal}
	}

	// Oracle with frontier block info and the exec msg in its logs.
	makeFrontierOracle := func() *stubOracle {
		oracle := newStubOracle()
		oracle.setBlockInfo(chain, blockID, oracleBlockInfo(blockID, blockTS))
		oracle.setBlockLogs(chain, blockID, []ExecMsg{execMsg})
		return oracle
	}

	t.Run("nil oracle skips (R5)", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		require.NoError(t, CheckAllInitMsgsPresent(i, nil, chain, blockID, blocks))
	})

	t.Run("pass: no exec msgs", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		oracle.setBlockLogs(chain, blockID, nil)
		i := dafnyTestInterop(t)
		require.NoError(t, CheckAllInitMsgsPresent(i, oracle, chain, blockID, blocks))
	})

	t.Run("pass: msg in frontier", func(t *testing.T) {
		t.Parallel()
		oracle := makeFrontierOracle()
		i := dafnyTestInterop(t)
		// logsDB doesn't have it, but frontier does.
		i.logsDBs[chain] = makeContainsMock(false)
		require.NoError(t, CheckAllInitMsgsPresent(i, oracle, chain, blockID, blocks))
	})

	t.Run("pass: msg in logsDB", func(t *testing.T) {
		t.Parallel()
		// Oracle returns exec msg but with a different checksum (not in frontier).
		oracle := newStubOracle()
		wrongChecksum := ExecMsg{Chain: chain, BlockNum: blockNum, LogIdx: 0, Timestamp: blockTS, Checksum: suptypes.MessageChecksum(common.Hash{0xdd})}
		oracle.setBlockLogs(chain, blockID, []ExecMsg{wrongChecksum})
		oracle.setBlockInfo(chain, blockID, oracleBlockInfo(blockID, blockTS))
		i := dafnyTestInterop(t)
		// logsDB has execMsg (matching the wrong-checksum msg's chain/block/logIdx/ts).
		i.logsDBs[chain] = makeContainsMock(true)
		require.NoError(t, CheckAllInitMsgsPresent(i, oracle, chain, blockID, blocks))
	})

	t.Run("conjunct 1: oracle has no BlockLogs", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		i := dafnyTestInterop(t)
		err := CheckAllInitMsgsPresent(i, oracle, chain, blockID, blocks)
		require.ErrorContains(t, err, "conjunct (1)")
	})

	t.Run("conjunct 2: msg not in frontier and not in logsDB", func(t *testing.T) {
		t.Parallel()
		// Oracle has exec msg but blocks map points to a different block number
		// (so InitMsgInFrontier fails) and logsDB doesn't have it.
		wrongBlocks := map[eth.ChainID]eth.BlockID{
			chain:           frontierBlockID(1, 99), // wrong block number
			dafnyChainID(2): frontierBlockID(2, 20),
		}
		oracle := newStubOracle()
		oracle.setBlockLogs(chain, blockID, []ExecMsg{execMsg})
		oracle.setBlockInfo(chain, frontierBlockID(1, 99), oracleBlockInfo(frontierBlockID(1, 99), blockTS))
		i := dafnyTestInterop(t)
		i.logsDBs[chain] = makeContainsMock(false)
		err := CheckAllInitMsgsPresent(i, oracle, chain, blockID, wrongBlocks)
		require.ErrorContains(t, err, "conjunct (2)")
	})
}

// TestCheckBlockIsCrossValid covers nil-oracle skip, pass, and violations.
func TestCheckBlockIsCrossValid(t *testing.T) {
	t.Parallel()

	chain := dafnyChainID(1)
	blockNum := uint64(10)
	blockID := frontierBlockID(1, blockNum)
	ts := uint64(1100)

	// A valid executing message: initChain=2, initTS=1050 (within window, <= ts).
	validExecMsg := ExecMsg{
		Chain:     dafnyChainID(2),
		BlockNum:  5,
		LogIdx:    0,
		Timestamp: 1050, // initTS <= ts(1100), ts <= initTS+expiry
	}

	t.Run("nil oracle skips (R5)", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		require.NoError(t, CheckBlockIsCrossValid(i, nil, ts, chain, blockID))
	})

	t.Run("pass: no exec msgs", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		oracle.setBlockLogs(chain, blockID, nil)
		i := dafnyTestInterop(t)
		require.NoError(t, CheckBlockIsCrossValid(i, oracle, ts, chain, blockID))
	})

	t.Run("pass: valid exec msg", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		oracle.setBlockLogs(chain, blockID, []ExecMsg{validExecMsg})
		i := dafnyTestInterop(t)
		require.NoError(t, CheckBlockIsCrossValid(i, oracle, ts, chain, blockID))
	})

	t.Run("conjunct 1: oracle has no BlockLogs", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		i := dafnyTestInterop(t)
		err := CheckBlockIsCrossValid(i, oracle, ts, chain, blockID)
		require.ErrorContains(t, err, "conjunct (1)")
	})

	t.Run("conjunct 2: exec msg violates ValidExecutingMessage", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		// initTS > execTS: conjunct (3) of ValidExecutingMessage.
		badMsg := ExecMsg{Chain: dafnyChainID(2), BlockNum: 5, LogIdx: 0, Timestamp: 2000}
		oracle.setBlockLogs(chain, blockID, []ExecMsg{badMsg})
		i := dafnyTestInterop(t)
		err := CheckBlockIsCrossValid(i, oracle, ts, chain, blockID)
		require.ErrorContains(t, err, "conjunct (2)")
	})

	t.Run("conjunct 0: nil Interop", func(t *testing.T) {
		t.Parallel()
		oracle := newStubOracle()
		oracle.setBlockLogs(chain, blockID, nil)
		err := CheckBlockIsCrossValid(nil, oracle, ts, chain, blockID)
		require.ErrorContains(t, err, "conjunct (0)")
	})
}

// TestT12AssertWrappers checks the Assert* wrappers introduced in T12.
func TestT12AssertWrappers(t *testing.T) {
	t.Parallel()

	chain := dafnyChainID(1)
	blockID := frontierBlockID(1, 10)
	blocks := frontierBlocks(10, 20)
	ts := uint64(1050)

	t.Run("AssertBlocksExistedOnChain nil-oracle pass", func(t *testing.T) {
		t.Parallel()
		ft := &fakeDafnyT{}
		i := dafnyTestInterop(t)
		AssertBlocksExistedOnChain(ft, i, nil, blocks)
		require.False(t, ft.failNowCalled)
	})

	t.Run("AssertBlocksExistedOnChain violation", func(t *testing.T) {
		t.Parallel()
		ft := &fakeDafnyT{}
		i := dafnyTestInterop(t)
		AssertBlocksExistedOnChain(ft, i, newStubOracle(), blocks) // empty oracle
		require.True(t, ft.failNowCalled)
	})

	t.Run("AssertFrontierBlocksConsistentWithTimestamp nil-oracle pass", func(t *testing.T) {
		t.Parallel()
		ft := &fakeDafnyT{}
		i := dafnyTestInterop(t)
		AssertFrontierBlocksConsistentWithTimestamp(ft, i, nil, ts, blocks)
		require.False(t, ft.failNowCalled)
	})

	t.Run("AssertInitMsgInFrontier nil-oracle pass", func(t *testing.T) {
		t.Parallel()
		ft := &fakeDafnyT{}
		i := dafnyTestInterop(t)
		m := ExecMsg{Chain: chain, BlockNum: 10, LogIdx: 0, Timestamp: ts}
		AssertInitMsgInFrontier(ft, i, nil, m, blocks)
		require.False(t, ft.failNowCalled)
	})

	t.Run("AssertAllInitMsgsInLogsDB nil-oracle pass", func(t *testing.T) {
		t.Parallel()
		ft := &fakeDafnyT{}
		i := dafnyTestInterop(t)
		AssertAllInitMsgsInLogsDB(ft, i, nil, chain, blockID)
		require.False(t, ft.failNowCalled)
	})

	t.Run("AssertAllInitMsgsPresent nil-oracle pass", func(t *testing.T) {
		t.Parallel()
		ft := &fakeDafnyT{}
		i := dafnyTestInterop(t)
		AssertAllInitMsgsPresent(ft, i, nil, chain, blockID, blocks)
		require.False(t, ft.failNowCalled)
	})

	t.Run("AssertBlockIsCrossValid nil-oracle pass", func(t *testing.T) {
		t.Parallel()
		ft := &fakeDafnyT{}
		i := dafnyTestInterop(t)
		AssertBlockIsCrossValid(ft, i, nil, ts, chain, blockID)
		require.False(t, ft.failNowCalled)
	})
}
