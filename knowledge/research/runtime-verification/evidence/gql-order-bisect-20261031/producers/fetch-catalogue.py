from pathlib import Path
from concurrent.futures import ThreadPoolExecutor
import json,urllib.request,datetime,hashlib,subprocess
p=Path('<LOCAL_TASK_ROOT>/apstndb/spanalyzer/gql-order-bisect-20261031/sources')
def fetch(which):
 if which=='releases':
  r=subprocess.run(['gh','api','--paginate','--slurp','repos/GoogleCloudPlatform/cloud-spanner-emulator/releases?per_page=100'],capture_output=True);assert r.returncode==0,r.stderr;data=r.stdout;url='https://api.github.com/repos/GoogleCloudPlatform/cloud-spanner-emulator/releases?per_page=100'
 else:
  url='https://gcr.io/v2/cloud-spanner-emulator/emulator/tags/list'
  with urllib.request.urlopen(url,timeout=45)as response:data=response.read()
 name=which+'.json';(p/name).write_bytes(data);return{'file':name,'url':url,'retrieved_at':datetime.datetime.now(datetime.timezone.utc).isoformat(),'sha256':hashlib.sha256(data).hexdigest()}
with ThreadPoolExecutor(max_workers=2)as pool:rows=list(pool.map(fetch,['releases','registry']))
(p/'retrieval.json').write_text(json.dumps(rows,indent=2)+'\n');releases=[r for page in json.loads((p/'releases.json').read_text())for r in page];print('releases',len(releases))
for entry in releases:
 body=entry.get('body')or''
 if any(x in body.lower()for x in ['graph','gql']):print(entry['tag_name'],entry['published_at'],body)
reg=json.loads((p/'registry.json').read_text());print('tags',reg['tags'])
