package gocover_test

import (
	"strings"
	"testing"

	_ "github.com/trep-dev/trep/pkg/coverage/parser/gocover"

	covparser "github.com/trep-dev/trep/pkg/coverage/parser"
)

func parse(t *testing.T, input string) interface {
	Stats() (int, int, int, int, int, int)
} {
	t.Helper()
	p, err := covparser.ForName("gocover")
	if err != nil {
		t.Fatalf("ForName: %v", err)
	}
	rep, err := p.Parse(strings.NewReader(input), "coverage.out")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return rep
}

func parseRep(t *testing.T, input string) interface {
	Stats() (int, int, int, int, int, int)
	LinePct() float64
} {
	t.Helper()
	p, _ := covparser.ForName("gocover")
	rep, err := p.Parse(strings.NewReader(input), "coverage.out")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return rep
}

const basicCover = `mode: set
github.com/example/pkg/file.go:1.20,3.5 2 1
github.com/example/pkg/file.go:5.10,7.3 1 0
`

func TestGocover_BasicParsing(t *testing.T) {
	p, _ := covparser.ForName("gocover")
	rep, err := p.Parse(strings.NewReader(basicCover), "coverage.out")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(rep.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(rep.Files))
	}
	f := rep.Files[0]
	if f.Path != "github.com/example/pkg/file.go" {
		t.Errorf("Path = %q", f.Path)
	}
	if f.LinesTotal == 0 {
		t.Error("LinesTotal should be > 0")
	}
	if f.LinesCovered == 0 {
		t.Error("LinesCovered should be > 0 (first block has count=1)")
	}
}

func TestGocover_UncoveredBlock(t *testing.T) {
	input := "mode: set\npkg/file.go:1.5,2.3 1 0\n"
	p, _ := covparser.ForName("gocover")
	rep, _ := p.Parse(strings.NewReader(input), "cov.out")
	if rep.Files[0].LinesCovered != 0 {
		t.Error("block with count=0 should contribute 0 covered lines")
	}
}

func TestGocover_CountMode(t *testing.T) {
	input := "mode: count\npkg/file.go:10.5,12.3 2 5\n"
	p, _ := covparser.ForName("gocover")
	rep, _ := p.Parse(strings.NewReader(input), "cov.out")
	if rep.Files[0].LinesCovered == 0 {
		t.Error("mode: count block with count>0 should have covered lines")
	}
}

func TestGocover_AtomicMode(t *testing.T) {
	input := "mode: atomic\npkg/file.go:1.5,3.3 2 10\n"
	p, _ := covparser.ForName("gocover")
	rep, _ := p.Parse(strings.NewReader(input), "cov.out")
	if len(rep.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(rep.Files))
	}
}

func TestGocover_MultipleFiles(t *testing.T) {
	input := "mode: set\npkg/a.go:1.5,2.3 1 1\npkg/b.go:1.5,2.3 1 0\n"
	p, _ := covparser.ForName("gocover")
	rep, _ := p.Parse(strings.NewReader(input), "cov.out")
	if len(rep.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(rep.Files))
	}
}

func TestGocover_LinesDeduped(t *testing.T) {
	// Two overlapping blocks that cover the same line range.
	input := "mode: count\npkg/file.go:1.5,3.3 2 1\npkg/file.go:1.5,3.3 2 2\n"
	p, _ := covparser.ForName("gocover")
	rep, _ := p.Parse(strings.NewReader(input), "cov.out")
	// Lines 1,2,3 should appear only once despite two blocks.
	seenLines := make(map[int]int)
	for _, l := range rep.Files[0].Lines {
		seenLines[l.Number]++
	}
	for ln, count := range seenLines {
		if count > 1 {
			t.Errorf("line %d appeared %d times, should be deduplicated", ln, count)
		}
	}
}

func TestGocover_EmptyInput(t *testing.T) {
	p, _ := covparser.ForName("gocover")
	rep, err := p.Parse(strings.NewReader("mode: set\n"), "cov.out")
	if err != nil {
		t.Fatalf("unexpected error on empty input: %v", err)
	}
	if len(rep.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(rep.Files))
	}
}

func TestGocover_MalformedLineSkipped(t *testing.T) {
	input := "mode: set\nthis is garbage\npkg/file.go:1.5,2.3 1 1\n"
	p, _ := covparser.ForName("gocover")
	rep, err := p.Parse(strings.NewReader(input), "cov.out")
	if err != nil {
		t.Fatalf("malformed line should be skipped, not error: %v", err)
	}
	if len(rep.Files) != 1 {
		t.Errorf("expected 1 file after skipping garbage line, got %d", len(rep.Files))
	}
}

func TestGocover_Detect(t *testing.T) {
	p, _ := covparser.ForName("gocover")
	if !p.Detect([]byte("mode: set\n")) {
		t.Error("should detect mode: set")
	}
	if !p.Detect([]byte("mode: count\n")) {
		t.Error("should detect mode: count")
	}
	if !p.Detect([]byte("mode: atomic\n")) {
		t.Error("should detect mode: atomic")
	}
	if p.Detect([]byte("<coverage>")) {
		t.Error("should not detect XML")
	}
}
