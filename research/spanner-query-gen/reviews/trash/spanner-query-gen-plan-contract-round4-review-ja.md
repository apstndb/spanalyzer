# `spanner-query-gen` v1alpha plan contract round 4 review

作成日: 2026-05-05

## 前提

今回も、実装コード・実行ログ・テスト出力は共有されていないため、レビュー対象は添付された response / README / DESIGN / JSON Schema から読み取れる public contract に限定する。

今回の更新では、`plan-report` を main `spanner-query-gen.yaml` に混ぜず、external `plan-contracts.yaml` と optional Omni-backed workflow として扱う方針が維持されている。これは引き続き良い。さらに、version fields、canonical target ID、stability tier、PROFILE / `execution_stats` の除外、aggregate classification unknown handling、`plan-report-schema` が追加されており、CI artifact としての解釈可能性は前回よりかなり上がっている。

## 総評

大きな方向転換は不要。現状の線引きはかなり良い。

特に良いのは次の点。

- main v1alpha config は DDL / SQL / DTO / writes に集中し、plan contract は `plan-report` の optional extension に閉じ込めている。
- stable CI contract の推奨入力を normalized `operators` / `operator_families` に置き、raw `spannerpb.QueryPlan` / `PlanNode` を advanced tier として扱っている。
- `PROFILE` / runtime execution stats を v1alpha plan contract から外している。
- `Aggregate` の分類が不明なときに黙って pass させず、`classification_warnings` と fail 寄りにしている。
- `auto_fix: false` を remediation に明示し、SQL/config を勝手に書き換えない方針を維持している。
- `plan-report-schema` を追加し、report output も machine-readable contract として扱い始めている。

この方向で進めてよいと思う。次に詰めるべきなのは、新機能ではなく **report artifact の監査性・schema の条件付き厳密性・contract input の単純化** だと思う。

## P0: `--render-mode=auto` の意味を固定するか、v1alpha から外す

README の help では、`plan-report --render-mode` は `plan or auto; profile is not supported` となっている。

これは前回の `profile` 除外方針に沿っていて良いが、`auto` が何を選ぶのかが少し曖昧に見える。

v1alpha では次のどちらかに寄せたい。

### Option A: 一番 simple

```text
-render-mode string
      spannerplan render mode: plan (default "plan")
```

つまり `auto` も削る。

### Option B: `auto` を残す

`auto` は v1alpha では常に structural PLAN に解決され、PROFILE / execution stats には絶対に解決されない、と明記する。

```yaml
render_mode: auto
resolved_render_mode: plan
```

report 側にも `requested_render_mode` と `resolved_render_mode` を分けて出すと、後から読み返したときに安全。

`PROFILE` を明示的に拒否するなら、`auto` が profile 的なものを含むように読める余地は潰した方がよい。

## P0: report に input digest を追加する

現状の report は SQL digest、DDL digest、operator tree digest を持つ。これは良い。ただ、contract CI artifact としては、**どの config とどの contract file を評価したのか** も固定した方がよい。

提案:

```yaml
input:
  config_sha256: ...
  contract_file_sha256: ...        # --contracts がある場合
  contract_file_path: null          # --stable では path は落とす、または relative only
```

`contract_file_version` と `contract_evaluator_version` は grammar / evaluator の version を示すが、契約内容そのものの同一性は示さない。CI artifact としては `contract_file_sha256` があると、あとから「どの contract で pass/fail したか」を追跡できる。

`--stable` では host path は消してよいが、contract file digest は semantic input なので残すべき。

## P0: schema は contract 評価あり/なしを条件付きで表現する

`plan-report` は `--contracts` なしでも出力でき、`--contracts` ありなら `contract_summary` / `contract_evaluations` / `contract_file_version` / `contract_evaluator_version` が意味を持つ。

現在の schema はこれらを optional properties として持っているように見える。これでも動くが、public report schema としては少し緩い。

提案:

```yaml
contract_evaluation:
  mode: none | report_only | check
```

または、JSON Schema 上で次の invariant を入れる。

```text
if contract_evaluations is present:
  require contract_summary
  require contract_file_version
  require contract_evaluator_version
  require input.contract_file_sha256

if contract_summary is present:
  require contract_evaluations
```

さらに、`--contracts` なしの report では、`contract_summary` が absent なのか、`contracts: 0` として present なのかを固定した方がよい。どちらでもよいが、snapshot / CI の読みやすさを考えると以下が好み。

```yaml
contract_evaluation:
  mode: none
```

`contract_summary` は `mode != none` のときだけ出す。

## P0: query status と required fields の整合性を schema で分ける

`plan-report` schema の `query` は `status: pending | ok | skipped | error` を許しつつ、`sql` と `sql_sha256` を required にしているように見える。

これは `ok` なら自然だが、`error` や `skipped` で常に SQL digest を持てるとは限らない。たとえば DDL setup / SQL rendering / target selection のどこで失敗したかによって、SQL text がないケースがあり得る。

提案:

