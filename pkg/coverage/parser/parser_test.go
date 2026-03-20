package parser_test

import (
	"os"
	"path/filepath"
	"testing"

	covparser "github.com/trep-dev/trep/pkg/coverage/parser"

	// Register all coverage parsers.
	_ "github.com/trep-dev/trep/pkg/coverage/parser/clover"
	_ "github.com/trep-dev/trep/pkg/coverage/parser/cobertura"
	_ "github.com/trep-dev/trep/pkg/coverage/parser/gocover"
	_ "github.com/trep-dev/trep/pkg/coverage/parser/lcov"
)

func TestForName_KnownFormats(t *testing.T) {
	cases := []struct {
		input    string
		wantName string
	}{
		{"lcov", "lcov"},
		{"info", "lcov"},
		{"gocover", "gocover"},
		{"go", "gocover"},
		{"go-cover", "gocover"},
		{"coverprofile", "gocover"},
		{"cobertura", "cobertura"},
		{"jacoco", "cobertura"},
		{"xml", "cobertura"},
		{"clover", "clover"},
	}
	for _, tc := range cases {
		p, err := covparser.ForName(tc.input)
		if err != nil {
			t.Errorf("ForName(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if p.Name() != tc.wantName {
			t.Errorf("ForName(%q).Name() = %q, want %q", tc.input, p.Name(), tc.wantName)
		}
	}
}

func TestForName_Unknown(t *testing.T) {
	_, err := covparser.ForName("doesnotexist")
	if err == nil {
		t.Error("expected error for unknown format")
	}
}

func TestParseFile_GocoverByExtension(t *testing.T) {
	content := "mode: set\npkg/file.go:1.5,3.3 2 1\n"
	f := writeTempFile(t, "coverage.out", content)
	rep, err := covparser.ParseFile(f, nil, "")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(rep.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(rep.Files))
	}
}

func TestParseFile_LCOVByExtension(t *testing.T) {
	content := "SF:src/a.go\nDA:1,5\nend_of_record\n"
	f := writeTempFile(t, "cov.info", content)
	rep, err := covparser.ParseFile(f, nil, "")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(rep.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(rep.Files))
	}
}

func TestParseFile_StripPrefix(t *testing.T) {
	content := "SF:/home/runner/work/repo/src/file.go\nDA:1,1\nend_of_record\n"
	f := writeTempFile(t, "cov.info", content)
	rep, err := covparser.ParseFile(f, nil, "/home/runner/work/repo/")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if rep.Files[0].Path != "src/file.go" {
		t.Errorf("Path after strip = %q, want 'src/file.go'", rep.Files[0].Path)
	}
}

func TestParseFile_FileNotFound(t *testing.T) {
	_, err := covparser.ParseFile("/nonexistent/cov.out", nil, "")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestParseFile_ExplicitParser(t *testing.T) {
	content := "mode: set\npkg/file.go:1.5,3.3 2 1\n"
	f := writeTempFile(t, "coverage.dat", content) // non-standard extension
	p, _ := covparser.ForName("gocover")
	rep, err := covparser.ParseFile(f, p, "")
	if err != nil {
		t.Fatalf("ParseFile with explicit parser: %v", err)
	}
	if len(rep.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(rep.Files))
	}
}

func TestParseFile_ContentDetectionUnknownExtension(t *testing.T) {
	content := "mode: set\npkg/file.go:1.5,3.3 2 1\n"
	f := writeTempFile(t, "coverage.whatever", content)
	rep, err := covparser.ParseFile(f, nil, "")
	if err != nil {
		t.Fatalf("content detection should succeed: %v", err)
	}
	if len(rep.Files) != 1 {
		t.Errorf("expected 1 file via content detection, got %d", len(rep.Files))
	}
}

func TestParseFile_StripPrefix_NoTrailingSlash(t *testing.T) {
	content := "SF:/home/runner/work/repo/src/file.go\nDA:1,1\nend_of_record\n"
	f := writeTempFile(t, "cov.info", content)
	// Strip prefix without trailing slash — ParseFile should normalise.
	rep, err := covparser.ParseFile(f, nil, "/home/runner/work/repo")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if rep.Files[0].Path != "src/file.go" {
		t.Errorf("Path = %q, want 'src/file.go'", rep.Files[0].Path)
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
