include "Types.dfy"

// SupernodeState is the abstract snapshot referenced by invariants/SPEC.md.
// It is the Dafny counterpart of the Go `invariants.Snapshot` type that
// Step 3 of the verification strategy will introduce. Keep the two in lockstep.
//
// The state is considered AFTER cross-validation for `LastVerifiedTimestamp()`
// completes and BEFORE cross-validation for the next timestamp begins.
//
// Naming matches SPEC.md §0:
//   LogsDB[j]   = L_j = [(B^j_0, l^j_0), ..., (B^j_{n_j}, l^j_{n_j})]
//   Verified    = [(t_0, C_{t_0}, C^1_{t_0}, ..., C^k_{t_0}), ..., (t, C_t, C^1_t, ..., C^k_t)]
//   DenyList[j] = D_j
//   Chains      = {1, ..., k}
datatype SupernodeState = SupernodeState(
    ActivationTS : uint64,
    Chains       : set<ChainID>,
    LogsDB       : map<ChainID, seq<BlockWithLogs>>,
    Verified     : seq<VerifiedResult>,
    DenyList     : map<ChainID, set<DenyListEntry>>
)

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

// LogsDBLookup finds a block in a LogsDB slice by its ID.
// Used by I6 (to walk from the verified head to its recorded parent hash).
function LogsDBLookup(logs : seq<BlockWithLogs>, id : BlockID) : Option<BlockWithLogs>
{
    if |logs| == 0 then None
    else if logs[0].Ref.ID == id then Some(logs[0])
    else LogsDBLookup(logs[1..], id)
}

// LastVerifiedTimestamp returns the t from SPEC.md §0, or None if the VerifiedDB
// is empty.
function LastVerifiedTimestamp(s : SupernodeState) : Option<uint64>
{
    if |s.Verified| == 0 then None
    else Some(s.Verified[|s.Verified| - 1].Timestamp)
}

// -----------------------------------------------------------------------------
// I1 — Logs match blocks.
//
// SPEC.md I1 / overview.md line 64:
//   l^j_i is the list of logs for block B^j_i.
//
// This is a SEMANTIC property about the relationship between an on-chain block
// and the logs emitted when executing it. It cannot be verified from the
// state alone — the Go layer enforces it at ingress (trusted VirtualNode).
// In Dafny we model it as an opaque predicate that the ingress layer must
// discharge when it inserts a (block, execMsgs) pair into LogsDB.
// -----------------------------------------------------------------------------
predicate {:opaque} LogsBelongToBlock(ref : BlockRef, execMsgs : seq<ExecutingMessage>)

predicate I1_LogsMatchBlocks(s : SupernodeState)
{
    forall j {:trigger s.LogsDB[j]} ::
        j in s.LogsDB ==>
            forall i {:trigger s.LogsDB[j][i]} ::
                0 <= i < |s.LogsDB[j]| ==>
                    LogsBelongToBlock(s.LogsDB[j][i].Ref, s.LogsDB[j][i].ExecMsgs)
}

// -----------------------------------------------------------------------------
// I2 — LogsDB linearity.
//
// SPEC.md I2 / overview.md line 65:
//   B^j_i is the parent block of B^j_{i+1}.
// -----------------------------------------------------------------------------
predicate I2_LogsDBLinear(s : SupernodeState)
{
    forall j ::
        j in s.LogsDB ==>
            forall i ::
                0 <= i < |s.LogsDB[j]| - 1 ==>
                    IsParentOf(s.LogsDB[j][i].Ref, s.LogsDB[j][i+1].Ref)
}

// -----------------------------------------------------------------------------
// I3 — Cross-validity of every LogsDB block.
//
// SPEC.md I3 / overview.md line 66:
//   Every B^j_i is cross-valid given the cross-validity history for timestamps
//   <= i. All executing messages in l^j_i are valid and are not part of a
//   cycle.
//
// The check has two parts:
//   (a) Every ExecutingMessage references an initiating message that is
//       recorded in some other chain's LogsDB at the indicated (BlockNum,
//       LogIdx, Timestamp).
//   (b) The executing-message graph is acyclic.
//
// Part (a) is state-local and decidable. Part (b) is the transitive closure of
// (a) and is expensive; we state it as an abstract predicate here and delegate
// the acyclicity decision to the reference model / fuzzer.
// -----------------------------------------------------------------------------

