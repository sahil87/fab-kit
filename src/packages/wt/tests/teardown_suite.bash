#!/usr/bin/env bash
#
# teardown_suite.bash - Global cleanup for all wt test suites
#

teardown_suite() {
    # Restore original PATH
    if [[ -n "${ORIGINAL_PATH:-}" ]]; then
        export PATH="$ORIGINAL_PATH"
    fi

    # Clean up temporary directory
    if [[ -d "${BATS_SUITE_TMPDIR:-}" ]]; then
        rm -rf "$BATS_SUITE_TMPDIR"
    fi

    # Clean up any leftover test repos in /tmp
    rm -rf /tmp/test-repo-* 2>/dev/null || true
}
