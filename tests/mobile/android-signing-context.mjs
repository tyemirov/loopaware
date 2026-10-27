// @ts-check
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { copyFileSync, mkdirSync, mkdtempSync, realpathSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { tmpdir } from 'node:os';
import { createRequire } from 'node:module';
const root = resolve(import.meta.dirname, '../..');
const require = createRequire(join(root, 'mobile/package.json'));
const { parse, stringify } = require('yaml');
const gateway = process.env.MPRLAB_GATEWAY_EXECUTABLE || 'mprlab-gateway';
const originalManifest = parse(readFileSync(join(root, '.mprlab/deploy/resources.yml'), 'utf8'));
const generated = readFileSync(join(root, 'mobile/prepared/android/app/build.gradle'), 'utf8');
const signingBody = /signingConfigs \{\s*release \{([\s\S]*?)\n        \}/.exec(generated)?.[1];
assert.ok(signingBody, 'Generated release signing configuration is required.');
const wrapper = join(root, 'mobile/prepared/android/gradlew');
const fixture = realpathSync(mkdtempSync(join(tmpdir(), 'loopaware-sdk-signing-')));

/** @param {string} path @param {string} body */
function write(path, body) { mkdirSync(dirname(path), { recursive: true }); writeFileSync(path, body); }
/** @param {string[]} args @param {string} cwd @param {NodeJS.ProcessEnv} env */
function git(args, cwd, env) {
  const result = spawnSync('git', args, { cwd, env, encoding: 'utf8' });
  assert.equal(result.status, 0, result.stderr);
}
try {
  for (const kind of ['relative', 'absolute']) {
    const owner = join(fixture, kind), app = join(owner, 'application'), operator = join(owner, 'operator');
    const tools = join(owner, 'tools'); mkdirSync(tools, { recursive: true });
    const document = structuredClone(originalManifest), selected = document.mprlab_resources;
    selected.resources = { private: selected.resources.private, mobile: selected.resources.mobile };
    delete selected.hosts;
    const names = Object.keys(selected.inputs);
    const env = { ...process.env };
    for (const name of Object.keys(env)) if (names.includes(name) || name.startsWith('GIT_') || name.startsWith('MPRLAB_APP_') || name.startsWith('MPRLAB_LIFECYCLE_')) delete env[name];
    Object.assign(env, { CI: '1', PATH: `${tools}:${process.env.PATH}`, MPRLAB_GATEWAY_CACHE_ROOT: join(owner, 'cache') });
    const keystore = join(app, 'configs/signing/upload.jks');
    write(keystore, 'public synthetic signing input');
    const assignments = Object.fromEntries(names.map(name => [name, 'synthetic-signing-context-input']));
    assignments.LOOPAWARE_ANDROID_KEYSTORE = kind === 'relative' ? 'configs/signing/upload.jks' : keystore;
    write(join(app, selected.environment_file), Object.entries(assignments).map(([name, value]) => `${name}=${JSON.stringify(value)}`).join('\n') + '\n');
    write(join(app, '.mprlab/deploy/resources.yml'), stringify(document));
    write(join(app, '.gitignore'), '/.mprlab/deploy/.env\n/configs/signing/\n');
    write(join(app, 'Makefile'), `ci:\n\t${process.execPath} ci-hook.mjs\n`);
    write(join(app, 'mobile/scripts/build-store-artifact.mjs'), "throw new Error('The fixture must stop in CI before artifact assembly.');\n");
    write(join(operator, 'inventory/hosts.yml'), 'all:\n  children:\n    gateway:\n      hosts: {}\n');
    const mobile = join(app, 'mobile/prepared');
    write(join(mobile, 'package.json'), '{"name":"sdk-signing-fixture","version":"1.0.0","private":true}\n');
    write(join(mobile, 'package-lock.json'), '{"name":"sdk-signing-fixture","version":"1.0.0","lockfileVersion":3,"packages":{"":{"name":"sdk-signing-fixture","version":"1.0.0"}}}\n');
    write(join(mobile, 'verify.mjs'), "console.log('synthetic native boundary ready');\n");
    write(join(mobile, 'android/settings.gradle'), "rootProject.name = 'signing-boundary'\n");
    write(join(mobile, 'android/gradlew'), `#!/bin/sh\nexec "${process.execPath}" "$PUBLIC_NATIVE_DRIVER"\n`);
    write(join(mobile, 'android/app/settings.gradle'), "rootProject.name = 'generated-signing-expression'\n");
    const closure = `{ ->\n${signingBody}\n}`;
    write(join(mobile, 'android/app/build.gradle'), `
class SigningProbe {
    org.gradle.api.Project project
    File selectedFile
    File file(Object path) { project.file(path) }
    void storeFile(File path) { selectedFile = path }
    void storePassword(String value) {}
    void keyAlias(String value) {}
    void keyPassword(String value) {}
}
tasks.register('verifySigning') {
    doLast {
        def probe = new SigningProbe(project: project)
        def configure = new GroovyShell().evaluate(${JSON.stringify(closure)})
        configure.delegate = probe
        configure.resolveStrategy = Closure.DELEGATE_ONLY
        configure.call()
        def captured = new File(System.getenv('LOOPAWARE_ANDROID_KEYSTORE'))
        def qualified = new File(System.getenv('MPRLAB_APP_ROOT'), 'configs/signing/upload.jks').canonicalFile
        def facts = [capturedIsRelative: !captured.isAbsolute(), exists: probe.selectedFile.isFile(), matchesQualifiedInput: probe.selectedFile.canonicalFile == qualified]
        new File(System.getenv('PUBLIC_SIGNING_RESULT')).text = groovy.json.JsonOutput.toJson(facts)
    }
}
`);
    const driver = join(app, 'native-driver.mjs');
    write(driver, `
import { spawnSync } from 'node:child_process';
import { resolve } from 'node:path';
const result = spawnSync('/bin/sh', [${JSON.stringify(wrapper)}, '--no-daemon', '--console=plain', '--max-workers=1', '-Dorg.gradle.jvmargs=-Xmx256m -XX:MaxMetaspaceSize=256m', '-p', resolve('app'), 'verifySigning'], { env: process.env, encoding: 'utf8', stdio: ['ignore','pipe','pipe'] });
if (result.status !== 0) { process.stderr.write(result.stdout + result.stderr); process.exit(result.status || 2); }
process.stderr.write('controlled native boundary stopped before bundle compilation\\n');
process.exit(29);
`);
    write(join(app, 'ci-hook.mjs'), `
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
const root = ${JSON.stringify(app)};
const resultPath = ${JSON.stringify(join(owner, 'facts.json'))};
const request = { schema_version: 3, source_root: resolve('mobile/prepared'), platform: 'android', output: ${JSON.stringify(join(owner, 'never-built.aab'))}, application_identifier: 'com.mprlab.loopaware', version: '1.0.0', build_number: '42', release_timestamp: '2026-10-27T10:00:00Z', verify_script: 'verify.mjs', android: { module: 'app', version_name_environment: 'MPRLAB_MOBILE_VERSION_NAME', version_code_environment: 'MPRLAB_MOBILE_VERSION_CODE', signing_environment: ['LOOPAWARE_ANDROID_KEYSTORE','LOOPAWARE_ANDROID_STORE_PASSWORD','LOOPAWARE_ANDROID_KEY_ALIAS','LOOPAWARE_ANDROID_KEY_PASSWORD'] } };
const result = spawnSync(process.env.MPRLAB_GATEWAY_EXECUTABLE, ['mobile-build-operation'], { env: { ...process.env, MPRLAB_APPLICATION_LIFECYCLE_LOCK: 'exclusive', MPRLAB_LIFECYCLE_OPERATION: 'release', MPRLAB_APP_ROOT: root, LOOPAWARE_ANDROID_KEYSTORE: ${JSON.stringify(keystore)}, PUBLIC_NATIVE_DRIVER: resolve('native-driver.mjs'), PUBLIC_SIGNING_RESULT: resultPath, MPRLAB_MOBILE_BUILD_INTENT: ${JSON.stringify(join(owner, 'intent.json'))} }, input: JSON.stringify(request), encoding: 'utf8' });
assert.notEqual(result.status, 0);
assert.match(result.stderr, /controlled native boundary stopped before bundle compilation/, result.stdout + result.stderr);
const facts = JSON.parse(readFileSync(resultPath, 'utf8'));
assert.equal(facts.capturedIsRelative, ${kind === 'relative'});
assert.equal(facts.exists && facts.matchesQualifiedInput, true, 'Captured signing input must resolve against the original qualified root.');
console.log('SDK_SIGNING_CONTEXT_OK:${kind}');
process.exit(73);
`);
    for (const name of ['npm', 'gh']) {
      const path = join(tools, name); copyFileSync(process.execPath, path);
      writeFileSync(path, `#!${process.execPath}\n${name === 'npm' ? "if(process.argv.includes('--version'))console.log('11.9.0');" : "console.error('Unexpected provider boundary in SDK signing fixture');process.exit(97);"}\n`);
    }
    git(['init', '--quiet', '--initial-branch=master'], app, env);
    git(['add', '.'], app, env);
    git(['-c','user.name=Signing Context Fixture','-c','user.email=signing-context@example.invalid','commit','--quiet','-m','Declare signing context fixture'], app, env);
    const remote = join(owner, 'origin.git'), origin = 'https://github.com/MarcoPoloResearchLab/loopaware.git';
    git(['clone','--quiet','--bare',app,remote], owner, env);
    git(['remote','add','origin',origin], app, env);
    git(['config',`url.${remote}.insteadOf`,origin], app, env);
    const result = spawnSync(gateway, ['app-release','--app-root',app,'--operator-root',operator], { cwd: app, env, input: '', encoding: 'utf8', timeout: 90_000, maxBuffer: 8*1024*1024 });
    assert.equal(result.error, undefined);
    assert.match(result.stdout + result.stderr, new RegExp(`SDK_SIGNING_CONTEXT_OK:${kind}`), result.stdout + result.stderr);
    assert.match(result.stdout + result.stderr, /Error 73/);
    console.log(`Actual SDK signing rebinding passed: ${kind}.`);
  }
} finally { rmSync(fixture, { recursive: true, force: true }); }
