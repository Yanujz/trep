package sarif_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	covmodel "github.com/Yanujz/trep/pkg/coverage/model"
	"github.com/Yanujz/trep/pkg/model"
	"github.com/Yanujz/trep/pkg/render/sarif"
)

// ── Helpers ───────────────────────────────────────────────────────────────────

type sarifDoc struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name    string      `json:"name"`
	Version string      `json:"version"`
	Rules   []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID string `json:"id"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   map[string]any  `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifLocation struct {
	PhysicalLocation struct {
		ArtifactLocation struct {
			URI       string `json:"uri"`
			URIBaseID string `json:"uriBaseId"`
		} `json:"artifactLocation"`
		Region *struct {
			StartLine int `json:"startLine"`
		} `json:"region"`
	} `json:"physicalLocation"`
}

func parseDoc(t *testing.T, buf *bytes.Buffer) sarifDoc {
	t.Helper()
	var doc sarifDoc
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("invalid SARIF JSON: %v\n%s", err, buf.String())
	}
	return doc
}

// ── RenderTest ────────────────────────────────────────────────────────────────

func TestRenderTest_Schema(t *testing.T) {
	var buf bytes.Buffer
	rep := &model.Report{}
	if err := sarif.RenderTest(&buf, rep, "1.0.0"); err != nil {
		t.Fatalf("RenderTest error: %v", err)
	}
	doc := parseDoc(t, &buf)
	if doc.Version != "2.1.0" {
		t.Errorf("version = %q, want 2.1.0", doc.Version)
	}
	if !strings.Contains(doc.Schema, "sarif") {
		t.Errorf("$schema = %q, should reference sarif", doc.Schema)
	}
	if len(doc.Runs) != 1 {
		t.Fatalf("runs len = %d, want 1", len(doc.Runs))
	}
	if doc.Runs[0].Tool.Driver.Name != "trep" {
		t.Errorf("driver.name = %q, want trep", doc.Runs[0].Tool.Driver.Name)
	}
	if doc.Runs[0].Tool.Driver.Version != "1.0.0" {
		t.Errorf("driver.version = %q, want 1.0.0", doc.Runs[0].Tool.Driver.Version)
	}
}

func TestRenderTest_NoFailures_EmptyResults(t *testing.T) {
	var buf bytes.Buffer
	rep := &model.Report{
		Suites: []model.Suite{{Cases: []model.TestCase{
			{Name: "TestA", Status: model.StatusPass},
			{Name: "TestB", Status: model.StatusSkip},
		}}},
	}
	if err := sarif.RenderTest(&buf, rep, "dev"); err != nil {
		t.Fatalf("RenderTest error: %v", err)
	}
	doc := parseDoc(t, &buf)
	if len(doc.Runs[0].Results) != 0 {
		t.Errorf("results len = %d, want 0 (no failures)", len(doc.Runs[0].Results))
	}
}

func TestRenderTest_FailedCasesBecome_ErrorResults(t *testing.T) {
	var buf bytes.Buffer
	rep := &model.Report{
		Timestamp: time.Now(),
		Suites: []model.Suite{
			{
				Name: "PkgA",
				Cases: []model.TestCase{
					{Name: "TestPass", Status: model.StatusPass},
					{Name: "TestFail", Status: model.StatusFail, Message: "expected 1 got 2",
						File: "pkg/a_test.go", Line: 42},
					{Name: "TestSkip", Status: model.StatusSkip},
				},
			},
			{
				Name: "PkgB",
				Cases: []model.TestCase{
					{Name: "TestBroken", Status: model.StatusFail},
				},
			},
		},
	}
	if err := sarif.RenderTest(&buf, rep, "dev"); err != nil {
		t.Fatalf("RenderTest error: %v", err)
	}
	doc := parseDoc(t, &buf)
	results := doc.Runs[0].Results
	if len(results) != 2 {
		t.Fatalf("results len = %d, want 2", len(results))
	}
	for _, r := range results {
		if r.RuleID != "test-failure" {
			t.Errorf("ruleId = %q, want test-failure", r.RuleID)
		}
		if r.Level != "error" {
			t.Errorf("level = %q, want error", r.Level)
		}
	}
}

