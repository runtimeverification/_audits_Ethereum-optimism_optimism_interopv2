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
			CheckI7_NoHigherL2Block(s, env),
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

// findExecMsgCycle runs an iterative DFS cycle detector over the static
// executing-message dependency graph. Iterative (rather than recursive)
// so a deep graph cannot blow the goroutine stack and panic the
// supernode under runtime assertions. Returns the first node observed on
// a back-edge if any cycle is found. Nodes are identified by
// (chain, blockNum).
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

	// Build the adjacency list once. The lookup-by-number scan inside the
	// per-call neighbors closure was O(n) per visit; this materializes it
	// as O(total blocks) once.
	adj := make(map[node][]node)
	for chain, logs := range s.LogsDB {
		for _, b := range logs {
			n := node{chain: chain, num: b.Ref.ID.Number}
			out := make([]node, 0, len(b.ExecMsgs))
			for _, m := range b.ExecMsgs {
				out = append(out, node{chain: m.ChainID, num: m.BlockNum})
			}
			adj[n] = out
		}
	}

	// frame is one entry on the explicit DFS stack. childIdx is the index
	// of the next child to visit; when childIdx == len(adj[node]) the
	// node is finished and gets colored black on pop.
	type frame struct {
		n        node
		childIdx int
	}

	for chain, logs := range s.LogsDB {
		for _, b := range logs {
			start := node{chain: chain, num: b.Ref.ID.Number}
			if color[start] != white {
				continue
			}
			stack := []frame{{n: start}}
			color[start] = gray
			for len(stack) > 0 {
				top := &stack[len(stack)-1]
				children := adj[top.n]
				if top.childIdx >= len(children) {
					color[top.n] = black
					stack = stack[:len(stack)-1]
					continue
				}
				nx := children[top.childIdx]
				top.childIdx++
				switch color[nx] {
				case gray:
					return nx.chain, nx.num, true
				case white:
					color[nx] = gray
					stack = append(stack, frame{n: nx})
				}
			}
		}
	}
	return eth.ChainID{}, 0, false
}

// -----------------------------------------------------------------------------
// I7 — Environmental clause: no higher L2 block exists with timestamp <= ts.
//
// SPEC.md I7 second sentence: "All children of C^j_i have timestamp > i."
// The LogsDB-internal half is checked structurally by CheckI7_HighestBlockLeqTS
// in invariants.go. This env predicate handles the "live L2 may have a
// higher block we have not yet imported" case by asking the oracle.
// -----------------------------------------------------------------------------

// CheckI7_NoHigherL2Block delegates the "no descendant on the live L2
// with timestamp <= verified timestamp other than C^j_i" clause to the
// environmental oracle. See SPEC I7 (HYBRID class) and SPEC §4.
func CheckI7_NoHigherL2Block(s Snapshot, env Env) error {
	if env == nil {
		return nil
	}
	var errs []error
	for vi, v := range s.Verified {
		for _, chain := range s.Chains {
			head, ok := v.L2Heads[chain]
			if !ok {
				continue // structural CheckI7 already reports this
			}
			higher, err := env.HasHigherL2BlockAtOrBelow(chain, head, v.Timestamp)
			if err != nil {
				errs = append(errs, newIndexedErr("I7", chain, vi, fmt.Sprintf(
					"env.HasHigherL2BlockAtOrBelow(%s, %s, %d) failed: %v",
					chain, head, v.Timestamp, err)))
				continue
			}
			if higher {
				errs = append(errs, newIndexedErr("I7", chain, vi, fmt.Sprintf(
					"live L2 has a block strictly higher than C^j_%d=%s with Time<=%d",
					vi, head, v.Timestamp)))
			}
		}
	}
	return errors.Join(errs...)
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
// L1 inclusion for every Verified entry is the maximum of the DeriveL1
// results for its L2 heads.
//
// "Maximum" is checked by full BlockID equality (not just .Number) so
// that two distinct L1 blocks at the same height after a reorg do not
// silently pass. Additionally, every per-chain DeriveL1(C^j) must be an
// ancestor of v.L1Inclusion (or equal), validating the "minimal L1
// covering all heads" half via env.IsL1Ancestor.
func CheckI9_MinimalL1Cover(s Snapshot, env Env) error {
	if env == nil {
		return nil
	}
	var errs []error
	for i, v := range s.Verified {
		var maxL1 eth.BlockID
		var maxSeen bool
		// Pass 1: find the max-by-number derive among all L2 heads, and
		// verify each derive is an ancestor of v.L1Inclusion.
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
			// Ancestry check: deriveL1(C^j) must lie on the same linear
			// chain as v.L1Inclusion (either equal, or an ancestor).
			if l1 != v.L1Inclusion {
				ok, ancErr := env.IsL1Ancestor(l1, v.L1Inclusion)
				if ancErr != nil {
					errs = append(errs, newErr("I9", fmt.Sprintf(
						"Verified[%d] env.IsL1Ancestor(%s, %s) failed: %v",
						i, l1, v.L1Inclusion, ancErr)))
					continue
				}
				if !ok {
					errs = append(errs, newErr("I9", fmt.Sprintf(
						"Verified[%d] DeriveL1(C^j on %s)=%s is not an ancestor of L1Inclusion=%s",
						i, chain, l1, v.L1Inclusion)))
				}
			}
		}
		if !maxSeen {
			continue
		}
		// Pass 2: the recorded L1Inclusion must equal the max DeriveL1
		// (by full BlockID, not just Number) so that two distinct L1
		// blocks at the same height after a reorg are not conflated.
		if v.L1Inclusion != maxL1 {
			errs = append(errs, newErr("I9", fmt.Sprintf(
				"Verified[%d].L1Inclusion=%s != max_j(DeriveL1(C^j))=%s",
				i, v.L1Inclusion, maxL1)))
		}
	}
	return errors.Join(errs...)
}
