// @ts-check
import assert from 'node:assert/strict';
import fs from 'node:fs';
import { mkdir, mkdtemp, writeFile, symlink, rm } from 'node:fs/promises';
import { createRequire } from 'node:module';
import { spawnSync } from 'node:child_process';
import { dirname, join, resolve, isAbsolute } from 'node:path';
import { pathToFileURL } from 'node:url';

const repository = resolve(import.meta.dirname, '../..');
const mobile = join(repository, 'mobile');
const prepared = join(mobile, 'prepared');
const expectedVersion = '14.0.2';
/** @type {NodeJS.ProcessEnv} */
const environment = { PATH: process.env.PATH, HOME: process.env.HOME, CI: '1', NODE_ENV: 'production', EXPO_NO_DOTENV: '1', EXPO_NO_TELEMETRY: '1' };
/** @param {string} root @param {string} program @param {string[]} args */
function command(root, program, args) {
  const result = spawnSync(program, args, { cwd: root, env: environment, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] });
  if (result.error) throw result.error;
  assert.equal(result.status, 0, result.stdout + result.stderr);
  return result.stdout;
}
/** @param {string} entry */
function packageMetadata(entry) {
  let directory = dirname(entry);
  while (true) {
    const path = join(directory, 'package.json');
    if (fs.existsSync(path)) {
      const data = JSON.parse(fs.readFileSync(path, 'utf8'));
      if (typeof data.name === 'string' && typeof data.version === 'string') return { root: directory, data };
    }
    const parent = dirname(directory); assert.notEqual(parent, directory, 'Cannot locate owning package metadata'); directory = parent;
  }
}
/** @param {string} root */
function consumers(root) {
  const require = createRequire(join(root, 'package.json'));
  const expo = createRequire(require.resolve('expo/package.json'));
  const entries = ['@expo/config', '@expo/config-plugins', '@expo/fingerprint', '@expo/metro-config'].map(name => [name, require.resolve(name)]);
  entries.push(['@expo/cli', expo.resolve('@expo/cli')]);
  return entries.map(([name, entry]) => {
    const fromConsumer = createRequire(entry);
    const selected = packageMetadata(fromConsumer.resolve('glob'));
    assert.equal(selected.data.version, expectedVersion, `${root}: ${name} must resolve patched glob`);
    return { name, require: createRequire(join(selected.root, 'package.json')), entry: selected.root };
  });
}
/** @param {string} entry @param {string} root */
function engines(entry, root) {
  const semver = createRequire(join(root, 'package.json'))('semver');
  const visited = new Set();
  /** @param {string} entry */
  function walk(entry) {
    const metadata = packageMetadata(entry);
    if (visited.has(metadata.root)) return;
    visited.add(metadata.root);
    const requirement = metadata.data.engines?.node;
    if (requirement) assert.ok(semver.satisfies(process.versions.node, requirement), `${metadata.data.name}@${metadata.data.version}: Node ${requirement}`);
    const require = createRequire(join(metadata.root, 'package.json'));
    for (const name of Object.keys(metadata.data.dependencies ?? {})) walk(require.resolve(name));
  }
  walk(entry);
  return visited.size;
}
/** @param {string} fixture */
async function corpus(fixture) {
  const files = ['ios/App.xcodeproj/project.pbxproj', 'android/app/MainActivity.java', 'android/lib/MainActivity.kt',
    'node_modules/hidden/MainActivity.java', 'node_modules/hidden/package.json', 'node_modules/hidden/.gitignore',
    'packages/one/package.json', 'packages/two/package.json', 'packages/two/.gitignore',
    'packages/two/node_modules/nested/package.json', 'Carthage/library/package.json', 'Pods/library/package.json',
    'docs/.gitignore', 'docs/deep/.gitignore', 'package.json'];
  for (const name of files) { await mkdir(dirname(join(fixture, name)), { recursive: true }); await writeFile(join(fixture, name), '{}\n'); }
  await symlink(join(fixture, 'packages'), join(fixture, 'linked-packages'), 'dir');
}
/** @param {any} glob @param {string} fixture */
async function patterns(glob, fixture) {
  const cases = [
    { pattern: '**/*.xcodeproj', options: {}, expected: ['ios/App.xcodeproj'] },
    { pattern: '**/MainActivity.@(java|kt)', options: { ignore: '**/node_modules/**' }, expected: ['android/app/MainActivity.java', 'android/lib/MainActivity.kt'] },
    { pattern: '**/.gitignore', options: { ignore: '**/node_modules/**', follow: false }, expected: ['docs/.gitignore', 'docs/deep/.gitignore', 'packages/two/.gitignore'] },
    { pattern: '**/package.json', options: { ignore: '**/@(Carthage|Pods|node_modules)/**', follow: false }, expected: ['package.json', 'packages/one/package.json', 'packages/two/package.json'] },
    { pattern: ['**/*.xcodeproj', '**/*.xcodeproj'], options: { absolute: true, follow: false }, expected: [join(fixture, 'ios/App.xcodeproj')] },
  ];
  for (const scenario of cases) {
    const options = { cwd: fixture, ...scenario.options };
    const expected = scenario.expected.toSorted();
    assert.deepEqual(glob.globSync(scenario.pattern, options).toSorted(), expected);
    assert.deepEqual((await glob.glob(scenario.pattern, options)).toSorted(), expected);
    const streamed = [];
    for await (const name of glob.globStream(scenario.pattern, options)) streamed.push(name);
    assert.deepEqual(streamed.toSorted(), expected);
  }
}
/** @param {any} glob @param {string} fixture */
async function abortAdmissions(glob, fixture) {
  const controller = new AbortController();
  /** @type {{operation:string,name:string,afterAbort:boolean}[]} */
  const admissions = [];
  let inFlight = 0;
  let completedAfterAbort = 0;
  /** @type {() => void} */
  let releaseCompletion;
  /** @type {Promise<void>} */
  const firstCompletion = new Promise(resolveCompletion => { releaseCompletion = resolveCompletion; });
  /** @param {string} operation @param {string} name */
  function admit(operation, name) { admissions.push({ operation, name, afterAbort: controller.signal.aborted }); }
  /** @type {import('path-scurry').FSOption} */
  const native = {
    lstatSync(name) { admit('lstatSync', name); return fs.lstatSync(name); },
    readdirSync(name, options) { admit('readdirSync', name); return fs.readdirSync(name, options); },
    readlinkSync(name) { admit('readlinkSync', name); return fs.readlinkSync(name); },
    realpathSync(name) { admit('realpathSync', name); return fs.realpathSync(name); },
    readdir(name, options, callback) {
      admit('readdir', name); inFlight++;
      fs.readdir(name, options, (error, entries) => {
        controller.abort(new Error('controlled glob cancellation'));
        completedAfterAbort++; inFlight--;
        callback(error, entries); releaseCompletion();
      });
    },
    promises: {
      async lstat(name) {
        admit('promises.lstat', name); inFlight++;
        try { return await fs.promises.lstat(name); } finally { inFlight--; }
      },
      async readlink(name) {
        admit('promises.readlink', name); inFlight++;
        try { return await fs.promises.readlink(name); } finally { inFlight--; }
      },
      async realpath(name) {
        admit('promises.realpath', name); inFlight++;
        try { return await fs.promises.realpath(name); } finally { inFlight--; }
      },
      async readdir(name, options) {
        admit('promises.readdir', name); inFlight++;
        try {
          const result = await fs.promises.readdir(name, options);
          controller.abort(new Error('controlled glob cancellation')); completedAfterAbort++;
          return result;
        } finally { inFlight--; releaseCompletion(); }
      },
    },
  };
  await assert.rejects(glob.glob('**/*', { cwd: fixture, signal: controller.signal, fs: native }), /controlled glob cancellation/);
  await firstCompletion;
  await new Promise(done => setImmediate(done));
  assert.ok(completedAfterAbort > 0, 'An already-admitted native filesystem request must complete after abort');
  assert.equal(inFlight, 0, 'Every admitted native request must finish');
  assert.deepEqual(admissions.filter(entry => entry.afterAbort), [], 'Abort must prevent new filesystem admissions, not cancel already-admitted completion');
}
/** @param {ReturnType<typeof createRequire>} require @param {string} fixture */
async function fingerprintConsumers(require, fixture) {
  const fingerprintModule = require('@expo/fingerprint');
  const fromFingerprint = createRequire(require.resolve('@expo/fingerprint'));
  const entry = fromFingerprint.resolve('minimatch');
  assert.equal(packageMetadata(entry).data.version, '11.0.4');
  engines(entry, dirname(require.resolve('expo/package.json')));
  const { Minimatch } = fromFingerprint('minimatch');
  const negative = new Minimatch('!inputs/skip/keep.txt', { dot: true });
  assert.equal(negative.pattern, 'inputs/skip/keep.txt'); assert.equal(negative.negate, true);
  assert.equal(negative.match('inputs/skip/keep.txt'), false);
  assert.equal(negative.match('inputs/skip/drop.txt'), true);
  for (const [pattern, filename] of [
    ['**/*', 'inputs/.hidden'], ['**/shared/**', '../shared/public.txt'.replace(/^(\.\.\/)+/, '')],
    ['inputs/braces/*.{js,ts}', 'inputs/braces/a.ts'], ['inputs/ext/*.@(java|kt)', 'inputs/ext/a.kt'],
    ['inputs/ext/*.@(java|kt)', String.raw`inputs\ext\a.kt`.replaceAll('\\', '/')],
    ['inputs/pruned/**', 'inputs/pruned/deep/file.txt'], ['inputs/pruned/**/*', 'inputs/pruned/deep/file.txt'],
  ]) {
    const matcher = new Minimatch(pattern, { dot: true });
    assert.equal(matcher.pattern, pattern); assert.equal(matcher.negate, false); assert.equal(matcher.match(filename), true);
  }
  const probeRoot = join(fixture, 'fingerprint-' + fingerprintSequence++);
  const project = join(probeRoot, 'application');
  try {
    await mkdir(project, { recursive: true });
    await writeFile(join(project, 'package.json'), JSON.stringify({ name: 'fingerprint-public-fixture', version: '1.0.0', private: true, dependencies: {} }));
    await symlink(join(dirname(require.resolve('expo/package.json')), '..'), join(project, 'node_modules'), 'dir');
    const files = ['inputs/.hidden', 'inputs/skip/drop.txt', 'inputs/skip/keep.txt', 'inputs/plain.txt',
      'inputs/pruned/deep/file.txt', 'inputs/pruned-star/deep/file.txt', 'inputs/braces/a.ts', 'inputs/ext/a.kt', '../shared/public.txt'];
    for (const name of files) { await mkdir(dirname(join(project, name)), { recursive: true }); await writeFile(join(project, name), 'public input\n'); }
    /** @param {string} filePath @returns {import('@expo/fingerprint').HashSourceFile} */
    function fileSource(filePath) { return { type: 'file', filePath, reasons: ['public minimatch compatibility'] }; }
    /** @type {import('@expo/fingerprint').Options} */
    const options = { platforms: [], ignorePaths: ['inputs/.hidden', 'inputs/skip/**', '!inputs/skip/keep.txt',
      'inputs/pruned/**', 'inputs/pruned-star/**/*', 'inputs/braces/*.{js,ts}', 'inputs/ext/*.@(java|kt)', '**/shared/**'],
      extraSources: [...files.map(fileSource),
        { type: 'dir', filePath: 'inputs/pruned', reasons: ['public directory pruning'] },
        { type: 'dir', filePath: 'inputs/pruned-star', reasons: ['public directory pruning'] }] };
    const probe = () => fingerprintModule.createFingerprintAsync(project, options);
    const initial = await probe(), repeated = await probe();
    assert.equal(initial.hash, repeated.hash, 'Public fingerprint must be deterministic for identical current inputs');
    for (const name of files) {
      const included = initial.sources.some(source => source.type !== 'contents' && source.filePath === name && source.hash !== null);
      assert.equal(included, ['inputs/skip/keep.txt', 'inputs/plain.txt'].includes(name), `Public fingerprint inclusion differs: ${name}`);
    }
    for (const name of ['inputs/pruned', 'inputs/pruned-star']) {
      assert.equal(initial.sources.some(source => source.type !== 'contents' && source.filePath === name && source.hash !== null), false, 'Public fingerprint must prune ignored directories');
    }
    await writeFile(join(project, 'inputs/skip/drop.txt'), 'ignored change\n');
    assert.equal((await probe()).hash, initial.hash, 'Ignored input must not alter public fingerprint output');
    await writeFile(join(project, 'inputs/skip/keep.txt'), 'included change\n');
    assert.notEqual((await probe()).hash, initial.hash, 'Reincluded input must alter public fingerprint output');
  } finally {
    await rm(probeRoot, { recursive: true, force: true });
    assert.equal(fs.existsSync(probeRoot), false);
  }
}
/** @param {string} root */
function reactNativeConfig(root) {
  const require = createRequire(join(root, 'package.json'));
  for (const name of ['@react-native-community/cli-config', '@react-native-community/cli-types']) {
    const fromConsumer = createRequire(require.resolve(name));
    const entry = fromConsumer.resolve('joi');
    assert.equal(packageMetadata(entry).data.version, '17.13.9');
    engines(entry, root);
    const Joi = fromConsumer('joi');
    assert.equal(Joi.string().email({ multiple: true }).validate('first@example.com,second@example.com').error, undefined);
    assert.ok(Joi.string().email({ multiple: true }).validate('first@example.com,invalid').error);
  }
  const metadata = packageMetadata(require.resolve('@react-native-community/cli'));
  const binary = join(metadata.root, metadata.data.bin['rnc-cli']);
  const config = JSON.parse(command(root, process.execPath, [binary, 'config']));
  assert.equal(fs.realpathSync(config.root), fs.realpathSync(root));
  assert.ok(config.reactNativePath.endsWith('react-native'));
  assert.ok(config.dependencies && typeof config.dependencies === 'object');
}
let fingerprintSequence = 0;

