// Package html renders a model.Report into a fully self-contained HTML page.
package html

import (
	_ "embed" // for go:embed directive
	"encoding/json"
	"fmt"
	htmlpkg "html"
	"io"
	"strings"
	"time"

	"github.com/Yanujz/trep/pkg/delta"
	"github.com/Yanujz/trep/pkg/model"
)

//go:embed template.html
var templateHTML string

// Options controls optional rendering features.
type Options struct {
	CovReportURL  string // cross-link to coverage page
	Delta         *delta.Delta
	BaselineLabel string
}

// Renderer produces a self-contained HTML report from a [model.Report].
type Renderer struct{}

// Name returns the renderer identifier.
func (Renderer) Name() string { return "html" }

// Render writes a fully self-contained HTML page to w.
func (Renderer) Render(w io.Writer, rep *model.Report, opts Options) error {
	total, passed, failed, skipped := rep.Stats()

	// Count flaky separately for the %%FLAKY%% template placeholder.
	flaky := 0
	for _, s := range rep.Suites {
		for _, c := range s.Cases {
			if c.Status == model.StatusFlaky {
				flaky++
			}
		}
	}
	// Each row: [suite, name, result, dur_ms, detail, stdout (, file, line), attachJSON]
	// file/line are appended only when present (GTest). attachJSON is always appended.
	rows := make([][]any, 0, total)
	for _, suite := range rep.Suites {
		for _, c := range suite.Cases {
			stdout := ""
			if c.Status == model.StatusFail {
				stdout = c.Stdout
			}
			row := []any{
				c.Suite,
				c.Name,
				string(c.Status),
				c.Duration.Milliseconds(),
				c.Message,
				stdout,
			}
			if c.File != "" || c.Line > 0 {
				row = append(row, c.File, c.Line)
			}
			attachJSON := "[]"
			if len(c.Attachments) > 0 {
				b, _ := json.Marshal(c.Attachments)
				attachJSON = string(b)
			}
			row = append(row, attachJSON)
			rows = append(rows, row)
		}
	}

	dataJSON, err := json.Marshal(rows)
	if err != nil {
		return fmt.Errorf("html render: marshal: %w", err)
	}
	// Guard against </script> injection and HTML entity confusion.
	dataStr := strings.NewReplacer(
		"<", `\u003c`,
		">", `\u003e`,
		"&", `\u0026`,
	).Replace(string(dataJSON))

	// Timestamp
	ts := rep.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	tsStr := ts.Format("2006-01-02 15:04:05 UTC")

	// Title
	title := rep.Title
	if title == "" {
		switch len(rep.Sources) {
		case 0:
			title = "Test Report"
		case 1:
			title = rep.Sources[0]
		default:
			title = fmt.Sprintf("%s (+%d more)", rep.Sources[0], len(rep.Sources)-1)
		}
	}

	// Overall status
	status := "PASSED"
	statusCls := "b-pass"
	if failed > 0 {
		status = "FAILED"
		statusCls = "b-fail"
	}

	pct := func(n, d int) string {
		if d == 0 {
			return "0.0"
		}
		return fmt.Sprintf("%.1f", float64(n)/float64(d)*100)
	}

	// Elapsed string
	d := int(rep.Duration.Seconds())
	var elapsed string
	switch {
	case rep.Duration == 0:
		elapsed = "–"
	case d < 60:
		elapsed = fmt.Sprintf("%ds", d)
	default:
		elapsed = fmt.Sprintf("%dm %ds", d/60, d%60)
	}

	// Delta JSON blob.
	deltaStr := "null"
	if opts.Delta != nil {
		d := opts.Delta
		type dj struct {
			HasTests     bool    `json:"hasTests"`
			FailedDelta  int     `json:"failedDelta"`
			PassedDelta  int     `json:"passedDelta"`
			SkippedDelta int     `json:"skippedDelta"`
			HasCoverage  bool    `json:"hasCoverage"`
			LinesPct     float64 `json:"linesPct"`
			BaseLabel    string  `json:"baseLabel"`
		}
		raw, _ := json.Marshal(dj{
			HasTests:     d.HasTests,
			FailedDelta:  d.FailedDelta,
			PassedDelta:  d.PassedDelta,
			SkippedDelta: d.SkippedDelta,
			HasCoverage:  d.HasCoverage,
			LinesPct:     d.LinesPctDelta,
			BaseLabel:    opts.BaselineLabel,
		})
		deltaStr = string(raw)
	}

	covURL := htmlpkg.EscapeString(opts.CovReportURL)

	r := strings.NewReplacer(
		"%%TITLE%%", htmlpkg.EscapeString(title),
		"%%STATUS%%", status,
		"%%STATUS_CLS%%", statusCls,
		"%%TS%%", htmlpkg.EscapeString(tsStr),
		"%%ELAPSED%%", htmlpkg.EscapeString(elapsed),
		"%%TOTAL%%", fmt.Sprintf("%d", total),
		"%%PASSED%%", fmt.Sprintf("%d", passed),
		"%%FAILED%%", fmt.Sprintf("%d", failed),
		"%%SKIPPED%%", fmt.Sprintf("%d", skipped),
		"%%FLAKY%%", fmt.Sprintf("%d", flaky),
		"%%PCT_PASS%%", pct(passed, total),
		"%%PCT_FAIL%%", pct(failed, total),
		"%%PCT_SKIP%%", pct(skipped, total),
		"%%DELTA%%", deltaStr,
		"%%COV_URL%%", covURL,
		"%%DATA%%", dataStr,
	)

	_, err = io.WriteString(w, r.Replace(templateHTML))
	return err
}
