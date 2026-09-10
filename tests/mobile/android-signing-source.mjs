// @ts-check
import assert from 'node:assert/strict';
import { copyFileSync, existsSync, mkdirSync, mkdtempSync, realpathSync, rmSync, symlinkSync, writeFileSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';
import { spawnSync } from 'node:child_process';
import { X509Certificate } from 'node:crypto';

const fixture = realpathSync(mkdtempSync(join(tmpdir(), 'loopaware signing source ')));
const checkout = join(fixture, 'working checkout');
const snapshot = join(fixture, 'source snapshot');
const prepared = join(snapshot, 'mobile/prepared');
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
    writeFileSync(join(prepared, 'app.config.snapshot.json'), JSON.stringify({ expo: { android: { package: 'com.example.fixture' } } }));
    writeFileSync(join(prepared, 'mobile-build-operation'), `
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const request = JSON.parse(fs.readFileSync(0, 'utf8'));
assert.equal(request.platform, 'android');
assert.equal(request.source_root, process.cwd());
assert.equal(request.application_identifier, 'com.example.fixture');
assert.equal(process.env.LOOPAWARE_ANDROID_KEYSTORE, path.join(process.env.MPRLAB_APP_ROOT, 'configs/signing/upload.jks'));
assert.ok(fs.readFileSync(process.env.LOOPAWARE_ANDROID_KEYSTORE).length);
for (const name of ['GH_TOKEN', 'GITHUB_TOKEN', 'NPM_API_KEY']) assert.equal(process.env[name], undefined);
console.log('native-builder-reached');
process.exit(29);
`);
    const outside = join(fixture, 'outside.jks');
    copyFileSync(keystore, outside);
    symlinkSync(outside, join(checkout, 'configs/signing/escaped.jks'));
    writeFileSync(join(checkout, 'configs/signing/empty.jks'), '');

    for (const scenario of ['relative file', 'absolute file', 'absent root', 'relative root', 'outside file', 'symlink escape', 'empty file', 'wrong identity']) {
        /** @type {NodeJS.ProcessEnv} */
        const environment = { ...process.env, ...signing, ANDROID_STUDIO_JAVA_HOME: javaHome, MPRLAB_APP_ROOT: checkout, MPRLAB_ARTIFACT_VERSION: '1.2.3',
            MPRLAB_GATEWAY_EXECUTABLE: process.execPath, GH_TOKEN: 'fixture-token', GITHUB_TOKEN: 'fixture-token', NPM_API_KEY: 'fixture-token' };
        if (scenario === 'absolute file') environment.LOOPAWARE_ANDROID_KEYSTORE = keystore;
        if (scenario === 'absent root') delete environment.MPRLAB_APP_ROOT;
        if (scenario === 'relative root') environment.MPRLAB_APP_ROOT = 'working checkout';
        if (scenario === 'outside file') environment.LOOPAWARE_ANDROID_KEYSTORE = outside;
        if (scenario === 'symlink escape') environment.LOOPAWARE_ANDROID_KEYSTORE = 'configs/signing/escaped.jks';
        if (scenario === 'empty file') environment.LOOPAWARE_ANDROID_KEYSTORE = 'configs/signing/empty.jks';
        if (scenario === 'wrong identity') writeFileSync(identityPath, JSON.stringify({ uploadKey: { sha256: 'wrong identity' } }));
        const result = spawnSync(process.execPath, [join(snapshot, 'mobile/scripts/build-store-artifact.mjs'),
            '--mobile-dir', prepared, '--output', join(fixture, 'android.aab'), '--release-timestamp', '2026-09-10T07:07:08Z'],
        { cwd: snapshot, env: environment, encoding: 'utf8' });
        if (scenario === 'relative file' || scenario === 'absolute file') {
            assert.equal(result.status, 29, `${scenario}: ${result.stderr}`);
            assert.match(result.stdout, /native-builder-reached/);
        } else {
            assert.equal(result.status, 2, `${scenario}: ${result.stderr}`);
            assert.equal(result.stdout, '');
            assert.match(result.stderr, scenario.includes('root') ? /absolute MPRLAB_APP_ROOT/
                : scenario === 'wrong identity' ? /differs from the registered upload identity/ : /nonempty file under configs\/signing/);
        }
        assert.equal(existsSync(join(snapshot, 'configs/signing')), false);
        console.info(`Android signing source check passed: ${scenario}.`);
    }
} finally {
    rmSync(fixture, { recursive: true, force: true });
}
