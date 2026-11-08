from pathlib import Path
from concurrent.futures import ThreadPoolExecutor
import hashlib,json,subprocess
p=Path('<LOCAL_TASK_ROOT>/apstndb/spanalyzer/gql-order-root-cause-20261030/sources');paths=['googlesql/analyzer/graph_query_resolver.cc','googlesql/reference_impl/algebrizer_graph.cc','googlesql/reference_impl/algebrizer.cc','googlesql/reference_impl/relational_op.cc','googlesql/reference_impl/tuple.h','googlesql/public/evaluator_base.cc','googlesql/resolved_ast/gen_resolved_ast.py']
def run(path):
 r=subprocess.run(['gh','api','-H','Accept: application/vnd.github.raw+json',f'repos/google/googlesql/contents/{path}?ref=2026.7.2'],capture_output=True);assert r.returncode==0,(path,r.stderr);name='googlesql-'+path.replace('/','__');(p/name).write_bytes(r.stdout);return{'path':path,'file':name,'sha256':hashlib.sha256(r.stdout).hexdigest()}
with ThreadPoolExecutor(max_workers=4)as pool:a=list(pool.map(run,paths))
r=subprocess.run(['gh','api','repos/google/googlesql/git/ref/tags/2026.7.2'],capture_output=True);assert r.returncode==0,r.stderr;(p/'googlesql-tag.json').write_bytes(r.stdout);(p/'googlesql-retrieval.json').write_text(json.dumps(a,indent=2)+'\n');print(json.dumps(a))
