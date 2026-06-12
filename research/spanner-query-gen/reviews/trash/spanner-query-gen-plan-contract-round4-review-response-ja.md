# `spanner-query-gen` v1alpha plan contract round 4 review response

作成日: 2026-05-06

レビューありがとうございます。今回の指摘は方向転換ではなく、`plan-report` artifact を CI/review 用の public contract として読むための監査性と schema 厳密性を上げるものだと理解しました。その方針で、議論余地が小さいものから反映しました。

## 反映した内容

### `--render-mode=auto` を削除

v1alpha の `plan-report` は structural PLAN 専用に固定しました。

- `--render-mode=plan` のみを受け付けます。
- `--render-mode=profile` は引き続き PROFILE / execution stats 非対応として拒否します。
- `--render-mode=auto` も拒否します。

`auto` を残して requested/resolved mode を分ける案もありましたが、v1alpha では機能面の利点が小さく、PROFILE と混同される余地を残す方が危険だと判断しました。

### report input digest を追加

`plan-report` output に `input` を追加しました。

```yaml
input:
  config_sha256: ...
  contract_file_sha256: ...
  contract_file_path: ...
```

`config_sha256` は常に出します。`contract_file_sha256` は `--contracts` 使用時に出します。`contract_file_path` は review しやすい相対パスだけを出し、`--stable` では省略します。

これで、SQL / DDL / operator tree digest に加えて、どの config と contract file を評価したかを artifact から追跡できます。

### contract evaluation mode を追加

`contract_evaluation_mode` を追加しました。

```yaml
contract_evaluation_mode: none | report_only | check
```

`--contracts` なしは `none`、`--contracts` ありは `report_only`、`--contracts --check` は `check` です。`contract_summary` は `mode != none` のときだけ出す形を schema でも表現しました。

### plan-report schema を条件付きに変更

`plan-report-schema` を更新し、次の invariant を JSON Schema に入れました。

- `contract_evaluation_mode: report_only | check` の場合、`contract_file_version` / `contract_evaluator_version` / `contract_evaluations` / `contract_summary` / `input.contract_file_sha256` を要求します。
- `contract_evaluation_mode: none` の場合、contract result fields は出しません。
- `queries[].status` ごとに required fields を分けました。
  - `ok`: SQL digest、DDL digest、operator tree digest、operators、plan などを要求します。
  - `error` / `skipped`: `error` を要求し、SQL/digest は resolved 済みの場合だけ出せます。

これで、error/skipped に fake digest や placeholder SQL を要求しない schema になりました。

### raw QueryPlan contract の `--check` warning を追加

raw `plan` / `nodes` CEL contract は従来通り `stability.tier: raw_query_plan` と `check_recommended: false` を出します。

加えて、`--check` で raw QueryPlan contract が使われた場合は、`contract_summary.environment_warnings` に次を追加します。

```yaml
raw_query_plan_contract_used_in_check
```

今回は明示 opt-in 必須までは入れていません。v1alpha では警告を強く出し、将来必要なら CLI flag または contract field による opt-in へ移行する余地を残します。

### aggregate unknown failure の説明性を追加

`no_hash_aggregate` / `no_stream_aggregate` が Aggregate classification unknown に依存して fail する場合、`status: fail` のまま次を出すようにしました。

```yaml
failure_kind: classification_unknown
diagnostic_id: aggregate_iterator_type_unknown
```

本当に禁止 operator が見つかった通常違反は `failure_kind: violation` です。これで二値 status は維持しつつ、metadata 不足による安全側 fail と実違反を区別できます。

### `operator_family_counts` を追加

各 query report に `operator_family_counts` を追加しました。

```yaml
operator_family_counts:
  scan: 2
  explicit_sort: 1
```

`operator_families` と `normalized_operators` から再計算可能ですが、review artifact と contract result の突き合わせがしやすくなるため出す価値があると判断しました。CEL でも `operator_family_counts` を参照できます。

追加で、CLI 実行経路でも `operator_family_counts` が空にならないことと、CEL から top-level / `query.operator_family_counts` の両方を参照できることをテストで固定しました。

### `remediation.auto_fix` を `const: false` に固定

report schema 上で `remediation.auto_fix` を `const: false` にしました。

v1alpha の plan contract remediation は advisory text のみで、SQL/config/DDL の自動変更はしません。将来 auto-fix を入れる場合は report schema version を上げる想定です。

### sentinel を schema で明示

`backend_identity.version` / `backend_identity.image_digest` と optimizer pinning sentinel を schema で明示しました。

- `backend_identity.version`: `not_recorded` または実値文字列
- `backend_identity.image_digest`: `not_recorded` または `sha256:<64 hex>`
- `optimizer.version`: `not_pinned` または実値文字列
- `optimizer.statistics_package`: `not_pinned` または実値文字列

shape を object 化する案は見送り、v1alpha では sentinel + schema restriction に留めました。report shape の重さに対して得られる利点がまだ小さいためです。

## 今回は入れなかった内容

### raw QueryPlan contract の明示 opt-in 必須化

`--allow-raw-query-plan-contracts` や `allow_raw_query_plan_in_check: true` は今回は入れていません。

理由は、v1alpha ではまだ CEL/contract grammar 自体が experimental であり、field を増やすよりも `stability` と `environment_warnings` で危険度を明示する方が simple だからです。実際に CI gate として使われ始め、warning だけでは不十分だと分かった時点で opt-in を追加するのが良いと考えています。

### `backend_identity` の object 化

`version.status/value` のような object shape は採用していません。

将来的には良い形ですが、現時点では `not_recorded` sentinel と schema pattern で十分に機械処理できます。report の読みやすさを保つため、今回は破壊的 shape 変更を避けました。

## 追加した検証

以下をテストで固定しました。

- `--render-mode=profile` と `--render-mode=auto` の拒否
- `plan-report` CLI 実行で `input.config_sha256` と `contract_evaluation_mode: none` が出ること
- `plan-report-schema` の出力と checked-in schema の同期
- `operator_family_counts`
- aggregate unknown の `failure_kind` / `diagnostic_id`
- raw QueryPlan contract を check mode で使った場合の warning
- stable output で contract file path を落とす path helper

## 結論

今回の反映で、`plan-report --contracts --check` はまだ experimental ではありますが、CI artifact として次の情報を持つようになりました。

- 評価した config / contract file の digest
- contract evaluation mode
- status-specific query fields
- normalized operator family counts
- raw QueryPlan contract を hard gate に使った場合の warning
- aggregate classification unknown と実違反の区別

これにより、plan contract は「production performance guarantee」ではなく「記述された plan environment に対する review contract」である、という位置付けを artifact と schema の両方でより明確にできたと思います。
