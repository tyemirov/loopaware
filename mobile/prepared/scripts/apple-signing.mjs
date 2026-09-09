// @ts-check
import { mkdtemp, mkdir, readFile, writeFile, copyFile, rm, realpath } from 'node:fs/promises';
import { join, relative, isAbsolute } from 'node:path';
import { homedir, tmpdir } from 'node:os';
import { createHash, randomBytes } from 'node:crypto';
import { spawn } from 'node:child_process';

const KEYCHAIN_ENVIRONMENT = 'LOOPAWARE_APPLE_KEYCHAIN';
const CERTIFICATE_PATH = 'LOOPAWARE_APPLE_CERTIFICATE_PATH';
const CERTIFICATE_PASSWORD = 'LOOPAWARE_APPLE_CERTIFICATE_PASSWORD';
const PROFILE_PATH = 'LOOPAWARE_APPLE_PROFILE_PATH';

/**
 * @typedef {{status: number | null, stdout: string}} SigningToolResult
 * @typedef {(name: string, args: string[], environment?: NodeJS.ProcessEnv) => Promise<SigningToolResult>} SigningTool
 * @typedef {{environment: NodeJS.ProcessEnv, keychainEnvironment: string, profilePath: string}} AppleSigningContext
 */

/** Execute a signing tool without putting passwords in process arguments or diagnostics.
 * @param {string} name
 * @param {string[]} args
 * @param {NodeJS.ProcessEnv} [sourceEnvironment]
 * @returns {Promise<SigningToolResult>}
 */
export async function executeSigningTool(name, args, sourceEnvironment = process.env) {
    /** @type {string | undefined} */
    let input;
    let commandArgs = args;
    if (name === 'security') {
        if (args.some(value => /[\r\n\0]/.test(value))) throw new Error('Apple signing input contains an unsupported control character.');
        input = args.map(value => '"' + value.replaceAll('\\', '\\\\').replaceAll('"', '\\"') + '"').join(' ') + '\n';
        commandArgs = ['-i'];
    }
    const environment = { ...sourceEnvironment };
    delete environment[CERTIFICATE_PASSWORD];
    delete environment.GH_TOKEN;
    delete environment.GITHUB_TOKEN;
    return new Promise((resolve, reject) => {
        const child = spawn(name, commandArgs, { env: environment, stdio: ['pipe', 'pipe', 'pipe'] });
        let stdout = '';
        child.stdout.on('data', chunk => { stdout += chunk; });
        child.stderr.resume();
        child.once('error', () => reject(new Error(`Apple signing tool could not start: ${name}`)));
        child.once('close', status => resolve({ status, stdout }));
        child.stdin.on('error', () => {}); // A rejected native command can close its input before EOF.
        child.stdin.end(input);
    });
}

/** Require a private input file inside the selected repository directory.
 * @param {string} repositoryRoot
 * @param {string | undefined} value
 * @param {string} name
 */
export async function repositorySigningFile(repositoryRoot, value, name) {
    if (!value) throw new Error(`Signing requires ${name} in configs/.env.loopaware.`);
    const root = await realpath(repositoryRoot);
    let path;
    try {
        path = await realpath(isAbsolute(value) ? value : join(root, value));
        const inside = relative(root, path);
        const privatePath = relative(join(root, 'configs/signing'), path);
        if (inside.startsWith('..') || isAbsolute(inside) || privatePath.startsWith('..') || isAbsolute(privatePath)) throw new Error('outside repository');
        const content = await readFile(path);
        if (!content.length) throw new Error('empty input');
    } catch {
        throw new Error(`Signing input ${name} must identify a nonempty file under configs/signing/.`);
    }
    return path;
}

/** Build with a private certificate and profile, then remove the temporary signing state.
 * @template Result
 * @param {{repositoryRoot: string, applicationIdentifier: string, environment?: NodeJS.ProcessEnv, userDirectory?: string, execute?: SigningTool, signal?: AbortSignal}} options
 * @param {(context: AppleSigningContext) => Promise<Result>} build
 * @returns {Promise<Result | undefined>}
 */
