package json_test

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	covmodel "github.com/Yanujz/trep/pkg/coverage/model"
	"github.com/Yanujz/trep/pkg/model"
	jsonrender "github.com/Yanujz/trep/pkg/render/json"
)

func TestRenderTest_Summary(t *testing.T) {
	rep := &model.Report{
		Timestamp: time.Now().UTC(),
		Suites: []model.Suite{
			{Name: "Suite1", Cases: []model.TestCase{
				{Name: "TestPass", Status: model.StatusPass, Duration: 10 * time.Millisecond},
				{Name: "TestFail", Status: model.StatusFail, Duration: 5 * time.Millisecond, Message: "expected 5 got 4"},
			}},
		},
	}

	var buf bytes.Buffer
	if err := jsonrender.RenderTest(&buf, rep); err != nil {
		t.Fatalf("RenderTest: %v", err)
	}

	var out jsonrender.TestOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	if out.Summary.Total != 2 {
		t.Errorf("Summary.Total = %d, want 2", out.Summary.Total)
	}
	if out.Summary.Passed != 1 {
		t.Errorf("Summary.Passed = %d, want 1", out.Summary.Passed)
	}
	if out.Summary.Failed != 1 {
		t.Errorf("Summary.Failed = %d, want 1", out.Summary.Failed)
	}
	if out.Summary.PassPct != 50.0 {
		t.Errorf("Summary.PassPct = %.1f, want 50.0", out.Summary.PassPct)
	}
	if out.GeneratedAt == "" {
		t.Error("GeneratedAt should be set")
	}
}

func TestRenderTest_SuitesAndCases(t *testing.T) {
	rep := &model.Report{
		Suites: []model.Suite{
			{Name: "MyPkg", Cases: []model.TestCase{
				{Name: "TestX", Status: model.StatusPass, Duration: 12 * time.Millisecond},
			}},
		},
	}

	var buf bytes.Buffer
	_ = jsonrender.RenderTest(&buf, rep)

	var out jsonrender.TestOutput
	_ = json.Unmarshal(buf.Bytes(), &out)

	if len(out.Suites) != 1 {
		t.Fatalf("expected 1 suite, got %d", len(out.Suites))
	}
	if out.Suites[0].Name != "MyPkg" {
		t.Errorf("suite name = %q, want 'MyPkg'", out.Suites[0].Name)
	}
	if len(out.Suites[0].Cases) != 1 {
		t.Fatalf("expected 1 case, got %d", len(out.Suites[0].Cases))
	}
	if out.Suites[0].Cases[0].DurationMs != 12 {
		t.Errorf("DurationMs = %d, want 12", out.Suites[0].Cases[0].DurationMs)
	}
}

func TestRenderTest_EmptyReport(t *testing.T) {
	var buf bytes.Buffer
	_ = jsonrender.RenderTest(&buf, &model.Report{})

	var out jsonrender.TestOutput
	_ = json.Unmarshal(buf.Bytes(), &out)

	if out.Summary.Total != 0 || out.Summary.PassPct != 0 {
		t.Errorf("empty report: Total=%d PassPct=%.1f, want 0/0", out.Summary.Total, out.Summary.PassPct)
	}
}

func TestRenderTest_CaseFileAndLine(t *testing.T) {
	rep := &model.Report{
		Suites: []model.Suite{
			{Cases: []model.TestCase{
				{Name: "T", Status: model.StatusFail, File: "src/foo.cpp", Line: 42},
			}},
		},
	}
	var buf bytes.Buffer
	_ = jsonrender.RenderTest(&buf, rep)

	var out jsonrender.TestOutput
	_ = json.Unmarshal(buf.Bytes(), &out)

	c := out.Suites[0].Cases[0]
	if c.File != "src/foo.cpp" {
		t.Errorf("File = %q, want 'src/foo.cpp'", c.File)
	}
	if c.Line != 42 {
		t.Errorf("Line = %d, want 42", c.Line)
	}
}

func TestRenderCov_Summary(t *testing.T) {
	rep := &covmodel.CovReport{
		Files: []*covmodel.FileCov{
			{Path: "src/a.go", LinesTotal: 10, LinesCovered: 8, BranchTotal: 4, BranchCovered: 3, FuncTotal: 2, FuncCovered: 2},
			{Path: "src/b.go", LinesTotal: 5, LinesCovered: 3},
		},
	}

	var buf bytes.Buffer
	if err := jsonrender.RenderCov(&buf, rep); err != nil {
		t.Fatalf("RenderCov: %v", err)
	}

	var out jsonrender.CovOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out.Summary.LinesTotal != 15 {
		t.Errorf("LinesTotal = %d, want 15", out.Summary.LinesTotal)
	}
	if out.Summary.LinesCov != 11 {
		t.Errorf("LinesCov = %d, want 11", out.Summary.LinesCov)
	}
	if out.Summary.FileCount != 2 {
		t.Errorf("FileCount = %d, want 2", out.Summary.FileCount)
	}
	if len(out.Files) != 2 {
		t.Errorf("expected 2 file entries, got %d", len(out.Files))
	}
}

func TestRenderCov_FileBranchOmittedWhenZero(t *testing.T) {
	rep := &covmodel.CovReport{
		Files: []*covmodel.FileCov{
			{Path: "src/a.go", LinesTotal: 5, LinesCovered: 5},
			// No branch or func data.
		},
	}
	var buf bytes.Buffer
	_ = jsonrender.RenderCov(&buf, rep)

	// Decode raw to check omitempty behaviour.
	var raw map[string]any
	_ = json.Unmarshal(buf.Bytes(), &raw)
	files := raw["files"].([]any)
	file := files[0].(map[string]any)

	if _, ok := file["branch_total"]; ok {
		t.Error("branch_total should be omitted when 0")
	}
	if _, ok := file["func_total"]; ok {
		t.Error("func_total should be omitted when 0")
	}
}
