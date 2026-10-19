class Gurl < Formula
  desc "Smart curl saver and API companion for the terminal"
  homepage "https://github.com/bsreeram08/gurl"
  version "0.5.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.5.0/gurl-darwin-arm64.tar.gz"
      sha256 "cf27627fa9c35fd5a05a077f0c999f8733bfa221ece2675de3dfeb585c291597"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.5.0/gurl-linux-arm64.tar.gz"
      sha256 "232eef5b650f0d9a41ad1a71bfb28fa32176cc6c5668963b378f7b40845fa29d"
    elsif Hardware::CPU.intel?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.5.0/gurl-linux-amd64.tar.gz"
      sha256 "49ac8528bb57c2b5fcefc60ae3e5a0f9f1b27f18963c08f90af12d61ea3ffac4"
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
