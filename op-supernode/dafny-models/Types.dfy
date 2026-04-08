const MAX_UINT32 := 4294967295
const MAX_UINT64 := 18446744073709551615
const MAX_UINT256 := 115792089237316195423570985008687907853269984665640564039457584007913129639935

type byte = i : int | 0 <= i < 256
type uint32 = i : int | 0 <= i <= MAX_UINT32
type uint64 = i : int | 0 <= i <= MAX_UINT64
type uint256 = i : int | 0 <= i <= MAX_UINT256
newtype Hash = uint256
newtype ChainID = uint256
datatype Option<T> = None | Some(value : T)

function Max(s : set<uint64>) : Option<uint64>
    ensures{:axiom} s == {} <==> Max(s) == None
    ensures{:axiom} s != {} ==> Max(s).value in s
    ensures{:axiom} s != {} ==> forall x :: x in s ==> x <= Max(s).value

function Enumerate<T(!new)>(s : set<T>) : seq<T>
    ensures{:axiom} |Enumerate(s)| == |s|
    ensures{:axiom} forall x :: x in Enumerate(s) <==> x in s

datatype BlockID = BlockID(
    Hash : Hash,
    Number : uint64
)

datatype BlockRef = BlockRef(
    ID : BlockID,
    ParentHash : Hash,
    Time : uint64
)

datatype ChainsReadyResult
    = NotReady
    | Ready(
        Blocks : map<ChainID, BlockID>, 
        L1Heads : map<ChainID, BlockID>
    )

datatype ChainConsistencyResult
    = InconsistentL2s
    | InconsistentVerifiedL1
    | Consistent(L1Inclusion : BlockID)

datatype RoundObservation
    = NotReady
    | InconsistentL2s
    | InconsistentVerifiedL1
    | Consistent(
        NextTimestamp : uint64,
        BlocksAtTS : map<ChainID, BlockID>,
        L1Inclusion : BlockID
    )

datatype RewindPlan = RewindPlan(
    RewindAtOrAfter : uint64,
    ResetAllChainsTo : Option<uint64>,
    TargetHeads : map<ChainID, BlockID>
)

datatype PendingTransition
    = Rewind(
        RewindPlan : RewindPlan
    )
    | Advance(
        Timestamp : uint64,
        L1Inclusion : BlockID,
        L2Heads : map<ChainID, BlockID>
    )
    | Invalidate(
        Timestamp : uint64,
        InvalidHeads : map<ChainID, BlockID>
    )

datatype StepOutput
    = Wait
    | Step(PendingTransition)

datatype VerifiedResult = VerifiedResult(
    Timestamp : uint64,
    L1Inclusion : BlockID,
    L2Heads : map<ChainID, BlockID>
)

datatype ExecutingMessage = ExecutingMessage(
    ChainID : ChainID,
	BlockNum : uint64,
	LogIdx : uint32,
	Timestamp : uint64
)

// BlockWithLogs is a LogsDB entry: a block paired with its executing messages.
// Used by SupernodeState.dfy to model L_j = [(B^j_0, l^j_0), ...].
// See invariants/SPEC.md §0 (Notation) and I1.
datatype BlockWithLogs = BlockWithLogs(
    Ref : BlockRef,
    ExecMsgs : seq<ExecutingMessage>
)

// DenyListEntry is a block that was invalidated during cross-validation,
// tagged with the timestamp at which the invalidation decision was made.
// The decision timestamp is required by invariants/SPEC.md §I11.
datatype DenyListEntry = DenyListEntry(
    Block : BlockID,
    DecisionTimestamp : uint64
)

// IsParentOf captures the linear-chain relationship used by I2, I6, T5.
// A block `child` is the immediate successor of `parent` iff its ParentHash
// points to `parent.ID.Hash` and its number is parent.Number + 1.
// The `as int` cast promotes to mathematical integers so `+ 1` has no
// overflow side-condition.
predicate IsParentOf(parent : BlockRef, child : BlockRef)
{
    child.ParentHash == parent.ID.Hash &&
    (child.ID.Number as int) == (parent.ID.Number as int) + 1
}

datatype FrontierBlockView = FrontierBlockView(
    Ref : BlockRef,
    ExecMsgs : seq<ExecutingMessage>
)