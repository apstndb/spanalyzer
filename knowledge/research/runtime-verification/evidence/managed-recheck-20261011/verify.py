"""Offline assertion checks for the dated sample; no service calls."""
from pathlib import Path
import hashlib, json, re, sys

ev = Path(sys.argv[1]) if len(sys.argv)>1 else Path(__file__).resolve().parent
load = lambda name:[json.loads(x) for x in (ev/(name+'.jsonl')).read_text().splitlines()]
pick = lambda rows,label:next(x for x in rows if x.get('label')==label)
records = {name:load(name) for name in ['readonly','google-features','pg-features','queue','cleanup','active-batch','queries','followup','distinctness','google-json','pg-streams','graph-controls','google-options','pg-options','google-capture','pg-capture']}
assert [len(records[x]) for x in ['readonly','google-features','pg-features','queue','active-batch','cleanup','queries','followup','distinctness']]==[86,53,40,32,19,4,115,10,64]
commands=json.loads((ev/'all-commands.json').read_text())
assert {x['name'] for x in commands if x['exit_code']}=={'managed-tests'}
log=(ev/'managed-tests.log').read_text()
assert re.findall(r'--- FAIL: (Test\S+)',log)==['TestDrift_RealTableMetas']
assert log.count('--- PASS: Test')==3 and '--- SKIP:' not in log
assert 'unknown_tables=[ACTIVE_BATCH_OPERATIONS]' in log
assert 'canonical_total=330' in log and 'generated_total=330' in log
for name in ['spanner-sys','spanner-sys-2','spanner-sys-3']:
    check=json.loads((ev/(name+'-check.json')).read_text())
    assert not check['material_change'] and check['current']['column_count']==598
    assert check['current']['surface_sha256']=='9bb70c640b69e4929e4b44ed834d69e1dfcd5fc17cc53e017f541b9f58f9ec8f'
info=json.loads((ev/'infoschema-check.json').read_text())
assert not info['material_change'] and info['current']['column_count']==310
source=json.loads((ev/'source-comparison.json').read_text())
assert source['measured_files']==123 and source['changed_files']==[]
for name,comparison in json.loads((ev/'feature-comparison.json').read_text()).items():
    assert comparison['comparable_records']==comparison['baseline_records']==comparison['current_records'],name
    assert not comparison['changed_codes'],name
for name,comparison in json.loads((ev/'json-comparison.json').read_text()).items():
    assert comparison['comparable_records']==comparison['baseline_records']==comparison['current_records'],name
    assert not comparison['changed_codes'],name
    assert not comparison['changed_values'] and not comparison['changed_row_types'],name
active=records['active-batch']
for row in active:
    if row['label'].startswith('active-star-') or row['label'].startswith('active-column-'):
        assert row['code']=='OK' and row['rows']==[]
assert pick(active,'active-count')['rows']==[{'n':'0'}]
assert pick(active,'unknown-column-control')['code']=='InvalidArgument'
readonly=records['readonly'];dist=[x for x in readonly if x.get('label','').startswith('distribution-')]
assert len(dist)==48 and all(x['code']=='OK' for x in dist)
nullable=[r for x in readonly if x.get('label','').startswith('txn-nullability-') for r in x['rows']]
assert nullable and all(x['AVG_PARTICIPANTS'] is None and x['AVG_TOTAL_LATENCY_SECONDS'] is None and x['COMMIT_ATTEMPT_COUNT']=='0' for x in nullable)
assert [x['code'] for x in records['cleanup']]==['OK','OK','OK','NotFound']
assert all(x['rows']==[] for x in records['cleanup'][:3])
queue=records['queue']
assert pick(queue,'after-rollback')['rows']==[]
assert pick(queue,'ack-commit')['rows_modified']==[1]
assert pick(queue,'duplicate-ack')['code']=='OutOfRange'
assert pick(queue,'future-not-received')['rows']==[]
assert pick(queue,'future-still-stored')['rows']==[{'MessageId':'future'}]
receiver=pick(queue,'receive-immediate')['rows'][0]['SpannerLeaseToken']
renewal=pick(queue,'control-array-default')['rows'][0]
assert renewal['SpannerOldLeaseToken']==receiver
assert pick(queue,'control-array-off')['rows'][0]['SpannerOldLeaseToken']==renewal['SpannerNewLeaseToken']
for record in queue:
    if isinstance(record.get('rows'),list):
        for row in record['rows']:
            for key,value in row.items():
                if 'Token' in key and value is not None:assert value.startswith('sha256:') and len(value)==71
for label in ['control-array-default','control-array-off']:
    assert pick(queue,label)['code']=='OK' and len(pick(queue,label)['rows'])==1
for label in ['control-scalar-default','control-scalar-off','invalid-renew-token']:
    assert pick(queue,label)['code']=='InvalidArgument'
assert pick(queue,'cleanup-confirmation')['rows']==[{'remaining':'0'}]
g=records['google-features'];p=records['pg-features']
for label in ['graph-metadata','semantic-metadata']:
    metadata=json.loads(pick(g,label)['rows'][0]['PROPERTY_GRAPH_METADATA_JSON'])
    assert {x['name']:x['value'] for x in metadata['options']}=={'validate_element_key_uniqueness':'TRUE' if label=='semantic-metadata' else 'FALSE','allow_view_with_volatile_column':'FALSE'}
