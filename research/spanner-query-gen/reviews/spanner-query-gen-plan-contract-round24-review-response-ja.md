# Response to plan contract round 24 review

## Summary

今回のフィードバックは、機能追加ではなく plan-report artifact の
provenance と contract candidate の誤用防止を締めるものとして受け取りました。
大きな設計変更は入れず、v1alpha の公開 contract を少し強くしました。

## 取り込んだ変更

### `backend_identity.source: manual` の invariant を固定

`backend_identity.source: manual` は caller-supplied assertion であり、
観測された backend evidence ではない、という wording に寄せました。

schema と runtime self-check の両方で、次を固定しました。

```text
if backend_identity.source == manual:
  version != not_recorded OR image_digest != not_recorded

if backend_identity.source == not_recorded:
  version == not_recorded AND image_digest == not_recorded
```

`source: spanemuboost` は現状どおり、version / image digest が
`not_recorded` のままでも許容します。これは backend/runtime 経由で identity を
得ようとしたが、安定した version/digest API がまだない、という意味に閉じています。

per-field provenance は v1alpha では採用していません。将来、
version は自動取得、image digest は手動指定、のような混在が必要になった時点で
検討します。

### contract evaluation mode 時の空 evaluations を禁止

plan-report schema で、
`contract_evaluation_mode: report_only | check` の場合は
`contract_evaluations.minItems: 1` になるようにしました。

runtime self-check でも同じ invariant を追加しました。これにより、
contract file が指定されているのに空の evaluation artifact が出る状態は
schema と runtime の両方で弾かれます。

### `spool_name` candidate の guard を明記

`PLAN_CONTRACT_CANDIDATES.md` の future spool consistency check に、
`spool_name != ""` guard を入れた CEL 例を追加しました。

CEL input では absent optional string metadata が `""` になるため、
`spool_scan` があるのに `spool_name` が空の場合は silent pass ではなく fail する
形の例にしています。

```cel
operators
  .filter(o, o.family == "spool_scan")
  .all(scan,
    scan.spool_name != "" &&
    operators.exists(build,
      build.family == "spool_build" &&
      build.spool_name == scan.spool_name))
```

### `spool_name` stability detection の最小 fixture を追加

`operators.exists(o, o.family == "spool_scan" && o.spool_name != "")`
のように `spool_name` だけを読む CEL について、stability reason が
field-specific に出ることを test で固定しました。

期待する reason は次の最小形です。

```yaml
reasons:
- contract uses the normalized plan-report view
- "contract reads metadata-derived normalized fields: spool_name"
```

### raw CEL variable 名を candidate 側にも追記

`PLAN_CONTRACT_CANDIDATES.md` に、raw CEL variables は
`raw_plan` / `raw_nodes` であり、serialized YAML/JSON report からは replay できない
evaluator input である、という一文を追加しました。

`PLAN_CONTRACTS.md` だけでなく candidate document 単体でも混同しにくくするためです。

## Schema artifacts

更新後の schema digest は次の通りです。

```text
schemas/spanner-query-gen.plan-report.v1alpha.schema.json sha256: 1c61edea0d08b3523cd6856408acbdccce4759182d62b172c992b8115d9eff98
schemas/spanner-query-gen.plan-contracts.v1alpha.schema.json sha256: f020bf302f2d540a706525e84cbfa02061e5313248a01382573bb23ada7b1baf
```

今回変更したのは plan-report schema です。plan-contract schema は digest を併記して、
review archive 上で現在参照している artifact set を追いやすくしています。

## 見送ったもの

- `backend_identity` の per-field provenance object 化は見送りました。
  v1alpha では single `source` + invariant で十分と判断しています。
- `spool_consistency` の predefined contract 化はまだ行っていません。
  まず candidate と CEL example として育てます。
- raw QueryPlan contract の external replay support は引き続き対象外です。
  `raw_plan` / `raw_nodes` は evaluator input であり、report artifact の
  stable replay surface ではありません。
- PROFILE / execution stats contract は引き続き対象外です。plan-report は PLAN
  artifact の structural review に限定します。

## 現時点の回答

指摘された P1 はすべて取り込みました。
`backend_identity.source: manual` の意味は caller assertion として固定し、
manual / not_recorded の invariant を schema と runtime self-check の両方に入れました。
また、contract evaluation mode では空の `contract_evaluations` を禁止しました。

P2 については、`spool_name` の field-specific stability test と
candidate document の raw variable 名追記まで取り込みました。
schema digest は response に残す運用を始めます。

この段階では新しい predefined contract を増やすより、現在の plan-report /
plan-contract artifact の意味を誤読しにくくする方を優先する、という方針を維持します。
