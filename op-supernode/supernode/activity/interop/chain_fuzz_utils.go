package interop

import (
	"context"
	"math/rand"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	types2 "github.com/ethereum/go-ethereum/core/types"
	params2 "github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func (rc *RandomChain) ExecMsgForLog(chain eth.ChainID, block eth.L2BlockRef, log *types2.Log) *types2.Log {
	payloadHash := crypto.Keccak256Hash(types.LogToMessagePayload(log))

	msg := types.Message{
		Identifier: types.Identifier{
			Origin:      log.Address,
			BlockNumber: block.Number,
			LogIndex:    uint32(log.Index),
			Timestamp:   block.Time,
			ChainID:     chain,
		},
		PayloadHash: payloadHash,
	}
	topics, data := msg.EncodeEvent()
	return &types2.Log{
		Address: params2.InteropCrossL2InboxAddress,
		Data:    data,
		Topics:  topics,
		Index:   log.Index,
	}
}

type ChainBlock struct {
	chain eth.ChainID
	block *eth.L2BlockRef
}

// InvalidationKind identifies how a RandomChain was invalidated, so the harness
// can predict which chains the SUT should report as invalid.
type InvalidationKind int

const (
	KindNone InvalidationKind = iota
	KindCycle
	KindSelfDependency
	KindInvalidIdentifier
	KindFutureDependency
	KindExpiredMessage
)

func (k InvalidationKind) String() string {
	switch k {
	case KindNone:
		return "None"
	case KindCycle:
		return "Cycle"
	case KindSelfDependency:
		return "SelfDependency"
	case KindInvalidIdentifier:
		return "InvalidIdentifier"
	case KindFutureDependency:
		return "FutureDependency"
	case KindExpiredMessage:
		return "ExpiredMessage"
	default:
		return "Unknown"
	}
}

type RandomChainParams struct {
	chainCount int

	minLength int
	maxLength int

	maxBlockTimeExclusive int

	invalidateChance       int // Percentage [0-100]
	dependencyChance       int // Percentage [0-100]
}

type L1Assignments struct {
	L1Block  eth.BlockRef
	L2Blocks []*ChainBlock
}

type RandomChain struct {
	t             *testing.T
	randomGenerator *rand.Rand
	chainIDs      []eth.ChainID
	allBlocks     []ChainBlock
	cbIndices     map[*eth.L2BlockRef]int // Lookup for a ChainBlock's index in allBlocks
	generatedLogs map[ChainBlock][]*types2.Log
	dependencies  map[ChainBlock][]ChainBlock
	chainBlocks   map[eth.ChainID][]*eth.L2BlockRef
	l1SourceMap   map[ChainBlock]eth.BlockRef
	l1Source      map[uint64]eth.BlockRef
	receipts      map[eth.ChainID]map[eth.BlockID]types2.Receipts
	blockTimes    map[eth.ChainID]int
	isInvalid     bool

	// invalidationKind records which Invalidate sub-routine successfully ran.
	// KindNone (the zero value) means the chain is expected to verify cleanly.
	invalidationKind InvalidationKind

	// expectedInvalidChains is the set of chains the harness predicts the SUT
	// should report in result.InvalidHeads when verifying the timestamp at
	// which the invalidation lives. Populated only when invalidationKind != KindNone.
	expectedInvalidChains map[eth.ChainID]bool

	// invalidationTimestamp is the L2 timestamp where the injected invalidation
	// lives. Used by the harness to decide whether the SUT actually got far
	// enough to observe it.
	invalidationTimestamp uint64

	// messageExpiryWindow is the value the harness will pass to interop.New
	// for this run. 0 means "use the SUT's default" (defaultMessageExpiryWindow
	// = 604800s). InsertExpiredMessage sets this to a small value so the
	// targeted dep crosses the expiry boundary; other invalidation modes
	// leave it at 0 so random deps (which can naturally span hundreds of
	// seconds) never trip ErrMessageExpired accidentally.
	messageExpiryWindow uint64
}

