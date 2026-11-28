#!/usr/bin/env python3
# Copyright 2025-2026 Nuvolaris Inc
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as published
# by the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.

"""Generate guida.html for each guide app from one template and shared sections."""
import os, sys

OUT = sys.argv[1]  # directory receiving <app>/guida.html

PAGE = """<!doctype html>
<html lang="it">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>@@TITLE@@ · Guida Trustant</title>
<link rel="stylesheet" href="guide-kit.css">
</head>
<body>
<div id="gk-nav"></div>
<main class="gk">
  <p class="tag">@@TAG@@</p>
  <h1>@@H1@@</h1>
  <p class="lead">@@LEAD@@</p>

  <section aria-labelledby="s-intento">
    <h2 id="s-intento">1. Intento</h2>
    @@INTENTO@@
  </section>

  <section aria-labelledby="s-tu">
    <h2 id="s-tu">2. Cosa fai tu</h2>
    @@TU@@
  </section>

  <section aria-labelledby="s-sotto">
    <h2 id="s-sotto">3. Cosa succede sotto</h2>
    @@SOTTO@@
    @@DEMO@@
  </section>

  <section aria-labelledby="s-risultato">
    <h2 id="s-risultato">4. Risultato</h2>
    @@RISULTATO@@
  </section>

  <section aria-labelledby="s-verifica">
    <h2 id="s-verifica">5. Verifica dal vivo</h2>
    <p>Questa tabella non è scritta a mano: il browser la legge adesso da questa stessa pubblicazione.</p>
    <div id="gk-probe" aria-live="polite"></div>
    <h3>Le altre guide</h3>
    <div id="gk-siblings"></div>
    @@VERIFICA@@
  </section>

  <footer>Fonti: @@FONTI@@ · Le righe tra virgolette nella sezione 3 sono messaggi reali del log del server.</footer>
</main>
<script src="guide-kit.js"></script>
@@SCRIPT@@
</body>
</html>
"""

LAUNCH = """
<ol class="timeline">
  <li><strong>Copia di lavoro.</strong> Il server clona il repository di lavoro nel workbench.
    <small>«Cloning workspace/&lt;app&gt; to workbench/&lt;app&gt;...»</small></li>
  <li><strong>Dipendenze.</strong> Installa i pacchetti del progetto.
    <small>«Running npm install in workbench/&lt;app&gt;...»</small></li>
  <li><strong>Skill dell’assistente.</strong> Scarica <code>trustable-ai/skills</code> (prima via SSH, poi via HTTPS se la chiave non è registrata su GitHub) e le registra nel repository di lavoro.
    <small>«SSH clone of skills failed, falling back to HTTPS» · «Skills pushed»</small></li>
  <li><strong>Utente OpenServerless.</strong> Ogni app ha il suo utente; <code>ops ide login</code> scrive le credenziali dei servizi.
    <small>«Checking OpenWhisk user for &lt;app&gt;...» · «Running ops ide login for &lt;app&gt;...»</small></li>
  <li><strong>Deploy.</strong> Pulisce e pubblica nel cluster locale <code>web/</code> e le action.
    <small>«Running ops ide clean ...» · «Running ops ide deploy ...»</small></li>
  <li><strong>Servizi.</strong> Parte l’assistente (TruACP, porta 4096) e il dev server (porta 5173).
    <small>«Services for &lt;app&gt; started - truacp on port 4096, opsdevel on port 5173»</small></li>
  <li><strong>Indirizzo.</strong> Un proxy sulla porta 8080 riscrive l’host <code>&lt;app&gt;.&lt;ip&gt;.nip.io</code> verso <code>&lt;app&gt;.miniops.me</code>, dove risponde OpenWhisk.</li>
</ol>
"""

PAGES = {}

