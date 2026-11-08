from pathlib import Path
import json,shutil
r=Path('<LOCAL_CHECKOUT>');old=Path('<LOCAL_TASK_ROOT>/apstndb/spanalyzer/gql-order-root-cause-20261030/fingraph-minimal');task=old.parent/'query-shapes';scratch=r/'.tmp/gql-order-patterns-20261030';(scratch/'module').mkdir(parents=True,exist_ok=True);(task/'sources').mkdir(parents=True,exist_ok=True);(task/'evidence').mkdir(exist_ok=True)
base=json.loads((old/'sources/matrix.json').read_text());steps=base['steps'][:2];spec=[]
def add(name,q,kind='ordered-rows',oracle=None,want=None):
 steps.append({'label':name,'family':'GQL-order-query-shapes','kind':'query','sql':q,'requires':['seed']});spec.append({'label':name,'kind':kind,'oracle':oracle,'expected':want})
for expr,sql,name in [('p.id','id','id'),('p.name','name','name'),('p.country','country','country'),('p.birthday','birthday','timestamp'),('p.id+10','id+10','arithmetic'),('CAST(p.id AS STRING)','CAST(id AS STRING)','string-cast')]:
 for direction in ['ASC','DESC']:
  label=name+'-'+direction;add(label,'GRAPH FinGraph MATCH(p:Person) RETURN '+expr+' AS v ORDER BY v '+direction,oracle='sql-'+label);add('sql-'+label,'SELECT '+sql+' AS v FROM Person ORDER BY v '+direction,kind='sql-oracle')
for direction in ['ASC','DESC']:
 add('distinct-'+direction,'GRAPH FinGraph MATCH(p:Person) RETURN DISTINCT p.id AS v ORDER BY v '+direction,want=list(map(str,[1,2,3]if direction=='ASC'else[3,2,1])))
 add('aggregate-'+direction,'GRAPH FinGraph MATCH(a:Account)-[t:Transfers]->() RETURN a.id AS id, SUM(t.amount) AS total GROUP BY id ORDER BY total '+direction,oracle='sql-aggregate-'+direction)
 add('sql-aggregate-'+direction,'SELECT id, SUM(amount) AS total FROM AccountTransferAccount GROUP BY id ORDER BY total '+direction,kind='sql-oracle')
 add('next-'+direction,'GRAPH FinGraph MATCH(p:Person) RETURN p.id AS id NEXT RETURN id AS v ORDER BY v '+direction,want=list(map(str,[1,2,3]if direction=='ASC'else[3,2,1])))
 for count in [1,2]:add(f'limit{count}-'+direction,f'GRAPH FinGraph MATCH(p:Person) RETURN p.id AS v ORDER BY v {direction} LIMIT {count}',kind='selection'if count==1 else'ordered-rows-and-selection',want=list(map(str,([1,2,3]if direction=='ASC'else[3,2,1])[:count])))
 add('primitive-limit2-'+direction,'GRAPH FinGraph MATCH(p:Person) ORDER BY p.id '+direction+' LIMIT 2 RETURN p.id AS v',kind='selection-only',want=list(map(str,([1,2,3]if direction=='ASC'else[3,2,1])[:2])))
add('primitive-no-page','GRAPH FinGraph MATCH(p:Person) ORDER BY p.id DESC RETURN p.id AS v',kind='undefined-output-order')
add('offset-limit','GRAPH FinGraph MATCH(p:Person) RETURN p.id AS v ORDER BY v DESC OFFSET 1 LIMIT 2',kind='ordered-rows-and-selection',want=['2','1'])
add('tie-keys','GRAPH FinGraph MATCH(p:Person) RETURN p.id AS id, p.id>2 AS k ORDER BY k',kind='tie-keys')
add('tie-keys-with-id','GRAPH FinGraph MATCH(p:Person) RETURN p.id AS id, p.id>2 AS k ORDER BY k,id',kind='ordered-ids',want=['1','2','3'])
add('aggregate-order','GRAPH FinGraph MATCH(p:Person) RETURN ARRAY_AGG(p.id ORDER BY p.id) AS ids',kind='array-order',want=['1','2','3'])
add('aggregate-order-desc','GRAPH FinGraph MATCH(p:Person) RETURN ARRAY_AGG(p.id ORDER BY p.id DESC) AS ids',kind='array-order',want=['3','2','1'])
add('next-array','GRAPH FinGraph MATCH(p:Person) RETURN p.id AS id ORDER BY id NEXT RETURN ARRAY_AGG(id) AS ids',kind='encounter-order-observation',want=['1','2','3'])
add('for-values','GRAPH FinGraph FOR x IN [3,1,2] RETURN x AS v ORDER BY v',want=['1','2','3'])
add('sql-outer-graph','SELECT v FROM GRAPH_TABLE(FinGraph MATCH(p:Person) RETURN p.id AS v ORDER BY v) ORDER BY v',kind='sql-outer-control',want=['1','2','3'])
(task/'sources/matrix.json').write_text(json.dumps({'dialect':'google','steps':steps},indent=2)+'\n');shutil.copyfile(task/'sources/matrix.json',scratch/'module/matrix.json');(task/'sources/case-contracts.json').write_text(json.dumps(spec,indent=2)+'\n');shutil.copyfile(old/'sources/selected-images.json',task/'sources/selected-images.json');shutil.copyfile(old/'sources/official-setup.json',task/'sources/official-setup.json');shutil.copyfile(old.parent/'gql-order-patterns-doc.json',task/'sources/official-query-statements.json')
s=(old/'sources/run.py').read_text().replace("scratch=root/'.tmp/fingraph-order-minimal-20261030'","scratch=root/'.tmp/gql-order-patterns-20261030'").replace(str(old),str(task)).replace("run('fingraph'","run('query-shapes'");(scratch/'run.py').write_text(s);shutil.copyfile(scratch/'run.py',task/'sources/run.py');(task/'README.md').write_text('# GQL ordering query-shape follow-up\n\nPurpose: determine which scalar types, final RETURN forms, internal ordering consumers and SQL controls execute and retain ordering on the pinned unmodified Emulator.\nAuthority: official-source reads and one disposable local Emulator/database; no managed/Omni calls, implementation changes or external publication.\nRetention: retain exact official FinGraph fixture, source documents, query contracts, stdout and cleanup until explicitly cleaned up.\n');print('queries',len(spec),'steps',len(steps))
