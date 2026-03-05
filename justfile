scripts := "src/scripts/just"

# Run all tests (bash + rust) with summary
test:
    just test-setup
    just _test-parallel
    # just test-rust  # uncomment when Rust libs exist

# Run test suites in parallel with prefixed live output
_test-parallel:
    {{scripts}}/test-parallel.sh

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

go_src := "src/fab-go"

# Build fab Go binary for current platform
build-go:
    cd {{go_src}} && CGO_ENABLED=0 go build -o ../../fab/.kit/bin/fab ./cmd/fab

# Run Go parity tests (bash vs Go binary output)
test-go:
    cd {{go_src}} && go test ./test/parity/... -count=1

# Run Go parity tests with verbose output
test-go-v:
    cd {{go_src}} && go test ./test/parity/... -v -count=1

# Check prerequisites and environment health
doctor:
    fab/.kit/scripts/fab-doctor.sh
    @echo ""
    {{scripts}}/dev-doctor.sh
