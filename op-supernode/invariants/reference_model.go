package invariants

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Reference model — pure Go implementation of the SPEC.md §3 transitions
// T1..T5. This is the "trusted" state machine that fuzz tests and traces
// diff against. Every method is a pure function of its inputs and must
// preserve AllInvariants (CheckAll) given valid inputs.
//
// Scope: T3 (Rollback), T4 (Invalidate), T5 (Advance). T1 and T2 are
// "no state update" — they are represented by the absence of a call, not
// by a method here.

// CloneSnapshot deep-copies a Snapshot so Apply* functions do not mutate
// their input. Exported so tests and fuzzers can make defensive copies.
func CloneSnapshot(s Snapshot) Snapshot {
	out := Snapshot{
		ActivationTS: s.ActivationTS,
		Chains:       append([]eth.ChainID{}, s.Chains...),
		LogsDB:       make(map[eth.ChainID][]BlockWithLogs, len(s.LogsDB)),
		Verified:     make([]VerifiedEntry, len(s.Verified)),
		DenyList:     make(map[eth.ChainID][]DenyListEntry, len(s.DenyList)),
	}
	for k, v := range s.LogsDB {
		out.LogsDB[k] = append([]BlockWithLogs{}, v...)
	}
	for k, v := range s.DenyList {
		out.DenyList[k] = append([]DenyListEntry{}, v...)
	}
	for i, v := range s.Verified {
		heads := make(map[eth.ChainID]eth.BlockID, len(v.L2Heads))
		for kk, vv := range v.L2Heads {
			heads[kk] = vv
		}
		out.Verified[i] = VerifiedEntry{
			Timestamp:   v.Timestamp,
			L1Inclusion: v.L1Inclusion,
			L2Heads:     heads,
		}
	}
	return out
}

// -----------------------------------------------------------------------------
// T5 — Advance
//
// SPEC.md T5 / overview.md lines 102-104:
//   - Verified extended with (t+1, C_{t+1}, C^j_{t+1}) where C^j_{t+1} = B_j
//     and C_{t+1} = max_j(B'_j).
//   - For every j such that B_j != B^j_{n_j}, L_j is extended with
//     (B_j, ℓ(B_j)); otherwise L_j is unchanged.
// -----------------------------------------------------------------------------

// ApplyAdvance advances the snapshot by one timestamp by committing a new
// VerifiedEntry and, for each chain whose head changed, extending its
// LogsDB with the new block.
//
// `newHeads[j]` is the full BlockWithLogs for chain j's new head. If a
// chain is NOT present in newHeads, its head stays the same (L2 block time
// exceeded the timestamp being verified).
func ApplyAdvance(
	s Snapshot,
	newL1 eth.BlockID,
	newHeads map[eth.ChainID]BlockWithLogs,
) (Snapshot, error) {
	var t uint64
	if len(s.Verified) == 0 {
		t = s.ActivationTS
	} else {
		last := s.Verified[len(s.Verified)-1].Timestamp
		if last == ^uint64(0) {
			return s, fmt.Errorf("advance: t+1 would overflow uint64")
		}
		t = last + 1
	}

	out := CloneSnapshot(s)
	entry := VerifiedEntry{
		Timestamp:   t,
		L1Inclusion: newL1,
		L2Heads:     make(map[eth.ChainID]eth.BlockID, len(s.Chains)),
	}

	for _, chain := range s.Chains {
		newBlock, moved := newHeads[chain]
		if moved {
			// SPEC A4: the VirtualNode never hands a denied block to the
			// Supernode. The reference model enforces this at the
			// precondition level rather than silently accepting an I10
			// violation.
			for _, d := range out.DenyList[chain] {
				if d.Block == newBlock.Ref.ID {
					return s, fmt.Errorf(
						"advance: chain %s block %s is in DenyList (A4 violation)",
						chain, newBlock.Ref.ID)
				}
			}
			entry.L2Heads[chain] = newBlock.Ref.ID
			logs := out.LogsDB[chain]
			// Only append if B_j != B^j_{n_j} (SPEC T5 second bullet).
			tailMatches := len(logs) > 0 &&
				logs[len(logs)-1].Ref.ID == newBlock.Ref.ID
			if !tailMatches {
				out.LogsDB[chain] = append(logs, newBlock)
			}
		} else {
			// Stay case: head is the previous verified head. Requires a
			// previous verified entry to copy from.
			if len(s.Verified) == 0 {
				return s, fmt.Errorf(
					"advance at t_0: chain %s has no newHead and no prior Verified",
					chain)
			}
			prev, ok := s.Verified[len(s.Verified)-1].L2Heads[chain]
			if !ok {
				return s, fmt.Errorf(
					"advance: chain %s missing from prior Verified.L2Heads", chain)
			}
			entry.L2Heads[chain] = prev
		}
	}

	out.Verified = append(out.Verified, entry)
	return out, nil
}

