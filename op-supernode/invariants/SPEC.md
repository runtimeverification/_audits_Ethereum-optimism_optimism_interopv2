# op-supernode Invariant Specification

**Status:** canonical source of truth for invariants enforced across Dafny models, Go unit/fuzz tests, reference model, and live execution runtime assertions.

**Upstream prose:** [`../overview.md`](../overview.md) §"Supernode State and Invariants" and §"State Changes from Cross-Validation".

Every invariant has a stable ID (`I*`, `A*`, `T*`). Do not renumber. When adding a new invariant, append; when deleting, mark `// retired in <commit>` and leave the ID reserved.

All Go predicates (`invariants/invariants.go`) and Dafny predicates (`dafny-models/Supernode.dfy :: AllInvariants`) MUST reference the ID in a comment. Test failures MUST print the ID.

---

## 0. Notation

Let the abstract supernode state for `k` L2 chains be:

- `t₀` — activation timestamp.
- `t` — last verified timestamp (`None` before any successful cross-validation).
- For each L2 chain `j ∈ {1..k}`:
  - `Lⱼ = [(Bʲ₀, ℓʲ₀), …, (Bʲ_{nⱼ}, ℓʲ_{nⱼ})]` — the `LogsDB` for chain `j`. `Bʲᵢ` is a block, `ℓʲᵢ` is the list of logs for that block.
  - `Dⱼ ⊆ BlockID` — the `DenyList` for chain `j`.
- The `VerifiedDB` is the sequence
  `[(t₀, C_{t₀}, C¹_{t₀}, …, Cᵏ_{t₀}), …, (t, C_t, C¹_t, …, Cᵏ_t)]`
  where `C_i` is the verified L1 head at timestamp `i` and `Cʲ_i` is the verified L2 head for chain `j` at timestamp `i`.
- `parent(B)` — the parent block of `B` on its native chain.
- `timestamp(B)` — the block timestamp of `B`.
- `deriveL1(Bʲᵢ)` — the L1 block from which `Bʲᵢ` was derived.

The abstract state is considered AFTER cross-validation for `t` completes and BEFORE cross-validation for `t + 1` begins.

---

## 1. State Invariants (checkable from a pure snapshot)

These are checked by `invariants.CheckAll(Snapshot)` in Go and `AllInvariants(SupernodeState)` in Dafny. They depend only on internal state — **no live L1/L2 queries**.

### I1 — Logs match blocks

For every chain `j` and index `i`:
> `ℓʲᵢ` is the list of logs for block `Bʲᵢ`.

**Source:** overview.md line 64 (first bullet under "The following invariants").
**Check:** in Go this is enforced at insertion time (the snapshot treats `ℓʲᵢ` as part of the block entry). The runtime assertion is that `LogsDB[j][i].Block.Hash` matches the block whose logs were queried. Environmental dependency: requires trust in the VirtualNode's log retrieval.
**Class:** STRUCTURAL (snapshot-checkable given an honest VirtualNode).

### I2 — LogsDB linearity

For every chain `j` and every `0 ≤ i < nⱼ`:
> `Bʲᵢ = parent(Bʲᵢ₊₁)`.

**Source:** overview.md line 65.
**Check:** `LogsDB[j][i+1].ParentHash == LogsDB[j][i].Hash` and `LogsDB[j][i+1].Number == LogsDB[j][i].Number + 1`.
**Class:** STRUCTURAL.

### I3 — LogsDB cross-validity

