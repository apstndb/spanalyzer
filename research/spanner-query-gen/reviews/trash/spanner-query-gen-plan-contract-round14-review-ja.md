# `spanner-query-gen` plan contract round 14 review

作成日: 2026-05-06

## 前提

今回も、実装コード・実行ログ・CI output は共有されていない前提でレビューする。したがって、ここでの評価対象は次に限定する。

- public contract としての README / DESIGN / PLAN_CONTRACTS / schema
- `plan-report` artifact の読みやすさと CI 利用時の意味
- Spanner Omni / QueryPlan operator observation documents の位置づけ

結論として、前回までの大きな懸念はかなり閉じている。特に `explicit_sort` の virtual family 化、`concrete_operator_family` と `operator_family` の分離、`matched_operator_indexes` の意味固定、rule-kind ごとの schema tightening は良い進展である。次に見るべき点は、新機能追加ではなく、**silent pass を避ける分類不能時の扱い**と、**plan-report 全体の status / target error の表現**だと思う。

## 良い更新

### 1. `explicit_sort` の derived / virtual family 化は正しい

`explicit_sort` を concrete PlanNode family ではなく、`full_sort + minor_sort` の umbrella family として定義したのは良い。これにより、`normalized_operators[].family` と `operator_families[]` は concrete family のみ、`operator_family_counts` / `subtree_family_counts` / `forbid.operator_family` は derived family も含める、という整理ができている。

この設計は、ユーザーが `no_explicit_sort` を使うときの期待に合っている。`Sort` / `Sort Limit` と `Minor Sort` / `Minor Sort Limit` を一括で拒否したい場合は `explicit_sort`、狭く見たい場合は `full_sort` または `minor_sort`、という使い分けができる。

### 2. `matched_operator_indexes` の semantics がかなり良くなった

`forbid_operator_family` では常に `matched_operator_indexes` を出し、concrete family ならその family の operator indexes、derived umbrella family なら derived count に寄与した concrete operator indexes を返す、という方針はレビューしやすい。特に `explicit_sort` 違反時に、実際には `full_sort` なのか `minor_sort` なのかを `normalized_operators[]` から追える。

### 3. `PLAN_CONTRACTS.md` の “minimal report outcomes” が schema-valid に寄ってきた

`status: pass` / `status: fail` / `status: not_evaluated` の最小例に `stability`、`target_id`、`query`、`scope`、`results` などが入るようになった点は良い。前回の “abridged snippet なのか schema-valid snippet なのか分からない” という問題はかなり解消された。

### 4. 観測 document の位置づけが良い

`QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md` が “observed vocabulary evidence” であり、normative contract は fixture / generated schema / unit test だと明記されているのは良い。さらに、Spanner Omni image digest、`spanemuboost`、`spannerplan`、Go version、optimizer pinning 状態が載っているため、将来 operator shape が変わったときに比較しやすい。

`OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md` も有用である。optimizer version と `ALLOW_DISTRIBUTED_MERGE` によって sort placement や join family が変わることを観測として残しているので、`plan-report` が “production performance guarantee” ではなく “described environment に対する plan-shape contract” であることを補強している。

## 追加フィードバック

### P1. aggregate classification unknown は direct `forbid` でも silent pass させない方がよい

今の docs では、`Aggregate` node が `iterator_type: Hash` / `Stream` を持つ場合に `hash_aggregate` / `stream_aggregate` へ分類し、metadata がない場合は `classification_warnings` を出し、predefined aggregate contract は `failure_kind: classification_unknown` で fail する、という方針に見える。

この方針自体は良い。ただし、**direct `forbid.operator_family: hash_aggregate` / `stream_aggregate` でも同じ保護をかけるべき**だと思う。

理由は単純で、ユーザーが direct rule で次のように書いた場合も、意味は “Hash Aggregate がないことを確認したい” だからである。

```yaml
contracts:
- name: NoHashAggregate
  target: query/AggregateBySinger
  forbid:
  - operator_family: hash_aggregate
    max_count: 0
```

ここで実際の plan に `Aggregate` があるが `iterator_type` が欠落しており、`hash_aggregate` か `stream_aggregate` か判断できない場合、`observed_count: 0` で pass すると危険な false pass になる。

推奨 invariant は次。

```text
If a forbid_operator_family rule targets hash_aggregate or stream_aggregate,
and the plan contains an Aggregate node whose iterator_type is missing or unknown,
then the rule result should be:
  status: fail
  failure_kind: classification_unknown
  diagnostic_id: plan.aggregate_classification_unknown
  matched_operator_indexes: indexes of ambiguous Aggregate nodes
```

この invariant は `source: use/no_hash_aggregate` でも `source: forbid[n]` でも同じにした方がよい。predefined と direct rule で安全性が違うと、CI 利用者にとって分かりにくい。

代替案として、direct `forbid` は exact match のみで、曖昧な `aggregate` はユーザーが別途 `forbid.operator_family: aggregate` で拾う、という設計も可能ではある。ただ、その場合は `PLAN_CONTRACTS.md` に “direct `hash_aggregate` は unclassified aggregate を fail しない” と明記する必要があり、私はこの挙動は少し危険だと思う。

### P1. `queries[].status` と top-level `status` の関係をもう少し明確にしたい

plan-report schema では top-level `status` は `ok | no_targets` に見える。一方で、各 query には `status: ok | error | skipped` がある。これは合理的だが、CI artifact として読むと次のケースが少し曖昧になる。

```text
status: ok
queries:
- name: ImportantQuery
  status: error
  error: ...
```

この場合、top-level `status: ok` は “report command が成功した” という意味であり、“すべての target が plan 取得に成功した” ではないはず。そうであれば、README / PLAN_CONTRACTS / schema description のどこかで明示した方がよい。

