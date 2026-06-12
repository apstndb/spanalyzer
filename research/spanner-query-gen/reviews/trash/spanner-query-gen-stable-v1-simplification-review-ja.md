# `spanner-query-gen` stable v1 cleanup plan への追加レビュー

## 前提

実装コード、実行ログ、テスト出力は共有されていません。したがって、このレビューでは「実装済みかどうか」は判定せず、添付された cleanup plan / README / DESIGN の仕様整理として評価します。

結論として、今回の方向性はかなり良いです。特に、stable v1 を compact config の延長にせず、`catalogs` / `queries` / `writes` / `emit` / `rules` に整理したこと、README の最初の例を Spanner-only にして BigQuery / `EXTERNAL_QUERY` / external dataset / rules を後ろに分けたことは、以前よりかなり simple になっています。

ただし、さらに simple にするなら、**新しい仕様を増やすことより、stable v1 で残す語彙を減らすこと**を優先した方がよいです。

---

## 良い点

### 1. README の入口が軽くなった

README の first example が Spanner-only になり、BigQuery、`EXTERNAL_QUERY`、external dataset、DTO sharing、suppressions を後段に逃がしているのは非常に良いです。これにより、読者はまず「Spanner DDL + table query + update write」だけを理解すればよくなっています。

この方針は維持してください。stable v1 の canonical README example に、external dataset や rules suppression を再び混ぜない方がよいです。

### 2. `catalogs` への rename は正しい

`schemas` はもはや DDL schema だけでなく、BigQuery catalog、`EXTERNAL_QUERY` connection binding、Spanner external dataset projection まで含むため、`catalogs` の方が実態に合っています。

`source` を `catalog` にするのも良いです。`external_source` と衝突しにくくなり、query / write がどの analyzer catalog に属するかが明確になります。

### 3. `kind` discriminated union は読みやすい

`queries[].kind: sql|table|index|external_query` は、従来の `sql | table | index | federated` exactly-one top-level union より明確です。

特に `federated` を `external_query` にしたのは良い判断です。BigQuery Spanner external dataset も federation の一種なので、`federated` という名前は曖昧でした。

### 4. external dataset を query kind にしない方針は維持すべき

external dataset は BigQuery catalog binding であり、query-level function ではありません。今の `catalogs[].bindings.spanner_external_datasets` に置く方針は正しいです。

この設計により、external dataset table は普通の BigQuery SQL の relation として扱えます。一方で、projection loss、read-only target、access assumptions は plan / vet 側に出せます。

### 5. branch-aware writes は安定する

`operation: upsert` + `insert.columns` + `update.columns` + `conflict.strategy` は、`insert_or_update` や `update_mask` よりかなり良いです。

Spanner write helper の仕様として、insert branch と update branch の責務が分かれます。将来 `ON CONFLICT` を入れても、`conflict.strategy` の拡張で扱えます。

---

## まだ直した方がよい点

### P0. README と DESIGN の CLI 表現を同期する

README は stable v1 CLI として、`generate` / `check` / `explain-plan` / `vet` subcommands を前提にしています。これは良いです。

一方、DESIGN の CLI UX にはまだ、

```text
spanner-query-gen --config path.yaml
spanner-query-gen --config path.yaml --out generated.go
```

のような no-subcommand 形式と、「Future subcommands can be added」という古い表現が残っています。

stable v1 で subcommand 方式に決めるなら、DESIGN も次のように統一してください。

```text
spanner-query-gen generate --config path.yaml --out generated.go
spanner-query-gen generate --config path.yaml --out -
spanner-query-gen check --config path.yaml
spanner-query-gen explain-plan --config path.yaml --output yaml
spanner-query-gen vet --config path.yaml --output json
```

no-subcommand generation を残すかどうかを迷うより、stable v1 では削除でよいと思います。CLI は後方互換より分かりやすさを優先できる段階です。

### P0. 「Current implementation supports」という表現を慎重にする

DESIGN には “The current implementation supports ...” として、stable v1 config、external dataset binding、`explain-plan`、`vet`、plan metadata などが列挙されています。

