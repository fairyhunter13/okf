---
type: Constraint
resource: testdata/google
title: Upstream changes the spec without bumping its version, so the vendored sha is the only baseline
description: §5 grew an explicit-offset requirement on 2026-08-20 with the version left at 0.2 and no tags in the repo, and we found it a day late by cloning — so testdata/google records the commit it was vendored at.
tags: [okf, spec, upstream, vendoring]
status: stable
generated: { by: claude/opus-5, at: 2026-08-21T14:30:00Z }
sources:
  - id: okf-spec
    resource: https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md
    title: Open Knowledge Format v0.2
    author: team:google-cloud-data-analytics
---

# Constraint

`okf_version` is `"0.2"` and has been through at least one substantive change to what a conformant
document looks like. Upstream ships no tags. So the version key answers nothing about which reading
of the spec a bundle was written against, and the only baseline a refresh can diff is the commit sha
the reference bundles were copied at — recorded in `testdata/google/README.md` and updated whenever
they are re-copied.

The reference bundles pin conformance to something outside our own reading: whatever `okf check`
rejects there is a bug in the checker, not in the bundle. That only holds while the vendored copy is
identifiable, which is the whole reason the sha is written down.

# Read §11 before turning a change into a rule

The 2026-08-20 change added the offset requirement and dropped the consumer MUST from §11 in the
same commit. So it is a producer rule: a conformant consumer may not reject a bundle over it. That
distinction decides which tier a rule can ever reach, not merely which one it starts in — see
[a rule waits in Strict](../decisions/a-rule-waits-in-strict-until-the-fleet-measures-zero.md).

Finding it a day late by cloning is the more serious half. Nothing polls upstream; the fleet's
bundles now cite `SPEC.md` as a source with a digest, so the nightly drift run reports the next
change on the day it lands.
