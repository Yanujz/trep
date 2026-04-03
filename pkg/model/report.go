// Package model defines the format-agnostic test result types that all parsers
// produce and all renderers consume.
package model

import "time"

// Status is the outcome of a single test case.
type Status string

// StatusPass, StatusFail, StatusSkip, and StatusFlaky are the possible test outcome values.
const (
	StatusPass  Status = "pass"
	StatusFail  Status = "fail"
	StatusSkip  Status = "skip"
	// StatusFlaky indicates a test that failed but ultimately passed after retries.
	StatusFlaky Status = "flaky"
)

// TestCase holds the normalised result of one test.
type TestCase struct {
	Suite    string // Logical grouping: classname, package path, file name, …
	Name     string // Test identifier
	Status   Status
	Duration time.Duration
	Message  string // Failure / error / skip reason
	Stdout   string // Captured output; non-empty only for failures
	File     string // Source file path (available from GTest XML)
	Line     int    // Source line number (available from GTest XML)
}

// Suite groups related test cases under one name.
type Suite struct {
	Name  string
	Cases []TestCase
}

// Report is the top-level, format-independent result tree.
type Report struct {
	Title     string
	Sources   []string // original input paths / labels
	Timestamp time.Time
	Duration  time.Duration
	Suites    []Suite
}

// Stats returns aggregate pass/fail/skip counts and total.
func (r *Report) Stats() (total, passed, failed, skipped int) {
	for _, s := range r.Suites {
		for _, c := range s.Cases {
			total++
			switch c.Status {
			case StatusPass:
				passed++
			case StatusFail:
				failed++
			case StatusSkip:
				skipped++
			case StatusFlaky:
				passed++ // flaky = eventually passed
			}
		}
	}
	return
}

// Merge folds other into r, combining suites that share a name.
// Duration is set to the maximum of the two reports.
func (r *Report) Merge(other *Report) {
	idx := make(map[string]int, len(r.Suites))
	for i, s := range r.Suites {
		idx[s.Name] = i
	}
	for _, s := range other.Suites {
		if i, ok := idx[s.Name]; ok {
			r.Suites[i].Cases = append(r.Suites[i].Cases, s.Cases...)
		} else {
			idx[s.Name] = len(r.Suites)
			r.Suites = append(r.Suites, s)
		}
	}
	if other.Duration > r.Duration {
		r.Duration = other.Duration
	}
	r.Sources = append(r.Sources, other.Sources...)
}
