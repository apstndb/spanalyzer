# `spanner-query-gen` external dataset round 4 への返答案

## 前提

今回のレビューは、共有された以下の文書だけを対象にしています。

- `spanner-query-gen-external-dataset-response-round3-response-ja.md`
- `README(6).md`
- `DESIGN(6).md`

実装コード、テストコード、golden output、実行ログは共有されていないため、回答中の「実装済み」「test で固定済み」という主張は確認していません。以下のコメントは、**実装の正否ではなく、仕様文書・設計方針・レビュー可能性・将来のテスト観点**に対するものです。

## 総評

今回の対応はかなり良いです。特に、external dataset を `EXTERNAL_QUERY` の派生表現にせず、BigQuery catalog binding として扱う判断は維持されており、これは正しいです。`Current Scope` と Phase 4.5 の関係も、「basic catalog binding は current、production hardening は Phase 4.5」という形に整理されていて、以前より誤読しにくくなっています。

また、以下の点はそのまま進めてよいと思います。

- `unsupported_columns: omit` を projection build では fatal にせず、`SELECT *` では BigQuery-visible columns のみを見せる。
- 明示的な omitted column / omitted table 参照は planning error にする。
- projected table だけでなく projected column の case-folding collision を error にする。
- BigQuery DDL table と external dataset projection の canonical path collision を silent override しない。
- `EXTERNAL_QUERY` output conversion matrix と external dataset projection matrix を分ける。
- external dataset table の read-only は「DML / metadata target にできない」という意味に限定し、source としての利用まで過剰に reject しない。
- `access_verified: false` のような曖昧な boolean ではなく、`not_checked|verified|mismatch|failed` の verification status に寄せる。
- Spanner PostgreSQL-dialect external dataset は、公式サービスとしては存在しても、この generator の GoogleSQL DDL/analyzer 前提では unsupported と明記する。

この round では大きな設計転換は不要です。追加フィードバックは、主に **「ユーザーが config に書いた metadata」と「generator が検証した事実」を混同しないこと**、および **実装コードなしでもレビューできる test evidence を残すこと** に絞ります。

## 追加で伝えたい P0

### 1. 「実装済み」と「仕様化済み」を文書上で分ける

回答では「今回の P0 は実装・test・docs まで進めます」とありますが、実装そのものは共有されていません。これは問題というより、レビューの粒度を合わせる必要があります。

今後の response / DESIGN / README では、各項目を次の 3 段階で表現するとよいです。

```text
specified: DESIGN/README に仕様として書いた
implemented: code path が存在する
tested: regression test または golden plan で固定した
```

例えば、external dataset については次のような表を response に入れると、実装を見ていない reviewer でも確認しやすくなります。

| Behavior | Specified | Implemented | Tested evidence |
| --- | --- | --- | --- |
| `unsupported_columns: omit` excludes unsupported columns from `SELECT *` | yes | claimed | `TestExternalDatasetOmitUnsupportedColumnStar` |
| explicit omitted column reference fails | yes | claimed | `TestExternalDatasetExplicitOmittedColumnReferenceError` |
| projected column case-folding collision fails | yes | claimed | `TestExternalDatasetProjectedColumnCaseCollision` |
| BigQuery DDL vs external projection collision fails | yes | claimed | `TestExternalDatasetNativeTableCollision` |
| `database_role` + `not_checked` emits warning | yes | claimed | `TestExternalDatasetRoleWithoutVerificationWarning` |

実装コードを共有しない場合でも、test names、代表的な input config、expected diagnostic、expected plan snippet があれば、設計レビューと実装レビューの境界がかなり明確になります。

### 2. `access.verification` は user assertion と verified fact を分ける

README の compact example では `access.verification: not_checked` が config に入っています。この形自体は分かりやすいですが、将来 `verified` / `mismatch` / `failed` をユーザーが手で書けるように見えると危険です。

`verified` は、本来 generator または別の verification step が観測した結果であり、ユーザーの希望や想定ではありません。したがって、config と plan で名前を分けた方が安全です。

提案:

```yaml
# config: ユーザーが宣言する前提・期待
access:
  mode: cloud_resource
  database_role: reader
  verification_hint: not_checked
```

```yaml
# plan: generator / verifier が出す観測結果
access_verification:
  status: not_checked
  source: static_config
  checked_at: null
  verifier: null
```

もし config 側で `verified` を許すなら、少なくとも以下を必須にした方がよいです。

