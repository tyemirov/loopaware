// @ts-check
import { chromium } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildAdminUser } from '../helpers/fixtures.js';
import { buildSessionCookie } from '../helpers/auth.js';
import { captureLocatorScreenshot, capturePageScreenshot, ensureScreenshotDirectory } from '../helpers/screenshot.js';

function printUsage() {
  console.log(`Usage:
  npm --prefix tests run capture:visual -- --path /widget-integration?site_id=demo --name widget-panel --selector '#mp-feedback-panel'
  npm --prefix tests run capture:visual -- --url http://localhost:8090/app --name dashboard --admin --wait-for '#user-name'

Options:
  --path <path>         Relative path to open on LOOPAWARE_BASE_URL.
  --url <url>           Absolute URL to open.
  --name <name>         Screenshot file name without extension.
  --label <label>       Directory label under tests/artifacts/<date>/.
  --selector <css>      Capture a locator instead of the full page.
  --wait-for <css>      Wait for a selector before capturing.
  --delay-ms <ms>       Sleep for a fixed number of milliseconds before capture.
  --admin               Add an authenticated admin session cookie before navigation.
  --full-page           Capture the full page instead of the viewport.
  --help                Show this message.
`);
}

function parseArgs(argv) {
  const options = {
    path: '',
    url: '',
    name: '',
    label: '',
    selector: '',
    waitFor: '',
    delayMs: 0,
    admin: false,
    fullPage: false
  };

  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index];
    switch (argument) {
      case '--path':
        options.path = argv[index + 1] || '';
        index += 1;
        break;
      case '--url':
        options.url = argv[index + 1] || '';
        index += 1;
        break;
      case '--name':
        options.name = argv[index + 1] || '';
        index += 1;
        break;
      case '--label':
        options.label = argv[index + 1] || '';
        index += 1;
        break;
      case '--selector':
        options.selector = argv[index + 1] || '';
        index += 1;
        break;
      case '--wait-for':
        options.waitFor = argv[index + 1] || '';
        index += 1;
        break;
      case '--delay-ms':
        options.delayMs = Number(argv[index + 1] || '0');
        index += 1;
        break;
      case '--admin':
        options.admin = true;
        break;
      case '--full-page':
        options.fullPage = true;
        break;
      case '--help':
        printUsage();
        process.exit(0);
        break;
      default:
        throw new Error(`unknown_argument:${argument}`);
    }
  }

  return options;
}

function resolveTargetURL(config, options) {
  const trimmedURL = String(options.url || '').trim();
  if (trimmedURL) {
    return trimmedURL;
  }
  const trimmedPath = String(options.path || '').trim();
  if (!trimmedPath) {
    throw new Error('missing_target');
  }
  return new URL(trimmedPath, config.baseURL).toString();
}

async function main() {
  const options = parseArgs(process.argv.slice(2));
  const screenshotName = String(options.name || '').trim();
  if (!screenshotName) {
    throw new Error('missing_name');
  }

  const config = resolveTestConfig();
  const targetURL = resolveTargetURL(config, options);
  const directory = ensureScreenshotDirectory(options.label || screenshotName);

  const browser = await chromium.launch({ headless: true });
  try {
    const context = await browser.newContext({
      baseURL: config.baseURL,
      viewport: { width: 1280, height: 720 },
      ignoreHTTPSErrors: true
    });

    if (options.admin) {
      const adminUser = buildAdminUser(config);
      await context.addCookies([buildSessionCookie(config, adminUser)]);
    }

    const page = await context.newPage();
    await page.goto(targetURL, { waitUntil: 'domcontentloaded' });
    if (options.waitFor) {
      await page.locator(options.waitFor).waitFor();
    }
    if (Number.isFinite(options.delayMs) && options.delayMs > 0) {
      await page.waitForTimeout(options.delayMs);
    }

    let filePath = '';
    if (options.selector) {
      filePath = await captureLocatorScreenshot(page.locator(options.selector), directory, screenshotName);
    } else {
      filePath = await capturePageScreenshot(page, directory, screenshotName, { fullPage: options.fullPage });
    }

    console.log(filePath);
    await context.close();
  } finally {
    await browser.close();
  }
}

main().catch((error) => {
  const message = error instanceof Error ? error.message : String(error);
  console.error(message);
  printUsage();
  process.exit(1);
});
