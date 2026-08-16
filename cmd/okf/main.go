// Command okf checks OKF bundles. Conformance failures exit 1; warnings do not.
package main

import (
	"os"

	"github.com/fairyhunter13/okf"
)

func main() {
	os.Exit(okf.Main(os.Args[1:], os.Stderr))
}
