"""Offline evidence validation; no service calls or projection updates."""
from pathlib import Path
import hashlib, json, re

root = Path(__file__).resolve().parent
load = lambda name: [json.loads(x) for x in (root / name).read_text().splitlines()]
select = lambda rows, label: [x for x in rows if x.get('label') == label]
g, p, q, r, a, cleanup = [load(name + '.jsonl') for name in ['google-features','pg-features','queue','readonly','active-batch','cleanup']]
assert [len(x) for x in [g,p,q,r,a,cleanup]] == [53,40,32,86,19,4]

for rows in [g,p]:
    for mode in ['DEFAULT','AUTO','ON','OFF']:
        observations = select(rows,'auto-parameter-' + mode)
        assert len(observations) == 2
        for i,x in enumerate(observations,1):
            assert x['code'] == 'OK' and x['rows'] == 1
            assert x['stats']['optimizer_version'] == '9'
            if mode == 'OFF':
                assert 'auto_parameters' not in x['stats']
            else:
                assert x['stats']['auto_parameters'] == {'_auto_p0_int64_':str(i)}
        for i,x in enumerate(select(rows,'result-' + mode),1):
            assert x['code'] == 'OK'
            assert {k.lower():v for k,v in x['rows'][0].items()} == {'id':str(i),'name':['Alice','Bob'][i-1]}
    assert all(x['code'] == 'InvalidArgument' for x in rows if x['label'].endswith('-INVALID'))
assert [x['code'] for x in select(g,'graph-uniqueness')] == ['FailedPrecondition','FailedPrecondition','OK']
assert select(g,'graph-match')[0]['rows'] == [{'n':'2'}]
for label in ['graph-metadata','semantic-metadata']:
    meta = json.loads(select(g,label)[0]['rows'][0]['PROPERTY_GRAPH_METADATA_JSON'])
    assert {x['name']:x['value'] for x in meta['options']} == {
        'validate_element_key_uniqueness':'TRUE' if label == 'semantic-metadata' else 'FALSE',
        'allow_view_with_volatile_column':'FALSE',
    }
assert select(g,'volatile-graph-0')[0]['code'] == select(g,'volatile-read-0')[0]['code'] == 'OK'
assert len(select(g,'volatile-read-0')[0]['rows']) == 2
for label in ['volatile-graph-1','volatile-graph-2']:
    x = select(g,label)[0]
    assert x['code'] == 'Unimplemented' and 'not supported' in x['error']
approx = select(g,'approx-version')
assert len(approx) == 9 and all(x['code'] == 'OK' and x['rows'] == [{'n':'2'}] for x in approx)
assert select(g,'cleanup')[0]['rows'] == []
assert all(x['code'] == 'OK' for x in g if x['label'].startswith('drop-') or x['label'] in ['volatile-drop-0','volatile-view-drop'])

matching = select(p,'ilike-isolated')
assert [x['rows'] for x in matching[:4]] == [[{'result':x}] for x in [True,False,True,False]]
assert all(x['code'] == 'InvalidArgument' and 'does not exist' in x['error'] for x in matching[4:])
samples = select(p,'tablesample')
assert [len(x['rows']) for x in samples[:2]] == [0,2]
assert all(x['code'] == 'OK' for x in samples[:2])
assert all(x['code'] == 'Unimplemented' for x in samples[2:])
assert [len(x['rows']) for x in select(p,'reservoir')] == [0,1,2]
assert select(p,'drop-database')[0]['code'] == 'OK'

queue = {x.get('label'):x for x in q}
for label in ['create','enqueue-immediate-and-future','receive-immediate','ack-commit','enqueue-default-delivery','cleanup','cleanup-confirmation']:
    assert queue[label]['code'] == 'OK',label
assert queue['after-rollback']['rows'] == []
assert queue['ack-commit']['rows_modified'] == [1]
assert queue['duplicate-ack']['code'] == 'OutOfRange'
assert queue['future-not-received']['rows'] == []
assert queue['future-still-stored']['rows'] == [{'MessageId':'future'}]
assert queue['cleanup-confirmation']['rows'] == [{'remaining':'0'}]
assert queue['stale-ordinary']['code'] == 'OK' and len(queue['stale-ordinary']['rows']) == 2
assert queue['stale-receive']['code'] == 'InvalidArgument' and 'strong' in queue['stale-receive']['error']
for label in ['control-array-default','control-array-off']:
    assert queue[label]['code'] == 'OK' and len(queue[label]['rows']) == 1
for label in ['control-scalar-default','control-scalar-off','invalid-renew-token']:
    assert queue[label]['code'] == 'InvalidArgument' and 'invalid lease token' in queue[label]['error']
