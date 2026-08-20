# okf

A checker for [OKF v0.2](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md)
bundles — directories of markdown with YAML frontmatter.

```
go install github.com/fairyhunter13/okf/cmd/okf@v0.1.0
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

## Fleet sweep

```
okf sweep --roots ~/git/github.com/fairyhunter13,~/go/src/github.com/fairyhunter13
okf sweep --roots ~/git/github.com/fairyhunter13 --memory ~/.claude/memory --json
```

Every directory under a root holding both `.git` and a `knowledge/` sibling is a
repo, so an eleventh is picked up the day it gets a bundle and there is no
registry to drift. Per repo it reports the check verdict, field coverage,
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
