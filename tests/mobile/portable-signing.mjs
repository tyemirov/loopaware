// @ts-check
import assert from 'node:assert/strict';
import { mkdtemp, mkdir, writeFile, readFile, readdir, rm } from 'node:fs/promises';
import { join } from 'node:path';
import { createHash } from 'node:crypto';
import { tmpdir } from 'node:os';
import { withAppleSigning } from '../../mobile/scripts/apple-signing.mjs';

const root = await mkdtemp(join(tmpdir(), 'portable signing '));
const certificatePassword = 'private fixture password';
const uuid = '4AF84C7F-4FA7-45EB-8A2F-8AF62C56BC74';
const certificateData = Buffer.from('certificate fixture').toString('base64');
const identity = createHash('sha1').update(Buffer.from(certificateData, 'base64')).digest('hex').toUpperCase();
try {
    for (const scenario of ['success', 'build failure', 'interruption', 'import failure', 'cleanup failure', 'wrong app', 'expired profile', 'missing certificate', 'outside repository', 'wrong identity']) {
        const repositoryRoot = join(root, scenario, 'relocated checkout');
        const userDirectory = join(root, scenario, 'host user');
        const signingDirectory = join(repositoryRoot, 'configs/signing');
        await mkdir(signingDirectory, { recursive: true });
        await writeFile(join(signingDirectory, 'distribution.p12'), 'private certificate fixture');
        await writeFile(join(signingDirectory, 'app.mobileprovision'), 'signed profile fixture');
        const environment = {
            GH_TOKEN: 'private GitHub fixture token',
            LOOPAWARE_APPLE_CERTIFICATE_PATH: join(signingDirectory, 'distribution.p12'),
            LOOPAWARE_APPLE_CERTIFICATE_PASSWORD: certificatePassword,
            LOOPAWARE_APPLE_PROFILE_PATH: join(signingDirectory, 'app.mobileprovision')
        };
        if (scenario === 'missing certificate') await rm(environment.LOOPAWARE_APPLE_CERTIFICATE_PATH);
        if (scenario === 'outside repository') {
            const outside = join(root, 'outside.p12');
            await writeFile(outside, 'outside secret');
            environment.LOOPAWARE_APPLE_CERTIFICATE_PATH = outside;
        }
        const profile = {
            UUID: uuid, Name: 'Portable Store Profile', TeamIdentifier: ['ABCDEFGHIJ'],
            ExpirationDate: scenario === 'expired profile' ? '2000-01-01T00:00:00Z' : '2099-01-01T00:00:00Z',
            Entitlements: { 'application-identifier': `ABCDEFGHIJ.${scenario === 'wrong app' ? 'wrong.application' : 'com.mprlab.loopaware'}`, 'get-task-allow': false },
            DeveloperCertificates: [certificateData]
        };
        const calls = [];
        let keychain;
        let built = false;
        const originalKeychains = ['/host/login.keychain-db', '/host/other.keychain-db'];
        let keychains = [...originalKeychains];
        const execute = async (name, args) => {
            calls.push([name, ...args]);
            if (name === 'security') {
                if (args[0] === 'list-keychains') {
                    if (args.includes('-s')) keychains = args.slice(args.indexOf('-s') + 1);
                    return { stdout: keychains.map(value => JSON.stringify(value)).join('\n'), status: 0 };
                }
                if (args[0] === 'cms') return { stdout: JSON.stringify(profile), status: 0 };
                if (args[0] === 'create-keychain') {
                    keychain = args.at(-1);
                    await writeFile(keychain, 'temporary keychain');
                }
                if (args[0] === 'import' && scenario === 'import failure') return { status: 31, stderr: certificatePassword };
                if (args[0] === 'find-identity') return { stdout: `1) ${scenario === 'wrong identity' ? 'B'.repeat(40) : identity} "Apple Distribution: Fixture (ABCDEFGHIJ)"\n1 valid identities found`, status: 0 };
                if (args[0] === 'delete-keychain') {
                    if (scenario === 'cleanup failure') return { status: 32, stderr: certificatePassword };
                    await rm(args.at(-1));
                }
            }
            if (name === 'plutil') {
                const key = args[1];
                const value = key ? key.split('.').reduce((value, component) => value[component], profile) : Object.keys(profile).join('\n');
                return { stdout: args[2] === 'json' ? JSON.stringify(value) : Array.isArray(value) ? String(value.length) : String(value), status: 0 };
            }
            return { status: 0, stdout: '' };
        };
        const controller = new AbortController();
        const build = async signing => {
            built = true;
            keychains.push('/host/concurrent.keychain-db');
            assert.equal(signing.environment.LOOPAWARE_APPLE_TEAM, 'ABCDEFGHIJ');
            assert.equal(signing.environment.LOOPAWARE_APPLE_PROFILE, uuid);
            assert.equal(signing.environment.LOOPAWARE_APPLE_IDENTITY, identity);
            assert.equal(signing.environment.LOOPAWARE_APPLE_CERTIFICATE_PASSWORD, undefined);
            assert.equal(signing.environment.GH_TOKEN, undefined);
            assert.equal(signing.keychainEnvironment, 'LOOPAWARE_APPLE_KEYCHAIN');
            assert.ok((await readFile(keychain, 'utf8')).includes('temporary'));
            assert.equal(await readFile(signing.profilePath, 'utf8'), 'signed profile fixture');
            if (scenario === 'interruption') controller.abort();
            if (scenario === 'build failure') throw new Error('gateway exit 47');
            return 47;
        };
        let failure;
        let result;
        try {
            result = await withAppleSigning({ repositoryRoot, userDirectory, applicationIdentifier: 'com.mprlab.loopaware', environment, execute, signal: controller.signal }, build);
        } catch (error) { failure = error; }
        if (scenario === 'success') {
            assert.equal(failure, undefined);
            assert.equal(result, 47);
        } else {
            assert.ok(failure, `${scenario} must fail`);
            assert.ok(!String(failure).includes(certificatePassword));
            const expectedErrors = {
                'build failure': /gateway exit 47/, 'interruption': /interrupted/,
                'import failure': /security import failed/, 'cleanup failure': /cleanup/,
                'wrong app': /unexpired App Store/, 'expired profile': /unexpired App Store/,
                'missing certificate': /must identify/, 'outside repository': /must identify/,
                'wrong identity': /exactly one valid identity/
            };
            assert.match(String(failure), expectedErrors[scenario]);
        }
        if (['wrong app', 'expired profile', 'missing certificate', 'outside repository'].includes(scenario)) {
            assert.equal(built, false);
            assert.ok(!calls.some(call => call[1] === 'create-keychain'));
        }
        if (scenario === 'wrong identity') assert.equal(built, false);
        if (keychain) {
            assert.ok(calls.some(call => call[1] === 'delete-keychain'), `${scenario} must clean the signing identity`);
            assert.deepEqual(keychains, [...originalKeychains, ...(built ? ['/host/concurrent.keychain-db'] : [])]);
            if (scenario !== 'cleanup failure') await assert.rejects(readFile(keychain), { code: 'ENOENT' });
        }
        const profileDirectory = join(userDirectory, 'Library/Developer/Xcode/UserData/Provisioning Profiles');
        assert.deepEqual(await readdir(profileDirectory).catch(error => error.code === 'ENOENT' ? [] : Promise.reject(error)), []);
        assert.equal(await readFile(environment.LOOPAWARE_APPLE_PROFILE_PATH, 'utf8'), 'signed profile fixture');
    }
    console.info('Portable signing validates private inputs and removes temporary state after success, failure, and interruption.');
} finally {
    await rm(root, { recursive: true, force: true });
}
