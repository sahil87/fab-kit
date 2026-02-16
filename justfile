# Run all tests (bash + rust) with summary
test:
    #!/usr/bin/env bash
    set -euo pipefail
    just test-bash
    # just test-rust  # uncomment when Rust libs exist

# Run bash tests: bats (.bats) + legacy (test.sh)
test-bash:
    #!/usr/bin/env bash
    set -uo pipefail

    failed_suites=()
    passed_suites=()
    total=0

    # Run bats test suites
    for t in src/lib/*/test.bats; do
        [ -f "$t" ] || continue
        suite=$(basename "$(dirname "$t")")
        total=$((total + 1))
        echo "── ${suite} (bats) ──"
        if bats "$t"; then
            passed_suites+=("$suite")
        else
            failed_suites+=("$suite")
        fi
        echo ""
    done

    # Run legacy test.sh suites
    for t in src/lib/*/test.sh; do
        [ -f "$t" ] || continue
        suite=$(basename "$(dirname "$t")")
        total=$((total + 1))
        echo "── ${suite} (legacy) ──"
        if bash "$t"; then
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

# Run Rust tests (placeholder)
test-rust:
    @echo "No Rust tests yet."
