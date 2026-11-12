"""Check compact FK observations and reproduction identities offline."""
import hashlib
import json
from pathlib import Path
import runpy
import subprocess

here = Path(__file__).resolve().parent
root = Path(subprocess.check_output(['git', 'rev-parse', '--show-toplevel'], cwd=here, text=True).strip())
receipt = json.loads((here / 'receipt.json').read_text())
make_plan = runpy.run_path(str(here / 'prepare.py'))['make_plan']
runs = {}
for item in receipt['runs']:
    rows = [json.loads(line) for line in (here / item['retained_file']).read_text().splitlines()]
    assert len({r['label'] for r in rows}) == len(rows), item['id']
    by = {r['label']: r for r in rows}
    plan = make_plan(item['variant'])
    data = (json.dumps(plan, indent=2) + '\n').encode()
    assert hashlib.sha256(data).hexdigest() == item['input_sha256'], item['id']
    assert [r['label'] for r in rows if r.get('kind')] == [s['label'] for s in plan['steps']], item['id']
    assert item['planned_steps'] == len(plan['steps']) and item['producer_exit_code'] == 0
    if item['id'] == 'managed':
        assert [by[k]['code'] for k in ['create-database', 'drop-database', 'database-absent']] == ['OK', 'OK', 'NotFound']
    else:
        assert by['fixture-open']['code'] == by['fixture-client-closed']['code'] == by['runtime-closed']['code'] == 'OK'
        actual = by['runtime-started']['provenance']
        assert actual['Source'] == 'container_inspect' and actual['ObservedByThisProcess']
        assert actual['Platform'] == 'linux/arm64' and actual['ManifestDigest']
        assert by['runtime-started']['image'].endswith('@' + actual['ManifestDigest'])
    for row in rows:
        assert 'requestID = "' not in row.get('error', '') or 'requestID = "<redacted>"' in row['error']
        if row.get('kind') == 'ddl' or row.get('kind') == 'query':
            assert row['code'] == 'OK', (item['id'], row['label'], row['code'])
        if row.get('kind') == 'dml':
            assert row.get('stats_present') or row.get('mutation_count') is None
            if row['code'] != 'OK':
                assert row.get('mutation_count') is None
        if row.get('code') == 'NOT_RUN':
            assert row['execution_status'] == 'blocked_by_fixture_dependency'
    runs[item['id']] = by


def indexes(run, label, table):
    rows = [r for r in runs[run][label]['rows'] if r['TABLE_NAME'] == table]
    grouped = {}
    for row in rows:
        grouped.setdefault(row['INDEX_NAME'], []).append(row)
    return grouped


def names(run, label, table):
    return set(indexes(run, label, table))


matrix = ['emulator', 'emulator-parent-first', 'omni-lts-fixed', 'omni-r5-fixed']
for run in matrix:
    assert len(names(run, 'not-null-non-pk/indexes', 'NotNullSide')) == 1
    assert not names(run, 'not-null-pk/indexes', 'PkCovered')
    assert not names(run, 'not-null-prefix/indexes', 'PrefixCovered')
    nullable = indexes(run, 'nullable-pk/indexes', 'NullablePk')
    assert len(nullable) == 1 and all(r['IS_NULL_FILTERED'] and r['SPANNER_IS_MANAGED'] for rows in nullable.values() for r in rows)
    assert all(not r['IS_NULL_FILTERED'] for rows in indexes(run, 'not-null-non-pk/indexes', 'NotNullSide').values() for r in rows)
    ordinary = indexes(run, 'existing-user-index/indexes', 'UserIndexed')
    assert len(ordinary) == 2 and {rows[0]['SPANNER_IS_MANAGED'] for rows in ordinary.values()} == {True, False}
    shared = names(run, 'shared/indexes', 'SharedSide')
    assert len(shared) == 1 and names(run, 'sharing/drop-first/indexes', 'SharedSide') == shared
    assert not names(run, 'sharing/drop-last/indexes', 'SharedSide')
    assert names(run, 'sharing/recreate/indexes', 'SharedSide') == shared
    assert names(run, 'repeat/first/indexes', 'RepeatSide') == names(run, 'repeat/recreate/indexes', 'RepeatSide')
    assert not names(run, 'informational/indexes', 'InfoSide')
    assert len(names(run, 'cyclic-covered-pk/indexes', 'CycleCovered')) == 1
    assert not names(run, 'cyclic-inverse-pk/indexes', 'CycleInverse')
    assert names(run, 'source-fixture/indexes', 'T') and names(run, 'source-fixture/indexes', 'U')

