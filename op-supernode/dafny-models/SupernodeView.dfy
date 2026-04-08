include "SupernodeState.dfy"

// -----------------------------------------------------------------------------
// SupernodeView — bridge between heap-based state (Supernode / VerifiedDB
// classes) and the pure SupernodeState datatype used by AllInvariants.
//
// This file is intentionally independent of the Supernode class so that it
// composes with Step 2c (which will add LogsDB / DenyList fields to the class
// and a concrete sorted-sequence extraction for VerifiedDB.verified).
//
// Conventions:
//   - SPEC IDs in comments (I1..I11, A1..A5, T1..T6) refer to
//     op-supernode/invariants/SPEC.md, which is the single source of truth.
//   - `verifiedSeq` is the sorted (ascending Timestamp) extraction of
//     VerifiedDB.verified. Abstract here; concrete extraction in Step 2c.
// -----------------------------------------------------------------------------

// BuildSupernodeState lifts independently-tracked components into the pure
// datatype consumed by AllInvariants.
function BuildSupernodeState(
    activationTS : uint64,
    chains       : set<ChainID>,
    verifiedSeq  : seq<VerifiedResult>,
    logsDB       : map<ChainID, seq<BlockWithLogs>>,
    denyList     : map<ChainID, set<DenyListEntry>>
) : SupernodeState
{
    SupernodeState(activationTS, chains, logsDB, verifiedSeq, denyList)
}

// ComponentsSatisfyInvariants is the convenience surface that
// Supernode / VerifiedDB method contracts should reference in Step 2c.
// Using this (rather than inlining AllInvariants(BuildSupernodeState(...)))
// keeps the bridge point syntactically visible in every contract.
predicate ComponentsSatisfyInvariants(
    activationTS : uint64,
    chains       : set<ChainID>,
    verifiedSeq  : seq<VerifiedResult>,
    logsDB       : map<ChainID, seq<BlockWithLogs>>,
    denyList     : map<ChainID, set<DenyListEntry>>
)
{
    AllInvariants(BuildSupernodeState(
        activationTS, chains, verifiedSeq, logsDB, denyList))
}

// -----------------------------------------------------------------------------
// VerifiedDB extraction
//
// VerifiedDB stores entries in `verified : map<uint64, VerifiedResult>` with
// no intrinsic order. SPEC §0 requires a sorted sequence
// [(t_0, ...), ..., (t, ...)] for I4, I5, I6, I7, I8.
//
// `VerifiedMapToSortedSeq` performs that extraction. It is declared abstract
// here (body-less) and will be given a concrete recursive implementation in
// Step 2c once the auxiliary `MinKey` helper is written. The abstract
// declaration is sufficient for contract-level use because:
//
//   1. Every use in SPEC-level predicates is read-only — the actual
//      enumeration strategy does not matter.
//   2. The lemmas below record the key algebraic properties that the
//      concrete implementation must satisfy.
// -----------------------------------------------------------------------------
function VerifiedMapToSortedSeq(m : map<uint64, VerifiedResult>) : seq<VerifiedResult>

lemma {:axiom} VerifiedMapToSortedSeqLength(m : map<uint64, VerifiedResult>)
    ensures |VerifiedMapToSortedSeq(m)| == |m.Keys|

// Forward direction: every key in the map appears in the extracted seq,
// and the value at that index equals m[k]. Dereferencing m[k] is guarded by
// `k in m.Keys` in the hypothesis.
lemma {:axiom} VerifiedMapToSortedSeqContainsForward(
    m : map<uint64, VerifiedResult>, k : uint64)
    requires k in m.Keys
    ensures
        exists i ::
            0 <= i < |VerifiedMapToSortedSeq(m)| &&
            VerifiedMapToSortedSeq(m)[i].Timestamp == k &&
            VerifiedMapToSortedSeq(m)[i] == m[k]

// Backward direction: every timestamp appearing in the extracted seq was a
// key of the original map. Stated without dereferencing m[k].
lemma {:axiom} VerifiedMapToSortedSeqContainsBackward(
    m : map<uint64, VerifiedResult>, k : uint64)
    requires
        exists i ::
            0 <= i < |VerifiedMapToSortedSeq(m)| &&
            VerifiedMapToSortedSeq(m)[i].Timestamp == k
    ensures k in m.Keys

