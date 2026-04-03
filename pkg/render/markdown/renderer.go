// Package markdown renders test and coverage reports into GitHub Flavored
// Markdown, suitable for piping to $GITHUB_STEP_SUMMARY.
package markdown

import (
	"fmt"
	"io"
	"time"

	covmodel "github.com/Yanujz/trep/pkg/coverage/model"
	"github.com/Yanujz/trep/pkg/model"
)

// RenderTest writes a GFM test summary to w.
func RenderTest(w io.Writer, rep *model.Report) error {
	total, passed, failed, skipped := rep.Stats()
	passPct := 0.0
	if total > 0 {
		passPct = float64(passed) / float64(total) * 100
	}

	title := rep.Title
	if title == "" {
		title = "Test Report"
	}

	fmt.Fprintf(w, "## %s\n\n", title)
	fmt.Fprintf(w, "_Generated: %s_\n\n", time.Now().UTC().Format(time.RFC3339))

	// Summary table
	fmt.Fprintf(w, "| Metric | Value |\n|--------|-------|\n")
	fmt.Fprintf(w, "| **Total** | %d |\n", total)
	fmt.Fprintf(w, "| ✅ Passed | %d |\n", passed)
	fmt.Fprintf(w, "| ❌ Failed | %d |\n", failed)
	fmt.Fprintf(w, "| ⏭️ Skipped | %d |\n", skipped)
	fmt.Fprintf(w, "| Pass rate | %.1f%% |\n\n", passPct)

	// Failures section
	if failed > 0 {
		fmt.Fprintf(w, "### ❌ Failed Tests\n\n")
		for _, s := range rep.Suites {
			for _, c := range s.Cases {
				if c.Status != model.StatusFail {
					continue
				}
				fmt.Fprintf(w, "#### `%s / %s`\n\n", s.Name, c.Name)
				if c.Message != "" {
					fmt.Fprintf(w, "```\n%s\n```\n\n", c.Message)
				}
				if c.Stdout != "" {
					fmt.Fprintf(w, "<details><summary>Output</summary>\n\n```\n%s\n```\n\n</details>\n\n", c.Stdout)
				}
			}
		}
	}

	return nil
}

// RenderCov writes a GFM coverage summary to w.
func RenderCov(w io.Writer, rep *covmodel.CovReport) error {
	fmt.Fprintf(w, "## Coverage Report\n\n")
	fmt.Fprintf(w, "_Generated: %s_\n\n", time.Now().UTC().Format(time.RFC3339))

	lt, lc, bt, bc, ft, fc := rep.Stats()
	lineEmoji := coverageEmoji(rep.LinePct())

	fmt.Fprintf(w, "| Metric | Covered | Total | %% |\n|--------|---------|-------|----|\n")
	fmt.Fprintf(w, "| %s Lines | %d | %d | %.1f%% |\n", lineEmoji, lc, lt, rep.LinePct())
	if bt > 0 {
		fmt.Fprintf(w, "| %s Branches | %d | %d | %.1f%% |\n", coverageEmoji(rep.BranchPct()), bc, bt, rep.BranchPct())
	}
	if ft > 0 {
		fmt.Fprintf(w, "| %s Functions | %d | %d | %.1f%% |\n", coverageEmoji(rep.FuncPct()), fc, ft, rep.FuncPct())
	}
	fmt.Fprintf(w, "\n")

	// Top 10 lowest coverage files (only if there are files)
	if len(rep.Files) > 0 {
		fmt.Fprintf(w, "<details><summary>Per-file coverage (%d files)</summary>\n\n", len(rep.Files))
		fmt.Fprintf(w, "| File | Lines %% | Lines | Branches %% | Functions %% |\n")
		fmt.Fprintf(w, "|------|---------|-------|------------|-------------|\n")
		for _, f := range rep.Files {
			branchStr := "–"
			funcStr := "–"
			if f.BranchTotal > 0 {
				branchStr = fmt.Sprintf("%.1f%%", f.BranchPct())
			}
			if f.FuncTotal > 0 {
				funcStr = fmt.Sprintf("%.1f%%", f.FuncPct())
			}
			fmt.Fprintf(w, "| `%s` | %.1f%% | %d/%d | %s | %s |\n",
				f.Path, f.LinePct(), f.LinesCovered, f.LinesTotal, branchStr, funcStr)
		}
		fmt.Fprintf(w, "\n</details>\n\n")
	}

	return nil
}

func coverageEmoji(pct float64) string {
	switch {
	case pct >= 80:
		return "🟢"
	case pct >= 50:
		return "🟡"
	default:
		return "🔴"
	}
}
