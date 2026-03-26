include "Types.dfy"

class VerifiedDB {
    var initialized : bool
    var lastTimestamp : Option<uint64>
    var verified : map<uint64, VerifiedResult>
    var pendingTransition : Option<PendingTransition>

    static predicate NoGaps(s : set<uint64>, max : uint64)
    {
        forall t1, t2 :: (t1 in s && t1 < t2 < max) ==> t2 in s
    }

    predicate Invariants()
        reads this
    {
        match lastTimestamp {
            case None =>
                initialized == false &&
                verified == map[]
            case Some(lastTs) =>
                initialized == true &&
                lastTs in verified.Keys &&
                (forall ts :: ts in verified.Keys ==> ts <= lastTs) &&
                NoGaps(verified.Keys, lastTs)
        }
    }

    constructor()
    {
        verified := map[];
        pendingTransition := None;
        new;
        initLastTimestamp();
    }

    constructor OpenVerifiedDB(verified_ : map<uint64, VerifiedResult>, pendingTransition_ : Option<PendingTransition>)
        requires verified_ != map[] ==> NoGaps(verified_.Keys, Max(verified_.Keys).value)
        ensures Invariants()
    {
        verified := verified_;
        pendingTransition := pendingTransition_;
        new;
        initLastTimestamp();
    }

    method initLastTimestamp()
        requires verified != map[] ==> NoGaps(verified.Keys, Max(verified.Keys).value)
        modifies this
        ensures Invariants()
        ensures verified == old(verified)
        ensures lastTimestamp == Max(verified.Keys)
    {
        lastTimestamp := Max(verified.Keys);
        initialized := lastTimestamp != None;
    }

    method Commit(result : VerifiedResult) returns (success : bool)
        requires Invariants()
        requires lastTimestamp != None ==> lastTimestamp.value < MAX_UINT64
        modifies this
        ensures Invariants()
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
        ensures (old(lastTimestamp) == None || old(lastTimestamp).value < timestamp) <==> !deleted
        ensures
            if !deleted then
                verified == old(verified) &&
                lastTimestamp == old(lastTimestamp) &&
                initialized == old(initialized)
            else
                lastTimestamp == None ||
                lastTimestamp.value < timestamp // actually lastTimestamp.value == timestamp - 1
    {
        var toDelete := (set t | t in verified.Keys && timestamp <= t);
        verified := verified - toDelete;

        if toDelete == {} {
            return false;
        } else {
            initLastTimestamp();
            return true;
        }
    }

    method SetPendingTransition(pending : PendingTransition)
        requires Invariants()
        modifies this
        ensures Invariants()
    {
        pendingTransition := Some(pending);
    }

    function GetPendingTransition() : Option<PendingTransition>
        reads this
    {
        pendingTransition
    }

    method ClearPendingTransition()
        requires Invariants()
        modifies this
        ensures Invariants()
    {
        pendingTransition := None;
    }
}