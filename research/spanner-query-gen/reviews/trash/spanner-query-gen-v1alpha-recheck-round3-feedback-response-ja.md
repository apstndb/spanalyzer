# `spanner-query-gen` v1alpha recheck round2/round3 feedback への回答

## 回答

round2 と round3 の両方を確認しました。大きな設計変更は不要で、public contract と実装の同期を進める段階、という評価に同意します。

今回の方針は「simple にするが、実ユースケースを持つ機能は削りすぎない」です。そのため、`conflict.strategy` や `result.cardinality` のような将来の query/write surface と接続するフィールドは残しつつ、schema と parser が曖昧な形を許さないようにします。

## 採用した点

### JSON Schema の強化

採用します。

`config-schema` を public v1alpha contract として出す以上、単なる structural schema では弱いと判断しました。DDL 依存の意味論までは schema に押し込みませんが、次の構文レベルの制約は schema 側にも入れます。

- `queries[].kind` ごとの required / forbidden fields。
- `writes[].operation` ごとの required / forbidden fields。
- `verification_evidence` の required fields と `checked_at` の `date-time`。
- `rules.suppressions[].scope` の prefix pattern。
- `rules.suppressions[].expires` の `date` / `YYYY-MM-DD` pattern。
- root config は `catalogs` を必須にし、`queries` または `writes` の少なくとも一方を要求する。

DDL 依存の column existence、primary key inference、unsupported type projection、upsert column-set equality は引き続き planner validation の責務にします。

### Canonical enum への整理

採用します。

v1alpha schema は canonical spelling だけを示すべきなので、alias 的な enum を削ります。

- `dialect`: `googlesql | postgresql`
- `projection.unsupported_columns`: `reject | omit`
- `projection.named_schema_tables`: `reject | warn_and_omit`

内部 plan / analyzer 実装では既存の `error` 語彙が残る箇所がありますが、public v1alpha YAML では `reject` を canonical にします。silent lossy projection になる `named_schema_tables: omit` は採用しません。

### `verification.source`

採用します。

config に evidence がない場合、plan は `status: not_checked` を出しますが、`source: user_config` は出さない方針にします。config-supplied evidence は `source: external_evidence`、future live probe は `source: live_probe` です。

### Generated query ordering

採用します。

Spanner は `ORDER BY` がないと結果順序が不定なので、自動生成される Spanner `table` / `index` shorthand query は deterministic ordering をデフォルトにします。ただし ORDER BY が最適化を阻害するケースもあるため、明示 opt-out を用意します。

config は `order_by: key | none` にしました。

- omitted / `key`: deterministic key ordering。
- `table` query: table primary key。
- `index` query: index key + base table primary key, with duplicates removed。
- `none`: ORDER BY を付けない。

`primary_key` では index shorthand を表現しきれないため採用しません。`external_query.inner_sql` には generator が ORDER BY を自動追加しません。root-partitionable な Spanner execution を阻害しうるためです。

### Mutation `InsertOrUpdate`

採用します。

`operation: upsert` の mutation helper は Spanner `InsertOrUpdate` を使う方針です。ただし `InsertOrUpdate` も one written column set の upsert であり、branch-different column sets を表せる根拠にはなりません。この点は README に明記します。

## 残した点

### `conflict.strategy`

v1alpha では削らず残します。

現時点の唯一の strategy は `insert_or_update` なので冗長ではあります。ただし、将来 `ON CONFLICT` を扱う時の拡張点として意味があり、write semantics を branch と conflict に分けてレビューできる利点があります。正式 v1 前なので破壊的変更は可能ですが、ここは今削るより、schema で許容範囲を狭くして残す方がユースケースを損ないにくいと判断しました。

### `result.cardinality`

v1alpha では残します。

query methods はまだ rejected ですが、cardinality は plan annotation / future method semantics / diagnostics として有用です。README と schema description では、v1alpha では DTO shape や query method generation を変えないことを明記します。

## 追加で反映するドキュメント整理

- `config-schema` は public field set と kind/operation syntax を検証し、DDL-dependent semantics は planner が検証する、と README に明記します。
- `--stable` は volatile audit fields を落とす snapshot aid であり、v1alpha の全 plan field を固定するものではない、と明記します。
- `params[].type` は GoogleSQL type expression として parse される、と明記します。
- external dataset projection default は conservative reject、lossy projection は explicit opt-in、と README / DESIGN を同期します。
- `auto_all_non_key_columns` の旧 sentinel wording は migration note 以外から外し、`update.all_non_key_columns: true` に揃えます。

## 実装状況

作業中の実装では、上記のうち次を反映しています。

- generated Spanner table/index query の default `ORDER BY`。
- `order_by: key | none`。
- `external_query` / explicit `sql` での `order_by` rejection。
- v1alpha parser の query kind / write operation shape validation。
- JSON Schema の discriminated union と required metadata。
- checked-in `schemas/spanner-query-gen.v1alpha.schema.json` の再生成。
- README / DESIGN / cleanup plan の同期。
