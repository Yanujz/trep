// Package parser defines the CovParser interface and global registry for
// coverage format auto-detection and dispatch.
package parser

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	covmodel "github.com/trep-dev/trep/pkg/coverage/model"
)

// CovParser converts a specific coverage format into a [covmodel.CovReport].
type CovParser interface {
	Name() string
	Extensions() []string
	Detect(header []byte) bool
	Parse(r io.Reader, source string) (*covmodel.CovReport, error)
}

var (
	registry []CovParser
	aliases  = map[string]string{
		"go":         "gocover",
		"go-cover":   "gocover",
		"coverprofile": "gocover",
		"lcov":       "lcov",
		"info":       "lcov",
		"cobertura":  "cobertura",
		"jacoco":     "cobertura",
		"xml":        "cobertura",
		"clover":     "clover",
	}
)

// Register adds p to the global registry. Earlier registrations win on Detect.
func Register(p CovParser) { registry = append(registry, p) }

// ForName returns the parser for the given name or alias.
func ForName(name string) (CovParser, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if c, ok := aliases[name]; ok {
		name = c
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
	return nil, fmt.Errorf("unknown coverage format %q; available: %s", name, strings.Join(names, ", "))
}

// ParseFile opens path and parses it; if p is nil format is auto-detected.
// Use "-" for stdin.
func ParseFile(path string, p CovParser, stripPrefix string) (*covmodel.CovReport, error) {
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

	if p != nil {
		rep, err := p.Parse(rc, source)
		if err != nil {
			return nil, err
		}
		applyStripPrefix(rep, stripPrefix)
		return rep, nil
	}

	// Try extension-based lookup first.
	if path != "-" {
		if ep := byExtension(path); ep != nil {
			rep, err := ep.Parse(rc, source)
			if err != nil {
				return nil, err
			}
			applyStripPrefix(rep, stripPrefix)
			return rep, nil
		}
	}

	// Read header, detect, replay.
	header := make([]byte, 512)
	n, _ := rc.Read(header)
	header = header[:n]
	combined := io.MultiReader(bytes.NewReader(header), rc)

	for _, pp := range registry {
		if pp.Detect(header) {
			rep, err := pp.Parse(combined, source)
			if err != nil {
				return nil, err
			}
			applyStripPrefix(rep, stripPrefix)
			return rep, nil
		}
	}
	return nil, fmt.Errorf("cannot detect coverage format for %q; specify with --format-cov", source)
}

func byExtension(path string) CovParser {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	for _, p := range registry {
		for _, e := range p.Extensions() {
			if e == ext {
				return p
			}
		}
	}
	return nil
}

// applyStripPrefix removes a common path prefix from all file paths.
func applyStripPrefix(rep *covmodel.CovReport, prefix string) {
	if prefix == "" {
		return
	}
	prefix = filepath.ToSlash(prefix)
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	for _, f := range rep.Files {
		p := filepath.ToSlash(f.Path)
		if strings.HasPrefix(p, prefix) {
			f.Path = p[len(prefix):]
		}
	}
}
