# `spanner-query-gen` external dataset round 4 review response

## 前提の確認

指摘のとおり、相手側には Markdown しか共有していない前提で考えます。
そのため、以後の回答では「仕様として書いたこと」と「こちらの作業ツリーで実装・テストしたこと」を
混同しないように、代表 test 名と expected behavior を併記します。

## 総評への回答

大きな設計転換は不要という評価に同意します。external dataset は引き続き
`EXTERNAL_QUERY` の別記法ではなく BigQuery catalog binding として扱います。
Current Scope は basic static catalog binding、Phase 4.5 は live verification や
より厳密な vet policy など production hardening という整理を維持します。

## Specified / Implemented / Tested

| Behavior | Specified | Implemented | Tested evidence |
| --- | --- | --- | --- |
| `unsupported_columns: omit` で `SELECT *` が BigQuery-visible columns のみを見る | yes | yes | `TestBigQueryAnalyzerSpannerExternalDatasetOmitPoliciesStillRejectExplicitReferences`, `TestGenerateQueryCodeBigQuerySpannerExternalDataset` |
| explicit omitted column reference は analyzer error | yes | yes | `TestBigQueryAnalyzerSpannerExternalDatasetOmitPoliciesStillRejectExplicitReferences` |
| named schema table omission は explicit reference で table-not-found | yes | yes | `TestBigQueryAnalyzerSpannerExternalDatasetOmitPoliciesStillRejectExplicitReferences` |
| projected table case-folding collision は planning error | yes | yes | `TestBigQueryAnalyzerSpannerExternalDatasetRejectsCaseInsensitiveTableNameConflict` |
| projected column case-folding collision は planning error | yes | yes | `TestBigQueryAnalyzerSpannerExternalDatasetRejectsCaseInsensitiveColumnNameConflict` |
| BigQuery DDL table と external dataset projection の collision は planning error | yes | yes | `TestBigQueryAnalyzerSpannerExternalDatasetRejectsCatalogTableCollision` |
| `database_role` + `not_checked` は warning | yes | yes | `TestGenerateQueryCodeBigQuerySpannerExternalDataset` |
| binding-level `vet.disable` は warning 表示だけを抑制し projection metadata は残す | yes | yes | `TestRunVetReportsWarnings`, `TestGenerateQueryCodeBigQuerySpannerExternalDataset` |
| `__external__` は string/comment 内を数えず token として exactly once | yes | yes | `TestGenerateQueryCodeBigQueryFederatedQueryPlaceholderIgnoresLiteralsAndComments` |
| JSON/YAML `vet` output は planning error を machine-readable diagnostic として stdout に出す | yes | yes | `TestRunVetJSONReportsPlanErrors` |
| JSON/YAML `vet` output は config parse error を machine-readable diagnostic として stdout に出す | yes | yes | `TestRunVetYAMLReportsConfigParseErrors` |
| diagnostic schema に `stage` / `suppressible` / `suppressed` を含める | yes | yes | `TestRunVetJSON`, `TestRunVetYAML`, `TestRunVetJSONReportsPlanErrors` |
| BigQuery DDL と external dataset projection の collision error に source kind を含める | yes | yes | `TestBigQueryAnalyzerSpannerExternalDatasetRejectsCatalogTableCollision` |
| external dataset の visible `TIMESTAMP` / `NUMERIC` caveat を warning として残す | yes | yes | `TestBigQueryAnalyzerSpannerExternalDataset` |

この表は今後の response にも残し、実装コードを共有しない設計レビューでも確認しやすくします。

## Access verification

`access.verification: verified` のような user assertion と verified fact が混ざる形は避けます。
config 側は `access.verification_hint` を user-declared metadata とし、observed fact は
structured `access_verification` に分けます。

```yaml
access:
  mode: cloud_resource
  database_role: reader
  verification_hint: not_checked

access_verification:
  status: verified
  verifier: terraform-plan
  checked_at: "2026-05-04T10:30:00Z"
  evidence_digest: sha256:...
```

`verified|mismatch|failed` を config に書く場合は `verifier` と `checked_at` を必須にします。
`verification_hint` は `not_checked` 以外を受け付けず、plan では
`access_verification.status/source/checked_at/verifier/evidence_digest` を出します。

## Access metadata consistency

`external_source` に database role が含まれる場合は、`access.database_role` と整合させます。

- `external_source` role と `access.database_role` が不一致なら planning error。
- `external_source` role だけがある場合は plan に `database_role_source: external_source` として残す。
- 両方が一致する場合は `database_role_source: config_and_external_source` として残す。
- `access.database_role` だけがある場合は `database_role_source: config` として残す。

`access.mode` は config alias と plan canonical value を分けます。
config では `euc` / `cloud_resource` を受け、plan では
`end_user_credentials` / `cloud_resource_connection` を出します。

また、`connection` が `project.location.connection` 形式で parse できる場合は `location` と照合します。
location mismatch は planning error、`cloud_resource_connection` なのに `connection` がない場合は warning、
`end_user_credentials` なのに `connection` がある場合も warning にします。

## Projection loss

`vet.disable` は diagnostic policy にだけ作用させ、projection metadata は消しません。
`projected_tables` には omitted columns / omitted named-schema tables を残します。

さらに `SELECT *` が external dataset projection から展開された場合は、query plan に
`star_expansion` を残します。

```yaml
star_expansion:
  source: external_dataset_projection
  projection_loss: true
  omitted_columns:
  - Singers.SearchName
```

これにより、DTO が underlying Spanner table の全列ではなく BigQuery-visible projection の列から
生成されたことを plan review で確認できます。

## `__external__` placeholder

raw string replacement は採用しません。`__external__` は lexical identifier token として検出し、
string literal、quoted identifier、comment 内の `__external__` は placeholder count に含めません。
置換は analyzer に渡す前に実行します。

FROM-position や alias shape のさらなる厳密検証は後続 work としますが、現在の方針でも
`SELECT * FROM __external__ AS e` のような alias 付き outer SQL は、置換後の BigQuery SQL として
analyzer に渡されます。

## 保留したもの

JSON/YAML `vet` output で planning error も machine-readable diagnostics にする提案と、
diagnostic schema に `stage` / `suppressible` / `suppressed` を入れる提案は実装しました。
fatal CLI error は引き続き stderr ですが、config parse error と planning error は
JSON/YAML report として stdout に出し、non-zero status を返します。

external dataset の `TIMESTAMP` / `NUMERIC` caveat は、runtime behavior を断定しない wording で
warning として残します。これは `EXTERNAL_QUERY` と同じ挙動を仮定するものではなく、
generated consumers が precision/range/loader assumption を review できるようにするための
conservative diagnostic です。
