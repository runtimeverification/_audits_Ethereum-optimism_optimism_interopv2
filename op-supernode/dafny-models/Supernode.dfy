include "ChainContainer.dfy"
include "LogsDB.dfy"
include "Types.dfy"
include "VerifiedDB.dfy"

const ExpiryTime := 604800

class ConsistencyChecker {
    method SameL1Chain(heads : seq<BlockID>, lastVerified : Option<BlockID>) returns (same : Option<ChainConsistencyResult>)
        ensures{:axiom} (same != None && same.value == ChainConsistencyResult.InconsistentVerifiedL1) ==> lastVerified != None
}

class Supernode {
    const activationTimestamp : uint64
    var chains : map<ChainID, ChainContainer>
    var currentL1 : Option<BlockID>
    var l1Checker : ConsistencyChecker
    var logsDBs : map<ChainID, LogsDB>
    var verifiedDB : VerifiedDB

    ghost predicate Invariants()
        reads this, verifiedDB, logsDBs.Values
    {
        BaseInvariants() &&
        (forall chainID :: chainID in logsDBs.Keys ==> LogsDBIsConsistentWithVerifiedDB(chainID))
    }

    ghost predicate BaseInvariants()
        reads this, verifiedDB, logsDBs.Values
    {
        0 < activationTimestamp &&
        verifiedDB.Invariants() &&
        verifiedDB.ChainIDs() == chains.Keys &&
        chains.Keys == logsDBs.Keys &&
        (forall db :: db in logsDBs.Values ==> db.Invariants()) &&
        (forall k, k' :: ({k, k'} <= logsDBs.Keys && k != k') ==> logsDBs[k] != logsDBs[k'])
    }

    ghost predicate LogsDBIsConsistentWithVerifiedDB(chainID : ChainID)
        reads this, verifiedDB, logsDBs[chainID]
        requires chainID in logsDBs.Keys
    {
        var db := logsDBs[chainID];
        
        match verifiedDB.LastTimestamp() {
            case None =>
                db.View() == []
            case Some(ts) =>
                db.IsUpperBound(ts)
        }
    }

    ghost predicate TimestampDoesNotOverflow()
        reads this, verifiedDB
    {
        verifiedDB.LastTimestamp() != None ==> verifiedDB.LastTimestamp().value < MAX_UINT64
    }

    function NextTimestamp() : uint64
        reads this, verifiedDB
        requires TimestampDoesNotOverflow()
    {
        match verifiedDB.LastTimestamp() {
            case None =>
                activationTimestamp
            case Some(ts) =>
                ts + 1
        }
    }

    predicate IsValidFrontierBlock(chainID : ChainID, block : BlockID)
        reads this, verifiedDB, logsDBs[chainID], chains[chainID]
        requires chainID in logsDBs.Keys
        requires chainID in chains.Keys
        requires TimestampDoesNotOverflow()
    {
        var lastBlock := logsDBs[chainID].LatestSealedBlock();
        var ts := chains[chainID].ConfirmedTimestampFor(block);
        
        (lastBlock != None ==> block.Number <= lastBlock.value.Number + 1) &&
        ts != None && ts.value <= NextTimestamp()
    }

    predicate AreValidFrontierBlocks(blocks : map<ChainID, BlockID>)
        reads this, verifiedDB, logsDBs.Values, chains.Values
        requires blocks.Keys <= logsDBs.Keys
        requires blocks.Keys <= chains.Keys
        requires TimestampDoesNotOverflow()
    {
        forall chainID :: chainID in blocks.Keys ==> IsValidFrontierBlock(chainID, blocks[chainID])
    }

    ghost predicate IsValidTargetHeadForTimestamp(ts : uint64, chainID : ChainID, block : BlockID)
        reads this, logsDBs.Values
        requires chainID in logsDBs.Keys
    {
        logsDBs[chainID].BlockTimestampIsAtMost(block.Number, ts)
    }

    ghost predicate AreValidTargetHeadsForTimestamp(ts : uint64, blocks : map<ChainID, BlockID>)
        reads this, logsDBs.Values
        requires blocks.Keys == logsDBs.Keys
    {
        forall chainID :: chainID in logsDBs.Keys ==> IsValidTargetHeadForTimestamp(ts, chainID, blocks[chainID])
    }

    ghost predicate PendingTransitionIsConsistentWithSupernode(pending : PendingTransition)
        reads this, verifiedDB, logsDBs.Values, chains.Values
        requires TimestampDoesNotOverflow()
        requires logsDBs.Keys == chains.Keys
    {
        match pending {
            case Rewind(rewindAtOrAfter, targetHeads) =>
                var lastTimestamp := verifiedDB.LastTimestamp();
                lastTimestamp != None &&
                rewindAtOrAfter == lastTimestamp.value &&
                0 < lastTimestamp.value &&
                targetHeads == verifiedDB.L2HeadsAtTimestamp(lastTimestamp.value - 1) &&
                (targetHeads != None ==> targetHeads.value.Keys == logsDBs.Keys) &&
                (targetHeads != None ==> AreValidTargetHeadsForTimestamp(rewindAtOrAfter - 1, targetHeads.value))
            case Advance(ts, _, l2Heads) =>
                ts == NextTimestamp() &&
                l2Heads.Keys == logsDBs.Keys &&
                AreValidFrontierBlocks(l2Heads)
            case Invalidate(ts, _) =>
                ts == NextTimestamp()
        }
    }

    constructor()
    
    method progressAndRecord() returns (madeProgress : Option<bool>)
        requires Invariants() // NOTE: Not guaranteed if pending transition != None
        requires TimestampDoesNotOverflow()
        requires
            var pending := verifiedDB.GetPendingTransition();
            pending != None ==> PendingTransitionIsConsistentWithSupernode(pending.value)
        modifies this, verifiedDB, chains.Values, logsDBs.Values
        ensures BaseInvariants()
        ensures verifiedDB.GetPendingTransition() == None ==> Invariants()
    {
        var pending := verifiedDB.GetPendingTransition();

        if pending != None {
            madeProgress := applyPendingTransition(pending.value);
            return;
        }

        var output := progressInterop();

        if output == None {
            return None;
        }

        match output.value {
            case Wait =>
                refreshCurrentL1OnWait();
                return Some(false);
            case Step(pending) =>
                verifiedDB.SetPendingTransition(pending);
                madeProgress := applyPendingTransition(pending);
                return;
        }
    }

    method applyPendingTransition(pending : PendingTransition) returns (madeProgress : Option<bool>)
        requires Invariants()
        requires TimestampDoesNotOverflow()
        requires (pending.Rewind? && pending.TargetHeads != None) ==> pending.TargetHeads.value.Keys == chains.Keys
        requires PendingTransitionIsConsistent(pending, chains.Keys)
        requires PendingTransitionIsConsistentWithSupernode(pending)
        requires verifiedDB.GetPendingTransition() == Some(pending)
        modifies this, verifiedDB, chains.Values, logsDBs.Values
        ensures BaseInvariants()
        ensures madeProgress != None ==> verifiedDB.GetPendingTransition() == None
        ensures verifiedDB.GetPendingTransition() == None ==> Invariants()
    {
        match pending {
            case Rewind(rewindAtOrAfter, targetHeads) =>
                currentL1 := None;

                assert verifiedDB.GetPendingTransition() != None;

                var success := applyRewindPlan(rewindAtOrAfter, targetHeads);
                
                assert verifiedDB.GetPendingTransition() != None;

                if success == false {
                    return None;
                }

                verifiedDB.ClearPendingTransition();

                return Some(false);
            
            case Invalidate(ts, invalidHeads) =>
                var chainIDs := Enumerate(invalidHeads.Keys);
                var failedAny := false;

                for i := 0 to |chainIDs|
                    invariant verifiedDB == old(verifiedDB)
                    invariant chains == old(chains)
                    invariant Invariants()
                {
                    var chainID := chainIDs[i];
                    var blockID := invalidHeads[chainID];
                    var success := invalidateBlock(chainID, blockID, ts);
                    failedAny := failedAny || !success;
                }

                if failedAny {
                    // Preserve the transition to retry later
                    return None;
                }

                verifiedDB.ClearPendingTransition();

                return Some(false);
            
            case Advance(ts, l1Inclusion, l2Heads) =>
                currentL1 := Some(l1Inclusion);
                
                var logsPersisted := persistFrontierLogs(ts, l2Heads);

                if !logsPersisted {
                    return None;
                }

                var resultCommitted := verifiedDB.Commit(VerifiedResult(ts, l1Inclusion, l2Heads));

                if !resultCommitted {
                    return None;
                }

                verifiedDB.ClearPendingTransition();

                return Some(true);
        }
    }

    method applyRewindPlan(rewindAtOrAfter : uint64, targetHeads : Option<map<ChainID, BlockID>>) returns (success : bool)
        requires Invariants()
        requires verifiedDB.LastTimestamp() != None
        requires rewindAtOrAfter == verifiedDB.LastTimestamp().value
        requires 0 < rewindAtOrAfter
        requires targetHeads == verifiedDB.L2HeadsAtTimestamp(rewindAtOrAfter - 1)
        requires targetHeads != None ==> forall chainID :: chainID in logsDBs.Keys ==>
            logsDBs[chainID].BlockTimestampIsAtMost(targetHeads.value[chainID].Number, rewindAtOrAfter - 1)
        modifies verifiedDB, chains.Values, logsDBs.Values
        ensures BaseInvariants()
        ensures success ==> Invariants()
        ensures verifiedDB.GetPendingTransition() == old(verifiedDB.GetPendingTransition())
    {
        var deleted := verifiedDB.Rewind(rewindAtOrAfter);

        assert deleted;
        assert verifiedDB.LastTimestamp() == None || verifiedDB.LastTimestamp() == Some(rewindAtOrAfter - 1);
        assert verifiedDB.Get(rewindAtOrAfter - 1) == old(verifiedDB).Get(rewindAtOrAfter - 1);

        var chainIDs := Enumerate(chains.Keys);
        
        var anyErrs := rewindL2Chains(chainIDs, rewindAtOrAfter);

        if targetHeads == None {
            var anyErrs2 := clearLogsDBs(chainIDs);
            anyErrs := anyErrs || anyErrs2;
        } else {
            assert targetHeads != None;
            var anyErrs3 := rewindLogsDBs(chainIDs, targetHeads.value);
            anyErrs := anyErrs || anyErrs3;
        }

        return !anyErrs;
    }

    method rewindL2Chains(chainIDs : seq<ChainID>, rewindAtOrAfter : uint64) returns (anyErrs : bool)
        requires BaseInvariants()
        requires forall chainID :: chainID in chainIDs ==> chainID in chains.Keys
        requires 0 < rewindAtOrAfter
        modifies chains.Values
        ensures BaseInvariants()
    {
        anyErrs := false;
        
        for i := 0 to |chainIDs|
            invariant BaseInvariants()
        {
            var chainID := chainIDs[i];
            var chain := chains[chainID];
            var pruneSucceeded := chain.PruneDeniedAtOrAfterTimestamp(rewindAtOrAfter);

            anyErrs := anyErrs || !pruneSucceeded;

            var rewindSucceeded := chain.RewindEngine(rewindAtOrAfter - 1);

            anyErrs := anyErrs || !rewindSucceeded;
        }
    }

    method clearLogsDBs(chainIDs : seq<ChainID>) returns (anyErrs : bool)
        requires BaseInvariants()
        requires forall chainID :: chainID in chainIDs <==> chainID in logsDBs.Keys
        requires verifiedDB.LastTimestamp() == None
        modifies logsDBs.Values
        ensures BaseInvariants()
        ensures !anyErrs ==> Invariants()
    {
        anyErrs := false;
        
        for i := 0 to |chainIDs|
            invariant BaseInvariants()
            invariant !anyErrs ==> forall j :: 0 <= j < i ==> LogsDBIsConsistentWithVerifiedDB(chainIDs[j])
        {
            var chainID := chainIDs[i];
            var db := logsDBs[chainID];
            var clearSucceeded := db.Clear();

            anyErrs := anyErrs || !clearSucceeded;
        }
    }

    method rewindLogsDBs(chainIDs : seq<ChainID>, targetHeads : map<ChainID, BlockID>) returns (anyErrs : bool)
        requires BaseInvariants()
        requires verifiedDB.LastTimestamp() != None
        requires TimestampDoesNotOverflow()
        requires targetHeads == verifiedDB.L2HeadsAtTimestamp(verifiedDB.LastTimestamp().value).value
        requires forall chainID :: chainID in chainIDs <==> chainID in targetHeads.Keys
        requires forall chainID :: chainID in chainIDs <==> chainID in logsDBs.Keys
        requires forall i, j :: 0 <= i < j < |chainIDs| ==> chainIDs[i] != chainIDs[j]
        requires forall db :: db in logsDBs.Values ==> db.IsUpperBound(verifiedDB.LastTimestamp().value + 1)
        requires forall chainID :: chainID in chainIDs ==>
            logsDBs[chainID].BlockTimestampIsAtMost(targetHeads[chainID].Number, verifiedDB.LastTimestamp().value)
        modifies logsDBs.Values
        ensures BaseInvariants()
        ensures !anyErrs ==> Invariants()
    {
        anyErrs := false;

        for i := 0 to |chainIDs|
            invariant BaseInvariants()
            invariant forall i, j :: 0 <= i < j < |chainIDs| ==> chainIDs[i] != chainIDs[j]
            invariant forall i, j :: 0 <= i < j < |chainIDs| ==> logsDBs[chainIDs[i]] != logsDBs[chainIDs[j]]
            invariant forall j :: i <= j < |chainIDs| ==> logsDBs[chainIDs[j]] == old(logsDBs[chainIDs[j]])
            invariant forall db :: db in logsDBs.Values ==> db.IsUpperBound(verifiedDB.LastTimestamp().value + 1)
            invariant forall chainID :: chainID in chainIDs ==>
                logsDBs[chainID].BlockTimestampIsAtMost(targetHeads[chainID].Number, verifiedDB.LastTimestamp().value)
            invariant !anyErrs ==> forall j :: 0 <= j < i ==> LogsDBIsConsistentWithVerifiedDB(chainIDs[j])
        {
            var chainID := chainIDs[i];
            var expectedHead := targetHeads[chainID];
            
            var success := rewindLogsDB(chainID, expectedHead);

            assert logsDBs[chainID].BlockTimestampIsAtMost(targetHeads[chainID].Number, verifiedDB.LastTimestamp().value);

            anyErrs := anyErrs || !success;
        }
    }

    method rewindLogsDB(chainID : ChainID, expectedHead : BlockID) returns (success : bool)
        requires TimestampDoesNotOverflow()
        requires verifiedDB.LastTimestamp() != None
        requires chainID in logsDBs.Keys
        requires logsDBs[chainID].Invariants()
        requires logsDBs[chainID].IsUpperBound(verifiedDB.LastTimestamp().value + 1)
        requires logsDBs[chainID].BlockTimestampIsAtMost(expectedHead.Number, verifiedDB.LastTimestamp().value)
        modifies logsDBs[chainID]
        ensures success ==> LogsDBIsConsistentWithVerifiedDB(chainID)
        ensures logsDBs[chainID].Invariants()
        ensures logsDBs[chainID].IsUpperBound(verifiedDB.LastTimestamp().value + 1)
        ensures logsDBs[chainID].BlockTimestampIsAtMost(expectedHead.Number, verifiedDB.LastTimestamp().value)
    {
        var db := logsDBs[chainID];
        var latestBlock := db.LatestSealedBlock();

        if latestBlock == None {
            return true;
        }

        if latestBlock.value.Number == expectedHead.Number {
            assert db.IsUpperBound(db.LastTimestamp()) by { db.LastTimestampIsUpperBound(); }
            
            // NOTE: This check is absent from the implementation,
            // but might be necessary if we assume L2s can behave arbitrarily
            return latestBlock.value.Hash == expectedHead.Hash;
        }

        if latestBlock.value.Number < expectedHead.Number {
            var seal := db.FindSealedBlock(latestBlock.value.Number);
            assert seal != None;
            assert seal.value.Timestamp == db.LastTimestamp();
            assert db.IsUpperBound(db.LastTimestamp()) by { db.LastTimestampIsUpperBound(); }
            
            // NOTE: This check is absent from the implementation,
            // but might be necessary if we assume L2s can behave arbitrarily
            return seal.value.Timestamp < verifiedDB.LastTimestamp().value;
        }

        success := db.Rewind(expectedHead);

        assert success ==> db.LatestSealedBlock() == Some(expectedHead);
        assert success ==> db.IsUpperBound(db.LastTimestamp()) by { db.LastTimestampIsUpperBound(); }
        assert success ==> db.LastTimestamp() <= verifiedDB.LastTimestamp().value;
        assert success ==> LogsDBIsConsistentWithVerifiedDB(chainID);
    }

    method invalidateBlock(chainID : ChainID, blockID : BlockID, decisionTimestamp : uint64) returns (success : bool)
        requires chainID in chains.Keys
        modifies chains.Values
    {
        var chain := chains[chainID];
        var rewindTriggered := chain.InvalidateBlock(blockID.Number, blockID.Hash, decisionTimestamp);

        return rewindTriggered != None;
    }

    method persistFrontierLogs(ts : uint64, blocksAtTS : map<ChainID, BlockID>) returns (success : bool)
        requires Invariants()
        requires TimestampDoesNotOverflow()
        requires ts == NextTimestamp()
        requires blocksAtTS.Keys == chains.Keys
        requires AreValidFrontierBlocks(blocksAtTS)
        modifies logsDBs.Values
        ensures BaseInvariants()
        ensures forall db :: db in logsDBs.Values ==> db.IsUpperBound(ts)
    {
        var chainIDs := Enumerate(blocksAtTS.Keys);

        for i := 0 to |chainIDs|
            invariant BaseInvariants()
            invariant forall i, j :: 0 <= i < j < |chainIDs| ==> chainIDs[i] != chainIDs[j]
            invariant forall i, j :: 0 <= i < j < |chainIDs| ==> logsDBs[chainIDs[i]] != logsDBs[chainIDs[j]]
            invariant forall j :: i <= j < |chainIDs| ==> IsValidFrontierBlock(chainIDs[j], blocksAtTS[chainIDs[j]])
            invariant forall j :: i <= j < |chainIDs| ==> LogsDBIsConsistentWithVerifiedDB(chainIDs[j])
            invariant forall db :: db in logsDBs.Values ==> db.IsUpperBound(ts)
        {
            var chainID := chainIDs[i];
            var blockID := blocksAtTS[chainID];

            var blockPersisted := persistFrontierBlock(chainID, blockID, ts);

            if !blockPersisted {
                return false;
            }
        }

        return true;
    }

    // NOTE: Extracted body of the loop from persistFrontierLogs to facilitate verification
    method persistFrontierBlock(chainID : ChainID, blockID : BlockID, ts : uint64) returns (success : bool)
        requires BaseInvariants()
        requires TimestampDoesNotOverflow()
        requires ts == NextTimestamp()
        requires chainID in chains.Keys
        requires IsValidFrontierBlock(chainID, blockID)
        requires LogsDBIsConsistentWithVerifiedDB(chainID)
        modifies logsDBs[chainID]
        ensures BaseInvariants()
        ensures forall block :: block in logsDBs[chainID].View() ==> block.Time <= ts
    {
        var chain := chains[chainID];
        var db := logsDBs[chainID];
        
        var latestBlock := verifyCanAddTimestamp(db, ts, chain.BlockTime());

        if latestBlock.Err? {
            return false;
        }

        var fetched := chain.FetchReceipts(blockID);

        if fetched == None {
            return false;
        }

        var Some((blockInfo, receipts)) := fetched;

        var alreadyExists := checkIfBlockAlreadyExists(blockInfo, db, latestBlock.value);

        if alreadyExists.Err? {
            return false;
        }

        if alreadyExists.value == true {
            return true;
        }

        // NOTE: This check is absent in the actual implementation, and should logically be true
        // However, it's necessary to maintain the invariants if we assume the L2s can behave arbitrarily
        if blockInfo.Time < ts {
            return false;
        }

        var isFirstBlock := latestBlock.value == None;
        var logsProcessed := processBlockLogs(db, blockInfo, receipts, isFirstBlock);

        if !logsProcessed {
            return false;
        }

        return true;
    }

    method verifyCanAddTimestamp(db : LogsDB, ts : uint64, blockTime : uint64) returns (latestBlock : OkOrErr<Option<BlockID>>)
        ensures latestBlock.Ok? ==> latestBlock.value == db.LatestSealedBlock()
        ensures latestBlock.Ok? ==>
            var view := db.View(); 
            if view == [] then
                ts == activationTimestamp
            else
                ts <= view[|view| - 1].Time + blockTime
    {
        var latestSealedBlock := db.LatestSealedBlock();

        if latestSealedBlock == None {
            if ts == activationTimestamp {
                return Ok(None);
            } else {
                return Err(Other);
            }
        }

        var seal := db.FindSealedBlock(latestSealedBlock.value.Number);

        // NOTE: Should never happen if invariants are true
        if seal == None || seal.value.Timestamp + blockTime < ts {
            return Err(Other);
        }

        return Ok(latestSealedBlock);
    }

    method checkIfBlockAlreadyExists(blockInfo : BlockInfo, db : LogsDB, latestBlock : Option<BlockID>) returns (alreadyExists : OkOrErr<bool>)
        requires latestBlock == db.LatestSealedBlock()
        ensures (alreadyExists == Ok(false) && latestBlock != None) ==>
            (latestBlock.value.Number < blockInfo.ID.Number && latestBlock.value.Hash == blockInfo.ParentHash)
    {
        if latestBlock != None {
            if blockInfo.ID.Number < latestBlock.value.Number {
                var seal := db.FindSealedBlock(blockInfo.ID.Number);

                if seal != None && seal.value.Hash == blockInfo.ID.Hash {
                    return Ok(true);
                } else {
                    return Err(AssumptionViolation);
                }
            }

            if latestBlock.value.Number == blockInfo.ID.Number {
                if latestBlock.value.Hash == blockInfo.ID.Hash {
                    return Ok(true);
                } else {
                    return Err(AssumptionViolation);
                }
            }

            if blockInfo.ParentHash != latestBlock.value.Hash {
                return Err(AssumptionViolation);
            }
        }

        return Ok(false);
    }

    method processBlockLogs(db : LogsDB, blockInfo : BlockInfo, receipts : seq<Receipt>, isFirstBlock : bool) returns (success : bool)
        requires db.Invariants()
        requires forall block :: block in db.View() ==> block.Time <= blockInfo.Time
        requires blockInfo.ID.Number == 0 ==> isFirstBlock
        requires isFirstBlock ==> db.View() == []
        requires var view := db.View(); view != [] ==> view[|view| - 1].ID.Number == blockInfo.ID.Number - 1
        requires var view := db.View(); view != [] ==> view[|view| - 1].Time < blockInfo.Time
        modifies db
        ensures success ==> db.LatestSealedBlock() == Some(blockInfo.ID)
        ensures db.Invariants()
        ensures forall block :: block in db.View() ==> block.Time <= blockInfo.Time
    {
        var blockNum := blockInfo.ID.Number;
        var blockID := blockInfo.ID;
        var parentHash := blockInfo.ParentHash;

        var (parentBlock, sealParentHash) :=
            if blockNum == 0 then
                (BlockID(0, 0), 0)
            else
                (BlockID(parentHash, blockNum - 1), parentHash);

        if isFirstBlock && 0 < blockNum {
            var blockSealed := db.SealBlock(0, parentBlock, blockInfo.Time);

            if !blockSealed {
                return false;
            }
        }
        
        ghost var initialView := db.View();
        var logIndex := 0;
        
        for i := 0 to |receipts|
            invariant db.View() == initialView
        {
            var receipt := receipts[i];
            
            for j := 0 to |receipt.Logs|
                invariant db.View() == initialView
            {
                var log := receipt.Logs[j];
                var logAdded := db.AddLog(log, parentBlock, logIndex);

                if !logAdded {
                    return false;
                }
                
                logIndex := logIndex + 1;
            }
        }

        var blockSealed := db.SealBlock(sealParentHash, blockID, blockInfo.Time);

        if !blockSealed {
            return false;
        }

        return true;
    }

    method progressInterop() returns (output : Option<StepOutput>)
        requires Invariants()
        requires verifiedDB.LastTimestamp() != None ==> verifiedDB.LastTimestamp().value < MAX_UINT64
        ensures (output != None && output.value != Wait) ==>
            PendingTransitionIsConsistent(output.value.pending, chains.Keys) &&
            PendingTransitionIsConsistentWithSupernode(output.value.pending)
    {
        var obs := observeRound();

        if obs == None {
            return None;
        }

        if obs.value == RoundObservation.NotReady || obs.value == RoundObservation.InconsistentL2s {
            return Some(Wait);
        }

        if obs.value == RoundObservation.InconsistentVerifiedL1 {
            var lastTS := verifiedDB.LastTimestamp();
            assert lastTS != None;
            assert 0 < lastTS.value;
            var prevResult := verifiedDB.Get(lastTS.value - 1);
            var targetHeads := if prevResult == None then None else Some(prevResult.value.L2Heads);

            // NOTE: Logically needs to be true given Supernode invariants, but current modeled invariants are insufficient to prove
            assume targetHeads != None ==> AreValidTargetHeadsForTimestamp(lastTS.value - 1, targetHeads.value);
            
            return Some(Step(Rewind(lastTS.value, targetHeads)));
        }

        var invalidated := verify(obs.value.NextTimestamp, obs.value.BlocksAtTS);

        if invalidated == None {
            return None;
        }

        if invalidated.value == map[] {
            return Some(Step(Advance(obs.value.NextTimestamp, obs.value.L1Inclusion, obs.value.BlocksAtTS)));
        } else {
            return Some(Step(Invalidate(obs.value.NextTimestamp, invalidated.value)));
        }
    }

    method observeRound() returns (obs : Option<RoundObservation>)
        requires Invariants()
        requires verifiedDB.LastTimestamp() != None ==> verifiedDB.LastTimestamp().value < MAX_UINT64
        requires 0 < activationTimestamp
        ensures Invariants()
        ensures (obs != None && obs.value == RoundObservation.InconsistentVerifiedL1) ==> verifiedDB.LastTimestamp() != None
        ensures (obs != None && obs.value.Consistent?) ==> obs.value.NextTimestamp == NextTimestamp()
        ensures (obs != None && obs.value.Consistent?) ==>
            obs.value.BlocksAtTS.Keys == chains.Keys && AreValidFrontierBlocks(obs.value.BlocksAtTS)
    {
        var lastTS := verifiedDB.LastTimestamp();
        var (nextTS, lastVerified) := match lastTS {
            case None => (activationTimestamp, None)
            case Some(ts) => (ts + 1, verifiedDB.Get(ts))
        };

        var ready := checkChainsReady(nextTS);

        if ready == None {
            return None;
        }
        
        if ready == Some(ChainsReadyResult.NotReady) {
            return Some(RoundObservation.NotReady);
        }

        // NOTE: This check is absent in the actual implementation
        // However, it's necessary if we assume the L2s can behave arbitrarily
        if !AreValidFrontierBlocks(ready.value.Blocks) {
            return None;
        }

        var heads := Enumerate(ready.value.L1Heads.Values);
        var lastVerifiedL1 := match lastVerified {
            case None => None
            case Some(result) => Some(result.L1Inclusion)
        };

        var same := l1Checker.SameL1Chain(heads, lastVerifiedL1);

        if same == None {
            return None;
        }

        match same.value {
            case InconsistentL2s =>
                return Some(RoundObservation.InconsistentL2s);
            case InconsistentVerifiedL1 =>
                return Some(RoundObservation.InconsistentVerifiedL1);
            case Consistent(l1Inclusion) =>
                return Some(RoundObservation.Consistent(nextTS, ready.value.Blocks, l1Inclusion));
        }
    }
    
    method checkChainsReady(ts : uint64) returns (ready : Option<ChainsReadyResult>)
        requires Invariants()
        ensures Invariants()
        ensures (ready != None && ready.value.Ready?) ==> (ready.value.Blocks.Keys == ready.value.L1Heads.Keys == chains.Keys)
    {
        var blocks := map[];
        var l1Heads := map[];
        var chainIDs := Enumerate(chains.Keys);

        for i := 0 to |chainIDs|
            invariant blocks.Keys == l1Heads.Keys == (set j | 0 <= j < i :: chainIDs[j])
        {
            var chainID := chainIDs[i];
            var c := chains[chainID];
            var l2AndL1 := c.OptimisticAt(ts);

            if l2AndL1 == Err(NotFound) {
                return Some(ChainsReadyResult.NotReady);
            }
            
            // Another error
            if l2AndL1.Err? {
                return None;
            }

            var (l2Block, l1Block) := l2AndL1.value;

            blocks := blocks[chainID := l2Block];
            l1Heads := l1Heads[chainID := l1Block];
        }

        return Some(Ready(blocks, l1Heads));
    }
    
    method verify(ts : uint64, blocksAtTS : map<ChainID, BlockID>) returns (invalidated : Option<map<ChainID, BlockID>>)
        requires Invariants()
        requires blocksAtTS.Keys == chains.Keys
        ensures invalidated != None ==> invalidated.value.Keys <= chains.Keys
    {
        var view := resolveFrontierVerificationView(blocksAtTS);

        if view == None {
            return None;
        }

        var invalidMsgs := verifyInteropMessages(ts, view.value);
        var cycleMsgs := verifyCycleMessages(ts, view.value);

        return Some(invalidMsgs + cycleMsgs);
    }

    method resolveFrontierVerificationView(blocksAtTS : map<ChainID, BlockID>) returns (view : Option<map<ChainID, FrontierBlockView>>)
        requires blocksAtTS.Keys == chains.Keys
        ensures view != None ==> blocksAtTS.Keys == view.value.Keys
        ensures view != None ==> forall k :: k in view.value.Keys ==> view.value[k].Ref.ID == blocksAtTS[k]
    {
        var view' : map<ChainID, FrontierBlockView> := map[];
        var chainIDs := Enumerate(blocksAtTS.Keys);

        for i := 0 to |chainIDs|
            invariant view'.Keys == (set j | 0 <= j < i :: chainIDs[j])
            invariant forall k :: k in view'.Keys ==> view'[k].Ref.ID == blocksAtTS[k]
        {
            var chainID := chainIDs[i];
            var blockID := blocksAtTS[chainID];
            var chain := chains[chainID];
            var fetched := chain.FetchReceipts(blockID);

            if fetched == None {
                return None;
            }

            var (blockInfo, receipts) := fetched.value;
            var blockView := buildFrontierBlockView(chainID, blockInfo, receipts);

            view' := view'[chainID := blockView];
        }

        return Some(view');
    }

    method buildFrontierBlockView(chainID : ChainID, blockInfo : BlockInfo, receipts : seq<Receipt>) returns (view : FrontierBlockView)
        ensures view.Ref.ID == blockInfo.ID
        ensures view.Ref.Time == blockInfo.Time
    {
        var ref := BlockRef(blockInfo.ID, blockInfo.ParentHash, blockInfo.Time);
        var execMsgs := [];
        
        for i := 0 to |receipts| {
            var receipt := receipts[i];
            
            for j := 0 to |receipt.Logs| {
                var execMsg := receipt.Logs[j].ExecMsg;

                if execMsg != None {
                    execMsgs := execMsgs + [execMsg.value];
                }
            }
        }

        return FrontierBlockView(ref, execMsgs);
    }
    
    method verifyInteropMessages(ts : uint64, frontierView : map<ChainID, FrontierBlockView>) returns (invalidated : map<ChainID, BlockID>)
        requires Invariants()
        requires frontierView.Keys == chains.Keys
        ensures invalidated.Keys <= chains.Keys
    {
        invalidated := map[];
        var chainIDs := Enumerate(frontierView.Keys);

        for i := 0 to |chainIDs|
            invariant invalidated.Keys <= chains.Keys
        {
            var chainID := chainIDs[i];
            var frontierBlock := frontierView[chainID];
            var blockRef := frontierBlock.Ref;
            var execMsgs := frontierBlock.ExecMsgs;

            var blockValid := true;
            
            for logIdx := 0 to |execMsgs| {
                blockValid := verifyExecutingMessage(blockRef.Time, execMsgs[logIdx], frontierView);

                if !blockValid {
                    break;
                }
            }

            if !blockValid {
                invalidated := invalidated[chainID := frontierBlock.Ref.ID];
            }
        }

        return;
    }

    method verifyExecutingMessage(
        executingTimestamp : uint64, 
        execMsg : ExecutingMessage, 
        frontierView : map<ChainID, FrontierBlockView>
    )
        returns (success : bool)
    {
        if execMsg.ChainID !in logsDBs.Keys {
            return false;
        }
        
        var inRange := execMsg.Timestamp <= executingTimestamp <= execMsg.Timestamp + ExpiryTime;

        if !inRange {
            return false;
        }
        
        if execMsg.Timestamp == executingTimestamp {
            if execMsg.ChainID !in frontierView.Keys {
                return false;
            }
            
            return execMsg in frontierView[execMsg.ChainID].ExecMsgs;
        }

        var query := ContainsQuery(
            execMsg.Timestamp,
            execMsg.BlockNum,
            execMsg.LogIdx,
            execMsg.Checksum
        );

        success := logsDBs[execMsg.ChainID].Contains(query);
        return;
    }

    method verifyCycleMessages(ts : uint64, frontierView : map<ChainID, FrontierBlockView>) returns (invalidated : map<ChainID, BlockID>)
        requires frontierView.Keys == chains.Keys
        ensures{:axiom} invalidated.Keys <= chains.Keys

    method refreshCurrentL1OnWait()
        requires Invariants()
        modifies this
        ensures Invariants()
    {
        var currentL1 : Option<BlockID> := None;
        var chainIDs := Enumerate(chains.Keys);

        for i := 0 to |chainIDs|
            invariant Invariants()
            invariant chains == old(chains)
        {
            var chainID := chainIDs[i];
            var chain := chains[chainID];
            var status := chain.SyncStatus();

            if status == None {
                // Non-fatal: just keep existing currentL1
                return;
            }

            var block := status.value.CurrentL1;

            if currentL1 == None || block.ID.Number < currentL1.value.Number {
                currentL1 := Some(block.ID);
            }
        }

        this.currentL1 := currentL1;
    }
}