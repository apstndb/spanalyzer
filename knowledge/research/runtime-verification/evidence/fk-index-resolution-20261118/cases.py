"""Build equivalent FK catalog and read-only query-plan probes in both dialects."""
import json
import argparse


def build(pg):
    steps = []

    def add(label, kind, *, requires=(), sql=None, statements=None):
        row = {"label": label, "kind": kind, "requires": list(requires)}
        if sql is not None:
            row["sql"] = sql
        if statements is not None:
            row["statements"] = statements
        steps.append(row)

    integer = "bigint" if pg else "INT64"
    string = "varchar(10)" if pg else "STRING(10)"
    schema = "public" if pg else ""

    def table(name, columns, primary):
        pk = "PRIMARY KEY (" + primary + ")"
        body = ", ".join(f"{column} {column_type}" for column, column_type in columns)
        if pg:
            return f"CREATE TABLE {name} ({body}, {pk})"
        return f"CREATE TABLE {name} ({body}) {pk}"

    identifier = ("id", integer + " NOT NULL")
    code = ("code", string)
    physical = """
SELECT i.table_name, i.index_name, i.index_type, i.is_unique, i.is_null_filtered,
       i.spanner_is_managed, k.column_name, k.ordinal_position, k.column_ordering
FROM information_schema.indexes i
JOIN information_schema.index_columns k
  ON k.table_catalog=i.table_catalog AND k.table_schema=i.table_schema
 AND k.table_name=i.table_name AND k.index_name=i.index_name AND k.index_type=i.index_type
WHERE i.table_schema='{schema}' AND k.ordinal_position IS NOT NULL
ORDER BY i.table_name, i.index_name, k.ordinal_position""".strip().format(schema=schema)
    reference = """
SELECT r.constraint_name, r.unique_constraint_name, ft.table_name AS child_table,
       ft.enforced, f.column_name AS child_column, f.ordinal_position AS child_position,
       f.position_in_unique_constraint AS parent_position, u.table_name AS parent_table,
       u.column_name AS parent_column
FROM information_schema.referential_constraints r
JOIN information_schema.table_constraints ft
  ON ft.constraint_catalog=r.constraint_catalog AND ft.constraint_schema=r.constraint_schema
 AND ft.constraint_name=r.constraint_name
JOIN information_schema.key_column_usage f
  ON f.constraint_catalog=r.constraint_catalog AND f.constraint_schema=r.constraint_schema
 AND f.constraint_name=r.constraint_name
LEFT JOIN information_schema.key_column_usage u
  ON u.constraint_catalog=r.unique_constraint_catalog AND u.constraint_schema=r.unique_constraint_schema
 AND u.constraint_name=r.unique_constraint_name AND u.ordinal_position=f.position_in_unique_constraint
WHERE r.constraint_schema='{schema}'
ORDER BY r.constraint_name, f.ordinal_position""".strip().format(schema=schema)
    present = """
SELECT table_name, index_name, spanner_is_managed FROM information_schema.indexes
WHERE table_schema='{schema}' AND table_name IN ('p','q','c')
ORDER BY table_name,index_name""".strip().format(schema=schema)

    def foreign(label, child, child_columns, parent, parent_columns, *deps, informational=False):
        sql = (f"ALTER TABLE {child} ADD CONSTRAINT {label} FOREIGN KEY ({child_columns}) "
               f"REFERENCES {parent} ({parent_columns})")
        if informational:
            sql += " NOT ENFORCED"
        add(label, "ddl", sql=sql, requires=deps)

    def probe(label, tbl, hint, columns, condition, *deps, mode="plan"):
        table_hint = f" /*@ FORCE_INDEX={hint} */" if pg else "@{FORCE_INDEX=" + hint + "}"
        add(label, mode, sql=f"SELECT {columns} FROM {tbl}{table_hint} WHERE {condition}", requires=deps)

    add("tables", "ddl", statements=[
        table(t, [identifier, code], "id") for t in ("p","q","c")
    ] + [
        table("covered", [("owner", integer+" NOT NULL"), identifier], "owner, id"),
        table("pa", [identifier, ("a", integer), ("b", integer)], "id"),
        table("pb", [identifier, ("a", integer), ("b", integer)], "id"),
        table("cc", [identifier, ("x", integer), ("y", integer)], "id"),
    ])
    add("user-indexes","ddl",statements=[
        "CREATE INDEX ix_c_code ON c(code)", "CREATE UNIQUE INDEX ix_p_code ON p(code)"
    ],requires=["tables"])
    foreign("fk_p","c","code","p","code","tables")
    add("after-p","query",sql=present,requires=["fk_p"])
    foreign("fk_q","c","code","q","code","tables")
    add("after-q","query",sql=present,requires=["fk_p","fk_q"])
    foreign("fk_covered","covered","owner","p","id","tables")
    foreign("fk_rev","cc","x, y","pa","b, a","tables")
    foreign("fk_fwd","cc","x, y","pb","a, b","tables")
    add("layout","query",sql=physical,requires=["tables"])
    add("reference","query",sql=reference,requires=["tables"])
    if pg:
        add("pg-constraint","query",sql="SELECT conname, conindid FROM pg_catalog.pg_constraint WHERE conname IN ('fk_p','fk_q','fk_covered','fk_rev','fk_fwd') ORDER BY conname",requires=["tables"])
    probe("child-p","c","fk_p","code","code='k'","fk_p")
    probe("child-q","c","fk_q","code","code='k'","fk_q")
    probe("parent-p","p","fk_p","code","code='k'","fk_p")
    probe("wrong-table","q","fk_p","code","code='k'","fk_p","fk_q")
    probe("unknown","c","fk_missing","code","code='k'","tables")
    probe("explicit-user-index","c","ix_c_code","code","code='k'","user-indexes")
    probe("base","c","_BASE_TABLE","code","code='k'","tables")
    probe("nullable-without-filter","c","fk_p","code","id=1","fk_p")
    probe("covered-child","covered","fk_covered","owner","owner=1","fk_covered")
    probe("covered-parent","p","fk_covered","id","id=1","fk_covered")
    probe("reverse","cc","fk_rev","x,y","x=11 AND y=22","fk_rev")
    probe("forward","cc","fk_fwd","x,y","x=11 AND y=22","fk_fwd")
    probe("normal-child","c","fk_p","code","code='k'","fk_p",mode="query")
    probe("normal-unknown","c","fk_missing","code","code='k'","tables",mode="query")
    add("drop-p","ddl",sql="ALTER TABLE c DROP CONSTRAINT fk_p",requires=["fk_p","fk_q"])
    add("after-drop-p","query",sql=present,requires=["drop-p"])
    probe("dropped-alias","c","fk_p","code","code='k'","drop-p")
    probe("survivor","c","fk_q","code","code='k'","drop-p","fk_q")
    add("drop-q","ddl",sql="ALTER TABLE c DROP CONSTRAINT fk_q",requires=["drop-p","fk_q"])
    add("after-drop-q","query",sql=present,requires=["drop-q"])
    foreign("fk_info","c","code","p","code","drop-q",informational=True)
    add("informational-layout","query",sql=present,requires=["fk_info"])
    add("informational-reference","query",sql=reference,requires=["fk_info"])
    probe("informational-child","c","fk_info","code","code='k'","fk_info")
    probe("informational-parent","p","fk_info","code","code='k'","fk_info")
    return {"dialect":"pg" if pg else "google","steps":steps}


