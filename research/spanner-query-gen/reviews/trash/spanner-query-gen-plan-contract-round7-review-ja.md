# `spanner-query-gen` plan contract round 7 review

対象: 2026-05-06 時点の `spanner-query-gen-plan-contract-round6-review-response-ja.md`, `README(26).md`, `DESIGN(25).md`, `spanner-query-gen.v1alpha.schema(13).json`, `spanner-query-gen.plan-contracts.v1alpha.schema(2).json`, `spanner-query-gen.plan-report.v1alpha.schema(4).json`, `PLAN_CONTRACT_CANDIDATES.md`, `IMPLEMENTATION_STATUS(1).md`。

実装コード・実行ログ・テスト出力は共有されていないため、このレビューは **public contract / README / DESIGN / schema / response の整合性レビュー** として扱う。

## 総評

かなり良い状態まで収束している。

今回の更新で特に良いのは、以下の点。

- `plan-report` は main `spanner-query-gen.yaml` に入れず、optional / experimental な Omni-backed workflow として分離されている。
- `--render-mode` を外し、v1alpha は structural PLAN 専用、report には `render_mode: PLAN` を出すだけになった。
- plan contract entry は canonical `target` のみに寄り、`query` / `scope` / `backend` alias が消えた。
- contract entry が exactly one of `use` / `forbid` / `cel` になった。
- `no_hash_join` が直感に合わせて広義化され、`no_standalone_hash_join` が追加された。
- `no_apply_join` も同じ整理になった。
- `not_evaluated` が `fail` と分離され、target availability failure と operator contract violation が report 上で混ざらなくなった。
- `optimizer.requested` / `optimizer.effective` の分離が入った。
- `IMPLEMENTATION_STATUS.md` で、実装済み surface と design-only / partial work が分けられている。

この方向性は維持でよい。大きな設計変更は不要。次は **CI artifact としての schema 厳密性** と **README の過剰な詳細の分離** を詰める段階だと思う。

## P0: Spanner Omni execution-plan support の Preview / Pre-GA caveat を README に入れる

round 6 response でも「まだ反映していない」と書かれているが、これは早めに入れた方がよい。

`plan-report` は optional とはいえ、CI gate として使う想定が出ている。Spanner Omni execution-plan support が Preview / Pre-GA 扱いで、development / testing / prototyping / demonstration 用であることは README の `plan-report` section に短く明記すべき。

提案文:

```md
`plan-report` depends on Spanner Omni execution-plan support, which is currently
Preview / Pre-GA in Google Cloud documentation. Treat plan contracts as
review/testing/prototyping artifacts for a described plan environment, not as
production performance guarantees or commercial data-processing guarantees.
```

すでに README は「production performance guarantee ではない」と書けているが、Preview / Pre-GA wording を直接入れると利用者の誤解がかなり減る。

## P0: `PLAN_CONTRACT_CANDIDATES.md` に旧 grammar が残っている

`PLAN_CONTRACT_CANDIDATES.md` の例にまだ旧 `query:` 形式が残っている。

現在の public plan contract grammar は canonical `target` のみなので、候補集でも旧 alias を使わない方がよい。

現在残っている例:

```yaml
contracts:
- query: LookupSingerByName
  forbid:
  - operator_family: explicit_sort
```

```yaml
contracts:
- query: AggregateBySinger
  use:
  - no_hash_aggregate
```

修正案:

```yaml
contracts:
- name: LookupSingerByNamePlan
  target: query/LookupSingerByName
  forbid:
  - operator_family: explicit_sort
```

```yaml
contracts:
- name: AggregateBySingerPlan
  target: query/AggregateBySinger
  use:
  - no_hash_aggregate
```

候補集は将来機能のメモではあるが、README の Documentation Map から参照されるので、古い grammar が残ると adoption 時に混乱しやすい。

## P1: plan-report schema の `contract_evaluation` を status-specific に締める

`not_evaluated` を導入した方向は良い。ただし、JSON Schema はまだ少し緩い。

現状の `contract_evaluation` はおおむね次の形に見える。

```json
{
  "required": ["name", "target_id", "status", "stability"],
  "properties": {
    "status": { "enum": ["pass", "fail", "not_evaluated"] },
    "reason": { "enum": ["target_not_found", "target_error"] },
    "results": { ... },
    "error": { ... }
  }
}
```

このままだと、schema 上は以下が通り得る。

- `status: pass` なのに `results` がない。
- `status: fail` なのに `results` がない。
- `status: not_evaluated` なのに `reason` がない。
- `status: pass` なのに `reason: target_not_found` が付いている。
- `status: not_evaluated` なのに `results` が付いている。

planner / reporter が正しく出していれば実害は小さいが、`plan-report` schema を CI artifact の public contract とするなら schema 側でも締めた方がよい。

提案:

