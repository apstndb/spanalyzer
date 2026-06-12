# `spanner-query-gen` v1alpha plan contract 対応後レビュー

## 前提

対象は、2026-05-05 時点の次の添付ファイルです。

- `spanner-query-gen-v1alpha-plan-contract-review-response-ja.md`
- `README(18).md`
- `DESIGN(18).md`
- `spanner-query-gen.v1alpha.schema(6).json`

実装コード、実行ログ、テスト出力は共有されていないため、ここでは **public contract / README / DESIGN / JSON Schema の整合性レビュー** に限定します。回答文にある test 名や “確認したこと” は、外部から検証済みの事実ではなく、開発側の内部作業ツリー上の主張として扱います。

## 総評

今回の対応はかなり良いです。特に、plan contract を main `spanner-query-gen.yaml` に入れず、`plan-report` の optional / experimental extension として扱った判断は、これまで進めてきた “v1alpha config を simple に保つ” 方針とよく合っています。

また、最初の predefined contract を `no_explicit_sort` だけに絞ったこと、`Hash Join` / `Hash Aggregate` をまだ predefined にしなかったこと、`Distributed Merge Union` を explicit sort と誤分類しない方針にしたことも妥当です。Spanner の operator 表現は最適化・backend・表示形式の影響を受けるので、最初から広い operator contract を入れるより、まず `Sort` / `Sort Limit` のような比較的説明しやすい contract から始める方が安全です。

一方で、`plan-report` が CI artifact / contract check として使われ始めるなら、次に固めるべき点は **plan source、target selection、contract file grammar、exit code、digest の意味** です。大きな設計変更は不要ですが、このあたりを曖昧なままにすると、あとで “plan contract は便利だが flaky” という印象になりやすいです。

## 良い反映

### 1. main v1alpha config に plan contract を混ぜなかった点

これは強く支持します。

`spanner-query-gen.yaml` は DDL / SQL / DTO / write helper の宣言に集中し、plan property contract は外部 `plan-contracts.yaml` に分ける。これは config / plan / generated Go の 3 層分離を保つうえで重要です。

```yaml
version: v1alpha-plan-contracts

contracts:
- name: SingerIndexLookupPlan
  query: ScanSingerIDsFast
  backend: omni
  use:
  - no_explicit_sort
```

この形なら、opinionated な plan validation を試せますが、core generator の public YAML surface は膨らみません。

### 2. `no_explicit_sort` だけから始めた点

これも妥当です。

`no_hash_join` や `no_hash_aggregate` は魅力的ですが、operator family の分類が安定するまで predefined contract にしない方がよいです。特に aggregate は、公式 operator 名としてはまず `Aggregate` であり、hash / stream の区別が表示名だけで安定的に取れるとは限りません。まずは direct rule の実験対象に留める、あるいはさらに後回しでよいです。

### 3. `Distributed Merge Union` を explicit sort に含めなかった点

これは非常に良い判断です。

`Distributed Merge Union` は sorted result を作りますが、`Sort` / `Sort Limit` という明示的な sort operator とは性質が違います。`description` に “sorted” のような文字列があるだけで `explicit_sort` に分類すると false positive が増えます。

現方針のように、

```text
explicit_sort:
  - Sort
  - Sort Limit

distributed_merge_union:
  - Distributed Merge Union
```

のように family を分けるのがよいです。

### 4. `--stable` と empty-target semantics を入れた点

`plan-report --stable`、`status: no_targets`、`--require-targets` は、CI で使うために必要な整理です。

特に、Spanner query target がない config でデフォルト exit code 0 にし、`--require-targets` だけ non-zero にするのは良いです。BigQuery-only config や write-only config で `plan-report` を一律に失敗させるより、ユーザーが CI policy を選べる方が自然です。

## 追加フィードバック

## P0: Spanner Omni の Pre-GA / Preview 性を README に明記した方がよい

README / DESIGN は `plan-report` を optional Omni-backed workflow と説明しており、これは良いです。ただ、Spanner Omni execution plan 機能は公式 docs 上 Preview / Pre-GA 扱いなので、README の `plan-report` 説明にも短い注意書きを入れた方が安全です。

