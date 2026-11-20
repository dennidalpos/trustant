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
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

import { ChildProcess, execFile, spawn } from "child_process";
import { mkdtemp, mkdir, readFile, rm, writeFile } from "fs/promises";
import { tmpdir } from "os";
import { join } from "path";
import { createInterface } from "readline";
import { PassThrough } from "stream";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
	claudeAcpPath,
	claudeCliPath,
	claudeInstallStatus,
	installClaude,
} from "../server/claude-install";
import { CLAUDE_PACKAGES } from "../server/claude-version";
import { buildAgentConfig, buildRuntime } from "../server/acp-host";
import { DEFAULT_CONFIG } from "../server/config-store";
import { AcpClient } from "../src/acp/acp-client";

vi.mock("child_process", async (importOriginal) => ({
	...(await importOriginal<typeof import("child_process")>()),
	execFile: vi.fn(),
	spawn: vi.fn(),
}));

let prefix: string;

beforeEach(async () => {
	vi.mocked(execFile).mockReset();
	vi.mocked(spawn).mockReset();
	prefix = await mkdtemp(join(tmpdir(), "claude-install-"));
	process.env.TRUACP_CLAUDE_PREFIX = prefix;
});

afterEach(async () => {
	delete process.env.TRUACP_CLAUDE_PREFIX;
	await rm(prefix, { recursive: true, force: true });
});

/** Lay out a fake install at the pinned versions. */
async function fakeInstall(): Promise<void> {
	for (const p of CLAUDE_PACKAGES) {
		const dir = join(prefix, "node_modules", p.name);
		await mkdir(dir, { recursive: true });
		await writeFile(
			join(dir, "package.json"),
			JSON.stringify({ name: p.name, version: p.version }),
		);
	}
	await mkdir(join(prefix, "node_modules", ".bin"), { recursive: true });
	await writeFile(claudeCliPath(), "");
	await writeFile(claudeAcpPath(), "");
}

describe("claude on-demand install", () => {
	it("reports not installed and not accepted on an empty prefix", () => {
		const s = claudeInstallStatus();
		expect(s.installed).toBe(false);
		expect(s.accepted).toBe(false);
		expect(s.termsUrl).toBe(
			"https://www.anthropic.com/legal/commercial-terms",
		);
	});

	it("refuses to install without explicit acceptance", async () => {
		await expect(installClaude(undefined)).rejects.toThrow(
			/Commercial Terms/,
		);
		await expect(installClaude("true")).rejects.toThrow();
		expect(execFile).not.toHaveBeenCalled();
		expect(claudeInstallStatus().accepted).toBe(false);
	});

	it("installs the pinned packages on demand after recording acceptance", async () => {
		vi.mocked(execFile).mockImplementationOnce((...call) => {
			const callback = call[3];
			if (typeof callback !== "function") {
				throw new Error("Missing npm callback");
			}
			void (async () => {
				const record = JSON.parse(
					await readFile(join(prefix, "license-accepted.json"), "utf8"),
				) as { specs: string[] };
				expect(record.specs).toEqual(claudeInstallStatus().specs);
				await fakeInstall();
			})().then(
				() => callback(null, "", ""),
				(error: Error) => callback(error, "", ""),
			);
			return new ChildProcess();
		});

		await expect(installClaude(true)).resolves.toMatchObject({
			accepted: true,
			installed: true,
		});
		expect(execFile).toHaveBeenCalledExactlyOnceWith(
			"npm",
			[
				"install",
				"--prefix",
				prefix,
				"--no-audit",
				"--no-fund",
				"--save-exact",
				"@anthropic-ai/claude-code@2.1.329",
				"@agentclientprotocol/claude-agent-acp@0.60.0",
			],
			expect.objectContaining({ cwd: prefix }),
			expect.any(Function),
		);
	});

	it("records acceptance and skips npm when already installed", async () => {
		await fakeInstall();
		const s = await installClaude(true);
		expect(s.installed).toBe(true);
		expect(s.accepted).toBe(true);
		expect(execFile).not.toHaveBeenCalled();
		const rec = JSON.parse(
			await readFile(join(prefix, "license-accepted.json"), "utf8"),
		) as { termsUrl: string; specs: string[] };
		expect(rec.termsUrl).toContain("commercial-terms");
		expect(rec.specs).toContain("@anthropic-ai/claude-code@2.1.329");
	});

	it("rejects the previous Claude Code pin", async () => {
		await fakeInstall();
		await writeFile(
			join(prefix, "node_modules", CLAUDE_PACKAGES[0].name, "package.json"),
			JSON.stringify({ version: "2.1.216" }),
		);
		expect(claudeInstallStatus().installed).toBe(false);
	});

	it("never spawns claude via npx, and refuses when not installed", async () => {
		const agent = {
			...DEFAULT_CONFIG.agents.claude,
			command: "npx",
			args: ["-y", "@agentclientprotocol/claude-agent-acp"],
		};
		expect(() => buildAgentConfig(agent, prefix)).toThrow(/not installed/);
		await fakeInstall();
		const cfg = buildAgentConfig(agent, prefix);
		expect(cfg.command).toBe(claudeAcpPath());
		expect(cfg.args).toEqual([]);
		expect(cfg.env?.CLAUDE_CODE_EXECUTABLE).toBe(claudeCliPath());
	});

	it("initializes ACP with the CLI from the pinned install", async () => {
		await fakeInstall();
		const stdin = new PassThrough();
		const stdout = new PassThrough();
		const child = Object.assign(new ChildProcess(), {
			stdin,
			stdout,
			kill: vi.fn(),
		});
		const messages = createInterface({ input: stdin });
		messages.once("line", (line) => {
			const request = JSON.parse(line);
			if (request.method !== "initialize") {
				stdout.end();
				return;
			}
			stdout.write(
				JSON.stringify({
					jsonrpc: "2.0",
					id: request.id,
					result: {
						protocolVersion: request.params.protocolVersion,
						agentInfo: { name: "claude-fixture", version: "0.60.0" },
					},
				}) + "\n",
			);
		});
		vi.mocked(spawn).mockReturnValueOnce(child);
		const client = new AcpClient({
			...buildRuntime(DEFAULT_CONFIG, "fixture"),
			resolveSecret: () => "",
		});
		try {
			const config = buildAgentConfig(DEFAULT_CONFIG.agents.claude, prefix);
			await expect(client.initialize(config)).resolves.toMatchObject({
				agentInfo: { name: "claude-fixture" },
			});
			expect(spawn).toHaveBeenCalledExactlyOnceWith(
				expect.any(String),
				expect.any(Array),
				expect.objectContaining({
					env: expect.objectContaining({
						CLAUDE_CODE_EXECUTABLE: claudeCliPath(),
					}),
					cwd: prefix,
				}),
			);
		} finally {
			await client.disconnect();
			messages.close();
			stdin.end();
			stdout.end();
		}
	});
});
