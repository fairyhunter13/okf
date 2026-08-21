---
type: Decision
resource: rules
title: okfrules is a package, not a module, because §11 separation needs an import boundary and not a version
description: A separate module bought spec separation at the price of two pins that never moved apart and a sweep that could grade only §11; a package buys the same separation, because Go imports packages.
tags: [okf, modules, spec, sweep]
status: stable
generated: { by: claude/opus-5, at: 2026-08-21T14:30:00Z }
---

# Decision

`okf` is the spec's reading of itself: §11's three failing rules and nothing more. `okf/rules`
holds what this fleet decided on top. That separation is a promise to a consumer — importing `okf`
gets a conformant checker with no fleet opinions attached — and for three tags it was enforced by
making `okfrules` a second Go module.

Two prices. The sweep imports both, so as a module it could reach only `okf`, and a cross-repo
report about rule conformance could grade §11 alone. And every consumer carried two pins for two
versions that have never once moved apart: both released the same day, twice.

A package buys the same boundary. Go imports packages, not modules, so a consumer that imports
`okf` still never links `rules`. One module, one pin, and the sweep can now run `rules.Standard()`
and say which rule set produced its verdict.

# What the merge had to prove

That the wider verdict cost nobody a red: measured over all ten fleet bundles before and after, 0
errors both times. A merge that silently promotes rules is a fleet-wide gate change disguised as
refactoring.

`okfrules`' own history was imported rather than left behind. proxy.golang.org holds its v0.1.0
through v0.2.1 immutably, so old pins keep resolving after the repo goes; the commits were the only
thing deletion would have taken.

The tier a rule sits in is a separate decision — see
[a rule waits in Strict until the fleet measures zero](a-rule-waits-in-strict-until-the-fleet-measures-zero.md).
