package main

import (
	"strings"
	"testing"
)

// Both verbs, because a dispatch sending everything one way would pass either
// arm alone. Neither runs a bundle: the usage lines say which CLI answered.
func TestBothVerbsReachTheirOwnCLI(t *testing.T) {
	var out strings.Builder
	if code := run([]string{"sweep"}, &out); code != 2 || !strings.Contains(out.String(), "--roots is required") {
		t.Errorf("sweep did not reach the sweep CLI: exit %d, %q", code, out.String())
	}
	out.Reset()
	if code := run([]string{"nonsense"}, &out); code != 2 || !strings.Contains(out.String(), "usage: okf check") {
		t.Errorf("check did not reach the check CLI: exit %d, %q", code, out.String())
	}
}
