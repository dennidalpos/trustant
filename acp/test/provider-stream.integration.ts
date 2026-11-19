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

import { once } from "node:events";
import { readFile } from "node:fs/promises";
import { createServer } from "node:http";
import { resolve } from "node:path";
import { pathToFileURL } from "node:url";
import { beforeAll, describe, expect, it } from "vitest";

interface StreamEvent {
	type: string;
	delta?: string;
	reason?: string;
	error?: { errorMessage?: string };
}

const model = {
	id: "stream-fixture",
	name: "Stream fixture",
	api: "openai-completions",
	provider: "fixture",
	reasoning: false,
	input: ["text"],
	cost: { input: 0, output: 0, cacheRead: 0, cacheWrite: 0 },
	contextWindow: 4096,
	maxTokens: 64,
};
const context = {
	messages: [
		{ role: "user", content: "Synthetic stream test", timestamp: 0 },
	],
};
let stream: (
	requestModel: typeof model & { baseUrl: string },
	requestContext: typeof context,
	options: { apiKey: string; signal: AbortSignal },
) => AsyncIterable<StreamEvent>;

beforeAll(async () => {
	const directory = process.env.TRUSTANT_PI_AI_DIR;
	if (!directory)
		throw new Error(
			"Set TRUSTANT_PI_AI_DIR to the pinned Pi AI package directory",
		);

	const pin = (await readFile("pi.version", "utf8")).match(
		/^@earendil-works\/pi-ai@([^\s]+)$/m,
	)?.[1];
	const manifest = JSON.parse(
		await readFile(resolve(directory, "package.json"), "utf8"),
	);
	if (
		!pin ||
		manifest.name !== "@earendil-works/pi-ai" ||
		manifest.version !== pin
	) {
		throw new Error(
			`Expected @earendil-works/pi-ai@${pin} from pi.version`,
		);
	}

	const entry = manifest.exports["./api/*"].import.replace(
		"*",
		"openai-completions",
	);
	const provider = await import(
		/* @vite-ignore */ pathToFileURL(resolve(directory, entry)).href
	);
	stream = provider.stream;
});

function sseChunk(
	delta: { content?: string },
	finishReason: "stop" | null,
): string {
	return `data: ${JSON.stringify({
		id: "fixture",
		object: "chat.completion.chunk",
		created: 0,
		model: model.id,
		choices: [{ index: 0, delta, finish_reason: finishReason }],
	})}\n\n`;
}

async function readStream(trailer: string): Promise<StreamEvent[]> {
	const server = createServer((request, response) => {
		request.resume();
		if (
			request.method !== "POST" ||
			request.url !== "/v1/chat/completions"
		) {
			response.writeHead(404).end();
			return;
		}
		response.writeHead(200, { "Content-Type": "text/event-stream" });
		response.write(sseChunk({ content: "partial text" }, null));
		response.end(trailer);
	});
	await once(server.listen(0, "127.0.0.1"), "listening");
	try {
		const address = server.address();
		if (!address || typeof address === "string")
			throw new Error("Fixture has no TCP address");

		const events: StreamEvent[] = [];
		const baseUrl = `http://127.0.0.1:${address.port}/v1`;
		for await (const event of stream({ ...model, baseUrl }, context, {
			apiKey: "fixture-key",
			signal: AbortSignal.timeout(3000),
		})) {
			events.push(event);
		}
		return events;
	} finally {
		const closed = once(server, "close");
		server.close();
		await closed;
	}
}

describe("Trustant's pinned Pi OpenAI-compatible stream", () => {
	it("finishes successfully after a valid finish_reason", async () => {
		const events = await readStream(
			sseChunk({}, "stop") + "data: [DONE]\n\n",
		);

		expect(events).toContainEqual(
			expect.objectContaining({
				type: "text_delta",
				delta: "partial text",
			}),
		);
		expect(events.at(-1)).toMatchObject({ type: "done", reason: "stop" });
		expect(events.some(({ type }) => type === "error")).toBe(false);
	});

	it.each([
		["the response ends early", ""],
		["the response only sends [DONE]", "data: [DONE]\n\n"],
	])(
		"reports an explicit error when %s without finish_reason",
		async (_scenario, trailer) => {
			const events = await readStream(trailer);

			expect(events).toContainEqual(
				expect.objectContaining({
					type: "text_delta",
					delta: "partial text",
				}),
			);
			expect(events.at(-1)).toMatchObject({
				type: "error",
				reason: "error",
				error: {
					errorMessage: expect.stringContaining(
						"Stream ended without finish_reason",
					),
				},
			});
			expect(events.some(({ type }) => type === "done")).toBe(false);
		},
	);
});
