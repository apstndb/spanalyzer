# `spanner-query-gen` plan contract round 15 review

作成日: 2026-05-06

## 前提

今回も実装コード、実行ログ、CI output は共有されていないため、README / DESIGN / JSON Schema / `PLAN_CONTRACTS.md` / 観測メモから見える public contract のレビューとして扱います。実装済みと書かれている内容は、外部レビュー上は「内部作業ツリーでの主張」として扱い、ここでは仕様・schema・docs の一貫性を見ます。

## 総評

今回の round 14 response はかなり良いです。新しい predefined contract を増やさず、silent pass 回避、target-level status の可視化、schema/docs の明確化を優先した判断は、v1alpha の CI artifact としての信頼性を上げています。

特に良い反映は次です。

- direct `forbid.operator_family: hash_aggregate` / `stream_aggregate` でも、分類不能な `Aggregate` を silent pass させず、`failure_kind: classification_unknown` と `diagnostic_id: plan.aggregate_classification_unknown` に寄せたこと。
- `matched_operator_indexes` に曖昧な `Aggregate` node index を返す方針を固定したこと。
- duplicate direct `forbid.operator_family` を runtime validation で reject すること。
- top-level `status` を artifact production status として位置づけ、target-level success を `target_summary` と `queries[].status` へ分離したこと。
- `queries[].status: error | skipped` の plan normalization fields を successful plan evidence として扱わない、と docs に明記したこと。
- function hint probe を developer tool に追加し、expression-level change を `nodes` / `json` / `yaml` で見やすくしたこと。

ここまで来ると、大きな方向転換は不要です。残るフィードバックは、ほぼ「schema-valid artifact を下流ツールがどう読むか」「operator vocabulary を誤読しないか」「観測メモと主張が同期しているか」です。

## P1: まだ締める価値がある点

### 1. `target_summary` の算術 invariant を docs と test で固定する

`target_summary.planned/errors/skipped` の追加は良いです。ただし、CI consumer が最初に見る summary なので、次の invariant を `PLAN_CONTRACTS.md` と test で固定するとさらに読みやすくなります。

```text
target_summary.included_count
  == target_summary.planned
   + target_summary.errors
   + target_summary.skipped
```

もし `target_summary.excluded` を report に出すなら、次のどちらかに寄せると良いです。

```yaml
target_summary:
  included_count: 3
  planned: 2
  errors: 1
  skipped: 0
  excluded: []          # always present
```

または、artifact size / verbosity を抑えたいなら:

```yaml
target_summary:
  included_count: 3
  planned: 2
  errors: 1
  skipped: 0
  excluded_count: 0
```

現在の schema では `excluded` が optional に見えるため、downstream は「除外対象がなかった」のか「古い/省略された artifact なのか」を区別しにくいです。v1alpha の review artifact としては、`excluded: []` を always present にする方が扱いやすいと思います。

### 2. `queries[].status: error | skipped` で許される partial fields を列挙する

Option B、つまり `error | skipped` でも SQL rendering や digest などの partial fields を許す判断は理解できます。レビュー上も、どこまで到達したかが見えるのは便利です。

ただし、docs の「only input/error fields are reliable」だけだと、どの field が safe partial field なのかが曖昧です。最低限、次のような whitelist を `PLAN_CONTRACTS.md` または schema description に入れるとよいです。

```text
For queries[].status: error | skipped:
  reliable:
    - name
    - target_id
    - kind
    - scope
    - source
    - sql / sql_sha256, only if SQL rendering succeeded
    - ddl_sha256, only if DDL resolution succeeded
    - error
  not successful plan evidence:
    - operator_tree_sha256
    - operator_families
    - operator_family_counts
    - normalized_operators
    - operator_edges
    - plan
```

さらに、golden fixture で「AnalyzeQuery が失敗したときに stale な `normalized_operators` / `operator_family_counts` が混入しない」ことを固定すると、Option B の安全性が上がります。schema で完全禁止しない方針でも、実装テストで stale plan evidence を防ぐ価値があります。

### 3. `aggregate` / `join` のような generic family の意味を明文化する

`explicit_sort` は derived / umbrella family として明確になりました。一方で、operator family enum には `aggregate` や `join` もあります。ここは誤読されやすいです。

特に `Aggregate` node について、docs は `hash_aggregate` / `stream_aggregate` へ分類できない場合に `classification_unknown` として fail すると説明しています。では、`aggregate` family は何を意味するのかを明記した方がよいです。

可能な整理は、たとえば次のどちらかです。

```text
Option A: aggregate is a generic fallback concrete family.
  - normalized_operators[].family == aggregate when the node is Aggregate but
    hash/stream classification is unavailable.
  - aggregate is not an umbrella count for hash_aggregate + stream_aggregate.
  - direct forbid.operator_family: aggregate catches only generic fallback
    aggregate nodes.
```

```text
Option B: aggregate is a derived umbrella family.
  - operator_family_counts["aggregate"] == hash_aggregate + stream_aggregate + unknown aggregate.
  - normalized_operators[].family never equals aggregate when a more specific
    aggregate family is available.
```

現在の schema では `aggregate` は `concrete_operator_family` に入っているため、読者は Option A に近いものとして解釈しそうです。その場合は、`aggregate` が “generic fallback concrete family” であり、`no_hash_aggregate` / `no_stream_aggregate` の代替ではない、と `PLAN_CONTRACTS.md` に短く書くと安全です。

同じことは `join` にも当てはまります。`join` が generic fallback なのか、特定できない join の concrete family なのか、あるいは umbrella なのかを一文で定義すると、direct `forbid.operator_family` を書く利用者が迷いません。

