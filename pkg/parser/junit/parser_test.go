package junit_test

import (
	"strings"
	"testing"
	"time"

	_ "github.com/trep-dev/trep/pkg/parser/junit"

	"github.com/trep-dev/trep/pkg/model"
	"github.com/trep-dev/trep/pkg/parser"
)

func parse(t *testing.T, xml string) *model.Report {
	t.Helper()
	rep, err := parser.ForName("junit")
	if err != nil {
		t.Fatalf("ForName: %v", err)
	}
	result, err := rep.Parse(strings.NewReader(xml), "test.xml")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return result
}

func TestJUnit_PassingTest(t *testing.T) {
	xml := `<?xml version="1.0"?>
<testsuites time="1.5">
  <testsuite name="MySuite">
    <testcase name="TestFoo" classname="com.example.TestSuite" time="0.5"/>
  </testsuite>
</testsuites>`
	rep := parse(t, xml)

	total, passed, failed, skipped := rep.Stats()
	if total != 1 || passed != 1 || failed != 0 || skipped != 0 {
		t.Errorf("Stats = %d/%d/%d/%d, want 1/1/0/0", total, passed, failed, skipped)
	}
	if rep.Duration != 1500*time.Millisecond {
		t.Errorf("Duration = %v, want 1.5s", rep.Duration)
	}
	tc := rep.Suites[0].Cases[0]
	if tc.Duration != 500*time.Millisecond {
		t.Errorf("TestCase.Duration = %v, want 500ms", tc.Duration)
	}
	if tc.Status != model.StatusPass {
		t.Errorf("status = %v, want pass", tc.Status)
	}
}

func TestJUnit_FailureElement(t *testing.T) {
	xml := `<testsuites>
  <testsuite>
    <testcase name="TestDiv" classname="MathTests" time="0.1">
      <failure message="expected 5 got 4">detailed failure output</failure>
    </testcase>
  </testsuite>
</testsuites>`
	rep := parse(t, xml)

	tc := rep.Suites[0].Cases[0]
	if tc.Status != model.StatusFail {
		t.Errorf("status = %v, want fail", tc.Status)
	}
	if tc.Message != "expected 5 got 4" {
		t.Errorf("message = %q, want 'expected 5 got 4'", tc.Message)
	}
}

func TestJUnit_FailureTextFallback(t *testing.T) {
	xml := `<testsuites>
  <testsuite>
    <testcase name="TestX" classname="Pkg">
      <failure>no message attr, text only</failure>
    </testcase>
  </testsuite>
</testsuites>`
	rep := parse(t, xml)
	tc := rep.Suites[0].Cases[0]
	if tc.Status != model.StatusFail {
		t.Errorf("status = %v, want fail", tc.Status)
	}
	if tc.Message != "no message attr, text only" {
		t.Errorf("message = %q, want text content", tc.Message)
	}
}

func TestJUnit_ErrorElement(t *testing.T) {
	xml := `<testsuites>
  <testsuite>
    <testcase name="TestPanic" classname="PkgTests">
      <error message="runtime error: nil pointer dereference"/>
    </testcase>
  </testsuite>
</testsuites>`
	rep := parse(t, xml)
	if rep.Suites[0].Cases[0].Status != model.StatusFail {
		t.Error("error element should map to StatusFail")
	}
}

func TestJUnit_SkippedElement(t *testing.T) {
	xml := `<testsuites>
  <testsuite>
    <testcase name="TestSkipped" classname="PkgTests">
      <skipped message="not implemented yet"/>
    </testcase>
  </testsuite>
</testsuites>`
	rep := parse(t, xml)
	tc := rep.Suites[0].Cases[0]
	if tc.Status != model.StatusSkip {
		t.Errorf("status = %v, want skip", tc.Status)
	}
	if tc.Message != "not implemented yet" {
		t.Errorf("message = %q", tc.Message)
	}
}

