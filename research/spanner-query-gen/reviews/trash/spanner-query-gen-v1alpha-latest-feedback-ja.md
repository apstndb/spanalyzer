# `spanner-query-gen` v1alpha 最新版へのフィードバック

対象:

- `spanner-query-gen-breaking-change-spec-cleanup-plan-ja(2).md`
- `spanner-query-gen-stable-v1-simplification-review-response-ja.md`
- `README(10).md`
- `DESIGN(10).md`

前提: 実装コード・実行ログ・テスト結果は共有されていないため、ここでは「仕様としての整合性」「README/DESIGN としての読みやすさ」「外部レビュー可能な contract」としてレビューする。`Implemented` や test 名の妥当性は検証していない。

## 総評

今回の方向性はかなり良い。特に、`version: v1alpha` に寄せたこと、README の first example を Spanner-only にしたこと、`catalogs` / `queries` / `writes` / `emit` / `rules` に public grammar を絞ったこと、`EXTERNAL_QUERY` と Spanner external dataset を分離したことは、以前よりずっと整理されている。

一方で、まだ「simple にする余地」はある。最大のポイントは、**新しい sugar を減らすことではなく、意味の重複・current/future の混在・実際の Spanner semantics とずれうる config 表現を減らすこと**だと思う。

結論としては、v1alpha では次のようにさらに絞るのがよい。

```text
version: v1alpha
go
emit
catalogs
queries
writes
rules.suppressions
```

`rules.severity`、write-level `emit`、`queries[].params`、raw SQL `EXTERNAL_QUERY` escape hatch、general upsert branch split は、可能なら v1alpha から外すか、少なくとも非常に厳しい制約つきにした方がよい。

## 良くなった点

### 1. `v1alpha` 化は正しい

`version: "1"` や stable v1 という表現を避け、`version: v1alpha` を mutable preview channel とする方針は妥当。正式 v1 までは breaking change を許容できることが明示され、利用者の期待値もずれにくい。

この方針なら、旧 compact config を v1alpha で無理に支えず、`schemas` / `source` / `client` / `update_mask` などを移行先つき diagnostic で拒否できる。

### 2. README の段階的構成はかなり改善した

README の最初の例が Spanner-only になり、BigQuery、`EXTERNAL_QUERY`、external dataset、rules を後段に分けたのは良い。config surface はまだ大きいが、最初に読む人の負荷は大きく下がっている。

### 3. external dataset の位置付けは収束している

Spanner external dataset を `EXTERNAL_QUERY` の別記法ではなく、BigQuery catalog binding として扱う方針はこのままでよい。README も DESIGN も「インフラは作らず、static catalog shape を model するだけ」という境界を示しており、product boundary と合っている。

### 4. `emit.spanner.query_methods` を v1alpha で拒否する判断は良い

query methods を future surface として予約し、今の最小 config から外したのは良い。DTO / mutation / DML helper と query method semantics を混ぜない方が、v1alpha の整理として安全。

## P0: 仕様として先に直したい点

### 1. `upsert` の branch semantics が Spanner `INSERT OR UPDATE` とずれうる

現在の README は、`upsert` を Spanner `INSERT OR UPDATE` に map しつつ、`insert.columns` と `update.columns` を分けている。

```yaml
operation: upsert
insert:
  columns:
  - SingerId
  - FirstName
  - LastName
update:
  columns:
  - FirstName
  - LastName
conflict:
  strategy: insert_or_update
```

この例は `insert.columns - key == update.columns` なので問題ない。しかし、仕様として branch を分けてしまうと、将来 users が次のような config を書けるように見える。

```yaml
insert:
  columns: [SingerId, FirstName, LastName, CreatedAt]
update:
  columns: [FirstName, LastName]
conflict:
  strategy: insert_or_update
```

これは Spanner `INSERT OR UPDATE` では忠実に表現できない可能性が高い。Spanner DML の `INSERT OR UPDATE` は、既存行がある場合、statement の column list に指定した値でその行を更新し、指定しない列だけが unchanged になる。つまり `insert.columns` に入れた non-key column は update branch でも touched column になる。`ON UPDATE` expression を持つ列にはさらに server-side side effect がある。

したがって v1alpha では、次のどちらかにした方がよい。

#### 推奨案 A: `insert_or_update` strategy では branch split を制限する

```text
For conflict.strategy: insert_or_update:
  update.columns must equal insert.columns minus key columns.
```

この invariant を README / DESIGN / plan diagnostic に明記する。

