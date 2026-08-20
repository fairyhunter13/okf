package sweep

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Main runs the `okf sweep` CLI. It never fails a build: it is typed, or looped, and a red not attached to
// a change is the accumulating-advisory failure the severity split exists to
// avoid. Exit 1 is reserved for the sweep itself not running.
func Main(args []string, w io.Writer) int {
	fs := flag.NewFlagSet("sweep", flag.ContinueOnError)
	fs.SetOutput(w)
	roots := fs.String("roots", "", "comma-separated directories to scan for repos")
	memory := fs.String("memory", "", "memory home that [[memory:...]] references resolve against")
	asJSON := fs.Bool("json", false, "emit the reports as JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*roots) == "" {
		fmt.Fprintln(w, "okf sweep: --roots is required")
		return 2
	}

	reports, err := Sweep(splitRoots(*roots), *memory, time.Now().UTC())
	if err != nil {
		fmt.Fprintf(w, "okf: %v\n", err)
		return 1
	}
	writeSweep(sweepOut, reports, *asJSON)
	return 0
}

// The report is the answer, not a diagnostic. `check` writes findings to the
// stderr its caller hands it, and sweep inherited that writer -- which put
// --json where `| jq` cannot reach it. Errors still go to w.
var sweepOut io.Writer = os.Stdout

func splitRoots(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
