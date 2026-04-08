package invariants

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// regenTraces, when set, rewrites the canonical traces under
// testdata/traces/. Run with:
//
//	go test ./op-supernode/invariants/ -run=TestRegenCanonicalTraces -regen-traces
//
// The regenerator is the canonical writer — hand-edited trace JSON is
// discouraged because the (chain ID, block ID) serialization rules are
// finicky. Use Go constructors and Save.
var regenTraces = flag.Bool("regen-traces", false,
	"rewrite canonical traces under testdata/traces/")

func TestRegenCanonicalTraces(t *testing.T) {
	if !*regenTraces {
		t.Skip("pass -regen-traces to rewrite traces")
	}
	dir := filepath.Join("testdata", "traces")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, c := range canonicalTraces(t) {
		p := filepath.Join(dir, c.Name+".json")
		if err := SaveTrace(p, c.Trace); err != nil {
			t.Fatalf("save %s: %v", p, err)
		}
		t.Logf("wrote %s", p)
	}
}

// TestCanonicalTracesInline exercises the trace generators themselves so
// they stay correct even if the on-disk files drift.
func TestCanonicalTracesInline(t *testing.T) {
	for _, c := range canonicalTraces(t) {
		c := c
		t.Run(c.Name, func(t *testing.T) {
			violations, err := VerifyTrace(c.Trace)
			if err != nil {
				for _, v := range violations {
					t.Log(v)
				}
				t.Fatalf("trace %q failed: %v", c.Name, err)
			}
		})
	}
}

type namedTrace struct {
	Name  string
	Trace *Trace
}

func canonicalTraces(t *testing.T) []namedTrace {
	return []namedTrace{
		{Name: "happy_single_chain_3_advances", Trace: traceHappySingleChain3(t)},
		{Name: "happy_invalidate_then_continue", Trace: traceInvalidateThenContinue(t)},
		{Name: "happy_rewind_after_two_advances", Trace: traceRewindAfterTwoAdvances(t)},
	}
}

// traceHappySingleChain3: activation at t_0=100, then three advances.
func traceHappySingleChain3(t *testing.T) *Trace {
	t.Helper()
	s0 := NewInitialSnapshot(100, []eth.ChainID{chainA()})
	s1, err := ApplyAdvance(s0, blockID(10, 1000),
		map[eth.ChainID]BlockWithLogs{chainA(): block(1, 0, 1, 100)})
	if err != nil {
		t.Fatal(err)
	}
	s2, err := ApplyAdvance(s1, blockID(11, 1001),
		map[eth.ChainID]BlockWithLogs{chainA(): block(2, 1, 2, 101)})
	if err != nil {
		t.Fatal(err)
	}
	s3, err := ApplyAdvance(s2, blockID(12, 1002),
		map[eth.ChainID]BlockWithLogs{chainA(): block(3, 2, 3, 102)})
	if err != nil {
		t.Fatal(err)
	}
	return &Trace{
		Description: "single chain, three T5 advances from t_0=100 through t=102",
		Origin:      "hand-crafted",
		Steps:       []Snapshot{s0, s1, s2, s3},
	}
}

// traceInvalidateThenContinue: advance, invalidate a speculative block at
// t+1, then advance with a different block at the same timestamp.
func traceInvalidateThenContinue(t *testing.T) *Trace {
	t.Helper()
	s0 := NewInitialSnapshot(100, []eth.ChainID{chainA()})
	s1, err := ApplyAdvance(s0, blockID(10, 1000),
		map[eth.ChainID]BlockWithLogs{chainA(): block(1, 0, 1, 100)})
	if err != nil {
		t.Fatal(err)
	}
	// At t=100, invalidate a block that would have been considered at t+1.
	s2, err := ApplyInvalidate(s1, map[eth.ChainID]eth.BlockID{
		chainA(): blockID(99, 2),
	})
	if err != nil {
		t.Fatal(err)
	}
	// Now advance with a DIFFERENT block at t+1.
	s3, err := ApplyAdvance(s2, blockID(11, 1001),
		map[eth.ChainID]BlockWithLogs{chainA(): block(2, 1, 2, 101)})
	if err != nil {
		t.Fatal(err)
	}
	return &Trace{
		Description: "advance, invalidate speculative block at t+1, advance with replacement",
		Origin:      "hand-crafted",
		Steps:       []Snapshot{s0, s1, s2, s3},
	}
}

// traceRewindAfterTwoAdvances: build up two advances then rewind one.
func traceRewindAfterTwoAdvances(t *testing.T) *Trace {
	t.Helper()
	s0 := NewInitialSnapshot(100, []eth.ChainID{chainA()})
	s1, err := ApplyAdvance(s0, blockID(10, 1000),
		map[eth.ChainID]BlockWithLogs{chainA(): block(1, 0, 1, 100)})
	if err != nil {
		t.Fatal(err)
	}
	s2, err := ApplyAdvance(s1, blockID(11, 1001),
		map[eth.ChainID]BlockWithLogs{chainA(): block(2, 1, 2, 101)})
	if err != nil {
		t.Fatal(err)
	}
	s3, err := ApplyRewind(s2)
	if err != nil {
		t.Fatal(err)
	}
	return &Trace{
		Description: "two advances followed by T3 rewind back to t=100",
		Origin:      "hand-crafted",
		Steps:       []Snapshot{s0, s1, s2, s3},
	}
}
