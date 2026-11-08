"""Offline assertions for the dated captures; no service calls."""
import json
from pathlib import Path

root = Path(__file__).resolve().parent
load = lambda name: [json.loads(line) for line in (root / name).read_text().splitlines()]
g, p = [load(name) for name in ('google-features.jsonl', 'pg-features.jsonl')]
f = p
select = lambda rows, label: [r for r in rows if r.get('label') == label]
for rows in (g, p):
    for mode in ('DEFAULT', 'AUTO', 'ON', 'OFF'):
        observations = select(rows, 'auto-parameter-' + mode)
        assert len(observations) == 2
        for i, r in enumerate(observations, 1):
            assert r['code'] == 'OK' and r['rows'] == 1
            assert r['stats']['optimizer_version'] == '9'
            if mode == 'OFF':
                assert 'auto_parameters' not in r['stats']
            else:
                assert r['stats']['auto_parameters'] == {'_auto_p0_int64_': str(i)}
        results = select(rows, 'result-' + mode)
        assert len(results) == 2
        for i, r in enumerate(results, 1):
            assert r['code'] == 'OK'
            row = {k.lower(): v for k, v in r['rows'][0].items()}
            assert row == {'id': str(i), 'name': ['Alice', 'Bob'][i-1]}
    assert all(r['code'] == 'InvalidArgument' for r in rows if r['label'].endswith('-INVALID'))
assert [r['code'] for r in select(g, 'graph-uniqueness')] == ['FailedPrecondition', 'FailedPrecondition', 'OK']
assert select(g, 'graph-match')[0]['rows'] == [{'n': '2'}]
meta = json.loads(select(g, 'semantic-metadata')[0]['rows'][0]['PROPERTY_GRAPH_METADATA_JSON'])
assert {o['name'] for o in meta['labels'][0]['options']} == {'description', 'synonyms'}
assert {o['name'] for o in meta['nodeTables'][0]['propertyDefinitions'][1]['options']} == {'description', 'synonyms'}
assert meta['options'][0]['value'] == 'TRUE'
meta = json.loads(select(g, 'graph-metadata')[0]['rows'][0]['PROPERTY_GRAPH_METADATA_JSON'])
assert meta['options'][0]['value'] == 'FALSE'
assert 'synonyms' in select(g, 'semantic-export')[0]['ddl'][0]
assert select(g, 'cleanup')[0]['rows'] == []
assert all(r['code'] == 'OK' for r in g if r['label'].startswith('drop-'))
for rows in (p, f):
    assert select(rows, 'drop-database')[0]['code'] == 'OK'
matching = select(f, 'ilike-isolated')
assert [r['rows'] for r in matching[:4]] == [[{'result': v}] for v in (True, False, True, False)]
assert all(r['code'] == 'InvalidArgument' and 'does not exist' in r['error'] for r in matching[4:])
samples = select(p, 'tablesample')
assert [len(r['rows']) for r in samples[:2]] == [0, 2]
assert all(r['code'] == 'OK' for r in samples[:2])
assert all(r['code'] == 'Unimplemented' for r in samples[2:])
assert [len(r['rows']) for r in select(f, 'reservoir')] == [0, 1, 2]
assert all(r['code'] == 'OK' for r in select(f, 'reservoir'))

for name, count in [('infoschema', 310), ('spanner-sys', 587)]:
    report = json.loads((root / (name + '-check.json')).read_text())
    baseline = json.loads((root.parent / 'managed-recheck-20260925' / (name + '-check.json')).read_text())
    assert report['current']['column_count'] == count
    assert not report['material_change']
    assert report['current']['surface_sha256'] == baseline['current']['surface_sha256']
r = load('readonly.jsonl')
d = [x for x in r if x.get('label', '').startswith('distribution-')]
assert len(d) == 48 and all(x['code'] == 'OK' for x in d)
assert len([x for x in d if x['rows']]) == 32
quantified = [x for x in r if x.get('label') in ['comparison', 'like']]
assert len(quantified) == 12 and all(x['code'] == 'InvalidArgument' for x in quantified)
parsed = [x for x in r if x.get('kind') == 'parser']
assert len(parsed) == 12 and all(x['error'] for x in parsed)
assert select(r, 'approx-count-distinct')[0]['rows'] == [{'n': '2'}]
assert [x for x in r if x.get('kind') == 'profile'][0]['stats']['optimizer_version'] == '9'
q = load('queue.jsonl')
b = {x.get('label'): x for x in q}
for label in ['create', 'enqueue-immediate-and-future', 'receive-immediate', 'ack-commit', 'enqueue-default-delivery', 'cleanup', 'cleanup-confirmation']:
    assert b[label]['code'] == 'OK', (label, b[label])
assert b['after-rollback']['rows'] == []
assert b['after-ack-rollback']['rows'] == [{'MessageId': 'future'}, {'MessageId': 'immediate'}]
assert b['ack-commit']['rows_modified'] == [1]
assert b['duplicate-ack']['code'] == 'OutOfRange'
assert b['future-not-received']['rows'] == []
assert b['future-still-stored']['rows'] == [{'MessageId': 'future'}]
assert b['cleanup-confirmation']['rows'] == [{'remaining': '0'}]
assert b['stale-ordinary']['code'] == 'OK' and len(b['stale-ordinary']['rows']) == 2
assert b['stale-receive']['code'] == 'InvalidArgument' and 'strong' in b['stale-receive']['error']
for label in ['control-array-default', 'control-array-off']:
    assert b[label]['code'] == 'OK' and len(b[label]['rows']) == 1
    assert b[label]['rows'][0]['SpannerNewLeaseToken'].startswith('sha256:')
for x in q:
    for row in x.get('rows', []) if isinstance(x.get('rows'), list) else []:
        for key, value in row.items():
            if 'Token' in key and value is not None:
                assert value.startswith('sha256:') and len(value) == 71
for label in ['control-scalar-default', 'control-scalar-off', 'invalid-renew-token']:
    assert b[label]['code'] == 'InvalidArgument' and 'invalid lease token' in b[label]['error']
cleanup = load('cleanup.jsonl')
assert len(cleanup) == 4
assert all(x['code'] == 'OK' and x['rows'] == [] for x in cleanup[:3])
assert cleanup[3]['code'] == 'NotFound'
for name in ['commands.json', 'readonly-command.json', 'feature-commands.json', 'cleanup-command.json']:
    assert all(x['exit_code'] == 0 for x in json.loads((root/name).read_text()))
log=(root/'managed-tests.log').read_text()
assert log.count('--- PASS: Test') == 4 and '--- SKIP:' not in log and '--- FAIL:' not in log
import hashlib
manifest=json.loads((root.parent/'artifact-sha256.json').read_text())['files']
for path in root.iterdir():
    if path.is_file():assert hashlib.sha256(path.read_bytes()).hexdigest() == manifest[str(path.relative_to(root.parent))],path.name
print('PASS: managed catalog, feature, queue, integration, cleanup, and retained-hash assertions')