```json
{
  "allOf": [
    {
      "if": { "properties": { "status": { "const": "pass" } }, "required": ["status"] },
      "then": {
        "required": ["results"],
        "not": { "anyOf": [ { "required": ["reason"] }, { "required": ["error"] } ] }
      }
    },
    {
      "if": { "properties": { "status": { "const": "fail" } }, "required": ["status"] },
      "then": {
        "required": ["results"],
        "not": { "required": ["reason"] }
      }
    },
    {
      "if": { "properties": { "status": { "const": "not_evaluated" } }, "required": ["status"] },
      "then": {
        "required": ["reason"],
        "not": { "required": ["results"] }
      }
    },
    {
      "if": { "properties": { "reason": { "const": "target_error" } }, "required": ["reason"] },
      "then": { "required": ["error"] }
    }
  ]
}
```

`target_not_found` の場合も human-readable message が欲しければ、`error` ではなく `message` にする方が自然かもしれない。`error` は target が存在したが plan acquisition / rendering に失敗した場合に限定した方が読みやすい。

## P1: `contract_rule_result` も pass/fail で不要 field を禁止する

`contract_rule_result` は `status: fail` のとき `failure_kind` を要求しているので良い。ただし、schema 上は `status: pass` に `failure_kind` や `diagnostic_id` が付くことを禁止していないように見える。

CI artifact としては、pass result と failure result の形が分かれている方が downstream tooling が簡単になる。

提案:

```text
status: pass
  require: rule
  forbid: failure_kind, diagnostic_id

status: fail
  require: failure_kind

failure_kind: classification_unknown
  require: diagnostic_id

failure_kind: violation
  diagnostic_id optional or forbidden;どちらかを明示
```

`violation` に `diagnostic_id` を持たせるなら `plan_contract_violation` のような固定 ID を入れてもよい。ただし、`failure_kind: violation` だけで十分なら禁止してもよい。

## P1: plan-contract schema の `target` は pattern で締められる

contract file schema の `target` は現在 `minLength: 1` のみのように見える。

v1alpha では target は Spanner query target だけで、canonical target ID は以下の形に限定されている。

```text
query/Foo
query/Foo#inner
```

また、public names は `[A-Za-z_][A-Za-z0-9_]*` に制限された。ならば schema 側でも次のように pattern 化できる。

```json
{
  "pattern": "^query/[A-Za-z_][A-Za-z0-9_]*(#inner)?$"
}
```

将来 outer SQL / writes / DML を target に入れるなら、その時点で union pattern を増やせばよい。今は permissive にしておくより、v1alpha の狭い surface を schema で表す方が良い。

## P1: main config schema の reference fields も identifier pattern に寄せる

public names を identifier に制限した判断は良い。ただし、schema を見る限り、名前を定義する field は `$defs.identifier` を使う一方、名前を参照する field は `minLength: 1` のままのものが残っている。

たとえば以下は `$defs.identifier` を使ってよいと思う。

```text
queries[].catalog
queries[].binding
writes[].catalog
catalogs[].bindings.external_query_connections[].spanner_catalog
catalogs[].bindings.spanner_external_datasets[].spanner_catalog
```

これらはすべて「ユーザー定義名への参照」なので、定義側と同じ lexical rule に揃えると editor integration の質が上がる。

もちろん「参照先が存在するか」は planner の責務でよい。ここで言っているのは lexical validation だけ。

## P1: optimizer pinning と statement hint の扱いを README に明記する

`optimizer.requested` / `optimizer.effective` の分離は良い。`effective` を v1alpha では `not_recorded` にするのも正直で良い。

ただし `--require-optimizer-pinning` は「CLI で requested pinning があるか」を見るものとして整理されている。Spanner では statement hint が optimizer option より高い優先度を持つので、SQL 側に `@{OPTIMIZER_VERSION=...}` や `@{OPTIMIZER_STATISTICS_PACKAGE=...}` があるケースをどう扱うかは利用者が気にする。

現時点では実装を増やさなくてよいが、README に次を明記するとよい。

```md
In v1alpha, `--require-optimizer-pinning` checks optimizer options requested by
`plan-report` flags. It does not infer pinning from SQL statement hints, and
`optimizer.effective` is reported as `not_recorded`. If you rely on statement
hints for pinning, review the rendered SQL and report environment together.
```

将来は、SQL analyzer が statement hint を読めるなら次のような report fields を検討してもよい。

```yaml
optimizer:
  requested:
    version: "7"
    statistics_package: not_pinned
  statement_hint:
    version: "6"
    statistics_package: auto_...
  effective:
    version: not_recorded
    statistics_package: not_recorded
  warnings:
  - optimizer_statement_hint_overrides_requested_version
```

ただし v1alpha では documentation で十分。

## P1: observed plan shape の golden fixture 化は早めにしたい

response では Push Broadcast / Distributed Cross Apply の normalization が observed structural PLAN shape に基づくと説明されている。これは妥当だが、ここは将来壊れやすい。

少なくとも以下の fixture を `tools/spanner-query-plan-shape` 由来で保存しておくと、operator family の意味が安定する。

