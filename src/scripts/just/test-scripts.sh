#!/usr/bin/env bash
set -uo pipefail

failed_suites=()
passed_suites=()
total=0

for t in src/sh/*/test.bats; do
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
if [ "$total" -eq 0 ]; then
    echo "No script tests found."
elif [ "$failed" -eq 0 ]; then
    echo "${passed}/${total} script tests passed     PASS"
else
    echo "${passed}/${total} script tests passed, ${failed} failed ($(IFS=', '; echo "${failed_suites[*]}"))     FAIL"
fi

[ "$total" -eq 0 ] || [ "$failed" -eq 0 ]
