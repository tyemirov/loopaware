// @ts-check
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";

const packageDirectory = fileURLToPath(new URL("../", import.meta.url));

for (const scenario of ["missing README", "included source"]) {
  test(`package verification rejects ${scenario}`, () => {
    const temporaryRoot = fs.mkdtempSync(path.join(os.tmpdir(), "loopaware-package-test-"));
    const fixtureDirectory = path.join(temporaryRoot, "client");
    try {
      fs.cpSync(packageDirectory, fixtureDirectory, {
        recursive: true,
        filter: (source) => !["node_modules", "dist"].includes(path.basename(source)),
      });
      fs.symlinkSync(path.join(packageDirectory, "node_modules"), path.join(fixtureDirectory, "node_modules"), "dir");
      if (scenario === "missing README") {
        fs.rmSync(path.join(fixtureDirectory, "README.md"));
      } else {
        const manifestPath = path.join(fixtureDirectory, "package.json");
        const manifest = JSON.parse(fs.readFileSync(manifestPath, "utf8"));
        manifest.files.push("src");
        fs.writeFileSync(manifestPath, JSON.stringify(manifest));
      }

      const result = spawnSync(process.execPath, ["scripts/verify-package.mjs"], {
        cwd: fixtureDirectory,
        encoding: "utf8",
      });
      assert.ifError(result.error);
      assert.notEqual(result.status, 0, "invalid package must fail verification");
      assert.match(
        result.stderr,
        scenario === "missing README"
          ? /package_verify_failed: missing README\.md/
          : /package_verify_failed: unexpected src\/index\.tsx/,
      );
    } finally {
      fs.rmSync(temporaryRoot, { recursive: true, force: true });
    }
  });
}
