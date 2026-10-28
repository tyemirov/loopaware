// @ts-check
import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { createRequire } from 'node:module';
import { mkdir, readFile, writeFile, rm } from 'node:fs/promises';
import { join } from 'node:path';
import { pathToFileURL } from 'node:url';

/** Qualify the shared Playwright policy through the real browser runner. */
export async function playwrightFirstFailure({ repository, owner, invoke, absent, results }) {
  const root = join(owner, 'playwright');
  await mkdir(root);
  const later = join(root, 'later-case');
  const cleanup = join(root, 'after-each');
  const browsers = join(root, 'browsers');
  const browserCleanup = join(root, 'browser-cleanup');
  const require = createRequire(join(repository, 'tests/package.json'));
  let laterRequests = 0;
  const server = createServer((request, response) => {
    if (request.url === '/later') laterRequests++;
    response.writeHead(200, { 'content-type': 'text/html' });
    response.end('<title>Owned LoopAware first-failure fixture</title>');
  });
  let primaryError;
  try {
    await new Promise(done => server.listen(0, '127.0.0.1', done));
    const address = server.address();
    assert.ok(address && typeof address !== 'string');
    const origin = `http://127.0.0.1:${address.port}`;
    await writeFile(join(root, 'package.json'), '{"type":"module"}\n');
    await writeFile(join(root, 'playwright.config.mjs'), `import base from ${JSON.stringify(pathToFileURL(join(repository, 'tests/playwright.config.js')).href)};export default {...base,testDir:${JSON.stringify(root)},outputDir:${JSON.stringify(join(root, 'output'))}};\n`);
    await writeFile(join(root, 'first-failure.spec.js'), `import{createRequire}from'node:module';import{writeFileSync,appendFileSync}from'node:fs';
const{test:base,expect,chromium}=createRequire(${JSON.stringify(join(repository, 'tests/package.json'))})('@playwright/test');
const test=base.extend({browser:[async({},use)=>{
 const browserServer=await chromium.launchServer({headless:true,...(process.env.LOOPAWARE_PLAYWRIGHT_CHANNEL?{channel:process.env.LOOPAWARE_PLAYWRIGHT_CHANNEL}:{})});
 const browser=await chromium.connect(browserServer.wsEndpoint());
 appendFileSync(${JSON.stringify(browsers)},String(browserServer.process().pid)+'\\n');
 try{await use(browser);}finally{
  const errors=[];for(const close of [()=>browser.close(),()=>browserServer.close()]){try{await close();}catch(error){errors.push(error);}}
  appendFileSync(${JSON.stringify(browserCleanup)},'cleanup\\n');if(errors.length)throw new AggregateError(errors,'Owned browser cleanup failed');
 }
},{scope:'worker'}]});
test.afterEach(()=>appendFileSync(${JSON.stringify(cleanup)},'cleanup\\n'));
test('first-controlled-failure',async({page})=>{await page.goto(${JSON.stringify(origin)});expect(false,'controlled-playwright-first-failure').toBe(true);});
test('later-body',async({page})=>{await page.goto(${JSON.stringify(origin+'/later')});writeFileSync(${JSON.stringify(later)},'started');});
`);
    const result = await invoke([require.resolve('@playwright/test/cli'), 'test', '--config', join(root, 'playwright.config.mjs')], root);
    process.stdout.write(result.output);
    assert.notEqual(result.code, 0, result.output);
    assert.match(result.output, /controlled-playwright-first-failure/);
    assert.match(result.output, /first-controlled-failure/);
    const pids = (await readFile(browsers, 'utf8')).trim().split('\n').map(Number);
    for (const pid of pids) assert.throws(() => process.kill(pid, 0), { code: 'ESRCH' });
    assert.equal((await readFile(browserCleanup, 'utf8')).trim().split('\n').length, pids.length);
    assert.ok((await readFile(cleanup, 'utf8')).startsWith('cleanup'));
    await absent(later);
    assert.equal(laterRequests, 0);
    results.push({ scenario: 'playwright-first-failure', browserPids: pids, browserPidsAbsent: true, enteredCleanup: true, laterRequests });
  } catch (error) { primaryError = error; throw error; }
  finally {
    try { server.closeAllConnections(); await new Promise((done, reject) => server.close(error => error ? reject(error) : done(undefined))); }
    catch (error) { throw primaryError ? new AggregateError([primaryError, error], 'Playwright qualification and server cleanup failed') : error; }
    process.stdout.write('LOOPAWARE_PLAYWRIGHT_FIXTURE_SERVER_CLOSED\n');
  }
}

