package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/spf13/cobra"

	covmodel "github.com/Yanujz/trep/pkg/coverage/model"
	covparser "github.com/Yanujz/trep/pkg/coverage/parser"
	covhtml "github.com/Yanujz/trep/pkg/coverage/render/html"
	"github.com/Yanujz/trep/pkg/delta"
	"github.com/Yanujz/trep/pkg/render/annotations"
	jsonrender "github.com/Yanujz/trep/pkg/render/json"
	sarifrender "github.com/Yanujz/trep/pkg/render/sarif"
)

type covOpts struct {
	output           string
	outFormat        string // "html" | "json"
	format           string
	title            string
	thresholdLine    float64
	thresholdBranch  float64
	thresholdFunc    float64
	failCI           bool
	open             bool
	quiet            bool
	stripPrefix      string
	exclude          []string
	annotate         bool
	annotatePlatform string
	saveSnapshot     string
	baseline         string
	baselineLabel    string
	// set by report command when producing a linked pair
	testReportURL string
}

func newCovCmd() *cobra.Command {
	o := &covOpts{}

	cmd := &cobra.Command{
		Use:   "cov [flags] <input>...",
		Short: "Generate an HTML coverage report",
		Long: `Parse one or more coverage files and produce a self-contained HTML report with a
collapsible directory tree, per-metric progress bars, and optional threshold
enforcement. When multiple files are provided they are merged into a single report.

Supported input formats
  lcov      LCOV .info  (gcov, Istanbul/nyc, Rust tarpaulin, …)
  gocover   Go coverprofile  (go test -coverprofile=coverage.out)
  cobertura Cobertura XML  (JaCoCo, coverage.py, .NET coverlet, …)
  clover    Clover XML

Format is auto-detected from file extension and content when --format is omitted.

Thresholds
  --threshold-line, --threshold-branch, --threshold-func set minimums per metric.
  With --fail, trep exits 1 if any enabled threshold is not met.
  A red marker is drawn on the line bar in the HTML report.

Examples
  trep cov coverage.out
  trep cov pkg1.out pkg2.out pkg3.out
  trep cov -o cov.html --threshold-line 80 coverage.info
  trep cov --output-format json coverage.out | jq .summary
  trep cov --annotate --threshold-line 80 coverage.info
  trep cov --fail --save-snapshot snap.json coverage.info
  trep cov --baseline prev.json coverage.out
  trep cov --exclude 'vendor/**' --exclude '**/*_gen.go' coverage.out`,
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		RunE:         o.run,
	}

	f := cmd.Flags()
	f.StringVarP(&o.output, "output", "o", "", "output file (default: input .html or .json; '-' for stdout)")
	f.StringVar(&o.outFormat, "output-format", "html", "output format: html | json | sarif")
	f.StringVarP(&o.format, "format", "f", "auto", "force input format: auto | lcov | gocover | cobertura | clover")
	f.StringVarP(&o.title, "title", "t", "", "report title")
	f.Float64Var(&o.thresholdLine, "threshold-line", 0, "minimum line coverage %   (0 = disabled)")
	f.Float64Var(&o.thresholdBranch, "threshold-branch", 0, "minimum branch coverage % (0 = disabled)")
	f.Float64Var(&o.thresholdFunc, "threshold-func", 0, "minimum function coverage %(0 = disabled)")
	// Convenience alias: --threshold sets line threshold.
	f.Float64Var(&o.thresholdLine, "threshold", 0, "alias for --threshold-line")
	f.BoolVar(&o.failCI, "fail", false, "exit 1 when any threshold is not met")
	f.BoolVar(&o.open, "open", false, "open the report in the browser after writing")
	f.BoolVarP(&o.quiet, "quiet", "q", false, "suppress progress output")
	f.StringVar(&o.stripPrefix, "strip-prefix", "", "remove this prefix from all file paths")
	f.StringArrayVar(&o.exclude, "exclude", nil, "glob pattern for paths to exclude (repeatable, e.g. 'vendor/**')")
	f.BoolVar(&o.annotate, "annotate", false, "emit CI annotations for files below threshold")
	f.StringVar(&o.annotatePlatform, "annotate-platform", "auto", "annotation platform: auto | github | gitlab")
	f.StringVar(&o.saveSnapshot, "save-snapshot", "", "write run snapshot JSON for future delta comparison")
	f.StringVar(&o.baseline, "baseline", "", "JSON snapshot from a previous run (enables delta badges)")
	f.StringVar(&o.baselineLabel, "baseline-label", "", "human label for the baseline")

	return cmd
}

