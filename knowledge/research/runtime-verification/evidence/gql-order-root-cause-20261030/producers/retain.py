from pathlib import Path
import json,hashlib,shutil,datetime,re
r=Path('<LOCAL_CHECKOUT>');s=r/'.tmp/gql-order-root-cause-20261030';d=Path('<LOCAL_TASK_ROOT>/apstndb/spanalyzer/gql-order-root-cause-20261030');e=r/'knowledge/research/runtime-verification/evidence/gql-order-root-cause-20261030';e.mkdir(exist_ok=True)
for folder in ['sources','evidence']:
 for f in(d/folder).iterdir():
  if f.is_file():dest=e/('sources'if folder=='sources'else'captures')/f.name;dest.parent.mkdir(exist_ok=True);shutil.copyfile(f,dest)
p=e/'producers';p.mkdir(exist_ok=True)
for name in ['fetch-sources.py','fetch-googlesql.py','fetch-head.py','extract-binary.py','patch-binaries.py','build-images.py','prepare-probes.py','run.py','cleanup-images.py','retain.py']:shutil.copyfile(s/name,p/name)
for name in ['go.mod','go.sum','probe.go','matrix.json']:shutil.copyfile(s/'module'/name,p/(name+'.txt'if name=='probe.go'else name))
shutil.copyfile(s/'schema.sql',p/'schema.sql')
for variant in ['original-copy','no-scramble','graph-ordered']:shutil.copyfile(s/'images'/variant/'Dockerfile',p/('Dockerfile.'+variant))
sha=lambda f:hashlib.sha256(f.read_bytes()).hexdigest();checks=[]
def check(name,value):
 checks.append({'name':name,'pass':bool(value)})
 if not value:raise AssertionError(name)
matrix=json.loads((p/'matrix.json').read_text())['steps'];query_steps=[x for x in matrix if x['kind']=='query'];ordered=[x for x in query_steps if x['label'].startswith('gql-')and x['label']!='gql-unspecified'or x['label'].startswith(('sql-value-','sql-outer-gql-'))];check('30 planned steps, 27 queries and 22 explicit-order contracts',len(matrix)==30 and len(query_steps)==27 and len(ordered)==22)
variants=['official-before','original-copy','no-scramble','graph-ordered','official-after'];by={};summary=[]
def expected(x):
 name=x['label'];v=list(range(1,6))if name.startswith(('gql-id-','gql-string-'))else list(range(11,16))if name.startswith('gql-expression-')else[10,20,30,40,50]
 if 'DESC'in name:v=v[::-1]
 if 'top3'in name:v=v[:3]
 return list(map(str,v))
def scramble(values):n=len(values);h=n//2;return[values[2*i+1]if i<h else values[2*(i-h)]for i in range(n)]
for name in variants:
 rows=[json.loads(x)for x in(e/'captures'/(name+'.stdout')).read_text().splitlines()];by[name]={x['label']:x for x in rows};labels=by[name];check(name+' complete unique matrix and 36 records',len(rows)==len(labels)==36 and all(x['label']in labels for x in matrix));passes=0
 for x in matrix:
  a=labels[x['label']];wanted='Internal'if name=='graph-ordered'and x['label']=='gql-unspecified'else'OK';check(name+' code '+x['label'],a['code']==wanted)
 for x in ordered:
  a=labels[x['label']];values=[row['value']for row in a['rows']];want=expected(x);gql=x['label'].startswith('gql-');diagnostic=name in ['no-scramble','graph-ordered'];check(name+' predicted sequence '+x['label'],values==(scramble(want)if gql and not diagnostic else want));passes+=values==want
 for x in query_steps:
  if x in ordered or x['label']=='sql-scalar-control'or name=='graph-ordered'and x['label']=='gql-unspecified':continue
  check(name+' unspecified order multiset '+x['label'],sorted(row['value']for row in labels[x['label']]['rows'])==sorted(map(str,[10,20,30,40,50])))
 check(name+' scalar',labels['sql-scalar-control']['rows']==[{'value':'7'}]);check(name+' database lifecycle',labels['create-database']['code']=='OK'and labels['drop-database']['code']=='OK'and labels['database-absent']['code']=='NotFound'and labels['runtime-cleanup']['code']=='OK')
 active=json.loads((e/'captures'/(name+'-active-containers.json')).read_text());cleanup=json.loads((e/'captures'/(name+'-container-cleanup.json')).read_text());check(name+' exact owned container absence',len(active)==len(cleanup['owned'])==1 and active[0]['id']==cleanup['owned'][0]['id']and cleanup['owned'][0]['absent']and cleanup['preexisting_containers_preserved']);summary.append({'variant':name,'planned_steps':30,'queries':27,'explicit_order_correct':passes,'explicit_order_total':22,'gql_order_correct':sum([row['value']for row in labels[x['label']]['rows']]==expected(x)for x in ordered if x['label'].startswith('gql-')),'gql_order_total':18,'non_OK_steps':[{'label':x['label'],'code':labels[x['label']]['code']}for x in matrix if labels[x['label']]['code']!='OK']})
