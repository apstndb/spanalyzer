# Spanner エミュレータ v1.5.52 vs 実機 INFORMATION_SCHEMA / SPANNER_SYS 差分調査レポート

**調査日**: 2026-04-29
**実機**: `gcpug-public-spanner / merpay-sponsored-instance / apstndb-sampledb3`
**エミュレータ**: `spanner-mycli --embedded-emulator --emulator-image=gcr.io/cloud-spanner-emulator/emulator:1.5.52`
**ドキュメント**: https://cloud.google.com/spanner/docs/information-schema

---

## サマリ

|  | エミュレータ v1.5.52 | 実機 |
|---|---|---|
| **INFORMATION_SCHEMA テーブル数** | 28 | 47 |
| **INFORMATION_SCHEMA カラム数** | 193 | 300 |
| **SPANNER_SYS テーブル数** | 0 (※) | 49 |
| **SPANNER_SYS カラム数** | 0 (※) | 523 |

※ エミュレータは `SPANNER_SYS.SUPPORTED_OPTIMIZER_VERSIONS` を直接クエリ可能だが、`INFORMATION_SCHEMA.COLUMNS` には登録されていない

前回レポート（2026-03-09）からの主な差分:

- 実機 `INFORMATION_SCHEMA` カラム数: **298 → 300**
- 実機 `SPANNER_SYS` カラム数: **514 → 523**
- ドキュメントは `COLUMNS.ON_UPDATE_EXPRESSION` を追記済みで、この項目は「ドキュメント未記載」ではなくなった

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

エミュレータには Fine-Grained Access Control、ルーティン、配置、シノニム関連のテーブルが一切存在しない。

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
| 追加カラム | - | `SEARCH_PARTITION_BY` (ARRAY<STRING(MAX)>) |
| 追加カラム | - | `SEARCH_ORDER_BY` (ARRAY<STRING(MAX)>) |

#### `SCHEMATA`

| 差分 | エミュレータ | 実機 |
|---|---|---|
| 追加カラム | - | `PROTO_BUNDLE` (PROTO<proto2.FileDescriptorSet>) |

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
| `INDEXES` | `SEARCH_PARTITION_BY` | ARRAY<STRING(MAX)> |
| `INDEXES` | `SEARCH_ORDER_BY` | ARRAY<STRING(MAX)> |
| `INDEX_COLUMNS` | `INDEX_TYPE` | STRING(MAX) |
| `PARAMETERS` | `SPANNER_TYPE` | STRING(MAX) |
| `ROUTINES` | `SPANNER_TYPE` | STRING(MAX) |

`ON_UPDATE_EXPRESSION` は前回レポート時点ではドキュメント未記載だったが、現在は公式ドキュメントに掲載されている。

### 型の不一致

| テーブル | カラム | ドキュメント | 実機 |
|---|---|---|---|
| `SCHEMATA` | `PROTO_BUNDLE` | STRING | PROTO<proto2.FileDescriptorSet> |
| `COLUMNS` | `IS_HIDDEN` | STRING | BOOL |
| `INDEXES` | `INDEX_STATE` | STRING | STRING(100) |
| `TABLES` | `TABLE_TYPE` | STRING | STRING(32) |

### カラム構成の不一致: `TABLE_SYNONYMS`

| ドキュメント | 実機 |
|---|---|
| TABLE_CATALOG, TABLE_SCHEMA, TABLE_NAME, SYNONYM_CATALOG, SYNONYM_SCHEMA, **SYNONYM_TABLE_NAME** | SYNONYM_CATALOG, SYNONYM_SCHEMA, **SYNONYM_NAME**, TABLE_CATALOG, TABLE_SCHEMA, TABLE_NAME |

カラム名が異なり (`SYNONYM_TABLE_NAME` vs `SYNONYM_NAME`)、カラム順序も異なる。

### ROLE_*_GRANTS テーブルの不一致

ドキュメントには `GRANTOR` と `IS_GRANTABLE` カラムが記載されているが、実機にはこれらのカラムが存在しない:

| テーブル | ドキュメントにあるが実機にないカラム |
|---|---|
| `ROLE_TABLE_GRANTS` | `GRANTOR`, `IS_GRANTABLE` |
| `ROLE_COLUMN_GRANTS` | `GRANTOR`, `IS_GRANTABLE` |
| `ROLE_MODEL_GRANTS` | `GRANTOR`, `IS_GRANTABLE` |
| `ROLE_ROUTINE_GRANTS` | `GRANTOR`, `IS_GRANTABLE` |

---

## 4. cloud-spanner-emulator conformance test とのクロスチェック

対象:

- `GoogleCloudPlatform/cloud-spanner-emulator/tests/conformance/cases/information_schema.cc`

### 一致している点

conformance test の `kUnsupportedTables` は、今回の実測で「実機のみに存在した
19 テーブル」を **すべて** unsupported として列挙している:

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

つまり、**実機にだけ存在する INFORMATION_SCHEMA テーブル群については、
conformance test の unsupported list と今回の差分調査は整合している**。

### ズレている点

#### `VIEWS`

conformance test は `VIEWS` を unsupported table に含めているが、実測では
エミュレータ v1.5.52 でも `VIEWS` は存在する。

#### `SCHEMATA` / `TABLES` の一部カラム

conformance test の `GSQLMetaColumns` は次のカラムを明示的に除外している:

