from pathlib import Path
import json,subprocess,uuid,datetime
s=Path('<LOCAL_CHECKOUT>/.tmp/gql-order-root-cause-20261030');task=Path('<LOCAL_TASK_ROOT>/apstndb/spanalyzer/gql-order-root-cause-20261030');selection={'official-before':{'image':'gcr.io/cloud-spanner-emulator/emulator:1.5.59@sha256:969af6a9349948fbfb4a826c18e160e258a4005f7c759f8e7f2dcec369cccba8'}};images=[]
for name in ['original-copy','no-scramble','graph-ordered']:
 tag='spanalyzer-gql-order-diagnostic:'+name+'-'+uuid.uuid4().hex[:8];args=['docker','build','--platform','linux/arm64','--tag',tag,str(s/'images'/name)];started=datetime.datetime.now(datetime.timezone.utc).isoformat()
 with(task/'evidence'/('build-'+name+'.log')).open('w')as log:r=subprocess.run(args,stdout=log,stderr=subprocess.STDOUT)
 assert r.returncode==0,('build',name,r.returncode);a=json.loads(subprocess.check_output(['docker','image','inspect',tag],text=True))[0];selection[name]={'image':a['Id']};images.append({'variant':name,'argv':args,'started_at':started,'tag':tag,'image_id':a['Id'],'os':a['Os'],'architecture':a['Architecture']});print(name,a['Id'],flush=True)
selection['official-after']=selection['official-before'];(task/'sources/selected-images.json').write_text(json.dumps(selection,indent=2)+'\n');(task/'evidence/diagnostic-images.json').write_text(json.dumps(images,indent=2)+'\n')
