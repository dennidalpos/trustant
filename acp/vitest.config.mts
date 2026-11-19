// Derived from obsidian-agent-client (Apache-2.0), Copyright 2025-2026 RAIT-09.
// Modified by Nuvolaris Inc. to enable tests against the pinned Pi runtime.

import { defineConfig } from "vitest/config";
import { fileURLToPath } from "node:url";

// `obsidian` has no real module outside Obsidian, so alias it to a lightweight
// stub for unit tests. Only the pieces the tested pure functions need (Platform)
// are provided.
export default defineConfig({
	test: {
		environment: "node",
		include: [
			"test/**/*.test.ts",
			...(process.env.TRUSTANT_PI_AI_DIR
				? ["test/provider-stream.integration.ts"]
				: []),
		],
	},
	resolve: {
		alias: {
			obsidian: fileURLToPath(
				new URL("./test/stubs/obsidian.ts", import.meta.url),
			),
		},
	},
});
