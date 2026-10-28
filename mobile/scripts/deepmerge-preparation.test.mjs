// @ts-check
import assert from 'node:assert/strict';
import { cpSync, readFileSync, writeFileSync, existsSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { createRequire } from 'node:module';
import { join, resolve } from 'node:path';
import { pathToFileURL } from 'node:url';

const repository = resolve(import.meta.dirname, '../..');
const { publicFixture } = await import(pathToFileURL(join(repository, 'tests/mobile/preparation.mjs')).href);
/** @type {{directory:string,mobile:string,prepared:string,environment:NodeJS.ProcessEnv,cleanup:()=>void}} */
const fixture = publicFixture();
let primaryError;
try {
  cpSync(join(repository, 'mobile/.gitignore'), join(fixture.mobile, '.gitignore'));
  cpSync(join(repository, 'mobile/vendor/deepmerge'), join(fixture.mobile, 'vendor/deepmerge'), {recursive:true});
  const hook = join(fixture.directory, 'stop-after-copy.mjs');
  writeFileSync(hook, `import cp from 'node:child_process';import{syncBuiltinESMExports}from'node:module';const actual=cp.spawnSync;cp.spawnSync=(command,args,options)=>{if(command==='git')return actual(command,args,options);if(command==='npm')return {status:23};throw new Error('Unexpected preparation command: '+command);};syncBuiltinESMExports();`);
  const copied = spawnSync(process.execPath, ['--import', hook, join(fixture.mobile, 'scripts/prepare-store.mjs')], {env:fixture.environment,encoding:'utf8',stdio:['ignore','pipe','pipe']});
  if(copied.error)throw copied.error;
  assert.equal(copied.status,23,copied.stdout+copied.stderr);
  assert.throws(()=>process.kill(copied.pid,0),{code:'ESRCH'});
  const selected = join(fixture.prepared,'vendor/deepmerge');
  const identity=JSON.parse(readFileSync(join(selected,'package.json'),'utf8'));
  assert.equal(identity.main,'dist/cjs.js');
  const require=createRequire(join(selected,'package.json'));
  for(const entry of [selected,join(selected,'dist/umd.js')]){
    const merge=require(entry);
    const result=merge(Object.create(null),JSON.parse('{"__proto__":{"loopawarePoisoned":true},"ordinary":{"first":1}}'));
    assert.equal(Object.getPrototypeOf(result),Object.prototype);
    assert.equal(result.loopawarePoisoned,undefined);
    assert.deepEqual(result.ordinary,{first:1});
  }
  console.log('deepmerge_preparation.ok: actual canonical copy retains package main/UMD and corrected public behavior');
} catch(error){primaryError=error;throw error;}
finally {
  try{fixture.cleanup();assert.equal(existsSync(fixture.directory),false);}
  catch(error){throw primaryError?new AggregateError([primaryError,error],'Preparation copy and fixture cleanup failed'):error;}
}
console.log('deepmerge_preparation.cleanup.ok: exact preparer reaped and owned public fixture removed');