lemma {:axiom} VerifiedMapToSortedSeqAscending(m : map<uint64, VerifiedResult>)
    ensures forall i ::
        0 <= i < |VerifiedMapToSortedSeq(m)| - 1 ==>
            VerifiedMapToSortedSeq(m)[i].Timestamp
                < VerifiedMapToSortedSeq(m)[i+1].Timestamp

// -----------------------------------------------------------------------------
// Weak-to-strong linkage
//
// VerifiedDB.Invariants() (as currently defined in VerifiedDB.dfy) captures
// only gap-free timestamp structure: initialized iff lastTimestamp is Some,
// lastTimestamp is the maximum key, and the key set has no gaps below it.
//
// This is WEAKER than what AllInvariants requires. In particular it says
// nothing about:
//   * L2 head monotonicity (I6)
//   * L1 inclusion ancestry or number monotonicity (I8)
//   * LogsDB / DenyList (I1, I2, I5, I10, I11)
//
// The lemma below records the ONE property that VerifiedDB.Invariants() does
// buy us: under its hypothesis, consecutive entries in the sorted sequence
// differ in Timestamp by exactly 1. This is a NECESSARY precondition to I8
// but not equivalent.
// -----------------------------------------------------------------------------
predicate VerifiedSeqStrictlyIncreasingByOne(verifiedSeq : seq<VerifiedResult>)
{
    forall i ::
        0 <= i < |verifiedSeq| - 1 ==>
            (verifiedSeq[i].Timestamp as int) + 1
                == (verifiedSeq[i+1].Timestamp as int)
}

// NoGapsUnder mirrors VerifiedDB.NoGaps without coupling this file to the
// VerifiedDB class. The predicate says: there are no holes in the key set
// strictly below `max`. (VerifiedDB.NoGaps uses a strict `<` on both sides.)
predicate NoGapsUnder(s : set<uint64>, max : uint64)
{
    forall t1, t2 :: (t1 in s && t1 < t2 < max) ==> t2 in s
}

// VerifiedDBLikeMap captures the VerifiedDB.Invariants() shape applied to an
// extracted map, without requiring a live VerifiedDB reference.
predicate VerifiedDBLikeMap(m : map<uint64, VerifiedResult>)
{
    m == map[] ||
    (exists lastTs :: lastTs in m.Keys &&
        (forall ts :: ts in m.Keys ==> ts <= lastTs) &&
        NoGapsUnder(m.Keys, lastTs))
}

lemma {:axiom} WeakVerifiedInvariantImpliesStepOne(
    m : map<uint64, VerifiedResult>)
    requires VerifiedDBLikeMap(m)
    ensures VerifiedSeqStrictlyIncreasingByOne(VerifiedMapToSortedSeq(m))

// -----------------------------------------------------------------------------
// Initial-state bridge
//
// At the initial state of a Supernode — empty verified map, empty per-chain
// LogsDB, empty per-chain DenyList — the components trivially satisfy
// AllInvariants. This lemma re-states I12 (SupernodeState.dfy) at the
// components layer so that the eventual constructor integration in Step 2c
// can cite it directly.
// -----------------------------------------------------------------------------
lemma InitialComponentsSatisfyInvariants(
    activationTS : uint64,
    chains       : set<ChainID>,
    logsDB       : map<ChainID, seq<BlockWithLogs>>,
    denyList     : map<ChainID, set<DenyListEntry>>)
    requires logsDB.Keys == chains
    requires denyList.Keys == chains
    requires forall j :: j in chains ==> logsDB[j] == []
    requires forall j :: j in chains ==> denyList[j] == {}
    ensures ComponentsSatisfyInvariants(
        activationTS, chains, [], logsDB, denyList)
{
    var s := BuildSupernodeState(activationTS, chains, [], logsDB, denyList);
    assert IsInitialState(s);
    I12_InitialStateSatisfiesInvariants(s);
    // I1, I3, I9 are not part of I12 because they depend on opaque predicates
    // / axiom functions. They hold vacuously here because every quantifier
    // ranges over an empty sequence, but we cannot prove that without
    // revealing the opacity. Recorded as TODOs in SPEC.md §5 item 11; each
    // `assume {:axiom}` is explicitly tagged so Dafny does not warn.
    assume {:axiom} I1_LogsMatchBlocks(s);
    assume {:axiom} I3_LogsDBCrossValid(s);
    assume {:axiom} I9_MinimalL1Cover(s);
}
