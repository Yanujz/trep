package cobertura_test

import (
	"strings"
	"testing"

	_ "github.com/Yanujz/trep/pkg/coverage/parser/cobertura"

	covparser "github.com/Yanujz/trep/pkg/coverage/parser"
)

const basicCobertura = `<?xml version="1.0"?>
<coverage line-rate="0.8">
  <packages>
    <package name="pkg">
      <classes>
        <class name="MyClass" filename="src/file.go">
          <lines>
            <line number="1" hits="5"/>
            <line number="2" hits="0"/>
          </lines>
        </class>
      </classes>
    </package>
  </packages>
</coverage>`

func TestCobertura_BasicLines(t *testing.T) {
	p, _ := covparser.ForName("cobertura")
	rep, err := p.Parse(strings.NewReader(basicCobertura), "coverage.xml")
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
	if f.LinesTotal != 2 || f.LinesCovered != 1 {
		t.Errorf("lines: total=%d covered=%d, want 2/1", f.LinesTotal, f.LinesCovered)
	}
}

func TestCobertura_MethodFuncs(t *testing.T) {
	xml := `<coverage line-rate="1.0">
  <packages><package name="pkg">
    <classes><class name="C" filename="c.go">
      <methods>
        <method name="MyMethod" line="10" hits="3">
          <lines><line number="11" hits="3"/></lines>
        </method>
      </methods>
    </class></classes>
  </package></packages>
</coverage>`
	p, _ := covparser.ForName("cobertura")
	rep, _ := p.Parse(strings.NewReader(xml), "cov.xml")
	f := rep.Files[0]
	if f.FuncTotal != 1 || f.FuncCovered != 1 {
		t.Errorf("funcs: total=%d covered=%d, want 1/1", f.FuncTotal, f.FuncCovered)
	}
}

func TestCobertura_ClassLinesDeduped(t *testing.T) {
	// Line 11 appears both inside method and at class level — should not be counted twice.
	xml := `<coverage line-rate="1.0">
  <packages><package name="pkg">
    <classes><class name="C" filename="c.go">
      <methods>
        <method name="F" line="10" hits="1">
          <lines><line number="11" hits="1"/></lines>
        </method>
      </methods>
      <lines><line number="11" hits="1"/><line number="12" hits="0"/></lines>
    </class></classes>
  </package></packages>
</coverage>`
	p, _ := covparser.ForName("cobertura")
	rep, _ := p.Parse(strings.NewReader(xml), "cov.xml")
	seenLines := make(map[int]int)
	for _, l := range rep.Files[0].Lines {
		seenLines[l.Number]++
	}
	if seenLines[11] > 1 {
		t.Errorf("line 11 appeared %d times, should be deduplicated", seenLines[11])
	}
}

func TestCobertura_FilenameAttrTakesPriority(t *testing.T) {
	xml := `<coverage line-rate="1.0">
  <packages><package name="pkg">
    <classes><class name="ClassName" filename="real/path.go">
      <lines><line number="1" hits="1"/></lines>
    </class></classes>
  </package></packages>
</coverage>`
	p, _ := covparser.ForName("cobertura")
	rep, _ := p.Parse(strings.NewReader(xml), "cov.xml")
	if rep.Files[0].Path != "real/path.go" {
		t.Errorf("Path = %q, want 'real/path.go'", rep.Files[0].Path)
	}
}

func TestCobertura_FallbackToNameWhenFilenameEmpty(t *testing.T) {
	xml := `<coverage line-rate="1.0">
  <packages><package name="pkg">
    <classes><class name="ClassName" filename="">
      <lines><line number="1" hits="1"/></lines>
    </class></classes>
  </package></packages>
</coverage>`
	p, _ := covparser.ForName("cobertura")
	rep, _ := p.Parse(strings.NewReader(xml), "cov.xml")
	if rep.Files[0].Path != "ClassName" {
		t.Errorf("Path = %q, want 'ClassName' (fallback)", rep.Files[0].Path)
	}
}

func TestCobertura_MalformedXML(t *testing.T) {
	p, _ := covparser.ForName("cobertura")
	_, err := p.Parse(strings.NewReader("<coverage><broken"), "bad.xml")
	if err == nil {
		t.Error("expected error for malformed XML")
	}
}

func TestCobertura_EmptyPackages(t *testing.T) {
	xml := `<coverage line-rate="1.0"><packages/></coverage>`
	p, _ := covparser.ForName("cobertura")
	rep, err := p.Parse(strings.NewReader(xml), "cov.xml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rep.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(rep.Files))
	}
}

func TestCobertura_Detect(t *testing.T) {
	p, _ := covparser.ForName("cobertura")
	if !p.Detect([]byte(`<coverage line-rate="0.8">`)) {
		t.Error("should detect Cobertura header")
	}
	if p.Detect([]byte(`<coverage clover="3.2">`)) {
		t.Error("should not detect Clover as Cobertura (no line-rate)")
	}
	if p.Detect([]byte("mode: set\n")) {
		t.Error("should not detect Go coverprofile")
	}
}
