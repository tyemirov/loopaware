// @ts-check
import assert from 'node:assert/strict';
import { spawnSync, spawn } from 'node:child_process';
import { existsSync, mkdirSync, readFileSync, writeFileSync, rmSync, symlinkSync } from 'node:fs';
import { join, resolve, dirname } from 'node:path';
import test from '../../scripts/native-test.mjs';
import { publicFixture } from './preparation.mjs';
import { createRequire } from 'node:module';

/** @param {(context:ReturnType<typeof setup>)=>void|Promise<void>} check */
async function withFixture(check) {
  const context = setup();
  try { await check(context); } finally { context.fixture.cleanup(); }
}

function setup() {
  const fixture = publicFixture();
  mkdirSync(join(fixture.prepared, 'src'), { recursive: true });
  writeFileSync(join(fixture.mobile, 'src/deleted.mjs'), 'old product source');
  writeFileSync(join(fixture.prepared, 'src/deleted.mjs'), 'old product source');
  const recorded = fixture.record(); assert.equal(recorded.status, 0, recorded.stderr);
  writeFileSync(join(fixture.mobile, 'src/product.mjs'), 'current captured product');
  writeFileSync(join(fixture.mobile, 'src/added.mjs'), 'new captured product');
  rmSync(join(fixture.mobile, 'src/deleted.mjs'));
  writeFileSync(join(fixture.mobile, 'app.config.js'), readFileSync(join(fixture.mobile, 'app.config.js'), 'utf8') + '\nmodule.exports.expo.extra.publicSourceMarker = "captured-current";\n');
  const privateRoot = join(fixture.directory, 'operator working checkout');
  mkdirSync(join(privateRoot, 'mobile/src'), { recursive: true });
  writeFileSync(join(privateRoot, 'mobile/src/product.mjs'), 'wrong mutable primary source');
  const resultPath = join(fixture.directory, 'native-result.json');
  const readyPath = join(fixture.directory, 'native-ready.json');
  const boundary = `const fs=require('node:fs'),cp=require('node:child_process');
const request=JSON.parse(fs.readFileSync(0,'utf8'));
const source=request.source_root;
const p=require('node:path');
const metadata=JSON.parse(fs.readFileSync(p.join(source,'package.json'),'utf8'));
const snapshot=JSON.parse(fs.readFileSync(p.join(source,'app.config.snapshot.json'),'utf8'));
const result={request,product:fs.readFileSync(p.join(source,'src/product.mjs'),'utf8'),added:fs.existsSync(p.join(source,'src/added.mjs')),deleted:fs.existsSync(p.join(source,'src/deleted.mjs')),podfile:fs.existsSync(p.join(source,'ios/Podfile')),workspace:fs.existsSync(p.join(source,'ios/LoopAware.xcworkspace/contents.xcworkspacedata')),devClient:metadata.dependencies?.['expo-dev-client'],publicSourceMarker:snapshot.expo.extra?.publicSourceMarker,privatePaths:['.env.fixture','private.jks','node_modules','ios/.xcode.env.local','android/local.properties','.gradle/cache'].filter(name=>fs.existsSync(p.join(source,name)))};
fs.writeFileSync(process.env.PUBLIC_NATIVE_RESULT,JSON.stringify(result));
if(process.env.PUBLIC_NATIVE_MODE==='metro'){
 (async()=>{
  const {default:FileMap,DiskCacheManager}=require(process.env.PUBLIC_METRO_FILE_MAP);
  const fileMap=FileMap.create({rootDir:source,roots:[source],extensions:['ts','tsx','mjs','json'],
   retainAllFiles:true,computeSha1:false,maxWorkers:1,resetCache:true,watch:false,useWatchman:false,
   forceNodeFilesystemAPI:true,healthCheck:{enabled:false},
   cacheManagerFactory:parameters=>new DiskCacheManager(parameters,{cacheDirectory:process.env.PUBLIC_METRO_CACHE})});
  try{
   const {fileSystem}=await fileMap.build();
   const files=['index.ts','src/product.mjs','vendor/runtime.mjs'];
   for(const file of files)if(!fileSystem.exists(p.join(source,file)))throw new Error('Metro did not index public native source: '+file);
   result.indexedFiles=files;
   fs.writeFileSync(process.env.PUBLIC_NATIVE_RESULT,JSON.stringify(result));
   fs.writeFileSync(request.output,'public Metro file-map consumer output');
  }finally{await fileMap.end();}
 })().catch(error=>{process.stderr.write(error.message+'\\n');process.exitCode=31;});
}else if(process.env.PUBLIC_NATIVE_MODE==='wait'){
 const descendant=cp.spawn(process.execPath,['-e','setInterval(()=>{},1000)'],{stdio:'ignore'});
 fs.writeFileSync(process.env.PUBLIC_NATIVE_READY,JSON.stringify({pid:process.pid,descendant:descendant.pid,source}));
 setInterval(()=>{},1000);
}else{process.stderr.write('public original native failure\\n');process.exit(29);}
`;
  for (const directory of [fixture.mobile, fixture.prepared]) writeFileSync(join(directory, 'mobile-build-operation'), boundary);
  const args = [join(fixture.mobile, 'scripts/build-store-artifact.mjs'), '--mobile-dir', fixture.prepared,
    '--output', join(fixture.directory, 'ios.ipa'), '--manifest', join(fixture.directory, 'ios.json'), '--release-timestamp', '2026-09-10T09:12:53Z'];
  const environment = { ...fixture.environment, MPRLAB_APP_ROOT: privateRoot, MPRLAB_GATEWAY_EXECUTABLE: process.execPath,
    MPRLAB_ARTIFACT_VERSION: '1.2.0', PUBLIC_NATIVE_RESULT: resultPath, PUBLIC_NATIVE_READY: readyPath };
  const invoke = () => spawnSync(process.execPath, args, { cwd: fixture.repository, env: environment, encoding: 'utf8', stdio: ['ignore','pipe','pipe'] });
  return { fixture, args, environment, invoke, resultPath, readyPath };
}