func TestRenderTest_Location_FilledFromTestCase(t *testing.T) {
	var buf bytes.Buffer
	rep := &model.Report{
		Suites: []model.Suite{{Cases: []model.TestCase{
			{Name: "TestX", Status: model.StatusFail,
				File: "pkg/foo_test.go", Line: 77},
		}}},
	}
	if err := sarif.RenderTest(&buf, rep, "dev"); err != nil {
		t.Fatalf("RenderTest error: %v", err)
	}
	doc := parseDoc(t, &buf)
	r := doc.Runs[0].Results[0]
	if len(r.Locations) == 0 {
		t.Fatal("expected at least one location")
	}
	loc := r.Locations[0].PhysicalLocation
	if loc.ArtifactLocation.URI != "pkg/foo_test.go" {
		t.Errorf("uri = %q, want pkg/foo_test.go", loc.ArtifactLocation.URI)
	}
	if loc.Region == nil || loc.Region.StartLine != 77 {
		t.Errorf("startLine = %v, want 77", loc.Region)
	}
}

func TestRenderTest_NoLocation_WhenFileEmpty(t *testing.T) {
	var buf bytes.Buffer
	rep := &model.Report{
		Suites: []model.Suite{{Cases: []model.TestCase{
			{Name: "TestY", Status: model.StatusFail, Message: "panic"},
		}}},
	}
	if err := sarif.RenderTest(&buf, rep, "dev"); err != nil {
		t.Fatalf("RenderTest error: %v", err)
	}
	doc := parseDoc(t, &buf)
	r := doc.Runs[0].Results[0]
	if len(r.Locations) != 0 {
		t.Errorf("expected no locations when File is empty, got %d", len(r.Locations))
	}
}

// ── RenderCov ─────────────────────────────────────────────────────────────────

func TestRenderCov_Schema(t *testing.T) {
	var buf bytes.Buffer
	rep := &covmodel.CovReport{}
	if err := sarif.RenderCov(&buf, rep, 0, "0.2.0"); err != nil {
		t.Fatalf("RenderCov error: %v", err)
	}
	doc := parseDoc(t, &buf)
	if doc.Version != "2.1.0" {
		t.Errorf("version = %q, want 2.1.0", doc.Version)
	}
}

func TestRenderCov_NoThreshold_EmptyResults(t *testing.T) {
	var buf bytes.Buffer
	rep := &covmodel.CovReport{
		Files: []*covmodel.FileCov{
			{Path: "pkg/a.go", LinesTotal: 100, LinesCovered: 60},
		},
	}
	// threshold=0 means "no threshold enforced" → empty results (clears GHAS alerts)
	if err := sarif.RenderCov(&buf, rep, 0, "dev"); err != nil {
		t.Fatalf("RenderCov error: %v", err)
	}
	doc := parseDoc(t, &buf)
	if len(doc.Runs[0].Results) != 0 {
		t.Errorf("results len = %d, want 0 when no threshold set", len(doc.Runs[0].Results))
	}
}

func TestRenderCov_Threshold_FilesBelow_AreWarnings(t *testing.T) {
	var buf bytes.Buffer
	rep := &covmodel.CovReport{
		Files: []*covmodel.FileCov{
			{Path: "pkg/a.go", LinesTotal: 100, LinesCovered: 90}, // 90% — above threshold
			{Path: "pkg/b.go", LinesTotal: 100, LinesCovered: 60}, // 60% — below threshold
			{Path: "pkg/c.go", LinesTotal: 100, LinesCovered: 80}, // exactly at threshold
		},
	}
	if err := sarif.RenderCov(&buf, rep, 80.0, "dev"); err != nil {
		t.Fatalf("RenderCov error: %v", err)
	}
	doc := parseDoc(t, &buf)
	results := doc.Runs[0].Results
	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1 (only pkg/b.go is below 80%%)", len(results))
	}
	r := results[0]
	if r.Level != "warning" {
		t.Errorf("level = %q, want warning", r.Level)
	}
	if r.RuleID != "coverage-below-threshold" {
		t.Errorf("ruleId = %q, want coverage-below-threshold", r.RuleID)
	}
	if len(r.Locations) == 0 {
		t.Fatal("expected location for coverage result")
	}
	if r.Locations[0].PhysicalLocation.ArtifactLocation.URI != "pkg/b.go" {
		t.Errorf("uri = %q, want pkg/b.go", r.Locations[0].PhysicalLocation.ArtifactLocation.URI)
	}
}

func TestRenderCov_AllAboveThreshold_EmptyResults(t *testing.T) {
	var buf bytes.Buffer
	rep := &covmodel.CovReport{
		Files: []*covmodel.FileCov{
			{Path: "pkg/a.go", LinesTotal: 100, LinesCovered: 95},
		},
	}
	if err := sarif.RenderCov(&buf, rep, 80.0, "dev"); err != nil {
		t.Fatalf("RenderCov error: %v", err)
	}
	doc := parseDoc(t, &buf)
	if len(doc.Runs[0].Results) != 0 {
		t.Errorf("results len = %d, want 0 when all files are above threshold", len(doc.Runs[0].Results))
	}
}
