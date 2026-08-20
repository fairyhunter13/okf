package okf

import (
	"path"
	"strings"
	"testing"
)

// The reason BundleRule exists: a Rule never sees index.md or log.md, so an
// invariant over the reserved files is unimplementable without one.
func TestBundleRuleSeesTheReservedFiles(t *testing.T) {
	root := "testdata/regress"

	var seen []string
	rule := func(b Bundle) []Finding {
		for _, d := range b.Docs {
			seen = append(seen, d.Rel)
		}
		return nil
	}
	if _, err := CheckBundleWith(root, refDate, Rules{Bundle: []BundleRule{rule}}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"index.md", "log.md", "decisions/linked.md"} {
		if !contains(seen, want) {
			t.Errorf("bundle rule never saw %s; saw %v", want, seen)
		}
	}

	var concepts []string
	only := func(b Bundle) []Finding {
		for _, d := range b.Concepts() {
			concepts = append(concepts, d.Rel)
		}
		return nil
	}
	if _, err := CheckBundleWith(root, refDate, Rules{Bundle: []BundleRule{only}}); err != nil {
		t.Fatal(err)
	}
	for _, c := range concepts {
		if base := path.Base(c); base == "index.md" || base == "log.md" {
			t.Errorf("Concepts() returned the reserved %s", c)
		}
	}
}

// A bundle rule reports whatever severity it says, and Main counts it like any
// other finding -- an Error exits 1 without -Werror, a Warning does not.
func TestBundleRuleSeverityReachesTheExitCode(t *testing.T) {
	for _, tc := range []struct {
		name string
		sev  Severity
		want int
	}{
		{name: "error", sev: Error, want: 1},
		{name: "warning", sev: Warning, want: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rule := func(b Bundle) []Finding {
				return []Finding{{Path: "log.md", Sev: tc.sev, Msg: "from a bundle rule"}}
			}
			var sb strings.Builder
			got := MainWith([]string{"check", "testdata/google/ga4"}, &sb,
				Rules{Bundle: []BundleRule{rule}})
			if got != tc.want {
				t.Fatalf("exit %d, want %d: %s", got, tc.want, sb.String())
			}
			if !strings.Contains(sb.String(), "from a bundle rule") {
				t.Errorf("finding not printed: %q", sb.String())
			}
		})
	}
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