test('the actual adapter stages added/edited/deleted captured product source and retains native inputs', () => withFixture(context => {
  const { fixture } = context;
  const result = context.invoke(); assert.equal(result.status, 29, result.stderr); assert.match(result.stderr, /public original native failure/);
  const observed = JSON.parse(readFileSync(context.resultPath, 'utf8'));
  assert.equal(observed.product, 'current captured product'); assert.equal(observed.added, true); assert.equal(observed.deleted, false);
  assert.equal(observed.podfile, true); assert.equal(observed.workspace, true); assert.equal(observed.devClient, undefined);
  assert.equal(observed.publicSourceMarker, 'captured-current');
  assert.notEqual(observed.request.source_root, fixture.prepared);
  assert.equal(dirname(observed.request.source_root), fixture.directory);
  assert.equal(existsSync(observed.request.source_root), false, 'native failure must remove only the owned build stage');
  assert.equal(existsSync(fixture.prepared), true);
}));

test('Metro indexes the staged public source when the sealed output is inside Git metadata', () => withFixture(context => {
  const outputDirectory = join(context.fixture.repository, '.git/mprlab-lifecycle/releases/v1.2.0/payloads/mobile');
  mkdirSync(outputDirectory, { recursive: true });
  const output = join(outputDirectory, 'ios.ipa');
  context.args[context.args.indexOf('--output') + 1] = output;
  context.args[context.args.indexOf('--manifest') + 1] = join(outputDirectory, 'ios.json');
  mkdirSync(join(context.fixture.mobile, 'vendor'));
  writeFileSync(join(context.fixture.mobile, 'vendor/runtime.mjs'), 'export const runtime = "current public vendor";\n');
  const require = createRequire(resolve(import.meta.dirname, '../../mobile/package.json'));
  context.environment.PUBLIC_NATIVE_MODE = 'metro';
  context.environment.PUBLIC_METRO_FILE_MAP = require.resolve('metro-file-map');
  context.environment.PUBLIC_METRO_CACHE = join(context.fixture.directory, 'metro-cache');
  mkdirSync(context.environment.PUBLIC_METRO_CACHE);
  const result = context.invoke();
  assert.equal(result.status, 0, result.stderr);
  const observed = JSON.parse(readFileSync(context.resultPath, 'utf8'));
  assert.deepEqual(observed.indexedFiles, ['index.ts', 'src/product.mjs', 'vendor/runtime.mjs']);
  assert.equal(observed.product, 'current captured product');
  assert.equal(observed.request.output, output);
  assert.equal(readFileSync(output, 'utf8'), 'public Metro file-map consumer output');
  assert.equal(existsSync(observed.request.source_root), false, 'successful consumer completion must remove its owned source stage');
  assert.equal(existsSync(context.fixture.prepared), true);
}));

test('public staging excludes private, signing, dependency cache, and local native inputs', () => withFixture(context => {
  const { fixture } = context;
  for (const [base, name] of [[fixture.mobile, '.env.fixture'], [fixture.mobile, 'private.jks'], [fixture.prepared, 'ios/.xcode.env.local'], [fixture.prepared, 'android/local.properties'], [fixture.mobile, '.gradle/cache']]) {
    mkdirSync(dirname(join(base, name)), { recursive: true }); writeFileSync(join(base, name), 'synthetic private fixture');
  }
  mkdirSync(join(fixture.mobile, 'node_modules'), { recursive: true });
  writeFileSync(join(fixture.mobile, 'node_modules/cache'), 'synthetic cache');
  const result = context.invoke(); assert.equal(result.status, 29, result.stderr);
  assert.deepEqual(JSON.parse(readFileSync(context.resultPath, 'utf8')).privatePaths, []);
}));

