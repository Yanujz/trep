package gotest_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Yanujz/trep/pkg/model"
	"github.com/Yanujz/trep/pkg/parser/gotest"
)

var p = gotest.Parser{}

func mustParse(t *testing.T, input string) *model.Report {
	t.Helper()
	rep, err := p.Parse(strings.NewReader(input), "test")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	return rep
}

const singlePass = `{"Action":"run","Package":"pkg","Test":"TestFoo"}
{"Action":"pass","Package":"pkg","Test":"TestFoo","Elapsed":0.001}
`

func TestSinglePassingTest(t *testing.T) {
	rep := mustParse(t, singlePass)

	if len(rep.Suites) != 1 {
		t.Fatalf("expected 1 suite, got %d", len(rep.Suites))
	}
	cases := rep.Suites[0].Cases
	if len(cases) != 1 {
		t.Fatalf("expected 1 case, got %d", len(cases))
	}
	if cases[0].Status != model.StatusPass {
		t.Errorf("status = %v, want pass", cases[0].Status)
	}
}

const singleFail = `{"Action":"run","Package":"pkg","Test":"TestFail"}
{"Action":"output","Package":"pkg","Test":"TestFail","Output":"    FAIL: expected 5 got 4\n"}
{"Action":"fail","Package":"pkg","Test":"TestFail","Elapsed":0.002}
`

func TestSingleFailingTest(t *testing.T) {
	rep := mustParse(t, singleFail)

	c := rep.Suites[0].Cases[0]
	if c.Status != model.StatusFail {
		t.Errorf("status = %v, want fail", c.Status)
	}
	if !strings.Contains(c.Message, "expected 5 got 4") {
		t.Errorf("message = %q, should contain failure text", c.Message)
	}
	if c.Stdout == "" {
		t.Error("stdout should be non-empty for failed test")
	}
}

func TestSkipWithReason(t *testing.T) {
	input := `{"Action":"run","Package":"pkg","Test":"TestSkip"}
{"Action":"output","Package":"pkg","Test":"TestSkip","Output":"    t.Skip(\"not ready\")\n"}
{"Action":"skip","Package":"pkg","Test":"TestSkip","Elapsed":0.001}
`
	rep := mustParse(t, input)

	c := rep.Suites[0].Cases[0]
	if c.Status != model.StatusSkip {
		t.Errorf("status = %v, want skip", c.Status)
	}
	want := `t.Skip("not ready")`
	if c.Message != want {
		t.Errorf("message = %q, want %q", c.Message, want)
	}
}

const multiPkg = `{"Action":"run","Package":"pkg1","Test":"TestA"}
{"Action":"pass","Package":"pkg1","Test":"TestA","Elapsed":0.001}
{"Action":"run","Package":"pkg2","Test":"TestB"}
{"Action":"pass","Package":"pkg2","Test":"TestB","Elapsed":0.001}
`

func TestMultiplePackages(t *testing.T) {
	rep := mustParse(t, multiPkg)

	if len(rep.Suites) != 2 {
		t.Fatalf("expected 2 suites, got %d", len(rep.Suites))
	}
	names := map[string]bool{}
	for _, s := range rep.Suites {
		names[s.Name] = true
	}
	for _, want := range []string{"pkg1", "pkg2"} {
		if !names[want] {
			t.Errorf("missing suite %q", want)
		}
	}
}

func TestPackageLevelEventsIgnored(t *testing.T) {
	const input = `{"Action":"start","Package":"pkg"}
{"Action":"run","Package":"pkg","Test":"TestA"}
{"Action":"pass","Package":"pkg","Test":"TestA","Elapsed":0.001}
`
	rep := mustParse(t, input)

	total, _, _, _ := rep.Stats()
	if total != 1 {
		t.Errorf("total = %d, want 1 (package-level events should not create cases)", total)
	}
}

func TestGarbledLinesSkipped(t *testing.T) {
	const input = `not valid json at all
{"Action":"run","Package":"pkg","Test":"TestA"}
{"Action":"pass","Package":"pkg","Test":"TestA","Elapsed":0.001}
`
	rep, err := p.Parse(strings.NewReader(input), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rep.Suites) != 1 {
		t.Fatalf("expected 1 suite, got %d", len(rep.Suites))
	}
}

func TestMaxElapsedDrivesDuration(t *testing.T) {
	const input = `{"Action":"run","Package":"pkg","Test":"TestA"}
{"Action":"pass","Package":"pkg","Test":"TestA","Elapsed":0.1}
{"Action":"run","Package":"pkg","Test":"TestB"}
{"Action":"pass","Package":"pkg","Test":"TestB","Elapsed":0.5}
`
	rep := mustParse(t, input)

	want := 500 * time.Millisecond
	if rep.Duration != want {
		t.Errorf("duration = %v, want %v", rep.Duration, want)
	}
}

func TestLongOutputCappedInMessage(t *testing.T) {
	// Build a single output line longer than 2048 chars.
	longLine := strings.Repeat("x", 2049)
	input := fmt.Sprintf(
		`{"Action":"run","Package":"pkg","Test":"TestLong"}`+"\n"+
			`{"Action":"output","Package":"pkg","Test":"TestLong","Output":%q}`+"\n"+
			`{"Action":"fail","Package":"pkg","Test":"TestLong","Elapsed":0.001}`+"\n",
		longLine+"\n",
	)
	rep := mustParse(t, input)

	c := rep.Suites[0].Cases[0]
	const maxLen = 2048
	if len(c.Message) > maxLen+10 { // +10 for the "\n…" suffix
		t.Errorf("message length %d exceeds cap; should be truncated", len(c.Message))
	}
	if !strings.HasSuffix(c.Message, "\n…") {
		end := len(c.Message)
		start := end - 5
		if start < 0 {
			start = 0
		}
		t.Errorf("message should end with ellipsis when truncated, got suffix: %q", c.Message[start:end])
	}
}
