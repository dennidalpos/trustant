// Derived from obsidian-agent-client (Apache-2.0), Copyright 2025-2026 RAIT-09.
// Modified by Nuvolaris Inc. to enable tests against the pinned Pi runtime.

import { defineConfig } from "vitest/config";

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
});
