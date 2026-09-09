// @ts-check
import { readdir, readFile, writeFile } from "node:fs/promises";
import { resolve, join, relative } from "node:path";
import { createHash } from "node:crypto";
import { preparationSources } from "./preparation-sources.mjs";

const repository = resolve(process.argv[2] ?? resolve(import.meta.dirname, "../.."));
const prepared = join(repository, "mobile/prepared");
const excludedDirectories = new Set(["node_modules", "Pods", "build", "DerivedData", ".gradle", ".kotlin", ".cxx", ".expo", "xcuserdata", "project.xcworkspace", ".git"]);
const excludedFiles = new Set(["native-preparation.json", ".DS_Store", "local.properties", ".xcode.env.local"]);

/** Collect prepared files without dependency caches or private inputs.
 * @param {string} directory
 * @returns {Promise<string[]>}
 */
async function collect(directory) {
  const files = [];
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    if (excludedDirectories.has(entry.name) || excludedFiles.has(entry.name) || entry.name.startsWith(".env") || /\.(keystore|jks|p12|p8|mobileprovision|xcuserstate)$/.test(entry.name)) continue;
    const path = join(directory, entry.name);
    if (entry.isSymbolicLink()) throw new Error(`Prepared native input is a symbolic link: ${path}`);
    if (entry.isDirectory()) files.push(...await collect(path));
    else if (entry.isFile()) files.push(path);
  }
  return files;
}

/** Bind file contents to their source-relative paths.
 * @param {string} root
 * @param {string[]} paths
 */
async function digests(root, paths) {
  /** @type {Record<string, string>} */
  const files = {};
  for (const path of paths.sort()) {
    const bytes = await readFile(path);
    if (bytes.length) files[relative(root, path)] = createHash("sha256").update(bytes).digest("hex");
  }
  return { schema_version: 1, files };
}

const inputs = preparationSources(repository);
await writeFile(join(prepared, "source-preparation.json"), `${JSON.stringify(await digests(repository, inputs.map(name => join(repository, name))), null, 2)}\n`);
const record = await digests(prepared, await collect(prepared));
for (const name of ["package.json", "package-lock.json", "app.config.snapshot.json", "scripts/verify-store-preparation.mjs", "android/gradlew", "android/app/build.gradle", "ios/Podfile", "ios/Podfile.lock", "ios/LoopAware.xcworkspace/contents.xcworkspacedata"]) {
  if (!record.files[name]) throw new Error(`Native preparation requires ${name}.`);
}
await writeFile(join(prepared, "native-preparation.json"), `${JSON.stringify(record, null, 2)}\n`);
console.info(`Recorded ${Object.keys(record.files).length} prepared native files.`);
