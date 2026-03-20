// Package cobertura parses Cobertura XML coverage files produced by JaCoCo,
// coverage.py (pytest-cov), .NET coverlet, and compatible tools.
package cobertura

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	covmodel "github.com/trep-dev/trep/pkg/coverage/model"
	covparser "github.com/trep-dev/trep/pkg/coverage/parser"
)

func init() { covparser.Register(Parser{}) }

// Parser handles Cobertura XML files.
type Parser struct{}

func (Parser) Name() string         { return "cobertura" }
func (Parser) Extensions() []string { return []string{"xml"} }
func (Parser) Detect(header []byte) bool {
	s := strings.ToLower(string(header))
	return strings.Contains(s, "<coverage") && strings.Contains(s, "line-rate")
}

// ── XML wire types ────────────────────────────────────────────────────────────

type xmlCoverage struct {
	XMLName  xml.Name     `xml:"coverage"`
	Packages []xmlPackage `xml:"packages>package"`
}

type xmlPackage struct {
	Name    string     `xml:"name,attr"`
	Classes []xmlClass `xml:"classes>class"`
}

type xmlClass struct {
	Name     string    `xml:"name,attr"`
	Filename string    `xml:"filename,attr"`
	Methods  []xmlMethod `xml:"methods>method"`
	Lines    []xmlLine `xml:"lines>line"`
}

type xmlMethod struct {
	Name string    `xml:"name,attr"`
	Line string    `xml:"line,attr"`
	Hits string    `xml:"hits,attr"`
	Lines []xmlLine `xml:"lines>line"`
}

type xmlLine struct {
	Number string `xml:"number,attr"`
	Hits   string `xml:"hits,attr"`
	Branch string `xml:"branch,attr"`       // "true"/"false"
	CondCoverage string `xml:"condition-coverage,attr"` // "50% (1/2)"
}

// ── Parser ────────────────────────────────────────────────────────────────────

func (Parser) Parse(r io.Reader, source string) (*covmodel.CovReport, error) {
	var cov xmlCoverage
	dec := xml.NewDecoder(r)
	if err := dec.Decode(&cov); err != nil {
		return nil, fmt.Errorf("cobertura: XML decode: %w", err)
	}

	rep := &covmodel.CovReport{
		Sources:   []string{source},
		Timestamp: time.Now().UTC(),
	}

	for _, pkg := range cov.Packages {
		for _, cls := range pkg.Classes {
			fc := &covmodel.FileCov{Path: cls.Filename}
			if fc.Path == "" {
				fc.Path = cls.Name
			}

			// Methods → FuncCov
			for _, m := range cls.Methods {
				lineNo, _ := strconv.Atoi(m.Line)
				hits, _   := strconv.Atoi(m.Hits)
				fc.Funcs = append(fc.Funcs, covmodel.FuncCov{
					Name:  m.Name,
					Line:  lineNo,
					Calls: hits,
				})
				// Also harvest lines nested inside the method element.
				for _, l := range m.Lines {
					if lc := parseXMLLine(l); lc != nil {
						fc.Lines = append(fc.Lines, *lc)
					}
				}
			}

			// Class-level lines (may duplicate method lines — deduplicate by number).
			seen := make(map[int]bool, len(fc.Lines))
			for _, l := range fc.Lines {
				seen[l.Number] = true
			}
			for _, l := range cls.Lines {
				if lc := parseXMLLine(l); lc != nil && !seen[lc.Number] {
					seen[lc.Number] = true
					fc.Lines = append(fc.Lines, *lc)
				}
			}

			fc.Compute()
			rep.Files = append(rep.Files, fc)
		}
	}

	return rep, nil
}

func parseXMLLine(l xmlLine) *covmodel.LineCov {
	lineNo, err := strconv.Atoi(l.Number)
	if err != nil {
		return nil
	}
	hits, _ := strconv.Atoi(l.Hits)
	return &covmodel.LineCov{Number: lineNo, Hits: hits}
}
