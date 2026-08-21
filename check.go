package okf

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	linkRe    = regexp.MustCompile(`\]\(([^)\s]+)`)
	logDateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	codeRe    = regexp.MustCompile("(?s)```.*?```|`[^`\n]*`")
)

// CheckBundle walks a bundle root and reports conformance errors (§11) and
// advisory warnings, plus whatever the caller's rules report.
func CheckBundle(root string, today time.Time, rules ...Rule) ([]Finding, error) {
	return CheckBundleWith(root, today, Rules{Doc: rules})
}

// CheckBundleWith is [CheckBundle] with bundle-wide rules as well. Their
// findings are appended after the per-concept ones and sorted among themselves,
// so adding one cannot move a line the stock check already prints.
func CheckBundleWith(root string, today time.Time, rules Rules) ([]Finding, error) {
	// A caller that spells the root with a trailing slash gets the same verdict as
	// one that does not: uncleaned, filepath.Dir(Root) is the bundle itself, so
	// every repo-relative `resource:` stops resolving and the bundle reports errors
	// for files that are there.
	root = filepath.Clean(root)
	var out []Finding
	var concepts []string
	bundle := Bundle{Root: root}
	linked := map[string]bool{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(p) != ".md" {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel := filepath.ToSlash(mustRel(root, p))
		out = append(out, checkFile(root, rel, string(b), today, rules.Doc)...)

		fm, body, perr := Parse(string(b))
		if perr != nil {
			return nil
		}
		bundle.Docs = append(bundle.Docs, Doc{Root: root, Rel: rel, FM: fm, Body: body, FMText: frontmatterText(string(b))})
		if !Reserved(rel) {
			concepts = append(concepts, rel)
		}
		for _, l := range bundleLinks(rel, body) {
			linked[l.rel] = true
		}
		return nil
	})
	if err != nil {
		return out, err
	}
	out = append(out, orphanFindings(concepts, linked)...)
	return append(out, bundleFindings(rules.Bundle, bundle)...), nil
}

// Load parses every .md under a root and reports nothing. A consumer that
// renders or indexes a bundle wants the documents without an opinion attached;
// a document whose frontmatter will not parse is skipped, because [CheckBundle]
// is where that is a finding.
func Load(root string) (Bundle, error) {
	root = filepath.Clean(root)
	b := Bundle{Root: root}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(p) != ".md" {
			return err
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		fm, body, perr := Parse(string(raw))
		if perr != nil {
			return nil
		}
		rel := filepath.ToSlash(mustRel(root, p))
		b.Docs = append(b.Docs, Doc{Root: root, Rel: rel, FM: fm, Body: body, FMText: frontmatterText(string(raw))})
		return nil
	})
	return b, err
}

// Reserved reports whether a bundle-relative path is one of OKF's two reserved
// names. A [Rule] never sees these; a [BundleRule] does, and has to say so.
func Reserved(rel string) bool {
	b := path.Base(rel)
	return b == "index.md" || b == "log.md"
}

func checkFile(root, rel, text string, today time.Time, rules []Rule) []Finding {
	add := func(sev Severity, format string, a ...any) Finding {
		return Finding{Path: rel, Sev: sev, Msg: fmt.Sprintf(format, a...)}
	}

	fm, body, err := Parse(text)
	if err != nil {
		return []Finding{add(Error, "%v", err)}
	}

	var out []Finding
	switch path.Base(rel) {
	case "index.md":
		// §8: an index carries no frontmatter, except okf_version at the root.
		for k := range fm {
			if k == "okf_version" && rel == "index.md" {
				continue
			}
			out = append(out, add(Error, "index.md must not carry frontmatter key %q", k))
		}
	case "log.md":
		// §9 states one MUST: date headings are ISO 8601.
		for line := range strings.SplitSeq(body, "\n") {
			if h, ok := strings.CutPrefix(strings.TrimSpace(line), "## "); ok && !logDateRe.MatchString(strings.TrimSpace(h)) {
				out = append(out, add(Error, "log.md date heading %q is not YYYY-MM-DD", strings.TrimSpace(h)))
			}
		}
	default:
		if fm == nil {
			out = append(out, add(Error, "no YAML frontmatter block"))
			return out
		}
		if t, _ := fm["type"].(string); strings.TrimSpace(t) == "" {
			out = append(out, add(Error, "frontmatter has no non-empty `type`"))
		}
		out = append(out, staleFinding(add, fm, today)...)
		out = append(out, ruleFindings(rules, Doc{Root: root, Rel: rel, FM: fm, Body: body, FMText: frontmatterText(text)})...)
	}

	return append(out, linkFindings(add, root, rel, body)...)
}

func ruleFindings(rules []Rule, doc Doc) []Finding {
	var out []Finding
	for _, r := range rules {
		for _, f := range r(doc) {
			if f.Path == "" {
				f.Path = doc.Rel
			}
			out = append(out, f)
		}
	}
	return out
}

func staleFinding(add func(Severity, string, ...any) Finding, fm map[string]any, today time.Time) []Finding {
	raw, ok := fm["stale_after"]
	if !ok {
		return nil
	}
	// Compare instants, not days: truncating to YYYY-MM-DD first read
	// 2026-12-31T23:00:00+07:00 as UTC midnight, a 17-hour error at the only
	// boundary this key has.
	d, err := ParseTimestamp(raw)
	if err != nil {
		return []Finding{add(Warning, "stale_after is not an ISO 8601 timestamp: %v", raw)}
	}
	if !today.Before(d) {
		return []Finding{add(Warning, "stale since %s: re-verify or update stale_after", d.Format(time.RFC3339))}
	}
	return nil
}

// link is one bundle-local markdown link, resolved to a bundle-relative path.
type link struct {
	raw string // as written, so a message can quote it
	rel string // bundle-relative, slash-separated
}

// bundleLinks resolves the bundle-local links in a body. Code spans and fences
// hold example links that name nothing real, so they are stripped first.
func bundleLinks(rel, body string) []link {
	var out []link
	for _, m := range linkRe.FindAllStringSubmatch(codeRe.ReplaceAllString(body, ""), -1) {
		target, _, _ := strings.Cut(m[1], "#")
		if target == "" || strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
			continue
		}
		if t, ok := strings.CutPrefix(target, "/"); ok {
			out = append(out, link{raw: target, rel: path.Clean(t)})
			continue
		}
		out = append(out, link{raw: target, rel: path.Join(path.Dir(rel), target)})
	}
	return out
}

// linkFindings reports dangling links. §6.1 tolerates one — it may mark
// knowledge not yet written — so it is never an error. Whether a leading "/" is
// acceptable is a producer opinion, and lives in rules.PreferRelativeLinks.
func linkFindings(add func(Severity, string, ...any) Finding, root, rel, body string) []Finding {
	var out []Finding
	for _, l := range bundleLinks(rel, body) {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(l.rel))); err != nil {
			out = append(out, add(Warning, "link target not in bundle: %s", l.raw))
		}
	}
	return out
}

// orphanFindings reports concepts nothing points at. A concept no reader can
// reach is knowledge that was written but not filed.
func orphanFindings(concepts []string, linked map[string]bool) []Finding {
	var out []Finding
	for _, c := range concepts {
		if !linked[c] {
			out = append(out, Finding{Path: c, Sev: Warning, Msg: "orphan: no index entry or concept links here"})
		}
	}
	return out
}

func mustRel(root, p string) string {
	r, err := filepath.Rel(root, p)
	if err != nil {
		return p
	}
	return r
}
