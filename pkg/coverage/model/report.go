// Package model defines the format-agnostic coverage types that all coverage
// parsers produce and all coverage renderers consume.
package model

import "time"

// LineCov records execution data for a single source line.
type LineCov struct {
	Number int
	Hits   int // 0 = not covered, >0 = covered
}

// BranchCov records one branch outcome (true/false side of a conditional).
type BranchCov struct {
	Line  int
	Block int
	Index int
	Taken int // -1 = not reachable, 0 = not taken, >0 = taken count
}

// FuncCov records call-count data for one function.
type FuncCov struct {
	Name  string
	Line  int // declaration line
	Calls int
}

// FileCov holds all coverage data for one source file.
type FileCov struct {
	Path string // normalised relative path (e.g. "src/core/parser.go")

	Lines    []LineCov
	Branches []BranchCov
	Funcs    []FuncCov

	// Precomputed totals (populated by parsers or via Compute).
	LinesTotal    int
	LinesCovered  int
	BranchTotal   int
	BranchCovered int
	FuncTotal     int
	FuncCovered   int
}

// Compute fills the precomputed totals from the raw slice data.
// Call this after a parser has finished populating Lines/Branches/Funcs.
func (f *FileCov) Compute() {
	f.LinesTotal, f.LinesCovered = 0, 0
	for _, l := range f.Lines {
		f.LinesTotal++
		if l.Hits > 0 {
			f.LinesCovered++
		}
	}
	f.BranchTotal, f.BranchCovered = 0, 0
	for _, b := range f.Branches {
		if b.Taken < 0 {
			continue // unreachable
		}
		f.BranchTotal++
		if b.Taken > 0 {
			f.BranchCovered++
		}
	}
	f.FuncTotal, f.FuncCovered = 0, 0
	for _, fn := range f.Funcs {
		f.FuncTotal++
		if fn.Calls > 0 {
			f.FuncCovered++
		}
	}
}

// LinePct returns line coverage as a percentage [0, 100].
func (f *FileCov) LinePct() float64 { return safePct(f.LinesCovered, f.LinesTotal) }

// BranchPct returns branch coverage as a percentage [0, 100].
func (f *FileCov) BranchPct() float64 { return safePct(f.BranchCovered, f.BranchTotal) }

// FuncPct returns function coverage as a percentage [0, 100].
func (f *FileCov) FuncPct() float64 { return safePct(f.FuncCovered, f.FuncTotal) }

// CovReport is the top-level, format-independent coverage result.
type CovReport struct {
	Sources   []string
	Timestamp time.Time
	Files     []*FileCov
}

// Stats returns aggregate line/branch/function totals across all files.
func (r *CovReport) Stats() (lt, lc, bt, bc, ft, fc int) {
	for _, f := range r.Files {
		lt += f.LinesTotal
		lc += f.LinesCovered
		bt += f.BranchTotal
		bc += f.BranchCovered
		ft += f.FuncTotal
		fc += f.FuncCovered
	}
	return
}

// LinePct returns overall line coverage %.
func (r *CovReport) LinePct() float64 {
	lt, lc, _, _, _, _ := r.Stats()
	return safePct(lc, lt)
}

// BranchPct returns overall branch coverage %.
func (r *CovReport) BranchPct() float64 {
	_, _, bt, bc, _, _ := r.Stats()
	return safePct(bc, bt)
}

// FuncPct returns overall function coverage %.
func (r *CovReport) FuncPct() float64 {
	_, _, _, _, ft, fc := r.Stats()
	return safePct(fc, ft)
}

// Merge incorporates all files from other into r.
// Files with duplicate paths are combined by appending their raw slice data and
// recomputing totals. Use this to merge per-package coverage profiles into one
// aggregate report.
func (r *CovReport) Merge(other *CovReport) {
	r.Sources = append(r.Sources, other.Sources...)

	byPath := make(map[string]int, len(r.Files))
	for i, f := range r.Files {
		byPath[f.Path] = i
	}
	for _, of := range other.Files {
		if idx, ok := byPath[of.Path]; ok {
			ex := r.Files[idx]
			ex.Lines = append(ex.Lines, of.Lines...)
			ex.Branches = append(ex.Branches, of.Branches...)
			ex.Funcs = append(ex.Funcs, of.Funcs...)
			ex.Compute()
		} else {
			byPath[of.Path] = len(r.Files)
			r.Files = append(r.Files, of)
		}
	}
}

func safePct(covered, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(covered) / float64(total) * 100
}
