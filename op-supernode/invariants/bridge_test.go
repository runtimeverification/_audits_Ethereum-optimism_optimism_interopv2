package invariants

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestSnapshotFrom_RoundTrip(t *testing.T) {
	// Build a non-trivial snapshot via the reference model.
	original := NewInitialSnapshot(100, []eth.ChainID{chainA(), chainB()})
	original, err := ApplyAdvance(original, blockID(10, 1000),
		map[eth.ChainID]BlockWithLogs{
			chainA(): block(1, 0, 1, 100),
			chainB(): block(2, 0, 1, 100),
		})
	if err != nil {
		t.Fatal(err)
	}
	original, err = ApplyAdvance(original, blockID(11, 1001),
		map[eth.ChainID]BlockWithLogs{
			chainA(): block(3, 1, 2, 101),
			chainB(): block(4, 2, 2, 101),
		})
	if err != nil {
		t.Fatal(err)
	}

	// Wrap in a StaticStateView and go through SnapshotFrom.
	view := StaticStateView{S: original}
	recovered := SnapshotFrom(view)

	// The recovered snapshot must still satisfy CheckAll.
	if err := CheckAll(recovered); err != nil {
		t.Fatalf("SnapshotFrom output violates invariants: %v", err)
	}

	// And every structural field should match.
	if recovered.ActivationTS != original.ActivationTS {
		t.Fatalf("activation drift: %d vs %d", recovered.ActivationTS, original.ActivationTS)
	}
	if len(recovered.Chains) != len(original.Chains) {
		t.Fatalf("chain count drift: %d vs %d", len(recovered.Chains), len(original.Chains))
	}
	if len(recovered.Verified) != len(original.Verified) {
		t.Fatalf("verified count drift: %d vs %d", len(recovered.Verified), len(original.Verified))
	}
	for _, c := range original.Chains {
		if len(recovered.LogsDB[c]) != len(original.LogsDB[c]) {
			t.Fatalf("LogsDB[%s] drift: %d vs %d", c, len(recovered.LogsDB[c]), len(original.LogsDB[c]))
		}
	}
}

func TestSnapshotFrom_Nil(t *testing.T) {
	s := SnapshotFrom(nil)
	if err := CheckAll(s); err != nil {
		t.Fatalf("nil view should produce a valid empty snapshot: %v", err)
	}
}
