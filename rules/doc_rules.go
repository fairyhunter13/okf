package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/fairyhunter13/okf"
)

// ResourceResolves refuses a concept whose `resource:` names a path that is
// gone. A concept describing something that no longer exists is the one kind of
// rot a checker can prove, and the fix is always cheaper than the bypass:
// repoint it, mark the concept deprecated, or drop the key, which is optional.
func ResourceResolves(d okf.Doc) []okf.Finding {
	raw, ok := d.FM["resource"]
	if !ok {
		return nil
	}
	var out []okf.Finding
	for _, ref := range resourceRefs(raw) {
		if strings.Contains(ref, "://") {
			continue
		}
		if !anyExists(d, ref) {
			out = append(out, okf.Finding{Sev: okf.Error, Msg: fmt.Sprintf("resource does not exist: %s", ref)})
		}
	}
	return out
}

// A resource is one path, a comma-separated list of them, or a YAML list.
func resourceRefs(raw any) []string {
	var out []string
	add := func(s string) {
		for _, part := range strings.Split(s, ",") {
			if part = strings.TrimSpace(part); part != "" {
				out = append(out, part)
			}
		}
	}
	switch v := raw.(type) {
	case string:
		add(v)
	case []any:
		for _, e := range v {
			if s, ok := e.(string); ok {
				add(s)
			}
		}
	}
	return out
}

// Bundles disagree on what a resource path is relative to — the repo, the
// bundle, or the concept — and all three readings are in use, so all three
// count. The rule is about existence, not about spelling.
func anyExists(d okf.Doc, ref string) bool {
	for _, base := range []string{filepath.Dir(d.Root), d.Root, filepath.Dir(filepath.Join(d.Root, filepath.FromSlash(d.Rel)))} {
		if _, err := os.Stat(filepath.Join(base, filepath.FromSlash(ref))); err == nil {
			return true
		}
	}
	return false
}

// TypeVocabulary refuses a `type` outside the fleet's table. `type: Constraints`
// is the drift this catches: a group nothing joins up across bundles.
func TypeVocabulary(types []string) okf.Rule {
	known := map[string]bool{}
	for _, t := range types {
		known[t] = true
	}
	return func(d okf.Doc) []okf.Finding {
		t, ok := str(d.FM["type"])
		if !ok || t == "" || known[t] {
			return nil
		}
		return errf(fmt.Sprintf("type %q is outside the vocabulary", t))
	}
}

// VerifiedWellFormed holds a stamp to §5.2's shape. Demanding a `human:` actor
// made §5.3's machine-confirmed tier unreachable — the spec's own example stamps
// `process:finance-nightly` — so actor shape is [ActorConvention]'s job now, and
// forgery is [Stamp]'s: it refuses `human:` under CLAUDECODE, which is a fact
// about who is running that no document rule can see.
func VerifiedWellFormed(d okf.Doc) []okf.Finding {
	raw, ok := d.FM["verified"]
	if !ok {
		return nil
	}
	events, err := verifiedEvents(raw)
	if err != nil {
		return errf(err.Error())
	}

	var written time.Time
	if gen, ok := d.FM["generated"].(map[string]any); ok {
		if w, err := parseWhen(gen["at"]); err == nil {
			written = w
		}
	}

	var out []okf.Finding
	for _, m := range events {
		stamped, err := parseWhen(m["at"])
		if err != nil {
			out = append(out, okf.Finding{Sev: okf.Error, Msg: fmt.Sprintf("verified.at is not a timestamp: %v", m["at"])})
			continue
		}
		if !written.IsZero() && stamped.Before(written) {
			out = append(out, okf.Finding{Sev: okf.Error, Msg: "verified.at precedes generated.at: the stamp is older than what it confirms"})
		}
	}
	return out
}

