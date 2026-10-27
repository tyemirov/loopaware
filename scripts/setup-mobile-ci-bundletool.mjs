// @ts-check
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { appendFileSync, mkdirSync, mkdtempSync, realpathSync, writeFileSync } from 'node:fs';
import { isAbsolute, join, resolve } from 'node:path';
const version = '1.18.3';
const repository = resolve(import.meta.dirname, '..');
assert.ok(process.env.RUNNER_TEMP && isAbsolute(process.env.RUNNER_TEMP), 'Hosted bundletool setup requires an absolute RUNNER_TEMP.');
const runner = realpathSync(process.env.RUNNER_TEMP);
const jobPath = process.env.GITHUB_PATH;
assert.ok(jobPath && isAbsolute(jobPath), 'Hosted bundletool setup requires an absolute GITHUB_PATH.');
assert.ok(process.env.JAVA_HOME && isAbsolute(process.env.JAVA_HOME), 'Set up the selected Java toolchain before bundletool.');
/** @param {string} executable @param {string[]} args @param {boolean} capture */
function run(executable, args, capture = false) {
  const result = spawnSync(executable, args, { cwd: repository, env: process.env,
    encoding: 'utf8', stdio: ['ignore', capture ? 'pipe' : 'inherit', 'inherit'] });
  if (result.error) throw result.error;
  if (result.status !== 0) process.exit(result.status ?? 1);
  return result.stdout;
}
const work = mkdtempSync(join(runner, 'loopaware-bundletool-'));
const source = join(work, 'package'), prefix = join(work, 'installation');
mkdirSync(source);
const jar = join(source, 'bundletool.jar');
run('curl', ['--fail', '--silent', '--show-error', '--location', '--output', jar,
  `https://github.com/google/bundletool/releases/download/${version}/bundletool-all-${version}.jar`]);
writeFileSync(join(source, 'package.json'), JSON.stringify({ name: 'loopaware-ci-bundletool', version,
  private: true, type: 'module', bin: { bundletool: './bundletool.mjs' } }));
writeFileSync(join(source, 'bundletool.mjs'), `#!/usr/bin/env node
import { spawnSync } from 'node:child_process';
import { join } from 'node:path';
const result = spawnSync(join(process.env.JAVA_HOME, 'bin/java'), ['-jar', join(import.meta.dirname, 'bundletool.jar'), ...process.argv.slice(2)], { stdio: 'inherit' });
if (result.error) throw result.error;
process.exit(result.status ?? 1);
`);
run('npm', ['install', '--global', '--prefix', prefix, '--offline', '--ignore-scripts', '--no-audit', '--no-fund', source]);
const bin = join(prefix, 'bin');
assert.equal(run(join(bin, 'bundletool'), ['version'], true).trim(), version, 'Install the declared official bundletool version.');
appendFileSync(jobPath, `${bin}\n`);
console.log(`Hosted Mobile checks select official bundletool ${version}.`);
