#!/usr/bin/env bash
set -uo pipefail

failed_suites=()
passed_suites=()
total=0

for pkg_tests in src/packages/*/tests; do
    [ -d "$pkg_tests" ] || continue
    suite=$(basename "$(dirname "$pkg_tests")")

    # Collect all bats files for this package
    files=()
    for t in "$pkg_tests"/*.bats; do
        [ -f "$t" ] || continue
        files+=("$t")
    done
    [ ${#files[@]} -eq 0 ] && continue

    total=$((total + 1))
    echo "── ${suite} (${#files[@]} files) ──"
    # Run all files within a package in parallel
    if bats --jobs 8 --no-parallelize-within-files "${files[@]}"; then
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
    echo "No package tests found."
elif [ "$failed" -eq 0 ]; then
    echo "${passed}/${total} package tests passed     PASS"
else
    echo "${passed}/${total} package tests passed, ${failed} failed ($(IFS=', '; echo "${failed_suites[*]}"))     FAIL"
fi

[ "$failed" -eq 0 ]
