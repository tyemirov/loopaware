// @ts-check
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { copyFileSync, existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
const repository = resolve(import.meta.dirname, '../..');
const fixture = mkdtempSync(join(tmpdir(), 'loopaware-ci-runtime-test-'));
const tool = String.raw`#!NODE_EXECUTABLE
const fs=require('node:fs'),path=require('node:path');
const role=path.basename(process.argv[1]),args=process.argv.slice(2);
const value=flag=>args[args.indexOf(flag)+1];
const trace=row=>fs.appendFileSync(process.env.PUBLIC_SETUP_TRACE,JSON.stringify(row)+'\n');
if(role==='python3'){
 console.log(JSON.stringify({executable:process.argv[1],version:process.env.PUBLIC_SETUP_CASE==='old-python'?[3,11]:[3,13]}));
}else if(role==='gh'){
 trace({role,args,tokenPresent:!!process.env.GH_TOKEN});
 if(process.env.PUBLIC_SETUP_CASE==='download-failure')process.exit(23);
 fs.writeFileSync(path.join(value('--dir'),value('--pattern')),'synthetic runtime archive');
}else if(role==='tar'){
 trace({role,args});
 const executable=path.join(value('-C'),'bin/mprlab-gateway');
 fs.mkdirSync(path.dirname(executable),{recursive:true});fs.copyFileSync(process.execPath,executable);fs.writeFileSync(executable,fs.readFileSync(process.argv[1]));
}else if(args[0]==='install'){
 trace({role:'installer',args});
 if(process.env.PUBLIC_SETUP_CASE==='install-failure')process.exit(17);
 const app=value('--repo-root');
 const selected=JSON.parse(fs.readFileSync(path.join(app,'.mprlab/deploy/resources.yml'))).mprlab_resources;
 if(selected.owner!=='loopaware'||!fs.readFileSync(path.join(app,selected.environment_file),'utf8').includes('synthetic-runtime-installation-input'))process.exit(96);
 const executable=path.join(value('--install-root'),'releases/v7.0.8/linux-amd64/runtime/bin/mprlab-gateway');
 fs.mkdirSync(path.dirname(executable),{recursive:true});fs.copyFileSync(process.execPath,executable);fs.writeFileSync(executable,fs.readFileSync(process.argv[1]));
}else if(args[0]==='version'){
 console.log(JSON.stringify({version:process.env.PUBLIC_SETUP_CASE==='wrong-version'?'v0.0.0':'v7.0.8',platform:'linux-amd64'}));
}else process.exit(95);
`.replace('NODE_EXECUTABLE', process.execPath);
try {
  for (const scenario of ['success', 'missing-token', 'old-python', 'download-failure', 'install-failure', 'wrong-version']) {
    const root = join(fixture, scenario), tools = join(root, 'tools'), runner = join(root, 'runner');
    mkdirSync(tools, { recursive: true }); mkdirSync(runner);
    for (const name of ['gh', 'tar', 'python3']) {
      const selected = join(tools, name); copyFileSync(process.execPath, selected); writeFileSync(selected, tool);
    }
    const jobEnvironment = join(root, 'job-environment'), trace = join(root, 'trace.jsonl');
    writeFileSync(jobEnvironment, 'EXISTING_JOB_VALUE=preserved\n');
    const env = { ...process.env, PATH: `${tools}:${process.env.PATH}`, RUNNER_TEMP: runner,
      GITHUB_ENV: jobEnvironment, PUBLIC_SETUP_TRACE: trace, PUBLIC_SETUP_CASE: scenario,
      GITHUB_TOKEN: 'ambient-token-must-not-be-used' };
    delete env.GH_TOKEN;
    if (scenario !== 'missing-token') env.GH_TOKEN = 'synthetic-gateway-read-token';
    const result = spawnSync('make', ['--no-print-directory', 'mobile-ci-gateway'], {
      cwd: repository, env, stdio: ['ignore', 'pipe', 'pipe'], encoding: 'utf8', maxBuffer: 1024 * 1024,
    });
    const output = result.stdout + result.stderr;
    const recorded = existsSync(trace) ? readFileSync(trace, 'utf8').trim().split('\n').map(line => JSON.parse(line)) : [];
    const exported = readFileSync(jobEnvironment, 'utf8');
    assert.ok(exported.startsWith('EXISTING_JOB_VALUE=preserved\n'));
    assert.ok(!output.includes('synthetic-gateway-read-token'));
    assert.ok(!exported.includes('synthetic-gateway-read-token'));
    assert.ok(!JSON.stringify(recorded).includes('synthetic-gateway-read-token'));
    if (scenario === 'success') {
      assert.equal(result.status, 0, output);
      assert.deepEqual(recorded.map(row => row.role), ['gh', 'tar', 'installer']);
      assert.deepEqual(recorded[0].args.slice(0, 6), ['release', 'download', 'v7.0.8', '--repo', 'MarcoPoloResearchLab/mprlab-gateway', '--pattern']);
      assert.equal(recorded[0].args[6], 'mprlab-gateway-v7.0.8-linux-amd64.tar.gz');
      assert.equal(recorded[0].tokenPresent, true);
      const install = recorded[2].args;
      assert.equal(install[0], 'install');
      for (const name of ['--version', '--repo-root', '--archive', '--install-root', '--bin-root', '--cache-root', '--python']) assert.ok(install.includes(name), name);
      assert.equal(install[install.indexOf('--version') + 1], 'v7.0.8');
      assert.match(exported, /MPRLAB_GATEWAY_EXECUTABLE=.*\/releases\/v7\.0\.8\/linux-amd64\/runtime\/bin\/mprlab-gateway\n$/);
    } else {
      assert.notEqual(result.status, 0, output);
      assert.equal(exported, 'EXISTING_JOB_VALUE=preserved\n');
      if (scenario === 'missing-token' || scenario === 'old-python') assert.deepEqual(recorded, []);
      if (scenario === 'missing-token') assert.match(output, /MPRLAB_GATEWAY_READ_TOKEN/);
      if (scenario === 'download-failure') assert.match(output, /Error 23/);
      if (scenario === 'install-failure') assert.match(output, /Error 17/);
      if (scenario === 'wrong-version') assert.match(output, /selected Gateway runtime/);
    }
    console.log(`Hosted runtime setup contract passed: ${scenario}.`);
  }
} finally { rmSync(fixture, { recursive: true, force: true }); }