/** @param {string} root @param {string} fixture */
async function qualify(root, fixture) {
  const selected = consumers(root);
  const closures = [];
  const qualified = new Set();
  for (const consumer of selected) {
    if (qualified.has(consumer.entry)) continue;
    qualified.add(consumer.entry);
    const glob = consumer.require('glob');
    closures.push(engines(consumer.require.resolve('glob'), root));
    await patterns(glob, fixture);
    await abortAdmissions(glob, fixture);
  }
  const require = createRequire(join(root, 'package.json'));
  const expo = createRequire(require.resolve('expo/package.json'));
  const cli = expo.resolve('@expo/cli');
  const config = JSON.parse(command(root, process.execPath, [cli, 'config', '--type', 'introspect', '--json']));
  assert.equal(config.ios.bundleIdentifier, 'com.mprlab.loopaware');
  assert.equal(config.android.package, 'com.mprlab.loopaware');
  reactNativeConfig(root);
  await fingerprintConsumers(require, fixture);
  const fingerprint = await require('@expo/fingerprint').createFingerprintAsync(root);
  assert.ok(fingerprint.sources.length > 0, 'Public fingerprint must inspect actual project sources');
  command(root, 'npm', ['ls', 'glob', '--all', '--json']);
  process.stdout.write(JSON.stringify({ check: 'glob-consumer-compatibility', root, version: expectedVersion, consumers: selected.map(item => item.name), engineClosurePackages: closures, fingerprintSources: fingerprint.sources.length }) + '\n');
  process.stdout.write(command(root, process.execPath, [join(root, 'scripts/audit-dependencies.mjs')]));
}

await mkdir(join(repository, '_scratch'), { recursive: true });
const fixture = await mkdtemp(join(repository, '_scratch/v8-glob-compatibility-'));
let staged;
try {
  consumers(mobile); consumers(prepared);
  await corpus(fixture);
  await qualify(mobile, fixture);
  await qualify(prepared, fixture);
  const { stageStoreSource } = await import(pathToFileURL(join(mobile, 'scripts/record-store-preparation.mjs')).href);
  staged = stageStoreSource(repository, prepared);
  assert.ok(isAbsolute(staged));
  command(staged, 'npm', ['ci', '--include=dev', '--ignore-scripts', '--no-audit', '--no-fund']);
  await qualify(staged, fixture);
} finally {
  if (staged) { await rm(staged, { recursive: true, force: true }); assert.equal(fs.existsSync(staged), false); }
  await rm(fixture, { recursive: true, force: true });
  assert.equal(fs.existsSync(fixture), false);
}
process.stdout.write('glob_compatibility.cleanup.ok: native children joined; owned corpus and canonical stage removed\n');
