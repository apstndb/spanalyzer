# `spanner-query-gen` 最新状態への再レビュー

作成日: 2026-05-06

## 前提

今回も、実装コード・実行ログ・テスト出力は共有されていない前提でレビューします。したがって、ここで評価しているのは **README / DESIGN / JSON Schema / レビュー返答に現れている public contract と仕様の一貫性** です。

確認対象は主に次です。

- `spanner-query-gen-plan-contract-round4-review-response-ja.md`
- `README(23).md`
- `DESIGN(23).md`
- `spanner-query-gen.v1alpha.schema(11).json`
- `spanner-query-gen.plan-report.v1alpha.schema(2).json`
- `spanner-query-gen.plan-contracts.v1alpha.schema.json`

## 総評

今回の更新はかなり良いです。前回の懸念だった `plan-report` artifact の監査性は大きく改善しています。

特に良い点は次です。

- `--render-mode=auto` を削除し、v1alpha の `plan-report` を structural PLAN 専用にした。
- `PROFILE` / `execution_stats` を v1alpha の contract 入力から明確に外した。
- `input.config_sha256` と `input.contract_file_sha256` を report に入れた。
- `contract_evaluation_mode: none | report_only | check` を導入した。
- contract evaluation あり/なしで report schema を条件付きにした。
- `queries[].status` ごとに required fields を分け、error/skipped に fake digest を要求しない形にした。
- raw QueryPlan CEL を `raw_query_plan` tier として明示し、`--check` 使用時に warning を出すようにした。
- aggregate classification unknown を単なる fail ではなく `failure_kind: classification_unknown` として説明できるようにした。
- `remediation.auto_fix: false` を schema 上も固定した。
- `plan-contract-schema` を追加し、contract file 自体も reviewable artifact に寄せた。

方向性としては、**core config は simple に保ちつつ、plan contract は外部ファイル + optional `plan-report` workflow に隔離する** という判断が維持されており、これは今の設計方針に合っています。

## 追加フィードバック

### P0. `--render-mode` は v1alpha から完全に外してもよい

今回 `--render-mode=auto` と `--render-mode=profile` を拒否し、`--render-mode=plan` のみ受け付ける形になっています。これは安全ですが、さらに simple にするなら **`--render-mode` flag 自体を v1alpha から消す** のがよいと思います。

現在の README help は次のような形です。

```text
-render-mode string
      spannerplan render mode: plan (default "plan")
```

唯一の値しか受け付けない public flag は、ユーザーに「将来 profile/auto が使えるのでは」と期待させます。v1alpha では structural PLAN 専用という判断が固まったので、CLI surface からも消してよいです。

おすすめは次です。

```text
v1alpha:
  no --render-mode flag
  report.render_mode: PLAN

future:
  if PROFILE support is added, introduce a new report schema version
```

`render_mode: PLAN` を report に出し続けるのは良いです。これは artifact の意味を示す semantic field だからです。一方、CLI input としての `--render-mode` は今の v1alpha では冗長です。

### P0. plan contract file は `target` のみに寄せるとさらに単純になる

現在の plan contract schema は、contract target として `target` か `query` のどちらかが必要です。ただし schema 上は `target` と `query` の両方を書けるように見えます。また `query` + `scope` は短い alias とされています。

これは便利ですが、v1alpha ではまだ正式互換性を約束していないので、**canonical target ID のみに寄せる** 方がきれいです。

```yaml
version: v1alpha-plan-contracts

contracts:
- name: SingerIndexLookupPlan
  target: query/ScanSingerIDsFast
  use:
  - no_explicit_sort
```

`query` / `scope` alias を残すと、次のような論点が増えます。

- `target` と `query` が両方ある場合にどちらを優先するのか。
- `scope` に許される値は何か。
- `external_query` inner SQL を `query + scope` でどう表すのか。
- 将来 writes / DML / outer SQL target が増えたときに alias をどう拡張するのか。

`target: query/ExternalQuerySingerIDs#inner` は少し長いですが、review artifact としては一番明確です。v1alpha の単純化方針なら、短縮 alias は後回しでよいです。

どうしても `query` alias を残すなら、schema は `anyOf` ではなく `oneOf` にして、`target` と `query` の同時指定を拒否してください。

### P0. `use` / `forbid` / `cel` の同時指定 semantics を決める

現在の plan contract schema は、`use`、`forbid`、`cel` の少なくとも一つを要求しています。しかし schema 上は複数を同時に指定できるように見えます。

これはどちらかに寄せた方がよいです。

単純化優先なら、**1 contract entry は exactly one mode** にします。

