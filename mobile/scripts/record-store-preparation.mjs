// @ts-check
import assert from 'node:assert/strict';
import { copyFileSync, existsSync, lstatSync, mkdirSync, mkdtempSync, readFileSync, readdirSync, realpathSync, rmSync, writeFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { dirname, isAbsolute, join, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { tmpdir } from 'node:os';

const remedy = 'run make mobile-prepare-store';
const excludedDirectories = new Set(['node_modules', 'Pods', 'build', 'DerivedData', '.gradle', '.kotlin', '.cxx', '.expo', 'xcuserdata', '.git']);
const excludedFiles = new Set(['.DS_Store', 'local.properties', '.xcode.env.local']);
const rootMetadata = new Set(['package.json', 'package-lock.json', 'app.config.snapshot.json', 'native-preparation.json', 'source-preparation.json']);

/** @param {string} root @param {string} name */
function required(root, name) {
  assert.ok(!isAbsolute(name) && !name.split(/[\\/]/).includes('..'), 'invalid native preparation path: ' + name);
  let current = resolve(root);
  const parts = name.split('/');
  for (const [index, part] of parts.entries()) {
    current = join(current, part);
    let entry;
    try { entry = lstatSync(current); } catch { throw new Error('missing native input: ' + name); }
    assert.ok(!entry.isSymbolicLink(), 'native input has a symbolic link component: ' + name);
    assert.ok(index < parts.length - 1 ? entry.isDirectory() : entry.isFile(), 'missing native input: ' + name);
  }
  return current;
}
/** @param {string} root @param {string} name */
function text(root, name) { return readFileSync(required(root, name), 'utf8'); }
/** @param {string} root @param {string} name */
function json(root, name) { return JSON.parse(text(root, name)); }
/** @param {string} value */
function escaped(value) { return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'); }
/** @param {string} source @param {string} name */
function field(source, name) { return source.match(new RegExp(`(?:^|[;\\n])\\s*${escaped(name)}\\s*=\\s*([^;\\n]+);`))?.[1].trim().replace(/^"|"$/g, ''); }
/** @param {string} source @param {string} name */
function block(source, name) {
  const match = new RegExp(`\\b${escaped(name)}\\s*\\{`).exec(source);
  assert.ok(match, 'native configuration block missing: ' + name);
  let depth = 1;
  for (let index = match.index + match[0].length; index < source.length; index++) {
    if (source[index] === '{') depth++;
    if (source[index] === '}' && --depth === 0) return source.slice(match.index + match[0].length, index);
  }
  throw new Error('incomplete native configuration block: ' + name);
}
/** @param {string} source @param {string|undefined} id */
function object(source, id) {
  assert.ok(id, 'native project object reference is missing');
  const values = [];
  for (const match of source.matchAll(new RegExp(`^\\s*${escaped(id)}\\s*=\\s*\\{`, 'gm'))) {
    let depth = 1;
    for (let index = match.index + match[0].length; index < source.length; index++) {
      if (source[index] === '{') depth++;
      if (source[index] === '}' && --depth === 0) {
        const body = source.slice(match.index + match[0].length, index);
        if (field(body, 'isa')) values.push(body);
        break;
      }
    }
  }
  assert.equal(values.length, 1, 'native project object is missing or ambiguous: ' + id);
  return values[0];
}
/** @param {string} source @param {string} name */
function attribute(source, name) { return source.match(new RegExp(`\\b${escaped(name)}\\s*=\\s*"([^"]*)"`))?.[1]; }
/** @param {string} source @param {string} name */
function plistString(source, name) {
  const matches = [...source.matchAll(new RegExp(`<key>${escaped(name)}</key>\\s*<string>([^<]*)</string>`, 'g'))];
  assert.equal(matches.length, 1, 'native Info.plist field missing or ambiguous: ' + name);
  return matches[0][1];
}
/** @param {string} source @param {string} name */
function plistBoolean(source, name) {
  const keys = [...source.matchAll(new RegExp(`<key>${escaped(name)}</key>`, 'g'))];
  assert.equal(keys.length, 1, 'native Info.plist field missing or ambiguous: ' + name);
  const value = source.match(new RegExp(`<key>${escaped(name)}</key>\\s*<(true|false)\\s*/>`));
  assert.ok(value, 'native Info.plist field must be Boolean: ' + name);
  return value[1] === 'true';
}
/** @typedef {{name:string, ios:{bundleIdentifier:string, version:string, scheme?:string[], config:{usesNonExemptEncryption:boolean}}, android:{package:string}, version:string, scheme:string|string[]}} ApplicationConfig */
/** @typedef {{declared:string, resolved:string}} DependencyVersion */
/** @typedef {{dependencies:Record<string,DependencyVersion>, devDependencies:Record<string,DependencyVersion>, overrides:Record<string,unknown>}} DependencyFacts */
/** @param {ApplicationConfig} config */
function application(config) {
  assert.equal(config.ios.config?.usesNonExemptEncryption, false, 'native iOS source encryption declaration must be false');
  return { owner: 'loopaware', name: config.name, iosIdentifier: config.ios.bundleIdentifier,
    androidIdentifier: config.android.package, iosVersion: config.ios.version, androidVersion: config.version,
    schemes: [...new Set([...(Array.isArray(config.scheme) ? config.scheme : [config.scheme]), ...(config.ios.scheme ?? [])])].sort(),
    workspace: 'ios/LoopAware.xcworkspace', scheme: 'LoopAware', entryFile: 'index.ts' };
}
/** @param {string} source @param {string} name @param {RegExp} expected @param {string} [separator] */
function assignment(source, name, expected, separator = '') {
  const values = [...source.matchAll(new RegExp(`^\\s*${escaped(name)}\\s+${separator ? escaped(separator) + '\\s*' : ''}([^\\n]+)$`, 'gm'))];
  assert.equal(values.length, 1, 'native assignment missing or ambiguous: ' + name);
  assert.match(values[0][1].split('//')[0].trim(), expected, 'native assignment differs: ' + name);
}
/** Direct package versions are semantic dependency facts, not a source-file inventory.
 * @param {string} root
 */
function dependencies(root) {
  const metadata = json(root, 'package.json'), lock = json(root, 'package-lock.json');
  /** @type {DependencyFacts} */
  const result = {dependencies: {}, devDependencies: {}, overrides: metadata.overrides ?? {}};
  for (const section of /** @type {const} */ (['dependencies', 'devDependencies'])) {
    result[section] = {};
    for (const [name, declared] of Object.entries(metadata[section] ?? {})) {
      const entry = lock.packages?.['node_modules/' + name];
      const version = entry?.link ? lock.packages?.[entry.resolved]?.version : entry?.version;
      assert.ok(typeof version === 'string' && version, 'native direct dependency version missing: ' + name);
      result[section][name] = { declared, resolved: version };
    }
  }
  return result;
}

/** Inspect only selected native application/configuration/dependency facts.
 * @param {string} prepared
 */
export function preparationFacts(prepared) {
  const config = json(prepared, 'app.config.snapshot.json').expo;
  const selected = application(config);
  assert.equal(selected.name, 'LoopAware', 'selected native application name differs');
  assert.equal(selected.iosIdentifier, 'com.mprlab.loopaware', 'selected native iOS identifier differs');
  assert.equal(selected.androidIdentifier, 'com.mprlab.loopaware', 'selected native Android identifier differs');
  const metadata = json(prepared, 'package.json');
  assert.equal(metadata.main, selected.entryFile, 'selected native entry file differs');
  assert.equal(Object.hasOwn(metadata.dependencies, 'expo-dev-client'), false, 'production native dependencies must exclude expo-dev-client');
  const project = text(prepared, 'ios/LoopAware.xcodeproj/project.pbxproj').replace(/\/\*[\s\S]*?\*\//g, '').replace(/^\s*\/\/[^\n]*$/gm, '');
  const ids = [...new Set([...project.matchAll(/^\s*([A-F0-9]{24})\s*=\s*\{/gm)].map(match => match[1]))];
  const targets = ids.map(id => ({ id, body: object(project, id) })).filter(item => field(item.body, 'isa') === 'PBXNativeTarget' && field(item.body, 'name') === 'LoopAware');
  assert.equal(targets.length, 1, 'selected native iOS target missing or ambiguous');
  const target = targets[0];
  assert.equal(field(target.body, 'productType'), 'com.apple.product-type.application', 'selected native iOS target must be an application');
  assert.equal(field(object(project, field(target.body, 'productReference')), 'path'), 'LoopAware.app', 'selected native product differs');
  const list = object(project, field(target.body, 'buildConfigurationList'));
  assert.equal(field(list, 'isa'), 'XCConfigurationList', 'selected native configuration list differs');
  const references = list.match(/\bbuildConfigurations\s*=\s*\(([\s\S]*?)\);/)?.[1].split(',').map(value => value.trim()).filter(Boolean);
  assert.ok(references?.length, 'selected native target configurations missing');
  const configurations = references.map(id => object(project, id));
  const releases = configurations.filter(value => field(value, 'name') === 'Release');
  assert.equal(releases.length, 1, 'selected native Release configuration missing or ambiguous');
  const settings = block(releases[0], 'buildSettings =');
  for (const [name, expected] of Object.entries({ PRODUCT_BUNDLE_IDENTIFIER: selected.iosIdentifier, PRODUCT_NAME: 'LoopAware',
    INFOPLIST_FILE: 'LoopAware/Info.plist', MARKETING_VERSION: selected.iosVersion, CODE_SIGN_STYLE: 'Automatic' })) {
    assert.equal(field(settings, name), expected, 'selected native iOS configuration differs: ' + name);
  }
  assert.ok(/^\d+$/.test(field(settings, 'CURRENT_PROJECT_VERSION') ?? ''), 'native iOS source build number is invalid');
  const workspace = text(prepared, selected.workspace + '/contents.xcworkspacedata').replace(/<!--[\s\S]*?-->/g, '');
  assert.ok([...workspace.matchAll(/<FileRef\b([^>]*)>/g)].some(match => attribute(match[1], 'location') === 'group:LoopAware.xcodeproj'), 'selected native workspace project differs');
  const scheme = text(prepared, 'ios/LoopAware.xcodeproj/xcshareddata/xcschemes/LoopAware.xcscheme').replace(/<!--[\s\S]*?-->/g, '');
  const buildables = [...scheme.matchAll(/<BuildableReference\b([^>]*)>/g)]; assert.ok(buildables.length, 'selected native scheme target missing');
  for (const [, attrs] of buildables) {
    for (const [key, expected] of Object.entries({ BlueprintIdentifier: target.id, BlueprintName: 'LoopAware', BuildableName: 'LoopAware.app', ReferencedContainer: 'container:LoopAware.xcodeproj' })) {
      assert.equal(attribute(attrs, key), expected, 'selected native scheme binding differs: ' + key);
    }
  }
  assert.equal(attribute(scheme.match(/<ArchiveAction\b([^>]*)>/)?.[1] ?? '', 'buildConfiguration'), 'Release', 'native archive must select Release');
  const plist = text(prepared, 'ios/LoopAware/Info.plist').replace(/<!--[\s\S]*?-->/g, '');
  const iosUsesNonExemptEncryption = plistBoolean(plist, 'ITSAppUsesNonExemptEncryption');
  assert.equal(iosUsesNonExemptEncryption, false, 'native iOS encryption declaration must be false');
  for (const [name, expected] of Object.entries({ CFBundleIdentifier: '$(PRODUCT_BUNDLE_IDENTIFIER)', CFBundleShortVersionString: '$(MARKETING_VERSION)', CFBundleVersion: '$(CURRENT_PROJECT_VERSION)' })) {
    assert.equal(plistString(plist, name), expected, 'native plist setting differs: ' + name);
  }
  for (const name of selected.schemes) assert.ok(plist.includes('<string>' + name + '</string>'), 'native URL scheme differs: ' + name);
  const phaseReferences = target.body.match(/\bbuildPhases\s*=\s*\(([\s\S]*?)\);/)?.[1].split(',').map(value => value.trim()).filter(Boolean) ?? [];
  const phases = phaseReferences.map(id => object(project, id)).filter(value => field(value, 'isa') === 'PBXShellScriptBuildPhase' && field(value, 'name') === 'Bundle React Native code and images');
  assert.equal(phases.length, 1, 'selected native bundle phase differs');
  const script = JSON.parse(phases[0].match(/\bshellScript\s*=\s*("(?:\\.|[^"\\])*")\s*;/)?.[1] ?? 'null')?.replace(/^[ \t]*#.*$/gm, '');
  assert.ok(script, 'native bundle script missing');
  assert.match(script, /react-native\/scripts\/react-native-xcode\.sh/, 'native iOS must use the React Native bundler');
  assert.doesNotMatch(script, /@expo\/cli|export:embed/, 'native iOS must not use Expo CLI export');
  const properties = json(prepared, 'ios/Podfile.properties.json');
  assert.equal(properties['ios.buildReactNativeFromSource'], 'true', 'native iOS source dependency selection differs');
  assert.equal(properties.EXPO_USE_PRECOMPILED_MODULES, 'false', 'native iOS precompiled module selection differs');
  required(prepared, 'ios/Podfile');
  const lock = text(prepared, 'ios/Podfile.lock');
  const podSection = lock.match(/^PODS:\s*\n([\s\S]*?)(?=^[A-Z][A-Z _]+:|$(?![\s\S]))/m)?.[1];
  assert.ok(podSection, 'native PODS versions missing');
  const pods = Object.fromEntries([...podSection.matchAll(/^  - ([^\n]+?) \(([^()]+)\)(?::)?$/gm)].map(match => [match[1], match[2]]));
  assert.ok(Object.keys(pods).length, 'native PODS versions missing');
  const deps = dependencies(prepared);
  assert.equal(pods.Expo, deps.dependencies.expo.resolved, 'native Expo pod version differs');
  assert.equal(pods['React-Core'], deps.dependencies['react-native'].resolved, 'native React-Core pod version differs');
  const android = text(prepared, 'android/app/build.gradle').replace(/\/\*[\s\S]*?\*\//g, '').replace(/^\s*\/\/[^\n]*$/gm, '');
  assignment(android, 'namespace', /^['"]com\.mprlab\.loopaware['"]$/);
  assignment(android, 'applicationId', /^['"]com\.mprlab\.loopaware['"]$/);
  assignment(android, 'versionCode', /^System\.getenv\("MPRLAB_MOBILE_VERSION_CODE"\)\.toInteger\(\)$/, '=');
  assignment(android, 'versionName', new RegExp('^System\\.getenv\\("MPRLAB_MOBILE_VERSION_NAME"\\) \\?: "' + escaped(selected.androidVersion) + '"$'));
  assignment(android, 'bundleCommand', /^"bundle"$/, '=');
  assignment(android, 'cliFile', /^new File\(\["node", "--print", "require\('@react-native-community\/cli'\)\.bin"\]\.execute\(null, rootDir\)\.text\.trim\(\)\)$/, '=');
  assignment(android, 'entryFile', /^file\(\["node", "-e", "require\('expo\/scripts\/resolveAppEntry'\)", projectRoot, "android", "absolute"\]\.execute\(null, rootDir\)\.text\.trim\(\)\)$/, '=');
  assignment(block(block(android, 'buildTypes'), 'release'), 'signingConfig', /^signingConfigs\.release$/);
  const signing = block(block(android, 'signingConfigs'), 'release');
  assignment(signing, 'def releaseKeystorePath', /^System\.getenv\("LOOPAWARE_ANDROID_KEYSTORE"\) \?: "missing-release-keystore"$/, '=');
  assignment(signing, 'def releaseKeystoreFile', /^new File\(releaseKeystorePath\)$/, '=');
  for (const [name, expression] of Object.entries({ storeFile: /^releaseKeystoreFile\.isAbsolute\(\) \? releaseKeystoreFile : new File\(System\.getenv\("MPRLAB_APP_ROOT"\), releaseKeystorePath\)$/,
    storePassword: /^System\.getenv\("LOOPAWARE_ANDROID_STORE_PASSWORD"\) \?: ""$/, keyAlias: /^System\.getenv\("LOOPAWARE_ANDROID_KEY_ALIAS"\) \?: ""$/, keyPassword: /^System\.getenv\("LOOPAWARE_ANDROID_KEY_PASSWORD"\) \?: ""$/ })) assignment(signing, name, expression);
  for (const name of ['android/gradlew', 'android/settings.gradle', 'android/build.gradle', 'android/gradle.properties', 'android/gradle/wrapper/gradle-wrapper.jar', 'android/app/src/main/AndroidManifest.xml']) required(prepared, name);
  const gradle = text(prepared, 'android/gradle/wrapper/gradle-wrapper.properties').match(/gradle-([\d.]+)-(?:bin|all)\.zip/)?.[1];
  assert.ok(gradle, 'native Gradle wrapper version missing');
  const cocoaPods = lock.match(/^COCOAPODS:\s*(\S+)/m)?.[1]; assert.ok(cocoaPods, 'native CocoaPods version missing');
  return { schema_version: 3, status: 'prepared', application: selected,
    configuration: { iosPreparedBuildNumber: field(settings, 'CURRENT_PROJECT_VERSION'), iosUsesNonExemptEncryption, iosSourceDependencies: true, expoPrecompiledModules: false,
      androidVersionNameEnvironment: 'MPRLAB_MOBILE_VERSION_NAME', androidVersionCodeEnvironment: 'MPRLAB_MOBILE_VERSION_CODE', productionDevelopmentClient: false },
    dependencies: deps, toolchain: { gradleWrapper: gradle, cocoaPodsRecorded: cocoaPods }, nativeCommands: { expoPrebuild: 'completed', cocoaPods: 'completed' } };
}

/** @param {string} prepared */
export function verifyPreparation(prepared) {
  const recorded = json(prepared, 'native-preparation.json');
  assert.ok(recorded.schema_version === 3 && recorded.status === 'prepared', 'obsolete native preparation record: ' + remedy);
  const current = preparationFacts(prepared);
  assert.deepEqual(Object.keys(recorded).sort(), Object.keys(current).sort(), 'native preparation record fields differ: ' + remedy);
  for (const section of /** @type {const} */ (['application', 'configuration', 'dependencies', 'toolchain', 'nativeCommands'])) assert.deepEqual(recorded[section], current[section], 'native preparation ' + section + ' differs: ' + remedy);
  return current;
}

/** @param {string} repository @param {string} prepared */
export function assertCurrentSource(repository, prepared) {
  const facts = verifyPreparation(prepared);
  const mobile = join(repository, 'mobile');
  const configPath = required(mobile, 'app.config.js');
  const require = createRequire(import.meta.url);
  delete require.cache[configPath];
  const config = require(configPath).expo;
  assert.deepEqual(application(config), facts.application, 'native source configuration differs: ' + remedy);
  const source = dependencies(mobile); delete source.dependencies['expo-dev-client'];
  assert.deepEqual(source, facts.dependencies, 'native source dependency configuration differs: ' + remedy);
  return config;
}

/** Copy actual public inputs, without a persisted file inventory or readiness comparison.
 * @param {string} source @param {string} destination @param {boolean} [currentSource] @param {string} [selectedRoot]
 */
function copyPublic(source, destination, currentSource = false, selectedRoot = source) {
  mkdirSync(destination, { recursive: true });
  for (const entry of readdirSync(source, { withFileTypes: true })) {
    const name = entry.name, path = join(source, name), inside = relative(selectedRoot, path);
    if (excludedDirectories.has(name) || excludedFiles.has(name) || name.startsWith('.env') || /\.(keystore|jks|p12|p8|mobileprovision|xcuserstate)$/.test(name)) continue;
    if (currentSource && !inside.includes('/') && (rootMetadata.has(name) || ['prepared', 'ios', 'android'].includes(name))) continue;
    assert.ok(!entry.isSymbolicLink(), 'public native source has a symbolic link: ' + inside);
    if (entry.isDirectory()) copyPublic(path, join(destination, name), currentSource, selectedRoot);
    else if (entry.isFile()) copyFileSync(path, join(destination, name));
  }
}

/** Current public fixtures use path selection only, independently of the preparation record.
 * @param {string} prepared @param {string} destination
 */
export function copyPreparedPublicInputs(prepared, destination) { copyPublic(prepared, destination); }

/** Build from captured current product source while retaining prepared native/dependency inputs.
 * @param {string} repository @param {string} prepared
 */
export function stageStoreSource(repository, prepared) {
  const config = assertCurrentSource(repository, prepared);
  const staged = mkdtempSync(join(tmpdir(), 'loopaware-native-source-'));
  try {
    for (const name of ['package.json', 'package-lock.json', 'native-preparation.json']) copyFileSync(required(prepared, name), join(staged, name));
    for (const name of ['ios', 'android']) copyPublic(join(prepared, name), join(staged, name));
    copyPublic(join(repository, 'mobile'), staged, true);
    writeFileSync(join(staged, 'app.config.snapshot.json'), JSON.stringify({ expo: config }, null, 2) + '\n');
    verifyPreparation(staged);
    return staged;
  } catch (error) { rmSync(staged, { recursive: true, force: true }); throw error; }
}

/** Record semantic facts only after canonical native preparation succeeds.
 * @param {string} repository
 */
export function recordPreparation(repository) {
  const prepared = join(repository, 'mobile/prepared');
  const facts = preparationFacts(prepared);
  const mobile = join(repository, 'mobile'), require = createRequire(import.meta.url);
  const config = require(required(mobile, 'app.config.js')).expo;
  assert.deepEqual(application(config), facts.application, 'native source configuration differs: ' + remedy);
  const source = dependencies(mobile); delete source.dependencies['expo-dev-client'];
  assert.deepEqual(source, facts.dependencies, 'native source dependency configuration differs: ' + remedy);
  writeFileSync(join(prepared, 'native-preparation.json'), JSON.stringify(facts, null, 2) + '\n');
}

if (process.argv[1] && realpathSync(process.argv[1]) === fileURLToPath(import.meta.url)) {
  recordPreparation(resolve(process.argv[2] ?? resolve(import.meta.dirname, '../..')));
  console.info('Recorded selected native application configuration and dependency versions.');
}
