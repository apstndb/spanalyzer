from pathlib import Path
import json,subprocess,uuid
s=Path('<LOCAL_CHECKOUT>/.tmp/gql-order-root-cause-20261030');p=Path('<LOCAL_TASK_ROOT>/apstndb/spanalyzer/gql-order-root-cause-20261030/evidence');image='gcr.io/cloud-spanner-emulator/emulator:1.5.59@sha256:969af6a9349948fbfb4a826c18e160e258a4005f7c759f8e7f2dcec369cccba8';info=json.loads(subprocess.check_output(['docker','image','inspect',image],text=True))[0];(p/'image-inspect.json').write_text(json.dumps({'id':info['Id'],'architecture':info['Architecture'],'os':info['Os'],'created':info['Created'],'config':info['Config']},indent=2)+'\n');r=subprocess.run(['docker','create','--name','spanalyzer-gql-extract-'+uuid.uuid4().hex[:8],image],capture_output=True,text=True);assert r.returncode==0,r.stderr;cid=r.stdout.strip()
try:
 r=subprocess.run(['docker','cp',cid+':/emulator_main',str(s/'emulator_main.original')],capture_output=True,text=True);assert r.returncode==0,r.stderr;print('binary bytes',(s/'emulator_main.original').stat().st_size)
finally:
 close=subprocess.run(['docker','rm',cid],capture_output=True,text=True);absent=subprocess.run(['docker','inspect',cid],capture_output=True,text=True);(p/'extract-container.json').write_text(json.dumps({'id':cid,'image':image,'remove_code':close.returncode,'absent':absent.returncode!=0},indent=2)+'\n')
