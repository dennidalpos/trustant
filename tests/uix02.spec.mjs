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

import { test, expect } from "@playwright/test";
import { extname } from "node:path";
import { fileURLToPath } from "node:url";

const fileTypes = {
  ".html": "text/html",
  ".css": "text/css",
  ".js": "application/javascript",
  ".svg": "image/svg+xml",
  ".png": "image/png",
  ".ico": "image/x-icon",
};

async function serveFixture(page, origin, assets, api) {
  await page.route("**/*", async (route) => {
    const url = new URL(route.request().url());
    if (url.origin !== origin) return route.abort();
    if (url.pathname.startsWith("/api/")) return api(route, url.pathname);

    const resource = url.pathname === "/" ? "/index.html" : url.pathname;
    const contentType = fileTypes[extname(resource)];
    if (!contentType) return route.fulfill({ status: 404 });
    return route.fulfill({
      path: fileURLToPath(new URL(`.${resource}`, assets)),
      contentType,
    });
  });
}

const privateConfiguration = {
  provider: "private",
  base_url: "http://models.example.test/v1",
  api_key: "dummy",
  models: { "saved-model:27b": { maxToken: 64000, maxOutput: 8192 } },
  pi: { default: "saved-model:27b" },
};

async function openPrivateConfiguration(page, discovery) {
  const origin = "http://trustant.example.test";
  await serveFixture(page, origin, new URL("../web/", import.meta.url), (route, path) => {
    if (path === "/api/discover-models") return route.fulfill(discovery);
    return route.fulfill({ json: path === "/api/configuration" ? privateConfiguration : {} });
  });
  await page.goto(`${origin}/configure.html`);
}

test("Private AI explains stale model choices when discovery fails", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await openPrivateConfiguration(page, { status: 502, json: { error: "Endpoint unreachable" } });

  const model = page.getByRole("combobox", { name: "Default Model" });
  const status = page.locator("#privateModelDiscoveryStatus");
  await expect(model).toHaveValue("saved-model:27b");
  await expect(status).toBeVisible();
  await expect(status).toContainText("The saved Private AI model list may be out of date.");
  await expect(model).toHaveAccessibleDescription(/Endpoint unreachable/);
});

test("Private AI preserves saved models when discovery returns an invalid list", async ({ page }) => {
  await openPrivateConfiguration(page, { json: {} });

  await expect(page.getByRole("combobox", { name: "Default Model" })).toHaveValue("saved-model:27b");
  await expect(page.locator("#privateModelDiscoveryStatus")).toContainText("invalid model list");
});

test("Private AI refreshes the Pi model menu when discovery succeeds", async ({ page }) => {
  await openPrivateConfiguration(page, { json: { models: ["saved-model:27b", "added-model:32b"] } });

  const model = page.getByRole("combobox", { name: "Default Model" });
  await expect(model.getByRole("option", { name: "added-model:32b" })).toHaveCount(1);
  await expect(model).toHaveValue("saved-model:27b");
  await expect(page.locator("#privateModelDiscoveryStatus")).toBeEmpty();
});

for (const failure of ["configuration stream", "model health check"]) {
  test(`Ollama setup can edit the model after a failed ${failure}`, async ({ page }) => {
    const origin = "http://trustant.example.test";
    const message = "Add usage credits or subscribe (ref: test-401)";
    const responses = {
      "/api/configuration": {
        provider: "ollama",
        base_url: "http://ollama.example.test:11434/v1",
        models: { "model:cloud": {}, "local-model:32b": {} },
        pi: { default: "model:cloud" },
      },
      "/api/ollama-instances": { instances: [] },
      "/api/testmodel": { error: message },
    };
    await serveFixture(page, origin, new URL("../web/", import.meta.url), (route, path) => {
      if (path === "/api/configure") {
        return route.fulfill({
          contentType: "text/plain",
          body: failure === "configuration stream" ? `ERROR: Pi model test failed: ${message}\n` : "DONE\n",
        });
      }
      return route.fulfill({ json: responses[path] ?? {} });
    });
    await page.goto(origin);

    await expect(page.getByRole("log")).toContainText(message);
    await expect(page.getByRole("button", { name: "Retry", exact: true })).toBeVisible();
    await expect(page.getByRole("link", { name: "Choose another provider" })).toBeVisible();
    await page.getByRole("link", { name: "Edit configuration" }).click();
    await expect(page).toHaveURL(`${origin}/configure.html?setup=1`);
    const model = page.getByRole("combobox", { name: "Default Model" });
    await expect(model).toHaveValue("model:cloud");
    await model.selectOption("local-model:32b");
    await expect(model).toHaveValue("local-model:32b");
  });
}

