// @ts-check
import assert from "node:assert/strict";
import { readFileSync, realpathSync } from "node:fs";
import { createRequire } from "node:module";
import path from "node:path";
import { spawnSync } from "node:child_process";

const require = createRequire(import.meta.url);
const depthLimit = 100;
const deep = "{".repeat(4000) + "a" + "}".repeat(4000);
/** @param {number} depth @param {string} [open] @param {string} [close] */
const nested = (depth, open = "{", close = "}") => open.repeat(depth) + "a" + close.repeat(depth);
/** @typedef {{type: string, value?: string, nodes?: Ast[], parent?: Ast}} Ast */
/** @param {number} depth @returns {Ast} */
const astAtDepth = (depth) => {
  /** @type {Ast} */
  let node = { type: "text", value: "a" };
  for (let i = 0; i < depth; i++) node = { type: "brace", nodes: [node] };
  return { type: "root", nodes: [node] };
};

const consumers = [
  ["@expo/metro-file-map", "micromatch"],
  ["@expo/metro", "metro-file-map", "micromatch"],
  ["metro-file-map", "micromatch"],
  ["@react-native-community/cli-clean", "fast-glob", "micromatch"],
];
const roots = new Set();
for (const chain of consumers) {
  let consumerRequire = require;
  for (const dependency of chain) consumerRequire = createRequire(consumerRequire.resolve(`${dependency}/package.json`));
  const braces = consumerRequire("braces");
  const identityPath = consumerRequire.resolve("braces/package.json");
  const micromatch = consumerRequire("./");
  const label = chain.join(" -> ");
  assert.throws(() => braces.compile(deep), /Input depth.*exceeds max depth/, `${label}: deep compilation must reject before stack exhaustion`);

  for (const method of ["parse", "compile", "expand", "stringify"]) {
    for (const pattern of [nested(101), nested(101, "(", ")"), "{".repeat(101), "(".repeat(101), "{(".repeat(51)]) {
      assert.throws(() => braces[method](pattern), /Input depth.*exceeds max depth/);
    }
    for (const maxDepth of [Infinity, NaN, false, 10000]) {
      assert.throws(() => braces[method](nested(101), { maxDepth }), /exceeds max depth/);
    }
    assert.doesNotThrow(() => braces[method](nested(depthLimit)));
    assert.doesNotThrow(() => braces[method](nested(depthLimit, "(", ")")));
    assert.throws(() => braces[method](nested(3), { maxDepth: 2 }), /exceeds max depth/);
    assert.doesNotThrow(() => braces[method](nested(2), { maxDepth: 2 }));
  }

  for (const method of ["compile", "expand", "stringify"]) {
    assert.doesNotThrow(() => braces[method](braces.parse(nested(depthLimit))));
    assert.throws(() => braces[method](astAtDepth(101)), /AST depth.*exceeds max depth/);
    assert.throws(() => braces[method](astAtDepth(4000)), /AST depth.*exceeds max depth/);
    for (const maxDepth of [Infinity, NaN, false, 10000]) {
      assert.throws(() => braces[method](astAtDepth(101), { maxDepth }), /AST depth.*exceeds max depth/);
    }
    assert.doesNotThrow(() => braces[method](astAtDepth(depthLimit)));
    const cycle = astAtDepth(0);
    cycle.nodes = [cycle];
    assert.throws(() => braces[method](cycle), /AST depth.*exceeds max depth/);
  }
  const parentCycle = spawnSync(process.execPath, ["-e", `
    const braces = require(require('node:path').dirname(process.argv[1]));
    const node = { type: 'paren', nodes: [] };
    node.parent = node;
    require('node:assert/strict').throws(
      () => braces.expand({ type: 'root', nodes: [node] }),
      /AST parent depth.*exceeds max depth/
    );
  `, identityPath], { timeout: 2000, encoding: "utf8" });
  assert.equal(parentCycle.error, undefined, `${label}: parent traversal did not terminate`);
  assert.equal(parentCycle.status, 0, parentCycle.stderr);
  assert.doesNotThrow(() => braces.parse("\\{".repeat(1000)));
  assert.doesNotThrow(() => braces.parse('"' + "{".repeat(1000) + '"'));
  assert.doesNotThrow(() => braces.parse("{a}".repeat(1000)));
  assert.deepEqual(braces.expand("src/{app,lib}/{a,b}.js"), ["src/app/a.js", "src/app/b.js", "src/lib/a.js", "src/lib/b.js"]);
  assert.deepEqual(braces.expand("{01..03}"), ["01", "02", "03"]);
  assert.equal(braces.compile("src/{app,lib}/*.js"), "src/(app|lib)/*.js");
  assert.equal(braces.stringify(braces.parse("src/{app,lib}/*.js")), "src/{app,lib}/*.js");
  for (const pattern of ["{foo}", "${a,b}", "{{a,b}}"]) {
    for (const escapeInvalid of [true, false]) assert.equal(braces.stringify(pattern, { escapeInvalid }), pattern);
  }
  assert.deepEqual(micromatch.braceExpand("{a,b}.js"), ["a.js", "b.js"]);
  assert.deepEqual(micromatch(["a.js", "b.ts", "c.txt"], "*.{js,ts}"), ["a.js", "b.ts"]);
  assert.throws(() => micromatch.braces(deep), /Input depth.*exceeds max depth/);
  assert.throws(() => micromatch.braceExpand(deep), /Input depth.*exceeds max depth/);
  const identity = JSON.parse(readFileSync(identityPath, "utf8"));
  assert.equal(identity.name, "@loopaware/braces");
  assert.equal(identity.version, "3.0.3-loopaware.1");
  roots.add(realpathSync(path.dirname(identityPath)));
}
assert.equal(roots.size, 1, "All mobile consumers must use one corrected Braces implementation");
const cliRequire = createRequire(require.resolve("@react-native-community/cli-clean/package.json"));
const glob = cliRequire("fast-glob");
assert.throws(() => glob.generateTasks(deep), /Input depth.*exceeds max depth/);
assert.deepEqual(glob.generateTasks("src/{app,lib}/*.js").flatMap(/** @param {{positive: string[]}} task */ (task) => task.positive), ["src/app/*.js", "src/lib/*.js"]);
console.log("mobile_braces_security.ok: Metro and CLI consumers reject excessive depth and preserve valid glob operations");
