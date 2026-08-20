package okfrules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/fairyhunter13/okf"
)

var (
	wikiRe    = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	headingRe = regexp.MustCompile(`(?m)^#{1,2} (.+)$`)
	logVerbRe = regexp.MustCompile(`(?m)^- \*\*(.+?)\*\*`)
)

// NoIntraBundleWikilinks refuses `[[name]]` inside a bundle. OKF's graph is
// ordinary markdown links, which okf resolves and reports on; a wikilink is
// invisible to it, so a target that stops existing goes quiet instead of
// warning. A reference into another home keeps the syntax with its home named —
// `[[memory:some-note]]` — which is the half no checker running in CI can
// resolve anyway.
func NoIntraBundleWikilinks(b okf.Bundle) []okf.Finding {
	var out []okf.Finding
	for _, d := range b.Docs {
		for _, m := range wikiRe.FindAllStringSubmatch(stripCode(d.Body), -1) {
			name := strings.TrimSpace(m[1])
			if _, _, ok := strings.Cut(name, ":"); ok {
				continue
			}
			out = append(out, at(d.Rel, fmt.Sprintf("wikilink [[%s]]: link inside the bundle with markdown, or name the home it belongs to", name)))
		}
	}
	return out
}

// IndexHeadingsAreSingularTypes keeps the index's groups joinable across
// bundles. The level is free — an index may open with a title and an orienting
// paragraph — but "Decisions" is a group nothing else can join.
func IndexHeadingsAreSingularTypes(types []string) okf.BundleRule {
	known := map[string]bool{}
	for _, t := range types {
		known[t] = true
	}
	return func(b okf.Bundle) []okf.Finding {
		idx, ok := b.Find("index.md")
		if !ok {
			return nil
		}
		var out []okf.Finding
		for _, m := range headingRe.FindAllStringSubmatch(idx.Body, -1) {
			h := strings.TrimSpace(m[1])
			if !known[h] && known[strings.TrimSuffix(h, "s")] {
				out = append(out, at("index.md", fmt.Sprintf("plural type heading %q: the table spells it %q", h, strings.TrimSuffix(h, "s"))))
			}
		}
		return out
	}
}

// LogFrontmatter: `title` is what the bootstrap template prescribes and what the
// two newest bundles dropped. Nothing noticed, because nothing looked.
func LogFrontmatter(b okf.Bundle) []okf.Finding {
	log, ok := b.Find("log.md")
	if !ok {
		return nil
	}
	var out []okf.Finding
	if t, _ := str(log.FM["type"]); t != "Log" {
		out = append(out, at("log.md", fmt.Sprintf("log.md needs `type: Log`, has %q", t)))
	}
	if title, _ := str(log.FM["title"]); !strings.HasSuffix(title, "knowledge history") {
		out = append(out, at("log.md", fmt.Sprintf("log.md title should end \"knowledge history\", has %q", title)))
	}
	return out
}

// LogVerbs holds the log to its four verbs. The log is the bundle's history and
// a free-text verb makes it unreadable as a series.
func LogVerbs(verbs []string) okf.BundleRule {
	known := map[string]bool{}
	for _, v := range verbs {
		known[v] = true
	}
	return func(b okf.Bundle) []okf.Finding {
		log, ok := b.Find("log.md")
		if !ok {
			return nil
		}
		var out []okf.Finding
		seen := map[string]bool{}
		for _, m := range logVerbRe.FindAllStringSubmatch(stripCode(log.Body), -1) {
			v := strings.TrimSpace(m[1])
			if known[v] || seen[v] {
				continue
			}
			seen[v] = true
			out = append(out, at("log.md", fmt.Sprintf("log verb %q is outside the vocabulary", v)))
		}
		return out
	}
}
