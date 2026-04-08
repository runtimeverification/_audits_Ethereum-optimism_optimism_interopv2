# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

`op-supernode` is a Go service inside the Optimism monorepo that runs multiple OP Stack chains in a single process by virtualizing `op-node`. Each chain runs as an isolated in-memory "Virtual Node" worker, while L1/beacon clients, RPC, metrics, and data dirs are shared/namespaced across chains.

Repo-wide conventions from `../CLAUDE.md` apply (default branch `develop`, Just-based build system under `../justfiles/`).

## Commands

Build and test from this directory (`op-supernode/`):

```bash
just op-supernode         # build ./bin/op-supernode (ldflags set via justfile)
just test                 # go test ./...
just coverage             # go test -coverprofile=coverage.out, then per-func report
just clean                # remove ./bin/op-supernode
```

The `Makefile` only re-exports these targets as deprecated shims that delegate to `just` via `../justfiles/deprecated.mk`. Prefer `just` directly.

Run a single Go test:

```bash
go test ./supernode/chain_container/... -run TestName -v
go test ./supernode -run TestShutdown -v -count=1
```

The shared `go_build` / `go_test` recipes live in `../justfiles/go.just`.

## Architecture

Entry point: `cmd/main.go` → `config/config.go` (`CLIConfig`) → `supernode/supernode.go` (`Supernode` struct). `Supernode` owns the lifecycle of every subsystem listed below and is started/stopped as a single unit.

Three main concepts to understand before editing:

### 1. Chain Containers (`supernode/chain_container/`)

A `ChainContainer` is the per-chain boundary. It wraps a Virtual Node (currently only `op-node`, see `virtual_node/`), an `engine_controller`, invalidation logic (`invalidation.go`), and a `super_authority` that mediates cross-chain authority. Containers expose a stable interface so the rest of `Supernode` never touches rollup-node internals directly. Adding a new capability for "a chain" almost always means extending this package.

### 2. Shared Resources (`supernode/resources/`)

Resources are dependencies injected into every Virtual Node so redundant work is eliminated:

- `shared_clients.go` — single L1 client + L1 beacon client reused by all chains (constructed in `Supernode.Start`). Per-chain `l1` / `l1.beacon` flags are intentionally ignored.
- `rpc_router.go` — namespaced RPC registry. Per-chain RPC methods are exposed under `<chainID>/` (e.g. `11155420/`, `84532/`). Activities register into the root namespace.
- `metrics_service.go` + `metrics_fan_in.go` — per-chain metrics are fanned into a single Prometheus endpoint; expect this to migrate to label-based dimensions in the future.
- Data directories are namespaced per chain to isolate SafeDB/P2P state.

### 3. Activities (`supernode/activity/`)

Activities are modular plugins for concerns that are **not** chain-specific. An activity may be an `RPCActivity` (registers handlers into the root RPC namespace) and/or a `RunnableActivity` (gets a goroutine during runtime). Current activities:

- `heartbeat/` — `heartbeat_check` RPC + liveness log emission.
- `superroot/` — `superroot_atTimestamp` RPC, builds SuperRoots from verified L2 blocks across the dependency set (proofs-oriented).
- `supernode/` — `supernode_syncStatus` aggregate sync across chains.
- `interop/` — interop-related activity.

When adding new cross-chain functionality, prefer a new Activity over touching `Supernode` directly. Chain-specific functionality belongs in `ChainContainer` or a shared `resource`.

## Flags (`flags/`)

`flags.go` defines top-level `op-supernode` flags. `virtual_flags.go` / `virtual_cli.go` upstream every `op-node` flag into the supernode CLI under a namespace:

- `--vn.<chainID>.<flag>` sets a flag for a single chain
- `--vn.all.<flag>` sets a flag for every chain

Several `op-node` flags are intentionally overridden at the supernode level and will **not** take effect per-chain — this is load-bearing, not a bug:

- `l1`, `l1.beacon` (shared client)
- log and metric settings (inherited from top-level)
- `p2p` (listen ports forced to `0`; global disable via `--disable-p2p`)

Environment variable names for these flags still contain `op-node` prefixes due to how upstreaming works — consult `--help` rather than guessing names.

## Dafny Models

`dafny-models/` contains formal models of supernode behavior (see also the `Add Dafny models` commit on `develop`). These are reference specs, not built by `just op-supernode`. Run with `just verify-dafny` (requires a local Dafny install).

File map:

- `Types.dfy` — shared datatypes (BlockID, VerifiedResult, ExecutingMessage, BlockWithLogs, DenyListEntry, `IsParentOf` helper).
- `VerifiedDB.dfy` — the VerifiedDB class; its `Invariants()` predicate covers only no-gaps timestamp structure.
- `Supernode.dfy` — Supernode class skeleton. `applyRewindPlan`, `sameL1Chain`, and `resolveFrontierVerificationView` are trivial no-op stubs (returning `false` / `None`); the `{:axiom}`-tagged postconditions have been removed. Real implementations are tracked in `invariants/SPEC.md` §5 items 1′–3′ (Step 2c).
- `SupernodeState.dfy` — abstract `SupernodeState` datatype + predicates I1–I11 + `AllInvariants` + `I12_InitialStateSatisfiesInvariants` lemma. This is the canonical Dafny form of the invariants in `invariants/SPEC.md`.
- `SupernodeView.dfy` — bridge layer. Contains `BuildSupernodeState`, `ComponentsSatisfyInvariants`, abstract `VerifiedMapToSortedSeq` with algebraic lemmas, and `InitialComponentsSatisfyInvariants`. Method contracts in Step 2c should reference `ComponentsSatisfyInvariants` rather than calling `AllInvariants` inline so the bridge point stays visible.

`invariants/SPEC.md` is the single source of truth tying the Dafny predicates, Go runtime checks, and fuzz/reference-model tests together. Every predicate, Go check, and test failure should cite the stable ID (I1–I12, A1–A5, T1–T6) from that file. Consult `safety-labels.md` for the orthogonal safety-label vocabulary used by the chain container.

## Invariant verification pipeline

Five layers enforce the same `invariants/SPEC.md` catalog:

1. **Dafny proofs** (`dafny-models/`) — formal predicates and lemmas. Run with `just verify-dafny`. Current status: 66 verified, 0 errors.
2. **Go invariant package** (`invariants/`) — `Snapshot` type mirroring Dafny's `SupernodeState`, plus `CheckI*` predicates returning structured `*Error` values carrying stable SPEC IDs. Run with `just verify-invariants-go`.
3. **Environmental oracle** (`invariants/env.go`) — `Env` interface + `CheckAllWithEnv` for I1/I3/I8/I9 (the hybrid clauses that need L1/L2 lookups). Implementations: `fakeEnv` in tests; production adapter pending in Step 3c.
4. **Reference model + fuzzer** (`invariants/reference_model.go`, `fuzz_test.go`) — pure Go implementation of T3/T4/T5 transitions, driven by a native Go fuzzer (`FuzzReferenceModel`). Runs with `just fuzz-invariants` (default 30s, `FUZZTIME=5m` to extend). Fuzz enforces `CheckAll` after every transition.
5. **Runtime assertions** (`invariants/runtime_on.go` / `runtime_off.go`) — behind the `supernode_invariants` build tag. `invariants.Assert(snapshot)` and `AssertWith(builder)` are zero-cost no-ops in release builds and full checks under the tag. Run tagged tests with `just verify-invariants-go-tagged`.

`just verify-invariants` runs all five layers as a single CI gate:
```
verify-dafny → verify-invariants-go → verify-invariants-go-tagged → fuzz-invariants
```

Trace replay: `invariants/testdata/traces/*.json` is the regression corpus. Add a trace via `invariants.SaveTrace`; `TestTraceCorpus` replays every file. Regenerate canonical traces with `just regen-traces`. Do NOT hand-edit trace JSON — use `TestRegenCanonicalTraces` so the reference-model constructors produce the canonical byte layout.

**Never introduce a new invariant without giving it a stable ID in `invariants/SPEC.md` first.** All five layers look up behavior by ID; divergence between Dafny / Go / SPEC is the single failure mode this pipeline is designed to catch.

## Invariants package file map

- `SPEC.md` — single source of truth for invariant IDs and transition semantics.
- `snapshot.go` — `Snapshot`, `BlockWithLogs`, `BlockRef`, `ExecutingMessage`, `VerifiedEntry`, `DenyListEntry`. Mirrors Dafny `SupernodeState`.
- `invariants.go` — `CheckAll` + structural `CheckI2/I4/I5/I6/I7/I8/I10/I11`. Every error carries a stable SPEC ID.
- `env.go` — `Env` interface and hybrid `CheckI1/I3/I8-env/I9` predicates; `CheckAllWithEnv`.
- `reference_model.go` — pure T3/T4/T5 `Apply*` functions + `CloneSnapshot` + `NewInitialSnapshot`.
- `bridge.go` — `StateView` interface and `SnapshotFrom(view)` lifter; `StaticStateView` fake for tests.
- `runtime_on.go` / `runtime_off.go` — build-tagged assertion hook.
- `trace.go` — JSON trace format and replay primitives.
- `testdata/traces/*.json` — regression corpus.

Integration with the actual `supernode.Supernode` (filling in `StateView` against the real chain-container / interop-activity state) is Step 3c. See `SPEC.md` §5 items 13–15 for the blocking gaps (missing `EnumerateSealed` / `ForEach` APIs on `interop.LogsDB` and `chain_container.DenyList`).
