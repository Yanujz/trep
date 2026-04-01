package main

import (
	"fmt"
	"sort"

	"github.com/Yanujz/trep/pkg/delta"
	"github.com/spf13/cobra"
)

type diffOpts struct {
	baseFile    string
	currentFile string
}

func newDiffCmd() *cobra.Command {
	var o diffOpts
	cmd := &cobra.Command{
		Use:   "diff <base-snapshot.json> <current-snapshot.json>",
		Short: "Compare two snapshot JSON files and print a terminal diff summary",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.baseFile = args[0]
			o.currentFile = args[1]
			return o.run()
		},
	}
	return cmd
}

func (o *diffOpts) run() error {
	baseSnap, err := delta.Load(o.baseFile)
	if err != nil {
		return fmt.Errorf("load baseline: %w", err)
	}

	curSnap, err := delta.Load(o.currentFile)
	if err != nil {
		return fmt.Errorf("load current: %w", err)
	}

	d := delta.Compute(baseSnap, curSnap)
	if d == nil {
		return fmt.Errorf("could not compute delta")
	}

	fmt.Printf("Comparison: %s -> %s\n", o.baseFile, o.currentFile)
	fmt.Println(stringsRepeat("=", 32+len(o.baseFile)+len(o.currentFile)))

	if d.HasTests {
		fmt.Println("Tests:")
		fmt.Printf("  Passed:  %s\n", formatSignedInt(d.PassedDelta, d.PassedDelta < 0))
		fmt.Printf("  Failed:  %s\n", formatSignedInt(d.FailedDelta, d.FailedDelta > 0))
		fmt.Printf("  Skipped: %s\n", formatSignedInt(d.SkippedDelta, false))
		fmt.Printf("  Total:   %s\n", formatSignedInt(d.TotalDelta, false))
		fmt.Println()
	} else if baseSnap.Tests != nil || curSnap.Tests != nil {
		fmt.Printf("Tests: (missing in one of the snapshots)\n\n")
	}

	if d.HasCoverage {
		fmt.Println("Coverage:")
		basePct := baseSnap.Coverage.LinesPct
		curPct := curSnap.Coverage.LinesPct
		arrow := "→"
		if curPct > basePct {
			arrow = "\033[32m↗\033[0m" // Green up-right arrow
		} else if curPct < basePct {
			arrow = "\033[31m↘\033[0m" // Red down-right arrow
		}

		fmt.Printf("  Overall: %.1f%% %s %.1f%% (%s)\n", basePct, arrow, curPct, formatSignedPct(d.LinesPctDelta))

		if len(d.FileDeltas) > 0 {
			fmt.Println("  Files changed:")
			// Sort file paths for stable output
			paths := make([]string, 0, len(d.FileDeltas))
			for p := range d.FileDeltas {
				paths = append(paths, p)
			}
			sort.Strings(paths)

			for _, p := range paths {
				diff := d.FileDeltas[p]
				fmt.Printf("    %s: %s\n", p, formatSignedPct(diff))
			}
		}
		fmt.Println()
	} else if baseSnap.Coverage != nil || curSnap.Coverage != nil {
		fmt.Printf("Coverage: (missing in one of the snapshots)\n\n")
	}

	return nil
}

func formatSignedInt(val int, bad bool) string {
	str := fmt.Sprintf("%d", val)
	if val > 0 {
		str = "+" + str
	}
	if val == 0 {
		return str
	}
	if bad {
		return "\033[31m" + str + "\033[0m" // Red
	} else if val > 0 {
		return "\033[32m" + str + "\033[0m" // Green
	}
	return str
}

func formatSignedPct(val float64) string {
	str := fmt.Sprintf("%.1f%%", val)
	if val > 0 {
		str = "+" + str
	}
	if val == 0 {
		return str
	}
	if val < 0 {
		return "\033[31m" + str + "\033[0m" // Red
	}
	return "\033[32m" + str + "\033[0m" // Green
}

func stringsRepeat(s string, count int) string {
	res := ""
	for i := 0; i < count; i++ {
		res += s
	}
	return res
}
