package annotations_test

import (
	"bytes"
	"strings"
	"testing"

	covmodel "github.com/trep-dev/trep/pkg/coverage/model"
	"github.com/trep-dev/trep/pkg/model"
	"github.com/trep-dev/trep/pkg/render/annotations"
)

// ── WriteTestAnnotations — GitHub ─────────────────────────────────────────────

func TestWriteTestAnnotations_GitHub_FailWithFileAndLine(t *testing.T) {
	rep := &model.Report{
		Suites: []model.Suite{{
			Name: "pkg.MyTests",
			Cases: []model.TestCase{{
				Name:    "TestFail",
				Status:  model.StatusFail,
				Message: "assertion failed",
				File:    "src/myfile.go",
				Line:    42,
			}},
		}},
	}

	var buf bytes.Buffer
	if err := annotations.WriteTestAnnotations(&buf, rep, annotations.GitHub); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()

	if !strings.HasPrefix(out, "::error file=src/myfile.go,line=42,title=TestFail::") {
		t.Errorf("unexpected output: %q", out)
	}
	if !strings.Contains(out, "assertion failed") {
		t.Errorf("message not in output: %q", out)
	}
}

func TestWriteTestAnnotations_GitHub_FailWithoutFile(t *testing.T) {
	rep := &model.Report{
		Suites: []model.Suite{{
			Name: "pkg.MyTests",
			Cases: []model.TestCase{{
				Name:    "TestNoFile",
				Status:  model.StatusFail,
				Message: "something broke",
				// File and Line deliberately absent (zero values).
			}},
		}},
	}

	var buf bytes.Buffer
	annotations.WriteTestAnnotations(&buf, rep, annotations.GitHub)
	out := buf.String()

	// Without a line number, the format uses the title-only form.
	if !strings.HasPrefix(out, "::error title=TestNoFile::") {
		t.Errorf("expected '::error title=...' format without file, got: %q", out)
	}
}

func TestWriteTestAnnotations_GitHub_PassingTestNoOutput(t *testing.T) {
	rep := &model.Report{
		Suites: []model.Suite{{
			Cases: []model.TestCase{{
				Name:   "TestPass",
				Status: model.StatusPass,
			}},
		}},
	}

	var buf bytes.Buffer
	annotations.WriteTestAnnotations(&buf, rep, annotations.GitHub)
	if buf.Len() != 0 {
		t.Errorf("passing test should produce no output, got: %q", buf.String())
	}
}

func TestWriteTestAnnotations_GitHub_EscapesPercent(t *testing.T) {
	rep := &model.Report{
		Suites: []model.Suite{{
			Cases: []model.TestCase{{
				Name:   "Test50%Coverage",
				Status: model.StatusFail,
			}},
		}},
	}

	var buf bytes.Buffer
	annotations.WriteTestAnnotations(&buf, rep, annotations.GitHub)
	out := buf.String()

	if !strings.Contains(out, "Test50%25Coverage") {
		t.Errorf("'%%' in name should be escaped to '%%25', got: %q", out)
	}
}

func TestWriteTestAnnotations_GitHub_EscapesNewlineInName(t *testing.T) {
	rep := &model.Report{
		Suites: []model.Suite{{
			Cases: []model.TestCase{{
				Name:   "Test\nFoo",
				Status: model.StatusFail,
			}},
		}},
	}

	var buf bytes.Buffer
	annotations.WriteTestAnnotations(&buf, rep, annotations.GitHub)
	out := buf.String()

	if !strings.Contains(out, "Test%0AFoo") {
		t.Errorf("newline in name should become '%%0A', got: %q", out)
	}
}

// ── WriteTestAnnotations — GitLab ─────────────────────────────────────────────

func TestWriteTestAnnotations_GitLab_FailedTest(t *testing.T) {
	rep := &model.Report{
		Suites: []model.Suite{{
			Name: "MySuite",
			Cases: []model.TestCase{{
				Name:    "TestGitLab",
				Status:  model.StatusFail,
				Message: "something went wrong",
			}},
		}},
	}

	var buf bytes.Buffer
	annotations.WriteTestAnnotations(&buf, rep, annotations.GitLab)
	out := buf.String()

	// GitLab format: ANSI red FAIL + suite :: name — message
	if !strings.Contains(out, "FAIL") {
		t.Errorf("GitLab output should contain FAIL, got: %q", out)
	}
	if !strings.Contains(out, "MySuite") {
		t.Errorf("GitLab output should contain suite name, got: %q", out)
	}
	if !strings.Contains(out, "TestGitLab") {
		t.Errorf("GitLab output should contain test name, got: %q", out)
	}
}

