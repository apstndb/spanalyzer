from pathlib import Path
import json,collections,hashlib,shutil,datetime
r=Path('<LOCAL_CHECKOUT>');task=Path('<LOCAL_TASK_ROOT>/apstndb/spanalyzer/gql-order-root-cause-20261030/query-shapes');checks=[]
def check(name,value):
 checks.append({'name':name,'pass':bool(value)})
 if not value:raise AssertionError(name)
def perm(rows):h=len(rows)//2;return[rows[2*i+1]if i<h else rows[2*(i-h)]for i in range(len(rows))]
def signature(rows):return sorted(json.dumps(x,sort_keys=True)for x in rows)
data={};count=0;roots=0
for name,source in [('query-shapes','matrix'),('cardinality','cardinality')]:
 rows=[json.loads(x)for x in(task/'evidence'/(name+'.stdout')).read_text().splitlines()];d={x['label']:x for x in rows};steps=json.loads((task/'sources'/(source+'.json')).read_text())['steps'];check(name+' unique complete planned records',len(rows)==len(d)==len(steps)+6);roots+=len(rows)
 for step in steps:check(name+' code '+step['label'],d[step['label']]['code']=='OK')
 for label,code in [('create-database','OK'),('drop-database','OK'),('database-absent','NotFound'),('runtime-cleanup','OK')]:check(name+' lifecycle '+label,d[label]['code']==code)
 active=json.loads((task/'evidence'/(name+'-active-containers.json')).read_text());clean=json.loads((task/'evidence'/(name+'-container-cleanup.json')).read_text());check(name+' container absent',len(active)==len(clean['owned'])==1 and active[0]['id']==clean['owned'][0]['id']and clean['owned'][0]['absent']and clean['preexisting_containers_preserved']);data[name]=d;count+=sum(x['kind']=='query'for x in steps)
main=data['query-shapes']
for direction in ['ASC','DESC']:
 for name in ['id','name','country','timestamp','arithmetic','string-cast']:
  label='sql-'+name+'-'+direction;values=[x['v']for x in main[label]['rows']];key=(lambda v:int(v))if name in ['id','arithmetic']else(lambda v:v);check(label+' SQL order',values==sorted(values,key=key,reverse=direction=='DESC'))
 label='sql-aggregate-'+direction;values=[x['total']for x in main[label]['rows']];check(label+' SQL order',values==sorted(values,reverse=direction=='DESC'))
check('SQL outer graph order',main['sql-outer-graph']['rows']==[{'v':str(x)}for x in [1,2,3]])
transfers=[{'amount':amount,'src':str(src),'dst':str(dst)}for src,dst,amount in [(7,16,300),(7,16,100),(16,20,300),(20,7,500),(20,16,200)]]
for direction in ['ASC','DESC']:
 ordered=sorted(transfers,key=lambda x:(x['amount'],int(x['src']),int(x['dst'])),reverse=direction=='DESC')
 for n in [2,3,4,5]:label='sql-'+direction+'-'+str(n);check(label+' independent fixture expectation',data['cardinality'][label]['rows']==ordered[:n])
