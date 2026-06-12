# `spanner-query-gen` external dataset round 4 response への追加レビュー

対象: `spanner-query-gen-external-dataset-response-round4-response-ja.md`, 最新 `README(7).md`, 最新 `DESIGN(7).md`

前提: こちらには実装コードそのものは共有されていない。したがって、このレビューでは「実装済み」「テスト済み」という主張を検証済み事実としては扱わず、公開されている Markdown 上の仕様、設計意図、expected behavior、テスト evidence の提示方法をレビュー対象にする。

## 総評

今回の対応はかなり良い。特に、external dataset を `EXTERNAL_QUERY` の別記法ではなく BigQuery catalog binding として扱う方針、`Specified / Implemented / Tested` の表を出す方針、`access.verification_hint` と `access_verification` を分ける方針、`vet.disable` が warning 表示だけを抑制し projection metadata を消さない方針は妥当である。

大きな設計転換は不要だと思う。次に詰めるべきなのは、実装を見ていないレビューアにも判断できるように、**仕様・実装主張・テスト evidence・plan/golden output をどう対応付けるか**である。

## 追加でフィードバックしたい点

### 1. `Specified / Implemented / Tested` は良いが、外部レビュー用には `Expected plan / diagnostic` も欲しい

今回の回答では、代表 test 名と expected behavior を表に出している。これは前回よりかなり良い。ただし、実装コードが共有されていない前提では、test 名だけでは挙動の確認材料として弱い。

今後の response では、各 behavior に対して少なくとも次を併記するとよい。

```text
- Specified: README/DESIGN のどの仕様か
- Claimed implemented: yes/no
- Claimed tested: test name
- Expected diagnostic ID or plan field
- Minimal expected output snippet
```

例えば、次のような形が望ましい。

```yaml
behavior: explicit omitted column reference is rejected
claimed_tests:
- TestBigQueryAnalyzerSpannerExternalDatasetOmitPoliciesStillRejectExplicitReferences
expected_diagnostic:
  code: external_dataset_omitted_column_reference
  stage: analyze
  severity: error
  suppressible: false
expected_plan_invariant:
  catalog_bindings[].projected_tables[].omitted_columns still includes the column
  even though the query that references it fails
```

これにより、レビューアは実装コードを読まなくても「何が正しい出力なのか」をレビューできる。

### 2. 「実装済み」と「仕様として望ましい」を回答内でさらに分ける

今回の表では `Implemented: yes` / `Tested evidence` と書かれているが、こちらからは実装コードや test output を確認できない。そのため、次回以降の回答では次のような表現にするとより誠実でレビューしやすい。

```text
In our worktree, this is implemented and covered by the following tests.
Because the implementation is not included in this review package, the externally reviewable contract is the expected behavior and plan/diagnostic output below.
```

日本語なら次のように書くとよい。

```text
実装コードは今回のレビュー対象に含めていないため、外部レビュー可能な evidence は test 名ではなく expected behavior と expected plan/diagnostic shape です。test 名は内部の追跡用として併記します。
```

これは相手を疑うという意味ではなく、設計レビューとしての検証可能性を高めるための線引きである。

### 3. `access_verification` は「observed fact」ではなく「verification evidence / attestation」と呼んだ方が安全

回答では、config 側の `access.verification_hint` を user-declared metadata とし、observed fact を structured `access_verification` に分けるとしている。この分離は良い。

ただし、`access_verification` を config に書けるなら、それは generator が直接観測した事実ではなく、外部ツールや人間が渡した evidence / attestation である。`verifier: terraform-plan` や `checked_at` があるとしても、generator が live check していない限り「observed fact」と呼ぶのは少し強い。

おすすめは、plan 上で `source` を必須化し、意味を分けること。

```yaml
access_verification:
  status: verified
  source: external_evidence   # user_hint | external_evidence | live_probe
  verifier: terraform-plan
  checked_at: "2026-05-04T10:30:00Z"
  evidence_digest: sha256:...
```

そして用語を次のように整理する。

```text
verification_hint:
  User-declared hint. Never treated as proof.

access_verification.source = external_evidence:
  Evidence supplied by config or an external tool. The generator records it but did not independently verify it.

access_verification.source = live_probe:
  Evidence collected by a future live verification command.
```

これにより、`verified` という言葉が「実際に generator が GCP に問い合わせて検証した」という誤解を生みにくくなる。

### 4. `access_verification` の入力 shape と plan 出力 shape を分ける

README では、observed verification facts は `access_verification` に属し、`verified` / `mismatch` / `failed` には `verifier` や `checked_at` が必要、と説明されている。これは方向としてよいが、config 入力と plan 出力が同じ名前だと少し曖昧になる。

