---
type: Constraint
resource: rules/doc_rules.go
title: yaml.v3 decodes a timestamp before a rule can read it, so the frontmatter text is the only witness to how it was spelled
description: An unquoted timestamp decodes to time.Time and a quoted one to a string, and 2026-12-31 and 2026-12-31T00:00:00Z decode to the same instant — so a rule about spelling has to read Doc.FMText, and one that read only FM was wrong twice.
tags: [okf, yaml, timestamps, rules]
status: stable
generated: { by: claude/opus-5, at: 2026-08-21T14:30:00Z }
---

# Constraint

`Doc.FM` is what the YAML decoder produced. It cannot answer any question about how a value was
written, and two of this checker's rules are exactly that question:

- `2026-12-31` and `2026-12-31T00:00:00Z` decode to the same `time.Time`, so §5's requirement that
  every timestamp carry an explicit offset is invisible in `FM`.
- An unquoted timestamp decodes to `time.Time`, a quoted one to `string`. A rule reading only the
  string case never fires on the idiomatic spelling — which is how the first draft of the
  stamp-ordering rule passed everything.

`Doc.FMText` exists for this. A rule about *spelling* reads the text; a rule about *value* reads
`FM`. Mixing them is a rule that measures the decoder.

# It has cost two rules already

`staleFinding` truncated to ten characters and parsed a date, so `2026-12-31T23:00:00+07:00` read as
UTC midnight on the 31st — a 17-hour error at the only boundary the key has. It compares instants
now. Both defects arrived with
[upstream tightening §5 without a version bump](upstream-changes-the-spec-without-bumping-it.md).