test("Ollama model override can be undone from the row actions", async ({ page }) => {
  const origin = "http://trustant.example.test";
  const responses = {
    "/api/configuration": { provider: "ollama", models: { "qwen3.5:4b": {} } },
    "/api/ollama-instances": { instances: [] },
  };
  await serveFixture(page, origin, new URL("../web/", import.meta.url), (route, path) =>
    route.fulfill({ json: responses[path] ?? {} }),
  );
  await page.goto(`${origin}/configure.html`);

  const row = page.getByRole("row").filter({ hasText: "qwen3.5:4b" });
  const actions = row.getByRole("cell").filter({ has: page.getByRole("button", { name: "Remove" }) });
  const hiddenBadge = row.getByText("Hidden", { exact: true });
  const reason = row.getByText("below the recommended 20B minimum");
  await expect(hiddenBadge).toBeVisible();
  await expect(reason).toBeHidden();
  await hiddenBadge.press("Enter");
  await expect(reason).toBeVisible();
  await hiddenBadge.press("Enter");
  await expect(reason).toBeHidden();
  page.once("dialog", (dialog) => dialog.accept());
  await actions.getByRole("button", { name: "Use anyway" }).click();
  await expect(row.getByText("Used anyway", { exact: true })).toBeVisible();

  await actions.getByRole("button", { name: "Hide again" }).click();
  await expect(row.getByText("Hidden", { exact: true })).toBeVisible();
  await expect(actions.getByRole("button", { name: "Use anyway" })).toBeVisible();
});

test("Inventory and Template expose table semantics and preserve keyboard focus", async ({ page }) => {
  const origin = "http://trustant.example.test";
  const templates = {
    Examples: [{ name: "starter", title: "Starter", repo: "example/starter" }],
    Utilities: [
      { name: "auditdemo", title: "Audit Demo", repo: "example/auditdemo" },
      { name: "second", title: "Second Demo", repo: "example/second" },
    ],
  };
  const responses = {
    "/api/configuration": { provider: "ollama", pi: { default: "fixture" }, base_url: "http://localhost:11434" },
    "/api/repo": [{ name: "auditdemo", repo: "example/auditdemo" }],
    "/api/starters": { starters: [], applications: templates },
    "/api/github/status": { authenticated: false },
  };
  await serveFixture(page, origin, new URL("../web/", import.meta.url), (route, path) =>
    route.fulfill({ json: responses[path] ?? {} }),
  );

  await page.goto(`${origin}/applist.html`);
  const inventory = page.getByRole("table", { name: "Applications" });
  await expect(inventory.getByRole("columnheader", { name: "Repository" })).toHaveCount(1);
  await expect(inventory.getByRole("row").filter({ hasText: "auditdemo" })).toBeVisible();

  const addApp = page.getByRole("button", { name: "Add App", exact: true });
  await addApp.press("Enter");
  const templateMode = page.getByRole("button", { name: /^From a template/i });
  await templateMode.press("Enter");
  await expect(templateMode).toHaveAttribute("aria-pressed", "true");

  const utilities = page.getByRole("button", { name: "Utilities", exact: true });
  await utilities.press("Enter");
  await expect(utilities).toBeFocused();
  await expect(utilities).toHaveAttribute("aria-pressed", "true");

  const next = page.getByRole("button", { name: "Next application" });
  await next.press("Enter");
  await expect(page.getByRole("dialog").getByRole("status")).toHaveText("Second Demo, 2 of 2");

  const repository = page.getByRole("link", { name: "Second Demo", exact: true });
  const name = page.getByRole("textbox", { name: "Application Name" });
  // Observe the navigation without sending an external request.
  await page.context().route("https://github.com/example/second", (route) =>
    route.fulfill({ body: "Synthetic repository" }),
  );
  const [popup] = await Promise.all([page.waitForEvent("popup"), repository.press("Enter")]);
  await popup.waitForURL("https://github.com/example/second");
  await popup.close();
  await expect(name).toBeHidden();

  const useTemplate = page.getByRole("button", { name: "Use template: Second Demo" });
  await useTemplate.press("Space");
  await expect(name).toBeFocused();
  await name.fill("auditdemo");
  await name.press("Enter");
  await expect(page.getByRole("alert")).toContainText('An application named "auditdemo" already exists');
  await expect(name).toHaveAttribute("aria-invalid", "true");
  await expect(name).toHaveAccessibleDescription(/already exists/);

  await page.getByRole("button", { name: "Cancel", exact: true }).press("Enter");
  await expect(useTemplate).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(addApp).toBeFocused();
});

