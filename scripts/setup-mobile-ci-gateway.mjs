// @ts-check
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { appendFileSync, mkdirSync, mkdtempSync, realpathSync } from 'node:fs';
import { isAbsolute, join, resolve } from 'node:path';

const version = 'v7.0.9';
const platform = 'linux-amd64';
const repository = resolve(import.meta.dirname, '..');
const required = (name) => {
  const value = process.env[name];
  assert.ok(value, `Hosted Mobile checks require ${name}.`);
  return value;
};
assert.ok(process.env.GH_TOKEN, 'Hosted Mobile checks require the MPRLAB_GATEWAY_READ_TOKEN Actions secret.');
const runner = realpathSync(required('RUNNER_TEMP'));
const environmentFile = required('GITHUB_ENV');
assert.ok(isAbsolute(environmentFile), 'GITHUB_ENV must be an absolute path.');

/** @param {string} executable @param {string[]} args @param {boolean} capture */
function run(executable, args, capture = false) {
  const result = spawnSync(executable, args, {
    cwd: repository, env: { ...process.env, CI: '1' }, encoding: 'utf8',
    stdio: ['ignore', capture ? 'pipe' : 'inherit', 'inherit'],
  });
  if (result.error) throw result.error;
  if (result.status !== 0) process.exit(result.status ?? 1);
  return result.stdout;
}
const python = JSON.parse(run('python3', ['-c', 'import json,sys,venv; print(json.dumps({"executable":sys.executable,"version":list(sys.version_info[:2])}))'], true));
assert.ok(python.version[0] === 3 && python.version[1] >= 12 && python.version[1] < 15, 'Gateway installation requires Python 3.12 through 3.14.');
assert.ok(isAbsolute(python.executable), 'Select an absolute Python executable.');
const work = mkdtempSync(join(runner, 'loopaware-mobile-gateway-'));
const download = join(work, 'download'), seed = join(work, 'seed'), application = join(work, 'application');
for (const directory of [download, seed, application]) mkdirSync(directory);
const asset = `mprlab-gateway-${version}-${platform}.tar.gz`;
run('gh', ['release', 'download', version, '--repo', 'MarcoPoloResearchLab/mprlab-gateway', '--pattern', asset, '--dir', download]);
const archive = join(download, asset);
run('tar', ['-xzf', archive, '-C', seed]);
run(process.execPath, [join(repository, 'tests/mobile/android-signing-context.mjs'), '--write-installation-fixture', application]);
const installation = join(work, 'installation');
run(join(seed, 'bin/mprlab-gateway'), ['install', '--version', version, '--repo-root', application,
  '--archive', archive, '--install-root', installation, '--bin-root', join(work, 'commands'),
  '--cache-root', join(work, 'cache'), '--python', python.executable]);
const executable = join(installation, 'releases', version, platform, 'runtime/bin/mprlab-gateway');
const identity = JSON.parse(run(executable, ['version', '--json'], true));
assert.ok(identity.version === version && identity.platform === platform, 'The selected Gateway runtime must match the declared version and platform.');
appendFileSync(environmentFile, `MPRLAB_GATEWAY_EXECUTABLE=${executable}\n`);
console.log(`Hosted Mobile checks select Gateway ${version} for ${platform}.`);
