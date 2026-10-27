// @ts-check
import { spawnSync } from 'node:child_process';
import { lstatSync } from 'node:fs';
import { join } from 'node:path';

/** Select current public filenames only for the preparation copy, without a readiness inventory.
 * @param {string} repository
 * @returns {string[]}
 */
export function preparationSources(repository) {
  /** @param {string[]} options */
  const select = options => {
    const result = spawnSync('git', ['ls-files', ...options, '-z', '--', 'mobile'], { cwd: repository, encoding: 'utf8' });
    if (result.status !== 0) throw new Error('Cannot select native preparation copy inputs.');
    return result.stdout.split('\0').filter(Boolean);
  };
  const deleted = new Set(select(['--deleted']));
  return [...new Set(select(['--cached', '--others', '--exclude-standard']))].filter(name => {
    if (deleted.has(name) || name.endsWith('/AGENTS.md') || name === 'mobile/.gitignore') return false;
    const parts = name.split('/');
    if (['prepared', 'android', 'ios'].includes(parts[1])) return false;
    if (parts.some(part => ['node_modules', 'Pods', 'build', 'DerivedData', '.gradle', '.kotlin', '.cxx', '.expo', 'xcuserdata', '.git'].includes(part) || part.startsWith('.env') || ['local.properties', '.xcode.env.local', '.DS_Store'].includes(part))) return false;
    if (/\.(keystore|jks|p12|p8|mobileprovision|xcuserstate)$/.test(name)) return false;
    let current = repository;
    for (const part of parts) {
      current = join(current, part);
      if (lstatSync(current).isSymbolicLink()) throw new Error('Native preparation copy input has a symbolic link: ' + name);
    }
    return true;
  });
}
