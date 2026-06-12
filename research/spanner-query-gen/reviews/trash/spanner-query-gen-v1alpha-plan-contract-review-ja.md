# `spanner-query-gen` 最新版レビューと Spanner Omni plan contract 案

レビュー日: 2026-05-05
対象: `spanner-query-gen-v1alpha-latest-check-feedback-response-ja(1).md`, `README(17).md`, `DESIGN(17).md`, `spanner-query-gen.v1alpha.schema(5).json`

## 前提

実装コード、実行ログ、テスト出力は共有されていないため、ここでは README / DESIGN / JSON Schema / 回答文から外部レビュー可能な public contract を見ています。実装済みかどうかではなく、仕様として読みやすいか、将来の事故を避けやすいか、CI contract として安定しそうかを評価します。

## 最新版への総評

かなり良い状態まで収束しています。

特に良い点は次です。

- `queries[].result.cardinality` が `one | maybe_one | many` に整理され、row-count-only DML と DML `THEN RETURN` は future `commands` surface に分離された。
- `write.conflict` / `conflict.strategy` が public YAML から外れ、`operation: upsert` の normalized semantics は plan 側の `conflict_strategy: insert_or_update` に寄せられた。
- JSON Schema が `kind` / `operation` の discriminated union、`minLength`、`minItems`、`uniqueItems`、suppression scope/date、external dataset evidence などをかなり表現するようになった。
- `plan-report` が追加され、Spanner Omni + `AnalyzeQuery` + `spannerplan` による review artifact を出す方向になった。
- README の first example はまだ Spanner-only で、BigQuery / `EXTERNAL_QUERY` / external dataset / suppressions / Omni は後段に分離されている。これは simple さを保てている。

今回の差分では、以前の「採用した単純化が README / DESIGN / schema に反映され切っていない」という問題はかなり解消されています。残る指摘は大きな仕様変更ではなく、**plan-report と JSON Schema を public contract としてどう安定化するか**です。

## 既存仕様への追加フィードバック

### 1. `plan-report` は optional / non-primary workflow と明記したままでよい

`plan-report` は価値がありますが、primary workflow にはしない方がよいです。現 DESIGN の「live Spanner / BigQuery に依存しない primary workflow」という非目標と整合させるには、次の線引きが重要です。

```text
generate / check / explain-plan / vet / config-schema:
  DDL-first static workflow。通常 CI の中核。

plan-report:
  Optional Omni-backed review artifact。重い、環境依存、optimizer-sensitive。
```

README はすでに Omni check を extra build tag / environment variable の下に置いており、この方向は正しいです。

### 2. `plan-report` にも `--stable` 相当が欲しい

`explain-plan` には `--stable` があり、volatile audit fields を落とせる説明になっています。一方、`plan-report` は Markdown / YAML / JSON の review artifact を出すため、こちらも snapshot CI に使われやすいです。

提案:

```sh
spanner-query-gen plan-report \
  --config spanner-query-gen.yaml \
  --output yaml \
  --stable
```

`--stable` では、少なくとも次を制御できるとよいです。

- Omni runtime start time
- container/image digest の volatile 表示
- host-specific path
- database instance temporary ID
- plan acquisition timestamp
- non-semantic renderer metadata

ただし、次は必ず残すべきです。

- backend: `omni`
- Omni version / image tag or digest if available
- plan source mode: plan vs profile
- SQL digest
- DDL digest
- optimizer options actually used
- query name / catalog / rendered SQL
- normalized operator tree digest

### 3. JSON Schema の `uniqueItems` は名前重複を保証しない

JSON Schema の `uniqueItems` は array item 全体の重複を防ぐだけで、`queries[].name` や `catalogs[].name` の一意性は標準 JSON Schema だけでは直接表現しにくいです。

したがって、planner 側で以下を specific diagnostic ID 付きで検証するとよいです。

```text
catalog-name-duplicate
query-name-duplicate
write-name-duplicate
external-query-binding-name-duplicate
external-dataset-binding-name-duplicate
suppression-duplicate
```

これは DDL-dependent semantics ではなく config semantics なので、README に「schema ではなく planner diagnostics で検証する」と書いておくと、`config-schema` への期待値が正しくなります。

### 4. `plan-report` の空対象 semantics を決める

