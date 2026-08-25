# Spanner Emulator v1.5.52 versus managed Spanner INFORMATION_SCHEMA and SPANNER_SYS

**Observation date**: 2026-04-29

**Managed target**: `gcpug-public-spanner / merpay-sponsored-instance / apstndb-sampledb3`

**Emulator**: `spanner-mycli --embedded-emulator --emulator-image=gcr.io/cloud-spanner-emulator/emulator:1.5.52`

**Documentation**: https://cloud.google.com/spanner/docs/information-schema

This is a dated observation, not a current compatibility claim.

---

## Summary

|  | Emulator v1.5.52 | Managed Spanner |
|---|---|---|
| **INFORMATION_SCHEMA tables** | 28 | 47 |
| **INFORMATION_SCHEMA columns** | 193 | 300 |
| **SPANNER_SYS tables** | 0 (*) | 49 |
| **SPANNER_SYS columns** | 0 (*) | 523 |

(*) The Emulator accepted direct queries against
`SPANNER_SYS.SUPPORTED_OPTIMIZER_VERSIONS`, but did not advertise that table
through `INFORMATION_SCHEMA.COLUMNS`.

The principal changes since the 2026-03-09 observation were:

- Managed `INFORMATION_SCHEMA` increased from 298 to 300 columns.
- Managed `SPANNER_SYS` increased from 514 to 523 columns.
- The documentation added `COLUMNS.ON_UPDATE_EXPRESSION`, so that item was no
  longer classified as undocumented.

---

## 1. INFORMATION_SCHEMA differences

### Tables present only on managed Spanner

| Table | Category |
|---|---|
| `CHANGE_STREAM_PRIVILEGES` | Fine-grained access control |
| `COLUMN_PRIVILEGES` | Fine-grained access control |
| `MODEL_PRIVILEGES` | Fine-grained access control |
| `ROUTINE_PRIVILEGES` | Fine-grained access control |
| `TABLE_PRIVILEGES` | Fine-grained access control |
| `ROLES` | Fine-grained access control |
| `ROLE_CHANGE_STREAM_GRANTS` | Fine-grained access control |
| `ROLE_COLUMN_GRANTS` | Fine-grained access control |
| `ROLE_GRANTEES` | Fine-grained access control |
| `ROLE_MODEL_GRANTS` | Fine-grained access control |
| `ROLE_ROUTINE_GRANTS` | Fine-grained access control |
| `ROLE_TABLE_GRANTS` | Fine-grained access control |
| `ROUTINES` | Routines |
| `ROUTINE_OPTIONS` | Routines |
| `PARAMETERS` | Routines |
| `INDEX_OPTIONS` | Indexes |
| `PLACEMENTS` | Placements |
| `PLACEMENT_OPTIONS` | Placements |
| `TABLE_SYNONYMS` | Synonyms |

The Emulator had none of the fine-grained access-control, routine, placement,
or synonym tables listed above.

### Column differences in common tables

#### `COLUMNS`

| Difference | Emulator | Managed Spanner |
|---|---|---|
| Additional column | - | `ON_UPDATE_EXPRESSION` (`STRING(MAX)`) |
| Column order | `IS_GENERATED` → **`IS_HIDDEN`** → `GENERATION_EXPRESSION` → `IS_STORED` | `IS_GENERATED` → `GENERATION_EXPRESSION` → `IS_STORED` → **`IS_HIDDEN`** |

#### `INDEXES`

| Difference | Emulator | Managed Spanner |
|---|---|---|
| Additional column | - | `FILTER` (`STRING(MAX)`) |
| Additional column | - | `SEARCH_PARTITION_BY` (`ARRAY<STRING(MAX)>`) |
| Additional column | - | `SEARCH_ORDER_BY` (`ARRAY<STRING(MAX)>`) |

#### `SCHEMATA`

| Difference | Emulator | Managed Spanner |
|---|---|---|
| Additional column | - | `PROTO_BUNDLE` (`PROTO<proto2.FileDescriptorSet>`) |

### `IS_HIDDEN`

No `INFORMATION_SCHEMA` or `SPANNER_SYS` column had `IS_HIDDEN = TRUE`.
Among user tables, only full-text-search `TOKENLIST` columns with the
`_Tokens` suffix had that value.

