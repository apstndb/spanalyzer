# `spanner-query-gen` v1alpha / plan contract 最新再レビュー

作成日: 2026-05-06

## 前提

今回も、実装コード・実行ログ・テスト出力は共有されていない前提で見る。したがって、ここでの評価は README / DESIGN / JSON Schema / `IMPLEMENTATION_STATUS.md` / 返答文から読み取れる **public contract と設計文書の整合性レビュー** であり、実装済み機能の検証ではない。

## 総評

かなり良くなっている。特に次の判断は採用でよい。

- `plan-report` から `--render-mode` を削除し、v1alpha は structural `PLAN` 専用にした。
- plan contract file を main `spanner-query-gen.yaml` に入れず、外部 `v1alpha-plan-contracts` file として分離した。
- contract target を canonical `target` に寄せ、`query` / `scope` / `backend` alias を削った。
- contract entry を exactly one mode、つまり `use` / `forbid` / `cel` のどれか一つにした。
- `forbid.operator_family` を known normalized family enum に寄せた。
- `IMPLEMENTATION_STATUS.md` で「実装済み」「設計 drift 解消済み」「partial / design-only」を分けた。
- `tools/spanner-query-plan-shape` を public CLI ではなく developer probe として `tools/` に置いた。

この方向は、core config を simple に保ちながら、plan contract という opinionated な機能を optional review workflow として育てる方針に合っている。

以降のフィードバックは、大きな方向転換ではなく、**CI artifact としての曖昧さをさらに減らすための契約整理**に絞る。

---

## P0 / P1: まだ詰めたい点

### 1. `no_hash_join` の意味は今のままだと名前と直感がずれる

現在の整理では、`no_hash_join` は standalone `hash_join` だけを禁止し、`Push Broadcast Hash Join` wrapper は `push_broadcast_hash_join` として別 family にする。内部実装の `Hash Join` を `push_broadcast_hash_join_internal_hash_join` として分ける判断自体は良い。

ただし、ユーザー視点では **`no_hash_join` と書いたのに `Push Broadcast Hash Join` は許可される** という挙動はやや意外に見える。README で説明されているとはいえ、contract 名は CI policy として長く残るので、名前で意味が伝わる方がよい。

候補は二つある。

#### 案 A: `no_hash_join` を広義にする

`no_hash_join` は次を禁止する。

```text
hash_join
push_broadcast_hash_join
```

ただし、内部実装 node である `push_broadcast_hash_join_internal_hash_join` は数えない。

そのうえで、standalone だけを禁止したい rare case には、次を追加する。

```text
no_standalone_hash_join
```

この方が利用者の直感に近い。

#### 案 B: 現在の意味を維持し、名前を変える

現在の `no_hash_join` を次に改名する。

```text
no_standalone_hash_join
```

`Push Broadcast Hash Join` を禁止する場合は引き続き `no_push_broadcast_hash_join` を使う。

v1alpha で破壊的変更を許容できるなら、私は **案 A** を推す。`no_hash_join` は「hash-based join wrapper を含めて禁止」とした方が、contract として読みやすい。

同じ観点で、`no_apply_join` と `no_distributed_cross_apply` も整理した方がよい。`no_apply_join` が standalone apply だけを意味するなら、`no_standalone_apply_join` のような名前の方が誤解が少ない。

### 2. canonical `target` を使うなら、query name の grammar か escaping を固定する

contract target は `query/Foo` や `query/ExternalQuery#inner` のような canonical ID に寄せられている。これは良い。ただし、main config schema 上の `queries[].name` は現状 `minLength: 1` 程度に見えるため、名前に `/` や `#` が入ると target ID が曖昧になる。

どちらかを選ぶべき。

#### 案 A: query / write / catalog / contract name を identifier に制限する

たとえば、少なくとも次のようにする。

```regex
^[A-Za-z_][A-Za-z0-9_]*$
```

Go generated name としても自然で、target ID も単純になる。

#### 案 B: target ID の escaping を仕様化する

任意の query name を許すなら、`query/<escaped-name>#inner` の escaping を明文化する必要がある。

simple 方針なら、案 A がよい。v1alpha なら今のうちに name grammar を絞った方が後で楽になる。

### 3. plan report schema でも operator family enum を共有する

plan contract schema では `forbid.operator_family` が known normalized family enum になっている。一方、plan report schema では、少なくとも `operator.family` / `operator_families[]` / `operator_family_counts` の key が string 寄りに見える。

