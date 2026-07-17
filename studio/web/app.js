/* Studio Forge — éditeur du site.json du BBS Oric.
 * Modèle = format serveur : { start, pages: { id: {title,type,entries,lines,applet,next} } }
 */
'use strict';

const SPECIALS = ['__quit__', '__back__', '__home__'];
const INKS = ['white', 'red', 'green', 'yellow', 'blue', 'magenta', 'cyan', 'black'];

let siteName = null;
let site = { start: '', pages: {} };
let current = null;
// État de navigation de l'aperçu de grille DataWindow (par page).
let gridNav = { page: null, n: 1, sel: 0, filtre: '', scroll: 0 };

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
  srcName = Object.keys(sources())[0] || null;
  renderPageList(); renderForm(); refreshPreview();
  loadProfiles();                 // profils propres à CE site
  refreshScreenPages();           // pages « écran brut » éditables
  refreshSources();               // sources DataWindow éditables
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

// Applets autonomes proposés pour une entrée de menu (enregistrés côté serveur).
// « form » est volontairement exclu : il se gère via le formulaire d'une page.
// Doit rester aligné sur les applets enregistrés via bbs.Register (server).
const KNOWN_APPLETS = ['login', 'register', 'guest', 'download', 'upload', 'who', 'chat', 'wall'];

// APPLET_DESC : libellé d'aide (infobulle) par applet, pour guider le câblage.
const APPLET_DESC = {
  login: 'identification (compte existant)',
  register: 'création de compte',
  guest: 'accès invité (lecture seule)',
  download: 'téléchargement de fichier (XMODEM)',
  upload: 'téléversement de fichier (XMODEM)',
  who: 'qui est en ligne (liste des connectés)',
  chat: 'salon de discussion temps réel',
  wall: 'mur de messages persisté (livre d\'or)',
};

// appletSelect : liste déroulante des applets connus (+ la valeur courante si
// personnalisée), pour câbler une entrée « ▶ applet » sans faute de frappe.
function appletSelect(value, onChange) {
  const sel = el('select');
  const opts = KNOWN_APPLETS.slice();
  if (value && !opts.includes(value)) opts.unshift(value); // préserve un nom custom
  if (!value) sel.append(el('option', { value: '', textContent: '— applet —', selected: true }));
  for (const a of opts) sel.append(el('option', { value: a, textContent: a, title: APPLET_DESC[a] || '', selected: value === a }));
  sel.onchange = () => { onChange(sel.value); refreshPreview(); };
  if (value && APPLET_DESC[value]) sel.title = APPLET_DESC[value];
  return sel;
}

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
    const dw = p.datawindow ? ['▦ ' + (p.datawindow.source || '?')] : [];
    const hi = p.hires ? ['◨ hires'] : [];
    const kind = (p.hires !== undefined) ? 'graphique'
      : (p.datawindow !== undefined) ? 'grille'
      : (p.applet !== undefined) ? 'applet'
      : (p.raw ? 'écran' : ((p.entries && p.entries.length) ? 'menu' : 'page'));
    const extra = [...apps, ...dw, ...hi, ...specs].join('  ');
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

  // Page grille (datawindow) : présente une source de données ; pas de menu/form.
  if (p.datawindow) {
    if (!Object.keys(sources()).length) host.append(el('p', { className: 'hint err', textContent: 'Aucune source définie — crée-en une dans l\'onglet Données.' }));
    host.append(dataWindowEditor(p));
    return;
  }

  // Page graphique HIRES : primitives de tracé et/ou fond bitmap (240×200).
  if (p.hires) {
    host.append(hiresEditor(p));
    return;
  }

  // Page normale : texte (lines), choix (entries) OU formulaire de saisie (form).
  host.append(linesEditor(p));
  if (!p.form) host.append(entriesEditor(p)); // un form pilote la page (pas de menu simultané)
  host.append(formEditor(p));
  if (Object.keys(sources()).length && !p.form) host.append(dataWindowAdd(p)); // convertir en grille
  if (!p.form) host.append(hiresAdd(p)); // convertir en page graphique HIRES
}

