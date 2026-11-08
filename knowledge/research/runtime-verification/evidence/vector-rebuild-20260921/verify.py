import json
from pathlib import Path
base = Path(__file__).resolve().parent
summary = {}
for name in ('managed.jsonl', 'pg-managed.jsonl'):
    rows = [json.loads(line) for line in (base / name).read_text().splitlines()]
    records = {row['label']: row for row in rows}
    for label in ('preflight', 'create-table', 'seed', 'create-vector',
                  'rebuild-submitted', 'ann-before', 'ann-after-submit-before-wait',
                  'write-after-submit-before-wait', 'ann-after', 'definition-before',
                  'definition-after', 'create-regular', 'drop-regular', 'drop-vector',
                  'drop-table', 'cleanup'):
        assert records[label]['code'] == 'OK', (name, label, records[label])
    assert records['cleanup']['rows'] == []
    assert records['definition-before']['ddl'] == records['definition-after']['ddl']
    key = 'Id' if name == 'managed.jsonl' else 'id'
    marker = 'Marker' if name == 'managed.jsonl' else 'marker'
    assert [r[key] for r in records['ann-before']['rows']] == ['0', '1', '2']
    assert [r[key] for r in records['ann-after']['rows']] == ['0', '1', '2']
    assert records['ann-after']['rows'][0][marker] == '9999'
    assert records['missing-index']['code'] == 'NotFound'
    assert records['regular-index']['code'] == 'NotFound'
    negative = 'extra-options' if name == 'managed.jsonl' else 'missing-concurrently'
    assert records[negative]['code'] == 'InvalidArgument'
    if name == 'pg-managed.jsonl':
        assert records['drop-pg-database']['code'] == 'OK'
    result = records['rebuild-complete']
    assert result['code'] in ('OK', 'Unimplemented'), result
    summary[name] = {'rebuild_code': result['code'], 'error': result['error'],
                     'ddl_unchanged': True, 'ann_and_write_verified': True,
                     'cleanup_remaining': 0, 'completed_at': result['at']}
print(json.dumps(summary, indent=2))
