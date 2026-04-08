package invariants

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// Snapshot is the pure abstract state consumed by every CheckI* predicate.
// It mirrors the Dafny SupernodeState datatype 1:1; the two MUST stay in
// lockstep. See dafny-models/SupernodeState.dfy and SPEC.md §0.
//
// Snapshot is JSON-serializable so that counter-examples and traces can be
// shared between the Go fuzzer, the reference model, and any external
// replay tool.
type Snapshot struct {
	// ActivationTS is t_0 from SPEC.md §0.
	ActivationTS uint64 `json:"activationTs"`

	// Chains is the set of L2 chain IDs the Supernode is tracking. It
	// corresponds to Dafny's `Chains : set<ChainID>`.
	Chains []eth.ChainID `json:"chains"`

	// LogsDB[j] is L_j from SPEC.md §0: the per-chain sequence
	// [(B^j_0, l^j_0), ..., (B^j_{n_j}, l^j_{n_j})].
	//
	// By SPEC invariant I10 and the well-formedness constraint used in the
	// Dafny IsInitialState predicate, LogsDB.keys should equal Chains.
	LogsDB map[eth.ChainID][]BlockWithLogs `json:"logsDb"`

	// Verified is the VerifiedDB as an ordered slice:
	//   [(t_0, C_{t_0}, C^1_{t_0}, ..., C^k_{t_0}),
	//    ..., (t, C_t, C^1_t, ..., C^k_t)].
	// Ascending by Timestamp. The sort order matches
	// SupernodeView.dfy :: VerifiedMapToSortedSeq.
	Verified []VerifiedEntry `json:"verified"`

	// DenyList[j] is D_j from SPEC.md §0, tracked with explicit decision
	// timestamps so I11 is checkable from a pure snapshot (SPEC.md §5 #7).
	DenyList map[eth.ChainID][]DenyListEntry `json:"denyList"`
}

// BlockWithLogs mirrors Dafny's `BlockWithLogs(Ref, ExecMsgs)` datatype. It
// is one LogsDB row. See SPEC.md §0, I1, I2, I3.
type BlockWithLogs struct {
	Ref      BlockRef           `json:"ref"`
	ExecMsgs []ExecutingMessage `json:"execMsgs"`
}

// BlockRef is the minimal block reference needed for invariant checking:
// identity (hash + number), parent hash (for the linear-chain check in I2),
// and timestamp (for I7). It mirrors Dafny's BlockRef.
type BlockRef struct {
	ID         eth.BlockID `json:"id"`
	ParentHash common.Hash `json:"parentHash"`
	Time       uint64      `json:"time"`
}

// ExecutingMessage mirrors the Dafny ExecutingMessage datatype. An executing
// message names an initiating log on a different chain by
// (ChainID, BlockNum, LogIdx, Timestamp). See SPEC.md I3.
type ExecutingMessage struct {
	ChainID   eth.ChainID `json:"chainId"`
	BlockNum  uint64      `json:"blockNum"`
	LogIdx    uint32      `json:"logIdx"`
	Timestamp uint64      `json:"timestamp"`
}

// VerifiedEntry mirrors the Dafny VerifiedResult datatype. It is one
// VerifiedDB row at a single timestamp across all tracked chains.
type VerifiedEntry struct {
	Timestamp   uint64                      `json:"timestamp"`
	L1Inclusion eth.BlockID                 `json:"l1Inclusion"`
	L2Heads     map[eth.ChainID]eth.BlockID `json:"l2Heads"`
}

// DenyListEntry carries the decision timestamp alongside the block ID so
// I11 can be checked from state alone. See SPEC.md §5 item 7 and Dafny's
// DenyListEntry datatype.
type DenyListEntry struct {
	Block             eth.BlockID `json:"block"`
	DecisionTimestamp uint64      `json:"decisionTimestamp"`
}

// LastVerifiedTimestamp returns the t from SPEC.md §0 along with a boolean
// indicating whether any cross-validation has occurred. It mirrors the
// Dafny LastVerifiedTimestamp helper.
func (s Snapshot) LastVerifiedTimestamp() (uint64, bool) {
	if len(s.Verified) == 0 {
		return 0, false
	}
	return s.Verified[len(s.Verified)-1].Timestamp, true
}

// LogsDBLookup finds a block in a LogsDB slice by ID. It mirrors the Dafny
// LogsDBLookup function and is used by I6 / I7.
func LogsDBLookup(logs []BlockWithLogs, id eth.BlockID) (BlockWithLogs, bool) {
	for _, b := range logs {
		if b.Ref.ID == id {
			return b, true
		}
	}
	return BlockWithLogs{}, false
}

// chainsSet returns s.Chains as a set for O(1) membership checks. Used by
// predicates that need to filter LogsDB / DenyList entries by the tracked
// chain set.
func (s Snapshot) chainsSet() map[eth.ChainID]struct{} {
	m := make(map[eth.ChainID]struct{}, len(s.Chains))
	for _, c := range s.Chains {
		m[c] = struct{}{}
	}
	return m
}
