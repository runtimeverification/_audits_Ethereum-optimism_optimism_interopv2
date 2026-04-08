package invariants

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// StateView is the pull-based adapter interface between the live
// `supernode.Supernode` and this package's pure `Snapshot` type.
//
// Why an interface and not a concrete function that takes *Supernode:
//
//  1. The `invariants` package stays decoupled from `supernode` — no
//     import cycle risk even if supernode grows to reference invariants
//     via the runtime-assertion hook.
//  2. The adapter lives in `op-supernode/supernode/invariants_bridge.go`
//     where it can read unexported fields (`sn.chains`, `sn.cfg`, and
//     activity-registry internals).
//  3. Tests and fuzzers can supply a fake StateView without standing up a
//     real Supernode — e.g. the reference model already exposes a
//     Snapshot, and tests can wrap it with a trivial StateView.
//
// Implementations are responsible for lock discipline. Per the subagent's
// structural map of op-supernode as of Step 3b:
//
//   - `sn.chains` is append-only at startup; no lock needed.
//   - `interop.VerifiedDB` has an internal RWMutex; callers get a
//     consistent read via `Get` / `LastTimestamp`.
//   - `chain_container.DenyList` has an internal RWMutex.
//   - `interop.LogsDB` wraps an `op-supervisor` logs DB with its own
//     concurrency control.
//
// So long as each query method respects the relevant internal mutex, the
// snapshot produced by `SnapshotFrom(view)` represents a valid
// interleaving of the underlying state at the time of the call.
//
// TIMING CONSTRAINT (SPEC.md §5 item 17): SnapshotFrom MUST be called at a
// stable checkpoint, after a completed applyPendingTransition cycle. Do
// NOT call it from inside a mutating method — the VirtualNode can be
// mid-recreation, LogsDB tails can hold speculative blocks, and DenyList
// mutations are not atomic with LogsDB mutations. The canonical runtime
// hook is:
//
//   // Inside Supernode.progressAndRecord, AFTER applyPendingTransition:
//   invariants.AssertWith(func() invariants.Snapshot {
//       return invariants.SnapshotFrom(s.stateView)
//   })
type StateView interface {
	// ActivationTS returns t_0 from SPEC.md §0.
	ActivationTS() uint64

	// Chains returns the chain IDs the Supernode is tracking. The
	// returned slice is owned by the caller; implementations should
	// return a fresh copy.
	Chains() []eth.ChainID

	// Verified returns the VerifiedDB contents sorted ascending by
	// timestamp. This matches the Dafny VerifiedMapToSortedSeq shape.
	Verified() []VerifiedEntry

	// LogsDBFor returns the LogsDB entries for a single chain, oldest
	// first, as of the moment of the call. Returns nil for chains
	// without any recorded entries. The returned slice is owned by the
	// caller.
	//
	// NOTE: as of Step 3b, `interop.LogsDB` has no "enumerate all"
	// method — only range-based queries (`FindSealedBlock`, `OpenBlock`,
	// `Contains`). Implementations must walk from block 0 (or the
	// latest-sealed back to genesis) using those methods. Adding a
	// first-class `EnumerateSealed(from, to)` method to the op-supervisor
	// logs DB is tracked as SPEC §5 item 13.
	LogsDBFor(chain eth.ChainID) []BlockWithLogs

	// DenyListFor returns the DenyList entries for a single chain in
	// insertion order. Returns nil if no blocks have been denied on that
	// chain. The returned slice is owned by the caller.
	//
	// NOTE: `chain_container.DenyList` currently only exposes
	// `IsDenied(height, hash)`, which is a point query. Enumeration
	// requires adding a `ForEach` / `Enumerate` method that iterates the
	// bbolt bucket. Tracked as SPEC §5 item 14.
	DenyListFor(chain eth.ChainID) []DenyListEntry
}

// SnapshotFrom assembles a pure Snapshot from a StateView. It is the
// single entry point every runtime / test / fuzz / replay consumer uses
// to get a snapshot for `CheckAll`.
//
// SnapshotFrom is resilient to nil slices and missing keys — the returned
// Snapshot is always well-formed (every chain in `Chains()` has an entry
// in LogsDB and DenyList, even if the entry is a nil slice).
func SnapshotFrom(v StateView) Snapshot {
	if v == nil {
		return Snapshot{}
	}
	chains := v.Chains()
	s := Snapshot{
		ActivationTS: v.ActivationTS(),
		Chains:       append([]eth.ChainID{}, chains...),
		LogsDB:       make(map[eth.ChainID][]BlockWithLogs, len(chains)),
		DenyList:     make(map[eth.ChainID][]DenyListEntry, len(chains)),
		Verified:     v.Verified(),
	}
	for _, c := range chains {
		s.LogsDB[c] = v.LogsDBFor(c)
		s.DenyList[c] = v.DenyListFor(c)
	}
	return s
}

// StaticStateView is a trivial StateView backed by an already-assembled
// Snapshot. Useful for tests and for replaying a serialized trace through
// the StateView machinery.
type StaticStateView struct {
	S Snapshot
}

func (v StaticStateView) ActivationTS() uint64  { return v.S.ActivationTS }
func (v StaticStateView) Chains() []eth.ChainID { return append([]eth.ChainID{}, v.S.Chains...) }

// Verified deep-copies the Verified slice — including each entry's
// L2Heads map — so callers can mutate the result without aliasing the
// underlying snapshot.
func (v StaticStateView) Verified() []VerifiedEntry {
	return cloneVerifiedSlice(v.S.Verified)
}

// LogsDBFor deep-copies the per-chain LogsDB slice including each
// block's ExecMsgs slice.
func (v StaticStateView) LogsDBFor(c eth.ChainID) []BlockWithLogs {
	return cloneBlockWithLogsSlice(v.S.LogsDB[c])
}

func (v StaticStateView) DenyListFor(c eth.ChainID) []DenyListEntry {
	return append([]DenyListEntry{}, v.S.DenyList[c]...)
}
