class Gurl < Formula
  desc "Smart curl saver and API companion for the terminal"
  homepage "https://github.com/bsreeram08/gurl"
  version "0.1.17"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.1.17/gurl-darwin-arm64.tar.gz"
      sha256 "5a848115268f8269eb2b565507232af90902f976464bb8f02b15431afd056732"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.1.17/gurl-linux-arm64.tar.gz"
      sha256 "3c8a27b8d9af21f8f7ca99902002603f2f6db3ca310ec2c9c7a0984e6f93873f"
    elsif Hardware::CPU.intel?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.1.17/gurl-linux-amd64.tar.gz"
      sha256 "87975e5c0a81c470107ae8ca411de1b9130725b5b0a105fcb16f44aa715e5a55"
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
