// @ts-check
import assert from 'node:assert/strict';
import { copyFileSync, existsSync, mkdirSync, mkdtempSync, realpathSync, rmSync, symlinkSync, writeFileSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';
import { spawnSync } from 'node:child_process';
import { X509Certificate } from 'node:crypto';
import { publicFixture } from './preparation.mjs';

const selected = publicFixture();
const fixture = selected.directory;
const checkout = join(fixture, 'working checkout');
const snapshot = selected.repository;
const prepared = selected.prepared;
const recorded = selected.record(); assert.equal(recorded.status, 0, recorded.stderr);
const javaHome = process.env.JAVA_HOME;
assert.ok(javaHome, 'Run the signing source checks through the repository Make target.');
const signing = {
    LOOPAWARE_ANDROID_KEYSTORE: 'configs/signing/upload.jks',
    LOOPAWARE_ANDROID_STORE_PASSWORD: 'fixture-password',
    LOOPAWARE_ANDROID_KEY_ALIAS: 'upload',
    LOOPAWARE_ANDROID_KEY_PASSWORD: 'fixture-password',
};
try {
    for (const directory of [join(checkout, 'configs/signing'), join(checkout, 'mobile'), join(snapshot, 'mobile/scripts'), prepared]) {
        mkdirSync(directory, { recursive: true });
    }
    for (const name of ['build-store-artifact.mjs', 'android-signing.mjs', 'signing-inputs.mjs', 'native-build-process.mjs', 'mobile-calver-version.mjs']) {
        copyFileSync(resolve('mobile/scripts', name), join(snapshot, 'mobile/scripts', name));
    }
    const keystore = join(checkout, signing.LOOPAWARE_ANDROID_KEYSTORE);
    const keytool = join(javaHome, 'bin/keytool');
    const generated = spawnSync(keytool, ['-genkeypair', '-keystore', keystore, '-alias', signing.LOOPAWARE_ANDROID_KEY_ALIAS,
        '-storepass:env', 'LOOPAWARE_ANDROID_STORE_PASSWORD', '-keypass:env', 'LOOPAWARE_ANDROID_KEY_PASSWORD',
        '-keyalg', 'RSA', '-dname', 'CN=Signing Fixture', '-validity', '1'], { env: { ...process.env, ...signing }, encoding: 'utf8' });
    assert.equal(generated.status, 0, generated.stderr);
    const exported = spawnSync(keytool, ['-exportcert', '-rfc', '-keystore', keystore, '-alias', signing.LOOPAWARE_ANDROID_KEY_ALIAS,
        '-storepass:env', 'LOOPAWARE_ANDROID_STORE_PASSWORD'], { env: { ...process.env, ...signing }, encoding: 'utf8' });
    assert.equal(exported.status, 0, exported.stderr);
    const identity = { uploadKey: { sha256: new X509Certificate(exported.stdout).fingerprint256 } };
    const identityPath = join(snapshot, 'mobile/android-release-identity.json');
    writeFileSync(identityPath, JSON.stringify(identity));
    writeFileSync(join(checkout, 'mobile/android-release-identity.json'), JSON.stringify({ uploadKey: { sha256: 'uncommitted identity' } }));
    writeFileSync(join(snapshot, 'private-environment-operation'), `
const fs = require('node:fs'), assert = require('node:assert/strict');
const request = JSON.parse(fs.readFileSync(0, 'utf8'));
assert.deepEqual(request, {root: process.env.MPRLAB_APP_ROOT, keys: ['LOOPAWARE_ANDROID_KEYSTORE', 'LOOPAWARE_ANDROID_STORE_PASSWORD', 'LOOPAWARE_ANDROID_KEY_ALIAS', 'LOOPAWARE_ANDROID_KEY_PASSWORD']});
if (process.env.TEST_CAPTURED_SIGNING_STATUS) process.exit(Number(process.env.TEST_CAPTURED_SIGNING_STATUS));
process.stdout.write(process.env.TEST_CAPTURED_SIGNING_INPUTS);
`);
    writeFileSync(join(snapshot, 'mobile/mobile-build-operation'), `
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const request = JSON.parse(fs.readFileSync(0, 'utf8'));
assert.equal(request.schema_version, 3);
assert.equal(Object.hasOwn(request, 'preparation_manifest'), false);
assert.equal(request.verify_script, 'scripts/verify-store-preparation.mjs');
assert.equal(request.platform, 'android');
assert.equal(fs.realpathSync(request.source_root), fs.realpathSync(process.cwd()));
assert.ok(request.source_root.includes('loopaware-native-source-'));
assert.equal(request.application_identifier, 'com.mprlab.loopaware');
assert.equal(process.env.LOOPAWARE_ANDROID_KEYSTORE, fs.realpathSync(path.join(process.env.MPRLAB_APP_ROOT, 'configs/signing/upload.jks')));
assert.ok(fs.readFileSync(process.env.LOOPAWARE_ANDROID_KEYSTORE).length);
for (const name of ['GH_TOKEN', 'GITHUB_TOKEN', 'NPM_API_KEY']) assert.equal(process.env[name], undefined);
console.log('native-builder-reached');
process.exit(29);
`);
    const outside = join(fixture, 'outside.jks');
    copyFileSync(keystore, outside);
    symlinkSync(outside, join(checkout, 'configs/signing/escaped.jks'));
    writeFileSync(join(checkout, 'configs/signing/empty.jks'), '');

    for (const scenario of ['captured-only file', 'conflicting ambient file', 'absolute file', 'absent root', 'relative root', 'outside file', 'symlink escape', 'empty file', 'invalid capture', 'missing captured value', 'capture failure', 'wrong identity']) {
        /** @type {NodeJS.ProcessEnv} */
        const captured = { ...signing };
        const environment = { ...selected.environment, JAVA_HOME: javaHome, ANDROID_STUDIO_JAVA_HOME: javaHome, MPRLAB_APP_ROOT: checkout, MPRLAB_ARTIFACT_VERSION: '1.2.3',
            MPRLAB_GATEWAY_EXECUTABLE: process.execPath, GH_TOKEN: 'fixture-token', GITHUB_TOKEN: 'fixture-token', NPM_API_KEY: 'fixture-token' };
        if (scenario === 'conflicting ambient file') Object.assign(environment, { LOOPAWARE_ANDROID_KEYSTORE: '/ambient-wrong-keystore', LOOPAWARE_ANDROID_STORE_PASSWORD: 'wrong', LOOPAWARE_ANDROID_KEY_ALIAS: 'wrong', LOOPAWARE_ANDROID_KEY_PASSWORD: 'wrong' });
        if (scenario === 'absolute file') captured.LOOPAWARE_ANDROID_KEYSTORE = keystore;
        if (scenario === 'absent root') delete environment.MPRLAB_APP_ROOT;
        if (scenario === 'relative root') environment.MPRLAB_APP_ROOT = 'working checkout';
        if (scenario === 'outside file') captured.LOOPAWARE_ANDROID_KEYSTORE = outside;
        if (scenario === 'symlink escape') captured.LOOPAWARE_ANDROID_KEYSTORE = 'configs/signing/escaped.jks';
        if (scenario === 'empty file') captured.LOOPAWARE_ANDROID_KEYSTORE = 'configs/signing/empty.jks';
        if (scenario === 'wrong identity') writeFileSync(identityPath, JSON.stringify({ uploadKey: { sha256: 'wrong identity' } }));
        environment.TEST_CAPTURED_SIGNING_INPUTS = JSON.stringify(captured);
        if (scenario === 'invalid capture') environment.TEST_CAPTURED_SIGNING_INPUTS = 'private fixture invalid response';
        if (scenario === 'missing captured value') environment.TEST_CAPTURED_SIGNING_INPUTS = JSON.stringify({ ...captured, LOOPAWARE_ANDROID_KEY_ALIAS: '' });
        if (scenario === 'capture failure') environment.TEST_CAPTURED_SIGNING_STATUS = '19';
        const result = spawnSync(process.execPath, [join(snapshot, 'mobile/scripts/build-store-artifact.mjs'),
            '--mobile-dir', prepared, '--output', join(fixture, 'android.aab'), '--release-timestamp', '2026-09-10T07:07:08Z'],
        { cwd: snapshot, env: environment, encoding: 'utf8' });
        if (['captured-only file', 'conflicting ambient file', 'absolute file'].includes(scenario)) {
            assert.equal(result.status, 29, `${scenario}: ${result.stderr}`);
            assert.match(result.stdout, /native-builder-reached/);
        } else {
            assert.equal(result.status, 2, `${scenario}: ${result.stderr}`);
            assert.equal(result.stdout, '');
            assert.match(result.stderr, scenario.includes('root') ? /absolute MPRLAB_APP_ROOT/
                : scenario === 'wrong identity' ? /differs from the registered upload identity/
                : ['invalid capture', 'missing captured value'].includes(scenario) ? /invalid captured private inputs/
                : scenario === 'capture failure' ? /could not resolve the captured private inputs/ : /nonempty file under configs\/signing/);
            assert.doesNotMatch(result.stderr, /private fixture invalid response/);
        }
        assert.equal(existsSync(join(snapshot, 'configs/signing')), false);
        console.info(`Android signing source check passed: ${scenario}.`);
    }
} finally {
    rmSync(fixture, { recursive: true, force: true });
}