`plan-report` は「configured Spanner queries」を分析します。config が writes-only、BigQuery-only、external dataset-only の場合に何を返すかを固定した方がよいです。

推奨:

```text
no Spanner query target found:
  text/markdown: warning summary + empty report
  json/yaml: status: no_targets
  exit code: 0 by default
  --require-targets: exit code non-zero
```

これにより、monorepo CI で一部 config だけ plan-report 対象がない場合も扱いやすくなります。

### 5. `result.cardinality` の説明は少しだけ言い換える

README には「generated method cardinality semantics matter」とありますが、v1alpha では query methods は reserved/rejected です。誤解を避けるなら、次のように言い換えるとよいです。

```text
result.cardinality is recorded in the resolved plan for future query methods
and review, but v1alpha generated output remains DTOs and SQL constants unless
future query method surfaces are enabled.
```

現在の説明でも大きな問題ではありませんが、simple さを優先するならこの方が読み手の混乱が少ないです。

## Spanner Omni plan contract 機能について

### 結論

提案は面白く、`spanner-query-gen` らしい強みになり得ます。ただし、**v1alpha core config には入れず、`plan-report` の発展形として experimental / optional に置く**のがよいです。

理由は、これは単なる codegen ではなく、optimizer-sensitive な validation だからです。`spanner-query-gen` の primary contract は DDL と SQL から Go DTO / helper を生成することです。一方、plan contract は「特定の planner backend・optimizer options・DDL・SQL digest の組み合わせで、plan tree がある性質を持つ」ことを検証します。これは強力ですが環境依存です。

したがって、最初の位置付けは次がよいです。

```text
plan-report:
  plan artifact を出す。

plan-report --check:
  optional plan contracts を評価して non-zero exit できる。

vet:
  DDL/static plan validation のまま。Omni を要求しない。
```

新しい subcommand を増やさず、既存の `plan-report` に `--check` を足すのが一番 simple です。

### 重要な制約

公式ドキュメント上、Spanner Omni では `EXPLAIN QUERY` / `EXPLAIN ANALYZE QUERY` で plan を取得できます。Go client 側でも `AnalyzeQuery` は plan-only の取得に使えると説明されています。このため、`plan-report` が Omni から plan を取る設計は妥当です。

ただし、Spanner の optimizer は SQL を単純な rule だけで固定するものではありません。Spanner は複数の execution plan を評価し、効率的と判断したものを選びます。また optimizer version と statistics package を固定すると plan stability を高められますが、これは「その条件下での plan」を固定するという意味です。production data / statistics / optimizer rollout まで完全に保証するものではありません。

そのため、plan contract の文言は次のようにすべきです。

```text
This contract is validated against the configured plan environment
(backend, DDL digest, SQL digest, optimizer options, and plan mode).
It is not a universal production performance guarantee.
```

### どの query を対象にするか

v1alpha / early experimental では、対象を Spanner plan が取れるものに絞るのがよいです。

```text
対象:
  - kind: sql on Spanner catalog
  - kind: table on Spanner catalog
  - kind: index on Spanner catalog
  - kind: external_query の inner_sql

対象外:
  - BigQuery outer_sql の plan
  - BigQuery Spanner external dataset table の BigQuery-side plan
  - future commands / DML execution plan
  - mutation helpers
```

`external_query` の場合、`inner_sql` は Spanner scope として Omni で検証できますが、BigQuery outer query の optimizer behavior までは見ません。この境界は plan output に明記すべきです。

### 最小 config 案

simple にするなら、最初は predefined contract + query reference だけで十分です。

```yaml
plan_contracts:
- name: SingerIndexLookupPlan
  query: ScanSingerIDsFast
  use:
  - no_explicit_sort
  - no_hash_join
```

ここで `query` は `queries[].name` を参照します。`kind: external_query` の inner SQL を対象にしたい場合は、scope を足します。

```yaml
plan_contracts:
- name: ExternalSingerInnerPlan
  query: ExternalQuerySingerIDs
  scope: inner
  use:
  - no_explicit_sort
```

### direct contract 案

predefined だけで足りない場合の直接指定は、operator label 文字列ではなく normalized operator family を使うべきです。

```yaml
plan_contracts:
- name: SingerIndexLookupPlan
  query: ScanSingerIDsFast
  forbid:
  - operator_family: explicit_sort
  - operator_family: hash_join
  - operator_family: hash_aggregate
```

