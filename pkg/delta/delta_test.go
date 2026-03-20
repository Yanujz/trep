package delta_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	covmodel "github.com/trep-dev/trep/pkg/coverage/model"
	"github.com/trep-dev/trep/pkg/delta"
	"github.com/trep-dev/trep/pkg/model"
)

func TestFromTestReport(t *testing.T) {
	rep := &model.Report{
		Suites: []model.Suite{{Cases: []model.TestCase{
			{Status: model.StatusPass},
			{Status: model.StatusPass},
			{Status: model.StatusFail},
			{Status: model.StatusSkip},
		}}},
	}
	snap := delta.FromTestReport(rep)
	if snap.Total != 4 || snap.Passed != 2 || snap.Failed != 1 || snap.Skipped != 1 {
		t.Errorf("TestSnap = {%d %d %d %d}, want {4 2 1 1}",
			snap.Total, snap.Passed, snap.Failed, snap.Skipped)
	}
}

func TestFromCovReport(t *testing.T) {
	rep := &covmodel.CovReport{
		Files: []*covmodel.FileCov{
			{LinesTotal: 100, LinesCovered: 80},
		},
	}
	snap := delta.FromCovReport(rep)
	if snap.LinesTotal != 100 || snap.LinesCov != 80 {
		t.Errorf("CoverageSnap lines: total=%d cov=%d", snap.LinesTotal, snap.LinesCov)
	}
	if snap.LinesPct != 80.0 {
		t.Errorf("LinesPct = %.1f, want 80.0", snap.LinesPct)
	}
}

func TestCompute_NilBaseline(t *testing.T) {
	cur := &delta.Snapshot{Tests: &delta.TestSnap{Total: 5}}
	if d := delta.Compute(nil, cur); d != nil {
		t.Error("Compute(nil, cur) should return nil")
	}
}

func TestCompute_NilCurrent(t *testing.T) {
	base := &delta.Snapshot{Tests: &delta.TestSnap{Total: 5}}
	if d := delta.Compute(base, nil); d != nil {
		t.Error("Compute(base, nil) should return nil")
	}
}

func TestCompute_TestDeltas(t *testing.T) {
	base := &delta.Snapshot{
		Tests: &delta.TestSnap{Total: 10, Passed: 9, Failed: 1, Skipped: 0},
	}
	cur := &delta.Snapshot{
		Tests: &delta.TestSnap{Total: 12, Passed: 11, Failed: 0, Skipped: 1},
	}
	d := delta.Compute(base, cur)
	if !d.HasTests {
		t.Error("HasTests should be true when both have test data")
	}
	if d.TotalDelta != 2 {
		t.Errorf("TotalDelta = %d, want 2", d.TotalDelta)
	}
	if d.PassedDelta != 2 {
		t.Errorf("PassedDelta = %d, want 2", d.PassedDelta)
	}
	if d.FailedDelta != -1 {
		t.Errorf("FailedDelta = %d, want -1", d.FailedDelta)
	}
	if d.SkippedDelta != 1 {
		t.Errorf("SkippedDelta = %d, want 1", d.SkippedDelta)
	}
}

func TestCompute_CoverageDeltas(t *testing.T) {
	base := &delta.Snapshot{
		Coverage: &delta.CoverageSnap{LinesPct: 75.0, BranchPct: 60.0, FuncPct: 80.0},
	}
	cur := &delta.Snapshot{
		Coverage: &delta.CoverageSnap{LinesPct: 80.0, BranchPct: 55.0, FuncPct: 85.0},
	}
	d := delta.Compute(base, cur)
	if !d.HasCoverage {
		t.Error("HasCoverage should be true when both have coverage data")
	}
	if d.LinesPctDelta != 5.0 {
		t.Errorf("LinesPctDelta = %.1f, want 5.0", d.LinesPctDelta)
	}
	if d.BranchPctDelta != -5.0 {
		t.Errorf("BranchPctDelta = %.1f, want -5.0", d.BranchPctDelta)
	}
}

func TestCompute_MissingTestsInBaseline(t *testing.T) {
	base := &delta.Snapshot{Coverage: &delta.CoverageSnap{LinesPct: 70.0}}
	cur := &delta.Snapshot{
		Tests:    &delta.TestSnap{Total: 5},
		Coverage: &delta.CoverageSnap{LinesPct: 80.0},
	}
	d := delta.Compute(base, cur)
	if d.HasTests {
		t.Error("HasTests should be false when baseline has no test data")
	}
	if !d.HasCoverage {
		t.Error("HasCoverage should be true when both have coverage data")
	}
}

func TestSaveLoad_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snap.json")

	original := &delta.Snapshot{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Label:     "main",
		Tests:     &delta.TestSnap{Total: 10, Passed: 9, Failed: 1, Skipped: 0},
		Coverage:  &delta.CoverageSnap{LinesPct: 85.5, LinesTotal: 100, LinesCov: 85},
	}

	if err := delta.Save(path, original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := delta.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Label != original.Label {
		t.Errorf("Label = %q, want %q", loaded.Label, original.Label)
	}
	if loaded.Tests.Total != original.Tests.Total {
		t.Errorf("Tests.Total = %d, want %d", loaded.Tests.Total, original.Tests.Total)
	}
	if loaded.Coverage.LinesPct != original.Coverage.LinesPct {
		t.Errorf("Coverage.LinesPct = %.1f, want %.1f", loaded.Coverage.LinesPct, original.Coverage.LinesPct)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := delta.Load("/nonexistent/snap.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("not json {{{"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := delta.Load(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoad_FutureVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "future.json")
	data, _ := json.Marshal(map[string]any{"version": 999, "timestamp": "2024-01-01T00:00:00Z"})
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	_, err := delta.Load(path)
	if err == nil {
		t.Error("expected error when snapshot version is newer than supported")
	}
}

func TestFormatPctDelta(t *testing.T) {
	cases := []struct{ in float64; want string }{
		{2.3, "+2.3%"},
		{-1.1, "-1.1%"},
		{0.0, "0.0%"},
	}
	for _, tc := range cases {
		if got := delta.FormatPctDelta(tc.in); got != tc.want {
			t.Errorf("FormatPctDelta(%.1f) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatIntDelta(t *testing.T) {
	cases := []struct{ in int; want string }{
		{3, "+3"},
		{-2, "-2"},
		{0, "0"},
	}
	for _, tc := range cases {
		if got := delta.FormatIntDelta(tc.in); got != tc.want {
			t.Errorf("FormatIntDelta(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
