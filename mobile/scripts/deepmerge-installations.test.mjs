// @ts-check
import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, cpSync, rmSync, readFileSync, writeFileSync, existsSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { join, resolve } from 'node:path';

const source = resolve(import.meta.dirname, '..');
const scratch = resolve(source, '../_scratch');
mkdirSync(scratch, {recursive:true});
for (const mode of ['source', 'prepared-install-mode']) {
  const fixture = mkdtempSync(join(scratch, 'B152-install-'));
  let primaryError;
  try {
    for (const name of ['package.json', 'package-lock.json', '.npmrc', 'vendor', 'scripts', 'config']) cpSync(join(source, name), join(fixture, name), {recursive:true});
    mkdirSync(join(fixture,'home'));
    /** @type {NodeJS.ProcessEnv} */
    const env = {PATH:process.env.PATH,HOME:join(fixture,'home'),TMPDIR:fixture,CI:'1',NODE_ENV:mode==='source'?'test':'production',EXPO_NO_DOTENV:'1',npm_config_userconfig:'/dev/null',npm_config_cache:resolve(source,'../.cache/npm')};
    /** @param {string} command @param {string[]} args */
    const run = (command,args) => {
      const result=spawnSync(command,args,{cwd:fixture,env,stdio:['ignore','pipe','pipe'],encoding:'utf8'});
      if(result.error)throw result.error;
      assert.equal(result.status,0,result.stdout+result.stderr);
      assert.throws(()=>process.kill(result.pid,0),{code:'ESRCH'});
      return result;
    };
    if(mode==='prepared-install-mode'){
      const file=join(fixture,'package.json');const metadata=JSON.parse(readFileSync(file,'utf8'));
      delete metadata.dependencies['expo-dev-client'];writeFileSync(file,JSON.stringify(metadata,null,2)+'\n');
      run('npm',['install','--package-lock-only','--ignore-scripts','--include=dev']);
    }
    const flags=mode==='source'?[]:['--include=dev'];
    run('npm',['ci',...flags]);
    const result=run(process.execPath,['scripts/audit-deepmerge-security.mjs']);
    assert.match(result.stdout,/mobile_deepmerge_security.ok/);
    console.log(`dependency_install.ok: ${mode}; npm ci ${flags.join(' ')}; CJS/UMD/Expo/RN behavior passed`);
  } catch(error){primaryError=error;throw error;}
  finally {
    try{rmSync(fixture,{recursive:true,force:true});assert.equal(existsSync(fixture),false);}
    catch(error){throw primaryError?new AggregateError([primaryError,error],'Dependency install and fixture cleanup failed'):error;}
  }
}
console.log('dependency_install.cleanup.ok: exact children reaped and both owned fixtures removed');
