package rules

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

// warnings is check's other half: PreferRelativeLinks is the one rule here that
// is deliberately not an Error, so asserting it needs the severity check flipped.
func warnings(t *testing.T, repo string, rules okf.Rules) []string {
	t.Helper()
	findings, err := okf.CheckBundleWith(filepath.Join(repo, "knowledge"), refDate, rules)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, f := range findings {
		if f.Sev == okf.Warning {
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

	// Same bundle, root spelled with a trailing slash. Before okf cleaned the root
	// this reported every repo-relative resource as missing -- 12 of them on one real
	// bundle -- so a gate's path spelling decided whether it was red.
	findings, err := okf.CheckBundleWith(filepath.Join(repo, "knowledge")+string(filepath.Separator), refDate, onlyDoc(ResourceResolves))
	if err != nil {
		t.Fatal(err)
	}
	var errs []string
	for _, f := range findings {
		if f.Sev == okf.Error {
			errs = append(errs, f.String())
		}
	}
	if len(errs) != 1 || !strings.Contains(errs[0], "scripts/gone.py") {
		t.Errorf("trailing-slash root reported %v, want only scripts/gone.py", errs)
	}

	// Same bundle again, spelled `.` from inside it. Cleaning the root does not
	// reach this one -- Dir(".") is "." -- so the repo base is recovered by abs.
	t.Chdir(filepath.Join(repo, "knowledge"))
	findings, err = okf.CheckBundleWith(".", refDate, onlyDoc(ResourceResolves))
	if err != nil {
		t.Fatal(err)
	}
	errs = nil
	for _, f := range findings {
		if f.Sev == okf.Error {
			errs = append(errs, f.String())
		}
	}
	if len(errs) != 1 || !strings.Contains(errs[0], "scripts/gone.py") {
		t.Errorf("dot root reported %v, want only scripts/gone.py", errs)
	}
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
		// §5.3's machine-confirmed tier: a stamp naming no human is legal.
		"knowledge/decisions/c.md": "---\ntype: Decision\nverified: {by: \"process:okf-verify\", at: 2026-08-02T00:00:00Z}\n---\n\nbody\n",
		"knowledge/decisions/d.md": "---\ntype: Decision\ngenerated: {by: claude-opus-5, at: 2026-08-05T00:00:00Z}\nverified: {by: \"human:fairyhunter13\", at: 2026-08-02T00:00:00Z}\n---\n\nbody\n",
	})

	want(t, check(t, repo, onlyDoc(VerifiedWellFormed)),
		"verified must be a mapping",
		"verified.at precedes generated.at")
}

