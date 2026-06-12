# `spanner-query-gen` v1alpha plan contract round 2 response

レビューありがとうございます。今回のレビューは、機能の拡大よりも `plan-report` artifact の意味を固定する点が主眼だと理解しました。実装コードやテストログは共有されていない前提のレビューであることも踏まえ、外部に見える README / DESIGN / report fields を中心に反映しました。

## 反映した点

- Spanner Omni execution-plan support に依存する optional workflow であり、review / testing / prototyping 用であって production performance guarantee ではないことを README / DESIGN に明記しました。
- report に `plan_source` を追加しました。
  - `plan_source.backend: omni`
  - `plan_source.api: analyze_query`
  - `plan_source.render_tool: spannerplan`
- report に backend identity placeholder を追加しました。
  - `backend_identity.kind: omni`
  - `backend_identity.version: not_recorded`
  - `backend_identity.image_digest: not_recorded`
- report に normalization version を追加しました。
  - `normalization.operator_tree_version: v1`
  - `normalization.operator_family_mapping_version: v1`
- report に target selection summary を追加しました。
  - `target_summary.included_count`
  - `target_summary.excluded[]`
- README / DESIGN に `plan-report` target selection を明記しました。
  - Spanner catalog の `kind: sql`, `kind: table`, `kind: index` は対象。
  - `kind: external_query` の `inner_sql` は Spanner SQL なので対象。
  - BigQuery outer SQL, BigQuery external dataset bindings, writes は対象外。
- `--contracts` without `--check` の意味を README / DESIGN に明記しました。
  - contract は評価して report に出す。
  - violation があっても exit code は 0。
  - `--check` を付けると violation で non-zero。
- contract file の unknown field rejection を追加しました。
- direct `forbid.operator_family` の `max_count` について、normalized operator nodes after operator-family normalization を数えることと、未指定時は `0` であることを README に明記しました。
- digest の入力を README / DESIGN に明記しました。
  - `sql_sha256`: generated/rendered SQL bytes。
  - `ddl_sha256`: resolved catalog DDL bytes。
  - `operator_tree_sha256`: normalized operator tree v1。
- `contract_summary.environment_warnings` を追加しました。
  - `optimizer_not_pinned`
  - `statistics_package_not_pinned`
  - query-level `query_optimizer_not_pinned`
- remediation に `applies_to` と `confidence` を追加しました。

## CEL と raw QueryPlan について

追加要望を受け、CEL には normalized view だけでなく raw protobuf も渡すようにしました。

- `plan`: raw `spannerpb.QueryPlan`
- `nodes`: raw `[]*spannerpb.PlanNode`

これにより、全 metadata, child links, short representation, execution stats を CEL から参照できます。例えば `google.protobuf.Struct` の metadata は次のように map 風に参照できます。

```cel
nodes.exists(n,
  n.display_name == "Aggregate" &&
  n.child_links.exists(c, c.child_index == 1 && c.type == "Input") &&
  n.metadata["iterator_type"] == "Hash")
```

ただし user-facing contract としては、まず `operators` / `operator_families` の normalized view を推奨し、normalized view で表現できない場合だけ raw protobuf を使う方針にします。raw QueryPlan metadata は backend, optimizer version, plan mode によって揺れる可能性があるためです。

## 取り入れなかった、または一部だけ採用した点

### predefined contract を `no_explicit_sort` のみに戻す案

今回は戻していません。

レビュー時点の添付では `no_explicit_sort` だけから始める判断が妥当でしたが、その後の要望で query plan hints を広く確認し、`GROUP@{GROUP_METHOD=HASH_GROUP} BY` / `STREAM_GROUP`、`JOIN@{JOIN_METHOD=...}` を Spanner Omni integration test で確認する方針になりました。そのため、join / aggregate 系は **experimental predefined** として残しています。

ただし、main v1alpha config には入れていません。`plan-report --contracts` の optional / experimental extension に閉じ込め、production guarantee ではなく described plan environment に対する review contract として扱います。

### plan contract suppressions

まだ入れていません。

レビューの通り、main config の `rules.suppressions` と混ぜるべきではないと考えています。将来必要になった場合は contract file 側に置くのが良いですが、MVP では violation を report するだけで十分です。

### plan-contract schema subcommand

まだ入れていません。

外部 contract file については unknown field rejection を先に入れました。schema 出力は contract grammar がもう少し固まってからで良いと判断しています。

## 現時点の整理

`spanner-query-gen.yaml` は引き続き DDL / SQL / DTO / write helper の宣言に集中します。`plan-report` は optional Omni-backed review artifact であり、`plan-report --contracts --check` は experimental plan contract enforcement です。

この分離を保ったまま、report artifact の source, target selection, digest, normalization, environment warnings を明示したので、CI やレビューで使う場合の解釈は以前より固定できたはずです。
