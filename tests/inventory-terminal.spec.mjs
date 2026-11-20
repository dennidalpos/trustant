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

test('inventory loads terminal assets on demand and retries a failed load', async ({ page }) => {
  const html = await readFile('web/applist.html', 'utf8');
  const origin = 'http://trustant.example.test';
  const requests = [];
  const terminalConnections = [];
  let initialScriptBytes = 0;
  let terminalOpened = false;
  let fail = true;
  await page.routeWebSocket('**/api/terminal/', socket => {
    terminalConnections.push(socket.url());
  });
  await page.route('**/*', async route => {
    const url = new URL(route.request().url());
    // The shared stylesheet imports remote fonts. Only fixture assets may load.
    if (url.origin !== origin) return route.abort();
    const { pathname } = url;
    if (pathname.includes('xterm')) {
      requests.push(pathname);
      if (fail && pathname.endsWith('xterm.min.js')) {
        fail = false;
        return route.abort();
      }
    }
    if (pathname === '/applist.html') return route.fulfill({ contentType: 'text/html', body: html });
    if (pathname.startsWith('/api/')) {
      const body = pathname === '/api/configuration' ? { provider: 'ollama' } : pathname === '/api/repo' ? [] : {};
      return route.fulfill({ json: body });
    }
    try {
      const body = await readFile(path.join('web', pathname));
      if (!terminalOpened && pathname.endsWith('.js')) initialScriptBytes += body.length;
      const contentType = pathname.endsWith('.js') ? 'application/javascript' : pathname.endsWith('.css') ? 'text/css' : 'application/octet-stream';
      return route.fulfill({ body, contentType });
    } catch { return route.fulfill({ status: 404, body: '' }); }
  });
  await page.goto(origin + '/applist.html');
  expect(requests).toEqual([]);
  expect(terminalConnections).toEqual([]);
  expect(initialScriptBytes).toBeGreaterThan(0);
  console.log(`Initial external scripts: ${initialScriptBytes} bytes (uncompressed fixture responses)`);
  terminalOpened = true;
  await page.locator('#terminalBtn').focus();
  await page.keyboard.press('Enter');
  await expect(page.getByRole('status').filter({ hasText: 'Could not load the terminal' })).toContainText('Close and open it to try again.');
  expect(terminalConnections).toEqual([]);
  await page.keyboard.press('Escape');
  await expect(page.locator('#terminalModal')).toBeHidden();
  await page.locator('#terminalBtn').focus();
  await page.keyboard.press('Enter');
  await expect(page.locator('#terminalHost .xterm-helper-textarea')).toBeFocused();
  await page.keyboard.press('Escape');
  await expect(page.locator('#terminalModal')).toBeHidden();
  await page.locator('#terminalBtn').focus();
  await page.keyboard.press('Enter');
  await expect(page.locator('#terminalHost .xterm-helper-textarea')).toBeFocused();
  await expect.poll(() => terminalConnections).toEqual([
    'ws://trustant.example.test/api/terminal/',
    'ws://trustant.example.test/api/terminal/',
  ]);
  await page.keyboard.press('Escape');
  await expect(page.locator('#terminalModal')).toBeHidden();
  await expect(page.locator('#terminalBtn')).toBeFocused();
  expect(requests.filter(p => p.endsWith('xterm.min.js'))).toHaveLength(2);
  expect(requests.filter(p => p.endsWith('xterm-addon-fit.min.js'))).toHaveLength(1);
  expect(requests.filter(p => p.endsWith('xterm.css'))).toHaveLength(1);
});