提案文:

```text
plan-report depends on Spanner Omni execution-plan support. This workflow is
intended for review, testing, and prototyping, and is not part of the primary
DDL-first generation path.
```

さらに明確にするなら:

```text
Do not treat plan-report contracts as production performance guarantees; they
are review contracts for a pinned or described plan environment.
```

これは今の “optional / experimental extension” という位置付けを補強します。

## P0: `AnalyzeQuery` と Omni `EXPLAIN` のどちらを source of truth にするかを report に出す

README では `plan-report` が Spanner Omni を起動し、`AnalyzeQuery` で configured Spanner queries を解析すると説明されています。一方、Spanner Omni の公式 docs は execution plan の取得方法として `EXPLAIN QUERY` / `EXPLAIN ANALYZE QUERY` を示しています。

実装が `AnalyzeQuery` RPC で QueryPlan proto を取っているなら、それでよいです。ただし、外部レビュー可能な contract としては report に plan source を出した方がよいです。

```yaml
plan_source:
  backend: omni
  api: analyze_query        # or explain_query
  render_tool: spannerplan
```

この区別はかなり重要です。

- `analyze_query`: structured QueryPlan を直接持つ前提で operator-family classification しやすい。
- `explain_query`: text/table rendering 由来なら parser 揺れを別に考える必要がある。

現在の方針が `AnalyzeQuery` なら、`plan_source.api: analyze_query` を report の stable semantic fields に入れることを勧めます。

## P0: plan-report target selection を明文化する

`plan-report` は “configured Spanner queries” を分析すると説明されていますが、どこまで含むかを固定した方がよいです。

少なくとも以下を README か DESIGN に明記してください。

```text
plan-report targets:
  included:
    - queries with a Spanner catalog and kind: sql
    - queries with a Spanner catalog and kind: table
    - queries with a Spanner catalog and kind: index
  maybe included:
    - kind: external_query inner_sql, because it is Spanner SQL
  excluded:
    - BigQuery outer SQL
    - BigQuery external dataset tables
    - writes, until future commands/DML plan support exists
```

私の推奨は、`kind: external_query` の `inner_sql` は plan-report target に含めることです。理由は単純で、`inner_sql` は Spanner SQL であり、`EXTERNAL_QUERY` 経由であっても plan property contract を持ちたいユースケースが自然にあります。

ただし、BigQuery outer SQL は対象外でよいです。external dataset も BigQuery catalog binding なので、`plan-report` の対象外でよいです。

report には、含めた target だけでなく、除外した target も出すとレビューしやすいです。

```yaml
targets:
  included_count: 3
  excluded:
  - query: AnalyticsOuterQuery
    reason: bigquery_outer_sql_not_supported_by_plan_report
```

## P0: contract target mismatch と exit code を固定する

`no_targets` の扱いは良くなりました。次は contract file がある場合の mismatch を明文化した方がよいです。

推奨 matrix:

| 状況 | status | default exit | `--check` exit | 備考 |
|---|---:|---:|---:|---|
| Spanner target なし、contracts なし | `no_targets` | 0 | N/A | `--require-targets` なら non-zero |
| `--check` あり、`--contracts` なし | `cli_error` | N/A | non-zero | 既に採用済み方針 |
| contracts が unknown query を参照 | `contract_error` | non-zero | non-zero | contract file の誤り |
| contracts が non-Spanner target を参照 | `contract_error` | non-zero | non-zero | 例: BigQuery outer query |
| contracts あり、violations なし | `ok` | 0 | 0 |  |
| contracts あり、violation あり | `contract_violation` | 0 or non-zero | non-zero | `--contracts` 単体の exit policy を要定義 |

特に、`--contracts` だけ渡して `--check` を渡さないケースをどう扱うか決めた方がよいです。私は次を推します。

```text
--contracts without --check:
  evaluate contracts and include results in the report, but exit 0.

--contracts with --check:
  evaluate contracts, include results in the report, and exit non-zero on violation.
```

この方が、まず report を観察し、その後 CI で enforce する、という移行がしやすいです。

