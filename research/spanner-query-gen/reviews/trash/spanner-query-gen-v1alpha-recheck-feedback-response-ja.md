# `spanner-query-gen` v1alpha recheck feedback への回答

## 回答

レビューありがとうございます。今回の指摘は、機能追加よりも public contract の同期と語彙の削減を優先すべき、という点で妥当だと判断しました。README / DESIGN / config schema / cleanup plan が同じ v1alpha surface を示すように修正します。

## 採用した点

### cleanup plan の同期

採用します。

cleanup plan はまだレビュー用メモとして残しますが、current contract と矛盾しないように更新します。

- `config-schema` を CLI 一覧に含める。
- `verification_evidence.source` を config 例から外す。
- Open questions を `Resolved / Deferred` に分ける。
- raw SQL `EXTERNAL_QUERY(...)` reject、audit-only projection matrix rows、mutation-only `replace`、write-level `emit` なし、`rules.severity` future work を resolved decision として明記する。

### `emit` の責務

採用します。

v1alpha では query DTO と SQL constants は query declaration の baseline output です。`emit` は Spanner mutations / DML statements など追加 runtime helper surface を gate するものとして README / DESIGN に明記します。`emit.*.query_methods` は引き続き reserved / rejected です。

### `params` の scope

採用します。

`queries[].params` は method signature customization ではなく analyzer type input / override として残します。v1alpha の scope は次に固定します。

- regular `sql` / `table` / `index` query では `scope` を省略する。
- `external_query` では `scope: inner | outer` を指定できる。
- unscoped `external_query` param は参照箇所から scope を推定する。
- inner / outer の両方に同名 param が現れる場合は planning error にする。

実装上も `QueryCodegenParam.Scope` を追加し、inner scoped params を Spanner inner analyzer に、outer scoped params を BigQuery analyzer に渡せるようにします。

### `verification_evidence.source`

採用します。

config に書かれた evidence は常に external evidence なので、v1alpha config から `source` を外します。plan 側で `source: external_evidence` に正規化します。`live_probe` は将来の generator-owned live verification だけが plan に出せる値にします。

### `key` の意味と upsert set equality

採用します。

v1alpha writes の `key` は table primary key set として固定します。省略時は DDL から推論します。partial primary key、unique index key、future conflict target は `key` では扱いません。

`operation: upsert` + `conflict.strategy: insert_or_update` では、`set(insert.columns) == set(key) + set(update.columns)` を検証し、duplicates を拒否します。rendering は deterministic key/update order に正規化します。

### DESIGN の表現

採用します。

`Current Documented Working-Tree Surface` は外部レビューでは強すぎるため、`Documented v1alpha Contract and Claimed Support` に変更します。README / config-schema が public YAML contract であり、DESIGN の working-tree support は実装主張であることを明記します。

### 未知フィールドの扱い

追加で対応します。

config schema を public contract の中心にする以上、parser が schema 外の未知フィールドを黙って無視するとレビュー可能性が落ちます。そのため、v1alpha YAML では unsupported / unknown field を明示的に拒否します。既知の破壊的変更対象については、可能な限り `use "..."` 付きの移行エラーを返し、単なる typo や未設計フィールドは unsupported field として落とします。

## 残した点

`result.cardinality` は v1alpha schema に残します。ただし README では、v1alpha では plan metadata / future query method semantics / diagnostics 用であり、query methods が有効になるまでは DTO shape を変えないと明記します。

`source: external_evidence` / `source: user_config` / `source: live_probe` は config ではなく plan metadata として残します。これは external dataset access assumptions を `explain-plan` / `vet` でレビュー可能にするためです。

内部 Go API には legacy config / direct struct construction 用の field がまだ残ります。public v1alpha YAML と checked-in JSON Schema を current contract とし、内部構造の完全 rename は別の整理として扱います。
