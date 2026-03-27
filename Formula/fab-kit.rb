class FabKit < Formula
  desc "Specification-driven development toolkit — shim, worktree manager, and backlog tool"
  homepage "https://github.com/wvrdz/fab-kit"
  url "https://github.com/wvrdz/fab-kit/archive/refs/tags/v0.42.0.tar.gz"
  # sha256 "UPDATE_ON_RELEASE"
  license "MIT"

  depends_on "go" => :build

  def install
    ldflags = "-X main.version=#{version}"

    # Build the fab shim (version-aware dispatcher).
    cd "src/go/shim" do
      system "go", "build", *std_go_args(ldflags:, output: bin/"fab"), "./cmd"
    end

    # Build wt (worktree management).
    cd "src/go/wt" do
      system "go", "build", *std_go_args(output: bin/"wt"), "./cmd"
    end

    # Build idea (backlog management).
    cd "src/go/idea" do
      system "go", "build", *std_go_args(output: bin/"idea"), "./cmd"
    end
  end

  test do
    assert_match "fab-shim", shell_output("#{bin}/fab --version")
    assert_match "wt", shell_output("#{bin}/wt --help 2>&1", 0).downcase
    assert_match "idea", shell_output("#{bin}/idea --help 2>&1", 0).downcase
  end
end
