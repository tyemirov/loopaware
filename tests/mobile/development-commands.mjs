// @ts-check
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { copyFileSync, existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync, symlinkSync } from 'node:fs';
import { resolve, join } from 'node:path';
import { tmpdir } from 'node:os';
import test from 'node:test';

const root = resolve(import.meta.dirname, '../..');

/** A public package fixture controls the native SDK boundary, while Make and npm stay real.
 * @param {(directory: string, invoke: (target: string, extra?: Record<string,string>) => import('node:child_process').SpawnSyncReturns<string>) => void} check
 */
function fixture(check) {
  const directory = mkdtempSync(join(tmpdir(), 'loopaware-development-'));
  try {
    mkdirSync(join(directory, 'mobile/scripts'), { recursive: true });
    mkdirSync(join(directory, 'home'));
    mkdirSync(join(directory, 'bin'));
    copyFileSync(join(root, 'Makefile'), join(directory, 'Makefile'));
    const scripts = Object.fromEntries(['ios:dev-build', 'android:dev-build', 'ios', 'android'].map(name => [name, 'node scripts/native-boundary.cjs ' + name]));
    const metadata = { name: 'public-development-fixture', version: '1.0.0', private: true, scripts };
    writeFileSync(join(directory, 'mobile/package.json'), JSON.stringify(metadata));
    writeFileSync(join(directory, 'mobile/package-lock.json'), JSON.stringify({ name: metadata.name, version: metadata.version, lockfileVersion: 3, requires: true, packages: { '': metadata } }));
    writeFileSync(join(directory, 'mobile/scripts/native-boundary.cjs'), `const fs=require('node:fs');
const phase=process.argv[2];
fs.appendFileSync(process.env.PUBLIC_NATIVE_LOG,phase+'\\n');
if(process.env.PUBLIC_NATIVE_FAIL===phase)process.exit(17);
`);
    symlinkSync(process.execPath, join(directory, 'bin/adb'));
    symlinkSync(process.execPath, join(directory, 'bin/xcrun'));
    writeFileSync(join(directory, 'simctl'), 'process.exit(1);');
    writeFileSync(join(directory, 'reverse'), 'process.exit(0);');
    const invoke = (target, extra = {}) => spawnSync('make', ['--no-print-directory', target], {
      cwd: directory, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'],
      env: { PATH: join(directory, 'bin') + ':' + process.env.PATH, HOME: join(directory, 'home'), TMPDIR: directory,
        PUBLIC_NATIVE_LOG: join(directory, 'native.log'), LOOPAWARE_MOBILE_METRO_PORT: '8199', ...extra },
    });
    check(directory, invoke);
  } finally { rmSync(directory, { recursive: true, force: true }); }
}

test('explicit mobile installation reconciles existing node_modules with the selected lock', () => fixture((directory, invoke) => {
  mkdirSync(join(directory, 'mobile/node_modules'), { recursive: true });
  writeFileSync(join(directory, 'mobile/node_modules/stale-manager-input'), 'old installation');
  const result = invoke('mobile-install');
  assert.equal(result.status, 0, result.stderr);
  assert.equal(existsSync(join(directory, 'mobile/node_modules/stale-manager-input')), false, 'canonical npm ci must consume the current lock even when node_modules exists');
}));

for (const platform of ['ios', 'android']) {
  test(platform + ' runs the canonical native build before launch', () => fixture((directory, invoke) => {
    const result = invoke('run-' + platform);
    assert.equal(result.status, 0, result.stdout + result.stderr);
    assert.deepEqual(readFileSync(join(directory, 'native.log'), 'utf8').trim().split('\n'), [platform + ':dev-build', platform]);
  }));
  test(platform + ' preserves native build failure and does not launch', () => fixture((directory, invoke) => {
    const result = invoke('run-' + platform, { PUBLIC_NATIVE_FAIL: platform + ':dev-build' });
    assert.notEqual(result.status, 0);
    assert.match(result.stdout + result.stderr, /17/);
    assert.deepEqual(readFileSync(join(directory, 'native.log'), 'utf8').trim().split('\n'), [platform + ':dev-build']);
  }));
}