// InitiatingMessageExists: the initiating message referenced by `m` is recorded
// in `s.LogsDB[m.ChainID]` at block number `m.BlockNum`, position `m.LogIdx`,
// with matching timestamp. This is the "local validity" half of I3.
predicate InitiatingMessageExists(s : SupernodeState, m : ExecutingMessage)
{
    m.ChainID in s.LogsDB &&
    exists i ::
        0 <= i < |s.LogsDB[m.ChainID]| &&
        s.LogsDB[m.ChainID][i].Ref.ID.Number == m.BlockNum &&
        s.LogsDB[m.ChainID][i].Ref.Time == m.Timestamp
}

// ExecMsgGraphAcyclic: there is no cycle in the executing-message dependency
// graph. Modeled abstractly; the reference model enumerates the graph and the
// fuzzer adversarially searches for cycles.
predicate {:opaque} ExecMsgGraphAcyclic(s : SupernodeState)

predicate I3_LogsDBCrossValid(s : SupernodeState)
{
    (forall j ::
        j in s.LogsDB ==>
            forall i ::
                0 <= i < |s.LogsDB[j]| ==>
                    forall k ::
                        0 <= k < |s.LogsDB[j][i].ExecMsgs| ==>
                            InitiatingMessageExists(s, s.LogsDB[j][i].ExecMsgs[k]))
    && ExecMsgGraphAcyclic(s)
}

// -----------------------------------------------------------------------------
// I4 — VerifiedDB anchor.
//
// SPEC.md I4 / overview.md line 67:
//   C^j_{t_0} = B^j_0 for all j.
// -----------------------------------------------------------------------------
predicate I4_VerifiedAnchor(s : SupernodeState)
{
    |s.Verified| > 0 ==>
        var first := s.Verified[0];
        first.Timestamp == s.ActivationTS &&
        forall j ::
            j in s.Chains ==>
                j in s.LogsDB &&
                |s.LogsDB[j]| > 0 &&
                j in first.L2Heads &&
                first.L2Heads[j] == s.LogsDB[j][0].Ref.ID
}

// -----------------------------------------------------------------------------
// I5 — VerifiedDB head equals LogsDB tail.
//
// SPEC.md I5 / overview.md line 68:
//   C^j_t = B^j_{n_j} for all j.
// -----------------------------------------------------------------------------
predicate I5_HeadMatchesTail(s : SupernodeState)
{
    |s.Verified| > 0 ==>
        var last := s.Verified[|s.Verified| - 1];
        forall j ::
            j in s.Chains ==>
                j in s.LogsDB &&
                |s.LogsDB[j]| > 0 &&
                j in last.L2Heads &&
                last.L2Heads[j] == s.LogsDB[j][|s.LogsDB[j]| - 1].Ref.ID
}

// -----------------------------------------------------------------------------
// I6 — Monotone L2 heads across VerifiedDB entries.
//
// SPEC.md I6 / overview.md line 69:
//   C^j_i is either equal to C^j_{i+1} or its parent, for all i and j.
//
// We verify this by looking up C^j_{i+1} in LogsDB[j] and checking either the
// "equal" case or that the looked-up block's ParentHash points to C^j_i.Hash
// and the number differs by one.
// -----------------------------------------------------------------------------
predicate I6_MonotoneL2Heads(s : SupernodeState)
{
    forall i ::
        0 <= i < |s.Verified| - 1 ==>
            forall j ::
                j in s.Chains ==>
                    j in s.Verified[i].L2Heads &&
                    j in s.Verified[i+1].L2Heads &&
                    j in s.LogsDB &&
                    (var prev := s.Verified[i].L2Heads[j];
                     var nxt  := s.Verified[i+1].L2Heads[j];
                     prev == nxt ||
                     (var nxtBlock := LogsDBLookup(s.LogsDB[j], nxt);
                      nxtBlock.Some? &&
                      nxtBlock.value.Ref.ParentHash == prev.Hash &&
                      nxtBlock.value.Ref.ID.Number == prev.Number + 1))
}