// -----------------------------------------------------------------------------
// T4 — Invalidate
//
// SPEC.md T4 / overview.md lines 98-101:
//   - Each invalid B_j is added to D_j with DecisionTimestamp = t + 1.
//   - Verified is unchanged.
//   - LogsDB is unchanged (any speculative additions are rolled back at
//     this atomic boundary).
// -----------------------------------------------------------------------------

// ApplyInvalidate adds one or more blocks to per-chain DenyLists with the
// current t+1 as the decision timestamp. The input `invalidHeads` names
// the block IDs being invalidated, keyed by chain.
func ApplyInvalidate(
	s Snapshot,
	invalidHeads map[eth.ChainID]eth.BlockID,
) (Snapshot, error) {
	if len(s.Verified) == 0 {
		return s, fmt.Errorf("invalidate: no prior verification (t is undefined)")
	}
	t := s.Verified[len(s.Verified)-1].Timestamp
	if t == ^uint64(0) {
		return s, fmt.Errorf("invalidate: t+1 would overflow uint64")
	}

	out := CloneSnapshot(s)
	for chain, blockID := range invalidHeads {
		out.DenyList[chain] = append(out.DenyList[chain], DenyListEntry{
			Block:             blockID,
			DecisionTimestamp: t + 1,
		})
	}
	return out, nil
}

// -----------------------------------------------------------------------------
// T3 — Rollback
//
// SPEC.md T3 / overview.md lines 94-97:
//   - Prune Verified by removing the last entry (t, C_t, {C^j_t}).
//   - Prune D_j by removing all entries with DecisionTimestamp >= t.
//   - For every chain j such that C^j_{t-1} != B^j_{n_j}, prune L_j by
//     removing its last entry.
// -----------------------------------------------------------------------------

// ApplyRewind rolls back the most recently committed VerifiedEntry and
// performs the associated DenyList + LogsDB pruning per SPEC T3.
func ApplyRewind(s Snapshot) (Snapshot, error) {
	if len(s.Verified) == 0 {
		return s, fmt.Errorf("rewind: Verified is empty")
	}
	out := CloneSnapshot(s)
	removed := out.Verified[len(out.Verified)-1]
	t := removed.Timestamp
	out.Verified = out.Verified[:len(out.Verified)-1]

	// Prune DenyList: remove any entry whose DecisionTimestamp >= t.
	for chain, denied := range out.DenyList {
		kept := denied[:0:0]
		for _, d := range denied {
			if d.DecisionTimestamp < t {
				kept = append(kept, d)
			}
		}
		out.DenyList[chain] = kept
	}

	// Prune LogsDB tails: only pop if the (now-new) last verified head does
	// not equal the LogsDB tail. If Verified became empty, pop the tail
	// unconditionally (no "previous head" to compare against).
	for _, chain := range s.Chains {
		logs := out.LogsDB[chain]
		if len(logs) == 0 {
			continue
		}
		tail := logs[len(logs)-1].Ref.ID

		var popTail bool
		if len(out.Verified) == 0 {
			// No prior verified entry: the tail was added in the round we
			// just rolled back.
			popTail = true
		} else {
			prev, ok := out.Verified[len(out.Verified)-1].L2Heads[chain]
			if !ok || prev != tail {
				popTail = true
			}
		}
		if popTail {
			out.LogsDB[chain] = logs[:len(logs)-1]
		}
	}

	return out, nil
}

// -----------------------------------------------------------------------------
// Initial state constructor
//
// For tests and fuzzers. Constructs a Snapshot satisfying I12 (initial
// state trivially satisfies all structural invariants). No VerifiedDB
// anchor yet — the first ApplyAdvance creates it.
// -----------------------------------------------------------------------------

// NewInitialSnapshot constructs an empty snapshot for the given chains and
// activation timestamp. LogsDB and DenyList start empty for every chain.
// No Verified entries. Mirrors the Dafny IsInitialState predicate.
func NewInitialSnapshot(activationTS uint64, chains []eth.ChainID) Snapshot {
	s := Snapshot{
		ActivationTS: activationTS,
		Chains:       append([]eth.ChainID{}, chains...),
		LogsDB:       make(map[eth.ChainID][]BlockWithLogs, len(chains)),
		DenyList:     make(map[eth.ChainID][]DenyListEntry, len(chains)),
	}
	for _, c := range chains {
		s.LogsDB[c] = nil
		s.DenyList[c] = nil
	}
	return s
}
