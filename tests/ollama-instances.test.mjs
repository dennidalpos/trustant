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
 * Unit tests for the Ollama instance switch (web/js/ollama-instances.js,
 * spec/2a-config.md "Ollama instances"), driven through a minimal DOM shim.
 *
 * Run with:  node --test tests/ollama-instances.test.mjs
 */

import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';

function makeElement(tag) {
    const el = {
        tag, className: '', textContent: '', value: '', checked: false, disabled: false, type: '',
        placeholder: '', children: [], listeners: {},
        classList: {
            set: new Set(),
            add(...c) { c.forEach((x) => this.set.add(x)); },
            remove(...c) { c.forEach((x) => this.set.delete(x)); },
            toggle(c, force) { (force ? this.set.add(c) : this.set.delete(c)); return force; },
            contains(c) { return this.set.has(c); },
        },
        setAttribute() {},
        focus() {},
        append(...kids) { el.children.push(...kids); },
        replaceChildren(...kids) { el.children = kids; },
        addEventListener(name, fn) { el.listeners[name] = fn; },
    };
    return el;
}

// Values built inside the vm context are not reference-equal to local ones.
const plain = (v) => JSON.parse(JSON.stringify(v));

function load() {
    const window = { fetch: () => { throw new Error('no network in tests'); } };
    window.fetch.bind = () => window.fetch;
    const context = { window, document: { createElement: makeElement } };
    vm.runInNewContext(readFileSync(new URL('../web/js/ollama-instances.js', import.meta.url), 'utf8'), context);
    return window.TrustantOllamaInstances;
}

// A fake server that mirrors the contract of /api/ollama-instances.
function fakeServer({ instances = [], active = '', models = ['m1', 'm2'], detected = null } = {}) {
    const calls = [];
    const state = { instances, active };
    const view = () => ({ ...(detected && !state.instances.some((i) => i.url === detected.url) ? { detected } : {}), instances: state.instances.map((i) => ({ ...i, key_set: false })), active: state.active, base_url: state.active ? 'http://x/v1' : 'http://localhost:11434/v1' });
    const reply = (data, status = 200) => Promise.resolve({ ok: status < 400, status, json: async () => data });
    const fetchImpl = (url, opts = {}) => {
        const body = opts.body ? JSON.parse(opts.body) : null;
        calls.push(`${opts.method || 'GET'} ${url} ${opts.body || ''}`.trim());
        if (url === '/api/discover-models') return reply({ models });
        if (!body) return reply(view());
        if (body.action === 'select') {
            if (body.id && !state.instances.find((i) => i.id === body.id)) return reply({ error: 'Unknown Ollama instance' }, 404);
            state.active = body.id;
            return reply(view());
        }
        if (body.action === 'save') {
            if (!body.instance.name.trim()) return reply({ error: 'Give the instance a name (up to 60 characters).' }, 400);
            const id = body.instance.id || body.instance.name.toLowerCase().replace(/[^a-z0-9]+/g, '-');
            state.instances = state.instances.filter((i) => i.id !== id).concat({ id, name: body.instance.name, url: body.instance.url });
            return reply(view());
        }
        if (body.action === 'delete') {
            state.instances = state.instances.filter((i) => i.id !== body.id);
            if (state.active === body.id) state.active = '';
            return reply(view());
        }
        return reply({ error: 'bad action' }, 400);
    };
    return { fetchImpl, calls, state };
}

function build(server, onChange) {
    const lib = load();
    const root = makeElement('div');
    const control = lib.mount(root, { fetch: server.fetchImpl, onChange });
    const [toggleRow, hint, pickRow, detectedRow, status, form] = root.children;
    const [detectedText, detectedBtn] = detectedRow.children;
    const toggle = toggleRow.children[0];
    const [select, addBtn, editBtn, removeBtn, testBtn] = pickRow.children;
    const [nameField, urlField, keyField, , formError, formButtons] = form.children;
    return {
        control, toggle, hint, select, detectedRow, detectedText, detectedBtn, addBtn, editBtn, removeBtn, testBtn, status, form, formError,
        nameInput: nameField.children[1], urlInput: urlField.children[1], keyInput: keyField.children[1],
        saveBtn: formButtons.children[0], cancelBtn: formButtons.children[1],
    };
}

test('request and discoverModels resolve instead of throwing', async () => {
    const lib = load();
    const bad = await lib.request(() => Promise.resolve({ ok: false, status: 500, json: async () => { throw new Error('x'); } }), 'GET');
    assert.equal(bad.ok, false);
    assert.match(bad.data.error, /500/);
    const down = await lib.discoverModels(() => Promise.reject(new Error('refused')), 'a');
    assert.deepEqual(plain(down), { error: 'refused' });
    const ok = await lib.discoverModels(() => Promise.resolve({ ok: true, status: 200, json: async () => ({ models: ['a', 'b'] }) }), 'a');
    assert.deepEqual(plain(ok), { models: ['a', 'b'] });
});

test('shows the default when nothing is active', async () => {
    const server = fakeServer({ instances: [{ id: 'a', name: 'Lab', url: 'http://10.0.0.5:11434' }] });
    const ui = build(server);
    await ui.control.refresh();
    assert.equal(ui.toggle.checked, false);
    assert.match(ui.hint.textContent, /this machine \(default\)/);
    assert.equal(ui.select.children.length, 1);
    assert.equal(ui.editBtn.disabled, false);
});

