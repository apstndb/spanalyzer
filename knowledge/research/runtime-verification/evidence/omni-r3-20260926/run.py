import os,pathlib,subprocess,json,time,datetime
root=pathlib.Path('<LOCAL_CHECKOUT>')
work=root/'.worktrees/omni-r3-20260926';out=root/'.tmp/omni-r3-20260926'
env=os.environ.copy();go='<LOCAL_HOME>/.local/share/mise/installs/go/1.26.8/bin/go'
env.update(GOROOT=str(pathlib.Path(go).parent.parent),GOTOOLCHAIN='local',GOCACHE=str(root/'.tmp/managed-recheck-20260925/go-cache'),GOFLAGS='-mod=readonly',TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE='/var/run/docker.sock')
env['PATH']=str(pathlib.Path(go).parent)+':'+env['PATH']
env['DOCKER_HOST']=subprocess.check_output(['docker','context','inspect','--format','{{.Endpoints.docker.Host}}']).decode().strip()
for k in ['SPANEMUBOOST_ENDPOINT_FILE','SPANNER_EMULATOR_HOST','TEST_REAL_SPANNER_DATABASE']:env.pop(k,None)
cases=[('infoschema','survey',['run','./cmd/infoschema-capture','--target','omni','--repo-root','..']),('spanner-sys','survey',['run','./cmd/spanner-sys-capture','--target','omni','--repo-root','..']),('omni-drift','survey',['test','-json','-count=1','-timeout=15m','-run','^TestDrift_OmniTableMetas$','./infoschem']),('omni-integration','tools',['run','./omni-integration-runner','--repo-root','..'])]
receipts=[]
for name,wd,args in cases:
 child=env.copy()
 if wd=='survey':child['GOWORK']='off'
 else:child.pop('GOWORK',None)
 started=datetime.datetime.now(datetime.timezone.utc).isoformat();start=time.time()
 print('Starting '+name,flush=True)
 with (out/(name+'.stdout')).open('w') as stdout,(out/(name+'.stderr')).open('w') as stderr:
  p=subprocess.run([go,*args],cwd=work/wd,env=child,stdout=stdout,stderr=stderr)
 receipt={'name':name,'started_at':started,'exit_code':p.returncode,'elapsed_seconds':round(time.time()-start,2),'cwd':str(work/wd),'args':args}
 receipts.append(receipt);(out/'commands.json').write_text(json.dumps(receipts,indent=2)+'\n');print(json.dumps(receipt),flush=True)
