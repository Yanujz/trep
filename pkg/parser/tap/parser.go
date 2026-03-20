// Package tap parses TAP (Test Anything Protocol) streams, versions 12 and 13.
package tap

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/trep-dev/trep/pkg/model"
	"github.com/trep-dev/trep/pkg/parser"
)

func init() { parser.Register(Parser{}) }

// Parser handles TAP 12/13 output.
type Parser struct{}

// Name returns the parser identifier.
func (Parser) Name() string { return "tap" }

// Extensions returns the file extensions this parser handles.
func (Parser) Extensions() []string { return []string{"tap"} }

// Detect reports whether header looks like TAP 12/13 output.
func (Parser) Detect(header []byte) bool {
	s := strings.TrimSpace(string(header))
	return strings.HasPrefix(s, "TAP version") || rePlan.MatchString(s)
}

var (
	rePlan = regexp.MustCompile(`^\d+\.\.\d+`)
	reTest = regexp.MustCompile(`^(ok|not ok)\s+\d+\s*(?:-\s*)?(.*)`)
	reSkip = regexp.MustCompile(`(?i)#\s*SKIP\b\s*(.*)`)
	reTime = regexp.MustCompile(`(?i)#\s*time=(\d+(?:\.\d+)?)`)
)

// Parse reads a TAP stream from r and returns a Report.
func (Parser) Parse(r io.Reader, source string) (*model.Report, error) {
	rep := &model.Report{
		Sources:   []string{source},
		Timestamp: time.Now().UTC(),
	}

	suite := model.Suite{Name: source}
	sc := bufio.NewScanner(r)
	inYAML := false // TAP 13 embedded YAML blocks

	for sc.Scan() {
		line := sc.Text()

		// TAP 13: YAML diagnostic block between "  ---" and "  ...".
		if inYAML {
			if strings.TrimSpace(line) == "..." {
				inYAML = false
			}
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "---") {
			inYAML = true
			continue
		}

		m := reTest.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		ok := m[1] == "ok"
		desc := strings.TrimSpace(m[2])

		// Extract optional # time=N.NNN annotation.
		var dur time.Duration
		if tm := reTime.FindStringSubmatch(desc); tm != nil {
			if f, err := strconv.ParseFloat(tm[1], 64); err == nil {
				dur = time.Duration(f * float64(time.Second))
				rep.Duration += dur
			}
			desc = strings.TrimSpace(reTime.ReplaceAllString(desc, ""))
		}
		// Trim trailing whitespace / stray '#' characters.
		desc = strings.TrimRight(desc, " \t#")

		var (
			status model.Status
			msg    string
		)

		switch {
		case reSkip.MatchString(desc):
			sm := reSkip.FindStringSubmatch(desc)
			status = model.StatusSkip
			msg = strings.TrimSpace(sm[1])
			desc = strings.TrimSpace(reSkip.ReplaceAllString(desc, ""))

		case !ok:
			status = model.StatusFail
			// Trailing "# reason" after the description is the failure note.
			if i := strings.LastIndex(desc, " # "); i >= 0 {
				msg = strings.TrimSpace(desc[i+3:])
				desc = strings.TrimSpace(desc[:i])
			}

		default:
			status = model.StatusPass
		}

		if desc == "" {
			desc = "test"
		}

		suite.Cases = append(suite.Cases, model.TestCase{
			Suite:    source,
			Name:     desc,
			Status:   status,
			Duration: dur,
			Message:  msg,
		})
	}

	rep.Suites = []model.Suite{suite}
	return rep, sc.Err()
}
