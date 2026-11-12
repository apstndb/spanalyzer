"""Prepare pinned FK reproducer sources/plans locally; never perform RPCs."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess

here = Path(__file__).resolve().parent


def make_plan(variant):
    plan = json.loads((here / 'plan.json').read_text())
    by = {step['label']: step for step in plan['steps']}
    if variant == 'parent-first':
        plan['steps'] = [by['parents'], by['data/seed'], *[s for s in plan['steps'] if s['label'] not in {'parents', 'data/seed'}]]
    elif variant == 'controls':
        labels = ['parents', 'data/seed']
        for case in ['not-null-pk', 'nullable-pk', 'cyclic-covered-pk', 'cyclic-inverse-pk']:
            labels.extend([case + '/ddl', case + '/indexes'])
        labels.extend(s['label'] for s in plan['steps'] if s['label'].startswith('data/') and s['label'] != 'data/seed')
        labels.append('final/ddl-export')
        plan['steps'] = [by[label] for label in labels]
    elif variant == 'ordering':
        plan = json.loads((here / 'ordering-controls.json').read_text())
    else:
        assert variant == 'schema-first'
    return plan


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('output', type=Path)
    parser.add_argument('--variant', choices=['schema-first', 'parent-first', 'controls', 'ordering'], default='parent-first')
    args = parser.parse_args()
    target = args.output.resolve()
    target.mkdir(parents=True, exist_ok=True)
    assert target != here, 'Use checkout scratch, not the evidence directory'
    producer = json.loads((here / 'producer.json').read_text())
    for name, item in producer['producers'].items():
        source = subprocess.check_output(['git', 'show', producer['source_commit'] + ':' + item['path']])
        assert hashlib.sha256(source).hexdigest() == item['source_sha256']
        patch = here / item['patch']
        assert hashlib.sha256(patch.read_bytes()).hexdigest() == item['patch_sha256']
        path = target / (name + '.go')
        # Exclusive creation prevents overwriting another unfinished experiment.
        with path.open('xb') as stream:
            stream.write(source)
        subprocess.run(['patch', '-p1', '--batch', '-i', str(patch)], cwd=target, check=True)
        assert hashlib.sha256(path.read_bytes()).hexdigest() == item['prepared_sha256']
    plan = make_plan(args.variant)
    data = (json.dumps(plan, indent=2) + '\n').encode()
    with (target / 'plan.json').open('xb') as stream:
        stream.write(data)
    print(json.dumps({'variant': args.variant, 'steps': len(plan['steps']), 'plan_sha256': hashlib.sha256(data).hexdigest(), 'network_or_database_actions': 0}))


if __name__ == '__main__':
    main()
