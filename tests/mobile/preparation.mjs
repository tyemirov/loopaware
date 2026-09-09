// @ts-check
import assert from "node:assert/strict";
import { mkdtemp, mkdir, copyFile, writeFile, readFile, rename, rm } from "node:fs/promises";
import { join, resolve } from "node:path";
import { tmpdir } from "node:os";
import { spawnSync } from "node:child_process";

const temporary = await mkdtemp(join(tmpdir(), "loopaware native preparation "));
let repository = join(temporary, "first checkout");
try {
  await mkdir(join(repository, "mobile/scripts"), { recursive: true });
  const initialization = spawnSync("git", ["init", "--quiet", repository], { encoding: "utf8" });
  assert.equal(initialization.status, 0, initialization.stderr);
  await writeFile(join(repository, "mobile/App.tsx"), "export const application = 'fixture';\n");
  await writeFile(join(repository, "Makefile"), "fixture:\n\ttrue\n");
  const deletedSource = join(repository, "mobile/obsolete.mjs");
  await writeFile(deletedSource, "obsolete\n");
  assert.equal(spawnSync("git", ["add", "mobile/obsolete.mjs"], { cwd: repository }).status, 0);
  await rm(deletedSource);
  const prepared = join(repository, "mobile/prepared");
  for (const name of ["package.json", "package-lock.json", "app.config.snapshot.json", "android/gradlew", "android/app/build.gradle", "ios/Podfile", "ios/Podfile.lock", "ios/LoopAware.xcworkspace/contents.xcworkspacedata"]) {
    await mkdir(resolve(prepared, name, ".."), { recursive: true });
    await writeFile(join(prepared, name), "prepared fixture\n");
  }
  await mkdir(join(prepared, "scripts"));
  for (const name of ["record-store-preparation.mjs", "verify-store-preparation.mjs", "preparation-sources.mjs"]) {
    await copyFile(resolve("mobile/scripts", name), join(repository, "mobile/scripts", name));
  }
  await copyFile(resolve("mobile/scripts/verify-store-preparation.mjs"), join(prepared, "scripts/verify-store-preparation.mjs"));
  const record = spawnSync(process.execPath, [resolve("mobile/scripts/record-store-preparation.mjs"), repository], { encoding: "utf8" });
  assert.equal(record.status, 0, record.stderr);
  const manifest = JSON.parse(await readFile(join(prepared, "native-preparation.json"), "utf8"));
  assert.ok(manifest.files["android/app/build.gradle"]);
  assert.ok(manifest.files["source-preparation.json"]);
  const destination = join(temporary, "relocated checkout");
  await rename(repository, destination);
  repository = destination;
  const verify = () => spawnSync(process.execPath, ["scripts/verify-store-preparation.mjs"], { cwd: join(repository, "mobile/prepared"), encoding: "utf8" });
  let result = verify();
  assert.equal(result.status, 0, result.stderr);
  await writeFile(join(repository, "mobile/App.tsx"), "export const application = 'changed';\n");
  result = verify();
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /Stale native preparation: mobile\/App.tsx/);
  await writeFile(join(repository, "mobile/App.tsx"), "export const application = 'fixture';\n");
  await writeFile(join(repository, "mobile/prepared/android/app/build.gradle"), "changed native build\n");
  result = verify();
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /Stale native preparation: android\/app\/build.gradle/);
  console.info("Prepared inputs survive relocation and reject changed source or native files.");
} finally {
  await rm(temporary, { recursive: true, force: true });
}