test('the switch activates an instance, reads its models and reports it', async () => {
    const server = fakeServer({ instances: [{ id: 'a', name: 'Lab', url: 'http://10.0.0.5:11434' }] });
    const seen = [];
    const ui = build(server, (state, discovered) => seen.push([state.active, discovered]));
    await ui.control.refresh();

    ui.toggle.checked = true;
    await ui.toggle.listeners.change();
    assert.deepEqual(plain(seen.at(-1)), ['a', { models: ['m1', 'm2'] }]);
    assert.equal(ui.toggle.checked, true);
    assert.match(ui.hint.textContent, /Using Lab/);
    assert.match(ui.status.textContent, /Found 2 models/);

    ui.toggle.checked = false;
    await ui.toggle.listeners.change();
    assert.deepEqual(plain(seen.at(-1)), ['', null]);
    assert.equal(ui.toggle.checked, false);
    assert.match(ui.status.textContent, /this machine/);
});

test('turning the switch on with no saved instance opens the form and activates the new one after saving', async () => {
    const server = fakeServer();
    const seen = [];
    const ui = build(server, (state, discovered) => seen.push([state.active, discovered && discovered.models]));
    await ui.control.refresh();

    ui.toggle.checked = true;
    ui.toggle.listeners.change();
    assert.equal(ui.toggle.checked, false, 'nothing to activate yet');
    assert.equal(ui.form.classList.contains('hidden'), false, 'the add form is open');

    ui.nameInput.value = 'Cluster B';
    ui.urlInput.value = 'http://10.0.0.9:11434';
    await ui.saveBtn.listeners.click();
    await new Promise((r) => setImmediate(r));
    assert.equal(server.state.active, 'cluster-b');
    assert.deepEqual(plain(seen.at(-1)), ['cluster-b', ['m1', 'm2']]);
    assert.equal(ui.form.classList.contains('hidden'), true);
});

test('a rejected save keeps the form open with the reason', async () => {
    const server = fakeServer();
    const ui = build(server);
    await ui.control.refresh();
    ui.addBtn.listeners.click();
    ui.nameInput.value = ' ';
    ui.urlInput.value = 'http://a:1';
    await ui.saveBtn.listeners.click();
    assert.match(ui.formError.textContent, /name/);
    assert.equal(ui.form.classList.contains('hidden'), false);
});

test('removing the active instance falls back to the default and reports it', async () => {
    const server = fakeServer({ instances: [{ id: 'a', name: 'Lab', url: 'http://h:1' }], active: 'a' });
    const seen = [];
    const ui = build(server, (state, discovered) => seen.push([state.active, discovered]));
    await ui.control.refresh();
    assert.equal(ui.toggle.checked, true);
    await ui.removeBtn.listeners.click();
    assert.deepEqual(plain(seen.at(-1)), ['', null]);
    assert.equal(ui.toggle.checked, false);
    assert.equal(ui.select.children.length, 0);
});

test('Refresh models re-reads the selected instance and updates the page when it is active', async () => {
    const server = fakeServer({ instances: [{ id: 'a', name: 'Lab', url: 'http://h:1' }], active: 'a', models: ['x'] });
    const seen = [];
    const ui = build(server, (state, discovered) => seen.push(discovered));
    await ui.control.refresh();
    await ui.testBtn.listeners.click();
    assert.match(ui.status.textContent, /Found 1 model on Lab/);
    assert.deepEqual(plain(seen.at(-1)), { models: ['x'] });
});

test('offers the detected host Ollama and saves it without switching it on', async () => {
    const detected = { name: 'Ollama host', url: 'http://172.27.224.1:11434', source: 'WSL host' };
    const server = fakeServer({ detected });
    const ui = build(server);
    await ui.control.refresh();
    assert.equal(ui.detectedRow.classList.contains('hidden'), false);
    assert.match(ui.detectedText.textContent, /WSL host.*172\.27\.224\.1:11434/);

    await ui.detectedBtn.listeners.click();
    assert.deepEqual(plain(server.state.instances), [{ id: 'ollama-host', name: 'Ollama host', url: detected.url }]);
    assert.equal(server.state.active, '');
    assert.equal(ui.toggle.checked, false);
    assert.equal(ui.detectedRow.classList.contains('hidden'), true);
    assert.match(ui.status.textContent, /Turn on the switch/);
    assert.ok(!server.calls.some((c) => c.includes('"action":"select"')), 'must not select the instance');
});

test('offers to update a stale host preset instead of adding a second one', async () => {
    const detected = { name: 'Ollama host', url: 'http://172.27.224.9:11434', source: 'WSL host', replaces_id: 'ollama-host', replaces_url: 'http://172.27.224.1:11434' };
    const server = fakeServer({ instances: [{ id: 'ollama-host', name: 'Ollama host', url: 'http://172.27.224.1:11434' }], detected });
    const ui = build(server);
    await ui.control.refresh();
    assert.match(ui.detectedText.textContent, /no longer answers.*172\.27\.224\.9/);
    assert.equal(ui.detectedBtn.textContent, 'Update');

    await ui.detectedBtn.listeners.click();
    assert.deepEqual(plain(server.state.instances), [{ id: 'ollama-host', name: 'Ollama host', url: detected.url }]);
    assert.match(ui.status.textContent, /Updated "Ollama host"/);
});

test('shows nothing extra when no host Ollama is detected', async () => {
    const ui = build(fakeServer());
    await ui.control.refresh();
    assert.equal(ui.detectedRow.classList.contains('hidden'), true);
});
