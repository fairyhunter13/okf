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
go install github.com/fairyhunter13/okfrules/cmd/okfrules@v0.2.1
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

`Standard()` is the first seven. `NoIntraBundleWikilinks` joined it in v0.2.1,
when the conversion it was waiting on finished and all ten bundles measured
zero; before that it would have been enforced only in the two repos that build
their own checker, which is the half of the fleet least likely to drift.

`Strict()` adds `LogVerbs` alone, and that one is opt-in for a different reason:
two logs lead 47 entries with a sentence, and rewriting dated history to satisfy
a closed vocabulary falsifies the record. It is not staging for a conversion
that will happen.

Everything is `okf.Error`. These are not the spec's advisory half: each one says
a concept is describing something that is not there, or is filed where nothing
will find it.
