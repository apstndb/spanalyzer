"""Offline assertions for the dated captures; no service calls."""
import json
from pathlib import Path

root = Path(__file__).resolve().parent
load = lambda name: [json.loads(line) for line in (root / name).read_text().splitlines()]
g, p, f = [load(name) for name in ('managed.jsonl', 'pg-managed.jsonl', 'pg-followup.jsonl')]
select = lambda rows, label: [r for r in rows if r['label'] == label]
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
for mode in ('DEFAULT', 'AUTO'):
    assert select(g, 'auto-parameter-' + mode)[1]['stats']['query_plan_cached'] == 'true'
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
print(json.dumps({'status': 'passed', 'records': len(g)+len(p)+len(f), 'scope': 'parameterization, graph metadata, matching, sampling and cleanup'}))
