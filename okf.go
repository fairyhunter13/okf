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
	"time"

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
	// FMText is the frontmatter as written. A rule needs it to grade how a
	// value was spelled: YAML decodes `2026-12-31` and `2026-12-31T00:00:00Z`
	// to the same time.Time, so FM alone cannot tell an offset was omitted.
	FMText string
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
	fmText, body, ok, err := splitFrontmatter(text)
	if err != nil || !ok {
		return nil, body, err
	}
	var fm map[string]any
	if err := yaml.Unmarshal([]byte(fmText), &fm); err != nil {
		return nil, "", fmt.Errorf("invalid YAML in frontmatter: %w", err)
	}
	if fm == nil {
		fm = map[string]any{}
	}
	return fm, body, nil
}

// splitFrontmatter cuts the leading `---` block off. ok is false when there is
// no block at all, which Parse reports as a nil map rather than an error.
func splitFrontmatter(text string) (fmText, body string, ok bool, err error) {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != delim {
		return "", text, false, nil
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == delim {
			return strings.Join(lines[1:i], "\n"), strings.Join(lines[i+1:], "\n"), true, nil
		}
	}
	return "", "", false, fmt.Errorf("unterminated YAML frontmatter block")
}

// frontmatterText is the frontmatter as written, for [Doc.FMText].
func frontmatterText(text string) string {
	fmText, _, _, _ := splitFrontmatter(text)
	return fmText
}

// ParseTimestamp reads an OKF timestamp. §5 requires an ISO 8601 datetime with
// an explicit UTC offset, and YAML hands those over already decoded; the date
// and quoted forms are still parsed here because grading how a value was
// spelled is a producer rule, not the engine's job.
func ParseTimestamp(raw any) (time.Time, error) {
	if t, ok := raw.(time.Time); ok {
		return t, nil
	}
	s := fmt.Sprintf("%v", raw)
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unparsable: %q", s)
}
