package invariants

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Error is the structured failure type returned by every CheckI* predicate.
// The SPEC ID (e.g. "I2") is the stable identifier that cross-references
// op-supernode/invariants/SPEC.md and dafny-models/SupernodeState.dfy.
type Error struct {
	ID      string // SPEC ID, e.g. "I2", "I10", "T3"
	Chain   *eth.ChainID
	Index   *int
	Message string
}

func (e *Error) Error() string {
	prefix := "[" + e.ID + "] "
	if e.Chain != nil {
		prefix += fmt.Sprintf("chain=%s ", e.Chain.String())
	}
	if e.Index != nil {
		prefix += fmt.Sprintf("idx=%d ", *e.Index)
	}
	return prefix + e.Message
}

func newErr(id, msg string) *Error {
	return &Error{ID: id, Message: msg}
}

func newChainErr(id string, chain eth.ChainID, msg string) *Error {
	return &Error{ID: id, Chain: &chain, Message: msg}
}

func newIndexedErr(id string, chain eth.ChainID, idx int, msg string) *Error {
	return &Error{ID: id, Chain: &chain, Index: &idx, Message: msg}
}

// -----------------------------------------------------------------------------
// CheckAll — the top-level predicate. Mirrors Dafny's AllInvariants.
// -----------------------------------------------------------------------------

