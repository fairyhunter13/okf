---
type: Decision
resource: rules/rules.go, okf_test.go
title: Standard() is this fleet's house style, and only CheckBundle answers §11
description: "okf.CheckBundle passes all four of the spec authors' bundles and rules.Standard() reds all four; that is correct for the rules that are ours and was a bug for the type vocabulary, which fired on §4.1's own example values."
tags: [okf, conformance, rules, spec, fleet]
status: stable
generated: { by: claude/opus-5, at: 2026-08-21T18:05:00Z }
sources:
  - id: okf-spec
    resource: https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md
    title: Open Knowledge Format v0.2
    author: team:google-cloud-data-analytics
---

# The two tiers are not two strictnesses

`okf.CheckBundle` is conformance. §11 lists what a consumer **MUST NOT** reject a bundle over —
unknown `type` values, broken cross-links, missing `index.md`, missing optional fields, unknown
keys — and `TestGoogleBundlesConform` pins that reading to the four reference bundles rather than
to ours.

`rules.Standard()` is the fleet's house style, and it errors on three of those five. That is not a
conformance claim and was never meant to be: §11 constrains a consumer judging **someone else's**
bundle, and a pre-push hook judging the repo it lives in is not that. A producer may hold itself
to more than it may impose.

# Measured, 2026-08-21

Run `Standard()` over the spec authors' own bundles and all four are red. Sorting the reds is what
makes the tier boundary an argument rather than an assertion:

- **ga4** and **crypto_bitcoin**: red *only* on `TypeVocabulary`, over `BigQuery Table` and
  `BigQuery Dataset` — values §4.1 prints as its own examples, in a vocabulary §4.1 says is not
  registered centrally. There is no house-style reading of that. It is the one rule where the
  fleet tier was simply wrong, and the fix is to admit §4.1's list; the `Constraints`-for-
  `Constraint` catch the rule exists for is untouched, and a test now pins the example values.
- **acme_retail**: red on `stale_after needs sources:` and on a `log.md` title ending
  `knowledge history`. Both are ours, neither is in the spec, and both stay. A house rule that
  reds a foreign bundle is the tier working.
- **stackoverflow**: 11 reds on footnotes labelled `[^1]`. That rule is not ours — §5.1 keys
  footnote labels to `sources[].id` and explains why positional labels misattribute silently when
  an agent reorders the list. The reference bundle is off its own spec, which is the second
  instance of what `upstream-changes-the-spec-without-bumping-it.md` records.

# The rule for a bundle from outside

Judge it with `CheckBundle`, not with `okfrules`. `cmd/okfrules` runs `Standard()` because every
bundle it is pointed at on this machine is one of ours; pointing it at someone else's and reading
the reds as defects is a category error, and two of the four above would be exactly that.
