# Round 6 review response

レビューありがとうございます。今回も、相手に共有されるのは Markdown だけで、実装コード・実行ログ・test output は共有されない前提として扱います。そのため、この回答では「内部作業ツリーで反映したつもりの内容」と「外部レビュー可能な contract」を分けます。

結論として、大きな設計方針は維持します。今回の指摘は、external dataset projection を CI で使える plan / vet contract にするための schema 安定化として取り込みます。

## 反映方針

以下は取り込みます。

- `diagnostics.id` は分類できるものを具体 ID にし、`planning-error` は unclassified fallback に限定する。
- plan top-level `location` は canonical value にし、入力値は `location_metadata.configured` に残す。
- `external_source` は公式の role 付き形式と no-role 形式だけを recognized grammar とし、それ以外の slash form は opaque metadata + warning にする。
- plan output は `access` ではなく `access_mode` を使う。
- `verification_hint` は configured hint、`verification_evidence` は resolved evidence として分け、plan では `configured_hint` を保持する。
- `--stable` は `checked_at`, `volatile`, `docs_last_checked` のような volatile audit field を省略する。
- projection matrix は `source_url`, `docs_last_updated`, `docs_last_checked`, `generator_matrix_version`, row-level `evidence_kind` を持つ。
- lossy projection は relation-level fact と selected output impact を分ける。
- hidden column omission の default severity は `info` にする。
- external dataset table の source / target は AST role として扱い、`select_source` は許可、DML / metadata target は planning error にする。
- PostgreSQL-dialect Spanner source は専用 diagnostic で失敗させる。
- suppressed diagnostics は machine-readable vet output に `suppressed: true` と suppression metadata 付きで残す。

## Internal worktree claims

この表は内部追跡用です。外部レビューでは次節の expected output contract を見てください。

| Behavior | Claimed status |
| --- | --- |
| Specific planning diagnostic IDs | reflected for classified external dataset and federated placeholder failures |
| Canonical plan `location` | reflected |
| Official no-role `google-cloudspanner:/projects/...` source | reflected |
| Unofficial slash forms such as `google-cloudspanner:///projects/...` | preserved as opaque metadata with warning |
| `access_mode` plan key | reflected |
| `configured_hint` in `access_verification` | reflected |
| Stable plan volatile-field omission | reflected |
| Projection matrix provenance rows | reflected |
| Relation projection split | reflected |
| Hidden column severity `info` | reflected |
| AST role-based external dataset target rejection | reflected |
| PostgreSQL-dialect unsupported diagnostic | reflected |
| Suppressed diagnostic metadata | reflected |

## External review contracts

### 1. Connection location mismatch diagnostic

Minimal config:

```yaml
package: querydemo
schemas:
- name: app_spanner
  dialect: spanner
  ddl: spanner.sql
- name: analytics_bigquery
  dialect: bigquery
  spanner_external_datasets:
  - dataset: analytics_spanner
    spanner_source: app_spanner
    location: us
    access:
      mode: cloud_resource
      cloud_resource_connection: example-project.eu.example-connection
queries:
- name: ListSingers
  source: analytics_bigquery
  sql: SELECT SingerId FROM analytics_spanner.Singers
  result_struct: SingerRow
```

Command:

```sh
spanner-query-gen vet --config connection-location-mismatch.yaml --output yaml
```

Expected exit code: `1`

Expected stderr: empty

Expected stdout contains these fields:

```yaml
diagnostics:
- id: external-dataset-connection-location-mismatch
  stage: external_dataset_binding
  subject: spanner_external_datasets.access.cloud_resource_connection
  scope: plan
  name: spanner-query-gen
  rule: external-dataset-connection-location-mismatch
  severity: error
  suppressible: false
  suppressed: false
```

`planning-error` must not be used here except as an unclassified fallback in future unrelated failures.

### 2. Canonical location and access mode plan shape

Command:

```sh
spanner-query-gen explain-plan --config external-dataset.yaml --output yaml
```

Expected plan fields:

```yaml
catalog_bindings:
- kind: spanner_external_dataset
  bigquery_dataset: example-project.analytics_spanner
  location: US
  location_metadata:
    configured: us
    canonical: US
  cloud_resource_connection: example-project.us.example-connection
  cloud_resource_connection_metadata:
    id: example-project.us.example-connection
    parsed_location: us
    parsed_location_canonical: US
    location_match: true
  access_mode: cloud_resource_connection
```

The top-level `location` is the canonical value. Consumers should not need to read `location_metadata.canonical` as the primary value.

