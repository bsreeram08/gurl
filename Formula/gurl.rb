class Gurl < Formula
  desc "Smart curl saver and API companion for the terminal"
  homepage "https://github.com/bsreeram08/gurl"
  version "0.7.2"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.7.2/gurl-darwin-arm64.tar.gz"
      sha256 "02fe952bf4d20dc3e0e1373149fe414f00025efac552c528451570ff3e2e275f"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.7.2/gurl-linux-arm64.tar.gz"
      sha256 "bc92ef6d7c2487d7dfbf418b434683830daeffdebf98f9851e6c093e572fe266"
    elsif Hardware::CPU.intel?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.7.2/gurl-linux-amd64.tar.gz"
      sha256 "3e0d8ed9a000f0b70292e14acb2a4cd38d69261235b4492bcbe9d80ff5371e6f"
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
