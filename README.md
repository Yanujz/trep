# trep — Test Report Generator

[![Go Report Card](https://goreportcard.com/badge/github.com/Yanujz/trep)](https://goreportcard.com/report/github.com/Yanujz/trep)
[![Go Reference](https://pkg.go.dev/badge/github.com/Yanujz/trep.svg)](https://pkg.go.dev/github.com/Yanujz/trep)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

`trep` converts test result files into a **self-contained, searchable HTML report** with no
external runtime dependencies. Drop it into any CI pipeline and get a clean, filterable
view of your test run in one portable file.

---

## Features

- **Multi-format input** — JUnit XML, Google Test XML, `go test -json`, TAP 12/13
- **Auto-detection** — extension + content sniffing; no `--format` needed in most cases
- **Multi-file merge** — combine N result files into one unified report (or keep them separate with `--no-merge`)
- **Self-contained HTML** — zero external dependencies; one file you can attach to a PR or email
- **Grouped + flat views** — collapsible suite groups; click any header to sort flat
- **Live search & filter** — debounced search across suite/test name; pass/fail/skip filter buttons
- **Pagination** — 200 tests / 50 suites per page; smart page-range widget
- **File/line links** — Google Test XML `file` and `line` attrs are preserved and shown inline
- **Slow test highlighting** — tests in the top 10% of duration are highlighted
- **CI integration** — `--fail` exits 1 when any test failed; `--quiet` suppresses noise
- **Browser launch** — `--open` opens the report immediately after writing

---

## Installation

```sh
git clone https://github.com/Yanujz/trep
cd trep
go build -o trep ./cmd/trep/
# Optional: put on PATH
mv trep /usr/local/bin/
```

Requires **Go 1.21+**.

---

## Usage

```
trep <command> [flags] [args]

Commands:
  test      Parse test results → HTML report
  cov       Parse coverage data → HTML report
  report    Parse both → two linked HTML pages
```

### `trep test` flags

| Flag | Short | Default | Description |
|---|---|---|---|
| `--output` | `-o` | `<input>.html` | Output file; `-` for stdout |
| `--output-format` | | `html` | Output format: `html` · `json` · `sarif` |
| `--title` | `-t` | derived | Report title |
| `--no-merge` | | `false` | One report per input instead of merging |
| `--fail` | | `false` | Exit 1 when any tests failed |
| `--open` | | `false` | Open in browser after writing |
| `--quiet` | `-q` | `false` | Suppress stderr output |
| `--annotate` | | `false` | Emit CI annotations for failed tests (GitHub/GitLab auto-detected) |
| `--annotate-platform` | | `auto` | Annotation platform: `auto` · `github` · `gitlab` |
| `--save-snapshot` | | | Write JSON snapshot for future delta comparison |
| `--baseline` | | | JSON snapshot from a previous run (enables delta badges) |
| `--baseline-label` | | | Human label for the baseline (e.g. `main`) |

### `trep cov` flags

| Flag | Short | Default | Description |
|---|---|---|---|
| `--output` | `-o` | `<input>.html` | Output file; `-` for stdout |
| `--output-format` | | `html` | Output format: `html` · `json` · `sarif` |
| `--format` | `-f` | `auto` | Force input format: `lcov` · `gocover` · `cobertura` · `clover` |
| `--title` | `-t` | derived | Report title |
| `--threshold` | | `0` (off) | Minimum line coverage %; alias for `--threshold-line` |
| `--threshold-line` | | `0` (off) | Minimum line coverage %; draws red marker |
| `--threshold-branch` | | `0` (off) | Minimum branch coverage % |
| `--threshold-func` | | `0` (off) | Minimum function coverage % |
| `--fail` | | `false` | Exit 1 if any enabled threshold is not met |
| `--strip-prefix` | | | Remove path prefix from all file paths |
| `--exclude` | | | Glob patterns to exclude from the report (repeatable; `vendor/**` excludes a directory tree) |
| `--open` | | `false` | Open in browser after writing |
| `--quiet` | `-q` | `false` | Suppress stderr output |
| `--annotate` | | `false` | Emit CI annotations for files below threshold |
| `--annotate-platform` | | `auto` | Annotation platform: `auto` · `github` · `gitlab` |
| `--save-snapshot` | | | Write JSON snapshot |
| `--baseline` | | | Previous snapshot (enables delta badges) |
| `--baseline-label` | | | Human label for the baseline |

### `trep report` flags

| Flag | Short | Default | Description |
|---|---|---|---|
| `--tests` | | required | Test result file(s) |
| `--cov` | | required | Coverage file |
| `--format-test` | | `auto` | Force test format |
| `--format-cov` | | `auto` | Force coverage format |
| `--output-dir` | | `.` | Directory to write both files into |
| `--prefix` | | `report` | Filename prefix (`report` → `report_tests.html` + `report_cov.html`) |
| `--title` | `-t` | | Title applied to both pages |
| `--threshold` | | `0` | Minimum line coverage % for `--fail-cov` |
| `--fail-tests` | | `false` | Exit 1 if any tests failed |
| `--fail-cov` | | `false` | Exit 1 if coverage below threshold |
| `--strip-prefix` | | | Remove path prefix from coverage file paths |
| `--open` | | `false` | Open both reports in browser |
| `--quiet` | `-q` | `false` | Suppress stderr output |
| `--annotate` | | `false` | Emit CI annotations for failures and low-coverage files |
| `--annotate-platform` | | `auto` | Annotation platform: `auto` · `github` · `gitlab` |
| `--save-snapshot` | | | Write combined JSON snapshot |
| `--baseline` | | | Previous snapshot |
| `--baseline-label` | | | Baseline label |

### Format values

| Value | Aliases | Input type |
|---|---|---|
| `junit` | `ctest`, `gtest`, `maven`, `xml` | JUnit / Google Test XML |
| `gotest` | `go`, `json` | `go test -json` streaming output |
| `tap` | | TAP v12/13 |
| `auto` | _(default)_ | Detect from extension then content |

---

## Examples

**Test results only:**
```sh
trep test results.xml
trep test -o report.html -t "PR #42" a.xml b.xml c.xml
go test -json ./... | trep test - -t "Unit Tests"
trep test --fail -q results.xml                         # CI: exit 1 on failure
trep test --no-merge suite1.xml suite2.xml              # → suite1.html + suite2_02.html
```

**Coverage only:**
```sh
trep cov coverage.out                                   # Go coverprofile
trep cov coverage.info                                  # LCOV (gcov / Istanbul)
trep cov coverage.xml                                   # Cobertura / JaCoCo
trep cov --threshold 80 --fail coverage.out             # CI: enforce 80% line coverage
trep cov --strip-prefix /home/runner/work/repo/ cov.info
```

**Delta badges (compare two runs):**
```sh
# Run 1 — save snapshot
trep test --save-snapshot snap.json results.xml

# Run 2 — compare against saved snapshot
trep test --baseline snap.json --baseline-label "main" results_new.xml
# → delta badges show +3 pass / -1 fail / +2.1% coverage etc.
```

**Combined linked report (two pages, shared nav):**
```sh
trep report --tests results.xml --cov coverage.out
# → report_tests.html + report_cov.html (cross-linked via nav bar)

trep report \
  --tests unit.xml integration.xml \
  --cov coverage.info \
  --threshold 80 --fail-tests --fail-cov \
  --output-dir dist/ --prefix nightly \
  --title "Nightly CI" \
  --save-snapshot snap.json
```

---

## Input Format Details

### JUnit XML / Google Test XML

The standard `<testsuites>` / `<testsuite>` / `<testcase>` schema. Both formats share
the same parser — Google Test XML is a superset that adds `file` and `line` attributes
to `<testcase>` elements, which `trep` preserves and renders.

CTest's synthetic `SKIP_REGULAR_EXPRESSION_MATCHED` skip messages are automatically
replaced with the human-readable reason extracted from `<system-out>`.

Compatible with: CTest, Maven Surefire, pytest-junit, JUnit 4/5, NUnit, xUnit.

### go test -json

Produced by `go test -json ./...`. The parser processes the streaming event log,
accumulates per-test output, and extracts meaningful failure messages while stripping
boilerplate `=== RUN` / `--- FAIL` header lines.

### TAP (Test Anything Protocol) v12/13

Line-oriented `ok` / `not ok` records. Supports:
- Optional `# time=N.NNN` annotations (accumulated into report duration)
- `# SKIP <reason>` pragmas
- TAP 13 embedded YAML diagnostic blocks (skipped cleanly)

---

## Project Structure

```
trep/
├── cmd/trep/
│   ├── main.go          # CLI entry point, Cobra root, browser launch helpers
│   ├── cmd_tests.go     # trep test subcommand
│   ├── cmd_cov.go       # trep cov subcommand
│   └── cmd_report.go    # trep report subcommand
└── pkg/
    ├── model/
    │   └── report.go    # Format-agnostic Report / Suite / TestCase types
    ├── parser/
    │   ├── parser.go    # Parser interface, registry, auto-detection
    │   ├── junit/       # JUnit XML + Google Test XML (streaming)
    │   ├── gotest/      # go test -json
    │   └── tap/         # TAP v12/13
    ├── coverage/
    │   ├── model/       # CovReport / FileCov types
    │   ├── parser/      # CovParser interface + lcov / gocover / cobertura / clover
    │   └── render/html/ # Coverage HTML renderer
    ├── delta/           # Snapshot save/load and run-over-run delta computation
    └── render/
        ├── html/        # Test HTML renderer
        ├── json/        # Structured JSON output
        └── annotations/ # GitHub / GitLab CI annotation lines
```

Adding a new test format is two steps:
1. Create `pkg/parser/<name>/parser.go` implementing `parser.Parser`
2. Add a blank import in `cmd/trep/main.go` — the `init()` self-registers it

---

## HTML Report Features

The generated report is fully self-contained — a single `.html` file with no external
scripts, fonts, or stylesheets.

- **Grouped view** (default): suites collapse/expand on click; each suite shows a pass/fail badge
- **Flat view**: switch with the toggle button; supports click-to-sort on all columns
- **Search**: debounced 200 ms; matches suite name and test name simultaneously
- **Filter buttons**: All / Passed / Failed / Skipped
- **Slow test detection**: tests exceeding 10% of the max duration in the current view are highlighted
- **Console output**: failures with captured stdout show a "Show Console Output" toggle revealing a dark-themed pre block
- **File/line**: Google Test source locations rendered under the test name

---


## CI Integration

### GitHub Actions

```yaml
- name: Generate test + coverage report
  run: |
    trep report \
      --tests build/test-results.xml \
      --cov build/coverage.out \
      --threshold 80 \
      --fail-tests --fail-cov \
      --annotate \
      --output-dir dist/ \
      --prefix ci \
      --baseline .trep/baseline.json \
      --baseline-label ${{ github.base_ref }}

- name: Upload report
  uses: actions/upload-artifact@v4
  with:
    name: test-report
    path: dist/
```

### GitLab CI

```yaml
test:
  script:
    - go test -json ./... > results.json
    - go test -coverprofile=coverage.out ./...
    - trep report
        --tests results.json
        --cov coverage.out
        --threshold 80
        --fail-tests --fail-cov
        --annotate --annotate-platform gitlab
        --output-dir public/
  artifacts:
    paths: [public/]
    expose_as: "Test Report"
```

### Snapshot workflow (delta badges)

```sh
# On main branch after merge: save the baseline
trep report --tests results.xml --cov cov.out \
  --save-snapshot .trep/baseline.json

# On PR branches: compare against baseline
trep report --tests results.xml --cov cov.out \
  --baseline .trep/baseline.json \
  --baseline-label "main"
# → delta badges appear: "+2 pass / -1 fail / +3.2% cov"
```

## Output Formats

| Flag | Output | Use case |
|---|---|---|
| _(default)_ | Self-contained HTML | Browser view, PR artifacts, email attachments |
| `--output-format json` | Structured JSON | Dashboard ingestion, `jq` scripting, CI metrics |
| `--output-format sarif` | SARIF 2.1.0 | GitHub Advanced Security code scanning alerts |
| `--annotate` | GitHub/GitLab annotation lines | Inline PR comments, pipeline step decorations |
| `--save-snapshot` | Compact JSON snapshot | Persistent baseline for future delta comparison |

### SARIF output (GitHub Advanced Security)

SARIF files can be uploaded to GitHub Advanced Security and displayed as code
scanning alerts directly in pull requests.

```sh
# Produce SARIF for failed tests
trep test --output-format sarif -o results.sarif junit.xml

# Produce SARIF for coverage — flags files below 80% as warnings
trep cov --output-format sarif --threshold 80 -o cov.sarif coverage.out
```

Upload in a workflow step:

```yaml
- uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: results.sarif
    category: test-results
```

Behaviour:
- Test SARIF: each failing test case becomes a SARIF result with level `error`. File and line info are included when the input format provides them (e.g. GTest XML). Passing and skipped tests are omitted.
- Coverage SARIF: when `--threshold` is set, every file below the threshold becomes a result with level `warning`. When no threshold is set, an empty results array is produced (which clears any previous GHAS alerts).

### JSON schema (test)

```json
{
  "generated_at": "2024-03-15T10:00:00Z",
  "summary": { "total": 42, "passed": 40, "failed": 1, "skipped": 1, "pass_pct": 95.2 },
  "suites": [
    { "name": "CoreTests", "cases": [
      { "name": "TestAdd", "status": "pass", "duration_ms": 12 },
      { "name": "TestDiv", "status": "fail", "duration_ms": 5,
        "message": "expected 5, got 4", "file": "core_test.go", "line": 30 }
    ]}
  ]
}
```

### JSON schema (coverage)

```json
{
  "generated_at": "2024-03-15T10:00:00Z",
  "summary": {
    "lines_pct": 84.2, "lines_covered": 320, "lines_total": 380,
    "branch_pct": 71.0, "branch_covered": 142, "branch_total": 200,
    "func_pct": 91.3, "func_covered": 63, "func_total": 69,
    "file_count": 12
  },
  "files": [
    { "path": "src/parser.go",
      "lines_pct": 68.4, "lines_covered": 13, "lines_total": 19,
      "branch_pct": 75.0, "branch_covered": 3, "branch_total": 4 }
  ]
}
```

## License

MIT