実装コードとテスト出力が共有されていない外部レビューでは、この表現は検証不能です。開発側の内部作業ツリーでは実装済みでも、外部レビュー用文書では次のように分けた方が安全です。

```text
Current documented surface:
- ...

Claimed implemented in the internal working tree:
- ...

Externally reviewable contract:
- expected config shape
- expected plan fields
- expected diagnostic IDs
- minimal output snippets
```

README は利用者向けなので「現在動くもの」だけを書いてよいですが、DESIGN は “implemented” と “intended stable v1” を混ぜない方がよいです。

### P1. Roadmap に古い `federated` 語彙が残っている

Stable v1 config では `kind: external_query` に整理されていますが、Roadmap Phase 4 にまだ “Add `federated` or `external_query` config” のような表現が残っています。

これは削除して、次のように書き換えるのがよいです。

```text
Phase 4: External Query UX and SQL Files
- Harden `kind: external_query` config.
- Generate separate constants for inner Spanner SQL and outer BigQuery SQL.
- Define parameter rules for outer BigQuery parameters and inner Spanner parameters.
```

stable v1 で `federated` を廃止するなら、文書上も完全に消した方がよいです。

### P1. `query_methods: true` が現在の実装範囲と矛盾しないか確認する

README の minimal config には、

```yaml
emit:
  spanner:
    query_methods: true
    mutations: true
    dml: true
```

が入っています。

一方、DESIGN の Roadmap では Spanner Query Methods が Phase 3 に置かれています。もし query methods がまだ未実装なら、README の minimal example で `query_methods: true` を出すのは危険です。

実装済みなら問題ありません。ただし実装コードが共有されていない前提では、外部向けには次のどちらかに寄せる方が安全です。

```yaml
emit:
  spanner:
    mutations: true
    dml: true
```

または、

```yaml
emit:
  spanner:
    query_methods: false # future / experimental
    mutations: true
    dml: true
```

README の最小例は「確実に動く surface」だけにした方が採用時の混乱が少ないです。

### P1. `emit` の global / per-write semantics を決める

README / DESIGN では global `emit` と write-level `emit` が両方あります。

```yaml
emit:
  spanner:
    mutations: true
    dml: true

writes:
- name: UpdateSingerName
  emit:
  - mutation
  - dml
```

この二重指定は便利ですが、stable v1 では少し複雑です。必ず次の semantics を明文化してください。

おすすめは **intersection semantics** です。

```text
generated write surfaces = global emit ∩ write emit
```

つまり、global `emit.spanner.dml: false` なら、write-level に `dml` があっても DML は出ない。write-level `emit` が省略された場合は global default に従う。

さらに simple にするなら、stable v1 では write-level `emit` を README の最小例から消し、advanced section だけに置くのがよいです。

### P1. `projection.named_schema_tables` と `named_schema_policy` の語彙を統一する

config example では、

```yaml
projection:
  unsupported_columns: omit
  named_schema_tables: warn_and_omit
```

になっています。一方、DESIGN の説明文には `named_schema_policy` という語彙が残っているように見えます。

stable v1 の public config は `named_schema_tables` にするなら、内部説明もすべてこの名前に揃えてください。policy という一般名は plan 側だけに残す、などの二層命名は不要です。

---

## さらに simple にするための具体案

### 1. stable v1 では shorthand sugar を増やさない

「もっと simple」に見せるために、例えば次のような shorthand を入れたくなります。

```yaml
result: SingerRow
update:
- FirstName
- LastName
```

しかし stable v1 では、こういう sugar は入れない方がよいです。config grammar が増え、plan normalization と migration policy が複雑になります。

今のように、少し冗長でも、次の形に固定するのがよいです。

```yaml
result:
  cardinality: many
  struct: SingerRow

update:
  columns:
  - FirstName
  - LastName
```

simple にする場所は config grammar ではなく、README の段階的説明です。

### 2. `result.cardinality` の default を決める

DTO generation だけなら cardinality は本質的ではありません。一方、query methods を生成するなら cardinality は API semantics そのものです。

おすすめは次です。

```text
- query_methods が disabled の場合: cardinality は省略可、plan では many に正規化
- query_methods が enabled の場合: cardinality は必須、または default many を明記
```