### 3. External source accepted grammar

Accepted recognized forms:

```text
google-cloudspanner://DATABASE_ROLE@/projects/<project>/instances/<instance>/databases/<database>
google-cloudspanner:/projects/<project>/instances/<instance>/databases/<database>
```

Role-bearing source contract:

```yaml
external_source: google-cloudspanner://reader@/projects/example-project/instances/app/databases/app
database_role: reader
database_role_source: external_source
```

Official no-role source contract:

```yaml
external_source: google-cloudspanner:/projects/example-project/instances/app/databases/app
database_role: ""
database_role_source: ""
```

Unofficial slash forms such as `google-cloudspanner:///projects/...` are not recognized. They remain in metadata and produce:

```yaml
warnings:
- id: external-dataset-external-source-unrecognized
  stage: external_dataset_projection
  severity: warning
  suppressible: true
  suppressed: false
```

### 4. Verification hint and evidence semantics

Config:

```yaml
access:
  verification_hint: not_checked
  verification_evidence:
    status: verified
    source: external_evidence
    verifier: terraform-plan
    checked_at: "2026-05-04T10:30:00Z"
```

Expected plan fields:

```yaml
access_verification:
  status: verified
  source: external_evidence
  configured_hint: not_checked
  verifier: terraform-plan
  checked_at: "2026-05-04T10:30:00Z"
  independently_verified_by_generator: false
  volatile: true
```

`verification_hint` is intentionally not allowed to carry proof-like values such as `verified`; those must be supplied as `verification_evidence`. Invalid proof-like hints or legacy/evidence conflicts use:

```yaml
diagnostics:
- id: external-dataset-access-verification-conflict
  stage: external_dataset_binding
  severity: error
  suppressible: false
```

This is stricter than a suppressible warning because it prevents two competing sources of truth from entering the plan.

### 5. Stable plan output

Normal plan output may include volatile audit metadata:

```yaml
access_verification:
  checked_at: "2026-05-04T10:30:00Z"
  volatile: true
projection_matrix:
  docs_last_checked: "2026-05-05"
```

Stable plan output:

```sh
spanner-query-gen explain-plan --config external-dataset.yaml --output yaml --stable
```

Expected stable invariant:

```text
checked_at: absent
volatile: absent
docs_last_checked: absent
semantic plan fields and diagnostic IDs: preserved
```

The projection matrix still keeps the documented source version:

```yaml
projection_matrix:
  source: google_cloud_docs
  source_url: https://docs.cloud.google.com/bigquery/docs/spanner-external-datasets
  docs_last_updated: "2026-05-02"
  generator_matrix_version: 1
```

### 6. Projection matrix row provenance

Expected plan shape:

```yaml
projection_matrix:
  source: google_cloud_docs
  source_url: https://docs.cloud.google.com/bigquery/docs/spanner-external-datasets
  docs_last_updated: "2026-05-02"
  docs_last_checked: "2026-05-05"
  generator_matrix_version: 1
  rows:
  - object: default_schema_table
    behavior: project
    default_severity: ok
    evidence_kind: documented
    source_url: https://docs.cloud.google.com/bigquery/docs/spanner-external-datasets
  - object: hidden_column
    behavior: omit
    default_severity: info
    evidence_kind: generator_safety_policy
  - object: case_folded_column_collision
    behavior: reject
    default_severity: error
    evidence_kind: generator_safety_policy
  - object: timestamp_runtime_precision
    behavior: project_with_caveat
    default_severity: warning
    evidence_kind: currently_inferred
```

`docs_last_checked` is generator audit metadata and is omitted by `--stable`; `docs_last_updated` is the referenced documentation version and remains.

### 7. Relation-level projection versus selected output impact

Explicit-column query:

```sql
SELECT SingerId, FirstName
FROM analytics_spanner.Singers
```

Expected plan excerpt:

```yaml
relations:
- sql_path: analytics_spanner.Singers
  source: spanner_external_dataset_projection
  role: select_source
  allowed: true
  projection_loss: true
  projection:
    relation_has_omitted_columns: true
    selected_output_affected: false
    omitted_columns:
    - Singers.SearchName
```

Star query:

```sql
SELECT *
FROM analytics_spanner.Singers
```

Expected plan excerpt:

```yaml
star_expansion:
  source: external_dataset_projection
  projection_loss: true
  projection:
    relation_has_omitted_columns: true
    selected_output_affected: true
    omitted_columns:
    - Singers.SearchName
```

This keeps relation provenance without falsely implying that every explicit-column result is missing a selected field.

