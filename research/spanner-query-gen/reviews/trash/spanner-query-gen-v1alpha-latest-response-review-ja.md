# `spanner-query-gen` v1alpha 最新版への再確認フィードバック

作成日: 2026-05-05

## 前提

今回も、共有されているのは README / DESIGN / cleanup plan / JSON Schema / 回答案であり、実装コード・実行ログ・テスト出力は共有されていない前提でレビューします。したがって、ここでの判断は「仕様として読みやすいか」「public YAML contract として矛盾がないか」「JSON Schema が editor / CI 用契約として十分か」に限定します。

## 総評

最新版はかなり収束しています。特に良い点は以下です。

- `version: v1alpha` の lifecycle が明確になった。
- public YAML surface が `version / go / emit / catalogs / queries / writes / rules.suppressions` に絞られた。
- README の first example が Spanner-only で、BigQuery / `EXTERNAL_QUERY` / external dataset / suppressions が後段に分離された。
- `config-schema` が public contract として位置づけられ、schema と checked-in file を同期する方針になった。
- Query DTOs / SQL constants は baseline output、`emit` は additional runtime helper surfaces、という責務分離が明文化された。
- `operation: upsert` で mutation helper は Spanner `InsertOrUpdate` を使う、ただし branch-different column set は許さない、という以前の疑問への回答が入った。
- `replace` mutation-only、raw SQL `EXTERNAL_QUERY(...)` reject、external dataset は catalog binding、という境界が維持されている。

大きな設計転換は不要です。次は、残っている optional/future-looking fields を schema 上どこまで strict にするか、または v1alpha からさらに削るか、という最終整理の段階だと思います。

## P0: `conflict.strategy` は「残すなら必須」、simple にするなら YAML から削除

開発側の回答では、`conflict.strategy` は v1alpha に残す方針です。将来の `ON CONFLICT` に接続する拡張点として残す理由は理解できます。

ただし、現在の JSON Schema では `operation: upsert` に `conflict` がなくても valid になります。これは README / DESIGN の説明と少しずれます。README は `conflict.strategy: insert_or_update` を例示し、説明も `For conflict.strategy: insert_or_update` を前提にしています。

選択肢は 2 つです。

### A. より simple にする案

v1alpha YAML から `conflict` を外す。

```yaml
writes:
- name: UpsertSingerName
  operation: upsert
  insert:
    columns: [SingerId, FirstName, LastName]
  update:
    columns: [FirstName, LastName]
```

plan 側では正規化結果として次を出します。

```yaml
conflict:
  strategy: insert_or_update
```

現時点で strategy が 1 つしかないなら、これが最も simple です。将来 `ON CONFLICT` が入った時点で `conflict` block を public YAML に追加すれば十分です。

### B. 現方針を維持する案

`conflict.strategy` を public YAML に残すなら、schema 上も必須にする。

```yaml
operation: upsert
insert:
  columns: [SingerId, FirstName, LastName]
update:
  columns: [FirstName, LastName]
conflict:
  strategy: insert_or_update
```

この場合、以下を schema で弾くべきです。

```yaml
operation: upsert
insert:
  columns: [SingerId, FirstName]
update:
  columns: [FirstName]
# conflict がないので invalid
```

また、空の `conflict: {}` も invalid にした方がよいです。

私の好みは A です。v1alpha を simple にするなら、唯一値の strategy は config ではなく plan に置いた方が読みやすいです。ただし、開発側が「branch semantics と conflict semantics を config review で明示したい」と判断するなら B でも問題ありません。

## P0: `result.cardinality: exec` は public YAML から外した方がよい

`result.cardinality` を残す判断自体は理解できます。`one / maybe_one / many` は将来 query method semantics と自然につながります。

ただし、`exec` は v1alpha の現在の public surface では不自然です。

理由は、README が「declared queries の baseline output は Query DTOs と SQL constants」と説明している一方、JSON Schema は `result.struct` を必須にしており、`cardinality: exec` でも result struct が必要になるからです。`exec` は行を返さない query method / SQL annotation の概念なので、現在の `queries` + `result.struct` と衝突します。

より simple には、v1alpha の public YAML enum を次に絞るのがよいです。

```yaml
result:
  cardinality: one | maybe_one | many
```

`exec` は plan model または future SQL file annotations に残せば十分です。

どうしても `exec` を残すなら、schema を分岐させる必要があります。

```text
cardinality: exec      -> result.struct forbidden or optional
cardinality != exec    -> result.struct required
```

ただ、これは v1alpha の YAML を複雑にするので、削る方を推します。

## P1: `params[].scope` は `external_query` 専用に締める

README は、通常の `sql` / `table` / `index` query では `params[].scope` を省略し、`external_query` では `inner` / `outer` を使える、と説明しています。

一方で、JSON Schema 上は `kind: table` や `kind: sql` の `params` にも `scope: inner` / `outer` を書けます。planner で reject できるなら安全ですが、`config-schema` を editor / CI contract として出すなら、ここも schema に寄せる価値があります。

simple な方針は次です。

- `kind: external_query` の `params` だけ `scope` を許可する。
- `kind: sql` / `table` / `index` の `params` では `scope` を forbidden にする。
- `external_query` でも `scope` は必要なときだけ optional のままでよい。

これにより、`params` が method signature customization ではなく analyzer input である、という説明とも一致します。

## P1: `all_non_key_columns: false` を schema で受け入れない

現在の schema だと、概念上は次のような config が通り得ます。

```yaml
update:
  columns: [FirstName]
  all_non_key_columns: false
```

しかし README / DESIGN の意味論では、`all_non_key_columns` は「明示的な broad update opt-in」です。したがって `false` は config として意味を持たせない方がよいです。