simple にするなら、top-level status enum を増やすより、`target_summary` に counts を足すのがよいと思う。

```yaml
target_summary:
  included_count: 3
  planned: 2
  errors: 1
  skipped: 0
```

または、既存の `queries[].status` を維持しつつ、`PLAN_CONTRACTS.md` に次のように書く。

```text
Top-level status reports whether the plan-report artifact was produced.
It does not mean every query target was successfully planned.
Use queries[].status and target_summary counts for target-level success.
```

`--check` と contract evaluation がある場合は `not_evaluated` が fail summary に入るのでまだ分かりやすい。一方、contracts なしの `plan-report` では target error をどう扱うかが少し見えにくい。

### P1. `queries[].status: error | skipped` の plan fields を schema で禁止するか、partial output として明文化する

現在の schema は `queries[].status: ok` のとき plan fields を required にしている一方、`status: error` / `skipped` のときに plan fields を明示的に forbidden にはしていないように見える。

どちらの設計でもよいが、public artifact としては次のどちらかに寄せた方が読みやすい。

#### Option A: error/skipped では plan fields を禁止する

```text
status: error or skipped
  required: error
  forbidden: sql_sha256, ddl_sha256, operator_tree_sha256,
             operator_families, operator_family_counts,
             normalized_operators, operator_edges, plan
```

これは一番読みやすい。

#### Option B: partial output を許す

たとえば SQL rendering までは成功したが AnalyzeQuery が失敗した場合、`sql` / `sql_sha256` は残したいかもしれない。その場合は `status: error` で許される partial fields を docs と schema description に明記する。

今のままだと、`status: error` なのに stale な `operator_tree_sha256` が残っていても schema 上は通り得るので、CI artifact として少し弱い。

### P2. report artifact の name fields も identifier pattern に寄せてよい

contract file 側の `name` と `target` は identifier grammar に寄っている。一方、plan-report schema の `query.name`、`contract_evaluation.name`、`contract_evaluation.query` は `minLength` のみのように見える。

生成元の config schema が identifier を保証するなら実行上は問題ないが、artifact schema としては次のように同じ pattern に寄せるとさらに締まる。

```text
^[A-Za-z_][A-Za-z0-9_]*$
```

`target_id` はすでに `^query/[A-Za-z_][A-Za-z0-9_]*(#inner)?$` に制約されているので、関連する name fields も同じ vocabulary にすると editor / CI validation が少し強くなる。

### P2. direct `forbid` 内の duplicate semantic rules は runtime validation で reject してよい

schema の `uniqueItems` は object-level uniqueness なので、次のような semantic duplicate は通り得る。

```yaml
forbid:
- operator_family: hash_join
  max_count: 0
- operator_family: hash_join
  max_count: 1
```

これは意図が曖昧なので、runtime validation で同一 contract 内の duplicate `forbid.operator_family` を reject してよいと思う。

```text
diagnostic_id: plan_contract.duplicate_forbid_operator_family
```

v1alpha では “reviewable artifact” を優先しているので、重複指定を merge するより reject の方が良い。

### P2. `matched_operator_indexes` の順序 invariant を schema ではなく docs/test で固定する

`matched_operator_indexes` は “ascending order” と docs に書かれており、これは良い。JSON Schema で昇順まで表す必要はないので、unit test / golden fixture で固定すれば十分。

virtual family の `explicit_sort` では、`matched_operator_indexes` が `full_sort` / `minor_sort` の concrete operator indexes を返すため、docs に one small fail example を足すとさらに分かりやすい。

```yaml
results:
- rule: forbid_operator_family
  source: use/no_explicit_sort
  predefined: no_explicit_sort
  operator_family: explicit_sort
  status: fail
  observed_count: 2
  max_count: 0
  matched_operator_indexes: [2, 5]

normalized_operators:
- index: 2
  family: full_sort
- index: 5
  family: minor_sort
```

### P2. optimizer version matrix は “recommendation source” ではなく “risk evidence” として扱うのがよい

`OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md` はかなり有用だが、ここから “古い optimizer version を推奨する” 方向には行かない方がよい。matrix 自体が示している通り、optimizer version によって available hints や plan families が変わる。たとえば `PUSH_BROADCAST_HASH_JOIN` は古い version では使えない場合がある。

したがって、この文書の位置づけは次が良い。

```text
Optimizer matrix is evidence that plan contracts are environment-bound.
It helps choose and document a pinned optimizer environment.
It is not a recommendation to pin to a globally older or newer version.
```

この方針は、現在の `--require-optimizer-pinning` と相性がよい。

## 今は増やさなくてよいもの

ここまでの plan contract surface は、v1alpha として十分に opinionated になっている。次のものはまだ増やさなくてよいと思う。

- `no_distributed_semi_apply` / `no_distributed_anti_semi_apply` predefined
- positive `require_stream_aggregate`
- `no_back_join`
- scan target / seekability direct rules
- PROFILE / execution_stats contracts
- remediation auto-fix

これらは候補としては良いが、今は schema / fixture / docs の同期を優先する段階だと思う。

## まとめ

今回の更新はかなり良い。`explicit_sort` の virtual family 化と rule-specific schema tightening により、`plan-report` artifact は CI で読める contract に近づいている。

残る主要な論点は次の 3 つ。

1. `hash_aggregate` / `stream_aggregate` の direct forbid でも classification unknown を silent pass させないこと。
2. top-level `status` と `queries[].status` の意味を明確化し、target error counts を report で見やすくすること。
3. `queries[].status: error | skipped` の partial plan fields を禁止するか、許すなら明文化すること。

これらを締めれば、plan contract 周りは v1alpha の public review artifact としてかなり安定して見える。
