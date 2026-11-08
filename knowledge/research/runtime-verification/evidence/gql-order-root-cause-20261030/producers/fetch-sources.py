from pathlib import Path
from concurrent.futures import ThreadPoolExecutor
import hashlib,json,subprocess,datetime
p=Path('<LOCAL_TASK_ROOT>/apstndb/spanalyzer/gql-order-root-cause-20261030/sources');paths=['MODULE.bazel','WORKSPACE','backend/query/query_engine.cc','build/bazel/zetasql.patch','build/bazel/googlesql.patch','backend/query/query_engine_test.cc'];records=[]
def get(path):
 r=subprocess.run(['gh','api','-H','Accept: application/vnd.github.raw+json',f'repos/GoogleCloudPlatform/cloud-spanner-emulator/contents/{path}?ref=v1.5.59'],capture_output=True);name='emulator-'+path.replace('/','__');(p/name).write_bytes(r.stdout);return{'path':path,'file':name,'exit_code':r.returncode,'sha256':hashlib.sha256(r.stdout).hexdigest(),'stderr':r.stderr.decode()}
with ThreadPoolExecutor(max_workers=4)as pool:records=list(pool.map(get,paths))
r=subprocess.run(['gh','api','repos/GoogleCloudPlatform/cloud-spanner-emulator/git/ref/tags/v1.5.59'],capture_output=True,text=True);assert r.returncode==0,r.stderr;(p/'emulator-tag.json').write_text(r.stdout)
(p/'retrieval.json').write_text(json.dumps({'at':datetime.datetime.now(datetime.timezone.utc).isoformat(),'records':records},indent=2)+'\n');print(json.dumps(records))
