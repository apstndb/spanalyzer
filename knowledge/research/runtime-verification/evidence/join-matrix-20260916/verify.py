"""Validate complete retained JOIN evidence independently of SQL execution."""
import json
from pathlib import Path
from collections import Counter
base=Path(__file__).resolve().parent
summary=json.loads((base/'audit-summary.json').read_text())
forms=('inner','left','right','full','in','exists','not_in','not_exists')
methods=('hash','apply','distributed_apply','merge','push')
labels={f'v9/{f}/{m}' for f in forms for m in methods}|{f'v{v}/full/{m}' for v in range(1,9) for m in ('hash','merge')}
L=[(1,10),(2,20),(3,20),(4,30),(5,None)]
R=[(101,10),(102,20),(103,20),(104,40),(105,None)]
match=lambda a,b:a is not None and b is not None and a==b
pairs=[{'l':i,'r':j} for i,a in L for j,b in R if match(a,b)]
left=[{'l':i,'r':None} for i,a in L if not any(match(a,b) for _,b in R)]
right=[{'l':None,'r':j} for j,b in R if not any(match(a,b) for _,a in L)]
expected={'inner':pairs,'left':pairs+left,'right':pairs+right,'full':pairs+left+right,'in':[{'l':i} for i,a in L if any(match(a,b) for _,b in R)],'exists':[{'l':i} for i,a in L if any(match(a,b) for _,b in R)],'not_exists':[{'l':i} for i,a in L if not any(match(a,b) for _,b in R)],'not_in':[]}
# SQL NOT IN rejects every row here because the right input contains NULL.
hash_types={'inner':'INNER','left':'BUILD_OUTER','right':'PROBE_OUTER','full':'BUILD_PROBE_OUTER','in':'BUILD_SEMI','exists':'BUILD_SEMI','not_in':'BUILD_ANTI_SEMI','not_exists':'BUILD_ANTI_SEMI'}
merge_types={'inner':'INNER','left':'LEFT_OUTER','right':'LEFT_OUTER','full':'FULL_OUTER','in':'LEFT_SEMI','exists':'LEFT_SEMI','not_in':'LEFT_ANTI_SEMI','not_exists':'LEFT_ANTI_SEMI'}
apply_names={'inner':'Cross Apply','left':'Outer Apply','right':'Outer Apply','in':'Semi Apply','exists':'Semi Apply','not_in':'Anti-Semi Apply','not_exists':'Anti-Semi Apply'}
dist_names={'inner':'Distributed Cross Apply','left':'Distributed Outer Apply','right':'Distributed Outer Apply','in':'Distributed Semi Apply','exists':'Distributed Semi Apply','not_in':'Distributed Anti Semi Apply','not_exists':'Distributed Anti Semi Apply'}
push_names={'inner':'Push Broadcast Hash Join','left':'Push Broadcast Hash Join Outer Apply','right':'Push Broadcast Hash Join Outer Apply','in':'Push Broadcast Hash Join Semi Apply','exists':'Push Broadcast Hash Join Semi Apply','not_in':'Push Broadcast Hash Join Anti Semi Apply','not_exists':'Push Broadcast Hash Join Anti Semi Apply'}
def bag(rows):return Counter(json.dumps(r,sort_keys=True) for r in rows)
for runtime in ('omni','managed'):
 s=summary[runtime]
 assert s['complete'] and {c['label'] for c in s['cells']}==labels
 records=[json.loads(l) for l in (base/(runtime+'.jsonl')).read_text().splitlines()]
 assert len(s['cleanup'])==4 and all(not r['error'] for r in s['cleanup'])
 for c in s['cells']:
  label=c['label'];v,f,m=label.split('/');version=int(v[1:])
  p=records[c['plan_record_line']-1];e=records[c['execute_record_line']-1]
  assert p['label']==e['label']==label and p['sql']==e['sql']==c['sql']
  assert not p['error'] and not e['error'],(runtime,label,p['error'],e['error'])
  assert bag([json.loads(x) for x in e['result'] or []])==bag(expected[f]),(runtime,label,'rows')
  nodes=p['plan']['planNodes'];ops=[n for n in nodes if any(w in n.get('displayName','') for w in ('Join','Apply','Union All'))]
  names=[n['displayName'] for n in ops]
  by_id={n.get('index',0):n for n in nodes}
  def descendants(start):
   seen=set();todo=[start]
   while todo:
    i=todo.pop()
    if i in seen:continue
    seen.add(i)
    todo.extend(link.get('childIndex',0) for link in by_id[i].get('childLinks',[]))
   return [by_id[i] for i in seen]
  for n in ops:
   if n['displayName'] in ('Distributed Cross Apply','Distributed Outer Apply','Distributed Semi Apply','Distributed Anti Semi Apply'):
    maps=[x.get('childIndex',0) for x in n.get('childLinks',[]) if x.get('type')=='Map' and by_id[x.get('childIndex',0)].get('kind')=='RELATIONAL']
    assert len(maps)==1,(runtime,label,'Map links')
    wanted='Semi Apply' if n['displayName'] in ('Distributed Semi Apply','Distributed Anti Semi Apply') else 'Cross Apply'
    assert any(x['displayName']==wanted for x in descendants(maps[0])),(runtime,label,wanted)
  if f=='right' and m=='merge':
   n=next(n for n in ops if n['displayName']=='Merge Join')
   for side,suffix in [('Left','RByK'),('Right','LByK')]:
    children=[x.get('childIndex',0) for x in n['childLinks'] if x.get('type')==side and by_id[x.get('childIndex',0)].get('kind')=='RELATIONAL']
    assert len(children)==1
    assert any(x.get('metadata',{}).get('scan_target')==s['identity']['prefix']+suffix for x in descendants(children[0]))
  def need(name):assert name in names,(runtime,label,name,names)
  if f=='full' and m in ('apply','distributed_apply','push'):
   for name in {'apply':['Union All','Outer Apply','Anti-Semi Apply'],'distributed_apply':['Union All','Distributed Outer Apply','Distributed Anti Semi Apply'],'push':['Union All','Push Broadcast Hash Join Outer Apply','Push Broadcast Hash Join Anti Semi Apply']}[m]:need(name)
  elif m=='hash':
   types=[n.get('metadata',{}).get('join_type') for n in ops if n['displayName']=='Hash Join']
   if f=='full' and version<=6:need('Union All');assert set(types)=={'BUILD_OUTER','BUILD_ANTI_SEMI'}
   else:assert types==[hash_types[f]],(runtime,label,types)
  elif m=='merge':assert [n.get('metadata',{}).get('join_type') for n in ops if n['displayName']=='Merge Join']==[merge_types[f]],(runtime,label)
  elif m=='apply':need(apply_names[f]);assert not any(n.startswith('Distributed ') for n in names)
  elif m=='distributed_apply':need(dist_names[f]);need('Semi Apply' if f in ('in','exists','not_in','not_exists') else 'Cross Apply')
  elif m=='push':need(push_names[f]);need('Hash Join')
print('PASS: 112 cells / 224 PLAN and execution records; all table cells, FULL version boundary, result multisets, and cleanup DDL validated across Omni and managed Spanner.')
