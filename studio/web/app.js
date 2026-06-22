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
  setStatus('chargé : ' + name, 'ok');
}

// --- liste des pages ---
function renderPageList() {
  const ul = $('page-list'); ul.innerHTML = '';
  for (const id of Object.keys(site.pages)) {
    const star = (id === site.start) ? '★ ' : '';
    const b = el('button', { textContent: star + id + '  (' + (site.pages[id].type || '?') + ')' });
    if (id === current) b.classList.add('sel');
    b.onclick = () => { current = id; renderPageList(); renderForm(); refreshPreview(); };
    ul.append(el('li', {}, b));
  }
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
  renderPageList(); renderForm(); refreshPreview();
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
  if (!current || !site.pages[current]) { host.append(el('p', { className: 'hint', textContent: 'Sélectionne une page.' })); return; }
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

  // type
  const typeSel = el('select');
  for (const t of ['menu', 'page', 'applet']) typeSel.append(el('option', { value: t, textContent: t, selected: p.type === t }));
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
  tbl.append(el('tr', {}, [el('th', { textContent: 'Touche' }), el('th', { textContent: 'Libellé' }), el('th', { textContent: 'Cible' }), el('th')]));
  (p.entries || []).forEach((e, i) => {
    const k = el('input', { type: 'text', value: e.key || '' }); k.oninput = () => { e.key = k.value; refreshPreview(); };
    const l = el('input', { type: 'text', value: e.label || '' }); l.oninput = () => { e.label = l.value; refreshPreview(); };
    const t = targetSelect(e.target, v => { e.target = v; });
    const del = el('button', { className: 'del', textContent: '✕' }); del.onclick = () => { p.entries.splice(i, 1); renderForm(); refreshPreview(); };
    tbl.append(el('tr', {}, [td(k), td(l), td(t), td(del)]));
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

// --- init ---
$('btn-load').onclick = () => loadSite($('site-select').value);
$('btn-validate').onclick = validate;
$('btn-save').onclick = save;
for (const b of document.querySelectorAll('.add-row button')) b.onclick = () => addPage(b.dataset.type);
loadSites();