assert names('emulator', 'source-fixture/indexes', 'T') == {'IDX_T_X_Y_U_5AD6E41B495C5BB9'}
assert names('emulator', 'source-fixture/indexes', 'U') == {'IDX_U_A_B_N_8C11B65ACA7F01B9'}
for label in [k for k, v in runs['omni-lts-fixed'].items() if v.get('kind') == 'query']:
    assert runs['omni-lts-fixed'][label]['rows'] == runs['omni-r5-fixed'][label]['rows'], label
    assert runs['managed'][label]['rows'] == runs['omni-lts-fixed'][label]['rows'], label
    shapes = lambda rows: sorted(json.dumps({k: v for k, v in r.items() if k != 'INDEX_NAME'}, sort_keys=True) for r in rows)
    assert shapes(runs['emulator'][label]['rows']) == shapes(runs['managed'][label]['rows']), label
assert names('emulator', 'nullable-pk/indexes', 'NullablePk') != names('omni-lts-fixed', 'nullable-pk/indexes', 'NullablePk')

for run in ['emulator', 'emulator-parent-first']:
    by = runs[run]
    assert by['data/cycle-covered-valid']['code'] == 'OK'
    assert by['data/cycle-inverse-valid']['code'] == 'FailedPrecondition'
    assert all(not r['stats_present'] and r['mutation_count'] is None for r in by.values() if r.get('kind') == 'dml')
for run in ['omni-lts-fixed', 'omni-r5-fixed']:
    assert runs[run]['data/seed']['code'] == 'Internal'
    assert sum(r.get('code') == 'NOT_RUN' for r in runs[run].values()) == 6
for run in ['omni-lts-followup', 'omni-r5-followup', 'managed']:
    by = runs[run]
    assert by['data/seed']['code'] == 'OK' and by['data/seed']['mutation_count'] == 6
    assert by['data/nullable-pk']['mutation_count'] == 3
    assert by['data/not-null-pk']['mutation_count'] == 2
    assert by['data/cycle-covered-valid']['mutation_count'] == 4
    assert by['data/cycle-covered-invalid']['code'] == 'FailedPrecondition'
    assert by['data/cycle-inverse-valid']['code'] == by['data/cycle-inverse-invalid']['code'] == 'Internal'
ordered = runs['omni-lts-ordering']
assert ordered['ordered/insert']['code'] == 'OK' and ordered['ordered/insert']['mutation_count'] == 4
assert [r['COLUMN_NAME'] for r in ordered['ordered/indexes']['rows']] == ['Y', 'X']

fingerprints = {r['text']: r['fingerprint_hex'] for r in map(json.loads, (here / 'fingerprint-readonly.jsonl').read_text().splitlines())}
assert fingerprints['2695C62E33338E4B_IDX_NotNullSide_Owner'] == '4E6D4D96C9D3C570'
assert fingerprints['2695C62E33338E4B_IDX_NullableSide_Owner_N'] == 'D89DBA0E613318D5'
assert fingerprints['IDX_NotNullSide_Owner'] == '544A76254222AC1C'
assert fingerprints['IDX_NullableSide_Owner_N'] == '630E75C40FF74EA7'
assert not any(name.endswith(value) for name in names('managed', 'not-null-non-pk/indexes', 'NotNullSide') for value in fingerprints.values())
assert hashlib.sha256((here / receipt['readonly_fingerprint_control']['source']).read_bytes()).hexdigest() == receipt['readonly_fingerprint_control']['source_sha256']

producer = json.loads((here / 'producer.json').read_text())
for item in producer['producers'].values():
    source = subprocess.check_output(['git', 'show', producer['source_commit'] + ':' + item['path']], cwd=root)
    assert hashlib.sha256(source).hexdigest() == item['source_sha256']
    assert hashlib.sha256((here / item['patch']).read_bytes()).hexdigest() == item['patch_sha256']
print(json.dumps({'status': 'pass', 'runs': len(runs), 'recorded_steps': sum(r['planned_steps'] for r in receipt['runs']), 'network_or_database_actions': 0}))
