package interop

import (
	"errors"
	"sync/atomic"
)

// errWALInjectorCrash is returned by walInjector once the configured WAL-op
// count is reached. It signals "the process would have crashed here" — the
// surrounding progressAndRecord call propagates the error up and the fuzz
// body then drops the Interop instance, simulating an unclean shutdown.
var errWALInjectorCrash = errors.New("walInjector: simulated crash")

// walInjector wraps a verifiedStore and forces a crash signal once the
// configured number of WAL operations has been observed. Used by the
// crash-recovery fuzz target to drop an Interop mid-transition (after the
// WAL persistence step, before ClearPendingTransition).
//
// The decorator is deliberately minimal — it counts SetPendingTransition,
// ClearPendingTransition, Commit, and Rewind because all four touch the
// durable WAL/state surface that PR #19505's crash-safety contract relies
// on. Reads (Get/Has/*Timestamp) are passed through verbatim so the
// post-crash Interop can re-observe its state without interference.
//
// crashAfter == 0 disables crashing; the wrapper then behaves identically
// to the inner store.
type walInjector struct {
	inner      verifiedStore
	crashAfter int
	calls      atomic.Int32
}

func newWALInjector(inner verifiedStore, crashAfter int) *walInjector {
	return &walInjector{inner: inner, crashAfter: crashAfter}
}

// crashed reports whether the injector has fired at least once. The fuzz
// body uses this to decide whether the crash run actually exercised the
// recovery path (some seeds don't generate enough WAL traffic to reach
// the threshold).
func (w *walInjector) crashed() bool {
	return w.crashAfter > 0 && int(w.calls.Load()) >= w.crashAfter
}

// noteWALOp counts a WAL-touching op and returns errWALInjectorCrash once
// the threshold is reached. After a single crash signal the counter is
// frozen so subsequent inner calls also see the error — that prevents the
// inner store from being torn down mid-bbolt-write while the fuzz body is
// still draining the progress loop.
func (w *walInjector) noteWALOp() error {
	if w.crashAfter <= 0 {
		return nil
	}
	if int(w.calls.Add(1)) >= w.crashAfter {
		return errWALInjectorCrash
	}
	return nil
}

// --- Read methods: pass-through ---

func (w *walInjector) Get(ts uint64) (VerifiedResult, error) { return w.inner.Get(ts) }
func (w *walInjector) Has(ts uint64) (bool, error)           { return w.inner.Has(ts) }
func (w *walInjector) FirstTimestamp() (uint64, bool)        { return w.inner.FirstTimestamp() }
func (w *walInjector) LastTimestamp() (uint64, bool)         { return w.inner.LastTimestamp() }
func (w *walInjector) GetPendingTransition() (*PendingTransition, error) {
	return w.inner.GetPendingTransition()
}

// --- Write methods: count and optionally short-circuit ---

func (w *walInjector) Commit(result VerifiedResult) error {
	if err := w.noteWALOp(); err != nil {
		return err
	}
	return w.inner.Commit(result)
}

func (w *walInjector) Rewind(timestamp uint64) (bool, error) {
	if err := w.noteWALOp(); err != nil {
		return false, err
	}
	return w.inner.Rewind(timestamp)
}

// SetPendingTransition writes the WAL entry through to the inner store
// *before* checking the crash threshold. The crash-safety contract under
// test is "WAL entry persisted but apply not run yet"; aborting the call
// AFTER the inner write produces exactly that on-disk state.
func (w *walInjector) SetPendingTransition(pending PendingTransition) error {
	if err := w.inner.SetPendingTransition(pending); err != nil {
		return err
	}
	return w.noteWALOp()
}

// ClearPendingTransition checks the crash threshold *before* the inner
// call. Aborting here is the second target state for the crash-safety
// contract: "transition applied, WAL not yet cleared". On recovery the
// new Interop must replay the pending idempotently.
func (w *walInjector) ClearPendingTransition() error {
	if err := w.noteWALOp(); err != nil {
		return err
	}
	return w.inner.ClearPendingTransition()
}

func (w *walInjector) Close() error { return w.inner.Close() }
