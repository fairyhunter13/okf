package viz

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"time"
)

// Main runs `okf viz` and returns the process exit code.
//
//	okf viz knowledge            # -> knowledge/viz.html
//	okf viz -o /tmp/x.html knowledge
//	okf viz -name "RSE" knowledge
func Main(args []string, w io.Writer) int {
	fs := flag.NewFlagSet("viz", flag.ContinueOnError)
	fs.SetOutput(w)
	out := fs.String("o", "", "output file (default <bundle>/viz.html)")
	name := fs.String("name", "", "page title (default the bundle directory name)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	root := "knowledge"
	if fs.NArg() > 0 {
		root = fs.Arg(0)
	}
	if *out == "" {
		*out = filepath.Join(root, "viz.html")
	}

	s, err := Generate(root, *out, *name, time.Now().UTC())
	if err != nil {
		fmt.Fprintf(w, "okf viz: %v\n", err)
		return 2
	}
	fmt.Fprintf(w, "%s: %d concepts, %d links, %d bytes\n", *out, s.Concepts, s.Edges, s.Bytes)
	return 0
}