// §5.2 makes the list the primary form; the bare mapping is its shorthand. The
// rule read only the mapping until 2026-08-21, so every reference bundle — all
// of which write the list — came back with one error per stamp.
func TestVerifiedListIsTheSpecPrimaryForm(t *testing.T) {
	repo := write(t, map[string]string{
		"knowledge/index.md": "---\nokf_version: \"0.2\"\n---\n\n* [a](decisions/a.md)\n* [b](decisions/b.md)\n* [c](decisions/c.md)\n* [d](decisions/d.md)\n",
		// One event, list form: the shape every Google bundle writes.
		"knowledge/decisions/a.md": "---\ntype: Decision\nverified:\n  - {by: \"human:jsmith\", at: 2026-07-01T09:00:00Z}\n---\n\nbody\n",
		// §5.2's own pairing: a human sign-off and a nightly process on one concept.
		"knowledge/decisions/b.md": "---\ntype: Decision\nverified:\n  - {by: \"human:jsmith\", at: 2026-07-01T09:00:00Z}\n  - {by: \"process:finance-nightly\", at: 2026-07-02T09:00:00Z}\n---\n\nbody\n",
		// Every entry is checked, not just the first.
		"knowledge/decisions/c.md": "---\ntype: Decision\nverified:\n  - {by: \"human:jsmith\", at: 2026-07-01T09:00:00Z}\n  - {by: \"process:okf-verify\", at: not-a-time}\n---\n\nbody\n",
		// A non-mapping entry is named, not silently dropped.
		"knowledge/decisions/d.md": "---\ntype: Decision\nverified:\n  - yes\n---\n\nbody\n",
	})

	want(t, check(t, repo, onlyDoc(VerifiedWellFormed)),
		"verified.at is not a timestamp",
		"verified entry must be a mapping")
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

// A dated sub-heading is a bolded list item too, and reading it as a verb is a
// finding no rename can answer. The second entry is the other direction: a real
// drift verb in the same log still has to red, or the skip is a hole.
func TestADatedSubHeadingIsNotALogVerb(t *testing.T) {
	repo := write(t, map[string]string{
		"knowledge/index.md":       "---\nokf_version: \"0.2\"\n---\n\n* [a](decisions/a.md)\n",
		"knowledge/decisions/a.md": okFM,
		"knowledge/log.md":         "---\ntype: Log\ntitle: fixture knowledge history\n---\n\n## 2026-08-20\n\n- **2026-08-20 — auditing the lean**\n- **Creation** of a\n- **Tidied** b\n",
	})

	want(t, check(t, repo, onlyBundle(LogVerbs(DefaultLogVerbs))), `log verb "Tidied"`)
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

// The wikilink is Standard's since v0.2.1 and the verb is still Strict's, so a
// bundle carrying one of each separates the two sets.
func TestStandardTakesTheWikilinkAndLeavesTheVerb(t *testing.T) {
	repo := write(t, map[string]string{
		"knowledge/index.md":       "---\nokf_version: \"0.2\"\n---\n\n## Decision\n\n* [a](decisions/a.md)\n",
		"knowledge/decisions/a.md": "---\ntype: Decision\n---\n\nsee [[a-sibling]]\n",
		"knowledge/log.md":         "---\ntype: Log\ntitle: fixture knowledge history\n---\n\n## 2026-08-20\n\n- **Tidied** a\n",
	})

	got := check(t, repo, Standard())
	if len(got) != 1 || !strings.Contains(got[0], "wikilink") {
		t.Errorf("Standard reported %q, want the wikilink alone", got)
	}
	if got := check(t, repo, Strict()); len(got) != 2 {
		t.Errorf("Strict reported %q, want the wikilink and the verb", got)
	}
}

// The engine resolves a "/"-rooted link against the bundle root and finds the
// file, so this fires on links nothing else reports. It moved out of the engine
// in v0.4.0: which link shape a bundle prefers is a producer opinion, and §11
// forbids the conformant half from holding one.
func TestPreferRelativeLinks(t *testing.T) {
	repo := write(t, map[string]string{
		"knowledge/index.md": "---\nokf_version: \"0.2\"\n---\n\n* [a](decisions/a.md)\n* [b](decisions/b.md)\n",
		// Resolves to a real file, so no dangling-link warning masks the result.
		"knowledge/decisions/a.md": "---\ntype: Decision\n---\n\nsee [b](/decisions/b.md) and [rel](b.md) and [ext](https://example.invalid/x)\n",
		// A fenced example of the shape is documentation, not a violation.
		"knowledge/decisions/b.md": "---\ntype: Decision\n---\n\n```\n[bad](/decisions/a.md)\n```\n",
	})

	want(t, warnings(t, repo, onlyBundle(PreferRelativeLinks)),
		"bundle-absolute link /decisions/b.md")
}

// The relative-link rule is Google's, not the spec's: every bundle their agent
// produced obeys it, because their authoring prompt mandates it. Only the
// hand-authored acme_retail uses the absolute form the spec recommends — the
// split is the evidence, so it is asserted in both directions. It lives here
// rather than beside the corpus because package okf can no longer see the rule.
func TestGoogleAgentBundlesUseRelativeLinks(t *testing.T) {
	absLinks := func(name string) int {
		findings, err := okf.CheckBundleWith(filepath.Join("..", "testdata", "google", name), refDate, onlyBundle(PreferRelativeLinks))
		if err != nil {
			t.Fatal(err)
		}
		var n int
		for _, f := range findings {
			if strings.Contains(f.Msg, "bundle-absolute") {
				n++
			}
		}
		return n
	}
	for _, b := range []string{"ga4", "stackoverflow", "crypto_bitcoin"} {
		if n := absLinks(b); n != 0 {
			t.Errorf("%s: %d absolute links, want 0", b, n)
		}
	}
	if n := absLinks("acme_retail"); n == 0 {
		t.Error("acme_retail no longer uses absolute links: re-read the spec's §6.1 recommendation")
	}
}

// The reference corpus is the one thing that can tell a spec rule apart from a
// fleet preference. The spec-derived half must pass it clean — before v0.4.0 it
// did not, and the eight verified stamps it rejected were all legal under §5.2.
// TypeVocabulary, StaleAfterHasAReason and LogFrontmatter are excluded because
// each is avowedly local, and each fails here: unknown types, undocumented
// shelf life, and a log title Google spells its own way.
func TestSpecDerivedRulesPassTheReferenceCorpus(t *testing.T) {
	set := okf.Rules{
		Doc:    []okf.Rule{ResourceResolves, VerifiedWellFormed},
		Bundle: []okf.BundleRule{IndexHeadingsAreSingularTypes(DefaultTypes), NoIntraBundleWikilinks, PreferRelativeLinks},
	}

	for _, b := range []string{"acme_retail", "crypto_bitcoin", "ga4", "stackoverflow"} {
		findings, err := okf.CheckBundleWith(filepath.Join("..", "testdata", "google", b), refDate, set)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range findings {
			if f.Sev == okf.Error {
				t.Errorf("%s: %s", b, f)
			}
		}
	}
}

func TestActorConvention(t *testing.T) {
	repo := write(t, map[string]string{
		"knowledge/index.md": "---\nokf_version: \"0.2\"\n---\n\n* [a](decisions/a.md)\n* [b](decisions/b.md)\n* [c](decisions/c.md)\n* [d](decisions/d.md)\n* [e](decisions/e.md)\n",
		// All four §7 forms, each of which must pass.
		"knowledge/decisions/a.md": "---\ntype: Decision\ngenerated: {by: claude/opus-5, at: 2026-07-01T09:00:00Z}\n---\n\nbody\n",
		"knowledge/decisions/b.md": "---\ntype: Decision\ngenerated: {by: \"team:data-platform\", at: 2026-07-01T09:00:00Z}\n---\n\nbody\n",
		// A model name with no version half reads as free text, which is the drift.
		"knowledge/decisions/c.md": "---\ntype: Decision\ngenerated: {by: claude-opus-5, at: 2026-07-01T09:00:00Z}\n---\n\nbody\n",
		// The stamp is checked on the same terms as the byline.
		"knowledge/decisions/d.md": "---\ntype: Decision\nverified:\n  - {by: jsmith, at: 2026-07-01T09:00:00Z}\n---\n\nbody\n",
		// §5.1 binds a source's author to §7 on the same terms.
		"knowledge/decisions/e.md": "---\ntype: Decision\nsources:\n  - {resource: \"https://example.invalid/x\", author: \"org:anthropic\"}\n---\n\nbody\n",
	})

	want(t, check(t, repo, onlyDoc(ActorConvention)),
		`generated.by is outside §7's actor forms`,
		`verified.by is outside §7's actor forms`,
		`sources[].author is outside §7's actor forms`)
}

func TestStatusVocabulary(t *testing.T) {
	repo := write(t, map[string]string{
		"knowledge/index.md": "---\nokf_version: \"0.2\"\n---\n\n* [a](decisions/a.md)\n* [b](decisions/b.md)\n* [c](decisions/c.md)\n",
		// Absent means stable (§5.4), so this is not a finding.
		"knowledge/decisions/a.md": okFM,
		"knowledge/decisions/b.md": "---\ntype: Decision\nstatus: deprecated\n---\n\nbody\n",
		"knowledge/decisions/c.md": "---\ntype: Decision\nstatus: resolved\n---\n\nbody\n",
	})

	want(t, check(t, repo, onlyDoc(StatusVocabulary)), `status "resolved" is outside §5.4`)
}

func TestFootnoteLabelsJoinSources(t *testing.T) {
	repo := write(t, map[string]string{
		"knowledge/index.md":       "---\nokf_version: \"0.2\"\n---\n\n* [a](decisions/a.md)\n* [b](decisions/b.md)\n* [c](decisions/c.md)\n",
		"knowledge/decisions/a.md": "---\ntype: Decision\nsources:\n  - {id: rfc9110, uri: \"https://example.invalid/rfc9110\"}\n---\n\nclaim[^rfc9110]\n\n[^rfc9110]: RFC 9110\n",
		// A position, not a key: renumber the list and the citation retargets.
		"knowledge/decisions/b.md": "---\ntype: Decision\nsources:\n  - {id: rfc9110, uri: \"https://example.invalid/rfc9110\"}\n---\n\nclaim[^1]\n\n[^1]: RFC 9110\n",
		// A fenced example of the shape is documentation, not a violation.
		"knowledge/decisions/c.md": "---\ntype: Decision\n---\n\n```\n[^1]: example\n```\n",
	})

	want(t, check(t, repo, onlyDoc(FootnoteLabelsJoinSources)), "footnote [^1] names no sources[].id")
}

func TestSourceHasAResource(t *testing.T) {
	repo := write(t, map[string]string{
		"knowledge/index.md": "---\nokf_version: \"0.2\"\n---\n\n* [a](decisions/a.md)\n* [b](decisions/b.md)\n* [c](decisions/c.md)\n",
		// A scope descriptor is a resource: §5.1 admits one a consumer cannot follow.
		"knowledge/decisions/a.md": "---\ntype: Decision\nsources:\n  - {resource: all queries in project X}\n---\n\nbody\n",
		"knowledge/decisions/b.md": "---\ntype: Decision\nsources:\n  - internal/x_test.go\n---\n\nbody\n",
		"knowledge/decisions/c.md": "---\ntype: Decision\nsources:\n  - {id: crbug, url: \"https://example.invalid/1\"}\n---\n\nbody\n",
	})

	want(t, check(t, repo, onlyDoc(SourceHasAResource)),
		"sources[0] is not a mapping", "sources[0] has no `resource:`")
}

func TestTimestampsCarryAnOffset(t *testing.T) {
	repo := write(t, map[string]string{
		"knowledge/index.md": "---\nokf_version: \"0.2\"\n---\n\n* [a](decisions/a.md)\n* [b](decisions/b.md)\n* [c](decisions/c.md)\n* [d](decisions/d.md)\n",
		// Every timestamp-valued key, spelled the way §5 asks.
		"knowledge/decisions/a.md": "---\ntype: Decision\ngenerated: {by: x/1, at: 2026-08-20T00:00:00Z}\nverified:\n  - {by: \"human:h\", at: 2026-08-20T09:00:00+07:00}\nstale_after: 2026-12-31T00:00:00Z\nusage_window: {from: 2026-06-01T00:00:00Z, to: 2026-08-01T00:00:00Z}\n---\n\nbody\n",
		// The pre-2026-08-20 spelling, on the two keys that carried it in the fleet.
		"knowledge/decisions/b.md": "---\ntype: Decision\ngenerated: {by: x/1, at: 2026-08-20}\nstale_after: 2026-12-31\n---\n\nbody\n",
		// `from:` and `to:` are ordinary words. A rule that fired on these would
		// cost more than the drift it catches.
		"knowledge/decisions/c.md": "---\ntype: Decision\nroute: {from: warehouse, to: store}\n---\n\nbody\n",
		// Quoting does not make a date an instant, and neither does a space.
		"knowledge/decisions/d.md": "---\ntype: Decision\nstale_after: \"2026-12-31\"\nusage_window: {from: 2026-06-01 00:00:00, to: 2026-08-01T00:00:00Z}\n---\n\nbody\n",
	})

	want(t, check(t, repo, onlyDoc(TimestampsCarryAnOffset)),
		"at: \"2026-08-20\"", "stale_after: \"2026-12-31\"",
		"stale_after: \"2026-12-31\"", "from: \"2026-06-01 00:00:00\"")
}

func TestAttestedComputationHasContract(t *testing.T) {
	repo := write(t, map[string]string{
		"knowledge/index.md":          "---\nokf_version: \"0.2\"\n---\n\n* [a](computations/a.md)\n* [b](computations/b.md)\n* [c](computations/c.md)\n* [d](computations/d.md)\n",
		"scripts/revenue.sql":         "select 1",
		"knowledge/computations/a.md": "---\ntype: Attested Computation\nruntime: bigquery\n---\n\n# Computation\n\n```sql\nselect 1\n```\n",
		"knowledge/computations/b.md": "---\ntype: Attested Computation\nruntime: bigquery\ncomputation: scripts/revenue.sql\n---\n\nbody\n",
		// No runtime, and neither carrier of the computation itself.
		"knowledge/computations/c.md": "---\ntype: Attested Computation\n---\n\n# Freshness\n\n```\ndaily\n```\n",
		// Both carriers: a reader has two sources of truth to reconcile.
		"knowledge/computations/d.md": "---\ntype: Attested Computation\nruntime: bigquery\ncomputation: scripts/revenue.sql\n---\n\n# Computation\n\n```sql\nselect 1\n```\n",
	})

	want(t, check(t, repo, onlyDoc(AttestedComputationHasContract)),
		"needs `runtime`",
		"needs exactly one of",
		"needs exactly one of")
}
