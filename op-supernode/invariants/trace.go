package invariants

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Trace is the on-disk format for an invariant-verification test case. It
// records a sequence of snapshots (one per step) plus a human-readable
// description. The Dafny counter-example printer, the Go fuzzer, and the
// live-execution snapshotter all serialize into this format so a single
// replay tool (`TestTraceCorpus`) can verify every source.
//
// On-disk location: invariants/testdata/traces/<name>.json.
type Trace struct {
	// Description is a short human note about the origin of the trace:
	// "fuzz-FuzzReferenceModel-7419c...", "dafny-counterexample-I6-2026-03-...",
	// "prod-op-sepolia-2026-03-15T12:00:00Z", etc.
	Description string `json:"description"`

	// Origin names the layer that produced the trace. One of:
	// "dafny" | "fuzz" | "reference" | "production" | "hand-crafted".
	Origin string `json:"origin"`

	// Steps is the sequence of snapshots, one per transition. The first
	// step is the initial state; every subsequent step MUST be the result
	// of a single T3/T4/T5 transition from the previous step.
	Steps []Snapshot `json:"steps"`

	// ExpectedFailures, if non-empty, names the SPEC invariant IDs that
	// SHOULD fail on at least one step of this trace. Used for regression
	// traces where the point of the trace is that the failure is caught.
	// An empty list means every step must satisfy CheckAll.
	ExpectedFailures []string `json:"expectedFailures,omitempty"`
}

// LoadTrace reads a trace from disk.
func LoadTrace(path string) (*Trace, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read trace %s: %w", path, err)
	}
	var tr Trace
	if err := json.Unmarshal(data, &tr); err != nil {
		return nil, fmt.Errorf("parse trace %s: %w", path, err)
	}
	return &tr, nil
}

// SaveTrace writes a trace to disk, creating parent directories as needed.
func SaveTrace(path string, tr *Trace) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(tr, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// VerifyTrace runs CheckAll on every step. Returns (violations found in
// order, error). If ExpectedFailures is non-empty, VerifyTrace instead
// checks that at least one step failed with one of those IDs.
func VerifyTrace(tr *Trace) ([]error, error) {
	if tr == nil {
		return nil, fmt.Errorf("nil trace")
	}
	if len(tr.Steps) == 0 {
		return nil, fmt.Errorf("trace has no steps")
	}

	stepErrs := make([]error, len(tr.Steps))
	for i, step := range tr.Steps {
		stepErrs[i] = CheckAll(step)
	}

	if len(tr.ExpectedFailures) == 0 {
		// Every step must pass.
		var violations []error
		for i, e := range stepErrs {
			if e != nil {
				violations = append(violations,
					fmt.Errorf("step %d: %w", i, e))
			}
		}
		if len(violations) > 0 {
			return violations, fmt.Errorf("%d step(s) violated invariants",
				len(violations))
		}
		return nil, nil
	}

	// Regression trace: at least one expected failure ID must appear.
	expected := make(map[string]struct{}, len(tr.ExpectedFailures))
	for _, id := range tr.ExpectedFailures {
		expected[id] = struct{}{}
	}
	matched := make(map[string]struct{})
	for _, e := range stepErrs {
		collectIDs(e, func(id string) {
			if _, want := expected[id]; want {
				matched[id] = struct{}{}
			}
		})
	}
	if len(matched) == 0 {
		return nil, fmt.Errorf("regression trace expected one of %v but none matched",
			tr.ExpectedFailures)
	}
	return nil, nil
}

// collectIDs walks a joined error tree invoking visit on every *Error's
// ID. Uses errors.As so wrapped errors (e.g. fmt.Errorf("%w", e)) are
// still discovered.
func collectIDs(err error, visit func(string)) {
	if err == nil {
		return
	}
	var ie *Error
	if errors.As(err, &ie) {
		visit(ie.ID)
	}
	// Walk multi-error trees too: errors.As only finds the first match, so
	// for joined errors we need to descend manually.
	type multi interface{ Unwrap() []error }
	if m, ok := err.(multi); ok {
		for _, c := range m.Unwrap() {
			collectIDs(c, visit)
		}
		return
	}
	// Single-wrap chain: descend via the standard Unwrap to find any
	// nested *Error past the first match.
	type single interface{ Unwrap() error }
	if s, ok := err.(single); ok {
		collectIDs(s.Unwrap(), visit)
	}
}

// LoadTraceCorpus loads every *.json file under dir as a Trace, keyed by
// filename (without extension). Non-JSON files are skipped.
func LoadTraceCorpus(dir string) (map[string]*Trace, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make(map[string]*Trace)
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		p := filepath.Join(dir, e.Name())
		tr, err := LoadTrace(p)
		if err != nil {
			return nil, err
		}
		name := e.Name()[:len(e.Name())-len(".json")]
		out[name] = tr
	}
	return out, nil
}
