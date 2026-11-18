"""Prepare isolated FK naming controls; perform no RPCs or cloud writes."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess

HERE = Path(__file__).resolve().parent


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("output", type=Path)
    parser.add_argument("--variant", choices=["full", "original", "reverse", "qualified", "practical-forward", "practical-reverse", "practical-different-nullability", "practical-lower-forward", "practical-lower-reverse", "managed-lower", "case-matrix"], default="full")
    args = parser.parse_args()
    # The existing reproducer reconstructs immutable producer bodies, checks
    # their hashes, and exclusively creates files in a fresh scratch directory.
    subprocess.run(["python3", str(HERE.parent / "fk-backing-index-20261112/prepare.py"), str(args.output)], check=True)
    plan = json.loads((HERE / "controls.json").read_text())
    if args.variant in {"managed-lower", "case-matrix"}:
        source = args.output / "managed.go"
        assert hashlib.sha256(source.read_bytes()).hexdigest() == "7e0330cbf8b02ab65849fbbf3411d33711c25a9091c9972fe8328f26227ee74c"
        patch = HERE / "managed.patch"
        assert hashlib.sha256(patch.read_bytes()).hexdigest() == "a7d2e56e5076930ee30a9de9f21c6cece2c04a2c7dfb8a46ced2f82a1cef7fc4"
        subprocess.run(["patch", "-p1", "--batch", "-i", str(patch)], cwd=args.output, check=True)
        assert hashlib.sha256(source.read_bytes()).hexdigest() == "721995b9afac317b12f662622fbe2accaa2d581869724562d32d25d19727de43"
        if args.variant == "case-matrix":
            patch = HERE / "case-matrix-managed.patch"
            assert hashlib.sha256(patch.read_bytes()).hexdigest() == "59cb4917e9f8c7fbe5ebf17592b2eed7a77622b5fff8bb966915ac26fe6995a3"
            subprocess.run(["patch", "-p1", "--batch", "-i", str(patch)], cwd=args.output, check=True)
            assert hashlib.sha256(source.read_bytes()).hexdigest() == "7c59d381100b86e21e6b729d9c7f7b1f0e5ce969accb5d54190dd17bf119abad"
            plan = json.loads((HERE / "case-matrix-controls.json").read_text())
        else:
            plan = json.loads((HERE / "managed-controls.json").read_text())
    elif args.variant.startswith("practical-"):
        shipping = "CREATE TABLE SALES_ORDER_SHIPPING(SHIPMENT_ID INT64 NOT NULL,ADDRESS_ID INT64 NOT NULL,CONSTRAINT SHIPPING_ADDRESS_FK FOREIGN KEY(ADDRESS_ID) REFERENCES ADDRESS(ADDRESS_ID)) PRIMARY KEY(SHIPMENT_ID)"
        order = "CREATE TABLE SALES_ORDER(ORDER_ID INT64 NOT NULL,SHIPPING_ADDRESS_ID INT64 NOT NULL,CONSTRAINT ORDER_ADDRESS_FK FOREIGN KEY(SHIPPING_ADDRESS_ID) REFERENCES ADDRESS(ADDRESS_ID)) PRIMARY KEY(ORDER_ID)"
        if args.variant == "practical-different-nullability":
            order = order.replace("SHIPPING_ADDRESS_ID INT64 NOT NULL", "SHIPPING_ADDRESS_ID INT64")
        statements = [order, shipping] if args.variant.endswith("-reverse") else [shipping, order]
        query = "SELECT i.TABLE_SCHEMA,i.TABLE_NAME,i.INDEX_NAME,i.IS_UNIQUE,i.IS_NULL_FILTERED,i.SPANNER_IS_MANAGED,c.COLUMN_NAME,c.COLUMN_ORDERING FROM INFORMATION_SCHEMA.INDEXES i JOIN INFORMATION_SCHEMA.INDEX_COLUMNS c USING(TABLE_CATALOG,TABLE_SCHEMA,TABLE_NAME,INDEX_NAME) WHERE i.INDEX_TYPE='INDEX' ORDER BY i.TABLE_NAME,i.INDEX_NAME,c.ORDINAL_POSITION"
        common = {"family": "practical-upper-snake-case", "requires": [], "timeout_seconds": 90}
        steps = [dict(common, label="address", kind="ddl", statements=["CREATE TABLE ADDRESS(ADDRESS_ID INT64 NOT NULL) PRIMARY KEY(ADDRESS_ID)"])]
        for number, statement in enumerate(statements, 1):
            steps.append(dict(common, label="child-" + str(number), kind="ddl", statements=[statement]))
            steps.append(dict(common, label="indexes-after-" + str(number), kind="query", sql=query))
        plan = {"dialect": "google", "steps": steps}
        if args.variant.startswith("practical-lower-"):
            # Lowercase only the fixture DDL. Metadata column names and the
            # INDEX_TYPE='INDEX' string literal must keep their original form.
            for step in steps:
                step["family"] = "practical-lower-snake-case"
                if "statements" in step:
                    step["statements"] = [sql.lower() for sql in step["statements"]]
    elif args.variant == "original":
        plan["steps"] = [s for s in plan["steps"] if not s["label"].startswith("schema/hint/physical-")]
    elif args.variant == "qualified":
        plan["steps"] = [s for s in plan["steps"] if s["label"] == "parents" or s["label"].startswith("schema/")]
    elif args.variant == "reverse":
        steps = [s for s in plan["steps"] if s["label"] == "parents"]
        for table, column, constraint in [("A", "B_C", "ExplicitA"), ("A_B", "C", "ExplicitAB")]:
            label = "reverse/" + table.lower()
            common = {"family": "name-collision", "requires": ["parents"], "timeout_seconds": 90}
            steps.append(dict(common, label=label, kind="ddl", statements=[f"CREATE TABLE {table}(Id INT64 NOT NULL,{column} INT64,CONSTRAINT {constraint} FOREIGN KEY({column}) REFERENCES P(K)) PRIMARY KEY(Id)"]))
            steps.append(dict(common, label="reverse/" + table + "/indexes", kind="query", sql="SELECT TABLE_SCHEMA,TABLE_NAME,INDEX_NAME,SPANNER_IS_MANAGED FROM INFORMATION_SCHEMA.INDEXES WHERE INDEX_TYPE='INDEX' AND TABLE_NAME IN ('A','A_B') ORDER BY TABLE_NAME,INDEX_NAME"))
        plan["steps"] = steps
    data = (json.dumps(plan, indent=2) + "\n").encode()
    (args.output / "plan.json").write_bytes(data)
    print(json.dumps({"variant": args.variant, "steps": len(plan["steps"]), "plan_sha256": hashlib.sha256(data).hexdigest(), "database_actions": 0}))


if __name__ == "__main__":
    main()
