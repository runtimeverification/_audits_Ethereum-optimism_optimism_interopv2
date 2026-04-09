include "Types.dfy"

class LogsDB {
    ghost function View() : seq<BlockRef>
        reads this

    ghost function LastTimestamp() : uint64
        reads this
        requires View() != []
    {
        var view := View();
        view[|view| - 1].Time
    }

    ghost predicate Invariants()
        reads this
    {
        var view := View();

        (forall i :: 0 < i < |view| ==> view[i - 1].ID.Number == view[i].ID.Number - 1) &&
        (forall i :: 0 < i < |view| ==> view[i - 1].ID.Hash == view[i].ParentHash) &&
        (forall i :: 0 < i < |view| ==> view[i - 1].Time <= view[i].Time) // Technically <, except for the first block
    }

    ghost predicate IsUpperBound(ts : uint64)
        reads this
    {
        forall block :: block in View() ==> block.Time <= ts
    }

    lemma LastTimestampIsUpperBound()
        requires Invariants()
        requires View() != []
        ensures IsUpperBound(LastTimestamp())
    {
        var view := View();

        assert 0 < |view|;
        assert view[|view| - 1].Time <= LastTimestamp();
        
        var i := |view| - 2;

        while 0 <= i
            invariant forall j :: (i < j < |view| && 0 <= j) ==> view[j].Time <= LastTimestamp()
        {
            assert view[i].Time <= view[i + 1].Time;

            i := i - 1;
        }
    }

    ghost function Find(number : uint64, list : seq<BlockRef>) : Option<nat>
        ensures match Find(number, list) {
            case None =>
                forall block :: block in list ==> block.ID.Number != number
            case Some(i) =>
                i < |list| && list[i].ID.Number == number
        }
    {
        var length := |list|;
        
        if length == 0 then
            None
        else if list[length - 1].ID.Number == number then
            Some(length - 1)
        else
            Find(number, list[..length - 1])
    }

    ghost function TimestampForBlock(number : uint64) : Option<uint64>
        reads this
    {
        var view := View();
        
        match Find(number, view) {
            case None => None
            case Some(index) => Some(view[index].Time)
        }
    }

    ghost predicate BlockTimestampIsAtMost(blockNumber : uint64, upperBound : uint64)
        reads this
    {
        TimestampForBlock(blockNumber) == None ||
        TimestampForBlock(blockNumber).value <= upperBound
    }
    
    function LatestSealedBlock() : Option<BlockID>
        reads this
        ensures{:axiom}
            var view := View();
            match LatestSealedBlock() {
                case None =>
                    view == []
                case Some(block) =>
                    0 < |view| && view[|view| - 1].ID == block
            }

    method Rewind(newHead : BlockID) returns (success : bool)
        modifies this
        ensures{:axiom} success ==>
            var oldView := old(View());
            var newView := View();
            var index := Find(newHead.Number, oldView);
            index != None &&
            oldView[index.value].ID == newHead &&
            newView == oldView[..index.value + 1]
        ensures{:axiom} !success ==> View() == old(View())
        // Note: The above might not be satisfied by the implementation, meaning that
        // if Rewind() fails, we can't guarantee the database state hasn't changed

    method Clear() returns (success : bool)
        modifies this
        ensures{:axiom} success ==> View() == []
        ensures{:axiom} !success ==> View() == old(View())
        // Note: Unsure if we can guarantee the above

    method Contains(query : ContainsQuery) returns (success : bool)
        ensures{:axiom}
            var view := View();
            var index := Find(query.BlockNum, view);
            assert index != None ==> view[index.value].ID.Number == query.BlockNum;
            success ==> (index != None && view[index.value].Time == query.Timestamp)

    method FindSealedBlock(number : uint64) returns (seal : Option<BlockSeal>)
        ensures{:axiom}
            match seal {
                case None =>
                    forall block :: block in View() ==> block.ID.Number != number
                case Some(block) =>
                    var view := View();
                    var index := Find(number, view);
                    index != None &&
                    view[index.value].ID.Number == block.Number &&
                    view[index.value].ID.Hash == block.Hash &&
                    view[index.value].Time == block.Timestamp &&
                    block.Number == number
            }

    method SealBlock(parentHash : Hash, block : BlockID, timestamp : uint64) returns (success : bool)
        modifies this
        ensures{:axiom}
            var oldView := old(View());
            var newView := View();
            if !success then
                newView == oldView // Assume state doesn't change on failure
            else
                newView == oldView + [BlockRef(block, parentHash, timestamp)] &&
                (oldView != [] ==> oldView[|oldView| - 1].ID.Hash == parentHash)

    method AddLog(log : Log, parentBlock : BlockID, logIdx : nat) returns (success : bool)
        modifies this
        ensures{:axiom} View() == old(View())
}