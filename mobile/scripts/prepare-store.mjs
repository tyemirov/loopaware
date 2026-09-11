// @ts-check
import { cp, mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { resolve, join } from "node:path";
import { spawnSync } from "node:child_process";
import { preparationSources } from "./preparation-sources.mjs";

const repository = resolve(import.meta.dirname, "../..");
const mobile = join(repository, "mobile");
const prepared = join(mobile, "prepared");
/** @type {NodeJS.ProcessEnv} */
const environment = { ...process.env, CI: "1", NODE_ENV: "production" };

/** Run a preparation command and preserve its failure status.
 * @param {string} command
 * @param {string[]} argumentsList
 */
function run(command, argumentsList) {
  const result = spawnSync(command, argumentsList, { cwd: prepared, env: environment, stdio: "inherit" });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error(`Native preparation ${command} failed with status ${result.status}.`);
}

const sources = preparationSources(repository).filter(name => name.startsWith("mobile/") && name !== "mobile/.gitignore");
await rm(prepared, { recursive: true, force: true });
await mkdir(prepared, { recursive: true });
for (const source of sources) {
  const destination = join(prepared, source.slice("mobile/".length));
  await mkdir(resolve(destination, ".."), { recursive: true });
  await cp(join(repository, source), destination);
}
const packagePath = join(prepared, "package.json");
const packageJSON = JSON.parse(await readFile(packagePath, "utf8"));
delete packageJSON.dependencies["expo-dev-client"];
await writeFile(packagePath, `${JSON.stringify(packageJSON, null, 2)}\n`);
run("npm", ["install", "--package-lock-only", "--ignore-scripts", "--include=dev"]);
run("npm", ["ci", "--include=dev"]);
run("npx", ["--no-install", "expo", "prebuild", "--platform", "all", "--no-install"]);
// Preserve the same generated text bytes in Git and in the preparation record.
for (const name of ["android/gradlew.bat", "android/settings.gradle"]) {
  const path = join(prepared, name);
  const text = await readFile(path, "utf8");
  await writeFile(path, text.replaceAll("\r\n", "\n").replace(/[ \t]+$/gm, ""));
}
run("node", ["scripts/fix-ios-project-warnings.mjs"]);
run("npx", ["--no-install", "pod-install", "ios"]);
run("node", ["--input-type=commonjs", "-e", "require('node:fs').writeFileSync('app.config.snapshot.json', JSON.stringify(require('./app.config.js'), null, 2)+'\\n')"]);
run("node", [join(mobile, "scripts/record-store-preparation.mjs"), repository]);
