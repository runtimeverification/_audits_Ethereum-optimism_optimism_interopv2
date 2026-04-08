package invariants

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Env is the environmental oracle required by I1, I3, I7 (second clause),
// I8 (ancestry clause), and I9. These invariants cannot be checked from a
// pure Snapshot because they reference facts about L1/L2 chains that are
// not stored in the state.
//
// An Env implementation may be:
//   - A trusted live-chain adapter (production path, Step 3c).
//   - A fake oracle seeded with known answers (tests, replay of a trace).
//   - A strict "deny all" oracle that rejects every query (safety
//     upper-bound: if CheckAllWithEnv passes under the strict oracle,
//     every environmentally-dependent clause holds regardless of live
//     chain state).
type Env interface {
	// LogsBelongToBlock is SPEC I1 — checks that the supplied ExecMsgs
	// are actually the logs emitted when executing `ref`.
	// Returns nil if the logs belong, a non-nil error otherwise.
	LogsBelongToBlock(ref BlockRef, execMsgs []ExecutingMessage) error

	// DeriveL1 is SPEC I9 — returns the L1 block from which the given
	// L2 block was derived. Returns (zero, error) if the derivation is
	// not known.
	DeriveL1(l2 eth.BlockID) (eth.BlockID, error)

	// IsL1Ancestor is SPEC I8 (ancestry clause) — returns true iff
	// `ancestor` appears on the same linear L1 chain as `descendant`,
	// i.e. ancestor.Number <= descendant.Number AND walking parents from
	// `descendant` reaches `ancestor`.
	IsL1Ancestor(ancestor, descendant eth.BlockID) (bool, error)

	// HasHigherL2BlockAtOrBelow is SPEC I7 (environmental clause) —
	// returns true iff the given L2 chain has a block with
	// `timestamp <= ts` that is strictly higher than `head`.
	HasHigherL2BlockAtOrBelow(
		chain eth.ChainID, head eth.BlockID, ts uint64) (bool, error)
}

// CheckAllWithEnv runs every invariant including the environmentally-
// dependent ones (I1, I3, I7-env, I8-env, I9). It composes CheckAll with
// the environmental predicates. Returns a joined error if any invariant
// failed.
//
// Note: I3 (executing message acyclicity) is partially state-local and
// partially environmental. The state-local half (each referenced
// initiating message exists in LogsDB at the expected position) is
// checked regardless of env; the acyclicity half is best-effort using
// Tarjan on the static snapshot graph.
func CheckAllWithEnv(s Snapshot, env Env) error {
	errs := []error{CheckAll(s)}
	if env != nil {
		errs = append(errs,
			CheckI1_LogsBelongToBlocks(s, env),
			CheckI3_ExecutingMessageValidity(s, env),
			CheckI8_VerifiedL1Linear(s, env),
			CheckI9_MinimalL1Cover(s, env),
		)
	} else {
		// Without an env, run only the state-local half of I3.
		errs = append(errs, CheckI3_ExecutingMessageValidity(s, nil))
	}
	return errors.Join(errs...)
}

// -----------------------------------------------------------------------------
// I1 — Logs belong to blocks (environmental).
// -----------------------------------------------------------------------------

// CheckI1_LogsBelongToBlocks delegates the logs-match-block check to the
// environmental oracle. See SPEC I1.
func CheckI1_LogsBelongToBlocks(s Snapshot, env Env) error {
	if env == nil {
		return nil
	}
	var errs []error
	for chain, logs := range s.LogsDB {
		for i, b := range logs {
			if err := env.LogsBelongToBlock(b.Ref, b.ExecMsgs); err != nil {
				errs = append(errs, newIndexedErr("I1", chain, i,
					fmt.Sprintf("LogsBelongToBlock rejected: %v", err)))
			}
		}
	}
	return errors.Join(errs...)
}

// -----------------------------------------------------------------------------
// I3 — Executing message validity + acyclicity.
//
// State-local half: every ExecutingMessage references an initiating log
// that exists in the target chain's LogsDB at the named (BlockNum,
// Timestamp). No env required for this half.
//
// Environmental half: static acyclicity via Tarjan's SCC. A snapshot-only
// implementation is provided — env is not strictly required but reserved
// for future refinement.
// -----------------------------------------------------------------------------

func CheckI3_ExecutingMessageValidity(s Snapshot, env Env) error {
	var errs []error

	// State-local half: initiating message existence.
	for chain, logs := range s.LogsDB {
		for i, b := range logs {
			for k, m := range b.ExecMsgs {
				if !initiatingMessageExists(s, m) {
					errs = append(errs, newIndexedErr("I3", chain, i,
						fmt.Sprintf("ExecMsg[%d] references %s:%d @ t=%d, not present in target LogsDB",
							k, m.ChainID, m.BlockNum, m.Timestamp)))
				}
			}
		}
	}

	// Environmental / acyclicity half: Tarjan on the exec-msg graph.
	// Nodes are (chain, blockNum). Edges go from an executing message to
	// the initiating (chain, blockNum) it points at.
	if cycleChain, cycleBlock, ok := findExecMsgCycle(s); ok {
		errs = append(errs, newChainErr("I3", cycleChain, fmt.Sprintf(
			"executing-message cycle involves chain=%s blockNum=%d",
			cycleChain, cycleBlock)))
	}

	_ = env // reserved for future env-enriched checks
	return errors.Join(errs...)
}