# ---------------------------------------------------------------- guidastarter
PAGES["guidastarter"] = dict(
    TITLE="Starter proprio", TAG="Modalità · New application → My Application Starter",
    H1="Partire da uno starter tuo",
    LEAD="Parti da un repository GitHub che scegli tu, compatibile con le convenzioni di Trustant, e costruisci le funzioni con l’assistente.",
    INTENTO="""<p>È la modalità per chi ha già uno scheletro (o vuole tenere il codice in un proprio repository) invece di usare uno starter del catalogo.
Trustant non scrive il codice per te: crea l’ambiente, collega il repository e ti lascia lavorare con l’assistente.</p>
<p class="note warn">Lo starter deve rispettare le «Trustant Conventions». Il repository deve esistere e l’account GitHub collegato deve poterlo leggere.</p>""",
    TU="""<ol>
<li>Dall’elenco applicazioni scegli <strong>Add App</strong> → <strong>New application</strong>.</li>
<li>Seleziona <strong>My Application Starter</strong> e leggi l’avviso: premi <em>Understood</em>.</li>
<li>Scrivi il <strong>nome</strong> (6–20 caratteri alfanumerici, inizia con una lettera) e il <strong>repository</strong> nel formato <code>org/repo</code>.</li>
<li>Premi <strong>Create</strong>: la creazione richiede circa un minuto.</li>
<li>Apri l’app con <strong>Edit</strong>: parte il lancio e compare l’anteprima accanto all’assistente.</li>
</ol>""",
    SOTTO="""<p><strong>Creazione</strong> (alla pressione di Create):</p>
<ol class="timeline">
  <li><strong>Lettura del repository.</strong> Trustant lo legge via HTTPS con l’account GitHub collegato (l’avviso dice che legge e scrive con quell’account).</li>
  <li><strong>Repository di lavoro.</strong> Crea la copia durevole in <code>workspace/workspace/&lt;nome&gt;</code>, un repository <em>bare</em>, e registra l’app nella configurazione del workspace.</li>
  <li><strong>Variabili condivise.</strong> Importa nel pool quelle dichiarate dall’app.
    <small>«Runtime lifecycle: ... import shared variables &lt;nome&gt;»</small></li>
</ol>
<p><strong>Primo lancio</strong> (alla pressione di Edit): il workbench non esiste ancora e viene creato.</p>""" + LAUNCH,
    DEMO="""<h3>Prova: il tuo nome e il tuo repository</h3>
<p>Le stesse regole applicate dal modulo, e i percorsi e l’indirizzo che ne derivano.</p>
<div class="grid2"><div><label for="d-name">Nome applicazione</label><input id="d-name" type="text" value="mioprogetto" autocomplete="off"></div>
<div><label for="d-repo">Repository</label><input id="d-repo" type="text" value="mia-org/mio-starter" autocomplete="off"></div></div>
<div id="d-out" class="out" aria-live="polite"></div>""",
    RISULTATO="""<p>Un’app web statica servita da <code>tools/dev.cjs</code> (senza dipendenze): <code>web/index.html</code> è la pagina, <code>packages/</code> è vuota e non ci sono action, database o login.
Gli script sono <code>dev</code>, <code>lint</code> e <code>build</code>. Il dettaglio file per file è nel <a href="riferimento.html">riferimento tecnico</a>.</p>
<p><a href="/">Apri la home dell’app</a>.</p>""",
    VERIFICA="""<p>Da una shell nel workbench: <code>npm run lint</code> (controlla la sintassi del launcher) e <code>npm run build</code> (controlla che <code>web/index.html</code> esista).</p>""",
    FONTI="spec/1-applist.md, spec/2-repo.md, spec/4-launch.md",
    SCRIPT="""<script>
(function () {
  var name = document.getElementById('d-name'), repo = document.getElementById('d-repo'), out = document.getElementById('d-out');
  function host(app) { var p = location.host.split('.'); p[0] = app; return location.protocol + '//' + p.join('.'); }
  function run() {
    var n = name.value.trim(), r = repo.value.trim(), e = GuideKit.esc, html = '';
    var okName = /^[a-zA-Z][a-zA-Z0-9]{5,19}$/.test(n), okRepo = /^[A-Za-z0-9_.-]+\\/[A-Za-z0-9_.-]+$/.test(r);
    html += '<p>Nome: <span class="' + (okName ? 'ok' : 'bad') + '">' + (okName ? 'valido' : 'non valido (6–20 alfanumerici, inizia con una lettera)') + '</span> · ';
    html += 'Repository: <span class="' + (okRepo ? 'ok' : 'bad') + '">' + (okRepo ? 'formato valido' : 'usa il formato org/repo') + '</span></p>';
    if (okName) html += '<table><tbody><tr><th scope="row">Repository di lavoro</th><td><code>workspace/workspace/' + e(n) + '</code></td></tr>' +
      '<tr><th scope="row">Workbench (dopo Edit)</th><td><code>workbench/' + e(n) + '</code></td></tr>' +
      '<tr><th scope="row">Indirizzo</th><td><code>' + e(host(n)) + '</code></td></tr>' +
      '<tr><th scope="row">Host riscritto verso</th><td><code>' + e(n) + '.miniops.me</code></td></tr></tbody></table>';
    out.innerHTML = html;
  }
  name.addEventListener('input', run); repo.addEventListener('input', run); run();
})();
</script>""")

