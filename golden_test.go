package okf

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden files")

// A repo pinned to an older tag and one pinned to a newer must grade the same
// tree the same way, so a bundle rule or a new subcommand may add surface and
// never change what `check` already says. testdata/regress exists because the
// four Google bundles are conformant and print almost nothing.
func TestStockCheckOutputIsByteIdentical(t *testing.T) {
	roots, err := filepath.Glob("testdata/google/*")
	if err != nil {
		t.Fatal(err)
	}
	roots = append(roots, "testdata/regress")

	for _, root := range roots {
		if fi, err := os.Stat(root); err != nil || !fi.IsDir() {
			continue
		}
		t.Run(filepath.Base(root), func(t *testing.T) {
			var buf bytes.Buffer
			code := Main([]string{"check", "-Werror", root}, &buf)
			fmt.Fprintf(&buf, "exit: %d\n", code)

			gold := filepath.Join("testdata", "golden", filepath.Base(root)+".txt")
			if *update {
				if err := os.WriteFile(gold, buf.Bytes(), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(gold)
			if err != nil {
				t.Fatalf("%v -- run `go test -run ByteIdentical -update`", err)
			}
			if !bytes.Equal(want, buf.Bytes()) {
				t.Errorf("output moved.\n--- %s\n%s\n--- now\n%s", gold, want, buf.Bytes())
			}
		})
	}
}
