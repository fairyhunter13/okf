# okf

A checker for [OKF v0.2](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md)
bundles — directories of markdown with YAML frontmatter.

```
go install github.com/fairyhunter13/okf/cmd/okf@v0.3.0
okf check knowledge          # conformance errors exit 1
okf check -Werror knowledge  # warnings exit 1 too
```

Pin the tag, don't take `@latest`: this binary decides whether a consumer's CI is
green, and `@latest` lets that verdict change with no commit in the repo it gates.

Only §11's three rules are errors: a concept needs a non-empty `type`,
`index.md` carries no frontmatter (bar `okf_version` at the root), and `log.md`
headings are `YYYY-MM-DD`. Dangling links, bundle-absolute links, stale
`stale_after` and orphans are warnings — §11 forbids rejecting a bundle over
them, and `TestSeverityNeverEscalates` keeps it that way.

Conformance is pinned to Google's four reference bundles in `testdata/google/`
rather than to a reading of the spec: whatever the checker rejects there is a
bug in the checker.

## Packages

| Package | Holds |
|---|---|
| `okf` | the engine: §11 conformance, `Parse`, `CheckBundle`, `Rule`/`BundleRule` |
| `okf/rules` | the fleet's own invariants, which §11 forbids the engine to enforce |
| `okf/sweep` | the fleet report, which imports both |

`rules` is a package and not part of `okf` so that importing the engine gets a
conformant checker and nothing this fleet decided. It was module
`github.com/fairyhunter13/okfrules` until v0.3.0; that split bought the same
separation at the price of two pins and a sweep that could not reach the rules,
because they imported it. Pins at `okfrules@v0.2.1` and earlier keep resolving
through the module proxy.

## Repo-local rules

An invariant the spec cannot express goes in as a `Rule` instead of upstream:

```go
findings, err := okf.CheckBundle("knowledge", time.Now().UTC(), myRule)
```

Rules run on concepts only, never on `index.md` or `log.md`. To ship them in
the CLI, call `okf.Main(os.Args[1:], os.Stderr, myRule)` from a local `main`.

An invariant over the reserved files, the link graph, or any two concepts at
once needs the whole bundle, which is what `BundleRule` is for:

```go
okf.MainWith(os.Args[1:], os.Stderr, okf.Rules{
	Doc:    []okf.Rule{myRule},
	Bundle: []okf.BundleRule{myBundleRule},
})
```

Bundle findings are appended after the per-concept ones and sorted among
themselves, so adding one never moves a line the stock check already prints —
`TestStockCheckOutputIsByteIdentical` holds that.

## The fleet rules

```
go install github.com/fairyhunter13/okf/cmd/okfrules@v0.3.0
okfrules check -Werror knowledge
okfrules -strict check -Werror knowledge
```

| Rule | Refuses |
|---|---|
| `ResourceResolves` | a `resource:` naming a path that is gone |
| `TypeVocabulary` | a `type` outside the skills' table |
| `VerifiedWellFormed` | a `verified:` stamp not naming a `human:`, or older than `generated.at` |
| `StaleAfterHasAReason` | a `stale_after` with no `sources:` naming what expires |
| `IndexHeadingsAreSingularTypes` | `## Decisions` where the table says `Decision` |
| `LogFrontmatter` | a `log.md` missing `type: Log` or its `title` |
| `NoIntraBundleWikilinks` | `[[name]]` inside a bundle — link with markdown, or name the home |
| `LogVerbs` | a log entry led by something outside the five verbs |

`Standard()` is the first seven and everything they report is an error, not the
spec's advisory half: each says a concept describes something that is not there,
or is filed where nothing will find it. `Strict()` adds `LogVerbs`, which is
opt-in on a measurement — 2026-08-21, 84 offending entries across the three
bundles that fired, 57 ordinary drift since renamed, and 26 left in two bundles:
13 sentences and 13 labels the five verbs have no word for (`Refused`,
`Refutation`, `Not changed`). Rewriting dated history to fit a closed vocabulary
falsifies it. The third bundle adopted `Strict()` the day its one drift entry
was renamed, which is what tells this apart from staging.

## Fleet sweep

```
okf sweep --roots ~/git/github.com/fairyhunter13,~/go/src/github.com/fairyhunter13
okf sweep --roots ~/git/github.com/fairyhunter13 --memory ~/.claude/memory --json
```

Every directory under a root holding both `.git` and a `knowledge/` sibling is a
repo, so an eleventh is picked up the day it gets a bundle and there is no
registry to drift. The verdict is `okf` plus `rules.Standard()` — until v0.3.0
it was §11 alone, because the rules were a module importing this one. A repo's
strict or local rules still run only in its own gate, which is why every line
names the checker that repo invokes. Per repo it reports the check verdict, field coverage,
expired `stale_after`, unresolved `[[memory:…]]` references, and the gates: a
hook that runs `okf` (directly, through a symlinked script, or through `make`),
a workflow step, and every version literal found. A hook present but not
executable is reported as such rather than as a gate — git skips it silently.
More than one pin is printed as drift.

`--memory` is where `[[memory:…]]` resolves; unset, every such reference reports
unresolved rather than being skipped, because the four profile homes differ and
a verdict depending on which one ran would be no verdict.

Sweep exits 0 on findings. It is a report, not a gate: a red not attached to a
change is the accumulating advisory the severity split exists to avoid. Exit 1
means the sweep itself did not run.

## Why this repo has no `knowledge/`

Swept and refused, not overlooked. Every non-obvious decision here is already a
load-bearing comment at the site it constrains — the severity split, the §11
refusals, the parse rules — and a bundle would give each of them two edit sites,
one of which would go stale first. `okf-bootstrap`'s own test is whether the
reasons are recoverable from the code; in ~400 lines describing a spec, they are.
