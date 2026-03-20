package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/trep-dev/trep/pkg/render/annotations"
	covparser "github.com/trep-dev/trep/pkg/coverage/parser"
	covhtml "github.com/trep-dev/trep/pkg/coverage/render/html"
	"github.com/trep-dev/trep/pkg/delta"
	"github.com/trep-dev/trep/pkg/model"
	"github.com/trep-dev/trep/pkg/parser"
	htmlrender "github.com/trep-dev/trep/pkg/render/html"
)

type reportOpts struct {
	// Inputs
	testInputs  []string
	covInput    string
	testFormat  string
	covFormat   string

	// Output
	outputDir   string
	prefix      string
	title       string

	// CI
	threshold     float64
	failTests     bool
	failCov       bool
	open          bool
	quiet         bool
	annotate         bool
	annotatePlatform string

	// Delta
	saveSnapshot  string
	baseline      string
	baselineLabel string
	stripPrefix   string
}

func newReportCmd() *cobra.Command {
	o := &reportOpts{}

	cmd := &cobra.Command{
		Use:   "report [flags]",
		Short: "Generate linked test + coverage HTML report pair",
		Long: `Parse both test results and a coverage file, then produce two self-contained
HTML pages that cross-link to each other via a shared nav bar.

Examples
  trep report --tests results.xml --cov coverage.out
  trep report --tests a.xml b.xml --cov coverage.info --threshold 80 --fail-cov
  trep report --tests results.xml --cov cov.xml --output-dir dist/ --prefix ci
  trep report --tests results.xml --cov cov.out --save-snapshot snap.json
  trep report --tests results.xml --cov cov.out --baseline prev.json --baseline-label main`,
		SilenceUsage: true,
		RunE:         o.run,
	}

	f := cmd.Flags()
	f.StringArrayVar(&o.testInputs, "tests", nil,   "test result file(s) (required)")
	f.StringVar     (&o.covInput,   "cov",   "",    "coverage file (required)")
	f.StringVar     (&o.testFormat, "format-test", "auto", "force test input format")
	f.StringVar     (&o.covFormat,  "format-cov",  "auto", "force coverage input format")
	f.StringVar     (&o.outputDir,  "output-dir",  ".",   "directory to write report files into")
	f.StringVar     (&o.prefix,     "prefix",       "",    "filename prefix (default: 'report' → report_tests.html + report_cov.html)")
	f.StringVarP    (&o.title,      "title",       "t", "","report title (applied to both pages)")
	f.Float64Var    (&o.threshold,  "threshold",        0, "minimum line coverage % for --fail-cov")
	f.BoolVar       (&o.failTests,  "fail-tests",       false, "exit 1 if any tests failed")
	f.BoolVar       (&o.failCov,    "fail-cov",         false, "exit 1 if coverage is below --threshold")
	f.BoolVar       (&o.open,       "open",             false, "open both reports in the browser after writing")
	f.BoolVarP      (&o.quiet,      "quiet",       "q", false, "suppress progress output")
	f.BoolVar       (&o.annotate,   "annotate",         false, "emit CI annotations for failures and low-coverage files")
	f.StringVar     (&o.annotatePlatform, "annotate-platform", "auto", "annotation platform: auto | github | gitlab")
	f.StringVar     (&o.saveSnapshot,  "save-snapshot",   "", "write combined snapshot JSON for future delta comparison")
	f.StringVar     (&o.baseline,      "baseline",        "", "JSON snapshot from a previous run")
	f.StringVar     (&o.baselineLabel, "baseline-label",  "", "label for the baseline run")
	f.StringVar     (&o.stripPrefix,   "strip-prefix",    "", "remove prefix from coverage file paths")

	_ = cmd.MarkFlagRequired("tests")
	_ = cmd.MarkFlagRequired("cov")

	return cmd
}