# ---------------------------------------------------------------- provachecklist
PAGES["provachecklist"] = dict(
    TITLE="App con l’assistente", TAG="Modalità · lavorare con l’assistente su uno starter",
    H1="Che cosa fa l’assistente sopra lo starter",
    LEAD="Parti dallo scheletro e descrivi l’app a parole: l’assistente scrive i file, tu guardi l’anteprima e decidi cosa tenere.",
    INTENTO="""<p>Qualunque modalità di creazione finisce qui: l’app esiste, è vuota, e la costruisci chiedendo. Questa app, una <em>Checklist di collaudo</em>, è stata costruita così
sopra lo starter proprio; la sua home è il risultato.</p>""",
    TU="""<ol>
<li>Apri l’app con <strong>Edit</strong>: a sinistra c’è l’assistente, a destra l’anteprima.</li>
<li>Descrivi cosa vuoi, per esempio: «una checklist di collaudo in italiano, con lo stato salvato solo nel browser».</li>
<li>Guarda l’anteprima aggiornarsi; correggi chiedendo.</li>
<li>Quando va bene, <strong>Commit</strong>: la barra indica quante modifiche ci sono (<em>N changes</em>).</li>
</ol>""",
    SOTTO="""<p>Il primo lancio crea l’ambiente (workbench, utente OpenServerless, dev server):</p>""" + LAUNCH + """
<p>Poi l’assistente lavora direttamente nel workbench. I file che ha prodotto qui, dal README dell’app:</p>
<table><thead><tr><th>File</th><th>Ruolo</th></tr></thead><tbody>
<tr><td><code>web/index.html</code></td><td>la home: titolo, form «Nuova attività», elenco, filtri e le sezioni descrittive</td></tr>
<tr><td><code>web/styles.css</code></td><td>tema chiaro/scuro automatico, stati di fuoco visibili</td></tr>
<tr><td><code>web/app.js</code></td><td>aggiunta, spunta, eliminazione, filtri, persistenza in <code>localStorage</code>; nessuna chiamata di rete</td></tr>
<tr><td><code>tools/check-app.cjs</code> + <code>dom-stub.cjs</code></td><td>controlli locali su sintassi, struttura, accessibilità e comportamento</td></tr>
</tbody></table>""",
    DEMO="""<h3>Prova: dove stanno i dati</h3>
<p>L’app non ha un server per i dati: li tiene nel tuo browser, sotto la chiave <code>collaudo.checklist.v1</code>.</p>
<button type="button" id="d-read">Leggi localStorage ora</button>
<pre id="d-out" class="out" aria-live="polite"><code>(premi il pulsante)</code></pre>""",
    RISULTATO="""<p><a href="/">Apri la checklist</a>: aggiungi una voce, spuntala, ricarica. Ricomparirà com’era, perché è salvata in questo browser e solo in questo.</p>""",
    VERIFICA="",
    FONTI="README dell’app, spec/3-app.md, spec/4-launch.md",
    SCRIPT="""<script>
document.getElementById('d-read').addEventListener('click', function () {
  var v = null;
  try { v = localStorage.getItem('collaudo.checklist.v1'); } catch (e) { v = 'localStorage non disponibile: ' + e.message; }
  document.getElementById('d-out').firstChild.textContent = v === null ? 'Nessun dato: apri la checklist e aggiungi una voce.' : v;
});
</script>""")

# ---------------------------------------------------------------- guidatruchat
PAGES["guidatruchat"] = dict(
    TITLE="Starter TruChat", TAG="Modalità · New application → starter del catalogo (truchat)",
    H1="Partire da uno starter con la funzione già pronta",
    LEAD="Lo starter «truchat» è una chat con RAG: scegli il modello, apri l’anteprima e la conversazione funziona già.",
    INTENTO="""<p>Gli starter del catalogo (<code>trustable-ai</code>) sono repository il cui nome è mostrato nel modulo di creazione con una breve descrizione.
Qui lo starter porta già con sé interfaccia e backend; tu cambi solo ciò che serve.</p>""",
    TU="""<ol>
<li><strong>Add App</strong> → <strong>New application</strong> → scegli <strong>truchat — AI chat UI with RAG</strong>.</li>
<li>Dai un nome e premi <strong>Create</strong>; poi <strong>Edit</strong>.</li>
<li>Il modello si imposta dalle variabili dell’app (<strong>Env</strong>): endpoint, nome del modello e chiave.</li>
<li>Apri la home dell’app e scrivi: la risposta arriva a flusso.</li>
</ol>""",
    SOTTO="""<p>Il lancio è lo stesso di ogni app:</p>""" + LAUNCH + """
<p>Poi, quando apri la chat:</p>
<ol class="timeline">
  <li><strong>Benvenuto.</strong> Appena la pagina si carica, invia una richiesta vuota a <code>/stream/web/v1/chat</code> e mostra la risposta: host, modello e numero di istruzioni RAG.</li>
  <li><strong>Backend.</strong> Una action Python (<code>packages/v1/chat</code>) risponde su quella rotta: riceve <code>AI_BASE_URL</code>, <code>AI_CHAT_MODEL</code> e <code>AI_API_KEY</code> come parametri, dichiarati in <code>__main__.py</code> e presi dall’ambiente dell’app, e inoltra al modello.</li>
  <li><strong>Flusso.</strong> La risposta arriva a righe: ogni riga è una stringa JSON che la pagina accoda al testo.</li>
  <li><strong>RAG.</strong> Il pulsante RAG invia il comando <code>@</code> senza mostrarlo nella conversazione.</li>
</ol>""",
    DEMO="""<h3>Prova: la richiesta di benvenuto, in chiaro</h3>
<p>Premi il pulsante: la pagina fa la stessa richiesta della chat e ti mostra le righe grezze, poi il testo ricomposto. Nessuna chiave compare: sta nel backend.</p>
<button type="button" id="d-go">Invia la richiesta di benvenuto</button>
<pre id="d-raw" class="out" aria-live="polite"><code>(in attesa)</code></pre>
<p id="d-text" class="out"></p>""",
    RISULTATO="""<p><a href="/">Apri la chat</a>. Il titolo, il benvenuto e i pulsanti Reset e RAG vengono dallo starter.</p>""",
    VERIFICA="",
    FONTI="README e sorgenti dell’app, spec/1-applist.md, spec/15-starters.md",
    SCRIPT="""<script>
document.getElementById('d-go').addEventListener('click', function () {
  var raw = document.getElementById('d-raw').firstChild, text = document.getElementById('d-text');
  var ctl = new AbortController(), timer = setTimeout(function () { ctl.abort(); }, 60000);
  raw.textContent = 'Richiesta in corso…'; text.textContent = '';
  var t0 = performance.now();
  fetch('/stream/web/v1/chat', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ input: '', messages: [] }), signal: ctl.signal })
    .then(function (r) {
      var head = 'HTTP ' + r.status + ' · ' + r.headers.get('content-type') + '\\n';
      if (!r.ok) throw new Error(head);
      return r.text().then(function (body) { return { head: head, body: body }; });
    })
    .then(function (x) {
      clearTimeout(timer);
      var out = x.head + 'Tempo: ' + Math.round(performance.now() - t0) + ' ms\\n\\n' + x.body, acc = '';
      raw.textContent = out;
      x.body.split('\\n').forEach(function (l) { if (!l.trim()) return; try { var p = JSON.parse(l); if (typeof p === 'string') acc += p; } catch (e) {} });
      text.textContent = acc ? 'Testo ricomposto: ' + acc.replace(/\\n/g, ' · ') : 'Nessun testo nel flusso.';
    })
    .catch(function (e) { clearTimeout(timer); raw.textContent = 'Errore: ' + (e.name === 'AbortError' ? 'nessuna risposta entro 60 secondi' : e.message); });
});
</script>""")

