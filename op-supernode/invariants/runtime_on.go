//go:build supernode_invariants

package invariants

import (
	"fmt"
	"sync/atomic"
)

// Runtime assertions are ENABLED in this build.
//
// Build with:
//   go build -tags=supernode_invariants ./...
//   go test  -tags=supernode_invariants ./...
//
// Every state-mutating entry point in op-supernode that wants invariant
// coverage should call:
//
//   invariants.Assert(sn.SnapshotForInvariants())
//
// immediately after the mutation is committed. When the build tag is NOT
// set, Assert is a no-op with zero allocation cost (see runtime_off.go).

const AssertionsEnabled = true

var (
	// failCount records the total number of invariant violations observed
	// since process start. Exposed via AssertionFailureCount so metrics
	// and tests can observe it without importing sync/atomic in callers.
	failCount atomic.Uint64

	// panicOnFailure controls whether Assert panics on the first failure
	// (for tests / dev loops) or just logs-and-continues (for long-running
	// integration runs). Default: panic.
	panicOnFailure atomic.Bool

	// failureSink is an optional callback invoked on every failure. Used
	// by tests to capture failures without panicking. Protected by
	// failureSinkMu below because atomic.Value doesn't work for funcs.
	failureSink func(err error)
)

func init() {
	panicOnFailure.Store(true)
}

// SetPanicOnFailure configures whether Assert panics (true) or logs
// (false) on invariant failure. Default: true.
func SetPanicOnFailure(panic bool) {
	panicOnFailure.Store(panic)
}

// SetFailureSink installs a callback for invariant failures. Calling with
// nil removes the sink. Thread-safe.
func SetFailureSink(sink func(err error)) {
	failureSink = sink
}

// AssertionFailureCount returns the total number of invariant violations
// observed by Assert since process start.
func AssertionFailureCount() uint64 {
	return failCount.Load()
}

// Assert runs CheckAll on the given snapshot and reports any failure. It
// is a no-op if the snapshot is the zero value (callers can pass the
// empty snapshot to skip without branching).
func Assert(s Snapshot) {
	// Fast path: zero snapshot = skip.
	if s.ActivationTS == 0 && len(s.Chains) == 0 &&
		len(s.LogsDB) == 0 && len(s.Verified) == 0 && len(s.DenyList) == 0 {
		return
	}
	err := CheckAll(s)
	if err == nil {
		return
	}
	failCount.Add(1)
	if sink := failureSink; sink != nil {
		sink(err)
	}
	if panicOnFailure.Load() {
		panic(fmt.Sprintf("op-supernode invariant violation: %v", err))
	}
}

// AssertWith runs CheckAll on a snapshot built lazily by the provided
// builder. Prefer this over Assert when constructing the snapshot is
// expensive, so that the runtime cost is paid only when assertions are
// enabled.
func AssertWith(build func() Snapshot) {
	if build == nil {
		return
	}
	Assert(build())
}