for label in ['volatile-graph-1','volatile-graph-2']:assert pick(g,label)['code']=='Unimplemented'
approx=[x for x in g if x.get('label')=='approx-version']
assert len(approx)==9 and all(x['code']=='OK' and x['rows']==[{'n':'2'}] for x in approx)
matching=[x for x in p if x.get('label')=='ilike-isolated']
assert [x['rows'] for x in matching[:4]]==[[{'result':x}] for x in [True,False,True,False]]
assert all(x['code']=='InvalidArgument' and 'does not exist' in x['error'] for x in matching[4:])
for name in ['google-json','pg-streams','graph-controls','google-options','pg-options','google-capture','pg-capture']:
    rows=records[name]
    assert pick(rows,'create-database')['code']==pick(rows,'setup')['code']==pick(rows,'drop-database')['code']=='OK',name
    assert pick(rows,'database-absent')['code']=='NotFound',name
for name in ['google-json','pg-streams']:
    for hb in [100,999,1000,300000]:assert pick(records[name],f'heartbeat-boundary-{hb}')['code']=='OK'
    for hb in [99,300001]:assert pick(records[name],f'heartbeat-boundary-{hb}')['code']=='InvalidArgument'
    assert pick(records[name],'heartbeat-child-100')['code']=='OK'
    assert len(pick(records[name],'heartbeat-child-100')['rows'])>=2
for name in ['google-options','pg-options']:
    assert len(records[name])==30
    assert pick(records[name],'old-for-all-control')['code']=='InvalidArgument'
    for setting in ['exclude_ttl_deletes=true','exclude_ttl_deletes=false','allow_txn_exclusion=true','allow_txn_exclusion=false']:
        assert pick(records[name],'option-only-'+setting)['code']=='OK'
for row in records['graph-controls']:
    if 'cast' in row['label'] and ('element' in row['label'] or 'path' in row['label']):assert row['code']=='Unimplemented'
    elif 'to-json' in row['label'] or 'property-cast' in row['label']:assert row['code']=='OK'
expected=json.loads((ev/'distinctness-expected.json').read_text())
for label,want in expected.items():
    row=pick(records['distinctness'],label)
    for k,v in want.items():assert row[k]==v,(label,k)
for row in records['distinctness']:
    if row['code']=='OK' and 'd' in row['rows'][0]:
        assert isinstance(row['rows'][0]['d'],bool) and row['rows'][0]['d'] is not row['rows'][0]['nd']
    if row['label'].endswith('-profile'):
        other=pick(records['distinctness'],row['label'].removesuffix('-profile'))
        assert row['code']==other['code'] and row['rows']==other['rows']
capture=json.loads((ev/'value-capture-summary.json').read_text())
for name,rows in capture.items():
    assert len(rows)==9,(name,len(rows))
    for mode in ['NEW_ROW_AND_OLD_VALUES','NEW_ROW','OLD_AND_NEW_VALUES']:
        found=[x for x in rows if x['value_capture_type']==mode]
        assert {x['mod_type'] for x in found}=={'INSERT','UPDATE','DELETE'},(name,mode)
        update=next(x for x in found if x['mod_type']=='UPDATE')
        assert len(update['mods'])==1
        mod=update['mods'][0]
        new={k.lower():v for k,v in mod['new_values'].items()}
        old={k.lower():v for k,v in mod['old_values'].items()}
        assert str(new['value'])=='20'
        if mode in ['NEW_ROW_AND_OLD_VALUES','NEW_ROW']:
            assert new=={'value':new['value'],'other':'stable','nullable':None}
        else:assert set(new)=={'value'}
        if mode=='NEW_ROW':assert old=={}
        else:assert set(old)=={'value'} and str(old['value'])=='10'
docs=json.loads((ev/'documentation.json').read_text())
assert docs['release_feed_latest']['title']=='September 18, 2026'
assert docs['release_html_first_dated_heading']=='September_18_2026'
google=(ev/'google-operators.md.txt').read_text().lower();pg=(ev/'pg-operators.md.txt').read_text().lower()
assert 'is distinct from' not in google and 'is distinct from' in pg
assert 'NEW_ROW_AND_OLD_VALUES' in (ev/'change-streams-manage.diff').read_text()
def strings(value):
    if isinstance(value,str):yield value
    elif isinstance(value,dict):
        for x in value.values():yield from strings(x)
    elif isinstance(value,list):
        for x in value:yield from strings(x)
for rows in records.values():
    for value in strings(rows):
        assert not re.search(r'requestID = "(?!<redacted>)[^"]*"',value)
        assert not re.search(r'projects/(?!<redacted>)[^/]+/instances/',value)
manifest=ev.parent/'artifact-sha256.json'
if manifest.exists():
    hashes=json.loads(manifest.read_text())['files']
    for path in ev.iterdir():
        if path.is_file():assert hashlib.sha256(path.read_bytes()).hexdigest()==hashes[str(path.relative_to(ev.parent))],path.name
print('PASS: unchanged repeated outcomes; successful bounded distinctness and capture-mode controls; retained managed drift failure; owned cleanup; evidence integrity')
