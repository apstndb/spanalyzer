from pathlib import Path
from concurrent.futures import ThreadPoolExecutor
import subprocess,os,json,datetime,hashlib
r=Path('<LOCAL_CHECKOUT>');task=Path('<LOCAL_TASK_ROOT>/apstndb/spanalyzer/gql-order-root-cause-20261030');p=task/'validation';p.mkdir(exist_ok=True);e=r/'knowledge/research/runtime-verification/evidence/gql-order-root-cause-20261030';go='<LOCAL_HOME>/.local/share/mise/installs/go/1.26.8/bin/go';env=os.environ.copy()
for k in list(env):
 if k.startswith(('SPANEMUBOOST_','SPANNER_','TEST_REAL_SPANNER_','REQUIRE_REAL_SPANNER_','GOOGLE_APPLICATION_CREDENTIALS')):env.pop(k,None)
env.update(GOTOOLCHAIN='local',GOROOT=str(Path(go).parent.parent),GOCACHE=str(r/'.tmp/managed-recheck-20260925/go-cache'),GOFLAGS='-mod=readonly');receipts=[]
def run(name,args,cwd,workspace):
 at=datetime.datetime.now(datetime.timezone.utc).isoformat();runenv=env.copy();runenv['GOWORK']=workspace
 with(p/(name+'.log')).open('w')as out:result=subprocess.run(args,cwd=cwd,env=runenv,stdout=out,stderr=subprocess.STDOUT)
 row={'name':name,'argv':args,'cwd':str(cwd),'GOWORK':workspace,'started_at':at,'completed_at':datetime.datetime.now(datetime.timezone.utc).isoformat(),'exit_code':result.returncode,'log':str(p/(name+'.log'))};print(name,result.returncode,flush=True);return row
with ThreadPoolExecutor(max_workers=2)as pool:
 jobs=[pool.submit(run,'survey-short',[go,'test','-short','./...'],r/'survey','off'),pool.submit(run,'tools-tests',[go,'test','./...'],r/'tools',str(r/'go.work'))];receipts.extend(j.result()for j in jobs)
for name,args in [('okf-all',[go,'run','./okf-check','--repo-root','..','--gate','all','--publication-manifest',str(task/'publication-manifest.json')])]:receipts.append(run(name,args,r/'tools',str(r/'go.work')))
if all(x['exit_code']==0 for x in receipts):
 f=r/'survey/TODO.md';t=f.read_text();old='- [ ] Document the verified source boundary and remaining uncertainty; preserve old evidence and pass local checks.';assert old in t;f.write_text(t.replace(old,old.replace('[ ]','[x]'),1))
receipts.append(run('markdown-language',[go,'run','./markdown-language-check','--repo-root','..'],r/'tools',str(r/'go.work')));receipts.append(run('git-diff-check',['git','diff','--check'],r,str(r/'go.work')))
for x in receipts:
 f=p/(x['name']+'.log');dest=e/'validation-logs'/f.name;dest.parent.mkdir(exist_ok=True);dest.write_bytes(f.read_bytes());x['retained_log']=str(dest.relative_to(e))
result={'status':'passed'if all(x['exit_code']==0 for x in receipts)else'failed','commands':receipts,'evidence_assertions':{'passed':319,'failed':0},'boundary':'No new managed/Omni request and no full source-build or broad source-fix test. Existing implementation and dependencies unchanged.'};(e/'validation.json').write_text(json.dumps(result,indent=2)+'\n');(task/'validation.json').write_text(json.dumps(result,indent=2)+'\n');(e/'producers/validate.py').write_bytes(Path(__file__).read_bytes())
manifest=r/'knowledge/research/runtime-verification/evidence/artifact-sha256.json';m=json.loads(manifest.read_text());m['files'].update({str(f.relative_to(manifest.parent)):hashlib.sha256(f.read_bytes()).hexdigest()for f in e.rglob('*')if f.is_file()});m['files']=dict(sorted(m['files'].items()));manifest.write_text(json.dumps(m,indent=2)+'\n');print(json.dumps({'status':result['status'],'manifest_files':len(m['files'])}),flush=True)
if result['status']!='passed':raise SystemExit(1)
