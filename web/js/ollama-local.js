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

// Status line and Start/Stop button for the Ollama service that runs next to
// Trustant (ollama.service, see /api/ollama-local). It is only shown when the
// Private AI endpoint points at this machine.
(function () {
    'use strict';

    const LOCAL_HOSTS = new Set(['localhost', '127.0.0.1', '[::1]', '::1']);

    function isLocalBaseUrl(value) {
        try {
            return LOCAL_HOSTS.has(new URL(String(value || '').trim()).hostname);
        } catch (e) {
            return false;
        }
    }

    function describe(st) {
        if (!st.active) return 'Stopped';
        if (!st.reachable) return 'Starting…';
        const n = Number(st.models) || 0;
        return 'Running · ' + n + (n === 1 ? ' model' : ' models');
    }

    // mount fills `root` with the control and returns { refresh, setVisible }.
    // The control stays hidden when this host has no controllable service.
    // opts.onChange runs after a successful Start/Stop, so the caller can
    // re-read the models the service now serves.
    function mount(root, opts) {
        let supported = false;
        let wanted = true;
        let busy = false;
        let status = null;

        const label = document.createElement('span');
        label.className = 'text-sm font-medium';
        const state = document.createElement('span');
        state.className = 'text-sm text-[color:var(--nu-muted)]';
        state.setAttribute('role', 'status');
        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'nu-btn nu-btn-secondary';
        button.style.minWidth = '5.5rem'; // keeps its place when the label flips Start/Stop
        const error = document.createElement('p');
        error.setAttribute('role', 'alert');
        error.className = 'hidden w-full text-sm text-[color:var(--nu-danger)]';
        label.textContent = 'Local Ollama' + ' (WSL)';
        root.classList.add('flex', 'flex-wrap', 'items-center', 'gap-3');
        root.replaceChildren(label, state, button, error);

        function render() {
            root.classList.toggle('hidden', !(supported && wanted) && !(status && status.state === 'masked' && wanted));
            if (!status) return;
            label.textContent = status.wsl ? 'Local Ollama (WSL)' : 'Local Ollama';
            state.textContent = busy ? 'Working…' : describe(status);
            button.textContent = status.active ? 'Stop' : 'Start';
            button.disabled = busy || !supported;
            button.classList.toggle('hidden', !supported);
            const message = status.error || '';
            error.textContent = message;
            error.classList.toggle('hidden', !message);
        }

        async function call(method, body) {
            const resp = await fetch('/api/ollama-local', {
                method,
                headers: body ? { 'Content-Type': 'application/json' } : undefined,
                body: body ? JSON.stringify(body) : undefined,
                signal: AbortSignal.timeout(40000),
            });
            let data = null;
            try { data = await resp.json(); } catch (e) { /* non-JSON error body */ }
            if (!data) data = { error: 'Unexpected response (' + resp.status + ').' };
            return data;
        }

        async function refresh() {
            try {
                status = await call('GET');
                supported = !!status.supported;
            } catch (e) {
                status = { supported: false, error: '' };
                supported = false;
            }
            render();
        }

        button.addEventListener('click', async () => {
            if (busy || !status) return;
            busy = true;
            render();
            try {
                status = await call('POST', { action: status.active ? 'stop' : 'start' });
                supported = !!status.supported;
                if (!status.error && opts && typeof opts.onChange === 'function') {
                    busy = false;
                    render();
                    await opts.onChange(status);
                }
            } catch (e) {
                status = Object.assign({}, status, { error: 'The request failed or timed out.' });
            }
            busy = false;
            render();
        });

        return {
            refresh,
            setVisible(visible) {
                wanted = !!visible;
                render();
            },
        };
    }

    window.TrustantOllamaLocal = { isLocalBaseUrl, mount };
})();
