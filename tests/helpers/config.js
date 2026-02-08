// @ts-check
import * as fs from 'node:fs';
import * as path from 'node:path';
import { fileURLToPath } from 'node:url';

const helperDirectory = path.dirname(fileURLToPath(import.meta.url));
const repositoryRoot = path.resolve(helperDirectory, '..', '..');
const configDirectory = path.join(repositoryRoot, 'configs');

function resolveEnvFilePath() {
  const override = String(process.env.LOOPAWARE_ENV_FILE || '').trim();
  if (override) {
    return path.resolve(override);
  }
  return path.join(configDirectory, '.env.loopaware');
}

function parseEnvLine(line) {
  const trimmed = line.trim();
  if (!trimmed || trimmed.startsWith('#')) {
    return null;
  }
  const separatorIndex = trimmed.indexOf('=');
  if (separatorIndex <= 0) {
    return null;
  }
  const key = trimmed.slice(0, separatorIndex).trim();
  if (!key) {
    return null;
  }
  let value = trimmed.slice(separatorIndex + 1).trim();
  if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))) {
    value = value.slice(1, -1);
  }
  return { key, value };
}

function loadEnvFile(filePath) {
  const resolved = path.resolve(filePath);
  if (!fs.existsSync(resolved)) {
    return {};
  }
  const contents = fs.readFileSync(resolved, 'utf-8');
  const env = {};
  contents.split(/\r?\n/).forEach((line) => {
    const parsed = parseEnvLine(line);
    if (!parsed) {
      return;
    }
    env[parsed.key] = parsed.value;
  });
  return env;
}

export function resolveTestConfig() {
  const loopawareEnv = loadEnvFile(resolveEnvFilePath());
  const baseURL = process.env.LOOPAWARE_BASE_URL || loopawareEnv.PUBLIC_BASE_URL || 'http://localhost:8090';
  const baseOrigin = new URL(baseURL).origin;
  return {
    repositoryRoot,
    baseURL,
    baseOrigin,
    sessionCookieName: loopawareEnv.TAUTH_SESSION_COOKIE_NAME || 'app_session',
    signingKey: loopawareEnv.TAUTH_JWT_SIGNING_KEY || '',
    tenantId: loopawareEnv.TAUTH_TENANT_ID || 'loopaware',
    subscriptionSecret: loopawareEnv.SESSION_SECRET || '',
    adminEmail: (loopawareEnv.ADMINS || 'admin@example.com').split(',')[0].trim() || 'admin@example.com',
    adminDisplayName: 'Admin Example'
  };
}

export function resolveRepositoryRoot() {
  return repositoryRoot;
}
