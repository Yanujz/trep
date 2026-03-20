package clover_test

import (
	"strings"
	"testing"

	_ "github.com/Yanujz/trep/pkg/coverage/parser/clover"

	covparser "github.com/Yanujz/trep/pkg/coverage/parser"
)

const basicClover = `<?xml version="1.0"?>
<coverage clover="3.2.0">
  <project>
    <package name="src">
      <file name="file.go" path="src/file.go">
        <line num="1" type="stmt" count="5"/>
        <line num="2" type="stmt" count="0"/>
        <line num="3" type="method" count="3"/>
      </file>
    </package>
  </project>
</coverage>`

func TestClover_BasicLines(t *testing.T) {
	p, _ := covparser.ForName("clover")
	rep, err := p.Parse(strings.NewReader(basicClover), "clover.xml")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(rep.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(rep.Files))
	}
	f := rep.Files[0]
	if f.Path != "src/file.go" {
		t.Errorf("Path = %q, want 'src/file.go'", f.Path)
	}
	// Lines come from stmt and cond types only; method goes to Funcs.
	if f.LinesTotal != 2 || f.LinesCovered != 1 {
		t.Errorf("lines: total=%d covered=%d, want 2/1", f.LinesTotal, f.LinesCovered)
	}
	if f.FuncTotal != 1 || f.FuncCovered != 1 {
		t.Errorf("funcs: total=%d covered=%d, want 1/1", f.FuncTotal, f.FuncCovered)
	}
}

func TestClover_TopLevelFiles(t *testing.T) {
	xml := `<coverage clover="3.2"><project>
    <file name="root.go" path="root.go">
      <line num="1" type="stmt" count="1"/>
    </file>
  </project></coverage>`
	p, _ := covparser.ForName("clover")
	rep, _ := p.Parse(strings.NewReader(xml), "clover.xml")
	if len(rep.Files) != 1 {
		t.Fatalf("expected 1 top-level file, got %d", len(rep.Files))
	}
}

func TestClover_CondTypeIsLine(t *testing.T) {
	xml := `<coverage clover="1"><project><package name="p">
    <file name="f.go" path="p/f.go">
      <line num="5" type="cond" count="2"/>
    </file>
  </package></project></coverage>`
	p, _ := covparser.ForName("clover")
	rep, _ := p.Parse(strings.NewReader(xml), "c.xml")
	f := rep.Files[0]
	if f.LinesTotal != 1 || f.LinesCovered != 1 {
		t.Errorf("cond type should be treated as a line; lines=%d/%d", f.LinesCovered, f.LinesTotal)
	}
}

func TestClover_FallbackToMetrics(t *testing.T) {
	xml := `<coverage clover="1"><project><package name="p">
    <file name="f.go" path="p/f.go">
      <metrics statements="4" coveredstatements="3" methods="2" coveredmethods="1"/>
    </file>
  </package></project></coverage>`
	p, _ := covparser.ForName("clover")
	rep, _ := p.Parse(strings.NewReader(xml), "c.xml")
	f := rep.Files[0]
	if f.LinesTotal != 4 {
		t.Errorf("metrics fallback: LinesTotal = %d, want 4", f.LinesTotal)
	}
	if f.LinesCovered != 3 {
		t.Errorf("metrics fallback: LinesCovered = %d, want 3", f.LinesCovered)
	}
}

func TestClover_PackageFilePathPriority(t *testing.T) {
	xml := `<coverage clover="1"><project><package name="mypkg">
    <file name="f.go" path="mypkg/f.go">
      <line num="1" type="stmt" count="1"/>
    </file>
  </package></project></coverage>`
	p, _ := covparser.ForName("clover")
	rep, _ := p.Parse(strings.NewReader(xml), "c.xml")
	if rep.Files[0].Path != "mypkg/f.go" {
		t.Errorf("Path = %q, want 'mypkg/f.go'", rep.Files[0].Path)
	}
}

func TestClover_MalformedXML(t *testing.T) {
	p, _ := covparser.ForName("clover")
	_, err := p.Parse(strings.NewReader("<coverage clover><broken"), "bad.xml")
	if err == nil {
		t.Error("expected error for malformed XML")
	}
}

func TestClover_Detect(t *testing.T) {
	p, _ := covparser.ForName("clover")
	if !p.Detect([]byte(`<coverage clover="3.2">`)) {
		t.Error("should detect Clover header")
	}
	if p.Detect([]byte(`<coverage line-rate="0.8">`)) {
		t.Error("should not detect Cobertura as Clover")
	}
	// Regression: a Cobertura file whose path contains the word "clover"
	// must not be detected as a Clover file.
	if p.Detect([]byte(`<coverage line-rate="0.9"><sources><source>/src/clover/report</source>`)) {
		t.Error("should not misdetect Cobertura with 'clover' in path as Clover")
	}
}
