package invariants

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// fakeEnv is a stub Env implementation that answers queries from two
// preloaded maps. It's the default fixture for I1/I8/I9 tests.
type fakeEnv struct {
	logsAccepted map[eth.BlockID]bool
	derive       map[eth.BlockID]eth.BlockID
	ancestor     map[[2]eth.BlockID]bool
	higherL2     map[eth.ChainID]map[uint64]bool
}

func newFakeEnv() *fakeEnv {
	return &fakeEnv{
		logsAccepted: map[eth.BlockID]bool{},
		derive:       map[eth.BlockID]eth.BlockID{},
		ancestor:     map[[2]eth.BlockID]bool{},
		higherL2:     map[eth.ChainID]map[uint64]bool{},
	}
}

func (f *fakeEnv) LogsBelongToBlock(ref BlockRef, _ []ExecutingMessage) error {
	if f.logsAccepted[ref.ID] {
		return nil
	}
	return fmt.Errorf("logs for %s not approved by fake env", ref.ID)
}

func (f *fakeEnv) DeriveL1(l2 eth.BlockID) (eth.BlockID, error) {
	l1, ok := f.derive[l2]
	if !ok {
		return eth.BlockID{}, fmt.Errorf("no derive mapping for %s", l2)
	}
	return l1, nil
}

func (f *fakeEnv) IsL1Ancestor(a, d eth.BlockID) (bool, error) {
	return f.ancestor[[2]eth.BlockID{a, d}], nil
}

func (f *fakeEnv) HasHigherL2BlockAtOrBelow(c eth.ChainID, _ eth.BlockID, ts uint64) (bool, error) {
	if m, ok := f.higherL2[c]; ok {
		return m[ts], nil
	}
	return false, nil
}

// -----------------------------------------------------------------------------
// I1 — logs-match-block env check.
// -----------------------------------------------------------------------------

func TestCheckI1_OK(t *testing.T) {
	s := Snapshot{
		Chains: []eth.ChainID{chainA()},
		LogsDB: map[eth.ChainID][]BlockWithLogs{
			chainA(): {block(1, 0, 1, 100)},
		},
	}
	env := newFakeEnv()
	env.logsAccepted[blockID(1, 1)] = true
	if err := CheckI1_LogsBelongToBlocks(s, env); err != nil {
		t.Fatalf("I1 should pass: %v", err)
	}
}

func TestCheckI1_Reject(t *testing.T) {
	s := Snapshot{
		Chains: []eth.ChainID{chainA()},
		LogsDB: map[eth.ChainID][]BlockWithLogs{
			chainA(): {block(1, 0, 1, 100)},
		},
	}
	env := newFakeEnv() // empty — rejects every block
	err := CheckI1_LogsBelongToBlocks(s, env)
	if err == nil || !strings.Contains(err.Error(), "[I1]") {
		t.Fatalf("expected I1 failure, got: %v", err)
	}
}

// -----------------------------------------------------------------------------
// I3 — executing-message validity (state-local + cycle detection).
// -----------------------------------------------------------------------------

func TestCheckI3_NoMessages_OK(t *testing.T) {
	s := Snapshot{
		Chains: []eth.ChainID{chainA()},
		LogsDB: map[eth.ChainID][]BlockWithLogs{
			chainA(): {block(1, 0, 1, 100)},
		},
	}
	if err := CheckI3_ExecutingMessageValidity(s, nil); err != nil {
		t.Fatalf("I3 should pass with no messages: %v", err)
	}
}

func TestCheckI3_MissingInitiator(t *testing.T) {
	a := BlockWithLogs{
		Ref: BlockRef{ID: blockID(1, 1), Time: 100},
		ExecMsgs: []ExecutingMessage{{
			ChainID:   chainB(),
			BlockNum:  999, // not in chainB's LogsDB
			LogIdx:    0,
			Timestamp: 100,
		}},
	}
	s := Snapshot{
		Chains: []eth.ChainID{chainA(), chainB()},
		LogsDB: map[eth.ChainID][]BlockWithLogs{
			chainA(): {a},
			chainB(): {},
		},
	}
	err := CheckI3_ExecutingMessageValidity(s, nil)
	if err == nil || !strings.Contains(err.Error(), "[I3]") {
		t.Fatalf("expected I3 failure, got: %v", err)
	}
}

