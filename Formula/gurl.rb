class Gurl < Formula
  desc "Smart curl saver and API companion for the terminal"
  homepage "https://github.com/bsreeram08/gurl"
  version "0.1.19"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.1.19/gurl-darwin-arm64.tar.gz"
      sha256 "a8286eae6f15d371e1db3f47c3840b2fd21ae4d433309cc4cf6955f0b46e09f1"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.1.19/gurl-linux-arm64.tar.gz"
      sha256 "43842f024cf3123ac67e5850c75d417fa06f803af1524133436d63ca56a70808"
    elsif Hardware::CPU.intel?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.1.19/gurl-linux-amd64.tar.gz"
      sha256 "f122a57ec7c035f3c8985e87bfafaab35fcba422969e6dbea394fcc565d7d5ae"
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