export async function withAppleSigning({ repositoryRoot, applicationIdentifier, environment = process.env, userDirectory = homedir(), execute = executeSigningTool, signal }, build) {
    const certificate = await repositorySigningFile(repositoryRoot, environment[CERTIFICATE_PATH], CERTIFICATE_PATH);
    const profileSource = await repositorySigningFile(repositoryRoot, environment[PROFILE_PATH], PROFILE_PATH);
    const password = environment[CERTIFICATE_PASSWORD];
    if (!password || /[\r\n\0]/.test(password)) throw new Error(`Apple signing requires a single-line ${CERTIFICATE_PASSWORD}.`);
    const temporary = await realpath(await mkdtemp(join(tmpdir(), 'loopaware-signing-')));
    const keychain = join(temporary, 'build.keychain-db');
    const keychainPassword = randomBytes(32).toString('hex');
    let keychainCreated = false;
    let searchListChanged = false;
    /** @type {string | undefined} */
    let profilePath;
    let failure;
    let result;
    /** @type {(name: string, args: string[]) => Promise<string>} */
    const command = async (name, args) => {
        const response = await execute(name, args, environment);
        if (response.status !== 0) throw new Error(`Apple signing ${name} ${args[0]} failed (status ${response.status}).`);
        return (response.stdout ?? '').trim();
    };
    /** @param {string[]} args */
    const security = (...args) => command('security', args);
    const checkInterrupted = () => {
        if (signal?.aborted) throw new Error('Apple signing build interrupted.');
    };
    /** @returns {Promise<string[]>} */
    const readKeychains = async () => {
        const text = await security('list-keychains', '-d', 'user');
        return text.split('\n').map(line => line.trim()).filter(Boolean).map(line => JSON.parse(line));
    };
    try {
        checkInterrupted();
        const decodedPath = join(temporary, 'profile.plist');
        const decoded = await security('cms', '-D', '-i', profileSource);
        await writeFile(decodedPath, decoded);
        /** @param {string} key @param {string} type */
        const extract = (key, type) => command('plutil', ['-extract', key, 'raw', '-expect', type, '-o', '-', decodedPath]);
        const uuid = await extract('UUID', 'string');
        const team = await extract('TeamIdentifier.0', 'string');
        const expiration = Date.parse(await extract('ExpirationDate', 'date'));
        const entitlementText = await command('plutil', ['-extract', 'Entitlements', 'json', '-o', '-', decodedPath]);
        const entitlements = JSON.parse(entitlementText);
        const deviceProfile = /<key>\s*(?:ProvisionedDevices|ProvisionsAllDevices)\s*<\/key>/.test(decoded);
        if (!/^[0-9A-F-]{36}$/i.test(uuid) || !/^[A-Z0-9]{10}$/.test(team) || !Number.isFinite(expiration) || expiration <= Date.now() ||
            entitlements['application-identifier'] !== `${team}.${applicationIdentifier}` || entitlements['get-task-allow'] !== false ||
            deviceProfile) {
            throw new Error('Apple provisioning profile must be an unexpired App Store profile for the selected application.');
        }
        const certificateCount = Number(await extract('DeveloperCertificates', 'array'));
        if (!Number.isInteger(certificateCount) || certificateCount < 1 || certificateCount > 20) throw new Error('Apple profile certificate list is invalid.');
        /** @type {string[]} */
        const fingerprints = [];
        for (let index = 0; index < certificateCount; index += 1) {
            const data = await extract(`DeveloperCertificates.${index}`, 'data');
            fingerprints.push(createHash('sha1').update(Buffer.from(data, 'base64')).digest('hex').toUpperCase());
        }
        checkInterrupted();
        // Register cleanup before creation: a native failure can follow a partial mutation.
        keychainCreated = true;
        await security('create-keychain', '-p', keychainPassword, keychain);
        await security('set-keychain-settings', keychain);
        await security('unlock-keychain', '-p', keychainPassword, keychain);
        await security('import', certificate, '-k', keychain, '-f', 'pkcs12', '-P', password, '-T', '/usr/bin/codesign');
        await security('set-key-partition-list', '-S', 'apple-tool:,apple:,codesign:', '-s', '-k', keychainPassword, keychain);
        const identities = await security('find-identity', '-v', '-p', 'codesigning', keychain);
        const matches = [...identities.matchAll(/\b([A-F0-9]{40})\b/g)].map(match => match[1]).filter(value => fingerprints.includes(value));
        if (matches.length !== 1) throw new Error('Apple certificate archive must contain exactly one valid identity from the selected profile.');
        const keychains = await readKeychains();
        searchListChanged = true;
        await security('list-keychains', '-d', 'user', '-s', ...keychains, keychain);
        const profiles = join(userDirectory, 'Library/Developer/Xcode/UserData/Provisioning Profiles');
        await mkdir(profiles, { recursive: true });
        profilePath = join(profiles, `${uuid}-${randomBytes(12).toString('hex')}.mobileprovision`);
        await copyFile(profileSource, profilePath);
        const buildEnvironment = { ...environment };
        for (const name of [CERTIFICATE_PATH, CERTIFICATE_PASSWORD, PROFILE_PATH, 'GH_TOKEN', 'GITHUB_TOKEN', 'NPM_API_KEY', 'NPM_TOKEN', 'APP_STORE_CONNECT_API_KEY_PATH', 'APP_STORE_CONNECT_API_KEY_ID', 'APP_STORE_CONNECT_API_ISSUER_ID', 'GOOGLE_PLAY_SERVICE_ACCOUNT_KEY_PATH', 'LOOPAWARE_ANDROID_KEYSTORE', 'LOOPAWARE_ANDROID_STORE_PASSWORD', 'LOOPAWARE_ANDROID_KEY_ALIAS', 'LOOPAWARE_ANDROID_KEY_PASSWORD']) delete buildEnvironment[name];
        buildEnvironment.LOOPAWARE_APPLE_TEAM = team;
        buildEnvironment.LOOPAWARE_APPLE_PROFILE = uuid;
        buildEnvironment.LOOPAWARE_APPLE_IDENTITY = matches[0];
        // Xcode treats OTHER_CODE_SIGN_FLAGS as shell words, including paths with spaces.
        buildEnvironment[KEYCHAIN_ENVIRONMENT] = '"' + keychain.replace(/[\\"$`]/g, '\\$&') + '"';
        checkInterrupted();
        result = await build({ environment: buildEnvironment, keychainEnvironment: KEYCHAIN_ENVIRONMENT, profilePath });
        checkInterrupted();
    } catch (error) {
        failure = error;
    }
    const cleanupErrors = [];
    for (const cleanup of [
        async () => { if (profilePath) await rm(profilePath, { force: true }); },
        async () => {
            if (searchListChanged) {
                const current = await readKeychains();
                await security('list-keychains', '-d', 'user', '-s', ...current.filter(path => path !== keychain));
            }
        },
        async () => { if (keychainCreated) await security('delete-keychain', keychain); }
    ]) {
        try { await cleanup(); } catch (error) { cleanupErrors.push(error); }
    }
    if (!cleanupErrors.length) await rm(temporary, { recursive: true, force: true });
    if (cleanupErrors.length) throw new AggregateError([...(failure ? [failure] : []), ...cleanupErrors], `Apple signing cleanup failed; temporary state: ${temporary}`);
    if (failure) throw failure;
    return result;
}
