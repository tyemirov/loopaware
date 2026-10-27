// @ts-check
import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, readFileSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { spawnSync } from 'node:child_process';
import { publicFixture } from './preparation.mjs';

const manifest = readFileSync('.mprlab/deploy/resources.yml', 'utf8');
const selected = /build:\s*\n\s+android: [^\n]+\n\s+ios: ([^\n]+)/.exec(manifest);
assert.ok(selected, 'The manifest must select an iOS adapter.');
const fixture = publicFixture();
const adapter = join(fixture.mobile, 'scripts', selected[1].trim().split('/').at(-1));
const root = fixture.directory;
const recorded = fixture.record(); assert.equal(recorded.status, 0, recorded.stderr);
try {
  const source = fixture.prepared;
  writeFileSync(join(fixture.mobile, 'mobile-build-operation'), `
    let input = '';
    process.stdin.on('data', chunk => input += chunk);
    process.stdin.on('end', () => {
      process.stdout.write(input);
      process.stderr.write('native build diagnostic\\n');
      process.exit(17);
    });
  `);
  const output = join(root, 'ios.ipa');
  const args = ['--mobile-dir', source, '--output', output, '--manifest', join(root, 'ios.json'), '--release-timestamp', '2026-09-10T09:12:53Z'];
  const env = { ...fixture.environment, MPRLAB_APP_ROOT: root, MPRLAB_GATEWAY_EXECUTABLE: process.execPath, MPRLAB_ARTIFACT_VERSION: '1.2.0' };
  const result = spawnSync(process.execPath, [adapter, ...args], { env, encoding: 'utf8' });
  assert.equal(result.status, 17, result.stderr);
  assert.match(result.stderr, /native build diagnostic/);
  const request = JSON.parse(result.stdout);
  assert.equal(request.schema_version, 3);
  assert.equal(Object.hasOwn(request, 'preparation_manifest'), false);
  assert.equal(request.verify_script, 'scripts/verify-store-preparation.mjs');
  assert.equal(request.platform, 'ios');
  assert.equal(request.application_identifier, 'com.mprlab.loopaware');
  assert.equal(request.version, '1.1.0');
  assert.equal(request.build_number, '211194773');
  assert.equal(request.output, output);
  assert.notEqual(request.source_root, source);
  assert.match(request.source_root, /loopaware-native-source-/);
  assert.equal(request.ios.owner, 'loopaware');
  assert.equal(request.ios.export_intent, 'internal-only');
  assert.equal(request.ios.workspace, 'ios/LoopAware.xcworkspace');
  assert.equal(request.ios.certificate_environment, 'LOOPAWARE_APPLE_CERTIFICATE_BASE64');
  assert.equal(request.ios.provisioning.private_key_environment, 'APP_STORE_CONNECT_API_KEY_PATH');
  assert.equal(request.android, undefined);
  for (const [name, value, message] of [['--output', join(root, 'ios.zip'), /IPA or AAB/], ['--manifest', join(root, 'other.json'), /gateway platform path/]]) {
    const invalid = [...args];
    invalid[invalid.indexOf(name) + 1] = value;
    const rejected = spawnSync(process.execPath, [adapter, ...invalid], { env, encoding: 'utf8' });
    assert.notEqual(rejected.status, 0);
    assert.match(rejected.stderr, message);
    assert.equal(rejected.stdout, '');
  }
  console.info('The selected iOS adapter forwards a native internal-TestFlight build and preserves its exit status.');
} finally {
  rmSync(root, { recursive: true, force: true });
}
