package html

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/Yanujz/trep/pkg/model"
)

func TestRenderer_OutputsHTMLTemplate(t *testing.T) {
	rep := &model.Report{
		Title: "Test My Awesome Project",
		Suites: []model.Suite{
			{
				Name: "Core Suite",
				Cases: []model.TestCase{
					{
						Suite:   "Core Suite",
						Name:    "TestPassing",
						Status:  model.StatusPass,
						Message: "",
					},
					{
						Suite:   "Core Suite",
						Name:    "TestFailing",
						Status:  model.StatusFail,
						Message: "Error occurred",
					},
				},
			},
		},
		Timestamp: time.Now(),
		Duration:  time.Second * 5,
	}

	r := Renderer{}
	opts := Options{
		CovReportURL: "coverage.html",
	}

	var buf bytes.Buffer
	err := r.Render(&buf, rep, opts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	out := buf.String()

	// 1. Verify CSS string contents (e.g. standard dark-mode config)
	if !strings.Contains(out, ":root") {
		t.Errorf("expected HTML template root CSS block, missing in output")
	}

	// 2. Verify variables replaced
	if !strings.Contains(out, "Test My Awesome Project") {
		t.Errorf("expected title Test My Awesome Project, but not found")
	}
	if !strings.Contains(out, ">FAILED<") && !strings.Contains(out, "FAILED") {
		t.Errorf("expected FAILED status due to 1 fail")
	}
	if !strings.Contains(out, "coverage.html") {
		t.Errorf("expected CovReportURL coverage.html, but not found")
	}

	// 3. Verify JSON payload injection
	if !strings.Contains(out, `"TestPassing"`) || !strings.Contains(out, `"TestFailing"`) {
		t.Errorf("expected JSON injected rows, but missing 'TestPassing' or 'TestFailing'")
	}
}

func TestRenderer_Name(t *testing.T) {
	if (Renderer{}).Name() != "html" {
		t.Errorf("Expected html for Renderer.Name()")
	}
}
