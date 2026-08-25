# Spanner Emulator v1.5.53 versus managed Spanner INFORMATION_SCHEMA and SPANNER_SYS

**Observation date**: 2026-05-09

**Managed target**: `gcpug-public-spanner / merpay-sponsored-instance / apstndb-sampledb3`

**Emulator**: `spanner-mycli --embedded-emulator --emulator-image=gcr.io/cloud-spanner-emulator/emulator:1.5.53`

**Documentation**: https://cloud.google.com/spanner/docs/information-schema

This is a dated observation, not a current compatibility claim.

---

## Summary

|  | Emulator v1.5.53 | Managed Spanner |
|---|---|---|
| **INFORMATION_SCHEMA tables** | 28 | 47 |
| **INFORMATION_SCHEMA columns** | 195 | 300 |
| **SPANNER_SYS tables** | 0 (*) | 49 |
| **SPANNER_SYS columns** | 0 (*) | 523 |

(*) The Emulator accepts direct queries against
`SPANNER_SYS.SUPPORTED_OPTIMIZER_VERSIONS`, but does not advertise that table
through `INFORMATION_SCHEMA.COLUMNS`.

The principal changes since v1.5.52 were:

- `INFORMATION_SCHEMA.COLUMNS.ON_UPDATE_EXPRESSION` was added.
- `INFORMATION_SCHEMA.INDEXES.FILTER` was added.
- Consequently, the common-table difference from managed Spanner decreased
  from five columns to three.

---

## 1. Current INFORMATION_SCHEMA differences

### Tables present only on managed Spanner

The set of 19 tables was unchanged from v1.5.52:

- `CHANGE_STREAM_PRIVILEGES`
- `COLUMN_PRIVILEGES`
- `MODEL_PRIVILEGES`
- `ROUTINE_PRIVILEGES`
- `TABLE_PRIVILEGES`
- `ROLES`
- `ROLE_CHANGE_STREAM_GRANTS`
- `ROLE_COLUMN_GRANTS`
- `ROLE_GRANTEES`
- `ROLE_MODEL_GRANTS`
- `ROLE_ROUTINE_GRANTS`
- `ROLE_TABLE_GRANTS`
- `ROUTINES`
- `ROUTINE_OPTIONS`
- `PARAMETERS`
- `INDEX_OPTIONS`
- `PLACEMENTS`
- `PLACEMENT_OPTIONS`
- `TABLE_SYNONYMS`

### Column differences in common tables

#### `COLUMNS`

- There were no longer any missing columns.
- Column order still differed from managed Spanner.

| Emulator v1.5.53 | Managed Spanner |
|---|---|
| `... IS_GENERATED, IS_HIDDEN, GENERATION_EXPRESSION, ON_UPDATE_EXPRESSION, IS_STORED, ...` | `... IS_GENERATED, GENERATION_EXPRESSION, IS_STORED, IS_HIDDEN, ... ON_UPDATE_EXPRESSION` |

#### `INDEXES`

- `FILTER` was added in v1.5.53.
- The following columns remained present only on managed Spanner:
  - `SEARCH_PARTITION_BY` (`ARRAY<STRING(MAX)>`)
  - `SEARCH_ORDER_BY` (`ARRAY<STRING(MAX)>`)

#### `SCHEMATA`

The following column remained present only on managed Spanner:

- `PROTO_BUNDLE` (`PROTO<proto2.FileDescriptorSet>`)

---

## 2. Differences resolved since v1.5.52

### `COLUMNS.ON_UPDATE_EXPRESSION`

The previous report recorded this as missing from the Emulator. In v1.5.53 it
was present in `INFORMATION_SCHEMA.COLUMNS`.

The survey's `infoschem.TableMeta` target definition was updated accordingly,
so Emulator queries also select `ON_UPDATE_EXPRESSION`.

### `INDEXES.FILTER`

The previous report recorded this as missing from the Emulator. In v1.5.53 it
was present in `INFORMATION_SCHEMA.INDEXES`.

The survey's `infoschem.TableMeta` target definition was updated accordingly,
so Emulator queries also select `FILTER`.

---

## 3. Relationship to open issues

As of 2026-05-09, the following issues remained open, but their reported
behavior did not reproduce in the v1.5.53 observation:

- [#338](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/338):
  missing `INFORMATION_SCHEMA.COLUMNS.ON_UPDATE_EXPRESSION`.
- [#330](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/330):
  missing `INFORMATION_SCHEMA.INDEXES.FILTER`.

The following gaps remained:

- [#340](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/340):
  `INFORMATION_SCHEMA.SCHEMATA.PROTO_BUNDLE`.
- [#339](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/339)
  and [#290](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/290):
  `TABLE_SYNONYMS`.
- [#261](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/261):
  routines and the `INFORMATION_SCHEMA.ROUTINES` family.
- [#205](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/205):
  `PLACEMENTS` and `PLACEMENT_OPTIONS`.

No open issue was found during this observation for:

- `INDEXES.SEARCH_PARTITION_BY`
- `INDEXES.SEARCH_ORDER_BY`

---

## 4. Reproduction

```bash
python3 ~/.claude/skills/spanner-schema-diff/scripts/spanner_schema_diff.py \
  --source1='--embedded-emulator --emulator-image=gcr.io/cloud-spanner-emulator/emulator:1.5.53' \
  --label1='emulator-v1.5.53' \
  --source2='--project=gcpug-public-spanner --instance=merpay-sponsored-instance --database=apstndb-sampledb3' \
  --label2='real' \
  --doc-html-url='https://cloud.google.com/spanner/docs/information-schema' \
  --output=/tmp/spanner-schema-diff-20260509
```

Historical generated artifacts:

- Draft report: `/tmp/spanner-schema-diff-20260509/diff_report.md`
- Managed dump: `/tmp/spanner-schema-diff-20260509/real.csv`
- Emulator dump: `/tmp/spanner-schema-diff-20260509/emulator-v1.5.53.csv`
- Documentation extraction: `/tmp/spanner-schema-diff-20260509/doc.csv`
