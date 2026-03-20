// Package lcov parses LCOV .info coverage files produced by gcov, Istanbul/nyc,
// Rust tarpaulin, and any other tool emitting the LCOV format.
package lcov

import (
	"bufio"
	"io"
	"strconv"
	"strings"
	"time"

	covmodel "github.com/trep-dev/trep/pkg/coverage/model"
	covparser "github.com/trep-dev/trep/pkg/coverage/parser"
)

func init() { covparser.Register(Parser{}) }

// Parser handles LCOV .info files.
type Parser struct{}

// Name returns the parser identifier.
func (Parser) Name() string { return "lcov" }

// Extensions returns the file extensions this parser handles.
func (Parser) Extensions() []string { return []string{"info"} }

// Detect reports whether header looks like an LCOV .info file.
func (Parser) Detect(header []byte) bool {
	s := string(header)
	return strings.Contains(s, "SF:") || strings.HasPrefix(strings.TrimSpace(s), "TN:")
}

// LCOV record tags we care about:
//   TN:  test name (optional)
//   SF:  source file path
//   FN:  line,funcname
//   FNDA:hits,funcname
//   DA:  line,hits[,checksum]
//   BRDA:line,block,branch,taken
//   end_of_record

// Parse reads an LCOV .info file from r and returns a CovReport.
func (Parser) Parse(r io.Reader, source string) (*covmodel.CovReport, error) {
	rep := &covmodel.CovReport{
		Sources:   []string{source},
		Timestamp: time.Now().UTC(),
	}

	var cur *covmodel.FileCov
	// funcLine maps funcname → declaration line for FNDA lookup.
	funcLine := map[string]int{}
	// funcHits accumulates hits before we can pair with declaration line.
	funcHits := map[string]int{}

	flush := func() {
		if cur == nil {
			return
		}
		// Merge function declarations with their hit counts.
		for i := range cur.Funcs {
			fn := &cur.Funcs[i]
			if h, ok := funcHits[fn.Name]; ok {
				fn.Calls = h
			}
		}
		cur.Compute()
		rep.Files = append(rep.Files, cur)
		cur = nil
		funcLine = map[string]int{}
		funcHits = map[string]int{}
	}

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		tag, val, _ := strings.Cut(line, ":")
		tag = strings.TrimSpace(tag)
		val = strings.TrimSpace(val)

		switch tag {
		case "SF":
			flush()
			cur = &covmodel.FileCov{Path: val}

		case "FN":
			if cur == nil {
				continue
			}
			// FN:line_number,function_name
			parts := strings.SplitN(val, ",", 2)
			if len(parts) != 2 {
				continue
			}
			lineNo, _ := strconv.Atoi(parts[0])
			name := strings.TrimSpace(parts[1])
			cur.Funcs = append(cur.Funcs, covmodel.FuncCov{Name: name, Line: lineNo})
			funcLine[name] = lineNo

		case "FNDA":
			if cur == nil {
				continue
			}
			// FNDA:hit_count,function_name
			parts := strings.SplitN(val, ",", 2)
			if len(parts) != 2 {
				continue
			}
			hits, _ := strconv.Atoi(parts[0])
			name := strings.TrimSpace(parts[1])
			funcHits[name] = hits

		case "DA":
			if cur == nil {
				continue
			}
			// DA:line_number,hit_count[,checksum]
			parts := strings.SplitN(val, ",", 3)
			if len(parts) < 2 {
				continue
			}
			lineNo, err1 := strconv.Atoi(parts[0])
			hits, err2 := strconv.Atoi(parts[1])
			if err1 != nil || err2 != nil {
				continue
			}
			cur.Lines = append(cur.Lines, covmodel.LineCov{Number: lineNo, Hits: hits})

		case "BRDA":
			if cur == nil {
				continue
			}
			// BRDA:line,block,branch,taken  (taken is count or '-')
			parts := strings.SplitN(val, ",", 4)
			if len(parts) != 4 {
				continue
			}
			lineNo, _ := strconv.Atoi(parts[0])
			block, _ := strconv.Atoi(parts[1])
			branch, _ := strconv.Atoi(parts[2])
			taken := -1
			if parts[3] != "-" {
				taken, _ = strconv.Atoi(parts[3])
			}
			cur.Branches = append(cur.Branches, covmodel.BranchCov{
				Line: lineNo, Block: block, Index: branch, Taken: taken,
			})

		case "end_of_record":
			flush()
		}
	}
	flush() // last record may lack end_of_record

	return rep, sc.Err()
}