推奨は、`all_non_key_columns` を boolean ではなく const true に寄せることです。

```json
"all_non_key_columns": { "const": true }
```

さらに、`update.columns` と `update.all_non_key_columns` は同時指定不可にするのが自然です。現在も `true` との同時指定は oneOf で弾けるように見えますが、`false` との同時指定は曖昧さが残ります。

## P1: array / string の基本制約を schema に足す

v1alpha schema はかなり強くなっていますが、editor / CI 用としては、以下の syntactic constraint を足すとさらに良いです。

- `insert.columns`, `update.columns`, `key`, `key_prefix`, `result.required` は `minItems: 1`。
- column list 系は `uniqueItems: true`。
- `queries[].params` は parameter name の重複を planner で弾くとしても、少なくとも `name` / `type` は `minLength: 1`。
- `name`, `catalog`, `table`, `input`, `rule`, `scope`, `reason`, `owner` などは `minLength: 1`。
- `rules.suppressions[].reason` は空文字を許さない。

DDL に依存する column existence や primary key inference は planner の責務でよいですが、空配列・空文字・重複 column list は schema 側で弾いても過剰ではありません。

## P1: catalog kind の discriminated union も検討する

現在の schema では、`kind: spanner` catalog に BigQuery catalog binding を書ける形に見えます。

```yaml
catalogs:
- name: app
  kind: spanner
  bindings:
    external_query_connections: ...
```

README / DESIGN 上は、`external_query_connections` と `spanner_external_datasets` は BigQuery catalog bindings です。そのため、schema の `catalog` も discriminated union にしてよいと思います。

例:

```text
kind: spanner
  allowed: ddl, proto_descriptors, dialect
  forbidden or planner-error: project, bindings

kind: bigquery
  allowed: ddl, project, bindings, dialect
  forbidden or planner-error: proto_descriptors
```

ここは planner validation でも問題ありません。ただし README が `config-schema` を public field set plus kind/operation syntax と位置付けているため、query/write だけでなく catalog kind も schema に寄せると、contract と実態がより揃います。

## P1: `conflict.strategy` を残すなら schema も説明も `upsert-only` にする

`conflict` は将来の `ON CONFLICT` のために残す、という説明は良いです。ただし、現在は `insert` / `update` / `replace` / `delete` では `conflict` を forbidden にしている一方、schema の `conflict` object 自体は operation-independent properties にあります。

この構造自体は問題ありませんが、README には一文あるとよいです。

```text
In v1alpha, conflict is only valid for operation: upsert.
The only supported strategy is insert_or_update.
```

さらに simple にするなら、前述の通り `conflict` は public YAML から外し、plan annotation にするのが一番明快です。

## P2: `order_by` default は良いが、plan に必ず出す

`table` / `index` shorthand が default で deterministic `ORDER BY` を持つ方針は良いです。README でも、Spanner は `ORDER BY` なしでは result order を保証しないため、生成 shorthand は `order_by: key` を default にする、と説明されています。

ただし、これは performance / optimizer behavior に関わるため、`explain-plan` で必ず以下を見せるべきです。

```yaml
ordering:
  configured: omitted
  effective: key
  generated_order_by:
  - SingerId
  reason: deterministic_generated_shorthand
```

`order_by: none` の場合も、明示 opt-out として plan に残すと review しやすいです。

```yaml
ordering:
  configured: none
  effective: none
  warning: deterministic_order_disabled
```

## P2: `config-schema --output yaml` の表現に注意する

README は「schema は canonical JSON Schema で、`--output yaml` は同じ schema document を読みやすく YAML に render する」と説明しています。これは良いです。

ただ、checked-in schema は JSON として byte-for-byte identical にする方針なので、README では次のように書き分けると誤解が減ります。

```text
The checked-in schema is the JSON output. YAML output is for readability and is not the canonical checked-in artifact.
```

現在の説明でも大きな問題はありませんが、CI 契約としては JSON が canonical artifact であることを強調するとよいです。

## P2: external dataset は現方針でよい

external dataset は `EXTERNAL_QUERY` の別記法ではなく BigQuery catalog binding として扱う、という方針を維持してよいです。

最新版では、以下が揃っており、かなり実運用向きになっています。

- generator は dataset / connection / IAM / Terraform を作らない。
- default projection は conservative reject。
- lossy projection は `unsupported_columns: omit` / `named_schema_tables: warn_and_omit` で explicit opt-in。
- DML / metadata mutation target は planning error。
- verification evidence は本物の外部 evidence がある場合だけ config に入れ、plan では `source: external_evidence` に正規化する。
- no evidence の場合は plan に `status: not_checked` を出す。

ここはこれ以上 YAML を増やさず、追加情報は plan / audit に逃がす方針でよいと思います。

## まとめ

大枠はかなり良いです。追加で強く言うなら、次の 5 点です。

1. `conflict.strategy` は、simple にするなら YAML から削って plan annotation にする。残すなら upsert で必須にする。
2. `result.cardinality: exec` は v1alpha public YAML から外す。`one / maybe_one / many` に絞る。
3. `params[].scope` は `external_query` 専用にする。
4. `all_non_key_columns` は `const: true` にして、`false` を受け入れない。
5. `minItems`, `uniqueItems`, `minLength` を schema に足し、空値・重複を schema で早期に弾く。

この段階では、もう大きな設計レビューというより、`config-schema` を public v1alpha contract として信頼できるものに仕上げる作業です。特に `conflict.strategy` と `result.cardinality: exec` は、将来拡張を見据えたフィールドとして残すほど、v1alpha の simple さを少し削っているので、正式 v1 前にもう一度削るか締めるかを決める価値があります。
