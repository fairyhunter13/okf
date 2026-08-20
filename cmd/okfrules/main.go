// Command okfrules is okf's check with this fleet's shared rules wired in, for
// the repos that are not Go and cannot build one of their own.
//
//	okfrules check -Werror knowledge
//	okfrules -strict check -Werror knowledge
package main

import (
	"os"

	"github.com/fairyhunter13/okf"
	"github.com/fairyhunter13/okfrules"
)

func main() {
	args := os.Args[1:]
	rules := okfrules.Standard()
	if len(args) > 0 && args[0] == "-strict" {
		rules, args = okfrules.Strict(), args[1:]
	}
	os.Exit(okf.MainWith(args, os.Stderr, rules))
}
