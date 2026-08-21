package viz

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/fairyhunter13/okf"
)

// Trust tiers, derived from §5.3: the tier is a property of the stamp's actor,
// not of the key's presence. A machine tier that `okf verify` can earn is why
// this keys on the `human:` prefix rather than on `verified:` existing at all.
const (
	trustNone    = "unverified"
	trustMachine = "machine-confirmed"
	trustHuman   = "human-reviewed"
)

func newNode(d okf.Doc, now time.Time) node {
	n := node{
		ID:          d.Rel,
		Type:        firstString(d.FM["type"]),
		Title:       firstString(d.FM["title"]),
		Description: firstString(d.FM["description"]),
		Resource:    strings.Join(stringList(d.FM["resource"]), ", "),
		Status:      firstString(d.FM["status"]),
		Tags:        stringList(d.FM["tags"]),
		Trust:       trustNone,
		Body:        Markdown(d.Body),
	}
	if n.Title == "" {
		n.Title = strings.TrimSuffix(path.Base(d.Rel), ".md")
	}
	if g, ok := d.FM["generated"].(map[string]any); ok {
		n.Generated = strings.TrimSpace(firstString(g["by"]) + " " + stamp(g["at"]))
	}
	for _, ev := range verifiedEvents(d.FM["verified"]) {
		by := firstString(ev["by"])
		n.Verified = append(n.Verified, strings.TrimSpace(by+" "+stamp(ev["at"])))
		if strings.HasPrefix(by, "human:") {
			n.Trust = trustHuman
		} else if n.Trust == trustNone {
			n.Trust = trustMachine
		}
	}
	if raw, ok := d.FM["stale_after"]; ok {
		n.StaleAfter = stamp(raw)
		if t, err := okf.ParseTimestamp(raw); err == nil && !now.Before(t) {
			n.Stale = true
		}
	}
	n.Sources = sourceList(d.FM["sources"])
	// A long concept is a big node: the one thing a reader wants to know before
	// clicking is how much sits behind it.
	n.R = 14 + min(26, float64(len(d.Body))/220)
	return n
}

// verifiedEvents normalizes §5.2's two spellings. A malformed entry is dropped
// rather than reported: rules.VerifiedWellFormed is what grades the stamp, and
// a viewer that refuses to render is worse than one that renders less.
func verifiedEvents(raw any) []map[string]any {
	switch v := raw.(type) {
	case map[string]any:
		return []map[string]any{v}
	case []any:
		var out []map[string]any
		for _, e := range v {
			if m, ok := e.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	}
	return nil
}

func sourceList(raw any) []source {
	list, _ := raw.([]any)
	var out []source
	for _, e := range list {
		m, ok := e.(map[string]any)
		if !ok {
			out = append(out, source{Resource: firstString(e)})
			continue
		}
		out = append(out, source{
			Resource:     firstString(m["resource"]),
			Title:        firstString(m["title"]),
			Author:       firstString(m["author"]),
			LastModified: stamp(m["last_modified"]),
		})
	}
	return out
}

// stamp prints a timestamp the way the spec asks for it. YAML hands back a
// time.Time for every ISO form, and %v on one of those is Go's own layout,
// which is not anything a bundle contains.
func stamp(raw any) string {
	if raw == nil {
		return ""
	}
	if t, ok := raw.(time.Time); ok {
		return t.Format(time.RFC3339)
	}
	return firstString(raw)
}

func firstString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

// A key that takes one value or many: `tags` is a list, `resource` is one path
// or a comma-separated set, and both spellings are in the fleet.
func stringList(v any) []string {
	switch t := v.(type) {
	case string:
		var out []string
		for _, p := range strings.Split(t, ",") {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		return out
	case []any:
		var out []string
		for _, e := range t {
			if s := firstString(e); s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}
