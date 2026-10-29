// @ts-check
import { fork } from 'node:child_process';
import { availableParallelism } from 'node:os';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { run } from 'node:test';
import { tap } from 'node:test/reporters';
import { cancelOwnedTests, recordOwnedFailure } from './native-test.mjs';

/** @typedef {{name?: string, file?: string, message?: string, observed_at_ns?: string}} FailureDetail */
/** @typedef {{type: 'failure', detail: FailureDetail} | {type: 'result', failed: boolean, output: string}} WorkerMessage */
/** @typedef {{code: number | null, signal: NodeJS.Signals | null}} WorkerExit */
/** @typedef {{file: string, diagnostics: string, result?: Extract<WorkerMessage, {type: 'result'}>, exit: WorkerExit}} Outcome */

const script = fileURLToPath(import.meta.url);
const WORKER = '--owned-file-worker';
const FIRST_FAILURE = 'LOOPAWARE_NODE_FIRST_FAILURE';

/** @param {string[]} args */
function selection(args) {
  let concurrency = Math.max(1, availableParallelism() - 1);
  const names = [];
  const files = [];
  for (const arg of args) {
    if (arg.startsWith('--test-concurrency=')) {
      concurrency = Number(arg.slice('--test-concurrency='.length));
      if (!Number.isInteger(concurrency) || concurrency < 1) throw new Error('Invalid test concurrency');
    } else if (arg.startsWith('--test-name-pattern=')) {
      names.push(new RegExp(arg.slice('--test-name-pattern='.length)));
    } else if (arg.startsWith('-')) {
      throw new Error(`Unsupported test option: ${arg}`);
    } else files.push(resolve(arg));
  }
  if (files.length === 0) throw new Error('At least one explicit test file is required');
  return { concurrency, names, files };
}

async function worker() {
  if (!process.send) throw new Error('File worker requires its owned IPC channel');
  const options = /** @type {{file: string, names: {source: string, flags: string}[]}} */ (JSON.parse(process.argv[3]));
  let failed = false;
  let canceled = false;
  const cancel = (/** @type {Error} */ reason) => { canceled = true; cancelOwnedTests(reason); };
  process.on('SIGTERM', () => cancel(new Error('Another test failed')));
  process.on('SIGINT', () => cancel(new Error('Test invocation interrupted')));
  const send = (/** @type {WorkerMessage} */ message) => process.send?.(message);
  process.argv = [process.execPath, options.file];
  const stream = run({
    files: [options.file], isolation: 'none',
    ...(options.names.length ? { testNamePatterns: options.names.map(pattern => new RegExp(pattern.source, pattern.flags)) } : {}),
    setup(events) {
      events.on('test:complete', event => {
        if (failed || event.skip || event.todo || event.details.passed) return;
        failed = true;
        const detail = { name: event.name, file: event.file, message: event.details.error?.message, observed_at_ns: process.hrtime.bigint().toString() };
        process.stderr.write(`${FIRST_FAILURE} ${JSON.stringify(detail)}\n`);
        recordOwnedFailure(event.details.error || new Error(`First failed test: ${event.name}`));
        send({ type: 'failure', detail });
      });
    },
  });
  let output = '';
  for await (const chunk of stream.compose(tap)) output += chunk;
  await new Promise((resolveSent, reject) => process.send?.({ type: 'result', failed, output }, error => error ? reject(error) : resolveSent(undefined)));
  process.exitCode = failed || canceled ? 1 : 0;
  process.disconnect();
}

