include "Types.dfy"

class ChainContainer {
    function ConfirmedTimestampFor(block : BlockID) : Option<uint64>
        // No reads clause so it's independent of the current state
    
    method PruneDeniedAtOrAfterTimestamp(timestamp : uint64) returns (success : bool)
        modifies this
    
    method RewindEngine(timestamp : uint64) returns (success : bool)
        modifies this

    method InvalidateBlock(height : uint64, payloadHash : Hash, decisionTimestamp : uint64) returns (rewindTriggered : Option<bool>)
        modifies this

    method OptimisticAt(ts : uint64) returns (l2AndL1 : OkOrErr<(BlockID, BlockID)>)
        ensures{:axiom} l2AndL1.Ok? ==>
            var confirmedTS := ConfirmedTimestampFor(l2AndL1.value.0);
            confirmedTS != None && confirmedTS.value <= ts

    method FetchReceipts(blockID : BlockID) returns (fetched : Option<(BlockInfo, seq<Receipt>)>)
        ensures{:axiom} fetched != None ==> fetched.value.0.ID == blockID
        ensures{:axiom} (fetched != None && ConfirmedTimestampFor(blockID) != None) ==>
            ConfirmedTimestampFor(blockID).value == fetched.value.0.Time

    method SyncStatus() returns (status : Option<SyncStatus>)

    function BlockTime() : uint64
        reads this
}