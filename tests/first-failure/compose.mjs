// @ts-check
import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { mkdir, readFile, realpath, writeFile } from 'node:fs/promises';
import { dirname, isAbsolute, join } from 'node:path';
import { pathToFileURL } from 'node:url';

/** Real, cached-image Compose resources; only the terminal CLI status is injected. */
export async function composeFirstFailure({ repository, owner, invoke, absent, results, cleanupFailure = 'guardian' }) {
  assert.ok(['guardian', 'main'].includes(cleanupFailure));
  const root = join(owner, 'compose');
  const home = join(root, 'home');
  const project = `loopaware-b148-${process.pid}`;
  const image = 'alpine:3.24';
  const resourceFile = join(root, 'resources.json');
  const injected = join(root, 'injected-down');
  const downCount = join(root, 'terminal-down-count');
  const failed = join(root, 'suite-failed');
  const later = join(root, 'later-case');
  for (const dir of ['tests/scripts', 'tests/configs', 'launcher', 'home']) await mkdir(join(root, dir), { recursive: true });
  const env = { PATH: process.env.PATH, HOME: home, TMPDIR: owner, CI: '1', LOOPAWARE_PLAYWRIGHT_CHANNEL: 'chromium', LOOPAWARE_TEST_SUITE: 'test' };
  const dockerPath = await invoke(['-c', 'command -v docker'], root, '/bin/sh', env);
  assert.equal(dockerPath.code, 0, dockerPath.output);
  const docker = dockerPath.output.trim();
  assert.ok(docker.startsWith('/'));
  const pluginMetadata = await invoke(['info', '--format', '{{json .ClientInfo.Plugins}}'], root, docker);
  assert.equal(pluginMetadata.code, 0, pluginMetadata.output);
  const composePlugins = JSON.parse(pluginMetadata.output).filter(plugin => plugin.Name === 'compose');
  assert.equal(composePlugins.length, 1, 'One installed Compose plugin is required');
  assert.ok(isAbsolute(composePlugins[0].Path));
  const pluginDirectory = dirname(await realpath(composePlugins[0].Path));
  const dockerConfig = join(root, 'docker-config');
  await mkdir(dockerConfig);
  await writeFile(join(dockerConfig, 'config.json'), JSON.stringify({ cliPluginsExtraDirs: [pluginDirectory] }));
  env.DOCKER_CONFIG = dockerConfig;
  const composeVersion = await invoke(['compose', 'version'], root, docker, env);
  assert.equal(composeVersion.code, 0, composeVersion.output);
  const imageCheck = await invoke(['image', 'inspect', '--format', '{{json .RepoTags}}', image], root, docker, env);
  assert.equal(imageCheck.code, 0, `Required cached image unavailable; no build or pull: ${imageCheck.output}`);
  async function resources() {
    const found = {};
    for (const [kind, args] of [['containers', ['ps', '-aq']], ['networks', ['network', 'ls', '-q']], ['volumes', ['volume', 'ls', '-q']]]) {
      const result = await invoke([...args, '--filter', `label=com.docker.compose.project=${project}`], root, docker, env);
      assert.equal(result.code, 0, result.output);
      found[kind] = result.output.trim().split('\n').filter(Boolean);
    }
    return found;
  }
  assert.deepEqual(await resources(), { containers: [], networks: [], volumes: [] });
  const composeFile = join(root, 'tests/docker-compose.yml');
  await writeFile(composeFile, `services:\n  loopaware-api:\n    image: ${image}\n    pull_policy: never\n    init: true\n    entrypoint: ["/bin/sh", "-c", "sleep 3600"]\n    ports: ["127.0.0.1::8080"]\n    volumes: ["owned:/owned"]\nvolumes:\n  owned: {}\n`);
  const source = await readFile(join(repository, 'tests/scripts/run-integration.sh'), 'utf8');
  const projectAssignment = 'export COMPOSE_PROJECT_NAME=loopaware-integration';
  assert.equal(source.split(projectAssignment).length, 2);
  await writeFile(join(root, 'tests/scripts/run-integration.sh'), source.replace(projectAssignment, `export COMPOSE_PROJECT_NAME=${project}`));
  await writeFile(join(root, 'tests/package.json'), JSON.stringify({ name: 'loopaware-b148-compose-fixture', version: '1.0.0', private: true, scripts: { test: `${JSON.stringify(process.execPath)} ${JSON.stringify(join(repository, 'scripts/run-node-tests.mjs'))} ${JSON.stringify(join(root, 'tests/first.test.mjs'))}` } }));
  await writeFile(join(root, 'launcher/package.json'), JSON.stringify({ name: 'loopaware-b148-docker-forwarder', version: '1.0.0', private: true, type: 'module', bin: { docker: 'docker.mjs' } }));
  await writeFile(join(root, 'launcher/docker.mjs'), `#!/usr/bin/env node
import{spawnSync}from'node:child_process';import{existsSync,readFileSync,writeFileSync}from'node:fs';
const args=process.argv.slice(2);const result=spawnSync(${JSON.stringify(docker)},args,{stdio:'inherit'});
if(result.error)throw result.error;
if(result.status===0&&args.includes('down')&&existsSync(${JSON.stringify(failed)})){const count=existsSync(${JSON.stringify(downCount)})?Number(readFileSync(${JSON.stringify(downCount)},'utf8'))+1:1;writeFileSync(${JSON.stringify(downCount)},String(count));if(count===${cleanupFailure === 'guardian' ? 1 : 2}){writeFileSync(${JSON.stringify(injected)},'29');process.exit(29);}}
if(result.signal)process.kill(process.pid,result.signal);else process.exit(result.status??1);
`);
  const prefix = join(root, 'prefix');
  let result = await invoke(['install', '--offline', '--ignore-scripts', '--no-audit', '--no-fund', '--prefix', prefix, join(root, 'launcher')], root, 'npm', env);
  assert.equal(result.code, 0, result.output);
  result = await invoke(['install', '--package-lock-only', '--offline', '--ignore-scripts', '--no-audit', '--no-fund', '--prefix', join(root, 'tests')], root, 'npm', env);
  assert.equal(result.code, 0, result.output);
  await writeFile(join(root, 'tests/first.test.mjs'), `import test from ${JSON.stringify(pathToFileURL(join(repository, 'scripts/native-test.mjs')).href)};import assert from'node:assert/strict';import{spawnSync}from'node:child_process';import{writeFileSync}from'node:fs';
function selected(args){const result=spawnSync(${JSON.stringify(docker)},args,{encoding:'utf8'});assert.equal(result.status,0,result.stderr);return result.stdout.trim().split('\\n').filter(Boolean);}
test('first-compose-controlled-failure',()=>{const label='label=com.docker.compose.project=${project}';const resources={containers:selected(['ps','-aq','--filter',label]),networks:selected(['network','ls','-q','--filter',label]),volumes:selected(['volume','ls','-q','--filter',label])};assert.equal(resources.containers.length,1);assert.equal(resources.networks.length,1);assert.equal(resources.volumes.length,1);writeFileSync(${JSON.stringify(resourceFile)},JSON.stringify(resources));writeFileSync(${JSON.stringify(failed)},'failed');assert.fail('controlled-compose-first-failure');});
test('later-case',()=>writeFileSync(${JSON.stringify(later)},'started'));
`);
  const server = createServer((_request, response) => { response.writeHead(200); response.end('Owned readiness fixture'); });
  let primary;
  let started = false;
  try {
    await new Promise((done, reject) => { server.once('error', reject); server.listen(8090, '127.0.0.1', () => { started = true; done(); }); });
    const runEnv = { ...env, PATH: `${prefix}/node_modules/.bin:${env.PATH}` };
    result = await invoke([join(root, 'tests/scripts/run-integration.sh')], root, 'bash', runEnv, true);
    process.stdout.write(result.output);
    assert.equal(result.code, 1, 'The original test exit must survive cleanup failure: ' + result.output);
    assert.match(result.output, /controlled-compose-first-failure/);
    await absent(later);
    assert.equal(await readFile(injected, 'utf8'), '29');
    const owned = JSON.parse(await readFile(resourceFile, 'utf8'));
    for (const [kind, values] of Object.entries(owned)) for (const identity of values) {
      const args = kind === 'containers' ? ['container', 'inspect'] : kind === 'networks' ? ['network', 'inspect'] : ['volume', 'inspect'];
      const inspection = await invoke([...args, '--format', '{{.Name}}', identity], root, docker, env);
      assert.notEqual(inspection.code, 0, `Owned ${kind} remains: ${identity}`);
      assert.match(inspection.output, /no such|not found/i);
    }
    assert.deepEqual(await resources(), { containers: [], networks: [], volumes: [] });
    await absent(join(root, '.cache/loopaware-integration.lock'));
    process.stdout.write('LOOPAWARE_COMPOSE_RESOURCES_ABSENT ' + JSON.stringify({ project, ...owned }) + '\n');
    assert.match(result.output, /Integration stack cleanup failed \(status 29\)/, 'Real runner must report cleanup failure without replacing first test failure');
    assert.ok(result.output.indexOf('controlled-compose-first-failure') < result.output.indexOf('Integration stack cleanup failed (status 29)'));
    if (cleanupFailure === 'guardian') assert.match(result.output, /Integration cleanup guardian failed \(status 29\)/);
    else assert.doesNotMatch(result.output, /Integration cleanup guardian failed/);
    results.push({ scenario: 'compose-first-failure', cleanupFailure, project, owned, originalExit: result.code, originalFirst: true, cleanupDiagnostic: true, exactResourcesAbsent: true });
  } catch (error) { primary = error; throw error; }
  finally {
    const failures = [];
    if (started) { try { server.closeAllConnections(); await new Promise((done, reject) => server.close(error => error ? reject(error) : done())); } catch (error) { failures.push(error); } }
    try { const down = await invoke(['compose', '-f', composeFile, '-p', project, 'down', '-v', '--remove-orphans'], root, docker, env); assert.equal(down.code, 0, down.output); } catch (error) { failures.push(error); }
    if (failures.length) throw new AggregateError(primary ? [primary, ...failures] : failures, 'Compose qualification cleanup failed');
  }
}
