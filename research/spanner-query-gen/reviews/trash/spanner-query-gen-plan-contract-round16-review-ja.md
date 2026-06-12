# `spanner-query-gen` plan contract round 16 review

作成日: 2026-05-06

## 前提

今回も、実装コード・実行ログ・CI output は共有されていない前提で見ています。したがって、このレビューは **public contract / README / DESIGN / JSON Schema / 補助ドキュメントの整合性レビュー** です。

今回共有された最新ファイルと、前回から未更新のため省略されたと思われる最新 schema を前提に見ています。

- `spanner-query-gen-plan-contract-round15-review-response-ja.md`
- `README(35).md`
- `DESIGN(32).md`
- `PLAN_CONTRACTS(10).md`
- `spanner-query-gen.plan-report.v1alpha.schema(14).json`
- 前回共有済みの最新 `spanner-query-gen.plan-contracts.v1alpha.schema(9).json`

## 総評

今回の更新はかなり良いです。新しい contract を増やさず、artifact contract を締める方向に寄せている判断は正しいと思います。

特に良い点は次です。

- `target_summary.included_count == planned + errors + skipped` を invariant として固定したこと。
- `target_summary.excluded` を always present にしたこと。
- `queries[].status: error | skipped` の partial-field semantics を明文化したこと。
- `aggregate` / `join` を derived umbrella ではなく generic fallback concrete family として定義したこと。
- `contract_evaluation.backend` を削除し、top-level `backend` / `plan_source.backend` との重複をなくしたこと。
- `diagnostic_id` に軽い namespaced pattern を入れたこと。
- `DISABLE_INLINE` function hint observation を observation 側に留め、新しい predefined contract にしなかったこと。

また、前回気にしていた plan-contract schema と plan-report schema の `operator_family` enum については、最新の未更新 schema を前提にすると一致しているように見えます。ここは良い状態です。

ここから先は、機能追加ではなく **schema と docs の最後の曖昧さを削る段階** だと思います。

## P0: `queries[].status: error | skipped` では success-only plan fields を schema でも禁止した方がよい

今回の回答では、`error | skipped` の場合に stale な plan normalization fields が混入しないことを実装テストで固定した、と説明されています。これは良いです。

ただし、最新 plan-report schema を読む限り、`status: error | skipped` のときに `operator_tree_sha256` / `operator_families` / `operator_family_counts` / `normalized_operators` / `operator_edges` / `plan` などを **禁止する条件** はまだ入っていないように見えます。schema は `error` を required にするだけで、success-only fields の混入自体は許してしまう形に見えます。

public artifact schema としては、ここも schema で締めた方がよいです。

```jsonc
if status == "error" or "skipped":
  forbid:
    - operator_tree_sha256
    - operator_families
    - operator_family_counts
    - normalized_operators
    - operator_edges
    - plan
```

`classification_warnings` も plan normalization 由来なら禁止でよいと思います。もし acquisition failure の warning として使うなら、別 field 名に分けた方が読みやすいです。

`sql` / `sql_sha256` / `ddl_sha256` については、失敗段階によって出せる場合と出せない場合がありそうです。そのため、どちらかに寄せるのがよいです。

### 選択肢 A: SQL は error/skipped でも必ず出す

この場合は schema で `sql` / `sql_sha256` も required にします。

### 選択肢 B: SQL は error/skipped では optional にする

この場合は docs の文言を次のように変えると正確です。

```text
For status: error | skipped, target identity and error are reliable.
SQL, SQL digest, and DDL digest are reliable only when present.
Plan normalization fields are absent and must not be interpreted as successful plan evidence.
```

現実には、SQL rendering 前に失敗するケースもあり得るので、私は **選択肢 B + success-only plan fields を schema で禁止** が一番安全だと思います。

## P1: `target_summary.excluded` と `queries[].status: skipped` の関係をもう少し明確にしたい

`PLAN_CONTRACTS.md` では、`target_summary.included_count` は `planned + errors + skipped` と定義されています。つまり `skipped` は `queries[]` に含まれる target です。

一方で、`target_summary.excluded` の説明に「no target was excluded or skipped from Spanner plan acquisition」のような表現があり、ここだけ読むと `skipped` が `excluded[]` にも入るのか、`queries[]` に入るのかが少し曖昧です。

ここは用語を分けた方が安全です。

```text
included target:
  appears in queries[]
  counted by target_summary.included_count
  status is ok | error | skipped

excluded target:
  does not appear in queries[]
  appears only in target_summary.excluded
  not counted by target_summary.included_count
```

そして `target_summary.excluded` の説明は、例えば次のようにすると誤解が減ります。

```text
target_summary.excluded is always present. It is [] when no targets were excluded from the report target set.
Skipped targets are included in queries[] with status: skipped and counted in target_summary.skipped.
```

## P1: join-specific contract でも `classification_unknown` を検討した方がよい

今回、`aggregate` は generic fallback concrete family として定義され、`hash_aggregate` / `stream_aggregate` を禁止する contract では、分類不能な `Aggregate` node がある場合に `classification_unknown` で fail する設計になっています。これはかなり良いです。

同じ考え方を `join` にも適用するかを検討した方がよいと思います。

現在の説明では、`join` は「join-like PlanNode だが `hash_join`, `merge_join`, `apply_join`, `push_broadcast_hash_join` などに分類できない fallback」とされています。この場合、例えば `no_hash_join` を評価するときに generic `join` が存在していたら、本当に hash join ではないと証明できていません。

安全側に倒すなら、join-specific contract でも次のような扱いがよいです。

