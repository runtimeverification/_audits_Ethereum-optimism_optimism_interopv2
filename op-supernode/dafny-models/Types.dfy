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

datatype Error
    = NotFound
    | AssumptionViolation
    | Other

datatype OkOrErr<T> = Ok(value : T) | Err(err : Error)

function Max(s : set<uint64>) : Option<uint64>
    ensures{:axiom} s == {} <==> Max(s) == None
    ensures{:axiom} s != {} ==> Max(s).value in s
    ensures{:axiom} s != {} ==> forall x :: x in s ==> x <= Max(s).value

function Enumerate<T(!new)>(s : set<T>) : seq<T>
    ensures{:axiom} |Enumerate(s)| == |s|
    ensures{:axiom} forall x :: x in Enumerate(s) <==> x in s
    ensures{:axiom} forall i, j :: 0 <= i < j < |Enumerate(s)| ==> Enumerate(s)[i] != Enumerate(s)[j]
    // ^ redundant, but helps verification since it's hard for Dafny to prove it otherwise

ghost predicate IsRange(s : set<uint64>, lower : uint64, upper : uint64)
{
    forall x :: (lower <= x <= upper) <==> x in s
}

datatype BlockID = BlockID(
    Hash : Hash,
    Number : uint64
)

datatype BlockSeal = BlockSeal(
    Hash : Hash,
    Number : uint64,
    Timestamp : uint64
)

datatype BlockRef = BlockRef(
    ID : BlockID,
    ParentHash : Hash,
    Time : uint64
)

datatype BlockInfo = BlockInfo(
    ID : BlockID,
    ParentHash : Hash,
    Time : uint64
)

datatype Log = Log(
    ExecMsg : Option<ExecutingMessage>
)

datatype Receipt = Receipt(
    Logs : seq<Log>
)

datatype SyncStatus = SyncStatus(
    CurrentL1 : BlockRef
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

datatype PendingTransition
    = Rewind(
        RewindAtOrAfter : uint64,
        TargetHeads : Option<map<ChainID, BlockID>>
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

predicate PendingTransitionIsConsistent(pending : PendingTransition, chainIDs : set<ChainID>)
{
    match pending {
        case Rewind(rewindAtOrAfter, targetHeads) =>
            0 < rewindAtOrAfter &&
            (targetHeads != None ==> targetHeads.value.Keys == chainIDs)
        case Advance(ts, _, l2Heads) =>
            0 < ts &&
            l2Heads.Keys == chainIDs
        case Invalidate(_, invalidHeads) =>
            invalidHeads != map[] &&
            invalidHeads.Keys <= chainIDs
    }
}

datatype StepOutput
    = Wait
    | Step(pending : PendingTransition)

datatype VerifiedResult = VerifiedResult(
    Timestamp : uint64,
    L1Inclusion : BlockID,
    L2Heads : map<ChainID, BlockID>
)

datatype ContainsQuery = ContainsQuery(
	Timestamp : uint64,
	BlockNum : uint64,
	LogIdx : uint32,
	Checksum : Hash
)

datatype ExecutingMessage = ExecutingMessage(
    ChainID : ChainID,
	BlockNum : uint64,
	LogIdx : uint32,
	Timestamp : uint64,
    Checksum : Hash
)

datatype FrontierBlockView = FrontierBlockView(
    Ref : BlockRef,
    ExecMsgs : seq<ExecutingMessage>
)