// Package okf checks OKF v0.2 bundles.
//
// The spec is deliberately permissive: only three rules are conformance
// failures. That split is why findings carry a severity — everything beyond
// §11 is advice and must never fail a build. A repo whose graph carries an
// invariant the spec cannot express passes it to [CheckBundle] as a [Rule].
package okf

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type Severity int

const (
	Warning Severity = iota
	Error
)

func (s Severity) String() string {
	if s == Error {
		return "error"
	}
	return "warning"
}

type Finding struct {
	Path string
	Sev  Severity
	Msg  string
}

func (f Finding) String() string {
	return fmt.Sprintf("%s: %s: %s", f.Path, f.Sev, f.Msg)
}

// Doc is one parsed concept, handed to a [Rule].
type Doc struct {
	Root string         // bundle root, as given to CheckBundle
	Rel  string         // bundle-relative path, slash-separated
	FM   map[string]any // frontmatter
	Body string
}

// Rule is a caller-supplied check over one concept. It runs on concepts only,
// never on the reserved index.md and log.md. A Finding with an empty Path is
// stamped with the concept's.
type Rule func(Doc) []Finding

const delim = "---"

// Parse splits an OKF document into its YAML frontmatter and body. A document
// with no frontmatter block yields a nil map and no error; whether that is a
// violation depends on the filename, which Parse does not know.
func Parse(text string) (map[string]any, string, error) {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != delim {
		return nil, text, nil
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == delim {
			end = i
			break
		}
	}
	if end < 0 {
		return nil, "", fmt.Errorf("unterminated YAML frontmatter block")
	}
	var fm map[string]any
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &fm); err != nil {
		return nil, "", fmt.Errorf("invalid YAML in frontmatter: %w", err)
	}
	if fm == nil {
		fm = map[string]any{}
	}
	return fm, strings.Join(lines[end+1:], "\n"), nil
}
