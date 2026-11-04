// @ts-check
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { copyFileSync, existsSync, mkdirSync, mkdtempSync, realpathSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import originalManifest from './fixtures/signing-context-resources.json' with { type: 'json' };
const root = resolve(import.meta.dirname, '../..');
const gateway = process.env.MPRLAB_GATEWAY_EXECUTABLE || 'mprlab-gateway';
/** @param {string} path @param {string} body */
function write(path, body) { mkdirSync(dirname(path), { recursive: true }); writeFileSync(path, body); }
if (process.argv[2] === '--write-installation-fixture') {
  const application = process.argv[3];
  assert.ok(application, 'Select an installation fixture directory.');
  const selected = originalManifest.mprlab_resources;
  write(join(application, '.mprlab/deploy/resources.yml'), JSON.stringify(originalManifest));
  write(join(application, selected.environment.source), selected.environment.private_inputs.map(input => `${input.variable}=synthetic-runtime-installation-input`).join('\n') + '\n');
  process.exit(0);
}
const generated = readFileSync(join(root, 'mobile/prepared/android/app/build.gradle'), 'utf8');
const signingBody = /signingConfigs \{\s*release \{([\s\S]*?)\n        \}/.exec(generated)?.[1];
assert.ok(signingBody, 'Generated release signing configuration is required.');
const wrapper = join(root, 'mobile/prepared/android/gradlew');
mkdirSync(join(root, '_scratch'), { recursive: true });
const fixture = realpathSync(mkdtempSync(join(root, '_scratch/v8-signing-context-')));
/** @param {string[]} args @param {string} cwd @param {NodeJS.ProcessEnv} env */
function git(args, cwd, env) {
  const result = spawnSync('git', args, { cwd, env, encoding: 'utf8', stdio: ['ignore','pipe','pipe'] });
  assert.ifError(result.error); assert.equal(result.status, 0, result.stdout + result.stderr);
  return result.stdout.trim();
}
try {
  for (const kind of ['relative', 'absolute']) {
    const owner = join(fixture, kind), app = join(owner, 'application'), operator = join(owner, 'operator');
    const tools = join(owner, 'tools'); mkdirSync(tools, { recursive: true });
    const document = structuredClone(originalManifest), selected = document.mprlab_resources;
    selected.owner = 'anonymous';
    selected.lifecycle.ci.enabled = false;
    delete selected.lifecycle.observations;
    selected.lifecycle.operations = selected.lifecycle.operations.filter(entry => entry.name !== 'app_lifecycle.repository_read');
    selected.lifecycle.operations.push({ name: 'app_lifecycle.repository_read', polling: { initial: '1ms', maximum: '2ms', increment: '0s', multiplier: 2, attempt_limit: 3 } });
    selected.artifacts.mobile[0].build = { android: 'mobile/scripts/build-store-artifact.mjs' };
    delete selected.artifacts.mobile[0].ios;
    selected.publications = {};
    const names = selected.environment.private_inputs.map(input => input.variable);
    const env = { HOME: process.env.HOME, PATH: `${tools}:${process.env.PATH}`, CI: '1', GIT_SSH_COMMAND: '/usr/bin/false', JAVA_HOME: process.env.JAVA_HOME };
    const keystore = join(app, 'configs/signing/upload.jks');
    write(keystore, 'public synthetic signing input');
    const assignments = Object.fromEntries(names.map(name => [name, 'synthetic-signing-context-input']));
    assignments.LOOPAWARE_ANDROID_KEYSTORE = kind === 'relative' ? 'configs/signing/upload.jks' : keystore;
    write(join(app, selected.environment.source), Object.entries(assignments).map(([name, value]) => `${name}=${JSON.stringify(value)}`).join('\n') + '\n');
    write(join(app, '.mprlab/deploy/resources.yml'), JSON.stringify(document));
    write(join(app, '.gitignore'), '/.mprlab/deploy/.env\n/configs/signing/\n');
    const mobile = join(app, 'mobile/prepared'), factsPath = join(owner, 'facts.json');
    write(join(mobile, 'package.json'), '{"name":"sdk-signing-fixture","version":"1.0.0","private":true}\n');
    write(join(mobile, 'package-lock.json'), '{"name":"sdk-signing-fixture","version":"1.0.0","lockfileVersion":3,"packages":{"":{"name":"sdk-signing-fixture","version":"1.0.0"}}}\n');
    write(join(mobile, 'verify.mjs'), "console.log('synthetic native boundary ready');\n");
    write(join(mobile, 'android/settings.gradle'), "rootProject.name = 'signing-boundary'\n");
    write(join(mobile, 'android/gradlew'), '#!/bin/sh\nexec node ../native-driver.mjs\n');
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
        new File(${JSON.stringify(factsPath)}).text = groovy.json.JsonOutput.toJson(facts)
    }
}
`);
    write(join(mobile, 'native-driver.mjs'), `
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { resolve } from 'node:path';
assert.equal(process.env.NPM_API_KEY, undefined, 'Unselected private input must not reach native Gradle');
const result = spawnSync('/bin/sh', [${JSON.stringify(wrapper)}, '--no-daemon', '--console=plain', '--max-workers=1', '-Dorg.gradle.jvmargs=-Xmx256m -XX:MaxMetaspaceSize=256m', '-p', resolve('app'), 'verifySigning'], { env: process.env, encoding: 'utf8', stdio: ['ignore','pipe','pipe'] });
if (result.error) throw result.error;
if (result.status !== 0) { process.stderr.write(result.stdout + result.stderr); process.exit(result.status ?? 2); }
process.stderr.write('controlled native boundary stopped before bundle compilation\\n');
process.exit(29);
`);
    write(join(app, 'mobile/scripts/build-store-artifact.mjs'), `
import { spawnSync } from 'node:child_process';
const args = process.argv.slice(2);
const value = flag => args[args.indexOf(flag) + 1];
const request = { schema_version: 3, source_root: value('--mobile-dir'), platform: 'android', output: value('--output'), application_identifier: 'com.mprlab.loopaware', version: process.env.MPRLAB_ARTIFACT_VERSION, build_number: '42', release_timestamp: value('--release-timestamp'), verify_script: 'verify.mjs', android: { module: 'app', version_name_environment: 'MPRLAB_MOBILE_VERSION_NAME', version_code_environment: 'MPRLAB_MOBILE_VERSION_CODE', signing_environment: ['LOOPAWARE_ANDROID_KEYSTORE','LOOPAWARE_ANDROID_STORE_PASSWORD','LOOPAWARE_ANDROID_KEY_ALIAS','LOOPAWARE_ANDROID_KEY_PASSWORD'] } };
const result = spawnSync(process.env.MPRLAB_GATEWAY_EXECUTABLE, ['mobile-build-operation'], { env: { ...process.env, LOOPAWARE_ANDROID_KEYSTORE: '/ambient-shadow-must-not-win.jks' }, input: JSON.stringify(request), encoding: 'utf8', stdio: ['pipe','pipe','pipe'] });
if (result.error) throw result.error;
process.stdout.write(result.stdout); process.stderr.write(result.stderr); process.exit(result.status ?? 2);
`);
    const providerLog = join(owner, 'provider.jsonl'); write(providerLog, '');
    for (const name of ['npm', 'gh']) {
      const path = join(tools, name); copyFileSync(process.execPath, path);
      const body = name === 'npm' ? "const args=process.argv.slice(2);if(args[0]==='--version')console.log('11.9.0');else if(args[0]!=='ci')throw new Error('Unexpected npm operation');" : `const fs=require('node:fs');const args=process.argv.slice(2);fs.appendFileSync(${JSON.stringify(providerLog)},JSON.stringify(args)+'\\n');if(JSON.stringify(args)!==JSON.stringify(['api','--method','GET','repos/example/anonymous','--jq','.full_name']))throw new Error('Unexpected provider operation');process.stdout.write('example/anonymous\\n');`;
      writeFileSync(path, `#!${process.execPath}\n${body}\n`);
    }
    const remote = join(owner, 'origin.git');
    git(['init','--bare','--initial-branch=fixture-default',remote], owner, env);
    git(['init','--initial-branch=fixture-default'], app, env);
    git(['config','user.name','Anonymous Signing Fixture'], app, env);
    git(['config','user.email','fixture@example.invalid'], app, env);
    git(['config','commit.gpgSign','false'], app, env);
    git(['add','.'], app, env); git(['commit','-m','Declare anonymous signing context'], app, env);
    git(['remote','add','origin',remote], app, env); git(['push','-u','origin','fixture-default'], app, env);
    const canonical = 'git@github.com:example/anonymous.git';
    git(['remote','set-url','origin',canonical], app, env);
    git(['config',`url.file://${remote}.insteadOf`,canonical], app, env); git(['remote','set-head','origin','--auto'], app, env);
    const result = spawnSync(gateway, ['app-release','--app-root',app,'--operator-root',operator], { cwd: app, env, input: '', encoding: 'utf8', timeout: 90_000, maxBuffer: 8*1024*1024 });
    assert.ifError(result.error); assert.notEqual(result.status, 0);
    const output = result.stdout + result.stderr;
    assert.match(output, /controlled native boundary stopped before bundle compilation/, output);
    assert.match(output, /29/, output);
    const facts = JSON.parse(readFileSync(factsPath, 'utf8'));
    assert.equal(facts.capturedIsRelative, kind === 'relative');
    assert.equal(facts.exists && facts.matchesQualifiedInput, true, 'Captured signing input must resolve against the original qualified root.');
    assert.ok(!output.includes('synthetic-signing-context-input'));
    assert.equal(git(['status','--porcelain'], app, env), '', 'Release failure must preserve synchronized application source');
    assert.doesNotMatch(readFileSync(providerLog, 'utf8'), /"(create|upload)"/);
    console.log(`Actual SDK signing rebinding passed: ${kind}.`);
  }
} finally { rmSync(fixture, { recursive: true, force: true }); assert.equal(existsSync(fixture), false); }
