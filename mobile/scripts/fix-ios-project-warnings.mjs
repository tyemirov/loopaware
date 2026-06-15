// @ts-check
/// <reference types="node" />

import fs from "node:fs";
import path from "node:path";

const mobileRoot = path.resolve(import.meta.dirname, "..");
const xcodeProjectPath = path.join(mobileRoot, "ios", "LoopAware.xcodeproj", "project.pbxproj");

if (!fs.existsSync(xcodeProjectPath)) {
  throw new Error("mobile_ios_project_missing: run expo prebuild --platform ios before fixing Xcode warnings");
}

const originalProject = fs.readFileSync(xcodeProjectPath, "utf8");
let patchedProject = originalProject.replace(/^\s+"-lc\+\+",\n/gm, "");

const devLauncherScriptPhasePattern =
  /(\n\t\t[0-9A-F]+ \/\* \[Expo Dev Launcher\] Strip Local Network Keys for Release \*\/ = {\n\t\t\tisa = PBXShellScriptBuildPhase;\n)(?!\t\t\talwaysOutOfDate = 1;\n)/;
patchedProject = patchedProject.replace(devLauncherScriptPhasePattern, "$1\t\t\talwaysOutOfDate = 1;\n");

if (patchedProject !== originalProject) {
  fs.writeFileSync(xcodeProjectPath, patchedProject);
}

console.log("ios project warning fixes applied");
