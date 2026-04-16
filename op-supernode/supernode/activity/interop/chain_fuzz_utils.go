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

type InvalidInfo interface {
	TestResult(result Result)
}

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
	invalidInfo   InvalidInfo
}

var _ cc.ChainContainer = RandomChainContainer{}

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
	return &eth.SyncStatus{CurrentL1: l1Origin}, nil
}

func (c RandomChainContainer) VerifiedAt(ctx context.Context, ts uint64) (l2, l1 eth.BlockID, err error) {
	//TODO
	return eth.BlockID{}, eth.BlockID{}, nil
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

func (c RandomChainContainer) OutputRootAtL2BlockNumber(ctx context.Context, l2BlockNum uint64) (eth.Bytes32, error) {
	//TODO
	return eth.Bytes32{}, nil
}

func (c RandomChainContainer) OptimisticOutputAtTimestamp(ctx context.Context, ts uint64) (*eth.OutputResponse, error) {
	//TODO
	return nil, nil
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

func (c RandomChainContainer) InvalidateBlock(ctx context.Context, height uint64, payloadHash common.Hash, decisionTimestamp uint64) (bool, error) {
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

func (rc *RandomChain) GetContainers() (map[eth.ChainID]cc.ChainContainer) {
	chains := make(map[eth.ChainID]cc.ChainContainer);
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
		receipts:      make(map[eth.ChainID]map[eth.BlockID]types2.Receipts),
		blockTimes:    make(map[eth.ChainID]int),
		isInvalid:     false,
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

	//
	// Create random dependencies between all blocks
	//
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

	if r.Intn(100) < p.invalidateChance {
		res.isInvalid = true
		res.Invalidate()
	}

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
	randomLogIndex := r.Intn(len(rc.generatedLogs[randomBlock]))
	randomLog := rc.generatedLogs[randomBlock][randomLogIndex]

	rc.addInvalidExecutingMessage(candidateBlock, randomBlock, randomLog)
}

func (rc *RandomChain) Invalidate() {
	r := rc.randomGenerator
	rc.t.Logf("Invalidating chains!")
	switch r.Intn(4) {
	case 0:
		rc.t.Logf("Creating a cycle")
		rc.CreateCycle()
	case 1:
		rc.t.Logf("Creating a self dependency")
		rc.InsertSelfDependency()
	case 2:
		rc.t.Logf("Creating an invalid message")
		rc.InsertMessageWithInvalidIdentifier()
	case 3:
		rc.t.Logf("Creating a future dependency")
		rc.InsertFutureDependency()
	default:
	}
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
}

var _ InvalidInfo = CycleInfo{}

type CycleInfo struct {
	t      *testing.T
	blocks []ChainBlock
}

func (c CycleInfo) TestResult(result Result) {
	for _, block := range c.blocks {
		require.Equal(c.t, block.block.ID(), result.InvalidHeads[block.chain])
	}
}

func (rc *RandomChain) CreateCycle() {
	sameTimeStampSets := SameTimeStampSets(rc.allBlocks[len(rc.chainIDs):])
	if len(sameTimeStampSets) == 0 {
		rc.t.Logf("CreateCycle: No set of blocks with the same timestamp exists. No cycle created")
		return
	}
	i := rc.randomGenerator.Intn(len(sameTimeStampSets))
	set := sameTimeStampSets[i]
	info := CycleInfo{ t: rc.t, blocks: make([]ChainBlock, 0) }
	for i, cb := range set {
		initiatingLog := rc.addRandomLog(cb)
		execcb := set[(i+1)%len(set)]
		rc.t.Logf("CreateCycle: Adding dependency at time %d. chain: %s, block: %d -> chain: %s, block: %d", cb.block.Time, execcb.chain.String(), execcb.block.Number, cb.chain.String(), cb.block.Number)
		rc.addExecutingMessageWithDependency(execcb, cb, initiatingLog)
		info.blocks = append(info.blocks, cb)
	}
	rc.invalidInfo = &info
}