test("TruACP announces permissions, retries safely and keeps controls within a narrow viewport", async ({ page }) => {
  // Run npm --prefix acp run build:web before running this regression.
  const origin = "http://truacp.example.test";
  const sessionId = "fixture-session";
  const responses = {
    "/api/agents": {
      agents: [{ id: "fixture", displayName: "Fixture Agent" }],
      defaultAgentId: "fixture",
      projectDir: "/fixture",
    },
    "/api/session/initialize": { result: {} },
    "/api/session/new": { result: { sessionId, configOptions: [] } },
    "/api/sessions/notebook/get": { state: null },
  };

  let sendUpdate;
  await page.routeWebSocket("**/ws", (socket) => {
    sendUpdate = (update) => socket.send(JSON.stringify({ type: "sessionUpdate", update: { sessionId, ...update } }));
  });

  let requestCount = 0;
  let finishPrompt;
  let failNextResponse = true;
  const submitted = [];
  await serveFixture(page, origin, new URL("../acp/dist-web/", import.meta.url), async (route, path) => {
    if (path === "/api/session/prompt") {
      await new Promise((finish) => {
        finishPrompt = finish;
        requestCount++;
        sendUpdate({
          type: "tool_call",
          toolCallId: `read-${requestCount}`,
          title: "Read fixture",
          status: "pending",
          permissionRequest: {
            requestId: `permission-${requestCount}`,
            isActive: true,
            options: [
              { optionId: "allow", name: "Allow", kind: "allow_once" },
              { optionId: "reject", name: "Reject", kind: "reject_once" },
            ],
          },
        });
      });
    }
    if (path === "/api/permission/respond") {
      if (failNextResponse) {
        failNextResponse = false;
        return route.fulfill({ status: 503, json: { error: "Fixture response failed" } });
      }
      submitted.push(route.request().postDataJSON());
      finishPrompt();
    }
    return route.fulfill({ json: responses[path] ?? {} });
  });

  await page.setViewportSize({ width: 320, height: 640 });
  await page.goto(origin);
  await expect(page.getByRole("combobox", { name: "Agent" })).toHaveValue("fixture");
  const composer = page.getByRole("textbox", { name: "Message the agent" });
  await expect(composer).toBeEnabled();
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBe(320);

  const permission = page.getByRole("group", { name: "Permission required: Read fixture" });
  const allow = permission.getByRole("button", { name: "Allow" });
  const reject = permission.getByRole("button", { name: "Reject" });
  const send = async () => {
    await composer.fill("Read the synthetic fixture");
    await composer.press("Enter");
    await expect(permission).toBeFocused();
    await expect(page.getByRole("status")).toContainText("Waiting for permission");
  };

  await send();
  await page.keyboard.press("Tab");
  await expect(allow).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("alert")).toHaveText("Fixture response failed");
  await expect(allow).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(permission).toBeHidden();
  await expect(composer).toBeFocused();

  await send();
  await page.keyboard.press("Tab");
  await page.keyboard.press("Tab");
  await expect(reject).toBeFocused();
  await page.keyboard.press("Space");
  await expect(permission).toBeHidden();
  await expect(composer).toBeFocused();
  expect(submitted).toEqual([
    { requestId: "permission-1", optionId: "allow" },
    { requestId: "permission-2", optionId: "reject" },
  ]);
});