`operator_family` は `spannerplan` の表示文字列ではなく、tool 側が QueryPlan node を正規化したものにします。これが重要です。表示文字列に直接依存すると、renderer や Spanner plan schema の小変更で CI が壊れます。

### user-defined named contract 案

ユーザー定義 contract を許すなら、top-level に definitions/checks を分けると読みやすいです。

```yaml
plan_contract_definitions:
- name: oltp_index_lookup
  forbid:
  - operator_family: explicit_sort
  - operator_family: hash_join
  - operator_family: hash_aggregate

plan_contracts:
- name: SingerIndexLookupPlan
  query: ScanSingerIDsFast
  use:
  - oltp_index_lookup
```

ただし、これは v1alpha core にはまだ入れない方がよいです。最初は `plan-report --check` 用の experimental section として扱い、stable v1 に入れる前に実際の CI flakiness を観察すべきです。

### predefined contract の命名

ユーザーが挙げた例は良いですが、名前は少し正確にした方がよいです。

```text
no_explicit_sort
  Sort / Sort Limit のような明示的 sort-family operator を禁止する。
  Distributed Merge Union など「順序を保つが explicit Sort ではない」operator をどう扱うかは別 contract にする。

no_hash_join
  Hash Join family を禁止する。
  Push Broadcast Hash Join も含めるかは named variant として明示する。
  例: no_classic_hash_join / no_any_hash_join

no_hash_aggregate
  Spanner plan が Aggregate operator の execution method を hash/stream として識別できる場合だけ有効。
  もし plan source が hash aggregate と stream aggregate を安定して区別できないなら、predefined としては出さない。
```

特に `Hash Aggregate` は注意が必要です。公式 operator docs では SQL `GROUP BY` は `Aggregate` operator に対応しますが、plan source に常に `Hash Aggregate` という独立 operator 名が出るとは限りません。実際の QueryPlan metadata から hash/stream が安定して読めることを確認してから predefined にすべきです。

### plan matching は tree 全体一致ではなく semantic predicate にする

やってはいけないのは「plan tree 全体を golden snapshot として完全一致させる」ことです。これは brittle です。

推奨は semantic predicate です。

```yaml
forbid:
- operator_family: explicit_sort
  max_count: 0

require:
- operator_family: index_scan
  table: Singers
  index: SingersByName
```

ただし `require` は `forbid` より壊れやすいので、最初は `forbid` 中心でよいです。

### plan environment を必ず記録する

contract evaluation の plan output には、少なくとも次を入れるべきです。

```yaml
plan_contract_evaluation:
  backend: omni
  backend_version: ...
  backend_image_digest: ...
  plan_mode: plan
  ddl_digest: ...
  sql_digest: ...
  query_name: ScanSingerIDsFast
  optimizer:
    version: ...
    statistics_package: ...
  normalized_operator_tree_digest: ...
  results:
  - contract: no_explicit_sort
    status: pass
```

optimizer options が未指定なら `not_pinned` として warning を出すとよいです。

```text
plan-contract-optimizer-not-pinned
```

### optimizer hints / hint recommendation の扱い

ヒント推薦は便利ですが、最初から auto-fix にはしない方がよいです。生成 SQL や config を自動で書き換えると、`SQL is contract` の原則と衝突します。

最初は `remediation` として出すだけにするのが安全です。

例:

```yaml
violations:
- rule: plan-contract-operator-forbidden
  query: ScanSingerIDsFast
  contract: no_explicit_sort
  observed:
    operator_family: explicit_sort
    operator: Sort
  remediation:
  - kind: config_change
    message: "For table/index shorthand, try order_by: none if result order is not required."
  - kind: index_design
    message: "Consider an index whose key order satisfies the requested ORDER BY."
  - kind: sql_hint
    message: "If manually reviewed, consider FORCE_INDEX or optimizer hints. Do not apply automatically."
```

具体的な候補:

- `no_explicit_sort` 違反
  - table/index shorthand なら `order_by: none` を候補にする。
  - 順序が必要なら、ORDER BY に合う index key order を提案する。
- `no_hash_join` 違反
  - join method hint の候補を出す。ただし APPLY / MERGE / HASH のどれが良いかは workload 依存なので confidence を低くする。
  - join input を小さくする predicate / index を提案する。