```yaml
access:
  verification:
    status: verified
    checked_by: terraform-plan-or-ci-job-name
    checked_at: "2026-05-04T10:30:00Z"
    evidence_digest: sha256:...
```

単なる `verification: verified` は、plan review 上は事実確認ではなく自己申告にしか見えません。

### 3. `external_source` 内の database role と `access.database_role` の整合性を検証する

README の例では、`external_source` に `google-cloudspanner://reader@/...` のような role 付き URI があり、同時に `access.database_role: reader` があります。この二重表現は auditability の面では便利ですが、ずれると危険です。

次の rule を入れるべきです。

```text
- external_source に database role が含まれ、access.database_role も指定されている場合、両者が一致しなければ planning error。
- external_source に database role が含まれ、access.database_role が省略されている場合、plan に inferred_database_role として記録する。
- access.database_role が指定され、external_source に role がない場合、plan に role_source: config として記録する。
```

これは小さい点ですが、external dataset の access assumption を reviewable にするという今回の設計目的にかなり合います。

### 4. `access.mode` と `connection` / `location` の整合性チェックを明文化する

external dataset では `CLOUD_RESOURCE` connection を使う場合、connection と dataset の location が重要です。README では `external_source`、`location`、`connection` を optional metadata としていますが、`access.mode: cloud_resource` のときに `connection` が無い、または `connection` の location と `location` が食い違うと、plan の前提がかなり弱くなります。

提案:

```text
- access.mode: cloud_resource かつ connection なし: warning または planning error。
- connection が project.location.connection_name 形式で parse でき、location も指定されている場合、location mismatch は planning error。
- access.mode: euc かつ connection が指定されている場合、意図が曖昧なので warning。
- access.mode: unknown の場合、database_role / connection / external_source は provenance として保持するが、visibility assumption は unknown にする。
```

ここも live verification なしで断定しない方針と相性が良いです。

## 追加で伝えたい P1

### 5. `unsupported_columns: omit` は warning 抑制後も plan から消さない

`unsupported_columns: omit` は現実の BigQuery external dataset behavior に寄せる default として妥当です。ただし、warning を `vet.disable` で抑制した場合でも、omitted column の一覧は plan から消してはいけません。

おすすめは、diagnostic と projection metadata を分けることです。

```yaml
catalog_bindings:
- kind: spanner_external_dataset
  dataset: example-project.analytics_spanner
  projected_tables:
  - table: Singers
    visible_columns:
    - SingerId
    - FirstName
    omitted_columns:
    - name: SingerProto
      reason: unsupported_type
      spanner_type: PROTO<Singer>
      suppressible_warning_rule: external-dataset-omitted-column
```

`vet.disable` は warning の表示や exit policy にだけ影響し、`omitted_columns` という事実は常に `explain-plan` に残すべきです。これは「warning を抑制したら projection loss が見えなくなる」という事故を防ぎます。

### 6. `SELECT *` の DTO には projection loss を記録する

`unsupported_columns: omit` で `SELECT *` が BigQuery-visible columns だけを返す仕様は妥当です。ただし、ユーザーの直感では「Spanner table の全列」と誤解されやすいです。

`SELECT *` を含む query result plan には、次のような field を入れるとよいです。

```yaml
queries:
- name: ExternalDatasetSingerNames
  source_kind: bigquery_spanner_external_dataset
  star_expansion:
    source: external_dataset_projection
    projection_loss: true
    omitted_columns:
    - SingerProto
    - SearchTokens
```

これにより、generated DTO が Spanner DDL の全列ではなく、BigQuery external dataset projection 後の列から作られていることが明確になります。

### 7. external dataset projection matrix に `TIMESTAMP` / `NUMERIC` caveat を残す

DESIGN の external dataset projection matrix では supported scalar column を `ok` としています。これは大枠では問題ありません。ただし、BigQuery external dataset の公式ドキュメントは limitations を列挙していますが、`EXTERNAL_QUERY` のような詳細な Spanner-to-BigQuery type mapping table を external dataset 用に明示しているわけではありません。

一方で、BigQuery 側の型として見える以上、`TIMESTAMP` や `NUMERIC` には precision / range / loader behavior の caveat が残ります。特に `EXTERNAL_QUERY` では Spanner `TIMESTAMP` の nanoseconds truncation が明記されています。

したがって、external dataset matrix では `supported scalar column: ok` のままでも、以下のような注記を足すと安全です。

