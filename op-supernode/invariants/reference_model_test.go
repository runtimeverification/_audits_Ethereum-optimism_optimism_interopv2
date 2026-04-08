package invariants

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// -----------------------------------------------------------------------------
// End-to-end scenario: start empty, perform a genuine sequence of T5 / T4 /
// T3 transitions, and assert CheckAll holds at every step. This exercises
// the reference model and the invariant predicates jointly.
// -----------------------------------------------------------------------------

func TestReferenceModel_AdvanceFromInitial(t *testing.T) {
	s := NewInitialSnapshot(100, []eth.ChainID{chainA()})
	if err := CheckAll(s); err != nil {
		t.Fatalf("initial state must satisfy invariants: %v", err)
	}

	// Advance at t_0 = 100. Supply genesis block.
	s, err := ApplyAdvance(s,
		blockID(10, 1000),
		map[eth.ChainID]BlockWithLogs{
			chainA(): block(1, 0, 1, 100),
		})
	if err != nil {
		t.Fatalf("advance failed: %v", err)
	}
	if err := CheckAll(s); err != nil {
		t.Fatalf("post-advance invariants failed: %v", err)
	}
	if len(s.Verified) != 1 || s.Verified[0].Timestamp != 100 {
		t.Fatalf("expected one verified entry at t=100, got %+v", s.Verified)
	}
	if len(s.LogsDB[chainA()]) != 1 {
		t.Fatalf("expected LogsDB length 1, got %d", len(s.LogsDB[chainA()]))
	}
}

func TestReferenceModel_AdvanceManyTimestamps(t *testing.T) {
	s := NewInitialSnapshot(100, []eth.ChainID{chainA()})
	s, err := ApplyAdvance(s, blockID(10, 1000), map[eth.ChainID]BlockWithLogs{
		chainA(): block(1, 0, 1, 100),
	})
	mustOK(t, err, s)

	// Advance five more timestamps, one new block each.
	for i := uint64(101); i <= 105; i++ {
		prevBlock := byte(i - 100)
		newBlock := byte(i - 99)
		s, err = ApplyAdvance(s,
			blockID(byte(i-90), 1000+(i-100)),
			map[eth.ChainID]BlockWithLogs{
				chainA(): block(newBlock, prevBlock, i-99, i),
			})
		mustOK(t, err, s)
	}

	if len(s.LogsDB[chainA()]) != 6 {
		t.Fatalf("expected LogsDB length 6, got %d", len(s.LogsDB[chainA()]))
	}
	if len(s.Verified) != 6 {
		t.Fatalf("expected 6 verified entries, got %d", len(s.Verified))
	}
}

func TestReferenceModel_AdvanceStayCase(t *testing.T) {
	// Chain produces no new block at the next timestamp; head should stay.
	s := NewInitialSnapshot(100, []eth.ChainID{chainA()})
	s, err := ApplyAdvance(s, blockID(10, 1000), map[eth.ChainID]BlockWithLogs{
		chainA(): block(1, 0, 1, 100),
	})
	mustOK(t, err, s)

	// Advance with empty newHeads => stay case.
	s, err = ApplyAdvance(s, blockID(11, 1001), map[eth.ChainID]BlockWithLogs{})
	mustOK(t, err, s)

	if len(s.LogsDB[chainA()]) != 1 {
		t.Fatalf("stay-case should not extend LogsDB, got len %d", len(s.LogsDB[chainA()]))
	}
	if s.Verified[1].L2Heads[chainA()] != blockID(1, 1) {
		t.Fatalf("stay-case should keep head, got %v", s.Verified[1].L2Heads[chainA()])
	}
}

func TestReferenceModel_Invalidate(t *testing.T) {
	// After one successful advance, invalidate a block at the next timestamp.
	s := NewInitialSnapshot(100, []eth.ChainID{chainA()})
	s, err := ApplyAdvance(s, blockID(10, 1000), map[eth.ChainID]BlockWithLogs{
		chainA(): block(1, 0, 1, 100),
	})
	mustOK(t, err, s)

	s, err = ApplyInvalidate(s, map[eth.ChainID]eth.BlockID{
		chainA(): blockID(2, 2),
	})
	mustOK(t, err, s)

	if len(s.DenyList[chainA()]) != 1 {
		t.Fatalf("expected one deny entry, got %d", len(s.DenyList[chainA()]))
	}
	if s.DenyList[chainA()][0].DecisionTimestamp != 101 {
		t.Fatalf("expected DecisionTS=101, got %d", s.DenyList[chainA()][0].DecisionTimestamp)
	}
	// Verified should not have grown.
	if len(s.Verified) != 1 {
		t.Fatalf("invalidate should not extend Verified, got len %d", len(s.Verified))
	}
}

