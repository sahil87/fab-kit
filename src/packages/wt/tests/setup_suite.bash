#!/usr/bin/env bash
#
# setup_suite.bash - Global setup for all wt test suites
#

setup_suite() {
    # Save original PATH
    export ORIGINAL_PATH="$PATH"

    # Add fixtures directory to PATH so mock apps are found first
    export PATH="$BATS_TEST_DIRNAME/fixtures:$PATH"

    # Create a temporary directory for test artifacts
    export BATS_SUITE_TMPDIR="${BATS_TEST_DIRNAME}/../.tmp"
    mkdir -p "$BATS_SUITE_TMPDIR"

    # Create a global mock log file
    export MOCK_LOG="${BATS_SUITE_TMPDIR}/mock.log"
    touch "$MOCK_LOG"

    # Set up git user for all test repos
    git config --global user.name "Test User" 2>/dev/null || true
    git config --global user.email "test@example.com" 2>/dev/null || true

    # Create cache directory for wt commands
    mkdir -p ~/.cache/wt
}
