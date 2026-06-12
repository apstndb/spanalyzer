# `spanner-query-gen` plan contract round 11 review

作成日: 2026-05-06

## 前提

このレビューは、共有された Markdown / JSON Schema / 補助ドキュメントだけを対象にしています。実装コード、実行ログ、CI output、golden fixture の実体は共有されていないため、開発側の「実装した」「テストした」という記述は内部作業ツリー上の主張として扱い、外部から確認できる public contract の整合性を中心に見ています。

今回確認した主なファイルは次です。

- `spanner-query-gen-plan-contract-round10-review-response-ja.md`
- `PLAN_CONTRACTS(5).md`
- `PLAN_CONTRACT_CANDIDATES(5).md`
- `QUERY_EXECUTION_OPERATORS_OBSERVATIONS(5).md`
- `spanner-query-gen.plan-contracts.v1alpha.schema(7).json`
- `spanner-query-gen.plan-report.v1alpha.schema(9).json`
- `spanner-query-gen.v1alpha.schema(15).json`

## 総評

かなり良い状態です。round 10 の対応は、plan contract の機能を広げるのではなく、artifact の契約性を締める方向に寄っており、これは正しいと思います。

特に良い点は次です。

- `PLAN_CONTRACTS.md` の minimal report outcomes が、現在の `contract_evaluation` shape に同期されています。
- `contract_rule_result.source` が `use/<predefined>` / `forbid[n]` / `cel` に絞られています。
- `predefined` が free-form string ではなく enum になっています。
- `status: pass | fail | not_evaluated` の invariant が docs / schema / test 方針として整理されています。
- `unknown` family の扱いが明文化され、未知 operator を arbitrary snake-case にせず `unknown` に落とす方針になっています。
- `PLAN_CONTRACTS.md` が Spanner Omni execution plan の Preview / Pre-GA caveat と、PLAN-only / no PROFILE stats の境界を明記しています。
- `QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md` が observation evidence であり、normative contract は fixture / schema / unit test だと明記されています。

ここまで来ると、大きな方向転換は不要です。次に見るべきなのは、**failure artifact を読んだときに、どの PlanNode が原因なのかを即座に追跡できるか**、および **CEL examples が map 欠損などで brittle にならないか** です。

## P0: `operator_family_counts` の CEL semantics を固定する

`PLAN_CONTRACT_CANDIDATES.md` には、次のような CEL 例があります。

```cel
operator_family_counts["hash_join"] == 0 &&
operator_family_counts["push_broadcast_hash_join"] == 0
```

この例は直感的ですが、`operator_family_counts` が「観測された family だけを持つ sparse map」なのか、「全 known family を 0 で埋めた complete map」なのかが明確でないと危険です。CEL の map index は、存在しない key を読んだときの挙動が利用者にとって分かりづらく、CI contract として brittle になりやすいです。

推奨は、`operator_family_counts` を **全 known operator family を含む zero-filled map** として定義することです。43 family 程度なら artifact サイズの問題は小さく、CEL が非常に書きやすくなります。

```yaml
operator_family_counts:
  explicit_sort: 0
  hash_join: 0
  push_broadcast_hash_join: 0
  scan: 2
  ...
```

この場合、次を docs と test で固定するとよいです。

```text
- operator_family_counts contains every operator_family enum value.
- absent family is never represented by missing key; it is represented by 0.
- operator_families remains the compact observed-family list.
```

もし zero-filled map を重いと判断するなら、候補ドキュメントの CEL examples は `operator_family_counts["..."]` ではなく、`operator_families` による existence check に寄せた方が安全です。ただし、`max_count` 系の CEL を書きやすくするなら zero-fill の方が良いと思います。

## P0: rule result に matched PlanNode を出す

現在の `contract_rule_result` は、`operator_family`、`observed_count`、`max_count`、`failure_kind` を持っています。これは summary としては十分ですが、CI で failure を見た利用者は、次に「どの PlanNode が原因だったのか」を探す必要があります。

特に以下のケースでは、rule result から直接 PlanNode に飛べる方がかなり便利です。

- `no_explicit_sort` が fail したときの `Sort` / `Sort Limit`
- `no_hash_join` が fail したときの standalone `Hash Join` または `Push Broadcast Hash Join` wrapper
- `operator_family: unknown` を direct forbid したときの unknown node
- `no_hash_aggregate` / `no_stream_aggregate` で `classification_unknown` が出たときの分類不能 `Aggregate`

提案は、`contract_rule_result` に次のような field を追加することです。

