package sweep

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/fairyhunter13/okf"
	"github.com/fairyhunter13/okf/rules"
)

// A repo is any directory holding both .git and a bundle, so an eleventh repo
// is picked up the day it gets one and there is no registry to drift.
var bundleNames = []string{"knowledge"}

type RepoReport struct {
	Path            string         `json:"path"`
	Bundle          string         `json:"bundle"`
	Concepts        int            `json:"concepts"`
	Errors          int            `json:"errors"`
	Warnings        int            `json:"warnings"`
	Findings        []string       `json:"findings,omitempty"`
	Gates           []string       `json:"gates,omitempty"`
	Pins            []string       `json:"pins,omitempty"`
	Checkers        []string       `json:"checkers,omitempty"`
	Stale           []string       `json:"stale,omitempty"`
	Memory          []string       `json:"unresolved_memory,omitempty"`
	MemoryUnchecked bool           `json:"memory_unchecked,omitempty"`
	Coverage        map[string]int `json:"coverage,omitempty"`
}

// Ungated is the finding the sweep exists for: a bundle nothing checks is not a
// clean bundle, it is an unmeasured one.
func (r RepoReport) Ungated() bool { return len(r.Gates) == 0 }

// Unpinned is the half of the pin story PinDrift cannot tell: it fires on one
// module at two versions, so a repo whose gates run but whose version literal
// nothing could read reports an empty list and passes. Empty is not agreement,
// it is no measurement -- and the repo that defines the fleet's pin was the one
// repo whose pin the sweep could not see.
func (r RepoReport) Unpinned() bool { return len(r.Gates) > 0 && len(r.Pins) == 0 }

// PinDrift names each module this repo pins at more than one version. Two pins
// are not drift once a repo runs both okf and okfrules; two versions of one
// module are, whichever files they came from.
func (r RepoReport) PinDrift() []string {
	byModule := map[string][]string{}
	for _, pin := range r.Pins {
		mod, ver, _ := strings.Cut(pin, "@")
		byModule[mod] = append(byModule[mod], ver)
	}
	var out []string
	for mod, vers := range byModule {
		if len(vers) > 1 {
			out = append(out, mod+": "+strings.Join(vers, ", "))
		}
	}
	sort.Strings(out)
	return out
}

var memoryRefRe = regexp.MustCompile(`\[\[memory:([^\]]+)\]\]`)

var coverageFields = []string{"generated", "verified", "sources", "resource", "stale_after", "status", "tags"}

