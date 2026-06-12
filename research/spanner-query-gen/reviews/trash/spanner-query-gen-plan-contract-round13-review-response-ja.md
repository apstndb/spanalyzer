# `spanner-query-gen` plan contract round 13 response

作成日: 2026-05-06

## 返答概要

round 13 の指摘は、v1alpha の plan-report artifact を外部レビュー可能な contract としてさらに締めるものとして妥当だと判断しました。今回は新しい predefined contract を増やすのではなく、`explicit_sort` の virtual family semantics、rule-specific schema、観測 evidence の同期を優先しました。

## 反映した内容

### `explicit_sort` を derived / virtual family として明文化

`explicit_sort` は concrete PlanNode family ではなく、`full_sort` と `minor_sort` から導出される umbrella family と定義しました。

```text
operator_family_counts["explicit_sort"]
  == operator_family_counts["full_sort"]
   + operator_family_counts["minor_sort"]

operators[i].subtree_family_counts["explicit_sort"]
  == operators[i].subtree_family_counts["full_sort"]
   + operators[i].subtree_family_counts["minor_sort"]
```

この invariant を `PLAN_CONTRACTS.md` と `DESIGN.md` に明記しました。

### concrete family と derived family を schema 上も分離

plan-report schema に `concrete_operator_family` を追加し、次の field は concrete family のみを許可するようにしました。

- `normalized_operators[].family`
- `operator_families[]`

一方、`operator_family_counts`、`operators[].subtree_family_counts`、plan contract file の `forbid.operator_family` は、`explicit_sort` を含む full `operator_family` enum を使います。

### `matched_operator_indexes` と virtual family の関係を固定

`forbid_operator_family` の `matched_operator_indexes` は次の semantics としました。

- concrete family: `operator.family == operator_family` の operator indexes
- derived umbrella family: derived count に寄与する concrete operator indexes
- `explicit_sort`: `full_sort` と `minor_sort` の operator indexes

実装は既にこの動作だったため、今回は docs と schema tests を補強しました。

### `contract_rule_result` schema を rule kind ごとに締めた

plan-report schema で、余分な field を相互に禁止しました。

- `rule: forbid_operator_family`
  - required: `operator_family`, `observed_count`, `max_count`, `matched_operator_indexes`
  - forbidden: `expression`
- `rule: cel`
  - required: `expression`
  - forbidden: `operator_family`, `observed_count`, `max_count`, `matched_operator_indexes`

これにより CI artifact として不要な mixed-shape result が通りにくくなります。

### 観測 evidence を同期

`QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md` に次を明示しました。

- `Minor Sort` / `Minor Sort Limit` の local synthetic probes
- repeated CTE による `SpoolBuild` / `SpoolScan` probe
- `MiniBatchAssign` / `MiniBatchKeyOrder` / `RowCount` が Cloud Spanner optimizer version 5 の plan では出る一方、Spanner Omni 2026.r1-beta の空 synthetic DB ではより単純な back join になること

### direct forbid example の冗長さを解消

`PLAN_CONTRACT_CANDIDATES.md` の例から、`explicit_sort`、`full_sort`、`minor_sort` を同時に指定する形を外しました。

- broad sort policy: `explicit_sort`
- narrower policy: `full_sort` または `minor_sort`

という説明に分けています。

### narrow distributed semi/anti apply の逃げ道を明記

v1alpha では `no_distributed_semi_apply` / `no_distributed_anti_semi_apply` を predefined として追加せず、直接 `forbid.operator_family` を使う方針を `PLAN_CONTRACTS.md` / `DESIGN.md` / `PLAN_CONTRACT_CANDIDATES.md` に明記しました。

### implementation status の表現を更新

operator-family catalog を “intentionally small” と呼ぶのは現状とずれてきたため、次の方針に言い換えました。

```text
bounded, registry-driven, and evidence-based
```

また、backend identity の `not_recorded` と observation documents の manual environment evidence が別 workflow であることも明記しました。

## 残した判断

### `source` / `predefined` equality は runtime test に寄せる

round 13 の P2 と同じく、JSON Schema だけで `source: use/<name>` と `predefined: <name>` の equality を完全表現する必要はないと判断しました。値域と presence / absence は schema、具体的な equality は runtime / regression test の責務とします。

### PROFILE / execution stats は引き続き対象外

PLAN-only の境界は維持します。execution stats を含む PROFILE contract は、今の structural review contract とは別物として扱う方針です。

## 内部検証

開発側では次を確認しました。

```sh
go test ./cmd/spanner-query-gen ./internal/plancontract ./tools/spanner-query-plan-shape
go test ./...
git diff --check
```