## P1: direct rule grammar は `max_count` を急がなくてよい

回答では direct rule として `forbid.operator_family` / `max_count` を受けられるようにしたとあります。これは分かりますが、MVP としては少しだけ複雑です。

最小形はこれで十分です。

```yaml
contracts:
- name: SingerIndexLookupPlan
  query: ScanSingerIDsFast
  backend: omni
  forbid:
    operator_families:
    - explicit_sort
```

または、既存方針に合わせるなら:

```yaml
contracts:
- name: SingerIndexLookupPlan
  query: ScanSingerIDsFast
  backend: omni
  forbid:
  - operator_family: explicit_sort
```

`max_count` を残すなら、以下を明記してください。

```text
max_count counts normalized operator nodes after operator-family normalization.
If omitted, max_count defaults to 0.
```

ただ、`max_count > 0` の明確な初期ユースケースがまだ弱いので、v1alpha-plan-contracts MVP では `forbid` のみに寄せる方が simple です。

## P1: digest の入力を固定する

`sql_sha256`、`ddl_sha256`、`operator_tree_sha256` は良い review fields です。次に必要なのは、何を hash しているかの定義です。

推奨:

```yaml
hashes:
  sql_sha256:
    input: rendered_sql_bytes
  ddl_sha256:
    input: resolved_catalog_ddl_bytes
  operator_tree_sha256:
    input: normalized_operator_tree_v1
```

特に `operator_tree_sha256` は、次を hash から除く方がよいです。

- node ID
- acquisition timestamp
- temporary database name
- runtime statistics
- row estimates / rows returned
- rendered table width / wrapping

逆に、次は含める候補です。

- normalized operator name
- operator family
- child relationship / tree shape
- relevant stable metadata, if classification depends on it

あわせて、次のような version field を入れると将来の変更に強くなります。

```yaml
normalization:
  operator_tree_version: v1
  operator_family_mapping_version: v1
```

operator family mapping を直した時に digest が変わるのは自然ですが、その理由を review できるようになります。

## P1: backend identity は “未記録” でも field として出す価値がある

回答では backend version / image digest はまだ report に入れていないとあります。境界整理まで待つ判断は理解できます。

ただし、plan contract の環境固定性を考えると、少なくとも “まだ記録していない” ことを明示する field があると親切です。

```yaml
backend:
  kind: omni
  version: not_recorded
  image_digest: not_recorded
```

これにより、将来 version / image digest を入れた時も、利用者は “以前は未記録だった field が具体化された” と理解できます。

## P1: optimizer not pinned は pass/fail とは別の environment warning にする

`optimizer.version: not_pinned` と `optimizer.statistics_package: not_pinned` を report に出すのは良いです。

ただし、contract が pass した時に optimizer が未 pin だと、ユーザーは contract を強く信頼しすぎる可能性があります。`contract_summary` に environment warning を分けて出すとよいです。

```yaml
contract_summary:
  status: passed
  violations: 0
  environment_warnings:
  - optimizer_not_pinned
  - statistics_package_not_pinned
```

将来、厳格な CI 向けに `--require-pinned-optimizer` のような flag を追加してもよいですが、今すぐ main config に pinning 設定を入れる必要はありません。

## P1: remediation text には confidence と対象を付ける

remediation text を出し、自動 rewrite しない方針は正しいです。

さらに、suggestion に confidence と対象を付けるとレビューしやすくなります。

```yaml
remediation:
- message: "Consider order_by: none for this generated shorthand query."
  applies_to: config
  confidence: high
- message: "Consider an index whose key order satisfies the requested ORDER BY."
  applies_to: ddl
  confidence: medium
- message: "Consider FORCE_INDEX only after validating selectivity and lock behavior."
  applies_to: sql
  confidence: low
```

特に `order_by: none` は `kind: table` / `kind: index` の generated shorthand にはよい提案ですが、raw SQL には直接適用できません。recommendation がどの surface に効くのかを明示してください。

## P1: plan contract suppressions は main `rules.suppressions` と混ぜない方がよい