func (o *covOpts) run(_ *cobra.Command, args []string) error {
	if o.outFormat != "html" && o.outFormat != "json" && o.outFormat != "sarif" {
		return fmt.Errorf("unknown --output-format %q: must be html, json, or sarif", o.outFormat)
	}
	if o.annotate {
		switch o.annotatePlatform {
		case "auto", "github", "gitlab":
		default:
			return fmt.Errorf("unknown --annotate-platform %q: must be auto, github, or gitlab", o.annotatePlatform)
		}
	}
	if err := validateThreshold("--threshold-line", o.thresholdLine); err != nil {
		return err
	}
	if err := validateThreshold("--threshold-branch", o.thresholdBranch); err != nil {
		return err
	}
	if err := validateThreshold("--threshold-func", o.thresholdFunc); err != nil {
		return err
	}

	var forced covparser.CovParser
	if o.format != "auto" {
		var err error
		forced, err = covparser.ForName(o.format)
		if err != nil {
			return err
		}
	}

	// Parse all input files using an error group.
	reports := make([]*covmodel.CovReport, len(args))
	eg := new(errgroup.Group)
	eg.SetLimit(runtime.NumCPU())
	var mu sync.Mutex

	for i, p := range args {
		idx, path := i, p
		eg.Go(func() error {
			if !o.quiet {
				mu.Lock()
				fmt.Fprintf(os.Stderr, "parsing  %s\n", path)
				mu.Unlock()
			}
			r, err := covparser.ParseFile(path, forced, o.stripPrefix)
			if err != nil {
				return err
			}
			reports[idx] = r
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	// Merge all reports.
	var rep *covmodel.CovReport
	for _, r := range reports {
		if rep == nil {
			rep = r
		} else {
			rep.Merge(r)
		}
	}

	// Apply --exclude patterns before rendering or threshold checks.
	if len(o.exclude) > 0 {
		rep.Files = excludeFiles(rep.Files, o.exclude)
	}

	if !o.quiet {
		lt, lc, bt, bc, ft, fc := rep.Stats()
		fmt.Fprintf(os.Stderr,
			"         files=%-4d  lines=%d/%d (%.1f%%)  branches=%d/%d  funcs=%d/%d\n",
			len(rep.Files), lc, lt, rep.LinePct(), bc, bt, fc, ft)
	}

	// Use the first input path as the basis for the default output name.
	inputPath := args[0]

	// Annotations.
	if o.annotate {
		p := annotations.Platform(o.annotatePlatform)
		if err := annotations.WriteCovAnnotations(os.Stderr, rep, o.thresholdLine, p); err != nil {
			return err
		}
	}

	// Delta.
	var base *delta.Snapshot
	if o.baseline != "" {
		var err error
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
			Coverage:  delta.FromCovReport(rep),
		}
		d = delta.Compute(base, cur)
	}

	// Output path.
	ext := ".html"
	switch o.outFormat {
	case "json":
		ext = ".json"
	case "sarif":
		ext = ".sarif"
	}
	outPath := o.output
	if outPath == "" {
		if inputPath == "-" {
			outPath = "-"
		} else {
			outPath = replaceExt(inputPath, ext)
		}
	}

	switch o.outFormat {
	case "json":
		if err := writeFile(outPath, func(w io.Writer) error {
			return jsonrender.RenderCov(w, rep)
		}); err != nil {
			return fmt.Errorf("render json %s: %w", outPath, err)
		}
	case "sarif":
		if err := writeFile(outPath, func(w io.Writer) error {
			return sarifrender.RenderCov(w, rep, o.thresholdLine, version)
		}); err != nil {
			return fmt.Errorf("render sarif %s: %w", outPath, err)
		}
	default:
		opts := covhtml.Options{
			Title:           o.title,
			ThresholdLine:   o.thresholdLine,
			ThresholdBranch: o.thresholdBranch,
			ThresholdFunc:   o.thresholdFunc,
			TestReportURL:   o.testReportURL,
			Delta:           d,
			BaselineLabel:   o.baselineLabel,
		}
		renderer := covhtml.Renderer{}
		if err := writeFile(outPath, func(w io.Writer) error {
			return renderer.Render(w, rep, opts)
		}); err != nil {
			return fmt.Errorf("render html %s: %w", outPath, err)
		}
	}

	lt, lc, _, _, _, _ := rep.Stats()
	logSize(o.quiet, outPath, fmt.Sprintf(", %d files, %d/%d lines (%.1f%%)",
		len(rep.Files), lc, lt, rep.LinePct()))

	if o.open && outPath != "-" {
		openBrowser(outPath)
	}

	// Save snapshot.
	if o.saveSnapshot != "" {
		snap := &delta.Snapshot{
			Version:   1,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Label:     o.title,
			Coverage:  delta.FromCovReport(rep),
		}
		if err := delta.Save(o.saveSnapshot, snap); err != nil {
			return err
		}
		if !o.quiet {
			fmt.Fprintf(os.Stderr, "snapshot %s\n", o.saveSnapshot)
		}
	}

	// CI threshold enforcement.
	if o.failCI {
		failed := false
		if o.thresholdLine > 0 && rep.LinePct() < o.thresholdLine {
			fmt.Fprintf(os.Stderr, "FAIL: line coverage %.1f%% < threshold %.1f%%\n",
				rep.LinePct(), o.thresholdLine)
			failed = true
		}
		if o.thresholdBranch > 0 && rep.BranchPct() < o.thresholdBranch {
			fmt.Fprintf(os.Stderr, "FAIL: branch coverage %.1f%% < threshold %.1f%%\n",
				rep.BranchPct(), o.thresholdBranch)
			failed = true
		}
		if o.thresholdFunc > 0 && rep.FuncPct() < o.thresholdFunc {
			fmt.Fprintf(os.Stderr, "FAIL: function coverage %.1f%% < threshold %.1f%%\n",
				rep.FuncPct(), o.thresholdFunc)
			failed = true
		}
		if failed {
			os.Exit(1)
		}
	}

	return nil
}

// excludeFiles returns a copy of files with any entry whose path matches one
// of the glob patterns removed. Patterns follow filepath.Match semantics;
// a pattern ending in "/**" is treated as a directory prefix match so that
// "vendor/**" excludes all files under vendor/.
func excludeFiles(files []*covmodel.FileCov, patterns []string) []*covmodel.FileCov {
	out := make([]*covmodel.FileCov, 0, len(files))
	for _, f := range files {
		if !matchesAny(f.Path, patterns) {
			out = append(out, f)
		}
	}
	return out
}

func matchesAny(filePath string, patterns []string) bool {
	// Normalise to forward slashes so patterns work on Windows too.
	filePath = filepath.ToSlash(filePath)
	for _, pat := range patterns {
		if strings.HasSuffix(pat, "/**") {
			prefix := strings.TrimSuffix(pat, "/**")
			if strings.HasPrefix(filePath, prefix+"/") || filePath == prefix {
				return true
			}
			continue
		}
		if ok, _ := path.Match(pat, filePath); ok {
			return true
		}
		// Also match against the base name so "*.go" works without a full path.
		if ok, _ := path.Match(pat, path.Base(filePath)); ok {
			return true
		}
	}
	return false
}

// validateThreshold returns an error if v is outside the valid (0, 100] range.
// A zero value means "disabled" and is always accepted.
func validateThreshold(name string, v float64) error {
	if v != 0 && (v < 0 || v > 100) {
		return fmt.Errorf("%s must be between 0 and 100, got %.2f", name, v)
	}
	return nil
}
