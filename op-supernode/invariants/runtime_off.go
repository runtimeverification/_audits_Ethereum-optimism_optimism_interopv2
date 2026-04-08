//go:build !supernode_invariants

package invariants

// Runtime assertions are DISABLED in this build.
//
// All Assert* entry points compile to no-ops with zero allocations so
// that production binaries carry no runtime overhead from the invariant
// machinery. Enable by passing -tags=supernode_invariants to go build or
// go test.

const AssertionsEnabled = false

// Assert is a no-op when the supernode_invariants build tag is not set.
func Assert(s Snapshot) {}

// AssertWith is a no-op when the supernode_invariants build tag is not set.
// The builder is NOT invoked, so any side effects inside it are skipped.
func AssertWith(build func() Snapshot) {}

// SetPanicOnFailure is a no-op in release builds.
func SetPanicOnFailure(panic bool) {}

// SetFailureSink is a no-op in release builds.
func SetFailureSink(sink func(err error)) {}

// AssertionFailureCount always returns zero in release builds.
func AssertionFailureCount() uint64 { return 0 }
