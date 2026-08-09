import { spawnSync } from "node:child_process";
import { lstatSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { createRequire } from "node:module";
import os from "node:os";
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

const tiffProbeSource = `
const fs = require("node:fs");
const imageSize = require(process.argv[1]);
const tiffPath = process.argv[2];
const mode = process.argv[3];
const originalOpen = fs.openSync;
const originalRead = fs.readSync;
const originalClose = fs.closeSync;
const opened = [];
const closed = [];
let reads = 0;
fs.openSync = (...arguments_) => {
  const descriptor = originalOpen(...arguments_);
  opened.push(descriptor);
  return descriptor;
};
fs.readSync = (...arguments_) => {
  reads += 1;
  if (mode === "read-failure" && reads === 2) {
    throw new Error("injected TIFF read failure");
  }
  return originalRead(...arguments_);
};
fs.closeSync = (descriptor) => {
  closed.push(descriptor);
  return originalClose(descriptor);
};
fs.statSync = () => {
  throw new Error("TIFF reader inspected the path after opening");
};
try {
  const result = imageSize(tiffPath);
  if (mode !== "success" || result.width !== 320 || result.height !== 240) {
    throw new Error("TIFF descriptor success probe failed");
  }
} catch (error) {
  if (mode !== "read-failure" || !String(error.message).includes("injected TIFF read failure")) {
    throw error;
  }
}
if (opened.length !== closed.length || opened.some((descriptor) => !closed.includes(descriptor))) {
  throw new Error("TIFF reader leaked a descriptor");
}
`;

const tiffProbeRoot = mkdtempSync(path.join(os.tmpdir(), "loopaware-image-size-tiff-"));
try {
  const tiffPath = path.join(tiffProbeRoot, "probe.tiff");
  const tiff = Buffer.alloc(1042);
  tiff.write("II", 0, "ascii");
  tiff.writeUInt16LE(42, 2);
  tiff.writeUInt32LE(8, 4);
  tiff.writeUInt16LE(2, 8);
  tiff.writeUInt16LE(256, 10);
  tiff.writeUInt16LE(4, 12);
  tiff.writeUInt32LE(1, 14);
  tiff.writeUInt32LE(320, 18);
  tiff.writeUInt16LE(257, 22);
  tiff.writeUInt16LE(4, 24);
  tiff.writeUInt32LE(1, 26);
  tiff.writeUInt32LE(240, 30);
  writeFileSync(tiffPath, tiff);

  for (const mode of ["success", "read-failure"]) {
    const result = spawnSync(process.execPath, ["-e", tiffProbeSource, imageSizePath, tiffPath, mode], {
      encoding: "utf8",
      timeout: 1000,
    });
    if (result.error) {
      throw new Error(`mobile_image_size_security.tiff_${mode}_did_not_terminate: ${result.error.message}`);
    }
    if (result.status !== 0) {
      throw new Error(`mobile_image_size_security.tiff_${mode}_failed: ${result.stderr}`);
    }
  }
} finally {
  rmSync(tiffProbeRoot, { force: true, recursive: true });
}

console.log("mobile_image_size_security.ok");
