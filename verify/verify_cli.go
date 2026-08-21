package verify

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/fairyhunter13/okf"
)

// Main runs `okf verify`. Without -stamp it writes nothing, so the report and
// the run that acts on it are the same code path. Exit 1 means a stampable
// concept is blocked — drift, a dead path, a failing verifier.
func Main(args []string, w io.Writer, set okf.Rules) int {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(w)
	stamp := fs.Bool("stamp", false, "write the stamp and any newly recorded digests")
	by := fs.String("by", DefaultActor, "actor for the stamp, in §7's form")
	offline := fs.Bool("offline", false, "skip URL sources instead of fetching them")
	runVerifiers := fs.Bool("run-verifiers", false, "run each concept's verifier.command")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	roots := fs.Args()
	if len(roots) == 0 {
		roots = []string{"knowledge"}
	}

	opts := Options{Rules: set, Actor: *by, Now: time.Now().UTC(), Offline: *offline, RunVerifiers: *runVerifiers}
	var blocked int
	for _, root := range roots {
		verdicts, err := Verify(root, opts)
		if err != nil {
			fmt.Fprintf(w, "okf: %v\n", err)
			return 2
		}
		blocked += report(w, root, verdicts, opts, *stamp)
	}
	if blocked > 0 {
		return 1
	}
	return 0
}

func report(w io.Writer, root string, verdicts []Verdict, opts Options, write bool) int {
	var stampable, passed, blocked int
	for _, v := range verdicts {
		if !v.Stampable {
			continue
		}
		stampable++
		if !v.Passed() {
			blocked++
			for _, b := range v.Blocked {
				fmt.Fprintf(w, "%s/%s: blocked: %s\n", root, v.Rel, b)
			}
			continue
		}
		passed++
		if write {
			if err := apply(root, v, opts); err != nil {
				fmt.Fprintf(w, "%s/%s: %v\n", root, v.Rel, err)
				blocked++
				continue
			}
		}
		fmt.Fprintf(w, "%s/%s: %s (%d check(s))\n", root, v.Rel, verb(write), len(v.Proven))
	}
	fmt.Fprintf(w, "okf verify %s: %d concept(s), %d stampable, %d passed, %d blocked\n",
		root, len(verdicts), stampable, passed, blocked)
	return blocked
}

// apply writes the digests before the stamp: the stamp asserts the digests are
// on record, so a crash between the two must not leave it claiming more than
// the file can show.
func apply(root string, v Verdict, opts Options) error {
	path := filepath.Join(root, filepath.FromSlash(v.Rel))
	for res, sum := range v.Digests {
		if _, err := SetSourceDigest(path, res, sum); err != nil {
			return err
		}
	}
	_, err := Stamp(path, opts.Actor, opts.Now)
	return err
}

func verb(write bool) string {
	if write {
		return "stamped"
	}
	return "would stamp"
}
