package verify

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// canonicalOrder is §5's key order as the fleet writes it. A new `verified:`
// goes before the first key that follows it here, which is what keeps a stamped
// file's diff to the inserted lines alone.
var canonicalOrder = []string{
	"type", "resource", "title", "description", "tags",
	"status", "generated", "verified", "stale_after", "sources",
}

var (
	topKeyRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*):(.*)$`)
	listItem = regexp.MustCompile(`^\s*-\s`)
	atValRe  = regexp.MustCompile(`(\bat:\s*)([^,}\s]+)`)
)

// ErrHandStamp reports a `verified:` shape the line writer will not touch. The
// fleet writes the flow mapping and the flow list; a block mapping is legal
// YAML that re-serializing would silently reformat, so it is refused instead.
var ErrHandStamp = fmt.Errorf("unsupported verified: shape, stamp by hand")

// Stamp appends a verification event to path, or moves the `at` of the event
// already naming actor. It edits the frontmatter lines and leaves every other
// byte alone: a full YAML round-trip would reflow the two flow-map spellings
// and the tag lists the fleet has in it.
func Stamp(path, actor string, at time.Time) (changed bool, err error) {
	if err := guardHumanActor(actor); err != nil {
		return false, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	out, changed, err := stampText(string(raw), actor, at)
	if err != nil || !changed {
		return false, err
	}
	return true, os.WriteFile(path, []byte(out), 0o644)
}

// guardHumanActor is what VerifiedWellFormed gave up. A `human:` stamp means a
// person read the claims; an agent writing one destroys the only signal that
// separates review from assertion, and CLAUDECODE is how the fleet knows.
func guardHumanActor(actor string) error {
	if !strings.HasPrefix(actor, "human:") {
		return nil
	}
	if os.Getenv("CLAUDECODE") != "" {
		return fmt.Errorf("refusing to write %s under CLAUDECODE: a human stamp is not an agent's to write", actor)
	}
	return nil
}

func stampText(text, actor string, at time.Time) (string, bool, error) {
	lines := strings.Split(text, "\n")
	fmStart, fmEnd, ok := frontmatterBounds(lines)
	if !ok {
		return "", false, fmt.Errorf("no frontmatter block")
	}
	event := fmt.Sprintf("{ by: %s, at: %s }", actor, at.UTC().Format(time.RFC3339))

	vStart, vEnd := blockBounds(lines, fmStart, fmEnd, "verified")
	if vStart < 0 {
		return join(insert(lines, insertionPoint(lines, fmStart, fmEnd), "verified:", "  - "+event)), true, nil
	}

	body, err := verifiedBody(lines, vStart, vEnd)
	if err != nil {
		return "", false, err
	}
	if i := indexOfActor(body, actor); i >= 0 {
		updated := atValRe.ReplaceAllString(body[i], "${1}"+at.UTC().Format(time.RFC3339))
		if updated == body[i] {
			return "", false, nil
		}
		body[i] = updated
		return join(splice(lines, vStart, vEnd, append([]string{"verified:"}, body...))), true, nil
	}
	body = append(body, "  - "+event)
	return join(splice(lines, vStart, vEnd, append([]string{"verified:"}, body...))), true, nil
}

// verifiedBody normalizes §5.2's two spellings into the list form's item lines.
// The bare mapping is carried over verbatim as the first item rather than
// re-emitted, so an existing human stamp keeps the bytes it was typed with.
func verifiedBody(lines []string, start, end int) ([]string, error) {
	m := topKeyRe.FindStringSubmatch(lines[start])
	inline := strings.TrimSpace(m[2])
	if inline != "" {
		if !strings.HasPrefix(inline, "{") {
			return nil, ErrHandStamp
		}
		return []string{"  - " + inline}, nil
	}
	var out []string
	for _, l := range lines[start+1 : end] {
		if strings.TrimSpace(l) == "" {
			continue
		}
		if !listItem.MatchString(l) {
			return nil, ErrHandStamp
		}
		out = append(out, l)
	}
	if len(out) == 0 {
		return nil, ErrHandStamp
	}
	return out, nil
}

func indexOfActor(body []string, actor string) int {
	for i, l := range body {
		if strings.Contains(l, "by: "+actor+",") || strings.Contains(l, "by: "+actor+" ") ||
			strings.Contains(l, `by: "`+actor+`"`) || strings.HasSuffix(strings.TrimSpace(l), "by: "+actor) {
			return i
		}
	}
	return -1
}

func frontmatterBounds(lines []string) (int, int, bool) {
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return 0, 0, false
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return 1, i, true
		}
	}
	return 0, 0, false
}

// blockBounds returns the half-open line range of a top-level key, which runs
// until the next top-level key: that is what makes a multi-line `sources:` or a
// list-form `verified:` one unit to replace.
func blockBounds(lines []string, from, to int, key string) (int, int) {
	start := -1
	for i := from; i < to; i++ {
		m := topKeyRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		if start >= 0 {
			return start, i
		}
		if m[1] == key {
			start = i
		}
	}
	if start >= 0 {
		return start, to
	}
	return -1, -1
}

func insertionPoint(lines []string, from, to int) int {
	rank := map[string]int{}
	for i, k := range canonicalOrder {
		rank[k] = i
	}
	want := rank["verified"]
	for i := from; i < to; i++ {
		m := topKeyRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		if r, known := rank[m[1]]; known && r > want {
			return i
		}
	}
	return to
}

func insert(lines []string, at int, add ...string) []string {
	out := make([]string, 0, len(lines)+len(add))
	out = append(out, lines[:at]...)
	out = append(out, add...)
	return append(out, lines[at:]...)
}

func splice(lines []string, start, end int, repl []string) []string {
	out := make([]string, 0, len(lines)-(end-start)+len(repl))
	out = append(out, lines[:start]...)
	out = append(out, repl...)
	return append(out, lines[end:]...)
}

func join(lines []string) string { return strings.Join(lines, "\n") }

var (
	itemStartRe = regexp.MustCompile(`^(\s*)-\s+(.*)$`)
	fieldRe     = regexp.MustCompile(`^(\s*)([A-Za-z_][A-Za-z0-9_]*):\s*(.*)$`)
)

// SetSourceDigest records what a URL source hashed to, on the entry naming it.
// §4.1 lets a producer add keys and forbids a consumer rejecting them, so the
// digest lives with the other credibility signals rather than in a sidecar that
// would split provenance from the concept.
func SetSourceDigest(path, resource, digest string) (bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	out, changed := setSourceDigest(string(raw), resource, digest)
	if !changed {
		return false, nil
	}
	return true, os.WriteFile(path, []byte(out), 0o644)
}

func setSourceDigest(text, resource, digest string) (string, bool) {
	lines := strings.Split(text, "\n")
	fmStart, fmEnd, ok := frontmatterBounds(lines)
	if !ok {
		return text, false
	}
	sStart, sEnd := blockBounds(lines, fmStart, fmEnd, "sources")
	if sStart < 0 {
		return text, false
	}
	for _, it := range items(lines, sStart+1, sEnd) {
		if fieldValue(lines, it.start, it.end, "resource") != resource {
			continue
		}
		if d := fieldLine(lines, it.start, it.end, "digest"); d >= 0 {
			if strings.TrimSpace(lines[d]) == "digest: "+digest {
				return text, false
			}
			lines[d] = indentOf(lines[d]) + "digest: " + digest
			return join(lines), true
		}
		return join(insert(lines, it.end, it.indent+"  digest: "+digest)), true
	}
	return text, false
}

type item struct {
	start, end int
	indent     string
}

func items(lines []string, from, to int) []item {
	var out []item
	for i := from; i < to; i++ {
		m := itemStartRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		if n := len(out); n > 0 {
			out[n-1].end = i
		}
		out = append(out, item{start: i, end: to, indent: m[1]})
	}
	return out
}

// fieldValue reads a key from a list item, including one written inline on the
// `- ` line itself, which is where the fleet puts `resource` about half the time.
func fieldValue(lines []string, start, end int, key string) string {
	if m := itemStartRe.FindStringSubmatch(lines[start]); m != nil {
		if f := fieldRe.FindStringSubmatch(m[2]); f != nil && f[2] == key {
			return strings.TrimSpace(f[3])
		}
	}
	if i := fieldLine(lines, start, end, key); i >= 0 {
		return strings.TrimSpace(fieldRe.FindStringSubmatch(lines[i])[3])
	}
	return ""
}

func fieldLine(lines []string, start, end int, key string) int {
	for i := start + 1; i < end; i++ {
		if itemStartRe.MatchString(lines[i]) {
			break
		}
		if m := fieldRe.FindStringSubmatch(lines[i]); m != nil && m[2] == key {
			return i
		}
	}
	return -1
}

func indentOf(s string) string { return s[:len(s)-len(strings.TrimLeft(s, " \t"))] }
