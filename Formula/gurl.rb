class Gurl < Formula
  desc "Smart curl saver and API companion for the terminal"
  homepage "https://github.com/bsreeram08/gurl"
  version "0.7.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.7.0/gurl-darwin-arm64.tar.gz"
      sha256 "8b9d4fb3d55f72f9d6afad3479fbaa4166e85dbc6e156666c7d57eaa11e8ec5c"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.7.0/gurl-linux-arm64.tar.gz"
      sha256 "b86be051751cc20d89ff1b56d91eae1700c52c94ffe651219bee0f3fad052242"
    elsif Hardware::CPU.intel?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.7.0/gurl-linux-amd64.tar.gz"
      sha256 "0956ff0f4ef70725bc40d9b9373a49b33a2166437c4d0f367e7cc8131908c48a"
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
