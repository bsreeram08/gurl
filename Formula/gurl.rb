class Gurl < Formula
  desc "Smart curl saver and API companion for the terminal"
  homepage "https://github.com/bsreeram08/gurl"
  version "0.6.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.6.0/gurl-darwin-arm64.tar.gz"
      sha256 "00f044161b2152e945910211aa451b48fb5f73cbb286ed7f16f09c9dd086be1a"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.6.0/gurl-linux-arm64.tar.gz"
      sha256 "6260b3f3a6f311a815a93b1ba1d735a272af5a1180e8a805fe2e0df88a49c54b"
    elsif Hardware::CPU.intel?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.6.0/gurl-linux-amd64.tar.gz"
      sha256 "558a4fcb3168ebf46d58de3b06f0a7dccc7157e6af8fa49c9c10b14fb53c02cf"
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
