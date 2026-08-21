---
type: Log
title: okf knowledge history
---

# Bundle history

## 2026-08-21

- **Creation**: eight concepts harvested from 25 commit bodies. The tool that gates every bundle on this fleet had none of its own, and the reasoning behind its rules lived in a *consumer's* bundle — readable from the repo that pins okf, invisible from a clone of okf. What was harvested: why `okfrules` stopped being a module, the one condition that moves a rule out of `Strict()`, why `verify` stamps 33 of 295 concepts and refuses the rest, why drift needs two fetches to agree, why a green run must not move `stale_after`, the YAML decoder that has now made two rules wrong, and the spec that changes without a version. Plus today's defect, whose interesting half is that a trailing slash was reachable by every user and by no test.
  Deliberately not written: the viewer's no-vendoring rule and the augment-never-shrink guard. Both are already concepts in `claude-code-workflows`, which is where the fleet's *policy* lives; copying them here would give the next change two edit sites, which is the argument this bundle makes about pins in [one module, one pin](decisions/one-module-one-pin.md).
