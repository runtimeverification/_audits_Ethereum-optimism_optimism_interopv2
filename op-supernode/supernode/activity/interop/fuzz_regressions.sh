#!/usr/bin/env bash
# fuzz_regressions.sh — run the 5 bug-injection regressions for the
# op-supernode interop fuzz harness and confirm each oracle still fires.
#
# Each bug-inject patch was originally applied by hand during the
# harness's development to validate that the oracle catches a broken
# SUT. The patches are sed-driven and have specific anchors; if the
# anchors fail to match (source moved) the script aborts loudly rather
# than silently passing.
#
# Files are snapshotted under /tmp and restored unconditionally on
# script exit, even on Ctrl-C, so a broken patch can't leave the tree
# in an inconsistent state.
#
# Usage (from this directory):
#   ./fuzz_regressions.sh         # run all 5
#   ./fuzz_regressions.sh expiry  # run a single one by short name
#
# Exit code: 0 if every selected oracle catches its bug; 1 otherwise.

set -euo pipefail

cd "$(dirname "$0")"

# Safety check: refuse to run while a `go test -fuzz=` is active on this
# package. Concurrent compiles on the same package can knock a fuzz worker
# offline with the "fuzzing process hung or terminated: exit status 2"
# pattern — same shape as the pre-existing 573eddeb1ebaf835 flake. Better
# to fail fast here than corrupt the long-running fuzz state.
if pgrep -f 'fuzz=\\\^FuzzVerifyInteropMessages' >/dev/null 2>&1; then
    echo "FAIL: a 'go test -fuzz' is running on FuzzVerifyInteropMessages." >&2
    echo "Wait for it to finish (or kill it) before running this script." >&2
    exit 3
fi

SNAPSHOT_DIR="$(mktemp -d -t interop-fuzz-regression-XXXXXX)"
declare -A SNAPSHOTTED=()

snapshot() {
    local file=$1
    if [[ -z "${SNAPSHOTTED[$file]:-}" ]]; then
        cp "$file" "$SNAPSHOT_DIR/$(basename "$file")"
        SNAPSHOTTED[$file]=1
    fi
}

restore_all() {
    local file
    for file in "${!SNAPSHOTTED[@]}"; do
        cp "$SNAPSHOT_DIR/$(basename "$file")" "$file"
    done
    rm -rf "$SNAPSHOT_DIR"
}
trap restore_all EXIT INT TERM

require_match() {
    # Anchors: file pattern_grep_arg
    local file=$1
    shift
    if ! grep -q "$@" "$file"; then
        echo "FAIL: $1 pattern not found in $file — source moved, update fuzz_regressions.sh anchors" >&2
        exit 2
    fi
}

run_target_and_count_fails() {
    local target=$1
    local log
    log=$(mktemp)
    # Use the existing testdata corpus; -count=1 disables go's test result cache.
    local rc
    go test -run="^${target}\$" -count=1 . > "$log" 2>&1 && rc=0 || rc=$?
    # Detect compile failures so they don't look like "oracle did not fire".
    # `go test` emits "FAIL\tpackage [build failed]" on compile error.
    if grep -qE '^FAIL[[:space:]].*\[build failed\]$|cannot find package|cannot use|undefined:' "$log"; then
        echo "BUILDFAIL: patch produced uncompilable Go — see log $log" >&2
        cat "$log" >&2
        # Don't rm the log; the caller may want to inspect it.
        echo "-1"
        return
    fi
    local fails
    fails=$(grep -cE '^[[:space:]]*--- FAIL:' "$log" || true)
    rm -f "$log"
    echo "$fails"
}

inject_and_check() {
    local name=$1
    local file=$2
    local target=$3
    local min_fails=$4
    local sed_script=$5

    echo "=== regression: $name ==="
    snapshot "$file"

    # Apply the patch.
    if ! sed -i "$sed_script" "$file"; then
        echo "FAIL: $name — sed script failed on $file" >&2
        return 1
    fi

    # Confirm the patch actually changed something.
    if cmp -s "$file" "$SNAPSHOT_DIR/$(basename "$file")"; then
        echo "FAIL: $name — sed script matched but produced no change in $file" >&2
        return 1
    fi

    echo "  patched, running $target..."
    local fails
    fails=$(run_target_and_count_fails "$target")
    echo "  observed --- FAIL: count = $fails (min required: $min_fails)"

    # Restore immediately so a later regression sees clean source.
    cp "$SNAPSHOT_DIR/$(basename "$file")" "$file"

    if [[ "$fails" == "-1" ]]; then
        echo "FAIL: $name — patched source did not compile (see BUILDFAIL output above)" >&2
        return 1
    fi
    if [[ "$fails" -lt "$min_fails" ]]; then
        echo "FAIL: $name — oracle did not catch the injected bug (saw $fails, expected >= $min_fails)" >&2
        return 1
    fi
    echo "PASS: $name"
    return 0
}

ALL_NAMES=(expiry frontier-checksum verified-block-at-l1 same-l1-chain commit-idempotent)
SELECTED=("${@:-${ALL_NAMES[@]}}")
FAILED=()