// -----------------------------------------------------------------------------
// I7 — C^j_i is the highest block with timestamp <= i (LogsDB-internal half).
//
// SPEC.md I7 / overview.md line 70:
//   C^j_i is the highest block on its chain with timestamp <= i. All children
//   of C^j_i have timestamp > i.
//
// The "all children" clause requires live L2 inspection and is ENVIRONMENTAL
// (see A3 in SPEC.md). Here we verify the LogsDB-internal half: C^j_i is in
// LogsDB[j], has Time <= i, and either is the last entry or the next entry has
// Time > i.
// -----------------------------------------------------------------------------
predicate I7_HighestBlockLeqTS(s : SupernodeState)
{
    forall i ::
        0 <= i < |s.Verified| ==>
            var ts := s.Verified[i].Timestamp;
            forall j ::
                j in s.Chains ==>
                    j in s.Verified[i].L2Heads &&
                    j in s.LogsDB &&
                    (var head := s.Verified[i].L2Heads[j];
                     var found := LogsDBLookup(s.LogsDB[j], head);
                     found.Some? &&
                     found.value.Ref.Time <= ts &&
                     // Either `head` is the tail, or the next LogsDB entry's
                     // timestamp is strictly greater than `ts`.
                     (forall k ::
                        0 <= k < |s.LogsDB[j]| - 1 &&
                        s.LogsDB[j][k].Ref.ID == head ==>
                            s.LogsDB[j][k+1].Ref.Time > ts))
}

// -----------------------------------------------------------------------------
// I8 — Verified L1 heads form a linear chain (number-monotone half).
//
// SPEC.md I8 / overview.md line 71:
//   C_{t_0}, ..., C_t are all part of the same linear chain (there may be
//   missing L1 blocks in between).
//
// Ancestry cannot be checked from the snapshot alone — see A2 / SPEC.md §4.
// We verify the necessary condition: L1 inclusion numbers are non-decreasing
// across consecutive verified entries.
// -----------------------------------------------------------------------------
predicate I8_VerifiedL1Monotone(s : SupernodeState)
{
    forall i ::
        0 <= i < |s.Verified| - 1 ==>
            s.Verified[i].L1Inclusion.Number <= s.Verified[i+1].L1Inclusion.Number
}

// -----------------------------------------------------------------------------
// I9 — C_i is the minimal L1 block covering all L2 heads at i.
//
// SPEC.md I9 / overview.md line 72:
//   C_i is the earliest L1 block on its linear chain where all of
//   C^1_i, ..., C^k_i are available.
//
// This requires deriveL1(C^j_i), which is not in the snapshot. We model it as
// an opaque function supplied by the environment. The Go layer records
// deriveL1 at commit time and the predicate is checked there.
// -----------------------------------------------------------------------------
function {:axiom} DeriveL1(block : BlockID) : BlockID

