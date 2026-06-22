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
    if (p.applet !== undefined) apps.push('▶' + (p.applet || '?'));
    const kind = (p.applet !== undefined) ? 'applet' : ((p.entries && p.entries.length) ? 'menu' : 'page');
    const extra = [...apps, ...specs].join('  ');
    const sub = kind + (extra ? '   ' + extra : '');
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

function addPage() {
  let i = 1, id = 'page' + i;
  while (site.pages[id]) id = 'page' + (++i);
  // Une page peut avoir du texte (lines) et/ou des choix (entries).
  site.pages[id] = { title: id.toUpperCase(), lines: [], entries: [] };
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

  // Page applet « auto-lancée » (compat JSON manuel) : édition de l'applet + next.
  if (p.applet !== undefined) {
    host.append(el('p', { className: 'hint', textContent: 'Page applet (lancé à l\'arrivée). Préférez une entrée de menu ▶ applet.' }));
    const apIn = el('input', { type: 'text', value: p.applet || '' });
    apIn.oninput = () => { p.applet = apIn.value; refreshPreview(); };
    host.append(field('Applet', apIn));
    host.append(field('Après succès (next)', pageSelect(p.next, v => { p.next = v; }, true)));
    host.append(linesEditor(p));
    return;
  }

  // Page normale : texte (lines) ET/OU choix (entries).
  host.append(linesEditor(p));
  host.append(entriesEditor(p));
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

const STYLE_KEYS = ['ink', 'paper', 'blink', 'doubleHeight', 'altCharset', 'inverse'];
function assignStyle(dst, src) { for (const k of STYLE_KEYS) if (src[k] !== undefined) dst[k] = src[k]; }
function clearLineStyle(ln) { delete ln.text; for (const k of STYLE_KEYS) delete ln[k]; }

// styleControls : encre, fond, et bascules C(lignotement)/H(auteur)/A(lt)/I(nverse),
// liées à l'objet o (ligne simple ou segment).
function styleControls(o) {
  const ink = el('select');
  for (const c of INKS) ink.append(el('option', { value: c, textContent: c, selected: (o.ink || 'white') === c }));
  ink.onchange = () => { o.ink = ink.value; refreshPreview(); };

  const paper = el('select');
  paper.append(el('option', { value: '', textContent: 'fond —', selected: !o.paper }));
  for (const c of INKS) paper.append(el('option', { value: c, textContent: 'fond ' + c, selected: o.paper === c }));
  paper.onchange = () => { if (paper.value) o.paper = paper.value; else delete o.paper; refreshPreview(); };

  const tog = (key, label, title) => {
    const cb = el('input', { type: 'checkbox', checked: !!o[key] });
    cb.onchange = () => { if (cb.checked) o[key] = true; else delete o[key]; refreshPreview(); };
    return el('label', { className: 'tog', title }, [cb, document.createTextNode(label)]);
  };
  return el('span', { className: 'style-ctl' }, [
    ink, paper,
    tog('blink', 'C', 'Clignotement'),
    tog('doubleHeight', 'H', 'Double hauteur'),
    tog('altCharset', 'A', 'Semi-graphiques'),
    tog('inverse', 'I', 'Inverse'),
  ]);
}

function linesEditor(p) {
  const wrap = el('div', { className: 'lines' });
  (p.lines || []).forEach((ln, i) => {
    const card = el('div', { className: 'line-card' });
    const head = el('div', { className: 'line-head' });
    head.append(el('span', { className: 'lbl', textContent: 'Ligne ' + (i + 1) + (ln.segments ? ' (segments)' : '') }));

    if (ln.segments) {
      const merge = el('button', { textContent: 'fusionner' });
      merge.onclick = () => { const s0 = ln.segments[0] || {}; const np = { text: s0.text || '' }; assignStyle(np, s0); delete ln.segments; Object.assign(ln, np); renderForm(); refreshPreview(); };
      head.append(merge);
    } else {
      const split = el('button', { textContent: 'segments' });
      split.onclick = () => { const seg = { text: ln.text || '' }; assignStyle(seg, ln); clearLineStyle(ln); ln.segments = [seg]; renderForm(); refreshPreview(); };
      head.append(split);
    }
    const delL = el('button', { className: 'del', textContent: '✕' });
    delL.onclick = () => { p.lines.splice(i, 1); renderForm(); refreshPreview(); };
    head.append(delL);
    card.append(head);

    if (ln.segments) {
      ln.segments.forEach((seg, j) => {
        const t = el('input', { type: 'text', value: seg.text || '', placeholder: 'fragment' });
        t.oninput = () => { seg.text = t.value; refreshPreview(); };
        const delS = el('button', { className: 'del', textContent: '✕' });
        delS.onclick = () => { ln.segments.splice(j, 1); if (!ln.segments.length) delete ln.segments; renderForm(); refreshPreview(); };
        card.append(el('div', { className: 'seg-row' }, [t, styleControls(seg), delS]));
      });
      const addS = el('button', { textContent: '+ segment' });
      addS.onclick = () => { ln.segments.push({ text: '' }); renderForm(); };
      card.append(addS);
    } else {
      const t = el('input', { type: 'text', value: ln.text || '' });
      t.oninput = () => { ln.text = t.value; refreshPreview(); };
      card.append(el('div', { className: 'seg-row' }, [t, styleControls(ln)]));
    }
    wrap.append(card);
  });
  const add = el('button', { textContent: '+ ligne' });
  add.onclick = () => { p.lines.push({ text: '' }); renderForm(); };
  return el('div', {}, [el('span', { className: 'lbl', textContent: 'Lignes' }), wrap, add]);
}

const td = (c) => el('td', {}, c);

// --- aperçu : simulateur ULA (rend le flux OASCII sur un canvas) ---
const COLS = 40, ROWS = 28, CW = 6, CH = 8;
// palette Oric : bit0=R, bit1=G, bit2=B (couleurs pures RVB).
const PAL = Array.from({ length: 8 }, (_, c) => [(c & 1) ? 255 : 0, (c & 2) ? 255 : 0, (c & 4) ? 255 : 0]);
let lastScreen = null;   // dernier buffer 40×28 rendu
let blinkOn = false;

// layoutScreen : reproduit putbyte du terminal (CR/LF/scroll/clamp 40) pour
// poser le flux OASCII dans un buffer 40×28.
function layoutScreen(bytes) {
  const buf = new Uint8Array(COLS * ROWS).fill(0x20);
  let col = 0, row = 0;
  for (const b of bytes) {
    if (b === 0x0D) { col = 0; }
    else if (b === 0x0A) {
      row++;
      if (row >= ROWS) { buf.copyWithin(0, COLS); buf.fill(0x20, COLS * (ROWS - 1)); row = ROWS - 1; }
    } else if (b === 0) { /* NUL ignoré */ }
    else if (col < COLS) { buf[row * COLS + col] = b; col++; }
  }
  return buf;
}

// drawScreen : rend le buffer 40×28 sur le canvas selon l'ULA (cf. video.c).
function drawScreen(buf) {
  const cv = $('oric-screen'); if (!cv || !window.ORIC_CHARSET) return;
  const ctx = cv.getContext('2d');
  const img = ctx.createImageData(COLS * CW, ROWS * CH);
  const put = (x, y, rgb) => { const o = (y * COLS * CW + x) * 4; img.data[o] = rgb[0]; img.data[o + 1] = rgb[1]; img.data[o + 2] = rgb[2]; img.data[o + 3] = 255; };

  for (let row = 0; row < ROWS; row++) {
    let ink = 7, paper = 0, attr = 0; // reset début de ligne (ULA)
    for (let col = 0; col < COLS; col++) {
      const b = buf[row * COLS + col];
      if ((b & 0x60) === 0) {                 // attribut
        const v = b & 0x1F;
        if ((v & 0x18) === 0x00) ink = v & 7;
        else if ((v & 0x18) === 0x08) attr = v & 7;
        else if ((v & 0x18) === 0x10) paper = v & 7;
        for (let cy = 0; cy < CH; cy++) for (let bx = 0; bx < CW; bx++) put(col * CW + bx, row * CH + cy, PAL[paper]);
      } else {                                 // caractère
        const idx = b & 0x7F;
        let inv = (b & 0x80) !== 0;
        if ((attr & 4) && blinkOn) inv = !inv;
        const fg = PAL[ink], bg = PAL[paper];
        for (let cy = 0; cy < CH; cy++) {
          const erow = (attr & 2) ? ((cy >> 1) + (row & 1 ? 4 : 0)) : cy;
          const glyph = (idx >= 0x20 && idx <= 0x7F) ? window.ORIC_CHARSET[(idx - 0x20) * 8 + erow] : 0;
          for (let bx = 0; bx < CW; bx++) {
            let on = (glyph >> (5 - bx)) & 1;
            if (inv) on = on ? 0 : 1;
            put(col * CW + bx, row * CH + cy, on ? fg : bg);
          }
        }
      }
    }
  }
  ctx.putImageData(img, 0, 0);
}

let previewTimer = null;
function refreshPreview() {
  clearTimeout(previewTimer);
  previewTimer = setTimeout(doPreview, 120);
}
async function doPreview() {
  if (!current) return;
  const r = await fetch('/api/screen?page=' + encodeURIComponent(current), { method: 'POST', body: JSON.stringify(site) });
  if (!r.ok) return;
  lastScreen = layoutScreen(new Uint8Array(await r.arrayBuffer()));
  drawScreen(lastScreen);
}
// clignotement : ré-affiche périodiquement le dernier écran.
setInterval(() => { blinkOn = !blinkOn; if (lastScreen) drawScreen(lastScreen); }, 320);

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

// --- profils & déploiement (propres au site courant) ---
let currentProfile = null;

async function loadProfiles() {
  const sel = $('profile-select'); sel.innerHTML = '';
  $('profile-form').innerHTML = '';
  if (!siteName) return;
  const names = await fetch('/api/profiles?site=' + encodeURIComponent(siteName)).then(r => r.json()).catch(() => []);
  for (const n of names) sel.append(el('option', { value: n, textContent: n }));
  if (sel.value) loadProfile(sel.value);
}

async function loadProfile(env) {
  if (!siteName || !env) { $('profile-form').innerHTML = ''; return; }
  currentProfile = await fetch('/api/profile?site=' + encodeURIComponent(siteName) + '&env=' + encodeURIComponent(env)).then(r => r.json()).catch(() => null);
  renderProfileForm();
}

function renderProfileForm() {
  const host = $('profile-form'); host.innerHTML = '';
  const p = currentProfile; if (!p) return;

  const localCb = el('input', { type: 'checkbox', checked: !!p.local });
  localCb.onchange = () => { p.local = localCb.checked; renderProfileForm(); };
  host.append(field('Local (copie)', localCb));

  if (p.local) {
    host.append(field('Fichier cible', textField(p, 'contentPath', 'ex. content/site.json')));
    host.append(el('p', { className: 'hint', textContent: 'Profil local : copie de fichier, le bbsd recharge à chaud.' }));
  } else {
    host.append(field('Hôte (SSH)', textField(p, 'host', 'ex. 10.0.0.1')));
    host.append(field('Utilisateur', textField(p, 'user', 'ex. root')));
    host.append(field('Port', textField(p, 'port', '22')));
    host.append(field('Fichier cible', textField(p, 'contentPath', 'ex. /etc/bbsoric/site.json')));
    host.append(field('Service systemd', textField(p, 'service', 'ex. bbsoric')));
  }

  const reload = el('select');
  for (const r of ['none', 'reload', 'restart']) reload.append(el('option', { value: r, textContent: r, selected: (p.reload || 'none') === r }));
  reload.onchange = () => { p.reload = reload.value; };
  host.append(field('Reload', reload));
}

// textField : champ texte lié à une clé d'un objet.
function textField(obj, key, placeholder) {
  const i = el('input', { type: 'text', value: obj[key] || '', placeholder: placeholder || '' });
  i.oninput = () => { obj[key] = i.value; };
  return i;
}

async function saveProfile() {
  const env = $('profile-select').value;
  if (!siteName || !env || !currentProfile) { setStatus('aucun profil sélectionné', 'err'); return; }
  const r = await fetch('/api/profile?site=' + encodeURIComponent(siteName) + '&env=' + encodeURIComponent(env), { method: 'POST', body: JSON.stringify(currentProfile) }).then(r => r.json());
  setStatus(r.ok ? 'profil enregistré ✓ ' + env : 'échec : ' + (r.error || ''), r.ok ? 'ok' : 'err');
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
$('btn-save-profile').onclick = saveProfile;
$('profile-select').onchange = () => loadProfile($('profile-select').value);
$('btn-add-page').onclick = () => addPage();
for (const t of document.querySelectorAll('.tab')) t.onclick = () => showTab(t.dataset.tab);
showTab('nav');
loadSites(); // charge le 1er site, qui charge ses propres profils
