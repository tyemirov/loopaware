import { spawnSync } from "node:child_process";
import { lstatSync, readFileSync } from "node:fs";
import { createRequire } from "node:module";
import path from "node:path";

const require = createRequire(import.meta.url);
const packagePath = require.resolve("image-size/package.json");
const packageRoot = path.dirname(packagePath);
const packageIdentity = JSON.parse(readFileSync(packagePath, "utf8"));

if (packageIdentity.name !== "@loopaware/image-size" || packageIdentity.version !== "1.2.2") {
  throw new Error("mobile_image_size_security.invalid_package_identity");
}
if (lstatSync(packageRoot).isSymbolicLink()) {
  throw new Error("mobile_image_size_security.linked_package_not_sealed");
}

const imageSizePath = require.resolve("image-size");

function requireRejectedInput(name, bytes, expectedError) {
  const source = `const imageSize = require(${JSON.stringify(imageSizePath)}); imageSize(Buffer.from(process.argv[1], "base64"));`;
  const result = spawnSync(process.execPath, ["-e", source, bytes.toString("base64")], {
    encoding: "utf8",
    timeout: 1000,
  });

  if (result.error) {
    throw new Error(`mobile_image_size_security.${name}_did_not_terminate: ${result.error.message}`);
  }
  if (result.status === 0 || !result.stderr.includes(expectedError)) {
    throw new Error(`mobile_image_size_security.${name}_was_not_rejected`);
  }
}

const invalidIcns = Buffer.alloc(16);
invalidIcns.write("icns", 0, "ascii");
invalidIcns.writeUInt32BE(invalidIcns.length, 4);
invalidIcns.write("ic07", 8, "ascii");
invalidIcns.writeUInt32BE(0, 12);
requireRejectedInput("invalid_icns", invalidIcns, "Invalid ICNS entry length");

function box(type, payload, declaredSize = payload.length + 8) {
  const result = Buffer.alloc(payload.length + 8);
  result.writeUInt32BE(declaredSize, 0);
  result.write(type, 4, "ascii");
  payload.copy(result, 8);
  return result;
}

const invalidJxl = Buffer.concat([
  box("JXL ", Buffer.alloc(4)),
  box("ftyp", Buffer.from("jxl \0\0\0\0", "binary")),
  box("jxlp", Buffer.alloc(0), 0),
]);
requireRejectedInput("invalid_jxl", invalidJxl, "No codestream found in JXL container");

console.log("mobile_image_size_security.ok");
