# Spanner エミュレータ v1.5.53 vs 実機 INFORMATION_SCHEMA / SPANNER_SYS 差分調査レポート

**調査日**: 2026-05-09
**実機**: `gcpug-public-spanner / merpay-sponsored-instance / apstndb-sampledb3`
**エミュレータ**: `spanner-mycli --embedded-emulator --emulator-image=gcr.io/cloud-spanner-emulator/emulator:1.5.53`
**ドキュメント**: https://cloud.google.com/spanner/docs/information-schema

---

## サマリ

|  | エミュレータ v1.5.53 | 実機 |
|---|---|---|
| **INFORMATION_SCHEMA テーブル数** | 28 | 47 |
| **INFORMATION_SCHEMA カラム数** | 195 | 300 |
| **SPANNER_SYS テーブル数** | 0 (※) | 49 |
| **SPANNER_SYS カラム数** | 0 (※) | 523 |

※ エミュレータは `SPANNER_SYS.SUPPORTED_OPTIMIZER_VERSIONS` を直接クエリ可能だが、
`INFORMATION_SCHEMA.COLUMNS` には登録されていない。

v1.5.52 からの主な変化:

- `INFORMATION_SCHEMA.COLUMNS.ON_UPDATE_EXPRESSION` が **追加された**
- `INFORMATION_SCHEMA.INDEXES.FILTER` が **追加された**
- その結果、共通テーブルの実機差分は **5 カラム → 3 カラム** に減少した

---

## 1. INFORMATION_SCHEMA: 現在の差分

### 実機のみに存在するテーブル（19テーブル）

v1.5.52 から変化なし。

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

### 共通テーブルのカラム差分

#### `COLUMNS`

- **欠落カラムはなくなった**
- ただしカラム順序はまだ実機と異なる

| エミュレータ v1.5.53 | 実機 |
|---|---|
| `... IS_GENERATED, IS_HIDDEN, GENERATION_EXPRESSION, ON_UPDATE_EXPRESSION, IS_STORED, ...` | `... IS_GENERATED, GENERATION_EXPRESSION, IS_STORED, IS_HIDDEN, ... ON_UPDATE_EXPRESSION` |

#### `INDEXES`

- `FILTER` は v1.5.53 で追加された
- まだ実機のみに存在するカラム:
  - `SEARCH_PARTITION_BY` (`ARRAY<STRING(MAX)>`)
  - `SEARCH_ORDER_BY` (`ARRAY<STRING(MAX)>`)

#### `SCHEMATA`

- まだ実機のみに存在するカラム:
  - `PROTO_BUNDLE` (`PROTO<proto2.FileDescriptorSet>`)

---

## 2. v1.5.52 から解消された項目

### `COLUMNS.ON_UPDATE_EXPRESSION`

前回レポートでは emulator 欠落カラムだったが、v1.5.53 では
`INFORMATION_SCHEMA.COLUMNS` に存在することを確認した。

これに合わせて、この repo でも `infoschem.TableMeta` の target 定義を更新し、
emulator 向けクエリでも `ON_UPDATE_EXPRESSION` を取得するよう修正した。

### `INDEXES.FILTER`

前回レポートでは emulator 欠落カラムだったが、v1.5.53 では
`INFORMATION_SCHEMA.INDEXES` に存在することを確認した。

これに合わせて、この repo でも `infoschem.TableMeta` の target 定義を更新し、
emulator 向けクエリでも `FILTER` を取得するよう修正した。

---

## 3. open issue との対応

2026-05-09 時点で、次の issue は **まだ open のまま** だが、少なくとも
v1.5.53 の実測では再現しなかった。

- [#338](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/338)  
  `INFORMATION_SCHEMA.COLUMNS.ON_UPDATE_EXPRESSION` 欠落
- [#330](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/330)  
  `INFORMATION_SCHEMA.INDEXES.FILTER` 欠落

一方で、次の gap は引き続き残っている。

- [#340](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/340)  
  `INFORMATION_SCHEMA.SCHEMATA.PROTO_BUNDLE`
- [#339](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/339) / [#290](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/290)  
  `TABLE_SYNONYMS`
- [#261](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/261)  
  ルーティン本体 / `INFORMATION_SCHEMA.ROUTINES` 系
- [#205](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/205)  
  `PLACEMENTS` / `PLACEMENT_OPTIONS`

今回確認した範囲では、依然として open issue が見当たらなかったのは次。

- `INDEXES.SEARCH_PARTITION_BY`
- `INDEXES.SEARCH_ORDER_BY`

---

## 4. 実行コマンド

```bash
python3 ~/.claude/skills/spanner-schema-diff/scripts/spanner_schema_diff.py \
  --source1='--embedded-emulator --emulator-image=gcr.io/cloud-spanner-emulator/emulator:1.5.53' \
  --label1='emulator-v1.5.53' \
  --source2='--project=gcpug-public-spanner --instance=merpay-sponsored-instance --database=apstndb-sampledb3' \
  --label2='real' \
  --doc-html-url='https://cloud.google.com/spanner/docs/information-schema' \
  --output=/tmp/spanner-schema-diff-20260509
```

### 生成物

- レポート下書き: `/tmp/spanner-schema-diff-20260509/diff_report.md`
- 実機ダンプ: `/tmp/spanner-schema-diff-20260509/real.csv`
- エミュレータダンプ: `/tmp/spanner-schema-diff-20260509/emulator-v1.5.53.csv`
- ドキュメント抽出: `/tmp/spanner-schema-diff-20260509/doc.csv`
