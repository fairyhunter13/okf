package okfrules

import (
	"fmt"
	"os"
	"path/filepath"
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

// VerifiedWellFormed refuses a stamp an agent could have written. v0.2 splits
// provenance from trust: `generated:` says who wrote a concept and `verified:`
// who confirmed it, so a stamp naming a model, or naming nobody, is the whole
// key defeated.
func VerifiedWellFormed(d okf.Doc) []okf.Finding {
	raw, ok := d.FM["verified"]
	if !ok {
		return nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return errf("verified must be a mapping with `by` and `at`")
	}
	by, _ := str(m["by"])
	if !strings.HasPrefix(by, "human:") {
		return errf(fmt.Sprintf("verified.by must name a human, as `human:<name>`: %q", by))
	}
	stamped, err := parseWhen(m["at"])
	if err != nil {
		return errf(fmt.Sprintf("verified.at is not a timestamp: %v", m["at"]))
	}
	if gen, ok := d.FM["generated"].(map[string]any); ok {
		if written, err := parseWhen(gen["at"]); err == nil && stamped.Before(written) {
			return errf("verified.at precedes generated.at: the stamp is older than what it confirms")
		}
	}
	return nil
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