```text
Supported scalar columns are BigQuery-visible, but runtime precision and loader behavior should be tracked separately where relevant. In particular, TIMESTAMP and NUMERIC should carry conversion caveats until external-dataset-specific behavior is verified by docs or live tests.
```

これは「external dataset でも必ず EXTERNAL_QUERY と同じ truncation が起きる」と断定する意図ではありません。むしろ、外部ドキュメントまたは live verification なしに安全側の caveat を plan に残すための提案です。

### 8. `__external__` placeholder は raw substring ではなく lexical placeholder として扱う

回答では、`__external__` は BigQuery identifier ではなく config-level placeholder であり、BigQuery SQL analysis の前に generated `EXTERNAL_QUERY(...)` table expression へ置換するとされています。この方針は良いです。

ただし、置換は raw string replacement にしない方がよいです。

危険な例:

```sql
SELECT '__external__' AS literal_value
```

```sql
SELECT * FROM SomeTable -- __external__
```

このような string literal / comment 内の `__external__` は placeholder として数えるべきではありません。最低限、BigQuery lexer 相当の処理で「token としての `__external__`」だけを検出し、exactly once を判定するべきです。

さらに、今は FROM-position / alias shape まで厳密検証しない判断でよいと思いますが、次のケースは BigQuery analyzer に渡す前に plan diagnostic を出せると親切です。

```sql
SELECT * FROM __external__ AS e
```

この場合、replacement 後の generated SQL と alias の結合がどうなるかを plan に出してください。

### 9. JSON/YAML `vet` output は planning error でも machine-readable にする

README では、JSON/YAML mode は machine-readable report を stdout に出し、plan 自体が fail した場合は例外のように読めます。CI UX を考えると、parse/config file read などの fatal CLI error は stderr でよいですが、**planning error は machine-readable diagnostics として stdout に出る方が扱いやすい**です。

提案:

```text
- config file not found / invalid CLI flags: stderr, non-zero, no JSON contract required
- config parse error after file read: JSON/YAML diagnostic if --output json|yaml
- planning error: JSON/YAML diagnostic on stdout, stderr empty, non-zero
- unsuppressed warning only: JSON/YAML diagnostic on stdout, exit 0 unless --strict / warnings-as-errors
```

`vet` が「plan validation」であるなら、planning error は reportable diagnostic として扱った方が自然です。

### 10. diagnostic schema に `suppressible` と `stage` を入れる

binding-level suppression が入ったので、diagnostic schema は早めに固定した方がよいです。

最小形:

```yaml
diagnostics:
- id: external-dataset-omitted-column
  severity: warning
  stage: external_dataset_projection
  scope:
    schema: analytics_bigquery
    binding: analytics_spanner
    table: Singers
    column: SingerProto
  suppressible: true
  suppressed: false
  reason: Spanner PROTO column is not visible in BigQuery external datasets.
  remediation: Project supported scalar fields or set unsupported_columns: error in CI.
```

重要なのは、planning error には `suppressible: false` を明示することです。特に以下は suppression 不能でよいと思います。

- explicit omitted-column reference
- explicit omitted named-schema table reference
- BigQuery DDL vs external dataset canonical path collision
- projected column case-folding collision
- malformed `__external__` placeholder usage

## 追加で伝えたい P2

### 11. external dataset と native BigQuery table の同名衝突は `source_kind` も plan に残す

衝突時に planning error にする方針は正しいです。その上で、plan diagnostic には、両者の origin を明確に出すと debug しやすいです。

```yaml
conflict:
  canonical_path: example-project.analytics_spanner.Singers
  candidates:
  - source_kind: bigquery_ddl
    schema: analytics_bigquery
    ddl_path: bigquery_schema.sql
  - source_kind: spanner_external_dataset_projection
    schema: analytics_bigquery
    binding: analytics_spanner
    spanner_source: app_spanner
    spanner_table: Singers
```

### 12. named schema omission は table-level omitted list に残す

Named schema table を projection から omit する方針は良いです。ただし、warning 抑制後も、omitted table の情報は plan に残すべきです。

```yaml
projected_tables:
- spanner_table: public.Singers
  projected: true
- spanner_table: archive.Singers
  projected: false
  reason: named_schema_not_supported
```

`projected_tables` という名前を使うなら、「projected false の omitted tables も含む」と DESIGN に明記しているのは良い判断です。ここはそのまま維持してください。

### 13. PostgreSQL-dialect external dataset unsupported は source dialect で error にする

公式 external dataset は GoogleSQL / PostgreSQL の Spanner database にリンクできますが、この generator は Spanner GoogleSQL DDL/analyzer 前提なので PostgreSQL-dialect external dataset を unsupported にする、という回答は妥当です。