for name in ['original-copy','official-after']:
 check(name+' all query values and codes reproduce official-before',all((by[name][x['label']]['code'],by[name][x['label']].get('rows'))==(by['official-before'][x['label']]['code'],by['official-before'][x['label']].get('rows'))for x in query_steps))
check('local graph flag intervention preserves SQL unspecified-order scrambling',by['graph-ordered']['sql-unspecified']['rows']==by['official-before']['sql-unspecified']['rows']);check('global scrambling disable changes SQL unspecified output',by['no-scramble']['sql-unspecified']['rows']!=by['official-before']['sql-unspecified']['rows']);check('SQL scramble matches source permutation',[x['value']for x in by['official-before']['sql-unspecified']['rows']]==scramble([x['value']for x in by['no-scramble']['sql-unspecified']['rows']]))
for row in json.loads((e/'captures/binary-interventions.json').read_text()):
 f=s/'images'/row['variant']/'emulator_main';check(row['variant']+' actual binary hash',sha(f)==row['variant_binary_sha256']);check(row['variant']+' exactly bounded changed bytes',len(row['changed_byte_offsets'])=={'original-copy':0,'no-scramble':1,'graph-ordered':4}[row['variant']])
check('sixth extraction container removed',json.loads((e/'captures/extract-container.json').read_text())['absent']);cleanup=json.loads((e/'captures/diagnostic-image-cleanup.json').read_text());check('three diagnostic image tags removed and official image retained',len(cleanup['owned'])==3 and all(x['tag_absent']and x['remove_exit_code']==0 for x in cleanup['owned'])and cleanup['official_base_preserved'])
base=json.loads((s/'initial-artifact-manifest.json').read_text())['files'];check('all 1491 preexisting evidence bytes unchanged',len(base)==1491 and all(sha(e.parent/k)==v for k,v in base.items()));source=json.loads((e.parent/'managed-recheck-20261029/source-hashes.json').read_text());check('all 123 measured implementation/dependency bytes unchanged',len(source)==123 and all(sha(r/k)==v for k,v in source.items()))
for name in ['gql','gql-limit','sql','gql-outer-sql']:
 text=(e/'captures'/(name+'-resolved-ast.txt')).read_text();check(name+' local AST contains ordered sort',bool(text)and'is_ordered=TRUE'in text)
(e/'assertions.json').write_text(json.dumps({'scope':'Offline assertions of retained controlled runtime evidence; not a new runtime execution.','checks':checks,'passed':len(checks),'failed':0},indent=2)+'\n')
(e/'summary.json').write_text(json.dumps({'observed_at':'2026-10-30T09:37:53Z/2026-10-30T09:38:15Z','runtime':'Emulator 1.5.59 linux/arm64','emulator_source':'d357ec1d2b08ae92ce1da4c5078de82fda0c07ab','googlesql_tag':'2026.7.2','googlesql_source':'0e7d7073ed0360be587a5efa0fa78abeee00f17b','current_googlesql_source_spot_check':'a09b0bd95dd55dba9478eb8f20a3971bc9e30b70','conditions':summary,'total_planned_steps':150,'total_queries':135,'total_root_records':180,'cleanup':{'databases':5,'runtime_containers':5,'extraction_containers':1,'diagnostic_image_tags':3,'verified_absent':True},'source_binary_boundary':'Official image and source tag are independently pinned; matching disassembly and causal interventions are retained. No source-to-image reproducible-build attestation or full source-rebuilt fix is claimed.','unverified':['first affected version','amd64 runtime','general source fix','managed or Omni internals','current GoogleSQL HEAD runtime']},indent=2)+'\n')
(e/'validation.json').write_text(json.dumps({'status':'pending','commands':[],'note':'Evidence checks complete; module and documentation checks have not run yet.'},indent=2)+'\n');print(json.dumps({'checks':len(checks),'summary':summary}))