- 一致する場合: mutation と DML を生成してよい。
- 一致しない場合: planning error。
- branch が異なる upsert は、将来の `conflict.strategy: on_conflict` で扱う。

#### 推奨案 B: v1alpha では upsert branch split をやめる

より simple にするなら、v1alpha の `upsert` は現実の Spanner `INSERT OR UPDATE` semantics に合わせて、単一の column list にする。

```yaml
writes:
- name: UpsertSingerName
  catalog: app
  table: Singers
  operation: upsert
  input: SingerWrite
  key:
  - SingerId
  upsert:
    columns:
    - SingerId
    - FirstName
    - LastName
  conflict:
    strategy: insert_or_update
```

本当の insert/update branch 分離は `ON CONFLICT DO UPDATE SET ...` を実装するタイミングで追加する。

個人的には、v1alpha では案 A が現実的。ただし「simple」を最優先するなら案 B の方が正直で読みやすい。

### 2. cleanup plan が README / response とまだ同期していない

`spanner-query-gen-breaking-change-spec-cleanup-plan-ja(2).md` には、まだ古い表現が残っている。

- write-level `emit` が list として出ている。

  ```yaml
  emit:
  - mutation
  - dml
  ```

  しかし response では boolean map に統一するとあり、README も `emit: { mutation: true, dml: true }` になっている。

- 廃止表では `methods: [mutation, dml]` → `emit: [mutation, dml]` になっているが、v1alpha の canonical shape は boolean map のはず。

- external dataset example に `access.mode`、`access.database_role`、`access.verification.status: not_checked` が config 入力として残っている。しかし README / response の方針では、config に `not_checked` を書かせず、外部 evidence がある場合だけ `access.verification_evidence` を書くことになっている。

cleanup plan はレビュー用の計画文書だとしても、現時点では README / DESIGN と同じ grammar を示した方がよい。特に今は「仕様整理」がテーマなので、plan 文書内の古い grammar は混乱の原因になる。

### 3. raw SQL `EXTERNAL_QUERY` の扱いが DESIGN 内で矛盾している

README と v1alpha 方針では、raw BigQuery SQL に含まれる `EXTERNAL_QUERY(...)` は rejected とされている。これは正しい。

しかし DESIGN の federated query section には、raw BigQuery SQL containing `EXTERNAL_QUERY` remains supported only when connection and inner SQL arguments are string literals、という趣旨の文が残っている。

これは v1alpha public grammar と衝突して見える。

修正案:

```text
v1alpha public YAML rejects raw BigQuery SQL containing EXTERNAL_QUERY(...).
A future or legacy-only parser may support literal-only EXTERNAL_QUERY extraction,
but that is not part of v1alpha public grammar.
```

さらに simple にするなら、literal-only extraction は future work からも消してよい。`kind: external_query` だけを canonical にすると、inner SQL / outer SQL / connection mapping / diagnostics がすべて reviewable になる。

### 4. `rules.severity` は v1alpha から外してもよい

README は `rules.severity` を説明しており、unsuppressed diagnostic を `error` にできるとしている。一方、DESIGN では current `vet` は plan validation であり、future work として per-rule severity configuration を挙げている。このあたりが少し混ざって見える。

simple にするなら、v1alpha の `rules` は `suppressions` だけにするのがよい。

```yaml
rules:
  suppressions:
  - scope: catalog-binding/analytics.app_dataset
    rule: external-dataset-access-unverified
    reason: access is validated by Terraform plan in CI
```

`rules.severity` は、full rule engine または `--strict` を入れるタイミングで追加すればよい。現在の v1alpha が「plan validation, not a full policy engine」なら、severity override はやや早い。

どうしても残すなら、DESIGN の future work から「per-rule severity configuration」を削るか、「implemented warning severity override」と「future custom rule policy」を分けて書くべき。

### 5. `result.cardinality` と `queries[].params` は README/DESIGN の最初の v1alpha shape から薄めたい

`emit.*.query_methods` を v1alpha で拒否するなら、`result.cardinality` の実運用上の意味はまだ限定的になる。README では `result.cardinality` defaults to `many` と書いているので、minimal example では省略した方が simple。

```yaml
queries:
- name: ListSingers
  catalog: app
  kind: table
  table: Singers
  result:
    struct: SingerRow
```

`params` も同様。DESIGN の v1alpha config sample には `params` が入っているが、analyzer-inferred params plus overrides という位置付けなら、最初の public sample からは外した方がよい。`params` は Phase 3 query methods or override section に移すのが自然。

## P1: さらに simple にするなら

### 1. write-level `emit` を v1alpha から外す