structured config では、次のように分ける案を検討してよい。

```yaml
access:
  mode: cloud_resource
  database_role: reader
  verification_hint: not_checked
  verification_evidence:
    status: verified
    source: terraform_plan
    checked_at: "2026-05-04T10:30:00Z"
    evidence_digest: sha256:...
```

plan では canonical にする。

```yaml
access_verification:
  status: verified
  source: external_evidence
  verifier: terraform_plan
  checked_at: "2026-05-04T10:30:00Z"
  evidence_digest: sha256:...
  independently_verified_by_generator: false
```

live verification が入ったら、同じ plan field に `source: live_probe` を入れられる。

### 5. `connection` という語が `EXTERNAL_QUERY` と external dataset で衝突しやすい

現在の config では、BigQuery schema に `external_query_connections` があり、external dataset binding にも `connection` がある。後者は external dataset を作成するときの Cloud resource connection metadata を表しており、前者は `EXTERNAL_QUERY` の connection mapping である。概念としては別物だが、利用者にとっては混同しやすい。

compact config ではこのままでもよいが、structured config では次のように名前を分けた方が安全である。

```yaml
spanner_external_datasets:
- dataset: analytics_spanner
  spanner_source: app_spanner
  external_source: google-cloudspanner://reader@/projects/example-project/instances/app/databases/app
  access:
    mode: cloud_resource_connection
    cloud_resource_connection: example-project.us.example-connection
    database_role: reader
```

または、plan だけでも次のように canonical field を分ける。

```yaml
external_dataset_binding:
  cloud_resource_connection: example-project.us.example-connection

external_query_connection_mapping:
  connection: example-project.us.example-connection
```

同じ文字列が使われる可能性はあるが、意味が違うことを plan で明示した方がよい。

### 6. `database_role` consistency は良いが、parse 対象を明示する

`external_source` に database role が含まれる場合に `access.database_role` と一致させる方針は良い。公式の Spanner external dataset 作成例でも、external source は `google-cloudspanner://[DATABASE_ROLE@]/projects/...` の形を取り得る。

ただし、parse 対象は外部サービスの URI 風表記であり、将来表記が増える可能性がある。仕様としては次を明記するとよい。

```text
- recognized format: google-cloudspanner://[DATABASE_ROLE@]/projects/...
- unrecognized external_source format: preserve as opaque string and emit warning, not parse-derived error
- recognized role mismatch: planning error
```

認識できない形式を即 error にすると、将来の Google Cloud 側の形式拡張や user-provided metadata に弱くなる。認識できる形式で矛盾したときだけ error にする方が実用的である。

### 7. `location` mismatch は planning error で良いが、`US` / `us` / region 表記の正規化を決める

回答では、`connection` が `project.location.connection` 形式で parse できる場合、configured `location` と照合し、mismatch を planning error にするとしている。これは良い。

ただし、BigQuery location は `US` / `EU` の multi-region と `us-central1` のような region があり、ユーザーは大小文字を混在させる可能性がある。plan では次を持つとよい。

```yaml
location:
  configured: us
  canonical: US
connection:
  parsed_location: us
  parsed_location_canonical: US
location_match: true
```

最低限、case-insensitive comparison なのか、BigQuery の canonical location string に正規化するのかを DESIGN に書くべきである。

### 8. `vet.disable` が projection metadata を消さない方針は強く維持する

今回の回答で、`vet.disable` は diagnostic policy にだけ作用し、projection metadata は消さないと明記された。これは非常に重要で、今後も変えない方がよい。

さらに、plan には以下を常に残すのがよい。

```yaml
warnings:
- code: external_dataset_omitted_columns
  suppressed: true
  suppressible: true
  reason: ...

catalog_bindings:
- projected_tables:
  - table: Singers
    visible_columns:
    - SingerId
    - FirstName
    omitted_columns:
    - SearchName
```

つまり、warning の表示抑制と、plan 上の事実の消去は絶対に分けるべきである。

### 9. `SELECT *` 以外でも lossy projection を見えるようにする

回答では、`SELECT *` が external dataset projection から展開された場合に `star_expansion.projection_loss` を残すとしている。これは良い。

追加で、明示列 query でも、その relation が lossy projection 由来であることを column provenance に残すとよい。例えば、次の query は `SELECT *` ではないため `star_expansion` は出ない。

```sql
SELECT SingerId, FirstName
FROM `example-project.analytics_spanner.Singers`
```

しかし、この query が参照している `Singers` は underlying Spanner table の完全な表ではなく、unsupported columns を落とした BigQuery-visible projection である。したがって result field provenance または relation provenance に次を残すと review しやすい。

