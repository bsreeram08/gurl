class Gurl < Formula
  desc "Smart curl saver and API companion for the terminal"
  homepage "https://github.com/bsreeram08/gurl"
  version "0.2.2"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.2.2/gurl-darwin-arm64.tar.gz"
      sha256 "1f339cbb93f16072498f435d34e126a76d29ead206c468a217409109c780f1f8"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.2.2/gurl-linux-arm64.tar.gz"
      sha256 "8b118fcc6d8d20f14cb3e05e6ae4603672bb9623ee1d0532235bad3fc8e14aab"
    elsif Hardware::CPU.intel?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.2.2/gurl-linux-amd64.tar.gz"
      sha256 "dec3f68091b3f438b74c2338d65402f0e24174c2823c4e44598bc2fc5b2bb55d"
    end
  end

  def install
    bin.install "gurl"
  end

  def post_install
    (var/"gurl").mkpath
  end

  test do
    system "#{bin}/gurl", "--version"
  end
end
