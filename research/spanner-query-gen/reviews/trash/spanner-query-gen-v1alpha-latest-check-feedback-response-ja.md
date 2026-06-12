# `spanner-query-gen` v1alpha latest-check feedback response

レビューありがとうございます。今回の指摘は、追加の設計論点というよりも「すでに採用した単純化方針を public contract へ反映しきる」ためのものとして扱い、反映しました。

## 反映した点

- `queries[].result.cardinality` は public v1alpha では `one | maybe_one | many` のみに固定しました。
- row-count-only DML と DML `THEN RETURN` は `queries` には混ぜず、future `commands` surface として DESIGN に残しました。
- `write.conflict` / `conflict.strategy` は public v1alpha YAML と JSON Schema から外しました。
- `operation: upsert` は public YAML では `operation: upsert` だけで表現し、resolved plan 側にだけ `conflict_strategy: insert_or_update` を normalized semantics として記録する方針にしました。
- JSON Schema に `minLength`, `minItems`, `uniqueItems` を足し、空文字・空配列・重複配列要素・重複カラムのような基本的な構文エラーを schema で弾けるようにしました。
- catalog schema を `kind: spanner` / `kind: bigquery` の discriminated union にし、Spanner catalog では `project` / `bindings`、BigQuery catalog では `proto_descriptors` を forbidden にしました。
- regular `sql` / `table` / `index` query では `params[].scope` を schema 上も禁止し、`external_query` 専用の概念として固定しました。
- README に `outer_sql` 省略時の default を明記しました。
- README と DESIGN の古い `exec` / `conflict.strategy` 表現を、future `commands` と resolved-plan annotation の説明へ整理しました。

## 追加で進めた点

- `spanner-query-gen plan-report` を追加しました。
- `plan-report` は Spanner Omni の `AnalyzeQuery` 結果を使い、`github.com/apstndb/spannerplan/plantree/reference` で human-readable な plan tree を Markdown / YAML / JSON に出力します。
- README に `plan-report --help` の実出力と、Markdown plan report の小さな出力例を追加しました。
- Spanner Omni は catalog ごとに起動せず、1 command invocation につき 1 runtime だけを lazy に起動し、catalog ごとに別 Spanner database を作る設計にしました。別 catalog の DDL 分離には別 database で十分であり、別 Omni container は不要です。
- optional Omni integration test では、生成された index query に Sort が出ないこと、違う `ORDER BY` では Sort が出ること、NULL_FILTERED index に必要な predicate を外すと失敗することを確認します。
- 同じ optional Omni integration test から `plan-report` を同一 Omni runtime に接続し、catalog ごとに別 database を開く経路も確認します。

## 現時点でまだ future work とした点

- `commands` はまだ public v1alpha config には入れていません。
- Future shape は DESIGN にだけ残し、row-count-only DML は `commands[].result.mode: row_count`、DML returning は `commands[].returning.cardinality: one|maybe_one|many` という語彙に寄せています。
- `ON CONFLICT` は、将来入れる場合に explicit conflict target / action fields として追加する方針です。現時点の `operation: upsert` には混ぜません。
- `plan-report` はまず Spanner catalog と `external_query.inner_sql` の Spanner scope を対象にしました。BigQuery federated query 全体や external dataset の runtime 検証は別 surface として扱うべきだと考えています。

## 確認したこと

- schema の public surface から `exec` と `conflict` が消えていることを CLI unit test で固定しました。
- catalog kind-specific schema、配列構文制約、regular query の `params[].scope` 禁止も unit test で固定しました。
- schema を通さない CLI 実行でも `cardinality: exec` や `write.conflict` が normalize 段で拒否されることを package test で固定しました。
- JSON Schema だけでは十分に表現しにくい catalog / query / write / binding の参照名重複も normalize 段で拒否するようにし、package test で固定しました。
- `config-schema --out schemas/spanner-query-gen.v1alpha.schema.json` で checked-in schema を再生成しました。
- `go test ./...`、command package tests、emulator integration、optional Omni integration、`golangci-lint` を確認しました。
