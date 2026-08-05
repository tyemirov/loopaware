import { spawnSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const scriptDirectory = path.dirname(fileURLToPath(import.meta.url));
const packageDirectory = path.resolve(scriptDirectory, "..");
const temporaryRoot = fs.mkdtempSync(path.join(os.tmpdir(), "loopaware-react-native-"));

try {
  const packOutput = run("npm", ["pack", "--json", "--pack-destination", temporaryRoot], packageDirectory);
  const packResults = JSON.parse(packOutput);
  const packageResult = packResults[0];
  if (!packageResult) {
    throw new Error("package_verify_failed: npm pack returned no package metadata");
  }

  const packageFiles = new Set(packageResult.files.map((fileEntry) => fileEntry.path));
  requirePackedFile(packageFiles, "dist/index.js");
  requirePackedFile(packageFiles, "dist/index.d.ts");
  requirePackedFile(packageFiles, "README.md");
  requirePackedFile(packageFiles, "LICENSE");
  rejectPackedFile(packageFiles, "src/index.tsx");

  const tarballPath = path.join(temporaryRoot, packageResult.filename);
  const consumerDirectory = path.join(temporaryRoot, "consumer");
  fs.mkdirSync(path.join(consumerDirectory, "types"), { recursive: true });
  fs.writeFileSync(path.join(consumerDirectory, "package.json"), consumerPackageJSON());
  fs.writeFileSync(path.join(consumerDirectory, "tsconfig.json"), consumerTSConfig());
  fs.writeFileSync(path.join(consumerDirectory, "App.tsx"), consumerAppSource());
  fs.copyFileSync(path.join(packageDirectory, "types", "react.d.ts"), path.join(consumerDirectory, "types", "react.d.ts"));
  fs.copyFileSync(path.join(packageDirectory, "types", "react-native.d.ts"), path.join(consumerDirectory, "types", "react-native.d.ts"));

  run("npm", ["install", tarballPath, "--ignore-scripts", "--no-audit", "--no-fund", "--legacy-peer-deps"], consumerDirectory);
  run(resolveTypeScriptBinary(), ["--noEmit", "-p", "tsconfig.json"], consumerDirectory);
  console.log("react-native package verification passed");
} finally {
  fs.rmSync(temporaryRoot, { force: true, recursive: true });
}

function requirePackedFile(packageFiles, filePath) {
  if (!packageFiles.has(filePath)) {
    throw new Error(`package_verify_failed: missing ${filePath}`);
  }
}

function rejectPackedFile(packageFiles, filePath) {
  if (packageFiles.has(filePath)) {
    throw new Error(`package_verify_failed: unexpected ${filePath}`);
  }
}

function resolveTypeScriptBinary() {
  const binaryName = process.platform === "win32" ? "tsc.cmd" : "tsc";
  const binaryPath = path.join(packageDirectory, "node_modules", ".bin", binaryName);
  if (!fs.existsSync(binaryPath)) {
    throw new Error("package_verify_failed: run npm ci in clients/react-native before package verification");
  }
  return binaryPath;
}

function run(command, args, cwd) {
  const result = spawnSync(command, args, {
    cwd,
    encoding: "utf8",
    env: process.env,
  });
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    throw new Error(
      [
        `command_failed: ${command} ${args.join(" ")}`,
        result.stdout.trim(),
        result.stderr.trim(),
      ]
        .filter(Boolean)
        .join("\n")
    );
  }
  return result.stdout;
}

function consumerPackageJSON() {
  return `${JSON.stringify(
    {
      name: "loopaware-react-native-consumer-check",
      private: true,
      type: "module",
    },
    null,
    2
  )}\n`;
}

function consumerTSConfig() {
  return `${JSON.stringify(
    {
      compilerOptions: {
        jsx: "react",
        lib: ["ES2022", "DOM"],
        module: "ES2022",
        moduleResolution: "Bundler",
        strict: true,
        target: "ES2022",
        typeRoots: ["./types"],
      },
      include: ["App.tsx", "types/**/*.d.ts"],
    },
    null,
    2
  )}\n`;
}

function consumerAppSource() {
  return `import React from "react";
import {
  LoopAwareFeedbackButton,
  LoopAwareProvider,
  submitLoopAwareFeedback,
  type LoopAwareConfig,
} from "@loopaware/react-native";

const config: LoopAwareConfig = {
  siteId: "site-id",
  mobileClientId: "mobile-client-id",
  apiOrigin: "https://loopaware.mprlab.com",
  app: {
    platform: "ios",
    applicationId: "com.example.app",
    version: "1.2.3",
    build: "44",
    environment: "production",
  },
};

export function CheckoutFeedback() {
  return (
    <LoopAwareProvider {...config}>
      <LoopAwareFeedbackButton
        screen={{ name: "Checkout", path: "/checkout/payment" }}
        context={{ step: "payment", plan: "pro" }}
      />
    </LoopAwareProvider>
  );
}

export async function submitFeedbackFromCustomUI() {
  await submitLoopAwareFeedback(config, {
    contact: "person@example.com",
    message: "The checkout button is confusing.",
    sentiment: "sad",
    screen: { name: "Checkout", path: "/checkout/payment" },
    context: { step: "payment" },
  });
}
`;
}