/** Qualify matrix rejection through the actual repository audit CLI. */
export async function matrixFirstFailure({ repository, owner, invoke, results }) {
  const root = join(owner, 'matrix');
  for (const directory of ['scripts', '.github/workflows']) await mkdir(join(root, directory), { recursive: true });
  for (const name of ['scripts/audit-github-workflow.py', '.github/dependabot.yml', 'Makefile']) {
    await writeFile(join(root, name), await readFile(join(repository, name), 'utf8'));
  }
  const original = await readFile(join(repository, '.github/workflows/ci.yml'), 'utf8');
  const selected = join(root, '.github/workflows/ci.yml');
  for (const [name, source, accepted] of [
    ['disabled matrix fail-fast', original.replace(/^(      fail-fast:) (true|false)$/m, '$1 false'), false],
    ['comment cannot replace matrix field', original.replace(/^(      fail-fast:) (true|false)$/m, '$1 false') + '\n# fail-fast: true\n', false],
    ['enabled matrix fail-fast', original.replace(/^(      fail-fast:) (true|false)$/m, '$1 true'), true],
  ]) {
    await writeFile(selected, source);
    const result = await invoke(['scripts/audit-github-workflow.py'], root, 'python3');
    process.stdout.write(result.output);
    assert.equal(result.code === 0, accepted, name);
    if (!accepted) assert.match(result.output, /github_workflow_audit_failed:/);
    results.push({ scenario: name, accepted });
  }
}