より安全なのは、query method generation が enabled の query では `result.cardinality` を必須にすることです。

README minimal example では `cardinality: many` を明示しているので問題ありません。ただし仕様として「省略時にどうなるか」は早めに決めるべきです。

### 3. `emit` は boolean map か list のどちらかに固定する

今の global `emit` は boolean map、write-level `emit` は list です。

```yaml
emit:
  spanner:
    mutations: true
    dml: true

writes:
- emit:
  - mutation
  - dml
```

これは読めますが、型が違うため少し気持ち悪いです。

選択肢は 2 つです。

#### 案 A: boolean map に統一

```yaml
writes:
- emit:
    mutation: true
    dml: true
```

#### 案 B: list に統一

```yaml
emit:
  spanner:
  - mutations
  - dml
  bigquery:
  - row_loader
  - table_schema

writes:
- emit:
  - mutation
  - dml
```

より simple に見えるのは案 B です。ただし future feature の default / false override を扱いやすいのは案 A です。

私なら stable v1 は案 A にします。YAML は多少長くなりますが、config schema と validation が単純です。

### 4. external dataset の access evidence は advanced-only にする

README の external dataset section に `verification_evidence` の例があるのは正しいですが、最初の external dataset example には入れない方がよいです。今の README はこの点で良いバランスです。

stable v1 では、config で書くのはこれだけで十分です。

```yaml
access:
  cloud_resource_connection_id: example-project.us.example-connection
```

`verification_evidence` は、CI / platform team が必要とする advanced feature として後段に置くべきです。

### 5. plan の audit metadata は default output から外す

cleanup plan にある通り、`projection_matrix`、docs source URL、docs checked date などは通常の stable plan には重いです。

おすすめは次です。

```text
explain-plan --output yaml           # stable semantic plan
explain-plan --output yaml --audit   # docs matrix, verification evidence, volatile metadata
explain-plan --stable                # timestamps and volatile fields omitted
```

`--stable` と `--audit` の関係も決めてください。

```text
--stable --audit is allowed, but volatile audit fields are omitted.
```

または、より単純に、

```text
--stable cannot be combined with --audit
```

でもよいです。CI snapshot 用途を考えると前者の方が便利です。

---

## Open questions への回答

### 1. `kind: spanner` は `kind: spanner_database` にすべきか

**`kind: spanner` のままでよい**です。

`catalogs` の配下なので、resource kind であることは十分に伝わります。`spanner_database` は長く、BigQuery 側を `bigquery_dataset` にする圧力も生みます。

```yaml
catalogs:
- name: app
  kind: spanner
```

で十分です。

### 2. `dialect: googlesql` は必須にすべきか

**default でよい**です。

README でも「Spanner / BigQuery は GoogleSQL default」と書いておけば十分です。`dialect` は PostgreSQL dialect など非 default のために残すのがよいです。

### 3. raw SQL `EXTERNAL_QUERY(...)` は stable v1 で完全拒否すべきか

**拒否でよい**です。

このツールの価値は、inner Spanner SQL と outer BigQuery SQL を別 scope で解析できることです。raw SQL の `EXTERNAL_QUERY` を許すと、その価値が弱くなります。

escape hatch は後で追加できます。stable v1 では拒否した方が仕様が単純です。

### 4. `projection_matrix` は default plan から外すべきか

**外すべきです。**

default plan は CI contract です。docs matrix は review / audit には有用ですが、通常 plan に入れると snapshot が重くなり、差分も読みにくくなります。

### 5. `replace` は `operation: replace` のままでよいか

**`operation: replace` のままでよい**です。

`replace` は mutation API の distinct semantics です。`operation: insert` の variation にすると、未指定列や child row への影響が見えにくくなります。

ただし stable v1 では、

```text
replace + dml emit = planning error
```

を明確にしてください。

### 6. DTO cross-catalog sharing は global `dto.structs` にすべきか

**stable v1 では不要**です。

今のように各 query が `result.struct` を指定し、同じ struct 名を使ったら conservative merge する形で十分です。cross-catalog sharing が発生したら plan / vet warning で見えるようにすればよいです。