# ---------------------------------------------------------------- guidatemplate
PAGES["guidatemplate"] = dict(
    TITLE="Da un template", TAG="Modalità · From a template",
    H1="Copiare un’app che già funziona",
    LEAD="Invece di uno scheletro vuoto, parti da un’applicazione completa del catalogo: la copi, la esegui e la rendi tua.",
    INTENTO="""<p>«New application» crea un’app <em>nuova</em> da uno starter. «From a template» copia un’app <em>esistente e funzionante</em>.
Qui il template scelto è <strong>Document Ingest</strong> (TruIngest): carichi un PDF e lo porti, passo dopo passo, fino a dati puliti e suddivisi in parti.</p>""",
    TU="""<ol>
<li><strong>Add App</strong> → pulsante <strong>From a template</strong> in alto.</li>
<li>Scegli il gruppo (Demo, Examples, Utilities) e scorri le anteprime.</li>
<li>Premi <strong>Use template</strong>: il modulo mostra «Creating a copy of … from &lt;repository&gt;».</li>
<li>Scegli il <strong>nome</strong> della tua copia e premi <strong>Confirm</strong>.</li>
</ol>""",
    SOTTO="""<ol class="timeline">
  <li><strong>Il catalogo.</strong> L’elenco viene da un file statico pubblicato (<code>index.json</code>), letto via HTTPS, con una cache di 5 minuti. Trustant non chiama l’API di GitHub.</li>
  <li><strong>Pulizia.</strong> Il server ricontrolla ogni voce: ne scarta alcune e normalizza le altre (prova qui sotto).</li>
  <li><strong>Copia.</strong> L’app viene creata dal repository della voce scelta, che è fisso e pubblico (qui <code>trustable-ai/truingest</code>); il nome lo decidi tu, con le stesse regole degli altri flussi.</li>
</ol>
<p>Poi, al primo <strong>Edit</strong>, il lancio:</p>""" + LAUNCH,
    DEMO="""<h3>Prova: come il server pulisce una voce del catalogo</h3>
<p>Modifica il JSON: vedi se la voce sopravvive e come viene riscritta.</p>
<label for="d-in">Voce</label>
<textarea id="d-in" spellcheck="false">{
  "name": "truingest",
  "title": "  Document   Ingest ",
  "repo": "https://github.com/trustable-ai/truingest.git",
  "icon": "https://trustant.ai/images/trustable-ai-truingest.png",
  "description": "Turn a PDF   into chunks"
}</textarea>
<div id="d-out" class="out" aria-live="polite"></div>""",
    RISULTATO="""<p>Una copia completa di TruIngest: un’app React con azioni Python su OpenServerless, che porta un PDF attraverso cinque passi (Upload, Extract, Clean, Chunk, Export), uno alla volta e con il risultato di ognuno visibile. <a href="/">Aprila</a>; la guida che stai leggendo sta accanto, a <code>/guida.html</code>.</p>""",
    VERIFICA="",
    FONTI="spec/15-starters.md, spec/1-applist.md, README dell’app",
    SCRIPT="""<script>
(function () {
  var inp = document.getElementById('d-in'), out = document.getElementById('d-out');
  function clean(o) {
    var why = [];
    if (!o || typeof o !== 'object') return { why: ['non è un oggetto'] };
    var name = String(o.name || '').trim();
    if (!name) return { why: ['senza name: scartata'] };
    var repo = String(o.repo || '').trim().replace(/^https:\\/\\/github\\.com\\//, '').replace(/\\/+$/, '').replace(/\\.git$/, '');
    if (!/^[^\\/\\s]+\\/[^\\/\\s]+$/.test(repo)) return { why: ['repo non riducibile a owner/repository: scartata'] };
    var icon = String(o.icon || '').trim();
    if (icon && icon.indexOf('https://') !== 0) return { why: ['icon non https: scartata (evita immagini rotte o URL javascript:)'] };
    var sp = function (s) { return String(s || '').replace(/\\s+/g, ' ').trim(); };
    var title = sp(o.title) || name;
    if (!sp(o.title)) why.push('title vuoto: usa il name');
    if (!icon) why.push('icon vuota: tenuta, la tessera mostra un segnaposto');
    return { entry: { name: name, title: title, repo: repo, icon: icon, description: sp(o.description) }, why: why };
  }
  function run() {
    var o, e = GuideKit.esc;
    try { o = JSON.parse(inp.value); } catch (x) { out.innerHTML = '<p class="bad">JSON non valido: ' + e(x.message) + '</p>'; return; }
    var r = clean(o);
    out.innerHTML = (r.entry ? '<p class="ok">Voce accettata</p><pre><code>' + e(JSON.stringify(r.entry, null, 2)) + '</code></pre>' : '<p class="bad">Voce scartata</p>') +
      (r.why.length ? '<ul class="plain"><li>' + r.why.map(e).join('</li><li>') + '</li></ul>' : '');
  }
  inp.addEventListener('input', run); run();
})();
</script>""")