```yaml
results:
- rule: forbid_operator_family
  source: use/no_hash_join
  predefined: no_hash_join
  operator_family: hash_join
  status: fail
  failure_kind: violation
  observed_count: 2
  max_count: 0
  matched_operator_indexes: [3, 7]
```

名前は `matched_operator_indexes` でも `violating_operator_indexes` でもよいですが、`normalized_operators[].index` に対応することを明記してください。

`max_count` が 1 で observed が 3 のようなケースでは、全 matched nodes を出すのがよいです。`max_count` を超えた分だけ出すと、なぜ observed_count がその値なのかが分かりにくくなります。

`classification_unknown` の場合も、次のように原因 node を出せるとかなりレビューしやすいです。

```yaml
results:
- rule: forbid_operator_family
  source: use/no_hash_aggregate
  predefined: no_hash_aggregate
  operator_family: hash_aggregate
  status: fail
  failure_kind: classification_unknown
  diagnostic_id: plan.operator.aggregate-classification-unknown
  matched_operator_indexes: [5]
```

これは機能拡大ではなく、既存 artifact の説明力を上げる変更なので、v1alpha の範囲に入れてよいと思います。

## P1: contract name の uniqueness を runtime validation で固定する

plan contract schema の root には `contracts` の `uniqueItems: true` がありますが、これは object 全体の重複を防ぐだけで、`name` の重複は防げません。

```yaml
contracts:
- name: NoBadPlan
  target: query/A
  use: [no_explicit_sort]
- name: NoBadPlan
  target: query/B
  use: [no_hash_join]
```

このような config は JSON Schema だけでは通り得ますが、report 上では contract name が重複して読みづらくなります。

`target` の重複は、複数観点の contract を同じ query にかけたいので許容でよいです。一方、`name` は identifier であり report の人間向け primary key に近いため、runtime validation で unique にするのがよいと思います。

```text
plan-contract.validation.duplicate-contract-name
```

のような diagnostic ID を用意し、`plan-report --contracts` の前段で reject してください。

## P1: report schema の canonical target / scope をもう少し締める

plan contract file 側では `target` が次の pattern に制限されています。

```text
^query/[A-Za-z_][A-Za-z0-9_]*(#inner)?$
```

これは良いです。一方、plan-report schema 側では、いくつかの target-like field がまだ `minLength` 中心に見えます。

具体的には、可能なら以下も同じ grammar に寄せたいです。

```text
queries[].target_id
contract_evaluations[].target_id
excluded_target.id
```

また、`contract_evaluations[].scope` は `minLength` ではなく、`queries[].scope` と同じ enum にできます。

```json
"scope": {
  "enum": ["query", "external_query.inner"]
}
```

`contract_evaluations[].backend` も、残すなら `enum: ["omni"]` にするか、root の `backend` / `plan_source.backend` と重複するので削るかのどちらかに寄せるとよいです。今のように optional free-form string だと、review artifact の正規化度が少し下がります。

## P1: `source` / `predefined` 対応 invariant を docs に明記する

response では、`source: use/<name>` と `predefined: <name>` の一致を unit test で固定したと書かれています。これは妥当です。JSON Schema だけで `source` の suffix と `predefined` の値一致を表すのは面倒なので、schema + implementation test の分担でよいと思います。

ただし、`PLAN_CONTRACTS.md` にも invariant として一文入れると、artifact consumer が安心できます。

```text
When source has the form use/<name>, predefined is present and equals <name>.
When source is forbid[n] or cel, predefined is absent.
```

現状も例から読み取れますが、明文化しておくと JSON consumer が仕様として扱いやすくなります。

## P1: `unknown` を strict CI で使う例を入れる

`unknown` family の扱いは良くなっています。特に、未知 operator を arbitrary family にせず `unknown` に落とす判断は schema との整合性が高いです。

一方で、predefined contract は無関係な `unknown` の存在だけでは fail しない方針なので、strict CI をしたいユーザー向けに direct forbid の例を `PLAN_CONTRACTS.md` か `PLAN_CONTRACT_CANDIDATES.md` に入れるとよいです。

```yaml
contracts:
- name: NoUnknownOperators
  target: query/ImportantQuery
  forbid:
  - operator_family: unknown
    max_count: 0
```

これは新しい predefined contract を追加する必要はありません。`no_unknown_operator` のような predefined name は、利用実績が出てからで十分です。

## P2: `max_count` の default は schema default ではなく normalizer contract として書く

