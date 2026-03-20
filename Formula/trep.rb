# This formula is generated automatically by goreleaser on each tagged release.
# To install the latest release:
#
#   brew tap Yanujz/trep https://github.com/Yanujz/trep
#   brew install trep
#
# To upgrade:
#   brew upgrade trep
#
# After the first tagged release the sha256 checksums below will be filled in by
# the release workflow.  Until then, build from source:
#
#   go install github.com/trep-dev/trep/cmd/trep@latest

class Trep < Formula
  desc "Convert test results and coverage data to rich self-contained HTML reports"
  homepage "https://github.com/Yanujz/trep"
  license "MIT"
  version "0.2.0"

  on_macos do
    on_arm do
      url "https://github.com/Yanujz/trep/releases/download/v#{version}/trep_darwin_arm64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
    on_intel do
      url "https://github.com/Yanujz/trep/releases/download/v#{version}/trep_darwin_amd64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Yanujz/trep/releases/download/v#{version}/trep_linux_arm64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
    on_intel do
      url "https://github.com/Yanujz/trep/releases/download/v#{version}/trep_linux_amd64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
  end

  def install
    bin.install "trep"
  end

  test do
    system "#{bin}/trep", "--version"
  end
end
