# `spanner-query-gen` external dataset round 3 review response

## Summary

レビューありがとうございます。今回の指摘は、ほぼそのまま採用します。

大きな設計転換は不要で、以後は external dataset の static projection が実サービスの
挙動とずれた時に、それを `explain-plan` / `vet` / tests でどう見える化するかを
詰める段階だと理解しています。

## 採用する点

### Current Scope と Roadmap の整理

Current Scope には「現時点で実装済みの external dataset binding declaration、
static projection、basic diagnostics」を残し、Roadmap Phase 4.5 は
“Enhance BigQuery Spanner External Dataset Catalog Integration” として、
field-level provenance、stricter vet rules、optional live verification などの
拡張項目に寄せます。

### `unsupported_columns: omit` の query behavior

`unsupported_columns: omit` は「binding/projection build では fatal にしない」という
意味に固定します。

- projection build: unsupported column is omitted and reported as a warning
- `SELECT *`: projected BigQuery-visible columns only
- explicit omitted-column reference: planning error

Named schema table も同様に、projection から omit することと明示参照を許すことは
分けます。明示参照は unresolved table / unsupported table として planning error にします。

### Projected column name collision

External dataset table names だけでなく、projected column names も case-folding
collision を検出します。BigQuery catalog として安全に表現できない `Foo` / `foo`
のような列は projection error にします。

### BigQuery catalog collision

BigQuery DDL と external dataset projection が同じ canonical table path を定義する場合は
planning error にします。project-qualified path と default-project 由来の unqualified alias の
どちらも silent override しない方針にします。

### External dataset type matrix

`EXTERNAL_QUERY` output conversion matrix と external dataset projection matrix は分けます。
前者は unsupported output が query failure なので error、後者は unsupported column が
BigQuery に見えないため default `omit` という違いを DESIGN に明記します。

### Read-only target semantics

“read-only” は「external dataset table を data modification target にできない」という意味に
限定して表現します。将来 BigQuery execution methods を扱う場合でも、external dataset table を
source として通常 BigQuery table に書き込む形まで雑に reject しないようにします。

### Access verification shape

`access_verified: false` は「検証して false」と「未検証」が曖昧なので、plan では
`access_verification.status` に寄せます。値は
`not_checked|verified|mismatch|failed` とし、まずは static generator として正直な
default である `not_checked` を出す形にします。

### Optional external resource metadata

generator は BigQuery dataset / connection / IAM / Terraform を作りませんが、review 用 metadata
として `external_source`, `location`, `connection` は受けられるようにします。これは codegen
の source of truth ではなく、plan 上の意図を明確にするための任意 metadata とします。

### Database role visibility warning

`database_role` が設定され、live verification されていない場合は、DDL-first projection が
runtime visibility を過大推定する可能性を warning として出します。

### Binding-level suppression

External dataset の warning は query ではなく schema binding に属するものが多いため、
`schemas[].spanner_external_datasets[].vet.disable` を追加します。将来の structured
`rules.suppressions` with selector は design work として残します。

### `vet` output contract

`vet --output text`、`vet --output json|yaml` の stdout/stderr/exit code 契約を README/DESIGN に
固定します。JSON/YAML mode では machine-readable diagnostics を stdout に集約し、stderr は
fatal CLI errors 用に残します。

### `__external__` replacement phase

`__external__` は BigQuery identifier ではなく config-level placeholder です。BigQuery SQL
analysis の前に generated `EXTERNAL_QUERY(...)` table expression へ置換する、と明文化します。

### PostgreSQL-dialect Spanner external dataset

公式 BigQuery external dataset は Spanner PostgreSQL dialect database も対象にできますが、
この generator はまだ Spanner GoogleSQL DDL/analyzer を前提にします。PostgreSQL-dialect
external dataset は明示的に unsupported と書きます。

## 今回すぐ進める範囲

- projected column name case-folding collision の error 化
- BigQuery DDL table と external dataset projection / alias の collision error 化
- `unsupported_columns: omit` の explicit column reference error を regression test で固定
- named schema omitted table の explicit reference error を regression test で固定
- external dataset projection matrix と read-only target semantics の docs 追記
- access verification を plan metadata として `not_checked|verified` 形へ寄せる
- `external_source` / `location` / `connection` optional metadata の plan 出力
- `database_role` without verification warning
- binding-level `vet.disable`
- `vet` stream/exit contract と `__external__` replacement phase の docs 追記
- Current Scope と Phase 4.5 の表現整理

## 後続に残す範囲

- warning-specific `--strict` / warnings-as-errors
- structured global `rules.suppressions` with selector
- role-based allowlist
- optional live BigQuery / Spanner verification
- precise source spans for catalog binding diagnostics
- BigQuery execution methods and source-vs-target DML handling
- Spanner PostgreSQL-dialect external dataset support

## Closing

この round では、feature scope を広げるよりも、static projection の failure mode を明確にする
ことを優先します。external dataset は query text に実体が出にくいため、catalog binding plan
の精度が信頼性そのものになります。今回の P0 は実装・test・docs まで進めます。
