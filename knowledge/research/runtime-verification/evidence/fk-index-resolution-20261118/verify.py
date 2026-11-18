"""Cross-check the retained observations and regenerate inputs without opening a connection."""
from datetime import datetime
import hashlib
import json
from pathlib import Path

from cases import build, composite_controls, physical_hint_controls
from predict import initial_shapes


def digest(obj):
    return hashlib.sha256((json.dumps(obj, indent=2) + "\n").encode()).hexdigest()


def instant(value):
    return datetime.fromisoformat(value.replace("Z", "+00:00"))


def main():
    receipt = json.loads(Path(__file__).with_name("receipt.json").read_text())
    forecast = receipt["forecast"]
    assert digest(forecast) == receipt["forecast_sha256"]
    for dialect in ("google", "pg"):
        for prefix, make in [("", build), ("control-", composite_controls), ("hints-", physical_hint_controls)]:
            assert digest(make(dialect == "pg")) == receipt["input_sha256"][prefix + dialect]

    expected_shapes = [[table, list(keys), unique, filtered] for table, keys, unique, filtered in initial_shapes()]
    for layout in receipt["catalogs"].values():
        assert sorted(layout.values()) == expected_shapes
    assert sorted(receipt["catalogs"]["managed-omni"]) == forecast["ordered_names"]
    assert set(receipt["catalogs"]["emulator"]).isdisjoint(forecast["ordered_names"])

    seen = set()
    for r in receipt["runs"]:
        engine, dialect, suite = r["engine"], r["dialect"], r["suite"]
        assert (engine, dialect, suite) not in seen
        seen.add((engine, dialect, suite))
        assert sum(r["outcomes"].values()) == r["planned"]
        if suite == "controls":
            v = receipt["local_composite_controls"][r["results"]]
            assert v["exported_fk_rev"] == "(x, y) REFERENCES pa(b, a)"
            assert v["cc_keys"]["one-remaining"] == [["x", "y"]] and v["cc_keys"]["none-remaining"] == []
            assert v["shared_parent_kcu_rows_per_key"] == dict.fromkeys(["a", "b"], 2 if engine == "emulator" else 1)
        elif suite == "physical-hints":
            v = receipt["local_physical_name_controls"][r["results"]]
            assert v["managed-index-name"] == v["user-index-name"] == "OK"
            assert v["emulator-index-name"] == "InvalidArgument"
            assert v["constraint-name"] == ("InvalidArgument" if engine == "emulator" else "OK")
        else:
            assert suite == "primary" and r["planned"] == (36 if dialect == "google" else 37)
            v = receipt["primary_profiles"][r["results"]]
            assert all(len(labels) == r["outcomes"][code] for code, labels in v["non_ok"].items())
            assert r["outcomes"].get("NOT_RUN", 0) == (4 if dialect == "pg" else 0)
            if engine == "emulator":
                assert set(forecast["case_targets"]) <= set(v["non_ok"]["InvalidArgument"])
                assert v["fk_rev_pairs"] == [["x", "a", 1], ["y", "b", 2]]
            else:
                for label, name in forecast["case_targets"].items():
                    assert v["plan_scans"][label] == [["IndexScan", name]] and name in receipt["catalogs"][r["catalog"]]
                assert v["fk_rev_pairs"] == [["x", "b", 2], ["y", "a", 1]]
                assert "dropped-alias" in v["non_ok"]["InvalidArgument"]
            if engine == "managed":
                assert instant(forecast["observed_utc"]) < instant(r["lifecycle_utc"]["create-database"])
                assert r["lifecycle"] == {"create-database":"OK", "drop-database":"OK", "database-absent":"NotFound"}
    for dialect in ("google", "pg"):
        for engine in ("managed", "omni-lts", "omni-r5", "emulator"):
            assert (engine, dialect, "primary") in seen
        for engine in ("omni-lts", "emulator"):
            for suite in ("controls", "physical-hints"):
                assert (engine, dialect, suite) in seen
    print(f"{len(seen)} run records verified, {len(expected_shapes)} predicted layouts")


if __name__ == "__main__":
    main()
