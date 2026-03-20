// Package sarif renders test and coverage results into SARIF 2.1.0 format,
// which GitHub Advanced Security can ingest and display as code scanning alerts.
package sarif

import (
	"encoding/json"
	"fmt"
	"io"

	covmodel "github.com/trep-dev/trep/pkg/coverage/model"
	"github.com/trep-dev/trep/pkg/model"
)

const (
	schemaURI   = "https://schemastore.azurewebsites.net/schemas/json/sarif-2.1.0.json"
	sarifVersion = "2.1.0"
	infoURI      = "https://github.com/Yanujz/trep"

	ruleTestFailure        = "test-failure"
	ruleCoverageThreshold  = "coverage-below-threshold"
)

// ── SARIF document types ──────────────────────────────────────────────────────

type document struct {
	Schema  string `json:"$schema"`
	Version string `json:"version"`
	Runs    []run  `json:"runs"`
}

type run struct {
	Tool    tool     `json:"tool"`
	Results []result `json:"results"`
}

type tool struct {
	Driver driver `json:"driver"`
}

type driver struct {
	Name           string  `json:"name"`
	Version        string  `json:"version"`
	InformationURI string  `json:"informationUri"`
	Rules          []rule  `json:"rules"`
}

type rule struct {
	ID                   string              `json:"id"`
	ShortDescription     message             `json:"shortDescription"`
	FullDescription      message             `json:"fullDescription"`
	DefaultConfiguration ruleConfig          `json:"defaultConfiguration"`
	HelpURI              string              `json:"helpUri"`
}

type ruleConfig struct {
	Level string `json:"level"`
}

type result struct {
	RuleID    string     `json:"ruleId"`
	Level     string     `json:"level"`
	Message   message    `json:"message"`
	Locations []location `json:"locations,omitempty"`
}

type message struct {
	Text string `json:"text"`
}

type location struct {
	PhysicalLocation physicalLocation `json:"physicalLocation"`
}

type physicalLocation struct {
	ArtifactLocation artifactLocation `json:"artifactLocation"`
	Region           *region          `json:"region,omitempty"`
}

type artifactLocation struct {
	URI       string `json:"uri"`
	URIBaseID string `json:"uriBaseId,omitempty"`
}

type region struct {
	StartLine int `json:"startLine"`
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func fileLocation(uri string, line int) []location {
	if uri == "" {
		return nil
	}
	al := artifactLocation{URI: uri, URIBaseID: "%SRCROOT%"}
	pl := physicalLocation{ArtifactLocation: al}
	if line > 0 {
		pl.Region = &region{StartLine: line}
	}
	return []location{{PhysicalLocation: pl}}
}

func write(w io.Writer, doc document) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}

// ── Test renderer ─────────────────────────────────────────────────────────────

// RenderTest writes a SARIF 2.1.0 document for failed test cases to w.
// Each failing test case becomes one SARIF result with level "error".
// Passing and skipped tests are omitted; SARIF alerts are problem-only.
// toolVersion is embedded in the tool.driver block (pass the build version string).
func RenderTest(w io.Writer, rep *model.Report, toolVersion string) error {
	rules := []rule{
		{
			ID:               ruleTestFailure,
			ShortDescription: message{Text: "Test case failed"},
			FullDescription:  message{Text: "A test case reported a failure or error."},
			DefaultConfiguration: ruleConfig{Level: "error"},
			HelpURI:          infoURI,
		},
	}

	var results []result
	for _, s := range rep.Suites {
		for _, c := range s.Cases {
			if c.Status != model.StatusFail {
				continue
			}
			text := c.Name
			if c.Message != "" {
				text = fmt.Sprintf("%s: %s", c.Name, c.Message)
			}
			r := result{
				RuleID:    ruleTestFailure,
				Level:     "error",
				Message:   message{Text: text},
				Locations: fileLocation(c.File, c.Line),
			}
			results = append(results, r)
		}
	}
	if results == nil {
		results = []result{} // ensure JSON array, not null
	}

	doc := document{
		Schema:  schemaURI,
		Version: sarifVersion,
		Runs: []run{{
			Tool: tool{Driver: driver{
				Name:           "trep",
				Version:        toolVersion,
				InformationURI: infoURI,
				Rules:          rules,
			}},
			Results: results,
		}},
	}
	return write(w, doc)
}

// ── Coverage renderer ─────────────────────────────────────────────────────────

// RenderCov writes a SARIF 2.1.0 document for coverage results to w.
//
// When threshold > 0, every file whose line coverage is below the threshold is
// emitted as a SARIF result with level "warning".  A threshold of 0 means no
// threshold is enforced; an empty results array is still produced so that a
// previous upload's alerts are cleared by GitHub Advanced Security.
//
// toolVersion is embedded in the tool.driver block.
func RenderCov(w io.Writer, rep *covmodel.CovReport, threshold float64, toolVersion string) error {
	rules := []rule{
		{
			ID:               ruleCoverageThreshold,
			ShortDescription: message{Text: "File coverage below threshold"},
			FullDescription: message{
				Text: fmt.Sprintf(
					"The file's line coverage is below the minimum threshold of %.1f%%.", threshold),
			},
			DefaultConfiguration: ruleConfig{Level: "warning"},
			HelpURI:              infoURI,
		},
	}

	var results []result
	if threshold > 0 {
		for _, f := range rep.Files {
			if f.LinePct() < threshold {
				text := fmt.Sprintf(
					"%s: line coverage %.1f%% is below the threshold of %.1f%%.",
					f.Path, f.LinePct(), threshold)
				r := result{
					RuleID:    ruleCoverageThreshold,
					Level:     "warning",
					Message:   message{Text: text},
					Locations: fileLocation(f.Path, 0),
				}
				results = append(results, r)
			}
		}
	}
	if results == nil {
		results = []result{}
	}

	doc := document{
		Schema:  schemaURI,
		Version: sarifVersion,
		Runs: []run{{
			Tool: tool{Driver: driver{
				Name:           "trep",
				Version:        toolVersion,
				InformationURI: infoURI,
				Rules:          rules,
			}},
			Results: results,
		}},
	}
	return write(w, doc)
}
