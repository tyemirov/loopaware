// @ts-check
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { copyFileSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
const repository = resolve(import.meta.dirname, '../..');
const fixture = mkdtempSync(join(tmpdir(), 'loopaware-bundletool-contract-'));
const tool = `#!${process.execPath}\nconst fs=require('node:fs'),path=require('node:path');\nconst args=process.argv.slice(2);\nif(path.basename(process.argv[1])==='curl'){\n if(process.env.PUBLIC_SETUP_CASE==='download-failure')process.exit(23);\n fs.writeFileSync(args[args.indexOf('--output')+1],'synthetic download; never executed');\n}else process.exit(17);\n`;
try {
  for (const scenario of ['download-failure', 'install-failure']) {
    const selected = join(fixture, scenario), tools = join(selected, 'tools'), runner = join(selected, 'runner');
    mkdirSync(tools, { recursive: true }); mkdirSync(runner);
    for (const name of ['curl', 'npm']) {
      const executable = join(tools, name); copyFileSync(process.execPath, executable); writeFileSync(executable, tool);
    }
    const pathFile = join(selected, 'job-path'); writeFileSync(pathFile, 'EXISTING_PATH\n');
    const result = spawnSync('make', ['--no-print-directory', 'mobile-ci-bundletool'], {
      cwd: repository, encoding: 'utf8', env: { ...process.env, RUNNER_TEMP: runner, GITHUB_PATH: pathFile,
        JAVA_HOME: '/synthetic/not-executed-jdk', PATH: `${tools}:${process.env.PATH}`, PUBLIC_SETUP_CASE: scenario },
    });
    assert.notEqual(result.status, 0);
    assert.match(result.stdout + result.stderr, new RegExp(`Error ${scenario === 'download-failure' ? 23 : 17}`));
    assert.equal(readFileSync(pathFile, 'utf8'), 'EXISTING_PATH\n');
    console.log(`Hosted bundletool setup contract passed: ${scenario}.`);
  }
} finally { rmSync(fixture, { recursive: true, force: true }); }
