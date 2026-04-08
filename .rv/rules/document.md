# False-Positive Suppression Rules — `op-supernode` Function Analysis

> **Purpose.** This document codifies patterns that produced false positives
> in `function_report.json` (Go, 628 functions, 534 raised "missing
> preconditions"). Attach this file to every future precondition /
> postcondition analysis run. Each finding produced by the analyzer must be
> checked against the rules below **before** being emitted; if a rule matches,
> the finding is suppressed (or downgraded to `informational`) with the
> matching rule ID recorded in `suppressed_by`.
>
> **Scope.** Rules are expressed in two layers:
>
> 1. **Global rules (G-\*)** — generalize across many functions. Match on
>    file path, call-site context, receiver type, or finding text.
> 2. **Per-function overrides (F-\*)** — pinned to one
>    `file_path::function_name` pair. Use only when a global rule is not
>    expressive enough.
>
> **Identifier convention.** Functions are identified as
> `<file_path>::<function_name>`. When the same name appears multiple times
> in one file (method overloads / test helpers), disambiguate with the
> receiver or line, e.g. `supernode.go::(*Supernode).Start`.

---

## 0. How the analyzer should apply these rules

For every candidate `missing_precondition`:

```text
1. Tag the function with context:
     is_test_file            = path endswith "_test.go"
     is_mock                 = summary mentions "mock" | "fake" | "stub" | "double"
     is_init_time            = function is called only from init / NewX / RegisterX / main
     is_getter_or_helper     = function has no state changes and body < 10 LoC
     caller_count            = len(callers)
     receiver_kind           = "value" | "pointer" | "none"

2. Check Global rules (G-1 … G-12) in order. First match wins.
3. If no global match, check Per-function overrides (F-*).
4. If still unmatched, keep the finding.
5. Every emitted finding must cite either the rule it survived (if any
   global rule "almost matched" but didn't) or "no-rule-applied".
```

The output schema for a suppressed finding is:

```json
{
  "short_description": "...",
  "status": "suppressed",
  "suppressed_by": "G-3",
  "reason": "pointer-receiver nil check on type constructed via NewX"
}
```

---

## 1. Global rules

### G-1 — Test files are not production code
**Match.** `file_path` ends with `_test.go`.
**Suppress.** Findings about concurrency, context cancellation, nil
receivers, overflow, and "mock ignores parameter X".
**Why.** Test helpers and fuzz utilities are intentionally simplified.
Flagging them produced **121 of 534** findings in the sample report — pure
noise. The only findings we keep from test files are ones that would cause
the *test itself* to panic on a legitimate input (e.g. empty slice indexing
where the test explicitly passes a slice).
**How to apply.** Drop all findings whose `file_path` matches `_test.go`
unless the finding is about a test-only invariant that the test relies on
(for example, `TestX` passing a literal that would divide by zero).

---

### G-2 — Mock / fake / stub implementations
**Match.** `summary` contains any of: `mock`, `fake`, `stub`, `double`,
`in-memory`, `for testing`, or the type name ends in `Mock`/`Fake`/`Stub`.
**Suppress.** Findings of the form:
- "ignores parameter X, always returns constant"
- "does not validate context"
- "not concurrency safe"
- "does not faithfully simulate cancellation"
**Why.** Test doubles are allowed — and usually *required* — to be
behaviorally incomplete. The "missing" behavior is by design.
**How to apply.** If the type or function summary classifies the target as
a test double, drop behavioural-completeness findings. Keep only findings
that would make the double *lie* in a way the tests rely on.
**Evidence.** `logdb_test.go::FindSealedBlock`, `logdb_test.go::OpenBlock`,
`super_authority_test.go::CurrentL1`, `L2BlockRefByLabel` mocks.

---

### G-3 — Pointer-receiver nil guards
**Match.** Finding text matches `/nil receiver|r must not be nil|nil pointer dereference/i`
AND the receiver type is constructed through an exported `NewX` /
`MakeX` / factory function OR stored in a struct field that is only
assigned during construction.
**Suppress.** "Method does not guard against nil receiver."
**Why.** In idiomatic Go, pointer-receiver methods are not expected to
handle `nil` unless the type documents that contract. All `*Result`,
`*Activity`, `*ChainContainer`, `*VirtualNode` etc. in this repo are
constructed through factories and stored in non-nil fields.
**Exceptions.** KEEP the finding if:
- the type is used as an interface value that callers may legitimately
  leave unset,
- or a sibling method already has an explicit `if r == nil { ... }` guard
  (then consistency matters).
**Evidence.** `types.go::(*Result).IsEmpty`, `types.go::(*Result).IsValid`,
`activity.go::(*Activity).RPCService`.

---

### G-4 — Context not checked in pure/non-I/O functions
**Match.** Finding text matches `/context.*cancel|ctx\.err|deadline/i` AND
the function does NOT perform network I/O, disk I/O, or channel receives.
**Suppress.** "Function accepts ctx but never checks ctx.Err()."
**Why.** Go's convention is: only functions that can block should
consult `ctx`. Pure lookups, map reads, and arithmetic helpers should
*not* inject context checks — doing so adds cost and noise.
**Exceptions.** KEEP if the function is on the hot path of a request and
the caller's contract explicitly says "context-cancellable."

---

### G-5 — "Not concurrency safe" for init-time / single-writer functions
**Match.** Finding text matches `/not concurrency safe|race|mutex|concurrent/i`
AND one of:
- `function_name` starts with `Register*`, `Set*`, `With*`, `NewX`, `Init*`
- the function is only called from `init()`, `main`, a `NewX` factory, or
  a `Build` method on a builder
- `caller_count <= 2` and all callers are constructors/builders
**Suppress.** "No mutex protects X."
**Why.** Builder-pattern and registration functions are single-threaded
by contract: they run before the service starts accepting requests.
Adding a mutex would be premature and misleading.
**Evidence.** `chain_container.go::RegisterVerifier`,
`chain_container.go::SetResetCallback`, `flags.go::RegisterActivityFlags`,
`virtual_cli.go::NewVirtualCLI`.

---

### G-6 — Protocol-bounded arithmetic overflow
**Match.** Finding text matches `/overflow|underflow|truncat/i` on a value
that is:
- a block number (`uint64`),
- an L2 timestamp derived from `Genesis.L2Time + n * BlockTime`,
- or a log index (`log.Index`) cast to `uint32`.
**Suppress.** "No overflow guard on blocknum * BlockTime."
**Why.** In this codebase:
- `BlockTime` is 2s and block numbers are bounded by wall-clock time;
  `n * BlockTime` overflow would require >500 billion years.
- `log.Index` is a per-block receipt index; Ethereum blocks cannot
  contain 2³² logs.
These are protocol invariants, not bugs. Flagging them is noise.
**Exceptions.** KEEP the finding if the multiplication uses a *user-supplied*
delta that is not validated upstream (e.g. a fuzz seed).
**Evidence.** `chain_container.go::BlockNumberToTimestamp`,
`chain_fuzz_utils.go::ExecMsgForLog`.

---

### G-7 — "Parameter must not be nil" when already non-nil at every call site
**Match.** Finding text matches `/must not be nil|non-nil|not validated for nil/i`
on a parameter P, AND every caller listed in `callers` passes P as either
`&literal`, the result of a `NewX` factory, or a struct field that is set
during construction.
**Suppress.** "P is not validated for nil before dereference."
**Why.** Go does not require defensive nil checks when call-site
invariants guarantee non-nil. The cost of a runtime check is small but
the *volume* of these findings is large; almost all are noise.
**How to apply.** Walk the caller list (already in `callers` /
`caller_ids`). If ALL callers provably pass non-nil, suppress. If one
caller is ambiguous (e.g. a return value from an interface method),
downgrade to `informational` instead of suppressing.
**Evidence.** `supernode.go::Start` (log, chains, cfg params),
`virtual_node.go::Setup` (vncfg, rpc).

---

### G-8 — "Intentional behavior" flagged as missing precondition
**Match.** Finding text matches `/ignores parameter|silently|always returns|unused/i`
AND function summary says the behavior is deliberate (e.g. "returns a
constant", "mock returns fixed value", "trivial helper").
**Suppress.** Completely.
**Why.** These are *descriptions* of behaviour, not bugs. The analyzer
was mis-labeling "this function does X" as "function fails to do Y".
**Evidence.** `ptrDecision` (trivial test helper flagged as "copies by
value"), `super_authority_test.go::Reset` (test stub).

---

### G-9 — Lifecycle ordering rules are not races
**Match.** Finding text matches `/data race|race condition|set.*while.*running/i`
AND the function is a pre-Start configuration hook (`Set*`, `With*`,
`Register*`) whose only legal call site is before `Start()`.
**Suppress.** "Calling SetX while Start() is running is a data race."
**Why.** The contract is: configuration happens before lifecycle. A
violator would already be wrong; the analyzer is flagging theoretical
misuse, not a defect.
**Exceptions.** KEEP if the function's godoc says "safe to call at any
time" or if the field is read from a goroutine that *could* observe a
torn write even under normal use.
**Evidence.** `chain_container.go::SetResetCallback`.

---

### G-10 — Bounds checks on values constrained by caller loops
**Match.** Finding text matches `/bounds|out of range|index.*len/i` AND
the function is called from inside a `for i := 0; i < len(xs); i++`
loop or equivalent ranged iteration.
**Suppress.** "i is not bounds-checked."
**Why.** The caller's loop bound IS the precondition.
**How to apply.** Requires looking at the callsite — if the callsite is
not inspected, downgrade to `informational` instead of suppressing.

---

### G-11 — "Caller must guarantee X" when X is already in the type system
**Match.** Finding text says "caller must ensure type-compatible input"
or "enum value must be valid" AND the parameter type is a Go enum-like
type (`type Decision int`) with exhaustive values.
**Suppress.** "Decision value must be one of {A,B,C}."
**Why.** Go's type system already restricts the domain; adding runtime
checks for "impossible" enum values is noise.

---

### G-12 — Map-shared-by-reference / aliasing notes
**Match.** Finding text matches `/shared by reference|aliasing|map.*reused/i`
**Downgrade, do not suppress.** Move to `informational`, not a missing
precondition. These are design observations; whether they are bugs
depends on the caller's intent.
**Why.** Go's value semantics make aliasing a normal design tool. The
analyzer should surface these once as an *observation* per type, not as
a precondition violation on every method that returns or accepts the
aliased value.
**Evidence.** `types.go::ToVerifiedResult` (L2Heads map aliasing).

---

## 2. Per-function overrides

Overrides use the format:

```
F-<n>  <file_path>::<function_name>
  rule:     <what to do>
  reason:   <why the generic rule is insufficient>
  extends:  <global rule ID that partially applies, if any>
```

### F-1  `op-supernode/supernode/activity/interop/decide_test.go::ptrDecision`
- **rule:** Suppress all findings. This function is a 2-line test helper
  `func ptrDecision(d Decision) *Decision { return &d }`.
- **reason:** Cannot fail under any input; any finding is by definition
  a false positive.
- **extends:** G-1, G-8.

### F-2  `op-supernode/supernode/activity/interop/types.go::(*Result).IsEmpty`
- **rule:** Suppress "nil receiver" finding.
- **reason:** `Result` is only ever constructed via struct literals in
  this package; no call site produces `(*Result)(nil)`.
- **extends:** G-3.

### F-3  `op-supernode/supernode/activity/interop/types.go::(*Result).IsValid`
- **rule:** Suppress "nil receiver" finding. Same reasoning as F-2.
- **extends:** G-3.

### F-4  `op-supernode/supernode/chain_container/chain_container.go::BlockNumberToTimestamp`
- **rule:** Suppress "overflow on blocknum * BlockTime" and "BlockTime == 0".
- **reason:** `BlockTime` is a rollup-config constant validated at
  service init; `blocknum` is bounded by wall-clock time.
- **extends:** G-6.

### F-5  `op-supernode/supernode/chain_container/chain_container.go::RegisterVerifier`
- **rule:** Suppress "not concurrency-safe". Keep the "nil verifier"
  finding if callers are not all inspected.
- **reason:** Called once per chain from `NewChainContainer` before
  `Start()`.
- **extends:** G-5.

### F-6  `op-supernode/supernode/chain_container/chain_container.go::SetResetCallback`
- **rule:** Suppress "data race with RewindEngine".
- **reason:** Contract: must be called before `Start()`. Document in
  godoc instead of guarding.
- **extends:** G-9.

### F-7  `op-supernode/flags/virtual_cli.go::NewVirtualCLI`
- **rule:** Suppress "base not validated for nil".
- **reason:** The only caller is `flags.go::upstreamVirtualFlags` which
  passes `flags.NewCLIConfig(...)` — a non-nil constructor.
- **extends:** G-7.

### F-8  `op-supernode/supernode/activity/interop/checker_test.go::L1BlockRefByNumber`
- **rule:** Suppress "ctx never checked", "returns sentinel on missing key".
- **reason:** Mock for test of upstream verifier; cancellation is not
  part of what the unit under test exercises.
- **extends:** G-1, G-2, G-4.

### F-9  `op-supernode/supernode/activity/interop/chain_fuzz_utils.go::ExecMsgForLog`
- **rule:** Suppress "log.Index uint→uint32 truncation".
- **reason:** Log index per block fits in uint32 by Ethereum design.
- **extends:** G-6.

### F-10  `op-supernode/supernode/activity/heartbeat/heartbeat.go::(*Heartbeat).Stop`
- **rule:** KEEP the finding (do not suppress). The Start/Stop interleaving
  is a *real* TOCTOU race on `h.cancel` that should be fixed with a mutex
  or atomic pointer.
- **reason:** This is the canonical example of a finding that *survives*
  suppression. Listed here as a positive marker so regression in future
  analysis runs is caught.

---

## 3. Output expectations for future runs

Every analyzer run must produce, in addition to `function_report.json`:

```
.rv/analysis/<timestamp>/
  ├── report.json            # same schema, but with `status` + `suppressed_by`
  ├── suppressions.csv       # (file, fn, rule, reason)   — audit trail
  └── rule_stats.json        # { "G-1": 121, "G-3": 6, ... }  — drift detection
```

If a rule fires more than **20%** above its historical baseline, the
analyzer should emit `rule_drift_warning` so humans can revisit whether
the rule is too aggressive.

---

## 4. Script plan — LangChain / LangGraph pipeline

### 4.1 Goals

1. **Speed.** The current run made 123 LLM calls (cache hit rate 0%). We
   want <30 calls per incremental run by caching at the function level.
2. **Cost.** Batch small functions into a single prompt; use the cheapest
   capable model (Haiku / GPT-4o-mini) for classification and only
   escalate to a larger model when the classifier disagrees with the
   rules engine.
3. **Reusability.** The pipeline must work on any Go package, not just
   `op-supernode`. All project-specific knowledge lives in
   `.rv/rules/document.md`.
4. **Explainability.** Every emitted finding must carry a
   `rule_applied` / `suppressed_by` field.

### 4.2 Architecture (LangGraph state machine)

```
            ┌────────────┐
            │  discover  │  scan repo, build callgraph
            └─────┬──────┘
                  ▼
            ┌────────────┐
            │  chunk     │  group functions: 1 function = 1 node; small
            │            │  helpers batched 8–12 per LLM call
            └─────┬──────┘
                  ▼
            ┌────────────┐
            │  classify  │  LLM cheap model: produce raw findings
            └─────┬──────┘
                  ▼
            ┌────────────┐
            │  rule_gate │  Python: apply .rv/rules/document.md
            │            │  (G-* and F-*), emit status + suppressed_by
            └─────┬──────┘
                  ▼
            ┌────────────┐
            │  escalate  │  if rule_gate is uncertain (edge cases)
            │            │  → re-ask a larger model with the rule text
            │            │  quoted in the prompt
            └─────┬──────┘
                  ▼
            ┌────────────┐
            │  persist   │  report.json + suppressions.csv + rule_stats
            └────────────┘
```

### 4.3 Caching strategy

- **Level 1 — function hash.** `llm_cache_hash` already exists in the
  current report. Reuse it: key = `sha256(source + deps + rules_hash)`.
  When `rules_hash` changes, invalidate everything that matched a global
  rule but not the per-function overrides.
- **Level 2 — rule verdict.** Store the rule decision separately so a
  rules-only update (common) does not re-invoke the LLM.
- **Level 3 — SQLite or DuckDB** for the cache; cheap, portable, diffable.

### 4.4 Prompting strategy

- **System prompt** includes: "You are auditing Go code. Before emitting
  any finding, check it against the attached false-positive rules. If a
  rule suppresses the finding, do NOT emit it."
- **Rules file is attached verbatim** (this document, <4 KB when
  stripped of comments). Small enough to fit in every call without
  blowing the context budget.
- **Few-shot examples** drawn from the "Evidence" sections above —
  5 real suppressed findings + 1 real KEPT finding (F-10) to anchor the
  model's calibration.
- **Output contract.** JSON-only, schema enforced via
  `ChatOpenAI.with_structured_output(Pydantic)` or Anthropic tool use.

### 4.5 Cost model (rough)

| Step        | Tokens in | Tokens out | Model          | Per-fn ¢ |
|-------------|-----------|------------|----------------|----------|
| classify    | 2k        | 300        | haiku-4.5      | 0.03     |
| rule_gate   | 0 (local) | 0          | —              | 0.00     |
| escalate    | 4k        | 600        | sonnet-4.6     | 0.2      |

With ~10% escalation rate over 628 functions: **~$0.32 per full run**,
**~$0.05 per incremental run** (cache hits dominate).

### 4.6 Reusability hooks

- `rules/document.md` is loaded as a LangChain `Document` and chunked if
  needed. The pipeline reads rule IDs from regex (`^### (G-\d+|F-\d+)`)
  so adding a rule requires no code change.
- The callgraph producer is pluggable: Go uses `golang.org/x/tools/go/ssa`;
  Rust would use `cargo-geiger` + `rust-analyzer`; Solidity would use
  `slither --print call-graph`. The `discover` node is the only language-
  specific component.
- The rule gate is a pure function
  `apply_rules(finding, fn_metadata, rules) -> verdict`
  with unit tests against a frozen corpus in `.rv/testdata/`.

---

## 5. Discovery phase — what to do FIRST on the next run

Before writing any code, run the following offline discovery on
`function_report.json` to validate the rules above and surface new
clusters:

1. **Cluster findings by their first 6 words.** The top-20 clusters
   already cover ~70% of volume (lock/mutex, nil-receiver, ctx-check).
   If any new cluster exceeds 15 findings, write a new G-\* rule for it
   *before* coding the pipeline.
2. **Compute suppression projection.** For each rule above, simulate
   applying it to the current report and record how many findings it
   would suppress. Target: **≥75% of the 534 findings**, concentrated in
   G-1, G-2, G-5, G-6, G-7.
3. **Precision sanity check.** Randomly sample 30 suppressed findings,
   have a human confirm each is indeed a false positive. If precision
   <90%, tighten the offending rule (usually by requiring a stronger
   contextual signal like caller-site inspection).
4. **Record a baseline.** Save the rule fire counts as
   `.rv/baselines/2026-04.json`. All future runs compare to this
   baseline; drift >20% triggers a human review.
5. **Write a "must-keep" corpus.** Start with F-10 (heartbeat Stop race)
   and grow it over time. Every rule change must re-run the must-keep
   corpus and confirm zero of them are suppressed.

---

## 6. Changelog

- **2026-04-08** — initial draft. 12 global rules + 10 per-function
  overrides, derived from `function_report.json` (628 fns, 534 raw
  findings). Projected suppression rate: ~78%.
