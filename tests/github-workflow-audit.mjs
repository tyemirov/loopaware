// @ts-check
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { copyFileSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
const repository = resolve(import.meta.dirname, '..');
const original = readFileSync(join(repository, '.github/workflows/ci.yml'), 'utf8');
const fixture = mkdtempSync(join(tmpdir(), 'loopaware-workflow-audit-'));
/** @type {Array<[string, (source: string) => string, boolean]>} */
const scenarios = [
  ['current six declarations', source => source, true],
  ['missing second Python setup', source => source.replace(/      - name: Set up Python for Gateway installation\n[\s\S]*?(?=      - name: Install Gateway for Mobile checks)/, ''), false],
  ['extra approved action', source => source.replace('      - name: Set up Go', '      - name: Extra checkout\n        uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1\n\n      - name: Set up Go'), false],
  ['wrong approved action counts', source => source.replace('uses: actions/setup-python@5fda3b95a4ea91299a34e894583c3862153e4b97', 'uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1'), false],
  ['unapproved pinned action', source => source.replace('uses: actions/setup-node@820762786026740c76f36085b0efc47a31fe5020', 'uses: unknown/setup-node@820762786026740c76f36085b0efc47a31fe5020'), false],
  ['mutable action reference', source => source.replace('actions/setup-python@5fda3b95a4ea91299a34e894583c3862153e4b97', 'actions/setup-python@v7'), false],
  ['write permission', source => source.replace('  contents: read', '  contents: write'), false],
  ['missing aggregate success gate', source => source.replace('run: test "$CHECKS_RESULT" = success', 'run: true'), false],
  ['filtered required pull request', source => source.replace('  pull_request:\n    branches:\n      - master', '  pull_request:\n    branches:\n      - master\n    paths:\n      - mobile/**'), false],
];
try {
  for (const [name, alter, accepted] of scenarios) {
    const root = join(fixture, name.replaceAll(' ', '-'));
    for (const relative of ['scripts/audit-github-workflow.py', '.github/dependabot.yml', 'Makefile']) {
      const destination = join(root, relative); mkdirSync(dirname(destination), { recursive: true });
      copyFileSync(join(repository, relative), destination);
    }
    const source = alter(original);
    if (!accepted) assert.notEqual(source, original, name);
    mkdirSync(join(root, '.github/workflows'));
    writeFileSync(join(root, '.github/workflows/ci.yml'), source);
    const result = spawnSync('python3', ['scripts/audit-github-workflow.py'], {
      cwd: root, stdio: ['ignore', 'pipe', 'pipe'], encoding: 'utf8',
    });
    if (accepted) assert.equal(result.status, 0, result.stdout + result.stderr);
    else { assert.notEqual(result.status, 0, name); assert.match(result.stderr, /github_workflow_audit_failed:/); }
    console.log(`Public workflow audit passed: ${name}.`);
  }
} finally { rmSync(fixture, { recursive: true, force: true }); }
