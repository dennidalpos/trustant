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

import { expect, test } from "@playwright/test";
import { fileURLToPath } from "node:url";

test("development variables support suggested, shared, and manual values", async ({ page }) => {
  const baseURL = "http://ollama.example.test:11434/v1";
  let offline = false;
  let requests = 0;
  const saves = [];
  await page.route("http://trustant.example.test/**", async (route) => {
    const request = route.request();
    if (request.method() === "POST") {
      saves.push(request.postDataJSON());
      await route.fulfill({ json: {} });
    } else if (new URL(request.url()).searchParams.has("suggestions")) {
      requests++;
      await route.fulfill({ json: { suggestions: offline ? {} : { baseURL, models: ["sample:latest"] } } });
    } else {
      await route.fulfill({ contentType: "text/html", body: "<!doctype html><main></main>" });
    }
  });
  await page.goto("http://trustant.example.test/");
  await page.addScriptTag({ path: fileURLToPath(new URL("../web/tailwind.js", import.meta.url)) });
  await page.addStyleTag({ path: fileURLToPath(new URL("../web/trustant-ui.css", import.meta.url)) });
  await page.addScriptTag({ path: fileURLToPath(new URL("../web/js/bind.js", import.meta.url)) });
  await page.evaluate(() => {
    const resolver = window.ImportResolver.mount("resolver");
    window.resumes = [];
    resolver.onResolved = (name) => window.resumes.push(name);
    window.rows = [
      { name: "AI_BASE_URL", pending: true },
      { name: "AI_CHAT_MODEL", pending: true },
      { name: "AI_API_KEY", pending: true, matches: ["APP__AI_KEY"], suggested: "APP__AI_KEY" },
    ];
    return resolver.open("demoapp", window.rows);
  });

  const url = page.getByLabel("AI_BASE_URL value");
  const model = page.getByLabel("AI_CHAT_MODEL value");
  const key = page.getByLabel("AI_API_KEY value");
  const shared = page.getByLabel("AI_API_KEY shared variable");
  const modelOptions = page.locator("datalist[id$='-AI_CHAT_MODEL'] option");
  await expect(page.locator("datalist[id$='-AI_BASE_URL'] option")).toHaveAttribute("value", baseURL);
  await expect(modelOptions).toHaveAttribute("value", "sample:latest");
  await url.fill("https://custom.example.test/v1");
  await expect(modelOptions).toHaveCount(0);
  await url.fill(baseURL);
  await expect(modelOptions).toHaveCount(1);
  await model.fill("sample:latest");
  await expect(shared).toHaveValue("APP__AI_KEY");
  await expect(key).toHaveAttribute("type", "password");
  await key.fill("synthetic-secret");
  await expect(shared).toHaveValue("");
  await shared.selectOption("APP__AI_KEY");
  await expect(key).toHaveValue("");
  await page.evaluate(() => Promise.all([window.resolver.confirm(), window.resolver.confirm()]));
  expect(saves).toEqual([{
    choices: { AI_API_KEY: "APP__AI_KEY" },
    values: { AI_BASE_URL: baseURL, AI_CHAT_MODEL: "sample:latest" },
  }]);
  expect(await page.evaluate(() => window.resumes)).toEqual(["demoapp"]);

  offline = true;
  await page.evaluate(() => window.resolver.open("demoapp", window.rows));
  await expect.poll(() => requests).toBe(2);
  await expect(modelOptions).toHaveCount(0);
  await model.fill("manual-model");
  await page.getByRole("button", { name: "Cancel" }).click();
  expect(saves).toHaveLength(1);
  await page.evaluate(() => window.resolver.open("demoapp", window.rows, {
    mode: "production", host: "publish.example.test",
  }));
  await expect(page.getByText("on publish.example.test")).toBeVisible();
  await expect(page.locator("datalist")).toHaveCount(0);
  expect(requests).toBe(2);
});
