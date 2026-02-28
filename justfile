scripts := "src/scripts/just"

# Run all tests (bash + rust) with summary
test:
    just test-setup
    just test-bash
    just test-packages
    just test-scripts
    # just test-rust  # uncomment when Rust libs exist

# Setup test dependencies
test-setup:
    {{scripts}}/test-setup.sh

# Run bash tests (bats)
test-bash:
    {{scripts}}/test-bash.sh

# Run package tests (bats)
test-packages:
    {{scripts}}/test-packages.sh

# Run script tests (bats)
test-scripts:
    {{scripts}}/test-scripts.sh

# Run Rust tests (placeholder)
test-rust:
    @echo "No Rust tests yet."
