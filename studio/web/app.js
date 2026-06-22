/* Studio Forge — éditeur du site.json du BBS Oric.
 * Modèle = format serveur : { start, pages: { id: {title,type,entries,lines,applet,next} } }
 */
'use strict';

const SPECIALS = ['__quit__', '__back__', '__home__'];
const INKS = ['white', 'red', 'green', 'yellow', 'blue', 'magenta', 'cyan', 'black'];

let siteName = null;
let site = { start: '', pages: {} };
let current = null;

const $ = (id) => document.getElementById(id);
const el = (tag, props = {}, children = []) => {
  const e = document.createElement(tag);
  Object.assign(e, props);
  for (const c of [].concat(children)) e.append(c);
  return e;
};

function setStatus(msg, kind) {
  const s = $('status'); s.textContent = msg || ''; s.className = 'status ' + (kind || '');
}
const esc = (s) => String(s == null ? '' : s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

// --- chargement ---
async function loadSites() {
  const names = await fetch('/api/sites').then(r => r.json()).catch(() => []);
  const sel = $('site-select'); sel.innerHTML = '';
  for (const n of names) sel.append(el('option', { value: n, textContent: n }));
  if (names.length) { sel.value = names[0]; await loadSite(names[0]); }
}

async function loadSite(name) {
  const r = await fetch('/api/site?name=' + encodeURIComponent(name));
  if (!r.ok) { setStatus('chargement impossible', 'err'); return; }
  site = await r.json();
  if (!site.pages) site.pages = {};
  siteName = name;
  current = site.start && site.pages[site.start] ? site.start : Object.keys(site.pages)[0] || null;
  renderPageList(); renderForm(); refreshPreview();
  loadProfiles();                 // profils propres à CE site
  setStatus('chargé : ' + name, 'ok');
}

// --- onglets ---
function showTab(name) {
  for (const p of document.querySelectorAll('.tabpane')) p.classList.toggle('active', p.id === 'tab-' + name);
  for (const t of document.querySelectorAll('.tab')) t.classList.toggle('active', t.dataset.tab === name);
}

// --- graphe de navigation (onglet Navigation) ---
const SPEC_LABEL = { '__quit__': '⏏ quitter', '__back__': '↩ retour', '__home__': '⌂ accueil' };

// entryIsApplet : une entrée lance-t-elle un applet (vs naviguer) ?
const entryIsApplet = (e) => Object.prototype.hasOwnProperty.call(e, 'applet');

// targetsOf renvoie les pages réelles vers lesquelles pointe une page : cible
// d'une entrée de navigation, page `next` d'une entrée-applet, ou next de page applet.
function targetsOf(id) {
  const p = site.pages[id] || {}; const t = [];
  for (const e of p.entries || []) {
    if (entryIsApplet(e)) { if (e.next && site.pages[e.next]) t.push(e.next); }
    else if (site.pages[e.target]) t.push(e.target);
  }
  if (p.next && site.pages[p.next]) t.push(p.next);
  return t;
}

function renderPageList() {
  const svg = $('graph'); if (!svg) return;
  const ids = Object.keys(site.pages);
  if (!ids.length) { svg.innerHTML = ''; return; }
  const start = (site.start && site.pages[site.start]) ? site.start : ids[0];

  // niveaux par parcours en largeur depuis la page de départ
  const level = { [start]: 0 }; const q = [start];
  while (q.length) { const id = q.shift(); for (const t of targetsOf(id)) if (level[t] == null) { level[t] = level[id] + 1; q.push(t); } }
  let maxLv = 0; for (const id of ids) { if (level[id] == null) level[id] = 0; maxLv = Math.max(maxLv, level[id]); }
  const byLv = {}; for (const id of ids) (byLv[level[id]] ||= []).push(id);

  const NW = 168, NH = 48, GX = 72, GY = 28; const pos = {}; let rows = 0;
  for (let lv = 0; lv <= maxLv; lv++) {
    const arr = byLv[lv] || [];
    arr.forEach((id, i) => { pos[id] = { x: lv * (NW + GX) + 16, y: i * (NH + GY) + 16 }; });
    rows = Math.max(rows, arr.length);
  }
  const W = (maxLv + 1) * (NW + GX) + 16, H = Math.max(1, rows) * (NH + GY) + 16;

  let edges = '';
  for (const id of ids) {
    const a = pos[id];
    for (const t of targetsOf(id)) {
      const b = pos[t];
      const x1 = a.x + NW, y1 = a.y + NH / 2, x2 = b.x, y2 = b.y + NH / 2, mx = (x1 + x2) / 2;
      edges += `<path d="M${x1} ${y1} C ${mx} ${y1} ${mx} ${y2} ${x2} ${y2}" class="edge" marker-end="url(#arrow)"/>`;
    }
  }
  let nodes = '';
  for (const id of ids) {
    const p = site.pages[id], a = pos[id];
    const specs = (p.entries || []).filter(e => SPEC_LABEL[e.target]).map(e => SPEC_LABEL[e.target]);
    const apps = (p.entries || []).filter(entryIsApplet).map(e => '▶' + (e.applet || '?'));
    if (p.type === 'applet') apps.push('▶' + (p.applet || '?'));
    const extra = [...apps, ...specs].join('  ');
    const sub = p.type + (extra ? '   ' + extra : '');
    const cls = 'node' + (id === current ? ' sel' : '') + (id === start ? ' start' : '');
    nodes += `<g class="${cls}" data-id="${esc(id)}" transform="translate(${a.x},${a.y})">`
      + `<rect width="${NW}" height="${NH}" rx="6"/>`
      + `<text x="10" y="20" class="nid">${id === start ? '★ ' : ''}${esc(id)}</text>`
      + `<text x="10" y="37" class="nsub">${esc(sub)}</text>`
      + `<text x="${NW - 13}" y="18" class="ndel" data-del="${esc(id)}">✕</text></g>`;
  }
  svg.setAttribute('viewBox', `0 0 ${W} ${H}`);
  svg.setAttribute('width', W); svg.setAttribute('height', H);
  svg.innerHTML = `<defs><marker id="arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto"><path d="M0 0 L10 5 L0 10 z" class="arrowhd"/></marker></defs>${edges}${nodes}`;

  svg.querySelectorAll('g.node').forEach(g => g.addEventListener('click', ev => {
    const id = g.getAttribute('data-id');
    if (ev.target.hasAttribute('data-del')) { if (confirm('Supprimer « ' + id + ' » ?')) deletePage(id); return; }
    current = id; renderPageList(); renderForm(); refreshPreview(); showTab('edit');
  }));
}

function addPage(type) {
  let i = 1, id = type + i;
  while (site.pages[id]) id = type + (++i);
  const p = { title: id.toUpperCase(), type };
  if (type === 'menu') p.entries = [];
  else if (type === 'applet') { p.applet = ''; p.next = ''; p.lines = []; }
  else p.lines = [];
  site.pages[id] = p;
  if (!site.start) site.start = id;
  current = id;
  renderPageList(); renderForm(); refreshPreview(); showTab('edit');
}

function deletePage(id) {
  delete site.pages[id];
  if (site.start === id) site.start = Object.keys(site.pages)[0] || '';
  if (current === id) current = site.start || Object.keys(site.pages)[0] || null;
  renderPageList(); renderForm(); refreshPreview();
}

// renomme une page et met à jour les références (start, targets, next).
function renamePage(oldId, newId) {
  newId = newId.trim();
  if (!newId || newId === oldId || site.pages[newId]) return;
  site.pages[newId] = site.pages[oldId];
  delete site.pages[oldId];
  if (site.start === oldId) site.start = newId;
  for (const p of Object.values(site.pages)) {
    for (const e of p.entries || []) if (e.target === oldId) e.target = newId;
    if (p.next === oldId) p.next = newId;
  }
  current = newId;
  renderPageList(); renderForm(); refreshPreview();
}

// --- formulaire d'édition de la page courante ---
function renderForm() {
  const host = $('page-form'); host.innerHTML = '';
  $('edit-page-name').textContent = current || '(aucune)';
  if (!current || !site.pages[current]) { host.append(el('p', { className: 'hint', textContent: 'Sélectionne une page dans Navigation.' })); return; }
  const p = site.pages[current];

  // id
  const idIn = el('input', { type: 'text', value: current });
  idIn.onchange = () => renamePage(current, idIn.value);
  host.append(field('Identifiant', idIn));

  // start
  const startCb = el('input', { type: 'checkbox', checked: site.start === current });
  startCb.onchange = () => { if (startCb.checked) site.start = current; renderPageList(); };
  host.append(field('Page de départ', startCb));

  // titre
  const titleIn = el('input', { type: 'text', value: p.title || '' });
  titleIn.oninput = () => { p.title = titleIn.value; refreshPreview(); };
  host.append(field('Titre', titleIn));

  // type (les applets se lancent via une entrée de menu, pas une page dédiée)
  const typeSel = el('select');
  const types = (p.type === 'applet') ? ['menu', 'page', 'applet'] : ['menu', 'page'];
  for (const t of types) typeSel.append(el('option', { value: t, textContent: t, selected: p.type === t }));
  typeSel.onchange = () => { p.type = typeSel.value; normalizePage(p); renderForm(); renderPageList(); refreshPreview(); };
  host.append(field('Type', typeSel));

  if (p.type === 'menu') host.append(entriesEditor(p));
  else {
    if (p.type === 'applet') {
      const apIn = el('input', { type: 'text', value: p.applet || '' });
      apIn.oninput = () => { p.applet = apIn.value; refreshPreview(); };
      host.append(field('Applet', apIn));
      host.append(field('Après succès (next)', pageSelect(p.next, v => { p.next = v; }, true)));
    }
    host.append(linesEditor(p));
  }
}

function normalizePage(p) {
  if (p.type === 'menu') { p.entries = p.entries || []; delete p.lines; delete p.applet; delete p.next; }
  else if (p.type === 'applet') { p.lines = p.lines || []; p.applet = p.applet || ''; p.next = p.next || ''; delete p.entries; }
  else { p.lines = p.lines || []; delete p.entries; delete p.applet; delete p.next; }
}

function field(label, control) {
  return el('label', {}, [el('span', { className: 'lbl', textContent: label }), control]);
}

function pageSelect(value, onChange, allowEmpty) {
  const sel = el('select');
  if (allowEmpty) sel.append(el('option', { value: '', textContent: '(aucune)', selected: !value }));
  for (const id of Object.keys(site.pages)) sel.append(el('option', { value: id, textContent: id, selected: value === id }));
  sel.onchange = () => { onChange(sel.value); refreshPreview(); };
  return sel;
}

function targetSelect(value, onChange) {
  const sel = el('select');
  for (const id of Object.keys(site.pages)) sel.append(el('option', { value: id, textContent: id, selected: value === id }));
  for (const s of SPECIALS) sel.append(el('option', { value: s, textContent: s, selected: value === s }));
  sel.onchange = () => { onChange(sel.value); refreshPreview(); };
  return sel;
}

function entriesEditor(p) {
  const tbl = el('table', { className: 'rows' });
  tbl.append(el('tr', {}, ['Touche', 'Libellé', 'Type', 'Destination', ''].map(t => el('th', { textContent: t }))));
  (p.entries || []).forEach((e, i) => {
    const k = el('input', { type: 'text', value: e.key || '' }); k.oninput = () => { e.key = k.value; refreshPreview(); };
    const l = el('input', { type: 'text', value: e.label || '' }); l.oninput = () => { e.label = l.value; refreshPreview(); };

    // Type d'entrée : navigation (→ page) ou applet (▶ applet)
    const kind = el('select');
    kind.append(el('option', { value: 'page', textContent: '→ page', selected: !entryIsApplet(e) }));
    kind.append(el('option', { value: 'applet', textContent: '▶ applet', selected: entryIsApplet(e) }));
    kind.onchange = () => {
      if (kind.value === 'applet') { delete e.target; e.applet = e.applet || ''; e.next = e.next || ''; }
      else { delete e.applet; delete e.next; e.target = e.target || Object.keys(site.pages)[0] || '__quit__'; }
      renderForm(); refreshPreview();
    };

    let dest;
    if (entryIsApplet(e)) {
      const ap = el('input', { type: 'text', value: e.applet || '', placeholder: 'nom applet' });
      ap.oninput = () => { e.applet = ap.value; refreshPreview(); };
      const nx = pageSelect(e.next, v => { e.next = v; }, true); // page après succès
      dest = el('div', { className: 'dest-applet' }, [ap, nx]);
    } else {
      dest = targetSelect(e.target, v => { e.target = v; });
    }

    const del = el('button', { className: 'del', textContent: '✕' });
    del.onclick = () => { p.entries.splice(i, 1); renderForm(); refreshPreview(); };
    tbl.append(el('tr', {}, [td(k), td(l), td(kind), td(dest), td(del)]));
  });
  const add = el('button', { textContent: '+ entrée' });
  add.onclick = () => { p.entries.push({ key: '', label: '', target: Object.keys(site.pages)[0] || '__quit__' }); renderForm(); };
  return el('div', {}, [el('span', { className: 'lbl', textContent: 'Entrées' }), tbl, add]);
}

function linesEditor(p) {
  const tbl = el('table', { className: 'rows' });
  tbl.append(el('tr', {}, [el('th', { textContent: 'Texte' }), el('th', { textContent: 'Encre' }), el('th')]));
  (p.lines || []).forEach((ln, i) => {
    const t = el('input', { type: 'text', value: ln.text || '' }); t.oninput = () => { ln.text = t.value; refreshPreview(); };
    const ink = el('select');
    for (const c of INKS) ink.append(el('option', { value: c, textContent: c, selected: (ln.ink || 'white') === c }));
    ink.onchange = () => { ln.ink = ink.value; refreshPreview(); };
    const del = el('button', { className: 'del', textContent: '✕' }); del.onclick = () => { p.lines.splice(i, 1); renderForm(); refreshPreview(); };
    tbl.append(el('tr', {}, [td(t), td(ink), td(del)]));
  });
  const add = el('button', { textContent: '+ ligne' });
  add.onclick = () => { p.lines.push({ text: '', ink: 'white' }); renderForm(); };
  return el('div', {}, [el('span', { className: 'lbl', textContent: 'Lignes' }), tbl, add]);
}

const td = (c) => el('td', {}, c);

// --- aperçu / validation / sauvegarde ---
let previewTimer = null;
function refreshPreview() {
  clearTimeout(previewTimer);
  previewTimer = setTimeout(doPreview, 150);
}
async function doPreview() {
  if (!current) { $('preview').innerHTML = ''; return; }
  const r = await fetch('/api/preview?page=' + encodeURIComponent(current), { method: 'POST', body: JSON.stringify(site) });
  $('preview').innerHTML = r.ok ? await r.text() : '<span style="color:#f55">' + (await r.text()) + '</span>';
}

async function validate() {
  const r = await fetch('/api/validate', { method: 'POST', body: JSON.stringify(site) }).then(r => r.json());
  if (r.ok) setStatus('valide ✓', 'ok'); else setStatus('invalide : ' + r.error, 'err');
  return r.ok;
}

async function save() {
  if (!siteName) { setStatus('aucun site chargé', 'err'); return; }
  const r = await fetch('/api/save?name=' + encodeURIComponent(siteName), { method: 'POST', body: JSON.stringify(site) }).then(r => r.json());
  if (r.ok) setStatus('enregistré ✓ ' + siteName, 'ok'); else setStatus('échec : ' + r.error, 'err');
}

// --- déploiement par profils (propres au site courant) ---
async function loadProfiles() {
  const sel = $('profile-select'); sel.innerHTML = '';
  if (!siteName) return;
  const names = await fetch('/api/profiles?site=' + encodeURIComponent(siteName)).then(r => r.json()).catch(() => []);
  for (const n of names) sel.append(el('option', { value: n, textContent: n }));
}

async function deploy(dryRun) {
  const profile = $('profile-select').value;
  if (!siteName) { $('deploy-log').textContent = 'aucun site chargé'; return; }
  if (!profile) { $('deploy-log').textContent = 'aucun profil pour ce site (voir deploy/profiles/' + siteName.replace(/\.json$/, '') + '/*.conf.example)'; return; }
  if (!dryRun && !confirm('Déployer (écraser) « ' + siteName + ' » sur le profil « ' + profile + ' » ?')) return;
  const url = '/api/deploy?site=' + encodeURIComponent(siteName) + '&profile=' + encodeURIComponent(profile) + '&dryRun=' + (dryRun ? 'true' : 'false');
  const r = await fetch(url, { method: 'POST', body: JSON.stringify(site) }).then(r => r.json());
  $('deploy-log').textContent = (r.log || []).join('\n');
  setStatus(dryRun ? 'simulation effectuée' : (r.ok ? 'déployé ✓ ' + profile : 'échec déploiement'), r.ok ? 'ok' : 'err');
}

// --- init ---
$('btn-load').onclick = () => loadSite($('site-select').value);
$('btn-validate').onclick = validate;
$('btn-save').onclick = save;
$('btn-dryrun').onclick = () => deploy(true);
$('btn-deploy').onclick = () => deploy(false);
for (const b of document.querySelectorAll('.add-row button')) b.onclick = () => addPage(b.dataset.type);
for (const t of document.querySelectorAll('.tab')) t.onclick = () => showTab(t.dataset.tab);
showTab('nav');
loadSites(); // charge le 1er site, qui charge ses propres profils