```text
When a contract forbids a specific join family and generic join nodes exist:
  status: fail
  failure_kind: classification_unknown
  diagnostic_id: plan.join_classification_unknown
  matched_operator_indexes: [generic join node indexes]
```

対象になり得るのは、少なくとも以下です。

```text
no_hash_join
no_standalone_hash_join
no_push_broadcast_hash_join
no_apply_join
no_standalone_apply_join
no_distributed_cross_apply
no_merge_join

direct forbid.operator_family:
  hash_join
  merge_join
  apply_join
  push_broadcast_hash_join
  distributed_cross_apply
  ...
```

もちろん、これはやや保守的です。もし v1alpha では generic `join` を silent fallback として扱うなら、それでも構いません。ただし、その場合は `PLAN_CONTRACTS.md` に次のような注意書きが必要です。

```text
Join-specific predefined contracts only match classified join families.
If you want to fail on unclassified join-like nodes, add:
  forbid.operator_family: join
```

私は aggregate と同じく `classification_unknown` に寄せる方が、CI contract としては安全だと思います。

## P1: `excluded_target.reason` は enum または namespaced pattern にした方がよい

`excluded_target` は `id`, `query`, `scope`, `reason` が required になっており、これは良いです。ただし、`reason` は現在 `minLength` 程度に見えます。

machine-readable artifact として使うなら、`reason` は enum か pattern にした方が downstream consumer が扱いやすくなります。

例えば enum なら次のような候補です。

```text
bigquery_outer_sql
bigquery_external_dataset
write_not_supported
non_spanner_catalog
unsupported_query_kind
```

将来増える可能性を残したいなら、diagnostic ID と同様に namespaced pattern でもよいです。

```text
^[a-z][a-z0-9_]*(\.[a-z0-9_]+)+$
```

今の `diagnostic_id` を namespaced pattern にした判断と揃えるなら、`excluded_target.reason` も同じ思想で締めるとよいです。

## P2: `contract_rule_result.source` / `predefined` equality は runtime test でよいが、docs に一文あると安心

schema では `source: use/<name>` のとき `predefined` が required になり、`source: forbid[n] | cel` のとき `predefined` が禁止されています。これは良いです。

ただし、JSON Schema だけでは `source: use/no_hash_join` と `predefined: no_explicit_sort` のような不一致までは完全に表現していないと思います。これは runtime validation / regression test で固定する判断でよいです。

`PLAN_CONTRACTS.md` にすでに近い説明はありますが、さらに明確にするなら次の一文を追加するとよいです。

```text
The equality between source use/<name> and predefined: <name> is a runtime invariant tested by the implementation; the schema restricts the value domains.
```

## P2: `target_summary.excluded_count` は必須ではないが、CI artifact としてはあると便利

前回は `excluded` を always present にする方向で十分だと思いました。今回も大きな問題ではありません。

ただ、`included_count` / `planned` / `errors` / `skipped` が count として出ているなら、`excluded_count` もあると機械処理では便利です。

```yaml
target_summary:
  included_count: 3
  planned: 2
  errors: 0
  skipped: 1
  excluded_count: 2
  excluded:
  - ...
```

ただし、これは必須ではありません。`len(excluded)` で十分、という方針なら今のままで問題ありません。

## P2: `status: no_targets` の target_summary shape を例示するとよい

top-level `status` は artifact production status で、`ok | no_targets` です。この整理は良いです。

ただ、`no_targets` の場合の `target_summary` がどうなるかは、利用者が CI output を読むときに気になります。

例えば docs に短い例があるとよいです。

```yaml
status: no_targets
target_summary:
  included_count: 0
  planned: 0
  errors: 0
  skipped: 0
  excluded: []
queries: []
contract_evaluation_mode: none
```

`--contracts` が指定されていて target が全部 unavailable だった場合は top-level `ok` なのか `no_targets` なのか、`contract_summary.status` が fail になるのか、という関係も一文あると安全です。

## このままでよい点

### plan-contract schema と plan-report schema の enum 同期

最新の未更新 schema まで含めると、`operator_family` enum は plan-report schema と plan-contract schema で一致しているように見えます。ここは良いです。今後も generation / unit test で固定するだけでよいと思います。

### `contract_evaluation.backend` の削除

top-level `backend` / `plan_source.backend` と重複するので、削除でよいです。CEL input の `backend` が report-level backend であることも自然です。

### `aggregate` と `join` を fallback concrete family とする整理

これは良いです。`aggregate` を `hash_aggregate + stream_aggregate` の umbrella にしない判断は、`explicit_sort` との違いを明確にします。

### function hint observation を contract にしない判断

`DISABLE_INLINE` は今は observation / hint recommendation の材料であって、predefined contract にするには早いと思います。今のように observation 側に残す判断でよいです。

### PROFILE / runtime statistics を入れない判断

引き続き良いです。`plan-report` は PLAN-only structural artifact に閉じる方が、v1alpha の CI contract として読みやすいです。

## 結論

今回の更新で、plan contract 周りはかなり収束しています。大きな設計変更は不要です。

次に直すなら、優先順位は次です。

1. **`queries[].status: error | skipped` で success-only plan fields を schema でも禁止する。**
2. **`target_summary.excluded` と `skipped` の用語を分離する。**
3. **join-specific contract でも `classification_unknown` を使うか、generic `join` fallback の扱いを明記する。**
4. **`excluded_target.reason` を enum または namespaced pattern にする。**
5. **`no_targets` の minimal report shape を docs に追加する。**

このあたりを締めれば、`plan-report --contracts` は v1alpha の optional / experimental workflow としてかなりレビューしやすい形になります。
