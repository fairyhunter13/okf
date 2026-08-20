// Command okf checks OKF bundles. Conformance failures exit 1; warnings do not.
package main

import (
	"io"
	"os"

	"github.com/fairyhunter13/okf"
	"github.com/fairyhunter13/okf/sweep"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

// The dispatch lives here rather than in okf.MainWith: sweep runs the fleet
// rules, which import okf, so the engine cannot reach them.
func run(args []string, w io.Writer) int {
	if len(args) > 0 && args[0] == "sweep" {
		return sweep.Main(args[1:], w)
	}
	return okf.Main(args, w)
}