// CheckAll runs every applicable invariant on the snapshot and returns a
// joined error if any fail. A nil return means every structural invariant
// held. Environmental invariants (A1..A5) and hybrid-clause halves that
// require live L1/L2 queries are NOT checked here — see CheckAllWithEnv.
func CheckAll(s Snapshot) error {
	checks := []func(Snapshot) error{
		CheckI2_LogsDBLinear,
		CheckI4_VerifiedAnchor,
		CheckI5_HeadMatchesTail,
		CheckI6_MonotoneL2Heads,
		CheckI7_HighestBlockLeqTS,
		CheckI8_VerifiedL1Monotone,
		CheckI10_LogsDBDisjointFromDenyList,
		CheckI11_DenyListBounded,
	}
	var errs []error
	for _, c := range checks {
		if err := c(s); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// -----------------------------------------------------------------------------
// I2 — LogsDB linearity.
// SPEC.md I2 / overview.md line 65: B^j_i is the parent of B^j_{i+1}.
// -----------------------------------------------------------------------------
func CheckI2_LogsDBLinear(s Snapshot) error {
	var errs []error
	for chain, logs := range s.LogsDB {
		for i := 0; i+1 < len(logs); i++ {
			cur := logs[i]
			nxt := logs[i+1]
			if nxt.Ref.ParentHash != cur.Ref.ID.Hash {
				errs = append(errs, newIndexedErr("I2", chain, i,
					fmt.Sprintf("LogsDB[%s][%d+1].ParentHash=%s != LogsDB[%s][%d].Hash=%s",
						chain, i, nxt.Ref.ParentHash, chain, i, cur.Ref.ID.Hash)))
				continue
			}
			if nxt.Ref.ID.Number != cur.Ref.ID.Number+1 {
				errs = append(errs, newIndexedErr("I2", chain, i,
					fmt.Sprintf("LogsDB[%s][%d+1].Number=%d != LogsDB[%s][%d].Number+1=%d",
						chain, i, nxt.Ref.ID.Number, chain, i, cur.Ref.ID.Number+1)))
			}
		}
	}
	return errors.Join(errs...)
}

// -----------------------------------------------------------------------------
// I4 — VerifiedDB anchor.
// SPEC.md I4 / overview.md line 67: C^j_{t_0} = B^j_0 for all j.
// -----------------------------------------------------------------------------
func CheckI4_VerifiedAnchor(s Snapshot) error {
	if len(s.Verified) == 0 {
		return nil // vacuous
	}
	first := s.Verified[0]
	if first.Timestamp != s.ActivationTS {
		return newErr("I4", fmt.Sprintf(
			"Verified[0].Timestamp=%d != ActivationTS=%d",
			first.Timestamp, s.ActivationTS))
	}
	var errs []error
	for _, chain := range s.Chains {
		logs, ok := s.LogsDB[chain]
		if !ok || len(logs) == 0 {
			errs = append(errs, newChainErr("I4", chain,
				"LogsDB missing or empty while Verified is non-empty"))
			continue
		}
		head, ok := first.L2Heads[chain]
		if !ok {
			errs = append(errs, newChainErr("I4", chain,
				"Verified[0] missing L2Head for this chain"))
			continue
		}
		if head != logs[0].Ref.ID {
			errs = append(errs, newChainErr("I4", chain, fmt.Sprintf(
				"C^j_{t_0}=%s != B^j_0=%s", head, logs[0].Ref.ID)))
		}
	}
	return errors.Join(errs...)
}

// -----------------------------------------------------------------------------
// I5 — VerifiedDB head = LogsDB tail.
// SPEC.md I5 / overview.md line 68: C^j_t = B^j_{n_j} for all j.
// -----------------------------------------------------------------------------
func CheckI5_HeadMatchesTail(s Snapshot) error {
	if len(s.Verified) == 0 {
		return nil // vacuous
	}
	last := s.Verified[len(s.Verified)-1]
	var errs []error
	for _, chain := range s.Chains {
		logs, ok := s.LogsDB[chain]
		if !ok || len(logs) == 0 {
			errs = append(errs, newChainErr("I5", chain,
				"LogsDB missing or empty while Verified is non-empty"))
			continue
		}
		head, ok := last.L2Heads[chain]
		if !ok {
			errs = append(errs, newChainErr("I5", chain,
				"Verified[last] missing L2Head for this chain"))
			continue
		}
		tail := logs[len(logs)-1].Ref.ID
		if head != tail {
			errs = append(errs, newChainErr("I5", chain, fmt.Sprintf(
				"C^j_t=%s != B^j_{n_j}=%s", head, tail)))
		}
	}
	return errors.Join(errs...)
}

// -----------------------------------------------------------------------------
// I6 — Monotone L2 heads across consecutive verified entries.
// SPEC.md I6 / overview.md line 69: C^j_i is C^j_{i+1} or its parent.
// -----------------------------------------------------------------------------
func CheckI6_MonotoneL2Heads(s Snapshot) error {
	var errs []error
	for i := 0; i+1 < len(s.Verified); i++ {
		prev := s.Verified[i]
		nxt := s.Verified[i+1]
		for _, chain := range s.Chains {
			prevHead, ok1 := prev.L2Heads[chain]
			nxtHead, ok2 := nxt.L2Heads[chain]
			if !ok1 || !ok2 {
				errs = append(errs, newIndexedErr("I6", chain, i,
					"missing L2Head in one of the adjacent Verified entries"))
				continue
			}
			if prevHead == nxtHead {
				continue // "stay" case
			}
			logs := s.LogsDB[chain]
			nxtBlock, found := LogsDBLookup(logs, nxtHead)
			if !found {
				errs = append(errs, newIndexedErr("I6", chain, i, fmt.Sprintf(
					"C^j_{%d+1}=%s not found in LogsDB", i, nxtHead)))
				continue
			}
			if nxtBlock.Ref.ParentHash != prevHead.Hash {
				errs = append(errs, newIndexedErr("I6", chain, i, fmt.Sprintf(
					"C^j_{%d+1}.ParentHash=%s != C^j_%d.Hash=%s",
					i, nxtBlock.Ref.ParentHash, i, prevHead.Hash)))
				continue
			}
			if nxtBlock.Ref.ID.Number != prevHead.Number+1 {
				errs = append(errs, newIndexedErr("I6", chain, i, fmt.Sprintf(
					"C^j_{%d+1}.Number=%d != C^j_%d.Number+1=%d",
					i, nxtBlock.Ref.ID.Number, i, prevHead.Number+1)))
			}
		}
	}
	return errors.Join(errs...)
}

// -----------------------------------------------------------------------------
// I7 — LogsDB-internal half: C^j_i is the highest block with timestamp <= i.
//
// The "all children have timestamp > i" clause requires live L2 inspection
// and is ENVIRONMENTAL (SPEC.md §4). This check verifies only the LogsDB
// half: C^j_i is in LogsDB[j], has Time <= i, and either is the last entry
// or the next entry has Time > i.
// -----------------------------------------------------------------------------
func CheckI7_HighestBlockLeqTS(s Snapshot) error {
	var errs []error
	for vi, v := range s.Verified {
		for _, chain := range s.Chains {
			head, ok := v.L2Heads[chain]
			if !ok {
				errs = append(errs, newIndexedErr("I7", chain, vi,
					"Verified entry missing L2Head for this chain"))
				continue
			}
			logs := s.LogsDB[chain]
			var headIdx = -1
			for i, b := range logs {
				if b.Ref.ID == head {
					headIdx = i
					break
				}
			}
			if headIdx == -1 {
				errs = append(errs, newIndexedErr("I7", chain, vi, fmt.Sprintf(
					"C^j_%d=%s not found in LogsDB", vi, head)))
				continue
			}
			if logs[headIdx].Ref.Time > v.Timestamp {
				errs = append(errs, newIndexedErr("I7", chain, vi, fmt.Sprintf(
					"C^j_%d.Time=%d > VerifiedTimestamp=%d",
					vi, logs[headIdx].Ref.Time, v.Timestamp)))
				continue
			}
			// If there's a next block in LogsDB, its Time must strictly
			// exceed the verified timestamp (otherwise a higher LogsDB block
			// with Time <= v.Timestamp exists).
			if headIdx+1 < len(logs) && logs[headIdx+1].Ref.Time <= v.Timestamp {
				errs = append(errs, newIndexedErr("I7", chain, vi, fmt.Sprintf(
					"LogsDB entry after C^j_%d has Time=%d <= VerifiedTimestamp=%d",
					vi, logs[headIdx+1].Ref.Time, v.Timestamp)))
			}
		}
	}
	return errors.Join(errs...)
}

// -----------------------------------------------------------------------------
// I8 — L1 inclusion number non-decreasing across consecutive verified entries.
//
// SPEC.md I8 / overview.md line 71: C_{t_0}, ..., C_t are on the same
// linear L1 chain. Ancestry cannot be checked from the snapshot — this is
// the necessary-condition half (number monotonicity). The ancestry half is
// environmental (A2 / SPEC.md §4) and verified by CheckAllWithEnv.
// -----------------------------------------------------------------------------
func CheckI8_VerifiedL1Monotone(s Snapshot) error {
	var errs []error
	for i := 0; i+1 < len(s.Verified); i++ {
		if s.Verified[i].L1Inclusion.Number > s.Verified[i+1].L1Inclusion.Number {
			errs = append(errs, newErr("I8", fmt.Sprintf(
				"Verified[%d].L1Inclusion.Number=%d > Verified[%d].L1Inclusion.Number=%d",
				i, s.Verified[i].L1Inclusion.Number,
				i+1, s.Verified[i+1].L1Inclusion.Number)))
		}
	}
	return errors.Join(errs...)
}

// -----------------------------------------------------------------------------
// I10 — LogsDB disjoint from DenyList.
// SPEC.md I10 / overview.md line 73: B^j_i not in D_j for any j and i.
// -----------------------------------------------------------------------------
func CheckI10_LogsDBDisjointFromDenyList(s Snapshot) error {
	var errs []error
	for chain, logs := range s.LogsDB {
		denied := s.DenyList[chain]
		if len(denied) == 0 {
			continue
		}
		// Build a set for O(1) lookup.
		deniedSet := make(map[eth.BlockID]struct{}, len(denied))
		for _, d := range denied {
			deniedSet[d.Block] = struct{}{}
		}
		for i, b := range logs {
			if _, bad := deniedSet[b.Ref.ID]; bad {
				errs = append(errs, newIndexedErr("I10", chain, i, fmt.Sprintf(
					"LogsDB[%s][%d].ID=%s appears in DenyList", chain, i, b.Ref.ID)))
			}
		}
	}
	return errors.Join(errs...)
}

// -----------------------------------------------------------------------------
// I11 — DenyList entries bounded by t + 1.
// SPEC.md I11 / overview.md line 74: DecisionTimestamp <= t + 1 (the next
// timestamp being cross-validated is allowed to have speculative entries).
// -----------------------------------------------------------------------------
func CheckI11_DenyListBounded(s Snapshot) error {
	last, hasLast := s.LastVerifiedTimestamp()
	var bound uint64
	if hasLast {
		if last == ^uint64(0) {
			return newErr("I11",
				"VerifiedDB head is MAX_UINT64, t+1 would overflow")
		}
		bound = last + 1
	} else {
		// No cross-validation yet. DenyList entries for the activation
		// timestamp are allowed from speculative rounds.
		if s.ActivationTS == ^uint64(0) {
			return newErr("I11",
				"ActivationTS is MAX_UINT64, t_0+1 would overflow")
		}
		bound = s.ActivationTS + 1
	}
	var errs []error
	for chain, denied := range s.DenyList {
		for i, d := range denied {
			if d.DecisionTimestamp > bound {
				errs = append(errs, newIndexedErr("I11", chain, i, fmt.Sprintf(
					"DecisionTimestamp=%d > bound=%d (last verified t=%d, hasLast=%v)",
					d.DecisionTimestamp, bound, last, hasLast)))
			}
		}
	}
	return errors.Join(errs...)
}
