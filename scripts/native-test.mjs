// @ts-check
import {
  test as nativeTestRegistration,
  suite as nativeSuite,
  before as nativeBefore,
  beforeEach as nativeBeforeEach,
} from 'node:test';

export { after, afterEach, assert, expectFailure, getTestContext, mock, only, run, skip, snapshot, todo } from 'node:test';

/** @typedef {(...args: any[]) => Promise<void>} NativeRegistration */
/** @typedef {'skip'|'todo'|'only'|'expectFailure'} RegistrationVariant */
/** @typedef {NativeRegistration & Partial<Record<RegistrationVariant, NativeRegistration>>} RegistrationWithVariants */
/** @typedef {import('node:test').TestContext | import('node:test').SuiteContext} OwnedContext */

const owner = new AbortController();
/** @type {Error | undefined} */
let firstFailure;

/** @param {Error} reason */
export function cancelOwnedTests(reason) {
  if (!owner.signal.aborted) owner.abort(reason);
}

/** @param {Error} error */
export function recordOwnedFailure(error) {
  firstFailure ||= error;
  cancelOwnedTests(new Error('Canceled after the first failed test', { cause: firstFailure }));
}

/** @param {import('node:test').TestOptions | undefined} options */
function ownedOptions(options) {
  const supplied = options || {};
  const signal = supplied.signal
    ? AbortSignal.any([/** @type {AbortSignal} */ (supplied.signal), owner.signal])
    : owner.signal;
  return { ...supplied, signal };
}

/** @template {object} T @param {T} context @returns {T} */
function ownedContext(context) {
  return new Proxy(context, {
    get(target, key) {
      const value = Reflect.get(target, key, target);
      if (key === 'test') {
        return (/** @type {unknown[]} */ ...args) => {
          const result = register(/** @type {NativeRegistration} */ (value).bind(target), args);
          return result.then(outcome => {
            if (firstFailure) throw firstFailure;
            if (owner.signal.aborted) throw owner.signal.reason;
            return outcome;
          });
        };
      }
      return typeof value === 'function' ? value.bind(target) : value;
    },
  });
}

/** @param {NativeRegistration} native @param {unknown[]} args @returns {Promise<void>} */
function register(native, args) {
  let name;
  let options;
  let fn;
  if (typeof args[0] === 'function') fn = args[0];
  else if (typeof args[0] === 'object') { options = /** @type {import('node:test').TestOptions} */ (args[0]); fn = args[1]; }
  else {
    name = args[0];
    if (typeof args[1] === 'function') fn = args[1];
    else { options = /** @type {import('node:test').TestOptions | undefined} */ (args[1]); fn = args[2]; }
  }
  let callback = fn;
  if (typeof fn === 'function') {
    const body = fn;
    callback = body.length === 2
      ? /** @this {unknown} @param {OwnedContext} context @param {(result?: unknown) => void} done */ function (context, done) { return body.call(this, ownedContext(context), done); }
      : /** @this {unknown} @param {OwnedContext} context */ function (context) { return body.call(this, ownedContext(context)); };
  }
  return native(name, ownedOptions(options), callback);
}

/** @template {RegistrationWithVariants} T @param {T} native @returns {T} */
function registration(native) {
  const bound = /** @type {RegistrationWithVariants} */ ((/** @type {unknown[]} */ ...args) => register(native, args));
  Object.assign(bound, native);
  for (const variant of /** @type {RegistrationVariant[]} */ (['skip', 'todo', 'only', 'expectFailure'])) {
    const method = native[variant];
    if (typeof method === 'function') bound[variant] = (...args) => register(method, args);
  }
  return /** @type {T} */ (bound);
}

export const test = registration(nativeTestRegistration);
export const it = test;
export const suite = registration(nativeSuite);
export const describe = suite;
/** @type {typeof nativeBefore} */
export const before = (fn, options) => nativeBefore(fn, ownedOptions(options));
/** @type {typeof nativeBeforeEach} */
export const beforeEach = (fn, options) => nativeBeforeEach(fn, ownedOptions(options));

Object.assign(test, { test, it, suite, describe, before, beforeEach });
export default test;
