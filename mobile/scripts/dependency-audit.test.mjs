// @ts-check
import assert from 'node:assert/strict';
import { mkdtemp, mkdir, readFile, writeFile, cp, symlink, readdir, rm } from 'node:fs/promises';
import { spawn } from 'node:child_process';
import { createServer } from 'node:http';
import path from 'node:path';

const source = path.resolve(import.meta.dirname, '..');
await mkdir(path.resolve(import.meta.dirname, '../../_scratch'), {recursive:true});
const fixture = await mkdtemp(path.join(path.resolve(import.meta.dirname, '../../_scratch'), 'B152-dependency-audit-'));
await mkdir(path.join(fixture, 'home'));
let primaryError;
const active = new Set(/** @type {import('node:child_process').ChildProcess[]} */ ([]));
const policyText = await readFile(path.join(source, 'config/dependency-audit.json'), 'utf8');
const policy = JSON.parse(policyText);
let boundary = 'known';
let requests = 0;
const server = createServer(async (request, response) => {
  requests++;
  for await (const chunk of request) void chunk;
  response.setHeader('content-type', 'application/json');
  if (boundary === 'malformed') return response.end('{');
  if (boundary === 'unavailable') { response.statusCode = 503; return response.end('{"error":"unavailable"}'); }
  if (request.url?.includes('/security/advisories/bulk')) {
    if (boundary === 'empty') return response.end('{}');
    const name = ['unknown', 'critical'].includes(boundary) ? 'expo' : 'deepmerge';
    return response.end(JSON.stringify({ [name]: [{ id: 1243505,
      url: `https://github.com/advisories/${boundary.startsWith('unknown') ? 'GHSA-xxxx-yyyy-zzzz' : policy.remediations[0].advisory}`,
      title: 'Controlled dependency advisory', severity: boundary === 'critical' ? 'critical' : 'low',
      vulnerable_versions: '*', cwe: [], cvss: { score: 5.3, vectorString: null },
    }] }));
  }
  const name = decodeURIComponent(request.url?.slice(1) ?? '');
  const version = name === 'expo' ? '57.0.31' : '4.3.1';
  response.end(JSON.stringify({ name, 'dist-tags': { latest: version }, versions: { [version]: { name, version } } }));
});
await new Promise(done => server.listen(0, '127.0.0.1', () => done(undefined)));
const address = server.address();
assert.ok(address && typeof address !== 'string');
const port = address.port;
let sequence = 0;
/** @param {string} label @param {boolean} accepted @param {RegExp} [diagnostic] @param {string} [transform] */
async function run(label, accepted, diagnostic, transform) {
  const before = requests;
  const args = ['scripts/audit-dependencies.mjs'];
  if (transform) {
    const hook=path.join(fixture,'report-boundary.mjs');
    await writeFile(hook, `import cp from 'node:child_process';import{syncBuiltinESMExports}from'node:module';const actual=cp.spawnSync;cp.spawnSync=(command,args,options)=>{const result=actual(command,args,options);if(command==='npm'&&args[0]==='audit'){const report=JSON.parse(result.stdout);${transform};return {...result,stdout:JSON.stringify(report)};}return result;};syncBuiltinESMExports();`);
    args.unshift('--import',hook);
  }
  const child = spawn(process.execPath, args, { cwd: fixture,
    env: { PATH: process.env.PATH, HOME: path.join(fixture,'home'), TMPDIR:fixture, CI:'1', NODE_ENV:'test', npm_config_userconfig:'/dev/null', npm_config_registry: `http://127.0.0.1:${port}`,
      npm_config_cache: path.join(fixture, `cache-${sequence++}`), npm_config_fetch_retries: '0' },
    stdio: ['ignore', 'pipe', 'pipe'] });
  active.add(child);
  let output = '';
  child.stdout.on('data', chunk => { output += chunk; }); child.stderr.on('data', chunk => { output += chunk; });
  const status = await new Promise((done, reject) => { child.once('error', reject); child.once('close', done); });
  active.delete(child);
  assert.throws(()=>process.kill(/** @type {number} */ (child.pid),0),{code:'ESRCH'});
  assert.equal(status === 0, accepted, `${label}: ${output}`);
  if (diagnostic) assert.match(output, diagnostic, label);
  if (accepted) assert.ok(requests > before, 'Real npm must contact the controlled registry');
  console.log(`dependency_audit.ok: ${label}`);
}
try {
  for (const name of ['scripts', 'config', 'vendor', 'package.json', 'package-lock.json'])
    await cp(path.join(source, name), path.join(fixture, name), { recursive: true });
  await mkdir(path.join(fixture, 'node_modules'));
  for (const name of await readdir(path.join(source, 'node_modules')))
    if (name !== 'deepmerge') await symlink(path.join(source, 'node_modules', name), path.join(fixture, 'node_modules', name));
  const merged = path.join(fixture, 'node_modules/deepmerge');
  await cp(path.join(fixture, 'vendor/deepmerge'), merged, { recursive: true });
  await run('known advisory with real local correction', true, /behavior-qualified/);
  boundary = 'empty';
  await run('empty feed still qualifies the local correction', true, /1 behavior-qualified local corrections with absent advisories/);
  for (const mode of ['unknown', 'unknown-deepmerge', 'critical', 'malformed', 'unavailable']) { boundary = mode; await run(`registry ${mode}`, false); }
  boundary='known';
  await run('invalid native report counts',false,/invalid vulnerability counts/,'report.metadata.vulnerabilities.total+=1');
  await run('unrooted transitive cycle',false,/unrooted or cyclic-only/,'for(const [name,other]of[["cycle-a","cycle-b"],["cycle-b","cycle-a"]])report.vulnerabilities[name]={via:[other],nodes:["node_modules/"+name],severity:"low"};report.metadata.vulnerabilities.low+=2;report.metadata.vulnerabilities.total+=2');
  await run('unqualified advisory node',false,/unremediated advisory/,'report.vulnerabilities.deepmerge.nodes=["node_modules/unqualified/deepmerge"]');
  const config = path.join(fixture, 'config/dependency-audit.json');
  boundary = 'empty';
  /** @type {[string,(value:any)=>void][]} */
  const mutations = [
    ['missing correction', value => { value.remediations = []; }],
    ['wrong package identity', value => { value.remediations[0].package = 'expo'; }],
    ['expired review', value => { value.review_after = '2000-01-01'; }],
    ['unknown policy shape', value => { value.extra = true; }],
    ['missing installation', value => { value.remediations[0].installations = []; }],
    ['unselected path', value => { value.remediations[0].installations = ['../deepmerge']; }],
    ['invalid provenance', value => { value.remediations[0].upstream.pull_request = 'invalid'; }],
  ];
  for (const [label, mutate] of mutations) { const value = JSON.parse(policyText); mutate(value); await writeFile(config, JSON.stringify(value)); await run(label, false); }
  await writeFile(config, policyText);
  const lockFile=path.join(fixture,'package-lock.json');
  const lockText=await readFile(lockFile,'utf8');
  for(const [label,selection]of /** @type {[string,{resolved:string,link?:boolean}][]} */ ([
    ['registry route is not selected local source',{resolved:'https://registry.npmjs.org/deepmerge/-/deepmerge-4.3.1.tgz'}],
    ['alternate link shape is not current manager contract',{resolved:'vendor/deepmerge',link:true}],
  ])){
    const value=JSON.parse(lockText);Object.assign(value.packages['node_modules/deepmerge'],selection);
    await writeFile(lockFile,JSON.stringify(value));await run(label,false,/selected local package source differs/);
  }
  await writeFile(lockFile,lockText);

  for (const entry of ['dist/cjs.js', 'dist/umd.js']) {
    const runtime = path.join(merged, entry);
    const corrected = await readFile(runtime, 'utf8');
    const unsafe = corrected.replace('if (propertyIsForbidden(key)) { return }', '')
      .replace('propertyIsForbidden(key) || propertyIsUnsafe(target, key)', 'propertyIsUnsafe(target, key)');
    assert.notEqual(unsafe, corrected);
    await writeFile(runtime, unsafe);
    for (const mode of ['known', 'empty']) { boundary = mode; await run(`${mode} rejects unsafe ${entry}`, false, /behavior failed/); }
    await writeFile(runtime, corrected);
  }
  boundary = 'empty';
  const provenance = path.join(fixture, 'vendor/deepmerge/PROVENANCE.json');
  const original = await readFile(provenance, 'utf8');
  await writeFile(provenance, JSON.stringify({ ...JSON.parse(original), upstream_fix_reference: 'https://github.com/TehShrike/deepmerge/pull/999' }));
  await run('provenance differs', false, /provenance differs/); await writeFile(provenance, original);
  const concealed = path.join(fixture, 'node_modules/extra/node_modules/deepmerge');
  await mkdir(path.dirname(concealed), { recursive: true }); await cp(merged, concealed, { recursive: true, dereference: true });
  await run('unselected installed copy', false, /installed copies differ/);
  await rm(path.join(fixture, 'node_modules/extra'), { recursive: true });
  const metadataPath = path.join(merged, 'package.json'); const metadata = await readFile(metadataPath, 'utf8');
  await writeFile(metadataPath, JSON.stringify({ ...JSON.parse(metadata), version: '0.0.0' }));
  await run('installed version differs', false, /installed version differs/); await writeFile(metadataPath, metadata);
  const patchPath = path.join(fixture, policy.remediations[0].patch); const patch = await readFile(patchPath, 'utf8');
  await writeFile(patchPath, patch + '\n'); await run('formatting is not readiness authority', true);
  await rm(patchPath); await run('required correction patch missing', false, /ENOENT/); await writeFile(patchPath, patch);
} catch(error) {primaryError=error;throw error;} finally {
  const cleanupErrors=[];
  for(const child of active){try{child.kill('SIGTERM');await new Promise(done=>child.once('close',done));}catch(error){cleanupErrors.push(error);}}
  try{server.closeAllConnections();await new Promise((done,reject)=>server.close(error=>error?reject(error):done(undefined)));}catch(error){cleanupErrors.push(error);}
  try{await rm(fixture,{recursive:true,force:true});await assert.rejects(readFile(fixture),{code:'ENOENT'});}catch(error){cleanupErrors.push(error);}
  if(cleanupErrors.length)throw new AggregateError(primaryError?[primaryError,...cleanupErrors]:cleanupErrors,'Dependency audit fixture cleanup failed');
  console.log('dependency_audit.cleanup.ok: exact children reaped, local server closed, owned scratch removed');
}
