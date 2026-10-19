// @ts-check
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { cp, mkdir, mkdtemp, readFile, realpath, rm, stat, symlink, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { test } from 'node:test';

const repository = path.resolve(import.meta.dirname, '..');
const phases = ['release', 'publish', 'deploy'];
const ownedInputs = ['GH_TOKEN', 'GITHUB_TOKEN', 'NPM_TOKEN', 'LOOPAWARE_ANDROID_KEYSTORE', 'LOOPAWARE_ANDROID_STORE_PASSWORD', 'LOOPAWARE_ANDROID_KEY_ALIAS', 'LOOPAWARE_ANDROID_KEY_PASSWORD', 'APP_STORE_CONNECT_API_KEY_ID', 'APP_STORE_CONNECT_API_ISSUER_ID', 'APP_STORE_CONNECT_API_KEY_PATH'];
const fileInputs = { GH_TOKEN: 'fixture-file-token', LOOPAWARE_ANDROID_KEYSTORE: 'fixture-keystore', NPM_API_KEY: 'fixture-npm-key' };

/** @param {string} cwd @param {string} command @param {string[]} args @param {NodeJS.ProcessEnv} environment */
function run(cwd, command, args, environment) {
  const result = spawnSync(command, args, { cwd, env: environment, encoding: 'utf8', input: 'caller input must not reach Gateway\n', timeout: 120_000 });
  assert.ifError(result.error);
  return result;
}

test('public lifecycle Make wrappers use an installed Gateway without a sibling checkout', async (context) => {
  const temporary = await realpath(await mkdtemp(path.join(tmpdir(), 'loopaware-lifecycle-command-')));
  try {
    const application = path.join(temporary, 'application checkout');
    const bin = path.join(temporary, 'bin');
    const executable = path.join(bin, 'gateway executable');
    const receipt = path.join(temporary, 'gateway-receipt.json');
    await mkdir(path.join(application, 'configs'), { recursive: true });
    await mkdir(bin);
    await cp(path.join(repository, 'Makefile'), path.join(application, 'Makefile'));
    const source = path.join(temporary, 'gateway.go');
    await writeFile(source, `package main
import ("encoding/json"; "io"; "os"; "strconv"; "strings")
func main() {
 input, err := io.ReadAll(os.Stdin); if err != nil { panic(err) }
 values := map[string]string{}; for _, key := range strings.Split(os.Getenv("FIXTURE_ENV_KEYS"), ",") { values[key] = os.Getenv(key) }
 data, err := json.Marshal(map[string]any{"args":os.Args[1:], "stdin":string(input), "env":values}); if err != nil { panic(err) }
 file, err := os.Create(os.Getenv("FIXTURE_RECEIPT")); if err != nil { panic(err) }; if _, err := file.Write(data); err != nil { panic(err) }; if err := file.Close(); err != nil { panic(err) }
 code, err := strconv.Atoi(os.Getenv("FIXTURE_EXIT")); if err != nil { panic(err) }; os.Exit(code)
}
`);
    const environment = { ...process.env, CI: '0', FIXTURE_EXIT: '0', FIXTURE_RECEIPT: receipt, FIXTURE_ENV_KEYS: [...ownedInputs, ...Object.keys(fileInputs), 'CI', 'APP_SENTINEL'].join(','), APP_SENTINEL: 'fixture-inherited', PATH: bin + path.delimiter + process.env.PATH };
    delete environment.NODE_TEST_CONTEXT;
    delete environment.MPRLAB_GATEWAY_EXECUTABLE;
    for (const key of ownedInputs) environment[key] = 'fixture-stale';
    assert.equal(run(temporary, 'go', ['build', '-o', executable, source], { ...environment, CI: '1' }).status, 0);
    await symlink(executable, path.join(bin, 'mprlab-gateway'));
    assert.equal(run(application, 'git', ['init', '--quiet'], environment).status, 0);
    await writeFile(path.join(application, 'configs/.env.loopaware'), Object.entries(fileInputs).map(([key, value]) => `${key}=${value}`).join('\n') + '\n');
    assert.equal(await stat(path.join(temporary, 'mprlab-gateway')).then(() => true, error => { assert.equal(error.code, 'ENOENT'); return false; }), false);

    for (const phase of phases) {
      for (const selection of ['default', 'explicit']) {
        await context.test(`${phase} uses the ${selection} installed executable and the owned environment`, async () => {
          await rm(receipt, { force: true });
          const selected = { ...environment };
          if (selection === 'explicit') selected.MPRLAB_GATEWAY_EXECUTABLE = executable;
          const result = run(application, 'make', [phase], selected);
          assert.equal(result.status, 0, result.stdout + result.stderr);
          const observed = JSON.parse(await readFile(receipt, 'utf8'));
          assert.deepEqual(observed.args, [`app-${phase}`, '--app-root', application]);
          assert.equal(observed.stdin, '');
          for (const key of ownedInputs) assert.equal(observed.env[key], fileInputs[key] ?? '', key);
          for (const [key, value] of Object.entries(fileInputs)) assert.equal(observed.env[key], value, key);
          assert.equal(observed.env.CI, '1');
          assert.equal(observed.env.APP_SENTINEL, 'fixture-inherited');
        });
      }
      await context.test(`${phase} preserves the Gateway failure status`, () => {
        const result = run(application, 'make', [phase], { ...environment, FIXTURE_EXIT: '37' });
        assert.notEqual(result.status, 0);
        assert.match(result.stdout + result.stderr, /Error 37/);
      });
    }
    await context.test('a missing explicit executable fails without another selection', async () => {
      await rm(receipt, { force: true });
      const missing = path.join(bin, 'missing Gateway');
      const result = run(application, 'make', ['release'], { ...environment, MPRLAB_GATEWAY_EXECUTABLE: missing });
      assert.notEqual(result.status, 0);
      assert.ok((result.stdout + result.stderr).includes(missing));
      assert.equal(await stat(receipt).then(() => true, () => false), false);
    });
    await context.test('an absent private environment file stops before Gateway', async () => {
      await rm(receipt, { force: true });
      await rm(path.join(application, 'configs/.env.loopaware'));
      const result = run(application, 'make', ['release'], environment);
      assert.notEqual(result.status, 0);
      assert.ok((result.stdout + result.stderr).includes('configs/.env.loopaware'));
      assert.equal(await stat(receipt).then(() => true, () => false), false);
    });
  } finally {
    await rm(temporary, { recursive: true, force: true });
  }
});
