# This formula is maintained in the trep repository.
# To install:
#
#   brew tap Yanujz/trep https://github.com/Yanujz/trep
#   brew install trep
#
# To upgrade:
#   brew upgrade trep

class Trep < Formula
  desc "Convert test results and coverage data to rich self-contained HTML reports"
  homepage "https://github.com/Yanujz/trep"
  license "MIT"
  version "0.1.0"

  on_macos do
    on_arm do
      url "https://github.com/Yanujz/trep/releases/download/v#{version}/trep_darwin_arm64.tar.gz"
      sha256 "8ed6f4461f0de34f89e3391d1660cf3aeed13854b69d447fb0f5adecdaf1cab0"
    end
    on_intel do
      url "https://github.com/Yanujz/trep/releases/download/v#{version}/trep_darwin_amd64.tar.gz"
      sha256 "3bd2ff525da64a7d42e0cc5a0f131fdf79e64d199eb9999f818cea651f70c453"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Yanujz/trep/releases/download/v#{version}/trep_linux_arm64.tar.gz"
      sha256 "dba4ed0f0cdc524b480dce9cac16fc81f43533f7224bc0995d11a5ee30da1a95"
    end
    on_intel do
      url "https://github.com/Yanujz/trep/releases/download/v#{version}/trep_linux_amd64.tar.gz"
      sha256 "56121a6e584110601868702a29652134bada71391730882b4da4134e8a3cb9ff"
    end
  end

  def install
    bin.install "trep"
  end

  test do
    system "#{bin}/trep", "--version"
  end
end
