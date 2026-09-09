// @ts-check
import { readFile, realpath } from "node:fs/promises";
import { join, relative, isAbsolute } from "node:path";
import { spawn } from "node:child_process";

/**
 * @typedef {{status: number | null, stdout: string}} SigningToolResult
 * @typedef {(name: string, args: string[], environment?: NodeJS.ProcessEnv) => Promise<SigningToolResult>} SigningTool
 */

/** Execute an Android signing tool with private environment inputs.
 * @param {string} name
 * @param {string[]} args
 * @param {NodeJS.ProcessEnv} [sourceEnvironment]
 * @returns {Promise<SigningToolResult>}
 */
export async function executeSigningTool(name, args, sourceEnvironment = process.env) {
  const environment = { ...sourceEnvironment };
  delete environment.GH_TOKEN;
  delete environment.GITHUB_TOKEN;
  return new Promise((resolve, reject) => {
    const child = spawn(name, args, { env: environment, stdio: ["ignore", "pipe", "pipe"] });
    let stdout = "";
    child.stdout.on("data", chunk => { stdout += chunk; });
    child.stderr.resume();
    child.once("error", () => reject(new Error(`Android signing tool could not start: ${name}`)));
    child.once("close", status => resolve({ status, stdout }));
  });
}

/** Require a private input file inside the selected repository directory.
 * @param {string} repositoryRoot
 * @param {string | undefined} value
 * @param {string} name
 */
export async function repositorySigningFile(repositoryRoot, value, name) {
    if (!value) throw new Error(`Signing requires ${name} in configs/.env.loopaware.`);
    const root = await realpath(repositoryRoot);
    let path;
    try {
        path = await realpath(isAbsolute(value) ? value : join(root, value));
        const inside = relative(root, path);
        const privatePath = relative(join(root, 'configs/signing'), path);
        if (inside.startsWith('..') || isAbsolute(inside) || privatePath.startsWith('..') || isAbsolute(privatePath)) throw new Error('outside repository');
        const content = await readFile(path);
        if (!content.length) throw new Error('empty input');
    } catch {
        throw new Error(`Signing input ${name} must identify a nonempty file under configs/signing/.`);
    }
    return path;
}
