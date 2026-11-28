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
/* Guide kit: shared navigation and live probes for the Trustant creation-mode guides.
   No dependencies, no network calls except to the page's own origin and to the sibling guides. */
(function () {
  var GUIDES = [
    { app: 'guidaprovider', label: 'Provider AI' },
    { app: 'guidastarter', label: 'Starter proprio' },
    { app: 'guidatruchat', label: 'Starter TruChat' },
    { app: 'guidatemplate', label: 'Da un template' },
    { app: 'guidashared', label: 'Variabili condivise' },
    { app: 'guidapublish', label: 'Pubblicazione' },
    { app: 'provachecklist', label: 'App con l’assistente' }
  ];

  function siblingUrl(app) {
    var parts = location.host.split('.');
    parts[0] = app;
    return location.protocol + '//' + parts.join('.') + '/guida.html';
  }
  function currentApp() { return location.host.split('.')[0]; }
  function esc(s) { return String(s).replace(/[&<>"']/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]; }); }

  function renderNav() {
    var host = document.getElementById('gk-nav');
    if (!host) return;
    var cur = currentApp();
    var html = '<nav class="gk-nav" aria-label="Guide Trustant"><ul><li><strong>Guide Trustant</strong></li>';
    GUIDES.forEach(function (g) {
      html += '<li><a href="' + esc(siblingUrl(g.app)) + '"' + (g.app === cur ? ' aria-current="page"' : '') + '>' + esc(g.label) + '</a></li>';
    });
    host.innerHTML = html + '</ul></nav>';
  }

  function timeIt(url) {
    var t0 = performance.now();
    return fetch(url, { cache: 'no-store' }).then(function (r) {
      return { r: r, ms: Math.round(performance.now() - t0) };
    });
  }

  /* Live facts about this very deployment: nothing is hard-coded. */
  function renderProbe() {
    var el = document.getElementById('gk-probe');
    if (!el) return;
    el.textContent = 'Lettura in corso…';
    timeIt(location.pathname + location.search).then(function (x) {
      var h = x.r.headers;
      var rows = [
        ['Indirizzo', location.origin],
        ['Applicazione (primo nome dell’host)', currentApp()],
        ['Risposta', x.r.status + ' ' + x.r.statusText + ' in ' + x.ms + ' ms'],
        ['Content-Type', h.get('content-type') || '—'],
        ['Dimensione', h.get('content-length') || '—'],
        ['Server', h.get('server') || '—'],
        ['Cache-Control', h.get('cache-control') || '—'],
        ['Ultima modifica', h.get('last-modified') || '—'],
        ['ETag', h.get('etag') || '—']
      ];
      var t = '<table><tbody>';
      rows.forEach(function (r) { t += '<tr><th scope="row">' + esc(r[0]) + '</th><td><code>' + esc(r[1]) + '</code></td></tr>'; });
      el.innerHTML = t + '</tbody></table><p class="live">Letto adesso dal browser con una richiesta alla pagina stessa. Se ricarichi dopo un deploy, cambiano data e ETag.</p>';
    }).catch(function (e) { el.innerHTML = '<p class="bad">Lettura non riuscita: ' + esc(e.message) + '</p>'; });
  }

  /* Reachability of the other guides (opaque cross-origin: only "reachable or not"). */
  function renderSiblings() {
    var el = document.getElementById('gk-siblings');
    if (!el) return;
    var html = '<table><thead><tr><th>Guida</th><th>Stato</th></tr></thead><tbody>';
    GUIDES.forEach(function (g) { html += '<tr><td>' + esc(g.label) + ' (<code>' + g.app + '</code>)</td><td id="gk-s-' + g.app + '">controllo…</td></tr>'; });
    el.innerHTML = html + '</tbody></table>';
    GUIDES.forEach(function (g) {
      var cell = document.getElementById('gk-s-' + g.app);
      fetch(siblingUrl(g.app), { mode: 'no-cors', cache: 'no-store' }).then(function () {
        cell.innerHTML = '<span class="ok">raggiungibile</span>';
      }).catch(function () { cell.innerHTML = '<span class="bad">non raggiungibile</span>'; });
    });
  }

  document.addEventListener('DOMContentLoaded', function () {
    renderNav(); renderProbe(); renderSiblings();
  });
  window.GuideKit = { esc: esc, renderProbe: renderProbe };
})();
