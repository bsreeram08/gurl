class Gurl < Formula
  desc "Smart curl saver and API companion for the terminal"
  homepage "https://github.com/bsreeram08/gurl"
  version "0.1.22"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.1.22/gurl-darwin-arm64.tar.gz"
      sha256 "ae1b41013af3eb551ecbc3704ded78522392996cbbb8c56df9707a8dae84f0c0"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.1.22/gurl-linux-arm64.tar.gz"
      sha256 "ecfe0867a7e39969b81598389c10d9f9d138010fc2f13aabe4a0d227be969edb"
    elsif Hardware::CPU.intel?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.1.22/gurl-linux-amd64.tar.gz"
      sha256 "03dd1d80387bcd06bc3b8cc63575fd4f812d6598a6742035f3201022e277968c"
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