var _ cc.InteropChain = RandomChainContainer{}

type RandomChainContainer struct {
	chainID            eth.ChainID
	randomChain        *RandomChain
}

func (c RandomChainContainer) ID() eth.ChainID                                  { return c.chainID }
func (c RandomChainContainer) Start(ctx context.Context) error                  { return nil }
func (c RandomChainContainer) Stop(ctx context.Context) error                   { return nil }
func (c RandomChainContainer) Pause(ctx context.Context) error                  { return nil }
func (c RandomChainContainer) Resume(ctx context.Context) error                 { return nil }
func (c RandomChainContainer) PauseAndStopVN(ctx context.Context) error         { return nil }
func (c RandomChainContainer) RegisterVerifier(v activity.VerificationActivity) {}
func (c RandomChainContainer) VerifierCurrentL1s() []eth.BlockID {
	return nil
}

func (c RandomChainContainer) LocalSafeBlockAtTimestamp(ctx context.Context, ts uint64) (eth.L2BlockRef, error) {
	var theblock *eth.L2BlockRef = nil;
	for _, block := range c.randomChain.chainBlocks[c.chainID] {
		if block.Time <= ts {
			theblock = block;
		} else {
			break
		}
	}
	if theblock == nil {
		return eth.L2BlockRef{}, ethereum.NotFound;
	}
	return *theblock, nil
}

func (c RandomChainContainer) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	blocks := c.randomChain.chainBlocks[c.chainID]
	block := blocks[len(blocks)-1]
	cb := ChainBlock{chain: c.chainID, block: block}
	l1Origin := c.randomChain.l1SourceMap[cb]
	// Upstream's resolveFirstVerifiableTimestamp requires both LocalSafeL2 and
	// SafeL2 to be present with non-zero Number — otherwise the round refuses
	// to start. The mock doesn't distinguish cross-safe from local-safe, so
	// report the chain's tip for both.
	return &eth.SyncStatus{
		CurrentL1:   l1Origin,
		LocalSafeL2: *block,
		SafeL2:      *block,
	}, nil
}

func (c RandomChainContainer) OptimisticAt(ctx context.Context, ts uint64) (l2, l1 eth.BlockID, err error) {
	//TODO
	block, err := c.LocalSafeBlockAtTimestamp(ctx, ts)
	if err != nil {
		return eth.BlockID{}, eth.BlockID{}, err
	}
	cb := ChainBlock{c.chainID, &block}
	l1 = c.randomChain.l1SourceMap[cb].ID()
	return block.ID(), l1, nil
}

func (c RandomChainContainer) TimestampToBlockNumber(ctx context.Context, ts uint64) (uint64, error) {
	//TODO
	block, err := c.LocalSafeBlockAtTimestamp(ctx, ts)
	if err != nil {
		return 0, err
	}
	return block.Number, nil
}

func (c RandomChainContainer) BlockNumberToTimestamp(ctx context.Context, blocknum uint64) (uint64, error) {
	//TODO
	for _, block := range c.randomChain.chainBlocks[c.chainID] {
		if block.Number == blocknum {
			return block.Time, nil
		}
	}
	return 0, ethereum.NotFound
}

func (c RandomChainContainer) OutputRootAtL2BlockHash(ctx context.Context, blockHash common.Hash) (eth.Bytes32, error) {
	//TODO
	return eth.Bytes32{}, nil
}

func (c RandomChainContainer) OptimisticOutputAtTimestamp(ctx context.Context, ts uint64) (*eth.OutputV0, error) {
	//TODO
	return nil, nil
}

func (c RandomChainContainer) GetDeniedOutput(height uint64, payloadHash common.Hash) (*eth.OutputV0, error) {
	//TODO
	return nil, nil
}

