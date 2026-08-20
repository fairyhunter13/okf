// Command okfrules is okf's check with this fleet's shared rules wired in, for
// the repos that are not Go and cannot build one of their own.
//
//	okfrules check -Werror knowledge
//	okfrules -strict check -Werror knowledge
package main

import (
	"os"

	"github.com/fairyhunter13/okf"
	"github.com/fairyhunter13/okf/rules"
)

func main() {
	set, args := selectRules(os.Args[1:])
	os.Exit(okf.MainWith(args, os.Stderr, set))
}

func selectRules(args []string) (okf.Rules, []string) {
	if len(args) > 0 && args[0] == "-strict" {
		return rules.Strict(), args[1:]
	}
	return rules.Standard(), args
}