実装上は、`spanner_source` の dialect が `spanner_postgresql` または今後追加される PostgreSQL dialect の場合、projection build warning ではなく planning error にするのがよいです。理由は、DDL parser / type system / identifier rules が違い、static projection の根拠が崩れるためです。

### 14. `access.mode` の値名は service/API 名と近づける

`unknown|euc|cloud_resource` は短くてよいですが、README では `CLOUD_RESOURCE` connection と書かれているので、plan output では official-ish な名前に寄せてもよいです。

候補:

```yaml
access:
  mode: end_user_credentials     # config alias: euc
```

```yaml
access:
  mode: cloud_resource_connection # config alias: cloud_resource
```

compact config では短い alias を受けても、plan では canonical enum に寄せると audit しやすいです。

## そのまま送れる返答案

> 今回の対応はかなり良いと思います。external dataset を `EXTERNAL_QUERY` の別記法にせず、BigQuery catalog binding として扱う方針、`Current Scope` と Phase 4.5 の整理、`unsupported_columns: omit` の query behavior、明示 omitted reference の planning error 化、case-folding collision / BigQuery catalog collision の error 化はいずれも妥当です。
>
> 追加で一番大きい指摘は、実装コードが共有されていない前提では、response の「実装済み」「test で固定済み」という主張をこちらでは検証できないことです。今後は各 behavior について `specified / implemented / tested` を分け、test name、representative input、expected diagnostic、expected plan snippet を response に添えると、設計レビューとしてもかなり確認しやすくなります。
>
> 次に、`access.verification` は user config と verified fact を分けた方が安全です。`verified` / `mismatch` / `failed` は本来 generator または別 verification step が観測した結果であり、ユーザーが手で書く自己申告に見えると危険です。config 側は `verification_hint` や `expected_access`、plan 側は `access_verification.status/source/checked_at/verifier` のように分けることを提案します。
>
> また、`external_source` に `reader@` のような database role が含まれ、同時に `access.database_role` が指定される場合、両者の不一致は planning error にしてください。`access.mode: cloud_resource` と `connection` / `location` の整合性も plan diagnostic にした方がよいです。たとえば connection の location と dataset location が食い違う場合は planning error、`cloud_resource` なのに connection が無い場合は warning または error にするのが安全です。
>
> `unsupported_columns: omit` は default として妥当ですが、warning を suppress しても omitted columns / omitted tables の一覧は `explain-plan` から消さないでください。suppression は diagnostic policy にだけ効かせ、projection metadata は常に残すべきです。特に `SELECT *` では、BigQuery-visible columns のみに展開されたこと、underlying Spanner columns の一部が消えたことを `star_expansion.projection_loss: true` のように plan に残すと誤読が減ります。
>
> `__external__` placeholder は raw string replacement にしないでください。string literal や comment 内の `__external__` を placeholder として数えると危険です。BigQuery lexer 相当で token としての `__external__` を exactly once 検出し、置換後 SQL と alias の形を `explain-plan` に出すのがよいです。
>
> 最後に、JSON/YAML `vet` output は planning error でも machine-readable diagnostics を stdout に出す契約に寄せると CI で扱いやすいです。fatal CLI error は stderr でよいですが、planning error は `vet` が扱うべき structured diagnostic として出せる方が自然です。

## 参考にした外部仕様

- BigQuery Spanner federation は external dataset と `EXTERNAL_QUERY` の 2 通りがある。external dataset は BigQuery から通常の table のように query できるが、`EXTERNAL_QUERY` は query-level federation として connection / inner SQL を使う。
- BigQuery Spanner external dataset は dataset-level の federation で、Spanner の GoogleSQL または PostgreSQL database にリンクできる。tables は BigQuery 側に自動的に見えるが、external dataset 側で data / metadata を変更することはできない。
- Spanner external dataset の limitations として、default Spanner schema の tables のみ accessible、Spanner primary / foreign key metadata は BigQuery 側に見えない、unsupported Spanner columns は BigQuery 側で accessible にならない、DML / metadata changes / `INFORMATION_SCHEMA` / Read API / Write API は unsupported、Data Boost は default で使われ変更できない、などがある。
- `EXTERNAL_QUERY` の Spanner-to-BigQuery type mapping では、Spanner `STRUCT` は unsupported、`TIMESTAMP` は nanoseconds truncated、unsupported type は query failure になる。