func (c RandomChainContainer) OutputV0AtBlockNumber(ctx context.Context, l2BlockNum uint64) (*eth.OutputV0, error) {
	// Upstream's newInvalidHead dereferences this without nil-checking, so we
	// must return a populated value whenever the block exists. StateRoot and
	// MessagePasserStorageRoot are left zero — the harness doesn't model them
	// and the SUT only compares BlockHash here.
	for _, block := range c.randomChain.chainBlocks[c.chainID] {
		if block.Number == l2BlockNum {
			return &eth.OutputV0{BlockHash: block.Hash}, nil
		}
	}
	return nil, ethereum.NotFound
}

func (c RandomChainContainer) HasDeniedAtOrAfterTimestamp(timestamp uint64) (bool, error) {
	//TODO
	return false, nil
}

func (c RandomChainContainer) RewindEngine(ctx context.Context, timestamp uint64, invalidatedBlock eth.BlockRef) error {
	//TODO?
	return nil
}

func (c RandomChainContainer) FetchReceipts(ctx context.Context, blockHash eth.BlockID) (eth.BlockInfo, types2.Receipts, error) {
	chainReceipts := c.randomChain.receipts[c.chainID];
	receipt := chainReceipts[blockHash];

	for _, block := range c.randomChain.chainBlocks[c.chainID] {
		if block.ID() == blockHash {
			header := &types2.Header{
				  ParentHash: block.ParentHash,
				  Number:     new(big.Int).SetUint64(block.Number),
				  Time:       block.Time,
			}
			return eth.HeaderBlockInfoTrusted(block.Hash, header), receipt, nil
		}
	}
	return nil, nil, ethereum.NotFound
}

func (c RandomChainContainer) BlockTime() uint64 {
	return uint64(c.randomChain.blockTimes[c.chainID])
}

func (c RandomChainContainer) InvalidateBlock(ctx context.Context, height uint64, payloadHash common.Hash, decisionTimestamp uint64, stateRoot, messagePasserStorageRoot eth.Bytes32) (bool, error) {
	//TODO
	return true, nil
}

func (c RandomChainContainer) PruneDeniedAtOrAfterTimestamp(timestamp uint64) (map[uint64][]common.Hash, error) {
	// TODO
	return nil, nil
}

func (c RandomChainContainer) IsDenied(height uint64, payloadHash common.Hash) (bool, error) {
	//TODO
	return false, nil
}

func (c RandomChainContainer) SetResetCallback(cb cc.ResetCallback) {
	//TODO
}

func (rc *RandomChain) GetContainers() (map[eth.ChainID]cc.InteropChain) {
	chains := make(map[eth.ChainID]cc.InteropChain);
	for _, chain := range rc.chainIDs {
		container := RandomChainContainer {
			chainID:     chain,
			randomChain: rc,
		}
		chains[chain] = container
	}
	return chains
}

// Merge all of the chains' blocks from separate arrays into one, ordered by timestamp
func MergeBlocks(chainBlocks map[eth.ChainID][]*eth.L2BlockRef) []ChainBlock {
	totalLength := 0
	for _, blocks := range chainBlocks {
		totalLength += len(blocks)
	}

	allBlocks := make([]ChainBlock, 0, totalLength)
	chainIndices := make(map[eth.ChainID]int)
	for range totalLength {
		var finalChain eth.ChainID
		var finalBlock *eth.L2BlockRef

		for chain := range chainBlocks {
			idx := chainIndices[chain]
			if idx < len(chainBlocks[chain]) {
				block := chainBlocks[chain][idx]
				if finalBlock == nil || block.Time < finalBlock.Time {
					finalChain = chain
					finalBlock = block
				}
			}
		}

		chainIndices[finalChain]++

		chainBlock := ChainBlock{
			chain: finalChain,
			block: finalBlock,
		}
		allBlocks = append(allBlocks, chainBlock)
	}

	return allBlocks
}