```yaml
# predefined contract
- name: NoSort
  target: query/ScanSingerIDsFast
  use:
  - no_explicit_sort

# direct operator-family contract
- name: NoSortDirect
  target: query/ScanSingerIDsFast
  forbid:
  - operator_family: explicit_sort
    max_count: 0

# advanced CEL contract
- name: CustomShape
  target: query/ScanSingerIDsFast
  cel: |
    operators.exists(o, o.family == "scan")
```

複数の条件を AND したい場合は、`use` 配列に複数 predefined を入れるか、contract entry を複数書けば十分です。

複数 mode の同時指定を許すなら、以下を README / DESIGN / report schema に明記する必要があります。

```text
A contract entry may contain use, forbid, and cel together.
The contract passes only when all contained rules pass.
Each contained rule appears as a separate contract_rule_result.
```

ただし、今の simple 方針からは exactly-one の方が読みやすいと思います。

### P1. `backend: omni` は contract entry から外してもよい

README の plan contract example には `backend: omni` が入っています。一方、`plan-report` の CLI は現時点で `--backend omni` のみで、report には backend identity も出ます。

v1alpha で backend が一つしかないなら、contract file 内の `backend` は冗長です。さらに、contract entry ごとに backend を持つと、将来 backend が増えたときに「CLI backend と contract backend が衝突した場合どうするか」という余計な仕様が生まれます。

おすすめは次です。

```yaml
version: v1alpha-plan-contracts

contracts:
- name: SingerIndexLookupPlan
  target: query/ScanSingerIDsFast
  use:
  - no_explicit_sort
```

backend は report 側で十分です。

```yaml
backend: omni
backend_identity:
  kind: omni
  version: not_recorded
  image_digest: not_recorded
```

将来 Cloud Spanner / emulator / Omni など複数 backend を比較するなら、その時点で contract file version を上げて `backend` policy を入れる方が安全です。

### P1. `no_hash_join` と `no_push_broadcast_hash_join` の階層を定義する

predefined contract に `no_hash_join` と `no_push_broadcast_hash_join` が両方あります。これは良いのですが、名前だけ見ると `Push Broadcast Hash Join` も hash join の一種に見えます。

そのため、次のどちらかを明文化した方がよいです。

#### 案 A: `no_hash_join` を広義の hash join 禁止にする

```text
no_hash_join forbids all hash-based join families:
  - hash_join
  - push_broadcast_hash_join
```

この場合、`no_push_broadcast_hash_join` は「push broadcast だけを特に禁止したい」場合の narrower contract です。

#### 案 B: exact family 名に寄せる

```text
no_plain_hash_join
no_push_broadcast_hash_join
```

ただし、ユーザーの直感に近いのは案 A だと思います。`no_hash_join` という名前を使うなら、hash-based join 全体を止める contract として扱う方が自然です。

### P1. direct `forbid.operator_family` は typo を弾きたい

plan contract schema では、`forbid[].operator_family` が任意の non-empty string になっています。contract file を CI/review 用 artifact にするなら、ここは typo を早めに弾いた方がよいです。

たとえば次のような typo は schema か contract-file validation で落ちてほしいです。

```yaml
forbid:
- operator_family: explicit_srot   # typo
```

選択肢は二つです。

1. `plan-contract-schema` の enum に既知の normalized operator families を入れる。
2. schema は緩く保ち、evaluator が unknown operator family を contract-file error にする。

私は v1alpha では 1 を推します。`plan-contract-schema` を出しているので、editor integration で typo を見つけられる方が価値があります。将来 operator family を増やす場合は schema version または schema content を更新すればよいです。

### P1. `contract_rule_result` schema はもう少し締められる

`plan-report` output schema の `contract_rule_result` は、現状では `status` だけが required です。artifact としては少し緩いです。

最低限、以下は required にした方が review しやすいです。

```text
contract_rule_result:
  required:
    - status
    - rule
```

さらに、rule type ごとに required fields を分けられると理想です。

```text
operator-family rule:
  required: rule, status, operator_family, observed_count, max_count

cel rule:
  required: rule, status, expression

fail:
  required: failure_kind

classification_unknown fail:
  required: failure_kind, diagnostic_id
```

ここは実装都合で段階的でもよいですが、`plan-report` を CI artifact にするなら、rule result は「なぜ pass/fail したか」を schema 上も保証できる方が強いです。

### P1. report schema の `format` は enum にした方がよい

README の help では `-format` は `current`, `traditional`, `compact` の三値です。一方、plan-report schema の `format` は任意の non-empty string に見えます。

public output schema と CLI help の整合性を取るなら、次の enum でよいと思います。

```json
"format": {
  "enum": ["current", "traditional", "compact"]
}
```

