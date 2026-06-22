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
	type containsMock struct {
		LogsDB
		seal   suptypes.BlockSeal
		found  bool
		findDB *sealsMockLogsDB
	}

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
