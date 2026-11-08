from pathlib import Path
import json,shutil
r=Path('<LOCAL_CHECKOUT>');old=Path('<LOCAL_TASK_ROOT>/apstndb/spanalyzer/gql-order-root-cause-20261030/query-shapes');task=old.parent/'limit-membership';s=r/'.tmp/gql-limit-membership-20261030';matrix=json.loads((old/'sources/matrix.json').read_text());steps=matrix['steps'][:2];spec=[]
def add(label,q,kind,want=None):steps.append({'label':label,'family':'limit-membership','kind':'query','sql':q,'requires':['seed']});spec.append({'label':label,'kind':kind,'want':want})
for i in range(4):
 add('person-tie-'+str(i),'GRAPH FinGraph MATCH(p:Person) RETURN p.id AS id,p.id>2 AS k ORDER BY k LIMIT 1','person-tie')
 add('person-total-'+str(i),'GRAPH FinGraph MATCH(p:Person) RETURN p.id AS id,p.id>2 AS k ORDER BY k,id LIMIT 1','person-total')
 add('sql-person-tie-'+str(i),'SELECT id,id>2 AS k FROM Person ORDER BY k LIMIT 1','person-tie')
 add('sql-person-total-'+str(i),'SELECT id,id>2 AS k FROM Person ORDER BY k,id LIMIT 1','person-total')
 add('transfer-tie-'+str(i),'GRAPH FinGraph MATCH ()-[t:Transfers]->() RETURN t.id AS src,t.to_id AS dst,t.amount AS amount ORDER BY amount LIMIT 3','transfer-tie')
 add('transfer-total-'+str(i),'GRAPH FinGraph MATCH ()-[t:Transfers]->() RETURN t.id AS src,t.to_id AS dst,t.amount AS amount ORDER BY amount,src,dst LIMIT 3','transfer-total')
for label,clause,want in [('unique-asc','ORDER BY id ASC LIMIT 2',['1','2']),('unique-desc','ORDER BY id DESC LIMIT 2',['3','2']),('unique-offset','ORDER BY id DESC OFFSET 1 LIMIT 2',['2','1'])]:add(label,'GRAPH FinGraph MATCH(p:Person) RETURN p.id AS id '+clause,'unique-final',want)
for i,values in enumerate(['[1,2,3]','[2,1,3]','[3,2,1]']):add('literal-tie-'+str(i),'GRAPH FinGraph FOR x IN '+values+' RETURN x AS id,x>2 AS k ORDER BY k LIMIT 1','literal-tie')
for i,values in enumerate(['[1,2,3]','[2,1,3]']):add('literal-total-'+str(i),'GRAPH FinGraph FOR x IN '+values+' RETURN x AS id,x>2 AS k ORDER BY k,id LIMIT 1','person-total')
base='GRAPH FinGraph FOR x IN [5,3,1,4,2] RETURN x AS id ORDER BY id LIMIT 5 NEXT RETURN id'
add('later-limit',base+' LIMIT 2','later-limit')
add('later-ordered-limit',base+' ORDER BY id LIMIT 2','unique-final',['1','2'])
add('no-order','GRAPH FinGraph MATCH(p:Person) RETURN p.id AS id LIMIT 2','no-order')
(task/'sources/matrix.json').write_text(json.dumps({'dialect':'google','steps':steps},indent=2)+'\n');shutil.copyfile(task/'sources/matrix.json',s/'module/matrix.json');(task/'sources/contracts.json').write_text(json.dumps(spec,indent=2)+'\n');shutil.copyfile(old/'sources/selected-images.json',task/'sources/selected-images.json');shutil.copyfile(old/'sources/official-setup.json',task/'sources/official-setup.json')
text=(old/'sources/run.py').read_text().replace("scratch=root/'.tmp/gql-order-patterns-20261030'","scratch=root/'.tmp/gql-limit-membership-20261030'").replace(str(old),str(task)).replace("run('query-shapes'","run('limit-membership'");(s/'run.py').write_text(text);shutil.copyfile(s/'run.py',task/'sources/run.py');shutil.copyfile(Path(__file__),task/'sources/prepare.py');(task/'README.md').write_text('# GQL LIMIT membership follow-up\n\nPurpose: distinguish deterministic output permutation, unique-key top-N membership, tied boundary choices and a downstream LIMIT without its own ORDER BY.\nAuthority: public docs reads and an owned disposable local Emulator/database; no managed/Omni calls, implementation edits or external posts.\nRetention: keep source, contracts, output and cleanup evidence until explicitly cleaned up.\n');print('Prepared',len(spec),'queries')
