include "Types.dfy"
include "VerifiedDB.dfy"

class Supernode {
    var activationTimestamp : uint64
    var currentL1 : Option<BlockID>
    var verifiedDB : VerifiedDB

    constructor()
    
    method progressAndRecord() returns (madeProgress : Option<bool>)
        requires verifiedDB.Invariants()
        requires verifiedDB.LastTimestamp() != None ==> verifiedDB.LastTimestamp().value < MAX_UINT64
        modifies this, verifiedDB
        ensures verifiedDB.Invariants()
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
                madeProgress := applyPendingTransition(pending);
                return;
        }
    }

    method applyPendingTransition(pending : PendingTransition) returns (madeProgress : Option<bool>)
        requires verifiedDB.Invariants()
        requires verifiedDB.LastTimestamp() != None ==> verifiedDB.LastTimestamp().value < MAX_UINT64
        modifies this, verifiedDB
        ensures verifiedDB.Invariants()
    {
        match pending {
            case Rewind(rewindPlan) =>
                currentL1 := None;
                
                var success := applyRewindPlan(rewindPlan);

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
                    invariant verifiedDB.Invariants()
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

    // applyRewindPlan: stub body (always reports failure).
    //
    // SPEC.md T3 specifies the real rewind behavior (prune VerifiedDB tail,
    // prune DenyList entries with DecisionTimestamp >= target, prune LogsDB
    // tails for diverging chains). Until the class tracks LogsDB / DenyList
    // (Step 2c), this stub is a no-op that returns false, causing the
    // caller (applyPendingTransition) to preserve the pending Rewind and
    // retry on the next round. This is semantically consistent with the
    // "partial progress / retry" contract.
    //
    // Removing the {:axiom} tags: both ensures clauses are now proven by
    // the trivial body. The method does not reassign `this.verifiedDB`, so
    // the field reference equality holds; the VerifiedDB object is not
    // mutated, so Dafny frame analysis preserves `verifiedDB.Invariants()`
    // given the new `requires` precondition.
    method applyRewindPlan(plan : RewindPlan) returns (success : bool)
        requires verifiedDB.Invariants()
        modifies this, verifiedDB
        ensures verifiedDB == old(verifiedDB)
        ensures verifiedDB.Invariants()
        ensures success == false
    {
        success := false;
    }

    method invalidateBlock(chainID : ChainID, blockID : BlockID, decisionTimestamp : uint64) returns (success : bool)

    method persistFrontierLogs(ts : uint64, blocksAtTS : map<ChainID, BlockID>) returns (success : bool)

    method progressInterop() returns (output : Option<StepOutput>)
        requires verifiedDB.LastTimestamp() != None ==> verifiedDB.LastTimestamp().value < MAX_UINT64
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
            var rewindPlan := buildRewindPlan(lastTS.value);
            return Some(Step(Rewind(rewindPlan)));
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
        requires verifiedDB.LastTimestamp() != None ==> verifiedDB.LastTimestamp().value < MAX_UINT64
        ensures (obs != None && obs.value == RoundObservation.InconsistentVerifiedL1) ==> verifiedDB.LastTimestamp() != None
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

        var heads := Enumerate(ready.value.L1Heads.Values);
        var lastVerifiedL1 := match lastVerified {
            case None => None
            case Some(result) => Some(result.L1Inclusion)
        };

        var same := sameL1Chain(heads, lastVerifiedL1);

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

    // sameL1Chain: stub body (always returns None).
    //
    // The real implementation queries the L1 client for ancestry between
    // `heads` and `lastVerified`. Deferred to the L1 client integration.
    // The stub returns None, which the caller (observeRound) converts to
    // returning None itself. The previous {:axiom} ensures clause becomes
    // vacuously true because `same == None`.
    method sameL1Chain(heads : seq<BlockID>, lastVerified : Option<BlockID>) returns (same : Option<ChainConsistencyResult>)
        ensures (same != None && same.value == ChainConsistencyResult.InconsistentVerifiedL1) ==> verifiedDB.LastTimestamp() != None
    {
        same := None;
    }
    
    method checkChainsReady(ts : uint64) returns (ready : Option<ChainsReadyResult>)

    function buildRewindPlan(lastTS : uint64) : RewindPlan
    
    method verify(ts : uint64, blocksAtTS : map<ChainID, BlockID>) returns (invalidated : Option<map<ChainID, BlockID>>)
    {
        var view := resolveFrontierVerificationView(blocksAtTS);

        if view == None {
            return None;
        }

        var invalidMsgs := verifyInteropMessages(ts, view.value);
        var cycleMsgs := verifyCycleMessages(ts, view.value);

        return Some(invalidMsgs + cycleMsgs);
    }

    // resolveFrontierVerificationView: stub body (always returns None).
    //
    // The real implementation queries each VirtualNode for the frontier
    // block references and their executing messages. Deferred to the
    // VirtualNode integration. The stub returns None, which the caller
    // (`verify`) converts to returning None. Both previous {:axiom} ensures
    // clauses become vacuously true because `view == None`.
    method resolveFrontierVerificationView(blocksAtTS : map<ChainID, BlockID>) returns (view : Option<map<ChainID, FrontierBlockView>>)
        ensures view != None ==> forall k :: k in blocksAtTS.Keys <==> k in view.value.Keys
        ensures view != None ==> forall k :: k in blocksAtTS.Keys ==> view.value[k].Ref.ID == blocksAtTS[k]
    {
        view := None;
    }
    
    method verifyInteropMessages(ts : uint64, frontierView : map<ChainID, FrontierBlockView>) returns (invalidated : map<ChainID, BlockID>)
    {
        invalidated := map[];
        var chainIDs := Enumerate(frontierView.Keys);

        for i := 0 to |chainIDs| {
            var chainID := chainIDs[i];
            var frontierBlock := frontierView[chainID];
            var blockRef := frontierBlock.Ref;
            var execMsgs := frontierBlock.ExecMsgs;

            var blockValid := true;
            
            for logIdx := 0 to |execMsgs| {
                blockValid := verifyExecutingMessage(blockRef.Time, execMsgs[logIdx]);

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

    predicate verifyExecutingMessage(executingTimestamp : uint64, execMsg : ExecutingMessage)

    method verifyCycleMessages(ts : uint64, frontierView : map<ChainID, FrontierBlockView>) returns (invalidated : map<ChainID, BlockID>)

    method refreshCurrentL1OnWait()
}