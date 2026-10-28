// @ts-check
import assert from 'node:assert/strict';
import { readFileSync, realpathSync, readdirSync, statSync, lstatSync, mkdtempSync, writeFileSync, rmSync } from 'node:fs';
import { createRequire } from 'node:module';
import path from 'node:path';
import { tmpdir } from 'node:os';

/** Locate every selected package copy in the actual dependency installation. */
/** @param {string} root @param {string} packageName */
function installedPackageCopies(root, packageName) {
  /** @type {string[]} */
  const selected = [];
  const active = new Set(/** @type {string[]} */ ([]));
  const visit = (/** @type {string} */ directory) => {
    const physical = realpathSync(directory);
    assert.ok(!active.has(physical), `Cyclic linked dependency directory: ${directory}`);
    active.add(physical);
    try {
      for (const item of readdirSync(directory, { withFileTypes: true })) {
        if (item.name === '.bin') continue;
        const location = path.join(directory, item.name);
        const directoryEntry = item.isDirectory() || (item.isSymbolicLink() && statSync(location).isDirectory());
        if (item.name === packageName && directoryEntry) selected.push(path.relative(root, location).split(path.sep).join('/'));
        if (!directoryEntry) continue;
        if (item.name.startsWith('@')) visit(location);
        else {
          const children = path.join(location, 'node_modules');
          try { lstatSync(children); }
          catch (error) { if (/** @type {NodeJS.ErrnoException} */ (error).code === 'ENOENT') continue; throw error; }
          visit(children);
        }
      }
    } finally { active.delete(physical); }
  };
  visit(path.join(root, 'node_modules'));
  return selected.sort();
}

/** @param {string} root */
export const installedDeepmergeCopies = root => installedPackageCopies(root, 'deepmerge');

/** Qualify the installed CJS and UMD public entry points without source-index imports. */
/** @param {string} installation @param {string[]} [entries] */
export function verifyInstalledDeepmerge(installation, entries = ['cjs', 'umd']) {
  const require = createRequire(path.join(installation, 'package.json'));
  const identity = require(path.join(installation, 'package.json'));
  assert.equal(identity.name, 'deepmerge');
  assert.equal(identity.version, '4.3.1');
  const selectedRoot = realpathSync(installation);
  for (const selected of entries) {
    const entry = selected === 'cjs' ? require.resolve(installation) : path.join(installation, 'dist/umd.js');
    const relative = path.relative(selectedRoot, realpathSync(entry));
    assert.ok(relative && !relative.startsWith('..') && !path.isAbsolute(relative), 'deepmerge entry must belong to the selected installation');
    const merge = require(entry);
    const payload = () => JSON.parse('{"__proto__":{"loopawarePoisoned":true},"ordinary":{"value":1}}');
    const safe = (/** @type {Record<string, unknown>} */ value) => {
      assert.equal(Object.getPrototypeOf(value), Object.prototype, `${entry}: result prototype changed`);
      assert.equal(Object.hasOwn(value, '__proto__'), false);
      assert.equal(value.loopawarePoisoned, undefined);
      assert.equal(/** @type {Record<string, unknown>} */ ({}).loopawarePoisoned, undefined);
    };
    safe(merge(Object.create(null), payload()));
    safe(merge(payload(), {}));
    safe(merge({ __proto__: null, ordinary: null }, payload()));
    safe(merge(JSON.parse('{"__proto__":null}'), payload()));
    const array = merge([], [payload()]);
    safe(array[0]);
    safe(merge({}, { nested: payload() }).nested);
    safe(merge({ nested: null }, { nested: payload() }).nested);
    assert.deepEqual(merge({ nested: { first: 1 }, items: [1] }, { nested: { second: 2 }, items: [2] }),
      { nested: { first: 1, second: 2 }, items: [1, 2] });
    const symbol = Symbol('ordinary');
    assert.equal(merge({}, { [symbol]: 3 })[symbol], 3);
    const original = { nested: { value: 1 } };
    const cloned = merge({}, original);
    assert.notEqual(cloned.nested, original.nested);
    assert.equal(merge({}, original, { clone: false }).nested, original.nested);
    assert.deepEqual(merge({ items: [1] }, { items: [2] }, { arrayMerge: (/** @type {unknown[]} */ _target, /** @type {unknown[]} */ source) => source }), { items: [2] });
    assert.deepEqual(merge({ count: 1 }, { count: 2 }, { customMerge: (/** @type {string} */ key) => key === 'count' ? (/** @type {number} */ left, /** @type {number} */ right) => left + right : undefined }), { count: 2 });
    assert.deepEqual(merge({ nested: { count: 1 } }, { nested: { count: 2 } },
      { customMerge: (/** @type {string} */ key) => key === 'nested' ? (/** @type {{count:number}} */ left, /** @type {{count:number}} */ right) => ({ count: left.count + right.count }) : undefined }), { nested: { count: 3 } });
    assert.deepEqual(merge.all([{ first: 1 }, { second: 2 }]), { first: 1, second: 2 });
    const sourceArray = [{ ordinary: 1 }];
    assert.equal(merge({items: []}, {items: sourceArray}, {arrayMerge: (/** @type {unknown[]} */ _target, /** @type {unknown[]} */ source) => source}).items, sourceArray);
    const date = new Date(0);
    assert.equal(merge({}, {date}).date, date);
    assert.deepEqual(merge({constructor: {ordinary: 1}, prototype: {ordinary: 2}}, {}), {constructor: {ordinary: 1}, prototype: {ordinary: 2}});
    const custom = {ordinary: {second: 2}};
    assert.equal(merge({ordinary: {first: 1}}, custom, {isMergeableObject: () => false}).ordinary, custom.ordinary);

  }
}