# ---------------------------------------------------------------- guidashared
PAGES["guidashared"] = dict(
    TITLE="Variabili condivise", TAG="Flusso · Share → Export e Import",
    H1="Far parlare due app senza copiare segreti",
    LEAD="Un’app pubblica certi valori con un nome stabile; un’altra li importa. Il valore non viene mai copiato a mano, quindi non invecchia.",
    INTENTO="""<p>Ogni app ha il proprio utente OpenServerless e <code>ops ide login</code> rigenera le credenziali a ogni accesso: dire a un’altra app «usa il mio database» una volta sola non basta, il valore diventa vecchio.
Le variabili condivise risolvono il problema con due metà: <strong>Export</strong> (pubblico) e <strong>Import</strong> (consumo).</p>""",
    TU="""<ol>
<li>Nell’elenco applicazioni premi <strong>Share</strong> sull’app che <em>produce</em>: scheda <strong>Export</strong>, scegli le voci dalla sua configurazione e salva.</li>
<li>Sull’app che <em>consuma</em>, <strong>Share</strong> → scheda <strong>Import</strong>: scrivi cosa ti serve (nome esatto, rinominato o con <code>*</code>).</li>
<li>Al lancio dell’app che consuma, se manca una scelta compare un popup con il suggerimento già selezionato: confermi con un clic.</li>
<li>Le voci condivise compaiono in <strong>Configure → Shared Variables</strong>.</li>
</ol>""",
    SOTTO="""<ol class="timeline">
  <li><strong>Export.</strong> Il file <code>.env.shared</code> contiene solo <em>puntatori</em> (<code>&lt;.postgres.url&gt;</code>), mai valori: è committato come <code>.env.dist</code>.</li>
  <li><strong>Risoluzione.</strong> Al salvataggio, all’installazione e <em>prima di ogni lancio</em> il server esegue un <code>ops ide login</code> per ogni app produttrice e legge i valori da <code>~/.ops/config.json</code>.</li>
  <li><strong>Il pool.</strong> I valori finiscono in un elenco unico del workspace (<code>predefined_env</code>), con nomi del tipo <code>APPSUITE__POSTGRES_URL</code>: il prefisso indica chi lo produce e impedisce collisioni.</li>
  <li><strong>Import.</strong> <code>.env.dist</code> dichiara cosa serve. La prima volta ogni voce è «in attesa», anche se c’è una sola candidata; confermando si scrive un riferimento <code>${{NOME}}</code> nella configurazione dell’app, non nel file committato.</li>
  <li><strong>Dopo.</strong> Il valore è riletto dal pool a ogni risoluzione: una credenziale ruotata arriva da sola. Se il produttore viene cancellato, la voce torna in attesa invece di puntare a un altro database.</li>
</ol>
<p class="note">Il pool contiene segreti veri e vive nel <code>trustant.json</code> locale, mai committato. I log riportano solo i nomi, mai i valori.</p>""",
    DEMO="""<h3>Prova: Export e Import con dati finti</h3>
<div class="grid2">
<div><label for="d-app">App produttrice</label><input id="d-app" type="text" value="appsuite" autocomplete="off">
<label for="d-shared"><code>.env.shared</code> della produttrice</label>
<textarea id="d-shared" spellcheck="false">&lt;app&gt;__POSTGRES_URL=&lt;.postgres.url&gt;
&lt;app&gt;__S3_KEY=&lt;.s3.access.key&gt;
&lt;app&gt;__DB=&lt;.postgres&gt;
PASSWORD=hunter2</textarea>
<label for="d-cfg"><code>~/.ops/config.json</code> (finto)</label>
<textarea id="d-cfg" spellcheck="false">{"postgres":{"url":"postgres://demo@db/app","host":"db"},"s3":{"access":{"key":"AKIA-DEMO"}}}</textarea></div>
<div><label for="d-dist"><code>.env.dist</code> della consumatrice</label>
<textarea id="d-dist" spellcheck="false">POSTGRES_URL=APPSUITE__POSTGRES_URL
S3_KEY=
EXT=*__S3_KEY
ALTRO=*__NOPE</textarea></div></div>
<div id="d-out" class="out" aria-live="polite"></div>""",
    RISULTATO="""<p>Due app che condividono un valore senza che nessuno lo digiti: il pool si aggiorna a ogni lancio e le voci si riconoscono dal prefisso.</p>""",
    VERIFICA="""<p>Nel workbench dell’app produttrice controlla che <code>.env.shared</code> non contenga valori reali: <code>sharedPathOf</code> li rifiuta, ma un valore scritto a mano sarebbe un segreto pubblicato.</p>""",
    FONTI="spec/18-shared.md, spec/19-import.md",
    SCRIPT="""<script>
(function () {
  var $ = function (id) { return document.getElementById(id); }, e = GuideKit.esc;
  function unesc(s) { return s.replace(/&lt;/g, '<').replace(/&gt;/g, '>'); }
  function lines(t) { return t.split('\\n').map(function (l) { return l.trim(); }).filter(function (l) { return l && l[0] !== '#'; }); }
  function resolve(app, shared, cfg) {
    var A = app.toUpperCase(), pool = [], errs = [];
    lines(shared).forEach(function (line) {
      var i = line.indexOf('=');
      if (i < 1) { errs.push('Riga non valida: ' + line); return; }
      var k = line.slice(0, i).trim(), v = line.slice(i + 1).trim();
      if (k.indexOf('<app>__') !== 0) { errs.push('«' + k + '»: deve iniziare con <app>__, altrimenti nel pool non si saprebbe chi la produce'); return; }
      var m = v.match(/^<\\.([A-Za-z0-9_.-]+)>$/);
      if (!m) { errs.push('«' + k + '»: il valore non è un puntatore <.percorso>; scritto qui sarebbe un segreto pubblicato'); return; }
      var cur = cfg;
      m[1].split('.').forEach(function (p) { cur = (cur && typeof cur === 'object' && p in cur) ? cur[p] : undefined; });
      if (cur === undefined) { errs.push('«' + k + '»: percorso ' + m[1] + ' non trovato (resta il valore precedente del pool)'); return; }
      if (typeof cur === 'object') { errs.push('«' + k + '»: ' + m[1] + ' è un gruppo di segreti, non un valore; si condividono solo le foglie'); return; }
      pool.push([A + '__' + k.slice(7), String(cur)]);
    });
    return { pool: pool, errs: errs };
  }
  function imports(dist, pool) {
    return lines(dist).map(function (line) {
      var i = line.indexOf('='), name = (i < 0 ? line : line.slice(0, i)).trim(), pat = i < 0 ? '' : line.slice(i + 1).trim(), c;
      if (!pat) pat = name;
      if (pat.indexOf('*') < 0) c = pool.filter(function (p) { return p[0] === pat; });
      else { var re = new RegExp('^' + pat.split('*').map(function (s) { return s.replace(/[.+?^${}()|[\\]\\\\]/g, '\\\\$&'); }).join('.*') + '$'); c = pool.filter(function (p) { return re.test(p[0]); }); }
      var st = c.length === 0 ? '<span class="bad">in attesa: nessuna corrispondenza</span>'
        : c.length === 1 ? 'in attesa con suggerimento <code>' + e(c[0][0]) + '</code>: un clic al primo lancio, poi riferimento <code>${{' + e(c[0][0]) + '}}</code>'
        : '<span class="bad">in attesa: più corrispondenze</span> (' + c.map(function (x) { return '<code>' + e(x[0]) + '</code>'; }).join(', ') + '): scegli tu';
      return '<tr><td><code>' + e(name) + '</code></td><td><code>' + e(line.slice(line.indexOf('=') + 1) || '(vuoto)') + '</code></td><td>' + st + '</td></tr>';
    });
  }
  function run() {
    var cfg; try { cfg = JSON.parse($('d-cfg').value); } catch (x) { $('d-out').innerHTML = '<p class="bad">config.json non valido: ' + e(x.message) + '</p>'; return; }
    var r = resolve($('d-app').value.trim() || 'app', unesc($('d-shared').value), cfg), h = '<h3>Il pool dopo la risoluzione</h3>';
    h += r.pool.length ? '<table><thead><tr><th>Nome nel pool</th><th>Valore</th></tr></thead><tbody>' + r.pool.map(function (p) { return '<tr><td><code>' + e(p[0]) + '</code></td><td><code>' + e(p[1].slice(0, 4)) + '…</code> (mascherato)</td></tr>'; }).join('') + '</tbody></table>' : '<p>Nessuna voce.</p>';
    if (r.errs.length) h += '<h3>Rifiutate o saltate</h3><ul class="plain"><li>' + r.errs.map(e).join('</li><li>') + '</li></ul>';
    var im = imports($('d-dist').value, r.pool);
    h += '<h3>Cosa succede alla consumatrice</h3><table><thead><tr><th>Variabile</th><th>Schema</th><th>Esito</th></tr></thead><tbody>' + im.join('') + '</tbody></table>';
    $('d-out').innerHTML = h;
  }
  ['d-app', 'd-shared', 'd-cfg', 'd-dist'].forEach(function (id) { $(id).addEventListener('input', run); });
  run();
})();
</script>""")

