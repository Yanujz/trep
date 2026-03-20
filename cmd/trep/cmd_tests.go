package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/trep-dev/trep/pkg/delta"
	"github.com/trep-dev/trep/pkg/model"
	"github.com/trep-dev/trep/pkg/parser"
	"github.com/trep-dev/trep/pkg/render/annotations"
	jsonrender "github.com/trep-dev/trep/pkg/render/json"
	htmlrender "github.com/trep-dev/trep/pkg/render/html"
)

type testOpts struct {
	output        string
	outFormat     string // "html" | "json"
	format        string
	title         string
	noMerge       bool
	failCI        bool
	open          bool
	quiet         bool
	annotate      bool
	annotatePlatform string
	saveSnapshot  string
	baseline      string
	baselineLabel string
	// set by report command when producing a linked pair
	covReportURL string
}

func newTestCmd() *cobra.Command {
	o := &testOpts{}

	cmd := &cobra.Command{
		Use:   "test [flags] <input>...",
		Short: "Generate a report from test result files",
		Long: `Parse one or more test result files and produce a searchable,
filterable, self-contained HTML (or JSON) report.

Supported input formats
  junit    JUnit XML  (CTest, Maven Surefire, pytest-junit, …)
  gtest    Google Test XML   (alias for junit)
  gotest   go test -json streaming output
  tap      TAP v12/13

Format is auto-detected from file extension and content when --format is omitted.
Use '-' as an input path to read from stdin.

Examples
  trep test results.xml
  trep test -o report.html a.xml b.xml c.xml
  trep test --output-format json results.xml | jq .summary
  trep test --annotate results.xml            # GitHub / GitLab annotations
  trep test --no-merge suite1.xml suite2.xml
  go test -json ./... | trep test - -t "Unit Tests"
  trep test --fail --save-snapshot snap.json results.xml
  trep test --baseline prev.json --baseline-label main results.xml`,
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		RunE:         o.run,
	}

	f := cmd.Flags()
	f.StringVarP(&o.output,   "output",        "o", "",     "output file (default: first input .html or .json; '-' for stdout)")
	f.StringVar (&o.outFormat,"output-format",       "html", "output format: html | json")
	f.StringVarP(&o.format,   "format",        "f", "auto", "force input format: auto | junit | gtest | gotest | tap")
	f.StringVarP(&o.title,    "title",         "t", "",     "report title")
	f.BoolVar   (&o.noMerge,  "no-merge",            false, "one report per input instead of merging")
	f.BoolVar   (&o.failCI,   "fail",                false, "exit 1 when any tests failed")
	f.BoolVar   (&o.open,     "open",                false, "open the report in the browser after writing")
	f.BoolVarP  (&o.quiet,    "quiet",         "q", false,  "suppress progress output")
	f.BoolVar   (&o.annotate, "annotate",            false, "emit CI annotations for failed tests (GitHub/GitLab auto-detected)")
	f.StringVar (&o.annotatePlatform, "annotate-platform", "auto", "annotation platform: auto | github | gitlab")
	f.StringVar (&o.saveSnapshot, "save-snapshot", "",       "write run snapshot JSON for future delta comparison")
	f.StringVar (&o.baseline,     "baseline",      "",       "JSON snapshot from a previous run (enables delta badges)")
	f.StringVar (&o.baselineLabel,"baseline-label","",       "human label for the baseline (e.g. 'main')")

	return cmd
}

