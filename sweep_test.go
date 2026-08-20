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

	// What the fleet actually runs. Its module path contains okf as a prefix,
	// which is how the pin readers came to match none of it.
	repo("fleetpinned", map[string]string{
		".git/HEAD": "ref: refs/heads/main\n", "knowledge/index.md": index,
		"knowledge/decisions/d.md": concept,
		".githooks/pre-push":       "#!/bin/sh\ngo install github.com/fairyhunter13/okfrules/cmd/okfrules@v0.2.0\nokfrules -strict check -Werror knowledge\n",
		// okf here and okfrules in the hook, so neither pin reader can be
		// carried by the other: each module reaches the report one way only.
		"scripts/okfcheck/go.mod": "module x\n\nrequire github.com/fairyhunter13/okf v0.2.0\n",
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
	if len(by) != 6 {
		t.Fatalf("found %d repos, want 6: %v", len(by), by)
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

// The pin readers matched `okf@` and `okf ` literally, so the day every fleet
// gate moved to okfrules they reported no pins at all -- and no pins reads as
// no drift. A repo running both modules is the case that tells them apart.
func TestPinsNameTheirModule(t *testing.T) {
	reports, err := Sweep([]string{fleet(t)}, "", refDate)
	if err != nil {
		t.Fatal(err)
	}
	var r RepoReport
	for _, rep := range reports {
		if filepath.Base(rep.Path) == "fleetpinned" {
			r = rep
		}
	}
	if got := strings.Join(r.Pins, ","); got != "okf@v0.2.0,okfrules@v0.2.0" {
		t.Errorf("pins = %q, want both modules named", got)
	}
	if drift := r.PinDrift(); len(drift) != 0 {
		t.Errorf("drift = %v, want none: two modules at one version each", drift)
	}
}

// --json existed for a pipe it could not reach: the report went to the stderr
// the caller passed in, so `okf sweep --json 2>/dev/null` printed nothing.
func TestTheReportGoesToStdoutAndErrorsDoNot(t *testing.T) {
	var out, errs strings.Builder
	sweepOut = &out
	t.Cleanup(func() { sweepOut = os.Stdout })

	if code := sweepMain([]string{"--json", "--roots", fleet(t)}, &errs); code != 0 {
		t.Fatalf("exit %d: %s", code, errs.String())
	}
	if !strings.HasPrefix(strings.TrimSpace(out.String()), "[") {
		t.Errorf("report did not reach stdout: %q", out.String())
	}
	if errs.Len() != 0 {
		t.Errorf("stderr carried report output: %q", errs.String())
	}

	out.Reset()
	errs.Reset()
	if code := sweepMain(nil, &errs); code != 2 || errs.Len() == 0 {
		t.Errorf("a usage error must reach stderr: exit %d, stderr %q", code, errs.String())
	}
	if out.Len() != 0 {
		t.Errorf("stdout carried a usage error: %q", out.String())
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
	if strings.Join(d.Pins, ",") != "okf@v0.1.0,okf@v0.2.0" {
		t.Errorf("pins = %v, want the hook literal and the go.mod require", d.Pins)
	}
	if strings.Join(d.PinDrift(), ";") != "okf: v0.1.0, v0.2.0" {
		t.Errorf("drift = %v, want one module at two versions", d.PinDrift())
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
