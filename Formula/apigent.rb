class Apigent < Formula
  desc "Saved API requests your AI agents can run without ever seeing the keys"
  homepage "https://github.com/bsreeram08/apigent"
  version "0.9.1"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/bsreeram08/apigent/releases/download/v0.9.1/apigent-darwin-arm64.tar.gz"
      sha256 "cf8c2f5e1afe542ab1cfdafab1c5fba1ab8062d9478b232b0389fb143797aaf8"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/bsreeram08/apigent/releases/download/v0.9.1/apigent-linux-arm64.tar.gz"
      sha256 "f43477b7ddce6db78e1e2abc7ee8fcf250ad653a94792b6260ca6cefaaa973a1"
    elsif Hardware::CPU.intel?
      url "https://github.com/bsreeram08/apigent/releases/download/v0.9.1/apigent-linux-amd64.tar.gz"
      sha256 "40b0bbd1a3412060f6084474e79c5ef071251bd10e3822778a30f368913cee47"
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
