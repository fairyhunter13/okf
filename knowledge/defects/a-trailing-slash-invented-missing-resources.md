---
type: Defect
resource: check.go, rules/doc_rules.go
title: A trailing slash on the bundle root invented missing resources
description: anyExists reads filepath.Dir(Root) as the repo root, so a root spelled knowledge/ resolved every repo-relative resource against the bundle itself — 12 conformance errors on one real bundle with the slash and 0 without.
tags: [okf, paths, rules, gates]
status: stable
generated: { by: claude/opus-5, at: 2026-08-21T14:30:00Z }
---

# Symptom

`okfrules -strict check ~/…/ai-cv-evaluator/knowledge/` reported 12 conformance errors, every one of
them a `resource: does not exist` naming a file that exists. The same command without the trailing
slash reported none. Same bundle, same checker, same day.

# Root cause

`anyExists` tries three bases, the first being `filepath.Dir(d.Root)` — the repo root, which is
where a `resource:` usually points. `filepath.Dir` of `x/knowledge/` is `x/knowledge`, so the repo
base and the bundle base collapse into one and the repo-relative reading disappears.

# Why nothing caught it

Every test and every gate spells its own paths, and none of them spells a trailing slash. The bug
needs a caller who types one — an audit, a shell completion, a `for` loop over `*/knowledge/` — and
the fleet's own gates never do. So it was reachable by users and unreachable by the suite: coverage
of the code path was total and coverage of the *input* was zero.

It also fails in the safe-looking direction. A checker that under-reports gets found; one that
over-reports on a path spelling gets read as a bad bundle, and the reader edits the bundle.

# The clean was half a fix

Found the same day, by running `okfrules check .` from inside a bundle: 131 errors against 13.
`filepath.Clean(".")` is `"."` and `filepath.Dir(".")` is `"."`, so cleaning the root cannot reach
this spelling — the collapse is not about the slash, it is about `Dir` being asked for a parent the
string does not carry. The first fix was written against the one spelling that had been observed,
which is how a defect gets closed at the symptom.

# What covers it now

`CheckBundleWith` and `Load` still clean the root, and `anyExists` derives the repo base from
`filepath.Abs(d.Root)` rather than from the string it was handed, which is correct for every
spelling including the two that were red. The test builds one fixture and checks it through all
three spellings; with the abs removed, the `.` arm reports the three resources that exist.