現在の global `emit` と write-level `emit` の intersection は正しく設計されているが、v1alpha の minimal grammar としては少し重い。

最も simple な仕様はこれ。

```text
- global emit.spanner.mutations controls mutation helpers for all mutation-compatible writes.
- global emit.spanner.dml controls DML helpers for all DML-compatible writes.
- operation-specific incompatibility, such as replace + dml, is rejected or silently unavailable according to documented rule.
- per-write emit is future work.
```

これなら intersection semantics、empty map semantics、write-level allow/deny の説明が不要になる。

必要になったら後で追加できる。逆に、一度 v1alpha で per-write `emit` を入れると、削るのは難しい。

妥協案として per-write `emit` を残す場合は、次を明記する。

```text
writes[].emit omitted: use global emit
writes[].emit.mutation missing: treated as false or inherit?  ← 必ず固定
writes[].emit empty map: emit no helpers or inherit?           ← 必ず固定
```

個人的には、v1alpha では write-level `emit` を削る方が好み。

### 2. `rules` は suppressions-only にする

前述の通り、`rules.severity` は便利だが policy layer っぽい。v1alpha は「plan validation + suppressions」だけに寄せると読みやすい。

```yaml
rules:
  suppressions:
  - scope: query/FederatedReviewedQuery
    rule: cross-dialect-timestamp-truncation
    reason: downstream uses microsecond precision only
```

severity は `vet --strict` や full rule engine と一緒に入れる。

### 3. `input` / `key` は「単一 canonical shape」だけにする

README/DESIGN は `input: SingerWrite`、`key: [SingerId]` を使っている。これは読みやすいのでこのままでよいと思う。

ただし、cleanup plan などで `input.struct` や `key.columns` が混ざるなら、どちらかに固定する。

より simple な canonical shape:

```yaml
input: SingerWrite
key:
- SingerId
```

将来 `input` に options が必要になった時だけ、breaking change ではなく nested form を追加するか、別 field を足す。

### 4. `kind: spanner` / `kind: bigquery` はこのままでよい

open question に `kind: spanner_database` があったが、README の `kind: spanner` の方が簡単で十分。resource kind は catalog の種類であり、database か dataset かは `kind` と周辺 field で分かる。

同様に、`dialect: googlesql` は default でよい。非 default の dialect だけ書かせる方が README が軽くなる。

### 5. external dataset config は README の形でよい。cleanup plan 側だけ同期する

README の external dataset config はかなり良い。

```yaml
bindings:
  spanner_external_datasets:
  - name: app_dataset
    dataset: analytics_spanner
    spanner_catalog: app
    spanner_database_uri: google-cloudspanner://reader@/projects/...
    location: US
    access:
      cloud_resource_connection_id: example-project.us.example-connection
```

config ではこれ以上 canonical plan field を書かせない方がよい。

- `access.mode` は plan が infer する。
- `database_role` は recognized URI から infer する。
- `not_checked` は plan default。
- `verification_evidence` は実 evidence がある時だけ optional。

この方針は維持するべき。

## 仕様同期チェックリスト

v1alpha をさらに整理するなら、次の差分を一度に直すのがよい。

```text
README
  [ ] Minimal Config から result.cardinality を省略するか、なぜ必要か説明する
  [ ] Writes example で per-write emit を使うなら boolean map semantics を完全に説明する
  [ ] rules.severity を v1alpha に残すか future に送るか決める

DESIGN
  [ ] raw SQL EXTERNAL_QUERY literal-only support 文を削除または future/legacy と明記する
  [ ] rules.severity current/future の矛盾を解消する
  [ ] queries[].params を v1alpha sample に残すか future に送るか決める
  [ ] upsert insert/update branch invariant を明記する

cleanup plan
  [ ] stable v1 表現が残っていないか再確認する
  [ ] write-level emit list を boolean map に直す、または per-write emit を削る
  [ ] methods -> emit の移行先を boolean map に直す
  [ ] access.verification.status: not_checked を config example から消す
  [ ] access.mode / database_role を config input ではなく plan output として扱う
```

## 推奨する v1alpha minimal config

さらに simple にするなら、README の first example はここまで削れる。

```yaml
version: v1alpha

go:
  package: querydemo
  out: querydemo.sql.go

emit:
  spanner:
    mutations: true
    dml: true

catalogs:
- name: app
  kind: spanner
  ddl: schema/spanner.sql

queries:
- name: ListSingers
  catalog: app
  kind: table
  table: Singers
  result:
    struct: SingerRow

writes:
- name: UpdateSingerName
  catalog: app
  table: Singers
  operation: update
  input: SingerWrite
  key:
  - SingerId
  update:
    columns:
    - FirstName
    - LastName
```