for (const condition of ['source identity', 'source dependency', 'native configuration', 'missing encryption declaration', 'non-exempt encryption declaration', 'string encryption declaration', 'duplicate encryption declaration', 'missing source encryption declaration', 'source encryption declaration', 'public symlink']) {
  test('the adapter rejects ' + condition + ' before the native SDK boundary', () => withFixture(context => {
    const { fixture } = context;
    if (condition === 'source identity') writeFileSync(join(fixture.mobile, 'app.config.js'), readFileSync(join(fixture.mobile, 'app.config.js'), 'utf8') + '\nmodule.exports.expo.ios.bundleIdentifier="com.public.invalid";\n');
    if (condition === 'source dependency') {
      const metadata = JSON.parse(readFileSync(join(fixture.mobile, 'package.json'), 'utf8')); metadata.dependencies['react-native'] = '0.0.0';
      writeFileSync(join(fixture.mobile, 'package.json'), JSON.stringify(metadata));
    }
    if (condition === 'native configuration') {
      const file = join(fixture.prepared, 'android/app/build.gradle');
      writeFileSync(file, readFileSync(file, 'utf8').replace('versionCode = System.getenv("MPRLAB_MOBILE_VERSION_CODE").toInteger()', 'versionCode = 1'));
    }
    if (condition === 'source encryption declaration') {
      const file = join(fixture.mobile, 'app.config.js');
      writeFileSync(file, readFileSync(file, 'utf8').replace('usesNonExemptEncryption: false', 'usesNonExemptEncryption: true'));
    }
    if (condition === 'missing source encryption declaration') {
      const file = join(fixture.mobile, 'app.config.js');
      writeFileSync(file, readFileSync(file, 'utf8').replace('config: { usesNonExemptEncryption: false },', ''));
    }
    if (['missing encryption declaration', 'non-exempt encryption declaration', 'string encryption declaration', 'duplicate encryption declaration'].includes(condition)) {
      const file = join(fixture.prepared, 'ios/LoopAware/Info.plist');
      const original = readFileSync(file, 'utf8').replace(/<key>ITSAppUsesNonExemptEncryption<\/key>\s*<false\s*\/>/, '');
      const declaration = condition === 'missing encryption declaration' ? '' : condition === 'duplicate encryption declaration' ? '<key>ITSAppUsesNonExemptEncryption</key><false/><key>ITSAppUsesNonExemptEncryption</key><string>false</string>' : '<key>ITSAppUsesNonExemptEncryption</key>' + (condition === 'non-exempt encryption declaration' ? '<true/>' : '<string>false</string>');
      writeFileSync(file, original.replace('<dict>', '<dict>' + declaration));
    }
    if (condition === 'public symlink') {
      const outside = join(fixture.directory, 'outside-public-fixture.mjs'); writeFileSync(outside, 'public fixture');
      symlinkSync(outside, join(fixture.mobile, 'src/link.mjs'));
    }
    const result = context.invoke(); assert.equal(result.status, 2, result.stderr);
    assert.equal(existsSync(context.resultPath), false, 'do not invoke native SDK for an invalid selected context');
    assert.match(result.stderr, /native|preparation|configuration|symlink|symbolic/i);
  }));
}

test('a signal preserves the original status, terminates native descendants, and removes the owned stage', async () => withFixture(async context => {
  const child = spawn(process.execPath, context.args, { cwd: context.fixture.repository, env: { ...context.environment, PUBLIC_NATIVE_MODE: 'wait' }, stdio: ['ignore','pipe','pipe'] });
  let diagnostics = ''; child.stderr.on('data', bytes => { diagnostics += bytes; });
  const exited = new Promise(resolveExit => child.once('close', (status, signal) => resolveExit({ status, signal })));
  try {
    const cutoff = Date.now() + 5000;
    while (!existsSync(context.readyPath) && Date.now() < cutoff) await new Promise(resolveWait => setTimeout(resolveWait, 10));
    assert.equal(existsSync(context.readyPath), true, diagnostics);
    const native = JSON.parse(readFileSync(context.readyPath, 'utf8'));
    child.kill('SIGTERM');
    const result = await exited;
    assert.deepEqual(result, { status: 143, signal: null });
    assert.equal(existsSync(native.source), false);
    for (const pid of [native.pid, native.descendant]) {
      const state = spawnSync('ps', ['-p', String(pid), '-o', 'state='], { encoding: 'utf8' }).stdout.trim();
      assert.ok(!state || state.startsWith('Z'), 'owned native process must terminate');
    }
  } finally { if (child.exitCode === null) child.kill('SIGTERM'); await exited; }
}));
