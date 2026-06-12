# `spanner-query-gen` v1alpha 最新版への再確認フィードバック

## 前提

今回も実装コード、実行ログ、テスト出力は共有されていません。したがって、このレビューは **README / DESIGN / 回答文から見える public spec と文書整合性のレビュー** です。

「実装済みかどうか」はここでは検証しません。外部レビュー可能な契約として、README、DESIGN、config schema、`explain-plan` / `vet` 出力がどれだけ一貫しているかを見ます。

## 総評

今回の更新はかなり良いです。特に、v1alpha の public YAML surface を次に絞った判断は強く支持します。

```text
version: v1alpha
go
emit
catalogs
queries
writes
rules.suppressions
```

`rules.severity`、write-level `emit`、raw SQL `EXTERNAL_QUERY(...)`、`connection_id` alias、external dataset `access.database_role` を v1alpha から外したのは、仕様をかなり読みやすくしています。

また、`upsert` の `insert_or_update` semantics を Spanner の実挙動に合わせ、

```text
insert.columns == key columns + update.columns
```

という invariant に寄せた点も正しいです。`INSERT OR UPDATE` は、既存行がある場合に statement column list に指定された非 key column を更新し、指定しない column は unchanged になるため、insert-only non-key column を許すと config の branch semantics が嘘になります。

README の最初の example も十分に小さくなりました。BigQuery、`EXTERNAL_QUERY`、external dataset、rules suppression を後段に分離したことで、以前より「小さい generator」に見えます。

## P0: cleanup plan がまだ最新方針に完全同期していない

一番大きい残り問題は、README / DESIGN / 回答文では最新方針に寄っているのに、cleanup plan に古い grammar が残っていることです。

特に、cleanup plan の canonical draft にまだ次が残っています。

```yaml
writes:
- name: UpdateSingerName
  ...
  emit:
  - mutation
  - dml

rules:
  severity:
    external-dataset-access-unverified: warning
    cross-dialect-timestamp-truncation: warning
    broad-update-mask: warning
```

これは、今回の回答で「write-level `emit` を削除」「`rules` は suppressions-only」とした方針と矛盾します。

さらに、cleanup plan の Open questions には、すでに決まったはずの問いが残っています。

```text
Should raw SQL EXTERNAL_QUERY(...) be entirely rejected in v1alpha?
Should projection_matrix be removed from default plan?
Should replace remain as operation: replace?
```

これらは README / DESIGN ではすでに答えが出ています。cleanup plan を残すなら、次のように更新した方がよいです。

```text
Resolved decisions:
- raw SQL EXTERNAL_QUERY(...) is rejected in v1alpha.
- projection matrix rows are audit-only by default.
- replace remains operation: replace and is mutation-only.
- v1alpha has no write-level emit.
- rules.severity is future work; v1alpha rules are suppressions-only.
```

または、cleanup plan を historical planning note として扱い、README / DESIGN / config-schema を唯一の current contract にする方がさらに simple です。

## P0: `emit` が何を gate するかを README で明確にしたい

README では `emit.spanner.mutations` と `emit.spanner.dml` が minimal config に出ています。一方で、同じ config に `queries` もありますが、`emit.spanner.query_methods` は future として rejected です。

そのため、利用者は次の点で迷う可能性があります。

```text
emit は DTO / SQL constants も gate するのか？
それとも runtime helper surface だけを gate するのか？
queries は emit がなくても DTO を生成するのか？
```

v1alpha を simple にするなら、README に次の一文を入れるのがよいです。

```text
Query DTOs and SQL constants are the baseline generated output for declared queries. `emit` controls additional runtime helper surfaces such as Spanner mutations and DML statements; query method surfaces are reserved and rejected in v1alpha.
```

これで `emit` の責務が明確になります。

## P0: `Current Documented Working-Tree Surface` の言い方を外部レビュー向けに弱める

DESIGN の `Current Documented Working-Tree Surface` は内容としては有用ですが、実装コードが外部共有されていない状況では、少し強く見えます。

特に、次のような表現は外部レビューでは検証できません。

```text
The current repository working tree is intended to support ...
```

おすすめは、次の三層に分けることです。

```text
## Documented v1alpha Contract
README and config-schema define the public YAML surface.

## Claimed Working-Tree Support
The maintainer claims the current working tree targets the following behavior.
Implementation and tests are not part of this document.

## Future Design
Longer-term directions that are intentionally not part of v1alpha.
```

さらに simple にするなら、DESIGN から `Claimed Working-Tree Support` を消し、実装主張は回答文や release note に寄せてもよいです。DESIGN は「設計」と「契約」に集中した方が読みやすいです。

