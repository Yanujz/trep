// Package nunit parses NUnit 2/3 and xUnit .NET XML test result files.
package nunit

import (
	"encoding/xml"
	"io"
	"strings"
	"time"

	"github.com/Yanujz/trep/pkg/model"
	"github.com/Yanujz/trep/pkg/parser"
)

func init() { parser.Register(Parser{}) }

// Parser handles NUnit and xUnit .NET XML output.
type Parser struct{}

func (Parser) Name() string         { return "nunit" }
func (Parser) Extensions() []string { return nil } // rely on content detection only
func (Parser) Detect(header []byte) bool {
	s := string(header)
	return strings.Contains(s, "<test-run") ||
		strings.Contains(s, "<test-results") ||
		strings.Contains(s, "<assemblies")
}

// nunitRun covers NUnit 3 <test-run> root.
type nunitRun struct {
	XMLName xml.Name     `xml:"test-run"`
	Suites  []nunitSuite `xml:"test-suite"`
}

// nunitResults covers NUnit 2 <test-results> root.
type nunitResults struct {
	XMLName xml.Name     `xml:"test-results"`
	Suites  []nunitSuite `xml:"test-suite"`
}

type nunitSuite struct {
	Name  string       `xml:"name,attr"`
	Cases []nunitCase  `xml:"test-case"`
	Inner []nunitSuite `xml:"test-suite"`
}

type nunitCase struct {
	Name     string  `xml:"name,attr"`
	FullName string  `xml:"fullname,attr"`
	Result   string  `xml:"result,attr"`
	Duration float64 `xml:"duration,attr"`
	Failure  *struct {
		Message    string `xml:"message"`
		StackTrace string `xml:"stack-trace"`
	} `xml:"failure"`
	Reason *struct {
		Message string `xml:"message"`
	} `xml:"reason"`
}

// xUnit types
type xunitAssemblies struct {
	XMLName    xml.Name        `xml:"assemblies"`
	Assemblies []xunitAssembly `xml:"assembly"`
}

type xunitAssembly struct {
	Name        string            `xml:"name,attr"`
	Collections []xunitCollection `xml:"collection"`
}

type xunitCollection struct {
	Name  string      `xml:"name,attr"`
	Tests []xunitTest `xml:"test"`
}

type xunitTest struct {
	Name    string  `xml:"name,attr"`
	Method  string  `xml:"method,attr"`
	Result  string  `xml:"result,attr"`
	Time    float64 `xml:"time,attr"`
	Failure *struct {
		Message    string `xml:"message"`
		StackTrace string `xml:"stack-trace"`
	} `xml:"failure"`
	Reason *struct {
		Message string `xml:"message"`
	} `xml:"reason"`
}

func (Parser) Parse(r io.Reader, source string) (*model.Report, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	rep := &model.Report{
		Sources:   []string{source},
		Timestamp: time.Now().UTC(),
	}

	s := string(data)

	if strings.Contains(s, "<assemblies") {
		// xUnit format
		var doc xunitAssemblies
		if err := xml.Unmarshal(data, &doc); err != nil {
			return nil, err
		}
		for _, asm := range doc.Assemblies {
			for _, col := range asm.Collections {
				suiteName := col.Name
				if suiteName == "" {
					suiteName = asm.Name
				}
				suite := model.Suite{Name: suiteName}
				for _, t := range col.Tests {
					tc := model.TestCase{
						Suite:    suiteName,
						Name:     t.Name,
						Duration: time.Duration(t.Time * float64(time.Second)),
						Status:   xunitStatus(t.Result),
					}
					if t.Failure != nil {
						tc.Message = strings.TrimSpace(t.Failure.Message + "\n" + t.Failure.StackTrace)
					}
					suite.Cases = append(suite.Cases, tc)
				}
				rep.Suites = append(rep.Suites, suite)
			}
		}
	} else if strings.Contains(s, "<test-results") {
		// NUnit 2
		var doc nunitResults
		if err := xml.Unmarshal(data, &doc); err != nil {
			return nil, err
		}
		for _, ns := range doc.Suites {
			collectNunitSuites(rep, &ns)
		}
	} else {
		// NUnit 3 test-run
		var doc nunitRun
		if err := xml.Unmarshal(data, &doc); err != nil {
			return nil, err
		}
		for _, ns := range doc.Suites {
			collectNunitSuites(rep, &ns)
		}
	}

	return rep, nil
}

func collectNunitSuites(rep *model.Report, ns *nunitSuite) {
	if len(ns.Cases) > 0 {
		suiteName := ns.Name
		suite := model.Suite{Name: suiteName}
		for _, c := range ns.Cases {
			tc := model.TestCase{
				Suite:    suiteName,
				Name:     c.Name,
				Duration: time.Duration(c.Duration * float64(time.Second)),
				Status:   nunitStatus(c.Result),
			}
			if c.Failure != nil {
				tc.Message = strings.TrimSpace(c.Failure.Message + "\n" + c.Failure.StackTrace)
			} else if c.Reason != nil {
				tc.Message = c.Reason.Message
			}
			suite.Cases = append(suite.Cases, tc)
		}
		rep.Suites = append(rep.Suites, suite)
	}
	for _, inner := range ns.Inner {
		collectNunitSuites(rep, &inner)
	}
}

func nunitStatus(result string) model.Status {
	switch strings.ToLower(result) {
	case "passed", "success":
		return model.StatusPass
	case "failed", "error":
		return model.StatusFail
	default:
		return model.StatusSkip
	}
}

func xunitStatus(result string) model.Status {
	switch strings.ToLower(result) {
	case "pass":
		return model.StatusPass
	case "fail":
		return model.StatusFail
	default:
		return model.StatusSkip
	}
}
