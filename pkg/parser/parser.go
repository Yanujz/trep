// Package parser defines the Parser interface and the global registry used for
// format auto-detection and dispatch.
package parser

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Yanujz/trep/pkg/model"
)

// Parser converts a specific test result format into a [model.Report].
type Parser interface {
	// Name is the canonical format identifier, lowercase, no spaces.
	Name() string
	// Extensions lists file extensions (without leading dot) this parser handles.
	Extensions() []string
	// Detect returns true if the given header bytes look like this format.
	// header is at most 512 bytes from the beginning of the file.
	Detect(header []byte) bool
	// Parse reads r to completion and returns a normalised Report.
	Parse(r io.Reader, source string) (*model.Report, error)
}

var (
	registry []Parser

	// aliases maps user-facing alternate names to canonical parser names.
	aliases = map[string]string{
		"gtest":  "junit",
		"ctest":  "junit",
		"maven":  "junit",
		"xml":    "junit",
		"go":     "gotest",
		"json":   "gotest",
		"xunit":  "nunit",
		"nunit3": "nunit",
	}
)

// Register adds p to the global registry.
// Parsers registered earlier take precedence during auto-detection.
func Register(p Parser) {
	registry = append(registry, p)
}

// ForName returns the parser registered under name or one of its known aliases.
func ForName(name string) (Parser, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if canonical, ok := aliases[name]; ok {
		name = canonical
	}
	for _, p := range registry {
		if p.Name() == name {
			return p, nil
		}
	}
	var names []string
	for _, p := range registry {
		names = append(names, p.Name())
	}
	return nil, fmt.Errorf("unknown format %q; available: %s (aliases: gtest/ctest→junit, go/json→gotest)",
		name, strings.Join(names, ", "))
}

// ParseFile opens path and parses it using p.
// If p is nil, the format is auto-detected from the file extension and content.
// Pass "-" as path to read from stdin; content detection is used in that case.
func ParseFile(path string, p Parser) (*model.Report, error) {
	source := path
	if path == "-" {
		source = "<stdin>"
	}

	var rc io.ReadCloser
	if path == "-" {
		rc = io.NopCloser(os.Stdin)
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", path, err)
		}
		rc = f
	}
	defer rc.Close()

	// Caller supplied an explicit parser — use it directly.
	if p != nil {
		return p.Parse(rc, source)
	}

	// Fast path: try extension-based lookup (avoids buffering the file).
	if path != "-" {
		if ep := byExtension(path); ep != nil {
			return ep.Parse(rc, source)
		}
	}

	// Slow path: read a header block, detect by content, then replay via MultiReader.
	header := make([]byte, 512)
	n, _ := rc.Read(header)
	header = header[:n]

	combined := io.MultiReader(bytes.NewReader(header), rc)
	for _, pp := range registry {
		if pp.Detect(header) {
			return pp.Parse(combined, source)
		}
	}

	return nil, fmt.Errorf("cannot detect format for %q; specify with --format", source)
}

// byExtension returns the first parser that claims the file's extension, or nil.
func byExtension(path string) Parser {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	if ext == "" {
		return nil
	}
	for _, p := range registry {
		for _, e := range p.Extensions() {
			if e == ext {
				return p
			}
		}
	}
	return nil
}