```yaml
relations:
- sql_path: `example-project.analytics_spanner.Singers`
  source: spanner_external_dataset_projection
  projection_loss: true
  omitted_columns:
  - SearchName
```

`SELECT *` だけでなく「この relation 自体が lossy catalog binding である」ことを表現するのがポイントである。

### 10. External dataset table の read-only 性は source/target role で検証する

README / DESIGN では、external dataset table は read-only target であり、BigQuery DML into native BigQuery tables の source として使う余地は残す、と説明されている。この整理は正しい。

今後 BigQuery execution surface を生成するなら、analyzer では table reference を単に `read_only: true` とするだけでなく、query 内での role を分けるとよい。

```yaml
table_reference:
  path: `example-project.analytics_spanner.Singers`
  catalog_source: spanner_external_dataset
  role: source        # source | dml_target | metadata_target
  allowed: true
```

```yaml
table_reference:
  path: `example-project.analytics_spanner.Singers`
  catalog_source: spanner_external_dataset
  role: dml_target
  allowed: false
  diagnostic: external_dataset_table_is_not_writable
```

この形なら、次のような区別が自然にできる。

```sql
-- OK: external dataset table is a source
INSERT INTO native_dataset.SingerSnapshot
SELECT * FROM external_dataset.Singers;

-- NG: external dataset table is a target
INSERT INTO external_dataset.Singers
SELECT * FROM native_dataset.SingerSnapshot;
```

### 11. `__external__` placeholder は table-expression position validation を早めに入れてよい

回答では、`__external__` を lexical identifier token として exactly once 検出し、string/comment/quoted identifier 内は数えないとしている。これは良い。

一方で、FROM-position や alias shape の厳密検証は後続 work とされている。現時点でも置換後の BigQuery analyzer が invalid SQL を弾くなら大きな問題ではないが、ユーザー体験としては、placeholder-specific diagnostic が出た方がよい。

例えば、以下は BigQuery analyzer error ではなく generator の configuration error として説明した方が親切である。

```sql
SELECT __external__
```

```text
error: federated.outer_sql placeholder must appear where a table expression is valid; use FROM __external__ or JOIN __external__.
```

P0 ではなくてもよいが、`federated.outer_sql` を feature として安定化する前には入れた方がよい。

### 12. Machine-readable planning errors は早めに定義した方がよい

今回の回答では、JSON/YAML `vet` output で planning error も machine-readable diagnostics にする提案を保留している。これは P0 に含めない判断としては理解できる。

ただし、README では `vet --output json/yaml` の machine-readable report を説明しているため、plan failure 時の出力契約は早めに固定した方がよい。

最低限、次を明記してほしい。

```text
When planning fails:
- text output: diagnostics to stderr, stdout empty
- json/yaml output: either
  A. diagnostics report to stdout and stderr empty, or
  B. stderr text only and stdout empty
```

CI ツールにとっては A の方が使いやすい。もし B を暫定にするなら、README に「planning errors are not yet emitted as machine-readable reports」と明記した方が誤解が少ない。

### 13. Plan determinism と `checked_at` の扱い

`access_verification.checked_at` のような時刻 metadata は監査上は有用だが、plan golden test や `--check` と相性が悪い場合がある。毎回時刻が変わると plan output が unstable になるからである。

解決策として、plan output の用途を分けるとよい。

```text
- explain-plan --output yaml: full plan, includes checked_at
- explain-plan --output yaml --stable: omits or normalizes volatile evidence fields
```

または、plan field に volatility を明示する。

```yaml
access_verification:
  checked_at: "2026-05-04T10:30:00Z"
  volatile: true
```

生成コードの `--check` と plan review の golden output を分けるなら必須ではないが、CI で plan を snapshot するつもりなら早めに考えた方がよい。

### 14. External dataset type/projection matrix には source version を持たせる

DESIGN には external dataset projection matrix が入り、unsupported columns や named schema policy が整理されている。これは良い。

ただし、BigQuery external dataset の対応型や制限は Google Cloud 側で変わり得る。実装では matrix に source version / doc date / generator version を持たせるとよい。

```yaml
external_dataset_projection_matrix:
  source: google_cloud_docs
  docs_last_checked: "2026-05-04"
  generator_matrix_version: 1
```

`explain-plan` にも matrix version を入れると、将来 Google Cloud 側の対応型が増えたときに、どの時点の generator 判断なのかを追跡できる。

### 15. `unsupported_columns: omit` の default は妥当だが、strict CI では fail できるようにする

external dataset では unsupported column が BigQuery 側で accessible にならないため、default を `omit` にするのは実サービスに近い。一方で、schema drift を検出したい CI では、ある日 Spanner table に unsupported column が追加されて DTO から静かに消えることを嫌う可能性がある。

