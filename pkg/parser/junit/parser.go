// Package junit implements a streaming JUnit XML parser that also handles
// Google Test XML output (a compatible superset with optional file/line attrs).
package junit

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/trep-dev/trep/pkg/model"
	"github.com/trep-dev/trep/pkg/parser"
)

func init() { parser.Register(Parser{}) }

// Parser handles JUnit XML and Google Test XML.
type Parser struct{}

// Name returns the parser identifier.
func (Parser) Name() string { return "junit" }

// Extensions returns the file extensions this parser handles.
func (Parser) Extensions() []string { return []string{"xml"} }

// Detect reports whether header looks like JUnit or Google Test XML.
func (Parser) Detect(header []byte) bool {
	s := strings.ToLower(strings.TrimSpace(string(header)))
	return strings.Contains(s, "<testsuites") || strings.Contains(s, "<testsuite")
}

// ── XML wire types ────────────────────────────────────────────────────────────

type xmlTestCase struct {
	XMLName   xml.Name `xml:"testcase"`
	Name      string   `xml:"name,attr"`
	Classname string   `xml:"classname,attr"`
	Time      string   `xml:"time,attr"`
	// Google Test extensions
	File string `xml:"file,attr"`
	Line string `xml:"line,attr"`
	// Result child elements
	Failure   *xmlMsg `xml:"failure"`
	Error     *xmlMsg `xml:"error"`
	Skipped   *xmlMsg `xml:"skipped"`
	SystemOut string  `xml:"system-out"`
}

type xmlMsg struct {
	Message string `xml:"message,attr"`
	Text    string `xml:",chardata"`
}

func (m *xmlMsg) content() string {
	if v := strings.TrimSpace(m.Message); v != "" {
		return v
	}
	return strings.TrimSpace(m.Text)
}

// ── Parser ────────────────────────────────────────────────────────────────────

// Parse reads JUnit XML from r and returns a Report.
func (Parser) Parse(r io.Reader, source string) (*model.Report, error) {
	rep := &model.Report{
		Sources:   []string{source},
		Timestamp: time.Now().UTC(),
	}

	suiteMap := make(map[string]*model.Suite)
	suiteOrder := []string{}

	dec := xml.NewDecoder(r)

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("junit: XML parse error: %w", err)
		}

		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		switch se.Name.Local {
		case "testsuites", "testsuite":
			// Harvest top-level timing / timestamp metadata.
			for _, a := range se.Attr {
				switch a.Name.Local {
				case "time":
					if f, err2 := strconv.ParseFloat(a.Value, 64); err2 == nil && rep.Duration == 0 {
						rep.Duration = floatSecsToDuration(f)
					}
				case "timestamp":
					for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05"} {
						if t, err2 := time.Parse(layout, a.Value); err2 == nil {
							rep.Timestamp = t.UTC()
							break
						}
					}
				}
			}

		case "testcase":
			var tc xmlTestCase
			if err2 := dec.DecodeElement(&tc, &se); err2 != nil {
				// Skip malformed testcase nodes rather than aborting the whole parse.
				continue
			}
			insertTestCase(rep, &tc, suiteMap, &suiteOrder)
		}
	}

	for _, name := range suiteOrder {
		rep.Suites = append(rep.Suites, *suiteMap[name])
	}
	return rep, nil
}

// insertTestCase normalises one <testcase> element and appends it to the suite.
func insertTestCase(
	rep *model.Report,
	tc *xmlTestCase,
	smap map[string]*model.Suite,
	order *[]string,
) {
	// Derive suite name from classname.
	// JUnit convention: "com.example.ClassName.methodName" → "com.example.ClassName"
	// GTest convention: classname == suite name already (e.g. "MyTestSuite")
	suiteName := tc.Classname
	if suiteName == "" {
		suiteName = tc.Name
	}
	// Strip the trailing segment only when the tool embeds the method name in
	// the classname (e.g. "com.example.ClassName.testMethod").  Standard JUnit
	// classnames are already fully-qualified class names, so we must not strip
	// unless the suffix exactly equals the test case name.
	if dot := strings.LastIndex(suiteName, "."); dot > 0 && dot < len(suiteName)-1 {
		if !strings.Contains(suiteName[:dot], "/") && suiteName[dot+1:] == tc.Name {
			suiteName = suiteName[:dot]
		}
	}

	c := model.TestCase{
		Suite:    suiteName,
		Name:     tc.Name,
		Duration: parseFloatSecs(tc.Time),
		File:     tc.File,
	}
	if tc.Line != "" {
		c.Line, _ = strconv.Atoi(tc.Line)
	}

	stdout := strings.TrimSpace(tc.SystemOut)

	switch {
	case tc.Failure != nil:
		c.Status = model.StatusFail
		c.Message = tc.Failure.content()
		c.Stdout = stdout

	case tc.Error != nil:
		c.Status = model.StatusFail
		c.Message = tc.Error.content()
		c.Stdout = stdout

	case tc.Skipped != nil:
		c.Status = model.StatusSkip
		c.Message = tc.Skipped.content()
		// CTest emits a synthetic "SKIP_REGULAR_EXPRESSION_MATCHED" message;
		// extract the human-readable reason from system-out instead.
		if (c.Message == "SKIP_REGULAR_EXPRESSION_MATCHED" || c.Message == "") && stdout != "" {
			for _, line := range strings.Split(stdout, "\n") {
				line = strings.TrimSpace(line)
				if strings.Contains(line, "Skipped") && strings.Contains(line, ": ") {
					c.Message = strings.SplitN(line, ": ", 2)[1]
					break
				}
			}
		}

	default:
		c.Status = model.StatusPass
	}

	if _, exists := smap[suiteName]; !exists {
		smap[suiteName] = &model.Suite{Name: suiteName}
		*order = append(*order, suiteName)
	}
	smap[suiteName].Cases = append(smap[suiteName].Cases, c)
}

func parseFloatSecs(s string) time.Duration {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f <= 0 {
		return 0
	}
	return floatSecsToDuration(f)
}

func floatSecsToDuration(f float64) time.Duration {
	return time.Duration(f * float64(time.Second))
}
