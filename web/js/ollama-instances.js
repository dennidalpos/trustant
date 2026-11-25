// Copyright 2025-2026 Nuvolaris Inc
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// Switch and preset list for the Ollama the "ollama" provider talks to
// (/api/ollama-instances, spec/2a-config.md "Ollama instances"). Off means the
// default: the Ollama of this machine. On means one of the saved instances.
// API keys are write-only here: the server never sends them back.
(function () {
    'use strict';

    // request wraps one call to the API and always resolves to { ok, data }.
    async function request(fetchImpl, method, body) {
        const resp = await fetchImpl('/api/ollama-instances', {
            method,
            headers: body ? { 'Content-Type': 'application/json' } : undefined,
            body: body ? JSON.stringify(body) : undefined,
        });
        let data = null;
        try { data = await resp.json(); } catch (e) { /* non-JSON error body */ }
        return { ok: !!resp.ok, data: data || { error: 'Unexpected response (' + resp.status + ').' } };
    }

    // discoverModels asks the server to list the models of a saved instance
    // (it adds the stored key itself) and resolves to { models } or { error }.
    async function discoverModels(fetchImpl, id) {
        try {
            const resp = await fetchImpl('/api/discover-models', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ instance_id: id }),
            });
            const data = await resp.json().catch(() => null);
            if (!resp.ok || !data || data.error) {
                return { error: (data && data.error) || ('HTTP ' + resp.status) };
            }
            return { models: Array.isArray(data.models) ? data.models : [] };
        } catch (e) {
            return { error: e.message };
        }
    }

    function plural(n, word) {
        return n + ' ' + word + (n === 1 ? '' : 's');
    }

    // mount builds the control inside `root`. opts.onChange(state, discovered)
    // runs after every successful switch, where `discovered` is null for the
    // default or { models } / { error } for an instance.
    function mount(root, opts) {
        opts = opts || {};
        const fetchImpl = opts.fetch || window.fetch.bind(window);
        let state = { instances: [], active: '' };
        let busy = false;
        let editingId = null; // null = closed, '' = new, otherwise the id being edited
        let activateAfterSave = false;

        const el = (tag, cls, text) => {
            const node = document.createElement(tag);
            if (cls) node.className = cls;
            if (text) node.textContent = text;
            return node;
        };
        const field = (labelText, input) => {
            const wrap = el('label', 'block text-sm');
            wrap.append(el('span', 'block font-medium mb-1', labelText), input);
            return wrap;
        };

        const toggle = el('input');
        toggle.type = 'checkbox';
        const toggleRow = el('label', 'flex items-center gap-2 text-sm font-medium');
        toggleRow.append(toggle, el('span', '', 'Use an external Ollama'));
        const hint = el('p', 'text-sm text-[color:var(--nu-muted)] mt-1');

        const select = el('select', 'nu-input');
        select.setAttribute('aria-label', 'Ollama instance');
        const addBtn = el('button', 'nu-btn nu-btn-secondary', 'Add');
        const editBtn = el('button', 'nu-btn nu-btn-secondary', 'Edit');
        const removeBtn = el('button', 'nu-btn nu-btn-secondary', 'Remove');
        const testBtn = el('button', 'nu-btn nu-btn-secondary', 'Refresh models');
        for (const b of [addBtn, editBtn, removeBtn, testBtn]) b.type = 'button';
        const pickRow = el('div', 'flex flex-wrap items-center gap-2 mt-3');
        pickRow.append(select, addBtn, editBtn, removeBtn, testBtn);

        const status = el('p', 'text-sm mt-2 hidden');
        status.setAttribute('role', 'status');

        const nameInput = el('input', 'nu-input w-full');
        nameInput.type = 'text';
        nameInput.placeholder = 'Lab cluster';
        nameInput.autocomplete = 'off';
        const urlInput = el('input', 'nu-input w-full');
        urlInput.type = 'text';
        urlInput.placeholder = 'http://host:11434';
        urlInput.autocomplete = 'off';
        urlInput.spellcheck = false;
        const keyInput = el('input', 'nu-input w-full');
        keyInput.type = 'password';
        keyInput.autocomplete = 'new-password';
        const clearKey = el('input');
        clearKey.type = 'checkbox';
        const clearRow = el('label', 'flex items-center gap-2 text-sm hidden');
        clearRow.append(clearKey, el('span', '', 'Remove the stored key'));
        const saveBtn = el('button', 'nu-btn nu-btn-primary', 'Save');
        const cancelBtn = el('button', 'nu-btn nu-btn-secondary', 'Cancel');
        saveBtn.type = cancelBtn.type = 'button';
        const formError = el('p', 'text-sm text-[color:var(--nu-danger)] hidden');
        formError.setAttribute('role', 'alert');
        const form = el('div', 'hidden mt-3 space-y-3');
        const formButtons = el('div', 'flex gap-2');
        formButtons.append(saveBtn, cancelBtn);
        form.append(field('Name', nameInput), field('URL', urlInput), field('API key (optional)', keyInput), clearRow, formError, formButtons);

        root.replaceChildren(toggleRow, hint, pickRow, status, form);

        function selected() {
            return state.instances.find((i) => i.id === select.value) || null;
        }

        function say(message, kind) {
            status.textContent = message || '';
            status.classList.toggle('hidden', !message);
            status.classList.toggle('text-[color:var(--nu-danger)]', kind === 'error');
            status.classList.toggle('text-[color:var(--nu-muted)]', kind !== 'error');
        }

        function render() {
            const previous = select.value;
            select.replaceChildren(...state.instances.map((inst) => {
                const option = el('option', '', inst.name + ' — ' + inst.url);
                option.value = inst.id;
                return option;
            }));
            const wanted = state.active || previous;
            if (state.instances.some((i) => i.id === wanted)) select.value = wanted;
            else if (state.instances.length) select.value = state.instances[0].id;

            const active = state.instances.find((i) => i.id === state.active);
            toggle.checked = !!active;
            hint.textContent = active
                ? 'Using ' + active.name + ' (' + active.url + ').'
                : 'Using the Ollama of this machine (default).';
            toggle.disabled = busy;
            select.disabled = busy || state.instances.length === 0;
            select.classList.toggle('hidden', state.instances.length === 0);
            const have = !!selected();
            addBtn.disabled = busy;
            editBtn.disabled = busy || !have;
            removeBtn.disabled = busy || !have;
            testBtn.disabled = busy || !have;
            form.classList.toggle('hidden', editingId === null);
        }

        function openForm(id, activate) {
            const inst = id ? state.instances.find((i) => i.id === id) : null;
            editingId = id === null ? null : (inst ? inst.id : '');
            activateAfterSave = !!activate;
            nameInput.value = inst ? inst.name : '';
            urlInput.value = inst ? inst.url : '';
            keyInput.value = '';
            keyInput.placeholder = inst && inst.key_set ? 'Stored — leave empty to keep it' : 'Optional';
            clearKey.checked = false;
            clearRow.classList.toggle('hidden', !(inst && inst.key_set));
            formError.textContent = '';
            formError.classList.add('hidden');
            render();
            nameInput.focus();
        }

        function closeForm() {
            editingId = null;
            activateAfterSave = false;
            render();
        }

        async function switchTo(id) {
            busy = true;
            say('', '');
            render();
            const result = await request(fetchImpl, 'POST', { action: 'select', id });
            if (!result.ok) {
                busy = false;
                render();
                say(result.data.error || 'Could not switch the Ollama instance.', 'error');
                return;
            }
            state = result.data;
            let discovered = null;
            if (id) {
                say('Reading the models of ' + (selected() ? selected().name : 'the instance') + '…', 'info');
                discovered = await discoverModels(fetchImpl, id);
            }
            busy = false;
            render();
            if (!id) {
                say('Using the Ollama of this machine. Check the model list, then Save & Configure.', 'info');
            } else if (discovered.error) {
                say('Switched, but the models could not be read: ' + discovered.error, 'error');
            } else {
                say('Found ' + plural(discovered.models.length, 'model') + '. Pick the Pi model below, then Save & Configure.', 'info');
            }
            if (opts.onChange) opts.onChange(state, discovered);
        }

        toggle.addEventListener('change', () => {
            if (toggle.checked) {
                if (!state.instances.length) {
                    toggle.checked = false;
                    openForm('', true);
                    return;
                }
                return switchTo(select.value || state.instances[0].id);
            }
            return switchTo('');
        });
        select.addEventListener('change', () => {
            if (state.active) return switchTo(select.value);
            return render();
        });
        addBtn.addEventListener('click', () => openForm('', false));
        editBtn.addEventListener('click', () => { if (selected()) openForm(selected().id, false); });
        cancelBtn.addEventListener('click', closeForm);

        removeBtn.addEventListener('click', async () => {
            const inst = selected();
            if (!inst || busy) return;
            if (typeof confirm === 'function' && !confirm('Remove "' + inst.name + '"?')) return;
            const wasActive = state.active === inst.id;
            busy = true;
            render();
            const result = await request(fetchImpl, 'POST', { action: 'delete', id: inst.id });
            busy = false;
            if (!result.ok) {
                render();
                say(result.data.error || 'Could not remove the instance.', 'error');
                return;
            }
            state = result.data;
            render();
            say(wasActive ? 'Removed. Back to the Ollama of this machine.' : 'Removed.', 'info');
            if (wasActive && opts.onChange) opts.onChange(state, null);
        });

        testBtn.addEventListener('click', async () => {
            const inst = selected();
            if (!inst || busy) return;
            busy = true;
            render();
            say('Contacting ' + inst.url + '…', 'info');
            const discovered = await discoverModels(fetchImpl, inst.id);
            busy = false;
            render();
            if (discovered.error) {
                say(discovered.error, 'error');
                return;
            }
            say('Found ' + plural(discovered.models.length, 'model') + ' on ' + inst.name + '.', 'info');
            if (state.active === inst.id && opts.onChange) opts.onChange(state, discovered);
        });

        saveBtn.addEventListener('click', async () => {
            if (busy || editingId === null) return;
            const before = new Set(state.instances.map((i) => i.id));
            const editedId = editingId;
            const instance = { name: nameInput.value, url: urlInput.value };
            if (editingId) instance.id = editingId;
            if (keyInput.value.trim()) instance.api_key = keyInput.value.trim();
            if (clearKey.checked) instance.clear_key = true;
            busy = true;
            render();
            const result = await request(fetchImpl, 'POST', { action: 'save', instance });
            busy = false;
            if (!result.ok) {
                render();
                formError.textContent = result.data.error || 'Could not save the instance.';
                formError.classList.remove('hidden');
                return;
            }
            state = result.data;
            const created = state.instances.find((i) => !before.has(i.id));
            const activate = activateAfterSave && created;
            closeForm();
            if (created) select.value = created.id;
            say('Saved.', 'info');
            if (activate) switchTo(created.id);
            else if (editedId && state.active === editedId && opts.onChange) opts.onChange(state, null);
        });

        async function refresh() {
            const result = await request(fetchImpl, 'GET');
            if (result.ok) state = result.data;
            render();
            return state;
        }

        return { refresh, getState: () => state };
    }

    window.TrustantOllamaInstances = { mount, request, discoverModels };
})();
