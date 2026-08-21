package verify

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClassifyCoversTheFleetsFourShapes(t *testing.T) {
	for res, want := range map[string]Kind{
		"https://github.com/o/r/blob/main/SPEC.md": URL,
		"commit 30102961":                          Commit,
		"commits 0f826d2, 87520ff":                 Commit,
		"repositories/cx-be/api/openapi.yml":       Path,
		"all queries in BigQuery project X":        Descriptor,
	} {
		if got := Classify(res); got != want {
			t.Errorf("Classify(%q) = %v, want %v", res, got, want)
		}
	}
}

func TestShasIgnoresTheProseHalf(t *testing.T) {
	got := Shas("commits 0f826d2, 87520ff (the rename)")
	if strings.Join(got, ",") != "0f826d2,87520ff" {
		t.Fatalf("got %v", got)
	}
}

func TestCandidateURLsPreferTheByteStableForm(t *testing.T) {
	got := candidateURLs("https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md")
	if got[0] != "https://raw.githubusercontent.com/GoogleCloudPlatform/knowledge-catalog/main/okf/SPEC.md" {
		t.Fatalf("blob not rewritten to raw: %v", got)
	}
	if got := candidateURLs("https://code.claude.com/docs/en/hooks"); got[0] != "https://code.claude.com/docs/en/hooks.md" {
		t.Fatalf("extensionless docs path did not try .md: %v", got)
	}
}

// A descriptor is not a failure and not a pass: it leaves the concept with no
// outward gate, so nothing is stamped on the strength of it.
func TestScopeDescriptorEarnsNoStamp(t *testing.T) {
	v := verdictFor(t, "sources:\n  - id: s\n    resource: all queries in project X\n", Options{})
	if v.Stampable {
		t.Fatal("a descriptor made the concept stampable")
	}
}

// VerifiedWellFormed rejects a stamp older than what it confirms, so writing one
// would put the bundle in a state its own checker fails.
func TestAConceptDatedInTheFutureIsNotStamped(t *testing.T) {
	fm := "generated: {by: claude/opus-5, at: 2026-08-21T08:00:00Z}\nverifier:\n  command: \"true\"\n"
	v := verdictFor(t, fm, Options{RunVerifiers: true})
	if v.Passed() {
		t.Fatal("stamped a concept written after the stamp")
	}
	if !strings.Contains(strings.Join(v.Blocked, " "), "generated.at is in the future") {
		t.Fatalf("not named: %v", v.Blocked)
	}
}

func TestDriftBlocksTheStamp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/markdown")
		w.Write([]byte("today's text"))
	}))
	defer srv.Close()

	fm := "sources:\n  - id: s\n    resource: " + srv.URL + "/doc.md\n    digest: sha256:0000\n"
	v := verdictFor(t, fm, Options{Client: srv.Client()})
	if v.Passed() {
		t.Fatal("a drifted source was stamped")
	}
	if !strings.Contains(strings.Join(v.Blocked, " "), "source drifted") {
		t.Fatalf("drift not named: %v", v.Blocked)
	}
}

func TestFirstRunRecordsTheDigest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/markdown")
		w.Write([]byte("stable text"))
	}))
	defer srv.Close()

	fm := "sources:\n  - id: s\n    resource: " + srv.URL + "/doc.md\n"
	v := verdictFor(t, fm, Options{Client: srv.Client()})
	if !v.Passed() || len(v.Digests) != 1 {
		t.Fatalf("passed=%v digests=%v blocked=%v", v.Passed(), v.Digests, v.Blocked)
	}
}

// Three of the fleet's sources are issue trackers and directory listings whose
// HTML differs on every request. Digesting those reported drift forever.
func TestAPageThatDiffersOnEveryFetchRecordsNoDigest(t *testing.T) {
	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		w.Header().Set("Content-Type", "text/markdown")
		fmt.Fprintf(w, "csrf token %d", n)
	}))
	defer srv.Close()

	fm := "sources:\n  - id: s\n    resource: " + srv.URL + "/issue\n"
	v := verdictFor(t, fm, Options{Client: srv.Client()})
	if !v.Passed() {
		t.Fatalf("a reachable source was blocked: %v", v.Blocked)
	}
	if len(v.Digests) != 0 {
		t.Fatalf("recorded a digest that can never match again: %v", v.Digests)
	}
}

func TestOfflineBlocksRatherThanAssumes(t *testing.T) {
	fm := "sources:\n  - id: s\n    resource: https://example.invalid/x.md\n"
	v := verdictFor(t, fm, Options{Offline: true})
	if v.Passed() {
		t.Fatal("an unchecked URL counted as verified")
	}
}

// An unevaluated gate is not a passing gate.
func TestVerifierNotRunBlocks(t *testing.T) {
	v := verdictFor(t, "verifier:\n  command: \"true\"\n", Options{})
	if v.Passed() {
		t.Fatal("verifier counted as passed without running")
	}
	v = verdictFor(t, "verifier:\n  command: \"true\"\n", Options{RunVerifiers: true})
	if !v.Passed() {
		t.Fatalf("passing verifier did not pass: %v", v.Blocked)
	}
	v = verdictFor(t, "verifier:\n  command: \"exit 3\"\n", Options{RunVerifiers: true})
	if v.Passed() {
		t.Fatal("failing verifier passed")
	}
}

func TestCommitSourceIsCheckedAgainstThisHistory(t *testing.T) {
	repo := gitRepo(t)
	sha, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Skipf("no usable git: %v", err)
	}
	head := strings.TrimSpace(string(sha))

	writeConcept(t, repo, "sources:\n  - id: s\n    resource: commit "+head+"\n")
	v := onlyStampable(t, repo)
	if !v.Passed() {
		t.Fatalf("real commit blocked: %v", v.Blocked)
	}

	writeConcept(t, repo, "sources:\n  - id: s\n    resource: commit 0000000\n")
	v = onlyStampable(t, repo)
	if v.Passed() {
		t.Fatal("a sha not in this history passed")
	}
}

func gitRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"}, {"config", "user.email", "t@t"}, {"config", "user.name", "t"},
		{"commit", "-q", "--allow-empty", "-m", "root"},
	} {
		if err := exec.Command("git", append([]string{"-C", repo}, args...)...).Run(); err != nil {
			t.Skipf("git unavailable: %v", err)
		}
	}
	return repo
}

func writeConcept(t *testing.T, repo, extra string) {
	t.Helper()
	dir := filepath.Join(repo, "knowledge", "decisions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	idx := "---\nokf_version: \"0.2\"\n---\n\n* [a](decisions/a.md)\n"
	if err := os.WriteFile(filepath.Join(repo, "knowledge", "index.md"), []byte(idx), 0o644); err != nil {
		t.Fatal(err)
	}
	doc := "---\ntype: Decision\ntitle: A\n" + extra + "---\n\nbody\n"
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
}

func verdictFor(t *testing.T, extra string, opts Options) Verdict {
	t.Helper()
	repo := t.TempDir()
	writeConcept(t, repo, extra)
	opts.Now = time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC)
	vs, err := Verify(filepath.Join(repo, "knowledge"), opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 1 {
		t.Fatalf("want 1 verdict, got %d", len(vs))
	}
	return vs[0]
}

func onlyStampable(t *testing.T, repo string) Verdict {
	t.Helper()
	vs, err := Verify(filepath.Join(repo, "knowledge"), Options{Now: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	return vs[0]
}