CI artifact としては、report 側も同じ enum に寄せたい。

推奨は、plan-report schema に `$defs.operator_family` を作り、次の全箇所で使うこと。

```text
operator.family
query.operator_families[]
query.operator_family_counts propertyNames
contract_rule_result.operator_family
```

特に `operator_family_counts` は `additionalProperties` だけだと typo key も schema 上は通りやすい。JSON Schema 2020-12 なら `propertyNames` で enum に寄せられる。

### 4. contract evaluation に `not_evaluated` / `target_error` を表現できるようにする

現在の report schema では query 自体に `status: ok | skipped | error` がある一方、`contract_evaluation.status` は `pass | fail` に見える。

しかし、contract target の query plan 取得が失敗した場合、それは contract violation ではなく **contract を評価できなかった状態** である。CI exit code は非ゼロでよいが、report artifact 上では `fail` と混ぜない方がよい。

推奨形は次のどちらか。

#### 案 A: contract evaluation status を増やす

```yaml
contract_evaluations:
- name: NoSort
  target_id: query/Foo
  status: not_evaluated
  reason: target_error
  diagnostic_id: plan_target_error
```

この場合、`results` は不要または空配列でよい。

#### 案 B: `fail` の `failure_kind` を広げる

```yaml
status: fail
results:
- rule: target_availability
  status: fail
  failure_kind: target_error
```

ただし、これはやや無理がある。simple さでは案 A がよい。

同様に、query `status: skipped` で `error` field を要求するより、`skip_reason` または `reason` を持たせた方が読みやすい。`error` は本当に失敗した場合だけに使う方が、CI artifact として自然。

### 5. optimizer field は requested / effective を混ぜない

`IMPLEMENTATION_STATUS.md` では、plan reports は requested optimizer options と pinning absence は記録するが、backend から見た effective optimizer version / source split はまだ未実装と整理されている。この整理は正しい。

ただし、report schema の root `optimizer.version` / `optimizer.statistics_package` は、読み手によっては effective 値に見えやすい。

v1alpha なら、次のように名前で明確化した方がよい。

```yaml
optimizer:
  requested:
    version: "8"
    statistics_package: latest
  effective:
    version: not_recorded
    statistics_package: not_recorded
```

あるいは、もっと simple にするなら、v1alpha report では次のようにする。

```yaml
optimizer_request:
  version: "8"
  statistics_package: latest
optimizer_effective: not_recorded
```

`--require-optimizer-pinning` は requested pinning の有無を判定している、と README / schema description にも明示するとよい。

Spanner の optimizer version / statistics package は plan stability に関係するが、query hint・statement hint・database option など複数の入力源があり得るため、将来 effective source を記録する余地を残す価値がある。

---

## P2: さらに単純化・監査性を上げる小さな提案

### 6. `stable` は report root で常に出す

README は `--stable` が snapshot-oriented output であり、semantic fields は残すと説明している。ならば machine-readable report では、`stable: true | false` を root required にした方がよい。

同様に、`warnings: []` や `contract_evaluations: []` のような空配列を常に出すかどうかも、snapshot / downstream tooling のために決めておくとよい。

### 7. plan contract schema の `oneOf` と `if/then` は少し冗長

contract entry の exactly-one mode は、次だけでかなり表現できる。

```json
"oneOf": [
  { "required": ["use"] },
  { "required": ["forbid"] },
  { "required": ["cel"] }
]
```

現在の `if/then not anyOf` は読み手にはやや重い。schema generation の都合で必要なら残してよいが、手書き schema として見せるなら one-of に寄せる方が simple。

### 8. `use` array は便利だが、contract result の粒度を固定する

`use` は複数 predefined rule を並べられる。これは実用的だが、review artifact では一つ一つの rule result が明確に出る必要がある。

今の `contract_rule_result` がそれを支えているなら問題ない。さらに単純化したいなら、v1alpha では `use` を string にして、複数 rule は複数 contract entry に分ける手もある。

```yaml
contracts:
- name: NoSort
  target: query/Foo
  use: no_explicit_sort
- name: NoHashJoin
  target: query/Foo
  use: no_hash_join
```

ただし、現状の array でも大きな問題はない。これは好みの問題に近い。

### 9. README の plan contract section はそろそろ分割してもよい

README はかなり読みやすくなったが、plan contract section は advanced 内容が増えている。

