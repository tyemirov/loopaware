// @ts-check
import assert from 'node:assert/strict';
import { copyFileSync, existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync, symlinkSync, renameSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { resolve, join, dirname } from 'node:path';
import { tmpdir } from 'node:os';
import { pathToFileURL } from 'node:url';
import test from 'node:test';

const root = resolve(import.meta.dirname, '../..');
const nativeFiles = [
  'ios/Podfile', 'ios/Podfile.lock', 'ios/Podfile.properties.json', 'ios/.xcode.env',
  'ios/LoopAware.xcodeproj/project.pbxproj', 'ios/LoopAware.xcodeproj/xcshareddata/xcschemes/LoopAware.xcscheme',
  'ios/LoopAware.xcworkspace/contents.xcworkspacedata', 'ios/LoopAware/Info.plist', 'ios/LoopAware/AppDelegate.swift',
  'android/gradlew', 'android/settings.gradle', 'android/build.gradle', 'android/gradle.properties',
  'android/gradle/wrapper/gradle-wrapper.jar', 'android/gradle/wrapper/gradle-wrapper.properties',
  'android/app/build.gradle', 'android/app/src/main/AndroidManifest.xml',
];

/** Public captured source and retained native context, with no private operator inputs.
 * @returns {{directory:string,repository:string,mobile:string,prepared:string,environment:Record<string,string>,record:()=>import('node:child_process').SpawnSyncReturns<string>,verify:()=>import('node:child_process').SpawnSyncReturns<string>,cleanup:()=>void}}
 */
export function publicFixture() {
  const directory = mkdtempSync(join(tmpdir(), 'loopaware-semantic-'));
  const repository = join(directory, 'captured application');
  const mobile = join(repository, 'mobile'), prepared = join(mobile, 'prepared');
  mkdirSync(join(mobile, 'scripts'), { recursive: true });
  mkdirSync(join(mobile, 'src'));
  mkdirSync(join(directory, 'home'));
  writeFileSync(join(repository, '.gitignore'), 'node_modules/\nmobile/prepared/\n.env*\n');
  writeFileSync(join(repository, 'Makefile'), 'fixture:\n\ttrue\n');
  assert.equal(spawnSync('git', ['init', '--quiet', repository], { encoding: 'utf8' }).status, 0);
  for (const name of ['app.config.js', 'package.json', 'package-lock.json']) copyFileSync(join(root, 'mobile', name), join(mobile, name));
  const metadata = JSON.parse(readFileSync(join(mobile, 'package.json'), 'utf8'));
  writeFileSync(join(mobile, 'index.ts'), 'export const entry = "public";\n');
  writeFileSync(join(mobile, 'App.tsx'), 'export const application = "public initial source";\n');
  writeFileSync(join(mobile, 'src/product.mjs'), 'export const product = "initial";\n');
  for (const name of ['record-store-preparation.mjs', 'verify-store-preparation.mjs', 'preparation-sources.mjs', 'prepare-store.mjs', 'build-store-artifact.mjs', 'android-signing.mjs', 'signing-inputs.mjs', 'native-build-process.mjs', 'mobile-calver-version.mjs']) {
    copyFileSync(join(root, 'mobile/scripts', name), join(mobile, 'scripts', name));
  }
  for (const name of nativeFiles) {
    mkdirSync(dirname(join(prepared, name)), { recursive: true });
    copyFileSync(join(root, 'mobile/prepared', name), join(prepared, name));
  }
  const production = structuredClone(metadata); delete production.dependencies['expo-dev-client'];
  writeFileSync(join(prepared, 'package.json'), JSON.stringify(production));
  copyFileSync(join(root, 'mobile/prepared/package-lock.json'), join(prepared, 'package-lock.json'));
  copyFileSync(join(root, 'mobile/prepared/app.config.snapshot.json'), join(prepared, 'app.config.snapshot.json'));
  mkdirSync(join(prepared, 'scripts'), { recursive: true });
  copyFileSync(join(root, 'mobile/scripts/verify-store-preparation.mjs'), join(prepared, 'scripts/verify-store-preparation.mjs'));
  mkdirSync(join(prepared, 'src'));
  copyFileSync(join(mobile, 'src/product.mjs'), join(prepared, 'src/product.mjs'));
  copyFileSync(join(mobile, 'App.tsx'), join(prepared, 'App.tsx'));
  copyFileSync(join(mobile, 'index.ts'), join(prepared, 'index.ts'));
  const environment = { PATH: process.env.PATH ?? '', HOME: join(directory, 'home'), TMPDIR: directory, NODE_ENV: 'production' };
  const record = () => spawnSync(process.execPath, [join(mobile, 'scripts/record-store-preparation.mjs'), repository], { env: environment, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] });
  const verify = () => spawnSync(process.execPath, [join(mobile, 'scripts/verify-store-preparation.mjs')], { cwd: prepared, env: environment, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] });
  return { directory, repository, mobile, prepared, environment, record, verify, cleanup: () => rmSync(directory, { recursive: true, force: true }) };
}

/** @param {(fixture: ReturnType<typeof publicFixture>)=>void} check */
function withFixture(check) {
  const fixture = publicFixture();
  try { const recorded = fixture.record(); assert.equal(recorded.status, 0, recorded.stderr); check(fixture); }
  finally { fixture.cleanup(); }
}

