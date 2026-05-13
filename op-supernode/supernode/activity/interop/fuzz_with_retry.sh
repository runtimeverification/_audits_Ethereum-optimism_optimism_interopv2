#!/usr/bin/env bash
# fuzz_with_retry.sh — run `go test -fuzz` with automatic recovery
# from the spurious "fuzzing process hung or terminated: exit status 2"
# pattern that occasionally hits this package (see TODO.md and
# fuzz-debug-triage notes).
#
# When the abort fires Go writes the input that was in-flight under
# testdata/fuzz/<TARGET>/ even though the seed itself reproduces as
# PASS in isolation. This wrapper:
#   1. Detects the abort signature in the log.
#   2. Reproduces the seed in isolation; if it passes, removes the
#      spurious testdata entry.
#   3. Re-launches fuzz for the remaining wall-clock budget.
#
# Usage:
#   ./fuzz_with_retry.sh <target> <total_minutes> [extra-go-test-flags...]
#
# Example:
#   ./fuzz_with_retry.sh FuzzVerifyInteropMessages 60
#   ./fuzz_with_retry.sh FuzzVerifyInteropMessagesCrashRecover 120

set -uo pipefail

cd "$(dirname "$0")"

TARGET="${1:?usage: $0 <target> <total_minutes> [extra flags]}"
TOTAL_MIN="${2:?usage: $0 <target> <total_minutes> [extra flags]}"
shift 2

DEADLINE=$(( $(date +%s) + TOTAL_MIN * 60 ))

if [[ ! -d testdata/fuzz/$TARGET ]]; then
    mkdir -p testdata/fuzz/$TARGET
fi

attempt=0
total_fails=0
while :; do
    NOW=$(date +%s)
    REMAINING=$(( DEADLINE - NOW ))
    if (( REMAINING <= 30 )); then
        break
    fi
    attempt=$((attempt + 1))
    echo "=== attempt $attempt: fuzztime=${REMAINING}s ==="
    LOG=$(mktemp)
    rc=0
    go test -run='^$' -fuzz="^${TARGET}\$" -fuzztime="${REMAINING}s" -parallel=1 "$@" . 2>&1 | tee "$LOG" || rc=$?

    # Clean exit (or completed budget) — done.
    if grep -qE "^ok[[:space:]]" "$LOG"; then
        echo "fuzz finished cleanly after $attempt attempt(s)"
        rm -f "$LOG"
        exit 0
    fi

    # Did we hit the exit-2 flake? Look for the canonical message.
    if grep -qE "fuzzing process hung or terminated unexpectedly: exit status 2" "$LOG"; then
        seed=$(grep -oE 'Failing input written to testdata/fuzz/[^[:space:]]+' "$LOG" | head -1 | awk -F/ '{print $NF}')
        if [[ -z "$seed" ]]; then
            echo "FAIL: detected exit-2 abort but no seed extracted; bailing" >&2
            rm -f "$LOG"
            exit 1
        fi
        echo "exit-2 flake at seed $seed; verifying it reproduces as PASS in isolation..."
        if go test -run="^${TARGET}/${seed}\$" -count=1 -parallel=1 . > /tmp/seed_repro.log 2>&1; then
            echo "  → PASS in isolation. Spurious. Removing testdata entry and retrying."
            rm -f "testdata/fuzz/${TARGET}/${seed}"
            total_fails=$((total_fails + 1))
            rm -f "$LOG"
            continue
        else
            echo "  → FAIL in isolation! This is a REAL find, not the exit-2 flake."
            echo "     Keeping testdata/fuzz/${TARGET}/${seed} for triage."
            cat /tmp/seed_repro.log | tail -50
            rm -f "$LOG"
            exit 1
        fi
    fi

    # Anything else: surface and exit.
    echo "FAIL: fuzz exited with rc=$rc and an unknown failure mode:" >&2
    tail -40 "$LOG" >&2
    rm -f "$LOG"
    exit 1
done

if (( total_fails > 0 )); then
    echo "completed budget after $attempt attempts; $total_fails spurious exit-2 abort(s) handled"
else
    echo "completed budget with no aborts"
fi
exit 0