README には次だけ残すと、core config の simple さがさらに伝わる。

- `plan-report` は optional Omni-backed workflow。
- 最小 `no_explicit_sort` 例。
- `--contracts` と `--check` の exit code 差。
- production performance guarantee ではない。

次の内容は `docs/plan-contracts.md` や DESIGN に逃がしてよい。

- raw QueryPlan CEL
- Push Broadcast Hash Join internal normalization
- Distributed Cross Apply internal normalization
- aggregate unknown classification
- optimizer pinning details
- operator-family catalog

### 10. observed plan shape の根拠を fixture として残す

返答文では、Spanner Omni の actual plan shape を確認したうえで Push Broadcast Hash Join / Distributed Cross Apply の internal node 分類を設計している。これは良い。

ただし、外部レビューでは実行ログが共有されていないので、将来的には次のような最小 fixture があると設計の根拠を追いやすい。

```text
testdata/plan-shapes/push_broadcast_hash_join.plan.json
testdata/plan-shapes/hash_join.plan.json
testdata/plan-shapes/distributed_cross_apply.plan.json
```

実装依存が強ければ public fixture でなくてもよいが、normalization rule を変えるときの golden input として残す価値は高い。

### 11. Spanner Omni execution-plan support が Preview / Pre-GA であることを README に明記する

README はすでに「review / testing / prototyping 用」「production performance guarantee ではない」と書いている。これは十分良い。

ただ、Google Cloud docs では Spanner Omni execution plan support は Preview / Pre-GA offering とされており、development / testing / prototyping / demonstration 用と説明されている。README に短く次の文を足すと、利用者の期待値調整がさらに明確になる。

```text
Spanner Omni execution-plan support is a Preview / Pre-GA feature; plan-report is intended for development, testing, and review workflows only.
```

---

## そのまま返せる短いフィードバック案

> 最新版はかなり良いです。`--render-mode` 削除、PLAN-only 化、canonical `target`、exactly-one rule mode、`backend` / `query` / `scope` alias 削除、`IMPLEMENTATION_STATUS.md` による実装差分の分離は、この方向でよいと思います。大きな方向転換は不要です。
>
> 追加で見るべき一番大きい点は、predefined contract 名の直感性です。現在の `no_hash_join` が standalone `hash_join` だけを禁止し、`Push Broadcast Hash Join` wrapper は許可するなら、名前と直感が少しずれます。v1alpha なら、`no_hash_join` を hash-based join wrapper 全体の禁止にするか、現在の意味を `no_standalone_hash_join` に改名した方が安全です。同じく `no_apply_join` も standalone だけなら `no_standalone_apply_join` の方が明確です。
>
> また、canonical target ID を使うなら `queries[].name` の grammar を identifier に制限するか、`query/<escaped-name>#inner` の escaping を明文化してください。今のまま `minLength: 1` だけだと `/` や `#` を含む名前で target ID が曖昧になります。
>
> plan-report schema では、plan contract schema と同じ `operator_family` enum を `$defs` 化して、`operator.family`、`operator_families[]`、`operator_family_counts` の propertyNames、`contract_rule_result.operator_family` に共通適用するとよいです。CI artifact として typo や unknown family を早めに検出できます。
>
> 最後に、contract target の plan 取得が失敗した場合を `pass/fail` と別に表現できるようにしてください。query 自体に `status: error` があるのに contract evaluation が `pass | fail` だけだと、plan contract violation と evaluation failure が混ざります。`not_evaluated` または `failure_kind: target_error` を追加するのがよいです。
>
> それ以外は細部です。`optimizer` は requested / effective を混ぜない名前にする、`stable` を report root の required field にする、README の plan contract 詳細は別 docs に逃がす、observed plan shape は fixture として残す、という程度です。全体としてはかなり収束しています。

## 結論

現時点では、core v1alpha config の整理はかなり収束している。plan contract も、main config に混ぜず optional `plan-report` extension として扱う方針が良い。

次に詰めるべきなのは、新機能ではなく次の契約部分。

1. `no_hash_join` / `no_apply_join` の名前と意味の一致。
2. canonical target ID の grammar。
3. plan-report schema の operator family enum 共通化。
4. contract evaluation failure と violation の分離。
5. optimizer requested/effective の分離。
6. README から advanced plan-contract 詳細を分けること。

ここを締めれば、v1alpha の plan contract surface は CI review artifact としてかなり扱いやすくなる。
