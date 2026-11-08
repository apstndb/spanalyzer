#!/usr/bin/env python3
"""Check retained Omni r3 observations offline; never contact a service."""
import collections
import hashlib
import json
from pathlib import Path

here = Path(__file__).resolve().parent

def records(name):
    result = []
    for line in (here / name).read_text().splitlines():
        try:
            result.append(json.loads(line))
        except json.JSONDecodeError:
            # Integration stderr also contains the outer runner's failure text.
            assert name == 'integration-tests.jsonl'
    return result

def outcomes(rows, nested=False):
    return collections.Counter(
        r['Action'] for r in rows
        if r.get('Test') and ('/' in r['Test']) == nested
        and r['Action'] in ('pass', 'fail', 'skip')
    )

comparison = json.loads((here / 'catalog-comparison.json').read_text())
for name, old, new, added in [('infoschema', 308, 310, 2), ('spanner-sys', 539, 587, 48)]:
    d = comparison[name]
    assert (d['r2_count'], d['r3_count'], len(d['added'])) == (old, new, added)
    assert d['matches_managed_surface'] and not d['removed'] and not d['changed']
assert outcomes(records('integration-tests.jsonl')) == {'pass': 20, 'fail': 2}
assert outcomes(records('integration-tests.jsonl'), True) == {'pass': 200, 'fail': 18}
assert outcomes(records('query-generator.jsonl')) == {'pass': 3}
assert outcomes(records('query-generator.jsonl'), True) == {'pass': 26}
for version in ['r2', 'r3']:
    rows = records(f'feature-{version}.jsonl')
    assert len(rows) == 92
    approx = [r for r in rows if r.get('label') == 'approx-count-distinct' and r.get('kind') == 'query']
    assert len(approx) == 9
    assert all(r['code'] == ('OK' if version == 'r3' else 'Unimplemented') for r in approx)
    if version == 'r3':
        assert all(r['rows'] == [{'n': '2'}] for r in approx)
    auto = [r for r in rows if r.get('kind') == 'plan-probe' and r.get('label') in ['auto-parameter-DEFAULT', 'auto-parameter-AUTO', 'auto-parameter-ON']]
    assert len(auto) == 6
    assert all(bool((r.get('stats') or {}).get('auto_parameters')) == (version == 'r3') for r in auto)
    quantifiers = [r for r in rows if r.get('label', '').startswith('quantified-')]
    assert len(quantifiers) == 12 and all(r['code'] == 'InvalidArgument' for r in quantifiers)
    ilike = [r for r in rows if r.get('label') == 'ilike-isolated']
    assert [r['rows'] for r in ilike[:4]] == [[{'result': True}], [{'result': False}], [{'result': True}], [{'result': False}]]
    assert all(r['code'] == 'InvalidArgument' for r in ilike[4:])
    reservoir = [r for r in rows if r.get('label') == 'reservoir']
    assert [len(r['rows']) for r in reservoir] == [0, 1, 2]
    samples = [r for r in rows if r.get('label') == 'tablesample']
    assert [len(r['rows']) for r in samples[:2]] == [0, 2]
    assert all(r['code'] == 'Unimplemented' for r in samples[2:])
    cleanup = [r for r in rows if r.get('label') in ['database-cleanup', 'runtime-cleanup']]
    assert len(cleanup) == 3 and all(r['code'] == 'OK' for r in cleanup)
coverage = json.loads((here / 'post-cutoff-coverage.json').read_text())
assert len(coverage['coverage']) == 4
assert {x['date'] for x in coverage['coverage']} == {'2026-09-11', '2026-09-17', '2026-09-18'}
for version in ['r2', 'r3']:
    features = records(f'feature-{version}.jsonl')
    assert [r['code'] for r in features if r.get('label') == 'graph-uniqueness'] == ['FailedPrecondition', 'FailedPrecondition', 'OK']
    assert next(r for r in features if r.get('label') == 'graph-match')['rows'] == [{'n': '2'}]
    assert next(r for r in features if r.get('label') == 'graph-semantic')['code'] == 'OK'
    exported = next(r for r in features if r.get('label') == 'semantic-export')
    assert exported['code'] == 'OK' and 'synonyms' in str(exported['ddl']) and 'description' in str(exported['ddl'])
    rows = records(f'queue-{version}.jsonl')
    assert len(rows) == 32
    by = {r.get('label'): r for r in rows}
    for label in ['create', 'enqueue-immediate-and-future', 'receive-immediate', 'renew-array-while-stream-open', 'ack-commit', 'enqueue-default-delivery', 'cleanup', 'database-cleanup', 'runtime-cleanup']:
        assert by[label]['code'] == 'OK', (version, label)
    assert by['after-rollback']['rows'] == []
    for label in ['enqueue-rollback', 'ack-rollback']:
        assert by[label]['intentional_rollback'] and by[label]['error'] == 'intentional probe rollback'
    assert by['receive-immediate']['rows'][0]['MessageId'] == 'immediate'
    assert by['after-ack-rollback']['rows'] == [{'MessageId': 'future'}, {'MessageId': 'immediate'}]
    assert by['ack-commit']['rows_modified'] == [1]
    assert by['duplicate-ack']['code'] == 'OutOfRange' and by['duplicate-ack']['rows_modified'] == [0]
    assert by['future-not-received']['rows'] == [] and by['future-not-received']['code'] == 'OK'
    assert by['future-still-stored']['rows'] == [{'MessageId': 'future'}]
    assert by['cleanup-confirmation']['rows'] == [{'remaining': '0'}]
    assert by['catalog-tables']['rows'][0]['TABLE_TYPE'] == 'QUEUE'
    assert len(by['queue-routines']['rows']) == 2
    plan = by['receive-plan']
    assert plan['code'] == 'OK' and 'TVF' in [n['displayName'] for n in plan['plan']['planNodes']]
    controls = records(f'queue-control-{version}.jsonl')
    assert len(controls) == 14
    cb = {r.get('label'): r for r in controls}
    assert cb['control-stale-ordinary']['rows'] == [{'MessageId': 'lease-control'}]
    assert cb['control-stale-receive']['code'] == 'InvalidArgument'
    assert 'must use strong concurrency' in cb['control-stale-receive']['error']
    for label in ['control-scalar-default', 'control-scalar-off']:
        assert cb[label]['code'] == 'InvalidArgument' and 'invalid lease token' in cb[label]['error']
    for label in ['control-array-default', 'control-array-off']:
        r = cb[label]
        assert r['code'] == 'OK' and len(r['rows']) == 1
        assert r['rows'][0]['SpannerNewLeaseToken'].startswith('sha256:')
    assert cb['cleanup-confirmation']['rows'] == [{'remaining': '0'}]
    for label in ['cleanup', 'database-cleanup', 'runtime-cleanup']:
        assert cb[label]['code'] == 'OK'
    # Tokens may appear only as hashes, never as reusable credentials.
    for r in rows + controls:
        for row in r.get('rows', []) if isinstance(r.get('rows'), list) else []:
            for key, value in row.items():
                if 'Token' in key and value is not None:
                    assert value.startswith('sha256:') and len(value) == 71
assert not (here / 'queue-containers-after.txt').read_text().strip()
assert not (here / 'containers-after.txt').read_text().strip()
manifest = json.loads((here.parent / 'artifact-sha256.json').read_text())['files']
for path in here.iterdir():
    if path.is_file():
        assert hashlib.sha256(path.read_bytes()).hexdigest() == manifest[str(path.relative_to(here.parent))], path.name
print('PASS: catalog deltas, post-cutoff coverage, paired features and queues, integration outcomes, cleanup, and retained hashes')