`forbid.max_count` に JSON Schema `default: 0` が入ったのは良いです。ただし、多くの schema validator は `default` を値補完としては使いません。

すでに `PLAN_CONTRACTS.md` では「omitted means 0」と説明されていますが、もう少し明確に、tool 側の normalization と report output の contract にするとよいです。

```text
If max_count is omitted, the tool treats it as 0 and the report always emits the resolved max_count.
```

report examples では `max_count: 0` が出ているので、方向性はすでに合っています。

## P2: `QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md` は今の位置付けでよい

観測メモの位置付けはかなり良いです。Spanner Omni image digest、`spanemuboost`、`spannerplan`、Go version、optimizer pinning 状態が記録されており、さらに「観測 vocabulary evidence であって normative contract ではない」と明記されています。

この文書は今後も、人間が operator family の追加理由を理解するための補助資料として維持するとよいと思います。operator family enum の source of truth を schema / normalizer registry / fixtures に寄せた判断は正しいです。

## 追加で今はやらなくてよいこと

以下は、まだ入れなくてよいと思います。

- 新しい predefined contract の大量追加
- CEL macro の導入
- user-defined named contract の導入
- PROFILE / execution stats の contract 化
- main `spanner-query-gen.yaml` への plan contract 統合
- raw QueryPlan contract の hard opt-in flag

今は、`plan-report` artifact の読みやすさと schema contract を締める段階です。

## そのまま返せる短いフィードバック案

> round 10 の反映はかなり良いです。特に `source` / `predefined` の制約、status invariant、`unknown` family の扱い、minimal report examples の同期は、CI artifact としての信頼性を上げています。大きな方向転換は不要です。
>
> 追加で一番気になるのは、CEL examples で使っている `operator_family_counts["hash_join"] == 0` の semantics です。`operator_family_counts` が sparse map だと、未出現 family の map index が brittle になります。v1alpha では全 known family を 0 で埋めた complete map として定義するか、CEL examples を `operator_families` ベースに直してください。私は zero-filled map の方が使いやすいと思います。
>
> 次に、contract failure の追跡性を上げるため、`contract_rule_result` に `matched_operator_indexes` を出すことを検討してください。`no_explicit_sort` や `no_hash_join` が fail したとき、どの PlanNode が原因なのかを report だけで追えるようになります。これは新機能追加というより artifact の説明力改善です。
>
> また、JSON Schema の `uniqueItems` は contract name の重複を防げないので、`contracts[].name` は runtime validation で unique にしてください。`target` の重複は同じ query に複数 contract をかける用途があるので許容でよいですが、`name` 重複は report を読みにくくします。
>
> 細かい点として、plan-report schema 側の `target_id` / `scope` / `backend` も contract schema と同じ canonical grammar に寄せるとよいです。`contract_evaluations[].scope` は `query | external_query.inner` enum、`target_id` は `^query/[A-Za-z_][A-Za-z0-9_]*(#inner)?$` にできます。
>
> `unknown` family の direct forbid は良いので、strict CI 用の例として `operator_family: unknown` を `PLAN_CONTRACTS.md` に載せると利用者に伝わりやすいです。新しい predefined contract はまだ不要です。

## 結論

今回の更新で、plan contract surface はかなり収束しています。残る論点は新しい contract を増やすことではなく、artifact consumer が failure を確実に追えるか、CEL が欠損 map で壊れないか、schema と runtime validation の役割分担が明確か、という細部です。

優先順位は次です。

1. `operator_family_counts` を zero-filled map にする、または CEL examples を修正する。
2. `contract_rule_result` に `matched_operator_indexes` を追加する。
3. `contracts[].name` uniqueness を runtime validation で固定する。
4. plan-report schema の `target_id` / `scope` / `backend` をさらに正規化する。
5. `unknown` direct forbid の strict CI example を追加する。

このあたりを締めれば、v1alpha の plan contract は「experimental だがレビュー可能な CI artifact」としてかなり強い形になります。

## 参考

- `spanner-query-gen-plan-contract-round10-review-response-ja.md`
- `PLAN_CONTRACTS(5).md`
- `PLAN_CONTRACT_CANDIDATES(5).md`
- `QUERY_EXECUTION_OPERATORS_OBSERVATIONS(5).md`
- `spanner-query-gen.plan-contracts.v1alpha.schema(7).json`
- `spanner-query-gen.plan-report.v1alpha.schema(9).json`
- Google Cloud: View Spanner Omni execution plans
- Google Cloud: Query execution operators
