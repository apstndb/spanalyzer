from pathlib import Path
from concurrent.futures import ThreadPoolExecutor
import subprocess,json,hashlib,datetime
p=Path('<LOCAL_TASK_ROOT>/apstndb/spanalyzer/gql-order-bisect-20261031/sources');ledger=[]
def api(args):
 r=subprocess.run(['gh','api',*args],capture_output=True);return r.returncode,r.stdout,r.stderr
for version in ['1.5.29','1.5.30']:
 code,data,err=api(['repos/GoogleCloudPlatform/cloud-spanner-emulator/commits/v'+version]);assert code==0,err;sha=json.loads(data)['sha'];(p/('commit-'+version+'.json')).write_bytes(data)
 for path in ['WORKSPACE','backend/query/query_engine.cc']:
  code,data,err=api(['-H','Accept: application/vnd.github.raw+json',f'repos/GoogleCloudPlatform/cloud-spanner-emulator/contents/{path}?ref={sha}']);name='source-'+version+'-'+path.replace('/','__');(p/name).write_bytes(data);ledger.append({'version':version,'commit':sha,'path':path,'file':name,'exit_code':code,'sha256':hashlib.sha256(data).hexdigest(),'retrieved_at':datetime.datetime.now(datetime.timezone.utc).isoformat()});print(version,path,code,flush=True)
(p/'boundary-source-retrieval.json').write_text(json.dumps(ledger,indent=2)+'\n')