## P1: `config-schema` を v1alpha contract の中心にするなら、cleanup plan にも反映する

今回 `config-schema` を追加する判断は良いです。外部レビュー可能性がかなり上がります。

ただし、cleanup plan の CLI 一覧にはまだ `config-schema` が入っていません。README では root help に出ているので、cleanup plan も同期してください。

また、simple にするなら、v1alpha では `config-schema --output json` だけで十分かもしれません。YAML schema 出力も残すなら、README には次のように書くと誤解が少ないです。

```text
`config-schema` emits the canonical JSON Schema. `--output yaml`, if supported, renders the same schema document as YAML for readability.
```

`config-schema` は future / rejected fields を含めないという回答方針は維持した方がよいです。これは「config は user intent」「plan は normalized semantics」という分離に合っています。

## P1: `queries[].params` は残してよいが、shape と scope を固定したい

`queries[].params` を残す判断には同意します。ARRAY / STRUCT parameter、推論不能 parameter、`external_query` の inner / outer scope 分離では、明示 parameter 型が必要になる可能性があります。

ただし、ここは将来肥大化しやすいので、v1alpha では目的を狭く固定した方がよいです。

```yaml
params:
- name: SingerIds
  type: ARRAY<INT64>
  scope: inner
```

提案するルールは次です。

```text
- params は analyzer type input / override であり、method signature customization ではない。
- normal sql/table/index query では scope を省略する。
- external_query では、必要な場合だけ scope: inner | outer を指定する。
- scope を省略して両方に同名 parameter が存在しうる場合は planning error にする。
```

これにより、`params` を残しても query method 設計や generated API naming に早く踏み込みすぎません。

## P1: `result.cardinality` は v1alpha での効き方を明記したい

README は `result.cardinality` の default を `many` とし、query method semantics が必要な場面で指定すると説明しています。一方で、`emit.*.query_methods` は v1alpha で rejected とされています。

この組み合わせ自体は問題ありませんが、ユーザーには次が曖昧です。

```text
query methods が rejected なのに、one / maybe_one / exec は何に効くのか？
```

README では、次のように書くとよいです。

```text
In v1alpha, `result.cardinality` is recorded in the plan and used for future query method semantics and diagnostics. Until query methods are enabled, it does not change DTO shape except where explicitly documented.
```

より simple にするなら、v1alpha MVP では `result.cardinality` を `many` / omitted に限定し、`one` / `maybe_one` / `exec` は query methods と同時に解禁する案もあります。ただ、DESIGN に query method semantics を残しておくのは問題ありません。

## P1: `verification_evidence.source` は config ではなく plan で補完してもよい

README の external dataset example では、config に次のように `source: external_evidence` を書かせています。

```yaml
access:
  verification_evidence:
    status: verified
    source: external_evidence
    verifier: terraform-plan
    checked_at: "2026-05-04T10:30:00Z"
```

ただ、v1alpha の config をさらに simple にするなら、`source` は config から消して plan で補完する方がよいです。

```yaml
access:
  verification_evidence:
    status: verified
    verifier: terraform-plan
    checked_at: "2026-05-04T10:30:00Z"
```

plan 側では次のように正規化します。

```yaml
access_verification:
  status: verified
  source: external_evidence
  independently_verified_by_generator: false
```

理由は単純です。config に書かれた evidence は常に external evidence であり、`source: live_probe` は generator-owned verification だけが plan に出せばよいからです。ユーザーに `source` を書かせると、将来 `live_probe` を config で誤指定する余地ができます。

## P1: `key` の意味を primary-key set に固定する

README では `delete` は table primary key を使い、`key` があればそれを使うと説明されています。v1alpha では、`key` を自由な WHERE key として広げない方が安全です。

提案は次です。

```text
For v1alpha writes, `key` must equal the table primary key set unless an operation explicitly documents another constraint target. If omitted, the table primary key is inferred.
```

特に `operation: upsert` + `conflict.strategy: insert_or_update` は primary key conflict なので、`key` が partial primary key や unique index key に見える形は planning error にした方がよいです。unique index / ON CONFLICT support は将来 `conflict.target` で扱う方が自然です。

## P1: `insert.columns == key + update.columns` は set equality と plan normalization を明記する

README の「exactly key plus `update.columns`」は良いですが、実装上は list equality ではなく set equality のはずです。

Spanner DML の column list は順序を持ち、values と positional に対応します。一方で、config の `key` と `update.columns` は user intent です。したがって、plan では次を明記するとよいです。

```text
- Validate set(insert.columns) == set(key) ∪ set(update.columns).
- Reject duplicates.
- Render columns in deterministic plan order, preferably table DDL order or a clearly documented config order.
- Preserve original config order as provenance if useful for diagnostics.
```

