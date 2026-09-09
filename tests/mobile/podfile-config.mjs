// @ts-check
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { copyFileSync, mkdirSync, mkdtempSync, readFileSync, rmSync, symlinkSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, resolve } from "node:path";
import test from "node:test";

const mobile = resolve(import.meta.dirname, "../../mobile/prepared");

for (const condition of ["valid", "missing", "malformed", "unavailable-prebuilt-service"]) {
  test(`CocoaPods evaluates ${condition} native properties`, () => {
    const fixture = mkdtempSync(resolve(tmpdir(), "loopaware-podfile-"));
    try {
      const record = JSON.parse(readFileSync(resolve(mobile, "native-preparation.json"), "utf8"));
      for (const name of Object.keys(record.files)) {
        mkdirSync(dirname(resolve(fixture, name)), { recursive: true });
        copyFileSync(resolve(mobile, name), resolve(fixture, name));
      }
      symlinkSync(resolve(mobile, "node_modules"), resolve(fixture, "node_modules"), "dir");
      const properties = resolve(fixture, "ios/Podfile.properties.json");
      if (condition === "missing") rmSync(properties);
      if (condition === "malformed") writeFileSync(properties, "{invalid JSON");
      /** @type {NodeJS.ProcessEnv} */
      const env = { ...process.env, CI: "1", NODE_ENV: "test", EXPO_NO_TELEMETRY: "1", npm_config_offline: "true" };
      if (condition === "unavailable-prebuilt-service") {
        const network = resolve(fixture, "unavailable-prebuilt-service.rb");
        writeFileSync(network, `module UnavailablePrebuiltService
  def \`(command)
    raise IOError, "prebuilt artifact service unavailable" if command.include?("curl") && command.include?("-Iw")
    super
  end
end
Kernel.prepend(UnavailablePrebuiltService)
`);
        env.RCT_USE_RN_DEP = "1";
        env.RCT_USE_PREBUILT_RNCORE = "1";
        env.EXPO_USE_PRECOMPILED_MODULES = "1";
        env.RUBYOPT = `${process.env.RUBYOPT || ""} -r${network}`;
      }
      const result = spawnSync("pod", ["ipc", "podfile", resolve(fixture, "ios/Podfile")], {
        cwd: resolve(fixture, "ios"),
        env,
        encoding: "utf8",
      });
      assert.ifError(result.error);
      const output = `${result.stdout}\n${result.stderr}`;
      if (condition === "valid" || condition === "unavailable-prebuilt-service") {
        assert.equal(result.status, 0, output);
        assert.match(result.stdout, /LoopAware/);
        if (condition === "unavailable-prebuilt-service") {
          assert.match(output, /Building from source: true/);
          assert.doesNotMatch(output, /Building from source: false/);
        }
      } else {
        assert.notEqual(result.status, 0, `CocoaPods must reject ${condition} Podfile.properties.json`);
        assert.match(output, /Podfile\.properties\.json/);
      }
    } finally {
      rmSync(fixture, { recursive: true, force: true });
    }
  });
}
