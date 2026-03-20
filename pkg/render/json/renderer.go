// Package json provides machine-readable JSON renderers for test and coverage
// reports, suitable for dashboards, scripts, and pipeline integrations.
package json

import (
	"encoding/json"
	"io"
	"time"

	covmodel "github.com/trep-dev/trep/pkg/coverage/model"
	"github.com/trep-dev/trep/pkg/model"
)

// ── Test report ───────────────────────────────────────────────────────────────

// TestOutput is the JSON envelope for a test report.
type TestOutput struct {
	GeneratedAt string        `json:"generated_at"`
	Summary     TestSummary   `json:"summary"`
	Suites      []SuiteOutput `json:"suites"`
}

// TestSummary holds aggregate counts for a test report.
type TestSummary struct {
	Total   int     `json:"total"`
	Passed  int     `json:"passed"`
	Failed  int     `json:"failed"`
	Skipped int     `json:"skipped"`
	PassPct float64 `json:"pass_pct"`
}

// SuiteOutput is the JSON representation of a test suite.
type SuiteOutput struct {
	Name  string       `json:"name"`
	Cases []CaseOutput `json:"cases"`
}

// CaseOutput is the JSON representation of a single test case.
type CaseOutput struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	DurationMs int64  `json:"duration_ms"`
	Message    string `json:"message,omitempty"`
	File       string `json:"file,omitempty"`
	Line       int    `json:"line,omitempty"`
}

// RenderTest writes a JSON test report to w.
func RenderTest(w io.Writer, rep *model.Report) error {
	total, passed, failed, skipped := rep.Stats()
	passPct := 0.0
	if total > 0 {
		passPct = float64(passed) / float64(total) * 100
	}

	out := TestOutput{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Summary: TestSummary{
			Total:   total,
			Passed:  passed,
			Failed:  failed,
			Skipped: skipped,
			PassPct: passPct,
		},
	}

	for _, s := range rep.Suites {
		so := SuiteOutput{Name: s.Name}
		for _, c := range s.Cases {
			so.Cases = append(so.Cases, CaseOutput{
				Name:       c.Name,
				Status:     string(c.Status),
				DurationMs: c.Duration.Milliseconds(),
				Message:    c.Message,
				File:       c.File,
				Line:       c.Line,
			})
		}
		out.Suites = append(out.Suites, so)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// ── Coverage report ───────────────────────────────────────────────────────────

// CovOutput is the JSON envelope for a coverage report.
type CovOutput struct {
	GeneratedAt string       `json:"generated_at"`
	Summary     CovSummary   `json:"summary"`
	Files       []FileOutput `json:"files"`
}

// CovSummary holds aggregate coverage statistics for a coverage report.
type CovSummary struct {
	LinesPct    float64 `json:"lines_pct"`
	LinesCov    int     `json:"lines_covered"`
	LinesTotal  int     `json:"lines_total"`
	BranchPct   float64 `json:"branch_pct"`
	BranchCov   int     `json:"branch_covered"`
	BranchTotal int     `json:"branch_total"`
	FuncPct     float64 `json:"func_pct"`
	FuncCov     int     `json:"func_covered"`
	FuncTotal   int     `json:"func_total"`
	FileCount   int     `json:"file_count"`
}

// FileOutput is the JSON representation of per-file coverage data.
type FileOutput struct {
	Path        string  `json:"path"`
	LinesPct    float64 `json:"lines_pct"`
	LinesCov    int     `json:"lines_covered"`
	LinesTotal  int     `json:"lines_total"`
	BranchPct   float64 `json:"branch_pct,omitempty"`
	BranchCov   int     `json:"branch_covered,omitempty"`
	BranchTotal int     `json:"branch_total,omitempty"`
	FuncPct     float64 `json:"func_pct,omitempty"`
	FuncCov     int     `json:"func_covered,omitempty"`
	FuncTotal   int     `json:"func_total,omitempty"`
}

// RenderCov writes a JSON coverage report to w.
func RenderCov(w io.Writer, rep *covmodel.CovReport) error {
	lt, lc, bt, bc, ft, fc := rep.Stats()

	out := CovOutput{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Summary: CovSummary{
			LinesPct:    rep.LinePct(),
			LinesCov:    lc,
			LinesTotal:  lt,
			BranchPct:   rep.BranchPct(),
			BranchCov:   bc,
			BranchTotal: bt,
			FuncPct:     rep.FuncPct(),
			FuncCov:     fc,
			FuncTotal:   ft,
			FileCount:   len(rep.Files),
		},
	}

	for _, f := range rep.Files {
		fo := FileOutput{
			Path:       f.Path,
			LinesPct:   f.LinePct(),
			LinesCov:   f.LinesCovered,
			LinesTotal: f.LinesTotal,
		}
		if f.BranchTotal > 0 {
			fo.BranchPct = f.BranchPct()
			fo.BranchCov = f.BranchCovered
			fo.BranchTotal = f.BranchTotal
		}
		if f.FuncTotal > 0 {
			fo.FuncPct = f.FuncPct()
			fo.FuncCov = f.FuncCovered
			fo.FuncTotal = f.FuncTotal
		}
		out.Files = append(out.Files, fo)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
