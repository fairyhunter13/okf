package okf

import (
	"fmt"
	"io"
	"time"
)

// Main runs the `okf check` CLI and returns the process exit code. A repo with
// its own rules wires them in here rather than forking the command.
func Main(args []string, w io.Writer, rules ...Rule) int {
	if len(args) == 0 || args[0] != "check" {
		fmt.Fprintln(w, "usage: okf check [-Werror] [bundle...]")
		return 2
	}
	roots := args[1:]
	var werror bool
	if len(roots) > 0 && roots[0] == "-Werror" {
		werror, roots = true, roots[1:]
	}
	if len(roots) == 0 {
		roots = []string{"knowledge"}
	}

	today := time.Now().UTC()
	var errs, warns int
	for _, root := range roots {
		findings, err := CheckBundle(root, today, rules...)
		if err != nil {
			fmt.Fprintf(w, "okf: %v\n", err)
			return 2
		}
		for _, f := range findings {
			fmt.Fprintf(w, "%s/%s\n", root, f)
			if f.Sev == Error {
				errs++
			} else {
				warns++
			}
		}
	}
	if errs > 0 {
		fmt.Fprintf(w, "okf: %d conformance error(s)\n", errs)
		return 1
	}
	if werror && warns > 0 {
		fmt.Fprintf(w, "okf: %d warning(s), -Werror\n", warns)
		return 1
	}
	return 0
}
