# Spanner エミュレータ vs 実機 INFORMATION_SCHEMA / SPANNER_SYS 差分調査レポート

**調査日**: 2026-03-09
**実機**: `gcpug-public-spanner / merpay-sponsored-instance / apstndb-sampledb3`
**エミュレータ**: `spanner-mycli --embedded-emulator`
**ドキュメント**: https://cloud.google.com/spanner/docs/information-schema

---

## サマリ

|  | エミュレータ | 実機 |
|---|---|---|
| **INFORMATION_SCHEMA テーブル数** | 28 | 47 |
| **INFORMATION_SCHEMA カラム数** | 193 | 298 |
| **SPANNER_SYS テーブル数** | 0 (※) | 49 |
| **SPANNER_SYS カラム数** | 0 (※) | 514 |

※ エミュレータは `SPANNER_SYS.SUPPORTED_OPTIMIZER_VERSIONS` をクエリ可能だが、`INFORMATION_SCHEMA.COLUMNS` には登録されていない

---

## 1. INFORMATION_SCHEMA: テーブルレベルの差分

### 実機のみに存在するテーブル（19テーブル）

| テーブル名 | カテゴリ |
|---|---|
| `CHANGE_STREAM_PRIVILEGES` | FGAC |
| `COLUMN_PRIVILEGES` | FGAC |
| `MODEL_PRIVILEGES` | FGAC |
| `ROUTINE_PRIVILEGES` | FGAC |
| `TABLE_PRIVILEGES` | FGAC |
| `ROLES` | FGAC |
| `ROLE_CHANGE_STREAM_GRANTS` | FGAC |
| `ROLE_COLUMN_GRANTS` | FGAC |
| `ROLE_GRANTEES` | FGAC |
| `ROLE_MODEL_GRANTS` | FGAC |
| `ROLE_ROUTINE_GRANTS` | FGAC |
| `ROLE_TABLE_GRANTS` | FGAC |
| `ROUTINES` | ルーティン |
| `ROUTINE_OPTIONS` | ルーティン |
| `PARAMETERS` | ルーティン |
| `INDEX_OPTIONS` | インデックス |
| `PLACEMENTS` | 配置 |
| `PLACEMENT_OPTIONS` | 配置 |
| `TABLE_SYNONYMS` | シノニム |

エミュレータにはFine-Grained Access Control、ルーティン、配置、シノニム関連のテーブルが一切存在しない。

### 共通テーブルのカラム差分

#### `COLUMNS`

| 差分 | エミュレータ | 実機 |
|---|---|---|
| 追加カラム | - | `ON_UPDATE_EXPRESSION` (STRING(MAX)) |
| カラム順序 | `IS_GENERATED` → **`IS_HIDDEN`** → `GENERATION_EXPRESSION` → `IS_STORED` | `IS_GENERATED` → `GENERATION_EXPRESSION` → `IS_STORED` → **`IS_HIDDEN`** |

#### `INDEXES`

| 差分 | エミュレータ | 実機 |
|---|---|---|
| 追加カラム | - | `FILTER` (STRING(MAX)) |
| 追加カラム | - | `SEARCH_PARTITION_BY` (ARRAY\<STRING(MAX)\>) |
| 追加カラム | - | `SEARCH_ORDER_BY` (ARRAY\<STRING(MAX)\>) |

#### `SCHEMATA`

| 差分 | エミュレータ | 実機 |
|---|---|---|
| 追加カラム | - | `PROTO_BUNDLE` (PROTO\<proto2.FileDescriptorSet\>) |

### IS_HIDDEN について

`INFORMATION_SCHEMA` および `SPANNER_SYS` のカラムには `IS_HIDDEN = TRUE` のものは存在しない。ユーザーテーブルでは Full-Text Search 用の TOKENLIST カラム（`_Tokens` サフィックス）のみが該当する。

---

## 2. SPANNER_SYS: テーブルレベルの差分

エミュレータは `INFORMATION_SCHEMA.COLUMNS` 経由で取得できる SPANNER_SYS テーブルが **0**。実機は **49テーブル**。

### `SUPPORTED_OPTIMIZER_VERSIONS`（唯一エミュレータでもクエリ可能）

| エミュレータのカラム順序 | 実機のカラム順序 |
|---|---|
| IS_DEFAULT, RELEASE_DATE, VERSION | VERSION, RELEASE_DATE, IS_DEFAULT |

### 実機のみの SPANNER_SYS テーブル一覧（48テーブル）

