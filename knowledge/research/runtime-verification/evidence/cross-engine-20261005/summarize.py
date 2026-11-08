from pathlib import Path
import json,hashlib,collections,sys
p=Path(sys.argv[1]) if len(sys.argv)>1 else Path('knowledge/research/runtime-verification/evidence/cross-engine-20261005')
raw={f.name:[json.loads(x) for x in f.read_text().splitlines()] for f in p.glob('*.jsonl')}
def selected(name,label):return [r for r in raw[name] if r.get('label')==label]
def one(name,label):
 rs=selected(name,label);assert len(rs)==1,(name,label,len(rs));return rs[0]
def tuples(rows):return sorted((r['TABLE_NAME'],r['COLUMN_NAME'],r['SPANNER_TYPE'],int(r['ORDINAL_POSITION'])) for r in rows)
summary={'scope':'Bounded October 5 samples; expected rejections are retained, not counted as feature successes','records':{},'catalogs':{},'feature_results':{},'cleanup':{}}
for name,rs in sorted(raw.items()):
 assert rs,(name,'empty capture')
 summary['records'][name]={'count':len(rs),'codes':dict(collections.Counter(r.get('code','no-code') for r in rs)),'first_at':rs[0]['at'],'last_at':rs[-1]['at']}
 assert 'requestID' not in json.dumps(rs),name
 for r in rs:
  for row in r.get('rows',[]) if isinstance(r.get('rows'),list) else []:
   if isinstance(row,dict):
    for k,v in row.items():
     if 'Token' in k and v is not None:assert v.startswith('sha256:'),(name,k)
for engine,f in [('managed','managed-catalog.jsonl'),('omni','omni-feature-probe.jsonl'),('emulator','emulator-feature-probe.jsonl')]:
 summary['catalogs'][engine]={}
 for schema in ['information-schema','spanner-sys']:
  label=schema+'-surface' if engine!='managed' else schema.upper().replace('-','_')+'-surface'
  r=one(f,label);assert r['code']=='OK';ts=tuples(r['rows']);summary['catalogs'][engine][schema]={'tables':len(set(x[0] for x in ts)),'columns':len(ts),'tuple_sha256':hashlib.sha256(json.dumps(ts,separators=(',',':')).encode()).hexdigest()}
  summary['catalogs'][engine][schema]['tuples']=ts
for schema in ['information-schema','spanner-sys']:
 a=set(summary['catalogs']['managed'][schema]['tuples']);b=set(summary['catalogs']['omni'][schema]['tuples'])
 summary['catalogs'][schema+'-managed-vs-omni']={'managed_only':sorted(a-b),'omni_only':sorted(b-a)}
for engine in ['managed','omni','emulator']:
 f='managed-google-features.jsonl' if engine=='managed' else engine+'-feature-probe.jsonl'
 rs=raw[f];fr={}
 fr['auto_parameter']=[{'label':r['label'],'code':r['code'],'auto_parameters':(r.get('stats') or {}).get('auto_parameters'),'plan_nodes':len((r.get('plan') or {}).get('planNodes',[]))} for r in rs if r.get('kind')=='plan-probe' and r.get('label','').startswith('auto-parameter')]
 fr['ddl']=[{k:r[k] for k in ['label','code','error','sql']} for r in rs if r.get('kind')=='ddl']
 fr['graph_metadata_options']=[json.loads(row['PROPERTY_GRAPH_METADATA_JSON']).get('options',[]) for r in rs if r.get('label') in ['graph-metadata','semantic-metadata'] for row in r['rows']]
 fr['approx']=[{k:r.get(k) for k in ['label','sql','code','rows','error']} for r in rs if r.get('kind')=='query' and r.get('label','').startswith('approx')]
 pg=raw['managed-pg-features.jsonl'] if engine=='managed' else rs
 fr['postgresql']=[{k:r.get(k) for k in ['label','sql','code','rows','error']} for r in pg if r.get('label') in ['ilike-isolated','tablesample','reservoir']]
 qr=raw['managed-readonly.jsonl'] if engine=='managed' else rs
 fr['quantifiers']=[{k:r.get(k) for k in ['label','sql','code','error']} for r in qr if r.get('label') in ['comparison','like','quantified-comparison','quantified-like']]
 fr['active_batch']=next({k:r.get(k) for k in ['label','code','rows','error']} for r in qr if r.get('label')=='active-batch-operations')
 summary['feature_results'][engine]=fr
summary['queue_controls']={}
for engine,f in [('managed','managed-queue.jsonl'),('omni','omni-queue-control.jsonl')]:
 out={}
 for label in ['control-scalar-default','control-scalar-off','control-array-default','control-array-off']:
  r=one(f,label);out[label]={k:r.get(k) for k in ['code','rows','error']}
  assert r['code']==('InvalidArgument' if 'scalar' in label else 'OK')
  if 'array' in label:assert len(r['rows'])==1
 received=one(f,'receive-immediate' if engine=='managed' else 'control-receive')['rows'][0]['SpannerLeaseToken']
 assert one(f,'control-array-default')['rows'][0]['SpannerOldLeaseToken']==received
 assert one(f,'control-array-off')['rows'][0]['SpannerOldLeaseToken']==one(f,'control-array-default')['rows'][0]['SpannerNewLeaseToken']
 out['receiver_token_matches_successful_array_renewal']=True
 summary['queue_controls'][engine]=out
