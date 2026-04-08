//go:build supernode_invariants

package invariants

import (
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestAssert_Violation exercises the enabled-build code path: a failing
// snapshot routes through the failureSink and failCount without panicking
// (panicOnFailure is disabled for the duration of the test).
func TestAssert_Violation(t *testing.T) {
	SetPanicOnFailure(false)
	defer SetPanicOnFailure(true)

	var captured error
	SetFailureSink(func(err error) { captured = err })
	defer SetFailureSink(nil)

	before := AssertionFailureCount()

	bad := Snapshot{
		Chains: []eth.ChainID{chainA()},
		LogsDB: map[eth.ChainID][]BlockWithLogs{
			chainA(): {
				block(1, 0, 1, 100),
				block(2, 99, 2, 101), // I2 violation
			},
		},
	}
	Assert(bad)

	if captured == nil {
		t.Fatal("failure sink should have been called")
	}
	var ie *Error
	if !errors.As(captured, &ie) {
		t.Fatalf("expected *invariants.Error, got %T: %v", captured, captured)
	}
	if AssertionFailureCount() != before+1 {
		t.Fatalf("failure count should have incremented")
	}
}

func TestAssert_OK(t *testing.T) {
	SetPanicOnFailure(false)
	defer SetPanicOnFailure(true)
	before := AssertionFailureCount()
	good := NewInitialSnapshot(100, []eth.ChainID{chainA()})
	Assert(good)
	if AssertionFailureCount() != before {
		t.Fatalf("valid snapshot should not increment failure count")
	}
}

func TestAssert_ZeroSnapshot(t *testing.T) {
	// An empty snapshot is a valid initial state and must not be skipped.
	// CheckAll runs every predicate; with no chains/logs/verified/deny
	// nothing has anything to violate, so the call must not increment
	// failCount.
	SetPanicOnFailure(false)
	defer SetPanicOnFailure(true)
	before := AssertionFailureCount()
	Assert(Snapshot{})
	if AssertionFailureCount() != before {
		t.Fatalf("empty snapshot should satisfy CheckAll")
	}
}

func TestAssertionsEnabledConst(t *testing.T) {
	if !AssertionsEnabled {
		t.Fatal("AssertionsEnabled should be true under -tags=supernode_invariants")
	}
}