// Given the chains' blocks and blockTimes, find the chain for which its next block wont put
// the other chains' current blocks behind on the new timestamp
func NextValidChain(chainBlocks map[eth.ChainID][]*eth.L2BlockRef, blockTimes map[eth.ChainID]int) eth.ChainID {
	lastTimeStamp := make(map[eth.ChainID]uint64)
	for chain := range chainBlocks {
		blocks := chainBlocks[chain]
		lastTimeStamp[chain] = blocks[len(blocks)-1].Time
	}
	first := true
	var nextChain eth.ChainID
	v := uint64(0)
	for chain := range chainBlocks {
		nextTimeStamp := lastTimeStamp[chain] + uint64(blockTimes[chain])
		if first || nextTimeStamp < v {
			nextChain = chain
			v = nextTimeStamp
			first = false
		}
	}
	return nextChain
}

func SameTimeStampSets(blocks []ChainBlock) [][]ChainBlock {
	res := make([][]ChainBlock, 0)
	i := 0
	nextSet := make([]ChainBlock, 0)
	for i < len(blocks)-1 {
		if blocks[i].block.Time != blocks[i+1].block.Time {
			if len(nextSet) != 0 {
				nextSet = append(nextSet, blocks[i])
				res = append(res, nextSet)
				nextSet = make([]ChainBlock, 0)
			}
		} else {
			nextSet = append(nextSet, blocks[i])
		}
		i++
	}
	if len(nextSet) != 0 {
		nextSet = append(nextSet, blocks[i])
		res = append(res, nextSet)
	}
	return res
}