/** Qualify actual installed Expo and React Native resolution and public configuration APIs. */
/** @param {string} root */
export async function verifyDeepmergeConsumers(root) {
  const require = createRequire(path.join(root, 'package.json'));
  const selected = realpathSync(path.dirname(require.resolve('deepmerge/package.json')));
  for (const name of ['@expo/config', '@react-native-community/cli', '@react-native-community/cli-config', '@react-native-community/cli-doctor']) {
    const consumer = createRequire(require.resolve(`${name}/package.json`));
    const installation = path.dirname(consumer.resolve('deepmerge/package.json'));
    assert.equal(realpathSync(installation), selected, `${name} must resolve the selected correction`);
    verifyInstalledDeepmerge(installation);
  }
  const fixture = mkdtempSync(path.join(tmpdir(), 'loopaware-deepmerge-consumers-'));
  let primaryError;
  try {
    writeFileSync(path.join(fixture, 'package.json'), JSON.stringify({ name: 'loopaware-deepmerge-fixture', version: '1.0.0' }));
    writeFileSync(path.join(fixture, 'app.json'), JSON.stringify({ expo: { name: 'Fixture', slug: 'fixture', extra: { first: 1 } } }));
    const expo = require('@expo/config');
    const result = await expo.modifyConfigAsync(fixture, { extra: { second: 2 } }, { skipSDKVersionRequirement: true });
    assert.equal(result.type, 'success');
    assert.ok(result.config);
    assert.deepEqual(result.config.extra, { first: 1, second: 2 });
    writeFileSync(path.join(fixture, 'react-native.config.js'), 'module.exports = { assets: ["fixture-assets"] };\n');
    const native = await require('@react-native-community/cli-config').loadConfigAsync({ projectRoot: fixture });
    assert.deepEqual(native.assets, ['fixture-assets']);
  } catch (error) {
    primaryError = error;
    throw error;
  } finally {
    try { rmSync(fixture, { recursive: true, force: true }); }
    catch (cleanupError) {
      if (primaryError) throw new AggregateError([primaryError, cleanupError], 'Consumer qualification and fixture cleanup failed');
      throw cleanupError;
    }
  }
}

/** Exercise the actual Expo config merge with a null target and untrusted JSON keys. */
/** @param {string} root */
export async function verifyExpoPoisoning(root) {
  const require = createRequire(path.join(root, 'package.json'));
  const fixture = mkdtempSync(path.join(tmpdir(), 'loopaware-expo-merge-'));
  let primaryError;
  try {
    writeFileSync(path.join(fixture, 'package.json'), JSON.stringify({name:'loopaware-config-fixture',version:'1.0.0'}));
    writeFileSync(path.join(fixture, 'app.json'), JSON.stringify({expo:{name:'Fixture',slug:'fixture',extra:null}}));
    const result = await require('@expo/config').modifyConfigAsync(fixture, {extra:JSON.parse('{"__proto__":{"loopawarePoisoned":true},"ordinary":1}')}, {skipSDKVersionRequirement:true});
    assert.equal(result.type,'success');
    assert.equal(Object.getPrototypeOf(result.config.extra),Object.prototype,'Actual Expo config result prototype changed');
    assert.equal(result.config.extra.loopawarePoisoned,undefined);
    assert.equal(result.config.extra.ordinary,1);
  } catch(error) {primaryError=error;throw error;}
  finally {
    try {rmSync(fixture,{recursive:true,force:true});}
    catch(error){throw primaryError?new AggregateError([primaryError,error],'Expo merge and fixture cleanup failed'):error;}
  }
}
