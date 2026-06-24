import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const scriptDirectory = path.dirname(fileURLToPath(import.meta.url));
const packageDirectory = path.resolve(scriptDirectory, "..");
const distDirectory = path.join(packageDirectory, "dist");

fs.rmSync(distDirectory, { force: true, recursive: true });
