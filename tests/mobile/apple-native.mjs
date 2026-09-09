// @ts-check
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
const prepared = "mobile/prepared/ios";
const project = readFileSync(`${prepared}/LoopAware.xcodeproj/project.pbxproj`, "utf8");
assert.ok(project.includes("CODE_SIGN_STYLE = Automatic;"), "The prepared Apple target must use automatic signing.");
assert.equal(readFileSync(`${prepared}/ci_scripts/ci_post_clone.sh`, "utf8"), readFileSync("mobile/cloud/ci_post_clone.sh", "utf8"));
assert.ok(project.includes(".xcode.env.local"), "The bundle phase must load its declared cloud Node environment.");
console.info("Prepared Apple inputs contain automatic signing and the cloud dependency hook.");
