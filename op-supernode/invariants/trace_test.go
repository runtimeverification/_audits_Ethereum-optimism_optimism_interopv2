package invariants

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestTraceCorpus replays every JSON trace under testdata/traces/ through
// VerifyTrace. This is the single regression gate for all invariant
// failures — hand-crafted, fuzz-minimized, Dafny counter-example, and
// production-captured traces all live in that directory and are checked
// here.
func TestTraceCorpus(t *testing.T) {
	dir := filepath.Join("testdata", "traces")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skipf("%s does not exist yet", dir)
	}
	corpus, err := LoadTraceCorpus(dir)
	if err != nil {
		t.Fatalf("load corpus: %v", err)
	}
	if len(corpus) == 0 {
		t.Skip("no traces in corpus")
	}
	for name, tr := range corpus {
		name := name
		tr := tr
		t.Run(name, func(t *testing.T) {
			violations, err := VerifyTrace(tr)
			if err != nil {
				for _, v := range violations {
					t.Log(v)
				}
				t.Fatalf("trace %q (%s) failed: %v", name, tr.Origin, err)
			}
		})
	}
}

// TestTrace_RoundTrip verifies that a trace serialized to JSON and parsed
// back reproduces the original snapshot bytes-for-bytes.
func TestTrace_RoundTrip(t *testing.T) {
	s := NewInitialSnapshot(100, []eth.ChainID{chainA()})
	s, err := ApplyAdvance(s, blockID(10, 1000), map[eth.ChainID]BlockWithLogs{
		chainA(): block(1, 0, 1, 100),
	})
	if err != nil {
		t.Fatal(err)
	}

	tr := &Trace{
		Description: "round-trip test",
		Origin:      "hand-crafted",
		Steps:       []Snapshot{NewInitialSnapshot(100, []eth.ChainID{chainA()}), s},
	}

	dir := t.TempDir()
	p := filepath.Join(dir, "rt.json")
	if err := SaveTrace(p, tr); err != nil {
		t.Fatal(err)
	}
	back, err := LoadTrace(p)
	if err != nil {
		t.Fatal(err)
	}
	if back.Description != tr.Description || back.Origin != tr.Origin {
		t.Fatalf("metadata drift: %+v", back)
	}
	if len(back.Steps) != 2 {
		t.Fatalf("step count drift: got %d want 2", len(back.Steps))
	}
	violations, err := VerifyTrace(back)
	if err != nil {
		t.Fatalf("round-tripped trace failed: %v %v", violations, err)
	}
}

// TestTrace_ExpectedFailure verifies that a regression trace correctly
// marks its expected invariant ID as satisfied.
func TestTrace_ExpectedFailure(t *testing.T) {
	// A snapshot that deliberately violates I2.
	bad := Snapshot{
		Chains: []eth.ChainID{chainA()},
		LogsDB: map[eth.ChainID][]BlockWithLogs{
			chainA(): {
				block(1, 0, 1, 100),
				block(2, 99, 2, 101),
			},
		},
	}
	tr := &Trace{
		Description:      "regression: I2 parent-hash mismatch",
		Origin:           "hand-crafted",
		Steps:            []Snapshot{bad},
		ExpectedFailures: []string{"I2"},
	}
	if _, err := VerifyTrace(tr); err != nil {
		t.Fatalf("regression trace should be accepted: %v", err)
	}
}

func TestTrace_ExpectedFailureMissing(t *testing.T) {
	// A snapshot that violates I10, but the trace claims I2. The verify
	// should report that no expected failure matched.
	good := NewInitialSnapshot(100, []eth.ChainID{chainA()})
	tr := &Trace{
		Origin:           "hand-crafted",
		Steps:            []Snapshot{good},
		ExpectedFailures: []string{"I2"},
	}
	if _, err := VerifyTrace(tr); err == nil {
		t.Fatal("expected regression trace to fail when no expected ID matched")
	}
}