現在の `unsupported_columns: error` はその用途に合っている。追加で、global rule policy ができたら次のような設定も自然である。

```yaml
rules:
  external_dataset_omitted_columns: error
```

binding-level policy と global CI policy の両方を持てると、production hardening として使いやすい。

### 16. PostgreSQL-dialect Spanner external dataset は「公式には存在するが generator は未対応」と明記する

README では「current projection assumes Spanner GoogleSQL DDL」と明記されており、これは良い。Google Cloud の external dataset 自体は GoogleSQL または PostgreSQL database にリンクできるため、generator の非対応範囲として明確にしておく必要がある。

さらに、`spanner_source` が PostgreSQL dialect だった場合の behavior を決めておくとよい。

```text
- spanner_source dialect: spanner_google_sql -> supported
- spanner_source dialect: spanner_postgresql -> planning error for now
- external_source URI alone cannot override dialect inferred from schema config
```

これにより、official service support と generator support の差分が明確になる。

## 今回の回答に対する「そのまま返せる文案」

以下のように返すとよいと思う。

> 今回の対応はかなり良いです。external dataset を `EXTERNAL_QUERY` の別記法ではなく BigQuery catalog binding として扱う方針、`Specified / Implemented / Tested` の表を出す方針、`access.verification_hint` と `access_verification` を分ける方針、`vet.disable` が projection metadata を消さない方針はいずれも妥当です。
>
> ただし、こちらには実装コードそのものが共有されていないため、`Implemented: yes` や test 名は「検証済み事実」ではなく、内部作業ツリー上の主張として扱います。外部レビュー可能にするには、test 名だけでなく expected diagnostic ID、expected plan field、最小の golden output snippet を併記してください。例えば explicit omitted column reference なら、`stage: analyze`, `severity: error`, `suppressible: false`, `code: external_dataset_omitted_column_reference` のような診断 shape と、`projected_tables[].omitted_columns` が warning suppression 後も残ることを示す plan snippet があるとレビューしやすいです。
>
> `access_verification` については、config に書ける `verified` / `mismatch` / `failed` を observed fact と呼ぶのは少し強いです。generator が live probe していない限り、それは external evidence / attestation です。plan には `source: user_hint | external_evidence | live_probe` のような field を持たせ、`verified` であっても `independently_verified_by_generator: false` を表現できるようにしてください。
>
> external dataset binding の `connection` は、`EXTERNAL_QUERY` の `external_query_connections` と混同しやすいので、structured config では `cloud_resource_connection` のような名前に寄せるとよいです。compact config では残しても構いませんが、plan では external dataset の Cloud resource connection と `EXTERNAL_QUERY` connection mapping を別 field として出した方が安全です。
>
> `SELECT *` の `star_expansion.projection_loss` は良い追加です。さらに、明示列 query でも、その relation が lossy external dataset projection 由来であることを relation provenance に残すとよいです。`SELECT SingerId, FirstName FROM external_dataset.Singers` のような query でも、underlying Spanner table から unsupported columns が落ちている事実は plan review で見えるべきです。
>
> `vet --output json/yaml` の planning error 出力契約は早めに固定してください。P0 で完全対応しない判断は理解できますが、README では machine-readable report を説明しているため、planning error 時に stdout/stderr に何が出るのかを明文化しないと CI 利用者が迷います。
>
> 最後に、実装が共有されていない設計レビューでは、今後の回答で `specified`, `claimed implemented`, `claimed tested`, `externally reviewable expected output` を分けて書くと、こちらも過剰に実装を仮定せずレビューできます。

## 結論

今回の対応は、設計としてはかなり安定してきた。external dataset support の大枠に追加の反対はない。今後の主な課題は、機能追加ではなく以下のレビュー可能性である。

1. 実装未共有でも検証できる expected plan / diagnostic output を出す。
2. `access_verification` を live observation と external evidence に分ける。
3. external dataset の Cloud resource connection と `EXTERNAL_QUERY` connection mapping を名前・plan field で分離する。
4. warning suppression と projection facts の不変性を維持する。
5. planning error の machine-readable output contract を早めに固定する。
6. lossy projection を `SELECT *` だけでなく relation provenance として見える化する。

この段階では、大きな設計転換よりも、README / DESIGN / diagnostics / plan schema の細部を固定していくのが最も効果的だと思う。

## 参考

- Google Cloud Documentation, “Create Spanner external datasets” — external dataset は dataset-level federation であり、Spanner tables を BigQuery から直接 query できるが、external dataset table への modification はできない。Data Boost、default schema、unsupported columns、DML / metadata / INFORMATION_SCHEMA / Read API / Write API の制約もこのページにまとまっている。
  https://docs.cloud.google.com/bigquery/docs/spanner-external-datasets
