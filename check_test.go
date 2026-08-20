package okf

import (
	"io"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	for _, tc := range []struct {
		name, in string
		wantErr  bool
		wantType any
	}{
		{name: "no frontmatter", in: "# Just a body\n"},
		{name: "unterminated", in: "---\ntype: X\n", wantErr: true},
		{name: "invalid yaml", in: "---\ntype: [unclosed\n---\n", wantErr: true},
		{name: "empty block", in: "---\n---\nbody\n"},
		{name: "typed", in: "---\ntype: Decision\n---\nbody\n", wantType: "Decision"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fm, _, err := Parse(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tc.wantErr)
			}
			if !tc.wantErr && fm["type"] != tc.wantType {
				t.Errorf("type = %v, want %v", fm["type"], tc.wantType)
			}
		})
	}
}

func TestCheckFile(t *testing.T) {
	for _, tc := range []struct {
		name, rel, text string
		wantSev         Severity
		wantFindings    int
	}{
		{name: "conformant concept", rel: "decisions/x.md", text: "---\ntype: Decision\n---\nbody\n"},
		{name: "missing type", rel: "decisions/x.md", text: "---\ntitle: X\n---\n", wantSev: Error, wantFindings: 1},
		{name: "empty type", rel: "decisions/x.md", text: "---\ntype: \"\"\n---\n", wantSev: Error, wantFindings: 1},
		{name: "no frontmatter", rel: "decisions/x.md", text: "# body\n", wantSev: Error, wantFindings: 1},
		{name: "index frontmatter", rel: "index.md", text: "---\ntype: Index\n---\n", wantSev: Error, wantFindings: 1},
		{name: "root okf_version", rel: "index.md", text: "---\nokf_version: \"0.2\"\n---\n"},
		{name: "nested okf_version", rel: "a/index.md", text: "---\nokf_version: \"0.2\"\n---\n", wantSev: Error, wantFindings: 1},
		{name: "log frontmatter ok", rel: "log.md", text: "---\ntype: Log\n---\n## 2026-08-16\n- x\n"},
		{name: "log bad heading", rel: "log.md", text: "## Last week\n- x\n", wantSev: Error, wantFindings: 1},
		{name: "stale", rel: "decisions/x.md", text: "---\ntype: Decision\nstale_after: 2026-01-01\n---\n", wantSev: Warning, wantFindings: 1},
		{name: "fresh", rel: "decisions/x.md", text: "---\ntype: Decision\nstale_after: 2027-01-01\n---\n"},
		{name: "broken link", rel: "decisions/x.md", text: "---\ntype: Decision\n---\n[a](nope.md)\n", wantSev: Warning, wantFindings: 1},
		// A "/" root means the bundle root, not the filesystem: the engine resolves
		// it and reports only what dangles. Preferring the relative spelling is a
		// producer opinion, and lives in rules.PreferRelativeLinks.
		{name: "absolute link", rel: "decisions/x.md", text: "---\ntype: Decision\n---\n[a](/nope.md)\n", wantSev: Warning, wantFindings: 1},
		{name: "external link", rel: "decisions/x.md", text: "---\ntype: Decision\n---\n[a](https://example.com/x.md)\n"},
		{name: "link in code span", rel: "decisions/x.md", text: "---\ntype: Decision\n---\nwrite `[a](path)` here\n"},
		{name: "link in fence", rel: "decisions/x.md", text: "---\ntype: Decision\n---\n```\n[a](path)\n```\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := checkFile(t.TempDir(), tc.rel, tc.text, refDate, nil)
			if len(got) != tc.wantFindings {
				t.Fatalf("got %d findings %v, want %d", len(got), got, tc.wantFindings)
			}
			if tc.wantFindings > 0 && got[0].Sev != tc.wantSev {
				t.Errorf("severity = %v, want %v", got[0].Sev, tc.wantSev)
			}
		})
	}
}

// The exit codes are the contract CI and the edit hook depend on: warnings
// alone must not fail a build unless -Werror asks for it.
func TestMainExitCodes(t *testing.T) {
	root := t.TempDir()
	write(t, root, "index.md", "# Decision\n\n* [D](decisions/d.md) - desc\n")
	write(t, root, "decisions/d.md", "---\ntype: Decision\n---\n[gone](nope.md)\n")

	bad := t.TempDir()
	write(t, bad, "decisions/d.md", "# no frontmatter\n")

	for _, tc := range []struct {
		name string
		args []string
		want int
	}{
		{name: "no subcommand", args: nil, want: 2},
		{name: "warnings only", args: []string{"check", root}, want: 0},
		{name: "warnings with -Werror", args: []string{"check", "-Werror", root}, want: 1},
		{name: "conformance error", args: []string{"check", bad}, want: 1},
		{name: "missing bundle", args: []string{"check", filepath.Join(root, "absent")}, want: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Main(tc.args, io.Discard); got != tc.want {
				t.Errorf("exit = %d, want %d", got, tc.want)
			}
		})
	}
}
