"""Validate retained public-fixture plan-report artifacts, not runtime execution."""
import hashlib
import importlib.metadata
import json
from pathlib import Path
import sys

from jsonschema import Draft202012Validator
import yaml

repo = Path(sys.argv[1]).resolve()
evidence = Path(sys.argv[2]).resolve()
checks = []
validators = {}
artifacts = {}
schema_hashes = {}

def check(name, condition):
    if not condition:
        raise AssertionError(name)
    checks.append(name)

for kind, filename in [("full", "spanner-query-gen.plan-report.v1alpha.schema.json"), ("summary", "spanner-query-gen.plan-summary.v1alpha.schema.json")]:
    schema_path = repo / "schemas" / filename
    schema_bytes = schema_path.read_bytes()
    schema = json.loads(schema_bytes)
    Draft202012Validator.check_schema(schema)
    validators[kind] = Draft202012Validator(schema)
    schema_hashes[filename] = hashlib.sha256(schema_bytes).hexdigest()
    for ext in ["json", "yaml"]:
        data = (evidence / f"{kind}.{ext}").read_text()
        artifact = json.loads(data) if ext == "json" else yaml.safe_load(data)
        validators[kind].validate(artifact)
        checks.append(f"{kind}.{ext} validates against Draft 2020-12 schema")
        if ext == "json":
            artifacts[kind] = artifact
        else:
            check(f"{kind} JSON/YAML semantic equality", artifact == artifacts[kind])

full, summary = artifacts["full"], artifacts["summary"]
check("target accounting unchanged", full["target_summary"] == summary["target_summary"])
check("contract summary unchanged", full["contract_summary"] == summary["contract_summary"])
check("version-only pin leaves statistics warning", "statistics_package_not_pinned" in summary["contract_summary"]["environment_warnings"] and "query_optimizer_not_pinned" not in summary["contract_summary"]["environment_warnings"] and "optimizer_not_pinned" not in summary["contract_summary"]["environment_warnings"])
for original, projected in zip(full["queries"], summary["queries"], strict=True):
    identity = original["target_id"]
    check(f"{identity} no heavy query fields", not ({"plan", "sql", "normalized_operators", "operator_edges", "operator_families", "operator_family_counts"} & projected.keys()))
    for field in ["target_id", "status", "sql_sha256", "ddl_sha256", "operator_tree_sha256", "access_paths", "error"]:
        check(f"{identity} {field} unchanged", original.get(field) == projected.get(field))
    check(f"{identity} version pinned", not projected.get("optimizer_not_pinned", False))
    if "variant" in projected:
        check(f"{identity} variant omits SQL", "sql" not in projected["variant"] and "sql_sha256" not in projected["variant"])
for original, projected in zip(full["contract_evaluations"], summary["contract_evaluations"], strict=True):
    check("summary replay disabled", projected["stability"]["replayable_from_report"] is False)
    check("live evaluation results unchanged", original["status"] == projected["status"] and original.get("results") == projected.get("results"))

for kind in ["full", "summary"]:
    unknown = json.loads(json.dumps(artifacts[kind]))
    path = unknown["queries"][0]["access_paths"][0]
    for field in ["scan_target", "scan_type", "base_table", "seekable_key_size"]:
        path[field] = ""
    path["scan_target_kind"] = "unknown"
    for group in ["seek_conditions", "residual_conditions", "timestamp_conditions", "search_conditions"]:
        for condition in path[group]:
            condition["expression"] = ""
    validators[kind].validate(unknown)
    checks.append(f"{kind} permits explicit unknown metadata")

# Negative controls catch accepting incompatible full artifacts or omitted evidence.
check("full schema rejects summary artifact", bool(list(validators["full"].iter_errors(summary))))
check("summary schema rejects full artifact", bool(list(validators["summary"].iter_errors(full))))
for label, mutate in [
    ("rendered plan", lambda a: a["queries"][0].update(plan="forbidden")),
    ("missing successful access_paths", lambda a: a["queries"][0].pop("access_paths")),
    ("incorrect replay claim", lambda a: a["contract_evaluations"][0]["stability"].update(replayable_from_report=True)),
]:
    bad = json.loads(json.dumps(summary))
    mutate(bad)
    check(f"summary schema rejects {label}", bool(list(validators["summary"].iter_errors(bad))))
result = {"status": "pass", "scope": "artifact/schema and projection checks only; execution attested separately by the runner receipt", "checks": len(checks), "schema_sha256": schema_hashes, "validator": {"jsonschema": importlib.metadata.version("jsonschema"), "PyYAML": importlib.metadata.version("PyYAML")}, "assertions": checks}
print(json.dumps(result, indent=2))
