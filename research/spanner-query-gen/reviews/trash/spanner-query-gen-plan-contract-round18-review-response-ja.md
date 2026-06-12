# spanner-query-gen plan contract round 18 response

レビューありがとうございます。round 18 は、新しい predefined contract を増やす
話ではなく、外部 review artifact として同じ意味で読み直せるかを締める指摘として
扱いました。指摘された項目は大きな設計変更なしで反映しました。

## 反映したこと

### CEL input default semantics を artifact に明示

`PLAN_CONTRACTS.md` に、CEL は serialized YAML/JSON を直接評価するのではなく、
tool 内部の normalized CEL input を評価することを明記しました。

- `operators[]` と `operator_edges[]` の optional string fields は CEL では `""`
  として見える
- optional boolean fields は CEL では `false` として見える
- 外部 evaluator が report から CEL を replay する場合は同じ default を適用する

また、plan-report 本体にも machine-readable metadata を追加しました。

```yaml
normalization:
  cel_input_defaults:
    optional_string: ""
    optional_boolean: false
    applies_to:
    - operators[]
    - operator_edges[]
```

schema も更新済みです。

### root-partitionable CEL を strict variant に変更

`ApproximateRootPartitionablePlanShape` の例に
`operator_family_counts["unknown"] == 0` を追加しました。

これは candidate/CEL example のままですが、CI contract として使う場合に、未知の
distributed operator が zero-distributed-fragment branch を silent pass しないように
するためです。loose な探索用に外すことは可能ですが、その場合は
`classification_warnings` を併せて読むべき、という説明も加えました。

### sort contracts と `ALLOW_DISTRIBUTED_MERGE` の関係を追記

`PLAN_CONTRACTS.md` の optimizer section に、sort 系 contract が
`ALLOW_DISTRIBUTED_MERGE` の影響を受けることを明記しました。

v1alpha の `--require-optimizer-pinning` は、この hint を optimizer pinning としては
扱いません。hint に依存する contract では rendered SQL、SQL digest、
optimizer matrix evidence を併せて確認する、という整理にしています。

### `scan_type` の表記ゆれを修正

`PLAN_CONTRACT_CANDIDATES.md` の
“Require A Specific Index Scan And No Table Scan” に残っていた
`IndexScan` / `TableScan` を normalized spelling に揃えました。

```cel
operators.exists(o,
  o.scan_target == "SongsBySongNameStoring" &&
  o.scan_type == "index_scan") &&
operators.all(o, o.scan_type != "table_scan")
```

### contract name uniqueness を明文化

`contracts[].name` は schema の `uniqueItems` ではなく runtime validation で
重複拒否する、という点を `PLAN_CONTRACTS.md` に追記しました。
`contract_evaluations[].name` を stable key として扱えるようにするためです。

## 反映しなかったこと

`stability.features` のような machine-readable stability feature list は今回入れて
いません。現在の `stability.reasons` で意図は説明できており、v1alpha で急ぐ必要は
ないというレビュー判断に同意しています。

serialized report の optional string / boolean fields をすべて明示出力する案も
採用していません。artifact が冗長になるため、今回は
`normalization.cel_input_defaults` と docs で replay semantics を示す方にしました。

## 検証

```sh
GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go run ./cmd/spanner-query-gen plan-report-schema --out schemas/spanner-query-gen.plan-report.v1alpha.schema.json
GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go test ./cmd/spanner-query-gen ./internal/plancontract
GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go test ./...
git diff --check
```