- standalone `Hash Join`
- `Push Broadcast Hash Join` wrapper + internal implementation `Hash Join`
- Push Broadcast の broadcast-side subtree に通常の nested `Hash Join` が含まれるケース
- standalone `Apply Join`
- `Distributed Cross Apply` wrapper + internal implementation `Apply`
- `Sort` / `Sort Limit`
- `Hash Aggregate` / `Stream Aggregate` / aggregate classification unknown

実装コードが未共有なので検証はできないが、この分類は plan contract の中心なので、golden fixture は regression test としてかなり重要。

## P2: README の plan contract section はそろそろ別 docs に分ける

README の first example と config surface はかなり simple になった。一方、plan contract section はかなり advanced になっている。

今の README は以下を一気に説明している。

- Omni-backed `plan-report`
- plan contract file
- predefined contracts の一覧
- raw QueryPlan CEL
- operator family normalization
- Push Broadcast Hash Join の内部実装 node
- Distributed Cross Apply の内部実装 node
- target inclusion / exclusion
- digest inputs
- optimizer pinning

これは正しい内容だが、README の役割を超え始めている。

次のように分けるとよい。

```text
README.md
  - plan-report の短い紹介
  - no_explicit_sort の最小例
  - optional / experimental / not production guarantee の caveat
  - docs/plan-contracts.md への誘導

docs/plan-contracts.md or PLAN_CONTRACTS.md
  - target grammar
  - predefined contracts
  - operator family catalog
  - raw CEL stability tier
  - Push Broadcast / Distributed Cross Apply normalization
  - schema/report fields
  - optimizer pinning

PLAN_CONTRACT_CANDIDATES.md
  - future candidates only
```

特に `PLAN_CONTRACT_CANDIDATES.md` があるので、current contract docs と candidate docs を分ける価値が出ている。

## P2: `no_hash_join` の現在の意味は良い。候補側では `no_distributed_join` を別 preset にするのも良い

`no_hash_join` が `hash_join` と `push_broadcast_hash_join` wrapper を禁止し、internal implementation node は数えない、という現在の意味は直感的で良い。

また、`PLAN_CONTRACT_CANDIDATES.md` の `no_distributed_join` を `no_hash_join` の implicit expansion にせず、別 preset candidate にする判断も良い。`no_distributed_join` は `hash_join` 以外に `distributed_cross_apply` や場合によっては `merge_join` も関係するため、`no_hash_join` と混ぜない方が安全。

このまま維持でよい。

## P2: `unknown` operator family を direct forbid できるのは残してよい

`operator_family` enum に `unknown` が含まれている点は一見 odd だが、これは便利。

たとえば次のように、normalization の未知化を CI で検出する用途がある。

```yaml
contracts:
- name: NoUnknownOperators
  target: query/ScanSingerIDsFast
  forbid:
  - operator_family: unknown
```

これは plan contract の安定性を上げるので、`unknown` を enum から外さない方がよい。

## P2: `skipped` query の field 名は `error` より `reason` が自然かもしれない

plan-report schema の per-query status は `ok | skipped | error` で、`skipped` と `error` の両方で `error` field が required になっているように見える。

実装上問題はないが、semantic には以下の方が読みやすい。

```yaml
status: skipped
reason: non_spanner_target
message: BigQuery outer SQL is not a plan-report target
```

```yaml
status: error
error: failed to analyze SQL: ...
```

ただし、これは P2。今のままでも致命的ではない。

## P2: `backend` flag は将来候補を増やすまで固定でもよい

`plan-report --backend omni` は、現時点で backend が一つしかないならやや冗長。ただし、すでに report の backend identity と合わせて workflow の境界を明確にしているので、`--render-mode` ほど削る必要はない。

将来 Cloud Spanner live / emulator / offline fixture backend などが出るなら意味が出る。今はこのままでもよい。

## まとめ

今回の response はかなり良い。前回の主要な指摘、つまり canonical target、exactly-one mode、operator family enum、not_evaluated、optimizer requested/effective、`no_hash_join` の直感的意味はだいたい閉じている。

次にやるべきことは大きな新機能ではなく、以下の仕上げ。

1. README に Spanner Omni execution-plan support の Preview / Pre-GA caveat を入れる。
2. `PLAN_CONTRACT_CANDIDATES.md` の旧 `query:` examples を `target:` に直す。
3. plan-report schema の `contract_evaluation` と `contract_rule_result` を status-specific に締める。
4. plan-contract schema の `target` を pattern 化する。
5. main config schema の reference fields も identifier pattern に寄せる。
6. `--require-optimizer-pinning` が SQL statement hints を見ないことを README に明記する。
7. Push Broadcast / Distributed Cross Apply / aggregate classification の observed plan shape を golden fixture 化する。
8. README の plan contract 詳細は、次のタイミングで `docs/plan-contracts.md` に分離する。

この段階では、`no_full_scan` や `require_scan_target` などの candidate を v1alpha predefined contract に急いで足さない方がよい。まずは今ある `plan-report` artifact と schema を CI で信用できる形にするのが優先だと思う。
