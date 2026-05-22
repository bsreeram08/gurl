class Gurl < Formula
  desc "Smart curl saver and API companion for the terminal"
  homepage "https://github.com/bsreeram08/gurl"
  version "0.2.1"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.2.1/gurl-darwin-arm64.tar.gz"
      sha256 "471d175ce96591a04a41cd8ee139cb331ceed1b8965b4d84994e3abfcc9d8abc"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.2.1/gurl-linux-arm64.tar.gz"
      sha256 "14d40f28a0928406213146e1c7925fcc128b865e19a30fed55ecd4c94a6b3cb4"
    elsif Hardware::CPU.intel?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.2.1/gurl-linux-amd64.tar.gz"
      sha256 "20acaea0262b134cb3d0ed4a10683d18e83a2208e6f7a0ddad46e516ee5fb7b7"
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
