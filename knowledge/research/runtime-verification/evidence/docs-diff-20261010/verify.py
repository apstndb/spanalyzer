"""Offline checks of retained observations; this does not contact Spanner."""
from pathlib import Path
import collections
import hashlib
import json
import re

base = Path(__file__).resolve().parent

def records(name):
    return [json.loads(s) for s in (base / (name + '.jsonl')).read_text().splitlines()]

def labeled(name):
    rows = records(name)
    assert len({r['label'] for r in rows}) == len(rows), name
    return {r['label']: r for r in rows}

q = labeled('queries') | labeled('followup')
assert len(q) == 125
assert collections.Counter(r['code'] for r in q.values()) == {
    'OK': 99, 'OutOfRange': 13, 'InvalidArgument': 12, 'Unimplemented': 1,
}
for label in ['json-equality', 'json-number-order', 'json-type-order',
              'object-key-order', 'object-keys-before-values', 'object-prefix',
              'array-prefix', 'array-numeric', 'json-null-equal',
              'cast-null-is-sql-null', 'in-json-first', 'between-json-first',
              'array-order-coerce', 'not-equal-coercion', 'in-mixed-list']:
    assert q[label]['code'] == 'OK' and q[label]['rows'] == [{'result': True}], label
for label in q:
    if label.startswith('coerce-'):
        assert q[label]['code'] == 'OK' and q[label]['rows'] == [{'result': True}], label
assert q['json-string-vs-object-text']['rows'] == [{'result': False}]
assert q['sql-null-equal']['rows'] == [{'result': None}]
assert q['json-null-is-sql-null']['rows'] == [{'result': False}]
assert q['cast-sql-null']['rows'] == [{'result': None}]
assert q['to-json-control-sql-null']['rows'] == [{'result': 'null'}]
for label, expected in {
    'cast-string': 'foo', 'cast-string-json-looking': '{"a":1}',
    'cast-bytes': 'YQ==', 'cast-interval': 'P1D',
    'cast-array': [1, None, 3], 'cast-struct': {'a': 1, 'b': None},
    'cast-wide-int': 9223372036854775807,
}.items():
    assert q[label]['code'] == 'OK'
    assert json.loads(q[label]['rows'][0]['result']) == expected, label
for label in ['cast-positive-infinity', 'cast-negative-infinity', 'cast-nan',
              'cast-nested-infinity', 'cast-wide-numeric']:
    assert q[label]['code'] == 'OutOfRange', label
for label in ['safe-cast-positive-infinity', 'safe-cast-wide-numeric',
              'safe-cast-nested-infinity']:
    assert q[label]['rows'] == [{'result': None}], label
assert [r['j'] for r in q['json-sort']['rows']] == [
    None, 'null', '"a"', '"b"', '2', '10', 'false', 'true', '[]', '{}',
]
assert q['json-group']['rows'] == [
    {'j': None, 'n': '1'}, {'j': 'null', 'n': '2'},
    {'j': '1', 'n': '2'}, {'j': '{"a":1,"b":2}', 'n': '2'},
]
for prefix in ['coalesce', 'ifnull', 'if', 'case-simple', 'case-searched',
               'coalesce-subquery']:
    assert q[prefix + '-unused']['rows'] == [{'result': 10}], prefix
    assert q[prefix + '-selected']['code'] == 'OutOfRange', prefix
assert q['coalesce-name-resolution']['code'] == 'InvalidArgument'
assert 'Syntax error: Unexpected keyword IF' in q['procedural-if-suggestion']['error']

g = labeled('google-features')
p = labeled('pg-features')
for dialect in [g, p]:
    for value in [100, 999, 1000, 300000]:
        r = dialect['heartbeat-boundary-' + str(value)]
        assert r['code'] == 'OK' and r['rows'], value
    for value in [99, 300001]:
        r = dialect['heartbeat-boundary-' + str(value)]
        assert r['code'] == 'InvalidArgument' and 'min 100 and max 300000' in r['error']
    assert dialect['read-options-omitted']['code'] == 'OK'
    for label in ['read-options-empty', 'read-options-nonnull']:
        assert dialect[label]['code'] == 'InvalidArgument'
        assert 'read_options must be null' in dialect[label]['error']
    assert dialect['heartbeat-child-100']['code'] == 'OK'
    assert dialect['drop-database']['code'] == 'OK'
    assert dialect['database-absent']['code'] == 'NotFound'
