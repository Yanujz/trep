package model_test

import (
	"testing"
	"time"

	"github.com/Yanujz/trep/pkg/model"
)

func TestReportStats_Empty(t *testing.T) {
	rep := &model.Report{}
	total, passed, failed, skipped := rep.Stats()
	if total != 0 || passed != 0 || failed != 0 || skipped != 0 {
		t.Errorf("empty report: got %d/%d/%d/%d, want 0/0/0/0", total, passed, failed, skipped)
	}
}

func TestReportStats(t *testing.T) {
	rep := &model.Report{
		Suites: []model.Suite{
			{Name: "a", Cases: []model.TestCase{
				{Status: model.StatusPass},
				{Status: model.StatusFail},
				{Status: model.StatusSkip},
			}},
			{Name: "b", Cases: []model.TestCase{
				{Status: model.StatusPass},
				{Status: model.StatusPass},
			}},
		},
	}
	total, passed, failed, skipped := rep.Stats()
	if total != 5 || passed != 3 || failed != 1 || skipped != 1 {
		t.Errorf("Stats() = %d/%d/%d/%d, want 5/3/1/1", total, passed, failed, skipped)
	}
}

func TestReportMerge_OverlappingSuites(t *testing.T) {
	r1 := &model.Report{
		Sources:  []string{"a.xml"},
		Duration: 2 * time.Second,
		Suites: []model.Suite{
			{Name: "SuiteA", Cases: []model.TestCase{
				{Name: "Test1", Status: model.StatusPass},
			}},
		},
	}
	r2 := &model.Report{
		Sources:  []string{"b.xml"},
		Duration: 5 * time.Second,
		Suites: []model.Suite{
			{Name: "SuiteA", Cases: []model.TestCase{
				{Name: "Test2", Status: model.StatusFail},
			}},
		},
	}
	r1.Merge(r2)

	if len(r1.Suites) != 1 {
		t.Fatalf("expected 1 suite after merge of overlapping names, got %d", len(r1.Suites))
	}
	if len(r1.Suites[0].Cases) != 2 {
		t.Errorf("expected 2 cases in merged suite, got %d", len(r1.Suites[0].Cases))
	}
	if r1.Duration != 5*time.Second {
		t.Errorf("Duration should be max(2s,5s)=5s, got %v", r1.Duration)
	}
	if len(r1.Sources) != 2 {
		t.Errorf("expected 2 sources after merge, got %d", len(r1.Sources))
	}
}

func TestReportMerge_DisjointSuites(t *testing.T) {
	r1 := &model.Report{
		Suites: []model.Suite{{Name: "A", Cases: []model.TestCase{{Name: "T1", Status: model.StatusPass}}}},
	}
	r2 := &model.Report{
		Suites: []model.Suite{{Name: "B", Cases: []model.TestCase{{Name: "T2", Status: model.StatusPass}}}},
	}
	r1.Merge(r2)

	if len(r1.Suites) != 2 {
		t.Fatalf("expected 2 suites after merge of disjoint suites, got %d", len(r1.Suites))
	}
}

func TestReportMerge_DurationMax(t *testing.T) {
	r1 := &model.Report{Duration: 10 * time.Second}
	r2 := &model.Report{Duration: 3 * time.Second}
	r1.Merge(r2)
	if r1.Duration != 10*time.Second {
		t.Errorf("Merge should keep max duration, got %v", r1.Duration)
	}
}
