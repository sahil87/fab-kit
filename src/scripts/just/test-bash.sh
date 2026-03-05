#!/usr/bin/env bash
set -uo pipefail

failed_suites=()
passed_suites=()
total=0

for t in src/lib/*/test.bats src/hooks/test-*.bats src/sync/test-*.bats; do
    [ -f "$t" ] || continue
    suite=$(basename "$(dirname "$t")")
    total=$((total + 1))
    echo "── ${suite} ──"
    if bats "$t"; then
        passed_suites+=("$suite")
    else
        failed_suites+=("$suite")
    fi
    echo ""
done

# Summary
passed=${#passed_suites[@]}
failed=${#failed_suites[@]}
echo "═══════════════════════════════════════════════════"
if [ "$failed" -eq 0 ]; then
    echo "${passed}/${total} suites passed     PASS"
else
    echo "${passed}/${total} suites passed, ${failed} failed ($(IFS=', '; echo "${failed_suites[*]}"))     FAIL"
fi

[ "$failed" -eq 0 ]
