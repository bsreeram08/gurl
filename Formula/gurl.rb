class Gurl < Formula
  desc "Smart curl saver and API companion for the terminal"
  homepage "https://github.com/bsreeram08/gurl"
  version "0.3.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.3.0/gurl-darwin-arm64.tar.gz"
      sha256 "23092df6321e6d9e672b61985c46963e5ed30ad0589e55d56439a9d7aef56511"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.3.0/gurl-linux-arm64.tar.gz"
      sha256 "6b61bddcabe08eeedba8b5c75315fc9980879ef994f6c4901fc84c0964efe7af"
    elsif Hardware::CPU.intel?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.3.0/gurl-linux-amd64.tar.gz"
      sha256 "3ac9e488838e1ec733533bce2884d7f6f57374d1e0dcb3af0bc72f4aa9bae94e"
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
