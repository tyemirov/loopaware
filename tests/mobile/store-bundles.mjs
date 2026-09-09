// @ts-check
import assert from 'node:assert/strict';
import { mkdtemp, readFile, rm } from 'node:fs/promises';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';
import { spawnSync } from 'node:child_process';
const source = resolve('mobile/prepared');
const temporary = await mkdtemp(join(tmpdir(), 'loopaware-store-bundles-'));
try {
  const install = spawnSync('npm', ['ci', '--include=dev'], { cwd: source, encoding: 'utf8' });
  assert.equal(install.status, 0, install.stderr);
  for (const platform of ['android', 'ios']) {
    const output = join(temporary, platform + '.bundle');
    const command = platform === 'android' ? 'node_modules/@react-native-community/cli/build/bin.js' : 'node_modules/react-native/scripts/bundle.js';
    const result = spawnSync(process.execPath, [command, 'bundle', '--platform', platform, '--dev', 'false', '--entry-file', 'index.ts', '--bundle-output', output, '--assets-dest', join(temporary, platform), '--max-workers', '2'], { cwd: source, encoding: 'utf8', env: { ...process.env, NODE_ENV: 'production' } });
    assert.equal(result.status, 0, result.stdout + result.stderr);
    assert.ok((await readFile(output)).length > 1000, 'Store bundle must contain application code.');
  }
  console.info('Android and iOS production JavaScript bundles compile through the native build entry points.');
} finally {
  await rm(temporary, { recursive: true, force: true });
}
