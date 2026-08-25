---
type: Decision
resource: verify/verify.go
title: Drift is called only when a second fetch agrees with the first
description: Four fleet sources are issue trackers and directory listings whose HTML differs on every request. So their digests never matched twice and the check reported drift forever. Noise exactly where the check was meant to be worth having.
tags: [okf, verify, drift, digest]
status: stable
generated: { by: claude/opus-5, at: 2026-08-21T14:30:00Z }
---

# Decision

A digest is recorded, and a mismatch called drift, only after a second fetch agrees with the first.
A source that fails that test proves reachability and nothing more. That is all it had ever
proved, silently, while reporting drift on every run.

Normalization is not the fix. It reaches a GitHub blob and a docs path; it does not reach an issue
tracker that stamps its own HTML with a request id. It never will, because the page is not
promising to be stable.

The steady state still costs one request: the second fetch runs only where the answer changes what
gets written.

# Why not drop those sources

They are the concept's provenance whether or not a digest can watch them. Dropping them to keep
the check quiet would trade a real citation for a green run. A reader following the link is the
point of the key. What the bundle loses is drift detection on four sources, which it never had.

The counterpart is [verify stamps only what it proved](verify-stamps-only-what-it-proved.md): the
same principle applied to the stamp instead of the digest.
