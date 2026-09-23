import json
from pathlib import Path
p=Path('knowledge/research/runtime-verification/evidence/fts-index-choice-20260915')
expected={'p0':1000,'p1':10,'p2':10,'p3':1000,'p4':9990}
shared={}
for name in ('emulator-default','omni','managed'):
 rows=[json.loads(l) for l in (p/(name+'.jsonl')).read_text().splitlines()]
 queries=[r for r in rows if r['kind']=='query' and r['label']!='preflight']
 assert len(queries)==(40 if name=='emulator-default' else 200),(name,len(queries))
 for r in queries:
  version,predicate,access=r['label'].split('/')
  should_fail=r['plan_requested'] if name=='emulator-default' else version=='v4' and access.endswith('Ngrams')
  assert bool(r['error'])==should_fail,(name,r['label'],r['error'])
  if should_fail:
   assert ('query plan unavailable' if name=='emulator-default' else 'query optimizer version 6 or above') in r['error']
  elif not r['plan_requested']:
   assert r['rows']==expected[predicate],(name,r)
   pair=(r['rows'],r['result_sha256'])
   assert shared.setdefault(predicate,pair)==pair
 for label in ('cleanup-search','cleanup-secondary','cleanup-documents'):
  matches=[r for r in rows if r.get('label')==label]
  assert len(matches)==1 and matches[0]['error']=='',(name,label)
print('PASS: all 440 query records have expected status; successful results match expected counts and hashes across three runtimes; fixture cleanup DDL succeeded.')