これを入れると、`insert.columns` の順序差による不要な golden diff を避けられます。

## P2: DESIGN の小さな文書バグ

DESIGN の `Current Documented Working-Tree Surface` に、次が重複しています。

```text
- Central `rules.suppressions` with required reasons.
- Central `rules.suppressions` with required reasons.
```

また、同じ箇所の public surface bullet は `catalogs`, `queries`, `writes`, `emit`, `rules` を挙げていますが、v1alpha grammar には `go` も含まれるため、`go` も入れてください。

```text
- v1alpha YAML config with `go`, `emit`, `catalogs`, `queries`, `writes`, and `rules.suppressions`.
```

## P2: root help を README に固定するなら、実装未共有レビューでは “example help” と呼ぶ

README の root help は分かりやすいです。ただし、実装コードが共有されていない外部レビューでは、それが実際の CLI 出力かどうかは確認できません。

README では `Current root help:` より、次のような表現の方が安全です。

```text
Expected root help:
```

または、実際の `--help` output を release artifact / test snapshot として出すなら、`Current root help` でも問題ありません。

## P2: cleanup plan の “Open questions” は “Resolved / Deferred” に分ける

今の段階では、open question として残すより、次のように分類した方が review しやすいです。

```text
Resolved:
- `kind: spanner` stays; do not rename to `spanner_database`.
- `dialect: googlesql` defaults for Spanner and BigQuery.
- raw SQL `EXTERNAL_QUERY(...)` is rejected.
- projection matrix rows are audit-only.
- `replace` remains a separate mutation-only operation.
- compact config breaks; no migrate-config before v1alpha unless user demand appears.

Deferred:
- DTO cross-catalog sharing declaration.
- full rule severity policy / strict mode.
- query methods.
- BigQuery method generation.
```

これで cleanup plan が議論メモとしても読みやすくなります。

## そのまま返せる短いフィードバック文

以下を開発側に返すとよいと思います。

> 最新版はかなり良いです。v1alpha の grammar を `version/go/emit/catalogs/queries/writes/rules.suppressions` に絞り、write-level `emit`、`rules.severity`、raw SQL `EXTERNAL_QUERY(...)`、external dataset `access.database_role` を外したのは、仕様をかなり simple にしています。`upsert` の `insert_or_update` についても、`insert.columns == key + update.columns` に制約した判断は Spanner semantics と合っています。
>
> 追加で直すべき最大点は、cleanup plan がまだ最新 README / DESIGN / 回答文と完全同期していないことです。canonical draft に write-level `emit: [mutation, dml]` と `rules.severity` が残っており、Open questions にも raw SQL `EXTERNAL_QUERY` reject、projection matrix、replace の扱いなど、すでに決まった事項が残っています。cleanup plan を更新するか、historical note として README / DESIGN / config-schema を current contract にしてください。
>
> さらに simple にするなら、`emit` が何を gate するかを README に明記してください。v1alpha では query DTOs と SQL constants が baseline output で、`emit` は Spanner mutations / DML など追加 runtime helper surface を gate する、という説明があると迷いません。
>
> `queries[].params` を残す判断には同意します。ただし、これは method signature customization ではなく analyzer type input / override であることを固定し、`external_query` では必要時だけ `scope: inner | outer` を使う、同名 parameter が曖昧なら planning error、という形にすると安全です。
>
> external dataset の `verification_evidence.source` は config から消して plan が `external_evidence` として補完してもよいと思います。config に書かれた evidence は常に external evidence であり、`live_probe` は generator-owned verification が実装された時だけ plan に出せばよいです。
>
> 最後に、DESIGN の `Current Documented Working-Tree Surface` は実装未共有の外部レビューでは強く見えるため、`Documented v1alpha Contract` / `Claimed Working-Tree Support` / `Future Design` に分けるか、実装主張を response/release note 側に寄せてください。DESIGN 内の `rules.suppressions` 重複行と、v1alpha surface bullet に `go` が抜けている点も小さく直せます。

## 結論

大きな方向転換は不要です。v1alpha の public YAML はかなり良いところまで削れています。

次にやるべきことは、機能追加ではなく次の 5 点です。

1. cleanup plan を最新方針に同期する、または historical note にする。
2. `emit` の責務を README で明確にする。
3. `params` の scope / analyzer override としての役割を固定する。
4. `verification_evidence` を config からさらに薄くできるか検討する。
5. DESIGN の実装主張を、実装未共有の外部レビューでも誤解されない表現にする。

この段階では、仕様をさらに足すより、README / DESIGN / config-schema / cleanup plan の single source of truth を揃えることが一番重要です。