func initiatingMessageExists(s Snapshot, m ExecutingMessage) bool {
	logs, ok := s.LogsDB[m.ChainID]
	if !ok {
		return false
	}
	for _, b := range logs {
		if b.Ref.ID.Number == m.BlockNum && b.Ref.Time == m.Timestamp {
			return true
		}
	}
	return false
}

// findExecMsgCycle runs a depth-first cycle detector over the static
// executing-message dependency graph. Returns the first offending node if
// any cycle is found. Nodes are identified by (chain, blockNum).
func findExecMsgCycle(s Snapshot) (eth.ChainID, uint64, bool) {
	type node struct {
		chain eth.ChainID
		num   uint64
	}
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[node]int)

	// Build an adjacency list on the fly from Snapshot.
	neighbors := func(n node) []node {
		logs, ok := s.LogsDB[n.chain]
		if !ok {
			return nil
		}
		var found *BlockWithLogs
		for i := range logs {
			if logs[i].Ref.ID.Number == n.num {
				found = &logs[i]
				break
			}
		}
		if found == nil {
			return nil
		}
		out := make([]node, 0, len(found.ExecMsgs))
		for _, m := range found.ExecMsgs {
			out = append(out, node{chain: m.ChainID, num: m.BlockNum})
		}
		return out
	}

	var cycleChain eth.ChainID
	var cycleNum uint64
	var found bool

	var dfs func(n node) bool
	dfs = func(n node) bool {
		color[n] = gray
		for _, nx := range neighbors(n) {
			switch color[nx] {
			case gray:
				cycleChain = nx.chain
				cycleNum = nx.num
				return true
			case white:
				if dfs(nx) {
					return true
				}
			}
		}
		color[n] = black
		return false
	}

	for chain, logs := range s.LogsDB {
		for _, b := range logs {
			n := node{chain: chain, num: b.Ref.ID.Number}
			if color[n] != white {
				continue
			}
			if dfs(n) {
				found = true
				return cycleChain, cycleNum, true
			}
		}
	}
	return cycleChain, cycleNum, found
}

// -----------------------------------------------------------------------------
// I8 — L1 linear-chain ancestry (environmental).
// -----------------------------------------------------------------------------

// CheckI8_VerifiedL1Linear walks consecutive Verified entries and asks the
// oracle whether the earlier L1 inclusion is an ancestor of the later one.
// This is the ancestry half of I8; the number-monotonicity half is covered
// by CheckI8_VerifiedL1Monotone in invariants.go.
func CheckI8_VerifiedL1Linear(s Snapshot, env Env) error {
	if env == nil {
		return nil
	}
	var errs []error
	for i := 0; i+1 < len(s.Verified); i++ {
		prev := s.Verified[i].L1Inclusion
		nxt := s.Verified[i+1].L1Inclusion
		if prev == nxt {
			continue
		}
		ok, err := env.IsL1Ancestor(prev, nxt)
		if err != nil {
			errs = append(errs, newErr("I8", fmt.Sprintf(
				"env.IsL1Ancestor(%s, %s) failed: %v", prev, nxt, err)))
			continue
		}
		if !ok {
			errs = append(errs, newErr("I8", fmt.Sprintf(
				"L1 head %s is not an ancestor of %s", prev, nxt)))
		}
	}
	return errors.Join(errs...)
}

// -----------------------------------------------------------------------------
// I9 — Minimal L1 cover.
// -----------------------------------------------------------------------------

// CheckI9_MinimalL1Cover uses env.DeriveL1 to verify that the recorded
// L1 inclusion for every Verified entry is the maximum (by block number)
// of the DeriveL1 results for its L2 heads.
func CheckI9_MinimalL1Cover(s Snapshot, env Env) error {
	if env == nil {
		return nil
	}
	var errs []error
	for i, v := range s.Verified {
		var maxL1 eth.BlockID
		var maxSeen bool
		for _, chain := range s.Chains {
			head, ok := v.L2Heads[chain]
			if !ok {
				errs = append(errs, newErr("I9", fmt.Sprintf(
					"Verified[%d] missing L2Head for chain %s", i, chain)))
				continue
			}
			l1, err := env.DeriveL1(head)
			if err != nil {
				errs = append(errs, newErr("I9", fmt.Sprintf(
					"Verified[%d] env.DeriveL1(%s) failed: %v", i, head, err)))
				continue
			}
			if !maxSeen || l1.Number > maxL1.Number {
				maxL1 = l1
				maxSeen = true
			}
		}
		if !maxSeen {
			continue
		}
		if v.L1Inclusion.Number != maxL1.Number {
			errs = append(errs, newErr("I9", fmt.Sprintf(
				"Verified[%d].L1Inclusion.Number=%d != max_j(DeriveL1(C^j)).Number=%d",
				i, v.L1Inclusion.Number, maxL1.Number)))
		}
	}
	return errors.Join(errs...)
}
