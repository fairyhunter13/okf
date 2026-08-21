package okf

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const shrinkBase = `---
type: Decision
title: A decision
sources:
  - resource: https://example.test/a
  - resource: https://example.test/b
verified:
  - by: agent/1
    at: 2026-08-01T00:00:00Z
---
# Context

Why it came up.

# Decision

What was decided.
`

// commit builds a one-concept bundle inside a real repository and returns its
// root. git is the whole point of the rule, so there is nothing to fake here.
func commit(t *testing.T, body string) string {
	t.Helper()
	repo := t.TempDir()
	root := filepath.Join(repo, "knowledge")
	if err := os.MkdirAll(filepath.Join(root, "decisions"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, root, "index.md", "---\nokf_version: \"0.2\"\n---\n# Index\n[d](/decisions/d.md)\n")
	write(t, root, "decisions/d.md", body)
	for _, args := range [][]string{
		{"init", "-q"},
		{"-c", "user.email=t@t", "-c", "user.name=t", "add", "-A"},
		{"-c", "user.email=t@t", "-c", "user.name=t", "commit", "-qm", "base"},
	} {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return root
}

func against(t *testing.T, root string) []string {
	t.Helper()
	f, err := CheckAgainst(root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, x := range f {
		if x.Sev != Warning {
			t.Errorf("%s is %s; §11 leaves this advisory", x.Msg, x.Sev)
		}
		out = append(out, x.Msg)
	}
	return out
}

func TestShrinkGuard(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(string) string
		want string // "" means silence
	}{
		{"unchanged", func(s string) string { return s }, ""},
		{"a heading is deleted", func(s string) string {
			return strings.Replace(s, "# Decision\n\nWhat was decided.\n", "", 1)
		}, `section "Decision" was dropped`},
		// Swapped, nothing lost. The finding names Decision because Context still
		// matches later in the new order and Decision then has nothing left to
		// match against — one finding per reorder, not two.
		{"headings are reordered", func(s string) string {
			s = strings.Replace(s, "# Context\n\nWhy it came up.\n\n", "", 1)
			return s + "\n# Context\n\nWhy it came up.\n"
		}, `section "Decision" was dropped or reordered`},
		{"a section is added", func(s string) string { return s + "\n# Consequences\n\nMore.\n" }, ""},
		{"a source is dropped", func(s string) string {
			return strings.Replace(s, "  - resource: https://example.test/b\n", "", 1)
		}, `source "https://example.test/b" was dropped`},
		{"a verification is dropped", func(s string) string {
			return strings.Replace(s, "verified:\n  - by: agent/1\n    at: 2026-08-01T00:00:00Z\n", "", 1)
		}, `verification event "agent/1 2026-08-01T00:00:00Z" was dropped`},
		{"the title is rewritten", func(s string) string {
			return strings.Replace(s, "title: A decision", "title: Something else", 1)
		}, `title changed from "A decision"`},
		{"deprecated is exempt", func(s string) string {
			s = strings.Replace(s, "type: Decision", "type: Decision\nstatus: deprecated", 1)
			return strings.Replace(s, "# Decision\n\nWhat was decided.\n", "", 1)
		}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := commit(t, shrinkBase)
			write(t, root, "decisions/d.md", tc.edit(shrinkBase))
			got := against(t, root)
			if tc.want == "" {
				if len(got) != 0 {
					t.Errorf("got %v, want silence", got)
				}
				return
			}
			for _, g := range got {
				if strings.Contains(g, tc.want) {
					return
				}
			}
			t.Errorf("got %v, want one containing %q", got, tc.want)
		})
	}
}

// A concept the ref never had cannot have shrunk, whatever it looks like now.
func TestShrinkGuardIgnoresNewConcepts(t *testing.T) {
	root := commit(t, shrinkBase)
	write(t, root, "decisions/new.md", "---\ntype: Decision\ntitle: New\n---\n# New\n")
	if got := against(t, root); len(got) != 0 {
		t.Errorf("got %v, want silence", got)
	}
}