// verifiedEvents normalizes §5.2's two spellings into the one the rule reads.
// An entry that is not a mapping is reported rather than skipped: the reference
// parser drops it silently, which turns a malformed stamp into no stamp at all.
func verifiedEvents(raw any) ([]map[string]any, error) {
	switch v := raw.(type) {
	case map[string]any:
		return []map[string]any{v}, nil
	case []any:
		out := make([]map[string]any, 0, len(v))
		for _, e := range v {
			m, ok := e.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("verified entry must be a mapping with `by` and `at`: %v", e)
			}
			out = append(out, m)
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("verified is empty: drop the key rather than claim an unnamed review")
		}
		return out, nil
	}
	return nil, fmt.Errorf("verified must be a mapping with `by` and `at`, or a list of them")
}

func parseWhen(raw any) (time.Time, error) {
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

// StaleAfterHasAReason keeps the date for genuine shelf life — a vendor EOL, a
// certificate — by requiring the external thing that expires to be named. A
// date set because a concept felt old is a re-verification treadmill nobody
// asked for, and two of 242 concepts carry one at all.
func StaleAfterHasAReason(d okf.Doc) []okf.Finding {
	if _, ok := d.FM["stale_after"]; !ok {
		return nil
	}
	switch v := d.FM["sources"].(type) {
	case []any:
		if len(v) > 0 {
			return nil
		}
	case string:
		if strings.TrimSpace(v) != "" {
			return nil
		}
	}
	return errf("stale_after needs sources: naming the external thing that expires")
}

var (
	// §7's four actor forms. The producer form is deliberately loose on the
	// version half — `opus-5` and `4.1.2` are both versions — and strict on there
	// being exactly one slash, which is what `claude-opus-5` lacks.
	actorRe = regexp.MustCompile(`^(?:(?:human|process|team):\S+|[^/\s:]+/[^/\s]+)$`)
	// A footnote definition, which is the half §5.1 binds. The inline reference
	// is not checked: a bundle may cite a source it defines nowhere yet.
	footnoteDefRe = regexp.MustCompile(`(?m)^\[\^([^\]]+)\]:`)
	anyHeadingRe  = regexp.MustCompile(`(?m)^#+ `)
)

// ActorConvention holds `generated.by` and every `verified[].by` to §7's four
// forms. An actor that is neither a producer with a version nor a prefixed
// identity cannot be told apart from a free-text note, which is what 146 fleet
// concepts spelling a model as `claude-opus-5` had become.
func ActorConvention(d okf.Doc) []okf.Finding {
	var out []okf.Finding
	check := func(key, by string) {
		if by != "" && !actorRe.MatchString(by) {
			out = append(out, okf.Finding{Sev: okf.Error, Msg: fmt.Sprintf("%s is outside §7's actor forms — `<producer>/<version>`, `human:`, `process:`, `team:`: %q", key, by)})
		}
	}
	if gen, ok := d.FM["generated"].(map[string]any); ok {
		by, _ := str(gen["by"])
		check("generated.by", by)
	}
	if raw, ok := d.FM["verified"]; ok {
		if events, err := verifiedEvents(raw); err == nil {
			for _, m := range events {
				by, _ := str(m["by"])
				check("verified.by", by)
			}
		}
	}
	// §5.1 binds `author` to §7 too. Unchecked, one bundle had spelled the same
	// publisher `org:anthropic` and `team:anthropic` on adjacent concepts.
	if list, ok := d.FM["sources"].([]any); ok {
		for _, e := range list {
			m, ok := e.(map[string]any)
			if !ok {
				continue
			}
			by, _ := str(m["author"])
			check("sources[].author", by)
		}
	}
	return out
}

// StatusVocabulary holds `status` to §5.4's three values. Absence is not a
// finding: the spec makes an absent status mean `stable`, so adding one says
// nothing the reader did not already know.
func StatusVocabulary(d okf.Doc) []okf.Finding {
	s, ok := str(d.FM["status"])
	if !ok || s == "" {
		return nil
	}
	switch s {
	case "draft", "stable", "deprecated":
		return nil
	}
	return errf(fmt.Sprintf("status %q is outside §5.4's `draft | stable | deprecated`", s))
}

// FootnoteLabelsJoinSources enforces §5.1: a footnote label MUST be a
// `sources[].id`, explicitly not a position. A numeric `[^1]` reads fine and
// joins nothing — renumber the list and every citation silently retargets.
//
// Fleet-scoped on purpose. The reference corpus fails it: ten stackoverflow
// files cite `[^1]` against ids like `meta_schema_doc`, and nine define a label
// they never reference.
func FootnoteLabelsJoinSources(d okf.Doc) []okf.Finding {
	defs := footnoteDefRe.FindAllStringSubmatch(stripCode(d.Body), -1)
	if len(defs) == 0 {
		return nil
	}
	ids := map[string]bool{}
	if list, ok := d.FM["sources"].([]any); ok {
		for _, e := range list {
			if m, ok := e.(map[string]any); ok {
				if id, _ := str(m["id"]); id != "" {
					ids[id] = true
				}
			}
		}
	}
	var out []okf.Finding
	for _, m := range defs {
		if label := strings.TrimSpace(m[1]); !ids[label] {
			out = append(out, okf.Finding{Sev: okf.Error, Msg: fmt.Sprintf("footnote [^%s] names no sources[].id: §5.1 keys citations, it does not number them", label)})
		}
	}
	return out
}

// AttestedComputationHasContract keeps §10's promise checkable. `runtime` is
// REQUIRED (§10.2), and the computation itself must be somewhere a consumer can
// read it: either a `computation:` path that resolves or a fenced `# Computation`
// section (§10.3). Both, and a reader has two sources of truth to reconcile.
func AttestedComputationHasContract(d okf.Doc) []okf.Finding {
	if t, _ := str(d.FM["type"]); t != "Attested Computation" {
		return nil
	}
	var out []okf.Finding
	if rt, _ := str(d.FM["runtime"]); rt == "" {
		out = append(out, okf.Finding{Sev: okf.Error, Msg: "Attested Computation needs `runtime`: §10.2 makes it required"})
	}

	path, _ := str(d.FM["computation"])
	hasPath := path != ""
	if hasPath && !anyExists(d, path) {
		out = append(out, okf.Finding{Sev: okf.Error, Msg: fmt.Sprintf("computation must be a path, and this one does not resolve: %s", path)})
	}
	hasFence := strings.Contains(computationSection(d.Body), "```")
	if hasPath == hasFence {
		out = append(out, okf.Finding{Sev: okf.Error, Msg: "Attested Computation needs exactly one of `computation:` or a fenced `# Computation` section"})
	}
	return out
}

// computationSection returns the body of the `# Computation` section alone, at
// any heading level. Bounding it at the next heading matters: a fence three
// sections later would otherwise stand in for the one that is missing.
func computationSection(body string) string {
	for _, m := range anyHeadingRe.FindAllStringIndex(body, -1) {
		line, _, _ := strings.Cut(body[m[0]:], "\n")
		if strings.TrimSpace(strings.TrimLeft(line, "# ")) != "Computation" {
			continue
		}
		rest := body[m[1]:]
		if next := anyHeadingRe.FindStringIndex(rest); next != nil {
			return rest[:next[0]]
		}
		return rest
	}
	return ""
}

// SourceHasAResource holds §5.1's one REQUIRED key. An entry written as a bare
// string, or keyed `url:`, reads like provenance and names nothing a consumer
// can follow: four fleet concepts were unverifiable that way, with the checker
// silent about all four.
func SourceHasAResource(d okf.Doc) []okf.Finding {
	list, _ := d.FM["sources"].([]any)
	var out []okf.Finding
	for i, e := range list {
		m, ok := e.(map[string]any)
		if !ok {
			out = append(out, okf.Finding{Sev: okf.Error, Msg: fmt.Sprintf("sources[%d] is not a mapping, so it has no `resource:` (§5.1)", i)})
			continue
		}
		if res, _ := str(m["resource"]); res == "" {
			out = append(out, okf.Finding{Sev: okf.Error, Msg: fmt.Sprintf("sources[%d] has no `resource:`, the one key §5.1 requires within an entry", i)})
		}
	}
	return out
}
