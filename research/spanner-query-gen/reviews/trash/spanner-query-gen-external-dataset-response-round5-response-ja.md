# Round 5 review response

レビューありがとうございます。今回のレビュー対象に実装コードや test output が含まれていない前提はその通りなので、以後の回答では「内部作業ツリーでの実装主張」と「Markdown だけで外部レビュー可能な契約」を分けて書きます。test 名は内部追跡用の evidence として併記しますが、外部レビュー可能な evidence は expected diagnostic / expected plan shape / 最小 output snippet として提示します。

## 反映方針

大きな設計転換は不要と判断しました。以下は取り込みます。

- `access_verification` は observed fact ではなく verification evidence / attestation と呼ぶ。
- config 入力は `access.verification_hint` と `access.verification_evidence` に分け、plan 出力は canonical `access_verification` に寄せる。
- plan の `access_verification.source` は `user_hint | external_evidence | live_probe` とし、`independently_verified_by_generator` を明示する。
- external dataset の Cloud Resource connection は `cloud_resource_connection` と呼び、`EXTERNAL_QUERY` の `connection` mapping と区別する。
- `external_source` は認識済み形式だけを parse し、未認識形式は opaque metadata として保存し warning にする。
- `location` は `US` / `EU` / region の canonical form を plan に出す。
- `SELECT *` 以外でも lossy projection が見えるように relation provenance を plan に出す。
- `vet.disable` は diagnostic policy にだけ作用し、projection facts は絶対に消さない。
- `__external__` placeholder は table expression 位置にあることを generator 側で早めに検証する。
- external dataset projection matrix に source / docs date / matrix version を持たせる。

## 内部作業ツリーでの実装主張

実装コードはこの Markdown パッケージには含めていないため、以下は内部作業ツリー上の主張です。外部レビュー可能な契約は次節の expected output です。

| Behavior | Claimed implemented | Claimed tests |
| --- | --- | --- |
| `access.verification_evidence` を config 入力に追加し、plan `access_verification` に canonicalize | yes | `TestGenerateQueryCodeBigQuerySpannerExternalDataset`, `TestBuildQueryCodegenPlanRejectsInvalidSpannerExternalDatasetPolicies` |
| `cloud_resource_connection` を external dataset 用 canonical field として出力 | yes | `TestGenerateQueryCodeBigQuerySpannerExternalDataset`, `TestRunExplainPlanSummaryIncludesCatalogBindingWarnings` |
| BigQuery location canonicalization と connection location matching | yes | `TestGenerateQueryCodeBigQuerySpannerExternalDataset`, `TestBuildQueryCodegenPlanRejectsInvalidSpannerExternalDatasetPolicies` |
| 未認識 `external_source` を warning として扱う | yes | `TestBuildQueryCodegenPlanExternalDatasetExplicitColumnRelationProvenance` |
| explicit-column query の relation provenance | yes | `TestBuildQueryCodegenPlanExternalDatasetExplicitColumnRelationProvenance` |
| external dataset table の read-only target 検証 | yes | `TestBuildQueryCodegenPlanRejectsExternalDatasetDMLTarget`, `TestBigQueryAnalyzerSpannerExternalDatasetRejectsInformationSchemaAndDML` |
| PostgreSQL-dialect Spanner source の明示的 rejection | yes | `TestBuildQueryCodegenPlanRejectsPostgreSQLSpannerExternalDatasetSource` |
| `__external__` placeholder の table-expression position check | yes | `TestGenerateQueryCodeBigQueryFederatedQueryRequiresTableExpressionPlaceholder` |
| `explain-plan --stable` で volatile evidence を省略 | yes | `TestRunExplainPlanStableOmitsVolatileEvidence` |
| `vet --output json/yaml` の planning error contract | already implemented | `TestRunVetJSONReportsPlanErrors`, `TestRunVetYAMLReportsConfigParseErrors` |
| suppressed warning を machine-readable diagnostics に残す契約 | yes | `TestRunVetJSONKeepsSuppressedDiagnostics` |

## Externally Reviewable Expected Output

### Verification evidence

Config input:

```yaml
access:
  mode: cloud_resource
  cloud_resource_connection: example-project.us.example-connection
  database_role: reader
  verification_hint: not_checked
  verification_evidence:
    status: verified
    source: external_evidence
    verifier: terraform-plan
    checked_at: "2026-05-04T10:30:00Z"
```

Expected plan invariant:

```yaml
access: cloud_resource_connection
cloud_resource_connection: example-project.us.example-connection
access_verification:
  status: verified
  source: external_evidence
  verifier: terraform-plan
  checked_at: "2026-05-04T10:30:00Z"
  independently_verified_by_generator: false
  volatile: true
```

