class Apigent < Formula
  desc "Saved API requests your AI agents can run without ever seeing the keys"
  homepage "https://github.com/bsreeram08/apigent"
  version "0.9.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/bsreeram08/apigent/releases/download/v0.9.0/apigent-darwin-arm64.tar.gz"
      sha256 "992ecb017507ed5b2cc0145d34512ae81ef92fcc8c2148a9cea8d361c8b88e0a"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/bsreeram08/apigent/releases/download/v0.9.0/apigent-linux-arm64.tar.gz"
      sha256 "0db60af01ef10a4acf7da81029a73fe084c644f1042f5b7dfaf91f34148d528a"
    elsif Hardware::CPU.intel?
      url "https://github.com/bsreeram08/apigent/releases/download/v0.9.0/apigent-linux-amd64.tar.gz"
      sha256 "d18470a8a740cae66c464cf0e385e4aa97e4a2d79cdaa6e6be2849bbd013b439"
    end
  end

  def install
    bin.install Dir["apigent-*"].first => "apigent"
    # The pre-rename command name keeps working.
    bin.install_symlink "apigent" => "gurl"
  end

  test do
    system "#{bin}/apigent", "--version"
  end
end
