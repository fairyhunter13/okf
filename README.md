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