func (p *RandomChainParams) MakeRandomChain(t *testing.T, seed int64) (res RandomChain) {
	r := rand.New(rand.NewSource(seed))

	totalLength := randomInRange(r, p.minLength, p.maxLength) + 2

	res = RandomChain{
		t:             t,
		randomGenerator: r,
		chainIDs:      make([]eth.ChainID, 0, p.chainCount),
		allBlocks:     make([]ChainBlock, 0, totalLength),
		cbIndices:     make(map[*eth.L2BlockRef]int),
		generatedLogs: make(map[ChainBlock][]*types2.Log),
		dependencies:  make(map[ChainBlock][]ChainBlock),
		chainBlocks:   make(map[eth.ChainID][]*eth.L2BlockRef),
		l1SourceMap:   make(map[ChainBlock]eth.BlockRef),
		l1Source:      make(map[uint64]eth.BlockRef),
		receipts:              make(map[eth.ChainID]map[eth.BlockID]types2.Receipts),
		blockTimes:            make(map[eth.ChainID]int),
		isInvalid:             false,
		invalidationKind:      KindNone,
		expectedInvalidChains: make(map[eth.ChainID]bool),
		messageExpiryWindow:   0, // 0 → SUT default; only set non-zero in KindExpiredMessage
	}

	for i := range p.chainCount {
		chain := eth.ChainIDFromUInt64(uint64(i))
		res.chainBlocks[chain] = make([]*eth.L2BlockRef, 0)
		res.blockTimes[chain] = randomInRange(r, 1, p.maxBlockTimeExclusive)
		res.chainIDs = append(res.chainIDs, chain)
		res.receipts[chain] = make(map[eth.BlockID]types2.Receipts)
	}

	//
	// Create array of all blocks
	//

	// First, guarantee that each chain contains at least one block
	for _, chain := range res.chainIDs {
		block := eth.L2BlockRef{}
		block.Hash = testutils.RandomHash(r)
		res.chainBlocks[chain] = append(res.chainBlocks[chain], &block)
	}

	// Then, generate the rest of the blocks.
	for range totalLength - p.chainCount {
		// Select the chain with the next valid timestamp
		nextChain := NextValidChain(res.chainBlocks, res.blockTimes)

		// Add a random block to it
		lastBlock := res.chainBlocks[nextChain][len(res.chainBlocks[nextChain])-1]
		block := testutils.NextRandomL2Ref(r, uint64(res.blockTimes[nextChain]), *lastBlock, eth.BlockID{})
		t.Logf("Adding block: chain=%s, timestamp=%d", nextChain.String(), block.Time)
		res.chainBlocks[nextChain] = append(res.chainBlocks[nextChain], &block)
		res.addRandomLog(ChainBlock{nextChain, &block})
	}

	// Populate res.allBlocks
	res.allBlocks = MergeBlocks(res.chainBlocks)
	for i, cb := range res.allBlocks {
		res.cbIndices[cb.block] = i
	}

	// Decide invalidation kind UP FRONT (was previously inside Invalidate()).
	// We need this before generating random dependencies because the
	// KindExpiredMessage case must run with a tiny messageExpiryWindow, and
	// random deps that span hundreds of seconds would also trip it — so we
	// suppress random deps for expired-message seeds and inject only one
	// expired dep manually.
	plannedKind := KindNone
	if r.Intn(100) < p.invalidateChance {
		// 5 invalidation modes (1..5). KindNone (0) is reserved for "no
		// invalidation requested" and is not part of the random draw.
		plannedKind = InvalidationKind(1 + r.Intn(5))
	}

	//
	// Create random dependencies between all blocks (skipped when fuzzing
	// expired-message — see comment above).
	//
	if plannedKind != KindExpiredMessage {
		for initIndex, initcb := range res.allBlocks {
			block := initcb.block
			if block.Number == 0 {
				continue
			}

			for r.Intn(100) < p.dependencyChance {
				execIndex := randomInRange(r, initIndex, totalLength)
				execcb := res.allBlocks[execIndex]
				initiatingLog := res.addRandomLog(initcb)
				res.addExecutingMessageWithDependency(execcb, initcb, initiatingLog)
			}
		}
	}

	// Apply the planned invalidation. Each sub-routine self-records via
	// res.invalidationKind on success; bailouts (CreateCycle with no
	// same-timestamp set, InsertExpiredMessage with too-short chain) leave
	// the chain valid.
	switch plannedKind {
	case KindCycle:
		res.CreateCycle()
	case KindSelfDependency:
		res.InsertSelfDependency()
	case KindInvalidIdentifier:
		res.InsertMessageWithInvalidIdentifier()
	case KindFutureDependency:
		res.InsertFutureDependency()
	case KindExpiredMessage:
		// Small window so the targeted gap exceeds it. The chain currently
		// has no random deps for this case (above), so this is the only
		// dep with a multi-step time gap.
		res.InsertExpiredMessage(10)
	}
	res.isInvalid = res.invalidationKind != KindNone

	//
	// Make L1 derivation info
	//
	taken := 0
	nextL1 := testutils.RandomBlockRef(r)
	for taken < totalLength {
		nextL1 = testutils.NextRandomRef(r, nextL1)
		take := randomInRange(r, 1, 5) // Take 1-4 L2 blocks
		take = min(totalLength-taken, take)
		for _, l2Block := range res.allBlocks[taken : taken+take] {
			res.l1SourceMap[l2Block] = nextL1
		}
		res.l1Source[nextL1.Number] = nextL1
		taken += take
	}

	res.GenerateReceiptsFromLogs()

	return res
}

func TestMakeRandomChain(t *testing.T) {
	params := RandomChainParams {
		chainCount:             3,
		minLength:              5,
		maxLength:              20,
		invalidateChance:       70,
		dependencyChance:       8,
	}

	chain := params.MakeRandomChain(t, 0)

	t.Run("Correct number of chains", func(t *testing.T) {
		require.Equal(t, params.chainCount, len(chain.chainIDs))
	})
}

var _ l1ByNumberSource = RandomChain{}

func (rc RandomChain) L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error) {
	return rc.l1Source[num], nil
}

func (rc *RandomChain) addRandomLog(initcb ChainBlock) *types2.Log {
	initiatingLog := testutils.RandomLog(rc.randomGenerator)
	initiatingLog.Index = uint(len(rc.generatedLogs[initcb]))
	rc.generatedLogs[initcb] = append(rc.generatedLogs[initcb], initiatingLog)
	return initiatingLog
}

