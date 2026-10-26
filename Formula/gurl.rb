class Gurl < Formula
  desc "Smart curl saver and API companion for the terminal"
  homepage "https://github.com/bsreeram08/gurl"
  version "0.8.1"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.8.1/gurl-darwin-arm64.tar.gz"
      sha256 "1499a3ade6716088014b4517e984bc6ccb98c98cd56d88c5b32baac7aa7787d3"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.8.1/gurl-linux-arm64.tar.gz"
      sha256 "fa066d622550eeb94ccec48a5825b5267a17ebdd18aeb56b00eedcb7025b7787"
    elsif Hardware::CPU.intel?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.8.1/gurl-linux-amd64.tar.gz"
      sha256 "881dfcb3c3119b44d586f97edc0ce0ef699874b08db3fd084a045ca352e8fef1"
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