同様に、report `queries[].kind` や `queries[].scope` も、現時点の値が決まっているなら enum 化してよいです。

### P1. optimizer pinning は requested / effective / source を分けたい

今回 `--optimizer-version` と `--optimizer-statistics-package`、さらに `--require-optimizer-pinning` が入ったのは良いです。ただし、Spanner では statement hint が optimizer option より優先されます。したがって、report では可能なら次を分けた方が安全です。

```yaml
optimizer:
  requested:
    version: "4"
    statistics_package: "auto_20240501_..."
  effective:
    version: "4"
    statistics_package: "auto_20240501_..."
  source:
    version: cli_query_option | statement_hint | database_default | unknown
    statistics_package: cli_query_option | statement_hint | database_default | unknown
```

ここまで実装できない場合でも、少なくとも SQL に statement hint が含まれていて CLI option と衝突し得る場合は warning を出したいです。

```yaml
warnings:
- id: optimizer-statement-hint-may-override-cli-option
  message: SQL statement hints have higher precedence than plan-report optimizer flags.
```

`--require-optimizer-pinning` を CI gate として使う場合、「CLI flag を渡した」ことと「実際にその optimizer setting で plan が取られた」ことを混同しない方がよいです。

### P1. aggregate classification unknown の扱いは direct `forbid` にも適用する

predefined `no_hash_aggregate` / `no_stream_aggregate` では、classification unknown を fail に寄せる方針が入りました。これは正しいです。

同じ semantics は direct `forbid` にも適用した方がよいです。

```yaml
forbid:
- operator_family: hash_aggregate
  max_count: 0
```

この contract も、`Aggregate` node が存在するが hash/stream を分類できない場合は、単に `observed_count: 0` として pass してはいけません。次のように fail するのが安全です。

```yaml
status: fail
failure_kind: classification_unknown
diagnostic_id: aggregate_iterator_type_unknown
```

つまり、classification unknown の扱いは predefined contract 固有ではなく、operator-family evaluation 全体の invariant にするとよいです。

### P2. `operator_family_counts` は便利だが consistency は plan 側で検証する

`operator_family_counts` の追加は良いです。review artifact としてかなり読みやすくなります。

ただし、schema だけでは次の整合性は保証できません。

```text
operator_family_counts == count(normalized_operators[].family)
operator_families == keys(operator_family_counts with count > 0)
```

これは schema ではなく report generation / test で固定するのがよいです。README か DESIGN に invariant として一行あると、reviewer が安心できます。

### P2. `plan-contract-schema` を追加したのは良いが、README の初期情報量は少し増えている

`plan-contract-schema` を追加した判断は良いです。外部 contract file を unknown-field rejection するなら、その schema も出せるべきです。

ただ、README の usage / help section はかなり長くなってきました。README の first-readability をさらに守るなら、root help は残しつつ、`plan-report-schema` / `plan-contract-schema` の詳細 help は plan-report section または別 docs に逃がしてもよいです。

これは P0 ではありません。今のままでも許容できます。

## 今回閉じたと見てよい論点

以下は、今回の更新で十分に整理されたと思います。

- `PROFILE` / `execution_stats` を v1alpha contract から外すこと。
- `--render-mode=auto` を拒否すること。
- `contract_evaluation_mode` によって report schema を分岐すること。
- `config_sha256` / `contract_file_sha256` を report に出すこと。
- `contract_file_path` を stable output から落とすこと。
- raw QueryPlan CEL を `raw_query_plan` tier として明示すること。
- raw QueryPlan contract を `--check` に使った場合に warning を出すこと。
- aggregate unknown を `failure_kind: classification_unknown` として説明すること。
- `remediation.auto_fix: false` を schema 上も固定すること。
- plan contract を main `spanner-query-gen.yaml` ではなく外部 file に隔離すること。

## まとめ

大きな方向転換は不要です。むしろ、設計はかなり収束しています。

次にやるなら、機能追加ではなく次の単純化・厳密化がよいと思います。

1. `--render-mode` flag を v1alpha から削除する。
2. plan contract target は `target` のみにし、`query` / `scope` alias を削る。
3. contract entry は exactly one of `use` / `forbid` / `cel` にする。
4. `backend: omni` を contract entry から削るか、少なくとも README example から外す。
5. `no_hash_join` の意味を broad hash-based join 禁止として明文化する。
6. `forbid.operator_family` の typo を schema または evaluator で弾く。
7. `contract_rule_result` の required fields を rule/status 別に締める。
8. optimizer settings は requested / effective / source を分ける。

このあたりを詰めると、`plan-report --contracts --check` は experimental でありながら、CI artifact としてかなり読みやすく、誤用しにくい形になると思います。
