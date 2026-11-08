from pathlib import Path
import json,shutil,hashlib
r=Path('<LOCAL_CHECKOUT>');s=r/'.tmp/gql-order-bisect-20261031';(s/'module').mkdir(exist_ok=True);t=Path('<LOCAL_TASK_ROOT>/apstndb/spanalyzer/gql-order-bisect-20261031');fixture=json.loads((r/'.tmp/gql-order-patterns-20261030/module/matrix.json').read_text());ddl=fixture['steps'][0]['statements'];steps=[{'label':'tables','family':'FinGraph','kind':'ddl','statements':ddl[:4]},{'label':'graph','family':'FinGraph','kind':'ddl','sql':ddl[4],'requires':['tables']},{**fixture['steps'][1],'requires':['tables']}];spec=[]
def add(label,sql,expected=None,gql=False,needs_graph=False):
 steps.append({'label':label,'family':'historical-ordering','kind':'query','sql':sql,'requires':['seed']+(['graph']if needs_graph else[])});spec.append({'label':label,'sql':sql,'expected':expected,'gql':gql})
for direction,want in [('ASC',['1','2','3']),('DESC',['3','2','1'])]:
 add('sql-id-'+direction,'SELECT id FROM Person ORDER BY id '+direction,[{'id':x}for x in want])
 add('gql-id-'+direction,'GRAPH FinGraph MATCH (p:Person) RETURN p.id ORDER BY p.id '+direction,[{'id':x}for x in want],True,direction!='ASC')
 add('gql-limit-'+direction,'GRAPH FinGraph MATCH(p:Person) RETURN p.id AS v ORDER BY v '+direction+' LIMIT 2',[{'v':x}for x in want[:2]],True,True)
add('gql-id-alias','GRAPH FinGraph MATCH(p:Person) RETURN p.id AS id ORDER BY id',[{'id':x}for x in ['1','2','3']],True,True)
add('gql-name','GRAPH FinGraph MATCH(p:Person) RETURN p.name AS name ORDER BY name',[{'name':x}for x in ['Alex','Dana','Lee']],True,True)
add('gql-for-five','GRAPH FinGraph FOR x IN [5,3,1,4,2] RETURN x AS v ORDER BY v LIMIT 5',[{'v':str(x)}for x in range(1,6)],True,True)
add('sql-outer-graph','SELECT id FROM GRAPH_TABLE(FinGraph MATCH(p:Person) RETURN p.id AS id ORDER BY id) ORDER BY id',[{'id':str(x)}for x in range(1,4)],False,True)
add('sql-unspecified','SELECT id FROM Person')
plan={'dialect':'google','steps':steps};(s/'module/matrix.json').write_text(json.dumps(plan,indent=2)+'\n');shutil.copyfile(s/'module/matrix.json',t/'sources/matrix.json');(t/'sources/case-contracts.json').write_text(json.dumps(spec,indent=2)+'\n');(t/'sources/probe-provenance.json').write_text(json.dumps({'binary':str(r/'.tmp/gql-order-root-cause-20261030/probe'),'sha256':hashlib.sha256((r/'.tmp/gql-order-root-cause-20261030/probe').read_bytes()).hexdigest(),'source':'Same unmodified probe compiled for the October 30 causal experiment; same client/fixture across all releases.'},indent=2)+'\n');shutil.copyfile(r/'.tmp/gql-order-root-cause-20261030/module/probe.go',t/'sources/probe.go.txt');shutil.copyfile(r/'.tmp/gql-order-root-cause-20261030/module/go.mod',t/'sources/probe.go.mod.txt');shutil.copyfile(r/'.tmp/gql-order-root-cause-20261030/module/go.sum',t/'sources/probe.go.sum.txt');print('queries',len(spec),'planned steps',len(steps))
