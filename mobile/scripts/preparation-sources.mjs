// @ts-check
import { spawnSync } from "node:child_process";

/** Select the current source files, including additions and deletions.
 * @param {string} repository
 * @returns {string[]}
 */
export function preparationSources(repository) {
  /** @param {string[]} options */
  const select = (options) => {
    const result = spawnSync("git", ["ls-files", ...options, "-z", "--", "mobile", "Makefile", ".mprlab/deploy/resources.yml"], { cwd: repository, encoding: "utf8" });
    if (result.status !== 0) throw new Error("Cannot select native preparation source inputs.");
    return result.stdout.split("\0").filter(Boolean);
  };
  const deleted = new Set(select(["--deleted"]));
  return [...new Set(select(["--cached", "--others", "--exclude-standard"]))].filter(name => !deleted.has(name) && !["mobile/prepared/", "mobile/android/", "mobile/ios/"].some(prefix => name.startsWith(prefix)));
}
