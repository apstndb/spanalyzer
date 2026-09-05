---
type: Research Note
title: "Catalog identity rules on Emulator and Omni, 2026-09-06"
description: "A bounded DDL and query comparison separating creation uniqueness, exact DDL references, and query lookup before a Catalog redesign."
tags: [spanner, emulator, spanner-omni, catalog, ddl]
status: draft
sources:
  - id: ddl
    resource: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language
    title: "Official Spanner DDL naming conventions and synonyms, retrieved 2026-09-06"
  - id: lexical
    resource: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/lexical
    title: "Official case sensitivity rules, retrieved 2026-09-06"
  - id: runtime
    resource: evidence/catalog-identity-20260906/runtime.log
    title: "Observed pinned Emulator and Omni probe results"
  - id: harness
    resource: evidence/catalog-identity-20260906/probe.go.txt
    title: "Exact probe source used for the retained successful run"
---

# Catalog identity rules

The current Catalog needs stronger creation conflict and missing-object
checks. Universal case folding would introduce incorrect DDL behavior.
This comparison concerns name resolution and schema semantics, not parser
acceptance or a stable compatibility promise.

## Evidence and scope

On 2026-09-06, the retained harness completed with exit status 0 against fresh,
isolated `linux/arm64` containers using these repository pins:

- Emulator `1.5.56`, manifest digest
  `sha256:5b1e3607fe8574fb04144eeabfa54120559fb01968ffe3ffc0a9a8f6776fc454`.
- Omni `2026.r2.1-beta`, manifest digest
  `sha256:ed31d9ee72eeee69cac78566eb3a6e72ee389b26234735f0ef449774cc006741`.

The source baseline was `21fd11f674f73462fbf6634ad32957b0f5cc40eb` with
uncommitted contract-validation and generator-resolution changes; Catalog and
runtime pins were unchanged. Go was `1.26.6`. Managed Spanner was not probed.
The [harness](evidence/catalog-identity-20260906/probe.go.txt) creates a table
`Singers`, synonym `Vocalists`, and view `Other`, then applies each DDL
separately before checking query lookup. Copy it to a scratch `.go` file and
run it from `tools/` to reproduce with the current repository pins; retain the
printed image identities when comparing a later run.

## Observed results

| Case | Emulator | Omni |
| --- | --- | --- |
| `ALTER TABLE Singers ADD COLUMN Age INT64` | Success | Success |
| `ALTER TABLE singers ADD COLUMN Age2 INT64` | NotFound | NotFound |
| `ALTER TABLE Vocalists ADD COLUMN Age3 INT64` | Success | NotFound |
| Create table `singers` after `Singers` | FailedPrecondition | FailedPrecondition |
| Create table `Other` after view `Other` | FailedPrecondition | FailedPrecondition |
| `DROP TABLE singers` | NotFound | NotFound |
| `DROP TABLE Missing` | NotFound | NotFound |
| `DROP TABLE IF EXISTS Missing` | Success | Success |
| Query `Singers`, `singers`, `SINGERS`, and `Vocalists` | All succeed | All succeed |

The [full log](evidence/catalog-identity-20260906/runtime.log) retains the exact
SQL, RPC codes, and errors. These are schema/name-resolution results; no syntax
rejections occurred. Evidence integrity hashes are in the
[artifact manifest](evidence/artifact-sha256.json).

## Interpretation and next scope

Official DDL documentation distinguishes creation uniqueness from reference
resolution: names differing only in case conflict at creation, DDL references
use original case, and queries use case-insensitive names. It also limits
synonyms to query/DML access, excluding DDL and schema changes. The lexical
reference separately makes protocol-buffer and enum type names case-sensitive.

Omni matches the documented synonym restriction; the pinned Emulator accepts
the tested ALTER through the synonym. This is a runtime semantic divergence,
not evidence to enable synonym-based DDL everywhere. Managed behavior remains
unverified, and no result here changes parser policy.

At the source baseline, `Catalog.applyCreateTable` permits folded duplicate
names and a table created after a same-named view; `ApplyDDL` drops table keys
unconditionally, silently succeeding for a missing exact name. In contrast,
rejecting a differently capitalized ALTER target follows the documented rule.
The next Catalog change should separate creation conflict checks, exact DDL
references, and folded query/DML lookup; honor `IF EXISTS`; and audit rename,
index, and synonym bookkeeping without silently changing public map keys.
