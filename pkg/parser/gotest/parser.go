// Package gotest parses the streaming JSON output produced by `go test -json`.
package gotest

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/trep-dev/trep/pkg/model"
	"github.com/trep-dev/trep/pkg/parser"
)

func init() { parser.Register(Parser{}) }

// Parser handles `go test -json` output.
type Parser struct{}

func (Parser) Name() string         { return "gotest" }
func (Parser) Extensions() []string { return []string{} } // typically piped via stdin
func (Parser) Detect(header []byte) bool {
	s := string(header)
	// The JSON stream uses "Action" and "Package" keys on every line.
	return strings.Contains(s, `"Action"`) && strings.Contains(s, `"Package"`)
}

// event mirrors one line of `go test -json` output.
type event struct {
	Time    time.Time `json:"Time"`
	Action  string    `json:"Action"`
	Package string    `json:"Package"`
	Test    string    `json:"Test"`
	Elapsed float64   `json:"Elapsed"`
	Output  string    `json:"Output"`
}

func (Parser) Parse(r io.Reader, source string) (*model.Report, error) {
	rep := &model.Report{
		Sources:   []string{source},
		Timestamp: time.Now().UTC(),
	}

	type key struct{ pkg, test string }
	type state struct{ buf strings.Builder }

	states     := make(map[key]*state)
	suiteMap   := make(map[string]*model.Suite)
	suiteOrder := []string{}
	var maxElapsed float64

	sc := bufio.NewScanner(r)
	// A single test's accumulated output can be large; give the scanner room.
	sc.Buffer(make([]byte, 1<<20), 1<<20)

	for sc.Scan() {
		var ev event
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue // skip garbled / non-JSON lines
		}

		// Package-level events have no Test field.
		if ev.Test == "" {
			if ev.Action == "start" && rep.Timestamp.IsZero() {
				rep.Timestamp = ev.Time.UTC()
			}
			continue
		}

		k := key{ev.Package, ev.Test}

		switch ev.Action {
		case "run":
			states[k] = &state{}

		case "output":
			if st, ok := states[k]; ok {
				st.buf.WriteString(ev.Output)
			}

		case "pass", "fail", "skip":
			st := states[k]
			if st == nil {
				st = &state{}
			}
			delete(states, k)

			if ev.Elapsed > maxElapsed {
				maxElapsed = ev.Elapsed
			}

			suiteName := ev.Package
			if suiteName == "" {
				suiteName = "default"
			}

			rawOut := st.buf.String()
			c := model.TestCase{
				Suite:    suiteName,
				Name:     ev.Test,
				Duration: time.Duration(ev.Elapsed * float64(time.Second)),
			}

			switch ev.Action {
			case "pass":
				c.Status = model.StatusPass

			case "skip":
				c.Status  = model.StatusSkip
				c.Message = extractSkipReason(rawOut)

			case "fail":
				c.Status  = model.StatusFail
				c.Message = extractFailMessage(rawOut)
				c.Stdout  = strings.TrimSpace(rawOut)
			}

			ensureSuite(suiteMap, &suiteOrder, suiteName)
			suiteMap[suiteName].Cases = append(suiteMap[suiteName].Cases, c)
		}
	}

	if maxElapsed > 0 {
		rep.Duration = time.Duration(maxElapsed * float64(time.Second))
	}
	for _, name := range suiteOrder {
		rep.Suites = append(rep.Suites, *suiteMap[name])
	}
	return rep, sc.Err()
}

// ensureSuite creates a Suite entry if it does not already exist.
func ensureSuite(m map[string]*model.Suite, order *[]string, name string) {
	if _, ok := m[name]; !ok {
		m[name] = &model.Suite{Name: name}
		*order = append(*order, name)
	}
}

// extractSkipReason returns the first non-boilerplate line from skip output.
func extractSkipReason(out string) string {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--- SKIP:") || strings.HasPrefix(line, "=== RUN") {
			continue
		}
		return line
	}
	return ""
}

// extractFailMessage strips boilerplate header/footer lines and returns the
// meaningful failure content, capped to 2 KiB.
func extractFailMessage(out string) string {
	var lines []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "=== RUN") {
			continue
		}
		lines = append(lines, line)
	}
	msg := strings.Join(lines, "\n")
	const maxLen = 2048
	if len(msg) > maxLen {
		return msg[:maxLen] + "\n…"
	}
	return msg
}
