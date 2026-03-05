class Quack < Formula
  desc "Terminal UI for OpenCode sessions"
  homepage "https://github.com/SmolNero/quack"
  url "https://github.com/SmolNero/quack/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "0927c518509c311f737e0b6759e8237cc8c0fa5badd0c28b21898576b5840cbf"
  license :cannot_represent

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args, "./cmd/quack"
  end

  test do
    output = shell_output("#{bin}/quack 2>&1", 1)
    assert_match "could not open a new TTY", output
  end
end