receiver = queue['receive-immediate']['rows'][0]['SpannerLeaseToken']
default_renewal = queue['control-array-default']['rows'][0]
assert default_renewal['SpannerOldLeaseToken'] == receiver
assert queue['control-array-off']['rows'][0]['SpannerOldLeaseToken'] == default_renewal['SpannerNewLeaseToken']
for x in q:
    for row in x.get('rows',[]) if isinstance(x.get('rows'),list) else []:
        for key,value in row.items():
            if 'Token' in key and value is not None:
                assert value.startswith('sha256:') and len(value) == 71

distribution = [x for x in r if x.get('label','').startswith('distribution-')]
assert len(distribution) == 48 and all(x['code'] == 'OK' for x in distribution)
quantified = [x for x in r if x.get('label') in ['comparison','like']]
assert len(quantified) == 12 and all(x['code'] == 'InvalidArgument' for x in quantified)
assert select(r,'approx-count-distinct')[0]['rows'] == [{'n':'2'}]
nullable = [x for x in r if x.get('label','').startswith('txn-nullability-')]
assert len(nullable) == 6 and all(x['code'] == 'OK' for x in nullable)
null_rows = [row for x in nullable for row in x['rows']]
assert len(null_rows) == 5
assert all(x['AVG_PARTICIPANTS'] is None and x['AVG_TOTAL_LATENCY_SECONDS'] is None and x['COMMIT_ATTEMPT_COUNT'] == '0' for x in null_rows)
assert select(r,'active-batch-operations')[0]['code'] == 'Internal'
for i in [1,2,3]:
    assert select(a,'active-star-' + str(i))[0]['code'] == 'Internal'
assert select(a,'active-limit-zero')[0]['code'] == 'OK'
assert select(a,'active-count')[0]['code'] == 'Internal'
assert select(a,'existing-active-pdml-control')[0]['code'] == 'OK'
assert 'Unrecognized name' in select(a,'unknown-column-control')[0]['error']
columns = [x for x in a if x['label'].startswith('active-column-')]
assert len(columns) == 11 and all(x['code'] in ['Internal','InvalidArgument'] for x in columns)
assert all('Table not found' in x['error'] for x in columns if x['code'] == 'InvalidArgument')

checks = [json.loads((root/(name + '-check.json')).read_text()) for name in ['spanner-sys','spanner-sys-2','spanner-sys-3']]
for x in checks:
    assert x['status'] == 'surface_changed' and x['material_change']
    assert x['current']['column_count'] == 598
    assert x['current']['surface_sha256'] == '9bb70c640b69e4929e4b44ed834d69e1dfcd5fc17cc53e017f541b9f58f9ec8f'
    assert len(x['surface_delta']['added_columns']) == 11
    assert {c['table_name'] for c in x['surface_delta']['added_columns']} == {'ACTIVE_BATCH_OPERATIONS'}
    assert not x['surface_delta'].get('removed_columns') and not x['surface_delta'].get('changed_columns')
info = json.loads((root/'infoschema-check.json').read_text())
assert not info['material_change'] and info['current']['column_count'] == 310
assert all(x['code'] == 'OK' and x['rows'] == [] for x in cleanup[:3]) and cleanup[3]['code'] == 'NotFound'
log = (root/'managed-tests.log').read_text()
assert re.findall(r'--- FAIL: (Test\S+)',log) == ['TestDrift_RealTableMetas']
assert log.count('--- PASS: Test') == 3 and '--- SKIP:' not in log
assert 'TXN_STATS_TOP_10MINUTE' in log and 'cannot support NULL SQL values' in log
commands = json.loads((root/'commands.json').read_text())
assert {x['name'] for x in commands if x['exit_code']} == {'spanner-sys','spanner-sys-2','spanner-sys-3','managed-tests'}
active_commands = json.loads((root/'active-batch-commands.json').read_text())
assert active_commands[-1]['name'] == 'active-batch-fixed' and active_commands[-1]['exit_code'] == 0
assert active_commands[0]['exit_code'] == 1 and 'imported' in (root/'active-batch.stderr').read_text()
for rows in [g,p,q,r,a,cleanup]:
    assert not re.search(r'requestID = "(?!<redacted>)[^"]*"',json.dumps(rows))

manifest_path = root.parent / 'artifact-sha256.json'
if manifest_path.exists():
    hashes = json.loads(manifest_path.read_text())['files']
    for path in root.iterdir():
        if path.is_file():
            assert hashlib.sha256(path.read_bytes()).hexdigest() == hashes[str(path.relative_to(root.parent))],path.name
print('PASS: changed catalog and graph observations, inconclusive active scans, NULL evidence, stable controls, retained failures, and owned cleanup')
