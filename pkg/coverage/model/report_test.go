package model_test

import (
	"testing"

	covmodel "github.com/trep-dev/trep/pkg/coverage/model"
)

func TestFileCovCompute_Basic(t *testing.T) {
	f := &covmodel.FileCov{
		Lines: []covmodel.LineCov{
			{Number: 1, Hits: 5},
			{Number: 2, Hits: 0},
			{Number: 3, Hits: 1},
		},
		Branches: []covmodel.BranchCov{
			{Line: 1, Taken: 3},
			{Line: 1, Taken: 0},
			{Line: 2, Taken: -1}, // unreachable
		},
		Funcs: []covmodel.FuncCov{
			{Name: "FuncA", Calls: 5},
			{Name: "FuncB", Calls: 0},
		},
	}
	f.Compute()

	if f.LinesTotal != 3 || f.LinesCovered != 2 {
		t.Errorf("lines: total=%d covered=%d, want 3/2", f.LinesTotal, f.LinesCovered)
	}
	if f.BranchTotal != 2 || f.BranchCovered != 1 {
		t.Errorf("branches: total=%d covered=%d, want 2/1 (unreachable excluded)", f.BranchTotal, f.BranchCovered)
	}
	if f.FuncTotal != 2 || f.FuncCovered != 1 {
		t.Errorf("funcs: total=%d covered=%d, want 2/1", f.FuncTotal, f.FuncCovered)
	}
}

func TestFileCovPctMethods(t *testing.T) {
	f := &covmodel.FileCov{
		LinesTotal: 10, LinesCovered: 7,
		BranchTotal: 4, BranchCovered: 3,
		FuncTotal: 2, FuncCovered: 2,
	}
	if got := f.LinePct(); got != 70.0 {
		t.Errorf("LinePct = %.1f, want 70.0", got)
	}
	if got := f.BranchPct(); got != 75.0 {
		t.Errorf("BranchPct = %.1f, want 75.0", got)
	}
	if got := f.FuncPct(); got != 100.0 {
		t.Errorf("FuncPct = %.1f, want 100.0", got)
	}
}

func TestFileCovPct_ZeroTotal(t *testing.T) {
	f := &covmodel.FileCov{}
	if f.LinePct() != 0 {
		t.Error("LinePct with zero total should return 0")
	}
	if f.BranchPct() != 0 {
		t.Error("BranchPct with zero total should return 0")
	}
	if f.FuncPct() != 0 {
		t.Error("FuncPct with zero total should return 0")
	}
}

func TestCovReportStats(t *testing.T) {
	rep := &covmodel.CovReport{
		Files: []*covmodel.FileCov{
			{LinesTotal: 10, LinesCovered: 8, BranchTotal: 4, BranchCovered: 2, FuncTotal: 3, FuncCovered: 3},
			{LinesTotal: 5, LinesCovered: 3, BranchTotal: 2, BranchCovered: 1, FuncTotal: 1, FuncCovered: 0},
		},
	}
	lt, lc, bt, bc, ft, fc := rep.Stats()
	if lt != 15 || lc != 11 || bt != 6 || bc != 3 || ft != 4 || fc != 3 {
		t.Errorf("Stats = %d/%d/%d/%d/%d/%d, want 15/11/6/3/4/3", lt, lc, bt, bc, ft, fc)
	}
}

func TestCovReportLinePct(t *testing.T) {
	rep := &covmodel.CovReport{
		Files: []*covmodel.FileCov{
			{LinesTotal: 100, LinesCovered: 80},
		},
	}
	if got := rep.LinePct(); got != 80.0 {
		t.Errorf("LinePct = %.1f, want 80.0", got)
	}
}

func TestCovReportLinePct_Empty(t *testing.T) {
	rep := &covmodel.CovReport{}
	if rep.LinePct() != 0 {
		t.Error("LinePct of empty report should be 0")
	}
}

func TestFileCovCompute_UnreachableBranchNotCounted(t *testing.T) {
	f := &covmodel.FileCov{
		Branches: []covmodel.BranchCov{
			{Line: 1, Taken: -1},
			{Line: 1, Taken: -1},
		},
	}
	f.Compute()
	if f.BranchTotal != 0 {
		t.Errorf("BranchTotal should be 0 when all branches are unreachable, got %d", f.BranchTotal)
	}
}

func TestCovReportMerge_DisjointFiles(t *testing.T) {
	a := &covmodel.CovReport{
		Sources: []string{"a.out"},
		Files: []*covmodel.FileCov{
			{Path: "src/a.go", LinesTotal: 10, LinesCovered: 8},
		},
	}
	b := &covmodel.CovReport{
		Sources: []string{"b.out"},
		Files: []*covmodel.FileCov{
			{Path: "src/b.go", LinesTotal: 5, LinesCovered: 3},
		},
	}
	a.Merge(b)

	if len(a.Files) != 2 {
		t.Fatalf("want 2 files after merge, got %d", len(a.Files))
	}
	if len(a.Sources) != 2 {
		t.Errorf("want 2 sources after merge, got %d", len(a.Sources))
	}
}

func TestCovReportMerge_DuplicatePath(t *testing.T) {
	a := &covmodel.CovReport{
		Files: []*covmodel.FileCov{
			{Path: "src/foo.go", Lines: []covmodel.LineCov{{Number: 1, Hits: 2}, {Number: 2, Hits: 0}}},
		},
	}
	b := &covmodel.CovReport{
		Files: []*covmodel.FileCov{
			{Path: "src/foo.go", Lines: []covmodel.LineCov{{Number: 2, Hits: 1}, {Number: 3, Hits: 5}}},
		},
	}
	a.Files[0].Compute()
	a.Merge(b)

	if len(a.Files) != 1 {
		t.Fatalf("duplicate path should not add a new file, got %d files", len(a.Files))
	}
	f := a.Files[0]
	if len(f.Lines) != 4 {
		t.Errorf("merged file should have 4 raw lines, got %d", len(f.Lines))
	}
	// After Merge, Compute is called: 3 total lines (1 hit, 3 total from raw)
	// Actually 4 raw lines: {1,2},{2,0},{2,1},{3,5} → 4 lines total, covered=3
	if f.LinesTotal != 4 {
		t.Errorf("LinesTotal = %d, want 4", f.LinesTotal)
	}
	if f.LinesCovered != 3 {
		t.Errorf("LinesCovered = %d, want 3", f.LinesCovered)
	}
}
