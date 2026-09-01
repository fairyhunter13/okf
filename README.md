# okf

A checker for [OKF v0.2](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md)
bundles — directories of markdown with YAML frontmatter.

```
go install github.com/fairyhunter13/okf/cmd/okf@v0.6.0
okf check knowledge          # conformance errors exit 1
okf check -Werror knowledge  # warnings exit 1 too
```

Pin the tag, don't take `@latest`. This binary decides whether a consumer's CI is green, and
`@latest` lets that verdict change with no commit in the repo it gates.

Only §11's three rules are errors: a concept needs a non-empty `type`,
`index.md` carries no frontmatter (bar `okf_version` at the root), and `log.md`
headings are `YYYY-MM-DD`. Dangling links, stale `stale_after` and orphans are
warnings — §11 forbids rejecting a bundle over
them, and `TestSeverityNeverEscalates` keeps it that way.

Conformance is pinned to Google's four reference bundles in `testdata/google/` rather than to a
reading of the spec. Whatever the checker rejects there is a bug in the checker. Upstream ships no tags, so the vendored copy records the
commit it came from and a refresh diffs against that.

## Packages

| Package | Holds |
|---|---|
| `okf` | the engine: §11 conformance, `Parse`, `CheckBundle`, `Rule`/`BundleRule` |
| `okf/rules` | the fleet's own invariants, which §11 forbids the engine to enforce |
| `okf/verify` | the machine stamp: source digests, commit and path checks, the writer |
| `okf/sweep` | the fleet report, which imports both |

`rules` is a package and not part of `okf` so that importing the engine gets a
conformant checker and nothing this fleet decided. It was module
`github.com/fairyhunter13/okfrules` until v0.3.0. That split bought the same separation. The price
was two pins, plus a sweep that could not reach the rules, because they imported it. Pins at `okfrules@v0.2.1` and earlier keep resolving
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

Bundle findings are appended after the per-concept ones and sorted among themselves. So adding one
never moves a line the stock check already prints — `TestStockCheckOutputIsByteIdentical` holds
that.

## The fleet rules

```
go install github.com/fairyhunter13/okf/cmd/okfrules@v0.6.0
okfrules check -Werror knowledge
okfrules -strict check -Werror knowledge
```

| Rule | Refuses |
|---|---|
| `ResourceResolves` | a `resource:` naming a path that is gone |
| `TypeVocabulary` | a `type` outside the skills' table |
| `VerifiedWellFormed` | a `verified:` stamp whose `at` will not parse, or is older than `generated.at` |
| `StaleAfterHasAReason` | a `stale_after` with no `sources:` naming what expires |
| `IndexHeadingsAreSingularTypes` | `## Decisions` where the table says `Decision` |
| `LogFrontmatter` | a `log.md` missing `type: Log` or its `title` |
| `NoIntraBundleWikilinks` | `[[name]]` inside a bundle — link with markdown, or name the home |
| `PreferRelativeLinks` | a link written `/from-the-bundle-root` — a warning, and the one here that is |
| `ActorConvention` | a `by:` outside §7's `<producer>/<version>`, `human:`, `process:`, `team:` |
| `StatusVocabulary` | a `status` outside §5.4's `draft \| stable \| deprecated` |
| `FootnoteLabelsJoinSources` | a `[^1]` footnote where §5.1 requires a `sources[].id` |
| `SourceHasAResource` | a `sources` entry with no `resource:`, §5.1's one required key |
| `AttestedComputationHasContract` | an `Attested Computation` with no `runtime`, or with no readable computation |
| `LogVerbs` | a log entry led by something outside the five verbs |
| `TimestampsCarryAnOffset` | a date where §5 wants an instant — `2026-12-31` names a different moment in every timezone |

`Standard()` is everything but the last two, and all but `PreferRelativeLinks` are errors rather
than the spec's advisory half. Each says a concept describes something that is not there, or is
filed where nothing will find it.
`PreferRelativeLinks` was an engine warning until v0.4.0; §6.1 only recommends a shape, so the
engine had no business holding the opinion. Moving it is why the four reference bundles now exit 0
under `-Werror`.

The four spec rules arrived in v0.4.0, firing on 172, 31, 27 and 3 fleet concepts. They waited one
tag in `Strict()` while that was converted. They are `Standard()` from v0.4.1, with all ten
bundles measuring zero. That is the
promotion condition, and it is the same one `NoIntraBundleWikilinks` met.

`Strict()` adds `LogVerbs` and `TimestampsCarryAnOffset`, both opt-in on a
measurement. The timestamp rule is upstream's 2026-08-20 change to §5, which shipped with no
version bump. It fires on 30 fleet concepts across four repos and promotes at zero, like the four
spec rules did.

`LogVerbs` measured 84 offending entries on 2026-08-21 across the three bundles that fired. The
triage recorded 57 as ordinary drift, since renamed, and 26 left in two bundles: 13 sentences and 13
labels the five verbs have no word for (`Refused`, `Refutation`, `Not changed`). Rewriting dated
history to fit a closed vocabulary falsifies it. The two groups sum to 83, one entry is unaccounted
for, and the 57 are renamed — so the measurement cannot be taken again to say which figure slipped.

## Augment, never shrink

```
okf check -against HEAD knowledge
```

