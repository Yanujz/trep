// Package codeowners implements a lightweight CODEOWNERS file parser.
package codeowners

import (
	"bufio"
	"io"
	"path"
	"strings"
)

// Owners maps file path patterns to their owners.
type Owners struct {
	rules []rule
}

type rule struct {
	pattern string
	owners  []string
}

// Parse reads a CODEOWNERS file and returns an Owners resolver.
// Lines starting with # are comments; blank lines are skipped.
// Each non-blank, non-comment line is: <pattern> <owner> [<owner> ...]
func Parse(r io.Reader) *Owners {
	var rules []rule
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		rules = append(rules, rule{pattern: parts[0], owners: parts[1:]})
	}
	// Reverse: last matching rule wins (CODEOWNERS semantics).
	for i, j := 0, len(rules)-1; i < j; i, j = i+1, j-1 {
		rules[i], rules[j] = rules[j], rules[i]
	}
	return &Owners{rules: rules}
}

// Match returns the owners for the given file path.
// Returns nil if no rule matches.
func (o *Owners) Match(filePath string) []string {
	filePath = "/" + strings.TrimPrefix(path.Clean("/"+filePath), "/")
	for _, r := range o.rules {
		pat := r.pattern
		if !strings.HasPrefix(pat, "/") {
			// Relative pattern matches anywhere in the tree.
			base := path.Base(filePath)
			if ok, _ := path.Match(pat, base); ok {
				return r.owners
			}
			// Also try matching the full path suffix.
			if matchSuffix(filePath, pat) {
				return r.owners
			}
		} else {
			if ok, _ := path.Match(pat, filePath); ok {
				return r.owners
			}
			// Directory prefix match.
			if strings.HasSuffix(pat, "/") && strings.HasPrefix(filePath, pat) {
				return r.owners
			}
			if strings.HasSuffix(pat, "/*") {
				dir := strings.TrimSuffix(pat, "/*")
				if strings.HasPrefix(filePath, dir+"/") {
					return r.owners
				}
			}
		}
	}
	return nil
}

func matchSuffix(filePath, pattern string) bool {
	parts := strings.Split(filePath, "/")
	for i := range parts {
		candidate := strings.Join(parts[i:], "/")
		if ok, _ := path.Match(pattern, candidate); ok {
			return true
		}
	}
	return false
}
