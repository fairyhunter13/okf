---
type: Log
title: okf knowledge history
---

# Bundle history

## 2026-08-25

- **Update** — rewrote the bundle and `README.md` in shorter sentences. The three rules the fleet
  gate refuses on (5.1, 6.3 and 6.6) reach zero across the read-often files. Corpus
  `39597edbb22b7162` before, `b6e4b2ffc63b4c60` after, dictionary
  `49c777026741bf0473a201bf194c08bed3d6a6d92d85d3a0c0e50bb2500ba7ed`, `terms_skipped` false. The 6
  findings left in the repo sit outside that set, in `SPEC.md` and the reference bundles. No claim
  of ASD-STE100 conformance is made, because the approved dictionary is not redistributable.

## 2026-08-21

- **Creation**: `standard-is-a-house-tier-not-a-conformance-check.md`, plus the type vocabulary
  admits §4.1's own example values. A fleet audit ran `Standard()` over the four reference bundles
  and all four were red — two of them *only* over `BigQuery Table` and `BigQuery Dataset`, which
  §4.1 prints as examples in a vocabulary it says is not registered. That one is a bug and is
  fixed; the reds on `acme_retail` are house rules and stay; `stackoverflow`'s 11 are §5.1's keyed
  footnotes, so the reference bundle is off its own spec. `CheckBundle` passed all four throughout,
  which is the distinction the concept records.

- **Update**: the repo base is derived from `filepath.Abs(d.Root)`, not from the string the caller
  typed. Cleaning the root closed the trailing slash and nothing else: `Dir(".")` is `"."`, so
  `okfrules check .` from inside a bundle still collapsed the repo and bundle bases -- 131 errors
  against 13 on rag-search-engine. Found the same day as the slash, by an audit that happened to
  `cd` into a `knowledge/` directory. The test now runs one fixture through all three spellings.


- **Creation**: eight concepts harvested from 25 commit bodies. The tool that gates every bundle on this fleet had none of its own, and the reasoning behind its rules lived in a *consumer's* bundle — readable from the repo that pins okf, invisible from a clone of okf. What was harvested: why `okfrules` stopped being a module, the one condition that moves a rule out of `Strict()`, why `verify` stamps 33 of 295 concepts and refuses the rest, why drift needs two fetches to agree, why a green run must not move `stale_after`, the YAML decoder that has now made two rules wrong, and the spec that changes without a version. Plus today's defect, whose interesting half is that a trailing slash was reachable by every user and by no test.
  Deliberately not written: the viewer's no-vendoring rule and the augment-never-shrink guard. Both are already concepts in `claude-code-workflows`, which is where the fleet's *policy* lives; copying them here would give the next change two edit sites, which is the argument this bundle makes about pins in [one module, one pin](decisions/one-module-one-pin.md).
