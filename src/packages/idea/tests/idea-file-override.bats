#!/usr/bin/env bats
#
# idea-file-override.bats - Tests for --file flag and IDEAS_FILE env var
#

setup() {
    load '../../tests/libs/bats-support/load'
    load '../../tests/libs/bats-assert/load'
    load 'test_helper'
    _common_setup
}

teardown() {
    _common_teardown
}

# --- --file flag ---

@test "file-override: --file flag overrides default backlog location" {
    mkdir -p custom
    echo "- [ ] [zz99] 2025-07-01: Custom file idea" > custom/ideas.md

    run idea list --file custom/ideas.md
    assert_success
    assert_output --partial "zz99"
    assert_output --partial "Custom file idea"
}

# --- IDEAS_FILE env var ---

@test "file-override: IDEAS_FILE env var overrides default location" {
    mkdir -p my
    echo "- [ ] [yy88] 2025-07-01: Env var idea" > my/ideas.md

    IDEAS_FILE=my/ideas.md run idea list
    assert_success
    assert_output --partial "yy88"
    assert_output --partial "Env var idea"
}

# --- Precedence ---

@test "file-override: --file flag takes precedence over IDEAS_FILE env var" {
    mkdir -p env flag
    echo "- [ ] [env1] 2025-07-01: Env idea" > env/ideas.md
    echo "- [ ] [flg1] 2025-07-01: Flag idea" > flag/ideas.md

    IDEAS_FILE=env/ideas.md run idea list --file flag/ideas.md
    assert_success
    assert_output --partial "flg1"
    assert_output --partial "Flag idea"
    refute_output --partial "env1"
}
