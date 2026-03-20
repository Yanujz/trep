// Package delta handles run-over-run comparison for test results and coverage.
// A Snapshot captures the key metrics from one run; two snapshots produce a Delta
// which the HTML renderers use to display change badges.
package delta

import (
	"encoding/json"
	"fmt"
	"os"

	covmodel "github.com/Yanujz/trep/pkg/coverage/model"
	testmodel "github.com/Yanujz/trep/pkg/model"
)

const snapshotVersion = 1

// Snapshot is the machine-readable summary of one run, saved to JSON.
// It can represent tests-only, coverage-only, or a combined run.
type Snapshot struct {
	Version   int    `json:"version"`
	Timestamp string `json:"timestamp"`
	Label     string `json:"label,omitempty"`

	Tests    *TestSnap     `json:"tests,omitempty"`
	Coverage *CoverageSnap `json:"coverage,omitempty"`
}

// TestSnap holds test result metrics.
type TestSnap struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

// CoverageSnap holds coverage metrics.
type CoverageSnap struct {
	LinesPct    float64 `json:"lines_pct"`
	LinesCov    int     `json:"lines_covered"`
	LinesTotal  int     `json:"lines_total"`
	BranchPct   float64 `json:"branch_pct"`
	BranchCov   int     `json:"branch_covered"`
	BranchTotal int     `json:"branch_total"`
	FuncPct     float64 `json:"func_pct"`
	FuncCov     int     `json:"func_covered"`
	FuncTotal   int     `json:"func_total"`
	// Files maps each file path to its line coverage percentage.
	// Used to produce per-file deltas when comparing two snapshots.
	Files map[string]float64 `json:"files,omitempty"`
}

// Delta is the computed difference between a baseline and current snapshot.
type Delta struct {
	// Tests
	TotalDelta   int
	PassedDelta  int
	FailedDelta  int
	SkippedDelta int
	HasTests     bool

	// Coverage — overall
	LinesPctDelta  float64
	BranchPctDelta float64
	FuncPctDelta   float64
	HasCoverage    bool

	// Coverage — per-file line coverage deltas (path → Δ%).
	// Only populated for files present in both baseline and current.
	FileDeltas map[string]float64
}

// FromTestReport builds a TestSnap from a parsed test report.
func FromTestReport(rep *testmodel.Report) *TestSnap {
	total, passed, failed, skipped := rep.Stats()
	return &TestSnap{Total: total, Passed: passed, Failed: failed, Skipped: skipped}
}

// FromCovReport builds a CoverageSnap from a parsed coverage report.
func FromCovReport(rep *covmodel.CovReport) *CoverageSnap {
	lt, lc, bt, bc, ft, fc := rep.Stats()
	snap := &CoverageSnap{
		LinesPct:    rep.LinePct(),
		LinesCov:    lc,
		LinesTotal:  lt,
		BranchPct:   rep.BranchPct(),
		BranchCov:   bc,
		BranchTotal: bt,
		FuncPct:     rep.FuncPct(),
		FuncCov:     fc,
		FuncTotal:   ft,
	}
	if len(rep.Files) > 0 {
		snap.Files = make(map[string]float64, len(rep.Files))
		for _, f := range rep.Files {
			snap.Files[f.Path] = f.LinePct()
		}
	}
	return snap
}

// Compute calculates the delta between baseline and current.
// Either field of current/baseline may be nil (treated as "not available").
func Compute(baseline, current *Snapshot) *Delta {
	if baseline == nil || current == nil {
		return nil
	}
	d := &Delta{}

	if baseline.Tests != nil && current.Tests != nil {
		d.HasTests = true
		d.TotalDelta = current.Tests.Total - baseline.Tests.Total
		d.PassedDelta = current.Tests.Passed - baseline.Tests.Passed
		d.FailedDelta = current.Tests.Failed - baseline.Tests.Failed
		d.SkippedDelta = current.Tests.Skipped - baseline.Tests.Skipped
	}

	if baseline.Coverage != nil && current.Coverage != nil {
		d.HasCoverage = true
		d.LinesPctDelta = current.Coverage.LinesPct - baseline.Coverage.LinesPct
		d.BranchPctDelta = current.Coverage.BranchPct - baseline.Coverage.BranchPct
		d.FuncPctDelta = current.Coverage.FuncPct - baseline.Coverage.FuncPct

		if len(baseline.Coverage.Files) > 0 && len(current.Coverage.Files) > 0 {
			d.FileDeltas = make(map[string]float64)
			for path, curPct := range current.Coverage.Files {
				if basePct, ok := baseline.Coverage.Files[path]; ok {
					delta := curPct - basePct
					if delta != 0 {
						d.FileDeltas[path] = delta
					}
				}
			}
			if len(d.FileDeltas) == 0 {
				d.FileDeltas = nil
			}
		}
	}

	return d
}

// Save writes snap to path as JSON.
func Save(path string, snap *Snapshot) error {
	snap.Version = snapshotVersion
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("delta: marshal snapshot: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("delta: write snapshot %s: %w", path, err)
	}
	return nil
}

// Load reads a snapshot from path.
// It returns an error if the file cannot be read, the JSON is malformed, or
// the snapshot version is newer than what this build supports.
func Load(path string) (*Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("delta: read baseline %s: %w", path, err)
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("delta: parse baseline %s: %w", path, err)
	}
	if snap.Version > snapshotVersion {
		return nil, fmt.Errorf("delta: snapshot %s uses version %d, this build supports up to %d",
			path, snap.Version, snapshotVersion)
	}
	return &snap, nil
}

// FormatPctDelta returns a human-readable signed string like "+2.3%" or "-1.1%".
func FormatPctDelta(d float64) string {
	if d > 0 {
		return fmt.Sprintf("+%.1f%%", d)
	}
	return fmt.Sprintf("%.1f%%", d)
}

// FormatIntDelta returns "+N" or "-N" or "0".
func FormatIntDelta(d int) string {
	if d > 0 {
		return fmt.Sprintf("+%d", d)
	}
	return fmt.Sprintf("%d", d)
}
