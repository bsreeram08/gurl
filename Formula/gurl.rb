class Gurl < Formula
  desc "Smart curl saver and API companion for the terminal"
  homepage "https://github.com/bsreeram08/gurl"
  version "0.2.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.2.0/gurl-darwin-arm64.tar.gz"
      sha256 "7d65616b0f4f05a450bea570b3e1d4b36a3a41954264a72c177f0077409600eb"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.2.0/gurl-linux-arm64.tar.gz"
      sha256 "362436f3724e97db41d8a3a729ca946edc275b47c676abb0be83b947a45896a3"
    elsif Hardware::CPU.intel?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.2.0/gurl-linux-amd64.tar.gz"
      sha256 "d398bd36a5bfca375124c48168eede5ec5175ed7a226b11fb3762a27e294666c"
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
