#!/usr/bin/env bash
#
# setup_suite.bash - Global setup for all idea test suites
#

setup_suite() {
    # Save original PATH
    export ORIGINAL_PATH="$PATH"

    # Add idea bin directory to PATH
    export PATH="${BATS_TEST_DIRNAME}/../bin:$PATH"

    # Create a temporary directory for test artifacts
    export BATS_SUITE_TMPDIR="${BATS_TEST_DIRNAME}/../.tmp"
    mkdir -p "$BATS_SUITE_TMPDIR"

    # Set up git user for all test repos
    git config --global user.name "Test User" 2>/dev/null || true
    git config --global user.email "test@example.com" 2>/dev/null || true
}
