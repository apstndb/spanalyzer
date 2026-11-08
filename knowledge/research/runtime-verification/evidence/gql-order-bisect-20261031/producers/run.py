"""Owned, digest-pinned local runtime probes; managed environment is excluded."""
from pathlib import Path
import datetime,json,os,subprocess,threading,time
root=Path('<LOCAL_CHECKOUT>');scratch=root/'.tmp/gql-order-bisect-20261031';module=scratch/'module';task=Path('<LOCAL_TASK_ROOT>/apstndb/spanalyzer/gql-order-bisect-20261031');out=task/'evidence';go='<LOCAL_HOME>/.local/share/mise/installs/go/1.26.8/bin/go';env=os.environ.copy()
for k in list(env):
 if k.startswith(('SPANEMUBOOST_','SPANNER_','TEST_REAL_SPANNER_','REQUIRE_REAL_SPANNER_','GOOGLE_APPLICATION_CREDENTIALS')):env.pop(k,None)
env.update(GOWORK='off',GOTOOLCHAIN='local',GOROOT=str(Path(go).parent.parent),GOCACHE=str(root/'.tmp/managed-recheck-20260925/go-cache'),GOFLAGS='-mod=readonly',TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE='/var/run/docker.sock');env['PATH']=str(Path(go).parent)+':'+env['PATH'];env['DOCKER_HOST']=subprocess.check_output(['docker','context','inspect','--format','{{.Endpoints.docker.Host}}'],text=True).strip()
images=json.loads((task/'sources/selected-images.json').read_text());ledger=out/'commands.json';receipts=json.loads(ledger.read_text())if ledger.exists()else[]
def container_ids():return subprocess.check_output(['docker','ps','-aq','--no-trunc'],text=True).split()
def inspect_owned(name,image,before):
 found=[]
 for cid in set(container_ids())-set(before):
  r=subprocess.run(['docker','inspect',cid],capture_output=True,text=True)
  if r.returncode:continue
  a=json.loads(r.stdout)[0]
  if a['Config']['Image']!=image:continue
  found.append({'id':cid,'image_id':a['Image'],'configured_image':a['Config']['Image'],'path':a['Path'],'args':a['Args'],'state':a['State']['Status'],'platform':a.get('Platform'),'labels':a['Config'].get('Labels')})
 (out/(name+'-active-containers.json')).write_text(json.dumps(found,indent=2)+'\n');return found

def run(name,args,timeout=2400,image=None):
 if any(r['name']==name and r['exit_code']==0 for r in receipts):print('RETAINED '+name,flush=True);return 0
 before=container_ids()if image else[];started=datetime.datetime.now(datetime.timezone.utc).isoformat();print('START '+name,flush=True);at=time.time();owned=[]
 with(out/(name+'.stderr')).open('w')as err,(out/(name+'.stdout')).open('w')as output:
  p=subprocess.Popen(args,cwd=module,env=env,stdout=subprocess.PIPE,stderr=err,text=True)
  def consume():
   nonlocal owned
   for line in p.stdout:
    output.write(line);output.flush()
    try:r=json.loads(line)
    except ValueError:continue
    if image and r.get('label')=='runtime-start'and r.get('code')=='OK':owned=inspect_owned(name,image,before)
  reader=threading.Thread(target=consume,daemon=True);reader.start()
  try:code=p.wait(timeout=timeout)
  except subprocess.TimeoutExpired:p.kill();code=p.wait();print('TIMEOUT '+name,flush=True)
  reader.join(timeout=30)
 if image:
  states=[]
  for row in owned:
   r=subprocess.run(['docker','inspect',row['id']],capture_output=True,text=True);states.append({'id':row['id'],'inspect_exit_code':r.returncode,'absent':r.returncode!=0})
  (out/(name+'-container-cleanup.json')).write_text(json.dumps({'owned':states,'preexisting_containers_preserved':set(before).issubset(set(container_ids()))},indent=2)+'\n')
 entry={'name':name,'argv':args,'cwd':str(module),'started_at':started,'completed_at':datetime.datetime.now(datetime.timezone.utc).isoformat(),'seconds':round(time.time()-at,3),'exit_code':code};receipts.append(entry);ledger.write_text(json.dumps(receipts,indent=2)+'\n');print(json.dumps(entry),flush=True);return code
import sys
for version in sys.argv[1:]:
 image=images[version]['image'];probe=root/'.tmp/gql-order-root-cause-20261030/probe';code=run('v'+version,[str(probe),'emulator',image,str(module/'matrix.json')],image=image,timeout=240)
 rows=[json.loads(x)for x in(out/('v'+version+'.stdout')).read_text().splitlines()if x.startswith('{')];by={x['label']:x for x in rows};primary=by.get('gql-id-ASC',{});sql=by.get('sql-id-ASC',{});actual=primary.get('rows');expected=[{'id':str(x)}for x in range(1,4)]
 if code:classification='process-failed'
 elif sql.get('code')!='OK'or sql.get('rows')!=expected:classification='control-blocked'
 elif primary.get('code')=='OK':classification='ordered'if actual==expected else'wrong-order'if sorted(x['id']for x in actual)==['1','2','3']else'wrong-membership'
 else:classification='query-unavailable'
 summary={'version':version,'image':image,'process_exit_code':code,'classification':classification,'primary_code':primary.get('code'),'primary_rows':actual,'primary_error':primary.get('error'),'graph_code':by.get('graph',{}).get('code'),'graph_error':by.get('graph',{}).get('error'),'create_database':by.get('create-database',{}).get('code'),'drop_database':by.get('drop-database',{}).get('code'),'database_absent':by.get('database-absent',{}).get('code'),'runtime_cleanup':by.get('runtime-cleanup',{}).get('code')};(out/('v'+version+'-summary.json')).write_text(json.dumps(summary,indent=2)+'\n');print('RESULT '+json.dumps(summary),flush=True)