- `SCHEMATA.EFFECTIVE_TIMESTAMP`
- `SCHEMATA.SCHEMA_OWNER`
- `TABLES.TABLE_TYPE`
- `TABLES.INTERLEAVE_TYPE`
- `TABLES.ROW_DELETION_POLICY_EXPRESSION`

しかし実測では、これらは **エミュレータ・実機の両方に存在**する。少なくとも
v1.5.52 時点では、test 側の除外条件は stale になっている可能性が高い。

#### `AAC_APPROVAL_CONFIGS` / `COLUMN_PARAMETERS`

conformance test はこれらも unsupported table に含めているが、今回使用した
実機サンプル DB (`apstndb-sampledb3`) の `INFORMATION_SCHEMA.COLUMNS` には
現れなかった。したがって:

- emulator が未対応なのは test 記述どおりかもしれない
- ただし今回の diff report のスコープでは **実機側にも出ていないため比較不能**

### まとめ

conformance test は **大筋では現在の emulator の未対応テーブル群を正しく反映**
しているが、少なくとも次の点は更新漏れの可能性がある:

1. `VIEWS` を未対応扱いしている
2. `SCHEMATA` / `TABLES` の一部カラムを未対応前提で除外している

このため、emulator の現在の `INFORMATION_SCHEMA` 能力を判断する際は、
conformance test の unsupported list をそのまま鵜呑みにせず、実測 CSV
（`/tmp/spanner-schema-diff-20260429/emulator-v1.5.52.csv`）で裏を取るのが安全。

### 関連 open issues (`GoogleCloudPlatform/cloud-spanner-emulator`)

今回の差分・クロスチェックに関連する open issue として確認できたもの:

- [#338](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/338)
  `INFORMATION_SCHEMA.COLUMNS.ON_UPDATE_EXPRESSION` が emulator に存在しない。
- [#330](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/330)
  `INFORMATION_SCHEMA.INDEXES.FILTER` が emulator に存在しない。
- [#339](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/339)
  `INFORMATION_SCHEMA.TABLE_SYNONYMS` 自体が emulator に存在しない。
- [#290](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/290)
  table synonym は実装されているが、emulator の information schema に表示されない。
- [#261](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/261)
  UDF サポート issue。コメントでは 2025-09 時点で「2026年初頭 ETA」と案内され、
  2026-02 には emulator v1.5.48 で UDF 実行自体は動く報告がある一方、
  2026-04 の follow-up で `ROUTINES` / `ROUTINE_OPTIONS` / `PARAMETERS` /
  `ROUTINE_PRIVILEGES` など `INFORMATION_SCHEMA` 側の不足が指摘されている。
- [#205](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/205)
  geo-partitioning 未対応。`CREATE PLACEMENT` / `PLACEMENT KEY` の未対応と整合する。
- [#340](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/340)
  `INFORMATION_SCHEMA.SCHEMATA.PROTO_BUNDLE` が emulator に存在しない。

一方で、今回確認した範囲では **open issue が見当たらなかった** 項目もある:

- `INDEXES.SEARCH_PARTITION_BY`
- `INDEXES.SEARCH_ORDER_BY`
- conformance test 側の stale と思われる除外
  （`VIEWS`, `SCHEMATA.EFFECTIVE_TIMESTAMP`, `SCHEMATA.SCHEMA_OWNER`,
  `TABLES.TABLE_TYPE`, `TABLES.INTERLEAVE_TYPE`,
  `TABLES.ROW_DELETION_POLICY_EXPRESSION`）

---

## 5. 調査方法

### 使用ツール

- `spanner-mycli` (embedded emulator / real Spanner 接続)
- `python3 ~/.claude/skills/spanner-schema-diff/scripts/spanner_schema_diff.py`
- `curl` 相当の raw HTML 取得（スクリプト内部で `urllib.request` を使用）

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

`SPANNER_SYS.SUPPORTED_OPTIMIZER_VERSIONS` だけは `INFORMATION_SCHEMA.COLUMNS` に出てこないため、必要に応じて別途 `SELECT * FROM SPANNER_SYS.SUPPORTED_OPTIMIZER_VERSIONS` で直接確認する。

### 実行コマンド

```bash
python3 ~/.claude/skills/spanner-schema-diff/scripts/spanner_schema_diff.py \
  --source1='--embedded-emulator --emulator-image=gcr.io/cloud-spanner-emulator/emulator:1.5.52' \
  --label1='emulator-v1.5.52' \
  --source2='--project=gcpug-public-spanner --instance=merpay-sponsored-instance --database=apstndb-sampledb3' \
  --label2='real' \
  --doc-html-url='https://cloud.google.com/spanner/docs/information-schema' \
  --output=/tmp/spanner-schema-diff-20260429
```

### 生成物

- レポート下書き: `/tmp/spanner-schema-diff-20260429/diff_report.md`
- 実機ダンプ: `/tmp/spanner-schema-diff-20260429/real.csv`
- エミュレータダンプ: `/tmp/spanner-schema-diff-20260429/emulator-v1.5.52.csv`
- ドキュメント抽出: `/tmp/spanner-schema-diff-20260429/doc.csv`

### 関連 Issue

- [apstndb/spanner-mycli#554](https://github.com/apstndb/spanner-mycli/issues/554) -- JSONL 出力フォーマットの追加提案