```text
query.status == ok:
  require sql, sql_sha256, ddl_sha256, plan, operator_tree_sha256, normalized_operators, operator_families

query.status == error:
  require error
  sql/sql_sha256 are present only if already resolved

query.status == skipped:
  require skip_reason
  sql/sql_sha256 are optional
```

あるいは、`queries[]` は analyzed targets only に限定し、`skipped` は `target_summary.excluded[]` にだけ出す。こちらの方が simple。

```text
queries[].status = ok | error
excluded targets are represented only under target_summary.excluded[]
```

この整理をすると、schema が fake digest や placeholder SQL を要求しなくなる。

## P1: raw QueryPlan contract を `--check` で使う場合はより明示的にする

現状、raw `plan` / `nodes` を使う CEL contract は `stability.tier: raw_query_plan` かつ `check_recommended: false` として report される。これは良い。

ただし、CI で `--check` を使う人は、report を読まずに exit code だけ見ることが多い。`check_recommended: false` だけでは、raw tier を誤って hard gate に使う可能性が残る。

選択肢は二つ。

### Option A: raw plan CEL は `--check` で明示 opt-in 必須

contract file 側に次を要求する。

```yaml
contracts:
- name: AdvancedRawPlanContract
  query: SomeQuery
  allow_raw_query_plan_in_check: true
  cel: |
    nodes.exists(...)
```

または CLI で:

```sh
spanner-query-gen plan-report --contracts plan-contracts.yaml --check --allow-raw-query-plan-contracts
```

### Option B: opt-in は不要だが warning を強く出す

```yaml
contract_summary:
  environment_warnings:
  - raw_query_plan_contract_used_in_check
```

`check_recommended: false` は report-level metadata として良いが、CI gate ではもう少し強く可視化した方が安全。

## P1: aggregate unknown は `fail` だけでなく failure reason を出す

`no_hash_aggregate` / `no_stream_aggregate` が unknown aggregate classification に依存する場合、`--check` で fail に寄せる判断は安全側で良い。

ただし、ユーザーには「本当に hash aggregate が見つかった」のか、「metadata が足りないため契約を評価不能として fail した」のかを区別して見せたい。

提案:

```yaml
results:
- rule: no_hash_aggregate
  status: fail
  failure_kind: classification_unknown   # violation | classification_unknown | environment
  operator_family: aggregate
  diagnostic_id: aggregate_iterator_type_unknown
```

`status: pass | fail` の二値は維持してよい。三値化しない方針も理解できる。ただ、`failure_kind` を足せば二値のまま explainability を保てる。

## P1: `remediation.auto_fix` は `const: false` にする

policy として「hint recommendation は remediation text のみで、SQL/config は自動変更しない」としているなら、report schema でも `auto_fix` は `const: false` にした方がよい。

現在の schema は boolean として見えるため、将来 true が出ても schema 上は通る。v1alpha で auto-fix を実装しないなら、public schema は次の方が正直。

```json
"auto_fix": {
  "const": false
}
```

将来 auto-fix を入れるときは、report schema version を上げればよい。

## P1: `backend_identity` と optimizer sentinel をもう少し機械可読にする

`backend_identity.version` / `image_digest` が `not_recorded` になる方針は現段階では妥当。ただし、string sentinel は後から処理しづらい。

最小変更なら、schema で sentinel と実値形式をある程度縛る。

```json
"image_digest": {
  "anyOf": [
    { "const": "not_recorded" },
    { "pattern": "^sha256:[a-f0-9]{64}$" }
  ]
}
```

より良い形は次。

```yaml
backend_identity:
  kind: omni
  version:
    status: not_recorded | recorded
    value: null
  image_digest:
    status: not_recorded | recorded
    value: null
```

ただし、これは report shape を少し重くする。v1alpha では sentinel + warning でもよいが、`not_recorded` を使うなら allowed sentinel と実値 pattern を schema に入れるとよい。

同様に `optimizer.version` / `optimizer.statistics_package` の `not_pinned` も、schema または docs で sentinel として明示した方がよい。

## P1: `operator_family_counts` を追加すると contract result が読みやすい

`normalized_operators[]` があれば count は再計算できる。ただ、contract artifact としては、operator family の presence と count がすぐ見えるとレビューしやすい。

提案:

```yaml
operator_families:
- explicit_sort
- table_scan

operator_family_counts:
  explicit_sort: 1
  table_scan: 2
```

`forbid` rule の `observed_count` と突き合わせやすくなる。

## P1: contract file の `backend: omni` は MVP では省略可能にしてもよい

contract example では各 contract に `backend: omni` がある。今は backend が `omni` しかないため、これは少し冗長。

さらに simple にするなら、contract file では省略可能にして、CLI / report の backend を使う。

```yaml
contracts:
- name: SingerIndexLookupPlan
  query: ScanSingerIDsFast
  use:
  - no_explicit_sort
```

ただし、将来 multiple backend を本当に入れるなら `backend` を残す意味はある。v1alpha の simple さを優先するなら省略可能、strictness を優先するなら今のままでもよい。

