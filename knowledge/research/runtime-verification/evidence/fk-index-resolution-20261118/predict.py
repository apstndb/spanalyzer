"""Predict backing-index shapes for the primary fixture from its declared DDL."""
from dataclasses import dataclass
import json


@dataclass(frozen=True)
class Table:
    name: str
    columns: tuple
    primary_key: tuple
    nullable: frozenset


def required_keys(child, child_columns, parent, parent_columns, enforced=True):
    """Use resolved column identities in one schema. The fixture uses ASC keys."""
    if (len(child_columns) != len(parent_columns)
            or len(set(parent_columns)) != len(parent_columns)
            or len(set(child_columns)) != len(child_columns)
            or not set(parent_columns) <= set(parent.columns)
            or not set(child_columns) <= set(child.columns)):
        raise ValueError("incomplete or ambiguous FK column mapping")
    pairs = dict(zip(parent_columns, child_columns))
    parent_key = []
    for column in parent.primary_key:
        if column not in pairs:
            break
        parent_key.append(column)
    parent_key.extend(c for c in parent.columns if c in pairs and c not in parent_key)
    child_key = tuple(pairs[c] for c in parent_key)
    filtered = bool(child.nullable.intersection(child_key))
    prefix = child.primary_key[:len(child_key)]
    if (enforced and not filtered and len(child_key) >= 3
            and child_key != child_columns and set(child_key) == set(prefix)):
        raise ValueError("inspect the physical catalog for this PK permutation")
    if enforced and (filtered or child_key != prefix):
        yield child.name, child_key, False, filtered
    if set(parent.primary_key) != set(parent_key):
        yield parent.name, tuple(parent_key), True, bool(parent.nullable.intersection(parent_key))


def initial_shapes():
    def table(name, columns, key, nullable=()):
        return Table(name, columns, key, frozenset(nullable))

    code = ("code",)
    xy = ("x", "y")
    ab = ("a", "b")
    p, q, c = (table(n, ("id", "code"), ("id",), code) for n in ("p", "q", "c"))
    pa, pb = (table(n, ("id", "a", "b"), ("id",), ab) for n in ("pa", "pb"))
    cc = table("cc", ("id", "x", "y"), ("id",), xy)
    covered = table("covered", ("owner", "id"), ("owner", "id"))
    keys = set()
    for child, columns, parent, referenced in [
        (c, code, p, code), (c, code, q, code), (covered, ("owner",), p, ("id",)),
        (cc, xy, pa, ("b", "a")), (cc, xy, pb, ab),
    ]:
        keys.update(required_keys(child, columns, parent, referenced))
    return sorted(keys)


if __name__ == "__main__":
    print(json.dumps(initial_shapes(), indent=2))
