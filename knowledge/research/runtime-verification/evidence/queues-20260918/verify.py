#!/usr/bin/env python3
"""Recheck retained observations; this does not rerun managed Spanner."""
import json
from pathlib import Path
p = Path(__file__).resolve().parent

def records(name):
    raw = [json.loads(x) for x in (p / name).read_text().splitlines()]
    labeled = {r['label']: r for r in raw if 'label' in r}
    assert len(labeled) == sum('label' in r for r in raw), name
    return labeled

a = records('managed-r2.jsonl')
b = records('schema.jsonl')
d = records('lease-r2.jsonl')
for rs in (a, b, d):
    assert rs['cleanup-confirmation']['rows'] == [{'remaining': '0'}]
assert a['catalog-tables']['rows'][0]['TABLE_TYPE'] == 'QUEUE'
cols = {r['COLUMN_NAME']: r for r in a['catalog-columns']['rows']}
assert cols['DeliverTime']['SPANNER_TYPE'] == 'TIMESTAMP'
assert cols['DeliverTime']['IS_NULLABLE'] == 'NO'
assert cols['DeliverTime']['COLUMN_DEFAULT'] is None
assert len(a['queue-routines']['rows']) == 2
assert a['after-rollback']['rows'] == []
assert a['enqueue-immediate-and-future']['rows_modified'] == [2]
assert a['stale-receive']['code'] == 'InvalidArgument'
assert a['receive-immediate']['rows'][0]['MessageId'] == 'immediate'
assert a['receive-immediate']['column_types']['SpannerLeaseToken'] == 'STRING'
assert {r['MessageId'] for r in a['after-ack-rollback']['rows']} == {'future', 'immediate'}
assert a['ack-commit']['rows_modified'] == [1]
assert a['duplicate-ack']['code'] == 'OutOfRange'
assert a['future-not-received']['rows'] == [] and a['future-not-received']['code'] == 'OK'
assert a['future-still-stored']['rows'] == [{'MessageId': 'future'}]
assert a['enqueue-default-delivery']['code'] == 'OK'
assert a['renew-while-stream-open']['code'] == 'InvalidArgument'
assert a['renew-lease']['code'] == 'InvalidArgument'
for label in ['missing-payload', 'nullable-payload', 'integer-payload', 'extra-nonkey-column']:
    assert b[label]['code'] == 'InvalidArgument'
assert b['create-interleaved-json']['code'] == 'OK'
assert b['disabled-send-atomic']['code'] == 'InvalidArgument'
assert b['disabled-send-parent-rollback']['rows'] == []
assert b['atomic-enqueue']['code'] == 'OK'
assert b['disabled-delivery']['rows'] == [] and b['disabled-delivery']['code'] == 'OK'
assert len(b['disabled-delivery-stored']['rows']) == 1
assert b['receive-json']['rows'][0]['MessageId'] == 'm1'
assert b['atomic-process-ack']['code'] == 'OK'
assert b['duplicate-atomic-process-ack']['code'] == 'OutOfRange'
assert b['processed-once']['rows'] == [{'Id': '1', 'Processed': '1'}]
assert b['after-cascade']['rows'] == [{'remaining': '0'}]
assert d['typed-receive']['error'] == ''
assert d['typed-receive']['remaining_ms'] > 0
assert d['typed-receive']['token_length'] > 0
for label in ['renew-scalar-first', 'renew-new-scalar']:
    assert d[label]['code'] == 'InvalidArgument'
for label in ['renew-array-second', 'renew-new-array']:
    assert d[label]['code'] == 'OK' and len(d[label]['rows']) == 1
first = d['renew-array-second']['rows'][0]
second = d['renew-new-array']['rows'][0]
assert first['SpannerOldLeaseToken'] == 'sha256:' + d['typed-receive']['token_sha256']
assert first['SpannerNewLeaseToken'] == second['SpannerOldLeaseToken']
assert first['SpannerNewLeaseToken'] != first['SpannerOldLeaseToken']
assert first['SpannerLeaseExpirationTimestamp'] > d['typed-receive']['lease_expiration']
assert second['SpannerLeaseExpirationTimestamp'] > first['SpannerLeaseExpirationTimestamp']
print('PASS: queue DDL, catalog, delivery, transaction rollback, ack, controls, cascade, lease discrepancy and cleanup observations')
