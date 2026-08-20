package okfrules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fairyhunter13/okf"
)

var refDate = time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)

// write lays out a repo with a bundle in it, keyed repo-relative so a fixture
// can name a file outside the bundle — which is what resource: usually does.
func write(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := t.TempDir()
	for rel, text := range files {
		p := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return repo
}

func check(t *testing.T, repo string, rules okf.Rules) []string {
	t.Helper()
	findings, err := okf.CheckBundleWith(filepath.Join(repo, "knowledge"), refDate, rules)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, f := range findings {
		if f.Sev == okf.Error {
			out = append(out, f.String())
		}
	}
	return out
}

func onlyDoc(r okf.Rule) okf.Rules          { return okf.Rules{Doc: []okf.Rule{r}} }
func onlyBundle(r okf.BundleRule) okf.Rules { return okf.Rules{Bundle: []okf.BundleRule{r}} }

func want(t *testing.T, got []string, substrings ...string) {
	t.Helper()
	if len(got) != len(substrings) {
		t.Fatalf("got %d error(s) %q, want %d", len(got), got, len(substrings))
	}
	for i, s := range substrings {
		if !strings.Contains(got[i], s) {
			t.Errorf("error %d = %q, want it to name %q", i, got[i], s)
		}
	}
}

const okFM = "---\ntype: Decision\n---\n\nbody\n"

func TestResourceResolves(t *testing.T) {
	repo := write(t, map[string]string{
		"knowledge/index.md":             "---\nokf_version: \"0.2\"\n---\n\n* [a](decisions/a.md)\n* [b](decisions/b.md)\n* [c](decisions/c.md)\n* [s](decisions/sibling.md)\n",
		"scripts/live.py":                "x",
		"knowledge/decisions/sibling.md": okFM,
		"knowledge/decisions/a.md":       "---\ntype: Decision\nresource: scripts/live.py, knowledge/decisions/sibling.md\n---\n\nbody\n",
		"knowledge/decisions/b.md":       "---\ntype: Decision\nresource: https://example.invalid/spec\n---\n\nbody\n",
		"knowledge/decisions/c.md":       "---\ntype: Decision\nresource: scripts/live.py, scripts/gone.py\n---\n\nbody\n",
	})

	want(t, check(t, repo, onlyDoc(ResourceResolves)), "resource does not exist: scripts/gone.py")
}

func TestTypeVocabulary(t *testing.T) {
	repo := write(t, map[string]string{
		"knowledge/index.md":       "---\nokf_version: \"0.2\"\n---\n\n* [a](decisions/a.md)\n* [b](decisions/b.md)\n",
		"knowledge/decisions/a.md": okFM,
		"knowledge/decisions/b.md": "---\ntype: Constraints\n---\n\nbody\n",
	})

	want(t, check(t, repo, onlyDoc(TypeVocabulary(DefaultTypes))), `type "Constraints" is outside`)
}

func TestVerifiedWellFormed(t *testing.T) {
	repo := write(t, map[string]string{
		"knowledge/index.md":       "---\nokf_version: \"0.2\"\n---\n\n* [a](decisions/a.md)\n* [b](decisions/b.md)\n* [c](decisions/c.md)\n* [d](decisions/d.md)\n",
		"knowledge/decisions/a.md": "---\ntype: Decision\ngenerated: {by: claude-opus-5, at: 2026-08-01T00:00:00Z}\nverified: {by: \"human:fairyhunter13\", at: 2026-08-02T00:00:00Z}\n---\n\nbody\n",
		"knowledge/decisions/b.md": "---\ntype: Decision\nverified: true\n---\n\nbody\n",
		"knowledge/decisions/c.md": "---\ntype: Decision\nverified: {by: claude-opus-5, at: 2026-08-02T00:00:00Z}\n---\n\nbody\n",
		"knowledge/decisions/d.md": "---\ntype: Decision\ngenerated: {by: claude-opus-5, at: 2026-08-05T00:00:00Z}\nverified: {by: \"human:fairyhunter13\", at: 2026-08-02T00:00:00Z}\n---\n\nbody\n",
	})

	want(t, check(t, repo, onlyDoc(VerifiedWellFormed)),
		"verified must be a mapping",
		"verified.by must name a human",
		"verified.at precedes generated.at")
}

func TestStaleAfterHasAReason(t *testing.T) {
	repo := write(t, map[string]string{
		"knowledge/index.md":       "---\nokf_version: \"0.2\"\n---\n\n* [a](decisions/a.md)\n* [b](decisions/b.md)\n",
		"knowledge/decisions/a.md": "---\ntype: Decision\nstale_after: 2027-01-01\nsources: [\"vendor EOL notice\"]\n---\n\nbody\n",
		"knowledge/decisions/b.md": "---\ntype: Decision\nstale_after: 2027-01-01\n---\n\nbody\n",
	})

	want(t, check(t, repo, onlyDoc(StaleAfterHasAReason)), "stale_after needs sources")
}

// The log.md arm is the one that proves the v0.2 BundleRule surface: a
// per-concept rule never sees the reserved files, so this wikilink was
// unreachable before it.
func TestNoIntraBundleWikilinks(t *testing.T) {
	repo := write(t, map[string]string{
		"knowledge/index.md":       "---\nokf_version: \"0.2\"\n---\n\n* [a](decisions/a.md)\n",
		"knowledge/decisions/a.md": "---\ntype: Decision\n---\n\nsee [[a-sibling]] and [[memory:a-note]] and `[[quoted]]`\n",
		"knowledge/log.md":         "---\ntype: Log\ntitle: fixture knowledge history\n---\n\n## 2026-08-20\n\n- **Creation** of [[a-sibling]]\n",
	})

	want(t, check(t, repo, onlyBundle(NoIntraBundleWikilinks)),
		"wikilink [[a-sibling]]",
		"wikilink [[a-sibling]]")
	got := check(t, repo, onlyBundle(NoIntraBundleWikilinks))
	if !strings.HasPrefix(got[0], "decisions/a.md") || !strings.HasPrefix(got[1], "log.md") {
		t.Errorf("findings = %q, want one in the concept and one in log.md", got)
	}
}

func TestIndexHeadingsAreSingularTypes(t *testing.T) {
	repo := write(t, map[string]string{
		"knowledge/index.md":       "---\nokf_version: \"0.2\"\n---\n\n# Fixture\n\n## Decisions\n\n* [a](decisions/a.md)\n\n## Policy\n",
		"knowledge/decisions/a.md": okFM,
	})

	want(t, check(t, repo, onlyBundle(IndexHeadingsAreSingularTypes(DefaultTypes))), `plural type heading "Decisions"`)
}

func TestLogFrontmatterAndVerbs(t *testing.T) {
	repo := write(t, map[string]string{
		"knowledge/index.md":       "---\nokf_version: \"0.2\"\n---\n\n* [a](decisions/a.md)\n",
		"knowledge/decisions/a.md": okFM,
		"knowledge/log.md":         "---\ntype: Journal\n---\n\n## 2026-08-20\n\n- **Creation** of a\n- **Tidied** b\n- **Tidied** c\n",
	})

	// Bundle findings sort by path then message, so the order here is the
	// printed one, not the order the rules ran in.
	want(t, check(t, repo, okf.Rules{Bundle: []okf.BundleRule{LogFrontmatter, LogVerbs(DefaultLogVerbs)}}),
		`log verb "Tidied"`,
		`needs `+"`type: Log`",
		`title should end`)
}

func TestStandardIsQuietOnAConformantBundle(t *testing.T) {
	repo := write(t, map[string]string{
		"knowledge/index.md":       "---\nokf_version: \"0.2\"\n---\n\n## Decision\n\n* [a](decisions/a.md)\n",
		"knowledge/decisions/a.md": "---\ntype: Decision\nresource: knowledge/index.md\n---\n\nsee [[memory:a-note]] and [the index](../index.md)\n",
		"knowledge/log.md":         "---\ntype: Log\ntitle: fixture knowledge history\n---\n\n## 2026-08-20\n\n- **Creation** of a\n",
	})

	if got := check(t, repo, Strict()); len(got) != 0 {
		t.Errorf("a conformant bundle reported %q", got)
	}
}

// Strict is Standard plus the two conversion-gated rules, and a repo that has
// not converted yet must still be able to take Standard.
func TestStandardLeavesTheConversionGatedRulesOut(t *testing.T) {
	repo := write(t, map[string]string{
		"knowledge/index.md":       "---\nokf_version: \"0.2\"\n---\n\n## Decision\n\n* [a](decisions/a.md)\n",
		"knowledge/decisions/a.md": "---\ntype: Decision\n---\n\nsee [[a-sibling]]\n",
		"knowledge/log.md":         "---\ntype: Log\ntitle: fixture knowledge history\n---\n\n## 2026-08-20\n\n- **Tidied** a\n",
	})

	if got := check(t, repo, Standard()); len(got) != 0 {
		t.Errorf("Standard reported %q on an unconverted bundle", got)
	}
	if got := check(t, repo, Strict()); len(got) != 2 {
		t.Errorf("Strict reported %q, want the wikilink and the verb", got)
	}
}