現在、plan contracts は external file に分離されています。この方針を維持するなら、plan contract violations の suppression も main config の `rules.suppressions` には混ぜない方がよいです。

理由は、main config の `rules.suppressions` は DDL-first static planning diagnostics のためのものだからです。plan contract は experimental / Omni-backed / optimizer-sensitive な別 workflow です。

将来 suppression が必要になったら、contract file 側に置くのがよいです。

```yaml
contracts:
- name: SingerIndexLookupPlan
  query: ScanSingerIDsFast
  backend: omni
  use:
  - no_explicit_sort
  suppressions:
  - rule: no_explicit_sort
    reason: "Temporary until index rollout completes"
    owner: app-db
    expires: "2026-12-31"
```

ただし、MVP では suppression 自体を入れなくてもよいです。まずは violations を report するだけで十分です。

## P2: plan-contract schema はまだ必須ではないが、unknown field rejection は必要

`config-schema` は public v1alpha config 用なので、`plan-contracts.yaml` は別 contract です。最初から `plan-contract-schema` subcommand を増やす必要はないと思います。

ただし、external contract file も CI で使われるなら、少なくとも次は明記した方がよいです。

```text
plan-contracts.yaml rejects unsupported or unknown fields.
```

将来、plan contracts が定着したら、次のいずれかを検討すればよいです。

```sh
spanner-query-gen plan-report --contract-schema --output yaml
```

または:

```sh
spanner-query-gen config-schema --kind plan-contracts --output yaml
```

ただ、今は subcommand を増やさず、README に grammar を明記するだけで十分です。

## P2: report summary に count fields を入れる

Markdown では人間が読めますが、YAML/JSON は CI artifact なので count fields があると便利です。

```yaml
summary:
  status: ok
  target_count: 3
  analyzed_count: 3
  excluded_count: 1
  contract_count: 2
  contract_evaluated_count: 2
  contract_violated_count: 0
  environment_warning_count: 2
```

これは新しい機能というより、machine-readable output の usability 改善です。

## `Hash Join` / `Hash Aggregate` について

引き続き、v1alpha-plan-contracts の predefined にはまだ入れない方がよいです。

`hash_join` family は `Hash Join` / `Push Broadcast Hash Join` から分類できる可能性がありますが、まず実際の Omni / Cloud Spanner QueryPlan metadata の揺れを見た方がよいです。

`Hash Aggregate` はさらに慎重でよいです。operator docs 上の基本名は `Aggregate` なので、hash/stream の区別をどの metadata から取るか、Omni で安定して出るか、Cloud Spanner と揺れないかを確認してからでよいです。

つまり、今の判断は正しいです。

```text
v1alpha-plan-contracts predefined:
  - no_explicit_sort

future candidates:
  - no_hash_join
  - no_hash_aggregate
```

## 最終判断

今回の状態は、かなり良いです。

主な設計判断はこのままでよいです。

- main v1alpha config に plan contract を入れない。
- `plan-report` は optional Omni-backed workflow とする。
- `plan-report --contracts --check` は experimental contract evaluation とする。
- predefined contract は `no_explicit_sort` だけから始める。
- `Hash Join` / `Hash Aggregate` / user-defined named contract は急がない。
- remediation は出すが、自動 rewrite はしない。

次に詰めるべき点は、機能追加ではなく contract と artifact の安定化です。

優先順位は次です。

1. Spanner Omni Preview / Pre-GA caveat を README に明記する。
2. `plan_source.api` を report に出し、`AnalyzeQuery` と `EXPLAIN` のどちら由来かを明確にする。
3. plan-report の target inclusion / exclusion を固定する。特に `external_query.inner_sql` を対象にするか決める。
4. `--contracts` without `--check`、unknown query、non-Spanner target の exit code を固定する。
5. direct rule grammar の `max_count` を削るか、数え方と default を明記する。
6. digest input と operator normalization version を定義する。
7. optimizer not pinned を pass/fail とは別の environment warning として見せる。

この範囲を固めれば、plan contract は “core config は simple だが、必要な人にはかなり強い review workflow を提供する” という差別化機能として成立すると思います。
