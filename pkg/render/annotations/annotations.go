// Package annotations emits CI-native annotation lines from test and coverage
// reports, enabling inline PR comments and pipeline summaries.
//
//	GitHub Actions  ::error file=…,line=…,title=…::message
//	GitLab CI       ANSI-coloured lines to stdout
package annotations

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	covmodel "github.com/Yanujz/trep/pkg/coverage/model"
	"github.com/Yanujz/trep/pkg/model"
)

// Platform selects the annotation syntax.
type Platform string

const (
	// GitHub is the annotation platform identifier for GitHub Actions.
	GitHub Platform = "github"
	// GitLab is the annotation platform identifier for GitLab CI.
	GitLab Platform = "gitlab"
	// Auto detects from $GITHUB_ACTIONS / $GITLAB_CI; falls back to GitHub format.
	Auto Platform = "auto"
)

// Detect returns the active Platform from environment variables.
func Detect() Platform {
	if os.Getenv("GITHUB_ACTIONS") != "" {
		return GitHub
	}
	if os.Getenv("GITLAB_CI") != "" {
		return GitLab
	}
	return GitHub
}

func resolve(p Platform) Platform {
	if p == Auto {
		return Detect()
	}
	return p
}

// WriteTestAnnotations emits one annotation line per failed test case.
func WriteTestAnnotations(w io.Writer, rep *model.Report, p Platform) error {
	p = resolve(p)
	for _, suite := range rep.Suites {
		for _, c := range suite.Cases {
			if c.Status != model.StatusFail {
				continue
			}
			msg := c.Message
			if msg == "" {
				msg = "test failed"
			}
			title := firstLine(msg)
			switch p {
			case GitHub:
				file := c.File
				if file == "" {
					file = strings.ReplaceAll(suite.Name, ".", "/")
				}
				if c.Line > 0 {
					fmt.Fprintf(w, "::error file=%s,line=%d,title=%s::%s\n",
						escGH(file), c.Line, escGH(c.Name), escGH(title))
				} else {
					fmt.Fprintf(w, "::error title=%s::%s\n",
						escGH(c.Name), escGH(title))
				}
			case GitLab:
				fmt.Fprintf(w, "\x1b[31mFAIL\x1b[0m  %s :: %s — %s\n",
					suite.Name, c.Name, title)
			}
		}
	}
	return nil
}

// WriteCovAnnotations emits one warning per file whose line coverage is below
// threshold, sorted worst-first, plus a summary error if overall is below too.
func WriteCovAnnotations(w io.Writer, rep *covmodel.CovReport, threshold float64, p Platform) error {
	if threshold <= 0 {
		return nil
	}
	p = resolve(p)

	type entry struct {
		path string
		pct  float64
		cov  int
		tot  int
	}
	var low []entry
	for _, f := range rep.Files {
		if f.LinesTotal == 0 {
			continue
		}
		if pct := f.LinePct(); pct < threshold {
			low = append(low, entry{f.Path, pct, f.LinesCovered, f.LinesTotal})
		}
	}
	sort.Slice(low, func(i, j int) bool { return low[i].pct < low[j].pct })

	for _, e := range low {
		detail := fmt.Sprintf("%.1f%% line coverage (%d/%d lines) — threshold %.1f%%",
			e.pct, e.cov, e.tot, threshold)
		switch p {
		case GitHub:
			fmt.Fprintf(w, "::warning file=%s,title=Low Coverage::%s\n",
				escGH(e.path), escGH(detail))
		case GitLab:
			fmt.Fprintf(w, "\x1b[33mWARN\x1b[0m  %s — %s\n", e.path, detail)
		}
	}

	if overall := rep.LinePct(); overall < threshold {
		summary := fmt.Sprintf("overall line coverage %.1f%% is below threshold %.1f%%",
			overall, threshold)
		switch p {
		case GitHub:
			fmt.Fprintf(w, "::error title=Coverage Below Threshold::%s\n", escGH(summary))
		case GitLab:
			fmt.Fprintf(w, "\x1b[31mFAIL\x1b[0m  %s\n", summary)
		}
	}
	return nil
}

func escGH(s string) string {
	s = strings.ReplaceAll(s, "%", "%25")
	s = strings.ReplaceAll(s, "\r", "%0D")
	s = strings.ReplaceAll(s, "\n", "%0A")
	s = strings.ReplaceAll(s, ":", "%3A")
	s = strings.ReplaceAll(s, ",", "%2C")
	return s
}

func firstLine(s string) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}
