# Decision

* [A passing verify counts down to stale_after and never advances
  it](staleness-counts-down-and-never-moves.md) - §5.5 makes staleness a plain comparison and §5.2
  makes re-verification independent of it. So a green run that moved the date would make the key
  say nothing until the day it says red.
* [A rule waits in Strict until the fleet measures zero, then promotes to
  Standard](a-rule-waits-in-strict-until-the-fleet-measures-zero.md) - Strict is a staging tier
  with one promotion condition: every bundle in the fleet reports zero. A rule left there is
  enforced only in the repos that build their own checker, which are the least likely to drift.
* [Drift is called only when a second fetch agrees with the
  first](drift-needs-a-second-fetch-to-agree.md) - Four fleet sources are issue trackers and
  directory listings whose HTML differs on every request. So their digests never matched twice and
  the check reported drift forever. Noise exactly where the check was meant to be worth having.
* [okfrules is a package, not a module, because §11 separation needs an import boundary and not a
  version](one-module-one-pin.md) - A separate module bought spec separation. The price was two pins that never moved apart, and a sweep that could grade only §11. A package buys the same
  separation, because Go imports packages.
* [Standard() is this fleet's house style, and only CheckBundle answers
  §11](standard-is-a-house-tier-not-a-conformance-check.md) - okf.CheckBundle passes all four of
  the spec authors' bundles, and rules.Standard() reds all four. That is correct for the rules
  that are ours. It was a bug for the type vocabulary, which fired on §4.1's own example values.
* [verify stamps only what it just proved, so most concepts earn no
  stamp](verify-stamps-only-what-it-proved.md) - 33 of 295 fleet concepts have something outward
  to check. Stamping the other 262 would cost the key its meaning. The forgery guard sits in the
  writer where it can see it is running under an agent.