func TestReferenceModel_Rewind(t *testing.T) {
	// Build: advance twice, then rewind once. After rewind we should be
	// back to the one-entry state.
	s := NewInitialSnapshot(100, []eth.ChainID{chainA()})
	s, err := ApplyAdvance(s, blockID(10, 1000), map[eth.ChainID]BlockWithLogs{
		chainA(): block(1, 0, 1, 100),
	})
	mustOK(t, err, s)
	s, err = ApplyAdvance(s, blockID(11, 1001), map[eth.ChainID]BlockWithLogs{
		chainA(): block(2, 1, 2, 101),
	})
	mustOK(t, err, s)

	if len(s.LogsDB[chainA()]) != 2 {
		t.Fatalf("expected LogsDB len 2 pre-rewind, got %d", len(s.LogsDB[chainA()]))
	}

	s, err = ApplyRewind(s)
	mustOK(t, err, s)

	if len(s.Verified) != 1 {
		t.Fatalf("expected 1 verified entry post-rewind, got %d", len(s.Verified))
	}
	if len(s.LogsDB[chainA()]) != 1 {
		t.Fatalf("expected LogsDB len 1 post-rewind, got %d", len(s.LogsDB[chainA()]))
	}
}

func TestReferenceModel_RewindAfterDeny(t *testing.T) {
	// Advance, invalidate (deny at t+1), rewind: the deny entry should be
	// pruned because its DecisionTimestamp >= t being rewound.
	s := NewInitialSnapshot(100, []eth.ChainID{chainA()})
	s, err := ApplyAdvance(s, blockID(10, 1000), map[eth.ChainID]BlockWithLogs{
		chainA(): block(1, 0, 1, 100),
	})
	mustOK(t, err, s)
	s, err = ApplyAdvance(s, blockID(11, 1001), map[eth.ChainID]BlockWithLogs{
		chainA(): block(2, 1, 2, 101),
	})
	mustOK(t, err, s)
	// At t = 101, invalidate a speculative block for t+1 = 102.
	s, err = ApplyInvalidate(s, map[eth.ChainID]eth.BlockID{
		chainA(): blockID(99, 3),
	})
	mustOK(t, err, s)
	if len(s.DenyList[chainA()]) != 1 {
		t.Fatalf("pre-rewind deny list len %d, want 1", len(s.DenyList[chainA()]))
	}

	// Now rewind t = 101. DecisionTS = 102 >= 101 => pruned.
	s, err = ApplyRewind(s)
	mustOK(t, err, s)

	if len(s.DenyList[chainA()]) != 0 {
		t.Fatalf("post-rewind deny list len %d, want 0", len(s.DenyList[chainA()]))
	}
}

// -----------------------------------------------------------------------------
// Error conditions
// -----------------------------------------------------------------------------

func TestReferenceModel_RewindEmpty(t *testing.T) {
	s := NewInitialSnapshot(100, []eth.ChainID{chainA()})
	if _, err := ApplyRewind(s); err == nil {
		t.Fatal("expected rewind of empty Verified to error")
	}
}

func TestReferenceModel_InvalidateNoVerified(t *testing.T) {
	s := NewInitialSnapshot(100, []eth.ChainID{chainA()})
	if _, err := ApplyInvalidate(s, map[eth.ChainID]eth.BlockID{
		chainA(): blockID(1, 1),
	}); err == nil {
		t.Fatal("expected invalidate on empty Verified to error")
	}
}

// -----------------------------------------------------------------------------
// Helper
// -----------------------------------------------------------------------------

func mustOK(t *testing.T, err error, s Snapshot) {
	t.Helper()
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if inv := CheckAll(s); inv != nil {
		t.Fatalf("invariants violated after apply: %v", inv)
	}
}