func (o *testOpts) run(_ *cobra.Command, args []string) error {
	if o.outFormat != "html" && o.outFormat != "json" {
		return fmt.Errorf("unknown --output-format %q: must be html or json", o.outFormat)
	}
	if o.annotate {
		switch o.annotatePlatform {
		case "auto", "github", "gitlab":
		default:
			return fmt.Errorf("unknown --annotate-platform %q: must be auto, github, or gitlab", o.annotatePlatform)
		}
	}

	reports, err := o.parseInputs(args)
	if err != nil {
		return err
	}

	// Annotations (written to stderr so they don't pollute stdout/file output).
	if o.annotate {
		p := annotations.Platform(o.annotatePlatform)
		for _, rep := range reports {
			if err := annotations.WriteTestAnnotations(os.Stderr, rep, p); err != nil {
				return err
			}
		}
	}

	// Load baseline snapshot for delta.
	var base *delta.Snapshot
	if o.baseline != "" {
		base, err = delta.Load(o.baseline)
		if err != nil {
			return err
		}
	}

	anyFailed := false
	renderer  := htmlrender.Renderer{}

	for i, rep := range reports {
		_, _, failed, _ := rep.Stats()
		if failed > 0 {
			anyFailed = true
		}

		ext := ".html"
		if o.outFormat == "json" {
			ext = ".json"
		}
		outPath := o.resolveOutput(args, rep, i, ext)

		if o.outFormat == "json" {
			if err := writeFile(outPath, func(w io.Writer) error {
				return jsonrender.RenderTest(w, rep)
			}); err != nil {
				return fmt.Errorf("render json %s: %w", outPath, err)
			}
			tot, _, fail, _ := rep.Stats()
			logSize(o.quiet, outPath, fmt.Sprintf(", %d tests, %d failed", tot, fail))
		} else {
			var d *delta.Delta
			if base != nil {
				cur := &delta.Snapshot{
					Version:   1,
					Timestamp: time.Now().UTC().Format(time.RFC3339),
					Tests:     delta.FromTestReport(rep),
				}
				d = delta.Compute(base, cur)
			}
			opts := htmlrender.Options{
				CovReportURL:  o.covReportURL,
				Delta:         d,
				BaselineLabel: o.baselineLabel,
			}
			if err := writeFile(outPath, func(w io.Writer) error {
				return renderer.Render(w, rep, opts)
			}); err != nil {
				return fmt.Errorf("render html %s: %w", outPath, err)
			}
			tot, _, fail, _ := rep.Stats()
			logSize(o.quiet, outPath, fmt.Sprintf(", %d tests, %d failed", tot, fail))
		}

		if o.open && outPath != "-" {
			openBrowser(outPath)
		}
	}

	// Save snapshot.
	if o.saveSnapshot != "" && len(reports) > 0 {
		snap := &delta.Snapshot{
			Version:   1,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Label:     o.title,
			Tests:     delta.FromTestReport(reports[0]),
		}
		if err := delta.Save(o.saveSnapshot, snap); err != nil {
			return err
		}
		if !o.quiet {
			fmt.Fprintf(os.Stderr, "snapshot %s\n", o.saveSnapshot)
		}
	}

	if o.failCI && anyFailed {
		os.Exit(1)
	}
	return nil
}

func (o *testOpts) parseInputs(args []string) ([]*model.Report, error) {
	var forced parser.Parser
	if o.format != "auto" {
		var err error
		forced, err = parser.ForName(o.format)
		if err != nil {
			return nil, err
		}
	}

	reports := make([]*model.Report, 0, len(args))
	for _, path := range args {
		if !o.quiet {
			fmt.Fprintf(os.Stderr, "parsing  %s\n", path)
		}
		rep, err := parser.ParseFile(path, forced)
		if err != nil {
			return nil, err
		}
		if !o.quiet {
			tot, pass, fail, skip := rep.Stats()
			fmt.Fprintf(os.Stderr, "         total=%-5d  pass=%-5d  fail=%-5d  skip=%d\n",
				tot, pass, fail, skip)
		}
		reports = append(reports, rep)
	}

	if !o.noMerge && len(reports) > 1 {
		base := reports[0]
		for _, r := range reports[1:] {
			base.Merge(r)
		}
		reports = reports[:1]
	}

	if o.title != "" {
		for _, r := range reports {
			r.Title = o.title
		}
	}
	return reports, nil
}

func (o *testOpts) resolveOutput(args []string, rep *model.Report, idx int, ext string) string {
	if o.output != "" {
		return o.output
	}
	if len(args) == 1 && args[0] == "-" {
		return "-"
	}
	if len(rep.Sources) == 0 {
		return "report" + ext
	}
	base := rep.Sources[0]
	if base == "<stdin>" {
		base = "report"
	}
	if !o.noMerge && len(args) > 1 {
		base = "report"
	}
	out := replaceExt(base, ext)
	if o.noMerge && idx > 0 {
		out = fmt.Sprintf("%s_%02d%s", stripExt(base), idx+1, ext)
	}
	return out
}
