#!/usr/bin/env python3
"""Compare retained queue-present catalog surfaces and decode observations."""
import json
from pathlib import Path
p = Path(__file__).resolve().parent
repo = p.parents[4]
r = {x['label']: x for x in map(json.loads, (p/'catalog.jsonl').read_text().splitlines()) if 'label' in x}
for catalog, baseline, expected in [
    ('INFORMATION_SCHEMA', 'survey/infoschem/evidence/managed/20260916T171439Z-89b4ab9e231d.json', 310),
    ('SPANNER_SYS', 'survey/spannersys/evidence/managed/20260916T175741Z-58205fd318c3.json', 587),
]:
    current = [dict(table_name=x['TABLE_NAME'], column_name=x['COLUMN_NAME'],
                    spanner_type=x['SPANNER_TYPE'], ordinal_position=int(x['ORDINAL_POSITION']))
               for x in r['surface-'+catalog]['rows']]
    assert len(current) == expected
    assert current == json.loads((repo/baseline).read_text())['columns']
assert r['rows-TABLES']['rows'][0]['TABLE_TYPE'] == 'QUEUE'
assert len(r['rows-ROUTINES']['rows']) == 2
assert all(x['ROUTINE_TYPE'] == 'TABLE FUNCTION' and x['SPANNER_TYPE'].startswith('TABLE<')
           for x in r['rows-ROUTINES']['rows'])
assert [x['PARAMETER_NAME'] for x in r['rows-PARAMETERS']['rows']] == ['lease_tokens']
assert all(x['OPTION_NAME'] == 'queue_name' for x in r['rows-ROUTINE_OPTIONS']['rows'])
for label, count in [('decode-tables', 1), ('decode-routines', 2), ('decode-parameters', 1)]:
    assert r[label]['error'] == '' and r[label]['rows'] == count
# This capture deliberately preserves the pre-fix reproduction, not fixed behavior.
assert r['reconstruct-queue-metadata']['error'] == ''
assert r['reconstruct-queue-metadata']['ddl_count'] == 0
assert r['cleanup-confirmation']['rows'] == [{'remaining': '0'}]
for name, count in [('is-check.json', 310), ('sys-check.json', 587)]:
    report = json.loads((p/name).read_text())
    assert report['status'] == 'producer_changed_only' and not report['material_change']
    assert report['current']['column_count'] == count
    assert report['current']['surface_sha256'] == report['retained']['surface_sha256']
print('PASS: unchanged 310/587 column surfaces, queue metadata decoding, pre-fix omission reproduction and cleanup')
