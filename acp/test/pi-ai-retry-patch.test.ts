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

import { spawnSync } from "node:child_process";
import { cpSync } from "node:fs";
import { mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { pathToFileURL } from "node:url";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

// Synthetic error messages shaped like the OpenAI SDK APIError text that
// pi-ai stores in AssistantMessage.errorMessage (`<status> <body error>`).
// The session wording follows trustant/trustant#206; the exact Ollama Cloud
// sentence is not reproduced here.
const SESSION_LIMIT =
	'429 "you (synthetic) have reached your session usage limit, please wait or upgrade to continue"';
const TRANSIENT = ["429 Too Many Requests", "429 rate limit exceeded"];

// Minimal synthetic stand-in for pi-ai 0.82.0 dist/utils/retry.js: same
// classifier shape and the reviewed patch anchor, no upstream code.
const FIXTURE_RETRY = `const NON_RETRYABLE = new RegExp([
    "GoUsageLimitError",
    "Monthly usage limit reached",
    "quota exceeded",
].join("|"), "i");
const RETRYABLE = new RegExp(["rate.?limit", "too many requests", "429"].join("|"), "i");
export function isRetryableAssistantError(message) {
    if (message.stopReason !== "error" || !message.errorMessage) return false;
    if (NON_RETRYABLE.test(message.errorMessage)) return false;
    return RETRYABLE.test(message.errorMessage);
}
`;

async function extractPatchScript(): Promise<string> {
	const setup = await readFile(join(process.cwd(), "setup.sh"), "utf8");
	const match = setup.match(
		/<<'PI_AI_RETRY_PATCH'\n([\s\S]*?)\nPI_AI_RETRY_PATCH\n/,
	);
	if (!match) throw new Error("PI_AI_RETRY_PATCH heredoc not found");
	return match[1];
}

function classify(retryJs: string, messages: string[]): boolean[] {
	const program = `
const { isRetryableAssistantError } = await import(${JSON.stringify(pathToFileURL(retryJs).href)});
const messages = ${JSON.stringify(messages)};
console.log(JSON.stringify(messages.map((errorMessage) =>
	isRetryableAssistantError({ stopReason: "error", errorMessage }))));
`;
	const run = spawnSync(
		process.execPath,
		["--input-type=module", "-e", program],
		{ encoding: "utf8" },
	);
	if (run.status !== 0) throw new Error(run.stderr);
	return JSON.parse(run.stdout);
}

describe("setup.sh pi-ai session usage limit patch", () => {
	let root: string;
	let script: string;
	let nodeModules: string;

	const runPatch = () =>
		spawnSync(process.execPath, [script, nodeModules, "0.82.0"], {
			encoding: "utf8",
		});

	async function installFixture(
		parent: string,
		version: string,
		retry = FIXTURE_RETRY,
	): Promise<string> {
		const dir = join(parent, "@earendil-works", "pi-ai");
		await mkdir(join(dir, "dist", "utils"), { recursive: true });
		await writeFile(
			join(dir, "package.json"),
			JSON.stringify({ name: "@earendil-works/pi-ai", version, type: "module" }),
		);
		const retryJs = join(dir, "dist", "utils", "retry.js");
		await writeFile(retryJs, retry);
		return retryJs;
	}

	beforeEach(async () => {
		root = await mkdtemp(join(tmpdir(), "pi-ai-retry-patch-"));
		script = join(root, "patch.cjs");
		await writeFile(script, await extractPatchScript());
		nodeModules = join(root, "lib", "node_modules");
	});

	afterEach(async () => {
		await rm(root, { recursive: true, force: true });
	});

	it("stops retrying the session limit and keeps transient 429 retryable", async () => {
		const topLevel = await installFixture(nodeModules, "0.82.0");
		const nested = await installFixture(
			join(nodeModules, "@earendil-works", "pi-coding-agent", "node_modules"),
			"0.82.0",
		);
		const other = await installFixture(
			join(nodeModules, "pi-mcp-adapter", "node_modules"),
			"0.74.2",
		);
		const cases = [SESSION_LIMIT, ...TRANSIENT];

		// Before the patch both kinds of 429 are retried.
		expect(classify(topLevel, cases)).toEqual([true, true, true]);

		const first = runPatch();
		expect(first.status, first.stderr).toBe(0);
		expect(first.stdout.match(/Applied pi-ai 0\.82\.0/g)).toHaveLength(2);
		for (const retryJs of [topLevel, nested]) {
			expect(classify(retryJs, cases)).toEqual([false, true, true]);
		}
		// Other pi-ai versions are out of scope and stay byte-identical.
		expect(await readFile(other, "utf8")).toBe(FIXTURE_RETRY);

		const patched = await readFile(topLevel, "utf8");
		const second = runPatch();
		expect(second.status, second.stderr).toBe(0);
		expect(second.stdout.match(/already applied/g)).toHaveLength(2);
		expect(await readFile(topLevel, "utf8")).toBe(patched);
	});

	it("fails closed when the reviewed anchor is missing", async () => {
		await installFixture(
			nodeModules,
			"0.82.0",
			FIXTURE_RETRY.replace('"Monthly usage limit reached",', ""),
		);
		const run = runPatch();
		expect(run.status).not.toBe(0);
		expect(run.stderr).toContain("expected one patch target");
	});

	it("fails when no pi-ai copy matches the pinned version", async () => {
		await installFixture(nodeModules, "0.81.0");
		const run = runPatch();
		expect(run.status).not.toBe(0);
		expect(run.stderr).toContain("no @earendil-works/pi-ai 0.82.0 found");
	});

	// Optional check against the integrity-verified upstream package, extracted
	// locally (for example under the ignored tmp/ directory).
	it.skipIf(!process.env.TRUSTANT_PI_AI_PACKAGE)(
		"patches the pinned upstream pi-ai retry classifier",
		async () => {
			const dir = join(nodeModules, "@earendil-works", "pi-ai");
			cpSync(process.env.TRUSTANT_PI_AI_PACKAGE as string, dir, {
				recursive: true,
			});
			const retryJs = join(dir, "dist", "utils", "retry.js");
			const cases = [SESSION_LIMIT, ...TRANSIENT];
			expect(classify(retryJs, cases)).toEqual([true, true, true]);
			const run = runPatch();
			expect(run.status, run.stderr).toBe(0);
			expect(classify(retryJs, cases)).toEqual([false, true, true]);
		},
	);
});
