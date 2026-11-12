"""Verify retained projections against their pinned historical Git inputs, offline."""
import collections
import gzip
import hashlib
from html.parser import HTMLParser
import json
from pathlib import Path
import subprocess

here = Path(__file__).resolve().parent
root = here.parents[4]
manifest = json.loads((here / 'disposition.json').read_text())


def digest(data):
    return hashlib.sha256(data).hexdigest()


def project(value):
    if isinstance(value, list):
        return [project(v) for v in value]
    if not isinstance(value, dict):
        return value
    if 'full_name' in value and 'owner' in value and 'fork' in value:
        return {k: value[k] for k in ['id', 'full_name', 'html_url', 'private', 'fork', 'default_branch'] if k in value}
    if 'login' in value and 'avatar_url' in value:
        return {k: value[k] for k in ['login', 'id', 'html_url', 'type'] if k in value}
    result = {}
    for key, item in value.items():
        if key == '_links' or (key.endswith('_url') and key not in {'html_url', 'diff_url', 'patch_url', 'raw_url', 'blob_url'}):
            continue
        if key == 'patch' and 'filename' in value and 'status' in value:
            continue
        result[key] = project(item)
    if {'sha', 'url', 'tree', 'truncated'} <= value.keys():
        result['tree'] = [{k: v for k, v in entry.items() if k != 'url'} for entry in value['tree']]
    return result


def go_projection(data):
    parsed = []
    for line in data.decode().splitlines():
        try:
            row = json.loads(line)
        except json.JSONDecodeError:
            row = None
        parsed.append((line, row))
    rows = [row for _, row in parsed if row is not None]
    assert all(isinstance(row, dict) and 'Action' in row and 'Package' in row for row in rows)
    failed = {(r['Package'], r.get('Test')) for r in rows if r['Action'] == 'fail' and r.get('Test')}
    # Fail closed for observation-only tests: a passing t.Log can be the only
    # primary record of a SQL rejection, scan path or metadata value.
    def framework_output(text):
        stripped = text.lstrip()
        return (stripped.startswith(('=== RUN   ', '=== PAUSE ', '=== CONT  ', '=== NAME  ', '--- PASS: '))
                or text == 'PASS\n' or text.startswith(('ok  \t', '?   \t')))
    retained = [line for line, row in parsed if row is None or row['Action'] != 'output'
                or (row['Package'], row.get('Test')) in failed
                or not framework_output(row.get('Output', ''))]
    return ('\n'.join(retained) + '\n').encode()


class ArticleText(HTMLParser):
    def __init__(self):
        super().__init__()
        self.active = 0
        self.excluded = 0
        self.text = []

    def handle_starttag(self, tag, attrs):
        if tag == 'article':
            self.active += 1
        if self.active and tag in {'script', 'style'}:
            self.excluded += 1

    def handle_endtag(self, tag):
        if tag == 'article':
            self.active -= 1
        if self.active and tag in {'script', 'style'}:
            self.excluded = max(0, self.excluded - 1)

    def handle_data(self, data):
        if self.active and not self.excluded:
            self.text.append(data)


def article(data):
    parser = ArticleText()
    parser.feed(gzip.decompress(data).decode())
    return ' '.join(' '.join(parser.text).split()).encode()


counts = collections.Counter()
protected = set(json.loads((root / 'plancontract/planvocab/catalog_source.json').read_text())['info']['local_evidence'])
assert not (protected & {x['path'] for x in manifest['changes']})
for record in manifest['changes']:
    path = root / record['path']
    original = subprocess.check_output(['git', 'show', manifest['source_commit'] + ':' + record['path']], cwd=root)
    assert len(original) == record['original_bytes'] and digest(original) == record['original_sha256'], record['path']
    if record['retained_sha256'] is None:
        assert not path.exists(), record['path']
        canonical = (root / record['canonical']).read_bytes()
        if record['kind'] == 'exact_duplicate':
            assert original == canonical, record['path']
        else:
            assert record['kind'] == 'article_text_duplicate'
            assert article(original) and article(original) == article(canonical)
            assert digest(article(original)) == record['normalized_article_sha256']
    else:
        retained = path.read_bytes()
        assert len(retained) == record['retained_bytes'] and digest(retained) == record['retained_sha256'], record['path']
        if record['kind'] == 'github_metadata_projection':
            assert project(json.loads(original)) == json.loads(retained), record['path']
        else:
            assert record['kind'] == 'go_test_event_projection'
            assert go_projection(original) == retained, record['path']
    counts[record['kind']] += 1
assert sum(counts.values()) == manifest['totals']['files']
assert manifest['totals']['saved_bytes'] == sum(x['original_bytes'] - x['retained_bytes'] for x in manifest['changes'])
print(json.dumps({'status': 'pass', 'files': sum(counts.values()), 'kinds': dict(counts), 'saved_bytes': manifest['totals']['saved_bytes']}))
