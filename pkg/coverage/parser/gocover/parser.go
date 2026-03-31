// Package gocover parses Go coverage profiles produced by
// go test -coverprofile=coverage.out (mode: set | count | atomic).
package gocover

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	covmodel "github.com/Yanujz/trep/pkg/coverage/model"
	covparser "github.com/Yanujz/trep/pkg/coverage/parser"
)

func init() { covparser.Register(Parser{}) }

// Parser handles Go coverage profiles.
type Parser struct{}

// Name returns the parser identifier.
func (Parser) Name() string { return "gocover" }

// Extensions returns the file extensions this parser handles.
func (Parser) Extensions() []string { return []string{"out"} }

// Detect reports whether header looks like a Go coverage profile.
func (Parser) Detect(header []byte) bool {
	s := strings.TrimSpace(string(header))
	return strings.HasPrefix(s, "mode: set") ||
		strings.HasPrefix(s, "mode: count") ||
		strings.HasPrefix(s, "mode: atomic")
}

// Parse reads a Go coverage profile from r and returns a CovReport.
// A coverprofile line looks like:
//
//	github.com/example/pkg/file.go:10.15,12.3 2 1
//	                                           ^stmts ^count
func (Parser) Parse(r io.Reader, source string) (*covmodel.CovReport, error) {
	rep := &covmodel.CovReport{
		Sources:   []string{source},
		Timestamp: time.Now().UTC(),
	}

	// fileMap accumulates per-file statement blocks.
	type block struct {
		startLine, endLine int
		stmts, count       int
	}
	fileMap := map[string][]block{}
	fileOrder := []string{}

	rBuf := bufio.NewReader(r)

	for {
		line, err := rBuf.ReadString('\n')
		if len(line) == 0 && err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("gocover: read error: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}

		// Format: file.go:startLine.startCol,endLine.endCol stmts count
		colonIdx := strings.LastIndex(line, ":")
		if colonIdx < 0 {
			continue
		}
		filePath := line[:colonIdx]
		rest := line[colonIdx+1:]

		// rest: "10.15,12.3 2 1"
		parts := strings.Fields(rest)
		if len(parts) < 3 {
			continue
		}
		coords := strings.Split(parts[0], ",")
		if len(coords) != 2 {
			continue
		}

		startLine, err1 := parseLineCol(coords[0])
		endLine, err2 := parseLineCol(coords[1])
		stmts, err3 := strconv.Atoi(parts[1])
		count, err4 := strconv.Atoi(parts[2])
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			continue
		}

		if _, exists := fileMap[filePath]; !exists {
			fileOrder = append(fileOrder, filePath)
			fileMap[filePath] = nil
		}
		fileMap[filePath] = append(fileMap[filePath], block{startLine, endLine, stmts, count})
	}

	// Convert blocks → per-line coverage data.
	// We synthesise one LineCov per statement block (Go profiles don't give
	// individual line data — each block covers a range of lines with stmts statements).
	for _, path := range fileOrder {
		fc := &covmodel.FileCov{Path: path}
		seenLines := map[int]bool{}

		for _, b := range fileMap[path] {
			for ln := b.startLine; ln <= b.endLine; ln++ {
				if seenLines[ln] {
					continue
				}
				seenLines[ln] = true
				hits := b.count
				if b.stmts == 0 {
					hits = 0
				}
				fc.Lines = append(fc.Lines, covmodel.LineCov{Number: ln, Hits: hits})
			}
		}

		fc.Compute()
		rep.Files = append(rep.Files, fc)
	}

	return rep, nil
}

// parseLineCol parses "10.15" → 10 (line number only; column ignored).
func parseLineCol(s string) (int, error) {
	parts := strings.SplitN(s, ".", 2)
	return strconv.Atoi(parts[0])
}