manifest=r/'knowledge/research/runtime-verification/evidence/artifact-sha256.json';old=json.loads(manifest.read_text())['files'];check('all 1594 preexisting artifact hashes preserved',len(old)==1594 and all(hashlib.sha256((manifest.parent/k).read_bytes()).hexdigest()==v for k,v in old.items()));source=json.loads((manifest.parent/'managed-recheck-20261029/source-hashes.json').read_text());check('all 123 measured implementation files unchanged',len(source)==123 and all(hashlib.sha256((r/k).read_bytes()).hexdigest()==v for k,v in source.items()))
items=[];main=data['query-shapes']
for spec in json.loads((task/'sources/case-contracts.json').read_text()):
 label=spec['label'];kind=spec['kind'];a=main[label];rows=a['rows'];wanted=main[spec['oracle']]['rows']if spec.get('oracle')else[{'v':x}for x in spec['expected']]if spec.get('expected')and kind not in ['array-order','encounter-order-observation','ordered-ids']else None
 row={'label':label,'kind':kind,'actual':rows}
 if kind in ['ordered-rows','ordered-rows-and-selection']:
  check(label+' predicted final permutation',rows==perm(wanted));check(label+' incorrect unique-key order',rows!=wanted);check(label+' membership',signature(rows)==signature(wanted));row.update(expected=wanted,ordered=False)
 elif kind in ['selection','selection-only','sql-outer-control']:
  check(label+' selection',signature(rows)==signature(wanted));row.update(expected=wanted,ordered=rows==wanted)
 elif kind in ['array-order','encounter-order-observation']:
  check(label+' array elements',rows==[{'ids':spec['expected']}]);row.update(expected=[{'ids':spec['expected']}],ordered=True,contract='Explicit aggregate ORDER BY'if kind=='array-order'else'Observed encounter order only; no explicit aggregate ORDER BY')
 elif kind=='ordered-ids':
  ids=[x['id']for x in rows];check(label+' predicted total-key permutation',ids==perm(spec['expected']));check(label+' total-key order wrong',ids!=spec['expected']);row.update(ordered=False)
 elif kind=='tie-keys':check(label+' tied keys valid despite swapped IDs',[x['k']for x in rows]==[False,False,True]);row.update(ordered=True,contract='Only k is ordered; swapped equal-key IDs do not violate that clause')
 items.append(row)
card=data['cardinality']
for spec in json.loads((task/'sources/cardinality-contracts.json').read_text()):
 label=spec['label'];kind=spec['kind'];rows=card[label]['rows'];wanted=card[spec['oracle']]['rows']if spec.get('oracle')else[{'v':str(x)}for x in range(1,6)];row={'label':label,'kind':kind,'actual':rows,'expected':wanted}
 if kind in ['primitive','final','for-five']:
  primitive=kind=='primitive'or label=='for-primitive-five';pred=perm(perm(wanted))if primitive else perm(wanted);check(label+' predicted single/double permutation',rows==pred);check(label+' selected membership',signature(rows)==signature(wanted));row.update(ordered=rows==wanted,permutations=2 if primitive else 1)
 items.append(row)
final=[x for x in items if x['kind']in ['ordered-rows','ordered-rows-and-selection','ordered-ids','final']or x['label']=='for-final-five'];check('32 final total-key cases all fail ordering',len(final)==32 and all(x['ordered']is False for x in final));check('73 query attempts in two runtime matrices',count==73);check('89 root step records retained',roots==89)
summary={'runtime':'Emulator 1.5.59 linux/arm64 manifest 969af6a9349948fbfb4a826c18e160e258a4005f7c759f8e7f2dcec369cccba8','queries':count,'planned_steps':77,'root_records':roots,'query_errors':0,'final_unique_key_order':{'correct':0,'total':32},'cases':items,'cleanup':{'databases_confirmed_not_found':2,'runtime_containers_confirmed_absent':2},'local_AST_boundary':'Initial full official schema analysis failed because the local catalog exposed no implicit Person properties. Both exact GQL queries analyzed with an explicit reduced Person(id) graph; these are local frontend dumps, not Emulator AST output. No implementation repair made.','scope':'Current pinned Emulator sample only; no managed or Omni request; array argument ordering, final row ordering, top-N membership and tied keys have distinct contracts.'};(task/'summary.json').write_text(json.dumps(summary,indent=2)+'\n');(task/'assertions.json').write_text(json.dumps({'passed':len(checks),'failed':0,'checks':checks},indent=2)+'\n');shutil.copyfile(Path(__file__),task/'sources/analyze.py');shutil.copyfile(r/'.tmp/gql-order-root-cause-20261030/prepare-patterns.py',task/'sources/prepare-patterns.py');print(json.dumps({'queries':count,'final_total_key_cases':len(final),'checks':len(checks)}))
e=r/'knowledge/research/runtime-verification/evidence/gql-order-root-cause-20261030/query-shapes';e.mkdir(exist_ok=True)
for folder in ['sources','evidence']:
 dest=e/folder;dest.mkdir(exist_ok=True)
 for f in(task/folder).iterdir():
  if f.is_file():shutil.copyfile(f,dest/f.name)
for f in ['summary.json','assertions.json']:shutil.copyfile(task/f,e/f)
