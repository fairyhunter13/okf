package viz

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestV2SignalsReachThePayload(t *testing.T) {
	_, p := generate(t)
	by := map[string]node{}
	for _, n := range p.Nodes {
		by[n.ID] = n
	}

	b := by["metrics/b.md"]
	if b.Trust != trustHuman {
		t.Errorf("b trust = %q, want %q: a human: actor outranks the machine stamp beside it", b.Trust, trustHuman)
	}
	if !b.Stale {
		t.Error("b is not stale, but stale_after is seven months behind the fixture clock")
	}
	if len(b.Sources) != 1 || b.Sources[0].LastModified != "2026-07-01T00:00:00Z" {
		t.Errorf("b sources = %+v, want one source carrying its last_modified", b.Sources)
	}
	if len(b.CitedBy) != 1 || b.CitedBy[0] != "decisions/a.md" {
		t.Errorf("b cited_by = %v, want the reverse index populated", b.CitedBy)
	}

	if c := by["metrics/c.md"]; c.Trust != trustMachine {
		t.Errorf("c trust = %q, want %q: §5.2's one-map spelling still counts", c.Trust, trustMachine)
	}
	if a := by["decisions/a.md"]; a.Trust != trustNone || a.Stale {
		t.Errorf("a trust/stale = %q/%v, want %q/false", a.Trust, a.Stale, trustNone)
	}
	if strings.Join(p.Types, " ") != "Decision Metric" {
		t.Errorf("types = %v, want the concept types only", p.Types)
	}
}

// The viewer's whole claim is that it still opens on a machine with no network.
func TestPageReferencesNothingExternal(t *testing.T) {
	page, _ := generate(t)
	external := regexp.MustCompile(`(?i)(?:src|href)\s*=\s*"(?:https?:)?//[^"]*`)
	for _, ref := range external.FindAllString(page, -1) {
		t.Errorf("external reference in the page: %s", ref)
	}
}

func TestPayloadIsDeterministic(t *testing.T) {
	a, _ := generate(t)
	b, _ := generate(t)
	if a != b {
		t.Error("two runs over the same bundle differ; the layout seed is not doing its job")
	}
}

// The golden is the payload rather than the page: it is the half derived from
// the bundle, and goldening the embedded CSS and JS would only churn.
func TestGoldenPayload(t *testing.T) {
	_, p := generate(t)
	got, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	golden := filepath.Join("testdata", "golden", "fixture.json")
	if *update {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("payload drifted from %s; read the diff before running -update", golden)
	}
}
