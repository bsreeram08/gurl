class Gurl < Formula
  desc "Smart curl saver and API companion for the terminal"
  homepage "https://github.com/bsreeram08/gurl"
  version "0.8.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.8.0/gurl-darwin-arm64.tar.gz"
      sha256 "69a5fd4ec85461b642b58c75e69adf03b73ce812bd011716f5b65304923b1481"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.8.0/gurl-linux-arm64.tar.gz"
      sha256 "11872e31bcada4c5ff3acfb088f7153d7683917ad91a61ee42cb6e057b5a6d32"
    elsif Hardware::CPU.intel?
      url "https://github.com/bsreeram08/gurl/releases/download/v0.8.0/gurl-linux-amd64.tar.gz"
      sha256 "1fcee3ceb82a14163e4dedc48f0e96426352239ef877f07b977444736ed004d2"
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
