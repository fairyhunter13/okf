package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fairyhunter13/okf"
)

// The flag lived in main(), so nothing graded the one thing it decides: which
// tier runs. A tier comparison would pass on two rule sets that behave alike,
// so this asks the rule only Strict carries whether it fired.
func TestStrictReachesTheTierStandardDoesNot(t *testing.T) {
	dir := t.TempDir()
	bundle := filepath.Join(dir, "knowledge")
	if err := os.MkdirAll(filepath.Join(bundle, "decisions"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(rel, text string) {
		if err := os.WriteFile(filepath.Join(bundle, rel), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("index.md", "---\nokf_version: \"0.2\"\n---\n\n## Decision\n\n* [a](decisions/a.md)\n")
	write("decisions/a.md", "---\ntype: Decision\n---\n\nbody\n")
	write("log.md", "---\ntype: Log\ntitle: fixture knowledge history\n---\n\n## 2026-08-21\n\n- **Tidied** a\n")

	run := func(args ...string) (int, string) {
		rules, rest := selectRules(args)
		var out strings.Builder
		return okf.MainWith(rest, &out, rules), out.String()
	}

	if code, out := run("check", "-Werror", bundle); code != 0 {
		t.Errorf("Standard reported the log verb: exit %d, %q", code, out)
	}
	code, out := run("-strict", "check", "-Werror", bundle)
	if code == 0 || !strings.Contains(out, `log verb "Tidied"`) {
		t.Errorf("-strict did not reach Strict(): exit %d, %q", code, out)
	}
}
