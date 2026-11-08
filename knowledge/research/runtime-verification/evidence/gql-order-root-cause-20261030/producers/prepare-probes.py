from pathlib import Path
import shutil,json
r=Path('<LOCAL_CHECKOUT>');s=r/'.tmp/gql-order-root-cause-20261030';m=s/'module';m.mkdir(exist_ok=True)
for n in ['go.mod','go.sum','probe.go']:shutil.copyfile(r/'.tmp/cross-engine-doc-delta-20261029/module'/n,m/n)
f=m/'probe.go';t=f.read_text();needle='for _, path := range os.Args[3:] {';assert needle in t;t=t.replace(needle,'time.Sleep(2*time.Second) // Keep the diagnostic runtime alive for active identity inspection.\n\t'+needle);f.write_text(t)
steps=[]
def add(label,kind,sql='',requires=(),**kw):
 row={'label':label,'family':'gql-order-causal-matrix','kind':kind,'sql':sql,'requires':list(requires)};row.update(kw);steps.append(row)
add('nodes','ddl','CREATE TABLE OrderNodes(id INT64 NOT NULL,val INT64) PRIMARY KEY(id)');add('graph','ddl','CREATE PROPERTY GRAPH OrderGraph NODE TABLES(OrderNodes LABEL Node PROPERTIES(id,val))',['nodes']);add('seed','dml',requires=['nodes'],statements=['INSERT INTO OrderNodes(id,val) VALUES(1,50),(2,10),(3,40),(4,20),(5,30)'])
for repetition in range(3):
 for direction in ['ASC','DESC']:
  for suffix,limit in [('all',''),('top3',' LIMIT 3')]:
   q='GRAPH OrderGraph MATCH(n) RETURN n.val AS value ORDER BY value '+direction+limit;add(f'gql-value-{direction}-{suffix}-r{repetition}','query',q,['graph','seed'])
for expr,name in [('n.id','id'),('n.id+10','expression'),('CAST(n.id AS STRING)','string')]:
 for d in ['ASC','DESC']:add('gql-'+name+'-'+d,'query',f'GRAPH OrderGraph MATCH(n) RETURN {expr} AS value ORDER BY value {d}',['graph','seed'])
for d in ['ASC','DESC']:
 add('sql-value-'+d,'query','SELECT val AS value FROM OrderNodes ORDER BY value '+d,['seed'])
 add('sql-outer-gql-'+d,'query','SELECT value FROM GRAPH_TABLE(OrderGraph MATCH(n) RETURN n.val AS value ORDER BY value '+d+') ORDER BY value '+d,['graph','seed'])
 add('sql-inner-gql-only-'+d,'query','SELECT value FROM GRAPH_TABLE(OrderGraph MATCH(n) RETURN n.val AS value ORDER BY value '+d+')',['graph','seed'])
add('gql-unspecified','query','GRAPH OrderGraph MATCH(n) RETURN n.val AS value',['graph','seed']);add('sql-unspecified','query','SELECT val AS value FROM OrderNodes',['seed']);add('sql-scalar-control','query','SELECT 7 AS value')
(m/'matrix.json').write_text(json.dumps({'dialect':'google','steps':steps},indent=2)+'\n');(s/'schema.sql').write_text('CREATE TABLE OrderNodes(id INT64 NOT NULL,val INT64) PRIMARY KEY(id);\nCREATE PROPERTY GRAPH OrderGraph NODE TABLES(OrderNodes LABEL Node PROPERTIES(id,val));\n');print(json.dumps({'planned_steps':len(steps),'queries':sum(x['kind']=='query'for x in steps)}))
