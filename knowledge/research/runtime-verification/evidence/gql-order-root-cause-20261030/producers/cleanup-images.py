from pathlib import Path
import json,subprocess,datetime
p=Path('<LOCAL_TASK_ROOT>/apstndb/spanalyzer/gql-order-root-cause-20261030');rows=[]
for entry in json.loads((p/'evidence/diagnostic-images.json').read_text()):
 tag=entry['tag'];assert tag.startswith('spanalyzer-gql-order-diagnostic:');r=subprocess.run(['docker','image','rm',tag],capture_output=True,text=True);check=subprocess.run(['docker','image','inspect',tag],capture_output=True);rows.append({'tag':tag,'image_id':entry['image_id'],'remove_exit_code':r.returncode,'stdout':r.stdout,'stderr':r.stderr,'tag_absent':check.returncode!=0});assert r.returncode==0 and check.returncode!=0
base=json.loads((p/'sources/selected-images.json').read_text())['official-before']['image'];b=subprocess.run(['docker','image','inspect',base],capture_output=True);assert b.returncode==0
(p/'evidence/diagnostic-image-cleanup.json').write_text(json.dumps({'recorded_at':datetime.datetime.now(datetime.timezone.utc).isoformat(),'owned':rows,'official_base_preserved':True,'force_or_prune_used':False},indent=2)+'\n');print('Removed three owned diagnostic image tags; official base remains.')
