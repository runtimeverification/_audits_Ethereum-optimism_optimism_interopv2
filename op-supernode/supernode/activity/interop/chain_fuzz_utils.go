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
	"github.com/stretchr/testify/require"
	"github.com/ethereum-optimism/optimism/op-node/params"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/processors"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func ExecMsgForLog(chain eth.ChainID, block eth.L2BlockRef, log *types2.Log) *types2.Log {
	msg := types.Message{
		Identifier: types.Identifier{
			Origin:      log.Address,
			BlockNumber: block.Number,
			LogIndex:    uint32(log.Index),
			Timestamp:   block.Time,
			ChainID:     chain,
		},
		PayloadHash: processors.LogToLogHash(log),
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
	randomGenerator *rand.Rand
	chainIDs      []eth.ChainID
	allBlocks     []*ChainBlock
	cbIndices     map[*eth.L2BlockRef]int // Lookup for a ChainBlock's index in allBlocks
	generatedLogs map[ChainBlock][]*types2.Log
	dependencies  map[ChainBlock][]*ChainBlock
	chainBlocks   map[eth.ChainID][]*eth.L2BlockRef
	l1SourceMap   map[ChainBlock]eth.BlockRef
	l1Source      map[uint64]eth.BlockRef
	receipts      map[eth.ChainID]map[eth.BlockID]types2.Receipts
	blockTimes    map[eth.ChainID]int
	isInvalid     bool
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
func (c RandomChainContainer) RegisterVerifier(v activity.VerificationActivity) {}

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
	return eth.BlockID{}, eth.BlockID{}, nil
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

func (c RandomChainContainer) InvalidateBlock(ctx context.Context, height uint64, payloadHash common.Hash) (bool, error) {
	//TODO
	return true, nil
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

func (p *RandomChainParams) MakeRandomChain(t *testing.T, seed int64) (res RandomChain) {
	r := rand.New(rand.NewSource(seed))

	totalLength := randomInRange(r, p.minLength, p.maxLength) + 2

	res = RandomChain{
		randomGenerator: r,
		chainIDs:      make([]eth.ChainID, 0, p.chainCount),
		allBlocks:     make([]*ChainBlock, 0, totalLength),
		cbIndices:     make(map[*eth.L2BlockRef]int),
		generatedLogs: make(map[ChainBlock][]*types2.Log),
		dependencies:  make(map[ChainBlock][]*ChainBlock),
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
	}

	//
	// Create array of all blocks
	//

	// First, guarantee that each chain contains at least one block
	lastTimeStamp := make(map[eth.ChainID]uint64)
	for _, chain := range res.chainIDs {
		block := testutils.RandomL2BlockRef(r)
		block.Number = 0
		block.Time = 0
		res.chainBlocks[chain] = append(res.chainBlocks[chain], &block)
	}

	// Then, generate the rest of the blocks.
	for range totalLength - p.chainCount {
		// Select the chain with the next valid timestamp
		first := true
		var nextChain eth.ChainID
		v := uint64(0)
		for _, chain := range res.chainIDs {
			nextTimeStamp := lastTimeStamp[chain] + uint64(res.blockTimes[chain])
			if first || nextTimeStamp < v {
				nextChain = chain
				v = nextTimeStamp
				first = false
			}
		}

		// Add a random block to it
		lastBlock := res.chainBlocks[nextChain][len(res.chainBlocks[nextChain])-1]
		block := testutils.NextRandomL2Ref(r, uint64(res.blockTimes[nextChain]), *lastBlock, eth.BlockID{})
		res.chainBlocks[nextChain] = append(res.chainBlocks[nextChain], &block)
		lastTimeStamp[nextChain] = block.Time
	}

	// Populate res.allBlocks
	//
	// The blocks need to be added in order by timestamp,
	// so all of the iterating logic through the chains here
	// finds the next block with the lowest timestamp.
	chainIndices := make(map[eth.ChainID]int)
	for i := range totalLength {
		var finalChain eth.ChainID
		var finalBlock *eth.L2BlockRef

		for _, chain := range res.chainIDs {
			idx := chainIndices[chain]
			if idx < len(res.chainBlocks[chain]) {
				block := res.chainBlocks[chain][idx]
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
		res.allBlocks = append(res.allBlocks, &chainBlock)
		res.cbIndices[finalBlock] = i
	}

	//
	// Create random dependencies between all blocks
	//
	for initIndex, initcb := range res.allBlocks {
		// Add an unimportant message at index 0 that can be modified later by the InsertCycle function
		addRandomInitiatingMessage(r, &res, initcb)

		block := initcb.block
		if block.Number == 0 {
			continue
		}

		for r.Intn(100) < p.dependencyChance {
			execIndex := randomInRange(r, initIndex, totalLength)
			execcb := res.allBlocks[execIndex]
			if block.Number == 0 {
				continue
			}
			res.dependencies[*execcb] = append(res.dependencies[*execcb], initcb)
		}
	}

	// Construct the dependencies by creating initiating/executing message pairs
	for _, execcb := range res.allBlocks {
		for _, initcb := range res.dependencies[*execcb] {
			initiatingLog := addRandomInitiatingMessage(r, &res, initcb)
			addExecutingMessage(&res, execcb, initcb, initiatingLog)
		}
	}

	if r.Intn(100) < p.invalidateChance {
		res.isInvalid = true
		index := r.Intn(len(res.allBlocks)-1)
		blockToInvalidate := res.allBlocks[index]
		cbIndex := res.cbIndices[blockToInvalidate.block]
		t.Logf("Randomly selected block index: %d", index)
		t.Logf("cbIndex: %d", cbIndex)

		InvalidateBlock(t, &res, blockToInvalidate)
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
			res.l1SourceMap[*l2Block] = nextL1
		}
		res.l1Source[nextL1.Number] = nextL1
		taken += take
	}

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

func addRandomInitiatingMessage(r *rand.Rand, res *RandomChain, initcb *ChainBlock) *types2.Log {
	initiatingLog := testutils.RandomLog(r)
	initiatingLog.Index = uint(len(res.generatedLogs[*initcb]))
	res.generatedLogs[*initcb] = append(res.generatedLogs[*initcb], initiatingLog)
	return initiatingLog
}

func addExecutingMessage(res *RandomChain, execcb *ChainBlock, initcb *ChainBlock, initiatingLog *types2.Log) {
	execLog := ExecMsgForLog(initcb.chain, *initcb.block, initiatingLog)
	execLog.Index = uint(len(res.generatedLogs[*execcb]))
	res.generatedLogs[*execcb] = append(res.generatedLogs[*execcb], execLog)
}

func addExecutingMessageWithDependency(res *RandomChain, execcb *ChainBlock, initcb *ChainBlock, initiatingLog *types2.Log) {
	addExecutingMessage(res, execcb, initcb, initiatingLog)
	res.dependencies[*execcb] = append(res.dependencies[*execcb], initcb)
}

func addInvalidExecutingMessage(r *rand.Rand, res *RandomChain, execcb *ChainBlock, initcb *ChainBlock, initiatingLog *types2.Log) {
	execLog := InvalidExecMsgForLog(r, res, initcb.chain, *initcb.block, initiatingLog)
	execLog.Index = uint(len(res.generatedLogs[*execcb]))
	res.generatedLogs[*execcb] = append(res.generatedLogs[*execcb], execLog)
}

func insertExecutingMessageAt(i uint, res *RandomChain, execcb *ChainBlock, initcb *ChainBlock, initiatingLog *types2.Log) {
	execLog := ExecMsgForLog(initcb.chain, *initcb.block, initiatingLog)
	execLog.Index = i
	res.generatedLogs[*execcb][i] = execLog
}

func GenerateReceiptsFromLogs(res *RandomChain) {
	for _, cb := range res.allBlocks {
		chainid, block := cb.chain, cb.block
		logs := res.generatedLogs[*cb]
		rcpt := types2.Receipt{
			Logs: logs,
		}
		res.receipts[chainid][block.ID()] = types2.Receipts{&rcpt};
	}
}

// Returns a random integer in the interval [lowerIncluding, upperExcluding)
func randomInRange(r *rand.Rand, lowerIncluding int, upperExcluding int) int {
	return r.Intn(upperExcluding-lowerIncluding) + lowerIncluding
}

func InvalidExecMsgForLog(r *rand.Rand, res *RandomChain, chain eth.ChainID, block eth.L2BlockRef, log *types2.Log) *types2.Log {
	msg := types.Message{
		Identifier: types.Identifier{
			Origin:      log.Address,
			BlockNumber: block.Number,
			LogIndex:    uint32(log.Index),
			Timestamp:   block.Time,
			ChainID:     chain,
		},
		PayloadHash: processors.LogToLogHash(log),
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
		impossibleChainID := len(res.chainIDs)
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

func InsertMessageWithInvalidIdentifier(r *rand.Rand, res *RandomChain, candidateIndex int) {
	candidateBlock := res.allBlocks[candidateIndex]
	randomIndex := r.Intn(candidateIndex + 1)
	randomBlock := res.allBlocks[randomIndex]
	randomLogIndex := r.Intn(len(res.generatedLogs[*randomBlock]))
	randomLog := res.generatedLogs[*randomBlock][randomLogIndex]

	addInvalidExecutingMessage(r, res, candidateBlock, randomBlock, randomLog)
}

func InvalidateBlock(t *testing.T, res *RandomChain, candidate *ChainBlock) {
	r := res.randomGenerator
	switch r.Intn(3) {
	case 0:
		InsertCycle(t, r, res, candidate)
	case 1:
		InsertSelfDependency(r, res, candidate)
	case 2:
		InsertMessageWithInvalidIdentifier(r, res, res.cbIndices[candidate.block])
	case 3:
		//InsertDependencyToExpiredMessage(t, r, res, res.cbIndices[*candidate])
	case 4:
		//InsertFutureDependency(t, r, res, res.cbIndices[candidate.block])
	default:
	}
}

func InsertFutureDependency(t *testing.T, r *rand.Rand, res *RandomChain, candidateIndex int) {
	candidateBlock := res.allBlocks[candidateIndex]
	t.Logf("Inserting a future dependency in candidate (%s, %2d)'s hazard set", candidateBlock.chain, candidateBlock.block.Number)

	// Find the next block with a timestamp in the future (guaranteed to exist since we added a special block at the end)
	i := candidateIndex + 1
	for res.allBlocks[i].block.Time <= candidateBlock.block.Time {
		i++
	}

	// Randomly pick a future block and create an executing message to it
	futureIndex := randomInRange(r, i, len(res.allBlocks))
	futureBlock := res.allBlocks[futureIndex]
	initiatingLog := addRandomInitiatingMessage(r, res, futureBlock)
	addExecutingMessageWithDependency(res, candidateBlock, futureBlock, initiatingLog)
}

func InsertDependencyToExpiredMessage(t *testing.T, r *rand.Rand, res *RandomChain, candidateIndex int) {
	candidate := res.allBlocks[candidateIndex]

	// We set the timestamps so that this is true for every block that can be selected as candidate
	require.Less(t, uint64(params.MessageExpiryTimeSecondsInterop), candidate.block.Time)

	// Any timestamp below this is expired
	expiryTimestamp := candidate.block.Time - params.MessageExpiryTimeSecondsInterop

	// Iterate until we find the first unexpired block
	i := 0
	for res.allBlocks[i].block.Time < expiryTimestamp {
		i++
	}

	// i is at least 1 since the block at index 0 is guaranteed to be expired
	expiredIndex := r.Intn(i)
	expiredBlock := res.allBlocks[expiredIndex]
	initiatingLog := addRandomInitiatingMessage(r, res, expiredBlock)
	addExecutingMessageWithDependency(res, candidate, expiredBlock, initiatingLog)
}

func InsertSelfDependency(r *rand.Rand, res *RandomChain, candidate *ChainBlock) {
	// Create a random initiating message to be inserted at index N+1
	initiatingLog := testutils.RandomLog(r)
	initiatingLog.Index = uint(len(res.generatedLogs[*candidate]) + 1)

	// Insert executing message at index N
	addExecutingMessageWithDependency(res, candidate, candidate, initiatingLog)

	// Insert initiating message at index N+1
	res.generatedLogs[*candidate] = append(res.generatedLogs[*candidate], initiatingLog)
}

func listHazards(t *testing.T, res *RandomChain, candidate *ChainBlock) []*ChainBlock {
	hazards := make([]*ChainBlock, 0)
	includedHazards := make(map[eth.ChainID]*ChainBlock)

	// Add the candidate itself as a hazard
	stack := []*ChainBlock{candidate}

	for len(stack) > 0 {
		// Pop hazard from the stack
		hazard := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		// Check if we already found a hazard from this chain
		includedHazard, ok := includedHazards[hazard.chain]
		if ok {
			// Ensure that there are not two different hazards from the same chain
			require.Equal(t, includedHazard.block.ID(), hazard.block.ID())
		} else {
			// If not already included, add hazard to the list
			hazards = append(hazards, hazard)
			includedHazards[hazard.chain] = hazard

			// For each new hazard, add all dependencies with the same timestamp to the stack
			for _, dependency := range res.dependencies[*hazard] {
				if dependency.block.Time == candidate.block.Time {
					stack = append(stack, dependency)
				}
			}
		}
	}

	return hazards
}

func InsertCycle(t *testing.T, r *rand.Rand, res *RandomChain, candidate *ChainBlock) {
	t.Logf("Inserting a cycle in candidate (%s, %2d)'s hazard set", candidate.chain, candidate.block.Number)

	candidateHazards := listHazards(t, res, candidate)
	t.Logf("Size of (%s, %2d)'s hazard set: %d", candidate.chain, candidate.block.Number, len(candidateHazards))
	cycleStart := candidateHazards[r.Intn(len(candidateHazards))]
	t.Logf("Picked random hazard set element to start the cycle: (%s, %2d)", cycleStart.chain, cycleStart.block.Number)

	// If the random element is equal to the candidate, no need to compute the hazards again
	var subHazards []*ChainBlock
	if cycleStart.chain == candidate.chain {
		require.Equal(t, cycleStart.block.Number, candidate.block.Number)
		subHazards = candidateHazards
	} else {
		subHazards = listHazards(t, res, cycleStart)
		t.Logf("Size of (%s, %2d)'s hazard set: %d", cycleStart.chain, cycleStart.block.Number, len(subHazards))
	}

	cycleEnd := subHazards[r.Intn(len(subHazards))]
	t.Logf("Picked random hazard set element to end the cycle: (%s, %2d)", cycleEnd.chain, cycleEnd.block.Number)

	// Add executing message from first log of cycleEnd to last log of cycleStart
	lastIndex := len(res.generatedLogs[*cycleStart]) - 1
	initiatingLog := res.generatedLogs[*cycleStart][lastIndex]
	// Replace dummy message at index 0
	insertExecutingMessageAt(0, res, cycleEnd, cycleStart, initiatingLog)
	res.dependencies[*cycleEnd] = append(res.dependencies[*cycleEnd], cycleStart)
	t.Logf("Added cyclic dependency: (%s, %2d) -> (%s, %2d)", cycleEnd.chain, cycleEnd.block.Number, cycleStart.chain, cycleStart.block.Number)
}