### 4. `contract_evaluation.backend` は削るか常時 required にする

top-level に `backend` / `plan_source.backend` / `backend_identity` があるため、`contract_evaluation.backend` は少し冗長です。現行 examples では pass/fail に `backend: omni` が出ますが、schema 上は optional に見えます。

v1alpha を simple にするなら、私は `contract_evaluation.backend` は削ってよいと思います。contract evaluation は単一 report backend に属するので、top-level backend だけで十分です。

どうしても per-evaluation backend を残すなら、少なくとも pass/fail では required、not_evaluated でも backend が既知なら emitted、とするなど、出現条件を固定した方がよいです。optional redundant field は downstream consumer にとって「読むべきか無視すべきか」が曖昧になります。

### 5. function hint の観測は observations document にも同期する

round 14 response では `DISABLE_INLINE` function hint probe が追加され、default / `DISABLE_INLINE=FALSE` と `DISABLE_INLINE=TRUE` の plan shape 差分が説明されています。tools README に使い方が入ったのは良いです。

ただし、`QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md` が vocabulary evidence という位置づけなら、function hint probe もそこに短い節を追加するとよいです。

```text
## Function hint probes

Environment: ...

case: function_hint/default
compact shape: ...
notable scalar nodes: SHA512 inline expansion ...

case: function_hint/disable_inline_true
compact shape: ... Compute ...
notable scalar nodes: SHA512 materialized as $x ...
```

これは plan contract の normative surface ではありませんが、「function hint によって `Compute` が増える」という観測は将来の remediation / hint recommendation に関わる可能性があります。tools README だけだと、観測 evidence としては少し弱いです。

## P2: 小さな改善案

### `skipped` に `error` field を要求する naming は少し違和感がある

schema では `queries[].status: skipped` でも `error` が required に見えます。実装上は単純でよいのですが、artifact を読む側には「skipped は error なのか？」という違和感が残ります。

破壊的変更がまだ許されるなら、`error` ではなく `message` または `reason` に寄せる案もあります。

```yaml
status: skipped
reason: non_spanner_target
message: BigQuery outer SQL is not a Spanner plan-report target
```

ただし、すでに `not_evaluated.reason` と `error` の使い分けがあるため、ここは優先度低めです。少なくとも docs に「`error` is the human-readable target acquisition or skip message」と書けば十分かもしれません。

### `diagnostic_id` に軽い pattern を付ける

`plan.aggregate_classification_unknown` のような ID を public artifact に出すなら、schema で次の程度の pattern を付けると typo を減らせます。

```text
^plan\.[a-z0-9_.]+$
```

将来 `config.` や `vet.` などを混ぜる可能性があるなら、もう少し広く:

```text
^[a-z][a-z0-9_]*(\.[a-z0-9_]+)+$
```

でもよいです。

### `excluded_target.id` / `scope` / `source` の required 化を検討する

`excluded_target` は現状 `query` と `reason` が required に見えますが、canonical target ID を中心に据えるなら `id` と `scope` も required にした方が下流ツールは扱いやすいです。

除外対象が BigQuery outer SQL や external dataset 由来で source が存在しない場合があるなら、`source` は optional でよいです。ただし `id` と `scope` は canonical artifact として残す価値があります。

## 追加しなくてよいもの

現時点で、以下はまだ追加しなくてよいと思います。

- 新しい predefined contract。
- positive `require_*` contract。
- function hint に基づく automatic remediation。
- PROFILE / runtime statistics based contract。
- CEL macro / custom recursive helper。
- plan contract の main `spanner-query-gen.yaml` への統合。

ここまでの整理で、`plan-report` は十分に opinionated ですが、main config から分離されているため、core config の simple さは保てています。

## そのまま返せるコメント案

> round 14 の反映はかなり良いです。direct aggregate-family rules で classification unknown を silent pass させないこと、`target_summary` を追加して top-level status と target-level success を分けたこと、duplicate direct forbid を reject することは、CI artifact として正しい tightening だと思います。
>
> 追加で大きい設計変更は不要です。残る主な点は、`target_summary.included_count == planned + errors + skipped` の invariant を docs/test で固定すること、`target_summary.excluded` を always present にするか `excluded_count` を出すこと、`queries[].status: error | skipped` で reliable な partial fields を列挙することです。Option B で partial fields を許す判断は理解しますが、stale `normalized_operators` / `operator_family_counts` が混入しない golden fixture は欲しいです。
>
> また、`explicit_sort` は derived umbrella family としてかなり明確になりましたが、`aggregate` や `join` のような generic family の意味も明文化してください。特に `aggregate` が generic fallback concrete family なのか、derived umbrella family なのかで direct `forbid.operator_family` の解釈が変わります。schema 上は concrete family に見えるので、その前提なら “generic fallback only, not an umbrella for hash/stream aggregate” と書くのが安全です。
>
> `contract_evaluation.backend` は top-level backend と重複するので、v1alpha を simple に保つなら削ってよいと思います。残すなら pass/fail で required など、出現条件を固定してください。
>
> 最後に、function hint probe は tools README に入ったので、`QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md` にも短い observation section を追加すると evidence として強くなります。これは新しい predefined contract にする必要はありませんが、将来の hint recommendation の根拠になります。

## 結論

今回の更新は、plan contract surface をかなりレビュー可能な形に近づけています。次にやるべきことは新しい contract を増やすことではなく、summary count invariant、partial-field semantics、generic operator family vocabulary、function hint observation evidence を締めることです。

特に `aggregate` / `join` の vocabulary は、今のうちに定義しておくと direct `forbid.operator_family` の誤用を防げます。