私の好みは「schema では optional、report では resolved backend を必ず出す」。

## P2: contract file input schema はまだなくてもよいが、将来は欲しい

`plan-report-schema` は report output schema であり、contract file input schema ではない。現時点では contract grammar が実験段階なので、input schema を急がない判断は理解できる。

ただし、`plan-report --contracts --check` を CI gate として使うなら、いずれは次のどちらかが欲しい。

```sh
spanner-query-gen plan-contract-schema --output json
```

または command を増やしたくないなら:

```sh
spanner-query-gen plan-report-schema --kind contracts --output json
spanner-query-gen plan-report-schema --kind report --output json
```

MVP では README/DESIGN に contract file grammar を明記し、unknown field rejection をテストで固定するだけでも十分。

## P2: README は現在の長さで許容。ただし advanced CEL は DESIGN に閉じ込める

README の plan contract section は前回よりかなり良くなっている。最初の例が `no_explicit_sort` だけで、CEL / raw QueryPlan / aggregate classification の詳細が DESIGN 寄りになっているのは良い。

この方針を維持したい。README に増やすなら次の 1 行程度で十分。

```text
For advanced CEL, raw QueryPlan, and operator classification details, see DESIGN.md.
```

README に raw CEL examples を増やし始めると、core config を simple にした努力が薄れる。

## Spanner Omni plan contract への評価

この機能はかなり差別化になる。特に、`table` / `index` shorthand が生成する SQL に対して、`Sort` が入らない、意図しない join method が出ない、といった contract をレビュー artifact として持てるのは `spanner-query-gen` らしい。

ただし、この機能は次のように表現すべき。

```text
A plan contract is a contract over a described plan environment,
not a production performance guarantee.
```

理由は、Spanner query optimizer は operator choice を optimizer version、statistics package、schema、SQL、hints、実行環境に応じて変え得るから。したがって、report は SQL / DDL / operator tree / optimizer / backend identity / contract file digest を残す必要がある。

今回の更新はその方向にかなり寄っている。あとは input digest、conditional schema、raw tier opt-in、failure reason を足せば、CI gate としてだいぶ安全になる。

## そのまま返せる短い返答案

> 今回の反映はかなり良いです。`plan-report` を main v1alpha config に入れず、optional Omni-backed workflow として維持し、normalized operator view を stable tier、raw QueryPlan を `raw_query_plan` tier、PROFILE / execution stats を対象外にした整理は妥当です。
>
> 追加で一番大事なのは、report artifact の入力同一性です。SQL / DDL / operator tree digest はありますが、CI artifact としては `config_sha256` と `contract_file_sha256` も欲しいです。`contract_file_version` は grammar の version であって、評価した contract 内容そのものの同一性を示さないためです。`--stable` でも host path は落としてよいですが、input digest は semantic field として残すべきです。
>
> 次に、`plan-report` schema は contract evaluation あり/なしを条件付きで表現した方がよいです。`contract_evaluations` が present なら `contract_summary` / `contract_file_version` / `contract_evaluator_version` / `contract_file_sha256` を required にする、または `contract_evaluation.mode: none | report_only | check` を出すと、後から読みやすくなります。
>
> また、`query.status` に `pending | skipped | error` を許すなら、`sql` / `sql_sha256` を常に required にするのは少し危険です。`ok` のときだけ plan / digests / operators を required にし、`error` は `error` required、`skipped` は `target_summary.excluded[]` に寄せる、という status-specific schema にした方がよいです。
>
> raw QueryPlan CEL は `stability.tier: raw_query_plan` と `check_recommended: false` だけでもかなり良いですが、`--check` で使う場合は明示 opt-in か、少なくとも `raw_query_plan_contract_used_in_check` warning を出すと誤用を減らせます。
>
> 最後に、`PROFILE` を拒否するなら `--render-mode=auto` の意味を固定してください。v1alpha では `auto` が常に structural PLAN に解決されると書くか、より simple に `auto` も外して `plan` のみにするのがよいです。

## 結論

最新の状態はかなり良い。前回の主要懸念だった raw QueryPlan の安定性、profile stats の扱い、aggregate unknown の扱い、README の過剰化はほぼ整理されている。

残るレビュー観点は以下。

1. `--render-mode=auto` の意味を固定するか削る。
2. `config_sha256` / `contract_file_sha256` を report に出す。
3. contract evaluation あり/なしの schema 条件を入れる。
4. `query.status` ごとの required fields を分ける。
5. raw QueryPlan contract を `--check` で使う場合の明示 opt-in / warning を追加する。
6. aggregate unknown failure を `failure_kind` で説明する。
7. `remediation.auto_fix` は v1alpha では `const: false` にする。
8. `backend_identity` / optimizer sentinel を schema 上でも機械可読にする。

このあたりを入れれば、`plan-report --contracts --check` は experimental ではありつつも、CI artifact としてかなりレビューしやすくなると思う。
