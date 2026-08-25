---
type: Decision
resource: verify/verify.go
title: A passing verify counts down to stale_after and never advances it
description: "§5.5 makes staleness a plain comparison and §5.2 makes re-verification independent of it. So a green run that moved the date would make the key say nothing until the day it says red."
tags: [okf, verify, staleness, spec]
status: stable
generated: { by: claude/opus-5, at: 2026-08-21T14:30:00Z }
---

# Decision

`stale_after` is a date the concept was written with; a passing check does not push it forward.
Inside 30 days `verify` prints `stale in N day(s)` and leaves the exit code alone.

A tool that renewed the date on every green run would make the key unfalsifiable. It would read
"fresh" continuously until the first day nobody ran the tool, which is the one day it cannot
report.
§5.5 makes staleness a comparison against the calendar and §5.2 makes re-verification a separate
event. So the two keys answer different questions and only one of them moves.

# What the calendar cannot see

A date fires on the calendar, not on the upstream page. `claude-p-isolation` cited CLI 2.1.233 while
the machine ran 2.1.237, and its `stale_after` would not have said so until November. That gap is
what the digest check exists for, and why
[verify stamps only what it proved](verify-stamps-only-what-it-proved.md) fetches rather than
trusting a date.
