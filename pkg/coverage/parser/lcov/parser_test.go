package lcov_test

import (
	"strings"
	"testing"

	_ "github.com/trep-dev/trep/pkg/coverage/parser/lcov"

	covparser "github.com/trep-dev/trep/pkg/coverage/parser"
)

const basicLCOV = `TN:test
SF:src/file.go
FN:10,myFunc
FNDA:5,myFunc
DA:10,5
DA:11,0
BRDA:10,0,0,3
BRDA:10,0,1,2
end_of_record
`

func TestLCOV_BasicFields(t *testing.T) {
	p, _ := covparser.ForName("lcov")
	rep, err := p.Parse(strings.NewReader(basicLCOV), "cov.info")
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
	if f.BranchTotal != 2 || f.BranchCovered != 2 {
		t.Errorf("branches: total=%d covered=%d, want 2/2", f.BranchTotal, f.BranchCovered)
	}
	if f.FuncTotal != 1 || f.FuncCovered != 1 {
		t.Errorf("funcs: total=%d covered=%d, want 1/1", f.FuncTotal, f.FuncCovered)
	}
}

func TestLCOV_FuncHitsMerged(t *testing.T) {
	input := "SF:src/a.go\nFN:10,myFunc\nFNDA:7,myFunc\nDA:10,7\nend_of_record\n"
	p, _ := covparser.ForName("lcov")
	rep, _ := p.Parse(strings.NewReader(input), "cov.info")
	f := rep.Files[0]
	if f.FuncCovered != 1 {
		t.Errorf("FNDA hits should mark func as covered; FuncCovered=%d", f.FuncCovered)
	}
}

func TestLCOV_MultipleFiles(t *testing.T) {
	input := "SF:a.go\nDA:1,1\nend_of_record\nSF:b.go\nDA:1,0\nend_of_record\n"
	p, _ := covparser.ForName("lcov")
	rep, _ := p.Parse(strings.NewReader(input), "cov.info")
	if len(rep.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(rep.Files))
	}
}

func TestLCOV_LastRecordWithoutEndOfRecord(t *testing.T) {
	input := "SF:src/a.go\nDA:1,1\n" // no end_of_record
	p, _ := covparser.ForName("lcov")
	rep, err := p.Parse(strings.NewReader(input), "cov.info")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rep.Files) != 1 {
		t.Errorf("last record without end_of_record should be flushed; got %d files", len(rep.Files))
	}
}

func TestLCOV_BranchUnreachable(t *testing.T) {
	input := "SF:a.go\nBRDA:1,0,0,-\nBRDA:1,0,1,3\nend_of_record\n"
	p, _ := covparser.ForName("lcov")
	rep, _ := p.Parse(strings.NewReader(input), "cov.info")
	f := rep.Files[0]
	// '-' means unreachable → not counted
	if f.BranchTotal != 1 {
		t.Errorf("BranchTotal = %d, want 1 (unreachable excluded)", f.BranchTotal)
	}
}

func TestLCOV_EmptyInput(t *testing.T) {
	p, _ := covparser.ForName("lcov")
	rep, err := p.Parse(strings.NewReader(""), "cov.info")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rep.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(rep.Files))
	}
}

func TestLCOV_Detect(t *testing.T) {
	p, _ := covparser.ForName("lcov")
	if !p.Detect([]byte("TN:\nSF:src/file.go\n")) {
		t.Error("should detect LCOV with TN: prefix")
	}
	if !p.Detect([]byte("SF:src/file.go\n")) {
		t.Error("should detect LCOV with SF: line")
	}
	if p.Detect([]byte("mode: set\n")) {
		t.Error("should not detect Go coverprofile as LCOV")
	}
}
