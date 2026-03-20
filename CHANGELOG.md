# Changelog

All notable changes to this project are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

---

## [0.1.0] — 2026-03-20

First public release.

### Added

**Core**
- `trep test` — parse JUnit XML, Google Test XML, `go test -json`, and TAP v12/13 into a self-contained HTML report
- `trep cov` — parse LCOV, Go coverprofile, Cobertura, and Clover coverage data into an HTML report
- `trep report` — combined command producing two cross-linked HTML pages (tests + coverage) in one run
- `trep completion` — generate shell completion scripts for bash, zsh, fish, and PowerShell

**HTML reports**
- Fully self-contained single-file output (no external dependencies)
- Grouped and flat views with click-to-sort columns
- Debounced search across suite/test names; pass/fail/skip filter buttons
- Pagination (200 tests / 50 suites per page)
- Slow-test highlighting (top 10% of duration in current view)
- File/line source links for Google Test XML inputs
- Console output toggle for captured `stdout`/`stderr` on failures

**Coverage reports**
- Line, branch, and function coverage metrics
- Per-file table with coverage bars and configurable thresholds
- Threshold markers (`--threshold-line`, `--threshold-branch`, `--threshold-func`) with `--fail` exit code support
- `--strip-prefix` to clean up absolute paths from CI runners
- `--exclude` glob patterns to omit generated/vendor files from the report

**Output formats**
- `--output-format html` (default) — self-contained HTML
- `--output-format json` — structured JSON for dashboard ingestion and `jq` scripting
- `--output-format sarif` — SARIF 2.1.0 for GitHub Advanced Security code scanning

**Delta / baseline comparison**
- `--save-snapshot` / `--baseline` — persist a run snapshot and compare future runs against it
- Delta badges in HTML reports: pass/fail/skip counts and coverage % change
- Per-file coverage delta badge in coverage reports

**CI integration**
- `--annotate` — emit GitHub Actions or GitLab CI annotation lines for failed tests / low-coverage files
- `--annotate-platform` — explicit platform selection (`auto` / `github` / `gitlab`)
- `--fail`, `--fail-tests`, `--fail-cov` — non-zero exit when thresholds are not met
- `--quiet` — suppress progress output in pipelines

**Distribution**
- Pre-built binaries for Linux (amd64/arm64), macOS (Intel/Apple Silicon), and Windows (amd64) via GitHub Releases
- Homebrew formula (`brew tap Yanujz/trep https://github.com/Yanujz/trep && brew install trep`)
- `go install github.com/trep-dev/trep/cmd/trep@latest`

[Unreleased]: https://github.com/Yanujz/trep/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/Yanujz/trep/releases/tag/v0.1.0
