class Gurl < Formula
  desc "Smart curl saver and API companion for the terminal"
  homepage "https://github.com/bsreeram08/gurl"
  version "0.1.18"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.1.18/gurl-darwin-arm64.tar.gz"
      sha256 "fa1df524eda7d206ca7cf940d41d7a6010ba1f5d22346938585a5f70ef27420d"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.1.18/gurl-linux-arm64.tar.gz"
      sha256 "140eb9e4eadfb6f17b6e393b889549a2ace990a9f0d3b304f9c2c81cc4735fee"
    elsif Hardware::CPU.intel?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.1.18/gurl-linux-amd64.tar.gz"
      sha256 "488f08e0429322450b18f8a40cc6c6ae12e4744f592099544bc7671eb03b3c4b"
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
