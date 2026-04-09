include "Types.dfy"

class VerifiedDB {
    ghost const chainIDs : set<ChainID>
    var initialized : bool
    var lastTimestamp : Option<uint64>
    var verified : map<uint64, VerifiedResult>
    var pendingTransition : Option<PendingTransition>

    static predicate NoGaps(s : set<uint64>, max : uint64)
    {
        forall t1, t2 :: (t1 in s && t1 < t2 < max) ==> t2 in s
    }

    ghost predicate Invariants()
        reads this
    {
        match lastTimestamp {
            case None =>
                initialized == false &&
                verified == map[]
            case Some(lastTs) =>
                initialized == true &&
                lastTs in verified.Keys &&
                (forall ts :: ts in verified.Keys ==> 0 < ts <= lastTs) &&
                NoGaps(verified.Keys, lastTs) &&
                (forall result :: result in verified.Values ==> result.L2Heads.Keys == chainIDs)
        } &&
        (pendingTransition != None ==> PendingTransitionIsConsistent(pendingTransition.value, chainIDs))
    }

    ghost function ChainIDs() : set<ChainID>
        reads this
    {
        chainIDs
    }

    ghost function L2HeadsAtTimestamp(ts : uint64) : Option<map<ChainID, BlockID>>
        reads this
    {
        match Get(ts) {
            case None => None
            case Some(result) => Some(result.L2Heads)
        }
    }

    ghost function Verified() : map<uint64, VerifiedResult>
        reads this
    {
        verified
    }

    constructor()
    {
        verified := map[];
        pendingTransition := None;
        new;
        initLastTimestamp();
    }

    constructor OpenVerifiedDB(
        ghost chainIDs_ : set<ChainID>,
        verified_ : map<uint64, VerifiedResult>,
        pendingTransition_ : Option<PendingTransition>
    )
        requires forall ts :: ts in verified_.Keys ==> 0 < ts
        requires verified_ != map[] ==> NoGaps(verified_.Keys, Max(verified_.Keys).value)
        requires forall result :: result in verified_.Values ==> result.L2Heads.Keys == chainIDs_
        requires pendingTransition_ != None ==> PendingTransitionIsConsistent(pendingTransition_.value, chainIDs_)
        ensures Invariants()
    {
        chainIDs := chainIDs_;
        verified := verified_;
        pendingTransition := pendingTransition_;
        new;
        initLastTimestamp();
    }

    method initLastTimestamp()
        requires forall ts :: ts in verified.Keys ==> 0 < ts
        requires verified != map[] ==> NoGaps(verified.Keys, Max(verified.Keys).value)
        requires forall result :: result in verified.Values ==> result.L2Heads.Keys == chainIDs
        requires pendingTransition != None ==> PendingTransitionIsConsistent(pendingTransition.value, chainIDs)
        modifies this
        ensures Invariants()
        ensures pendingTransition == old(pendingTransition)
        ensures verified == old(verified)
        ensures lastTimestamp == Max(verified.Keys)
        ensures verified != map[] ==> lastTimestamp != None
    {
        lastTimestamp := Max(verified.Keys);
        initialized := lastTimestamp != None;
    }

    method Commit(result : VerifiedResult) returns (success : bool)
        requires Invariants()
        requires lastTimestamp != None ==> lastTimestamp.value < MAX_UINT64
        requires 0 < result.Timestamp
        requires result.L2Heads.Keys == chainIDs
        modifies this
        ensures Invariants()
        ensures pendingTransition == old(pendingTransition)
        ensures success ==> (result.Timestamp in verified && verified[result.Timestamp] == result)
        ensures
            if !success || result.Timestamp in old(verified) then
                verified == old(verified) &&
                lastTimestamp == old(lastTimestamp) &&
                initialized == old(initialized)
            else
                result.Timestamp in verified.Keys &&
                verified[result.Timestamp] == result &&
                lastTimestamp == Some(result.Timestamp) &&
                initialized == true
    {
        var ts := result.Timestamp;
        
        if lastTimestamp != None && ts != lastTimestamp.value + 1 {
            if lastTimestamp.value < ts {
                return false; // Non-sequential timestamp
            }

            assert ts <= lastTimestamp.value; // Previous timestamp

            if ts !in verified {
                return false; // This timestamp hasn't been verified before
            }

            if result != verified[ts] {
                return false; // Doesn't match previously-verified result
            }

            return true; // Ignore repeated insertion
        }

        assert lastTimestamp == None || ts == lastTimestamp.value + 1;
        assert ts !in verified;

        verified := verified[ts := result];
        lastTimestamp := Some(ts);
        initialized := true;

        return true;
    }

    function Get(ts : uint64) : Option<VerifiedResult>
        reads this
    {
        if ts in verified then Some(verified[ts]) else None
    }

    predicate Has(ts : uint64)
        reads this
    {
        ts in verified
    }
    
    function LastTimestamp() : Option<uint64>
        reads this
    {
        lastTimestamp
    }

    method Rewind(timestamp : uint64) returns (deleted : bool)
        requires Invariants()
        modifies this
        ensures Invariants()
        ensures pendingTransition == old(pendingTransition)
        ensures
            if old(lastTimestamp) == None || old(lastTimestamp).value < timestamp then
                !deleted &&
                verified == old(verified) &&
                lastTimestamp == old(lastTimestamp) &&
                initialized == old(initialized)
            else if timestamp == 0 || (timestamp - 1) !in old(verified) then
                deleted &&
                verified == map[] &&
                lastTimestamp == None &&
                initialized == false
            else
                deleted &&
                verified == old(verified) - (set t | t in old(verified).Keys && timestamp <= t) &&
                lastTimestamp != None &&
                lastTimestamp.value == timestamp - 1
    {
        var toDelete := (set t | t in verified.Keys && timestamp <= t);
        verified := verified - toDelete;

        if toDelete == {} {
            return false;
        } else {
            assert old(verified) != map[];
            assert verified != map[] ==> (timestamp - 1) in verified;
            assert verified != map[] ==> forall t :: timestamp <= t ==> t !in verified;
            assert verified != map[] ==> Max(verified.Keys) == Some(timestamp - 1);
            
            initLastTimestamp();
            
            assert (timestamp != 0 && (timestamp - 1) in old(verified)) ==> (timestamp - 1) in verified;
            assert lastTimestamp != None ==> lastTimestamp.value == timestamp - 1;
            
            return true;
        }
    }

    method SetPendingTransition(pending : PendingTransition)
        requires Invariants()
        requires PendingTransitionIsConsistent(pending, chainIDs)
        modifies this
        ensures Invariants()
        ensures pendingTransition == Some(pending)
        ensures verified == old(verified)
    {
        pendingTransition := Some(pending);
    }

    function GetPendingTransition() : Option<PendingTransition>
        requires Invariants()
        reads this
        ensures GetPendingTransition() != None ==> PendingTransitionIsConsistent(GetPendingTransition().value, chainIDs)
    {
        pendingTransition
    }

    method ClearPendingTransition()
        requires Invariants()
        modifies this
        ensures Invariants()
        ensures pendingTransition == None
        ensures lastTimestamp == old(lastTimestamp)
    {
        pendingTransition := None;
    }
}