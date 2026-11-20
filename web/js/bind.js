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

// Importing variables from the shared pool (spec/19-import.md).
//
// Two pieces, both the consumer side of the Share dialog's Export tab:
//
//   ImportEditor    the Import tab: rows of name + pattern, written to .env.dist
//   ImportResolver  the launch popup: every variable, and what it will be set to.
//                   In production mode (publish) the same rows are resolved
//                   against the target host's pool and answers go to
//                   apps.<name>.production.
//
// The vocabulary is fixed and worth keeping straight: EXPORT is what this app
// publishes for others (.env.shared, shared.js), IMPORT is what it takes from
// them (.env.dist, this file).
(function (global) {
    'use strict';

    function escapeHTML(s) {
        return String(s == null ? '' : s).replace(/[&<>"']/g, (c) => ({
            '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
        }[c]));
    }

    function maskValue(value) {
        if (!value) return '';
        if (value.length <= 8) return '••••••••';
        return value.slice(0, 4) + '••••••••' + value.slice(-2);
    }

    // --- ImportEditor: the Import tab ---------------------------------------

    function ImportEditor(options) {
        options = options || {};
        this.prefix = options.prefix || 'importEditor';
        this.appName = '';
        this.bindings = [];
        this.pool = [];
        this.resolutions = {};
    }

    ImportEditor.prototype.el = function (suffix) {
        return document.getElementById(this.prefix + suffix);
    };

    ImportEditor.prototype.url = function () {
        return '/api/imports/' + encodeURIComponent(this.appName);
    };

    ImportEditor.prototype.load = async function (appName) {
        this.appName = appName;
        try {
            const response = await fetch('/api/imports/' + encodeURIComponent(appName));
            if (!response.ok) throw new Error(await response.text());
            const data = await response.json();
            this.bindings = (data.bindings || []).map((b) => ({ name: b.name, pattern: b.pattern || '' }));
            this.pool = data.pool || [];
            this.resolutions = {};
            (data.resolutions || []).forEach((r) => { this.resolutions[r.name] = r; });
            this.render();
        } catch (err) {
            this.showError('Failed to load imports: ' + err.message);
        }
    };

    ImportEditor.prototype.showError = function (message) {
        const el = this.el('Error');
        if (!el) return;
        if (!message) {
            el.classList.add('hidden');
            return;
        }
        el.textContent = message;
        el.classList.remove('hidden');
    };

    // matchCount mirrors the server's matcher so the row can show, as the user
    // types, how many pool variables a pattern selects. The server is still the
    // authority; this only avoids a round trip per keystroke.
    ImportEditor.prototype.matchCount = function (binding) {
        const pattern = (binding.pattern || '').trim();
        if (!pattern) {
            return this.pool.indexOf(binding.name) >= 0 ? 1 : 0;
        }
        if (pattern.indexOf('*') < 0) {
            return this.pool.indexOf(pattern) >= 0 ? 1 : 0;
        }
        const parts = pattern.split('*');
        return this.pool.filter((name) => {
            if (name.indexOf(parts[0]) !== 0) return false;
            let rest = name.slice(parts[0].length);
            for (let i = 1; i < parts.length - 1; i++) {
                if (!parts[i]) continue;
                const idx = rest.indexOf(parts[i]);
                if (idx < 0) return false;
                rest = rest.slice(idx + parts[i].length);
            }
            const last = parts[parts.length - 1];
            if (!last) return true;
            return rest.length >= last.length && rest.slice(-last.length) === last;
        }).length;
    };

    ImportEditor.prototype.render = function () {
        const body = this.el('Body');
        if (!body) return;
        const g = this.globalName();

        if (this.bindings.length === 0) {
            body.innerHTML = `
                <div class="text-xs text-[color:var(--nu-muted)] p-3">
                  This application imports nothing yet. Add a variable to take its value
                  from a shared variable another application exports.
                </div>`;
            return;
        }

        body.innerHTML = this.bindings.map((b, i) => {
            const count = this.matchCount(b);
            const res = this.resolutions[b.name];
            let hint;
            if (count === 0) {
                hint = '<span class="text-red-500">no match</span>';
            } else if (count === 1) {
                const source = res && res.source ? ' — ' + escapeHTML(res.source) : '';
                hint = '<span class="text-[color:var(--nu-muted)]">1 match' + source + '</span>';
            } else {
                hint = '<span class="text-amber-500">' + count + ' matches — you will be asked to choose</span>';
            }
            return `
                <div class="nu-card p-2 mb-2">
                  <div class="flex gap-2 items-center">
                    <input type="text" value="${escapeHTML(b.name)}"
                        oninput="${g}.setName(${i}, this.value)"
                        placeholder="VARIABLE_NAME"
                        class="nu-input nu-mono text-xs flex-1" />
                    <span class="text-xs text-[color:var(--nu-muted)]">=</span>
                    <input type="text" value="${escapeHTML(b.pattern)}"
                        oninput="${g}.setPattern(${i}, this.value)"
                        placeholder="leave empty for an exact name, or use * "
                        class="nu-input nu-mono text-xs flex-1" />
                    <button type="button" onclick="${g}.remove(${i})"
                        class="nu-btn nu-btn-secondary nu-btn-compact">Remove</button>
                  </div>
                  <div class="text-[11px] mt-1 pl-1">${hint}</div>
                </div>`;
        }).join('');
    };

    ImportEditor.prototype.setName = function (index, value) {
        if (!this.bindings[index]) return;
        this.bindings[index].name = value.trim();
        this.renderHintsSoon();
    };

    ImportEditor.prototype.setPattern = function (index, value) {
        if (!this.bindings[index]) return;
        this.bindings[index].pattern = value.trim();
        this.renderHintsSoon();
    };

    // Re-rendering on every keystroke would steal focus from the input being
    // typed into, so the match count is refreshed on a short debounce and only
    // the hint lines are rewritten.
    ImportEditor.prototype.renderHintsSoon = function () {
        clearTimeout(this._hintTimer);
        this._hintTimer = setTimeout(() => {
            const body = this.el('Body');
            if (!body) return;
            const hints = body.querySelectorAll('.nu-card > div:last-child');
            this.bindings.forEach((b, i) => {
                if (!hints[i]) return;
                const count = this.matchCount(b);
                if (count === 0) {
                    hints[i].innerHTML = '<span class="text-red-500">no match</span>';
                } else if (count === 1) {
                    hints[i].innerHTML = '<span class="text-[color:var(--nu-muted)]">1 match</span>';
                } else {
                    hints[i].innerHTML = '<span class="text-amber-500">' + count +
                        ' matches — you will be asked to choose</span>';
                }
            });
        }, 150);
    };

    ImportEditor.prototype.add = function () {
        this.bindings.push({ name: '', pattern: '' });
        this.render();
    };

    ImportEditor.prototype.remove = function (index) {
        this.bindings.splice(index, 1);
        this.render();
    };

    ImportEditor.prototype.save = async function () {
        this.showError('');
        const cleaned = this.bindings.filter((b) => b.name);
        try {
            const response = await fetch(this.url(), {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ bindings: cleaned })
            });
            const data = await response.json().catch(() => ({}));
            if (!response.ok) {
                // The server refuses in full, so nothing was written and the rows
                // on screen are still exactly what the user meant to save.
                this.showError(data.error || 'Failed to save');
                return false;
            }
            this.bindings = (data.bindings || []).map((b) => ({ name: b.name, pattern: b.pattern || '' }));
            this.resolutions = {};
            (data.resolutions || []).forEach((r) => { this.resolutions[r.name] = r; });
            this.render();
            if (this.onSaved) this.onSaved(data);
            return true;
        } catch (err) {
            this.showError('Failed to save: ' + err.message);
            return false;
        }
    };

    ImportEditor.prototype.globalName = function () {
        return this._globalName;
    };

    ImportEditor.mount = function (globalName, options) {
        const editor = new ImportEditor(options);
        editor._globalName = globalName;
        global[globalName] = editor;
        return editor;
    };

    // --- ImportResolver: the launch popup -----------------------------------

    function ImportResolver(options) {
        options = options || {};
        this.prefix = options.prefix || 'importResolver';
        this.appName = '';
        this.rows = [];
        this.choices = {};
        this.values = {};
        this.mode = 'development';
        this.host = '';
        this.opening = 0;
        this.suggestions = {};
        this.saving = false;
    }

    ImportResolver.prototype.el = function (suffix) {
        return document.getElementById(this.prefix + suffix);
    };

    ImportResolver.prototype.globalName = function () {
        return this._globalName;
    };

    // The endpoint for this mode: production carries the target host.
    ImportResolver.prototype.url = function () {
        let url = '/api/imports/' + encodeURIComponent(this.appName);
        if (this.mode === 'production') {
            url += '?mode=production&host=' + encodeURIComponent(this.host);
        }
        return url;
    };

    // options: { mode: 'production', host } opens the publish popup for a host.
    ImportResolver.prototype.open = async function (appName, rows, options) {
        options = options || {};
        const opening = ++this.opening;
        this.appName = appName;
        this.mode = options.mode === 'production' ? 'production' : 'development';
        this.host = options.host || '';
        this.choices = {};
        this.values = {};
        this.suggestions = {};
        this.saving = false;
        let error = '';

        if (rows && rows.length) {
            this.rows = rows;
        } else {
            try {
                const response = await fetch(this.url());
                if (!response.ok) throw new Error(await response.text());
                const data = await response.json();
                if (opening !== this.opening) return;
                this.rows = data.resolutions || [];
            } catch (err) {
                if (opening !== this.opening) return;
                this.rows = [];
                error = 'Failed to load variables: ' + err.message;
            }
        }

        // A first-time binding arrives pending with its single candidate
        // suggested: pre-select it, so confirming is one click.
        this.rows.forEach((row) => {
            if (row.pending && row.suggested) this.choices[row.name] = row.suggested;
        });

        const nameEl = this.el('AppName');
        if (nameEl) nameEl.textContent = appName;
        const hostEl = this.el('Host');
        if (hostEl) {
            hostEl.textContent = this.mode === 'production' ? ' on ' + this.host : '';
        }
        const introEl = this.el('Intro');
        if (introEl) {
            introEl.textContent = this.mode === 'production'
                ? 'Every imported variable needs a value on this host before the application is published. '
                  + 'Choose the shared variable published to this host, or enter the value.'
                : 'Every variable needs a value before the application starts. Variables '
                  + 'imported from another application are filled from the shared variable '
                  + 'you choose here.';
        }
        this.showError(error);
        this.render();

        const modal = this.el('Modal');
        if (modal) {
            modal.classList.remove('hidden');
            modal.classList.add('flex');
        }
        if (this.mode === 'development' && this.rows.some((row) =>
            row.pending && (row.name === 'AI_BASE_URL' || row.name === 'AI_CHAT_MODEL'))) {
            void this.loadSuggestions(opening);
        }
    };

    ImportResolver.prototype.loadSuggestions = async function (opening) {
        try {
            const response = await fetch(this.url() + '?suggestions=1');
            if (!response.ok) return;
            const data = await response.json();
            if (opening !== this.opening) return;
            this.suggestions = data.suggestions || {};
            this.renderSuggestions();
        } catch {
            // Manual entry and shared references remain available offline.
        }
    };

    ImportResolver.prototype.renderSuggestions = function () {
        const url = this.suggestions.baseURL;
        const urlRow = this.rows.find((row) => row.name === 'AI_BASE_URL');
        const selectedURL = this.values.AI_BASE_URL ?? urlRow?.value ?? '';
        const usesSharedURL = !!this.choices.AI_BASE_URL &&
            this.choices.AI_BASE_URL !== (urlRow?.source || urlRow?.suggested);
        const matchesURL = !usesSharedURL && (!selectedURL.trim() ||
            selectedURL.trim().replace(/\/+$/, '') === url);
        const options = {
            AI_BASE_URL: url ? [url] : [],
            AI_CHAT_MODEL: matchesURL ? this.suggestions.models || [] : []
        };
        for (const [name, values] of Object.entries(options)) {
            const list = this.el('Suggestions-' + name);
            if (!list) continue;
            list.replaceChildren(...values.map((value) => {
                const option = document.createElement('option');
                option.value = value;
                return option;
            }));
        }
    };

    ImportResolver.prototype.close = function () {
        ++this.opening;
        const modal = this.el('Modal');
        if (modal) {
            modal.classList.add('hidden');
            modal.classList.remove('flex');
        }
    };

    ImportResolver.prototype.showError = function (message) {
        const el = this.el('Error');
        if (!el) return;
        if (!message) {
            el.classList.add('hidden');
            return;
        }
        el.textContent = message;
        el.classList.remove('hidden');
    };

    ImportResolver.prototype.render = function () {
        const body = this.el('Body');
        if (!body) return;
        const g = this.globalName();

        if (!this.rows.length) {
            body.innerHTML = '<div class="text-xs text-[color:var(--nu-muted)] p-3">Nothing to resolve.</div>';
            return;
        }

        body.innerHTML = this.rows.map((row) => {
            const label = `<div class="nu-mono text-xs font-semibold">${escapeHTML(row.name)}</div>`;
            let control;
            const editable = this.mode === 'development' && row.pending &&
                ['AI_BASE_URL', 'AI_CHAT_MODEL', 'AI_API_KEY'].includes(row.name);
            const literalID = this.prefix + 'Value-' + row.name;
            const choiceID = this.prefix + 'Choice-' + row.name;
            const suggestionsID = this.prefix + 'Suggestions-' + row.name;

            if (row.matches && row.matches.length) {
                // A wildcard: the user picks which producer feeds it. The current
                // source is preselected so re-opening the popup does not silently
                // re-point an already resolved variable.
                const current = row.source || row.suggested;
                const options = ['<option value="">Choose a shared variable…</option>'].concat(
                    row.matches.map((m) => {
                        const selected = m === current ? ' selected' : '';
                        return `<option value="${escapeHTML(m)}"${selected}>${escapeHTML(m)}</option>`;
                    })
                ).join('');
                control = `
                    <select ${editable ? `id="${choiceID}" aria-label="${row.name} shared variable"` : ''}
                        onchange="${g}.setChoice('${escapeHTML(row.name)}', this.value)"
                        class="nu-input nu-mono text-xs w-full">${options}</select>
                    <div class="text-[11px] text-[color:var(--nu-muted)] mt-1">
                      matches <span class="nu-code">${escapeHTML(row.pattern)}</span>
                    </div>`;
            } else if (editable) {
                control = '';
            } else if (row.pattern && !row.source) {
                // Declared as an import but nothing matches: the producer is not
                // installed here. Typing the value is the only way forward, and
                // without it the app could never be launched.
                control = `
                    <input type="text" value="${escapeHTML(row.value || '')}"
                        oninput="${g}.setValue('${escapeHTML(row.name)}', this.value)"
                        placeholder="No shared variable matches — enter the value"
                        class="nu-input nu-mono text-xs w-full" />
                    <div class="text-[11px] text-amber-500 mt-1">
                      nothing matches <span class="nu-code">${escapeHTML(row.pattern)}</span>
                    </div>`;
            } else if (row.pending) {
                control = `
                    <input type="text" value="${escapeHTML(row.value || '')}"
                        oninput="${g}.setValue('${escapeHTML(row.name)}', this.value)"
                        placeholder="Enter a value"
                        class="nu-input nu-mono text-xs w-full" />`;
            } else {
                // Already resolved: shown so the popup states the whole picture,
                // not only what is missing.
                const from = row.source
                    ? `<div class="text-[11px] text-[color:var(--nu-muted)] mt-1">from <span class="nu-code">${escapeHTML(row.source)}</span></div>`
                    : '';
                control = `
                    <div class="nu-mono text-xs text-[color:var(--nu-muted)] px-2 py-1">${escapeHTML(maskValue(row.value))}</div>
                    ${from}`;
            }
            if (editable) {
                const secret = row.name === 'AI_API_KEY';
                control += `
                    <input id="${literalID}" type="${secret ? 'password' : 'text'}"
                        ${secret ? 'autocomplete="off"' : `list="${suggestionsID}"`}
                        aria-label="${row.name} value"
                        oninput="${g}.setValue('${row.name}', this.value)"
                        placeholder="${row.matches?.length ? 'Or enter a value' : 'Enter a value'}"
                        class="nu-input nu-mono text-xs w-full" />
                    ${secret ? '' : `<datalist id="${suggestionsID}"></datalist>`}`;
            }

            const flag = row.pending
                ? `<span class="text-[11px] text-amber-500 ml-2">${row.suggested ? 'confirm' : 'needs a value'}</span>`
                : '';
            // Production: the producer chosen in development has nothing on this
            // host yet. Publishing it there first is the usual way out.
            if (row.producer) {
                control += `
                    <div class="text-[11px] text-amber-500 mt-1">
                      publish <strong>${escapeHTML(row.producer)}</strong> to this host first, or enter the value
                    </div>`;
            }
            return `
                <div class="nu-card p-2 mb-2">
                  <div class="flex items-center mb-1">${label}${flag}</div>
                  ${control}
                </div>`;
        }).join('');
        this.renderSuggestions();
    };

    ImportResolver.prototype.setChoice = function (name, source) {
        if (source) {
            this.choices[name] = source;
            delete this.values[name];
            const input = this.el('Value-' + name);
            if (input) input.value = '';
        } else {
            delete this.choices[name];
        }
        if (name === 'AI_BASE_URL') this.renderSuggestions();
    };

    ImportResolver.prototype.setValue = function (name, value) {
        this.values[name] = value;
        delete this.choices[name];
        const select = this.el('Choice-' + name);
        if (select) select.value = '';
        if (name === 'AI_BASE_URL') this.renderSuggestions();
    };

    // Every variable must end up with a value, so the popup refuses to confirm
    // while one is still blank rather than letting the launch fail later.
    ImportResolver.prototype.unresolved = function () {
        return this.rows.filter((row) => {
            if (!row.pending) return false;
            if (this.choices[row.name]) return false;
            const typed = this.values[row.name];
            return !(typed && typed.trim());
        }).map((row) => row.name);
    };

    ImportResolver.prototype.confirm = async function () {
        if (this.saving) return false;
        this.showError('');
        const missing = this.unresolved();
        if (missing.length) {
            this.showError('Still missing a value: ' + missing.join(', '));
            return false;
        }

        const opening = this.opening;
        this.saving = true;
        try {
            const response = await fetch(this.url(), {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ choices: this.choices, values: this.values })
            });
            const data = await response.json().catch(() => ({}));
            if (opening !== this.opening) return false;
            if (!response.ok) {
                this.showError(data.error || 'Failed to save');
                return false;
            }
            this.close();
            if (this.onResolved) this.onResolved(this.appName);
            return true;
        } catch (err) {
            if (opening === this.opening) this.showError('Failed to save: ' + err.message);
            return false;
        } finally {
            if (opening === this.opening) this.saving = false;
        }
    };

    ImportResolver.mount = function (globalName, options) {
        const resolver = new ImportResolver(options);
        resolver._globalName = globalName;
        global[globalName] = resolver;

        const prefix = resolver.prefix;
        const container = document.createElement('div');
        container.innerHTML = `
            <div id="${prefix}Modal" class="hidden fixed inset-0 nu-modal-backdrop items-center justify-center z-50 p-4">
              <div class="nu-modal w-full max-w-2xl max-h-[90vh] flex flex-col">
                <div class="flex items-center justify-between px-5 py-4 border-b border-[color:var(--nu-border)]">
                  <h3 class="text-lg font-semibold">Set the variables for <span id="${prefix}AppName" class="nu-mono"></span><span id="${prefix}Host" class="nu-mono"></span></h3>
                  <button onclick="${globalName}.close()" class="nu-btn nu-btn-secondary nu-btn-compact">Close</button>
                </div>
                <div id="${prefix}Intro" class="px-5 py-3 text-xs text-[color:var(--nu-muted)] border-b border-[color:var(--nu-border)]">
                  Every variable needs a value before the application starts. Variables
                  imported from another application are filled from the shared variable
                  you choose here.
                </div>
                <div id="${prefix}Error" class="hidden mx-5 mt-3 px-3 py-2 text-xs rounded bg-red-500/10 text-red-500"></div>
                <div id="${prefix}Body" class="flex-1 overflow-y-auto p-5"></div>
                <div class="flex gap-2 px-5 py-4 border-t border-[color:var(--nu-border)]">
                  <button onclick="${globalName}.confirm()" class="nu-btn nu-btn-primary flex-1">Continue</button>
                  <button onclick="${globalName}.close()" class="nu-btn nu-btn-secondary flex-1">Cancel</button>
                </div>
              </div>
            </div>`;
        document.body.appendChild(container.firstElementChild);
        return resolver;
    };

    global.ImportEditor = ImportEditor;
    global.ImportResolver = ImportResolver;
})(window);
