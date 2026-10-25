// The scene map editor: paints a format 2 map's ground and heights, places its objects, draws its wall lines and
// sets gates in them in the browser, and leaves the drawing to the server (viewer.js preview, the game's rules) and
// the writing to /api/save.
(() => {
  const viewer = window.sceneViewer;
  const bar = document.getElementById('editbar');
  const palette = document.getElementById('palette');
  const shelf = document.getElementById('shelf');
  const shelfTabs = document.getElementById('shelftabs');
  const shelfItems = document.getElementById('shelfitems');
  const search = document.getElementById('shelfsearch');
  const inspector = document.getElementById('inspector');
  const saveButton = document.getElementById('save');
  const undoButton = document.getElementById('undo');
  const redoButton = document.getElementById('redo');
  const state = document.getElementById('editstate');
  const GROUND = [['.', '풀', '#6f9a4a'], ['d', '흙', '#b08a55'], ['~', '물', '#4f86b8'], ['i', '포석', '#9a9a8e'],
    ['T', '군막 마루', '#9b7a4c'], ['W', '나무 마루', '#7b4a2a'], ['P', '궁궐 바닥', '#3c4a5e'], ['C', '붉은 깔개', '#8a1d33']];
  const HEIGHT = [['0', '평지', '#6f9a4a'], ['2', '높은 절벽', '#7a6040']];
  // pictures that are parts the rules lay (walls, gates' slices, shadows, cliff faces), not objects to place
  const PARTS = /^(swall|sgate|sparapet|jshade|cfoot|tcliff|gate_|bird_)/;
  const FAMILIES = { swall: '성벽' }, GATES = { sgate_open: '열린 성문', sgate_closed: '닫힌 성문' };
  const SHELVES = [['all', '전체', () => true],
    ['building', '건물', n => /^(house_|hall_|tavern|tent|pavilion|watchtower|burnt_|altar|dais)/.test(n)],
    ['nature', '나무·돌', n => /^(stree_|peach_tree|taihu_rock|well)/.test(n)],
    ['wall', '벽·휘장', n => /^(wall_|curtain_|screen_|sfence_|pillar_|shield_wall|antler_barrier)/.test(n)],
    ['flat', '바닥에 깔림', (n, meta) => meta.layer === 'ground'],
    ['prop', '소품', null]];
  let mode = 'view', brush = { ground: '.', height: '2', object: null, wall: 'swall', gate: 'sgate_open' }, size = 1, snap = true, shelfKey = 'all';
  let editing = null, undo = [], redo = [], stroke = null, selected = -1, drag = null;
  let checking = false, checked = null, checkSeq = 0;

  const editor = window.sceneEditor = {
    grid: false,
    get dirty() { return !!editing && undo.length > 0; },
    edited(id) { return editing?.id === id ? editing : null; },
    loaded(m) {
      if (editing?.id !== m.id) { editing = null; undo = []; redo = []; selected = -1; }
      bar.hidden = m.format !== 2;
      if (m.format === 2 && !editing) editing = structuredClone(m);
      refresh();
      check();
    },
    discard() { editing = null; undo = []; redo = []; selected = -1; checked = null; },
    // decorate marks the walkable tiles and the problems (검사), and the selected object, on the drawn map
    decorate(ctx, hits) {
      if (checking && checked && editing) {
        const diamond = (U, V) => {
          const [x, y] = viewer.tilePoint(U, V);
          ctx.moveTo(x, y - 8); ctx.lineTo(x + 16, y); ctx.lineTo(x, y + 8); ctx.lineTo(x - 16, y); ctx.closePath();
        };
        ctx.beginPath();
        checked.walk.forEach((row, y) => [...row].forEach((c, x) => { if (c === '1') diamond(x + editing.origin[0], y + editing.origin[1]); }));
        ctx.fillStyle = 'rgba(120, 255, 140, .18)'; ctx.fill();
        ctx.beginPath();
        for (const p of checked.problems) diamond(p.u, p.v);
        ctx.strokeStyle = '#ff4040'; ctx.lineWidth = 3; ctx.stroke();
      }
      if (mode !== 'object' || selected < 0) return;
      for (const h of hits) {
        if (h.obj !== selected) continue;
        ctx.save(); ctx.setLineDash([4, 3]); ctx.strokeStyle = '#ffe36e'; ctx.lineWidth = 2;
        ctx.strokeRect(h.x - 2, h.y - 2, h.w + 4, h.h + 4); ctx.restore();
      }
    },
  };

  function inside(U, V) {
    const x = U - editing.origin[0], y = V - editing.origin[1];
    return x >= 0 && y >= 0 && y < editing.ground.length && x < editing.ground[0].length;
  }
  function put(rows, U, V, c) {
    const x = U - editing.origin[0], y = V - editing.origin[1];
    if (rows[y][x] === c) return false;
    rows[y] = rows[y].slice(0, x) + c + rows[y].slice(x + 1);
    return true;
  }
  // paint lays the brush on the size × size small tiles from (U, V), the cursor's tile in their middle
  function paint(U, V) {
    const layer = mode;
    if (layer === 'height' && !editing.height) editing.height = editing.ground.map(row => '0'.repeat(row.length));
    const rows = editing[layer];
    const c = brush[layer], from = Math.floor((size - 1) / 2);
    let changed = false;
    for (let dv = 0; dv < size; dv++) {
      for (let du = 0; du < size; du++) {
        const u = U - from + du, v = V - from + dv;
        if (inside(u, v)) changed = put(rows, u, v, c) || changed;
      }
    }
    return changed;
  }
  // the map as it is sent: a height grid of nothing but level ground is left out
  function body() {
    const m = structuredClone(editing);
    if (m.height && m.height.every(row => /^0*$/.test(row))) delete m.height;
    return m;
  }
  function show() { viewer.preview(body()).then(check); refresh(); }
  // check asks the server for the problems and the walkable tiles of the map as it is now (the latest answer wins)
  async function check() {
    if (!checking || !editing) return;
    const seq = ++checkSeq;
    try {
      const result = await viewer.fetchJSON('/api/check', JSON.stringify(body()));
      if (seq !== checkSeq) return;
      checked = result;
    } catch (error) {
      checked = null; viewer.status(error.message);
    }
    viewer.render(); refresh();
  }
  function remember() { undo.push(structuredClone(editing)); if (undo.length > 50) undo.shift(); redo = []; }

  // An object [name, U, V, dx, dy] stands dx, dy canvas pixels from the centre of small tile (U, V): on the ground
  // at (U + .5 + dx/32 + dy/16, V + .5 - dx/32 + dy/16).
  const objectPoint = o => [o[1] + .5 + (o[3] || 0) / 32 + (o[4] || 0) / 16, o[2] + .5 - (o[3] || 0) / 32 + (o[4] || 0) / 16];
  function objectAt(name, cu, cv) {
    const U = Math.floor(cu), V = Math.floor(cv);
    if (snap) return [name, U, V];
    const fu = cu - U - .5, fv = cv - V - .5, dx = Math.round(16 * (fu - fv)), dy = Math.round(8 * (fu + fv));
    return dx || dy ? [name, U, V, dx, dy] : [name, U, V];
  }

  // A wall line [family, [U, V], [U, V]] runs along u or along v; a gate is an object standing on one, its slices
  // over the small tiles gate_offset game cells from its point.
  function lineTiles(w) {
    const [, a, b] = w, out = [];
    if (a[1] === b[1] && a[0] !== b[0]) for (let u = Math.min(a[0], b[0]); u <= Math.max(a[0], b[0]); u++) out.push([u, a[1]]);
    else for (let v = Math.min(a[1], b[1]); v <= Math.max(a[1], b[1]); v++) out.push([a[0], v]);
    return out;
  }
  const lineAxis = w => w[1][1] === w[2][1] && w[1][0] !== w[2][0] ? 'u' : 'v';
  const onLine = (w, U, V) => lineTiles(w).some(([u, v]) => u === U && v === V);
  function gateSpan(o) {
    const w = (editing.walls || []).find(w => onLine(w, o[1], o[2]));
    if (!w) return [];
    const axis = lineAxis(w), slices = viewer.rules.gates[o[0]], out = [];
    for (let k = 0; ; k++) {
      const meta = viewer.spriteMeta(`${slices}_${axis}_${k}`)[0];
      if (!meta || meta.gate_offset === undefined) return out;
      const t = Math.round(2 * meta.gate_offset);
      out.push(axis === 'u' ? [o[1] + t, o[2]] : [o[1], o[2] + t]);
    }
  }
  const gateAt = (U, V) => editing.objects.findIndex(o => o[0] in viewer.rules.gates && gateSpan(o).some(([u, v]) => u === U && v === V));
  // the wall tile the cursor points at: the one whose picture is under it, else the ground tile
  function wallTile(h) {
    const parts = Object.values(viewer.rules.walls).flatMap(f => [f.block, f.parapet, f.shade]).concat(Object.values(viewer.rules.gates));
    if (h.hit && parts.some(p => h.hit.name.startsWith(p))) return [h.hit.U, h.hit.V];
    return [h.U, h.V];
  }
  // cut takes small tile (U, V) out of every wall line through it, and the gates whose slices stood on it
  function cut(U, V) {
    let changed = false;
    for (let i = editing.objects.length - 1; i >= 0; i--) {
      const o = editing.objects[i];
      if (o[0] in viewer.rules.gates && gateSpan(o).some(([u, v]) => u === U && v === V)) { editing.objects.splice(i, 1); changed = true; }
    }
    const walls = [];
    for (const w of editing.walls || []) {
      if (!onLine(w, U, V)) { walls.push(w); continue; }
      changed = true;
      const tiles = lineTiles(w), k = tiles.findIndex(([u, v]) => u === U && v === V);
      for (const part of [tiles.slice(0, k), tiles.slice(k + 1)]) if (part.length) walls.push([w[0], part[0], part.at(-1)]);
    }
    if (walls.length) editing.walls = walls; else delete editing.walls;
    if (changed) selected = -1;
    return changed;
  }

  function begin(event) {
    const h = viewer.hovered;
    if (mode === 'view' || !editing || !h) return;
    if (mode === 'object') return beginObject(event, h);
    if (mode === 'wall') return beginWall(event, h);
    if (mode === 'gate') return beginGate(event, h);
    if (event.button === 2) { // pick what lies under the cursor
      const x = h.U - editing.origin[0], y = h.V - editing.origin[1];
      if (inside(h.U, h.V)) brush[mode] = mode === 'ground' ? editing.ground[y][x] : (editing.height?.[y][x] ?? '0');
      refresh();
      return;
    }
    if (event.button !== 0) return;
    event.preventDefault();
    stroke = { before: structuredClone(editing), last: null, changed: false };
    move();
  }
  function beginWall(event, h) {
    if (event.button !== 0 && event.button !== 2) return;
    event.preventDefault();
    stroke = { before: structuredClone(editing), last: null, changed: false, erase: event.button === 2, from: [h.U, h.V], walls: structuredClone(editing.walls || []) };
    move();
  }
  // a line from where the drag began to the cursor, along whichever axis the cursor has gone further
  function drawLine(h) {
    if (!inside(...stroke.from)) return false;
    const [U0, V0] = stroke.from, du = h.U - U0, dv = h.V - V0;
    const to = Math.abs(du) >= Math.abs(dv) ? [U0 + du, V0] : [U0, V0 + dv];
    while (!inside(...to)) to[to[0] !== U0 ? 0 : 1] -= Math.sign(to[0] !== U0 ? du : dv);
    editing.walls = [...structuredClone(stroke.walls), [brush.wall, [U0, V0], to]];
    return true;
  }
  function beginGate(event, h) {
    const [U, V] = wallTile(h), at = gateAt(U, V);
    if (event.button === 2) {
      if (at >= 0) { remember(); editing.objects.splice(at, 1); selected = -1; show(); }
      return;
    }
    if (event.button !== 0) return;
    event.preventDefault();
    if (at >= 0) { // a gate there already: make it the brush's (open or closed)
      if (editing.objects[at][0] !== brush.gate) { remember(); editing.objects[at][0] = brush.gate; show(); }
      return;
    }
    const w = (editing.walls || []).find(w => onLine(w, U, V));
    if (!w) { viewer.status(`(${U}, ${V})는 성벽 줄 위가 아님: 성문은 성벽 줄 위에 놓는다`); return; }
    const o = [brush.gate, U, V], span = gateSpan(o);
    if (!span.every(([u, v]) => onLine(w, u, v))) { viewer.status(`성문은 성벽 줄 안에 ${span.length}칸이 다 들어가야 함`); return; }
    if (span.some(([u, v]) => gateAt(u, v) >= 0)) { viewer.status('다른 성문과 겹침'); return; }
    remember(); editing.objects.push(o); show();
  }
  function beginObject(event, h) {
    const obj = h.hit?.obj ?? -1;
    if (event.button === 2) {
      if (obj >= 0) { brush.object = editing.objects[obj][0]; refresh(); }
      return;
    }
    if (event.button !== 0) return;
    event.preventDefault();
    if (obj >= 0) {
      selected = obj;
      const o = editing.objects[obj];
      drag = { before: structuredClone(editing), from: [h.cu, h.cv], at: objectPoint(o), moved: false };
      viewer.render(); refresh();
      return;
    }
    if (!brush.object || !inside(Math.floor(h.cu), Math.floor(h.cv))) return;
    remember();
    editing.objects.push(objectAt(brush.object, h.cu, h.cv));
    selected = editing.objects.length - 1;
    show();
  }
  function move() {
    const h = viewer.hovered;
    if (drag && h) {
      const cu = drag.at[0] + h.cu - drag.from[0], cv = drag.at[1] + h.cv - drag.from[1];
      const o = editing.objects[selected], next = objectAt(o[0], cu, cv);
      if (JSON.stringify(next) !== JSON.stringify(o) && inside(next[1], next[2])) {
        editing.objects[selected] = next; drag.moved = true; show();
      }
      return;
    }
    if (!stroke || !h) return;
    // a stroke acts once per ground tile the cursor crosses; erasing takes the wall whose picture is under it
    if (stroke.last === `${h.U},${h.V}`) return;
    stroke.last = `${h.U},${h.V}`;
    const [U, V] = mode === 'wall' && stroke.erase ? wallTile(h) : [h.U, h.V];
    const changed = mode === 'wall' ? (stroke.erase ? cut(U, V) : drawLine(h)) : paint(U, V);
    if (changed) { stroke.changed = true; show(); }
  }
  function end() {
    if (drag) {
      if (drag.moved) { undo.push(drag.before); if (undo.length > 50) undo.shift(); redo = []; }
      drag = null; refresh();
    }
    if (!stroke) return;
    if (stroke.changed) { undo.push(stroke.before); if (undo.length > 50) undo.shift(); redo = []; }
    stroke = null;
    refresh();
  }
  function step(from, to) {
    if (!from.length) return;
    to.push(structuredClone(editing));
    editing = from.pop();
    if (selected >= editing.objects.length) selected = -1;
    show();
  }
  function change(fn) {
    if (selected < 0) return;
    remember(); fn(editing.objects[selected]); show();
  }
  function remove() {
    if (selected < 0) return;
    remember(); editing.objects.splice(selected, 1); selected = -1; show();
  }
  async function save() {
    if (!editing) return;
    try {
      const response = await fetch(`/api/save/${encodeURIComponent(editing.id)}`, { method: 'POST', body: JSON.stringify(body()) });
      if (!response.ok) throw new Error((await response.text()).trim());
      undo = []; redo = [];
      viewer.status(`${editing.id} 저장함`);
    } catch (error) {
      viewer.status(`저장하지 못함: ${error.message}`);
    }
    refresh();
  }

  // the shelf of pictures to place, by kind, with a thumbnail each
  let thumbs = null;
  function objectNames() {
    const g = viewer.graphics, gates = viewer.rules.gates || {};
    return [...new Set([...Object.keys(g.props.sprites), ...Object.keys(g.extra.sprites)])]
      .filter(n => !PARTS.test(n) && !(n in gates)).sort();
  }
  function thumb(name) {
    const [meta, img] = viewer.spriteMeta(name);
    const [sx, sy, w, h] = meta.rect, k = Math.min(1, 56 / Math.max(w, h)) * (Math.max(w, h) < 28 ? 2 : 1);
    const c = document.createElement('canvas'); c.width = 56; c.height = 56;
    const x = c.getContext('2d'); x.imageSmoothingEnabled = false;
    x.drawImage(img, sx, sy, w, h, (56 - w * k) / 2, (56 - h * k) / 2, w * k, h * k);
    return c;
  }
  function fillShelf() {
    const q = search.value.trim();
    const [, , test] = SHELVES.find(s => s[0] === shelfKey);
    const named = SHELVES.filter(s => s[2] && s[0] !== 'all');
    shelfItems.replaceChildren(...objectNames().filter(n => {
      const meta = viewer.spriteMeta(n)[0];
      const kind = test ? test(n, meta) : !named.some(s => s[2](n, meta));
      return kind && (!q || n.includes(q));
    }).map(n => {
      thumbs ||= new Map();
      if (!thumbs.has(n)) thumbs.set(n, thumb(n));
      const b = document.createElement('button');
      b.title = n; b.dataset.name = n;
      b.append(thumbs.get(n), Object.assign(document.createElement('span'), { textContent: n }));
      b.setAttribute('aria-pressed', String(n === brush.object));
      b.addEventListener('click', () => { brush.object = n; refresh(); });
      return b;
    }));
  }

  function refresh() {
    for (const b of bar.querySelectorAll('[data-mode]')) b.setAttribute('aria-pressed', String(b.dataset.mode === mode));
    for (const b of bar.querySelectorAll('[data-size]')) b.setAttribute('aria-pressed', String(Number(b.dataset.size) === size));
    document.getElementById('sizes').hidden = mode !== 'ground' && mode !== 'height';
    document.getElementById('grid').setAttribute('aria-pressed', String(editor.grid));
    document.getElementById('snap').setAttribute('aria-pressed', String(snap));
    document.getElementById('snap').hidden = mode !== 'object';
    const list = mode === 'ground' ? GROUND : mode === 'height' ? HEIGHT
      : mode === 'wall' ? Object.keys(viewer.rules?.walls || {}).map(k => [k, FAMILIES[k] || k, '#d6c8a8'])
        : mode === 'gate' ? Object.keys(viewer.rules?.gates || {}).map(k => [k, GATES[k] || k, k.includes('open') ? '#3a2e24' : '#7e4a2a']) : [];
    const current = brush[mode];
    const known = list.some(([c]) => c === current);
    palette.replaceChildren(...[...list, ...(known || !list.length ? [] : [[current, `'${current}'`, '#555']])].map(([c, name, color]) => {
      const b = document.createElement('button');
      b.innerHTML = `<i style="background:${color}"></i>${name}`;
      b.setAttribute('aria-pressed', String(c === current));
      b.addEventListener('click', () => { brush[mode] = c; refresh(); });
      return b;
    }));
    shelf.hidden = mode !== 'object' || bar.hidden;
    if (!shelf.hidden) {
      for (const b of shelfTabs.querySelectorAll('button')) b.setAttribute('aria-pressed', String(b.dataset.shelf === shelfKey));
      for (const b of shelfItems.querySelectorAll('button')) b.setAttribute('aria-pressed', String(b.dataset.name === brush.object));
    }
    const o = mode === 'object' && editing ? editing.objects[selected] : null;
    inspector.hidden = !o;
    if (o) {
      document.getElementById('objname').textContent = o[0];
      for (const [i, id] of [[1, 'obju'], [2, 'objv'], [3, 'objdx'], [4, 'objdy']]) {
        const input = document.getElementById(id);
        if (document.activeElement !== input) input.value = o[i] || 0;
      }
    }
    document.getElementById('check').setAttribute('aria-pressed', String(checking));
    const problems = document.getElementById('problems');
    problems.hidden = !checking || !checked;
    if (!problems.hidden) {
      problems.replaceChildren(checked.problems.length ? `문제 ${checked.problems.length}개` : '문제 없음',
        ...checked.problems.map(p => Object.assign(document.createElement('li'), { textContent: `(${p.u}, ${p.v}) ${p.msg}` })));
    }
    undoButton.disabled = !undo.length; redoButton.disabled = !redo.length;
    saveButton.disabled = !editor.dirty;
    state.textContent = !editing ? '' : editor.dirty ? '저장하지 않은 편집 있음' : '저장된 상태';
    viewer.surface.style.cursor = mode === 'view' ? '' : 'crosshair';
  }

  shelfTabs.replaceChildren(...SHELVES.map(([key, label]) => {
    const b = document.createElement('button'); b.textContent = label; b.dataset.shelf = key;
    b.addEventListener('click', () => { shelfKey = key; fillShelf(); refresh(); });
    return b;
  }));
  search.addEventListener('input', fillShelf);
  for (const b of bar.querySelectorAll('[data-mode]')) b.addEventListener('click', () => {
    mode = b.dataset.mode;
    if (mode === 'object' && !thumbs) fillShelf();
    viewer.render(); refresh();
  });
  for (const b of bar.querySelectorAll('[data-size]')) b.addEventListener('click', () => { size = Number(b.dataset.size); refresh(); });
  document.getElementById('grid').addEventListener('click', () => { editor.grid = !editor.grid; viewer.render(); refresh(); });
  document.getElementById('check').addEventListener('click', () => { checking = !checking; if (!checking) viewer.render(); check(); refresh(); });
  document.getElementById('snap').addEventListener('click', () => { snap = !snap; refresh(); });
  document.getElementById('objdelete').addEventListener('click', remove);
  document.getElementById('objswap').addEventListener('click', () => { if (brush.object) change(o => { o[0] = brush.object; }); });
  for (const [i, id] of [[1, 'obju'], [2, 'objv'], [3, 'objdx'], [4, 'objdy']]) {
    document.getElementById(id).addEventListener('change', event => {
      const n = Math.round(Number(event.target.value));
      if (!Number.isFinite(n)) return;
      change(o => {
        while (o.length < 5 && i >= 3) o.push(0);
        o[i] = n;
        if (o.length === 5 && !o[3] && !o[4]) o.length = 3;
      });
    });
  }
  undoButton.addEventListener('click', () => step(undo, redo));
  redoButton.addEventListener('click', () => step(redo, undo));
  saveButton.addEventListener('click', save);
  viewer.surface.addEventListener('mousedown', begin);
  viewer.surface.addEventListener('mousemove', move);
  window.addEventListener('mouseup', end);
  viewer.surface.addEventListener('contextmenu', event => { if (mode !== 'view') event.preventDefault(); });
  window.addEventListener('keydown', event => {
    if (!editing || event.target.tagName === 'INPUT') return;
    const key = event.key.toLowerCase();
    if (event.ctrlKey || event.metaKey) {
      if (key === 's') { event.preventDefault(); save(); }
      if (key === 'z') { event.preventDefault(); event.shiftKey ? step(redo, undo) : step(undo, redo); }
      return;
    }
    if (mode !== 'object' || selected < 0) return;
    if (key === 'delete' || key === 'backspace') { event.preventDefault(); remove(); }
    if (key === 'escape') { selected = -1; viewer.render(); refresh(); }
    // arrows move the selected object a pixel on screen (8 with Shift), turned back into the map's own axes
    const d = { arrowleft: [-1, 0], arrowright: [1, 0], arrowup: [0, -1], arrowdown: [0, 1] }[key];
    if (!d) return;
    event.preventDefault();
    const k = event.shiftKey ? 8 : 1, ru = d[0] * k / 32 + d[1] * k / 16, rv = d[1] * k / 16 - d[0] * k / 32;
    const [cu, cv] = [[ru, rv], [-rv, ru], [-ru, -rv], [rv, -ru]][viewer.turn];
    change(o => {
      while (o.length < 5) o.push(0);
      o[3] += Math.round(16 * (cu - cv)); o[4] += Math.round(8 * (cu + cv));
      if (!o[3] && !o[4]) o.length = 3;
    });
  });
  window.addEventListener('beforeunload', event => { if (editor.dirty) event.preventDefault(); });
  refresh();
})();
