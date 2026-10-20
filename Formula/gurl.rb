class Gurl < Formula
  desc "Smart curl saver and API companion for the terminal"
  homepage "https://github.com/bsreeram08/gurl"
  version "0.7.1"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.7.1/gurl-darwin-arm64.tar.gz"
      sha256 "41b82bb54ee27dcb0b9f28022c112089c89d81a3a07b4aa0e594aa1a17a25963"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.7.1/gurl-linux-arm64.tar.gz"
      sha256 "d28fbdbacfa965ffff6063a7a263075fc2f4ced195510cca9bbb2fe2d7829e8b"
    elsif Hardware::CPU.intel?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.7.1/gurl-linux-amd64.tar.gz"
      sha256 "aa668c5ca824d017cf07b3ac98e3f60bc9522f01c2873ddde44b79855d72f2db"
    end
  end

  def install
    bin.install Dir["gurl-*"].first => "gurl"
  end

  def post_install
    (var/"gurl").mkpath
  end

  test do
    system "#{bin}/gurl", "--version"
  end
end
