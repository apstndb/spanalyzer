from pathlib import Path
from concurrent.futures import ThreadPoolExecutor
import json,subprocess,hashlib,datetime
p=Path('<LOCAL_TASK_ROOT>/apstndb/spanalyzer/gql-order-root-cause-20261030/sources');r=subprocess.run(['gh','api','repos/google/googlesql/commits/HEAD'],capture_output=True,text=True);assert r.returncode==0,r.stderr;head=json.loads(r.stdout)['sha'];(p/'googlesql-current-head.json').write_text(r.stdout)
paths=['googlesql/analyzer/graph_query_resolver.cc','googlesql/reference_impl/algebrizer_graph.cc','googlesql/resolved_ast/gen_resolved_ast.py']
def get(path):
 r=subprocess.run(['gh','api','-H','Accept: application/vnd.github.raw+json',f'repos/google/googlesql/contents/{path}?ref={head}'],capture_output=True);assert r.returncode==0,r.stderr;name='head-'+path.replace('/','__');(p/name).write_bytes(r.stdout);return{'path':path,'file':name,'sha256':hashlib.sha256(r.stdout).hexdigest()}
with ThreadPoolExecutor(max_workers=3)as pool:files=list(pool.map(get,paths))
(p/'head-retrieval.json').write_text(json.dumps({'head':head,'retrieved_at':datetime.datetime.now(datetime.timezone.utc).isoformat(),'files':files},indent=2)+'\n');print(head)
