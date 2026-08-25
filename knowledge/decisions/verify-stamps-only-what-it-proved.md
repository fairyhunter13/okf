---
type: Decision
resource: verify
title: verify stamps only what it just proved, so most concepts earn no stamp
description: "33 of 295 fleet concepts have something outward to check. Stamping the other 262 would cost the key its meaning. The forgery guard sits in the writer where it can see it is running under an agent."
tags: [okf, verify, trust, spec]
status: stable
generated: { by: claude/opus-5, at: 2026-08-21T14:30:00Z }
---

# Decision

`okf verify -stamp` writes §5.3's machine tier: it fetches each `sources[].resource`, compares it
against the recorded `digest:`, runs the concept's `verifier:`, and stamps `process:okf-verify` on
what passed. A concept with nothing outward to check gets no stamp. That is the design and not a
gap. 33 of 295 fleet concepts qualified, and rubber-stamping the other 262 would make the key mean
"a tool ran" instead of "something was checked".

Before this the fleet was 1 of 295 verified, and that one was typed by hand. What blocked the
machine tier was a local rule demanding a `human:` actor, which §5.3 does not ask for. The spec's
own example pairs a human sign-off with `process:finance-nightly`. Actor *shape* was already
`ActorConvention`'s job, so the rule lost the demand.

# The forgery guard moved to the writer

"An agent never stamps a human tier" was prose in a skill file, enforced by nothing. It lives in the
writer now, where it can see `CLAUDECODE` in the environment and refuse. A guard in the prose of the
document that asks for the guard is the weakest possible placement.

The writer inserts lines rather than round-tripping YAML, so every byte outside the inserted block
survives. Including an existing `human:` event, through the bare-mapping to list conversion §5.2
allows.

# The stamp obeys the rules it is stamped under

A stamp can be written at 02:26Z onto a concept whose `generated.at` was rounded up to 08:00Z.
Such a stamp says the content was authored after the run that confirmed it. `VerifiedWellFormed`
fails it. The writer
refuses to produce a stamp its own checker would reject; one fleet bundle went red before it did.

What a stamp cannot see is the calendar: [staleness counts down and never
moves](staleness-counts-down-and-never-moves.md). What a digest cannot see is a page that differs
on every request, which is [drift needs a second fetch](drift-needs-a-second-fetch-to-agree.md).