func TestJUnit_CTestSkipReasonExtraction(t *testing.T) {
	xml := `<testsuites>
  <testsuite>
    <testcase name="MyTest" classname="CTestSuite">
      <skipped message="SKIP_REGULAR_EXPRESSION_MATCHED"/>
      <system-out>
Output line 1
Skipped: requires network access
Output line 2
      </system-out>
    </testcase>
  </testsuite>
</testsuites>`
	rep := parse(t, xml)
	tc := rep.Suites[0].Cases[0]
	if tc.Status != model.StatusSkip {
		t.Errorf("status = %v, want skip", tc.Status)
	}
	if tc.Message != "requires network access" {
		t.Errorf("message = %q, want 'requires network access'", tc.Message)
	}
}

func TestJUnit_GTestFileAndLine(t *testing.T) {
	xml := `<testsuites>
  <testsuite>
    <testcase name="MyTest" classname="MySuite" file="src/test.cpp" line="42"/>
  </testsuite>
</testsuites>`
	rep := parse(t, xml)
	tc := rep.Suites[0].Cases[0]
	if tc.File != "src/test.cpp" {
		t.Errorf("File = %q, want 'src/test.cpp'", tc.File)
	}
	if tc.Line != 42 {
		t.Errorf("Line = %d, want 42", tc.Line)
	}
}

func TestJUnit_MultipleSuites(t *testing.T) {
	xml := `<testsuites>
  <testsuite>
    <testcase name="T1" classname="Suite1"/>
    <testcase name="T2" classname="Suite2"/>
    <testcase name="T3" classname="Suite3"/>
  </testsuite>
</testsuites>`
	rep := parse(t, xml)
	if len(rep.Suites) != 3 {
		t.Errorf("expected 3 suites, got %d", len(rep.Suites))
	}
}

func TestJUnit_TimestampParsed(t *testing.T) {
	xml := `<testsuites>
  <testsuite timestamp="2024-03-15T10:00:00">
    <testcase name="T1" classname="S1"/>
  </testsuite>
</testsuites>`
	rep := parse(t, xml)
	if rep.Timestamp.IsZero() {
		t.Error("Timestamp should be set from XML attribute")
	}
	if rep.Timestamp.Year() != 2024 {
		t.Errorf("Timestamp year = %d, want 2024", rep.Timestamp.Year())
	}
}

func TestJUnit_SystemOutCaptured(t *testing.T) {
	xml := `<testsuites>
  <testsuite>
    <testcase name="T1" classname="S1">
      <failure message="oops"/>
      <system-out>captured stdout line</system-out>
    </testcase>
  </testsuite>
</testsuites>`
	rep := parse(t, xml)
	tc := rep.Suites[0].Cases[0]
	if tc.Stdout != "captured stdout line" {
		t.Errorf("Stdout = %q, want 'captured stdout line'", tc.Stdout)
	}
}

func TestJUnit_MalformedXML(t *testing.T) {
	p, _ := parser.ForName("junit")
	_, err := p.Parse(strings.NewReader("<testsuites><broken"), "bad.xml")
	if err == nil {
		t.Error("expected error for malformed XML, got nil")
	}
}

func TestJUnit_Detect(t *testing.T) {
	p, _ := parser.ForName("junit")
	if !p.Detect([]byte("<testsuites>")) {
		t.Error("Detect should return true for <testsuites>")
	}
	if !p.Detect([]byte("<testsuite>")) {
		t.Error("Detect should return true for <testsuite>")
	}
	if p.Detect([]byte(`{"Action":"run"}`)) {
		t.Error("Detect should return false for JSON content")
	}
}

func TestJUnit_SourcePreserved(t *testing.T) {
	xml := `<testsuites><testsuite><testcase name="T1" classname="S1"/></testsuite></testsuites>`
	rep := parse(t, xml)
	if len(rep.Sources) != 1 || rep.Sources[0] != "test.xml" {
		t.Errorf("Sources = %v, want [test.xml]", rep.Sources)
	}
}
