package parser_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/trep-dev/trep/pkg/parser"

	// Register all parsers.
	_ "github.com/trep-dev/trep/pkg/parser/gotest"
	_ "github.com/trep-dev/trep/pkg/parser/junit"
	_ "github.com/trep-dev/trep/pkg/parser/tap"
)

func TestForName_KnownFormats(t *testing.T) {
	cases := []struct {
		name     string
		wantName string
	}{
		{"junit", "junit"},
		{"gtest", "junit"},
		{"ctest", "junit"},
		{"maven", "junit"},
		{"xml", "junit"},
		{"gotest", "gotest"},
		{"go", "gotest"},
		{"json", "gotest"},
		{"tap", "tap"},
	}
	for _, tc := range cases {
		p, err := parser.ForName(tc.name)
		if err != nil {
			t.Errorf("ForName(%q): unexpected error: %v", tc.name, err)
			continue
		}
		if p.Name() != tc.wantName {
			t.Errorf("ForName(%q).Name() = %q, want %q", tc.name, p.Name(), tc.wantName)
		}
	}
}

func TestForName_Unknown(t *testing.T) {
	_, err := parser.ForName("unknownformat")
	if err == nil {
		t.Error("expected error for unknown format, got nil")
	}
}

func TestForName_CaseInsensitive(t *testing.T) {
	p, err := parser.ForName("JUNIT")
	if err != nil {
		t.Fatalf("ForName(JUNIT): %v", err)
	}
	if p.Name() != "junit" {
		t.Errorf("Name() = %q, want junit", p.Name())
	}
}

func TestParseFile_XMLByExtension(t *testing.T) {
	content := `<testsuites><testsuite><testcase name="T1" classname="S1" time="0.1"/></testsuite></testsuites>`
	f := writeTempFile(t, "results.xml", content)
	rep, err := parser.ParseFile(f, nil)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	total, _, _, _ := rep.Stats()
	if total != 1 {
		t.Errorf("expected 1 test, got %d", total)
	}
}

func TestParseFile_ContentDetectionUnknownExtension(t *testing.T) {
	content := `<testsuites><testsuite><testcase name="T1" classname="S1" time="0.1"/></testsuite></testsuites>`
	f := writeTempFile(t, "results.dat", content)
	rep, err := parser.ParseFile(f, nil)
	if err != nil {
		t.Fatalf("ParseFile with unknown extension should fall back to content detection: %v", err)
	}
	total, _, _, _ := rep.Stats()
	if total != 1 {
		t.Errorf("expected 1 test via content detection, got %d", total)
	}
}

func TestParseFile_FileNotFound(t *testing.T) {
	_, err := parser.ParseFile("/nonexistent/path/results.xml", nil)
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestParseFile_ExplicitParser(t *testing.T) {
	content := `<testsuites><testsuite><testcase name="T1" classname="S1"/></testsuite></testsuites>`
	f := writeTempFile(t, "results.dat", content) // non-standard extension
	p, _ := parser.ForName("junit")
	rep, err := parser.ParseFile(f, p)
	if err != nil {
		t.Fatalf("ParseFile with explicit parser: %v", err)
	}
	total, _, _, _ := rep.Stats()
	if total != 1 {
		t.Errorf("expected 1 test, got %d", total)
	}
}

func TestParseFile_UndetectableContent(t *testing.T) {
	f := writeTempFile(t, "results.dat", "this is not any recognized format")
	_, err := parser.ParseFile(f, nil)
	if err == nil {
		t.Error("expected error for unrecognizable content, got nil")
	}
}

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}