---

## 2. SPANNER_SYS differences

The Emulator advertised zero `SPANNER_SYS` tables through
`INFORMATION_SCHEMA.COLUMNS`; managed Spanner advertised 49.

### `SUPPORTED_OPTIMIZER_VERSIONS`

This was the only `SPANNER_SYS` table that the Emulator accepted when queried
directly.

| Emulator column order | Managed column order |
|---|---|
| `IS_DEFAULT, RELEASE_DATE, VERSION` | `VERSION, RELEASE_DATE, IS_DEFAULT` |

### SPANNER_SYS tables present only on managed Spanner

The remaining 48 tables were:

| Category | Tables |
|---|---|
| Active queries | `ACTIVE_QUERIES_SUMMARY`, `OLDEST_ACTIVE_QUERIES`, `ACTIVE_PARTITIONED_DMLS` |
| Query statistics | `QUERY_STATS_TOP_{MINUTE,10MINUTE,HOUR}`, `QUERY_STATS_TOTAL_{MINUTE,10MINUTE,HOUR}` |
| Query profiles | `QUERY_PROFILES_TOP_{MINUTE,10MINUTE,HOUR}` |
| Query recommendations | `QUERY_RECOMMENDATIONS` |
| Read statistics | `READ_STATS_TOP_{MINUTE,10MINUTE,HOUR}`, `READ_STATS_TOTAL_{MINUTE,10MINUTE,HOUR}` |
| Transaction statistics | `TXN_STATS_TOP_{MINUTE,10MINUTE,HOUR}`, `TXN_STATS_TOTAL_{MINUTE,10MINUTE,HOUR}` |
| Lock statistics | `LOCK_STATS_TOP_{MINUTE,10MINUTE,HOUR}`, `LOCK_STATS_TOTAL_{MINUTE,10MINUTE,HOUR}` |
| Column-operation statistics | `COLUMN_OPERATIONS_STATS_{MINUTE,10MINUTE,HOUR}` |
| Table-operation statistics | `TABLE_OPERATIONS_STATS_{MINUTE,10MINUTE,HOUR}` |
| Table sizes | `TABLE_SIZES_STATS_1HOUR`, `TABLE_SIZES_STATS_PER_LOCALITY_GROUP_1HOUR` |
| Splits | `SPLIT_STATS_TOP_{MINUTE,10MINUTE,HOUR}`, `SPLIT_HOTNESS_STATS_TOP_MINUTE`, `USER_SPLIT_POINTS` |
| Tasks and policies | `TASKS`, `ROW_DELETION_POLICIES` |
| Schema recommendations | `SCHEMA_RECOMMENDATIONS` |
| Vector indexes | `VECTOR_INDEX_METRICS_HISTORY` |

---

## 3. Documentation differences observed on managed Spanner

These statements describe the documentation snapshot read on 2026-04-29.

### Tables absent from the documentation

- `COLUMN_COLUMN_USAGE`, present on both Emulator and managed Spanner.
- `INDEX_OPTIONS`, present only on managed Spanner.

### Columns absent from the documentation

| Table | Column | Observed type |
|---|---|---|
| `SCHEMATA` | `EFFECTIVE_TIMESTAMP` | `INT64` |
| `SCHEMATA` | `SCHEMA_OWNER` | `STRING(MAX)` |
| `INDEXES` | `FILTER` | `STRING(MAX)` |
| `INDEXES` | `SEARCH_PARTITION_BY` | `ARRAY<STRING(MAX)>` |
| `INDEXES` | `SEARCH_ORDER_BY` | `ARRAY<STRING(MAX)>` |
| `INDEX_COLUMNS` | `INDEX_TYPE` | `STRING(MAX)` |
| `PARAMETERS` | `SPANNER_TYPE` | `STRING(MAX)` |
| `ROUTINES` | `SPANNER_TYPE` | `STRING(MAX)` |

`ON_UPDATE_EXPRESSION` had been undocumented in the previous report, but was
present in the official documentation by this observation.

### Type differences