// Sweep reports every bundle under roots, graded by okf plus rules.Standard().
// It ran the stock check alone while the rules were a separate module importing
// okf; this package imports both, so the report now covers what the fleet's
// gates actually enforce. Strict and repo-local rules still do not run here --
// the checker column names who runs them.
func Sweep(roots []string, memoryDir string, today time.Time) ([]RepoReport, error) {
	var out []RepoReport
	for _, root := range roots {
		repos, err := discover(root)
		if err != nil {
			return nil, err
		}
		for _, r := range repos {
			rep, err := sweepRepo(r.repo, r.bundle, memoryDir, today)
			if err != nil {
				return nil, err
			}
			out = append(out, rep)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

type found struct{ repo, bundle string }

func discover(root string) ([]found, error) {
	var out []found
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("sweep root %s: %w", root, err)
	}
	for _, e := range entries {
		repo := filepath.Join(root, e.Name())
		if fi, err := os.Stat(repo); err != nil || !fi.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(repo, ".git")); err != nil {
			continue
		}
		for _, name := range bundleNames {
			if fi, err := os.Stat(filepath.Join(repo, name)); err == nil && fi.IsDir() {
				out = append(out, found{repo: repo, bundle: name})
				break
			}
		}
	}
	return out, nil
}

func sweepRepo(repo, bundle, memoryDir string, today time.Time) (RepoReport, error) {
	rep := RepoReport{Path: repo, Bundle: bundle, Coverage: map[string]int{}}

	findings, err := okf.CheckBundleWith(filepath.Join(repo, bundle), today, rules.Standard())
	if err != nil {
		return rep, err
	}
	for _, f := range findings {
		if f.Sev == okf.Error {
			rep.Errors++
		} else {
			rep.Warnings++
		}
		rep.Findings = append(rep.Findings, f.String())
	}

	if err := walkConcepts(filepath.Join(repo, bundle), func(rel string, fm map[string]any, body string) {
		rep.Concepts++
		for _, k := range coverageFields {
			if _, ok := fm[k]; ok {
				rep.Coverage[k]++
			}
		}
		if s, ok := staleDate(fm); ok && !today.Before(s) {
			rep.Stale = append(rep.Stale, fmt.Sprintf("%s: stale since %s", rel, s.Format("2006-01-02")))
		}
		if memoryDir == "" {
			return
		}
		for _, m := range memoryRefRe.FindAllStringSubmatch(body, -1) {
			if !memoryResolves(memoryDir, m[1]) {
				rep.Memory = append(rep.Memory, fmt.Sprintf("%s: [[memory:%s]]", rel, m[1]))
			}
		}
	}); err != nil {
		return rep, err
	}

	rep.MemoryUnchecked = memoryDir == ""
	rep.Gates, rep.Pins, rep.Checkers = gates(repo)
	return rep, nil
}

func walkConcepts(root string, fn func(rel string, fm map[string]any, body string)) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(p) != ".md" {
			return err
		}
		rel := filepath.ToSlash(mustRel(root, p))
		if okf.Reserved(rel) {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		fm, body, perr := okf.Parse(string(b))
		if perr != nil {
			return nil
		}
		fn(rel, fm, body)
		return nil
	})
}

func staleDate(fm map[string]any) (time.Time, bool) {
	raw, ok := fm["stale_after"]
	if !ok {
		return time.Time{}, false
	}
	s := fmt.Sprintf("%v", raw)
	if len(s) > 10 {
		s = s[:10]
	}
	d, err := time.Parse("2006-01-02", s)
	return d, err == nil
}

// The four profile directories hold different memory trees, so a verdict that
// depends on which one ran would be no verdict. Unset used to report every
// reference unresolved, which put nine advisory-looking lines under the largest
// bundle on every default run; the report now says once that the check did not
// run, which is the same refusal to guess without the false findings.
func memoryResolves(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, strings.TrimSuffix(name, ".md")+".md"))
	return err == nil
}

// The header names the scope because the counts were read as a verdict on the
// shared rules, which this cannot run (see Sweep). The per-repo checker line is
// the other half: it names who does grade them here.
func writeSweep(w io.Writer, reports []RepoReport, asJSON bool) {
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(reports)
		return
	}
	fmt.Fprintln(w, "verdict: okf plus the Standard fleet rules -- a repo's strict or local rules run only in its own gate, named per line")
	if len(reports) > 0 && reports[0].MemoryUnchecked {
		fmt.Fprintln(w, "         [[memory:...]] references unchecked: --memory not set")
	}
	for _, r := range reports {
		gate := strings.Join(r.Gates, ", ")
		if r.Ungated() {
			gate = "UNGATED"
		}
		checker := strings.Join(r.Checkers, ", ")
		if checker == "" {
			checker = "unknown"
		}
		fmt.Fprintf(w, "%s  %d concepts  %d error(s)  %d warning(s)  gates: %s  checker: %s\n",
			r.Path, r.Concepts, r.Errors, r.Warnings, gate, checker)
		for _, group := range [][]string{r.Findings, r.Stale, r.Memory} {
			for _, line := range group {
				fmt.Fprintf(w, "    %s\n", line)
			}
		}
		if r.Unpinned() {
			fmt.Fprintf(w, "    UNPINNED: gates run, no version literal any reader could find\n")
		}
		for _, drift := range r.PinDrift() {
			fmt.Fprintf(w, "    pin drift: %s\n", drift)
		}
	}
}

func mustRel(root, p string) string {
	r, err := filepath.Rel(root, p)
	if err != nil {
		return p
	}
	return r
}
