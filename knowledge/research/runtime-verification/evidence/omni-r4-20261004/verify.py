#!/usr/bin/env python3
"""Validate retained r4 observations and hashes without contacting a service."""
import collections
import hashlib
import json
from pathlib import Path

here = Path(__file__).resolve().parent

def records(name):
    rows = []
    for line in (here / name).read_text().splitlines():
        try:
            rows.append(json.loads(line))
        except json.JSONDecodeError:
            assert name == 'integration-tests.jsonl'
    return rows

def outcomes(name, nested=False):
    return collections.Counter(r['Action'] for r in records(name)
        if r.get('Test') and ('/' in r['Test']) == nested
        and r['Action'] in ('pass', 'fail', 'skip'))

c = json.loads((here / 'catalog-comparison.json').read_text())
assert c['infoschema']['r3_columns'] == c['infoschema']['r4_columns'] == 310
assert c['infoschema']['r3_surface_sha256'] == c['infoschema']['r4_surface_sha256']
assert (c['spanner-sys']['r3_columns'], c['spanner-sys']['r4_columns']) == (587, 598)
assert len(c['spanner-sys']['added']) == 11
assert {r['table_name'] for r in c['spanner-sys']['added']} == {'ACTIVE_BATCH_OPERATIONS'}
assert all(not v['removed'] and not v['changed'] for v in c.values())
assert outcomes('integration-tests.jsonl') == {'pass': 20, 'fail': 2}
assert outcomes('integration-tests.jsonl', True) == {'pass': 200, 'fail': 18}
assert outcomes('query-generator.jsonl') == {'pass': 3}
assert outcomes('query-generator.jsonl', True) == {'pass': 26}

features = records('feature-r4.jsonl')
assert len(features) == 93
approx = [r for r in features if r.get('label') == 'approx-count-distinct' and r.get('kind') == 'query']
assert len(approx) == 9 and all(r['code'] == 'OK' and r['rows'] == [{'n': '2'}] for r in approx)
auto = [r for r in features if r.get('kind') == 'plan-probe' and r.get('label') in ['auto-parameter-DEFAULT', 'auto-parameter-AUTO', 'auto-parameter-ON']]
assert len(auto) == 6 and all(r['stats']['auto_parameters'] for r in auto)
quantifiers = [r for r in features if r.get('label', '').startswith('quantified-')]
assert len(quantifiers) == 12 and all(r['code'] == 'InvalidArgument' for r in quantifiers)
ilike = [r for r in features if r.get('label') == 'ilike-isolated']
assert [r['rows'] for r in ilike[:4]] == [[{'result': True}], [{'result': False}], [{'result': True}], [{'result': False}]]
assert all(r['code'] == 'InvalidArgument' for r in ilike[4:])
assert [len(r['rows']) for r in features if r.get('label') == 'reservoir'] == [0, 1, 2]
assert next(r for r in features if r.get('label') == 'active-batch-operations')['code'] == 'OK'
assert [r['code'] for r in features if r.get('label') == 'graph-uniqueness'] == ['FailedPrecondition', 'FailedPrecondition', 'OK']
for r in features:
    if r.get('label') in ['graph-metadata', 'semantic-metadata']:
        options = json.loads(r['rows'][0]['PROPERTY_GRAPH_METADATA_JSON'])['options']
        assert {'name':'allow_view_with_volatile_column','type':'BOOL','value':'FALSE'} in options

queue = records('queue-r4.jsonl')
controls = records('queue-control-r4.jsonl')
assert len(queue) == 32 and len(controls) == 14
q = {r.get('label'):r for r in queue}
assert q['after-rollback']['rows'] == []
assert q['receive-immediate']['rows'][0]['MessageId'] == 'immediate'
assert q['after-ack-rollback']['rows'] == [{'MessageId':'future'}, {'MessageId':'immediate'}]
assert q['ack-commit']['code'] == 'OK' and q['ack-commit']['rows_modified'] == [1]
assert q['duplicate-ack']['code'] == 'OutOfRange'
assert q['future-not-received']['rows'] == []
assert q['future-still-stored']['rows'] == [{'MessageId':'future'}]
assert q['cleanup-confirmation']['rows'] == [{'remaining':'0'}]
cb = {r.get('label'):r for r in controls}
assert cb['control-stale-ordinary']['rows'] == [{'MessageId':'lease-control'}]
assert cb['control-stale-receive']['code'] == 'InvalidArgument'
assert 'must use strong concurrency' in cb['control-stale-receive']['error']
for label in ['control-scalar-default', 'control-scalar-off']:
    assert cb[label]['code'] == 'InvalidArgument' and 'invalid lease token' in cb[label]['error']
for label in ['control-array-default', 'control-array-off']:
    assert cb[label]['code'] == 'OK' and len(cb[label]['rows']) == 1
for rows in [features, queue, controls]:
    cleanup = [r for r in rows if r.get('label') in ['database-cleanup','runtime-cleanup']]
    assert cleanup and all(r['code'] == 'OK' for r in cleanup)
for r in queue + controls:
    for row in r.get('rows', []) if isinstance(r.get('rows'), list) else []:
        for key,value in row.items():
            if 'Token' in key and value is not None:
                assert value.startswith('sha256:') and len(value) == 71

security = {r['label']:r for r in records('security.jsonl')}
for label in ['create-mtls','mtls','password','drop-mtls']:
    assert security[label]['code'] == 'OK'
for label in ['mtls','password']:
    assert security[label]['value'] == 1
for label in ['bad-password','no-credentials']:
    assert security[label]['code'] == 'Unauthenticated'
assert all(r['exit_code'] == 0 for r in json.loads((here/'security-commands.json').read_text()))
volatile = {r['label']:r for r in records('volatile-r4.jsonl')}
assert volatile['graph-0']['code'] == volatile['graph-read-0']['code'] == 'OK'
assert len(volatile['graph-read-0']['rows']) == 1
for label in ['graph-1','graph-2']:
    assert volatile[label]['code'] == 'Unimplemented'
    assert 'option is not supported' in volatile[label]['error']
for label in ['database-cleanup','runtime-cleanup']:
    assert volatile[label]['code'] == 'OK'
assert not (here/'containers-after.txt').read_text().strip()
manifest = json.loads((here.parent/'artifact-sha256.json').read_text())['files']
for p in here.rglob('*'):
    if p.is_file():
        assert hashlib.sha256(p.read_bytes()).hexdigest() == manifest[str(p.relative_to(here.parent))], p.name
print('PASS: r4 catalog, SQL, queues, integration outcomes, TLS authentication, cleanup, and retained hashes')
