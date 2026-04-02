# Homebrew formula for fab-kit
# Installs three binaries: fab (shim), wt (worktree mgmt), idea (backlog mgmt)
#
# Usage:
#   brew tap wvrdz/tap
#   brew install fab-kit
#
# This formula builds from source. Pre-built bottles may be added later.

class FabKit < Formula
  desc "Structured development workflow for AI agents"
  homepage "https://github.com/sahil87/fab-kit"
  url "https://github.com/sahil87/fab-kit/archive/refs/tags/v0.43.1.tar.gz"
  sha256 ""  # TODO: populate on release
  license "MIT"

  depends_on "go" => :build

  def install
    # Build shim (fab)
    cd "src/go/shim" do
      ldflags = "-X main.version=#{version}"
      system "go", "build", *std_go_args(ldflags: ldflags, output: bin/"fab"), "./cmd"
    end

    # Build wt
    cd "src/go/wt" do
      system "go", "build", *std_go_args(output: bin/"wt"), "./cmd"
    end

    # Build idea
    cd "src/go/idea" do
      system "go", "build", *std_go_args(output: bin/"idea"), "./cmd"
    end
  end

  test do
    assert_match "fab #{version}", shell_output("#{bin}/fab --version")
  end
end