func (rc *RandomChain) addExecutingMessage(execcb ChainBlock, initcb ChainBlock, initiatingLog *types2.Log) {
	execLog := rc.ExecMsgForLog(initcb.chain, *initcb.block, initiatingLog)
	execLog.Index = uint(len(rc.generatedLogs[execcb]))
	rc.generatedLogs[execcb] = append(rc.generatedLogs[execcb], execLog)
}

func (rc *RandomChain) addExecutingMessageWithDependency(execcb ChainBlock, initcb ChainBlock, initiatingLog *types2.Log) {
	rc.addExecutingMessage(execcb, initcb, initiatingLog)
	rc.dependencies[execcb] = append(rc.dependencies[execcb], initcb)
}

func (rc *RandomChain) addInvalidExecutingMessage(execcb ChainBlock, initcb ChainBlock, initiatingLog *types2.Log) {
	execLog := rc.InvalidExecMsgForLog(initcb.chain, *initcb.block, initiatingLog)
	execLog.Index = uint(len(rc.generatedLogs[execcb]))
	rc.generatedLogs[execcb] = append(rc.generatedLogs[execcb], execLog)
}

func (rc *RandomChain) GenerateReceiptsFromLogs() {
	for _, cb := range rc.allBlocks {
		chainid, block := cb.chain, cb.block
		logs := rc.generatedLogs[cb]
		rcpt := types2.Receipt{
			Logs: logs,
		}
		rc.receipts[chainid][block.ID()] = types2.Receipts{&rcpt};
	}
}

// Returns a random integer in the interval [lowerIncluding, upperExcluding)
func randomInRange(r *rand.Rand, lowerIncluding int, upperExcluding int) int {
	return r.Intn(upperExcluding-lowerIncluding) + lowerIncluding
}

func (rc *RandomChain) InvalidExecMsgForLog(chain eth.ChainID, block eth.L2BlockRef, log *types2.Log) *types2.Log {
	payloadHash := crypto.Keccak256Hash(types.LogToMessagePayload(log))

	r := rc.randomGenerator
	msg := types.Message{
		Identifier: types.Identifier{
			Origin:      log.Address,
			BlockNumber: block.Number,
			LogIndex:    uint32(log.Index),
			Timestamp:   block.Time,
			ChainID:     chain,
		},
		PayloadHash: payloadHash,
	}

	switch r.Intn(5) {
	case 0:
		// Invalid origin
		msg.Identifier.Origin = common.HexToAddress("0xffffffffffffffffffffffffffffffffffffffff")
	case 1:
		// Invalid block number
		msg.Identifier.BlockNumber += uint64(randomInRange(r, 1, 10))
	case 2:
		// Invalid log index
		msg.Identifier.LogIndex += uint32(randomInRange(r, 1, 5))
	case 3:
		// Invalid timestamp
		msg.Identifier.Timestamp -= uint64(randomInRange(r, 1, 100))
	case 4:
		// Invalid chain ID
		impossibleChainID := len(rc.chainIDs)
		msg.Identifier.ChainID = eth.ChainIDFromUInt64(uint64(impossibleChainID))
	}

	topics, data := msg.EncodeEvent()
	return &types2.Log{
		Address: params2.InteropCrossL2InboxAddress,
		Data:    data,
		Topics:  topics,
		Index:   log.Index,
	}
}

func (rc *RandomChain) InsertMessageWithInvalidIdentifier() {
	r := rc.randomGenerator
	candidateIndex := randomInRange(r, len(rc.chainIDs), len(rc.allBlocks))
	randomIndex := randomInRange(r, len(rc.chainIDs), len(rc.allBlocks))
	candidateBlock := rc.allBlocks[candidateIndex]
	randomBlock := rc.allBlocks[randomIndex]
	if len(rc.generatedLogs[randomBlock]) == 0 {
		return
	}
	randomLogIndex := r.Intn(len(rc.generatedLogs[randomBlock]))
	randomLog := rc.generatedLogs[randomBlock][randomLogIndex]

	rc.addInvalidExecutingMessage(candidateBlock, randomBlock, randomLog)

	rc.invalidationKind = KindInvalidIdentifier
	rc.invalidationTimestamp = candidateBlock.block.Time
	rc.expectedInvalidChains[candidateBlock.chain] = true
}