# Pre-flight: confirm every anchor is present, before mutating anything.
require_match algo.go -F 'if execMsg.Timestamp+i.messageExpiryWindow < executingTimestamp {'
require_match verification_view.go -F 'seal, ok := block.contains[frontierQueryKey{'
require_match interop.go -F 'func (i *Interop) VerifiedBlockAtL1(chainID eth.ChainID, l1Block eth.L1BlockRef) (eth.BlockID, uint64) {'
require_match checker.go -F 'func (c *l1ByNumberChecker) SameL1Chain(ctx context.Context, heads []eth.BlockID) (bool, error) {'
require_match verified_db.go -F 'if reflect.DeepEqual(existing, result) {'
require_match chain_fuzz_utils.go -F 'mode := rc.randomInvalidIdentifierMode()'

for name in "${SELECTED[@]}"; do
    case "$name" in
    expiry)
        # Remove the expiry-check `if` block in algo.go (3 lines).
        inject_and_check expiry algo.go FuzzVerifyInteropMessages 1 \
            '/if execMsg\.Timestamp+i\.messageExpiryWindow < executingTimestamp {/,/^[[:space:]]*}$/d' \
            || FAILED+=("$name")
        ;;
    frontier-checksum)
        # Two patches:
        #   (a) chain_fuzz_utils.go: force the InvalidIdentifier sub-mode to
        #       5 (wrong-checksum) so every InvalidIdentifier seed exercises
        #       the frontier-view path. Without this, only ~1/6 of those
        #       seeds happen to pick mode=5 and the oracle may not fire
        #       deterministically on small corpora.
        #   (b) verification_view.go: replace the checksum-keyed map lookup
        #       with a loop that ignores checksum, so the bogus message is
        #       wrongly accepted by the frontier view.
        # We snapshot both files up-front; the trap restores both on exit.
        snapshot chain_fuzz_utils.go
        snapshot verification_view.go
        if ! sed -i '/mode := rc.randomInvalidIdentifierMode()/a\
\	mode = 5 // FORCE for fuzz_regressions.sh frontier-checksum' chain_fuzz_utils.go; then
            echo "FAIL: frontier-checksum mode-force sed failed" >&2
            FAILED+=("$name")
        elif cmp -s chain_fuzz_utils.go "$SNAPSHOT_DIR/chain_fuzz_utils.go"; then
            echo "FAIL: frontier-checksum mode-force sed matched but no change" >&2
            FAILED+=("$name")
        elif ! sed -i '/seal, ok := block.contains\[frontierQueryKey{/,/return seal, ok/c\
\	for k, vv := range block.contains {\
\		if k.blockNum == query.BlockNum \&\& k.timestamp == query.Timestamp \&\& k.logIdx == query.LogIdx {\
\			return vv, true\
\		}\
\	}\
\	return suptypes.BlockSeal{}, false' verification_view.go; then
            echo "FAIL: frontier-checksum view sed failed" >&2
            FAILED+=("$name")
        else
            echo "=== regression: frontier-checksum (mode=5 forced + checksum bypassed) ==="
            echo "  patched, running FuzzVerifyInteropMessages..."
            FAILS=$(run_target_and_count_fails FuzzVerifyInteropMessages)
            echo "  observed --- FAIL: count = $FAILS (min required: 1)"
            cp "$SNAPSHOT_DIR/chain_fuzz_utils.go" chain_fuzz_utils.go
            cp "$SNAPSHOT_DIR/verification_view.go" verification_view.go
            if [[ "$FAILS" == "-1" ]]; then
                echo "FAIL: frontier-checksum — patched source did not compile" >&2
                FAILED+=("$name")
            elif [[ "$FAILS" -lt 1 ]]; then
                echo "FAIL: frontier-checksum — oracle did not catch the injected bug" >&2
                FAILED+=("$name")
            else
                echo "PASS: frontier-checksum"
            fi
        fi
        ;;
    verified-block-at-l1)
        # Insert an immediate zero return at the top of VerifiedBlockAtL1.
        inject_and_check verified-block-at-l1 interop.go FuzzVerifyInteropMessages 1 \
            '/^func (i \*Interop) VerifiedBlockAtL1(chainID eth\.ChainID, l1Block eth\.L1BlockRef) (eth\.BlockID, uint64) {/a\
\	return eth.BlockID{}, 0' \
            || FAILED+=("$name")
        ;;
    same-l1-chain)
        # Insert an immediate "consistent" return at the top of SameL1Chain.
        inject_and_check same-l1-chain checker.go FuzzVerifyInteropMessages 1 \
            '/^func (c \*l1ByNumberChecker) SameL1Chain(ctx context\.Context, heads \[\]eth\.BlockID) (bool, error) {/a\
\	return true, nil' \
            || FAILED+=("$name")
        ;;
    commit-idempotent)
        # Replace the DeepEqual idempotent-replay clause with a hard ErrAlreadyCommitted.
        inject_and_check commit-idempotent verified_db.go FuzzVerifyInteropMessagesCrashRecover 1 \
            '/if reflect\.DeepEqual(existing, result) {/,/^[[:space:]]*}$/c\
\			_ = reflect.DeepEqual\
\			return fmt.Errorf("%w: %d", ErrAlreadyCommitted, ts)' \
            || FAILED+=("$name")
        ;;
    *)
        echo "unknown regression name: $name (valid: ${ALL_NAMES[*]})" >&2
        exit 2
        ;;
    esac
done

echo
if [[ "${#FAILED[@]}" -eq 0 ]]; then
    echo "ALL ${#SELECTED[@]} regressions PASSED"
    exit 0
else
    echo "FAILED regressions: ${FAILED[*]}"
    exit 1
fi