func (o *reportOpts) run(_ *cobra.Command, _ []string) error {
	// ── Determine output paths ─────────────────────────────────────────────
	pfx := o.prefix
	if pfx == "" {
		pfx = "report"
	}
	if err := os.MkdirAll(o.outputDir, 0755); err != nil {
		return fmt.Errorf("output-dir %s: %w", o.outputDir, err)
	}
	testOut := filepath.Join(o.outputDir, pfx+"_tests.html")
	covOut  := filepath.Join(o.outputDir, pfx+"_cov.html")

	// Relative cross-links (both files live in the same directory).
	testFilename := filepath.Base(testOut)
	covFilename  := filepath.Base(covOut)

	// ── Parse test inputs ──────────────────────────────────────────────────
	var forcedTest parser.Parser
	if o.testFormat != "auto" {
		var err error
		forcedTest, err = parser.ForName(o.testFormat)
		if err != nil {
			return err
		}
	}

	var merged *model.Report
	for _, path := range o.testInputs {
		if !o.quiet {
			fmt.Fprintf(os.Stderr, "parsing  %s\n", path)
		}
		rep, err := parser.ParseFile(path, forcedTest)
		if err != nil {
			return err
		}
		if !o.quiet {
			tot, pass, fail, skip := rep.Stats()
			fmt.Fprintf(os.Stderr, "         total=%-5d  pass=%-5d  fail=%-5d  skip=%d\n",
				tot, pass, fail, skip)
		}
		if merged == nil {
			merged = rep
		} else {
			merged.Merge(rep)
		}
	}
	if o.title != "" {
		merged.Title = o.title
	}

	// ── Parse coverage ─────────────────────────────────────────────────────
	var forcedCov covparser.CovParser
	if o.covFormat != "auto" {
		var err error
		forcedCov, err = covparser.ForName(o.covFormat)
		if err != nil {
			return err
		}
	}

	if !o.quiet {
		fmt.Fprintf(os.Stderr, "parsing  %s\n", o.covInput)
	}
	covRep, err := covparser.ParseFile(o.covInput, forcedCov, o.stripPrefix)
	if err != nil {
		return err
	}
	if !o.quiet {
		lt, lc, _, _, _, _ := covRep.Stats()
		fmt.Fprintf(os.Stderr, "         files=%-4d  lines=%d/%d (%.1f%%)\n",
			len(covRep.Files), lc, lt, covRep.LinePct())
	}

	// ── Delta ──────────────────────────────────────────────────────────────
	var base *delta.Snapshot
	if o.baseline != "" {
		base, err = delta.Load(o.baseline)
		if err != nil {
			return err
		}
	}

	var d *delta.Delta
	if base != nil {
		cur := &delta.Snapshot{
			Version:   1,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Tests:     delta.FromTestReport(merged),
			Coverage:  delta.FromCovReport(covRep),
		}
		d = delta.Compute(base, cur)
	}

	// ── Annotations ───────────────────────────────────────────────────────────
	if o.annotate {
		ap := annotations.Platform(o.annotatePlatform)
		_ = annotations.WriteTestAnnotations(os.Stderr, merged, ap)
		_ = annotations.WriteCovAnnotations(os.Stderr, covRep, o.threshold, ap)
	}

	// ── Render test report ─────────────────────────────────────────────────
	testRenderer := htmlrender.Renderer{}
	testOpts := htmlrender.Options{
		CovReportURL:  covFilename,
		Delta:         d,
		BaselineLabel: o.baselineLabel,
	}
	if err := writeFile(testOut, func(w io.Writer) error {
		return testRenderer.Render(w, merged, testOpts)
	}); err != nil {
		return fmt.Errorf("render tests: %w", err)
	}
	tot, _, fail, _ := merged.Stats()
	logSize(o.quiet, testOut, fmt.Sprintf(", %d tests, %d failed", tot, fail))

	// ── Render coverage report ─────────────────────────────────────────────
	covRenderer := covhtml.Renderer{}
	covOpts := covhtml.Options{
		Title:         o.title,
		ThresholdLine: o.threshold,
		TestReportURL: testFilename,
		Delta:         d,
		BaselineLabel: o.baselineLabel,
	}
	if err := writeFile(covOut, func(w io.Writer) error {
		return covRenderer.Render(w, covRep, covOpts)
	}); err != nil {
		return fmt.Errorf("render coverage: %w", err)
	}
	lt, lc, _, _, _, _ := covRep.Stats()
	logSize(o.quiet, covOut, fmt.Sprintf(", %d files, %.1f%% lines", len(covRep.Files), covRep.LinePct()))

	// ── Save combined snapshot ─────────────────────────────────────────────
	if o.saveSnapshot != "" {
		snap := &delta.Snapshot{
			Version:   1,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Label:     o.title,
			Tests:     delta.FromTestReport(merged),
			Coverage:  delta.FromCovReport(covRep),
		}
		if err := delta.Save(o.saveSnapshot, snap); err != nil {
			return err
		}
		if !o.quiet {
			fmt.Fprintf(os.Stderr, "snapshot %s\n", o.saveSnapshot)
		}
	}

	// ── Open browser ───────────────────────────────────────────────────────
	if o.open {
		openBrowser(testOut)
		openBrowser(covOut)
	}

	// ── CI exit codes ──────────────────────────────────────────────────────
	exitCode := 0
	if o.failTests && fail > 0 {
		fmt.Fprintf(os.Stderr, "FAIL: %d test(s) failed\n", fail)
		exitCode = 1
	}
	if o.failCov {
		linePct := covRep.LinePct()
		if o.threshold > 0 && linePct < o.threshold {
			fmt.Fprintf(os.Stderr, "FAIL: line coverage %.1f%% is below threshold %.1f%%\n",
				linePct, o.threshold)
			exitCode = 1
		} else if o.threshold == 0 {
			// threshold=0 + --fail-cov: fail only if no coverage at all
			if lc == 0 && lt > 0 {
				fmt.Fprintf(os.Stderr, "FAIL: zero line coverage\n")
				exitCode = 1
			}
		}
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}

	return nil
}
