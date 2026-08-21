package verify

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Kind is what a §5.1 `resource` turns out to be. The spec allows a population
// or scope descriptor a consumer cannot follow, so classifying comes first:
// demanding reachability of everything would fail concepts that are correct.
type Kind int

const (
	Descriptor Kind = iota // "all queries in project X" — unverifiable by design
	Commit                 // "commit 30102961" — immutable, checkable offline
	URL                    // an absolute URL — fetched and digested
	Path                   // a repo- or bundle-relative file
)

var (
	commitRe = regexp.MustCompile(`^commits?\s+(.+)$`)
	shaRe    = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
	blobRe   = regexp.MustCompile(`^https://github\.com/([^/]+/[^/]+)/blob/(.+)$`)
)

func Classify(resource string) Kind {
	r := strings.TrimSpace(resource)
	switch {
	case strings.HasPrefix(r, "http://"), strings.HasPrefix(r, "https://"):
		return URL
	case commitRe.MatchString(r):
		return Commit
	case !strings.ContainsAny(r, " \t") && strings.Contains(r, "/"):
		return Path
	}
	return Descriptor
}

// Shas pulls the revisions out of a `commit`/`commits` descriptor. A token that
// is not a hex sha is dropped rather than failed: the prose half of "commits
// a, b (the rename)" is not a claim about a revision.
func Shas(resource string) []string {
	m := commitRe.FindStringSubmatch(strings.TrimSpace(resource))
	if m == nil {
		return nil
	}
	var out []string
	for _, tok := range strings.FieldsFunc(m[1], func(r rune) bool { return r == ',' || r == ' ' }) {
		if tok = strings.TrimSpace(tok); shaRe.MatchString(tok) {
			out = append(out, tok)
		}
	}
	return out
}

// FetchURL returns the sha256 of a source's stable text form. A docs site that
// answers a bare path with a JS shell rewrites that HTML on every deploy, so
// digesting it reports drift forever; the `.md` sibling both Claude docs hosts
// serve is the byte-stable thing to hash.
func FetchURL(client *http.Client, rawURL string) (string, error) {
	for _, u := range candidateURLs(rawURL) {
		sum, ct, err := fetch(client, u)
		if err != nil {
			continue
		}
		if u == rawURL || strings.Contains(ct, "markdown") || strings.Contains(ct, "text/plain") {
			return sum, nil
		}
	}
	return "", fmt.Errorf("unreachable: %s", rawURL)
}

// candidateURLs is the normalization, most stable first: a GitHub blob page is
// UI around a file that raw serves verbatim, and a docs path usually has an
// `.md` sibling. The original is last so an ordinary URL still works.
func candidateURLs(rawURL string) []string {
	var out []string
	if m := blobRe.FindStringSubmatch(rawURL); m != nil {
		out = append(out, "https://raw.githubusercontent.com/"+m[1]+"/"+m[2])
	}
	if filepath.Ext(rawURL) == "" {
		out = append(out, rawURL+".md")
	}
	return append(out, rawURL)
}

func fetch(client *http.Client, u string) (sum, contentType string, err error) {
	resp, err := client.Get(u)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("%s: %s", u, resp.Status)
	}
	h := sha256.New()
	if _, err := io.Copy(h, io.LimitReader(resp.Body, 32<<20)); err != nil {
		return "", "", err
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), resp.Header.Get("Content-Type"), nil
}

// CommitExists asks the repo, not the network. This is why most of the fleet
// verifies offline: a sha names an object that either is in this history or is
// not, and it can never change afterwards.
func CommitExists(repo, sha string) bool {
	cmd := exec.Command("git", "-C", repo, "cat-file", "-e", sha+"^{commit}")
	return cmd.Run() == nil
}

// PathExists resolves a §6.2 path-valued source against the repo root and the
// bundle root, because the fleet writes both.
func PathExists(repo, bundle, rel string) bool {
	for _, base := range []string{repo, bundle} {
		if base == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(base, filepath.FromSlash(rel))); err == nil {
			return true
		}
	}
	return false
}
