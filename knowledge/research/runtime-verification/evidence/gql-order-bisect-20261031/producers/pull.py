from pathlib import Path
from concurrent.futures import ThreadPoolExecutor
import json,subprocess,datetime,sys
p=Path('<LOCAL_TASK_ROOT>/apstndb/spanalyzer/gql-order-bisect-20261031');selected=json.loads((p/'sources/selected-images.json').read_text());path=p/'evidence/pulls.json';ledger=json.loads(path.read_text())if path.exists()else[]
def pull(version):
 image=selected[version]['image'];before=subprocess.run(['docker','image','inspect',image],capture_output=True,text=True);start=datetime.datetime.now(datetime.timezone.utc).isoformat();code=0;log='Already present; inspected without changing the image.\n'
 if before.returncode:
  a=subprocess.run(['docker','pull','--platform','linux/arm64',image],capture_output=True,text=True);code=a.returncode;log=a.stdout+a.stderr
 (p/'evidence'/('pull-'+version+'.log')).write_text(log);row={'version':version,'image':image,'preexisting':before.returncode==0,'started_at':start,'completed_at':datetime.datetime.now(datetime.timezone.utc).isoformat(),'exit_code':code}
 if code==0:
  a=json.loads(subprocess.check_output(['docker','image','inspect',image],text=True))[0];row['local_image']={k:a[k]for k in ['Id','RepoDigests','Os','Architecture','Created']};assert a['Os']=='linux'and a['Architecture']=='arm64';(p/'sources'/('local-image-'+version+'.json')).write_text(json.dumps(row['local_image'],indent=2)+'\n')
 print(version,'pull',code,'preexisting',row['preexisting'],flush=True);return row
with ThreadPoolExecutor(max_workers=3)as pool:rows=list(pool.map(pull,sys.argv[1:]));ledger.extend(rows)
path.write_text(json.dumps(ledger,indent=2)+'\n')
if any(x['exit_code']for x in rows):raise SystemExit(1)
