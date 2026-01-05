import * as cp from "child_process";
import * as util from "util";
import * as vscode from "vscode";
import { getConfig } from "./config";

const exec = util.promisify(cp.exec);

/**
 * Version compatibility information
 */
export interface VersionInfo {
  cli: string | null;
  extension: string;
  compatible: boolean;
  message: string;
}

/**
 * Compatibility matrix: extension version -> minimum CLI version
 * Versions are synced across the ecosystem.
 */
const COMPATIBILITY_MATRIX: Record<string, string> = {
  "0.0.1-alpha": "0.0.1-alpha",
};

/**
 * Current extension version (synced with VERSION file)
 */
const EXTENSION_VERSION = "0.0.1-alpha";

/**
 * Parses a version string into comparable parts
 * Handles formats like "0.0.1", "0.0.1-rc1", "v0.0.1"
 */
function parseVersion(version: string): {
  major: number;
  minor: number;
  patch: number;
  prerelease: string | null;
} {
  // Remove 'v' prefix if present
  const cleaned = version.replace(/^v/, "").trim();

  // Split into version and prerelease
  const [versionPart, prerelease] = cleaned.split("-");
  const [major, minor, patch] = versionPart.split(".").map(Number);

  return {
    major: major || 0,
    minor: minor || 0,
    patch: patch || 0,
    prerelease: prerelease || null,
  };
}

/**
 * Compares two versions
 * Returns: negative if a < b, 0 if a == b, positive if a > b
 */
function compareVersions(a: string, b: string): number {
  const vA = parseVersion(a);
  const vB = parseVersion(b);

  // Compare major.minor.patch
  if (vA.major !== vB.major) return vA.major - vB.major;
  if (vA.minor !== vB.minor) return vA.minor - vB.minor;
  if (vA.patch !== vB.patch) return vA.patch - vB.patch;

  // Handle prereleases: release > prerelease
  if (vA.prerelease === null && vB.prerelease !== null) return 1;
  if (vA.prerelease !== null && vB.prerelease === null) return -1;

  // Both are prereleases or both are releases
  if (vA.prerelease && vB.prerelease) {
    return vA.prerelease.localeCompare(vB.prerelease);
  }

  return 0;
}

/**
 * Gets the CLI version by running `palace --version`
 */
async function getCLIVersion(): Promise<string | null> {
  const config = getConfig();
  const bin = config.binaryPath;

  try {
    const { stdout } = await exec(`${bin} --version`);
    // Expected format: "palace version 0.0.1-rc1" or similar
    const match = stdout.match(/(\d+\.\d+\.\d+(?:-[\w.]+)?)/);
    return match ? match[1] : null;
  } catch (error) {
    return null;
  }
}

/**
 * Checks version compatibility between CLI and extension
 */
export async function checkVersionCompatibility(): Promise<VersionInfo> {
  const cliVersion = await getCLIVersion();
  const extensionVersion = EXTENSION_VERSION;
  const minCLIVersion = COMPATIBILITY_MATRIX[extensionVersion];

  if (!cliVersion) {
    return {
      cli: null,
      extension: extensionVersion,
      compatible: false,
      message: "Could not determine CLI version. Is palace installed?",
    };
  }

  if (!minCLIVersion) {
    // No compatibility info for this extension version (development)
    return {
      cli: cliVersion,
      extension: extensionVersion,
      compatible: true,
      message: `CLI ${cliVersion}, Extension ${extensionVersion} (no compatibility check)`,
    };
  }

  const isCompatible = compareVersions(cliVersion, minCLIVersion) >= 0;

  if (isCompatible) {
    return {
      cli: cliVersion,
      extension: extensionVersion,
      compatible: true,
      message: `CLI ${cliVersion}, Extension ${extensionVersion} (compatible)`,
    };
  }

  return {
    cli: cliVersion,
    extension: extensionVersion,
    compatible: false,
    message: `CLI ${cliVersion} is older than required ${minCLIVersion}. Please upgrade the CLI.`,
  };
}

/**
 * Shows a warning if versions are incompatible
 * Call this on extension activation
 */
export async function warnIfIncompatible(): Promise<void> {
  const info = await checkVersionCompatibility();

  if (!info.compatible) {
    const action = info.cli ? "Upgrade CLI" : "Install CLI";

    const selection = await vscode.window.showWarningMessage(
      `Mind Palace: ${info.message}`,
      action,
      "Ignore"
    );

    if (selection === action) {
      // Open installation docs
      vscode.env.openExternal(
        vscode.Uri.parse(
          "https://github.com/koksalmehmet/mind-palace#install-no-go-toolchain-required"
        )
      );
    }
  }
}
