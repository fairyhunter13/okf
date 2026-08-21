package verify

import (
	"strings"
	"testing"
	"time"
)

var stampAt = time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)

const noStamp = `---
type: Constraint
title: A thing
generated: { by: claude/opus-5, at: 2026-08-20T00:00:00Z }
stale_after: 2026-11-21
sources:
  - id: s
    resource: commit deadbee
---

body
`

func TestStampInsertsAtTheCanonicalPosition(t *testing.T) {
	got, changed, err := stampText(noStamp, "process:okf-verify", stampAt)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	want := `generated: { by: claude/opus-5, at: 2026-08-20T00:00:00Z }
verified:
  - { by: process:okf-verify, at: 2026-08-21T10:00:00Z }
stale_after: 2026-11-21`
	if !strings.Contains(got, want) {
		t.Fatalf("stamp landed wrong:\n%s", got)
	}
	assertOnlyAdded(t, noStamp, got, "verified:", "  - { by: process:okf-verify, at: 2026-08-21T10:00:00Z }")
}

// The fleet's one existing stamp is the bare mapping of §5.2. Appending to it
// must convert to the list without retyping the human's event.
func TestStampConvertsBareMappingAndKeepsTheHumanEventVerbatim(t *testing.T) {
	src := `---
type: Constraint
generated: { by: claude/opus-5, at: 2026-08-20T00:00:00Z }
verified: { by: human:hafiz, at: 2026-08-21T00:00:00Z }
stale_after: 2026-11-21
---

body
`
	got, changed, err := stampText(src, "process:okf-verify", stampAt)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	want := `verified:
  - { by: human:hafiz, at: 2026-08-21T00:00:00Z }
  - { by: process:okf-verify, at: 2026-08-21T10:00:00Z }
`
	if !strings.Contains(got, want) {
		t.Fatalf("bare mapping not carried over:\n%s", got)
	}
}

func TestStampMovesItsOwnEventAndNeverDuplicates(t *testing.T) {
	src := `---
type: Constraint
verified:
  - { by: human:hafiz, at: 2026-08-21T00:00:00Z }
  - { by: process:okf-verify, at: 2026-08-21T00:00:00Z }
---

body
`
	got, changed, err := stampText(src, "process:okf-verify", stampAt)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	if n := strings.Count(got, "process:okf-verify"); n != 1 {
		t.Fatalf("want 1 process event, got %d:\n%s", n, got)
	}
	if !strings.Contains(got, "- { by: process:okf-verify, at: 2026-08-21T10:00:00Z }") {
		t.Fatalf("at not moved:\n%s", got)
	}
	if !strings.Contains(got, "- { by: human:hafiz, at: 2026-08-21T00:00:00Z }") {
		t.Fatal("human event was disturbed")
	}
}

func TestStampIsANoOpWhenNothingMoves(t *testing.T) {
	src := "---\ntype: Constraint\nverified:\n  - { by: process:okf-verify, at: 2026-08-21T10:00:00Z }\n---\n\nbody\n"
	_, changed, err := stampText(src, "process:okf-verify", stampAt)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("re-stamping the same instant rewrote the file")
	}
}

// A block mapping is legal YAML the line writer cannot edit without reflowing
// it. Refusing beats a silent reformat of someone's hand-typed stamp.
func TestStampRefusesAShapeItWouldMangle(t *testing.T) {
	src := "---\ntype: Constraint\nverified:\n  by: human:hafiz\n  at: 2026-08-21T00:00:00Z\n---\n\nbody\n"
	if _, _, err := stampText(src, "process:okf-verify", stampAt); err != ErrHandStamp {
		t.Fatalf("want ErrHandStamp, got %v", err)
	}
}

func TestStampRefusesAHumanActorUnderCLAUDECODE(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")
	if err := guardHumanActor("human:hafiz"); err == nil {
		t.Fatal("an agent was allowed to write a human stamp")
	}
	if err := guardHumanActor("process:okf-verify"); err != nil {
		t.Fatalf("machine actor refused: %v", err)
	}
}

// assertOnlyAdded is the byte-preservation property: a stamp adds lines and
// changes nothing else, which is what makes the writer safe on 295 concepts.
func assertOnlyAdded(t *testing.T, before, after string, added ...string) {
	t.Helper()
	want := strings.Split(before, "\n")
	got := strings.Split(after, "\n")
	extra := map[string]int{}
	for _, a := range added {
		extra[a]++
	}
	var kept []string
	for _, l := range got {
		if extra[l] > 0 {
			extra[l]--
			continue
		}
		kept = append(kept, l)
	}
	if strings.Join(kept, "\n") != strings.Join(want, "\n") {
		t.Fatalf("bytes outside the stamp changed:\nwant %q\ngot  %q", before, strings.Join(kept, "\n"))
	}
}