Compares each concept against its blob at a git ref and reports what it lost. A dropped or
reordered `#` heading, a `sources` entry gone, a `verified` event gone, a rewritten `type`,
`title` or `resource`. The reference implementation refuses the same writes at its tool boundary.
An agent here writes files directly, so a gate with git behind it is the only place left that can
see the previous version.

Findings are warnings and nothing wires the flag into a gate yet. A real deprecation does drop
things, which is why `status: deprecated` is exempt and why this waits a tag like every other
rule. A concept absent at the ref is new
and cannot have shrunk.

## The viewer

```
okf viz knowledge                 # -> knowledge/viz.html
okf viz -o /tmp/x.html -name RSE knowledge
```

One self-contained page: the concept graph laid out at generate time, each
concept's frontmatter and rendered body, trust tier, staleness, and who cites
whom. It vendors nothing: no CDN and no bundled library. A bundle is markdown in a git repo, and
its viewer must still open on a machine with no network in five years.

Nodes are concepts, so `index.md` and `log.md` are excluded, and edges come from
the same `bundleLinks` the dangling-link check uses. That resolves the
`/from-the-bundle-root` spelling, which the reference generator drops: on
`testdata/google/acme_retail` its own committed page draws 6 edges where this
draws 14.

Generate it on demand; don't commit it. Upstream commits theirs, and the commit
that fixed it records why not.

## Machine verification

```
okf verify knowledge                       # report only; writes nothing
okf verify -stamp -run-verifiers knowledge
```

§5.3 derives three trust tiers from the stamp's *actor*: no `verified` key is
unverified, a non-`human:` actor is machine-confirmed, and a `human:` one is
human-reviewed. `verify` writes the middle tier — `process:okf-verify` by
default — and it writes it only on evidence it just checked:

| Gate | What it settles |
|---|---|
| the bundle check | the document raises no error under the caller's rules |
| `sources[].resource` | a `commit <sha>` is in this history, a path exists, a URL still hashes to its recorded `digest:` |
| `verifier.command` | the repo's own test for the claim exits 0, and only under `-run-verifiers` |

A concept with nothing outward to check is **not stampable** and stays unverified. 34 of 296 fleet
concepts earn a stamp, and rubber-stamping the other 262 would cost the key its meaning. A `resource` that is a scope descriptor
(§5.1 allows one a consumer cannot follow) earns no credit and is not a failure.

`digest:` is a producer-defined key under §4.1. The first run records it, later runs compare. A
mismatch blocks the stamp and names both digests. That is the drift a `stale_after` date cannot
see, because it fires on the calendar and not on the upstream page. GitHub `blob` URLs are hashed through `raw.githubusercontent.com`, and
an extensionless docs path is tried as `.md` first. The HTML those hosts serve is an app shell
whose hash moves on every site deploy.

Normalization does not reach every host. An issue tracker or a directory listing rewrites its HTML
per request, and four of the fleet's sources do. Their digests never matched twice and reported
drift forever. So a digest is recorded, and a
mismatch called drift, only after a second fetch agrees with the first. A source
that fails that test proves reachability and nothing more, which is all it had
ever proved.

A passing run never advances `stale_after`: §5.5 makes staleness a plain date and §5.2 makes
re-verification independent of it. So a check that moved the date would be silencing the calendar
rather than answering it. Instead `verify` counts down — inside 30 days it prints `warning: stale
in N day(s)` and leaves the exit code alone. So a bundle whose CI promotes that warning learns
about the cliff a month before it falls off.

`Stamp` refuses `-by human:*` when `CLAUDECODE` is set. A human sign-off is the one tier no run
can earn. So it is the one an agent must not write, and that is a fact about who is running that
no document rule can see.

The write is a line insert at the canonical position, never a YAML round-trip. Every byte outside
the inserted block is unchanged, and an existing `human:` event is carried through verbatim.
Re-running updates the process event in place, rather than appending a second one.

## Fleet sweep

```
okf sweep --roots ~/git/github.com/fairyhunter13,~/go/src/github.com/fairyhunter13
okf sweep --roots ~/git/github.com/fairyhunter13 --memory ~/.claude/memory --json
```

Every directory under a root holding both `.git` and a `knowledge/` sibling is a repo. So an
eleventh is picked up the day it gets a bundle and there is no registry to drift. The verdict is
`okf` plus `rules.Standard()` — until v0.3.0 it was §11 alone, because the rules were a module
importing this one. A repo's strict or local rules still run only in its own gate, which is why
every line names the checker that repo invokes.

Per repo it reports the check verdict, field coverage, expired `stale_after`, unresolved
`[[memory:…]]` references, and the gates. A hook that runs `okf` (directly, through a symlinked
script, or through `make`), a workflow step, and every version literal found. A hook present but
not executable is reported as such rather than as a gate — git skips it silently. More than one
pin is printed as drift.

`--memory` is where `[[memory:…]]` resolves. Unset, every such reference reports unresolved rather
than being skipped. The four profile homes differ, and a verdict depending on which one ran would
be no verdict.

Sweep exits 0 on findings. It is a report, not a gate: a red not attached to a
change is the accumulating advisory the severity split exists to avoid. Exit 1
means the sweep itself did not run.

## Knowledge bundle

`knowledge/` is an OKF v0.2 bundle. Read the concepts that touch the task before starting; write
them back in the same commit as the code. The `okf-knowledge-bundle` skill owns how.
Gate: `go run ./cmd/okfrules -strict check -Werror ./knowledge` in CI — this repo builds the
checker, so it runs the version under test rather than a pinned install.
