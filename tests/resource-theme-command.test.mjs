// @ts-check
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { cp, mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { test } from '../scripts/native-test.mjs';

const repository = path.resolve(import.meta.dirname, '..');

test('the resource theme Make target installs locked dependencies in a clean shard', async () => {
  const fixture = await mkdtemp(path.join(tmpdir(), 'loopaware-resource-theme-command-'));
  try {
    await mkdir(path.join(fixture, 'tests'));
    await mkdir(path.join(fixture, 'scripts'));
    await mkdir(path.join(fixture, '.cache'));
    for (const name of ['Makefile', 'tests/package.json', 'tests/package-lock.json', 'scripts/run-node-tests.mjs', 'scripts/native-test.mjs']) {
      await cp(path.join(repository, name), path.join(fixture, name));
    }
    // Exercise the real Make recipe with an observable public package import.
    await writeFile(path.join(fixture, 'tests/resource-theme.test.js'), `import { chromium } from '@playwright/test';
import { writeFileSync } from 'node:fs';
if (typeof chromium.launch !== 'function') throw new Error('missing Chromium entry point');
writeFileSync(new URL('./resource-theme-command-passed', import.meta.url), 'passed');
`);
    const environment = { ...process.env, CI: '1', npm_config_yes: 'false' };
    delete environment.NODE_TEST_CONTEXT;
    const result = spawnSync('make', ['test-resource-theme'], {
      cwd: fixture,
      env: environment,
      stdio: ['ignore', 'pipe', 'pipe'],
      encoding: 'utf8',
      timeout: 120_000,
    });
    assert.ifError(result.error);
    assert.equal(result.status, 0, result.stdout + result.stderr);
    assert.equal(await readFile(path.join(fixture, 'tests/resource-theme-command-passed'), 'utf8').catch(() => result.stdout + result.stderr), 'passed');
  } finally {
    await rm(fixture, { recursive: true, force: true });
  }
});
