package okf

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// The skill's "augment, never shrink" rule, moved out of prose. An agent here
// writes files directly rather than through a tool that could refuse, so the
// only place left to see the previous version is a gate with git behind it.
// Findings are warnings: §11 forbids rejecting a bundle for anything but its
// three rules, and a real deprecation does drop things — which is why
// `status: deprecated` is exempt.

var headingRe = regexp.MustCompile(`(?m)^#{1,6}[ \t]+(.+?)[ \t]*$`)

// CheckAgainst reports what root lost relative to the same bundle at a git ref.
// A concept absent from the ref is new and cannot have shrunk.
func CheckAgainst(root, ref string) ([]Finding, error) {
	b, err := Load(root)
	if err != nil {
		return nil, err
	}
	prefix, err := repoPrefix(root)
	if err != nil {
		return nil, err
	}
	var out []Finding
	for _, d := range b.Concepts() {
		if strings.EqualFold(firstScalar(d.FM["status"]), "deprecated") {
			continue
		}
		old, ok, err := blobAt(root, ref, prefix+d.Rel)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		fm, body, err := Parse(old)
		if err != nil {
			continue // the ref holds something unparseable; not this rule's finding
		}
		for _, msg := range shrinkage(Doc{FM: fm, Body: body}, d) {
			out = append(out, Finding{Path: d.Rel, Sev: Warning, Msg: msg})
		}
	}
	return out, nil
}

func shrinkage(prev, cur Doc) []string {
	var out []string
	for _, k := range []string{"type", "title", "resource"} {
		o, n := firstScalar(prev.FM[k]), firstScalar(cur.FM[k])
		if o != "" && o != n {
			out = append(out, fmt.Sprintf("%s changed from %q to %q: rewriting a concept's identity makes it a different concept", k, o, n))
		}
	}
	for _, h := range missing(headings(prev.Body), headings(cur.Body)) {
		out = append(out, fmt.Sprintf("section %q was dropped or reordered", h))
	}
	for _, s := range missing(keyed(prev.FM["sources"], "resource"), keyed(cur.FM["sources"], "resource")) {
		out = append(out, fmt.Sprintf("source %q was dropped: provenance only accumulates", s))
	}
	for _, v := range missing(keyed(prev.FM["verified"], "by", "at"), keyed(cur.FM["verified"], "by", "at")) {
		out = append(out, fmt.Sprintf("verification event %q was dropped", v))
	}
	return out
}

// missing walks the old entries against the new ones in order. Matching as a
// subsequence is what makes this catch a reorder as well as a deletion while
// still letting an insertion through.
func missing(old, cur []string) []string {
	var out []string
	i := 0
	for _, o := range old {
		j := i
		for j < len(cur) && cur[j] != o {
			j++
		}
		if j == len(cur) {
			out = append(out, o)
			continue
		}
		i = j + 1
	}
	return out
}

func headings(body string) []string {
	var out []string
	for _, m := range headingRe.FindAllStringSubmatch(codeRe.ReplaceAllString(body, ""), -1) {
		out = append(out, strings.TrimSpace(m[1]))
	}
	return out
}

// keyed renders each entry of a §5 list as one comparable string. Only the
// identifying keys go in: a source whose title was edited is not a lost source.
func keyed(raw any, keys ...string) []string {
	list, ok := raw.([]any)
	if !ok {
		m, isMap := raw.(map[string]any)
		if !isMap {
			return nil
		}
		list = []any{m}
	}
	var out []string
	for _, e := range list {
		m, ok := e.(map[string]any)
		if !ok {
			out = append(out, firstScalar(e))
			continue
		}
		var parts []string
		for _, k := range keys {
			parts = append(parts, firstScalar(m[k]))
		}
		out = append(out, strings.TrimSpace(strings.Join(parts, " ")))
	}
	return out
}

func firstScalar(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case time.Time:
		// yaml hands back a time.Time for every ISO form, and %v on one of those
		// is Go's own layout — which is not what any bundle says.
		return t.Format(time.RFC3339)
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

// repoPrefix is root's path from the repository root, which is what a
// ref-relative `git cat-file` path needs.
func repoPrefix(root string) (string, error) {
	out, err := exec.Command("git", "-C", root, "rev-parse", "--show-prefix").Output()
	if err != nil {
		return "", fmt.Errorf("%s is not in a git repository: %w", root, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func blobAt(root, ref, path string) (string, bool, error) {
	out, err := exec.Command("git", "-C", root, "cat-file", "-p", ref+":"+path).Output()
	if err == nil {
		return string(out), true, nil
	}
	if _, ok := err.(*exec.ExitError); ok {
		return "", false, nil // absent at the ref, so it is a new concept
	}
	return "", false, fmt.Errorf("git cat-file: %w", err)
}
