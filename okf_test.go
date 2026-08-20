package okf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var refDate = time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)

// The spec's leniency is easy to over-tighten by accident, so conformance is
// pinned to Google's own bundles rather than to our reading of §11.
func TestGoogleBundlesConform(t *testing.T) {
	roots, err := filepath.Glob("testdata/google/*")
	if err != nil || len(roots) == 0 {
		t.Fatalf("no fixtures: %v", err)
	}
	for _, root := range roots {
		if fi, err := os.Stat(root); err != nil || !fi.IsDir() {
			continue
		}
		t.Run(filepath.Base(root), func(t *testing.T) {
			findings, err := CheckBundle(root, refDate)
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range findings {
				if f.Sev == Error {
					t.Errorf("rejected a conformant bundle: %s", f)
				}
			}
		})
	}
}

// §11 forbids rejecting a bundle over links, staleness, or filing, so those
// findings must never escalate to Error, however the checker is later extended.
func TestSeverityNeverEscalates(t *testing.T) {
	roots, _ := filepath.Glob("testdata/google/*")
	for _, root := range roots {
		findings, err := CheckBundle(root, refDate)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range findings {
			if f.Sev != Error {
				continue
			}
			for _, soft := range []string{"link", "stale", "orphan"} {
				if strings.Contains(f.Msg, soft) {
					t.Errorf("advisory finding raised to error: %s", f)
				}
			}
		}
	}
}

func TestOrphanFindings(t *testing.T) {
	root := t.TempDir()
	write(t, root, "index.md", "# Decision\n\n* [Linked](decisions/linked.md) - desc\n")
	write(t, root, "decisions/linked.md", "---\ntype: Decision\n---\nbody\n")
	write(t, root, "decisions/orphan.md", "---\ntype: Decision\n---\nbody\n")

	findings, err := CheckBundle(root, refDate)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("got %v, want one orphan warning", findings)
	}
	if f := findings[0]; f.Sev != Warning || f.Path != "decisions/orphan.md" {
		t.Errorf("got %s, want a warning on decisions/orphan.md", f)
	}
}

// A repo-local rule sees every concept and no reserved file: index.md and
// log.md have no `type`, so a rule keying off one would misfire on them.
func TestRulesSeeConceptsOnly(t *testing.T) {
	root := t.TempDir()
	write(t, root, "index.md", "# Decision\n\n* [D](decisions/d.md) - desc\n")
	write(t, root, "log.md", "## 2026-08-16\n- x\n")
	write(t, root, "decisions/d.md", "---\ntype: Decision\n---\nbody\n")

	var saw []string
	rule := func(d Doc) []Finding {
		saw = append(saw, d.Rel)
		if d.Root != root {
			t.Errorf("Root = %q, want %q", d.Root, root)
		}
		return []Finding{{Sev: Error, Msg: "local rule"}}
	}

	findings, err := CheckBundle(root, refDate, rule)
	if err != nil {
		t.Fatal(err)
	}
	if len(saw) != 1 || saw[0] != "decisions/d.md" {
		t.Errorf("rule ran on %v, want only decisions/d.md", saw)
	}
	if len(findings) != 1 {
		t.Fatalf("got %v, want the rule's one finding", findings)
	}
	if f := findings[0]; f.Path != "decisions/d.md" || f.Sev != Error {
		t.Errorf("got %s, want an error stamped with the concept path", f)
	}
}

func write(t *testing.T, root, rel, text string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}
