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
	case was == "":
		v.Digests[res] = sum
		v.Proven = append(v.Proven, "digest recorded: "+res)
	case was != sum:
		v.Blocked = append(v.Blocked, fmt.Sprintf("source drifted: %s (%s -> %s)", res, short(was), short(sum)))
	default:
		v.Proven = append(v.Proven, "unchanged: "+res)
	}
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
