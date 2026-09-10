// @ts-check
import assert from 'node:assert/strict';
import { mkdtempSync, writeFileSync, rmSync, mkdirSync, realpathSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { resolve } from 'node:path';
import { spawnSync } from 'node:child_process';
import { X509Certificate } from 'node:crypto';
import { rootCertificates } from 'node:tls';
import { androidSigningEnvironment } from '../../mobile/scripts/android-signing.mjs';
import { executeSigningTool } from '../../mobile/scripts/signing-inputs.mjs';

const androidFixture = mkdtempSync(resolve(tmpdir(), 'loopaware-android-inputs-'));
try {
    mkdirSync(resolve(androidFixture, 'configs/signing'), { recursive: true });
    mkdirSync(resolve(androidFixture, 'mobile'));
    writeFileSync(resolve(androidFixture, 'configs/signing/upload.jks'), 'fixture');
    const certificate = rootCertificates[0];
    writeFileSync(resolve(androidFixture, 'mobile/android-release-identity.json'), JSON.stringify({ uploadKey: { sha256: new X509Certificate(certificate).fingerprint256 } }));
    const environment = { LOOPAWARE_ANDROID_KEYSTORE: 'configs/signing/upload.jks', LOOPAWARE_ANDROID_STORE_PASSWORD: 'fixture', LOOPAWARE_ANDROID_KEY_ALIAS: 'upload', LOOPAWARE_ANDROID_KEY_PASSWORD: 'fixture', JAVA_HOME: '/fixture/jdk', GH_TOKEN: 'private fixture', APP_STORE_CONNECT_API_KEY_ID: 'private fixture' };
    const result = await androidSigningEnvironment(androidFixture, androidFixture, environment, async (name, args, supplied) => {
        assert.equal(name, '/fixture/jdk/bin/keytool');
        assert.ok(args.includes('-storepass:env'));
        assert.equal(supplied.LOOPAWARE_ANDROID_STORE_PASSWORD, 'fixture');
        return { status: 0, stdout: certificate };
    });
    assert.equal(result.LOOPAWARE_ANDROID_KEYSTORE, realpathSync(resolve(androidFixture, 'configs/signing/upload.jks')));
    assert.equal(result.GH_TOKEN, undefined);
    assert.equal(result.APP_STORE_CONNECT_API_KEY_ID, undefined);
    const tool = await executeSigningTool(process.execPath, ['-e', 'process.stdout.write(process.env.GH_TOKEN || "isolated"); process.exit(31)'], { ...process.env, GH_TOKEN: 'private fixture' });
    assert.deepEqual(tool, { status: 31, stdout: 'isolated' });
    console.info('Android signing retains its registered identity and isolated process environment.');
} finally {
    rmSync(androidFixture, { recursive: true, force: true });
}

await import('./android-signing-source.mjs');