summary = json.loads((base / 'heartbeat-summary.json').read_text())
assert summary['google-features']['heartbeat_count'] == 18
assert summary['pg-features']['heartbeat_count'] == 17
assert all(90 <= r['adjacent_gap_ms']['median'] <= 110 for r in summary.values())
assert [r['Id'] for r in g['stored-json-sort']['rows']] == ['2', '4', '1', '3']
assert [r['n'] for r in g['stored-json-group']['rows']] == ['2', '2']
assert g['cast-enum']['rows'] == [{'result': '"ONE"'}]
assert json.loads(g['cast-proto']['rows'][0]['result']) == {'name': 'x', 'value': '1'}
for label in ['enum-equality', 'enum-array-equality']:
    assert g[label]['rows'] == [{'result': True}]
for label in ['enum-order-reject', 'enum-array-order-reject']:
    assert g[label]['code'] == 'InvalidArgument'
assert set(g['nam15']['optional_read_only_locations']) == {'us-east1', 'us-west3', 'us-central2'}
for label in ['conditional-profile', 'conditional-profile-error-suppression']:
    r = g[label]
    assert r['code'] == 'OK' and float(r['rows'][0]['result']) == 10
    assert r['stats']['rows_scanned'] == '1'
    scans = [n for n in r['plan']['planNodes'] if n.get('displayName') == 'Scan']
    assert len(scans) >= 2
    assert sum(bool(n.get('executionStats')) for n in scans) == 1

graph = labeled('graph-controls')
for label, n in [('graph-node-control', 4), ('graph-edge-control', 1),
                 ('graph-element-to-json-control', 4), ('graph-path-to-json-control', 1),
                 ('graph-property-cast-control', 4)]:
    assert graph[label]['code'] == 'OK' and len(graph[label]['rows']) == n, label
for label in ['graph-element-to-json-control', 'graph-path-to-json-control']:
    assert graph[label]['metadata']['rowType']['fields'] == [
        {'name': 'j', 'type': {'code': 'JSON'}},
    ]
for label in ['graph-element-cast-normal', 'graph-path-cast-normal']:
    assert graph[label]['code'] == 'Unimplemented', label
for label in ['graph-element-cast', 'graph-path-cast']:
    assert g[label]['code'] == 'Unimplemented', label

for name in ['google-options', 'pg-options']:
    options = labeled(name)
    selected = None
    for r in options.values():
        if r['label'].endswith('-CHANGE_STREAM_COLUMNS'):
            assert r['code'] == 'OK' and len(r['rows']) == 1
            row = {k.lower(): v for k, v in r['rows'][0].items()}
            state = (row['table_name'], row['column_name'])
            if selected is None:
                selected = state
            assert state == selected
        if r['label'].startswith('option-only-'):
            assert r['code'] == 'OK', r['label']
        if r['label'].endswith('-CHANGE_STREAM_OPTIONS') and r['label'].startswith('after-'):
            setting = r['label'].removeprefix('after-').removesuffix('-CHANGE_STREAM_OPTIONS')
            option_name, expected = setting.split('=')
            actual = {row['option_name']: row['option_value'] for row in [
                {k.lower(): v for k, v in x.items()} for x in r['rows']
            ]}
            assert actual[option_name] == expected.upper()
    assert options['old-for-all-control']['code'] == 'InvalidArgument'
    assert 'syntax error' in options['old-for-all-control']['error'].lower()
    assert options['drop-database']['code'] == 'OK'
    assert options['database-absent']['code'] == 'NotFound'

for name in ['diagnostic-pg-catalog-case', 'diagnostic-google-tracking-control',
             'diagnostic-pg-tracking-control', 'diagnostic-pg-parameter-syntax',
             'graph-controls']:
    r = labeled(name)
    assert r['drop-database']['code'] == 'OK'
    assert r['database-absent']['code'] == 'NotFound'
assert labeled('diagnostic-pg-quota')['create-database']['code'] == 'ResourceExhausted'
docs = json.loads((base / 'documentation.json').read_text())
assert len(docs['comparison']) == 11
assert all(d['equal_body_after_frontmatter'] for d in docs['comparison'])
receipt = json.loads((base / 'receipt.json').read_text())
for name, expected in receipt['source_sha256'].items():
    assert hashlib.sha256((base / name).read_bytes()).hexdigest() == expected
for file in base.glob('*.jsonl'):
    data = file.read_text()
    assert not re.search(r'projects/(?!<)[^/\s]+/instances/', data), file.name
    assert not re.search(r'requestID = "(?!<redacted>)[^"]', data), file.name
    assert not re.search(r'docdiff-20261010[gp]\d+', data), file.name
result = {
    'status': 'pass', 'readonly_cases': len(q),
    'created_temporary_databases_deleted_and_not_found': 9,
    'matching_official_document_bodies': 11,
    'scope': 'Offline integrity and bounded result assertions; not a runtime attestation.',
}
(base / 'verification.json').write_text(json.dumps(result, indent=2) + '\n')
print(json.dumps(result))
