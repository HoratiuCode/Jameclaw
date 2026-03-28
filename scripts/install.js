#!/usr/bin/env node

const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");

const OWNER = process.env.JAMECLAW_NPM_REPO_OWNER || "sipeed";
const REPO = process.env.JAMECLAW_NPM_REPO_NAME || "jameclaw";
const VERSION = process.env.JAMECLAW_NPM_VERSION || process.env.npm_package_version || "latest";
const ROOT_DIR = path.resolve(__dirname, "..");
const BIN_DIR = path.join(ROOT_DIR, ".bin");

function resolveReleasePath() {
  if (
    VERSION === "latest" ||
    VERSION === "0.0.0-dev" ||
    VERSION.includes("-")
  ) {
    return "latest/download";
  }

  return `download/v${VERSION}`;
}

function resolveTarget() {
  const platformMap = {
    darwin: "Darwin",
    linux: "Linux",
    win32: "Windows",
  };

  const archMap = {
    x64: "amd64",
    arm64: "arm64",
    arm: "arm",
  };

  const platform = platformMap[process.platform];
  const arch = archMap[process.arch];

  if (!platform || !arch) {
    throw new Error(`unsupported platform: ${process.platform}/${process.arch}`);
  }

  if (platform === "Windows") {
    return {
      assetName: `jameclaw_${platform}_${arch}.zip`,
      binaryName: "jameclaw.exe",
      installedName: "jameclaw.exe",
      archiveType: "zip",
    };
  }

  return {
    assetName: `jameclaw_${platform}_${arch}.tar.gz`,
    binaryName: "jameclaw",
    installedName: "jameclaw",
    archiveType: "tar.gz",
  };
}

function getInstalledBinaryPath() {
  const target = resolveTarget();
  return path.join(BIN_DIR, target.installedName);
}

function shouldSkipPostinstall() {
  return process.env.JAMECLAW_NPM_SKIP_DOWNLOAD === "1";
}

function ensureInstalled(options = {}) {
  const quiet = options.quiet === true;
  const binaryPath = getInstalledBinaryPath();
  if (fs.existsSync(binaryPath)) {
    return binaryPath;
  }

  if (shouldSkipPostinstall()) {
    throw new Error(
      "native binary is missing and download is disabled by JAMECLAW_NPM_SKIP_DOWNLOAD=1",
    );
  }

  installBinary({ quiet });
  return binaryPath;
}

function installBinary(options = {}) {
  const quiet = options.quiet === true;
  const target = resolveTarget();
  const releasePath = resolveReleasePath();
  const downloadUrl = `https://github.com/${OWNER}/${REPO}/releases/${releasePath}/${target.assetName}`;

  fs.mkdirSync(BIN_DIR, { recursive: true });
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "jameclaw-npm-"));
  const archivePath = path.join(tempDir, target.assetName);

  if (!quiet) {
    console.error(`[jameclaw] downloading ${downloadUrl}`);
  }

  downloadFile(downloadUrl, archivePath);
  extractArchive(archivePath, tempDir, target.archiveType);

  const extractedBinary = path.join(tempDir, target.binaryName);
  if (!fs.existsSync(extractedBinary)) {
    throw new Error(`downloaded archive did not contain ${target.binaryName}`);
  }

  const installedBinary = getInstalledBinaryPath();
  fs.copyFileSync(extractedBinary, installedBinary);
  if (process.platform !== "win32") {
    fs.chmodSync(installedBinary, 0o755);
  }

  if (!quiet) {
    console.error(`[jameclaw] installed native binary to ${installedBinary}`);
  }
}

function downloadFile(url, destination) {
  const response = request(url);

  if (response.statusCode < 200 || response.statusCode >= 300) {
    throw new Error(`download failed with HTTP ${response.statusCode} for ${url}`);
  }

  fs.writeFileSync(destination, response.body);
}

function request(url) {
  const maxRedirects = 5;
  let currentUrl = url;

  for (let i = 0; i < maxRedirects; i += 1) {
    const response = requestOnce(currentUrl);

    if (
      response.statusCode >= 300 &&
      response.statusCode < 400 &&
      response.headers.location
    ) {
      currentUrl = new URL(response.headers.location, currentUrl).toString();
      continue;
    }

    return response;
  }

  throw new Error(`too many redirects while downloading ${url}`);
}

function requestOnce(url) {
  const result = { statusCode: 0, headers: {}, body: Buffer.alloc(0) };

  const response = spawnSync(
    process.execPath,
    [
      "-e",
      `
const https = require("node:https");
const url = process.argv[1];
https.get(url, (res) => {
  const chunks = [];
  res.on("data", (chunk) => chunks.push(chunk));
  res.on("end", () => {
    const payload = JSON.stringify({
      statusCode: res.statusCode,
      headers: res.headers,
      body: Buffer.concat(chunks).toString("base64"),
    });
    process.stdout.write(payload);
  });
}).on("error", (err) => {
  console.error(err.message);
  process.exit(1);
});
      `,
      url,
    ],
    { encoding: "utf8", maxBuffer: 1024 * 1024 * 200 },
  );

  if (response.status !== 0) {
    throw new Error(response.stderr.trim() || `failed to download ${url}`);
  }

  const parsed = JSON.parse(response.stdout);
  result.statusCode = parsed.statusCode;
  result.headers = parsed.headers || {};
  result.body = Buffer.from(parsed.body || "", "base64");
  return result;
}

function extractArchive(archivePath, destination, archiveType) {
  if (archiveType === "tar.gz") {
    const result = spawnSync("tar", ["-xzf", archivePath, "-C", destination], {
      stdio: "inherit",
    });
    if (result.status !== 0) {
      throw new Error("failed to extract tar.gz archive");
    }
    return;
  }

  if (archiveType === "zip") {
    const result = spawnSync("unzip", ["-o", archivePath, "-d", destination], {
      stdio: "inherit",
    });
    if (result.status !== 0) {
      throw new Error("failed to extract zip archive");
    }
    return;
  }

  throw new Error(`unsupported archive type: ${archiveType}`);
}

if (require.main === module && !shouldSkipPostinstall()) {
  try {
    ensureInstalled({ quiet: false });
  } catch (error) {
    console.error(`[jameclaw] ${error.message}`);
    process.exit(1);
  }
}

module.exports = {
  ensureInstalled,
  getInstalledBinaryPath,
  resolveReleasePath,
  resolveTarget,
};
