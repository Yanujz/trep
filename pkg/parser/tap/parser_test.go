package tap_test

import (
	"strings"
	"testing"
	"time"

	_ "github.com/Yanujz/trep/pkg/parser/tap"

	"github.com/Yanujz/trep/pkg/model"
	"github.com/Yanujz/trep/pkg/parser"
)

func parse(t *testing.T, input string) *model.Report {
	t.Helper()
	p, err := parser.ForName("tap")
	if err != nil {
		t.Fatalf("ForName: %v", err)
	}
	rep, err := p.Parse(strings.NewReader(input), "test.tap")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return rep
}

func TestTAP_BasicPassFailSkip(t *testing.T) {
	input := "TAP version 12\n1..3\nok 1 - First\nnot ok 2 - Second\nok 3 - Third # SKIP not implemented\n"
	rep := parse(t, input)

	total, passed, failed, skipped := rep.Stats()
	if total != 3 || passed != 1 || failed != 1 || skipped != 1 {
		t.Errorf("Stats = %d/%d/%d/%d, want 3/1/1/1", total, passed, failed, skipped)
	}

	cases := rep.Suites[0].Cases
	if cases[0].Status != model.StatusPass {
		t.Errorf("test 1 status = %v, want pass", cases[0].Status)
	}
	if cases[1].Status != model.StatusFail {
		t.Errorf("test 2 status = %v, want fail", cases[1].Status)
	}
	if cases[2].Status != model.StatusSkip {
		t.Errorf("test 3 status = %v, want skip", cases[2].Status)
	}
	if cases[2].Message != "not implemented" {
		t.Errorf("skip message = %q, want 'not implemented'", cases[2].Message)
	}
}

func TestTAP_TimeAnnotation(t *testing.T) {
	input := "TAP version 12\n1..1\nok 1 - Fast test # time=1.5\n"
	rep := parse(t, input)

	cases := rep.Suites[0].Cases
	if cases[0].Duration != 1500*time.Millisecond {
		t.Errorf("duration = %v, want 1.5s", cases[0].Duration)
	}
	if rep.Duration != 1500*time.Millisecond {
		t.Errorf("report duration = %v, want 1.5s", rep.Duration)
	}
}

func TestTAP_TimeAccumulated(t *testing.T) {
	input := "1..2\nok 1 - A # time=1.0\nok 2 - B # time=2.0\n"
	rep := parse(t, input)
	if rep.Duration != 3*time.Second {
		t.Errorf("report duration = %v, want 3s", rep.Duration)
	}
}

func TestTAP_YAML13BlockSkipped(t *testing.T) {
	input := "TAP version 13\n1..2\nok 1 - First\n  ---\n  message: details\n  ...\nok 2 - Second\n"
	rep := parse(t, input)

	total, _, _, _ := rep.Stats()
	if total != 2 {
		t.Errorf("expected 2 tests (YAML block should be skipped), got %d", total)
	}
}

func TestTAP_PlanFirstNoVersionHeader(t *testing.T) {
	input := "1..2\nok 1 - A\nok 2 - B\n"
	rep := parse(t, input)

	total, passed, _, _ := rep.Stats()
	if total != 2 || passed != 2 {
		t.Errorf("Stats = %d/%d, want 2/2", total, passed)
	}
}

func TestTAP_EmptyDescriptionBecomesTest(t *testing.T) {
	input := "1..1\nok 1\n"
	rep := parse(t, input)
	cases := rep.Suites[0].Cases
	if cases[0].Name == "" {
		t.Error("empty description should default to 'test'")
	}
}

func TestTAP_FailureNote(t *testing.T) {
	input := "1..1\nnot ok 1 - MyTest # failed assertion\n"
	rep := parse(t, input)
	tc := rep.Suites[0].Cases[0]
	if tc.Status != model.StatusFail {
		t.Errorf("status = %v, want fail", tc.Status)
	}
	if tc.Message != "failed assertion" {
		t.Errorf("message = %q, want 'failed assertion'", tc.Message)
	}
}

func TestTAP_Detect(t *testing.T) {
	p, _ := parser.ForName("tap")
	if !p.Detect([]byte("TAP version 12\n")) {
		t.Error("should detect TAP version header")
	}
	if !p.Detect([]byte("1..10\n")) {
		t.Error("should detect TAP plan line")
	}
	if p.Detect([]byte("<testsuites/>")) {
		t.Error("should not detect XML as TAP")
	}
}

func TestTAP_SourceName(t *testing.T) {
	rep := parse(t, "1..0\n")
	if rep.Suites[0].Name != "test.tap" {
		t.Errorf("suite name = %q, want source name", rep.Suites[0].Name)
	}
}