def composite_controls(pg):
    primary = build(pg)
    cases = {row["label"]: row for row in primary["steps"]}
    tables = cases["tables"]
    tables["statements"] = [
        sql for sql in tables["statements"]
        if sql.startswith(("CREATE TABLE pa ", "CREATE TABLE pb ", "CREATE TABLE cc "))
    ]
    integer = "bigint" if pg else "INT64"
    for definition, key in [
        (f"CREATE TABLE cp (id {integer} NOT NULL, v {integer}, w {integer}", "id"),
        (f"CREATE TABLE pr (a {integer} NOT NULL, b {integer} NOT NULL", "a, b"),
    ]:
        suffix = f", PRIMARY KEY ({key}))" if pg else f") PRIMARY KEY ({key})"
        tables["statements"].append(definition + suffix)
    steps = [tables, cases["fk_rev"], cases["fk_fwd"]]

    def add(label, kind, sql, *dependencies):
        steps.append({"label": label, "kind": kind, "sql": sql, "requires": list(dependencies)})

    add("fk_copy", "ddl", "ALTER TABLE cp ADD CONSTRAINT fk_copy FOREIGN KEY (v,w) REFERENCES pa(a,b)", "tables")
    add("fk_pk", "ddl", "ALTER TABLE cp ADD CONSTRAINT fk_pk FOREIGN KEY (v,w) REFERENCES pr(b,a)", "tables")
    add("export", "ddl-export", "", "fk_rev", "fk_fwd", "fk_copy", "fk_pk")
    layout = cases["layout"]["sql"]
    add("all-indexes", "query", layout, "tables")
    add("pairs", "query", cases["reference"]["sql"], "tables")
    parent_usage = (
        "SELECT constraint_name,column_name,ordinal_position,COUNT(*) AS n "
        "FROM information_schema.key_column_usage WHERE table_name='pa' "
        "GROUP BY constraint_name,column_name,ordinal_position ORDER BY constraint_name,ordinal_position"
    )
    add("parent-kcu", "query", parent_usage, "fk_copy", "fk_rev")
    add("drop-reverse", "ddl", "ALTER TABLE cc DROP CONSTRAINT fk_rev", "fk_rev", "fk_fwd")
    add("one-remaining", "query", layout, "drop-reverse")
    add("drop-forward", "ddl", "ALTER TABLE cc DROP CONSTRAINT fk_fwd", "drop-reverse")
    add("none-remaining", "query", layout, "drop-forward")
    return {"dialect": primary["dialect"], "steps": steps}