// formEditor édite une page de saisie déclarative (content.Form) : action
// (login/inscription), champs et page « next ». La logique reste serveur.
function formEditor(p, rebuild) {
  rebuild = rebuild || renderForm; // re-construit l'éditeur après changement
  const wrap = el('div', { className: 'form-editor' });
  wrap.append(el('span', { className: 'lbl', textContent: 'Formulaire de saisie' }));
  if (!p.form) {
    const add = el('button', { textContent: '+ formulaire (login / inscription)' });
    add.onclick = () => {
      p.form = { action: 'login', next: '', fields: [
        { key: 'login', label: 'Pseudo' },
        { key: 'password', label: 'Mot de passe', secret: true },
      ] };
      delete p.entries;
      rebuild(); refreshPreview();
    };
    wrap.append(add);
    return wrap;
  }
  const f = p.form;
  const act = el('select');
  ['login', 'register'].forEach(a => act.append(el('option', { value: a, textContent: a, selected: f.action === a })));
  act.onchange = () => {
    f.action = act.value;
    if (f.action === 'register' && !(f.fields || []).some(x => x.key === 'confirm')) {
      f.fields.push({ key: 'confirm', label: 'Confirmer', secret: true });
    }
    rebuild(); refreshPreview();
  };
  wrap.append(field('Action', act));
  wrap.append(field('Après succès (next)', pageSelect(f.next, v => { f.next = v; }, true)));
  wrap.append(field('En cas d\'échec', pageSelect(f.fail, v => { f.fail = v; }, true)));
  const ret = el('input', { type: 'number', value: f.retries || '' }); ret.min = 1; ret.style.width = '60px'; ret.placeholder = '3';
  ret.oninput = () => { const n = parseInt(ret.value, 10); if (n > 0) f.retries = n; else delete f.retries; };
  wrap.append(field('Tentatives', ret));

  const tbl = el('table', { className: 'rows' });
  tbl.append(el('tr', {}, ['Clé', 'Libellé', 'Secret', 'X', 'Y', ''].map(t => el('th', { textContent: t }))));
  (f.fields || []).forEach((fld, i) => {
    const k = el('input', { type: 'text', value: fld.key || '' }); k.oninput = () => { fld.key = k.value; };
    const l = el('input', { type: 'text', value: fld.label || '' }); l.oninput = () => { fld.label = l.value; };
    const sec = el('input', { type: 'checkbox', checked: !!fld.secret }); sec.onchange = () => { fld.secret = sec.checked; };
    // Position absolue optionnelle (plot X,Y) ; X et Y vides = invite séquentielle.
    const xIn = el('input', { type: 'number', value: fld.at ? fld.at[0] : '' }); xIn.min = 0; xIn.max = 39; xIn.style.width = '52px';
    const yIn = el('input', { type: 'number', value: fld.at ? fld.at[1] : '' }); yIn.min = 0; yIn.max = 27; yIn.style.width = '52px';
    const updAt = () => {
      const xv = String(xIn.value).trim(), yv = String(yIn.value).trim();
      if (xv === '' && yv === '') delete fld.at;
      else fld.at = [parseInt(xv || '0', 10) || 0, parseInt(yv || '0', 10) || 0];
      refreshPreview();
    };
    xIn.oninput = updAt; yIn.oninput = updAt;
    const del = el('button', { className: 'del', textContent: '✕' });
    del.onclick = () => { f.fields.splice(i, 1); rebuild(); };
    tbl.append(el('tr', {}, [td(k), td(l), td(el('label', { className: 'tog' }, [sec])), td(xIn), td(yIn), td(del)]));
  });
  wrap.append(tbl);
  const addF = el('button', { textContent: '+ champ' });
  addF.onclick = () => { f.fields.push({ key: '', label: '' }); rebuild(); };
  wrap.append(addF);

  const rm = el('button', { className: 'del', textContent: 'supprimer le formulaire' });
  rm.onclick = () => { delete p.form; rebuild(); refreshPreview(); };
  wrap.append(rm);
  wrap.append(el('p', { className: 'hint', textContent: 'Clés attendues : login, password' + (f.action === 'register' ? ', confirm' : '') + '. Le décor (titre ou écran raw) s\'affiche, puis les champs se saisissent.' }));
  return wrap;
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

function entriesEditor(p, rebuild, opts) {
  rebuild = rebuild || renderForm; // qui re-construit l'éditeur après +/✕/type
  // hideLabel : sur un « menu sur fond d'écran » (page raw), le libellé est dessiné
  // dans le décor — e.label est ignoré au rendu, on masque donc sa colonne.
  const hideLabel = !!(opts && opts.hideLabel);
  const tbl = el('table', { className: 'rows' });
  const headers = hideLabel ? ['Touche', 'Type', 'Destination', ''] : ['Touche', 'Libellé', 'Type', 'Destination', ''];
  tbl.append(el('tr', {}, headers.map(t => el('th', { textContent: t }))));
  (p.entries || []).forEach((e, i) => {
    const k = el('input', { type: 'text', value: e.key || '' }); k.oninput = () => { e.key = k.value; refreshPreview(); };
    let l = null;
    if (!hideLabel) { l = el('input', { type: 'text', value: e.label || '' }); l.oninput = () => { e.label = l.value; refreshPreview(); }; }

    // Type d'entrée : navigation (→ page) ou applet (▶ applet)
    const kind = el('select');
    kind.append(el('option', { value: 'page', textContent: '→ page', selected: !entryIsApplet(e) }));
    kind.append(el('option', { value: 'applet', textContent: '▶ applet', selected: entryIsApplet(e) }));
    kind.onchange = () => {
      if (kind.value === 'applet') { delete e.target; e.applet = e.applet || ''; e.next = e.next || ''; }
      else { delete e.applet; delete e.next; e.target = e.target || Object.keys(site.pages)[0] || '__quit__'; }
      rebuild(); refreshPreview();
    };

    let dest;
    if (entryIsApplet(e)) {
      const ap = appletSelect(e.applet, v => { e.applet = v; }); ap.title = 'applet';
      const nx = pageSelect(e.next, v => { e.next = v; }, true); nx.title = 'page si succès (next)';
      const fl = pageSelect(e.fail, v => { e.fail = v; }, true); fl.title = 'page si échec (fail)';
      dest = el('div', { className: 'dest-applet' }, [ap, nx, fl]);
    } else {
      dest = targetSelect(e.target, v => { e.target = v; });
    }

    const del = el('button', { className: 'del', textContent: '✕' });
    del.onclick = () => { p.entries.splice(i, 1); rebuild(); refreshPreview(); };
    const cells = hideLabel ? [td(k), td(kind), td(dest), td(del)] : [td(k), td(l), td(kind), td(dest), td(del)];
    tbl.append(el('tr', {}, cells));
  });
  const add = el('button', { textContent: '+ entrée' });
  add.onclick = () => { p.entries.push({ key: '', label: '', target: Object.keys(site.pages)[0] || '__quit__' }); rebuild(); };
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
// drawScreen : aperçu d'une page (buffer 40×28) — réutilise le rendu ULA partagé.
function drawScreen(buf) { renderScreenBuf($('oric-screen'), buf, null); }

// --- palette de glyphes BBS (alimente le compositeur) ---
function glyphCanvas(code) {
  const cv = document.createElement('canvas'); cv.width = CW; cv.height = CH; cv.className = 'gly';
  const ctx = cv.getContext('2d'); const img = ctx.createImageData(CW, CH);
  for (let r = 0; r < CH; r++) {
    const b = window.ORIC_ALTCHARSET[code * 8 + r];
    for (let x = 0; x < CW; x++) {
      const on = (b >> (5 - x)) & 1, o = (r * CW + x) * 4;
      img.data[o] = img.data[o + 1] = img.data[o + 2] = on ? 238 : 0; img.data[o + 3] = 255;
    }
  }
  ctx.putImageData(img, 0, 0); return cv;
}

function renderPaletteInto(hostId, onPick) {
  const host = $(hostId); if (!host || !window.ORIC_ALTCHARSET) return;
  host.innerHTML = '';
  for (let c = 0x20; c < 0x80; c++) {
    let blank = true;
    for (let r = 0; r < 8; r++) if (window.ORIC_ALTCHARSET[c * 8 + r]) { blank = false; break; }
    if (blank) continue;
    const btn = el('button', { className: 'gly-btn', title: 'glyphe « ' + String.fromCharCode(c) + ' » (0x' + c.toString(16) + ')' });
    btn.append(glyphCanvas(c));
    btn.onclick = () => onPick(c);
    host.append(btn);
  }
}


// --- éditeur d'écran plein (40×28, page « écran brut ») ---
// Modèle OCTETS, fidèle à l'ULA : la grille est le buffer écran 40×28 ; les
// attributs (encre/fond/texte) sont des CASES qu'on pose explicitement et qui
// s'appliquent jusqu'au prochain attribut (comme sur Oric). Aucune coloration
// « par cellule » incohérente avec la sérialisation.
const COLOR_NAMES = ['black', 'red', 'green', 'yellow', 'blue', 'magenta', 'cyan', 'white'];
let gridBuf = null;        // Uint8Array(40*28) : octets écran (chars + attributs)
let brushByte = 0x20;      // octet posé par le pinceau
let cur = { col: 0, row: 0 };
let screenName = null;

function initGrid() { gridBuf = new Uint8Array(COLS * ROWS).fill(0x20); cur = { col: 0, row: 0 }; }

// renderScreenBuf : rend un buffer 40×28 d'octets selon l'ULA (attributs = cases,
// état encre/fond/texte réinitialisé à chaque début de ligne).
function renderScreenBuf(cvEl, buf, cursor) {
  if (!cvEl || !window.ORIC_CHARSET) return;
  const ctx = cvEl.getContext('2d');
  const img = ctx.createImageData(COLS * CW, ROWS * CH);
  const setpx = (col, row, cy, x, c) => { const o = ((row * CH + cy) * COLS * CW + col * CW + x) * 4; img.data[o] = c[0]; img.data[o + 1] = c[1]; img.data[o + 2] = c[2]; img.data[o + 3] = 255; };
  for (let row = 0; row < ROWS; row++) {
    let ink = 7, paper = 0, attr = 0;
    for (let col = 0; col < COLS; col++) {
      const b = buf[row * COLS + col] ?? 0x20;
      if ((b & 0x60) === 0) {                 // attribut -> change l'état ; case = bloc fond
        const v = b & 0x1F;
        if ((v & 0x18) === 0x00) ink = v & 7;
        else if ((v & 0x18) === 0x08) attr = v & 7;
        else if ((v & 0x18) === 0x10) paper = v & 7;
        for (let cy = 0; cy < CH; cy++) for (let x = 0; x < CW; x++) setpx(col, row, cy, x, PAL[paper]);
      } else {                                 // caractère
        const idx = b & 0x7F; let inv = (b & 0x80) !== 0; if ((attr & 4) && blinkOn) inv = !inv;
        const altFont = (attr & 1) && window.ORIC_ALTCHARSET;
        for (let cy = 0; cy < CH; cy++) {
          const erow = (attr & 2) ? ((cy >> 1) + (row & 1 ? 4 : 0)) : cy;
          const g = altFont ? window.ORIC_ALTCHARSET[idx * 8 + erow] : ((idx >= 0x20 && idx <= 0x7F) ? window.ORIC_CHARSET[(idx - 0x20) * 8 + erow] : 0);
          for (let x = 0; x < CW; x++) { let on = (g >> (5 - x)) & 1; if (inv) on = on ? 0 : 1; setpx(col, row, cy, x, on ? PAL[ink] : PAL[paper]); }
        }
      }
    }
  }
  ctx.putImageData(img, 0, 0);
  if (cursor) { ctx.strokeStyle = '#22dddd'; ctx.lineWidth = 1; ctx.strokeRect(cursor.col * CW + 0.5, cursor.row * CH + 0.5, CW - 1, CH - 1); }
}

function drawGrid() { renderScreenBuf($('screen-canvas'), gridBuf, cur); }

function brushDesc(b) {
  if ((b & 0x60) === 0) {
    const v = b & 0x1F;
    if ((v & 0x18) === 0x00) return 'encre ' + COLOR_NAMES[v & 7];
    if ((v & 0x18) === 0x08) { const a = []; if (v & 1) a.push('alt'); if (v & 2) a.push('2×h'); if (v & 4) a.push('cli'); return 'texte ' + (a.join('+') || 'normal'); }
    if ((v & 0x18) === 0x10) return 'fond ' + COLOR_NAMES[v & 7];
    return 'vidéo';
  }
  return "car '" + String.fromCharCode(b & 0x7F) + "'" + ((b & 0x80) ? ' inv' : '');
}
function setBrush(b) { brushByte = b & 0xFF; const d = $('brush-desc'); if (d) d.textContent = brushDesc(brushByte); }

function paintAt(col, row) { if (col >= 0 && col < COLS && row >= 0 && row < ROWS) gridBuf[row * COLS + col] = brushByte; }
function typeAt(ch) {
  const inv = $('brush-inv').checked ? 0x80 : 0;
  gridBuf[cur.row * COLS + cur.col] = (ch.charCodeAt(0) & 0x7F) | inv;
  cur.col++; if (cur.col >= COLS) { cur.col = 0; if (cur.row < ROWS - 1) cur.row++; }
}
// putByteAdvance pose un octet brut (attribut ou caractère) à la position du
// curseur puis avance — comme typeAt mais sans le masque/bit inverse réservés
// aux caractères imprimables.
function putByteAdvance(b) {
  gridBuf[cur.row * COLS + cur.col] = b & 0xFF;
  cur.col++; if (cur.col >= COLS) { cur.col = 0; if (cur.row < ROWS - 1) cur.row++; }
}
// pickAttr est appelé par les pastilles couleur / boutons d'attribut : sur Oric un
// attribut OCCUPE une case (l'« espace » coloré), on le POSE donc au curseur et on
// avance. On garde aussi le pinceau (peinture au clic) et on redonne le focus au
// canvas pour enchaîner la frappe sans cliquer dans la grille.
function pickAttr(b) {
  setBrush(b);
  putByteAdvance(b);
  drawGrid();
  const c = $('screen-canvas'); if (c) c.focus();
}
// altActiveAt indique si le charset alternatif (police BBS) est actif à la cellule
// (col,row), d'après la sérialisation depuis le début de ligne (l'ULA réinitialise
// les attributs à chaque ligne ; seul le groupe 0x08 porte le bit alt).
function altActiveAt(col, row) {
  let attr = 0;
  for (let x = 0; x < col; x++) {
    const b = gridBuf[row * COLS + x];
    if ((b & 0x60) === 0 && ((b & 0x1F) & 0x18) === 0x08) attr = b & 0x07;
  }
  return (attr & 1) !== 0;
}
// dropGlyph dépose un glyphe BBS au curseur. Un glyphe n'est rendu en police BBS
// que si l'attribut « charset alternatif » est actif : on POSE donc d'abord la
// case d'attribut alt (0x09) si elle ne l'est pas déjà, puis le glyphe.
function dropGlyph(c) {
  c &= 0x7F;
  setBrush(c);
  if (!altActiveAt(cur.col, cur.row)) putByteAdvance(0x09);
  putByteAdvance(c);
  drawGrid();
  const cv = $('screen-canvas'); if (cv) cv.focus();
}

// renderScreenNav affiche, sous la grille, l'éditeur de navigation de la page
// d'écran courante : on compose le décor au-dessus (fond raw) et on câble ici les
// touches (→ page ou ▶ applet). Présentation et logique au même endroit.
function renderScreenNav() {
  const host = $('screen-nav'); if (!host) return;
  host.innerHTML = '';
  if (!screenName || !site.pages[screenName]) {
    host.append(el('p', { className: 'hint', textContent: 'Navigation du menu : charge ou crée une page écran ci-dessus pour câbler ses touches (→ page ou ▶ applet).' }));
    return;
  }
  const p = site.pages[screenName];
  if (p.form) {
    // Page de saisie : le décor (grille) sert de fond, le formulaire pose ses
    // champs (positionnables par X/Y). C'est l'applet « form ».
    host.append(el('p', { className: 'hint', textContent: 'Formulaire de saisie (applet) sur cet écran : compose le décor ci-dessus, place les champs avec X/Y.' }));
    host.append(formEditor(p, renderScreenNav));
  } else {
    host.append(el('p', { className: 'hint', textContent: 'Navigation (menu sur fond d\'écran) : ces touches routent par-dessus le décor (→ page ou ▶ applet). Les libellés sont dessinés dans l\'écran ci-dessus.' }));
    host.append(entriesEditor(p, renderScreenNav, { hideLabel: true }));
    host.append(formEditor(p, renderScreenNav)); // bouton « + formulaire » (login/inscription)
  }
}

function refreshScreenPages() {
  const sel = $('screen-page'); sel.innerHTML = '';
  // Toutes les pages sont chargeables dans l'éditeur d'écran : une page « écran
  // brut » (raw) reprend son buffer, une page normale est rendue par le serveur
  // puis convertie en buffer éditable (l'enregistrement la passe en raw).
  for (const id of Object.keys(site.pages || {})) {
    sel.append(el('option', { value: id, textContent: id + (site.pages[id].raw ? ' (écran)' : '') }));
  }
}

const bufToB64 = (buf) => { let s = ''; for (const b of buf) s += String.fromCharCode(b); return btoa(s); };
function b64ToBuf(b64) { const s = atob(b64), u = new Uint8Array(COLS * ROWS).fill(0x20); for (let i = 0; i < s.length && i < u.length; i++) u[i] = s.charCodeAt(i); return u; }

async function screenLoad(id) {
  if (!id || !site.pages[id]) return;
  screenName = id; const p = site.pages[id];
  if (p.screen) gridBuf = b64ToBuf(p.screen);
  else { // page raw décrite par des lignes : on récupère le rendu serveur
    const r = await fetch('/api/screen?page=' + encodeURIComponent(id), { method: 'POST', body: JSON.stringify(site) });
    gridBuf = r.ok ? layoutScreen(new Uint8Array(await r.arrayBuffer())) : new Uint8Array(COLS * ROWS).fill(0x20);
  }
  cur = { col: 0, row: 0 }; drawGrid(); renderScreenNav();
  setStatus('écran chargé : ' + id, 'ok');
}
function screenNew() {
  const id = ($('screen-newid').value || '').trim();
  if (!id) { setStatus('donne un identifiant', 'err'); return; }
  if (site.pages[id]) { setStatus('cet identifiant existe déjà', 'err'); return; }
  initGrid();
  site.pages[id] = { title: id.toUpperCase(), raw: true, screen: bufToB64(gridBuf) };
  if (!site.start) site.start = id;
  screenName = id; $('screen-newid').value = '';
  refreshScreenPages(); $('screen-page').value = id; drawGrid(); renderScreenNav(); renderPageList();
  setStatus('page écran créée : ' + id, 'ok');
}
function screenSave() {
  if (!screenName || !site.pages[screenName]) { setStatus('crée/charge une page écran', 'err'); return; }
  const p = site.pages[screenName];
  p.raw = true; delete p.lines; p.screen = bufToB64(gridBuf);
  refreshScreenPages(); $('screen-page').value = screenName; renderPageList();
  setStatus('écran enregistré dans « ' + screenName + ' » (raw' + (p.entries && p.entries.length ? ' + menu' : '') + ')', 'ok');
}

let previewTimer = null;
function refreshPreview() {
  clearTimeout(previewTimer);
  previewTimer = setTimeout(doPreview, 120);
}
async function doPreview() {
  if (!current) return;
  const p = site.pages[current];
  if (p && p.hires) { lastScreen = null; renderHiresPreview(p); return; } // aperçu graphique local
  if (p && p.datawindow) { await doGridPreview(); return; }               // aperçu de grille interactif
  const r = await fetch('/api/screen?page=' + encodeURIComponent(current), { method: 'POST', body: JSON.stringify(site) });
  if (!r.ok) return;
  lastScreen = layoutScreen(new Uint8Array(await r.arrayBuffer()));
  drawScreen(lastScreen);
}

// doGridPreview : rend la grille de la page datawindow courante via /api/grid
// (MÊME rendu que le serveur, depuis les données seed de la source) et l'affiche.
// L'état de navigation (page, sélection, filtre) est repris à chaque changement de page.
async function doGridPreview() {
  if (gridNav.page !== current) gridNav = { page: current, n: 1, sel: 0, filtre: '', scroll: 0 };
  const q = '?page=' + encodeURIComponent(current) + '&n=' + gridNav.n +
    '&sel=' + gridNav.sel + '&scroll=' + gridNav.scroll + '&filtre=' + encodeURIComponent(gridNav.filtre);
  let buf;
  try {
    const r = await fetch('/api/grid' + q, { method: 'POST', body: JSON.stringify(site) });
    buf = r.ok ? new Uint8Array(await r.arrayBuffer()) : new Uint8Array(COLS * ROWS).fill(0x20);
  } catch { buf = new Uint8Array(COLS * ROWS).fill(0x20); }
  lastScreen = buf;
  drawScreen(buf);
}
// clignotement : ré-affiche périodiquement le dernier écran.
setInterval(() => {
  blinkOn = !blinkOn;
  if (lastScreen) drawScreen(lastScreen);
  if (gridBuf && document.getElementById('tab-screen').classList.contains('active')) drawGrid();
}, 320);

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

// --- sources de données (onglet Données) : édition de site.sources_donnees ---
// Le modèle JSON utilise les clés serveur (snake_case) : type_source, tri_defaut,
// lignes_par_page, cle_primaire, auto_increment, longueur_max, valeur_defaut,
// auto_date, ttl_sec — on les manipule telles quelles.
const SQL_TYPES = ['TEXT', 'INTEGER', 'REAL', 'NUMERIC', 'BOOLEAN', 'DATE', 'DATETIME', 'BLOB'];
let srcName = null;

// sources renvoie (en le créant au besoin) le dictionnaire des sources du site.
function sources() { return site.sources_donnees || (site.sources_donnees = {}); }

// numberField : champ numérique entier lié à une clé d'un objet (vide = supprime).
function numberField(obj, key, placeholder, min, max) {
  const i = el('input', { type: 'number', value: obj[key] != null ? obj[key] : '', placeholder: placeholder || '' });
  if (min != null) i.min = min; if (max != null) i.max = max; i.style.width = '90px';
  i.oninput = () => { const v = i.value.trim(); if (v === '') delete obj[key]; else obj[key] = parseInt(v, 10) || 0; };
  return i;
}

// coerceCell convertit la saisie d'une cellule seed selon le type de colonne :
// numérique pour INTEGER/REAL/NUMERIC (sinon texte) ; vide = clé supprimée.
function coerceCell(col, val) {
  const v = String(val).trim();
  if (v === '') return undefined;
  const t = (col.type || 'TEXT').toUpperCase();
  if ((t === 'INTEGER' || t === 'REAL' || t === 'NUMERIC') && v !== '' && Number.isFinite(Number(v))) return Number(v);
  return val;
}

function refreshSources() {
  const sel = $('src-select'); if (!sel) return;
  sel.innerHTML = '';
  const ids = Object.keys(sources());
  for (const id of ids) sel.append(el('option', { value: id, textContent: id }));
  if (!srcName || !sources()[srcName]) srcName = ids[0] || null;
  if (srcName) sel.value = srcName;
  renderSourceForm();
}

// renomme une source et reporte la référence dans les pages grille (datawindow.source).
function renameSource(oldId, newId) {
  newId = newId.trim();
  if (!newId || newId === oldId || sources()[newId]) return;
  const next = {};
  for (const [k, v] of Object.entries(sources())) next[k === oldId ? newId : k] = v;
  site.sources_donnees = next;
  for (const p of Object.values(site.pages)) if (p.datawindow && p.datawindow.source === oldId) p.datawindow.source = newId;
  srcName = newId; refreshSources(); renderPageList();
}

function srcCreate() {
  const id = ($('src-newid').value || '').trim();
  if (!id) { setStatus('donne un identifiant de source', 'err'); return; }
  if (sources()[id]) { setStatus('cette source existe déjà', 'err'); return; }
  sources()[id] = {
    table: id, tri_defaut: '', lignes_par_page: 15,
    colonnes: { id: { type: 'INTEGER', libelle: 'ID', cle_primaire: true, auto_increment: true } },
    donnees: [],
  };
  srcName = id; $('src-newid').value = '';
  refreshSources(); setStatus('source créée : ' + id, 'ok');
}

function srcDelete() {
  if (!srcName || !sources()[srcName]) { setStatus('aucune source sélectionnée', 'err'); return; }
  const refs = Object.entries(site.pages).filter(([, p]) => p.datawindow && p.datawindow.source === srcName).map(([id]) => id);
  const warn = refs.length ? '\nPages grille qui la référencent : ' + refs.join(', ') : '';
  if (!confirm('Supprimer la source « ' + srcName + ' » ?' + warn)) return;
  delete sources()[srcName];
  srcName = Object.keys(sources())[0] || null;
  refreshSources(); renderPageList();
}

// renomme une colonne en préservant l'ordre d'insertion (la map est reconstruite).
function renameCol(src, oldK, newK) {
  newK = newK.trim();
  if (!newK || newK === oldK || src.colonnes[newK]) return;
  const next = {};
  for (const [k, v] of Object.entries(src.colonnes)) next[k === oldK ? newK : k] = v;
  src.colonnes = next;
  renderSourceForm();
}

// colonnesEditor : table des colonnes typées (tous les champs de ColonneDef).
function colonnesEditor(src) {
  const wrap = el('div', {});
  wrap.append(el('span', { className: 'lbl', textContent: 'Colonnes' }));
  const tbl = el('table', { className: 'rows' });
  tbl.append(el('tr', {}, ['Clé', 'Type', 'Libellé', 'PK', 'Auto+', 'Requis', 'LongMax', 'Pattern', 'Défaut', 'Date', ''].map(t => el('th', { textContent: t }))));
  for (const [name, col] of Object.entries(src.colonnes || {})) {
    const k = el('input', { type: 'text', value: name }); k.style.width = '90px'; k.onchange = () => renameCol(src, name, k.value);
    const ty = el('select');
    for (const t of SQL_TYPES) ty.append(el('option', { value: t, textContent: t, selected: (col.type || 'TEXT').toUpperCase() === t }));
    ty.onchange = () => { col.type = ty.value; };
    const lb = el('input', { type: 'text', value: col.libelle || '' }); lb.style.width = '90px'; lb.oninput = () => { col.libelle = lb.value; };
    const tog = (key) => { const cb = el('input', { type: 'checkbox', checked: !!col[key] }); cb.onchange = () => { if (cb.checked) col[key] = true; else delete col[key]; }; return el('label', { className: 'tog' }, [cb]); };
    const lm = el('input', { type: 'number', value: col.longueur_max || '' }); lm.min = 0; lm.style.width = '64px';
    lm.oninput = () => { const v = parseInt(lm.value, 10); if (v > 0) col.longueur_max = v; else delete col.longueur_max; };
    const pat = el('input', { type: 'text', value: col.pattern || '' }); pat.style.width = '90px'; pat.oninput = () => { if (pat.value) col.pattern = pat.value; else delete col.pattern; };
    const def = el('input', { type: 'text', value: col.valeur_defaut != null ? col.valeur_defaut : '' }); def.style.width = '80px';
    def.oninput = () => { const v = coerceCell(col, def.value); if (v === undefined) delete col.valeur_defaut; else col.valeur_defaut = v; };
    const del = el('button', { className: 'del', textContent: '✕' });
    del.onclick = () => { delete src.colonnes[name]; renderSourceForm(); };
    tbl.append(el('tr', {}, [td(k), td(ty), td(lb), td(tog('cle_primaire')), td(tog('auto_increment')), td(tog('requis')), td(lm), td(pat), td(def), td(tog('auto_date')), td(del)]));
  }
  wrap.append(tbl);
  const add = el('button', { textContent: '+ colonne' });
  add.onclick = () => {
    let i = 1, key = 'col' + i; while (src.colonnes[key]) key = 'col' + (++i);
    src.colonnes[key] = { type: 'TEXT', libelle: key }; renderSourceForm();
  };
  wrap.append(add);
  return wrap;
}

// seedEditor : table des données initiales (seed) d'une source SQLite. Une ligne
// par enregistrement ; les colonnes auto-incrémentées sont omises (générées).
function seedEditor(src) {
  const wrap = el('div', {});
  wrap.append(el('span', { className: 'lbl', textContent: 'Données initiales (seed, importées table vide)' }));
  const cols = Object.entries(src.colonnes || {}).filter(([, c]) => !c.auto_increment).map(([k]) => k);
  if (!cols.length) { wrap.append(el('p', { className: 'hint', textContent: 'Ajoute des colonnes pour saisir des données.' })); return wrap; }
  src.donnees = src.donnees || [];
  const tbl = el('table', { className: 'rows' });
  tbl.append(el('tr', {}, [...cols, ''].map(t => el('th', { textContent: t }))));
  src.donnees.forEach((row, i) => {
    const cells = cols.map(c => {
      const inp = el('input', { type: 'text', value: row[c] != null ? row[c] : '' }); inp.style.width = '110px';
      inp.oninput = () => { const v = coerceCell(src.colonnes[c], inp.value); if (v === undefined) delete row[c]; else row[c] = v; };
      return td(inp);
    });
    const del = el('button', { className: 'del', textContent: '✕' });
    del.onclick = () => { src.donnees.splice(i, 1); renderSourceForm(); };
    tbl.append(el('tr', {}, [...cells, td(del)]));
  });
  wrap.append(tbl);
  const add = el('button', { textContent: '+ ligne' });
  add.onclick = () => { src.donnees.push({}); renderSourceForm(); };
  wrap.append(add);
  return wrap;
}

function renderSourceForm() {
  const host = $('src-form'); if (!host) return;
  host.innerHTML = '';
  if (!srcName || !sources()[srcName]) { host.append(el('p', { className: 'hint', textContent: 'Crée ou charge une source. Elle alimente une page « grille » (onglet Édition).' })); return; }
  const src = sources()[srcName];

  const idIn = el('input', { type: 'text', value: srcName }); idIn.onchange = () => renameSource(srcName, idIn.value);
  host.append(field('Identifiant', idIn));

  const typeSel = el('select');
  typeSel.append(el('option', { value: 'sqlite', textContent: 'SQLite (CRUD)', selected: src.type_source !== 'api' }));
  typeSel.append(el('option', { value: 'api', textContent: 'API REST (lecture seule)', selected: src.type_source === 'api' }));
  typeSel.onchange = () => {
    if (typeSel.value === 'api') { src.type_source = 'api'; src.api = src.api || { url: '', ttl_sec: 60 }; delete src.donnees; }
    else { delete src.type_source; delete src.api; src.donnees = src.donnees || []; }
    renderSourceForm();
  };
  host.append(field('Type', typeSel));

  if (src.type_source === 'api') {
    src.api = src.api || { url: '', ttl_sec: 60 };
    host.append(field('URL', textField(src.api, 'url', 'https://exemple/data.json')));
    host.append(field('Clé racine', textField(src.api, 'racine', 'ex. results (vide = tableau)')));
    host.append(field('Cache (s)', numberField(src.api, 'ttl_sec', '60', 0)));
  } else {
    host.append(field('Table SQL', textField(src, 'table', 'ex. annuaire')));
  }
  host.append(field('Tri par défaut', textField(src, 'tri_defaut', 'ex. nom ASC')));
  host.append(field('Lignes par page', numberField(src, 'lignes_par_page', '15', 1)));

  host.append(colonnesEditor(src));
  if (src.type_source !== 'api') host.append(seedEditor(src));
}

// --- éditeur du descripteur grille (page datawindow), onglet Édition ---

// dataWindowAdd : convertit la page courante en page « grille » (datawindow).
function dataWindowAdd(p) {
  const wrap = el('div', { className: 'form-editor' });
  wrap.append(el('span', { className: 'lbl', textContent: 'Grille de données' }));
  const sids = Object.keys(sources());
  const add = el('button', { textContent: '+ grille de données (DataWindow)' });
  add.onclick = () => {
    const sid = sids[0];
    const cols = Object.entries(sources()[sid].colonnes || {}).filter(([, c]) => !c.auto_increment).map(([k]) => k).slice(0, 1);
    p.datawindow = { source: sid, colonnes_affichees: cols, largeurs: cols.map(() => 8), editable: false };
    delete p.lines; delete p.entries; delete p.form;
    renderForm(); refreshPreview(); renderPageList();
  };
  wrap.append(add);
  wrap.append(el('p', { className: 'hint', textContent: 'Présente une source de données (onglet Données) sous forme de grille paginée navigable au clavier.' }));
  return wrap;
}

// inkSelect : choix d'encre (couleur) optionnel lié à une clé d'un objet.
function inkSelect(obj, key) {
  const sel = el('select');
  sel.append(el('option', { value: '', textContent: '(défaut)', selected: !obj[key] }));
  for (const c of INKS) sel.append(el('option', { value: c, textContent: c, selected: obj[key] === c }));
  sel.onchange = () => { if (sel.value) obj[key] = sel.value; else delete obj[key]; refreshPreview(); };
  return sel;
}

// dataWindowEditor édite le descripteur grille d'une page : source, colonnes
// affichées (ordre + largeurs, budget 40 cases), couleurs et droits d'édition.
function dataWindowEditor(p) {
  const dw = p.datawindow;
  const wrap = el('div', { className: 'form-editor' });
  wrap.append(el('span', { className: 'lbl', textContent: 'Grille de données (DataWindow)' }));

  const sids = Object.keys(sources());
  const srcSel = el('select');
  for (const id of sids) srcSel.append(el('option', { value: id, textContent: id, selected: dw.source === id }));
  srcSel.onchange = () => { dw.source = srcSel.value; dw.colonnes_affichees = []; dw.largeurs = []; renderForm(); refreshPreview(); renderPageList(); };
  wrap.append(field('Source', srcSel));

  const src = sources()[dw.source];
  if (!src) { wrap.append(el('p', { className: 'hint', textContent: 'Source « ' + (dw.source || '?') + ' » introuvable (onglet Données).' })); return wrap; }

  // Colonnes affichées : ordre + largeur. On garde largeurs aligné sur colonnes_affichees.
  dw.colonnes_affichees = dw.colonnes_affichees || [];
  if (!dw.largeurs || dw.largeurs.length !== dw.colonnes_affichees.length) {
    dw.largeurs = dw.colonnes_affichees.map((_, i) => (dw.largeurs && dw.largeurs[i]) || 8);
  }
  const tbl = el('table', { className: 'rows' });
  tbl.append(el('tr', {}, ['Colonne', 'Largeur', 'Ordre', ''].map(t => el('th', { textContent: t }))));
  dw.colonnes_affichees.forEach((c, i) => {
    const name = el('span', { textContent: c + (src.colonnes[c] ? '' : ' (absente)') });
    const wIn = el('input', { type: 'number', value: dw.largeurs[i] }); wIn.min = 1; wIn.max = 40; wIn.style.width = '64px';
    wIn.oninput = () => { dw.largeurs[i] = parseInt(wIn.value, 10) || 1; refreshPreview(); };
    const up = el('button', { textContent: '↑' }); up.disabled = i === 0;
    up.onclick = () => { [dw.colonnes_affichees[i - 1], dw.colonnes_affichees[i]] = [dw.colonnes_affichees[i], dw.colonnes_affichees[i - 1]]; [dw.largeurs[i - 1], dw.largeurs[i]] = [dw.largeurs[i], dw.largeurs[i - 1]]; renderForm(); refreshPreview(); };
    const down = el('button', { textContent: '↓' }); down.disabled = i === dw.colonnes_affichees.length - 1;
    down.onclick = () => { [dw.colonnes_affichees[i + 1], dw.colonnes_affichees[i]] = [dw.colonnes_affichees[i], dw.colonnes_affichees[i + 1]]; [dw.largeurs[i + 1], dw.largeurs[i]] = [dw.largeurs[i], dw.largeurs[i + 1]]; renderForm(); refreshPreview(); };
    const del = el('button', { className: 'del', textContent: '✕' });
    del.onclick = () => { dw.colonnes_affichees.splice(i, 1); dw.largeurs.splice(i, 1); renderForm(); refreshPreview(); };
    tbl.append(el('tr', {}, [td(name), td(wIn), td(el('span', {}, [up, down])), td(del)]));
  });
  wrap.append(el('span', { className: 'lbl', textContent: 'Colonnes affichées' }));
  wrap.append(tbl);

  const remaining = Object.keys(src.colonnes || {}).filter(c => !dw.colonnes_affichees.includes(c));
  if (remaining.length) {
    const addSel = el('select');
    addSel.append(el('option', { value: '', textContent: '+ colonne…', selected: true }));
    for (const c of remaining) addSel.append(el('option', { value: c, textContent: c }));
    addSel.onchange = () => { if (addSel.value) { dw.colonnes_affichees.push(addSel.value); dw.largeurs.push(8); renderForm(); refreshPreview(); } };
    wrap.append(addSel);
  }
  // Budget de largeur : col attribut + index + Σ(largeur+1) ≤ 40 (cf. content.validate).
  const total = 1 + 3 + dw.largeurs.reduce((s, w) => s + (w || 0) + 1, 0);
  wrap.append(el('p', { className: 'hint' + (total > 40 ? ' err' : ''), textContent: 'Largeur grille : ' + total + '/40 cases' + (total > 40 ? ' — trop large !' : '') }));
  wrap.append(el('p', { className: 'hint', textContent: 'Aperçu interactif (à droite) : clique l’aperçu puis ↑/↓ = sélection, →/← = scroll de la ligne, S/R = pages, F = filtre, C = effacer le filtre. Données de l’onglet Données.' }));

  wrap.append(field('Couleur entête', inkSelect(dw, 'couleur_entete')));
  wrap.append(field('Couleur lignes', inkSelect(dw, 'couleur_lignes')));
  wrap.append(field('Couleur sélection', inkSelect(dw, 'couleur_selection')));
  wrap.append(field('Lignes par écran', numberField(dw, 'lignes_max', 'défaut', 1)));

  const ed = el('input', { type: 'checkbox', checked: !!dw.editable });
  ed.onchange = () => { if (ed.checked) dw.editable = true; else delete dw.editable; };
  wrap.append(field('Éditable (N/E/D)', ed));
  wrap.append(el('p', { className: 'hint', textContent: 'Éditable = créer/modifier/supprimer si connecté (sources SQLite). Désactivé sur serveur public.' }));

  const rm = el('button', { className: 'del', textContent: 'supprimer la grille' });
  rm.onclick = () => { delete p.datawindow; p.lines = p.lines || []; p.entries = p.entries || []; renderForm(); refreshPreview(); renderPageList(); };
  wrap.append(rm);
  return wrap;
}

// --- éditeur de page graphique HIRES (240×200) ---
// Le JSON porte `hires` = { background?: base64(8000 octets VRAM), draw?: [HiresOp] }.
// Primitives : curset/point/line/box/fillbox (X,Y), circle (R), char (X,Y,ch),
// ink/paper (couleur Oric 0-7). L'aperçu est rastérisé EN JS (miroir du firmware) :
// l'op « ink » colore les primitives suivantes (8 encres) ; sur Oric la couleur est
// un attribut par ligne (cf. hint), l'aperçu l'applique pixel par pixel. paper non rendu.
const HIRES_OPS = ['curset', 'point', 'line', 'box', 'fillbox', 'circle', 'char', 'ink', 'paper'];
const ORIC_COLORS = ['noir', 'rouge', 'vert', 'jaune', 'bleu', 'magenta', 'cyan', 'blanc']; // index = n° Oric
const HIW = 240, HIH = 200;

const b64ToBytes = (b64) => { const s = atob(b64), u = new Uint8Array(s.length); for (let i = 0; i < s.length; i++) u[i] = s.charCodeAt(i); return u; };
const bytesToB64 = (u) => { let s = ''; for (const b of u) s += String.fromCharCode(b); return btoa(s); };

// hiresOpNeeds : quels champs une primitive utilise (pilote les colonnes affichées).
function hiresOpNeeds(op) {
  return {
    coord: ['curset', 'point', 'line', 'box', 'fillbox', 'char'].includes(op),
    r: op === 'circle',
    c: op === 'ink' || op === 'paper',
    ch: op === 'char',
  };
}

// hiresAdd : convertit la page courante en page graphique HIRES.
function hiresAdd(p) {
  const wrap = el('div', { className: 'form-editor' });
  wrap.append(el('span', { className: 'lbl', textContent: 'Page graphique HIRES' }));
  const add = el('button', { textContent: '+ page graphique (HIRES 240×200)' });
  add.onclick = () => {
    p.hires = { draw: [{ op: 'curset', x: 8, y: 8 }, { op: 'box', x: 231, y: 191 }] };
    delete p.lines; delete p.entries; delete p.form; delete p.datawindow;
    renderForm(); refreshPreview(); renderPageList();
  };
  wrap.append(add);
  wrap.append(el('p', { className: 'hint', textContent: 'Dessin vectoriel (primitives) et/ou fond bitmap, rendu en haute résolution sur le terminal Oric.' }));
  return wrap;
}

// hiresEditor : édite la liste de primitives + le fond bitmap d'une page HIRES.
function hiresEditor(p) {
  const h = p.hires; h.draw = h.draw || [];
  const wrap = el('div', { className: 'form-editor' });
  wrap.append(el('span', { className: 'lbl', textContent: 'Primitives de tracé (crayon : curset → line/box/circle…)' }));

  const tbl = el('table', { className: 'rows' });
  tbl.append(el('tr', {}, ['Op', 'X', 'Y', 'R', 'Couleur', 'Car.', ''].map(t => el('th', { textContent: t }))));
  h.draw.forEach((op, i) => {
    const need = hiresOpNeeds(op.op);
    const sel = el('select');
    for (const o of HIRES_OPS) sel.append(el('option', { value: o, textContent: o, selected: op.op === o }));
    sel.onchange = () => { op.op = sel.value; renderForm(); refreshPreview(); };

    const num = (key, max) => {
      if (!need.coord && (key === 'x' || key === 'y')) return el('span');
      if (!need.r && key === 'r') return el('span');
      const inp = el('input', { type: 'number', value: op[key] != null ? op[key] : '' });
      inp.min = 0; inp.max = max; inp.style.width = '54px';
      inp.oninput = () => { const v = inp.value.trim(); if (v === '') delete op[key]; else op[key] = parseInt(v, 10) || 0; refreshPreview(); };
      return inp;
    };
    let coul = el('span');
    if (need.c) {
      coul = el('select');
      ORIC_COLORS.forEach((nm, n) => coul.append(el('option', { value: n, textContent: n + ' ' + nm, selected: (op.c || 0) === n })));
      coul.onchange = () => { op.c = parseInt(coul.value, 10); refreshPreview(); };
    }
    let car = el('span');
    if (need.ch) {
      car = el('input', { type: 'text', value: op.ch || '', maxLength: 1 }); car.style.width = '36px';
      car.oninput = () => { op.ch = car.value; refreshPreview(); };
    }
    const ctrl = el('span', {}, [
      mkBtn('↑', i === 0, () => { [h.draw[i - 1], h.draw[i]] = [h.draw[i], h.draw[i - 1]]; renderForm(); refreshPreview(); }),
      mkBtn('↓', i === h.draw.length - 1, () => { [h.draw[i + 1], h.draw[i]] = [h.draw[i], h.draw[i + 1]]; renderForm(); refreshPreview(); }),
      mkBtn('✕', false, () => { h.draw.splice(i, 1); renderForm(); refreshPreview(); }, 'del'),
    ]);
    tbl.append(el('tr', {}, [td(sel), td(num('x', 239)), td(num('y', 199)), td(num('r', 239)), td(coul), td(car), td(ctrl)]));
  });
  wrap.append(tbl);
  const add = el('button', { textContent: '+ primitive' });
  add.onclick = () => { h.draw.push({ op: 'line', x: 120, y: 100 }); renderForm(); refreshPreview(); };
  wrap.append(add);

  // Fond bitmap (modèle « bitmap ») : import d'image → 240×200 monochrome.
  wrap.append(el('span', { className: 'lbl', textContent: 'Fond bitmap (optionnel)' }));
  const imp = el('input', { type: 'file', accept: 'image/*' });
  imp.onchange = () => importHiresImage(imp.files[0], h);
  const bgRow = el('div', { className: 'seg-row' }, [imp]);
  if (h.background) {
    bgRow.append(el('span', { className: 'hint', textContent: 'fond présent (' + b64ToBytes(h.background).length + ' o)' }));
    const rmbg = el('button', { className: 'del', textContent: 'supprimer le fond' });
    rmbg.onclick = () => { delete h.background; renderForm(); refreshPreview(); };
    bgRow.append(rmbg);
  }
  wrap.append(bgRow);
  wrap.append(el('p', { className: 'hint', textContent: 'L\'op « ink » colore les primitives suivantes (sur Oric : un attribut par ligne, donc la couleur déborde sur toute la ligne et sacrifie la 1re cellule). L\'aperçu a un fond noir (paper non rendu) : un tracé en encre 0 (noir) y est invisible.' }));

  const rm = el('button', { className: 'del', textContent: 'supprimer la page graphique' });
  rm.onclick = () => { delete p.hires; p.lines = p.lines || []; p.entries = p.entries || []; renderForm(); refreshPreview(); renderPageList(); };
  wrap.append(rm);
  return wrap;
}

// mkBtn : petit bouton optionnellement désactivé (utilitaire pour les contrôles).
function mkBtn(label, disabled, onclick, cls) {
  const b = el('button', { textContent: label, disabled: !!disabled });
  if (cls) b.className = cls;
  b.onclick = onclick;
  return b;
}

// importHiresImage : charge une image, la réduit en 240×200 et la seuille en 1 bit
// (luminance) pour produire le buffer VRAM HIRES (octets bit6 + 6 pixels).
function importHiresImage(file, h) {
  if (!file) return;
  const img = new Image();
  img.onload = () => {
    const c = document.createElement('canvas'); c.width = HIW; c.height = HIH;
    const x = c.getContext('2d'); x.fillStyle = '#000'; x.fillRect(0, 0, HIW, HIH);
    x.drawImage(img, 0, 0, HIW, HIH);
    const d = x.getImageData(0, 0, HIW, HIH).data;
    const bytes = new Uint8Array(40 * HIH);
    for (let y = 0; y < HIH; y++) for (let bx = 0; bx < 40; bx++) {
      let b = 0x40; // bit6 = octet pixel
      for (let bit = 0; bit < 6; bit++) {
        const px = bx * 6 + bit, o = (y * HIW + px) * 4;
        if ((d[o] + d[o + 1] + d[o + 2]) / 3 >= 128) b |= (1 << (5 - bit));
      }
      bytes[y * 40 + bx] = b;
    }
    h.background = bytesToB64(bytes);
    URL.revokeObjectURL(img.src);
    renderForm(); refreshPreview();
    setStatus('image importée en fond HIRES', 'ok');
  };
  img.src = URL.createObjectURL(file);
}

// --- aperçu HIRES : rastériseur JS (miroir du firmware client/hires.s) ---
// hiInk : encre courante (0-7) ; un pixel allumé stocke (encre+1), 0 = éteint.
let hiInk = 7;
function hiSet(px, x, y) { x |= 0; y |= 0; if (x >= 0 && x < HIW && y >= 0 && y < HIH) px[y * HIW + x] = hiInk + 1; }
function hiLine(px, x0, y0, x1, y1) {
  x0 |= 0; y0 |= 0; x1 |= 0; y1 |= 0;
  const dx = Math.abs(x1 - x0), dy = Math.abs(y1 - y0), sx = x0 < x1 ? 1 : -1, sy = y0 < y1 ? 1 : -1;
  let err = dx - dy;
  for (;;) { hiSet(px, x0, y0); if (x0 === x1 && y0 === y1) break; const e2 = 2 * err; if (e2 > -dy) { err -= dy; x0 += sx; } if (e2 < dx) { err += dx; y0 += sy; } }
}
function hiBox(px, x0, y0, x1, y1) { hiLine(px, x0, y0, x1, y0); hiLine(px, x1, y0, x1, y1); hiLine(px, x1, y1, x0, y1); hiLine(px, x0, y1, x0, y0); }
function hiFillBox(px, x0, y0, x1, y1) { const a = Math.min(y0, y1), b = Math.max(y0, y1); for (let y = a; y <= b; y++) hiLine(px, x0, y, x1, y); }
function hiCircle(px, cx, cy, r) {
  if (r <= 0) return;
  let x = r, y = 0, err = 1 - r;
  while (x >= y) {
    for (const o of [[x, y], [-x, y], [x, -y], [-x, -y], [y, x], [-y, x], [y, -x], [-y, -x]]) hiSet(px, cx + o[0], cy + o[1]);
    y++; if (err <= 0) err += 2 * y + 1; else { x--; err += 2 * (y - x) + 1; }
  }
}
function hiChar(px, x, y, code) {
  const f = window.ORIC_CHARSET; if (!f) return;
  const idx = code & 0x7F; if (idx < 0x20 || idx > 0x7F) return;
  const base = (idx - 0x20) * 8;
  for (let row = 0; row < 8; row++) { const g = f[base + row] || 0; for (let bit = 0; bit < 6; bit++) if (g & (1 << (5 - bit))) hiSet(px, x + bit, y + row); }
}

// renderHiresPreview : exécute fond bitmap + primitives dans un buffer 240×200 puis
// le peint sur le canvas d'aperçu (240×224, les 24 px du bas restent noirs).
function renderHiresPreview(p) {
  const cv = $('oric-screen'); if (!cv || !cv.getContext) return;
  const px = new Uint8Array(HIW * HIH);
  const h = p.hires || {};
  hiInk = 7; // encre par défaut = blanc
  if (h.background) {
    const by = b64ToBytes(h.background);
    for (let y = 0; y < HIH; y++) for (let bx = 0; bx < 40; bx++) {
      const b = by[y * 40 + bx] || 0; if ((b & 0x60) === 0) continue; // attribut → ignoré
      for (let bit = 0; bit < 6; bit++) if (b & (1 << (5 - bit))) px[y * HIW + bx * 6 + bit] = 8; // blanc
    }
  }
  let penx = 0, peny = 0;
  for (const op of h.draw || []) {
    const x = op.x | 0, y = op.y | 0;
    switch (op.op) {
      case 'ink': if (typeof op.c === 'number' && op.c >= 0 && op.c <= 7) hiInk = op.c | 0; break; // c absent/hors 0-7 → encre inchangée
      case 'curset': penx = x; peny = y; break;
      case 'point': penx = x; peny = y; hiSet(px, x, y); break;
      case 'line': hiLine(px, penx, peny, x, y); penx = x; peny = y; break;
      case 'box': hiBox(px, penx, peny, x, y); penx = x; peny = y; break;
      case 'fillbox': hiFillBox(px, penx, peny, x, y); penx = x; peny = y; break;
      case 'circle': hiCircle(px, penx, peny, op.r | 0); break;
      case 'char': hiChar(px, x, y, (op.ch || ' ').charCodeAt(0)); break;
      default: break; // paper : non rendu dans l'aperçu
    }
  }
  const ctx = cv.getContext('2d');
  const img = ctx.createImageData(HIW, 224);
  for (let i = 0; i < HIW * 224; i++) {
    const v = i < HIW * HIH ? px[i] : 0;            // 0 = éteint, sinon encre+1
    const c = v ? PAL[(v - 1) & 7] : [0, 0, 0];
    const o = i * 4; img.data[o] = c[0]; img.data[o + 1] = c[1]; img.data[o + 2] = c[2]; img.data[o + 3] = 255;
  }
  ctx.putImageData(img, 0, 0);
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
// compositeur de ligne

// --- éditeur d'écran (modèle octets) ---
function colorSwatches(hostId, mk) {
  const host = $(hostId); host.innerHTML = '';
  COLOR_NAMES.forEach((n, i) => {
    const c = PAL[i];
    const b = el('button', { className: 'swatch', title: mk.label + ' ' + n });
    b.style.background = 'rgb(' + c[0] + ',' + c[1] + ',' + c[2] + ')';
    b.onclick = () => pickAttr(mk.attr(i));
    host.append(b);
  });
}
initGrid();
colorSwatches('ink-swatches', { label: 'encre', attr: (i) => i });          // encre = 0..7
colorSwatches('paper-swatches', { label: 'fond', attr: (i) => 0x10 | i });  // fond = 16..23
$('attr-alt').onclick = () => pickAttr(0x09);   // texte: charset alternatif
$('attr-blink').onclick = () => pickAttr(0x0C); // texte: clignotement
$('attr-norm').onclick = () => pickAttr(0x08);  // texte: normal
$('brush-char').oninput = () => { const v = $('brush-char').value; setBrush(((v ? v.charCodeAt(0) : 0x20) & 0x7F) | ($('brush-inv').checked ? 0x80 : 0)); };
$('brush-inv').onchange = () => { if ((brushByte & 0x60) !== 0) setBrush((brushByte & 0x7F) | ($('brush-inv').checked ? 0x80 : 0)); };
renderPaletteInto('screen-palette', (c) => dropGlyph(c)); // dépose le glyphe (+ alt si besoin)
setBrush(0x20);

const scv = $('screen-canvas');
scv.addEventListener('click', (e) => {
  const r = scv.getBoundingClientRect();
  cur.col = Math.min(COLS - 1, Math.max(0, Math.floor((e.clientX - r.left) / (r.width / COLS))));
  cur.row = Math.min(ROWS - 1, Math.max(0, Math.floor((e.clientY - r.top) / (r.height / ROWS))));
  paintAt(cur.col, cur.row); drawGrid(); scv.focus();
});
scv.addEventListener('keydown', (e) => {
  const k = e.key;
  if (k === 'ArrowLeft') cur.col = Math.max(0, cur.col - 1);
  else if (k === 'ArrowRight') cur.col = Math.min(COLS - 1, cur.col + 1);
  else if (k === 'ArrowUp') cur.row = Math.max(0, cur.row - 1);
  else if (k === 'ArrowDown') cur.row = Math.min(ROWS - 1, cur.row + 1);
  else if (k === 'Enter') { cur.col = 0; cur.row = Math.min(ROWS - 1, cur.row + 1); }
  else if (k === 'Backspace') { cur.col = Math.max(0, cur.col - 1); gridBuf[cur.row * COLS + cur.col] = 0x20; }
  else if (k === 'Delete') { gridBuf[cur.row * COLS + cur.col] = 0x20; }
  else if (k.length === 1 && !e.ctrlKey && !e.altKey && !e.metaKey) { typeAt(k); }
  else return;
  e.preventDefault(); drawGrid();
});
// Aperçu de grille DataWindow navigable au clavier (onglet Édition). Le canvas
// d'aperçu prend le focus au clic ; flèches ↑/↓ = sélection, S/R = pages, F = filtre.
const oscv = $('oric-screen');
if (oscv) {
  oscv.tabIndex = 0;
  oscv.addEventListener('keydown', (e) => {
    const p = current && site.pages[current];
    if (!p || !p.datawindow) return; // uniquement pour une page grille
    let handled = true;
    const k = e.key;
    if (k === 'ArrowRight') gridNav.scroll += 8;                 // scroll horizontal de la ligne
    else if (k === 'ArrowLeft') gridNav.scroll = Math.max(0, gridNav.scroll - 8);
    else if (k === 'ArrowDown') { gridNav.sel++; gridNav.scroll = 0; }
    else if (k === 'ArrowUp') { gridNav.sel--; gridNav.scroll = 0; }
    else if (k === 'PageDown' || k === 's' || k === 'S') { gridNav.n++; gridNav.sel = 0; gridNav.scroll = 0; }
    else if (k === 'PageUp' || k === 'r' || k === 'R') { gridNav.n = Math.max(1, gridNav.n - 1); gridNav.sel = 0; gridNav.scroll = 0; }
    else if (k === 'f' || k === 'F') { const f = prompt('Filtre LIKE (vide = tout)', gridNav.filtre); if (f !== null) { gridNav.filtre = f.trim(); gridNav.n = 1; gridNav.sel = 0; gridNav.scroll = 0; } }
    else if (k === 'c' || k === 'C') { gridNav.filtre = ''; gridNav.n = 1; gridNav.sel = 0; gridNav.scroll = 0; } // effacer le filtre
    else handled = false;
    if (handled) { e.preventDefault(); gridNav.sel = Math.max(0, Math.min(19, gridNav.sel)); doGridPreview(); }
  });
}
$('screen-load').onclick = () => screenLoad($('screen-page').value);
$('screen-new').onclick = screenNew;
$('screen-save').onclick = screenSave;
$('screen-clear').onclick = () => { initGrid(); drawGrid(); };

// onglet Données : sources DataWindow
$('src-load').onclick = () => { srcName = $('src-select').value; renderSourceForm(); };
$('src-select').onchange = () => { srcName = $('src-select').value; renderSourceForm(); };
$('src-new').onclick = srcCreate;
$('src-del').onclick = srcDelete;
$('btn-validate-data').onclick = validate;
$('btn-save-data').onclick = save;

for (const t of document.querySelectorAll('.tab')) t.onclick = () => { showTab(t.dataset.tab); if (t.dataset.tab === 'screen') { drawGrid(); renderScreenNav(); } if (t.dataset.tab === 'data') refreshSources(); };
showTab('nav');
loadSites(); // charge le 1er site, qui charge ses propres profils