func TestCheckI3_Cycle(t *testing.T) {
	// Two chains, each with one block, and each references the other's
	// block at the same timestamp — a 2-cycle.
	a := BlockWithLogs{
		Ref: BlockRef{ID: blockID(1, 1), Time: 100},
		ExecMsgs: []ExecutingMessage{{
			ChainID: chainB(), BlockNum: 1, LogIdx: 0, Timestamp: 100,
		}},
	}
	b := BlockWithLogs{
		Ref: BlockRef{ID: blockID(2, 1), Time: 100},
		ExecMsgs: []ExecutingMessage{{
			ChainID: chainA(), BlockNum: 1, LogIdx: 0, Timestamp: 100,
		}},
	}
	s := Snapshot{
		Chains: []eth.ChainID{chainA(), chainB()},
		LogsDB: map[eth.ChainID][]BlockWithLogs{
			chainA(): {a},
			chainB(): {b},
		},
	}
	err := CheckI3_ExecutingMessageValidity(s, nil)
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected cycle detection, got: %v", err)
	}
}

// -----------------------------------------------------------------------------
// I8 — L1 ancestry.
// -----------------------------------------------------------------------------

func TestCheckI8Linear_OK(t *testing.T) {
	s := Snapshot{
		Verified: []VerifiedEntry{
			{L1Inclusion: blockID(10, 1000)},
			{L1Inclusion: blockID(11, 1001)},
		},
	}
	env := newFakeEnv()
	env.ancestor[[2]eth.BlockID{blockID(10, 1000), blockID(11, 1001)}] = true
	if err := CheckI8_VerifiedL1Linear(s, env); err != nil {
		t.Fatalf("I8 should pass: %v", err)
	}
}

func TestCheckI8Linear_NotAncestor(t *testing.T) {
	s := Snapshot{
		Verified: []VerifiedEntry{
			{L1Inclusion: blockID(10, 1000)},
			{L1Inclusion: blockID(11, 1001)},
		},
	}
	env := newFakeEnv() // ancestor map is empty => returns false
	err := CheckI8_VerifiedL1Linear(s, env)
	if err == nil || !strings.Contains(err.Error(), "[I8]") {
		t.Fatalf("expected I8 ancestor failure, got: %v", err)
	}
}

// -----------------------------------------------------------------------------
// I9 — minimal L1 cover.
// -----------------------------------------------------------------------------

func TestCheckI9_Cover_OK(t *testing.T) {
	s := Snapshot{
		Chains: []eth.ChainID{chainA(), chainB()},
		Verified: []VerifiedEntry{{
			Timestamp:   100,
			L1Inclusion: blockID(11, 1001),
			L2Heads: map[eth.ChainID]eth.BlockID{
				chainA(): blockID(1, 1),
				chainB(): blockID(2, 2),
			},
		}},
	}
	env := newFakeEnv()
	env.derive[blockID(1, 1)] = blockID(10, 1000)
	env.derive[blockID(2, 2)] = blockID(11, 1001) // higher number; becomes the max
	// I9 now requires every per-chain DeriveL1 to be an ancestor (or
	// equal to) the recorded L1Inclusion. blockID(10,1000) must be an
	// ancestor of blockID(11,1001).
	env.ancestor[[2]eth.BlockID{blockID(10, 1000), blockID(11, 1001)}] = true
	if err := CheckI9_MinimalL1Cover(s, env); err != nil {
		t.Fatalf("I9 should pass: %v", err)
	}
}

func TestCheckI9_Cover_WrongMax(t *testing.T) {
	s := Snapshot{
		Chains: []eth.ChainID{chainA(), chainB()},
		Verified: []VerifiedEntry{{
			L1Inclusion: blockID(10, 1000), // claims 1000
			L2Heads: map[eth.ChainID]eth.BlockID{
				chainA(): blockID(1, 1),
				chainB(): blockID(2, 2),
			},
		}},
	}
	env := newFakeEnv()
	env.derive[blockID(1, 1)] = blockID(10, 1000)
	env.derive[blockID(2, 2)] = blockID(11, 1001) // but chain B derives from 1001
	err := CheckI9_MinimalL1Cover(s, env)
	if err == nil || !strings.Contains(err.Error(), "[I9]") {
		t.Fatalf("expected I9 max-mismatch, got: %v", err)
	}
}

// TestCheckI9_DifferentBlockSameNumber regression-tests the strengthened
// I9 check: a recorded L1Inclusion that has the same .Number as the max
// DeriveL1 but a different hash (post-reorg) must NOT pass I9.
func TestCheckI9_DifferentBlockSameNumber(t *testing.T) {
	s := Snapshot{
		Chains: []eth.ChainID{chainA()},
		Verified: []VerifiedEntry{{
			Timestamp:   100,
			L1Inclusion: blockID(0xAA, 1001), // hash-byte 0xAA at height 1001
			L2Heads: map[eth.ChainID]eth.BlockID{
				chainA(): blockID(1, 1),
			},
		}},
	}
	env := newFakeEnv()
	// DeriveL1 returns a sibling at the same height with a different hash.
	env.derive[blockID(1, 1)] = blockID(0xBB, 1001)
	err := CheckI9_MinimalL1Cover(s, env)
	if err == nil || !strings.Contains(err.Error(), "[I9]") {
		t.Fatalf("expected I9 hash-mismatch failure, got: %v", err)
	}
}