# ---------------------------------------------------------------- guidapublish
PAGES["guidapublish"] = dict(
    TITLE="Pubblicazione", TAG="Flusso · Publish, Redeploy e Clean",
    H1="Dal workbench al cluster di produzione",
    LEAD="Pubblicare non è un salvataggio: passa da controlli di autorizzazione, mostra ogni passo in diretta e non fa uscire le password.",
    INTENTO="""<p>Finché lavori sei in sviluppo: il cluster è locale e tutto è ricostruibile. <strong>Publish</strong> porta l’app verso un cluster di produzione.
<strong>Redeploy</strong> e <strong>Clean</strong> servono invece a rimettere in ordine l’ambiente di sviluppo.</p>""",
    TU="""<ol>
<li>Dall’elenco applicazioni premi <strong>Publish</strong> sull’app: si apre il modulo di pubblicazione con la destinazione OpenServerless.</li>
<li>Se manca una licenza valida compare la finestra della licenza; l’invio al repository Git (<em>push</em>) non la richiede.</li>
<li>Conferma: si apre la finestra di avanzamento e le righe arrivano mentre i comandi lavorano.</li>
<li>In sviluppo: <strong>Utils → Redeploy</strong> (o Maiusc+clic su Reload) per ripetere il deploy; <strong>Clean</strong> per ripartire pulito.</li>
</ol>""",
    SOTTO="""<h3>I controlli, nell’ordine</h3>
<ol class="timeline">
  <li><strong>Licenza.</strong> Prima di fare qualsiasi cosa il server verifica la firma e la scadenza. Se manca risponde 402: <code>License required: &lt;motivo&gt;</code>.</li>
  <li><strong>Configurazione di produzione.</strong> Se mancano valori, la pubblicazione torna al modulo (<code>needs_config</code>) e un valore che l’host di destinazione non ha la fa rifiutare.</li>
  <li><strong>Host.</strong> L’indirizzo di destinazione deve essere coperto dalla licenza, con confronto esatto su schema, host e porta, senza caratteri jolly. Altrimenti 402: <code>Host not licensed: &lt;apihost&gt; is not covered by your license</code>. Gli host locali (<code>miniops.me</code>, <code>localhost</code>, <code>127.0.0.1</code>, <code>::1</code>) saltano solo questo controllo.</li>
  <li><strong>Lavoro vero.</strong> Solo dopo i controlli parte <code>ops ide login --mode=production</code> e il deploy: nulla tocca il cluster se l’host non è autorizzato.</li>
</ol>
<h3>L’avanzamento in diretta</h3>
<p>Il server risponde con eventi (<em>server-sent events</em>) quando il browser lo chiede: <code>progress</code> (passo N di T), <code>output</code> (una riga completa di output), poi <code>done</code> o <code>error</code>. Lo stato HTTP è sempre 200 perché la risposta parte prima di sapere l’esito: l’esito è nell’evento finale.
La password di produzione è sostituita da <code>********</code> in ogni riga e nel risultato finale.</p>
<h3>Redeploy in sviluppo</h3>
<p>I passi, uno per evento: ferma <code>ops ide devel</code>, attende che la porta 5173 sia libera, <code>ops ide deploy</code>, elenca le action (<code>ops action list</code>), riavvia il dev server e attende che risponda.</p>""",
    DEMO="""<h3>Prova: i controlli di una pubblicazione</h3>
<div class="grid2">
<div><label for="d-host">Destinazione (apihost)</label><input id="d-host" type="text" value="https://api.esempio.it" autocomplete="off">
<label for="d-lic">Licenza</label><select id="d-lic"><option value="ok">valida</option><option value="none">assente</option><option value="expired">scaduta</option></select></div>
<div><label for="d-hosts">Host coperti dalla licenza (uno per riga)</label><textarea id="d-hosts" spellcheck="false">https://api.esempio.it
https://prod.esempio.it:8443</textarea></div></div>
<label><input type="checkbox" id="d-cfg" checked style="width:auto"> la configurazione di produzione è completa</label>
<div id="d-out" class="out" aria-live="polite"></div>""",
    RISULTATO="""<p>Una pubblicazione che si ferma <em>prima</em> di toccare il cluster quando manca un’autorizzazione, e che mostra a ogni passo cosa sta facendo senza rivelare segreti.</p>""",
    VERIFICA="""<p>Dopo un Redeploy riapri questa guida: nella tabella di sopra cambiano <em>Ultima modifica</em> ed <em>ETag</em>, segno che la nuova versione è davvero in linea.</p>""",
    FONTI="spec/6-publish.md, spec/3-app.md (Redeploy, Clean), spec/14-license.md",
    SCRIPT="""<script>
(function () {
  var $ = function (id) { return document.getElementById(id); }, e = GuideKit.esc;
  var LOCAL = ['miniops.me', 'localhost', '127.0.0.1', '::1'];
  function hostOf(u) { try { return new URL(u).hostname.replace(/^\\[|\\]$/g, ''); } catch (x) { return u; } }
  function run() {
    var host = $('d-host').value.trim(), lic = $('d-lic').value, list = $('d-hosts').value.split('\\n').map(function (s) { return s.trim(); }).filter(Boolean);
    var st = [], stop = false;
    function add(ok, t) { st.push('<li><span class="' + (ok ? 'ok' : 'bad') + '">' + (ok ? 'passa' : 'blocca') + '</span> · ' + t + '</li>'); if (!ok) stop = true; }
    if (lic === 'ok') add(true, 'Licenza: firma e scadenza verificate');
    else add(false, 'Licenza: <code>HTTP 402 License required: ' + (lic === 'none' ? 'no license installed' : 'license expired') + '</code>');
    if (!stop) { if ($('d-cfg').checked) add(true, 'Configurazione di produzione completa'); else { st.push('<li><span class="bad">torna al modulo</span> · mancano valori di produzione (<code>needs_config</code>)</li>'); stop = true; } }
    if (!stop) {
      if (LOCAL.indexOf(hostOf(host)) >= 0) add(true, 'Host locale: il controllo sull’host viene saltato (la licenza serve comunque)');
      else if (list.indexOf(host) >= 0) add(true, 'Host coperto dalla licenza (confronto esatto su schema, host e porta)');
      else add(false, 'Host: <code>HTTP 402 Host not licensed: ' + e(host) + ' is not covered by your license</code>');
    }
    if (!stop) st.push('<li><span class="ok">parte</span> · <code>ops ide login --mode=production</code>, poi il deploy; ogni riga arriva come evento <code>output</code></li>');
    else st.push('<li>Il cluster di destinazione non è stato toccato.</li>');
    $('d-out').innerHTML = '<ol class="timeline">' + st.join('') + '</ol><p class="live">Il push su Git non passa da questi controlli.</p>';
  }
  ['d-host', 'd-lic', 'd-hosts', 'd-cfg'].forEach(function (id) { $(id).addEventListener('input', run); $(id).addEventListener('change', run); });
  run();
})();
</script>""")