func (rc *RandomChain) InsertFutureDependency() {
	t := rc.t
	r := rc.randomGenerator
	latestPossibleIndex := 0
	latestTimestamp := rc.allBlocks[len(rc.allBlocks)-1].block.Time
	for i := len(rc.allBlocks)-1; rc.allBlocks[i].block.Time == latestTimestamp; i-- {
		latestPossibleIndex = i
	}
	candidateIndex := randomInRange(r, len(rc.chainIDs), latestPossibleIndex)
	candidateBlock := rc.allBlocks[candidateIndex]
	t.Logf("Inserting a future dependency in candidate (%s, %2d)'s hazard set", candidateBlock.chain, candidateBlock.block.Number)

	// Find the next block with a timestamp in the future (guaranteed to exist since we added a special block at the end)
	i := candidateIndex + 1
	for rc.allBlocks[i].block.Time <= candidateBlock.block.Time {
		i++
	}

	// Randomly pick a future block and create an executing message to it
	futureIndex := randomInRange(r, i, len(rc.allBlocks))
	futureBlock := rc.allBlocks[futureIndex]
	initiatingLog := rc.addRandomLog(futureBlock)
	rc.addExecutingMessageWithDependency(candidateBlock, futureBlock, initiatingLog)

	rc.invalidationKind = KindFutureDependency
	rc.invalidationTimestamp = candidateBlock.block.Time
	rc.expectedInvalidChains[candidateBlock.chain] = true
}

// InsertExpiredMessage injects a single executing message whose initiating
// message lives more than expiryWindow seconds in the past, so the SUT's
// verifyExecutingMessage trips the `initTimestamp + messageExpiryWindow <
// executingTimestamp` branch (algo.go) and returns ErrMessageExpired.
//
// On success: sets invalidationKind=KindExpiredMessage,
// invalidationTimestamp to the executing block's time, expectedInvalidChains
// to the executing block's chain, and messageExpiryWindow to the supplied
// window value so the harness's New() call uses it.
//
// On failure (chain too short to support the required time gap): leaves the
// chain valid (invalidationKind stays KindNone).
func (rc *RandomChain) InsertExpiredMessage(expiryWindow uint64) {
	r := rc.randomGenerator
	t := rc.t

	// Walk back from the chain's last block to find an executing candidate
	// whose time strictly exceeds expiryWindow (otherwise no initiating block
	// could sit > expiryWindow earlier).
	candidateIndex := -1
	for i := len(rc.allBlocks) - 1; i >= len(rc.chainIDs); i-- {
		if rc.allBlocks[i].block.Time > expiryWindow+1 {
			candidateIndex = i
			break
		}
	}
	if candidateIndex == -1 {
		t.Logf("InsertExpiredMessage: chain too short for window=%d, skipping", expiryWindow)
		return
	}
	candidateBlock := rc.allBlocks[candidateIndex]

	// Find the latest initiating-block index whose time is at most
	// candidate.Time - expiryWindow - 1. Picking the latest such block
	// makes the gap as small as possible while still crossing the boundary,
	// which is good for keeping the seed visualizable.
	threshold := candidateBlock.block.Time - expiryWindow - 1
	initIndex := -1
	for i := len(rc.chainIDs); i < candidateIndex; i++ {
		if rc.allBlocks[i].block.Time <= threshold {
			initIndex = i
		} else {
			break
		}
	}
	if initIndex == -1 {
		t.Logf("InsertExpiredMessage: no init block before candidate@%d satisfies threshold=%d",
			candidateBlock.block.Time, threshold)
		return
	}
	initBlock := rc.allBlocks[initIndex]

	t.Logf("InsertExpiredMessage: init (chain=%s num=%d time=%d) → exec (chain=%s num=%d time=%d), gap=%d window=%d (seed-driven)",
		initBlock.chain, initBlock.block.Number, initBlock.block.Time,
		candidateBlock.chain, candidateBlock.block.Number, candidateBlock.block.Time,
		candidateBlock.block.Time-initBlock.block.Time, expiryWindow)

	// Add the initiating message to the init block and an executing message
	// referencing it on the candidate block.
	initiatingLog := rc.addRandomLog(initBlock)
	rc.addExecutingMessageWithDependency(candidateBlock, initBlock, initiatingLog)

	rc.invalidationKind = KindExpiredMessage
	rc.invalidationTimestamp = candidateBlock.block.Time
	rc.expectedInvalidChains[candidateBlock.chain] = true
	rc.messageExpiryWindow = expiryWindow

	// Quiet the unused-warning if r is not used in the picker (kept for symmetry
	// with the other Insert* routines which all draw from rc.randomGenerator).
	_ = r
}

