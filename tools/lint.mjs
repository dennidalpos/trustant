/*
 * Copyright 2025-2026 Nuvolaris Inc
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published
 * by the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import { spawnSync } from "node:child_process";
import { existsSync, mkdtempSync, rmSync, rmdirSync, statSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

process.chdir(fileURLToPath(new URL("..", import.meta.url)));

let failed = false;

function check(name, command, args, options = {}) {
  console.log(`\n--- ${name} ---`);
  const result = spawnSync(command, args, { stdio: "inherit", ...options });
  if (result.error) {
    console.error(`${name}: ${result.error.message}. See CONTRIBUTING.md for setup.`);
  } else if (result.signal) {
    console.error(`${name}: terminated by ${result.signal}.`);
  }
  if (result.status !== 0) failed = true;
}

const inventory = spawnSync("git", ["ls-files", "-z", "--cached", "--others", "--exclude-standard"], {
  encoding: "utf8",
});
if (inventory.status !== 0) {
  console.error(inventory.error?.message ?? inventory.stderr);
  process.exit(1);
}
const paths = [...new Set(inventory.stdout.split("\0"))]
  .filter(path => path && statSync(path, { throwIfNoEntry: false })?.isFile())
  .map(path => `./${path}`);
const select = pattern => paths.filter(path => pattern.test(path));

const goEnvironment = { ...process.env, GOOS: "linux", CGO_ENABLED: "0" };
if (existsSync("_build.txt")) {
  check("Go vet", "go", ["vet", "."], { env: goEnvironment });
} else {
  // Supply the go:embed input without creating or replacing build metadata in the checkout.
  const scratch = mkdtempSync(join(tmpdir(), "trustant-lint-"));
  try {
    const metadata = join(scratch, "_build.txt");
    const overlay = join(scratch, "overlay.json");
    writeFileSync(metadata, "Version: lint\n");
    writeFileSync(overlay, JSON.stringify({ Replace: { [resolve("_build.txt")]: metadata } }));
    check("Go vet", "go", ["vet", "-overlay", overlay, "."], { env: goEnvironment });
  } finally {
    rmSync(join(scratch, "_build.txt"), { force: true });
    rmSync(join(scratch, "overlay.json"), { force: true });
    rmdirSync(scratch);
  }
}

check("Biome", process.execPath, [
  "node_modules/@biomejs/biome/bin/biome", "lint",
  ...select(/\.(?:[cm]?[jt]sx?|html|css|jsonc?|ya?ml)$/),
]);
const python = select(/\.py$|^\.\/image\/(?:mcp-s3|redis-mcp)$/);
check("Ruff", "ruff", ["check", "--no-cache", ...python]);

const shell = select(/\.sh$|^\.\/image\/ollama$/);
check("ShellCheck", "shellcheck", ["--severity=error", "--format=gcc", ...shell]);
check("Hadolint", "hadolint", ["--failure-threshold", "error", ...select(/\/Dockerfile(?:\.[^/]+)?$/)]);

check("TOML syntax", process.platform === "win32" ? "python" : "python3", [
  "-c",
  "import sys, tomllib\nfor path in sys.argv[1:]:\n    with open(path, 'rb') as source:\n        tomllib.load(source)",
  ...select(/\.toml$/),
]);

const powerShellCheck = `
  $ErrorActionPreference = 'Stop'
  Import-Module PSScriptAnalyzer -RequiredVersion 1.25.0
  $paths = [Console]::In.ReadToEnd() | ConvertFrom-Json
  $findings = foreach ($path in $paths) {
    Invoke-ScriptAnalyzer -Path $path -Severity Error,ParseError
  }
  if ($findings) {
    $findings | Format-Table RuleName,ScriptName,Line,Message -Wrap | Out-Host
    exit 1
  }
`;
check("PSScriptAnalyzer", "pwsh", ["-NoProfile", "-NonInteractive", "-Command", powerShellCheck], {
  input: JSON.stringify(select(/\.ps(?:1|m1|d1)$/)),
  stdio: ["pipe", "inherit", "inherit"],
});

process.exitCode = failed ? 1 : 0;
