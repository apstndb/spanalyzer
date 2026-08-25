# Spanner Emulator versus managed Spanner INFORMATION_SCHEMA and SPANNER_SYS

**Observation date**: 2026-03-09

**Managed target**: `gcpug-public-spanner / merpay-sponsored-instance / apstndb-sampledb3`

**Emulator**: `spanner-mycli --embedded-emulator`

**Documentation**: https://cloud.google.com/spanner/docs/information-schema

This is a dated observation, not a current compatibility claim.

---

## Summary

|  | Emulator | Managed Spanner |
|---|---|---|
| **INFORMATION_SCHEMA tables** | 28 | 47 |
| **INFORMATION_SCHEMA columns** | 193 | 298 |
| **SPANNER_SYS tables** | 0 (*) | 49 |
| **SPANNER_SYS columns** | 0 (*) | 514 |

(*) The Emulator accepted direct queries against
`SPANNER_SYS.SUPPORTED_OPTIMIZER_VERSIONS`, but did not advertise that table
through `INFORMATION_SCHEMA.COLUMNS`.

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

These statements describe the documentation snapshot read on 2026-03-09.

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
| `COLUMNS` | `ON_UPDATE_EXPRESSION` | `STRING(MAX)` |

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

## 4. Method

Tools used:

- `spanner-mycli` for embedded-Emulator and managed-Spanner access.
- `curl` to retrieve documentation HTML.
- Python 3 `html.parser` to parse the documentation DOM.

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

The historical automation lived at
`~/.claude/skills/spanner-schema-diff/` and was invoked as:

```bash
python3 ~/.claude/skills/spanner-schema-diff/scripts/spanner_schema_diff.py \
  --source1='--embedded-emulator' --label1='emulator' \
  --source2='--project=PROJECT --instance=INSTANCE --database=DATABASE' --label2='real' \
  --doc-html-url='https://cloud.google.com/spanner/docs/information-schema' \
  --output=/tmp/spanner-schema-diff
```

Related issue:

- [apstndb/spanner-mycli#554](https://github.com/apstndb/spanner-mycli/issues/554):
  proposal for JSONL output.