func (rc *RandomChain) InsertSelfDependency() {
	r := rc.randomGenerator
	candidateIndex := randomInRange(r, len(rc.chainIDs), len(rc.allBlocks))
	candidate := rc.allBlocks[candidateIndex]

	// Create a random initiating message to be inserted at index N+1
	initiatingLog := testutils.RandomLog(r)
	initiatingLog.Index = uint(len(rc.generatedLogs[candidate]) + 1)

	// Insert executing message at index N
	rc.addExecutingMessageWithDependency(candidate, candidate, initiatingLog)

	// Insert initiating message at index N+1
	rc.generatedLogs[candidate] = append(rc.generatedLogs[candidate], initiatingLog)

	rc.invalidationKind = KindSelfDependency
	rc.invalidationTimestamp = candidate.block.Time
	rc.expectedInvalidChains[candidate.chain] = true
}

func (rc *RandomChain) CreateCycle() {
	sameTimeStampSets := SameTimeStampSets(rc.allBlocks[len(rc.chainIDs):])
	if len(sameTimeStampSets) == 0 {
		rc.t.Logf("CreateCycle: No set of blocks with the same timestamp exists. No cycle created")
		return
	}
	i := rc.randomGenerator.Intn(len(sameTimeStampSets))
	set := sameTimeStampSets[i]
	firstLog := rc.addRandomLog(set[0])
	rc.addRandomLog(set[0])
	for i, cb := range set[:len(set)-1] {
		// Add a chain of dependencies to the blocks.
		initiatingLog := rc.generatedLogs[cb][len(rc.generatedLogs[cb])-1]
		execcb := set[i+1]
		rc.t.Logf("CreateCycle: Adding dependency at time %d. chain: %s, block: %d -> chain: %s, block: %d", cb.block.Time, execcb.chain.String(), execcb.block.Number, cb.chain.String(), cb.block.Number)
		rc.addExecutingMessageWithDependency(execcb, cb, initiatingLog)
	}
	// Add the final dependency which closes the cycle.
	cb := set[len(set)-1]
	execcb := set[0]
	rc.t.Logf("CreateCycle: Adding dependency at time %d. chain: %s, block: %d -> chain: %s, block: %d", cb.block.Time, execcb.chain.String(), execcb.block.Number, cb.chain.String(), cb.block.Number)
	initiatingLog := rc.generatedLogs[cb][len(rc.generatedLogs[cb])-1]
	execLog := rc.ExecMsgForLog(cb.chain, *cb.block, initiatingLog)
	execLog.Index = firstLog.Index
	*firstLog = *execLog
	rc.dependencies[execcb] = append(rc.dependencies[execcb], cb)

	rc.invalidationKind = KindCycle
	rc.invalidationTimestamp = set[0].block.Time
	for _, p := range set {
		rc.expectedInvalidChains[p.chain] = true
	}
}
