#!/usr/bin/env bash
set -uo pipefail

failed_suites=()
passed_suites=()
total=0

for pkg_tests in src/packages/*/tests; do
    [ -d "$pkg_tests" ] || continue
    suite=$(basename "$(dirname "$pkg_tests")")
    for t in "$pkg_tests"/*.bats; do
        [ -f "$t" ] || continue
        total=$((total + 1))
        test_name="${suite}/$(basename "$t" .bats)"
        echo "── ${test_name} ──"
        if bats "$t"; then
            passed_suites+=("$test_name")
        else
            failed_suites+=("$test_name")
        fi
        echo ""
    done
done

# Summary
passed=${#passed_suites[@]}
failed=${#failed_suites[@]}
echo "═══════════════════════════════════════════════════"
if [ "$total" -eq 0 ]; then
    echo "No package tests found."
elif [ "$failed" -eq 0 ]; then
    echo "${passed}/${total} package tests passed     PASS"
else
    echo "${passed}/${total} package tests passed, ${failed} failed ($(IFS=', '; echo "${failed_suites[*]}"))     FAIL"
fi

[ "$failed" -eq 0 ]
