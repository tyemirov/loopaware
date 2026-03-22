// @ts-check
import * as fs from 'node:fs';
import * as path from 'node:path';
import { resolveRepositoryRoot } from './config.js';

function sanitizeSegment(value) {
  if (!value) {
    return 'unnamed';
  }
  return String(value)
    .split('')
    .map((char) => {
      if (/[a-zA-Z0-9_-]/.test(char)) {
        return char;
      }
      return '_';
    })
    .join('');
}

function resolveLabel(testInfoOrLabel) {
  if (typeof testInfoOrLabel === 'string') {
    return testInfoOrLabel;
  }
  const titleSegments = Array.isArray(testInfoOrLabel?.titlePath)
    ? testInfoOrLabel.titlePath
    : [testInfoOrLabel?.title || 'test'];
  return titleSegments.join('_');
}

export function ensureScreenshotDirectory(testInfoOrLabel) {
  const root = resolveRepositoryRoot();
  const dateSegment = new Date().toISOString().slice(0, 10);
  const nameSegment = sanitizeSegment(resolveLabel(testInfoOrLabel));
  const directory = path.join(root, 'tests', 'artifacts', dateSegment, nameSegment);
  fs.mkdirSync(directory, { recursive: true });
  return directory;
}

export function saveScreenshot(directory, name, buffer) {
  const filename = `${sanitizeSegment(name)}.png`;
  const filePath = path.join(directory, filename);
  fs.writeFileSync(filePath, buffer);
  return filePath;
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {string} directory
 * @param {string} name
 * @param {Parameters<import('@playwright/test').Page['screenshot']>[0]} [options]
 * @returns {Promise<string>}
 */
export async function capturePageScreenshot(page, directory, name, options) {
  const buffer = await page.screenshot(options);
  return saveScreenshot(directory, name, buffer);
}

/**
 * @param {import('@playwright/test').Locator} locator
 * @param {string} directory
 * @param {string} name
 * @param {Parameters<import('@playwright/test').Locator['screenshot']>[0]} [options]
 * @returns {Promise<string>}
 */
export async function captureLocatorScreenshot(locator, directory, name, options) {
  const buffer = await locator.screenshot(options);
  return saveScreenshot(directory, name, buffer);
}
