package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/Yanujz/trep/pkg/delta"
	"github.com/spf13/cobra"
)

type badgeOpts struct {
	testsFile   string
	covFile     string
	outputTests string
	outputCov   string
	quiet       bool
}

func newBadgeCmd(cfg *GlobalConfig) *cobra.Command {
	var o badgeOpts
	cmd := &cobra.Command{
		Use:   "badge",
		Short: "Generate SVG badges for tests or coverage from snapshot files",
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}
	f := cmd.Flags()
	f.StringVarP(&o.testsFile, "tests", "t", "", "JSON snapshot file with test results")
	f.StringVarP(&o.covFile, "coverage", "c", "", "JSON snapshot file with coverage results")
	f.StringVar(&o.outputTests, "out-tests", "tests.svg", "output path for test badge")
	f.StringVar(&o.outputCov, "out-coverage", "coverage.svg", "output path for coverage badge")
	f.BoolVarP(&o.quiet, "quiet", "q", false, "suppress progress output")
	return cmd
}

func (o *badgeOpts) run() error {
	if o.testsFile == "" && o.covFile == "" {
		return fmt.Errorf("must specify --tests and/or --coverage snapshot file")
	}

	if o.covFile != "" {
		snap, err := delta.Load(o.covFile)
		if err != nil {
			return fmt.Errorf("load coverage snapshot: %w", err)
		}
		if snap.Coverage == nil {
			return fmt.Errorf("snapshot %s does not contain coverage data", o.covFile)
		}

		pct := snap.Coverage.LinesPct
		color := "#e05d44" // red
		if pct >= 80 {
			color = "#4c1" // brightgreen
		} else if pct >= 50 {
			color = "#dfb317" // yellow
		}

		svg := generateSVG("coverage", fmt.Sprintf("%.1f%%", pct), color)
		if err := os.WriteFile(o.outputCov, []byte(svg), 0644); err != nil {
			return fmt.Errorf("write coverage badge: %w", err)
		}
		if !o.quiet {
			fmt.Printf("✓ Wrote coverage badge to %s\n", o.outputCov)
		}
	}

	if o.testsFile != "" {
		snap, err := delta.Load(o.testsFile)
		if err != nil {
			return fmt.Errorf("load tests snapshot: %w", err)
		}
		if snap.Tests == nil {
			return fmt.Errorf("snapshot %s does not contain test data", o.testsFile)
		}

		text := fmt.Sprintf("%d/%d", snap.Tests.Passed, snap.Tests.Total)
		color := "#4c1" // brightgreen
		if snap.Tests.Failed > 0 || snap.Tests.Passed < snap.Tests.Total {
			color = "#e05d44" // red
		}

		svg := generateSVG("tests", text, color)
		if err := os.WriteFile(o.outputTests, []byte(svg), 0644); err != nil {
			return fmt.Errorf("write tests badge: %w", err)
		}
		if !o.quiet {
			fmt.Printf("✓ Wrote test badge to %s\n", o.outputTests)
		}
	}

	return nil
}

func generateSVG(label, value, color string) string {
	// A standard shields.io flat badge template.
	// For accurate text width, we guess ~7px per character for standard sans-serif.
	labelW := len(label)*7 + 24
	valueW := len(value)*7 + 24
	totalW := labelW + valueW

	svg := `<?xml version="1.0"?>
<svg xmlns="http://www.w3.org/2000/svg" width="%%TOTAL%%" height="20">
	<linearGradient id="b" x2="0" y2="100%%">
		<stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
		<stop offset="1" stop-opacity=".1"/>
	</linearGradient>
	<clipPath id="a">
		<rect width="%%TOTAL%%" height="20" rx="3" fill="#fff"/>
	</clipPath>
	<g clip-path="url(#a)">
		<path fill="#555" d="M0 0h%%LABEL_W%%v20H0z"/>
		<path fill="%%COLOR%%" d="M%%LABEL_W%% 0h%%VALUE_W%%v20H%%LABEL_W%%z"/>
		<path fill="url(#b)" d="M0 0h%%TOTAL%%v20H0z"/>
	</g>
	<g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11">
		<text x="%%LABEL_X%%" y="15" fill="#010101" fill-opacity=".3">%%LABEL%%</text>
		<text x="%%LABEL_X%%" y="14">%%LABEL%%</text>
		<text x="%%VALUE_X%%" y="15" fill="#010101" fill-opacity=".3">%%VALUE%%</text>
		<text x="%%VALUE_X%%" y="14">%%VALUE%%</text>
	</g>
</svg>
`
	svg = strings.ReplaceAll(svg, "%%TOTAL%%", fmt.Sprint(totalW))
	svg = strings.ReplaceAll(svg, "%%LABEL_W%%", fmt.Sprint(labelW))
	svg = strings.ReplaceAll(svg, "%%VALUE_W%%", fmt.Sprint(valueW))
	svg = strings.ReplaceAll(svg, "%%COLOR%%", color)
	svg = strings.ReplaceAll(svg, "%%LABEL%%", label)
	svg = strings.ReplaceAll(svg, "%%VALUE%%", value)
	svg = strings.ReplaceAll(svg, "%%LABEL_X%%", fmt.Sprint(labelW/2))
	svg = strings.ReplaceAll(svg, "%%VALUE_X%%", fmt.Sprint(labelW+valueW/2))

	return svg
}
