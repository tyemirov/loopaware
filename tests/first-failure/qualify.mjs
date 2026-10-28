// @ts-check
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { join, resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import { playwrightFirstFailure, matrixFirstFailure, makeFirstFailure, goFirstFailure, goResistant, goControls, goCleanupError, firstGoFailure } from './frameworks.mjs';

import { composeFirstFailure } from './compose.mjs';

const repository = resolve(import.meta.dirname, '../..');
const scenario = process.argv[2] || 'node-same-file';
const referenceNative = process.argv.includes('--reference-native');
const scratch = join(repository, '_scratch');
await mkdir(scratch, { recursive: true });
const owner = await mkdtemp(join(scratch, 'B148-first-failure-'));
const adapter = pathToFileURL(join(repository, 'scripts/native-test.mjs')).href;
const adapterURL = adapter;
const results = [];
const active = new Set();

/** @param {string} file */
async function absent(file) {
  let value;
  try { value = await readFile(file, 'utf8'); }
  catch (error) { if (error.code !== 'ENOENT') throw error; }
  assert.equal(value, undefined, `Later work executed: ${file}`);
}

/** @param {string[]} args @param {string} [cwd] @param {string} [executable] */
async function invoke(args, cwd = repository, executable = process.execPath, env = process.env, live = false, launch = spawn) {
  const child = launch(executable, args, { cwd, env, stdio: ['ignore', 'pipe', 'pipe'] });
  active.add(child);
  let output = '';
  let stdout = '';
  let stderr = '';
  process.stdout.write('LOOPAWARE_QUALIFIER_CHILD_STARTED ' + JSON.stringify({ pid: child.pid }) + '\n');
  child.stdout.on('data', chunk => { stdout += chunk; output += chunk; if (live) process.stdout.write(chunk); });
  child.stderr.on('data', chunk => { stderr += chunk; output += chunk; if (live) process.stdout.write(chunk); });
  const result = await new Promise((done, reject) => {
    child.once('error', reject);
    child.once('close', (code, signal) => done({ code, signal, pid: child.pid, output, stdout, stderr }));
  });
  active.delete(child);
  assert.throws(() => process.kill(result.pid, 0), { code: 'ESRCH' });
  for (const match of result.output.matchAll(/^LOOPAWARE_NODE_WORKER_CLOSED (.+)$/gm)) {
    const worker = JSON.parse(match[1]);
    assert.throws(() => process.kill(worker.pid, 0), { code: 'ESRCH' });
  }
  process.stdout.write(`LOOPAWARE_QUALIFIER_CHILD_CLOSED ${JSON.stringify({ pid: result.pid, code: result.code, signal: result.signal })}\n`);
  return result;
}

async function goReceiptChannels() {
  const file = join(owner, 'receipt-channels.mjs');
  const receipt = { action: 'output', package: 'loopaware.local/channelprobe', test: 'TestFailure', output_type: 'frame' };
  const diagnostic = `LOOPAWARE_GO_FIRST_FAILURE ${JSON.stringify(receipt)}\n`;
  await writeFile(file, `process.stdin.once('data',()=>{process.stderr.write(${JSON.stringify(diagnostic)});process.exitCode=1;});process.stdout.write('# go: ');\n`);
  const result = await invoke([file], repository, process.execPath, process.env, false, (command, args, options) => {
    const child = spawn(command, args, { ...options, stdio: ['pipe', 'pipe', 'pipe'] });
    child.stdout.once('data', () => child.stdin.end('continue\n'));
    return child;
  });
  assert.equal(result.code, 1, result.output);
  assert.equal(result.output, `# go: ${diagnostic}`);
  assert.equal(result.stdout, '# go: ');
  assert.equal(result.stderr, diagnostic);
  assert.deepEqual(firstGoFailure(result.stderr), receipt);
  results.push({ scenario: 'go-receipt-channels', mixedPresentationReproduced: true, receiptPreserved: true });
}

async function sameFile() {
  const later = join(owner, 'later-body');
  const cleanup = join(owner, 'entered-cleanup');
  const rootCleanup = join(owner, 'root-cleanup');
  const file = join(owner, 'same-file.test.mjs');
  const registration = referenceNative ? 'node:test' : adapter;
  await writeFile(file, `import test,{after} from ${JSON.stringify(registration)};
import assert from 'node:assert/strict';import{writeFileSync}from'node:fs';
after(()=>writeFileSync(${JSON.stringify(rootCleanup)},'cleanup'));
test('first-controlled-failure',t=>{t.after(()=>writeFileSync(${JSON.stringify(cleanup)},'cleanup'));assert.fail('controlled-first-failure');});
test('later-body',()=>writeFileSync(${JSON.stringify(later)},'started'));
`);
  const args = referenceNative ? ['--test', file] : [join(repository, 'scripts/run-node-tests.mjs'), file];
  const result = await invoke(args);
  assert.notEqual(result.code, 0, result.output);
  assert.match(result.output, /controlled-first-failure/);
  if (!referenceNative) {
    assert.match(result.output, /LOOPAWARE_NODE_FIRST_FAILURE/);
    assert.match(result.output, /not ok 1 - first-controlled-failure/);
    assert.match(result.output, /LOOPAWARE_NODE_WORKER_CLOSED/);
  }
  await absent(later);
  assert.equal(await readFile(cleanup, 'utf8'), 'cleanup');
  assert.equal(await readFile(rootCleanup, 'utf8'), 'cleanup');
  process.stdout.write(result.output);
  process.stdout.write(JSON.stringify({ scenario: 'node-same-file', originalFailure: 'controlled-first-failure', laterBodyAbsent: true, testCleanup: true, rootCleanup: true }) + '\n');
}

/** @param {string} name @param {string} body @param {string[]} forbidden @param {string[]} cleanup @param {boolean} fails */
async function contract(name, body, forbidden = [], cleanup = [], fails = true) {
  const file = join(owner, `${name}.test.mjs`);
  await writeFile(file, `import test,{describe,it,before,after,todo,only,expectFailure,skip} from ${JSON.stringify(adapterURL)};
import assert from 'node:assert/strict';
import {writeFileSync} from 'node:fs';
const enter=name=>process.stdout.write('LOOPAWARE_FIXTURE_ADMISSION '+JSON.stringify({name,at_ns:process.hrtime.bigint().toString()})+'\\n');
${body}\n`);
  const result = await invoke([join(repository, 'scripts/run-node-tests.mjs'), file]);
  assert.equal(result.code === 0, !fails, result.output);
  if (fails) assert.match(result.output, /LOOPAWARE_NODE_FIRST_FAILURE/, result.output);
  else assert.doesNotMatch(result.output, /LOOPAWARE_NODE_FIRST_FAILURE/, result.output);
  for (const path of forbidden) await absent(path);
  for (const path of cleanup) assert.equal(await readFile(path, 'utf8'), 'cleanup', `${name}: cleanup ${path}`);
  const admissions = [...result.output.matchAll(/LOOPAWARE_FIXTURE_ADMISSION (.+)$/gm)].map(match => JSON.parse(match[1]));
  const observation = result.output.match(/^LOOPAWARE_NODE_FIRST_FAILURE (.+)$/m);
  if (observation) {
    const observed = JSON.parse(observation[1]);
    for (const admission of admissions) assert.ok(BigInt(admission.at_ns) <= BigInt(observed.observed_at_ns), `${name}: body admitted after failure observation`);
  }
  process.stdout.write(result.output);
  results.push({ scenario: name, result: 'passed', pid: result.pid, admissions });
}

async function nodeWorkerError() {
  const cleanupFiles = [join(owner, 'first-worker-cleanup'), join(owner, 'second-worker-cleanup')];
  const files = [];
  for (let index = 0; index < 2; index++) {
    const file = join(owner, `worker-error-${index}.test.mjs`);
    files.push(file);
    await writeFile(file, `import test from ${JSON.stringify(adapter)};import{writeFileSync}from'node:fs';
 test('entered-worker-${index}',async t=>{try{await new Promise((done,reject)=>{t.signal.addEventListener('abort',()=>reject(t.signal.reason),{once:true});process.send({type:'fixture-ready'});});}finally{writeFileSync(${JSON.stringify(cleanupFiles[index])},'cleanup');}});`);
  }
  const later = join(owner, 'worker-error-queued');
  const queued = join(owner, 'queued-error.test.mjs');
  await writeFile(queued, `import{writeFileSync}from'node:fs';writeFileSync(${JSON.stringify(later)},'started');`);
  const entry = join(owner, 'worker-error-entry.mjs');
  await writeFile(entry, `import{fork}from'node:child_process';import{coordinate}from ${JSON.stringify(pathToFileURL(join(repository, 'scripts/run-node-tests.mjs')).href)};
const children=[];let ready=0;await coordinate(${JSON.stringify(['--test-concurrency=2', ...files, queued])},(...args)=>{const child=fork(...args);children.push(child);child.on('message',message=>{if(message?.type==='fixture-ready'&&++ready===2)child.emit('error',new Error('controlled-worker-boundary-error'));});return child;});
if(children.length!==2)throw new Error('Queued worker started');
for(const child of children){if(child.exitCode===null&&child.signalCode===null)throw new Error('Worker returned before actual close');}
`);
  const result = await invoke([entry]);
  process.stdout.write(result.output);
  assert.notEqual(result.code, 0, result.output);
  const first = JSON.parse(/^LOOPAWARE_NODE_FIRST_FAILURE (.+)$/m.exec(result.output)[1]);
  assert.equal(first.message, 'controlled-worker-boundary-error');
  assert.equal([...result.output.matchAll(/^LOOPAWARE_NODE_WORKER_CLOSED /gm)].length, 2);
  for (const file of cleanupFiles) assert.equal(await readFile(file, 'utf8'), 'cleanup');
  await absent(later);
  results.push({scenario:'node-worker-error',firstCause:first.message,closedWorkers:2,enteredCleanup:2,queuedAbsent:true});
}

async function nodeContracts() {
  for (const variant of ['todo', 'only', 'expectFailure']) {
    const later = join(owner, `named-${variant}-after-failure`);
    await contract(`node-named-${variant}-cancellation`, `
${variant === 'only' ? 'only' : 'test'}('first-controlled-failure',()=>assert.fail('controlled-first-failure'));
${variant}('later-named-variant',()=>writeFileSync(${JSON.stringify(later)},'started'));`, [later]);
  }
  const suiteSetup = join(owner, 'later-suite-setup');
  const suiteBody = join(owner, 'later-suite-body');
  const rootCleanup = join(owner, 'suite-root-cleanup');
  await contract('node-later-suite', `
after(()=>writeFileSync(${JSON.stringify(rootCleanup)},'cleanup'));
test('first-controlled-failure',()=>{enter('first-controlled-failure');assert.fail('controlled-first-failure');});
describe('later-suite',()=>{before(()=>{enter('later-suite-before');writeFileSync(${JSON.stringify(suiteSetup)},'started');});it('later',()=>{enter('later-suite-body');writeFileSync(${JSON.stringify(suiteBody)},'started');});});`, [suiteSetup, suiteBody], [rootCleanup]);
  const continuation = join(owner, 'nested-parent-continuation');
  const secondChild = join(owner, 'nested-second-child');
  const parentCleanup = join(owner, 'nested-parent-cleanup');
  const childCleanup = join(owner, 'nested-child-cleanup');
  const nestedRootCleanup = join(owner, 'nested-root-cleanup');
  await contract('node-nested', `
after(()=>writeFileSync(${JSON.stringify(nestedRootCleanup)},'cleanup'));
test('parent',async t=>{
 enter('parent');
 t.after(()=>writeFileSync(${JSON.stringify(parentCleanup)},'cleanup'));
 await t.test('first-controlled-failure',child=>{enter('first-controlled-failure');child.after(()=>writeFileSync(${JSON.stringify(childCleanup)},'cleanup'));assert.fail('controlled-first-failure');});
 writeFileSync(${JSON.stringify(continuation)},'started');
 await t.test('later-child',()=>writeFileSync(${JSON.stringify(secondChild)},'started'));
});`, [continuation, secondChild], [parentCleanup, childCleanup, nestedRootCleanup]);
  const skippedBody = join(owner, 'skipped-body');
  await contract('node-negative-controls', `
test('caught-negative',()=>{enter('caught-negative');assert.throws(()=>{throw new Error('expected-negative');},/expected-negative/);});
test.skip('skipped',()=>writeFileSync(${JSON.stringify(skippedBody)},'started'));
test.todo('todo-failure',()=>assert.fail('expected-todo'));
skip('named-skipped',()=>writeFileSync(${JSON.stringify(skippedBody)},'started'));
todo('named-todo-failure',()=>assert.fail('expected-named-todo'));
expectFailure('named-expected-failure',()=>assert.fail('expected-named-failure'));
test('successful-control',()=>assert.ok(true));`, [skippedBody], [], false);
  const callerBody = join(owner, 'caller-aborted-body');
  const callerCleanup = join(owner, 'caller-root-cleanup');
  await contract('node-caller-signal', `
after(()=>writeFileSync(${JSON.stringify(callerCleanup)},'cleanup'));
test('caller-aborted',{signal:AbortSignal.abort(new Error('caller-requested-abort'))},()=>writeFileSync(${JSON.stringify(callerBody)},'started'));`, [callerBody], [callerCleanup]);
  const runningCleanup = join(owner, 'running-sibling-finally');
  const runningParentCleanup = join(owner, 'running-parent-cleanup');
  const runningContinuation = join(owner, 'running-parent-continuation');
  await contract('node-running-finally', `
const caller=new AbortController();
test('parallel-parent',{concurrency:2},async t=>{
 enter('parallel-parent');
 t.after(()=>writeFileSync(${JSON.stringify(runningParentCleanup)},'cleanup'));
 let ready;const started=new Promise(resolve=>{ready=resolve;});
 const running=t.test('running-sibling',{signal:caller.signal},async ctx=>{
  enter('running-sibling');
  try{await new Promise((resolve,reject)=>{ctx.signal.addEventListener('abort',()=>reject(ctx.signal.reason),{once:true});ready();});}
  finally{assert.equal(caller.signal.aborted,false);writeFileSync(${JSON.stringify(runningCleanup)},'cleanup');}
 });
 const failing=t.test('first-controlled-failure',async()=>{await started;enter('first-controlled-failure');assert.fail('controlled-first-failure');});
 await Promise.all([running,failing]);writeFileSync(${JSON.stringify(runningContinuation)},'started');
});`, [runningContinuation], [runningCleanup, runningParentCleanup]);
}

async function selectors() {
  const file = join(owner, 'selection.test.mjs');
  const selected = join(owner, 'selected');
  const forbidden = join(owner, 'unselected');
  await writeFile(file, `import test from ${JSON.stringify(adapter)};import{writeFileSync}from'node:fs';import assert from'node:assert/strict';
test('unselected failure',()=>{writeFileSync(${JSON.stringify(forbidden)},'started');throw new Error('not-selected');});
test('selected explicit options',{concurrency:1},()=>{assert.ok(process.execArgv.includes('--conditions=loopaware-first-failure'));writeFileSync(${JSON.stringify(selected)},'selected');});
test('selected callback',(context,done)=>done());
`);
  const result = await invoke(['--conditions=loopaware-first-failure', join(repository, 'scripts/run-node-tests.mjs'), '--test-concurrency=1', '--test-name-pattern=^selected', file]);
  assert.equal(result.code, 0, result.output);
  assert.match(result.output, /# tests 2/);
  assert.match(result.output, /# pass 2/);
  assert.doesNotMatch(result.output, /LOOPAWARE_NODE_FIRST_FAILURE/);
  await absent(forbidden);
  assert.equal(await readFile(selected, 'utf8'), 'selected');
  process.stdout.write(result.output);
  results.push({ scenario: 'node-selectors', cardinality: 2, status: 'passed' });
}

async function browserCleanup() {
  const serverFile = join(owner, 'http-server.mjs');
  const slowFile = join(owner, 'browser-owner.test.mjs');
  const fastFile = join(owner, 'controlled-failure.test.mjs');
  const nextFile = join(owner, 'queued-file.test.mjs');
  const slowReady = join(owner, 'browser-ready.json');
  const fastReady = join(owner, 'failure-ready');
  const cleanup = join(owner, 'browser-cleanup');
  const fastCleanup = join(owner, 'failure-root-cleanup');
  const nextFileSentinel = join(owner, 'queued-file-body');
  const nextMakeSentinel = join(owner, 'later-make-target');
  await writeFile(serverFile, `import {createServer} from 'node:http';
const server=createServer((request,response)=>{response.writeHead(200,{'content-type':'text/html'});response.end('<title>LoopAware owned fixture</title>');});
server.listen(0,'127.0.0.1',()=>process.send({port:server.address().port,pid:process.pid}));
process.once('SIGTERM',()=>server.close(()=>{process.disconnect();}));\n`);
  await writeFile(slowFile, `import test from ${JSON.stringify(adapterURL)};
import assert from 'node:assert/strict';import {spawn} from 'node:child_process';import {createRequire} from 'node:module';import {writeFileSync} from 'node:fs';
const {chromium}=createRequire(${JSON.stringify(join(repository, 'tests/package.json'))})('@playwright/test');
test('owned-browser-and-server',async t=>{
 let browserServer;let browser;let server;let serverClosed;
 t.after(async()=>{
  const cleanupErrors=[];
  for(const close of [async()=>{if(browser)await browser.close();},async()=>{if(browserServer)await browserServer.close();},async()=>{if(server){server.kill('SIGTERM');await serverClosed;}}]){try{await close();}catch(error){cleanupErrors.push(error);}}
  writeFileSync(${JSON.stringify(cleanup)},'cleanup');
  if(cleanupErrors.length)throw new AggregateError(cleanupErrors,'Owned fixture cleanup failed');
 });
 server=spawn(process.execPath,[${JSON.stringify(serverFile)}],{stdio:['ignore','ignore','pipe','ipc']});
 serverClosed=new Promise((resolve,reject)=>{server.once('error',reject);server.once('close',(code,signal)=>code===0?resolve():reject(new Error('HTTP server exit '+code+'/'+signal)));});
 const endpoint=await new Promise((resolve,reject)=>{server.once('error',reject);server.once('message',resolve);});
 browserServer=await chromium.launchServer({headless:true,...(process.env.LOOPAWARE_PLAYWRIGHT_CHANNEL?{channel:process.env.LOOPAWARE_PLAYWRIGHT_CHANNEL}:{})});
 browser=await chromium.connect(browserServer.wsEndpoint());
 const page=await browser.newPage();await page.goto('http://127.0.0.1:'+endpoint.port);assert.equal(await page.title(),'LoopAware owned fixture');
 writeFileSync(${JSON.stringify(slowReady)},JSON.stringify({browserPid:browserServer.process().pid,serverPid:endpoint.pid,workerPid:process.pid,port:endpoint.port}));
 await new Promise((resolve,reject)=>{if(t.signal.aborted)reject(t.signal.reason);else t.signal.addEventListener('abort',()=>reject(t.signal.reason),{once:true});});
});\n`);
  await writeFile(fastFile, `import test,{after} from ${JSON.stringify(adapterURL)};
import assert from 'node:assert/strict';import {existsSync,writeFileSync} from 'node:fs';import {setTimeout} from 'node:timers/promises';
after(()=>writeFileSync(${JSON.stringify(fastCleanup)},'cleanup'));
test('first-controlled-failure',async t=>{writeFileSync(${JSON.stringify(fastReady)},'ready');while(!existsSync(${JSON.stringify(slowReady)}))await setTimeout(5,undefined,{signal:t.signal});assert.fail('controlled-first-failure');});\n`);
  await writeFile(nextFile, `import test from ${JSON.stringify(adapterURL)};import{writeFileSync}from'node:fs';test('queued-file-must-not-start',()=>writeFileSync(${JSON.stringify(nextFileSentinel)},'started'));\n`);
  const quote = value => `'${value.replaceAll("'", "'\\''")}'`;
  const makeFile = join(owner, 'Makefile');
  await writeFile(makeFile, `.NOTPARALLEL:\n.PHONY: all first next\nall: first next\nfirst:\n\t${quote(process.execPath)} ${quote(join(repository, 'scripts/run-node-tests.mjs'))} --test-concurrency=2 ${[slowFile, fastFile, nextFile].map(quote).join(' ')}\nnext:\n\t/usr/bin/touch ${quote(nextMakeSentinel)}\n`);
  const result = await invoke(['--no-print-directory', '-f', makeFile, 'all'], owner, '/usr/bin/make');
  assert.notEqual(result.code, 0, result.output);
  assert.match(result.output, /controlled-first-failure/, result.output);
  assert.equal(await readFile(fastReady, 'utf8'), 'ready');
  const resources = JSON.parse(await readFile(slowReady, 'utf8'));
  for (const pid of [resources.browserPid, resources.serverPid, resources.workerPid]) assert.throws(() => process.kill(pid, 0), { code: 'ESRCH' });
  assert.equal(await readFile(cleanup, 'utf8'), 'cleanup');
  assert.equal(await readFile(fastCleanup, 'utf8'), 'cleanup');
  await absent(nextFileSentinel);
  await absent(nextMakeSentinel);
  process.stdout.write(`LOOPAWARE_QUALIFICATION_OWNED_RESOURCES_ABSENT ${JSON.stringify(resources)}\n`);
  process.stdout.write(result.output);
  results.push({ scenario: 'node-browser-server-cleanup', result: 'passed', resources });
}

async function pytestFirstFailure() {
  const root = join(owner, 'pytest');
  await mkdir(root);
  const later = join(root, 'later-case');
  const cleanup = join(root, 'fixture-cleanup');
  await writeFile(join(root, 'Makefile'), await readFile(join(repository, 'tests/deployment/Makefile'), 'utf8'));
  await writeFile(join(root, 'test_npm_visibility.py'), `from pathlib import Path
import pytest
@pytest.fixture
def owned():
    yield
    Path(${JSON.stringify(cleanup)}).write_text("cleanup")
def test_first_controlled_failure(owned):
    assert False, "controlled-pytest-first-failure"
def test_later_must_not_run():
    Path(${JSON.stringify(later)}).write_text("started")
`);
  const environment = {
    PATH: process.env.PATH, HOME: process.env.HOME, CI: '1',
    TMPDIR: owner, UV_OFFLINE: 'true', UV_NO_CONFIG: 'true', PYTEST_DISABLE_PLUGIN_AUTOLOAD: '1',
    MPRLAB_GATEWAY_EXECUTABLE: '',
  };
  const result = await invoke(['--no-print-directory', '-f', join(root, 'Makefile'), 'test'], root, 'make', environment);
  process.stdout.write(result.output);
  assert.notEqual(result.code, 0, result.output);
  assert.match(result.output, /controlled-pytest-first-failure/, result.output);
  assert.equal(await readFile(cleanup, 'utf8'), 'cleanup');
  await absent(later);
  results.push({ scenario: 'pytest-first-failure', laterCaseAbsent: true, fixtureCleanup: true });
}

let primaryError;
try {
  if (scenario === 'node-same-file') await sameFile();
  else if (scenario === 'node-worker-error') await nodeWorkerError();
  else if (scenario === 'node-contracts') { await nodeContracts(); assert.equal(results.length, 8); await selectors(); }
  else if (scenario === 'node-browser-cleanup') await browserCleanup();
  else if (scenario === 'compose-first-failure') await composeFirstFailure({ repository, owner, invoke, absent, results });
  else if (scenario === 'compose-main-cleanup-error') await composeFirstFailure({ repository, owner, invoke, absent, results, cleanupFailure: 'main' });
  else if (scenario === 'pytest-first-failure') await pytestFirstFailure();
  else if (scenario === 'playwright-first-failure') await playwrightFirstFailure({ repository, owner, invoke, absent, results });
  else if (scenario === 'matrix-first-failure') await matrixFirstFailure({ repository, owner, invoke, results });
  else if (scenario === 'make-first-failure') await makeFirstFailure({ repository, owner, invoke, absent, results });
  else if (scenario === 'go-receipt-channels') await goReceiptChannels();
  else if (scenario === 'go-first-failure') await goFirstFailure({ repository, owner, invoke, absent, results, referenceNative });
  else if (scenario === 'go-resistant') await goResistant({ repository, owner, invoke, absent, results });
  else if (scenario === 'go-cleanup-error') await goCleanupError({ repository, owner, invoke, results });
  else if (scenario === 'go-controls') { await goControls({ repository, owner, invoke, absent, results }); assert.equal(results.length, 3); }
  else assert.fail(`Unknown first-failure scenario: ${scenario}`);
  process.stdout.write(JSON.stringify({ status: 'passed', scenario, results }) + '\n');
} catch (error) {
  primaryError = error;
  throw error;
} finally {
  const cleanupErrors = [];
  for (const child of active) {
    try { child.kill('SIGTERM'); await new Promise(done => child.once('close', done)); }
    catch (error) { cleanupErrors.push(error); }
  }
  try { await rm(owner, { recursive: true, force: true }); await absent(owner); }
  catch (error) { cleanupErrors.push(error); }
  if (cleanupErrors.length) throw new AggregateError(primaryError ? [primaryError, ...cleanupErrors] : cleanupErrors, 'First-failure qualifier cleanup failed');
}