`dto.structs` は、共有方針、nullable policy、custom wrapper、rename などが増えた Phase 5 以降で足せばよいです。

### 7. `migrate-config` は stable v1 前に必要か

**必須ではありません。**

正式リリース前なので、compact config は壊してよいです。README に migration table があれば十分です。

ただし既存利用者がチーム内にいるなら、`migrate-config` ではなく、まずは `check` / `vet` の diagnostic で旧 field を具体的に指摘する方が費用対効果が高いです。

```text
unsupported field `schemas`; use `catalogs`
unsupported field `source`; use `catalog`
unsupported field `update_mask`; use `update.columns`
```

---

## 推奨する stable v1 の最小語彙

stable v1 で public grammar として残すのは、これだけでよいと思います。

```text
version

go
  package
  out

emit
  spanner
  bigquery

catalogs
  name
  kind
  dialect
  ddl
  project
  bindings

queries
  name
  catalog
  kind
  sql | table | index | inner_sql/outer_sql
  result
  params

writes
  name
  catalog
  table
  operation
  input
  key
  insert
  update
  conflict

rules
  severity
  suppressions
```

まだ stable v1 に入れなくてよいものは次です。

```text
dto.structs
overrides
models
queries_file
plugins
templates
BigQuery query methods
Partitioned DML
DML THEN RETURN
migrate-config
```

これらは設計上の場所を DESIGN に書いておくのは良いですが、README の stable v1 public config からは外す方が simple です。

---

## そのまま送れる返答案

> 方針はかなり良くなっています。特に、README の first example を Spanner-only にして、BigQuery / `EXTERNAL_QUERY` / external dataset / rules を別 section に分けたのは正しいです。`schemas -> catalogs`、`source -> catalog`、`federated -> external_query`、branch-aware writes、central `rules.suppressions` も stable v1 として採用してよいと思います。
>
> さらに simple にするなら、まず文書内の古い語彙を完全に消してください。README は subcommand 方式の stable v1 CLI になっていますが、DESIGN の CLI UX にはまだ no-subcommand generation と “Future subcommands” の表現が残っています。stable v1 では `generate` / `check` / `explain-plan` / `vet` に統一してよいです。
>
> また、stable v1 で `kind: external_query` に決めるなら、Roadmap の “`federated` or `external_query`” という表現は削除してください。`federated` は external dataset と紛らわしいので、public grammar から完全に消すのがよいです。
>
> README の minimal config に `emit.spanner.query_methods: true` が入っていますが、もし query methods がまだ Phase 3 の future work なら、最小例には入れない方が安全です。README の最小例は実装済みで確実に動く surface だけにしてください。実装済みなら、その evidence を expected output / test output として出すと外部レビューしやすいです。
>
> global `emit` と write-level `emit` を両方残すなら、intersection semantics を明文化してください。つまり `generated write surfaces = global emit ∩ write emit` です。さらに simple にするなら、write-level `emit` は README minimal example から外し、advanced section だけに置くのがよいです。
>
> external dataset 周りは今の方向でよいです。config には `access.cloud_resource_connection_id` だけを書かせ、`verification_evidence` は実際の外部 evidence がある場合だけ advanced feature として書かせるのが安全です。`not_checked` を user config に書かせない方針も維持してください。
>
> stable v1 では sugar を増やさない方がよいです。`result: SingerRow` や `update: [FirstName]` のような shorthand は便利そうですが、config grammar と migration policy を増やします。単純化は grammar の短縮ではなく、README の段階的説明と plan の正規化で実現する方がよいです。

---

## 最終判断

この cleanup は採用でよいと思います。以前の設計に比べて、仕様境界はかなり明確になっています。

残る課題は、大きな設計変更ではなく、次の同期作業です。

1. README / DESIGN / Roadmap から古い CLI と `federated` 語彙を消す。
2. `query_methods` が current なのか future なのかを明確にする。
3. global `emit` と per-write `emit` の関係を定義する。
4. `projection.named_schema_tables` など config field 名を全文書で統一する。
5. “current implementation supports” と “intended stable v1” を外部レビュー可能な形で分ける。

ここまで整理できれば、stable v1 の public surface はかなり読みやすくなると思います。
