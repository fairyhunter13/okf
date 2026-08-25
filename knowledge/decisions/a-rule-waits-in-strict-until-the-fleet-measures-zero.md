---
type: Decision
resource: rules/rules.go
title: A rule waits in Strict until the fleet measures zero, then promotes to Standard
description: "Strict is a staging tier with one promotion condition: every bundle in the fleet reports zero. A rule left there is enforced only in the repos that build their own checker, which are the least likely to drift."
tags: [okf, rules, tiers, fleet]
status: stable
generated: { by: claude/opus-5, at: 2026-08-21T14:30:00Z }
---

# Decision

A new fleet rule lands in `Strict()` and fires on whatever it fires on. It moves to `Standard()`
at the next tag, once every bundle measures zero. `NoIntraBundleWikilinks` set the precedent at
v0.2.1. Four spec rules — `ActorConvention`, `StatusVocabulary`, `FootnoteLabelsJoinSources`,
`AttestedComputationHasContract` — followed at v0.5.0 after shipping at 146, 32, 27 and 4 firings.

The condition is not politeness. `Strict()` is reached by the `-strict` flag and by a repo that
calls `rules.Strict()` in its own `cmd/`. The pinned majority of the fleet never sees it. So a
rule parked there is enforced in the two repos that build their own checker, and nowhere else.
Those two are the half already paying the most attention. Promotion is where a rule starts doing work.

A rule that fires is measured before it is trusted. `TimestampsCarryAnOffset` fired 18 times on
the pre-change reference bundles, and 0 on the refreshed ones. That is what says it discriminates,
rather than that it is quiet.

# One rule is not staging

`LogVerbs` stays in `Strict()` permanently. Its remaining entries are labels the five verbs have
no word for. Rewriting a dated log to fit a closed vocabulary falsifies the record. So its count
will never reach zero, and nothing is waiting on it. A staging tier that also holds one permanent
resident needs that said out loud, or its occupancy reads as a backlog.

The pin moves when a rule changes tier, which is a fleet-wide event. See [okfrules is a package,
not a module](one-module-one-pin.md) for what the pin now covers.
