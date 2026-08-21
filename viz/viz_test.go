package viz

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "rewrite the golden payload")

// now is fixed so `stale` is a property of the fixture and not of the clock.
var now = time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC)

// The fixture exercises every link spelling a bundle actually contains. The
// absolute form is the one that matters most: the reference extractor drops it,
// which is why upstream's own acme_retail viewer renders 6 edges where this
// renders 14 on the identical bundle.
var fixture = map[string]string{
	"index.md": "---\nokf_version: \"0.2\"\n---\n# Index\n[a](/decisions/a.md)\n",
	"log.md":   "---\ntype: Log\n---\n# Log\n- did a thing\n",
	"decisions/a.md": `---
type: Decision
title: The absolute form is a link
description: Both spellings resolve.
tags: [okf, viz]
generated: {by: agent/1, at: 2026-08-20T00:00:00Z}
---
# Why

Rooted: [b](/metrics/b.md), relative: [c](../metrics/c.md).

Absent: [gone](/metrics/gone.md). Reserved: [log](/log.md). Self: [a](/decisions/a.md).

` + "```" + `
Fenced: [b](/metrics/b.md)
` + "```" + `
`,
	"metrics/b.md": `---
type: Metric
title: Human-reviewed and stale
resource: sql/b.sql
status: active
stale_after: 2026-01-01T00:00:00Z
verified:
  - by: agent/1
    at: 2026-08-01T00:00:00Z
  - by: "human:hafiz"
    at: 2026-08-02T00:00:00Z
sources:
  - resource: https://example.test/spec
    title: The spec
    last_modified: 2026-07-01T00:00:00Z
---
# B
`,
	"metrics/c.md": `---
type: Metric
title: Machine-confirmed only
verified: {by: "okf verify", at: 2026-08-03T00:00:00Z}
---
# C
`,
}

func build(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for rel, text := range fixture {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

var dataRe = regexp.MustCompile(`(?s)<script id="data" type="application/json">(.*?)</script>`)

func generate(t *testing.T) (string, payload) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "viz.html")
	if _, err := Generate(build(t), out, "Fixture", now); err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	m := dataRe.FindSubmatch(page)
	if m == nil {
		t.Fatal("no data block in the page")
	}
	var p payload
	if err := json.Unmarshal(m[1], &p); err != nil {
		t.Fatalf("data block is not the payload: %v", err)
	}
	return string(page), p
}

func TestNodesAreConceptsOnly(t *testing.T) {
	_, p := generate(t)
	var got []string
	for _, n := range p.Nodes {
		got = append(got, n.ID)
	}
	want := "decisions/a.md metrics/b.md metrics/c.md"
	if strings.Join(got, " ") != want {
		t.Errorf("nodes = %q, want %q (index.md and log.md are reserved)", strings.Join(got, " "), want)
	}
}

func TestBothLinkSpellingsBecomeEdges(t *testing.T) {
	_, p := generate(t)
	var got []string
	for _, e := range p.Edges {
		got = append(got, e.From+" -> "+e.To)
	}
	// One edge per resolvable target: the "/"-rooted b and the relative c. The
	// dangling, reserved, self and fenced links each have to produce nothing,
	// and the fenced one is a second mention of b, so a leak there shows up as
	// a duplicate rather than as a new target.
	want := "decisions/a.md -> metrics/b.md decisions/a.md -> metrics/c.md"
	if strings.Join(got, " ") != want {
		t.Errorf("edges = %q, want %q", strings.Join(got, " "), want)
	}
}
