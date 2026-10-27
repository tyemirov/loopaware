// @ts-check
import { readFile, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
// Normalize generated text for source-control checks. This does not decide native readiness.
for (const name of ['android/gradlew.bat', 'android/settings.gradle']) {
  const path = join(process.cwd(), name);
  const text = await readFile(path, 'utf8');
  await writeFile(path, text.replaceAll('\r\n', '\n').replace(/[ \t]+$/gm, ''));
}