| Table | Column | Documentation | Managed Spanner |
|---|---|---|---|
| `SCHEMATA` | `PROTO_BUNDLE` | `STRING` | `PROTO<proto2.FileDescriptorSet>` |
| `COLUMNS` | `IS_HIDDEN` | `STRING` | `BOOL` |
| `INDEXES` | `INDEX_STATE` | `STRING` | `STRING(100)` |
| `TABLES` | `TABLE_TYPE` | `STRING` | `STRING(32)` |

### `TABLE_SYNONYMS` column-shape difference

| Documentation | Managed Spanner |
|---|---|
| `TABLE_CATALOG, TABLE_SCHEMA, TABLE_NAME, SYNONYM_CATALOG, SYNONYM_SCHEMA, SYNONYM_TABLE_NAME` | `SYNONYM_CATALOG, SYNONYM_SCHEMA, SYNONYM_NAME, TABLE_CATALOG, TABLE_SCHEMA, TABLE_NAME` |

Both the column name (`SYNONYM_TABLE_NAME` versus `SYNONYM_NAME`) and column
order differed.

### `ROLE_*_GRANTS` column differences

The documentation listed `GRANTOR` and `IS_GRANTABLE`, but those columns were
absent from the managed target:

| Table | Documented but not observed |
|---|---|
| `ROLE_TABLE_GRANTS` | `GRANTOR`, `IS_GRANTABLE` |
| `ROLE_COLUMN_GRANTS` | `GRANTOR`, `IS_GRANTABLE` |
| `ROLE_MODEL_GRANTS` | `GRANTOR`, `IS_GRANTABLE` |
| `ROLE_ROUTINE_GRANTS` | `GRANTOR`, `IS_GRANTABLE` |

---

## 4. Cross-check against the cloud-spanner-emulator conformance test

The comparison used:

- `GoogleCloudPlatform/cloud-spanner-emulator/tests/conformance/cases/information_schema.cc`

### Consistent findings

The conformance test's `kUnsupportedTables` listed every one of the 19
`INFORMATION_SCHEMA` tables observed only on managed Spanner:

- `CHANGE_STREAM_PRIVILEGES`
- `COLUMN_PRIVILEGES`
- `INDEX_OPTIONS`
- `MODEL_PRIVILEGES`
- `PARAMETERS`
- `PLACEMENTS`
- `PLACEMENT_OPTIONS`
- `ROLES`
- `ROLE_CHANGE_STREAM_GRANTS`
- `ROLE_COLUMN_GRANTS`
- `ROLE_GRANTEES`
- `ROLE_MODEL_GRANTS`
- `ROLE_ROUTINE_GRANTS`
- `ROLE_TABLE_GRANTS`
- `ROUTINES`
- `ROUTINE_OPTIONS`
- `ROUTINE_PRIVILEGES`
- `TABLE_PRIVILEGES`
- `TABLE_SYNONYMS`

Thus, the conformance test's unsupported-table list agreed with this
observation for the managed-only table set.

### Inconsistent findings

#### `VIEWS`

The conformance test classified `VIEWS` as unsupported, but the table was
present on Emulator v1.5.52.

#### Selected `SCHEMATA` and `TABLES` columns

The conformance test's `GSQLMetaColumns` explicitly excluded:

- `SCHEMATA.EFFECTIVE_TIMESTAMP`
- `SCHEMATA.SCHEMA_OWNER`
- `TABLES.TABLE_TYPE`
- `TABLES.INTERLEAVE_TYPE`
- `TABLES.ROW_DELETION_POLICY_EXPRESSION`

All five columns were present on both Emulator and managed Spanner. At least for
v1.5.52, the exclusions appeared stale.

#### `AAC_APPROVAL_CONFIGS` and `COLUMN_PARAMETERS`

The conformance test also classified these tables as unsupported, but they did
not appear in `INFORMATION_SCHEMA.COLUMNS` on the managed sample database.
The Emulator might still have lacked them as stated by the test, but this
observation could not compare them because the managed side did not advertise
them either.

### Cross-check conclusion

The conformance test broadly represented the Emulator's unsupported table set,
but at least these assumptions appeared stale:

1. `VIEWS` was treated as unsupported.
2. Several `SCHEMATA` and `TABLES` columns were excluded as unsupported.

