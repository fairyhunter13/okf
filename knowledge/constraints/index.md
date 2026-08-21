# Constraint

* [Upstream changes the spec without bumping its version, so the vendored sha is the only baseline](upstream-changes-the-spec-without-bumping-it.md) - §5 grew an explicit-offset requirement on 2026-08-20 with the version left at 0.2 and no tags in the repo, and we found it a day late by cloning — so testdata/google records the commit it was vendored at.
* [yaml.v3 decodes a timestamp before a rule can read it, so the frontmatter text is the only witness to how it was spelled](yaml-decodes-a-timestamp-so-the-text-is-the-only-witness.md) - An unquoted timestamp decodes to time.Time and a quoted one to a string, and 2026-12-31 and 2026-12-31T00:00:00Z decode to the same instant — so a rule about spelling has to read Doc.FMText, and one that read only FM was wrong twice.