`source: live_probe` だけが generator による独立検証を意味します。config 由来の `verified` は external evidence として保持し、observed fact とは呼びません。

### Location and connection metadata

Expected plan invariant:

```yaml
location: us
location_metadata:
  configured: us
  canonical: US
cloud_resource_connection: example-project.us.example-connection
cloud_resource_connection_metadata:
  id: example-project.us.example-connection
  parsed_location: us
  parsed_location_canonical: US
  location_match: true
```

Expected planning error:

```yaml
diagnostics:
- id: planning-error
  stage: planning
  severity: error
  suppressible: false
  suppressed: false
  message: external dataset analytics_spanner: external dataset connection location "eu" conflicts with location "us"
```

### External source parsing

Recognized format:

```text
google-cloudspanner://[DATABASE_ROLE@]/projects/...
```

Expected behavior:

```yaml
database_role: reader
database_role_source: config_and_external_source
```

If the recognized URI role conflicts with `access.database_role`, planning fails:

```yaml
diagnostics:
- id: planning-error
  stage: planning
  severity: error
  suppressible: false
  suppressed: false
  message: external dataset analytics_spanner: external_source database role "reader" conflicts with access.database_role "writer"
```

Unrecognized format is preserved and warned, not parsed as a role:

```yaml
external_source: opaque-spanner-source
warnings:
- rule: external-dataset-external-source-unrecognized
  severity: warning
```

### Projection matrix and lossy relation provenance

Expected catalog binding snippet:

```yaml
projection_matrix:
  source: google_cloud_docs
  docs_last_checked: "2026-05-04"
  generator_matrix_version: 1
projected_tables:
- bigquery_table: example-project.analytics_spanner.Singers
  source_table: Singers
  columns:
  - name: SingerId
    visible: true
  - name: SearchName
    visible: false
    reason: hidden_spanner_column_not_visible
```

For explicit-column SQL:

```sql
SELECT SingerId, FirstName
FROM analytics_spanner.Singers
```

Expected query plan snippet:

```yaml
star_expansion: null
relations:
- sql_path: analytics_spanner.Singers
  source: spanner_external_dataset_projection
  role: source
  allowed: true
  projection_loss: true
  omitted_columns:
  - Singers.SearchName
```

For `SELECT *`, the same omitted column also appears under `star_expansion.projection_loss`.

### Vet suppression and projection facts

Expected invariant:

```yaml
diagnostics:
- id: external-dataset-access-unverified
  stage: external_dataset_projection
  suppressible: true
  suppressed: true
catalog_bindings:
- projected_tables:
  - source_table: Singers
    columns:
    - name: SearchName
      visible: false
      reason: hidden_spanner_column_not_visible
```

`vet.disable` affects warning presentation only. It must not remove `projected_tables`, omitted columns, or relation provenance from the plan.

### `__external__` placeholder

Expected generator error for invalid placement:

```text
query ExternalSingerIDs federated: federated.outer_sql placeholder must appear where a table expression is valid; use FROM __external__ or JOIN __external__
```

Valid placements include:

```sql
SELECT *
FROM __external__
```

```sql
SELECT q.SingerId
FROM __external__ AS q
JOIN NativeTable AS n USING (SingerId)
```

### External dataset table role validation

External dataset tables are read-only catalog bindings. They may appear as
query sources, but not as DML or metadata targets.

Expected relation shape for allowed reads:

```yaml
relations:
- sql_path: analytics_spanner.Singers
  source: spanner_external_dataset_projection
  role: source
  allowed: true
```

Expected generator/analyzer error for target usage:

```text
external dataset table analytics_spanner.Singers is read-only and cannot be used as dml_target (external_dataset_table_is_not_writable)
```

## 残す課題

- BigQuery DML の source / target role analysis は次の段階で入れる。external dataset table は source としては許容し得るが、DML target / metadata target としては拒否する。
- `rules.external_dataset_omitted_columns: error` のような global rule policy は、現行の binding-level `unsupported_columns: error` を残した上で後続の rule engine に寄せる。
- PostgreSQL-dialect Spanner external dataset は公式サービス上は存在するが、この generator はまだ Spanner GoogleSQL DDL の projection に限定し、PostgreSQL source は planning error とする。
- `--stable` は `access_verification.checked_at` のような volatile evidence
  field を省略する最小実装まで入れた。将来 live probe が増えたら対象 field
  を同じ policy に追加する。

## 結論

今回の指摘は基本的に取り込みます。特に「Markdown だけでレビューできる expected output を出す」点は、今後の回答様式として固定します。実装未共有のレビューでは、test 名だけでなく expected diagnostic ID / expected plan field / minimal output snippet を必ず併記します。