// TestCheckI9_DeriveNotAncestor exercises the new ancestry clause: a
// per-chain DeriveL1 that is not an ancestor of L1Inclusion must fail.
func TestCheckI9_DeriveNotAncestor(t *testing.T) {
	s := Snapshot{
		Chains: []eth.ChainID{chainA(), chainB()},
		Verified: []VerifiedEntry{{
			L1Inclusion: blockID(11, 1001),
			L2Heads: map[eth.ChainID]eth.BlockID{
				chainA(): blockID(1, 1),
				chainB(): blockID(2, 2),
			},
		}},
	}
	env := newFakeEnv()
	env.derive[blockID(1, 1)] = blockID(10, 1000)
	env.derive[blockID(2, 2)] = blockID(11, 1001)
	// blockID(10,1000) is NOT marked as an ancestor of blockID(11,1001).
	err := CheckI9_MinimalL1Cover(s, env)
	if err == nil || !strings.Contains(err.Error(), "ancestor") {
		t.Fatalf("expected ancestry failure, got: %v", err)
	}
}

// TestCheckI7_NoHigherL2Block_OK exercises the new env clause for I7
// when the oracle says no higher block exists.
func TestCheckI7_NoHigherL2Block_OK(t *testing.T) {
	s := Snapshot{
		Chains: []eth.ChainID{chainA()},
		Verified: []VerifiedEntry{{
			Timestamp: 100,
			L2Heads:   map[eth.ChainID]eth.BlockID{chainA(): blockID(1, 1)},
		}},
	}
	env := newFakeEnv() // higherL2 map empty -> false
	if err := CheckI7_NoHigherL2Block(s, env); err != nil {
		t.Fatalf("I7 env clause should pass: %v", err)
	}
}

// TestCheckI7_NoHigherL2Block_Fail exercises the failure path: oracle
// reports a higher live L2 block that should have been imported.
func TestCheckI7_NoHigherL2Block_Fail(t *testing.T) {
	s := Snapshot{
		Chains: []eth.ChainID{chainA()},
		Verified: []VerifiedEntry{{
			Timestamp: 100,
			L2Heads:   map[eth.ChainID]eth.BlockID{chainA(): blockID(1, 1)},
		}},
	}
	env := newFakeEnv()
	env.higherL2[chainA()] = map[uint64]bool{100: true}
	err := CheckI7_NoHigherL2Block(s, env)
	if err == nil || !strings.Contains(err.Error(), "[I7]") {
		t.Fatalf("expected I7 env failure, got: %v", err)
	}
}

// -----------------------------------------------------------------------------
// CheckAllWithEnv composition.
// -----------------------------------------------------------------------------

func TestCheckAllWithEnv_NilEnv(t *testing.T) {
	s := NewInitialSnapshot(100, []eth.ChainID{chainA()})
	if err := CheckAllWithEnv(s, nil); err != nil {
		t.Fatalf("initial state with nil env should pass: %v", err)
	}
}

func TestCheckAllWithEnv_CompositeFailure(t *testing.T) {
	// Build a state that violates I2 AND I1 simultaneously.
	s := Snapshot{
		Chains: []eth.ChainID{chainA()},
		LogsDB: map[eth.ChainID][]BlockWithLogs{
			chainA(): {
				block(1, 0, 1, 100),
				block(2, 99, 2, 101), // I2 parent-hash violation
			},
		},
	}
	env := newFakeEnv() // rejects everything => I1 violation
	err := CheckAllWithEnv(s, env)
	if err == nil {
		t.Fatal("expected compound failure")
	}
	var ie *Error
	foundI1, foundI2 := false, false
	type multi interface{ Unwrap() []error }
	var walk func(error)
	walk = func(e error) {
		if e == nil {
			return
		}
		if errors.As(e, &ie) {
			switch ie.ID {
			case "I1":
				foundI1 = true
			case "I2":
				foundI2 = true
			}
			ie = nil
		}
		if m, ok := e.(multi); ok {
			for _, c := range m.Unwrap() {
				walk(c)
			}
		}
	}
	walk(err)
	if !foundI1 || !foundI2 {
		t.Fatalf("expected both I1 and I2 in joined error, got: %v", err)
	}
}