### 8. External dataset relation role rejection

Allowed source role:

```yaml
relations:
- sql_path: analytics_spanner.Singers
  source: spanner_external_dataset_projection
  role: select_source
  allowed: true
  writable_target: false
```

DML target query:

```sql
INSERT INTO analytics_spanner.Singers (SingerId) VALUES (1)
```

Expected diagnostic:

```yaml
diagnostics:
- id: external-dataset-dml-target-unsupported
  stage: bigquery_analysis
  severity: error
  suppressible: false
  subject: analytics_spanner.Singers
```

Metadata target query:

```sql
CREATE TABLE analytics_spanner.NewTable (SingerId INT64)
```

Expected diagnostic:

```yaml
diagnostics:
- id: external-dataset-metadata-target-unsupported
  stage: bigquery_analysis
  severity: error
  suppressible: false
  subject: analytics_spanner.newtable
```

### 9. PostgreSQL-dialect external dataset source

Config:

```yaml
schemas:
- name: app_spanner_pg
  dialect: spanner_postgresql
  ddl: schema.sql
- name: analytics_bigquery
  dialect: bigquery
  spanner_external_datasets:
  - dataset: analytics_spanner
    spanner_source: app_spanner_pg
```

Expected diagnostic:

```yaml
diagnostics:
- id: external-dataset-postgresql-dialect-unsupported
  stage: external_dataset_projection
  severity: error
  suppressible: false
  subject: schemas.app_spanner_pg
```

The service can link PostgreSQL-dialect Spanner databases, but this generator currently projects only Spanner GoogleSQL DDL.

### 10. Suppressed diagnostics remain visible

Config:

```yaml
vet:
  disable:
  - rule: external-dataset-access-unverified
    reason: Static CI verifies external dataset access outside this generator.
    owner: analytics-platform
    expires: "2026-12-31"
```

Command:

```sh
spanner-query-gen vet --config suppressed-access.yaml --output yaml
```

Expected exit code: `0`

Expected stderr: empty

Expected stdout contains these fields:

```yaml
diagnostics:
- id: external-dataset-access-unverified
  stage: external_dataset_projection
  severity: warning
  suppressible: true
  suppressed: true
  suppression:
    scope: catalog_binding
    name: analytics_spanner
    rule: external-dataset-access-unverified
    reason: Static CI verifies external dataset access outside this generator.
    owner: analytics-platform
    expires: "2026-12-31"
```

Projection facts such as omitted columns, omitted named-schema tables, and relation provenance remain in `explain-plan`; suppression changes diagnostic presentation only.

### 11. Federated placeholder diagnostics

The `__external__` placeholder still has concrete diagnostic IDs:

```yaml
diagnostics:
- id: federated-placeholder-missing
  stage: federated_query_analysis
  severity: error
  suppressible: false
```

```yaml
diagnostics:
- id: federated-placeholder-duplicate
  stage: federated_query_analysis
  severity: error
  suppressible: false
```

```yaml
diagnostics:
- id: federated-placeholder-invalid-position
  stage: federated_query_analysis
  severity: error
  suppressible: false
```

## Documentation provenance

The external dataset projection matrix is based on the Google Cloud page:

- https://docs.cloud.google.com/bigquery/docs/spanner-external-datasets

The checked document metadata reported `updateTime: 2026-05-02T03:51:23Z`, so the plan uses:

```yaml
docs_last_updated: "2026-05-02"
docs_last_checked: "2026-05-05"
```

`docs_last_checked` is intentionally volatile generator audit metadata.

## 残す論点

- `database_role_source: config_and_external_source` の分解はまだ未実装です。将来は `database_role.value`, `database_role.sources`, `database_role.consistency` のような shape にできますが、現時点では既存 plan schema 変更の範囲を最小化します。
- Global rule policy はまだ設計中です。現時点では binding-level `unsupported_columns` / `named_schema_policy` と `vet.disable` を維持します。
- BigQuery execution surface で `INSERT INTO native_table SELECT ... FROM external_dataset.table` や `MERGE native_table USING external_dataset.table` を生成するかどうかは、query method generation の設計が固まってから扱います。今回の contract は external dataset table 自体を target にしないことだけを固定します。

## 結論

round 6 の指摘はほぼ取り込みます。主な変更点は、機能追加というより、CI と downstream tooling が message string ではなく stable diagnostic ID / stable plan field に依存できるようにすることです。

今後のレビュー回答でも、実装コードが共有されない場合は、test 名よりも external review bundle と expected output contract を優先して書きます。