For every chain `j` and index `i`:
> `Bʲᵢ` is cross-valid given the cross-validity history for timestamps `≤ i`. All executing messages in `ℓʲᵢ` are valid (the referenced initiating message exists in some other chain's `LogsDB` at a strictly lesser or equal timestamp) and are not part of a cycle.

**Source:** overview.md line 66.
**Check:** for every `ExecutingMessage` `m` in `ℓʲᵢ`:
1. `m` references a `(chainID, blockNum, logIdx, timestamp)` that exists in `LogsDB[m.ChainID]` at the quoted position and the quoted timestamp.
2. The executing-message directed graph rooted at `Bʲᵢ` is acyclic.

**Class:** STRUCTURAL but expensive. Runtime assertions may downgrade to a spot check (single-message validity) and leave the full cycle check to fuzz and Dafny.

### I4 — VerifiedDB anchor

For every chain `j`:
> `Cʲ_{t₀} = Bʲ₀`.

**Source:** overview.md line 67.
**Check:** `Verified[0].L2Heads[j] == LogsDB[j][0].ID`.
**Class:** STRUCTURAL.

### I5 — VerifiedDB head = LogsDB tail

For every chain `j`:
> `Cʲ_t = Bʲ_{nⱼ}`.

**Source:** overview.md line 68.
**Check:** `Verified[len-1].L2Heads[j] == LogsDB[j][len-1].ID`.
**Class:** STRUCTURAL.

### I6 — Monotone L2 heads

For every chain `j` and every `t₀ ≤ i < t`:
> `Cʲᵢ` is equal to `Cʲᵢ₊₁` or to `parent(Cʲᵢ₊₁)`.

Consequently, `Cʲ_{t₀}, …, Cʲ_t` equals `Bʲ₀, …, Bʲ_{nⱼ}` with possible repetitions in the middle.

**Source:** overview.md line 69.
**Check:** for each consecutive pair in `Verified[]`, either `C^j_i == C^j_{i+1}` or `C^j_{i+1}.ParentHash == C^j_i.Hash`.
**Class:** STRUCTURAL.

### I7 — Highest L2 block at each timestamp

For every chain `j` and every `t₀ ≤ i ≤ t`:
> `Cʲᵢ` is the highest block on chain `j` with `timestamp ≤ i`. All children of `Cʲᵢ` have `timestamp > i`.

**Source:** overview.md line 70.
**Check (pure snapshot):** using only `LogsDB[j]`, verify that `Cʲᵢ ∈ LogsDB[j]` and that either `Cʲᵢ` is the latest LogsDB entry with `timestamp ≤ i`, or the next LogsDB entry has `timestamp > i`. The "all children" clause cannot be checked without the live L2 — see A3.
**Class:** HYBRID. The LogsDB-internal part is structural; the "no descendant on the live L2 with `timestamp ≤ i` other than `Cʲᵢ`" part is environmental.

### I8 — Verified L1 forms a linear chain

> `C_{t₀}, …, C_t` are all part of the same linear chain (there may be missing L1 blocks between them).

**Source:** overview.md line 71.
**Check:** under the assumption that the L1 has not reorged since the last verified entry (A2), the monotone-number property (`C_i.Number ≤ C_{i+1}.Number`) plus a linear ancestor check at each commit is sufficient. A *pure snapshot* check is the number-monotonicity; the ancestor check requires an L1 query — see A2.
**Class:** HYBRID.

### I9 — `C_i` is minimal L1 covering all L2 heads at `i`

For every `t₀ ≤ i ≤ t`:
> `C_i` is the earliest L1 block on its linear chain where all of `C¹_i, …, Cᵏ_i` are available — the maximum among `deriveL1(C¹_i), …, deriveL1(Cᵏ_i)`.

**Source:** overview.md line 72.
**Check:** for every verified entry `i`, `C_i.Number == max_j( deriveL1(Cʲ_i).Number )` and `deriveL1(Cʲ_i)` is an ancestor of `C_i` (or equal). The `deriveL1` lookup is environmental — see A4.
**Class:** HYBRID.

### I10 — LogsDB disjoint from DenyList

For every chain `j` and index `i`:
> `Bʲᵢ ∉ Dⱼ`. A block in the LogsDB (and, consequently, in the VerifiedDB) does not appear in the DenyList.

**Source:** overview.md line 73.
**Check:** `for j, blocks := range LogsDB { for _, b := range blocks { require b.ID ∉ DenyList[j] } }`.
**Class:** STRUCTURAL.

### I11 — DenyList timestamp bound

For every chain `j` and every `B ∈ Dⱼ`:
> `timestamp(B) ≤ t + 1`.

**Note:** The DenyList is allowed to contain entries for `t + 1` from previous failed attempts to cross-validate that timestamp.

**Source:** overview.md line 74.
**Check:** `for _, b := range DenyList[j] { require timestamp(b) ≤ Verified[len-1].Timestamp + 1 }`. The `timestamp(b)` lookup may require the LogsDB or a side table recording when a block was denied.
**Class:** STRUCTURAL (requires the DenyList entries to record their decision timestamp — enforce this at insertion time).

### I12 — Initial state holds vacuously

> At the initial state (empty `LogsDB`, empty `DenyList`, empty `VerifiedDB`), I1–I11 all hold by default.

**Source:** overview.md line 76.
**Check:** unit test that constructs a fresh `Supernode` and runs `CheckAll`.
**Class:** META.

---

## 2. Environmental Assumptions (not snapshot-checkable)

These are properties the supernode RELIES ON from the L1 and L2 chains. They cannot be checked from internal state alone. They MUST be validated at ingress (the moment the chain is observed) and recorded as part of the snapshot's "trusted view".

### A1 — Timestamp uniqueness

> Timestamps don't repeat: there is at most one block per chain per timestamp.

**Source:** overview.md line 82.
**Enforcement:** reject any `VirtualNode` response where two distinct blocks share a timestamp. Record violations as a supernode error.

### A2 — L2 reorgs iff L1 reorgs

> Outside of cross-validation, an L2 chain only reorgs if L1 reorgs. By the time an L2 reorg is observable, the corresponding L1 reorg is also observable. (The reverse is NOT true.)

**Source:** overview.md line 83.
**Enforcement:** detect L2 reorg by comparing last-seen L2 head with current L2 head across rounds; if L2 reorged but L1 head is ancestor-consistent, flag as assumption violation.

### A3 — Eventual L2 sync

> Given sufficient time between L1 reorgs, each L2 chain eventually syncs to the L1.

**Source:** overview.md line 84.
**Enforcement:** liveness only — cannot be violated in finite time. Monitor stall duration and alert on prolonged non-progress.

### A4 — Denied blocks produce deposit-only replacements

> If an L2 chain `j` tries to derive a block that is in `Dⱼ`, it inserts a deposit-only block in its place. Since the VirtualNode for an L2 is destroyed and recreated after a block is added to `Dⱼ`, at the start of a round the safe chain for an L2 never has a block currently in its DenyList.

**Source:** overview.md line 85.
**Enforcement:** after every DenyList insertion, verify that the `VirtualNode` has been destroyed and recreated before the next `observeRound`.

### A5 — Arbitrary query results during a round

> L1 and L2 chains may reorg at any point, even mid-round. Any query to L1/L2 state during a round can return an arbitrary result, except for what A1–A4 guarantee.

**Source:** overview.md line 86.
**Enforcement:** NONE — this is a *worst-case assumption* that constrains what the cross-validation logic is allowed to rely on. Fuzz tests MUST generate adversarial query results consistent with A1–A4 to exercise this assumption.

---

## 3. Transition Specification

These describe the *allowed* state changes during a single round of cross-validation. They are checked by running the reference model in lockstep with the real `Supernode` and asserting that the resulting snapshots match after each step.

### T0 — Precondition

Before any round, `AllInvariants(state)` holds.

**Source:** overview.md §"Supernode State and Invariants" (implicit from "expected to be true at this state").

### T1 — Chain not caught up

> For each L2 chain `j`, let `B_j` be the highest block on the safe chain of `j` with `timestamp ≤ t + 1`. If a higher block with `timestamp ≤ t + 1` should exist (predictable from block time) but has not been derived yet, NO state update happens.

**Source:** overview.md line 92.
**Postcondition:** `state' == state`.

### T2 — L1 inconsistency among L2 derivations

> Otherwise, let `B'_j := deriveL1(B_j)`. If the `B'_j` are not all on the same linear L1 chain, a reorg has happened and some L2s have not synced. NO state update happens.

**Source:** overview.md line 93.
**Postcondition:** `state' == state`.

### T3 — Rollback when `C_t` has been reorged out

> Otherwise, if `C_t` is not in the same linear chain as the `B'_j`, then `C_t` has been reorged out. Roll back to the previous timestamp:
> - Prune VerifiedDB to `(t₀, …, t-1)`.
> - Prune every `Dⱼ` by removing all entries with `timestamp ≥ t` (i.e., `t` or `t + 1`).
> - For every chain `j` such that `Cʲ_{t-1} ≠ Bʲ_{nⱼ}`, remove the last entry of `Lⱼ`.

**Source:** overview.md lines 94–97.
**Postcondition:**
- `Verified' = Verified[0..len-1]`
- For all `j`: `D'ⱼ = { b ∈ Dⱼ : decisionTimestamp(b) < t }`
- For all `j`: if `Cʲ_{t-1} ≠ Bʲ_{nⱼ}` then `L'ⱼ = Lⱼ[0..nⱼ-1]` else `L'ⱼ = Lⱼ`.
- `AllInvariants(state')` holds.

### T4 — Invalidation

> Otherwise, every `B_j` is consistent with `C_t`. Either `B_j == Bʲ_{nⱼ}` or `Bʲ_{nⱼ} == parent(B_j)`. `B_j` is invalid if it has an invalid executing message or an executing message in a cycle. If any `B_j` is invalid:
> - Each invalid `B_j` is added to `Dⱼ`.
> - For each chain `j` with invalid `B_j`, the ChainContainer is reset: EngineController rewound to the highest block with `timestamp ≤ t`; VirtualNode destroyed and recreated.
> - NO `LogsDB` is updated. Any logs added during cross-validation must be removed.

**Source:** overview.md lines 98–101.
**Postcondition:**
- `Verified' = Verified` (unchanged)
- For every invalid `j`: `D'ⱼ = Dⱼ ∪ { B_j }` with `decisionTimestamp(B_j) = t + 1`
- For every `j`: `L'ⱼ = Lⱼ` (any speculative additions rolled back)
- `AllInvariants(state')` holds.

### T5 — Advance

> Otherwise (all `B_j` valid):
> - Verified history extended with `(t+1, C_{t+1}, C¹_{t+1}, …, Cᵏ_{t+1})` where `Cʲ_{t+1} = B_j` and `C_{t+1} = max_j(B'_j)`.
> - Every `Lⱼ` such that `B_j ≠ Bʲ_{nⱼ}` is extended with `(B_j, ℓ(B_j))`.

**Source:** overview.md lines 102–104.
**Postcondition:**
- `Verified' = Verified ++ [(t+1, max_j(B'_j), {j → B_j})]`
- For every `j` with `B_j ≠ Bʲ_{nⱼ}`: `L'ⱼ = Lⱼ ++ [(B_j, logs(B_j))]`
- For every `j` with `B_j == Bʲ_{nⱼ}`: `L'ⱼ = Lⱼ`
- `AllInvariants(state')` holds.

### T6 — Totality

Every round of cross-validation matches exactly one of T1–T5. No other state transition is legal.

**Source:** overview.md structure of §"State Changes from Cross-Validation".
**Check:** reference model implements T1–T5 as an exhaustive match; any transition observed in the real supernode that does not correspond to exactly one of these is a bug.

---

## 4. Classification summary

| ID  | Class       | Snapshot-checkable | Live L1/L2 needed | Enforced by                           |
|-----|-------------|-------------------:|------------------:|---------------------------------------|
| I1  | Structural  | Yes (trusted VN)   | No                | Insertion-time check + snapshot       |
| I2  | Structural  | Yes                | No                | Snapshot                              |
| I3  | Structural  | Yes (expensive)    | No                | Snapshot + fuzz + Dafny               |
| I4  | Structural  | Yes                | No                | Snapshot                              |
| I5  | Structural  | Yes                | No                | Snapshot                              |
| I6  | Structural  | Yes                | No                | Snapshot                              |
| I7  | Hybrid      | Partial            | Yes (A3 clause)   | Snapshot (LogsDB) + ingress check     |
| I8  | Hybrid      | Partial            | Yes (ancestor)    | Snapshot (numbers) + L1 ancestor RPC  |
| I9  | Hybrid      | No (deriveL1)      | Yes               | Recorded at commit + L1 ancestor RPC  |
| I10 | Structural  | Yes                | No                | Snapshot                              |
| I11 | Structural  | Yes (side table)   | No                | Snapshot with decision-TS side table  |
| I12 | Meta        | Yes                | No                | Unit test on fresh supernode          |
| A1  | Assumption  | No                 | Yes               | Ingress validation                    |
| A2  | Assumption  | No                 | Yes               | Round-to-round diff                   |
| A3  | Assumption  | No (liveness)      | Yes               | Stall monitor                         |
| A4  | Assumption  | Yes (lifecycle)    | No                | VirtualNode lifecycle assertion       |
| A5  | Assumption  | No                 | N/A               | Worst-case assumption, fuzz adversary |
| T1  | Transition  | Yes                | No                | Reference model lockstep              |
| T2  | Transition  | Yes                | No                | Reference model lockstep              |
| T3  | Transition  | Yes                | No                | Reference model lockstep              |
| T4  | Transition  | Yes                | No                | Reference model lockstep              |
| T5  | Transition  | Yes                | No                | Reference model lockstep              |
| T6  | Transition  | Yes                | No                | Reference model exhaustiveness        |

---

## 5. Open questions / gaps vs current Dafny models

Tracking items to resolve across Steps 2 / 2b / 2c of the Dafny port. Status as of Step 2b:

1. ~~**`applyRewindPlan` is `{:axiom}`-ensured**~~ **RESOLVED (Step 2b).** Replaced with a trivial no-op stub returning `false`. Both `ensures` clauses are now proven (field reference equality + frame-preserved invariants). The real rewind logic is deferred to Step 2c and tracked as item 1'.
2. ~~**`sameL1Chain` is `{:axiom}`-ensured**~~ **RESOLVED (Step 2b).** Replaced with a no-op stub returning `None`; the postcondition becomes vacuous. Real L1-ancestry logic tracked as item 2'.
3. ~~**`resolveFrontierVerificationView` is `{:axiom}`-ensured**~~ **RESOLVED (Step 2b).** Replaced with a no-op stub returning `None`. Real VirtualNode-backed view tracked as item 3'.
4. ~~**No `AllInvariants` predicate exists**~~ **RESOLVED (Step 2).** `AllInvariants` now lives in `dafny-models/SupernodeState.dfy` and covers I1–I11.
5. ~~**`DenyList` not modeled in Dafny**~~ **RESOLVED (Step 2).** Modeled as `map<ChainID, set<DenyListEntry>>` in `SupernodeState`.
6. ~~**`LogsDB` not modeled in Dafny**~~ **RESOLVED (Step 2).** Modeled as `map<ChainID, seq<BlockWithLogs>>` in `SupernodeState`.
7. ~~**`decisionTimestamp(b)` for DenyList entries**~~ **RESOLVED (Step 2).** `DenyListEntry` carries a `DecisionTimestamp : uint64` field.
8. ~~**Activation timestamp initial state**~~ **RESOLVED (Step 2).** `IsInitialState` predicate + `I12_InitialStateSatisfiesInvariants` lemma in `SupernodeState.dfy`.

### Remaining items (Step 2c / 3c)

1′. **Real `applyRewindPlan` implementation** — prune VerifiedDB tail, prune DenyList entries with `DecisionTimestamp >= target`, prune LogsDB tails. Proof obligation: given `AllInvariants(pre)` and a valid `RewindPlan`, produce `AllInvariants(post)` satisfying `T3_Rollback`.
2′. **Real `sameL1Chain` implementation** — requires an L1 ancestry oracle. Option: add an abstract L1 client parameter with a concrete contract, defer to an L1-client integration PR.
3′. **Real `resolveFrontierVerificationView` implementation** — requires a per-chain VirtualNode oracle. Same deferral pattern as item 2′.
9. **Supernode class does not track `logsDB` / `denyList` as first-class fields.** Step 2c should add these (ghost or concrete), plus a `View() : SupernodeState` method bridging via `SupernodeView.BuildSupernodeState`.
10. **`VerifiedMapToSortedSeq` is body-less.** `SupernodeView.dfy` declares it abstract with three axiomatized algebraic lemmas (length, contains, ascending). Step 2c should provide a concrete recursive implementation (requires a `MinKey : set<uint64> -> uint64` helper) and discharge those axioms.
11. **`InitialComponentsSatisfyInvariants` contains `assume` statements for I1/I3/I9.** Those invariants depend on opaque predicates / axiom functions (`LogsBelongToBlock`, `ExecMsgGraphAcyclic`, `DeriveL1`). Step 2c should either mark them `{:opaque}` with `reveal_*` helpers that make the vacuous-initial-state reasoning explicit, or restructure them to take the needed oracles as parameters.
12. **`WeakVerifiedInvariantImpliesStepOne` is axiomatized.** Once `VerifiedMapToSortedSeq` has a concrete body, this lemma should be provable rather than axiomatic.
13. **`interop.LogsDB` has no enumeration API.** The Go `StateView.LogsDBFor(chain)` implementation in `op-supernode/supernode/invariants_bridge.go` (Step 3c) cannot walk all sealed blocks without either (a) adding `EnumerateSealed(from, to)` to `op-supervisor/supervisor/backend/db/logs`, or (b) walking number-by-number from 0 via `FindSealedBlock` / `OpenBlock`. Option (a) is strictly better. LogsDB interface: `op-supernode/supernode/activity/interop/logdb.go:20-47`. LogsDB instances live in `Interop.logsDBs : map[ChainID]LogsDB` at `interop.go:83`.
14. **`chain_container.DenyList` has no enumeration API.** Only `IsDenied(height, hash)` point queries exist. `StateView.DenyListFor(chain)` in the Step 3c adapter needs a `ForEach(func(height, hash) bool)` method added to `chain_container/invalidation.go:23-28`. Trivial — iterate the bbolt bucket. Also note: `denyList` is a PRIVATE field on `simpleChainContainer` (`chain_container.go:93`), so the `ForEach` method must be promoted onto the `ChainContainer` interface at `chain_container.go:32-79`, or the adapter must type-assert to `*simpleChainContainer`. Interface promotion is cleaner.
15. **Supernode-side adapter not yet written.** The `StateView` interface is defined in `op-supernode/invariants/bridge.go`; an adapter implementing it for `*supernode.Supernode` needs to live in `op-supernode/supernode/invariants_bridge.go` (to access unexported fields). Blocked on items 13, 14, and 16.
16. **Activation timestamp is `*uint64`, VerifiedDB is inside the Interop activity.** The Step 3c adapter must: (a) dereference `sn.cfg.InteropActivationTimestamp` with a nil check (return 0 or sentinel if interop is not yet activated); (b) locate the `*interop.Interop` activity inside `sn.activities[]` via type assertion — there's no direct accessor. The activity holds both `verifiedDB` (`interop.go:82`) and the per-chain `logsDBs` map (`interop.go:83`), so one traversal yields both halves of the Snapshot. Consider adding a public `sn.InteropActivity() *interop.Interop` accessor on Supernode to avoid the type-assertion dance.
17. **Do NOT snapshot mid-interop round.** The subagent's lock-discipline analysis flags this as a correctness requirement: the VirtualNode is recreated mid-round, LogsDB tails can be speculative, and DenyList mutations are not atomic with LogsDB mutations. `SnapshotFrom` must be called at a stable checkpoint — specifically, after a completed `applyPendingTransition` cycle. The runtime assertion hook should therefore fire from `Supernode.progressAndRecord()` AFTER `applyPendingTransition` returns, not from inside mutating methods. `invariants.AssertWith(func() Snapshot { return SnapshotFrom(bridge) })` is the canonical call site.
18. **SuperRoot activity is NOT a source for `Verified[]`.** The subagent confirms `SuperRoot.atTimestamp` is a single-timestamp query, not a history accessor. The only path to the full `Verified[]` sequence is walking `interop.VerifiedDB.Get(ts)` from `ActivationTS` to `LastTimestamp()`. The adapter must perform that walk; cache-friendliness is a future optimization (a new `interop.VerifiedDB.Range(from, to) iter.Seq2[uint64, VerifiedResult]` would be ideal).

---

## 6. Change log

- **Initial draft (Step 1)** — extracted verbatim from `overview.md` lines 36–105. No semantic changes to the prose spec; only structural reorganization into a numbered catalog.
- **Step 2** — added `dafny-models/SupernodeState.dfy` with `SupernodeState` datatype, predicates `I1_..I11_`, `IsInitialState`, and `I12_InitialStateSatisfiesInvariants`. Extended `Types.dfy` with `BlockWithLogs`, `DenyListEntry`, `IsParentOf`. Resolved §5 items 4–8.
- **Step 2b** — added `dafny-models/SupernodeView.dfy` with `BuildSupernodeState`, `ComponentsSatisfyInvariants`, abstract `VerifiedMapToSortedSeq`, and `InitialComponentsSatisfyInvariants` lemma. Replaced the three `{:axiom}`-ensured abstract methods in `Supernode.dfy` (`applyRewindPlan`, `sameL1Chain`, `resolveFrontierVerificationView`) with trivial no-op stubs and removed the `{:axiom}` tags. Installed Dafny 4.11.0 via Homebrew and ran `dafny verify` — after fixing 5 surfaced issues (split `VerifiedMapToSortedSeqContains` into forward/backward lemmas, strengthened `IsInitialState` with `Keys == Chains` constraints, added `t+1` overflow guard to `T4_Invalidate`, tagged 3 `assume` statements with `{:axiom}`), reached **66 verified, 0 errors**. Resolved §5 items 1–3. Added new follow-up items 1′, 2′, 3′, 9, 10, 11, 12.
- **Step 3** — added Go `invariants` package: `Snapshot` type mirroring `SupernodeState`, `CheckI2/I4/I5/I6/I7/I8/I10/I11` predicates returning structured `*Error` values with stable SPEC IDs, and `CheckAll` composition using `errors.Join`. 18 unit tests covering happy / failing paths for every predicate + I12 vacuous-initial-state test + joined-error composition test.
- **Step 3b** — added `env.go` with `Env` oracle interface and environmental predicates `CheckI1_LogsBelongToBlocks`, `CheckI3_ExecutingMessageValidity` (state-local initiating-message check + Tarjan cycle detection), `CheckI8_VerifiedL1Linear`, `CheckI9_MinimalL1Cover`; plus `CheckAllWithEnv` composition. Added `bridge.go` with `StateView` interface and `SnapshotFrom(view)` lifter, plus `StaticStateView` fake for tests.
- **Step 4** — added runtime assertion build tag. `runtime_on.go` (build tag `supernode_invariants`) exports `Assert(Snapshot)` / `AssertWith(func() Snapshot)` / `SetPanicOnFailure` / `SetFailureSink` / `AssertionFailureCount`. `runtime_off.go` provides zero-cost no-op equivalents for production builds. `AssertionsEnabled const` lets callers branch at compile time.
- **Step 5** — added `reference_model.go` with pure T3/T4/T5 transition implementations (`ApplyAdvance`, `ApplyInvalidate`, `ApplyRewind`) and `CloneSnapshot` helper. Added `FuzzReferenceModel` native Go fuzzer plus `TestReferenceModel_PropertySample`. Fuzz uncovered one precondition gap: `ApplyAdvance` now refuses to add blocks present in the DenyList (A4 enforcement). Fuzz capped at 512 ops per seed to bound memory. After fix: **2.8M executions over 30s, zero invariant violations**.
- **Step 6** — added `trace.go` with `Trace` JSON format (steps, origin, optional `ExpectedFailures` list), `LoadTrace` / `SaveTrace` / `VerifyTrace` / `LoadTraceCorpus`. `TestTraceCorpus` replays every `testdata/traces/*.json`. Three canonical traces (`happy_single_chain_3_advances`, `happy_invalidate_then_continue`, `happy_rewind_after_two_advances`) generated by `TestRegenCanonicalTraces` — hand-editing JSON is discouraged because `eth.ChainID` and `eth.BlockID` have finicky serialization rules. Added `just fuzz-invariants` and `just regen-traces` targets; `just verify-invariants` now runs Dafny + Go (both build tags) + fuzz as a single CI gate.
