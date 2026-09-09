// @ts-check
import { readFile, realpath } from "node:fs/promises";
import { resolve, relative, isAbsolute } from "node:path";
import { createHash } from "node:crypto";

/** Verify a recorded set of source or native file contents.
 * @param {string} root
 * @param {string} manifest
 */
async function verify(root, manifest) {
  const record = JSON.parse(await readFile(resolve(manifest), "utf8"));
  if (record.schema_version !== 1 || !record.files || Object.keys(record.files).length === 0) throw new Error("Native preparation record is invalid.");
  for (const [path, expected] of Object.entries(record.files)) {
    const target = await realpath(resolve(root, path));
    const inside = relative(root, target);
    if (isAbsolute(path) || inside.startsWith("../") || isAbsolute(inside) || inside !== path) throw new Error(`Native preparation path is invalid: ${path}`);
    const actual = createHash("sha256").update(await readFile(target)).digest("hex");
    if (actual !== expected) throw new Error(`Stale native preparation: ${path}`);
  }
}
await verify(await realpath(resolve(process.cwd(), "../..")), "source-preparation.json");
await verify(await realpath(process.cwd()), "native-preparation.json");
console.info("Prepared native files and source match the selected release inputs.");