`result.cardinality` は future query method example で出せばよい。

## 推奨する `upsert` の v1alpha rule

現在の config shape を残すなら、この rule を入れる。

```text
For operation: upsert with conflict.strategy: insert_or_update:
  inserted_non_key_columns = insert.columns - key
  updated_columns = update.columns
  inserted_non_key_columns must equal updated_columns
```

つまり、次は valid。

```yaml
operation: upsert
key: [SingerId]
insert:
  columns: [SingerId, FirstName, LastName]
update:
  columns: [FirstName, LastName]
conflict:
  strategy: insert_or_update
```

次は invalid。

```yaml
operation: upsert
key: [SingerId]
insert:
  columns: [SingerId, FirstName, LastName, CreatedAt]
update:
  columns: [FirstName, LastName]
conflict:
  strategy: insert_or_update
```

invalid 理由:

```text
conflict.strategy: insert_or_update cannot represent insert-only non-key columns.
Spanner INSERT OR UPDATE updates all specified non-key columns when the row already exists.
Use future conflict.strategy: on_conflict or split this into separate insert/update writes.
```

## そのまま送れる返答案

> 今回の v1alpha 化はかなり良いです。特に README の first example が Spanner-only になり、`catalogs` / `queries` / `writes` / `emit` / `rules` に grammar が絞られた点は以前よりかなり simple です。`EXTERNAL_QUERY` と external dataset を分ける方針もこのままでよいと思います。
>
> ただし、さらに simple にするなら、まず cleanup plan と README/DESIGN の grammar を完全に同期してください。cleanup plan にはまだ write-level `emit: [mutation, dml]`、`methods -> emit: [mutation, dml]`、`access.verification.status: not_checked` のような古い形が残っています。response / README の方針どおり、write-level emit を残すなら boolean map に統一し、`not_checked` は config ではなく plan default に寄せるべきです。
>
> 一番重要なのは `upsert` semantics です。v1alpha では `upsert` を Spanner `INSERT OR UPDATE` に map すると書かれていますが、`INSERT OR UPDATE` は既存行がある場合に statement の column list で指定した値を更新します。つまり `insert.columns` に含まれる non-key column は update branch でも touched column になります。したがって、`conflict.strategy: insert_or_update` で `insert.columns - key` と `update.columns` が異なる config は、少なくとも DML / mutation helper では忠実に生成できないはずです。v1alpha では `update.columns == insert.columns - key` を planning invariant にするか、より simple に `insert_or_update` strategy では単一 column list に戻すのがよいです。本当の branch-different upsert は将来の `ON CONFLICT DO UPDATE SET` strategy に送るべきです。
>
> さらに削るなら、write-level `emit` と `rules.severity` は v1alpha から外してもよいと思います。global `emit` だけにすれば intersection semantics が不要になりますし、current `vet` が plan validation で full policy engine ではないなら、`rules` は `suppressions` だけの方が読みやすいです。severity override は `--strict` や full rule engine と一緒に後から追加できます。
>
> README の minimal config では `result.cardinality` も省略してよいと思います。`emit.*.query_methods` が v1alpha で rejected なら、cardinality は future query method example で初めて出す方が自然です。
>
> 最後に、DESIGN 内の raw SQL `EXTERNAL_QUERY` に関する記述を整理してください。v1alpha public config では raw `EXTERNAL_QUERY(...)` を拒否する方針なのに、literal-only なら supported と読める文が残っているため、ここは future/legacy escape hatch と明記するか削除した方がよいです。

## 参考にした外部仕様

- Cloud Spanner GoogleSQL DML syntax: `INSERT OR UPDATE` は primary key がなければ insert、既存行があれば statement で指定した値で update し、未指定列は unchanged。`ON CONFLICT DO UPDATE` は `SET` clause で更新列を指定できる。
- Cloud Spanner mutations: `InsertOrUpdate` mutation は行が存在しない場合に追加し、存在する場合に column values を更新する。
- Cloud Spanner DML vs mutations: DML と mutations は read-your-writes や constraint checking が異なり、同一 transaction で混ぜないことが best practice。
- BigQuery Spanner external datasets: external dataset tables は通常の BigQuery table と同じように query できるが、DML は unsupported。Data Boost は default で変更できない。default Spanner schema の table のみ accessible、unsupported columns は BigQuery 側で accessible ではない。
- BigQuery federated queries: source database に実行される external query は read-only で、unsupported data types が含まれると query fails immediately。
