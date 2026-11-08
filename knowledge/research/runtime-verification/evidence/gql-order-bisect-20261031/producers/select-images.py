from pathlib import Path
from concurrent.futures import ThreadPoolExecutor
import json,urllib.request,hashlib,datetime,sys
p=Path('<LOCAL_TASK_ROOT>/apstndb/spanalyzer/gql-order-bisect-20261031/sources');registry=json.loads((p/'registry.json').read_text());selected=json.loads((p/'selected-images.json').read_text())if(p/'selected-images.json').exists()else{};versions=sys.argv[1:]
def fetch(version):
 indexes=[(d,m)for d,m in registry['manifest'].items()if version in m.get('tag',[])];assert len(indexes)==1,indexes;digest,_=indexes[0];url='https://gcr.io/v2/cloud-spanner-emulator/emulator/manifests/'+digest;req=urllib.request.Request(url,headers={'Accept':'application/vnd.docker.distribution.manifest.list.v2+json,application/vnd.oci.image.index.v1+json'})
 with urllib.request.urlopen(req,timeout=45)as response:raw=response.read();header_digest=response.headers.get('Docker-Content-Digest')
 assert'sha256:'+hashlib.sha256(raw).hexdigest()==digest;data=json.loads(raw);(p/('manifest-index-'+version+'.json')).write_bytes(raw);platforms=[x['platform']for x in data.get('manifests',[])];candidates=[x for x in data['manifests']if x['platform'].get('os')=='linux'and x['platform'].get('architecture')=='arm64'];assert len(candidates)==1,platforms;child=candidates[0]['digest'];return version,{'version':version,'index_digest':digest,'platform_digest':child,'platform':'linux/arm64','available_platforms':platforms,'image':'gcr.io/cloud-spanner-emulator/emulator:'+version+'@'+child,'retrieved_at':datetime.datetime.now(datetime.timezone.utc).isoformat(),'source_url':url,'header_digest':header_digest}
with ThreadPoolExecutor(max_workers=3)as pool:
 for version,row in pool.map(fetch,versions):selected[version]=row;print(version,row['platform'],row['platform_digest'],flush=True)
(p/'selected-images.json').write_text(json.dumps(selected,indent=2)+'\n')