if (process.argv[1] && pathToFileURL(resolve(process.argv[1])).href === import.meta.url) {
  test('ordinary source and native metadata changes do not invalidate native preparation', () => withFixture(fixture => {
    writeFileSync(join(fixture.mobile, 'src/product.mjs'), 'export const product = "current source";\n');
    writeFileSync(join(fixture.prepared, 'ios/Podfile.lock'), readFileSync(join(fixture.prepared, 'ios/Podfile.lock'), 'utf8') + '\n# native metadata can regenerate\n');
    const result = fixture.verify(); assert.equal(result.status, 0, result.stderr);
    const record = JSON.parse(readFileSync(join(fixture.prepared, 'native-preparation.json'), 'utf8'));
    assert.equal(record.schema_version, 3); assert.equal(Object.hasOwn(record, 'files'), false);
    assert.equal(existsSync(join(fixture.prepared, 'source-preparation.json')), false);
  }));
  for (const [name, mutate] of [
    ['old full-content schema', fixture => writeFileSync(join(fixture.prepared, 'native-preparation.json'), JSON.stringify({ schema_version: 2, files: { 'source.js': 'b2xk' } }))],
    ['native identifier', fixture => { const p = join(fixture.prepared, 'ios/LoopAware.xcodeproj/project.pbxproj'); writeFileSync(p, readFileSync(p, 'utf8').replaceAll('PRODUCT_BUNDLE_IDENTIFIER = com.mprlab.loopaware;', 'PRODUCT_BUNDLE_IDENTIFIER = com.public.invalid;')); }],
    ['native identifier comment bait', fixture => { const p = join(fixture.prepared, 'android/app/build.gradle'); writeFileSync(p, readFileSync(p, 'utf8').replace("applicationId 'com.mprlab.loopaware'", "// applicationId 'com.mprlab.loopaware'\n        applicationId 'com.public.invalid'")); }],
    ['lifecycle version assignment', fixture => { const p = join(fixture.prepared, 'android/app/build.gradle'); writeFileSync(p, readFileSync(p, 'utf8').replace('versionCode = System.getenv("MPRLAB_MOBILE_VERSION_CODE").toInteger()', 'versionCode = 1 // System.getenv("MPRLAB_MOBILE_VERSION_CODE")')); }],
    ['scheme target reference', fixture => { const p = join(fixture.prepared, 'ios/LoopAware.xcodeproj/xcshareddata/xcschemes/LoopAware.xcscheme'); writeFileSync(p, readFileSync(p, 'utf8').replace(/BlueprintIdentifier = "[^"]+"/g, 'BlueprintIdentifier = "000000000000000000000001"')); }],
    ['plist comment bait', fixture => { const p = join(fixture.prepared, 'ios/LoopAware/Info.plist'); writeFileSync(p, readFileSync(p, 'utf8').replace('<string>$(PRODUCT_BUNDLE_IDENTIFIER)</string>', '<!-- <string>$(PRODUCT_BUNDLE_IDENTIFIER)</string> --><string>wrong</string>')); }],
    ['missing required file', fixture => rmSync(join(fixture.prepared, 'ios/Podfile'))],
    ['ancestor symlink', fixture => { const p = join(fixture.prepared, 'ios'), moved = join(fixture.directory, 'public-ios'); renameSync(p, moved); symlinkSync(moved, p, 'dir'); }],
  ]) {
    test('rejects ' + name, () => withFixture(fixture => {
      mutate(fixture); const result = fixture.verify(); assert.notEqual(result.status, 0, name); assert.match(result.stderr, /native|preparation|configuration|selected/i);
    }));
  }
  for (const phase of ['npm', 'expo']) {
    test('canonical preparation preserves controlled ' + phase + ' failure without a success record', () => withFixture(fixture => {
      const hook = join(fixture.directory, 'provider-boundary.mjs');
      writeFileSync(hook, `import cp from 'node:child_process';
import { syncBuiltinESMExports } from 'node:module';
const actual=cp.spawnSync;
cp.spawnSync=(command,args,options)=>{
 if(command==='git')return actual(command,args,options);
 if(command==='npm'){
  if(process.env.PUBLIC_FAILURE_PHASE==='npm'){process.stderr.write('public native provider failure\\n');return {status:23};}
  return {status:0};
 }
 if(command==='npx' && args.includes('expo')){process.stderr.write('public native provider failure\\n');return {status:27};}
 throw new Error('unexpected native boundary: '+command);
};
syncBuiltinESMExports();
`);
      const result = spawnSync(process.execPath, ['--import', hook, join(fixture.mobile, 'scripts/prepare-store.mjs')], {
        env: { ...fixture.environment, PUBLIC_FAILURE_PHASE: phase }, encoding: 'utf8', stdio: ['ignore','pipe','pipe'],
      });
      assert.equal(result.status, phase === 'npm' ? 23 : 27, result.stderr);
      assert.match(result.stderr, /public native provider failure/);
      assert.equal(existsSync(join(fixture.prepared, 'native-preparation.json')), false);
    }));
  }

}
