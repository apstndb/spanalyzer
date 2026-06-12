# `spanner-query-gen` v1alpha latest response review への回答

## 回答

レビューありがとうございます。今回の指摘は、v1alpha の public YAML
contract をさらに単純化し、JSON Schema を editor / CI contract として
信頼できるものに仕上げるという方向で概ね採用します。

特に、正式 v1 前で破壊的変更が許される現在は、将来拡張のためだけに
public YAML に残しているフィールドを削る方を優先します。内部 plan や
future feature のための情報は、config ではなく resolved plan 側に寄せます。

## 採用する点

### `conflict.strategy`

採用します。

`operation: upsert` の strategy は現時点で `insert_or_update` しかなく、
public YAML に残すと v1alpha の simple さを削るため、config からは削る方針に
変更します。

public YAML は次の形に寄せます。

```yaml
writes:
- name: UpsertSingerName
  operation: upsert
  insert:
    columns: [SingerId, FirstName, LastName]
  update:
    columns: [FirstName, LastName]
```

resolved plan では正規化された意味論として次を出します。

```yaml
conflict:
  strategy: insert_or_update
```

将来 Spanner DML の `ON CONFLICT` を扱う段階で、必要なら public YAML に
`conflict` block を再導入します。

### `result.cardinality: exec`

採用します。

v1alpha の public YAML では `result.cardinality` を
`one | maybe_one | many` に絞ります。`exec` は行を返さない command / DML method
semantics であり、現在の `queries` が `result.struct` を要求する設計とは
衝突します。

`exec` が必要になる場合は、将来の SQL annotation、DML query method、または
plan-only concept として扱います。

### `params[].scope`

採用します。

`params[].scope` は `kind: external_query` 専用にします。

- `kind: sql`, `table`, `index`: `scope` は forbidden。
- `kind: external_query`: `scope: inner | outer` を必要に応じて許可。

`params` 自体は analyzer input として必要なので v1alpha MVP に残しますが、
scope は BigQuery outer SQL と Spanner inner SQL が分かれるケースに限定します。

### `all_non_key_columns`

採用します。

`update.all_non_key_columns` は opt-in sentinel なので、schema 上も
`const: true` にします。`false` は意味を持たせません。

また、`update.columns` と `update.all_non_key_columns` は同時指定不可にします。

### JSON Schema の基本制約

採用します。

空文字、空配列、重複 column list は public contract として早期に弾いてよいので、
schema に次を追加します。

- column list 系: `minItems: 1`, `uniqueItems: true`
- required field list: `minItems: 1`, `uniqueItems: true`
- name / catalog / table / input / rule / reason / owner など: `minLength: 1`
- `rules.suppressions[].reason`: non-empty

DDL 依存の column existence、primary key inference、upsert column-set equality は
引き続き planner validation の責務にします。

### catalog kind の discriminated union

採用します。

`catalogs[].kind` ごとに許可する field を schema でも分けます。

- `kind: spanner`: `ddl`, `proto_descriptors`, `dialect` を許可。
- `kind: bigquery`: `ddl`, `project`, `bindings`, `dialect` を許可。
- BigQuery binding は BigQuery catalog 専用。
- proto descriptors は Spanner catalog 専用。

planner validation でも同じ invariant を維持しますが、config-schema を public
contract とする以上、kind-specific shape は schema に寄せます。

### `order_by` の plan 表現

採用します。

`table` / `index` shorthand が default で deterministic ordering を持つ方針は
維持します。ただし、plan では単なる文字列ではなく、configured value と
effective value、生成された `ORDER BY` column、opt-out warning を明示します。

例:

```yaml
ordering:
  configured: omitted
  effective: key
  generated_order_by:
  - SingerId
  reason: deterministic_generated_shorthand
```

`order_by: none` も明示 opt-out として plan に残します。

### `config-schema --output yaml`

採用します。

canonical artifact は JSON Schema の JSON 出力です。YAML 出力は同じ schema
document を読みやすく表示するための render であり、checked-in artifact には
JSON を使います。この点を README でも明確にします。

### external dataset

現方針を維持します。

external dataset は BigQuery catalog binding として扱い、generator は dataset /
connection / IAM / Terraform を作りません。lossy projection は引き続き explicit
opt-in とし、verification evidence がない場合は plan に `not_checked` を出します。

## パッケージ構成について

指摘の通り、`spanner-query-gen` は root analyzer library とは性質が異なります。
生成設定、Go renderer、emulator integration test、将来の sqlc/yo 的な runtime
surface は `spanner-query-gen` 固有なので、これらの依存は root package から
切り離します。

方針は次です。

```text
root package
  Spanner DDL catalog
  GoogleSQL frontend helper
  Analyzer result conversion
  TypeSpec / shared type conversion helpers

cmd/spanner-query-gen
  CLI only
  command-package integration tests

internal/querygen/config
  v1alpha YAML parser
  schema generation
  path resolution

internal/querygen/plan
  catalog/query/write planning
  vet diagnostics
  explain-plan model

internal/querygen/rendergo
  Go struct / SQL constant / helper rendering

internal/querygen/sqlfiles
  future SQL file and annotation parsing
```

最初の no-regret step として、`spanemuboost` / `testcontainers` を使う integration
test は `cmd/spanner-query-gen` package に移しました。これにより root analyzer
package の test graph から generator-specific runtime dependencies が外れます。

次の分割は config/schema から進めるのが安全です。現状の querygen 実装は root
package の analyzer internals と Go struct rendering helper をかなり再利用している
ため、一気に全移動すると import cycle や過剰な exported API を作りやすいです。
そのため、以下の順で進めます。

1. command-package integration test dependency を隔離する。
2. v1alpha config parser / schema を `internal/querygen/config` に移す。
3. resolved plan model と vet diagnostics を `internal/querygen/plan` に移す。
4. Go rendering を `internal/querygen/rendergo` に移す。
5. root package に残す public helper は、必要なものだけ thin wrapper にする。

`spanner-analyzer --mode=go_struct` でも Go struct rendering を使うため、Go type
mapping は完全に `spanner-query-gen` 専用とは扱いません。最終的には querygen に
依存しない shared helper として root package に残すか、`internal/gostruct` のような
小さな shared package に分けるのがよいと考えています。

## 反論または保留した点

大きな反論はありません。

ただし、`params` 自体は削りません。これは query method signature のためではなく、
GoogleSQL frontend analyzer に parameter type を渡すために必要です。特に
`ARRAY` / `STRUCT` parameter や `external_query` の inner / outer scope では、
explicit parameter type がないと解析できないケースがあります。

また、DDL 依存の semantic validation を JSON Schema に押し込むことはしません。
schema は public field set と syntactic shape を固定し、catalog-dependent validation
は planner の責務にします。

## 次に実装すること

次の作業単位で進めます。

1. `conflict` を v1alpha public YAML から削除し、plan annotation に移す。
2. `result.cardinality` enum から `exec` を削除する。
3. `params[].scope` を `external_query` 専用にする。
4. `all_non_key_columns` を `const: true` にし、`false` を拒否する。
5. schema に `minItems`, `uniqueItems`, `minLength` を追加する。
6. catalog kind discriminated union を schema と parser validation に反映する。
7. plan ordering を structured output にする。
8. config/schema を `internal/querygen/config` に切り出す。
