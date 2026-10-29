// @ts-check
import assert from 'node:assert/strict';
import path from 'node:path';
import { installedDeepmergeCopies, verifyInstalledDeepmerge, verifyDeepmergeConsumers, verifyExpoPoisoning } from './deepmerge-security-probes.mjs';
const root = path.resolve(import.meta.dirname, '..');
const args = process.argv.slice(2);
assert.ok(args.length <= 1 && (!args.length || ['--entry=cjs','--entry=umd','--expo-merge'].includes(args[0])), 'Unsupported deepmerge qualification selector');
if(args[0]==='--expo-merge') await verifyExpoPoisoning(root);
else {
  const copies=installedDeepmergeCopies(root);
  assert.ok(copies.length>0,'No installed deepmerge copy');
  const entries=args.length?[args[0].slice('--entry='.length)]:['cjs','umd'];
  for(const copy of copies)verifyInstalledDeepmerge(path.join(root,copy),entries);
  if(!args.length){await verifyDeepmergeConsumers(root);await verifyExpoPoisoning(root);}
}
console.log('mobile_deepmerge_security.ok: selected installed entries and consumers');
