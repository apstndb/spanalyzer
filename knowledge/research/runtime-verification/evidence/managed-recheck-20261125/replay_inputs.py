"""Print one of the November 25 read-only inputs as JSON for the retained Go producer."""

import argparse
import json
from pathlib import Path


EVIDENCE = Path(__file__).resolve().parent.parent
PHASES = (
    "readonly", "supplemental", "statistics", "statistics-controls",
    "histogram-controls", "observed-histogram", "scalar-correction", "new-readonly",
    "distribution-frames", "distribution-null-controls",
)
WINDOWS = ("MINUTE", "10MINUTE", "HOUR")


def load(day, name="receipt.json"):
    return json.loads((EVIDENCE / f"managed-recheck-202611{day}" / name).read_text())


def inputs_from_rows(rows, prefix=""):
    return [{"label": prefix + row["label"], "sql": row["sql"]} for row in rows]


def lock_and_transaction_shapes():
    for scope, seconds in (("TOP", "LOCK_WAIT_SECONDS"), ("TOTAL", "TOTAL_LOCK_WAIT_SECONDS")):
        for period in WINDOWS:
            suffix = f"{scope}_{period}"
            table = f"SPANNER_SYS.LOCK_STATS_{suffix}"
            distribution = "LOCK_WAIT_DISTRIBUTION"
            element = f"{distribution}[SAFE_OFFSET(0)]"
            percentile = "SPANNER_SYS.DISTRIBUTION_PERCENTILE"
            cases = (
                ("lock-shape", f"SELECT {distribution}, {distribution}_JSON_STRING, {seconds} FROM {table} LIMIT 0"),
                ("lock-percentile", f"SELECT {element}.count AS n, {percentile}({element},50.0) AS struct_p50, {percentile}({distribution}_JSON_STRING,50.0) AS json_p50 FROM {table} WHERE ARRAY_LENGTH({distribution})>0 LIMIT 1"),
                ("lock-whole-array", f"SELECT {percentile}({distribution},50.0) AS p50 FROM {table} LIMIT 0"),
                ("txn-shape", f"SELECT REPEATABLE_READ_PESSIMISTIC_TXN_COUNT,SERIALIZABLE_OPTIMISTIC_TXN_COUNT FROM SPANNER_SYS.TXN_STATS_{suffix} LIMIT 0"),
            )
            for label, sql in cases:
                yield {"label": f"nov20/{label}-{suffix}", "sql": sql}


def statistics_inputs():
    for family in ("TXN_STATS_TOP", "TXN_STATS_TOTAL", "LOCK_STATS_TOP", "LOCK_STATS_TOTAL"):
        hist = "lock_latency_distribution" if family.startswith("TXN") else "lock_wait_distribution"
        element = f"{hist}[SAFE_OFFSET(0)]"
        fields = (
            f"ARRAY_LENGTH({hist}) AS cardinality", f"{element}.count AS event_count",
            f"{element}.mean AS mean", f"{element}.sum_of_squared_deviation AS ssd",
            f"{element}.num_finite_buckets AS finite", f"{element}.growth_factor AS growth",
            f"{element}.scale AS scale", f"ARRAY_LENGTH({element}.bucket_counts) AS bins",
        )
        for period in WINDOWS:
            suffix = f"{family}_{period}"
            table = f"spanner_sys.{suffix.lower()}"
            encoded = f"{hist}_json_string"
            cases = (
                ("json", f"SELECT {encoded} AS hist FROM {table} LIMIT 1"),
                ("element", f"SELECT {','.join(fields)} FROM {table} LIMIT 1"),
                ("percentile", f"SELECT spanner_sys.distribution_percentile({encoded},50.0) AS p50 FROM {table} WHERE {encoded} IS NOT NULL LIMIT 1"),
            )
            for label, sql in cases:
                yield {"label": f"google-stats/{label}/{suffix}", "sql": sql}


def build(batch):
    previous = load(24)["managed"]
    placements = inputs_from_rows(previous["placements"]["queries"])
    if batch == "controls":
        return inputs_from_rows(previous["distribution_controls"]) + placements[-1:]

    cases = []
    for phase in PHASES:
        for case in load(12, f"{phase}-cases.json"):
            cases.append({**case, "label": f"{phase}/{case['label']}"})
    cases.extend(inputs_from_rows(load(18)["focused_cases"]))
    tokens = load(18, "mirror-followup.json")["spanalyzer_comparison"]["paired_cases"]
    cases.extend(inputs_from_rows(tokens, "token/"))
    cases.extend(lock_and_transaction_shapes())
    cases.extend(statistics_inputs())
    return cases + placements[:-1]


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("batch", choices=("primary", "controls"))
    print(json.dumps(build(parser.parse_args().batch), indent=2))
