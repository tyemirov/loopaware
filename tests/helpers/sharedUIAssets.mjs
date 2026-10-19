// @ts-check
import { createHash } from 'node:crypto';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import path from 'node:path';

export const revision = 'd29f43b96991ff409db968f56e6b3fc57b6d20e0';
export const assets = {
  'mpr-ui-config.js': '3f56fbd212a516d2bd8b0b95f73ae7ad82952c10d8d5f4e6f8b44d3233f01304',
  'mpr-ui.js': '632336dc393d89cbc53f93cfe75a3686971e4ba24acae3e9393a13d9144559d3',
  'mpr-ui.css': 'b6ee455076512890aedddfd249ba8d73ec8e67c69cdfe8ad127fc412dc809db7',
};
const directory = path.join(import.meta.dirname, '../../.cache/mpr-ui-candidate', revision);
let preparation;

async function prepareSharedUI() {
  await mkdir(directory, { recursive: true });
  for (const [name, digest] of Object.entries(assets)) {
    const destination = path.join(directory, name);
    let bytes;
    try {
      bytes = await readFile(destination);
    } catch (error) {
      if (error.code !== 'ENOENT') throw error;
      const response = await fetch(`https://raw.githubusercontent.com/MarcoPoloResearchLab/mpr-ui/${revision}/${name}`, {
        signal: AbortSignal.timeout(15_000),
      });
      if (!response.ok) throw new Error(`shared_ui_download:${name}:${response.status}`);
      bytes = Buffer.from(await response.arrayBuffer());
    }
    if (createHash('sha256').update(bytes).digest('hex') !== digest) throw new Error(`shared_ui_digest:${name}`);
    await writeFile(destination, bytes);
  }
}

/** @param {import('@playwright/test').Page} page */
export async function installSharedUIAssets(page) {
  preparation ??= prepareSharedUI();
  await preparation;
  for (const name of Object.keys(assets)) {
    const body = await readFile(path.join(directory, name));
    await page.route(`https://cdn.jsdelivr.net/gh/MarcoPoloResearchLab/mpr-ui@*/${name}*`, route =>
      route.fulfill({ body, contentType: name.endsWith('.css') ? 'text/css' : 'application/javascript' }));
  }
}