| カテゴリ | テーブル |
|---|---|
| アクティブクエリ | `ACTIVE_QUERIES_SUMMARY`, `OLDEST_ACTIVE_QUERIES`, `ACTIVE_PARTITIONED_DMLS` |
| クエリ統計 | `QUERY_STATS_TOP_{MINUTE,10MINUTE,HOUR}`, `QUERY_STATS_TOTAL_{MINUTE,10MINUTE,HOUR}` |
| クエリプロファイル | `QUERY_PROFILES_TOP_{MINUTE,10MINUTE,HOUR}` |
| クエリ推奨 | `QUERY_RECOMMENDATIONS` |
| Read統計 | `READ_STATS_TOP_{MINUTE,10MINUTE,HOUR}`, `READ_STATS_TOTAL_{MINUTE,10MINUTE,HOUR}` |
| Txn統計 | `TXN_STATS_TOP_{MINUTE,10MINUTE,HOUR}`, `TXN_STATS_TOTAL_{MINUTE,10MINUTE,HOUR}` |
| ロック統計 | `LOCK_STATS_TOP_{MINUTE,10MINUTE,HOUR}`, `LOCK_STATS_TOTAL_{MINUTE,10MINUTE,HOUR}` |
| カラム操作統計 | `COLUMN_OPERATIONS_STATS_{MINUTE,10MINUTE,HOUR}` |
| テーブル操作統計 | `TABLE_OPERATIONS_STATS_{MINUTE,10MINUTE,HOUR}` |
| テーブルサイズ | `TABLE_SIZES_STATS_1HOUR`, `TABLE_SIZES_STATS_PER_LOCALITY_GROUP_1HOUR` |
| スプリット | `SPLIT_STATS_TOP_{MINUTE,10MINUTE,HOUR}`, `SPLIT_HOTNESS_STATS_TOP_MINUTE`, `USER_SPLIT_POINTS` |
| タスク/ポリシー | `TASKS`, `ROW_DELETION_POLICIES` |
| スキーマ推奨 | `SCHEMA_RECOMMENDATIONS` |
| ベクトルインデックス | `VECTOR_INDEX_METRICS_HISTORY` |

---

## 3. ドキュメント vs 実機の差分

### ドキュメントに記載がないテーブル

- `COLUMN_COLUMN_USAGE` -- エミュレータ・実機の両方に存在
- `INDEX_OPTIONS` -- 実機のみに存在

### ドキュメントに記載がないカラム

| テーブル | カラム | 型 |
|---|---|---|
| `SCHEMATA` | `EFFECTIVE_TIMESTAMP` | INT64 |
| `SCHEMATA` | `SCHEMA_OWNER` | STRING(MAX) |
| `INDEXES` | `FILTER` | STRING(MAX) |
| `INDEXES` | `SEARCH_PARTITION_BY` | ARRAY\<STRING(MAX)\> |
| `INDEXES` | `SEARCH_ORDER_BY` | ARRAY\<STRING(MAX)\> |
| `INDEX_COLUMNS` | `INDEX_TYPE` | STRING(MAX) |
| `COLUMNS` | `ON_UPDATE_EXPRESSION` | STRING(MAX) |

### 型の不一致

| テーブル | カラム | ドキュメント | 実機 |
|---|---|---|---|
| `SCHEMATA` | `PROTO_BUNDLE` | STRING | PROTO\<proto2.FileDescriptorSet\> |
| `COLUMNS` | `IS_HIDDEN` | STRING | BOOL |
| `INDEXES` | `INDEX_STATE` | STRING | STRING(100) |
| `TABLES` | `TABLE_TYPE` | STRING | STRING(32) |

### カラム構成の不一致: `TABLE_SYNONYMS`

| ドキュメント | 実機 |
|---|---|
| TABLE_CATALOG, TABLE_SCHEMA, TABLE_NAME, SYNONYM_CATALOG, SYNONYM_SCHEMA, **SYNONYM_TABLE_NAME** | SYNONYM_CATALOG, SYNONYM_SCHEMA, **SYNONYM_NAME**, TABLE_CATALOG, TABLE_SCHEMA, TABLE_NAME |

カラム名が異なり (`SYNONYM_TABLE_NAME` vs `SYNONYM_NAME`)、カラム順序も異なる。

### ROLE\_\*\_GRANTS テーブルの不一致

ドキュメントには `GRANTOR` と `IS_GRANTABLE` カラムが記載されているが、実機にはこれらのカラムが存在しない:

| テーブル | ドキュメントにあるが実機にないカラム |
|---|---|
| `ROLE_TABLE_GRANTS` | `GRANTOR`, `IS_GRANTABLE` |
| `ROLE_COLUMN_GRANTS` | `GRANTOR`, `IS_GRANTABLE` |
| `ROLE_MODEL_GRANTS` | `GRANTOR`, `IS_GRANTABLE` |
| `ROLE_ROUTINE_GRANTS` | `GRANTOR`, `IS_GRANTABLE` |

---

## 4. 調査方法

### 使用ツール

- `spanner-mycli` (embedded emulator / real Spanner 接続)
- `curl` (ドキュメント HTML 取得)
- Python 3 `html.parser` (ドキュメント DOM パース)

### データ取得クエリ

エミュレータ・実機ともに以下の1クエリでバルク取得:

```sql
SELECT TABLE_SCHEMA, TABLE_NAME, COLUMN_NAME, ORDINAL_POSITION, SPANNER_TYPE
FROM INFORMATION_SCHEMA.COLUMNS
WHERE TABLE_SCHEMA IN ('INFORMATION_SCHEMA', 'SPANNER_SYS')
ORDER BY TABLE_SCHEMA, TABLE_NAME, ORDINAL_POSITION;
```

```bash
spanner-mycli --format=csv -e "SELECT ..."
```

### 自動化スキル

調査を自動化するスキルを作成: `~/.claude/skills/spanner-schema-diff/`

```bash
python3 ~/.claude/skills/spanner-schema-diff/scripts/spanner_schema_diff.py \
  --source1='--embedded-emulator' --label1='emulator' \
  --source2='--project=PROJECT --instance=INSTANCE --database=DATABASE' --label2='real' \
  --doc-html-url='https://cloud.google.com/spanner/docs/information-schema' \
  --output=/tmp/spanner-schema-diff
```

### 関連 Issue

- [apstndb/spanner-mycli#554](https://github.com/apstndb/spanner-mycli/issues/554) -- JSONL 出力フォーマットの追加提案