for engine in ['omni','emulator']:
 for name,rs in raw.items():
  if not name.startswith(engine+'-'):continue
  for r in rs:
   if r.get('label') in ['database-cleanup','runtime-cleanup']:assert r['code']=='OK',(name,r)
 images=json.loads((p/'images.json').read_text());assert images[engine]['remaining_containers_for_tested_image']==[]
 summary['cleanup'][engine]='Database and runtime cleanup records OK; no container remains for tested image'
for label in ['remaining-tables','remaining-graphs','remaining-queue-routines']:
 r=one('managed-cleanup.jsonl',label);assert r['code']=='OK' and not r['rows']
assert one('managed-cleanup.jsonl','temporary-pg-absent')['code']=='NotFound'
summary['cleanup']['managed']='Owned GoogleSQL fixture scans empty; temporary PostgreSQL database NotFound'
# Check the distinctions reported in the comparison, not just process exit codes.
assert summary['catalogs']['information-schema-managed-vs-omni']=={'managed_only':[],'omni_only':[]}
delta=summary['catalogs']['spanner-sys-managed-vs-omni'];assert not delta['managed_only'] and len(delta['omni_only'])==11
assert all(t[0]=='ACTIVE_BATCH_OPERATIONS' for t in delta['omni_only'])
for engine,fr in summary['feature_results'].items():
 assert len(fr['quantifiers'])==12 and all(r['code']=='InvalidArgument' and 'not supported' in r['error'] for r in fr['quantifiers'])
 ilike=[r for r in fr['postgresql'] if r['label']=='ilike-isolated'];assert len(ilike)==8
 assert [r['rows'][0]['result'] for r in ilike[:4]]==[True,False,True,False]
 assert all(r['code']=='InvalidArgument' and 'does not exist' in r['error'] for r in ilike[4:])
 reservoir=[r for r in fr['postgresql'] if r['label']=='reservoir'];assert [len(r['rows']) for r in reservoir]==[0,1,2]
 sample=[r for r in fr['postgresql'] if r['label']=='tablesample'];assert [len(r['rows']) for r in sample[:2]]==[0,2]
 assert all(r['code']=='Unimplemented' for r in sample[2:])
 for r in fr['auto_parameter']:
  mode=r['label'].removeprefix('auto-parameter-')
  if engine=='emulator':
   assert r['code']==('OK' if mode=='DEFAULT' else 'InvalidArgument') and not r['auto_parameters'] and r['plan_nodes']==0
  else:
   assert r['code']==('InvalidArgument' if mode=='INVALID' else 'OK')
   assert bool(r['auto_parameters'])==(mode in ['DEFAULT','AUTO','ON'])
 expected=['OK','OK','OK'] if engine=='emulator' else ['FailedPrecondition','FailedPrecondition','OK']
 assert [r['code'] for r in fr['ddl'] if r['label']=='graph-uniqueness']==expected
 if engine!='emulator':
  approx=[r for r in fr['approx'] if 'OPTIMIZER_VERSION' in r['sql']];assert len(approx)==9 and all(r['code']=='OK' and r['rows']==[{'n':'2'}] for r in approx)
 else:
  approx=[r for r in fr['approx'] if 'OPTIMIZER_VERSION' in r['sql']];assert len(approx)==9 and all(r['code']=='InvalidArgument' and 'Unsupported hint' in r['error'] for r in approx)
  assert next(r for r in fr['approx'] if r['label']=='approx-null-duplicate-control')['code']=='Unimplemented'
for engine in ['omni','emulator']:
 f=engine+'-volatile-probe.jsonl';expected=['OK','OK','OK'] if engine=='emulator' else ['OK','Unimplemented','Unimplemented']
 assert [one(f,'graph-'+str(i))['code'] for i in range(3)]==expected
f='emulator-option-control.jsonl'
for i in range(2):
 assert one(f,'graph-'+str(i))['code']=='OK'
 assert one(f,'graph-read-'+str(i))['code']=='OK'
 assert 'options' not in json.loads(one(f,'graph-options-metadata-'+str(i))['rows'][0]['PROPERTY_GRAPH_METADATA_JSON'])
 assert all('OPTIONS' not in stmt for stmt in one(f,'graph-options-export-'+str(i))['statements'])
assert [r['rows'] for r in selected('emulator-feature-probe.jsonl','graph-match')]==[[{'n':'2'}]]*3
assert one('emulator-queue-probe.jsonl','create')['code']=='InvalidArgument'
assert 'Syntax error' in one('emulator-queue-probe.jsonl','create')['error']
assert one('emulator-feature-probe.jsonl','optimizer-versions')['rows']==[{'IS_DEFAULT':True,'RELEASE_DATE':'2023-09-19','VERSION':'42'}]
summary['assertions']='Catalog deltas, parameter stats, quantified errors, matching/sampling rows, graph option outcomes, receiver-token continuity, and cleanup verified from retained records'
(p/'summary.json').write_text(json.dumps(summary,indent=2)+'\n')
print(json.dumps({'records':{k:v['count'] for k,v in summary['records'].items()},'catalogs':{k:{s:{a:b for a,b in v.items() if a!='tuples'} for s,v in es.items()} for k,es in summary['catalogs'].items() if k in ['managed','omni','emulator']},'cleanup':summary['cleanup']},indent=2))
