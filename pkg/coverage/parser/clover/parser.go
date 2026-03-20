// Package clover parses Clover XML coverage files.
package clover

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

// Parser handles Clover XML files.
type Parser struct{}

func (Parser) Name() string         { return "clover" }
func (Parser) Extensions() []string { return []string{} } // XML ext claimed by cobertura; detect by content
func (Parser) Detect(header []byte) bool {
	s := strings.ToLower(string(header))
	return strings.Contains(s, "<coverage") && strings.Contains(s, "clover")
}

// ── XML wire types ────────────────────────────────────────────────────────────

type xmlCoverage struct {
	XMLName  xml.Name    `xml:"coverage"`
	Project  xmlProject  `xml:"project"`
}

type xmlProject struct {
	Packages []xmlPackage `xml:"package"`
	Files    []xmlFile    `xml:"file"` // top-level files (no package)
}

type xmlPackage struct {
	Name  string    `xml:"name,attr"`
	Files []xmlFile `xml:"file"`
}

type xmlFile struct {
	Name    string    `xml:"name,attr"`
	Path    string    `xml:"path,attr"`
	Lines   []xmlLine `xml:"line"`
	Metrics *xmlMetrics `xml:"metrics"`
}

type xmlLine struct {
	Num        string `xml:"num,attr"`
	Type       string `xml:"type,attr"` // "stmt", "method", "cond"
	Count      string `xml:"count,attr"`
	TrueCount  string `xml:"truecount,attr"`
	FalseCount string `xml:"falsecount,attr"`
}

type xmlMetrics struct {
	Statements        int `xml:"statements,attr"`
	CoveredStatements int `xml:"coveredstatements,attr"`
	Methods           int `xml:"methods,attr"`
	CoveredMethods    int `xml:"coveredmethods,attr"`
}

// ── Parser ────────────────────────────────────────────────────────────────────

func (Parser) Parse(r io.Reader, source string) (*covmodel.CovReport, error) {
	var cov xmlCoverage
	if err := xml.NewDecoder(r).Decode(&cov); err != nil {
		return nil, fmt.Errorf("clover: XML decode: %w", err)
	}

	rep := &covmodel.CovReport{
		Sources:   []string{source},
		Timestamp: time.Now().UTC(),
	}

	addFile := func(xf xmlFile, pkgName string) {
		path := xf.Path
		if path == "" {
			path = xf.Name
		}
		if pkgName != "" && path == xf.Name {
			path = pkgName + "/" + xf.Name
		}

		fc := &covmodel.FileCov{Path: path}

		for _, l := range xf.Lines {
			lineNo, _ := strconv.Atoi(l.Num)
			count, _  := strconv.Atoi(l.Count)
			switch l.Type {
			case "stmt", "cond":
				fc.Lines = append(fc.Lines, covmodel.LineCov{Number: lineNo, Hits: count})
			case "method":
				fc.Funcs = append(fc.Funcs, covmodel.FuncCov{Line: lineNo, Calls: count})
			}
		}

		// Fall back to metrics block when no line-level data is available.
		if len(fc.Lines) == 0 && xf.Metrics != nil {
			m := xf.Metrics
			for i := 0; i < m.Statements; i++ {
				hits := 0
				if i < m.CoveredStatements {
					hits = 1
				}
				fc.Lines = append(fc.Lines, covmodel.LineCov{Number: i + 1, Hits: hits})
			}
		}

		fc.Compute()
		rep.Files = append(rep.Files, fc)
	}

	for _, f := range cov.Project.Files {
		addFile(f, "")
	}
	for _, pkg := range cov.Project.Packages {
		for _, f := range pkg.Files {
			addFile(f, pkg.Name)
		}
	}

	return rep, nil
}
