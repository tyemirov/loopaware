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

  const devLauncherScriptPhasePattern =
    /(\n\t\t[0-9A-F]+ \/\* \[Expo Dev Launcher\] Strip Local Network Keys for Release \*\/ = {\n\t\t\tisa = PBXShellScriptBuildPhase;\n)(?!\t\t\talwaysOutOfDate = 1;\n)/;
  patchedProject = patchedProject.replace(devLauncherScriptPhasePattern, "$1\t\t\talwaysOutOfDate = 1;\n");

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
