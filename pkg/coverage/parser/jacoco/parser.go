package jacoco

import (
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
	"time"

	covmodel "github.com/Yanujz/trep/pkg/coverage/model"
	covparser "github.com/Yanujz/trep/pkg/coverage/parser"
)

func init() { covparser.Register(Parser{}) }

// Parser handles JaCoCo XML files.
type Parser struct{}

// Name returns the parser identifier.
func (Parser) Name() string { return "jacoco" }

// Extensions returns the file extensions this parser handles.
func (Parser) Extensions() []string { return []string{"xml"} }

// Detect reports whether header looks like a JaCoCo XML file.
func (Parser) Detect(header []byte) bool {
	s := strings.ToLower(string(header))
	return strings.Contains(s, "<report") && strings.Contains(s, "jacoco")
}

// ── XML wire types ────────────────────────────────────────────────────────────

type xmlReport struct {
	XMLName  xml.Name     `xml:"report"`
	Packages []xmlPackage `xml:"package"`
}

type xmlPackage struct {
	Name        string          `xml:"name,attr"`
	Classes     []xmlClass      `xml:"class"`
	Sourcefiles []xmlSourcefile `xml:"sourcefile"`
}

type xmlClass struct {
	Name           string      `xml:"name,attr"`
	Sourcefilename string      `xml:"sourcefilename,attr"`
	Methods        []xmlMethod `xml:"method"`
}

type xmlMethod struct {
	Name string `xml:"name,attr"`
	Line string `xml:"line,attr"`
}

type xmlSourcefile struct {
	Name  string    `xml:"name,attr"`
	Lines []xmlLine `xml:"line"`
}

type xmlLine struct {
	Nr string `xml:"nr,attr"`
	Ci string `xml:"ci,attr"`
	Cb string `xml:"cb,attr"`
}

// ── Parser ────────────────────────────────────────────────────────────────────

// Parse reads a JaCoCo XML file from r and returns a CovReport.
func (Parser) Parse(r io.Reader, source string) (*covmodel.CovReport, error) {
	var cov xmlReport
	dec := xml.NewDecoder(r)
	if err := dec.Decode(&cov); err != nil {
		return nil, fmt.Errorf("jacoco: XML decode: %w", err)
	}

	rep := &covmodel.CovReport{
		Sources:   []string{source},
		Timestamp: time.Now().UTC(),
	}

	for _, pkg := range cov.Packages {
		pkgName := pkg.Name

		// Create a map to cache class method names to lines, although it's rough
		classMethods := make(map[string][]covmodel.FuncCov)
		for _, cls := range pkg.Classes {
			for _, m := range cls.Methods {
				lineNo, _ := strconv.Atoi(m.Line)
				classMethods[cls.Sourcefilename] = append(classMethods[cls.Sourcefilename], covmodel.FuncCov{
					Name: m.Name,
					Line: lineNo,
				})
			}
		}

		for _, sf := range pkg.Sourcefiles {
			fullPath := sf.Name
			if pkgName != "" {
				fullPath = path.Join(pkgName, sf.Name)
			}

			fc := &covmodel.FileCov{
				Path: fullPath,
			}

			for _, l := range sf.Lines {
				lineNo, err := strconv.Atoi(l.Nr)
				if err != nil {
					continue
				}

				ci, _ := strconv.Atoi(l.Ci)
				cb, _ := strconv.Atoi(l.Cb)

				hits := 0
				if ci > 0 || cb > 0 {
					hits = 1 // Simplified: if any covered instructions or branches, we count line as hit
				}

				fc.Lines = append(fc.Lines, covmodel.LineCov{
					Number: lineNo,
					Hits:   hits,
				})
			}

			// Add methods found in classes referencing this source filename
			if funcs, ok := classMethods[sf.Name]; ok {
				// We don't have easily accessible call counts per method in standard jacoco method node
				// unless we parse method counters.
				// We'll approximate method coverage if it exists in the covered lines.
				coveredLines := make(map[int]bool)
				for _, l := range fc.Lines {
					if l.Hits > 0 {
						coveredLines[l.Number] = true
					}
				}

				for _, fn := range funcs {
					calls := 0
					if fn.Line > 0 && coveredLines[fn.Line] {
						calls = 1
					}
					fc.Funcs = append(fc.Funcs, covmodel.FuncCov{
						Name:  fn.Name,
						Line:  fn.Line,
						Calls: calls,
					})
				}
			}

			fc.Compute()
			rep.Files = append(rep.Files, fc)
		}
	}

	return rep, nil
}
