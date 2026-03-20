// Package html renders a coverage report into a self-contained HTML page.
package html

import (
	_ "embed" // for go:embed directive
	"encoding/json"
	"fmt"
	htmlpkg "html"
	"io"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"time"

	covmodel "github.com/Yanujz/trep/pkg/coverage/model"
	"github.com/Yanujz/trep/pkg/delta"
)

//go:embed template.html
var templateHTML string

// Options controls optional rendering features.
type Options struct {
	Title           string
	ThresholdLine   float64 // 0 = disabled; minimum acceptable line coverage %
	ThresholdBranch float64 // 0 = disabled; minimum acceptable branch coverage %
	ThresholdFunc   float64 // 0 = disabled; minimum acceptable function coverage %
	TestReportURL   string  // cross-link to test report page
	Delta           *delta.Delta
	BaselineLabel   string
}

// Renderer produces a self-contained HTML coverage report.
type Renderer struct{}

// Name returns the renderer name.
func (Renderer) Name() string { return "html-cov" }

// Render writes a fully self-contained HTML coverage page to w.
func (Renderer) Render(w io.Writer, rep *covmodel.CovReport, opts Options) error {
	lt, lc, bt, bc, ft, fc := rep.Stats()

	// Build flat file data for the JS layer.
	// Each entry: [path, dirPath, linePct, linesCov, linesTotal,
	//              branchPct, branchCov, branchTotal,
	//              funcPct,   funcCov,   funcTotal]
	type fileRow struct {
		Path        string
		Dir         string
		LinePct     float64
		LinesCov    int
		LinesTotal  int
		BranchPct   float64
		BranchCov   int
		BranchTotal int
		FuncPct     float64
		FuncCov     int
		FuncTotal   int
	}

	rows := make([]fileRow, 0, len(rep.Files))
	for _, f := range rep.Files {
		rows = append(rows, fileRow{
			Path:        f.Path,
			Dir:         dirOf(f.Path),
			LinePct:     round2(f.LinePct()),
			LinesCov:    f.LinesCovered,
			LinesTotal:  f.LinesTotal,
			BranchPct:   round2(f.BranchPct()),
			BranchCov:   f.BranchCovered,
			BranchTotal: f.BranchTotal,
			FuncPct:     round2(f.FuncPct()),
			FuncCov:     f.FuncCovered,
			FuncTotal:   f.FuncTotal,
		})
	}
	// Sort by path for stable output.
	sort.Slice(rows, func(i, j int) bool { return rows[i].Path < rows[j].Path })

	// Serialise to a JS-friendly array of arrays for compact embedding.
	rawRows := make([][]any, len(rows))
	for i, r := range rows {
		rawRows[i] = []any{
			r.Path, r.Dir,
			r.LinePct, r.LinesCov, r.LinesTotal,
			r.BranchPct, r.BranchCov, r.BranchTotal,
			r.FuncPct, r.FuncCov, r.FuncTotal,
		}
	}
	dataJSON, err := json.Marshal(rawRows)
	if err != nil {
		return fmt.Errorf("cov html: marshal: %w", err)
	}
	dataStr := strings.NewReplacer("<", `\u003c`, ">", `\u003e`, "&", `\u0026`).
		Replace(string(dataJSON))

	// Timestamps / title.
	ts := rep.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	tsStr := ts.Format("2006-01-02 15:04:05 UTC")

	title := opts.Title
	if title == "" {
		if len(rep.Sources) == 1 {
			title = rep.Sources[0]
		} else {
			title = "Coverage Report"
		}
	}

	// Overall stats.
	linePct := round2(safePct(lc, lt))
	branchPct := round2(safePct(bc, bt))
	funcPct := round2(safePct(fc, ft))

	// Threshold marker.
	thresh := opts.ThresholdLine
	threshStr := "0"
	if thresh > 0 {
		threshStr = fmt.Sprintf("%.1f", thresh)
	}

	// Delta JSON blob (null if not available).
	deltaStr := "null"
	if opts.Delta != nil {
		d := opts.Delta
		type dj struct {
			HasTests    bool               `json:"hasTests"`
			FailedDelta int                `json:"failedDelta"`
			PassedDelta int                `json:"passedDelta"`
			HasCoverage bool               `json:"hasCoverage"`
			LinesPct    float64            `json:"linesPct"`
			BranchPct   float64            `json:"branchPct"`
			FuncPct     float64            `json:"funcPct"`
			BaseLabel   string             `json:"baseLabel"`
			Files       map[string]float64 `json:"files,omitempty"`
		}
		raw, _ := json.Marshal(dj{
			HasTests:    d.HasTests,
			FailedDelta: d.FailedDelta,
			PassedDelta: d.PassedDelta,
			HasCoverage: d.HasCoverage,
			LinesPct:    round2(d.LinesPctDelta),
			BranchPct:   round2(d.BranchPctDelta),
			FuncPct:     round2(d.FuncPctDelta),
			BaseLabel:   opts.BaselineLabel,
			Files:       d.FileDeltas,
		})
		deltaStr = string(raw)
	}

	// Cross-link to test report.
	testURL := htmlpkg.EscapeString(opts.TestReportURL)

	// Status badge.
	belowThreshold := thresh > 0 && linePct < thresh
	statusLabel := "PASS"
	statusCls := "b-pass"
	if belowThreshold {
		statusLabel = "BELOW THRESHOLD"
		statusCls = "b-fail"
	}

	pct := func(n, d int) string { return fmt.Sprintf("%.1f", safePct(n, d)) }

	r := strings.NewReplacer(
		"%%TITLE%%", htmlpkg.EscapeString(title),
		"%%TS%%", htmlpkg.EscapeString(tsStr),
		"%%STATUS%%", statusLabel,
		"%%STATUS_CLS%%", statusCls,
		"%%LINE_PCT%%", fmt.Sprintf("%.1f", linePct),
		"%%LINE_COV%%", fmt.Sprintf("%d", lc),
		"%%LINE_TOTAL%%", fmt.Sprintf("%d", lt),
		"%%BRANCH_PCT%%", fmt.Sprintf("%.1f", branchPct),
		"%%BRANCH_COV%%", fmt.Sprintf("%d", bc),
		"%%BRANCH_TOTAL%%", fmt.Sprintf("%d", bt),
		"%%FUNC_PCT%%", fmt.Sprintf("%.1f", funcPct),
		"%%FUNC_COV%%", fmt.Sprintf("%d", fc),
		"%%FUNC_TOTAL%%", fmt.Sprintf("%d", ft),
		"%%FILE_COUNT%%", fmt.Sprintf("%d", len(rep.Files)),
		"%%PCT_LINE%%", pct(lc, lt),
		"%%PCT_BRANCH%%", pct(bc, bt),
		"%%PCT_FUNC%%", pct(fc, ft),
		"%%THRESHOLD%%", threshStr,
		"%%DELTA%%", deltaStr,
		"%%TEST_URL%%", testURL,
		"%%DATA%%", dataStr,
	)

	_, err = io.WriteString(w, r.Replace(templateHTML))
	return err
}

// ── Helpers ────────────────────────────────────────────────────────────────────

func dirOf(path string) string {
	d := filepath.ToSlash(filepath.Dir(path))
	if d == "." {
		return ""
	}
	return d
}

func safePct(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d) * 100
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}