/** Inject marker recipes for every real target before testing CI flag rejection. */
export async function makeFirstFailure({ repository, owner, invoke, absent, results }) {
  const root = join(owner, 'make-flags');
  await mkdir(root);
  const source = await readFile(join(repository, 'Makefile'), 'utf8');
  const targets = new Set();
  for (const match of source.matchAll(/^([a-zA-Z][a-zA-Z0-9_. -]*):([^=].*)?$/gm)) {
    for (const target of match[1].trim().split(/\s+/)) targets.add(target);
  }
  for (const name of ['ci', 'ci-checks', 'ci-backend', 'ci-mobile', 'ci-integration', 'build', 'test-unit', 'test-integration-all']) assert.ok(targets.has(name), `Missing controlled target: ${name}`);
  let owners = [];
  let recipes = 0;
  for (const line of source.split('\n')) {
    if (line.startsWith('\t')) {
      assert.ok(owners.length && owners.every(name => targets.has(name)), 'Unknown actual recipe owner; do not launch fixture');
      recipes++;
    } else if (line && !/^\s|#/.test(line)) {
      const rule = /^([a-zA-Z][a-zA-Z0-9_. -]*):([^=].*)?$/.exec(line);
      owners = rule ? rule[1].trim().split(/\s+/) : [];
    }
  }
  process.stdout.write('LOOPAWARE_MAKE_FIXTURE_CONTROLLED ' + JSON.stringify({ targets: targets.size, recipes }) + '\n');
  const marker = join(root, 'effects');
  const realGuards = [...targets].filter(name => name === 'ci-stage-policy');
  assert.deepEqual(realGuards, ['ci-stage-policy'], 'Keep exactly the real CI policy recipe');
  const policy = /^ci-stage-policy:\n(\t[^\n]*\n)/m.exec(source);
  assert.ok(policy, 'Missing real CI policy recipe');
  assert.match(policy[1], /^\t@set -eu; for specification in /);
  assert.doesNotMatch(policy[1], /\$\(|`|\b(?:docker|node|npm|uv|go|git|make|exec|source)\s/);
  const overrides = [...targets].filter(name => name !== 'ci-stage-policy').map(name => `${name}:\n\t@printf '%s\\n' '${name}' >> '${marker}'\n`).join('\n');
  const file = join(root, 'Makefile');
  await writeFile(file, '$(info LOOPAWARE_MAKE_PARSE_FLAGS parallel=$(if $(filter -j% --jobs%,$(MAKEFLAGS)),yes,no) keepgoing=$(if $(findstring k,$(firstword $(MAKEFLAGS))),yes,no))\n' + source + '\n# Every application recipe is replaced by an owned marker.\n' + overrides);
  for (const [name, args, inherited, gnuInherited = ''] of [
    ['parallel ci', ['-j2', 'ci'], ''],
    ['keep-going ci', ['-k', 'ci'], ''],
    ['inherited parallel backend', ['ci-backend'], '-j2'],
    ['inherited keep-going integration', ['ci-integration'], '-k'],
    ['GNU inherited parallel mobile', ['ci-mobile'], '', '-j2'],
    ['GNU inherited keep-going checks', ['ci-checks'], '', '-k'],
  ]) {
    const env = { ...process.env, MAKEFLAGS: inherited, MFLAGS: '', GNUMAKEFLAGS: gnuInherited };
    const result = await invoke(['--no-print-directory', '-f', file, ...args], root, 'make', env);
    process.stdout.write(result.output);
    let effects = ''; try { effects = await readFile(marker, 'utf8'); } catch (error) { if (error.code !== 'ENOENT') throw error; }
    process.stdout.write('LOOPAWARE_MAKE_CONTROLLED_EFFECTS ' + JSON.stringify({ name, count: effects.trim() ? effects.trim().split('\n').length : 0 }) + '\n');
    assert.notEqual(result.code, 0, name);
    assert.match(result.output, /LoopAware CI requires serial stages and stops on failure/);
    await absent(marker);
    results.push({ scenario: name, testEffects: 0, controlledTargets: targets.size });
  }
  for (const assignment of [[], ['HARMLESS_ASSIGNMENT=-j8 --keep-going --jobserver-auth=fixture']]) {
    const env = { ...process.env, MAKEFLAGS: '', MFLAGS: '', GNUMAKEFLAGS: '' };
    const result = await invoke(['--no-print-directory', '-f', file, 'ci', ...assignment], root, 'make', env);
    process.stdout.write(result.output);
    assert.equal(result.code, 0, result.output);
    const effects = (await readFile(marker, 'utf8')).trim().split('\n');
    const stages = effects.filter(name => ['ci-checks', 'ci-backend', 'ci-mobile', 'ci-integration'].includes(name));
    assert.deepEqual(stages, ['ci-checks', 'ci-backend', 'ci-mobile', 'ci-integration']);
    results.push({ scenario: assignment.length ? 'flag text in assignment is data' : 'default serial order', controlledEffects: effects.length, stages });
    await rm(marker);
  }
}

/** Reproduce native Go continuation in one anonymous standard-library-only module. */
export async function goFirstFailure({ repository, owner, invoke, absent, results, referenceNative }) {
  const root = join(owner, 'go');
  await mkdir(root);
  const later = join(root, 'later-case');
  const cleanup = join(root, 'case-cleanup');
  const workerPID = join(root, 'worker-pid');
  await writeFile(join(root, 'go.mod'), 'module loopaware.local/firstfailure\n\ngo 1.27.3\n');
  await writeFile(join(root, 'first_failure_test.go'), `package firstfailure
import("os";"strconv";"testing")
func writeFixture(t *testing.T,path,value string){t.Helper();file,err:=os.Create(path);if err!=nil{t.Fatal(err)};_,writeErr:=file.WriteString(value);closeErr:=file.Close();if writeErr!=nil{t.Fatal(writeErr)};if closeErr!=nil{t.Fatal(closeErr)}}
func TestFirstControlledFailure(t *testing.T){writeFixture(t,${JSON.stringify(workerPID)},strconv.Itoa(os.Getpid()));t.Cleanup(func(){writeFixture(t,${JSON.stringify(cleanup)},"cleanup")});t.Fatal("controlled-go-first-failure")}
func TestLaterMustNotRun(t *testing.T){writeFixture(t,${JSON.stringify(later)},"started")}
`);
  const env = { PATH: process.env.PATH, HOME: process.env.HOME, CI: '1', GOWORK: 'off', GOPROXY: 'off', CGO_ENABLED: '0' };
  for (const key of ['GOCACHE', 'GOMODCACHE', 'GOTOOLCHAIN']) if (process.env[key]) env[key] = process.env[key];
  const captured = await invoke(['env', '-json', 'GOROOT', 'GOVERSION'], repository, 'go', env);
  assert.equal(captured.code, 0, captured.output);
  const selected = JSON.parse(captured.stdout);
  assert.equal(selected.GOVERSION, 'go1.27.3');
  const executable = join(selected.GOROOT, 'bin/go');
  process.stdout.write('LOOPAWARE_GO_SELECTED_TOOLCHAIN ' + JSON.stringify({ version: selected.GOVERSION, executable }) + '\n');
  const args = referenceNative ? ['-C', root, 'test', '-json', './...']
    : ['run', './tools/testmonitor', '--', '-C', root, './...'];
  const result = await invoke(args, repository, executable, env);
  process.stdout.write(result.output);
  assert.notEqual(result.code, 0, result.output);
  assert.match(result.output, /controlled-go-first-failure/, result.output);
  if (!referenceNative) {
    const starts = [...result.stderr.matchAll(/^LOOPAWARE_GO_DRIVER_STARTED (.+)$/gm)].map(match => JSON.parse(match[1]));
    assert.equal(starts.length, 1, result.output);
    for (const pid of [starts[0].pid, starts[0].monitor_pid]) assert.throws(() => process.kill(pid, 0), { code: 'ESRCH' });
    const failure = /^LOOPAWARE_GO_FIRST_FAILURE (.+)$/m.exec(result.stderr);
    assert.ok(failure, result.output);
    assert.deepEqual(JSON.parse(failure[1]), { action: 'output', package: 'loopaware.local/firstfailure', test: 'TestFirstControlledFailure', output_type: 'frame' });
    assert.match(result.stderr, /LOOPAWARE_GO_DRIVER_REAPED/);
    process.stdout.write('LOOPAWARE_GO_OWNER_PIDS_ABSENT ' + JSON.stringify(starts[0]) + '\n');
  }
  assert.equal(await readFile(cleanup, 'utf8'), 'cleanup');
  const testPID = Number(await readFile(workerPID, 'utf8'));
  assert.throws(() => process.kill(testPID, 0), { code: 'ESRCH' });
  process.stdout.write('LOOPAWARE_GO_NATIVE_WORKER_ABSENT ' + JSON.stringify({ pid: testPID }) + '\n');
  await absent(later);
  results.push({ scenario: 'go-first-failure', laterCaseAbsent: true, enteredCleanup: true });
}


/** Qualify a native parallel failure with a signal-resistant owned descendant. */
export async function goResistant({ repository, owner, invoke, absent, results }) {
  const root = join(owner, 'go-resistant');
  await mkdir(root);
  const ready = join(root, 'helper-ready');
  const ignored = join(root, 'helper-ignored-term');
  const worker = join(root, 'worker-ready');
  const cleanup = join(root, 'failed-case-cleanup');
  await writeFile(join(root, 'go.mod'), 'module loopaware.local/resistant\n\ngo 1.27.3\n');
  await writeFile(join(root, 'resistant_test.go'), `package resistant
import("encoding/json";"os";"os/exec";"os/signal";"path/filepath";"runtime";"syscall";"testing")
func write(t *testing.T,path,value string){t.Helper();file,err:=os.CreateTemp(filepath.Dir(path),"barrier-");if err!=nil{t.Fatal(err)};_,writeErr:=file.WriteString(value);closeErr:=file.Close();if writeErr!=nil{t.Fatal(writeErr)};if closeErr!=nil{t.Fatal(closeErr)};if err:=os.Rename(file.Name(),path);err!=nil{t.Fatal(err)}}
func present(path string)bool{_,err:=os.Stat(path);return err==nil}
func wait(path string){for !present(path){runtime.Gosched()}}
func record(t *testing.T,path string){data,err:=json.Marshal(map[string]int{"pid":os.Getpid(),"pgid":syscall.Getpgrp()});if err!=nil{t.Fatal(err)};write(t,path,string(data))}
func TestOwnedHelper(t *testing.T){if os.Getenv("LOOPAWARE_RESISTANT_HELPER")!="1"{t.Skip("helper subprocess only")};signals:=make(chan os.Signal,1);signal.Notify(signals,syscall.SIGTERM);record(t,${JSON.stringify(ready)});for range signals{if !present(${JSON.stringify(ignored)}){write(t,${JSON.stringify(ignored)},"ignored")}}}
func TestBlockedParallel(t *testing.T){t.Parallel();record(t,${JSON.stringify(worker)});child:=exec.Command(os.Args[0],"-test.run=^TestOwnedHelper$");child.Env=append(os.Environ(),"LOOPAWARE_RESISTANT_HELPER=1");child.Stdout=os.Stdout;child.Stderr=os.Stderr;if err:=child.Start();err!=nil{t.Fatal(err)};select{}}
func TestFirstParallelFailure(t *testing.T){t.Parallel();wait(${JSON.stringify(ready)});data,err:=os.ReadFile(${JSON.stringify(ready)});if err!=nil{t.Fatal(err)};var child map[string]int;if err:=json.Unmarshal(data,&child);err!=nil{t.Fatal(err)};if err:=syscall.Kill(child["pid"],syscall.SIGTERM);err!=nil{t.Fatal(err)};wait(${JSON.stringify(ignored)});t.Cleanup(func(){write(t,${JSON.stringify(cleanup)},"cleanup")});t.Fatal("controlled-go-parallel-first-failure")}
`);
  const env = { PATH: process.env.PATH, HOME: process.env.HOME, CI: '1', GOWORK: 'off', GOPROXY: 'off', CGO_ENABLED: '0' };
  for (const key of ['GOCACHE', 'GOMODCACHE', 'GOTOOLCHAIN']) if (process.env[key]) env[key] = process.env[key];
  const captured = await invoke(['env', '-json', 'GOROOT', 'GOVERSION'], repository, 'go', env);
  assert.equal(captured.code, 0, captured.output);
  const selected = JSON.parse(captured.stdout);
  assert.equal(selected.GOVERSION, 'go1.27.3');
  const result = await invoke(['run', './tools/testmonitor', '--', '-C', root, '-count=1', '-parallel=2', './...'], repository, join(selected.GOROOT, 'bin/go'), env, true);
  assert.notEqual(result.code, 0, result.output);
  assert.match(result.output, /controlled-go-parallel-first-failure/);
  const first = /^LOOPAWARE_GO_FIRST_FAILURE (.+)$/m.exec(result.stderr);
  assert.ok(first, result.output);
  assert.deepEqual(JSON.parse(first[1]), { action: 'output', package: 'loopaware.local/resistant', test: 'TestFirstParallelFailure', output_type: 'frame' });
  assert.equal(await readFile(cleanup, 'utf8'), 'cleanup');
  assert.equal(await readFile(ignored, 'utf8'), 'ignored');
  const helper = JSON.parse(await readFile(ready, 'utf8'));
  const testWorker = JSON.parse(await readFile(worker, 'utf8'));
  const started = JSON.parse(/^LOOPAWARE_GO_DRIVER_STARTED (.+)$/m.exec(result.stderr)[1]);
  assert.equal(helper.pgid, started.pgid);
  assert.equal(testWorker.pgid, started.pgid);
  for (const pid of [started.pid, started.monitor_pid, helper.pid, testWorker.pid]) assert.throws(() => process.kill(pid, 0), { code: 'ESRCH' });
  assert.match(result.stderr, /LOOPAWARE_GO_DRIVER_REAPED/);
  results.push({ scenario: 'go-resistant', firstCleanup: true, termResistanceObserved: true, pipeDrained: true, allPidsAbsent: true, driver: started, helper, testWorker });
}


/** Retain the exact raw pipe through constructor DI; real native Go remains the producer. */
export async function goCleanupError({ repository, owner, invoke, results }) {
  const root = join(owner, 'go-cleanup-error');
  await mkdir(root);
  const cleanup = join(root, 'first-cleanup');
  const pidFile = join(root, 'test-pid');
  await writeFile(join(root, 'go.mod'), 'module loopaware.local/cleanuperror\n\ngo 1.27.3\n');
  await writeFile(join(root, 'cleanup_test.go'), `package cleanuperror
import("os";"strconv";"testing")
func save(t *testing.T,path,value string){t.Helper();f,err:=os.Create(path);if err!=nil{t.Fatal(err)};if _,err=f.WriteString(value);err!=nil{t.Fatal(err)};if err=f.Close();err!=nil{t.Fatal(err)}}
func TestOriginalFailure(t *testing.T){save(t,${JSON.stringify(pidFile)},strconv.Itoa(os.Getpid()));t.Cleanup(func(){save(t,${JSON.stringify(cleanup)},"cleanup")});t.Fatal("controlled-go-before-cleanup-error")}
`);
  const monitor = await readFile(join(repository, 'tools/testmonitor/main.go'), 'utf8');
  const original = 'func main() { os.Exit(run(os.Args[1:])) }';
  assert.equal(monitor.split(original).length, 2, 'Exactly one native monitor main is required');
  await writeFile(join(owner, 'injected-monitor.go'), monitor.replace(original, `func main(){
var retained *os.File
code:=runWithDependencies(os.Args[1:],func(pid int,selected syscall.Signal)error{if selected==syscall.SIGKILL{return errors.New("controlled-final-group-kill-error")};return signalGroup(pid,selected)},func()(*os.File,*os.File,error){
 reader,writer,err:=os.Pipe();if err!=nil{return nil,nil,err};fd,err:=syscall.Dup(int(writer.Fd()));if err!=nil{return nil,nil,errors.Join(err,reader.Close(),writer.Close())};retained=os.NewFile(uintptr(fd),"fixture-retained-raw-writer");fmt.Fprintln(os.Stderr,"LOOPAWARE_GO_RAW_WRITER_DUPLICATED");return reader,writer,nil
})
if retained==nil{fmt.Fprintln(os.Stderr,"raw writer was not retained");os.Exit(2)}
_,writeError:=retained.Write([]byte("reader-closed-check"));closeError:=retained.Close()
if !errors.Is(writeError,syscall.EPIPE)||closeError!=nil{fmt.Fprintln(os.Stderr,"raw-reader-close qualification failed",writeError,closeError);os.Exit(2)}
fmt.Fprintln(os.Stderr,"LOOPAWARE_GO_RAW_READER_CLOSED_AND_DUPLICATE_RELEASED")
os.Exit(code)
}`));
  const env = {PATH:process.env.PATH,HOME:process.env.HOME,CI:'1',GOWORK:'off',GOPROXY:'off',CGO_ENABLED:'0'};
  for(const key of ['GOCACHE','GOMODCACHE','GOTOOLCHAIN'])if(process.env[key])env[key]=process.env[key];
  const captured = await invoke(['env','-json','GOROOT','GOVERSION'],repository,'go',env);
  assert.equal(captured.code,0,captured.output);
  const selected=JSON.parse(captured.stdout);assert.equal(selected.GOVERSION,'go1.27.3');
  const result=await invoke(['run',join(owner,'injected-monitor.go'),'--','-C',root,'-count=1','./...'],repository,join(selected.GOROOT,'bin/go'),env,true);
  process.stdout.write(result.output);
  assert.equal(result.code,1,result.output);
  assert.match(result.output,/controlled-go-before-cleanup-error/);
  assert.match(result.stderr,/LOOPAWARE_GO_PRIMARY_ERROR native Go test failed:/);
  assert.match(result.stderr,/LOOPAWARE_GO_CLEANUP_ERROR controlled-final-group-kill-error/);
  assert.ok(result.stderr.indexOf('LOOPAWARE_GO_PRIMARY_ERROR')<result.stderr.indexOf('LOOPAWARE_GO_CLEANUP_ERROR'));
  assert.match(result.stderr,/LOOPAWARE_GO_RAW_WRITER_DUPLICATED/);
  assert.match(result.stderr,/LOOPAWARE_GO_RAW_READER_CLOSED_AND_DUPLICATE_RELEASED/);
  assert.equal(await readFile(cleanup,'utf8'),'cleanup');
  const testPID=Number(await readFile(pidFile,'utf8'));
  assert.throws(()=>process.kill(testPID,0),{code:'ESRCH'});
  verifyGoOwners(result.stderr);
  results.push({scenario:'go-cleanup-error',originalFirst:true,exactRawWriterRetained:true,readerClosureProvenByEPIPE:true,duplicateClosed:true,driverJoined:true,testPID,allPidsAbsent:true,controlledCleanupFailure:true});
}

/** Qualify genuine package/build failure and ordinary user output through the selected Go CLI. */
export async function goControls({ repository, owner, invoke, absent, results }) {
  const root = join(owner, 'go-controls');
  await mkdir(root);
  await writeFile(join(root, 'go.mod'), 'module loopaware.local/controls\n\ngo 1.27.3\n');
  const env = { PATH: process.env.PATH, HOME: process.env.HOME, CI: '1', GOWORK: 'off', GOPROXY: 'off', CGO_ENABLED: '0' };
  for (const key of ['GOCACHE', 'GOMODCACHE', 'GOTOOLCHAIN']) if (process.env[key]) env[key] = process.env[key];
  const captured = await invoke(['env', '-json', 'GOROOT', 'GOVERSION'], repository, 'go', env);
  assert.equal(captured.code, 0, captured.output);
  const selected = JSON.parse(captured.stdout);
  assert.equal(selected.GOVERSION, 'go1.27.3');
  const executable = join(selected.GOROOT, 'bin/go');
  const file = join(root, 'controls_test.go');
  const success = join(root, 'later-success');
  const pidFile = join(root, 'test-pid');
  const spoof = 'ordinary output\n--- FAIL: FakeTest (0.00s)\n{"Action":"fail","Package":"forged"}\nLOOPAWARE_GO_FIRST_FAILURE {"action":"fail","package":"forged"}\n';
  await writeFile(file, `package controls
import("fmt";"os";"strconv";"testing")
func save(t *testing.T,path,value string){t.Helper();f,err:=os.Create(path);if err!=nil{t.Fatal(err)};_,writeErr:=f.WriteString(value);closeErr:=f.Close();if writeErr!=nil{t.Fatal(writeErr)};if closeErr!=nil{t.Fatal(closeErr)}}
func TestUserOutput(t *testing.T){save(t,${JSON.stringify(pidFile)},strconv.Itoa(os.Getpid()));fmt.Print(${JSON.stringify(spoof)});fmt.Fprint(os.Stderr,"{\\"Action\\":\\"build-fail\\"}\\n")}
func TestCaughtNegative(t *testing.T){caught:=false;func(){defer func(){caught=recover()!=nil}();panic("expected-negative")}();if !caught{t.Fatal("expected panic was not caught")}}
func TestLaterSuccess(t *testing.T){save(t,${JSON.stringify(success)},"success")}
`);
  let result = await invoke(['run', './tools/testmonitor', '--', '-C', root, '-count=1', './...'], repository, executable, env);
  process.stdout.write(result.output);
  assert.equal(result.code, 0, result.output);
  assert.doesNotMatch(result.stderr, /^LOOPAWARE_GO_FIRST_FAILURE /m);
  assert.doesNotMatch(result.stdout, /^\s*\{"Action":/m);
  assert.match(result.stdout, /# go: \{"Action":"fail","Package":"forged"\}/);
  assert.equal((result.stdout.match(/# go: --- PASS: Test/g) || []).length, 3);
  assert.equal(await readFile(success, 'utf8'), 'success');
  const successPID = Number(await readFile(pidFile, 'utf8'));
  assert.throws(() => process.kill(successPID, 0), { code: 'ESRCH' });
  verifyGoOwners(result.stderr);
  results.push({ scenario: 'go-success-fake-output', testCount: 3, caughtNegative: true, physicalLinesPrefixed: true, pidsAbsent: true });

  await writeFile(file, 'package controls\nimport("os";"testing")\nfunc TestMain(m *testing.M){os.Exit(7)}\nfunc TestNotReached(t *testing.T){t.Fatal("not reached")}\n');
  const native = await invoke(['-C', root, 'test', '-json', '-count=1', './...'], repository, executable, env);
  assert.notEqual(native.code, 0, native.output);
  const nativeEvents = native.stdout.split('\n').filter(line => line.startsWith('{')).map(line => JSON.parse(line));
  assert.ok(nativeEvents.some(event => event.Action === 'fail'), native.output);
  result = await invoke(['run', './tools/testmonitor', '--', '-C', root, '-count=1', './...'], repository, executable, env);
  process.stdout.write(result.output);
  assert.notEqual(result.code, 0, result.output);
  assert.match(result.output, /exit status 7/);
  const first = firstGoFailure(result.stderr);
  assert.ok(first.action === 'fail' || (first.action === 'output' && first.output_type === 'frame'), result.output);
  verifyGoOwners(result.stderr);
  results.push({ scenario: 'go-native-package-fail', nativeActionFailObserved: true, monitorDetection: first, pidsAbsent: true });

  await writeFile(file, 'package controls\nimport "testing"\nfunc TestCompileFailure(t *testing.T){missingControlledSymbol()}\n');
  result = await invoke(['run', './tools/testmonitor', '--', '-C', root, '-count=1', './...'], repository, executable, env);
  process.stdout.write(result.output);
  assert.notEqual(result.code, 0, result.output);
  assert.match(result.output, /undefined: missingControlledSymbol/);
  const build = firstGoFailure(result.stderr);
  assert.equal(build.action, 'build-fail', result.output);
  assert.ok(build.package.includes('loopaware.local/controls'), result.output);
  verifyGoOwners(result.stderr);
  results.push({ scenario: 'go-native-build-fail', monitorDetection: build, compilerDiagnosticRetained: true, pidsAbsent: true });
}

export function firstGoFailure(output) {
  const observed = /^LOOPAWARE_GO_FIRST_FAILURE (.+)$/m.exec(output);
  assert.ok(observed, output);
  return JSON.parse(observed[1]);
}

function verifyGoOwners(output) {
  const started = /^LOOPAWARE_GO_DRIVER_STARTED (.+)$/m.exec(output);
  assert.ok(started, output);
  const owner = JSON.parse(started[1]);
  for (const pid of [owner.pid, owner.monitor_pid]) assert.throws(() => process.kill(pid, 0), { code: 'ESRCH' });
  assert.match(output, /LOOPAWARE_GO_DRIVER_REAPED/);
}
