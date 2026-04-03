// Package cucumber parses Cucumber JSON test result files.
package cucumber

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/Yanujz/trep/pkg/model"
	"github.com/Yanujz/trep/pkg/parser"
)

func init() { parser.Register(Parser{}) }

// Parser handles Cucumber JSON output.
type Parser struct{}

func (Parser) Name() string         { return "cucumber" }
func (Parser) Extensions() []string { return nil } // rely on content detection only

// Detect checks for the Cucumber JSON array-of-features structure.
func (Parser) Detect(header []byte) bool {
	s := strings.TrimSpace(string(header))
	if !strings.HasPrefix(s, "[") {
		return false
	}
	return strings.Contains(s, `"elements"`) || strings.Contains(s, `"uri"`)
}

type cucumberFeature struct {
	Name     string            `json:"name"`
	URI      string            `json:"uri"`
	Elements []cucumberElement `json:"elements"`
}

type cucumberElement struct {
	Name  string         `json:"name"`
	Type  string         `json:"type"`
	Steps []cucumberStep `json:"steps"`
}

type cucumberStep struct {
	Name   string         `json:"name"`
	Result cucumberResult `json:"result"`
}

type cucumberResult struct {
	Status       string `json:"status"`
	DurationNs   int64  `json:"duration"` // nanoseconds
	ErrorMessage string `json:"error_message"`
}

func (Parser) Parse(r io.Reader, source string) (*model.Report, error) {
	var features []cucumberFeature
	if err := json.NewDecoder(r).Decode(&features); err != nil {
		return nil, err
	}

	rep := &model.Report{
		Sources:   []string{source},
		Timestamp: time.Now().UTC(),
	}

	for _, f := range features {
		suiteName := f.Name
		if suiteName == "" {
			suiteName = f.URI
		}
		suite := model.Suite{Name: suiteName}
		for _, el := range f.Elements {
			if el.Type == "background" {
				continue
			}
			// Determine overall scenario status from steps.
			status := model.StatusPass
			var messages []string
			var totalDur time.Duration
			for _, step := range el.Steps {
				totalDur += time.Duration(step.Result.DurationNs)
				switch strings.ToLower(step.Result.Status) {
				case "failed":
					if status != model.StatusFail {
						status = model.StatusFail
					}
					if step.Result.ErrorMessage != "" {
						messages = append(messages, step.Result.ErrorMessage)
					}
				case "skipped", "pending", "undefined":
					if status == model.StatusPass {
						status = model.StatusSkip
					}
				}
			}
			tc := model.TestCase{
				Suite:    suiteName,
				Name:     el.Name,
				Status:   status,
				Duration: totalDur,
				Message:  strings.Join(messages, "\n"),
			}
			suite.Cases = append(suite.Cases, tc)
		}
		rep.Suites = append(rep.Suites, suite)
	}

	return rep, nil
}
