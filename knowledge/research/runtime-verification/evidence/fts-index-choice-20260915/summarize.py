import json, re, hashlib
from pathlib import Path
src=Path('.tmp/fts-choice')
dst=Path('knowledge/research/runtime-verification/evidence/fts-index-choice-20260915')
summary={}
for name in ('emulator','emulator-default','postgres','omni','managed'):
    rows=[json.loads(l) for l in (src/(name+'.jsonl')).read_text().splitlines()]
    groups={}
    plans=[]
    for r in rows:
        if 'error' in r:
            r['error']=re.sub(r', requestID = "[^"]+"','',r['error'])
        if r['kind']!='query' or r['label']=='preflight':continue
        if r['plan_requested']:
            scans=[]
            for n in r.get('plan',{}).get('planNodes',[]):
                m=n.get('metadata',{})
                if 'scan_type' in m:scans.append({'display':n.get('displayName'),'metadata':m,'links':n.get('childLinks',[])})
            plans.append({'label':r['label'],'error':r.get('error'),'scans':scans})
        elif not r.get('error'):
            key=r['label'].split('/')[1]
            groups.setdefault(key,set()).add((r['rows'],r['result_sha256']))
    for key,values in groups.items():assert len(values)==1,(name,key,values)
    (dst/(name+'.jsonl')).write_text(''.join(json.dumps(r,separators=(',',':'))+'\n' for r in rows))
    summary[name]={'records':len(rows),'successful_result_groups':{k:list(v)[0] for k,v in groups.items()},'plans':plans,'ddl':[r for r in rows if r['kind']=='ddl']}
(dst/'summary.json').write_text(json.dumps(summary,indent=2)+'\n')
for name, value in summary.items():
 print(name, value['records'],value['successful_result_groups'])
 for p in value['plans']:
  if p['label'].endswith('/auto'):print(p['label'],[(s['metadata'].get('scan_type'),s['metadata'].get('scan_target')) for s in p['scans']],p['error'][:90])