// ── WriteCovAnnotations ───────────────────────────────────────────────────────

func TestWriteCovAnnotations_ThresholdZero_NoOutput(t *testing.T) {
	rep := &covmodel.CovReport{
		Files: []*covmodel.FileCov{{
			Path:         "src/a.go",
			LinesTotal:   10,
			LinesCovered: 0,
		}},
	}

	var buf bytes.Buffer
	annotations.WriteCovAnnotations(&buf, rep, 0, annotations.GitHub)
	if buf.Len() != 0 {
		t.Errorf("threshold=0 should produce no output, got: %q", buf.String())
	}
}

func TestWriteCovAnnotations_GitHub_FileBelowThreshold(t *testing.T) {
	rep := &covmodel.CovReport{
		Files: []*covmodel.FileCov{{
			Path:         "src/low.go",
			LinesTotal:   10,
			LinesCovered: 6, // 60%
		}},
	}

	var buf bytes.Buffer
	annotations.WriteCovAnnotations(&buf, rep, 80.0, annotations.GitHub)
	out := buf.String()

	if !strings.Contains(out, "::warning file=src/low.go,title=Low Coverage::") {
		t.Errorf("expected ::warning for low-coverage file, got: %q", out)
	}
	if !strings.Contains(out, "60.0%25") {
		t.Errorf("'%%' in detail should be escaped to '%%25', got: %q", out)
	}
}

func TestWriteCovAnnotations_GitHub_OverallBelowThreshold(t *testing.T) {
	rep := &covmodel.CovReport{
		Files: []*covmodel.FileCov{
			{Path: "src/low.go", LinesTotal: 10, LinesCovered: 6},  // 60% → warning
			{Path: "src/high.go", LinesTotal: 10, LinesCovered: 9}, // 90% → no warning
		},
	}
	// Overall = 15/20 = 75% < 80% → also fires the overall error.

	var buf bytes.Buffer
	annotations.WriteCovAnnotations(&buf, rep, 80.0, annotations.GitHub)
	out := buf.String()

	if !strings.Contains(out, "::warning file=src/low.go") {
		t.Errorf("expected warning for low.go, got: %q", out)
	}
	if strings.Contains(out, "::warning file=src/high.go") {
		t.Errorf("high.go at 90%% should not get a warning, got: %q", out)
	}
	if !strings.Contains(out, "::error title=Coverage Below Threshold::") {
		t.Errorf("expected overall-threshold error, got: %q", out)
	}
}

func TestWriteCovAnnotations_GitHub_AllAboveThreshold_NoOutput(t *testing.T) {
	rep := &covmodel.CovReport{
		Files: []*covmodel.FileCov{{
			Path:         "src/good.go",
			LinesTotal:   10,
			LinesCovered: 10, // 100%
		}},
	}

	var buf bytes.Buffer
	annotations.WriteCovAnnotations(&buf, rep, 80.0, annotations.GitHub)
	if buf.Len() != 0 {
		t.Errorf("all files above threshold should produce no output, got: %q", buf.String())
	}
}

func TestWriteCovAnnotations_GitLab_FileBelowThreshold(t *testing.T) {
	rep := &covmodel.CovReport{
		Files: []*covmodel.FileCov{{
			Path:         "src/low.go",
			LinesTotal:   10,
			LinesCovered: 5, // 50%
		}},
	}

	var buf bytes.Buffer
	annotations.WriteCovAnnotations(&buf, rep, 80.0, annotations.GitLab)
	out := buf.String()

	if !strings.Contains(out, "WARN") {
		t.Errorf("GitLab low coverage should contain WARN, got: %q", out)
	}
	if !strings.Contains(out, "src/low.go") {
		t.Errorf("GitLab output should contain file path, got: %q", out)
	}
}

func TestWriteTestAnnotations_DefaultMessageWhenEmpty(t *testing.T) {
	rep := &model.Report{
		Suites: []model.Suite{{
			Cases: []model.TestCase{{
				Name:    "TestEmpty",
				Status:  model.StatusFail,
				Message: "", // empty — should default to "test failed"
			}},
		}},
	}

	var buf bytes.Buffer
	annotations.WriteTestAnnotations(&buf, rep, annotations.GitHub)
	out := buf.String()

	if !strings.Contains(out, "test failed") {
		t.Errorf("empty message should default to 'test failed', got: %q", out)
	}
}