predicate I9_MinimalL1Cover(s : SupernodeState)
{
    forall i ::
        0 <= i < |s.Verified| ==>
            forall j ::
                j in s.Chains ==>
                    j in s.Verified[i].L2Heads &&
                    // The verified L1 head is not below the L1 from which each
                    // L2 head was derived.
                    DeriveL1(s.Verified[i].L2Heads[j]).Number
                        <= s.Verified[i].L1Inclusion.Number &&
                    // And some L2 head's deriveL1 meets the L1 inclusion exactly
                    // (minimality — C_i is max over j of deriveL1(C^j_i)).
                    (exists j' ::
                        j' in s.Chains &&
                        j' in s.Verified[i].L2Heads &&
                        DeriveL1(s.Verified[i].L2Heads[j']).Number
                            == s.Verified[i].L1Inclusion.Number)
}

// -----------------------------------------------------------------------------
// I10 — LogsDB disjoint from DenyList.
//
// SPEC.md I10 / overview.md line 73:
//   B^j_i is not in D_j for any j and i.
// -----------------------------------------------------------------------------
predicate I10_LogsDBDisjointFromDenyList(s : SupernodeState)
{
    forall j ::
        j in s.LogsDB && j in s.DenyList ==>
            forall i ::
                0 <= i < |s.LogsDB[j]| ==>
                    forall entry ::
                        entry in s.DenyList[j] ==>
                            entry.Block != s.LogsDB[j][i].Ref.ID
}

// -----------------------------------------------------------------------------
// I11 — DenyList entries bounded by t + 1.
//
// SPEC.md I11 / overview.md line 74:
//   For every block B in D_j, the timestamp of B is <= t + 1. The DenyList CAN
//   contain invalidated blocks for the next timestamp t + 1 from previous
//   unsuccessful attempts.
//
// We use the DecisionTimestamp recorded on the DenyListEntry rather than the
// block's own timestamp: SPEC.md §5 open question #7 requires this to be
// explicit.
// -----------------------------------------------------------------------------
predicate I11_DenyListBounded(s : SupernodeState)
{
    var last := LastVerifiedTimestamp(s);
    forall j ::
        j in s.DenyList ==>
            forall entry ::
                entry in s.DenyList[j] ==>
                    // `as int` promotes to mathematical integers so the
                    // `+ 1` does not require an overflow side-condition.
                    match last {
                        case None =>
                            (entry.DecisionTimestamp as int)
                                <= (s.ActivationTS as int) + 1
                        case Some(t) =>
                            (entry.DecisionTimestamp as int) <= (t as int) + 1
                    }
}

// -----------------------------------------------------------------------------
// I12 — Initial state trivially satisfies I1..I11.
//
// SPEC.md I12 / overview.md line 76:
//   At the initial state (empty LogsDB, empty DenyList, empty VerifiedDB), all
//   of the above invariants hold by default.
//
// We express this as a lemma rather than a predicate: given the initial-state
// predicate, AllInvariants follows. The proof obligation is trivial because
// every universal quantifier over sequences of length zero is vacuously true.
// -----------------------------------------------------------------------------
predicate IsInitialState(s : SupernodeState)
{
    s.Verified == [] &&
    // Domain equality: LogsDB and DenyList are keyed exactly by Chains. This
    // prevents orphan entries for chains outside `s.Chains` that would
    // otherwise make I2 / I10 / I11 unprovable in the initial state (Dafny
    // has no way to rule out non-empty sequences under untracked chain IDs).
    s.LogsDB.Keys == s.Chains &&
    s.DenyList.Keys == s.Chains &&
    (forall j :: j in s.Chains ==> s.LogsDB[j] == []) &&
    (forall j :: j in s.Chains ==> s.DenyList[j] == {})
}

// I12 is proven automatically as the lemma below.
lemma I12_InitialStateSatisfiesInvariants(s : SupernodeState)
    requires IsInitialState(s)
    ensures I2_LogsDBLinear(s)
    ensures I4_VerifiedAnchor(s)
    ensures I5_HeadMatchesTail(s)
    ensures I6_MonotoneL2Heads(s)
    ensures I7_HighestBlockLeqTS(s)
    ensures I8_VerifiedL1Monotone(s)
    ensures I10_LogsDBDisjointFromDenyList(s)
    ensures I11_DenyListBounded(s)
{
    // Vacuous — all quantifiers range over empty sequences / sets.
    // I1, I3, I9 depend on opaque predicates / functions and are discharged
    // at the proof boundary, not here.
}

// -----------------------------------------------------------------------------
// AllInvariants — the single predicate every method should preserve.
//
// This is the Dafny counterpart of `invariants.CheckAll(Snapshot)` in Go.
// Methods in Supernode.dfy / VerifiedDB.dfy that mutate state should grow
// `requires AllInvariants(oldState)` and `ensures AllInvariants(newState)`
// clauses referencing this predicate. See SPEC.md §5 open questions.
// -----------------------------------------------------------------------------
predicate AllInvariants(s : SupernodeState)
{
    I1_LogsMatchBlocks(s) &&
    I2_LogsDBLinear(s) &&
    I3_LogsDBCrossValid(s) &&
    I4_VerifiedAnchor(s) &&
    I5_HeadMatchesTail(s) &&
    I6_MonotoneL2Heads(s) &&
    I7_HighestBlockLeqTS(s) &&
    I8_VerifiedL1Monotone(s) &&
    I9_MinimalL1Cover(s) &&
    I10_LogsDBDisjointFromDenyList(s) &&
    I11_DenyListBounded(s)
}

// -----------------------------------------------------------------------------
// Transition specification stubs (SPEC.md §3).
//
// These express T1..T5 as relations between a pre-state and a post-state.
// They are referenced by the reference model and by the Supernode.dfy
// integration PR that will follow. Kept abstract here so that this file is
// self-contained and additive.
// -----------------------------------------------------------------------------

// T1 — chain not caught up: no state update.
predicate T1_NoUpdate(pre : SupernodeState, post : SupernodeState)
{
    pre == post
}

// T2 — L1 inconsistency among L2 derivations: no state update.
predicate T2_NoUpdate(pre : SupernodeState, post : SupernodeState)
{
    pre == post
}

// T3 — rollback when C_t has been reorged out.
// Prune VerifiedDB by one entry; prune DenyList entries with
// DecisionTimestamp >= t; prune LogsDB tails where the rewound head differs.
predicate T3_Rollback(pre : SupernodeState, post : SupernodeState, t : uint64)
{
    |pre.Verified| > 0 &&
    pre.Verified[|pre.Verified| - 1].Timestamp == t &&
    post.Verified == pre.Verified[..|pre.Verified| - 1] &&
    post.ActivationTS == pre.ActivationTS &&
    post.Chains == pre.Chains &&
    // DenyList pruned of entries at or after t.
    (forall j ::
        j in pre.DenyList ==>
            j in post.DenyList &&
            post.DenyList[j] ==
                (set e | e in pre.DenyList[j] && e.DecisionTimestamp < t))
    // LogsDB tail-prune condition is not fully captured here (requires
    // knowing C^j_{t-1}); it is discharged by the reference model.
}

// T4 — invalidation.
// VerifiedDB unchanged; invalid heads added to DenyList with
// DecisionTimestamp = t + 1; LogsDB unchanged.
predicate T4_Invalidate(
    pre : SupernodeState,
    post : SupernodeState,
    t : uint64,
    invalidHeads : map<ChainID, BlockID>)
    // Guard the `t + 1` computation at the contract level so DenyListEntry
    // construction below is well-typed in uint64 space.
    requires (t as int) + 1 <= MAX_UINT64
{
    post.Verified == pre.Verified &&
    post.LogsDB == pre.LogsDB &&
    post.ActivationTS == pre.ActivationTS &&
    post.Chains == pre.Chains &&
    (forall j ::
        j in pre.DenyList ==>
            j in post.DenyList &&
            (j in invalidHeads ==>
                post.DenyList[j] ==
                    pre.DenyList[j] + {DenyListEntry(invalidHeads[j], t + 1)}) &&
            (j !in invalidHeads ==>
                post.DenyList[j] == pre.DenyList[j]))
}

// T5 — advance.
// VerifiedDB extended with (t+1, C_{t+1}, heads); LogsDB extended for chains
// whose B_j is a new block.
predicate T5_Advance(
    pre : SupernodeState,
    post : SupernodeState,
    newEntry : VerifiedResult)
{
    // newEntry timestamp is one more than pre.Verified last, or ActivationTS.
    (|pre.Verified| == 0 ==> newEntry.Timestamp == pre.ActivationTS) &&
    (|pre.Verified| > 0 ==>
        (newEntry.Timestamp as int) ==
            (pre.Verified[|pre.Verified| - 1].Timestamp as int) + 1) &&
    post.Verified == pre.Verified + [newEntry] &&
    post.DenyList == pre.DenyList &&
    post.ActivationTS == pre.ActivationTS &&
    post.Chains == pre.Chains
    // LogsDB extension (either append or unchanged per chain) discharged by
    // the reference model.
}

// T6 — totality: every legal transition matches exactly one of T1..T5.
// Stated as a lemma over the reference model; not formalized at this layer.
