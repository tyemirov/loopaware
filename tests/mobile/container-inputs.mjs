// @ts-check
import assert from 'node:assert/strict';
import { mkdtemp, mkdir, writeFile, copyFile, readFile, rm } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { spawnSync } from 'node:child_process';

const temporary = await mkdtemp(join(tmpdir(), 'loopaware-container-inputs-'));
try {
  const context = join(temporary, 'context');
  const output = join(temporary, 'output');
  await mkdir(context);
  await copyFile('.dockerignore', join(context, '.dockerignore'));
  for (const name of ['configs/signing/upload.jks', 'mobile/prepared/ios/Pods/native-cache', 'configs/.env.loopaware', 'configs/public.yml']) {
    await mkdir(join(context, name, '..'), { recursive: true });
    await writeFile(join(context, name), 'synthetic fixture\n');
  }
  await writeFile(join(context, 'Dockerfile'), 'FROM scratch\nCOPY . /context/\n');
  const result = spawnSync('docker', ['build', '--output', `type=local,dest=${output}`, context], { encoding: 'utf8' });
  assert.equal(result.status, 0, result.stderr);
  assert.equal(await readFile(join(output, 'context/configs/public.yml'), 'utf8'), 'synthetic fixture\n');
  for (const name of ['configs/signing/upload.jks', 'mobile/prepared/ios/Pods/native-cache', 'configs/.env.loopaware']) {
    await assert.rejects(readFile(join(output, 'context', name)), { code: 'ENOENT' }, `Docker context must exclude ${name}`);
  }
  console.info('Docker excludes private signing inputs and prepared mobile files.');
} finally {
  await rm(temporary, { recursive: true, force: true });
}
