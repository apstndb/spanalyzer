# Response to external dataset feedback for `spanner-query-gen`

Review source: `spanner-query-gen-response-external-dataset-feedback-ja.md`

## Summary

フィードバックの大半に同意します。ただし一点、BigQuery Spanner external dataset
support はユーザー要望なので、単なる future non-goal ではなく、明示的な将来対象として
扱います。

方針としては、レビューの提案どおり `EXTERNAL_QUERY` の別記法にはしません。
external dataset は query-level federation ではなく dataset-level federation なので、
BigQuery catalog に Spanner DDL 由来の external table projection を追加する
integration として扱います。

## Applied now

### `vet` scope clarification

README/DESIGN には、current `vet` と future full rule engine の違いを明確にします。

Current `vet`:

- code generation と同じ resolved plan を構築する。
- planning errors で失敗する。
- implemented warning rules を structured warnings として plan に出す。
- suppressions は plan に保存する。

Future rule engine:

- per-rule severity policy。
- `--strict` / warnings-as-errors。
- project-level rule configuration。
- rule registry / custom rules。

これにより、structured warnings は実装済みだが full rule engine は deferred という状態を
利用者が誤読しにくくします。

### Compact vs structured update mask

`auto_all_non_key_columns` を list sentinel として長期化しない方針に同意します。

current compact config では既存の `update_mask: [auto_all_non_key_columns]` を残します。
structured config では、文字列 sentinel と column list を混ぜず、次のような別 shape に
移行する方針を DESIGN に入れます。

```yaml
update:
  mask:
    columns:
    - FirstName
    - LastName
```

```yaml
update:
  mask:
    all_non_key_columns: true
```

### README sample validity

`insert_or_update` の insert branch validation を強化したため、README の examples は
validation と矛盾しないように維持します。特に `NOT NULL` で default/server value がない列を
含む sample DDL では、`update_mask` に required insert columns を含めるか、sample schema 側で
nullable/default を明示します。

### External dataset support

external dataset support は実装対象として扱います。ただし、`queries[].federated` には
入れません。

初期 implementation の shape:

```yaml
schemas:
- name: analytics_bigquery
  dialect: bigquery
  spanner_external_datasets:
  - dataset: example-project.analytics_spanner
    spanner_source: app_spanner
```

The generator projects the Spanner DDL tables into the BigQuery catalog under
`example-project.analytics_spanner.*`. Queries stay ordinary BigQuery SQL.

The projection follows the documented BigQuery Spanner external dataset limits:

- only default Spanner schema tables are visible;
- primary/foreign key metadata is not visible to BigQuery;
- unsupported Spanner columns are not visible on the BigQuery side;
- external dataset tables are read-only;
- `INFORMATION_SCHEMA` and metadata changes are not part of this catalog binding;
- Data Boost and access delegation/EUC notes belong in plan diagnostics/docs, not runtime setup.

### Config naming

`external_schemas` is currently kept as a backward-compatible compact alias for
`EXTERNAL_QUERY` connection mapping. Structured config should use the narrower
name:

```yaml
external_query_connections:
- connection: example-project.us.example-connection
  spanner_source: app_spanner
```

External datasets use a separate key:

```yaml
spanner_external_datasets:
- dataset: example-project.analytics_spanner
  spanner_source: app_spanner
```

This keeps `EXTERNAL_QUERY` and external dataset semantics distinct.

## Deferred intentionally

### Creating BigQuery external datasets

The generator will not create BigQuery datasets, connections, IAM bindings, or
Terraform resources. That belongs to infrastructure configuration. This tool only
models the static catalog shape needed for query analysis and code generation.

### Live BigQuery introspection

The default workflow remains DDL-first. Live BigQuery metadata can be considered
later as verification, but it should not become the source of truth for CI.

### PostgreSQL-dialect Spanner external datasets

Official external datasets can link to GoogleSQL or PostgreSQL Spanner databases,
but this tool is currently centered on Spanner GoogleSQL DDL. PostgreSQL support
should be a separate dialect decision, not implicit external dataset behavior.

### Full external dataset vet policy

Initial support can expose plan diagnostics and warnings. A strict policy engine
for expired suppressions, warnings-as-errors, source spans, and custom rules remains
future work.

## Adjusted interpretation

The previous DESIGN text said BigQuery Spanner external datasets were an initial
non-goal. That remains true only for the first federation slice. Since external
dataset support is now explicitly desired, the text should say:

> Do not support BigQuery Spanner external datasets in the initial `EXTERNAL_QUERY`
> slice. When added, support them as a separate BigQuery catalog integration, not
> as a federated-query shorthand.

## References checked

- https://docs.cloud.google.com/bigquery/docs/spanner-external-datasets

## Validation performed

- `go test ./...`
- `mise x -- golangci-lint run --timeout=5m`
- `git diff --check`
