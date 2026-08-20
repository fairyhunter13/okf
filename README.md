# okfrules

The invariants every [OKF](https://github.com/fairyhunter13/okf) bundle in this
fleet keeps and the spec deliberately does not. §11 forbids rejecting a bundle
over an unknown type, a missing key or a broken link, so a local vocabulary is
only a rule while something asserts it — two bundles drifted before anything did.

```go
okf.MainWith(os.Args[1:], os.Stderr, okfrules.Standard())
```

A repo that is not Go installs the binary instead of building a checker:

```
go install github.com/fairyhunter13/okfrules/cmd/okfrules@v0.2.0
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
| `LogVerbs` | a log entry led by something outside the four verbs |

`Standard()` is the first six: measured over all ten fleet bundles they fire
twice in total, and both are real. `Strict()` adds the last two, which need a
conversion first — 82 wikilinks in one bundle, and two logs writing a bold
summary where the vocabulary wants a verb. A rule that lands sixty reds on the
commit that turns it on is a bulk edit, not a gate, so a repo takes those on the
commit that finishes converting.

Everything is `okf.Error`. These are not the spec's advisory half: each one says
a concept is describing something that is not there, or is filed where nothing
will find it.
