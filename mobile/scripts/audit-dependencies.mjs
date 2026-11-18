// @ts-check
import { spawnSync } from 'node:child_process';
import { readFile } from 'node:fs/promises';
import { verifyInstalledDeepmerge, installedDeepmergeCopies } from './deepmerge-security-probes.mjs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

/** @typedef {{package:string,version:string,patch:string,upstream:{pull_request:string},installations:string[]}} LocalCorrection */
/** @typedef {{name:string,dependency:string,url:string,severity:string,range:string}} Advisory */
/** @typedef {{via:(string|Advisory)[],nodes:string[],severity:string}} Finding */
/** @typedef {{auditReportVersion:number,error?:unknown,vulnerabilities:Record<string,Finding>,metadata:{vulnerabilities:Record<string,number>}}} AuditReport */

const root = fileURLToPath(new URL('..', import.meta.url));
const json = async (/** @type {string} */ file) => JSON.parse(await readFile(path.join(root, file), 'utf8'));
/** @returns {never} */
const fail = (/** @type {string} */ message) => { throw new Error(`Dependency audit: ${message}`); };
const keys = (/** @type {unknown} */ object, /** @type {string[]} */ expected, /** @type {string} */ subject) => {
  if (!object || Array.isArray(object) || typeof object !== 'object' ||
      Object.keys(object).sort().join() !== [...expected].sort().join()) fail(`${subject} schema is invalid.`);
};
const policy = await json('config/dependency-audit.json');
keys(policy, ['schema', 'local_corrections'], 'policy');
if (policy.schema !== 'loopaware.dependency-audit.v2' || !Array.isArray(policy.local_corrections) || policy.local_corrections.length !== 1) fail('policy schema is invalid.');
const lock = /** @type {{lockfileVersion:number,packages:Record<string,{version?:string,link?:boolean,resolved?:string}>}} */ (await json('package-lock.json'));
if (lock.lockfileVersion !== 3 || !lock.packages) fail('lockfile schema is invalid.');
const corrections = new Map([
  ['deepmerge', { package: 'deepmerge', version: '4.3.1', patch: 'PROTOTYPE-CORRECTION.patch', upstream: 'https://github.com/TehShrike/deepmerge/pull/274', copies: installedDeepmergeCopies, verify: verifyInstalledDeepmerge }],
]);
const localCorrections = new Map(/** @type {[string,LocalCorrection][]} */ ([]));
for (const entry of /** @type {LocalCorrection[]} */ (policy.local_corrections)) {
  keys(entry, ['package', 'version', 'patch', 'upstream', 'installations'], 'remediation');
  keys(entry.upstream, ['pull_request'], 'upstream');
  const correction = corrections.get(entry.package);
  if (!correction || localCorrections.has(entry.package) ||
      entry.package !== correction.package || entry.version !== correction.version || entry.upstream.pull_request !== correction.upstream ||
      !Array.isArray(entry.installations) || entry.installations.length === 0 || new Set(entry.installations).size !== entry.installations.length ||
      entry.installations.some((location) => typeof location !== 'string' || !location.startsWith('node_modules/') || !location.endsWith(`node_modules/${entry.package}`) || location.includes('\\') || location.split('/').some((part) => !part || part === '.' || part === '..')) ||
      !/^https:\/\/github.com\/[^/]+\/[^/]+\/pull\/\d+$/.test(entry.upstream.pull_request)) fail('local correction identity is invalid.');
  const patchPath = `vendor/${entry.package}/${correction.patch}`;
  if (entry.patch !== patchPath) fail(`${entry.package} patch path is invalid.`);
  await readFile(path.join(root, patchPath));
  const provenance = await json(`vendor/${entry.package}/PROVENANCE.json`);
  if (provenance.upstream_package !== entry.package || provenance.upstream_version !== entry.version ||
      provenance.advisory !== 'GHSA-gcqx-w48m-w2h4' || provenance.advisory_status !== 'withdrawn' ||
      provenance.advisory_withdrawn_at !== '2026-11-16' || provenance.upstream_fix_reference !== entry.upstream.pull_request || provenance.source !== `https://registry.npmjs.org/${entry.package}/${entry.version}`) fail('local correction provenance differs.');
  const expected = Object.entries(lock.packages).filter(([location]) => location.endsWith(`node_modules/${entry.package}`));
  if (expected.some(([, value]) => value.version !== entry.version) ||
      JSON.stringify(expected.map(([location]) => location).sort()) !== JSON.stringify([...entry.installations].sort())) fail(`${entry.package} lockfile installations differ.`);
  for (const location of entry.installations) {
    const selected = lock.packages[location];
    if (selected.link || selected.resolved !== `file:vendor/${entry.package}`) fail('selected local package source differs.');
  }
  const installed = correction.copies(root);
  if (JSON.stringify(installed.sort()) !== JSON.stringify([...entry.installations].sort())) fail(`${entry.package} installed copies differ.`);
  for (const location of entry.installations) {
    if ((await json(`${location}/package.json`)).version !== entry.version) fail(`${location} installed version differs.`);
    try { correction.verify(path.join(root, location)); }
    catch (error) { fail(`${location} behavior failed: ${/** @type {Error} */ (error).message}`); }
  }
  localCorrections.set(entry.package, entry);
}
const result = spawnSync('npm', ['audit', '--json', ...process.argv.slice(2)], { cwd: root, encoding: 'utf8' });
if (result.error) throw result.error;
/** @type {AuditReport} */
let report;
try { report = JSON.parse(result.stdout); } catch { fail('npm audit did not return JSON.'); }
if ((result.status !== 0 && result.status !== 1) || report.error || report.auditReportVersion !== 2 ||
    !report.vulnerabilities || Array.isArray(report.vulnerabilities) || typeof report.vulnerabilities !== 'object' || !report.metadata?.vulnerabilities ||
    !Number.isInteger(report.metadata.vulnerabilities.total)) fail('npm audit failed or returned an invalid report.');
const severities = ['info', 'low', 'moderate', 'high', 'critical'];
const counts = report.metadata.vulnerabilities;
if ([...severities, 'total'].some((severity) => !Number.isInteger(counts[severity]) || counts[severity] < 0) ||
    severities.reduce((total, severity) => total + counts[severity], 0) !== counts.total ||
    Object.keys(report.vulnerabilities).length !== counts.total) fail('invalid vulnerability counts.');
if (counts.total !== 0) fail('reported vulnerabilities are not permitted.');
if (result.status !== 0) fail('npm audit exit status disagrees with the zero-vulnerability report.');
process.stdout.write(`Dependency audit passed: zero reported vulnerabilities; ${localCorrections.size} behavior-qualified local corrections.\n`);