/** @param {string[]} args @param {typeof fork} [spawnWorker] */
export async function coordinate(args, spawnWorker = fork) {
  const options = selection(args);
  let next = 0;
  let stopped = false;
  /** @type {(FailureDetail & {file: string}) | undefined} */
  let firstFailure;
  const active = new Set(/** @type {import('node:child_process').ChildProcess[]} */ ([]));
  /** @type {(Outcome | undefined)[]} */
  const outcomes = [];
  /** @type {unknown[]} */
  const cleanupErrors = [];
  const cancel = () => {
    stopped = true;
    for (const child of active) {
      try { child.kill('SIGTERM'); } catch (error) { cleanupErrors.push(error); }
    }
  };
  /** @param {string} file @param {unknown} error */
  const fail = (file, error) => {
    if (!firstFailure) {
      firstFailure = { file, message: /** @type {Error} */ (error).message };
      process.stderr.write(`${FIRST_FAILURE} ${JSON.stringify(firstFailure)}\n`);
    }
    cancel();
  };
  process.once('SIGTERM', cancel);
  process.once('SIGINT', cancel);
  /** @param {string} file @param {number} index */
  async function launch(file, index) {
    const child = spawnWorker(script, [WORKER, JSON.stringify({ file, names: options.names.map(pattern => ({ source: pattern.source, flags: pattern.flags })) })], { stdio: ['ignore', 'pipe', 'pipe', 'ipc'] });
    active.add(child);
    let diagnostics = '';
    /** @type {Extract<WorkerMessage, {type: 'result'}> | undefined} */
    let result;
    /** @type {NonNullable<typeof child.stdout>} */ (child.stdout).on('data', chunk => { diagnostics += chunk; });
    /** @type {NonNullable<typeof child.stderr>} */ (child.stderr).on('data', chunk => { diagnostics += chunk; });
    child.on('message', incoming => {
      const message = /** @type {WorkerMessage} */ (incoming);
      if (message?.type === 'failure') {
        if (!firstFailure) {
          firstFailure = { file, ...message.detail };
          process.stderr.write(`${FIRST_FAILURE} ${JSON.stringify(firstFailure)}\n`);
        }
        cancel();
      } else if (message?.type === 'result') result = message;
    });
    const exit = await /** @type {Promise<WorkerExit>} */ (new Promise(resolveExit => {
      child.on('error', error => fail(file, error));
      child.once('close', (code, signal) => resolveExit({ code, signal }));
    }));
    active.delete(child);
    if (exit.code !== 0 && !stopped) {
      firstFailure = { file, message: 'Test file worker failed', ...exit };
      cancel();
    }
    outcomes[index] = { file, diagnostics, result, exit };
    process.stderr.write(`LOOPAWARE_NODE_WORKER_CLOSED ${JSON.stringify({ file, pid: child.pid, ...exit })}\n`);
  }
  async function lane() {
    while (!stopped && next < options.files.length) {
      const index = next++;
      try { await launch(options.files[index], index); }
      catch (error) { fail(options.files[index], error); }
    }
  }
  await Promise.allSettled(Array.from({ length: Math.min(options.concurrency, options.files.length) }, lane));
  process.removeListener('SIGTERM', cancel);
  process.removeListener('SIGINT', cancel);
  process.stdout.write('TAP version 13\n');
  for (let index = 0; index < options.files.length; index++) {
    const outcome = outcomes[index];
    const name = options.files[index].replace(/[\r\n]/g, ' ');
    if (!outcome) {
      process.stdout.write(`ok ${index + 1} - ${name} # SKIP canceled after first failure\n`);
      continue;
    }
    process.stdout.write(`# Subtest: ${name}\n`);
    for (const line of outcome.diagnostics.split(/\r?\n/)) if (line) process.stdout.write(`    # ${line}\n`);
    if (outcome.result) for (const line of outcome.result.output.split(/\r?\n/)) if (line !== 'TAP version 13' && line) process.stdout.write(`    ${line}\n`);
    process.stdout.write(`${outcome.exit.code === 0 ? 'ok' : 'not ok'} ${index + 1} - ${name}\n`);
  }
  process.stdout.write(`1..${options.files.length}\n`);
  if (firstFailure) process.stderr.write(`${FIRST_FAILURE} ${JSON.stringify(firstFailure)}\n`);
  for (const error of cleanupErrors) process.stderr.write(`LOOPAWARE_NODE_CLEANUP_ERROR ${/** @type {Error} */ (error).message}\n`);
  process.exitCode = stopped || cleanupErrors.length ? 1 : 0;
}

if (process.argv[1] && resolve(process.argv[1]) === script) {
  if (process.argv[2] === WORKER) await worker();
  else await coordinate(process.argv.slice(2));
}
