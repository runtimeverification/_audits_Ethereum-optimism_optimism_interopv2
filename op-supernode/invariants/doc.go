// Package invariants is the Go counterpart of the Dafny models in
// op-supernode/dafny-models. It provides pure predicates that check the
// state invariants documented in op-supernode/invariants/SPEC.md.
//
// The package is organized around a single pure Snapshot type that mirrors
// the Dafny SupernodeState datatype. Every invariant check takes a Snapshot
// and returns an error identifying which invariant (by stable SPEC ID)
// failed. Checks are pure and have no hidden dependencies on the live
// Supernode — they can replay serialized traces from Dafny counter-examples,
// fuzz failures, or production logs through the same code path.
//
// Stable IDs (I1..I12, A1..A5, T1..T6) are the canonical reference for
// cross-layer bug triage. Every error message and every godoc comment that
// references an invariant MUST cite the ID so failures can be traced back
// to the specification.
//
// See:
//   - op-supernode/invariants/SPEC.md (source of truth)
//   - op-supernode/dafny-models/SupernodeState.dfy (Dafny mirror)
//   - op-supernode/dafny-models/SupernodeView.dfy (bridge)
package invariants
