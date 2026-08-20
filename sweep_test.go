package okf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fleet(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	repo := func(name string, files map[string]string, exec map[string]bool) {
		for rel, text := range files {
			p := filepath.Join(root, name, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			mode := os.FileMode(0o644)
			if exec[rel] {
				mode = 0o755
			}
			if err := os.WriteFile(p, []byte(text), mode); err != nil {
				t.Fatal(err)
			}
		}
	}
	concept := "---\ntype: Decision\n---\n\nbody\n"
	index := "---\nokf_version: \"0.2\"\n---\n\n* [d](decisions/d.md)\n"

	repo("gated", map[string]string{
		".git/HEAD": "ref: refs/heads/main\n", "knowledge/index.md": index,
		"knowledge/decisions/d.md": concept,
		".githooks/pre-push":       "#!/bin/sh\ngo install github.com/fairyhunter13/okf/cmd/okf@v0.1.0\nokf check -Werror knowledge\n",
	}, map[string]bool{".githooks/pre-push": true})

	repo("ungated", map[string]string{
		".git/HEAD": "ref: refs/heads/main\n", "knowledge/index.md": index,
		"knowledge/decisions/d.md": concept,
	}, nil)

	repo("disarmed", map[string]string{
		".git/HEAD": "ref: refs/heads/main\n", "knowledge/index.md": index,
		"knowledge/decisions/d.md": concept,
		".githooks/pre-push":       "#!/bin/sh\nokf check -Werror knowledge\n",
	}, nil)

	repo("viamake", map[string]string{
		".git/HEAD": "ref: refs/heads/main\n", "knowledge/index.md": index,
		"knowledge/decisions/d.md": concept,
		".githooks/pre-commit":     "#!/bin/sh\nmake lint-all\n",
		"Makefile":                 "lint-all:\n\tokf check -Werror knowledge\n",
	}, map[string]bool{".githooks/pre-commit": true})

	repo("drifted", map[string]string{
		".git/HEAD": "ref: refs/heads/main\n", "knowledge/index.md": index,
		"knowledge/decisions/d.md": "---\ntype: Decision\nstale_after: 2020-01-01\n---\n\nsee [[memory:absent-note]]\n",
		".githooks/pre-push":       "#!/bin/sh\ngo install github.com/fairyhunter13/okf/cmd/okf@v0.1.0\nokf check knowledge\n",
		"go.mod":                   "module x\n\nrequire github.com/fairyhunter13/okf v0.2.0\n",
	}, map[string]bool{".githooks/pre-push": true})

	repo("notarepo", map[string]string{"knowledge/index.md": index}, nil)
	return root
}

func TestSweepDiscoveryAndGates(t *testing.T) {
	reports, err := Sweep([]string{fleet(t)}, "", refDate)
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]RepoReport{}
	for _, r := range reports {
		by[filepath.Base(r.Path)] = r
	}

	if _, ok := by["notarepo"]; ok {
		t.Error("a directory with a bundle but no .git is not a repo")
	}
	if len(by) != 5 {
		t.Fatalf("found %d repos, want 5: %v", len(by), by)
	}
	if by["ungated"].Ungated() != true {
		t.Error("a repo nothing checks must report ungated, not clean")
	}
	if by["gated"].Ungated() {
		t.Errorf("gated repo reported ungated: %v", by["gated"].Gates)
	}
	if g := strings.Join(by["disarmed"].Gates, ","); !strings.Contains(g, "NOT EXECUTABLE") {
		t.Errorf("a hook git cannot execute is not a gate: %q", g)
	}
	if g := strings.Join(by["viamake"].Gates, ","); !strings.Contains(g, "Makefile") {
		t.Errorf("a hook that shells out to make still runs okf: %q", g)
	}
}

func TestSweepReportsDriftStalenessAndMemory(t *testing.T) {
	root := fleet(t)
	reports, err := Sweep([]string{root}, "", refDate)
	if err != nil {
		t.Fatal(err)
	}
	var d RepoReport
	for _, r := range reports {
		if filepath.Base(r.Path) == "drifted" {
			d = r
		}
	}
	if len(d.Pins) != 2 {
		t.Errorf("pins = %v, want the hook literal and the go.mod require", d.Pins)
	}
	if len(d.Stale) != 1 {
		t.Errorf("stale = %v, want the one past stale_after", d.Stale)
	}
	if len(d.Memory) != 1 {
		t.Errorf("memory = %v, want the one unresolved reference", d.Memory)
	}

	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "absent-note.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	reports, err = Sweep([]string{root}, home, refDate)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if filepath.Base(r.Path) == "drifted" && len(r.Memory) != 0 {
			t.Errorf("memory = %v, want none once the home holds the note", r.Memory)
		}
	}
}
