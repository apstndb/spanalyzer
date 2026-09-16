import json,re,hashlib
from pathlib import Path
root=Path(__file__).resolve().parent
L=[(1,10),(2,20),(3,20),(4,30),(5,None)]
R=[(101,10),(102,20),(103,20),(104,40),(105,None)]
pairs=[{'l':li,'r':ri} for li,lk in L for ri,rk in R if lk is not None and lk==rk]
left_extra=[{'l':li,'r':None} for li,lk in L if not any(lk is not None and lk==rk for _,rk in R)]
right_extra=[{'l':None,'r':ri} for ri,rk in R if not any(rk is not None and rk==lk for _,lk in L)]
expected={'inner':pairs,'left':pairs+left_extra,'right':pairs+right_extra,'full':pairs+left_extra+right_extra,'in':[{'l':i} for i in (1,2,3)],'exists':[{'l':i} for i in (1,2,3)],'not_exists':[{'l':i} for i in (4,5)],'not_in':[]}
def canon(rows):return sorted(json.dumps(v,sort_keys=True) for v in rows)
methods=['hash','apply','distributed_apply','merge','push']
labels={f'v9/{f}/{m}' for f in expected for m in methods}|{f'v{v}/full/{m}' for v in range(1,9) for m in ('hash','merge')}
summary={}
for runtime in ('omni','managed'):
 path=root/(runtime+'.jsonl')
 rows=[json.loads(l) for l in path.read_text().splitlines()]
 queries={}
 for r in rows:
  if r.get('kind')!='query' or r['label']=='preflight':continue
  label=r['label'];key='plan' if r['plan_requested'] else 'execute'
  assert key not in queries.setdefault(label,{}),(runtime,label,key)
  queries[label][key]=r
 result=[]
 for label,q in queries.items():
  if set(q)!={'plan','execute'}:continue
  p,e=q['plan'],q['execute'];joins=[]
  for n in p.get('plan',{}).get('planNodes',[]):
   if any(x in n.get('displayName','') for x in ('Join','Apply','Union All')):
    joins.append({'index':n.get('index',0),'name':n['displayName'],'metadata':n.get('metadata',{}),'children':n.get('childLinks',[])})
  if not e['error']:assert canon([json.loads(v) for v in (e['result'] or [])])==canon(expected[label.split('/')[1]]),(runtime,label,e['result'])
  result.append({'plan_record_line':rows.index(p)+1,'execute_record_line':rows.index(e)+1,'label':label,'sql':p['sql'],'plan_error':re.sub(r', requestID = "[^"]+"','',p['error']),'execute_error':re.sub(r', requestID = "[^"]+"','',e['error']),'expected_rows':len(expected[label.split('/')[1]]),'actual_rows':e['rows'],'result_match':not bool(e['error']),'operators':joins})
 summary[runtime]={'identity':rows[0], 'cleanup':[r for r in rows if r.get('kind')=='ddl' and r.get('label','').startswith('cleanup-')], 'complete':set(queries)==labels and all(set(q)=={'plan','execute'} for q in queries.values()),'records':len(rows),'cells':result}
 print(runtime,len(result),'complete',summary[runtime]['complete'])
 for r in result:
  if r['label'].startswith('v9/'):
   print(r['label'], [(n['name'],n['metadata'].get('join_type')) for n in r['operators']],r['plan_error'][:130],r['execute_error'][:130])
(root/'audit-summary.json').write_text(json.dumps(summary,indent=2)+'\n')
