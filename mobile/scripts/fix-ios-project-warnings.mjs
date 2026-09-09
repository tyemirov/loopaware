// @ts-check
/// <reference types="node" />

import fs from "node:fs";
import path from "node:path";

const mobileRoot = path.resolve(import.meta.dirname, "..");
const xcodeProjectPath = path.join(mobileRoot, "ios", "LoopAware.xcodeproj", "project.pbxproj");

let projectFile;
try {
  projectFile = fs.openSync(xcodeProjectPath, fs.constants.O_RDWR | fs.constants.O_NOFOLLOW);
} catch (error) {
  if (error && typeof error === "object" && "code" in error && error.code === "ENOENT") {
    throw new Error("mobile_ios_project_missing: run expo prebuild --platform ios before fixing Xcode warnings", {
      cause: error,
    });
  }
  throw error;
}

try {
  if (!fs.fstatSync(projectFile).isFile()) {
    throw new Error("mobile_ios_project_invalid: Xcode project must be a regular file");
  }

  const originalProject = fs.readFileSync(projectFile, "utf8");
  let patchedProject = originalProject.replace(/^\s+"-lc\+\+",\n/gm, "");

  const metadataScript = `set -eu
if [ "$CONFIGURATION" = "Debug" ]; then
  exit 0
fi
SDKROOT="$(/usr/bin/xcrun --sdk macosx --show-sdk-path)"
export SDKROOT
exec /usr/bin/xcrun --sdk macosx swift "$PROJECT_DIR/../scripts/strip-release-metadata.swift" "$TARGET_BUILD_DIR/$INFOPLIST_PATH"
`;
  const phasePattern = /\n\t\t[0-9A-F]+ \/\* \[Expo Dev Launcher\] Strip Local Network Keys for Release \*\/ = \{\n[\s\S]*?\n\t\t\};/g;
  const phases = [...patchedProject.matchAll(phasePattern)];
  if (phases.length !== 1) {
    throw new Error("mobile_ios_metadata_phase_invalid: expected one release metadata phase");
  }
  const phase = phases[0][0];
  const inputPattern = /\t\t\tinputPaths = \([\s\S]*?\n\t\t\t\);/;
  const scriptPattern = /\t\t\tshellScript = "(?:\\[\s\S]|[^"\\])*";/;
  if (!inputPattern.test(phase) || !scriptPattern.test(phase)) {
    throw new Error("mobile_ios_metadata_phase_invalid: release metadata phase fields are missing");
  }
  const configuredPhase = phase
    .replace(/\n\t\t\talwaysOutOfDate = [^\n]*;/, "")
    .replace("isa = PBXShellScriptBuildPhase;", "isa = PBXShellScriptBuildPhase;\n\t\t\talwaysOutOfDate = 1;")
    .replace(inputPattern, '\t\t\tinputPaths = (\n\t\t\t\t"$(TARGET_BUILD_DIR)/$(INFOPLIST_PATH)",\n\t\t\t\t"$(PROJECT_DIR)/../scripts/strip-release-metadata.swift",\n\t\t\t);')
    .replace(scriptPattern, () => `\t\t\tshellScript = ${JSON.stringify(metadataScript)};`);
  patchedProject = patchedProject.replace(phase, () => configuredPhase);

  if (patchedProject !== originalProject) {
    const patchedBytes = Buffer.from(patchedProject, "utf8");
    let writtenBytes = 0;
    while (writtenBytes < patchedBytes.length) {
      const writeCount = fs.writeSync(
        projectFile,
        patchedBytes,
        writtenBytes,
        patchedBytes.length - writtenBytes,
        writtenBytes,
      );
      if (writeCount === 0) {
        throw new Error("mobile_ios_project_write_failed: Xcode project write made no progress");
      }
      writtenBytes += writeCount;
    }
    fs.ftruncateSync(projectFile, patchedBytes.length);
    fs.fsyncSync(projectFile);
  }
} finally {
  fs.closeSync(projectFile);
}

console.log("ios project warning fixes applied");
