class Gurl < Formula
  desc "Smart curl saver and API companion for the terminal"
  homepage "https://github.com/bsreeram08/gurl"
  version "0.4.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.4.0/gurl-darwin-arm64.tar.gz"
      sha256 "b31dd1110e12bbba43fd617730159959624cc45572f65a6e6d512b0eac28d8d4"
    elsif Hardware::CPU.intel?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.4.0/gurl-darwin-amd64.tar.gz"
      sha256 "5b359155fcff6e8dec17a4a7472931d52027462d8433b2e72ffbe8894feffec2"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.4.0/gurl-linux-arm64.tar.gz"
      sha256 "e96eb8afc478a0428b8cd1aa2a8161e9eb28b0a47fa4caf686f8d287454d455e"
    elsif Hardware::CPU.intel?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.4.0/gurl-linux-amd64.tar.gz"
      sha256 "6805b6a4a0f7b60e30af0f3f10ab802aa4fba79eca3a9bac4649e362a77fb682"
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
