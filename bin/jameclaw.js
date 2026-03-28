#!/usr/bin/env node

const { spawnSync } = require("node:child_process");
const { ensureInstalled, getInstalledBinaryPath } = require("../scripts/install");

function main() {
  try {
    ensureInstalled({ quiet: false });
  } catch (error) {
    console.error(`[jameclaw] ${error.message}`);
    process.exit(1);
  }

  const binaryPath = getInstalledBinaryPath();
  const result = spawnSync(binaryPath, process.argv.slice(2), {
    stdio: "inherit",
  });

  if (result.error) {
    console.error(`[jameclaw] failed to start native binary: ${result.error.message}`);
    process.exit(1);
  }

  process.exit(result.status === null ? 1 : result.status);
}

main();
