package verify

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// checkSource is gate 2. It dispatches on the shape of §5.1's `resource`
// because the spec allows a scope descriptor a consumer cannot follow —
// demanding reachability of every source would fail concepts that are correct.
func checkSource(v *Verdict, src map[string]any, repo, root string, opts Options) {
	res, _ := src["resource"].(string)
	if res == "" {
		return
	}
	switch Classify(res) {
	case Commit:
		checkCommits(v, res, repo)
	case Path:
		v.Stampable = true
		if !PathExists(repo, root, res) {
			v.Blocked = append(v.Blocked, "path is gone: "+res)
			return
		}
		v.Proven = append(v.Proven, res)
	case URL:
		checkURL(v, src, res, opts)
	}
}

func checkCommits(v *Verdict, res, repo string) {
	shas := Shas(res)
	if len(shas) == 0 {
		return
	}
	v.Stampable = true
	for _, sha := range shas {
		if !CommitExists(repo, sha) {
			v.Blocked = append(v.Blocked, "commit not in this history: "+sha)
			return
		}
	}
	v.Proven = append(v.Proven, res)
}

// checkURL records a digest the first time and compares it afterwards. Drift is
// the finding worth having: a reachability check passes forever on a doc site
// that rewrote the page underneath the concept.
func checkURL(v *Verdict, src map[string]any, res string, opts Options) {
	v.Stampable = true
	if opts.Offline {
		v.Blocked = append(v.Blocked, "not checked (offline): "+res)
		return
	}
	sum, err := FetchURL(opts.Client, res)
	if err != nil {
		v.Blocked = append(v.Blocked, err.Error())
		return
	}
	was, _ := src["digest"].(string)
	switch {
	case was == sum && was != "":
		v.Proven = append(v.Proven, "unchanged: "+res)
	case !stable(opts, res, sum):
		// An issue tracker or a directory listing rewrites its HTML per request, so its
		// digest never matches twice and reports drift forever. Reachability is all such
		// a source can prove, and claiming more of it was the false half of the check.
		v.Proven = append(v.Proven, "reachable, digest unstable: "+res)
	case was == "":
		v.Digests[res] = sum
		v.Proven = append(v.Proven, "digest recorded: "+res)
	default:
		v.Blocked = append(v.Blocked, fmt.Sprintf("source drifted: %s (%s -> %s)", res, short(was), short(sum)))
	}
}

// stable re-fetches once and reports whether the two answers agree. It runs only
// where the answer changes what is written -- recording a new digest, or calling a
// mismatch drift -- so a source that matches its record costs one request as before.
func stable(opts Options, res, sum string) bool {
	again, err := FetchURL(opts.Client, res)
	return err == nil && again == sum
}

// checkVerifier is gate 3. A command named in a data file runs only when the
// operator asked for it, and an unevaluated gate blocks the stamp rather than
// being quietly counted as passing.
func checkVerifier(v *Verdict, fm map[string]any, repo string, opts Options) {
	spec, ok := fm["verifier"].(map[string]any)
	if !ok {
		return
	}
	cmdline, _ := spec["command"].(string)
	if strings.TrimSpace(cmdline) == "" {
		return
	}
	v.Stampable = true
	if res, _ := spec["resource"].(string); res != "" && !PathExists(repo, "", res) {
		v.Blocked = append(v.Blocked, "verifier.resource is gone: "+res)
		return
	}
	if !opts.RunVerifiers {
		v.Blocked = append(v.Blocked, "verifier not run (--run-verifiers)")
		return
	}
	cmd := exec.Command("sh", "-c", cmdline)
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		v.Blocked = append(v.Blocked, fmt.Sprintf("verifier failed: %v: %s", err, firstLine(out)))
		return
	}
	v.Proven = append(v.Proven, "verifier passed: "+cmdline)
}

// checkNotYetWritten refuses to write a stamp its own rules would reject. A
// `generated.at` rounded up to the next hour puts the concept in the future, and
// a stamp cannot confirm what has not been written yet.
func checkNotYetWritten(v *Verdict, fm map[string]any, now time.Time) {
	if !v.Stampable {
		return
	}
	gen, ok := fm["generated"].(map[string]any)
	if !ok {
		return
	}
	// yaml.v3 decodes an unquoted timestamp into time.Time, a quoted one into a
	// string, and the fleet writes both.
	at := fmt.Sprintf("%v", gen["at"])
	written, ok := gen["at"].(time.Time)
	if !ok {
		var err error
		if written, err = time.Parse(time.RFC3339, at); err != nil {
			return
		}
	}
	if now.Before(written) {
		v.Blocked = append(v.Blocked, "generated.at is in the future: "+at)
	}
}

// warnStaleAhead names the cliff before a bundle falls off it. §5.5 makes
// staleness a plain date comparison, which says nothing until the day it says
// red; a passing run must not move the date, so the only honest thing left is
// to count down to it.
func warnStaleAhead(v *Verdict, fm map[string]any, now time.Time) {
	raw, ok := fm["stale_after"]
	if !ok {
		return
	}
	// yaml.v3 resolves a bare YYYY-MM-DD to time.Time and a quoted one to a string.
	s := fmt.Sprintf("%v", raw)
	if len(s) > 10 {
		s = s[:10]
	}
	on, err := time.Parse("2006-01-02", s)
	if err != nil {
		return
	}
	if days := int(on.Sub(now).Hours() / 24); days >= 0 && days <= staleWarnDays {
		v.Warn = append(v.Warn, fmt.Sprintf("stale in %d day(s), on %s", days, s))
	}
}

// Far enough out that a fix is scheduling rather than an interrupt.
const staleWarnDays = 30

func short(digest string) string {
	if i := strings.IndexByte(digest, ':'); i >= 0 && len(digest) > i+13 {
		return digest[i+1 : i+13]
	}
	return digest
}

func firstLine(b []byte) string {
	s := strings.TrimSpace(string(b))
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
