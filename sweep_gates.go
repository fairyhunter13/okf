package okf

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// A pin is module@version, not a bare version. The fleet's gates run okfrules
// rather than okf, and `okf(?:/cmd/okf)?@` never matched that path -- which is
// the failure this reports on other repos, happening here: pins read empty
// everywhere and drift went undetectable. Longest alternative first, or `okf`
// wins and the module reads as `okf` for both.
var (
	pinRe    = regexp.MustCompile(`github\.com/fairyhunter13/(okfrules|okf)(?:/cmd/\w+)?@(v[\w.+-]+|latest)`)
	modPinRe = regexp.MustCompile(`github\.com/fairyhunter13/(okfrules|okf)\s+(v[\w.+-]+)`)
)

// Where a gate can live. A repo hook only fires where core.hooksPath points at
// it, and .git/hooks is what a clone actually runs, so both are read.
var gatePaths = []string{
	".githooks/pre-commit", ".githooks/pre-push",
	".git/hooks/pre-commit", ".git/hooks/pre-push",
}

// gates names every place this repo runs okf, and every version literal it
// pins. A hook that exists but is not executable is reported as disarmed rather
// than as a gate: git skips it without a word.
func gates(repo string) (found []string, pins []string) {
	seen := map[string]bool{}
	add := func(m []string) {
		pin := m[1] + "@" + m[2]
		if !seen[pin] {
			seen[pin] = true
			pins = append(pins, pin)
		}
	}
	note := func(label, text string) {
		found = append(found, label)
		for _, m := range pinRe.FindAllStringSubmatch(text, -1) {
			add(m)
		}
	}

	for _, rel := range gatePaths {
		p := filepath.Join(repo, rel)
		fi, err := os.Stat(p)
		if err != nil {
			continue
		}
		text := readWithTarget(p)
		label := rel
		if !strings.Contains(text, "okf") {
			mk, ok := viaMake(repo, text)
			if !ok {
				continue
			}
			text, label = mk, rel+" (via Makefile)"
		}
		if fi.Mode()&0o111 == 0 {
			found = append(found, label+" (NOT EXECUTABLE)")
			continue
		}
		note(label, text)
	}

	for _, p := range workflowFiles(repo) {
		b, err := os.ReadFile(p)
		if err != nil || !strings.Contains(string(b), "okf") {
			continue
		}
		note(filepath.ToSlash(mustRel(repo, p)), string(b))
	}

	for _, p := range goModFiles(repo) {
		if b, err := os.ReadFile(p); err == nil {
			for _, m := range modPinRe.FindAllStringSubmatch(string(b), -1) {
				add(m)
			}
		}
	}

	sort.Strings(pins)
	return found, pins
}

// A repo whose gate is `go run .` pins okf in go.mod rather than in a version
// literal, so reading only the gate text would report it as unpinned.
func goModFiles(repo string) []string {
	out, _ := filepath.Glob(filepath.Join(repo, "go.mod"))
	for _, pat := range []string{"scripts/*/go.mod", "internal/*/go.mod", "cmd/*/go.mod"} {
		m, _ := filepath.Glob(filepath.Join(repo, filepath.FromSlash(pat)))
		out = append(out, m...)
	}
	sort.Strings(out)
	return out
}

// A hook is often a symlink to a script that holds the real invocation, so the
// text that decides whether okf runs is the target's, not the link's.
func readWithTarget(p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	text := string(b)
	if target, err := os.Readlink(p); err == nil {
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(p), target)
		}
		if tb, err := os.ReadFile(target); err == nil {
			text += "\n" + string(tb)
		}
	}
	return text
}

// A hook that shells out to `make` is where the invocation stops being visible
// in the hook. Reporting that as UNGATED is the more expensive error of the two.
func viaMake(repo, hook string) (string, bool) {
	if !strings.Contains(hook, "make ") && !strings.Contains(hook, "$(MAKE)") {
		return "", false
	}
	b, err := os.ReadFile(filepath.Join(repo, "Makefile"))
	if err != nil || !strings.Contains(string(b), "okf") {
		return "", false
	}
	return string(b), true
}

func workflowFiles(repo string) []string {
	var out []string
	for _, pat := range []string{"*.yml", "*.yaml"} {
		m, _ := filepath.Glob(filepath.Join(repo, ".github", "workflows", pat))
		out = append(out, m...)
	}
	sort.Strings(out)
	return out
}
