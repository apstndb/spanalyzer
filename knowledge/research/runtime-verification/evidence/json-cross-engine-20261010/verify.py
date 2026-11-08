"""Check retained evidence offline; no service or container is started."""
from pathlib import Path
import collections
import hashlib
import json
import re

base = Path(__file__).resolve().parent
managed_base = base.parent / 'docs-diff-20261010'
managed = {r['label']: r for name in ['queries', 'followup']
           for r in map(json.loads, (managed_base / (name + '.jsonl')).read_text().splitlines())}
assert len(managed) == 125
comparison = json.loads((base / 'comparison.json').read_text())
receipt = json.loads((base / 'receipt.json').read_text())

def phase(rows, name):
    selected = [r for r in rows if r['phase'] == name]
    assert len({r['label'] for r in selected}) == len(selected)
    return {r['label']: r for r in selected}

def strings(value):
    if isinstance(value, str):
        yield value
    elif isinstance(value, dict):
        for key, item in value.items():
            yield key
            yield from strings(item)
    elif isinstance(value, list):
        for item in value:
            yield from strings(item)

successful_emulator = {
    'json-null-is-sql-null', 'cast-null-is-sql-null', 'cast-json', 'cast-sql-null',
    'to-json-control-sql-null', 'safe-cast-sql-null', 'to-json-control-positive-infinity',
    'to-json-control-wide-int', 'coalesce-unused', 'ifnull-unused', 'if-unused',
    'case-simple-unused', 'case-searched-unused', 'coalesce-subquery-unused',
    'if-subquery-unused',
}
for engine in ['omni', 'emulator']:
    rows = [json.loads(s) for s in (base / (engine + '.jsonl')).read_text().splitlines()]
    assert len(rows) == 230
    q = {k: r for k, r in phase(rows, 'google-constants').items() if k in managed}
    assert set(q) == set(managed)
    if engine == 'omni':
        assert all(q[k]['code'] == managed[k]['code'] and q[k]['rows'] == managed[k]['rows'] for k in q)
        assert collections.Counter(r['code'] for r in q.values()) == {
            'OK': 99, 'InvalidArgument': 12, 'OutOfRange': 13, 'Unimplemented': 1,
        }
    else:
        assert collections.Counter(r['code'] for r in q.values()) == {
            'OK': 15, 'InvalidArgument': 102, 'OutOfRange': 8,
        }
        assert {k for k, r in q.items() if r['code'] == 'OK'} == successful_emulator
        assert all(q[k]['rows'] == managed[k]['rows'] for k in successful_emulator)
        for label in ['json-equality', 'json-type-order', 'cast-int', 'cast-string',
                      'cast-array', 'cast-struct', 'json-sort', 'json-group', 'json-distinct']:
            assert q[label]['code'] == 'InvalidArgument'
            assert 'syntax error' not in q[label]['error'].lower()
        assert all(r['code'] == 'InvalidArgument' for k, r in q.items() if k.startswith('coerce-'))
    for prefix in ['coalesce', 'ifnull', 'if', 'case-simple', 'case-searched', 'coalesce-subquery']:
        assert q[prefix + '-unused']['rows'] == [{'result': 10}]
        assert q[prefix + '-selected']['code'] == 'OutOfRange'
    assert q['coalesce-name-resolution']['code'] == 'InvalidArgument'
    assert q['procedural-if-suggestion']['code'] == 'InvalidArgument'
    assert 'syntax error' in q['procedural-if-suggestion']['error'].lower()

    for name in ['google-fixtures', 'pg-fixtures']:
        p = phase(rows, name)
        assert p['database-cleanup']['code'] == 'OK'
        assert p['database-absent']['code'] == 'NotFound'
        for value in [100, 999, 1000, 300000]:
            r = p['heartbeat-boundary-' + str(value)]
            assert r['code'] == 'OK' and len(r['rows']) == 2
        for value in [99, 300001]:
            r = p['heartbeat-boundary-' + str(value)]
            assert r['code'] == ('InvalidArgument' if engine == 'omni' else 'OutOfRange')
            assert 'min 100 and max 300000' in r['error']
        assert p['read-options-omitted']['code'] == 'OK'
        for label in ['read-options-empty', 'read-options-nonnull']:
            assert p[label]['code'] == ('InvalidArgument' if engine == 'omni' else 'Unimplemented')
            assert 'read_options must be null' in p[label]['error']
        assert p['heartbeat-child-100']['code'] == 'OK'
        heartbeat = comparison['runtimes'][engine]['heartbeats'][name]
        assert heartbeat['records'] >= 15
        assert 90 <= heartbeat['timestamp_gap_ms']['median'] <= 120
        assert p['old-for-all-control']['code'] == 'InvalidArgument'
        assert 'syntax error' in p['old-for-all-control']['error'].lower()
        tracking = None
        for label, r in p.items():
            if label.endswith('-CHANGE_STREAM_COLUMNS'):
                assert r['code'] == 'OK' and len(r['rows']) == 1
                row = {k.lower(): v for k, v in r['rows'][0].items()}
                value = (row['table_name'], row['column_name'])
                if tracking is None:
                    tracking = value
                assert value == tracking
            if label.startswith('option-only-'):
                assert r['code'] == 'OK'
            if label.startswith('after-') and label.endswith('-CHANGE_STREAM_OPTIONS'):
                setting = label.removeprefix('after-').removesuffix('-CHANGE_STREAM_OPTIONS')
                key, value = setting.split('=')
                options = {x['option_name']: x['option_value'] for x in [
                    {k.lower(): v for k, v in row.items()} for row in r['rows']
                ]}
                if engine == 'omni':
                    assert options[key] == value.upper()
                else:
                    assert r['code'] == 'OK' and r['rows'] == []

    g = phase(rows, 'google-fixtures')
    assert g['graph-node-control']['code'] == 'OK' and len(g['graph-node-control']['rows']) == 4
    assert g['graph-edge-control']['code'] == 'OK' and len(g['graph-edge-control']['rows']) == 1
    for label in ['graph-element-cast-normal', 'graph-path-cast-normal']:
        assert g[label]['code'] == ('Unimplemented' if engine == 'omni' else 'InvalidArgument')
    path = g['graph-path-to-json-control']
    assert path['code'] == 'OK'
    assert path['metadata']['rowType']['fields'] == [{'name': 'j', 'type': {'code': 'JSON'}}]
    value = json.loads(path['rows'][0]['j'])
    assert [e['kind'] for e in value] == ['node', 'edge', 'node']
    assert [e['properties'] for e in value] == [
        {'Id': 1, 'Value': 10}, {'Src': 1, 'Dst': 2}, {'Id': 2, 'Value': 20},
    ]
    assert value[1]['source_node_identifier'] == value[0]['identifier']
    assert value[1]['destination_node_identifier'] == value[2]['identifier']
    for label in ['conditional-profile', 'conditional-profile-error-suppression']:
        assert g[label]['code'] == 'OK' and float(g[label]['rows'][0]['result']) == 10
        if engine == 'omni':
            assert g[label]['stats']['rows_scanned'] == '1'
            scans = [n for n in g[label]['plan']['planNodes'] if n.get('displayName') == 'Scan']
            assert len(scans) >= 2 and sum(bool(n.get('executionStats')) for n in scans) == 1
        else:
            assert not g[label].get('plan', {}).get('planNodes')
            assert 'rows_scanned' not in g[label].get('stats', {})
    if engine == 'omni':
        assert g['graph-element-to-json-control']['code'] == 'OK'
        assert g['cast-enum']['rows'] == [{'result': '"ONE"'}]
        assert json.loads(g['cast-proto']['rows'][0]['result']) == {'name': 'x', 'value': '1'}
        assert [r['Id'] for r in g['stored-json-sort']['rows']] == ['2', '4', '1', '3']
        assert [r['n'] for r in g['stored-json-group']['rows']] == ['2', '2']
    else:
        for label in ['graph-element-to-json-control', 'graph-property-cast-control',
                      'cast-enum', 'cast-proto', 'stored-json-sort', 'stored-json-group']:
            assert g[label]['code'] == 'InvalidArgument'
    assert phase(rows, 'cleanup')['runtime-cleanup']['code'] == 'OK'
    # Go's encoder escapes angle brackets as Unicode sequences. Audit decoded
    # strings so an escaped redaction marker is not mistaken for a destination.
    for value in strings(rows):
        assert not re.search(r'projects/(?!<)[^/\s]+/instances/', value)
        assert not re.search(r'requestID = "(?!<redacted>)[^"]', value)

cleanup = json.loads((base / 'container-cleanup.json').read_text())
for target in cleanup['targets'].values():
    assert target['containers_remaining'] == []
    assert target['local_pinned_image']['Architecture'] == 'arm64'
for name, expected in receipt['source_sha256'].items():
    assert hashlib.sha256((base / name).read_bytes()).hexdigest() == expected
for name, expected in comparison['cases_sha256'].items():
    assert hashlib.sha256((base / name).read_bytes()).hexdigest() == expected
    assert (base / name).read_bytes() == (managed_base / name).read_bytes()
assert 'kChangeStreamsMinHeartbeatMilliseconds = 100;' in (base / 'common__limits.h.txt').read_text()
result = {
    'status': 'pass', 'container_records': 460, 'constant_cases_per_runtime': 125,
    'omni_matching_managed_codes': 125, 'omni_matching_successful_results': 99,
    'emulator_matching_managed_codes': 35, 'emulator_matching_successful_results': 15,
    'owned_databases_absent': 4, 'remaining_tested_containers': 0,
    'scope': 'Offline evidence consistency; no additional runtime execution or attestation.',
}
(base / 'verification.json').write_text(json.dumps(result, indent=2) + '\n')
print(json.dumps(result))
