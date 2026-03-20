package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	// Register test parsers.
	_ "github.com/Yanujz/trep/pkg/parser/gotest"
	_ "github.com/Yanujz/trep/pkg/parser/junit"
	_ "github.com/Yanujz/trep/pkg/parser/tap"

	// Register coverage parsers.
	_ "github.com/Yanujz/trep/pkg/coverage/parser/clover"
	_ "github.com/Yanujz/trep/pkg/coverage/parser/cobertura"
	_ "github.com/Yanujz/trep/pkg/coverage/parser/gocover"
	_ "github.com/Yanujz/trep/pkg/coverage/parser/lcov"
)

const version = "0.1.0"

func main() {
	root := &cobra.Command{
		Use:          "trep",
		Short:        "Generate self-contained HTML reports from test results and coverage data",
		Version:      version,
		SilenceUsage: true,
	}

	root.AddCommand(
		newTestCmd(),
		newCovCmd(),
		newReportCmd(),
		newCompletionCmd(root),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// ── Shared helpers ─────────────────────────────────────────────────────────

func replaceExt(path, ext string) string { return stripExt(path) + ext }

func stripExt(path string) string {
	e := filepath.Ext(path)
	if e == "" {
		return path
	}
	return path[:len(path)-len(e)]
}

func openBrowser(path string) {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", abs)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", abs)
	default:
		cmd = exec.Command("xdg-open", abs)
	}
	_ = cmd.Start()
}

func writeFile(outPath string, fn func(w io.Writer) error) error {
	var w io.Writer
	if outPath == "-" {
		w = os.Stdout
	} else {
		f, err := os.Create(outPath)
		if err != nil {
			return fmt.Errorf("create %s: %w", outPath, err)
		}
		defer f.Close()
		w = f
	}
	return fn(w)
}

func logSize(quiet bool, outPath string, extra string) {
	if quiet || outPath == "-" {
		return
	}
	info, _ := os.Stat(outPath)
	kb := int64(0)
	if info != nil {
		kb = info.Size() / 1024
	}
	fmt.Fprintf(os.Stderr, "wrote    %s  (%d KB%s)\n", outPath, kb, extra)
}
