"""Verify retained names and compact observations offline; perform no RPCs."""
import hashlib
import json
from pathlib import Path

import farmhash

HERE = Path(__file__).resolve().parent
EARLIER = HERE.parent / "fk-backing-index-20261112"


def extract(path):
    groups = {}
    for line in path.read_text().splitlines():
        event = json.loads(line)
        for row in event.get("rows", []):
            if not row.get("SPANNER_IS_MANAGED"):
                continue
            key = row["TABLE_SCHEMA"], row["INDEX_NAME"]
            group = groups.setdefault(key, {"row": row, "keys": {}})
            group["keys"][int(row["ORDINAL_POSITION"])] = row["COLUMN_NAME"]
    result = {}
    for group in groups.values():
        row = group["row"]
        code = "U" if row["IS_UNIQUE"] else "N" if row["IS_NULL_FILTERED"] else ""
        base = "IDX_" + row["TABLE_NAME"] + "_" + "_".join(group["keys"][n] for n in sorted(group["keys"]))
        candidate = (row["TABLE_SCHEMA"] + "." if row["TABLE_SCHEMA"] else "") + base + ("_" + code if code else "")
        result[candidate] = row["INDEX_NAME"]
    return result


def main():
    observed = json.loads((HERE / "observations.json").read_text())
    for path, expected in [(EARLIER / "managed.jsonl", observed["retained_managed_source"]["sha256"]), (HERE / "controls.json", observed["controls_sha256"]), (HERE / "check_prefix.py", observed["symbolic_check"]["script_sha256"]), (HERE / "prefix-result.json", observed["symbolic_check"]["result_sha256"])]:
        assert hashlib.sha256(path.read_bytes()).hexdigest() == expected, path
    managed = extract(EARLIER / "managed.jsonl")
    assert len(managed) == 18
    assert managed == extract(EARLIER / "omni-lts-fixed.jsonl") == extract(EARLIER / "omni-r5-fixed.jsonl")
    emulator = extract(EARLIER / "emulator.jsonl")
    assert managed.keys() == emulator.keys()
    for signature, name in emulator.items():
        assert f"{farmhash.Fingerprint64(('2695C62E33338E4B_' + signature).encode()):016X}" == name.rsplit("_", 1)[1]
    assert sum(r["databases"] * r["steps_per_database"] for r in observed["runs"]) == 160
    assert all(r["process_exit_code"] == 0 and r["runtime_closed"] and r["owned_container_absent"] for r in observed["runs"])
    assert observed["runs"][0]["fixtures"][0]["normalized_result_sha256"] == observed["runs"][0]["fixtures"][1]["normalized_result_sha256"]
    cases = observed["lts_reference_results"]
    assert {k: r["code"] for k, r in cases.items() if r["code"] != "OK"} == {"ambiguous/a": "FailedPrecondition", "schema/hint/unqualified": "InvalidArgument", "schema/hint/path": "InvalidArgument"}
    for left, right in [("relation/first", "relation/recreate"), ("type/int", "type/string")]:
        assert cases[left + "/indexes"]["rows"] == cases[right + "/indexes"]["rows"]
    asc, desc = (cases["direction/" + direction + "/indexes"]["rows"][0] for direction in ["asc", "desc"])
    assert asc["INDEX_NAME"] == desc["INDEX_NAME"] and (asc["COLUMN_ORDERING"], desc["COLUMN_ORDERING"]) == ("ASC", "DESC")
    long_names = cases["long/two-columns/indexes"]["rows"]
    # The limit is 128, but removing a trailing separator from the truncated
    # base leaves these concrete names one byte shorter.
    assert len(long_names) == 2 and all(len(r["INDEX_NAME"]) == 127 for r in long_names)
    assert long_names[0]["INDEX_NAME"] != long_names[1]["INDEX_NAME"]
    assert long_names[0]["INDEX_NAME"].rsplit("_", 1)[0] == long_names[1]["INDEX_NAME"].rsplit("_", 1)[0]
    reverse = observed["reverse_results"]
    assert reverse["reverse/a_b"]["code"] == "FailedPrecondition"
    forward = cases["ambiguous/a_b/indexes"]["rows"][0]
    backward = reverse["reverse/A/indexes"]["rows"][0]
    assert forward["TABLE_NAME"] == "A_B" and backward["TABLE_NAME"] == "A"
    assert forward["INDEX_NAME"] == backward["INDEX_NAME"] == "IDX_A_B_C_N_99E878FAF8057D7D"
    assert cases["schema/hint/quoted"]["code"] == "OK"
    assert observed["r5_additional_hints"]["schema/hint/physical-qualified"]["code"] == "OK"
    assert observed["r5_additional_hints"]["schema/hint/physical-unqualified"]["code"] == "InvalidArgument"
    proof = json.loads((HERE / "prefix-result.json").read_text())
    assert proof["native_controls"] == 300 and all(r["status"] == "unsat" for r in proof["results"])
    assert [len(r["pairs"]) for r in proof["results"]] == [2, 3]
    practical = json.loads((HERE / "practical-observations.json").read_text())
    assert practical["process_exit_code"] == 0 and practical["cleanup"]["runtime_closed"] and practical["cleanup"]["owned_container_absent"]
    assert len(practical["fixtures"]) == 3
    for fixture, table in zip(practical["fixtures"][:2], ["SALES_ORDER_SHIPPING", "SALES_ORDER"]):
        results = fixture["results"]
        assert len(results) == 5 and results["child-1"]["code"] == "OK" and results["child-2"]["code"] == "FailedPrecondition"
        row = results["indexes-after-1"]["rows"][0]
        assert row["TABLE_NAME"] == table and row["INDEX_NAME"] == "IDX_SALES_ORDER_SHIPPING_ADDRESS_ID_C53AEFD47826CE09"
        assert row["IS_NULL_FILTERED"] is False and row["IS_UNIQUE"] is False
    different = practical["fixtures"][2]["results"]
    assert different["child-2"]["code"] == "OK"
    assert {r["IS_NULL_FILTERED"] for r in different["indexes-after-2"]["rows"]} == {False, True}
    assert len({r["INDEX_NAME"] for r in different["indexes-after-2"]["rows"]}) == 2
    lowercase = json.loads((HERE / "lowercase-observations.json").read_text())
    assert lowercase["process_exit_code"] == 0 and lowercase["cleanup"]["runtime_closed"] and lowercase["cleanup"]["owned_container_absent"]
    assert len(lowercase["fixtures"]) == 2
    for fixture, table in zip(lowercase["fixtures"], ["sales_order_shipping", "sales_order"]):
        results = fixture["results"]
        assert len(results) == 5 and results["child-1"]["code"] == "OK" and results["child-2"]["code"] == "FailedPrecondition"
        row = results["indexes-after-1"]["rows"][0]
        assert row["TABLE_NAME"] == table and row["INDEX_NAME"] == "IDX_sales_order_shipping_address_id_90E532EDA843ECD5"
    managed_followup = json.loads((HERE / "managed-observations.json").read_text())
    assert managed_followup["database_count"] == 1 and managed_followup["plan_steps"] == 11
    assert managed_followup["process_exit_code"] == 0 and managed_followup["cleanup_verified"]
    assert hashlib.sha256((HERE / "managed-controls.json").read_bytes()).hexdigest() == managed_followup["plan_sha256"]
    assert hashlib.sha256((HERE / "managed.patch").read_bytes()).hexdigest() == "a7d2e56e5076930ee30a9de9f21c6cece2c04a2c7dfb8a46ced2f82a1cef7fc4"
    assert managed_followup["producer_source_sha256"] == "721995b9afac317b12f662622fbe2accaa2d581869724562d32d25d19727de43"
    lifecycle = managed_followup["lifecycle"]
    assert {label: row["code"] for label, row in lifecycle.items()} == {"create-database": "OK", "drop-database": "OK", "database-absent": "NotFound"}
    results = managed_followup["results"]
    successful = {"address", "shipping-first", "indexes-shipping", "indexes-first-collision", "drop-shipping", "order-alone", "indexes-order", "indexes-second-collision"}
    assert {label for label, row in results.items() if row["code"] == "OK"} == successful
    for label in ["order-collision", "shipping-collision"]:
        assert results[label]["code"] == "FailedPrecondition"
        assert results[label]["error"] == "rpc error: code = FailedPrecondition desc ="
    # Preserve the optional final-inventory fixture mistake explicitly; it is
    # neither a successful metadata check nor evidence of a service defect.
    assert len(results) == 11 and results["tables-final"]["code"] == "InvalidArgument"
    assert "Unrecognized name: TABLE_STATE" in results["tables-final"]["error"]
    for first, after, fixture in [("indexes-shipping", "indexes-first-collision", lowercase["fixtures"][0]), ("indexes-order", "indexes-second-collision", lowercase["fixtures"][1])]:
        rows = results[first]["rows"]
        assert rows == results[after]["rows"] == fixture["results"]["indexes-after-1"]["rows"]
        assert len(rows) == 1 and rows[0]["SPANNER_IS_MANAGED"] is True
        assert rows[0]["IS_UNIQUE"] is False and rows[0]["IS_NULL_FILTERED"] is False
    matrix = json.loads((HERE / "case-matrix-observations.json").read_text())
    assert hashlib.sha256((HERE / "case-matrix-controls.json").read_bytes()).hexdigest() == matrix["plan_sha256"]
    assert hashlib.sha256((HERE / "case-matrix-managed.patch").read_bytes()).hexdigest() == "59cb4917e9f8c7fbe5ebf17592b2eed7a77622b5fff8bb966915ac26fe6995a3"
    runs = {run["backend"]: run for run in matrix["runs"]}
    assert set(runs) == {"managed", "omni", "emulator"}
    assert matrix["planned_steps_total"] == 96 and matrix["executed_steps_total"] == 88
    totals = {"OK": 0, "FailedPrecondition": 0, "NOT_RUN": 0}
    for backend, run in runs.items():
        assert run["process_exit_code"] == 0 and run["cleanup_verified"] and run["planned_steps"] == 32
        if backend == "managed":
            assert run["cleanup"] == {"create-database": "OK", "drop-database": "OK", "database-absent": "NotFound"}
        else:
            assert run["cleanup"]["runtime-closed"] == "OK" and run["cleanup"]["owned_container_absent"]
            assert run["runtime"]["ObservedByThisProcess"] and run["runtime"]["Platform"] == "linux/arm64"
        for case, fixture in run["cases"].items():
            assert case in {"upper", "lower"} and len(fixture["step_codes"]) == 16
            expected = "OK" if backend == "emulator" else "FailedPrecondition"
            assert fixture["step_codes"]["order-second"] == fixture["step_codes"]["shipping-second"] == expected
            assert fixture["step_codes"]["shipping-first"] == fixture["step_codes"]["order-first"] == "OK"
            assert fixture["after_case_tables_empty"] and fixture["after_case_indexes_empty"]
            assert fixture["after_second_create_index_rows_unchanged"] == {"forward": True, "reverse": True}
            left = fixture["index_snapshots"]["indexes-shipping"][0]
            right = fixture["index_snapshots"]["indexes-order"][0]
            assert left["TABLE_NAME"] != right["TABLE_NAME"] and left["INDEX_NAME"] == right["INDEX_NAME"]
            assert all(row["SPANNER_IS_MANAGED"] and not row["IS_UNIQUE"] and not row["IS_NULL_FILTERED"] for row in [left, right])
            if backend == "emulator":
                signature = "IDX_SALES_ORDER_SHIPPING_ADDRESS_ID" if case == "upper" else "IDX_sales_order_shipping_address_id"
                assert left["INDEX_NAME"] == signature + f"_{farmhash.Fingerprint64(('2695C62E33338E4B_' + signature).encode()):016X}"
                assert not fixture["ddl_errors"]
            else:
                assert fixture["ddl_errors"] == {"order-second": "rpc error: code = FailedPrecondition desc =", "shipping-second": "rpc error: code = FailedPrecondition desc ="}
            for code in fixture["step_codes"].values():
                totals[code] += 1
    assert totals == {"OK": 80, "FailedPrecondition": 8, "NOT_RUN": 8}
    for case in ["upper", "lower"]:
        assert runs["managed"]["cases"][case]["index_snapshots"] == runs["omni"]["cases"][case]["index_snapshots"]
    print("Retained names, prefix-family results, earlier naming probes and all six case/engine cells (96 planned, 88 executed), with verified cleanup: OK")


if __name__ == "__main__":
    main()