For that reason, the historical report used its measured CSV
(`/tmp/spanner-schema-diff-20260429/emulator-v1.5.52.csv`) rather than treating
the conformance-test list as sufficient evidence.

### Related open issues at the observation date

- [#338](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/338):
  missing `INFORMATION_SCHEMA.COLUMNS.ON_UPDATE_EXPRESSION`.
- [#330](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/330):
  missing `INFORMATION_SCHEMA.INDEXES.FILTER`.
- [#339](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/339):
  missing `INFORMATION_SCHEMA.TABLE_SYNONYMS`.
- [#290](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/290):
  table synonyms were implemented but absent from the Emulator information
  schema.
- [#261](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/261):
  UDF support. A September 2025 comment gave an early-2026 estimate; a February
  2026 report said UDF execution worked on Emulator v1.5.48, while an April
  follow-up identified missing `ROUTINES`, `ROUTINE_OPTIONS`, `PARAMETERS`,
  and `ROUTINE_PRIVILEGES` metadata.
- [#205](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/205):
  geo-partitioning, consistent with missing `CREATE PLACEMENT` and
  `PLACEMENT KEY` support.
- [#340](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/340):
  missing `INFORMATION_SCHEMA.SCHEMATA.PROTO_BUNDLE`.

No open issue was found during this observation for:

- `INDEXES.SEARCH_PARTITION_BY`
- `INDEXES.SEARCH_ORDER_BY`
- the apparently stale conformance-test exclusions for `VIEWS`,
  `SCHEMATA.EFFECTIVE_TIMESTAMP`, `SCHEMATA.SCHEMA_OWNER`,
  `TABLES.TABLE_TYPE`, `TABLES.INTERLEAVE_TYPE`, and
  `TABLES.ROW_DELETION_POLICY_EXPRESSION`.

---

## 5. Method

Tools used:

- `spanner-mycli` for embedded-Emulator and managed-Spanner access.
- `python3 ~/.claude/skills/spanner-schema-diff/scripts/spanner_schema_diff.py`.
- Raw HTML retrieval equivalent to `curl`, implemented with
  `urllib.request` inside the script.

Both environments were queried in bulk with:

```sql
SELECT TABLE_SCHEMA, TABLE_NAME, COLUMN_NAME, ORDINAL_POSITION, SPANNER_TYPE
FROM INFORMATION_SCHEMA.COLUMNS
WHERE TABLE_SCHEMA IN ('INFORMATION_SCHEMA', 'SPANNER_SYS')
ORDER BY TABLE_SCHEMA, TABLE_NAME, ORDINAL_POSITION;
```

```bash
spanner-mycli --format=csv -e "SELECT ..."
```

Because `SPANNER_SYS.SUPPORTED_OPTIMIZER_VERSIONS` was not advertised through
`INFORMATION_SCHEMA.COLUMNS`, it was queried directly when needed:

```sql
SELECT * FROM SPANNER_SYS.SUPPORTED_OPTIMIZER_VERSIONS;
```

Reproduction command:

```bash
python3 ~/.claude/skills/spanner-schema-diff/scripts/spanner_schema_diff.py \
  --source1='--embedded-emulator --emulator-image=gcr.io/cloud-spanner-emulator/emulator:1.5.52' \
  --label1='emulator-v1.5.52' \
  --source2='--project=gcpug-public-spanner --instance=merpay-sponsored-instance --database=apstndb-sampledb3' \
  --label2='real' \
  --doc-html-url='https://cloud.google.com/spanner/docs/information-schema' \
  --output=/tmp/spanner-schema-diff-20260429
```

Historical generated artifacts:

- Draft report: `/tmp/spanner-schema-diff-20260429/diff_report.md`
- Managed dump: `/tmp/spanner-schema-diff-20260429/real.csv`
- Emulator dump: `/tmp/spanner-schema-diff-20260429/emulator-v1.5.52.csv`
- Documentation extraction: `/tmp/spanner-schema-diff-20260429/doc.csv`

Related issue:

- [apstndb/spanner-mycli#554](https://github.com/apstndb/spanner-mycli/issues/554):
  proposal for JSONL output.
