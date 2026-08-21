package okf

import (
	"fmt"
	"io"
	"time"
)

// Main runs the `okf check` CLI and returns the process exit code. A repo with
// its own rules wires them in here rather than forking the command. `sweep` is
// dispatched by cmd/okf instead: it runs the fleet rules, which import this
// package, so the engine cannot reach them.
func Main(args []string, w io.Writer, rules ...Rule) int {
	return MainWith(args, w, Rules{Doc: rules})
}

// MainWith is [Main] with bundle-wide rules as well.
func MainWith(args []string, w io.Writer, rules Rules) int {
	if len(args) == 0 || args[0] != "check" {
		fmt.Fprintln(w, "usage: okf check [-Werror] [-against <ref>] [bundle...]")
		return 2
	}
	roots := args[1:]
	var werror bool
	var against string
parse:
	for len(roots) > 0 {
		switch {
		case roots[0] == "-Werror":
			werror, roots = true, roots[1:]
		case roots[0] == "-against" && len(roots) > 1:
			against, roots = roots[1], roots[2:]
		default:
			break parse
		}
	}
	if len(roots) == 0 {
		roots = []string{"knowledge"}
	}

	today := time.Now().UTC()
	var errs, warns int
	for _, root := range roots {
		findings, err := CheckBundleWith(root, today, rules)
		if err != nil {
			fmt.Fprintf(w, "okf: %v\n", err)
			return 2
		}
		if against != "" {
			shrunk, err := CheckAgainst(root, against)
			if err != nil {
				fmt.Fprintf(w, "okf: %v\n", err)
				return 2
			}
			findings = append(findings, shrunk...)
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
