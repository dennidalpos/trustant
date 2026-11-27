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

import { test, expect } from '@playwright/test';
import { readFile } from 'node:fs/promises';
import path from 'node:path';

const types = { '.js': 'text/javascript', '.css': 'text/css', '.svg': 'image/svg+xml', '.html': 'text/html' };

const origin = 'http://trustant.example.test';
const models = ['qwen-a', 'qwen-b', 'qwen-c'];

// Opens configure.html for a Private AI endpoint with a fixture backend. The
// page is served from memory, so no cluster or provider is needed.
async function openConfigure(page, piDefault) {
  const posted = [];
  const config = {
    provider: 'private',
    base_url: 'http://models.example.test/v1',
    api_key: 'dummy',
    models: Object.fromEntries(models.map(m => [m, { maxToken: 32768, maxOutput: 8192 }])),
    pi: { default: piDefault },
  };
  await page.route('**/*', async route => {
    const url = new URL(route.request().url());
    if (url.origin !== origin) return route.abort();
    const { pathname } = url;
    if (pathname === '/api/configuration' && route.request().method() === 'POST') {
      posted.push(route.request().postDataJSON());
      return route.fulfill({ json: { testmodel: { ok: true } } });
    }
    if (pathname === '/api/configuration') return route.fulfill({ json: config });
    if (pathname === '/api/discover-models') return route.fulfill({ json: { models } });
    if (pathname === '/api/version') return route.fulfill({ json: {} });
    if (pathname.startsWith('/api/')) return route.fulfill({ json: {} });
    try {
      const file = path.join('web', path.normalize(pathname.slice(1)));
      const body = await readFile(file);
      return route.fulfill({ contentType: types[path.extname(file)] || 'application/octet-stream', body });
    } catch {
      return route.fulfill({ status: 404, body: '' });
    }
  });
  await page.goto(`${origin}/configure.html`);
  await expect(page.locator('#configurationLoading')).toBeHidden();
  return posted;
}

test('opening without a default shows an empty choice and blocks saving', async ({ page }) => {
  const posted = await openConfigure(page, '');
  const select = page.locator('#piDefault');
  await expect(select).toHaveValue('');
  await expect(select.locator('option:checked')).toHaveText(/Select the default model/);
  await expect(select.locator('option')).toHaveCount(models.length + 1);

  let dialog = '';
  page.once('dialog', d => { dialog = d.message(); d.dismiss(); });
  await page.locator('#saveBtn').click();
  await expect.poll(() => dialog).toBe('Please select the default model before saving.');
  expect(posted).toHaveLength(0);
});

test('an explicit selection is saved as pi.default', async ({ page }) => {
  const posted = await openConfigure(page, '');
  await page.locator('#piDefault').selectOption('qwen-b');
  await page.locator('#saveBtn').click();
  await expect.poll(() => posted.length).toBe(1);
  expect(posted[0].pi.default).toBe('qwen-b');
});

test('an already configured default is preserved', async ({ page }) => {
  const posted = await openConfigure(page, 'qwen-c');
  const select = page.locator('#piDefault');
  await expect(select).toHaveValue('qwen-c');
  await expect(select.locator('option')).toHaveCount(models.length);
  await page.locator('#saveBtn').click();
  await expect.poll(() => posted.length).toBe(1);
  expect(posted[0].pi.default).toBe('qwen-c');
});