def physical_hint_controls(pg):
    primary = build(pg)
    cases = {row["label"]: row for row in primary["steps"]}
    tables = cases["tables"]
    tables["statements"] = [sql for sql in tables["statements"] if sql.startswith(("CREATE TABLE c ", "CREATE TABLE p "))]
    steps = [tables, cases["user-indexes"], cases["fk_p"], cases["layout"]]
    for label, name in [
        ("constraint-name", "fk_p"),
        ("managed-index-name", "IDX_c_code_N_E985B7CD5F04A313"),
        ("emulator-index-name", "IDX_c_code_N_D99AE4F461677C4A"),
        ("user-index-name", "ix_c_code"),
    ]:
        hint = f' /*@ FORCE_INDEX="{name}" */' if pg else "@{FORCE_INDEX=`" + name + "`}"
        steps.append({"label": label, "kind": "query", "requires": ["fk_p", "user-indexes"],
                      "sql": f"SELECT code FROM c{hint} WHERE code='k'"})
    return {"dialect": primary["dialect"], "steps": steps}


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Print the ordered FK probe input. No connection is opened.")
    parser.add_argument("dialect", choices=("google", "pg"))
    group = parser.add_mutually_exclusive_group()
    group.add_argument("--controls", action="store_true", help="print the separate 13-step local fixture")
    group.add_argument("--physical-hints", action="store_true", help="compare physical names and the FK name")
    args = parser.parse_args()
    make_cases = composite_controls if args.controls else build
    if args.physical_hints:
        make_cases = physical_hint_controls
    print(json.dumps(make_cases(args.dialect == "pg"), indent=2))
