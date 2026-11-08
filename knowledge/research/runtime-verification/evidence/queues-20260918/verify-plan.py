#!/usr/bin/env python3
"""Check retained managed PLAN/PROFILE and cleanup; no live requests."""
import json
from pathlib import Path
rs = [json.loads(x) for x in Path(__file__).with_name('plan.jsonl').read_text().splitlines()]
probes = {(r['mode'], r['label']): r for r in rs if r.get('kind') == 'plan-probe'}
assert len(probes) == 14
assert sum(k[0] == 'PLAN' for k in probes) == 11

def relational(r):
    return [n for n in r['plan']['planNodes'] if n.get('kind') == 'RELATIONAL']

for key, r in probes.items():
    if key[1] in ['receive-filter-key', 'receive-filter-payload', 'receive-limit', 'receive-count', 'receive-join', 'receive-cte']:
        assert r['code'] == 'Unimplemented' and 'Queue TVF must not be combined' in r['error']
        continue
    assert r['code'] == 'OK'
    nodes = relational(r)
    names = [n['displayName'] for n in nodes]
    if key[1] == 'ordinary':
        assert names == ['Distributed Union', 'Distributed Union', 'Serialize Result', 'Scan']
        assert nodes[-1]['metadata']['scan_type'] == 'TableScan'
    else:
        assert names == ['Serialize Result', 'TVF', 'Unit Relation']
        assert all(n['metadata']['execution_method'] == 'Row' for n in nodes)
        assert nodes[1]['metadata'] == {'execution_method': 'Row'}
        assert any(l['childIndex'] == nodes[2]['index'] for l in nodes[1]['childLinks'])
    if key[0] == 'PROFILE':
        assert r['stats']['optimizer_version'] == '9' and r['rows'] == 0
arr = probes['PLAN', 'renew-array']['plan']['planNodes']
scalar = probes['PLAN', 'renew-scalar']['plan']['planNodes']
assert any(n.get('metadata', {}).get('type') == 'array' for n in arr)
assert any(n['displayName'] == 'Array Constructor' for n in scalar)
assert rs[-1]['label'] == 'cleanup-confirmation' and rs[-1]['rows'] == [{'remaining': '0'}]
print('PASS: 11 PLAN requests, 3 completed PROFILE requests, TVF shape, composition rejection and cleanup')
