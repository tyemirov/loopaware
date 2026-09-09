// @ts-check
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { copyFileSync, mkdirSync, mkdtempSync, readFileSync, rmSync, symlinkSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, resolve } from "node:path";
import test from "node:test";

const mobile = resolve(import.meta.dirname, "../../mobile/prepared");

test("Gradle evaluates the prepared Android release version", async (t) => {
  const fixture = mkdtempSync(resolve(tmpdir(), "loopaware-android-config-"));
  try {
    const install = spawnSync("npm", ["ci", "--include=dev"], { cwd: mobile, encoding: "utf8" });
    assert.ifError(install.error);
    assert.equal(install.status, 0, `${install.stdout}\n${install.stderr}`);
    const record = JSON.parse(readFileSync(resolve(mobile, "native-preparation.json"), "utf8"));
    for (const name of Object.keys(record.files)) {
      mkdirSync(dirname(resolve(fixture, name)), { recursive: true });
      copyFileSync(resolve(mobile, name), resolve(fixture, name));
    }
    symlinkSync(resolve(mobile, "node_modules"), resolve(fixture, "node_modules"), "dir");
    const initScript = resolve(fixture, "verify-version.gradle");
    const outputFile = resolve(fixture, "version.json");
    writeFileSync(initScript, `gradle.afterProject { project ->
    if (project.path == ':app') {
        project.tasks.register('verifyReleaseVersion') {
            doLast {
                def config = project.android.defaultConfig
                new File(System.getenv('ANDROID_CONFIG_TEST_OUTPUT')).text = groovy.json.JsonOutput.toJson([
                    versionCode: config.versionCode,
                    versionName: config.versionName
                ])
            }
        }
    }
}
`);
    for (const scenario of [
      { name: "release timestamp", code: "211417622", version: "2026.9.9" },
      { name: "next release", code: "211504022", version: "2026.9.10" },
      { name: "missing code", code: undefined, version: "2026.9.9" },
      { name: "invalid code", code: "invalid-version-code", version: "2026.9.9" },
    ]) {
      await t.test(scenario.name, () => {
        /** @type {NodeJS.ProcessEnv} */
        const env = {
          ...process.env, CI: "1", NODE_ENV: "production", EXPO_NO_TELEMETRY: "1",
          MPRLAB_MOBILE_VERSION_NAME: scenario.version,
          ANDROID_CONFIG_TEST_OUTPUT: outputFile,
          LOOPAWARE_ANDROID_KEYSTORE: resolve(fixture, "unused-test.keystore"),
          LOOPAWARE_ANDROID_STORE_PASSWORD: "unused-test-password",
          LOOPAWARE_ANDROID_KEY_ALIAS: "unused-test-alias",
          LOOPAWARE_ANDROID_KEY_PASSWORD: "unused-test-password",
        };
        delete env.MPRLAB_MOBILE_VERSION_CODE;
        if (scenario.code !== undefined) env.MPRLAB_MOBILE_VERSION_CODE = scenario.code;
        rmSync(outputFile, { force: true });
        const result = spawnSync("/bin/sh", ["./gradlew", "--no-daemon", "--console=plain", "--init-script", initScript, ":app:verifyReleaseVersion"], {
          cwd: resolve(fixture, "android"), env, encoding: "utf8",
        });
        assert.ifError(result.error);
        const output = `${result.stdout}\n${result.stderr}`;
        if (scenario.code === undefined || scenario.code === "invalid-version-code") {
          assert.notEqual(result.status, 0, `Gradle must reject ${scenario.name}`);
          assert.match(output, /app\/build\.gradle/);
          assert.match(output, scenario.code === undefined ? /null|MPRLAB_MOBILE_VERSION_CODE/ : /invalid-version-code/);
        } else {
          assert.equal(result.status, 0, output);
          assert.deepEqual(JSON.parse(readFileSync(outputFile, "utf8")), {
            versionCode: Number(scenario.code), versionName: scenario.version,
          });
        }
      });
    }
  } finally {
    rmSync(fixture, { recursive: true, force: true });
  }
});
