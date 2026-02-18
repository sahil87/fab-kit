#!/usr/bin/env bash
#
# setup_suite.bash - Global setup for all idea test suites
#

setup_suite() {
    # Save original PATH
    export ORIGINAL_PATH="$PATH"

    # Add idea bin directory to PATH (production code lives in fab/.kit/packages/)
    REPO_ROOT="$(cd "${BATS_TEST_DIRNAME}/../../../.." && pwd)"
    export PATH="$REPO_ROOT/fab/.kit/packages/idea/bin:$PATH"

    # Create a temporary directory for test artifacts
    export BATS_SUITE_TMPDIR="${BATS_TEST_DIRNAME}/../.tmp"
    mkdir -p "$BATS_SUITE_TMPDIR"

    # Set up git user for all test repos
    git config --global user.name "Test User" 2>/dev/null || true
    git config --global user.email "test@example.com" 2>/dev/null || true
}