- `no_hash_aggregate` 違反
  - GROUP BY key order に合う scan / index を検討する。
  - plan source が hash/stream を安定識別できないなら recommendation を出さない。
- optimizer version 未固定
  - `OPTIMIZER_VERSION` statement hint または DB-level option で pin する選択肢を提示する。
  - Cloud Spanner production では optimizer statistics package も stability に関係するため、必要に応じて pinning を検討する。

### diagnostics ID 案

```text
plan-contract-backend-unavailable
plan-contract-target-unsupported
plan-contract-optimizer-not-pinned
plan-contract-operator-forbidden
plan-contract-operator-required-missing
plan-contract-operator-family-unknown
plan-contract-plan-source-unsupported
plan-contract-remediation-suggested
```

ユーザーが明示した plan contract の violation は、デフォルトで error にすべきです。通常の `rules.suppressions` で warning 表示だけ抑える、という扱いにはしない方がよいです。contract は「守りたい」と宣言した性質なので、破るなら config から外すか、将来 `allow_failure` のような明示を入れるべきです。

### v1alpha に入れるべきか

私は **v1alpha の public YAML にはまだ入れない** 方がよいと思います。

ただし、`plan-report` がもう入っているなら、次の順序で進めるのはありです。

```text
Phase A:
  plan-report の YAML/JSON に normalized_operator_tree を出す。
  まだ contract config はなし。

Phase B:
  plan-report --check --contract-file plan-contracts.yaml を追加する。
  main v1alpha config には混ぜない。

Phase C:
  CI で安定することが分かった predefined contract だけを
  main config の optional plan_contracts に昇格する。
```

この段階化が一番安全です。main YAML にいきなり入れると、今まで減らしてきた v1alpha config がまた重くなります。一方、外部 contract file に逃がせば、opinionated な機能を試しつつ、core config の simple さを保てます。

### 推奨する最初の MVP

最初に実装するなら、これだけで十分です。

```yaml
# plan-contracts.yaml, not spanner-query-gen.yaml initially
version: v1alpha-plan-contracts

contracts:
- name: ScanSingerIDsFastPlan
  query: ScanSingerIDsFast
  backend: omni
  use:
  - no_explicit_sort
```

CLI:

```sh
spanner-query-gen plan-report \
  --config spanner-query-gen.yaml \
  --contracts plan-contracts.yaml \
  --check \
  --output json
```

MVP predefined は `no_explicit_sort` だけでよいです。今の README/DESIGN にすでに「generated index query avoids Sort」を optional Omni check として書いているため、そこから自然に一般化できます。

次に `no_any_hash_join` を追加し、最後に plan source が十分に安定してから `no_hash_aggregate` を追加するのがよいです。

## まとめ

最新版の v1alpha config 整理はかなり良いです。`conflict.strategy` と `exec` の整理も反映され、JSON Schema も public contract としてかなり使える状態になっています。

Spanner Omni plan contract は、方向性としては非常に面白く、`spanner-query-gen` の差別化にもなります。ただし、core generator の simple さと DDL-first workflow を壊さないために、まずは `plan-report` の optional extension として、別 contract file + `plan-report --check` から始めるのがよいです。

特に強く推奨するのは次です。

1. まず `plan-report` の normalized operator tree / operator family を安定化する。
2. 最初の predefined は `no_explicit_sort` だけにする。
3. `Hash Join` / `Hash Aggregate` は QueryPlan metadata で安定識別できることを確認してから predefined にする。
4. optimizer version / plan environment / SQL digest / DDL digest を必ず plan artifact に残す。
5. hint recommendation は remediation に留め、自動適用しない。
6. main v1alpha YAML にすぐ混ぜず、外部 `plan-contracts.yaml` で実験する。

この順なら、opinionated な価値を試しつつ、ここまで整理してきた `spanner-query-gen` の simple な public config を守れます。

## 参考資料

- `README(17).md`
- `DESIGN(17).md`
- `spanner-query-gen.v1alpha.schema(5).json`
- Google Cloud: View Spanner Omni execution plans
- Google Cloud: Query execution plans / operators
- Google Cloud: Manage the query optimizer
- Google Cloud Blog: A technical overview of Cloud Spanner's query optimizer
