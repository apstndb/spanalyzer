---
type: Research Note
title: Historical research evidence retention review, November 12
description: Compact historical API metadata and test transcripts while preserving reproduction inputs, dated observations, failures and source identities.
tags: [evidence, provenance, research, retention, reproducibility]
status: draft
sources:
  - id: policy
    resource: ../../concepts/evidence-lifecycle.md
    title: Minimum retained evidence policy
  - id: disposition
    resource: evidence/research-retention-20261112/disposition.json
    title: Original and retained hashes, transformations and replacements
  - id: verifier
    resource: evidence/research-retention-20261112/verify.py
    title: Offline comparison against immutable historical Git inputs
  - id: source
    resource: https://github.com/apstndb/spanalyzer/tree/c38773c0a6a696b2a8110f195b3124b8d974fd79/knowledge/research/runtime-verification/evidence
    title: Evidence before this retention review
---

# Historical research evidence retention review

This review retires redundant publication bytes from 44 historical artifacts,
reducing those files from 24,466,183 to 6,189,763 bytes: **18,276,420 bytes
removed** before the small disposition and verifier are added. It performs no
new historical runtime verification and changes no survey captures, analyzer
projection selectors, active hashed plan-vocabulary inputs or regression
fixtures. Observation dates and earlier failure conclusions remain unchanged.

## Alternatives and disposition

| Artifact | Alternatives considered | Retained result |
| --- | --- | --- |
| 23 GitHub API snapshots | Keep complete responses; keep only issue numbers; remove repeated transport metadata | Preserve every Issue/PR/comment body, array/page order, lifecycle state, labels and source identities. Remove repeated repository/account DTO fields and derived API links; commit patch bodies remain recoverable from the immutable source commit |
| Eight Go test event streams | Keep all stdout; keep only a green status; retain events and meaningful output | Preserve all non-output events exactly and every non-framework output, including observation-only passing tests, logged plans/results, failures and image identities. Remove only duplicate Go test framework and successful package-summary lines |
| Eleven stdout copies | Keep two extensions; use one canonical capture | Remove only byte-identical copies; the same bytes remain in the paired JSONL file |
| Two repeated release-page captures | Treat complete HTML as identical; preserve every response; compare the article | Retain the original acquisition timestamps/hashes and one canonical HTML capture. Normalized article text is identical; raw HTML bytes differ and no exact-byte identity is claimed |
| Consumed comparison and PLAN reports | Remove every large report; retain evidence consumed by checks | Keep the October 29 per-case cross-engine comparison and November 8 full/summary JSON/YAML parity fixtures because their verifiers use the detailed results |
| Official documents and source snapshots | Remove all large inputs; retain dated sources used by historical claims | Keep source snapshots needed to assess historical documentation/source differences. Their size alone does not make them routine command logs |

The [disposition](evidence/research-retention-20261112/disposition.json) lists
each original commit/path/hash and its current projection or canonical
replacement. Original bytes remain recoverable with `git show` at the pinned
commit; Git history is not rewritten. Retrieval and import receipts continue
to describe their acquisition/import events. A later retention transformation
does not turn an old receipt into a fresh observation. Updated retrieval
annotations and runtime-log receipts distinguish these boundaries; the
[artifact manifest](evidence/artifact-sha256.json) checks current retained
bytes.

## Offline verification

From the repository root, run:

```sh
python3 knowledge/research/runtime-verification/evidence/research-retention-20261112/verify.py
python3 knowledge/research/runtime-verification/evidence/omni-r3-20260926/verify.py
python3 knowledge/research/runtime-verification/evidence/omni-r4-20261004/verify.py
python3 knowledge/research/runtime-verification/evidence/managed-recheck-20261007/verify.py
python3 knowledge/research/runtime-verification/evidence/managed-recheck-20261009/verify.py
```

The first command checks every original hash against the pinned Git revision,
reapplies the exact projection rules, checks duplicate equivalence, and rejects
any selected active plan-vocabulary input. It needs that historical commit
available locally; a shallow clone may need to fetch it. The older verifiers
check the retained engine observations and failures, rather than re-executing
the service. Full maintenance logs remain in task scratch or CI artifacts.