# ---------------------------------------------------------------- guidaprovider
PAGES["guidaprovider"] = dict(
    TITLE="Provider AI", TAG="Flusso · scelta del provider",
    H1="Che cosa succede quando scegli il provider",
    LEAD="La home spiega le tre modalità. Questa pagina mostra dove finisce la scelta e che cosa fa Trustant subito dopo.",
    INTENTO="""<p>Il provider decide chi risponde all’assistente: un servizio cloud, un servizio a crediti o un tuo server compatibile. La scelta si fa una volta e vale per tutto il workspace;
la <a href="/">home</a> confronta Cloud AI, Sovereign AI e Private AI.</p>""",
    TU="""<ol>
<li>Al primo avvio, o da <strong>Configure</strong>, scegli il provider nella finestra <em>Provider Choice</em>.</li>
<li>Indica endpoint, modello e chiave se la modalità li richiede.</li>
<li>Leggi la finestra <strong>Configuring</strong> mentre lavora e premi <strong>Continue</strong>.</li>
</ol>""",
    SOTTO="""<p>La finestra «Configuring» mostra i passi reali, in ordine:</p>
<ol class="timeline">
  <li><strong>Identità Git.</strong> Imposta nome ed e-mail Git.<small>«OK: git user.name set …» · «OK: git user.email set …»</small></li>
  <li><strong>Server locale.</strong> Con Private AI non installa un Ollama proprio.<small>«OK: Skipping Ollama setup (Private AI)»</small></li>
  <li><strong>Assistente.</strong> Scrive la configurazione globale dell’agente Pi.<small>«OK: Pi global configuration written»</small></li>
  <li><strong>Prova del modello.</strong> Chiede al modello di rispondere «OK».<small>«Testing AI model connection…» · «OK: AI model connection successful»</small></li>
</ol>
<p>Il server registra la stessa prova e scarta i modelli non adatti all’agente (audio, voce):
<small>«Testing model … at &lt;endpoint&gt;…» · «Skipping Pi model …: not suitable for Pi agent work»</small>.</p>
<p>Dopo la scelta, ogni pagina che presuppone un provider controlla <code>GET /api/configuration</code> e, se il campo <code>provider</code> manca, rimanda alla scelta. Se la versione dei modelli cambia, la pagina chiede di riselezionare il modello.
Il riquadro dei crediti compare solo con il provider Trustant.</p>""",
    DEMO="",
    RISULTATO="""<p>Un workspace con provider, modello e identità Git configurati e provati. Le app nuove ereditano queste impostazioni: il modello si legge, per esempio, nel benvenuto di TruChat.</p>""",
    VERIFICA="",
    FONTI="spec/1-applist.md, spec/1-index.md, spec/2a-config.md, log del server",
    SCRIPT="")

# ---------------------------------------------------------------- write
def render(app, d):
    html = PAGE
    for k in ("TITLE", "TAG", "H1", "LEAD", "INTENTO", "TU", "SOTTO", "DEMO", "RISULTATO", "VERIFICA", "FONTI", "SCRIPT"):
        html = html.replace("@@%s@@" % k, d[k])
    return html

for app, d in PAGES.items():
    folder = os.path.join(OUT, app)
    os.makedirs(folder, exist_ok=True)
    with open(os.path.join(folder, "guida.html"), "w", encoding="utf-8") as f:
        f.write(render(app, d))
print("generated", ", ".join(sorted(PAGES)))
