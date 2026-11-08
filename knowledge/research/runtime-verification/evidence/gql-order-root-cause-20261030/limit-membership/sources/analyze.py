from pathlib import Path
import json,hashlib,shutil,collections
r=Path('<LOCAL_CHECKOUT>');t=Path('<LOCAL_TASK_ROOT>/apstndb/spanalyzer/gql-order-root-cause-20261030/limit-membership');checks=[]
def check(name,v):
 checks.append({'name':name,'pass':bool(v)})
 if not v:raise AssertionError(name)
def perm(xs):h=len(xs)//2;return[xs[i*2+1]if i<h else xs[(i-h)*2]for i in range(len(xs))]
def sig(rows):return sorted(json.dumps(x,sort_keys=True)for x in rows)
rows=[json.loads(x)for x in(t/'evidence/limit-membership.stdout').read_text().splitlines()];d={x['label']:x for x in rows};m=json.loads((t/'sources/matrix.json').read_text());spec=json.loads((t/'sources/contracts.json').read_text());check('37 steps, 35 queries, 43 unique root records',len(m['steps'])==37 and len(spec)==35 and len(rows)==len(d)==43)
for step in m['steps']:check(step['label']+' code',d[step['label']]['code']=='OK')
choices=[];base=[{'src':'7','dst':'16','amount':100},{'src':'20','dst':'16','amount':200}];ties=[{'src':'7','dst':'16','amount':300},{'src':'16','dst':'20','amount':300}]
for a in spec:
 name=a['label'];kind=a['kind'];actual=d[name]['rows']
 if kind in ['person-tie','literal-tie']:
  check(name+' legal false-key choice',len(actual)==1 and actual[0]['id']in['1','2']and actual[0]['k']is False)
 elif kind=='person-total':check(name+' unique-key selection',actual==[{'id':'1','k':False}])
 elif kind=='transfer-tie':
  selected=[q for q in ties if sig(actual)==sig(base+[q])];check(name+' one valid tied boundary row',len(selected)==1);choices.append((selected[0]['src'],selected[0]['dst']));check(name+' correct sorted-key values then permutation',[x['amount']for x in actual]==[200,100,300])
 elif kind=='transfer-total':check(name+' exact complete-key top-N then permutation',actual==perm(base+[ties[0]]))
 elif kind=='unique-final':
  expected=[{'id':x}for x in a['want']];check(name+' membership',sig(actual)==sig(expected));check(name+' exact final permutation',actual==perm(expected))
 elif kind=='later-limit':check(name+' observed downstream unordered choice',actual==[{'id':'4'},{'id':'2'}])
 elif kind=='no-order':check(name+' legal unordered membership',len(actual)==2 and len({x['id']for x in actual})==2 and all(x['id']in['1','2','3']for x in actual))
check('identical partial-key transfer SQL across four repeats',len({d['transfer-tie-'+str(i)]['sql']for i in range(4)})==1);check('observed both 300-amount boundary identities',set(choices)=={('7','16'),('16','20')});check('literal source-order change demonstrates both legal choices',{d['literal-tie-'+str(i)]['rows'][0]['id']for i in range(3)}=={'1','2'})
for name,code in [('create-database','OK'),('drop-database','OK'),('database-absent','NotFound'),('runtime-cleanup','OK')]:check(name,d[name]['code']==code)
active=json.loads((t/'evidence/limit-membership-active-containers.json').read_text());clean=json.loads((t/'evidence/limit-membership-container-cleanup.json').read_text());check('owned container confirmed absent',len(active)==len(clean['owned'])==1 and active[0]['id']==clean['owned'][0]['id']and clean['owned'][0]['absent']and clean['preexisting_containers_preserved'])
root=r/'knowledge/research/runtime-verification/evidence';old=json.loads((root/'artifact-sha256.json').read_text())['files'];check('all 1632 preceding evidence hashes unchanged',len(old)==1632 and all(hashlib.sha256((root/k).read_bytes()).hexdigest()==v for k,v in old.items()));sources=json.loads((root/'managed-recheck-20261029/source-hashes.json').read_text());check('all 123 implementation/dependency hashes unchanged',len(sources)==123 and all(hashlib.sha256((r/k).read_bytes()).hexdigest()==v for k,v in sources.items()))
summary={'runtime':'Emulator 1.5.59 linux/arm64 969af6a9349948fbfb4a826c18e160e258a4005f7c759f8e7f2dcec369cccba8','queries':35,'planned_steps':37,'root_records':43,'query_errors':0,'tied_transfer_boundary_choices':choices,'tie_variation_observed_with_identical_query_and_unchanged_data':True,'unique_key_membership':'All tested direct and later re-ordered LIMIT/OFFSET cases match expected row multisets, followed by deterministic output permutation. This is bounded evidence, not a general implementation guarantee.','later_limit_without_own_order':{'query':d['later-limit']['sql'],'rows':d['later-limit']['rows'],'classification':'No final-stage ordering contract; earlier ordering alone is not a safe top-N selection assumption.'},'later_limit_with_own_order':{'query':d['later-ordered-limit']['sql'],'rows':d['later-ordered-limit']['rows']},'cleanup':{'database_not_found':True,'runtime_container_absent':True},'cases':[{**a,'sql':d[a['label']]['sql'],'rows':d[a['label']]['rows']}for a in spec]};(t/'summary.json').write_text(json.dumps(summary,indent=2)+'\n');(t/'assertions.json').write_text(json.dumps({'passed':len(checks),'failed':0,'checks':checks},indent=2)+'\n');shutil.copyfile(Path(__file__),t/'sources/analyze.py');e=root/'gql-order-root-cause-20261030/limit-membership';e.mkdir(exist_ok=True)
for name in ['sources','evidence']:
 dest=e/name;dest.mkdir(exist_ok=True)
 for f in(t/name).iterdir():
  if f.is_file():shutil.copyfile(f,dest/f.name)
for name in ['summary.json','assertions.json']:shutil.copyfile(t/name,e/name)
print(json.dumps({'queries':35,'checks':len(checks),'choices':choices}))
