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

/**
 * Unit tests for the local Ollama Start/Stop control (web/js/ollama-local.js,
 * spec/2a-config.md "Local Ollama control"), driven through a minimal DOM shim.
 *
 * Run with:  node --test tests/ollama-local.test.mjs
 */

import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';

function makeElement(tag) {
    const el = {
        tag, style: {}, className: '', textContent: '', disabled: false, type: '', children: [], listeners: {},
        classList: {
            set: new Set(),
            add(...c) { c.forEach((x) => this.set.add(x)); },
            toggle(c, force) { (force ? this.set.add(c) : this.set.delete(c)); return force; },
            contains(c) { return this.set.has(c); },
        },
        setAttribute() {},
        replaceChildren(...kids) { el.children = kids; },
        addEventListener(name, fn) { el.listeners[name] = fn; },
    };
    // className seeds the hidden flag the same way a class attribute would.
    return el;
}

function load(fetchImpl) {
    const window = {};
    const context = {
        window,
        document: { createElement: makeElement },
        fetch: fetchImpl,
        AbortSignal: { timeout: () => undefined },
        URL,
        Number,
    };
    vm.runInNewContext(readFileSync(new URL('../web/js/ollama-local.js', import.meta.url), 'utf8'), context);
    return window.TrustantOllamaLocal;
}

const reply = (data, status = 200) => Promise.resolve({ status, json: async () => data });

test('isLocalBaseUrl only accepts loopback hosts', () => {
    const lib = load(() => reply({}));
    for (const ok of ['http://localhost:11434/v1', 'http://127.0.0.1:11434/v1', 'http://[::1]:11434/v1']) {
        assert.equal(lib.isLocalBaseUrl(ok), true, ok);
    }
    for (const bad of ['https://api.example.com/v1', 'http://192.168.1.5:11434/v1', 'not a url', '', undefined]) {
        assert.equal(lib.isLocalBaseUrl(bad), false, String(bad));
    }
});

test('shows the state and toggles the service with the button', async () => {
    const calls = [];
    let active = false;
    const lib = load((url, opts = {}) => {
        calls.push(`${opts.method} ${opts.body || ''}`.trim());
        if (opts.method === 'POST') active = JSON.parse(opts.body).action === 'start';
        return reply({ supported: true, wsl: true, state: 'loaded', active, reachable: active, models: active ? 6 : 0 });
    });
    const root = makeElement('div');
    const control = lib.mount(root);
    const [label, state, button] = root.children;

    control.setVisible(true);
    await control.refresh();
    assert.equal(root.classList.contains('hidden'), false);
    assert.equal(label.textContent, 'Local Ollama (WSL)');
    assert.equal(state.textContent, 'Stopped');
    assert.equal(button.textContent, 'Start');

    await button.listeners.click();
    assert.equal(calls.at(-1), 'POST {"action":"start"}');
    assert.equal(state.textContent, 'Running · 6 models');
    assert.equal(button.textContent, 'Stop');

    await button.listeners.click();
    assert.equal(calls.at(-1), 'POST {"action":"stop"}');
    assert.equal(state.textContent, 'Stopped');
});

test('hides itself when the host has no controllable service or the URL is remote', async () => {
    const lib = load(() => reply({ supported: false, wsl: false, state: '', active: false, reachable: false, models: 0 }));
    const root = makeElement('div');
    const control = lib.mount(root);
    control.setVisible(true);
    await control.refresh();
    assert.equal(root.classList.contains('hidden'), true);

    const supported = load(() => reply({ supported: true, state: 'loaded', active: true, reachable: true, models: 1 }));
    const root2 = makeElement('div');
    const control2 = supported.mount(root2);
    await control2.refresh();
    control2.setVisible(false);
    assert.equal(root2.classList.contains('hidden'), true);
});

test('reports a failed Start without throwing', async () => {
    const lib = load((url, opts = {}) => opts.method === 'POST'
        ? reply({ supported: true, state: 'loaded', active: false, reachable: false, models: 0, error: 'Could not start Ollama: sudo: a password is required' }, 500)
        : reply({ supported: true, state: 'loaded', active: false, reachable: false, models: 0 }));
    const root = makeElement('div');
    const control = lib.mount(root);
    control.setVisible(true);
    await control.refresh();
    await root.children[2].listeners.click();
    const error = root.children[3];
    assert.match(error.textContent, /password is required/);
    assert.equal(error.classList.contains('hidden'), false);
});
