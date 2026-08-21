// Package verify re-confirms what a concept points outward at, and records the
// result as §5.3's machine-confirmed tier. It answers one question per concept:
// does everything this document derives from still resolve, unchanged?
package verify

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fairyhunter13/okf"
)

const DefaultActor = "process:okf-verify"

type Options struct {
	Rules        okf.Rules
	Actor        string
	Now          time.Time
	Offline      bool
	RunVerifiers bool
	Client       *http.Client
}

// Verdict is one concept's outcome. Stampable is the load-bearing field: a
// concept with nothing outward to check is not evidence of anything, so it is
// left unverified rather than stamped on the strength of the rules alone.
type Verdict struct {
	Rel       string
	Stampable bool
	Proven    []string
	Blocked   []string
	Digests   map[string]string
}

func (v Verdict) Passed() bool { return v.Stampable && len(v.Blocked) == 0 }

// Verify walks a bundle and judges every concept. It writes nothing; Stamp and
// SetSourceDigest are the caller's to run, which is what keeps --dry-run honest.
func Verify(root string, opts Options) ([]Verdict, error) {
	if opts.Client == nil {
		opts.Client = &http.Client{Timeout: 30 * time.Second}
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}
	repo := repoRoot(root)

	blockers, err := ruleErrors(root, opts)
	if err != nil {
		return nil, err
	}

	var out []Verdict
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(p) != ".md" {
			return err
		}
		rel := filepath.ToSlash(mustRel(root, p))
		if okf.Reserved(rel) {
			return nil
		}
		text, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		fm, _, perr := okf.Parse(string(text))
		if perr != nil {
			return nil
		}
		v := Verdict{Rel: rel, Digests: map[string]string{}}
		for _, src := range sourcesOf(fm) {
			checkSource(&v, src, repo, root, opts)
		}
		checkVerifier(&v, fm, repo, opts)
		checkNotYetWritten(&v, fm, opts.Now)
		if n := blockers[rel]; n > 0 {
			v.Blocked = append(v.Blocked, fmt.Sprintf("%d conformance error(s)", n))
		}
		out = append(out, v)
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Rel < out[j].Rel })
	return out, err
}

// ruleErrors is gate 1, and it is the existing engine run unchanged: a concept
// that does not pass its own bundle's rules has not earned a trust signal.
func ruleErrors(root string, opts Options) (map[string]int, error) {
	findings, err := okf.CheckBundleWith(root, opts.Now, opts.Rules)
	if err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, f := range findings {
		if f.Sev == okf.Error {
			out[f.Path]++
		}
	}
	return out, nil
}

func sourcesOf(fm map[string]any) []map[string]any {
	switch s := fm["sources"].(type) {
	case []any:
		var out []map[string]any
		for _, e := range s {
			if m, ok := e.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	case map[string]any:
		return []map[string]any{s}
	}
	return nil
}

func repoRoot(bundle string) string {
	cmd := exec.Command("git", "-C", bundle, "rev-parse", "--show-toplevel")
	if out, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	return filepath.Dir(bundle)
}

func mustRel(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return p
	}
	return rel
}
