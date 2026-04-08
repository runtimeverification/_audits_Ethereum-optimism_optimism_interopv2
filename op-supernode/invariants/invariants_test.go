package invariants

import (
	"errors"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// chain helpers --------------------------------------------------------------

func chainA() eth.ChainID { return eth.ChainIDFromUInt64(1) }
func chainB() eth.ChainID { return eth.ChainIDFromUInt64(2) }

func hash(b byte) common.Hash {
	var h common.Hash
	h[31] = b
	return h
}

func blockID(b byte, num uint64) eth.BlockID {
	return eth.BlockID{Hash: hash(b), Number: num}
}

func block(b, parent byte, num, time uint64) BlockWithLogs {
	return BlockWithLogs{
		Ref: BlockRef{
			ID:         blockID(b, num),
			ParentHash: hash(parent),
			Time:       time,
		},
	}
}

// -----------------------------------------------------------------------------
// I12 — initial (empty) state satisfies CheckAll vacuously.
// -----------------------------------------------------------------------------
func TestI12_EmptyStateVacuous(t *testing.T) {
	s := Snapshot{
		ActivationTS: 100,
		Chains:       []eth.ChainID{chainA(), chainB()},
		LogsDB: map[eth.ChainID][]BlockWithLogs{
			chainA(): nil,
			chainB(): nil,
		},
		DenyList: map[eth.ChainID][]DenyListEntry{
			chainA(): nil,
			chainB(): nil,
		},
	}
	if err := CheckAll(s); err != nil {
		t.Fatalf("expected empty state to satisfy all invariants, got: %v", err)
	}
}

// -----------------------------------------------------------------------------
// I2 — LogsDB linearity.
// -----------------------------------------------------------------------------
func TestI2_Linear_OK(t *testing.T) {
	s := Snapshot{
		Chains: []eth.ChainID{chainA()},
		LogsDB: map[eth.ChainID][]BlockWithLogs{
			chainA(): {
				block(1, 0, 1, 10),
				block(2, 1, 2, 11),
				block(3, 2, 3, 12),
			},
		},
		DenyList: map[eth.ChainID][]DenyListEntry{chainA(): nil},
	}
	if err := CheckI2_LogsDBLinear(s); err != nil {
		t.Fatalf("I2 should pass: %v", err)
	}
}

func TestI2_Linear_BrokenParentHash(t *testing.T) {
	s := Snapshot{
		Chains: []eth.ChainID{chainA()},
		LogsDB: map[eth.ChainID][]BlockWithLogs{
			chainA(): {
				block(1, 0, 1, 10),
				block(2, 99, 2, 11), // parent hash is 99, not 1
			},
		},
	}
	err := CheckI2_LogsDBLinear(s)
	if err == nil || !strings.Contains(err.Error(), "[I2]") {
		t.Fatalf("expected I2 failure, got: %v", err)
	}
}

func TestI2_Linear_BrokenNumber(t *testing.T) {
	s := Snapshot{
		Chains: []eth.ChainID{chainA()},
		LogsDB: map[eth.ChainID][]BlockWithLogs{
			chainA(): {
				block(1, 0, 1, 10),
				block(2, 1, 5, 11), // number jumps from 1 to 5
			},
		},
	}
	err := CheckI2_LogsDBLinear(s)
	if err == nil || !strings.Contains(err.Error(), "[I2]") {
		t.Fatalf("expected I2 failure on number mismatch, got: %v", err)
	}
}

// -----------------------------------------------------------------------------
// I4 — VerifiedDB anchor.
// -----------------------------------------------------------------------------
func TestI4_Anchor_OK(t *testing.T) {
	s := Snapshot{
		ActivationTS: 100,
		Chains:       []eth.ChainID{chainA()},
		LogsDB: map[eth.ChainID][]BlockWithLogs{
			chainA(): {block(1, 0, 1, 100)},
		},
		Verified: []VerifiedEntry{{
			Timestamp:   100,
			L1Inclusion: blockID(10, 1000),
			L2Heads:     map[eth.ChainID]eth.BlockID{chainA(): blockID(1, 1)},
		}},
	}
	if err := CheckI4_VerifiedAnchor(s); err != nil {
		t.Fatalf("I4 should pass: %v", err)
	}
}

func TestI4_Anchor_TimestampMismatch(t *testing.T) {
	s := Snapshot{
		ActivationTS: 100,
		Chains:       []eth.ChainID{chainA()},
		LogsDB: map[eth.ChainID][]BlockWithLogs{
			chainA(): {block(1, 0, 1, 99)},
		},
		Verified: []VerifiedEntry{{
			Timestamp: 99, // should be 100
			L2Heads:   map[eth.ChainID]eth.BlockID{chainA(): blockID(1, 1)},
		}},
	}
	if err := CheckI4_VerifiedAnchor(s); err == nil ||
		!strings.Contains(err.Error(), "[I4]") {
		t.Fatalf("expected I4 timestamp mismatch, got: %v", err)
	}
}

// -----------------------------------------------------------------------------
// I5 — VerifiedDB head = LogsDB tail.
// -----------------------------------------------------------------------------
func TestI5_HeadMatchesTail_OK(t *testing.T) {
	s := Snapshot{
		ActivationTS: 100,
		Chains:       []eth.ChainID{chainA()},
		LogsDB: map[eth.ChainID][]BlockWithLogs{
			chainA(): {
				block(1, 0, 1, 100),
				block(2, 1, 2, 101),
			},
		},
		Verified: []VerifiedEntry{
			{
				Timestamp: 100,
				L2Heads:   map[eth.ChainID]eth.BlockID{chainA(): blockID(1, 1)},
			},
			{
				Timestamp: 101,
				L2Heads:   map[eth.ChainID]eth.BlockID{chainA(): blockID(2, 2)},
			},
		},
	}
	if err := CheckI5_HeadMatchesTail(s); err != nil {
		t.Fatalf("I5 should pass: %v", err)
	}
}

func TestI5_HeadMatchesTail_Divergent(t *testing.T) {
	s := Snapshot{
		Chains: []eth.ChainID{chainA()},
		LogsDB: map[eth.ChainID][]BlockWithLogs{
			chainA(): {
				block(1, 0, 1, 100),
				block(2, 1, 2, 101),
			},
		},
		Verified: []VerifiedEntry{{
			Timestamp: 101,
			L2Heads:   map[eth.ChainID]eth.BlockID{chainA(): blockID(1, 1)}, // not tail
		}},
	}
	if err := CheckI5_HeadMatchesTail(s); err == nil ||
		!strings.Contains(err.Error(), "[I5]") {
		t.Fatalf("expected I5 divergence, got: %v", err)
	}
}

// -----------------------------------------------------------------------------
// I6 — monotone L2 heads.
// -----------------------------------------------------------------------------
func TestI6_Monotone_StayCase(t *testing.T) {
	// L2 produces no block at timestamp 101; C^j stays the same.
	s := Snapshot{
		Chains: []eth.ChainID{chainA()},
		LogsDB: map[eth.ChainID][]BlockWithLogs{
			chainA(): {block(1, 0, 1, 100)},
		},
		Verified: []VerifiedEntry{
			{Timestamp: 100, L2Heads: map[eth.ChainID]eth.BlockID{chainA(): blockID(1, 1)}},
			{Timestamp: 101, L2Heads: map[eth.ChainID]eth.BlockID{chainA(): blockID(1, 1)}},
		},
	}
	if err := CheckI6_MonotoneL2Heads(s); err != nil {
		t.Fatalf("I6 stay-case should pass: %v", err)
	}
}

func TestI6_Monotone_AdvanceCase(t *testing.T) {
	s := Snapshot{
		Chains: []eth.ChainID{chainA()},
		LogsDB: map[eth.ChainID][]BlockWithLogs{
			chainA(): {
				block(1, 0, 1, 100),
				block(2, 1, 2, 101),
			},
		},
		Verified: []VerifiedEntry{
			{Timestamp: 100, L2Heads: map[eth.ChainID]eth.BlockID{chainA(): blockID(1, 1)}},
			{Timestamp: 101, L2Heads: map[eth.ChainID]eth.BlockID{chainA(): blockID(2, 2)}},
		},
	}
	if err := CheckI6_MonotoneL2Heads(s); err != nil {
		t.Fatalf("I6 advance-case should pass: %v", err)
	}
}

func TestI6_Monotone_Skip(t *testing.T) {
	// L2 head advances by 2 blocks between consecutive verified entries.
	s := Snapshot{
		Chains: []eth.ChainID{chainA()},
		LogsDB: map[eth.ChainID][]BlockWithLogs{
			chainA(): {
				block(1, 0, 1, 100),
				block(2, 1, 2, 101),
				block(3, 2, 3, 102),
			},
		},
		Verified: []VerifiedEntry{
			{Timestamp: 100, L2Heads: map[eth.ChainID]eth.BlockID{chainA(): blockID(1, 1)}},
			{Timestamp: 102, L2Heads: map[eth.ChainID]eth.BlockID{chainA(): blockID(3, 3)}},
		},
	}
	if err := CheckI6_MonotoneL2Heads(s); err == nil ||
		!strings.Contains(err.Error(), "[I6]") {
		t.Fatalf("expected I6 failure on 2-block jump, got: %v", err)
	}
}

// -----------------------------------------------------------------------------
// I8 — L1 inclusion monotonicity.
// -----------------------------------------------------------------------------
func TestI8_L1Monotone_OK(t *testing.T) {
	s := Snapshot{
		Verified: []VerifiedEntry{
			{Timestamp: 100, L1Inclusion: blockID(10, 1000)},
			{Timestamp: 101, L1Inclusion: blockID(11, 1000)}, // same number OK
			{Timestamp: 102, L1Inclusion: blockID(12, 1001)},
		},
	}
	if err := CheckI8_VerifiedL1Monotone(s); err != nil {
		t.Fatalf("I8 should pass: %v", err)
	}
}

func TestI8_L1Monotone_Regress(t *testing.T) {
	s := Snapshot{
		Verified: []VerifiedEntry{
			{L1Inclusion: blockID(10, 1001)},
			{L1Inclusion: blockID(11, 1000)}, // regression
		},
	}
	if err := CheckI8_VerifiedL1Monotone(s); err == nil ||
		!strings.Contains(err.Error(), "[I8]") {
		t.Fatalf("expected I8 failure, got: %v", err)
	}
}

// -----------------------------------------------------------------------------
// I10 — LogsDB disjoint from DenyList.
// -----------------------------------------------------------------------------
func TestI10_Disjoint_OK(t *testing.T) {
	s := Snapshot{
		Chains: []eth.ChainID{chainA()},
		LogsDB: map[eth.ChainID][]BlockWithLogs{
			chainA(): {block(1, 0, 1, 100)},
		},
		DenyList: map[eth.ChainID][]DenyListEntry{
			chainA(): {{Block: blockID(99, 1), DecisionTimestamp: 100}},
		},
	}
	if err := CheckI10_LogsDBDisjointFromDenyList(s); err != nil {
		t.Fatalf("I10 should pass: %v", err)
	}
}

func TestI10_Disjoint_Violation(t *testing.T) {
	s := Snapshot{
		Chains: []eth.ChainID{chainA()},
		LogsDB: map[eth.ChainID][]BlockWithLogs{
			chainA(): {block(1, 0, 1, 100)},
		},
		DenyList: map[eth.ChainID][]DenyListEntry{
			chainA(): {{Block: blockID(1, 1), DecisionTimestamp: 100}}, // collides
		},
	}
	if err := CheckI10_LogsDBDisjointFromDenyList(s); err == nil ||
		!strings.Contains(err.Error(), "[I10]") {
		t.Fatalf("expected I10 failure, got: %v", err)
	}
}

// -----------------------------------------------------------------------------
// I11 — DenyList bounded by t + 1.
// -----------------------------------------------------------------------------
func TestI11_Bounded_OK(t *testing.T) {
	s := Snapshot{
		ActivationTS: 100,
		Verified: []VerifiedEntry{
			{Timestamp: 100},
			{Timestamp: 101},
		},
		DenyList: map[eth.ChainID][]DenyListEntry{
			chainA(): {
				{Block: blockID(1, 1), DecisionTimestamp: 101},
				{Block: blockID(2, 1), DecisionTimestamp: 102}, // t + 1 allowed
			},
		},
	}
	if err := CheckI11_DenyListBounded(s); err != nil {
		t.Fatalf("I11 should pass: %v", err)
	}
}

func TestI11_Bounded_Exceeds(t *testing.T) {
	s := Snapshot{
		Verified: []VerifiedEntry{{Timestamp: 101}},
		DenyList: map[eth.ChainID][]DenyListEntry{
			chainA(): {{Block: blockID(1, 1), DecisionTimestamp: 103}},
		},
	}
	if err := CheckI11_DenyListBounded(s); err == nil ||
		!strings.Contains(err.Error(), "[I11]") {
		t.Fatalf("expected I11 failure, got: %v", err)
	}
}

// -----------------------------------------------------------------------------
// CheckAll composition: multiple independent failures are joined.
// -----------------------------------------------------------------------------
func TestCheckAll_JoinsFailures(t *testing.T) {
	s := Snapshot{
		Chains: []eth.ChainID{chainA()},
		LogsDB: map[eth.ChainID][]BlockWithLogs{
			chainA(): {
				block(1, 0, 1, 100),
				block(2, 99, 2, 101), // I2 failure (parent hash wrong)
			},
		},
		DenyList: map[eth.ChainID][]DenyListEntry{
			chainA(): {{Block: blockID(1, 1), DecisionTimestamp: 100}}, // I10
		},
	}
	err := CheckAll(s)
	if err == nil {
		t.Fatal("expected compound failure")
	}
	msg := err.Error()
	if !strings.Contains(msg, "[I2]") {
		t.Errorf("expected I2 in joined error: %v", err)
	}
	if !strings.Contains(msg, "[I10]") {
		t.Errorf("expected I10 in joined error: %v", err)
	}
	// Ensure the returned error is the joined form (errors.Join produces
	// an error that wraps each sub-error; errors.Is should find each).
	for _, sub := range []string{"I2", "I10"} {
		if !containsID(err, sub) {
			t.Errorf("errors.Unwrap chain missing ID %q", sub)
		}
	}
}

func containsID(err error, id string) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*Error); ok {
		return e.ID == id
	}
	// errors.Join returns an error whose Unwrap() []error enumerates the children.
	type multi interface{ Unwrap() []error }
	if m, ok := err.(multi); ok {
		for _, c := range m.Unwrap() {
			if containsID(c, id) {
				return true
			}
		}
	}
	var e *Error
	if errors.As(err, &e) && e.ID == id {
		return true
	}
	return false
}
