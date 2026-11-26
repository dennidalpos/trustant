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

import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const html = readFileSync(new URL('../web/applist.html', import.meta.url), 'utf8');

test('inventory dialog fields have explicit accessible names', () => {
    const labelIds = ['gitPushRepo', 'publishApiHost', 'publishUser', 'publishPassword', 'publishProvisionToken'];
    const sshKeyIds = ['addAppSshKey', 'ownStarterSshKey', 'gitPushSshKey'];

    for (const id of labelIds) {
        assert.match(html, new RegExp('<input\\b[^>]*\\bid="' + id + '"'), id);
        assert.match(html, new RegExp('<label\\b[^>]*\\bfor="' + id + '"[^>]*>\\s*\\S'), id);
    }
    for (const id of sshKeyIds) {
        assert.match(html, new RegExp('<textarea\\b[^>]*\\bid="' + id + '"[^>]*\\baria-label="[^"]+"'), id);
    }
});
