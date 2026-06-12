# `spanner-query-gen` v1alpha plan contract review response

レビューありがとうございます。方針としては、提案どおり **main v1alpha config には plan contract を入れず、`plan-report` の optional / experimental extension として扱う** 方向を採用しました。

## 反映した点

- `plan-report --stable` を追加しました。
  - 現在の report は host path、temporary database ID、plan acquisition timestamp のような volatile metadata を出していません。
  - `--stable` は snapshot-oriented output mode が要求されたことを report に記録しつつ、semantic fields は残します。
- `plan-report` の empty-target semantics を固定しました。
  - Spanner query target が 0 件の場合は `status: no_targets` と warning を出します。
  - default exit code は 0 です。
  - `--require-targets` を指定した場合は non-zero にします。
- `plan-report` に external contract file を追加しました。

```yaml
version: v1alpha-plan-contracts

contracts:
- name: SingerIndexLookupPlan
  query: ScanSingerIDsFast
  backend: omni
  use:
  - no_explicit_sort
```

- CLI は次の形です。

```sh
spanner-query-gen plan-report \
  --config spanner-query-gen.yaml \
  --contracts plan-contracts.yaml \
  --check \
  --output yaml
```

- `--check` は `--contracts` を要求し、contract violation があれば report 出力後に non-zero で終了します。
- experimental predefined contract は `no_explicit_sort`, `no_hash_join`, `no_push_broadcast_hash_join`, `no_apply_join`, `no_merge_join`, `no_hash_aggregate`, `no_stream_aggregate` にしました。
- direct rule としては `forbid.operator_family` / `max_count` を受けられるようにしています。
- user-defined rule として `cel` を受けられるようにしました。CEL input は raw renderer output ではなく、`backend`, `query`, `operators`, `operator_families`, `optimizer` からなる normalized report view を主対象にします。加えて advanced use case では raw `spannerpb.QueryPlan` を `plan`、raw `[]*spannerpb.PlanNode` を `nodes` として参照できるようにしました。これにより metadata, child links, short representation, execution stats も CEL から検査できます。
- `backend: omni` は contract 側で受け取り、report backend と一致しない場合はエラーにします。
- plan report に以下の review fields を追加しました。
  - `status`
  - `sql_sha256`
  - `ddl_sha256`
  - `operator_tree_sha256`
  - `operator_families`
  - `normalized_operators`
  - `optimizer.version: not_pinned`
  - `optimizer.statistics_package: not_pinned`
  - query-level `optimizer_not_pinned` / `plan_environment_notes`
- contract evaluation は `contract_summary` と `contract_evaluations` に出します。
- violation には remediation text を出します。自動 rewrite はしません。
- contract name duplicate は contract file 読み込み時に拒否します。
- README / DESIGN に、`plan-report` が optional Omni-backed workflow であり primary DDL-first workflow ではないことを追記しました。
- README の `result.cardinality` 説明は、v1alpha では generated query output が DTOs and SQL constants に留まることが分かるように言い換えました。
- JSON Schema の `uniqueItems` は名前一意性を保証しないため、semantic name uniqueness は planning diagnostics で検証する旨を README に明記しました。

## QueryPlan operator family について

Spanner 側の正しい参照先は [Query execution operators](https://docs.cloud.google.com/spanner/docs/query-execution-operators) として扱います。

このページに合わせ、`no_explicit_sort` は `Sort` / `Sort Limit` を explicit sort family として検出します。一方で、`Distributed Merge Union` は sorted result を作る operator ですが explicit `Sort` / `Sort Limit` ではないため、別 family の `distributed_merge_union` として扱います。

これにより、description に `sorted result` のような文字列が出るだけで `explicit_sort` と誤分類しないようにしました。`Distributed Merge Union` を sort violation にしない回帰テストも追加しています。

また、`Aggregate` node については QueryPlan metadata の `iterator_type: Hash` / `iterator_type: Stream` を見て `hash_aggregate` / `stream_aggregate` に分類する実験的な実装を追加しました。Spanner の group hint である `GROUP@{GROUP_METHOD=HASH_GROUP} BY` と `GROUP@{GROUP_METHOD=STREAM_GROUP} BY` を使い、Omni integration test で両方の operator family、predefined contract violation、CEL contract evaluation を確認しています。

## まだ入れなかった点

- join 系 predefined contract は追加しました。
  - `JOIN@{JOIN_METHOD=HASH_JOIN}` / `APPLY_JOIN` / `MERGE_JOIN` / `PUSH_BROADCAST_HASH_JOIN` を Omni integration test で検証し、それぞれ `hash_join`, `apply_join`, `merge_join`, `push_broadcast_hash_join` として分類します。
  - ただし optimizer-sensitive な contract であることは変わらないため、production guarantee ではなく、特定 optimizer/backend/version の review artifact として扱う方針です。
- user-defined named contract definitions は入れていません。
  - 最初は external file + predefined/direct forbid の最小形に留めました。
- `require.operator_family` は入れていません。
  - `forbid` より brittle なので、first MVP からは外しました。
- backend version / image digest はまだ report に入れていません。
  - `--stable` で落とすべき volatile metadata と、残すべき backend identity metadata の境界をもう少し整理してから入れるのがよいと考えています。
- optimizer version / statistics package を config から pin する機能はまだありません。
  - 現状は `not_pinned` として report に残し、plan environment の制約として明示するだけです。

## 確認したこと

- `go test -count=1 ./...`
- `mise x -- golangci-lint run --timeout=5m`
- `go test -count=1 -tags=integration -run '^TestIntegration' ./cmd/spanner-query-gen`
- `SPANEMUBOOST_ENABLE_OMNI_TESTS=1 go test -count=1 -v -tags='integration omni' -run '^TestIntegration.*Omni' ./cmd/spanner-query-gen`
- `git diff --check`

Omni integration では、生成された index query が Sort を含まないこと、ORDER BY を index order と違う形にすると Sort が出ること、NULL_FILTERED index predicate を外すと AnalyzeQuery が失敗すること、`plan-report` が同じ Omni runtime 上で動き contract evaluation まで通ることを確認しました。

## 現時点の判断

plan contract は強い機能ですが、optimizer-sensitive であり、core generator の simple な public config に混ぜるにはまだ早いと判断しました。

そのため当面は次の線引きにします。

- `generate` / `check` / `explain-plan` / `vet` / `config-schema`: DDL-first static workflow。
- `plan-report`: optional Omni-backed review artifact。
- `plan-report --contracts --check`: optional experimental plan contract evaluation。

この形なら、opinionated な plan contract を実験しつつ、v1alpha main config の単純さを維持できます。
