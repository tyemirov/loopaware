// @ts-check
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { copyFileSync, mkdirSync, mkdtempSync, readFileSync, rmSync, symlinkSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, resolve } from "node:path";
import test from "node:test";
import { copyPreparedPublicInputs } from "../../mobile/scripts/record-store-preparation.mjs";

const mobile = resolve(import.meta.dirname, "../../mobile/prepared");

test("Gradle evaluates the prepared Android release version", async (t) => {
  const fixture = mkdtempSync(resolve(tmpdir(), "loopaware-android-config-"));
  try {
    copyPreparedPublicInputs(mobile, fixture);
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


test("prepared Android launcher supplies its compiler policy and preserves caller arguments", () => {
  const fixture = mkdtempSync(resolve(tmpdir(), "loopaware-android-policy-"));
  try {
    copyPreparedPublicInputs(mobile, fixture);
    const tools = resolve(fixture, "controlled-java"); mkdirSync(tools);
    const java = resolve(tools, "java"), log = resolve(fixture, "java-arguments.json");
    copyFileSync(process.execPath, java);
    writeFileSync(java, `#!${process.execPath}
require('node:fs').writeFileSync(${JSON.stringify(log)}, JSON.stringify(process.argv.slice(2)));
`);
    const callerArguments = ["--no-daemon", ":app:bundleRelease", "--max-workers=2", "--parallel", "-PreactNativeArchitectures=arm64-v8a", "-Dorg.gradle.jvmargs=-Xmx2048m -XX:MaxMetaspaceSize=1g"];
    const properties = resolve(fixture, "android/gradle.properties");
    for (const pass of [1, 2]) {
      if (pass === 2) {
        writeFileSync(properties, readFileSync(properties, "utf8").replace(/^org\.gradle\.jvmargs=.*$/m, "org.gradle.jvmargs=-Xmx3072m -XX:MaxMetaspaceSize=512m") + "\norg.gradle.workers.max=7\n");
        const prepared = spawnSync(process.execPath, [resolve(fixture, "scripts/prepare-android-build-policy.mjs")], { cwd: fixture, encoding: "utf8" });
        assert.equal(prepared.status, 0, prepared.stderr);
        assert.match(readFileSync(properties, "utf8"), /^org\.gradle\.jvmargs=-Xmx3072m -XX:MaxMetaspaceSize=1g$/m);
        assert.match(readFileSync(properties, "utf8"), /^org\.gradle\.workers\.max=7$/m);
      }
      const launched = spawnSync("/bin/sh", [resolve(fixture, "android/gradlew"), ...callerArguments], {
        cwd: resolve(fixture, "android"), encoding: "utf8", env: { PATH: tools + ":/usr/bin:/bin", JAVA_HOME: "" },
      });
      assert.equal(launched.status, 0, launched.stderr);
      const javaArguments = JSON.parse(readFileSync(log, "utf8"));
      assert.deepEqual(javaArguments.slice(javaArguments.indexOf("-jar") + 2), ["-Pkotlin.compiler.execution.strategy=in-process", ...callerArguments]);
      assert.match(readFileSync(properties, "utf8"), /-XX:MaxMetaspaceSize=1g/);
    }
  } finally {
    rmSync(fixture, { recursive: true, force: true });
  }
});
