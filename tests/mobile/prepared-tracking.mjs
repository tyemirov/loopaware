// @ts-check
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { spawnSync } from 'node:child_process';

const manifest = JSON.parse(await readFile('mobile/prepared/native-preparation.json', 'utf8'));
const selection = spawnSync('git', ['ls-files', '--cached', '--others', '--exclude-standard', '-z', '--', 'mobile/prepared'], { encoding: 'utf8' });
assert.equal(selection.status, 0, selection.stderr);
const selected = new Set(selection.stdout.split('\0'));
const missing = Object.keys(manifest.files).filter(name => !selected.has('mobile/prepared/' + name));
assert.deepEqual(missing, [], 'Every recorded native input must be selected by Git for the release snapshot.');
for (const name of Object.keys(manifest.files)) {
  const path = 'mobile/prepared/' + name;
  const stored = spawnSync('git', ['hash-object', '--path=' + path, path], { encoding: 'utf8' });
  const original = spawnSync('git', ['hash-object', '--no-filters', path], { encoding: 'utf8' });
  assert.equal(stored.status, 0, stored.stderr);
  assert.equal(original.status, 0, original.stderr);
  assert.equal(stored.stdout, original.stdout, `Git must preserve the recorded bytes for ${name}`);
}
console.info('Git selects and preserves every recorded production native input.');
