(() => {
  const viewport = document.getElementById('viewport');
  const surface = document.getElementById('surface');
  const canvas = document.getElementById('map');
  const hoverCanvas = document.getElementById('hover');
  const ctx = canvas.getContext('2d');
  const hoverCtx = hoverCanvas.getContext('2d');
  const select = document.getElementById('scene');
  const info = document.getElementById('info');
  const cursor = document.getElementById('cursor');
  const params = new URLSearchParams(location.search);
  let graphics, rules, scenery, map, plan, baseMap, images = {}, left, top;
  let turn = Math.max(0, Math.min(3, Number(params.get('turn')) || 0));
  let zoom = 1, pointer = null, hovered = null, animating = false;
  // what is shown: the layers (0 flat, 1 + standing, 2 all), decoration, the footprints and the sort order, and for
  // a format 2 map which picture: the rules' ('new'), the comparison revision's the old way ('old'), or the two compared
  let step = params.has('step') ? Number(params.get('step')) : 2;
  let decor = params.get('decor') !== '0', showFoot = params.get('foot') === '1', showOrder = params.get('order') === '1';
  let view = params.get('view') || 'new';
  let hits = [], objectIndex = new Map();
  const whiteMasks = new Map(), alphaMasks = new Map();

  function unrotate(u, v) {
    if (turn === 1) return [-v, u];
    if (turn === 2) return [-u, -v];
    if (turn === 3) return [v, -u];
    return [u, v];
  }
  function rotate(u, v) {
    if (turn === 1) return [v, -u];
    if (turn === 2) return [-u, -v];
    if (turn === 3) return [-v, u];
    return [u, v];
  }
  function turnSide(side) {
    const sides = ['NW', 'NE', 'SE', 'SW'];
    return sides[(sides.indexOf(side) + 4 - turn) % 4];
  }
  function propPoint(u, v) {
    [u, v] = rotate(u, v);
    return [((u - v) * 16 - left) * 2, ((u + v + 1) * 8 - top) * 2];
  }
  function measure(bounds, tall) {
    const [x0, y0, x1, y1] = bounds;
    const corners = [[x0, y0], [x0, y1], [x1, y0], [x1, y1]].map(([x, y]) => {
      const centerY = y + 2;
      const [u, v] = rotate(x / 32 + centerY / 16 - .5, centerY / 16 - x / 32 - .5);
      return [16 * (u - v), 8 * (u + v + 1) - 2];
    });
    left = Math.floor(Math.min(...corners.map(p => p[0])) - 100);
    const right = Math.ceil(Math.max(...corners.map(p => p[0])) + 100);
    top = Math.floor(Math.min(...corners.map(p => p[1])) - (tall ? 190 : 110));
    const bottom = Math.ceil(Math.max(...corners.map(p => p[1])) + 100);
    canvas.width = hoverCanvas.width = (right - left) * 2;
    canvas.height = hoverCanvas.height = (bottom - top) * 2;
    ctx.imageSmoothingEnabled = hoverCtx.imageSmoothingEnabled = false;
  }
  function spriteMeta(name) {
    const meta = graphics.props.sprites[name];
    if (meta) return [meta, images.props];
    const extra = graphics.extra.sprites[name];
    return extra ? [extra, images.extra] : [null, null];
  }
  function structureMeta(name) {
    const st = graphics.structures, [w, h] = st.cell;
    return { rect: [st.sprites[name] * w, 0, w, h], pivot: st.pivot };
  }
  // one picture of a plan (internal/scenemap Draw), its canvas pixels offset by the view's origin
  function draw(c, d) {
    const ox = -2 * left, oy = -2 * top;
    if (d.k === 'tile') {
      const r = graphics.tileset.sprites[d.t];
      c.drawImage(images.tiles, r[0], r[1], r[2], r[3], ox + d.x, oy + d.y, r[2], r[3]);
    } else if (d.k === 'sprite') {
      const [meta, img] = d.s === 'scenery' ? [scenery.sprites[d.n], images.scenery]
        : d.s === 'structures' ? [structureMeta(d.n), images.structures] : spriteMeta(d.n);
      const [sx, sy, w, h] = meta.rect;
      if (d.clip) { // cut to a polygon: a cliff face's lip, drawn with the top it lies on
        c.save(); c.beginPath();
        for (const [x, y] of d.clip) c.lineTo(ox + x, oy + y);
        c.clip();
      }
      c.drawImage(img, sx, sy, w, h, ox + d.x, oy + d.y, w, h);
      if (d.clip) c.restore();
      return { name: d.n, actual: d.n, x: ox + d.x, y: oy + d.y, w, h, meta, img };
    } else if (d.k === 'anim') {
      const meta = graphics.anims[d.n], img = images[`anim:${d.n}`], [sx, sy, w, h] = meta.rect;
      if (img) c.drawImage(img, sx, sy, w, h, ox + d.x, oy + d.y, w, h); // frame 0; the game plays the strip
    } else if (d.k === 'fill') {
      c.fillStyle = d.c;
      for (let k = 0; k < 32; k++) {
        const hw = k < 16 ? 2 * (k + 1) : 2 * (32 - k);
        c.fillRect(ox + d.x - hw, oy + d.y + k, 2 * hw, 1);
      }
    }
    return null;
  }
  function renderPlan(c) {
    const out = { hits: [], index: new Map() };
    for (const d of plan.flat) {
      const hit = draw(c, d);
      if (hit && d.o) { hit.obj = d.o - 1; out.hits.push(hit); }
    }
    if (step < 1) return out;
    const stand = solid => {
      for (const p of solid) {
        for (const d of p.draws) {
          const hit = draw(c, d);
          if (!hit) continue;
          hit.U = p.at[0]; hit.V = p.at[1];
          if (d.o) hit.obj = d.o - 1;
          out.hits.push(hit);
          const key = `${p.at[0]},${p.at[1]}`;
          if (!out.index.has(key)) out.index.set(key, []);
          out.index.get(key).push(hit);
        }
      }
    };
    // the ground's standing layer, then each raised level: its tops over all that stands lower, then what stands on it
    stand(plan.solid);
    for (const level of plan.levels || []) {
      for (const d of level.flat) draw(c, d);
      stand(level.solid);
    }
    return out;
  }
  function renderLegacy(c, m, withDecor) {
    return legacyScene.render({ ctx: c, map: m, graphics, images, turn, left, top, step: step >= 1 ? 2 : 0, decor: withDecor });
  }
  function offscreen() {
    const c = document.createElement('canvas'); c.width = canvas.width; c.height = canvas.height;
    const x = c.getContext('2d'); x.imageSmoothingEnabled = false;
    x.fillStyle = '#675d42'; x.fillRect(0, 0, c.width, c.height);
    return x;
  }
  // compare draws the old picture and the rules' picture without decoration and marks every pixel that differs:
  // magenta over the rules' picture, dimmed. Clusters of differing pixels go to window.sceneDiff for the report.
  function compare() {
    const a = offscreen(), b = offscreen();
    renderLegacy(a, baseMap, false);
    const drawn = renderPlan(b);
    const w = canvas.width, h = canvas.height;
    const da = a.getImageData(0, 0, w, h), db = b.getImageData(0, 0, w, h);
    const out = ctx.createImageData(w, h);
    const cell = 32, cols = Math.ceil(w / cell), rows = Math.ceil(h / cell), counts = new Int32Array(cols * rows);
    let count = 0;
    for (let i = 0; i < w * h; i++) {
      const o = i * 4;
      const same = da.data[o] === db.data[o] && da.data[o + 1] === db.data[o + 1] && da.data[o + 2] === db.data[o + 2];
      if (same) {
        out.data[o] = db.data[o] * .4; out.data[o + 1] = db.data[o + 1] * .4; out.data[o + 2] = db.data[o + 2] * .4;
      } else {
        out.data[o] = 255; out.data[o + 1] = 0; out.data[o + 2] = 255; count++;
        counts[Math.floor((i % w) / cell) + cols * Math.floor(i / w / cell)]++;
      }
      out.data[o + 3] = 255;
    }
    ctx.putImageData(out, 0, 0);
    const seen = new Uint8Array(cols * rows), boxes = [];
    for (let s = 0; s < cols * rows; s++) {
      if (!counts[s] || seen[s]) continue;
      const box = { x0: w, y0: h, x1: 0, y1: 0, pixels: 0 }, stack = [s]; seen[s] = 1;
      while (stack.length) {
        const k = stack.pop(), x = k % cols, y = Math.floor(k / cols);
        box.x0 = Math.min(box.x0, x * cell); box.y0 = Math.min(box.y0, y * cell);
        box.x1 = Math.max(box.x1, (x + 1) * cell); box.y1 = Math.max(box.y1, (y + 1) * cell); box.pixels += counts[k];
        for (const [dx, dy] of [[1, 0], [-1, 0], [0, 1], [0, -1]]) {
          const nx = x + dx, ny = y + dy, n = nx + cols * ny;
          if (nx >= 0 && ny >= 0 && nx < cols && ny < rows && counts[n] && !seen[n]) { seen[n] = 1; stack.push(n); }
        }
      }
      boxes.push(box);
    }
    boxes.sort((p, q) => q.pixels - p.pixels);
    window.sceneDiff = { count, boxes, old: a.canvas.toDataURL('image/png'), new: b.canvas.toDataURL('image/png') };
    return { count, drawn };
  }
  function overlay() {
    const ox = -2 * left, oy = -2 * top;
    [...plan.solid, ...(plan.levels || []).flatMap(l => l.solid)].forEach((p, i) => {
      if (p.rank === 0) return; // the raised ground: a footprint and a number on every tile would hide the map
      const [u0, v0, u1, v1] = p.box;
      const pt = (u, v) => [ox + 16 * (u - v), oy + 8 * (u + v) - p.lift];
      if (showFoot) {
        const wide = u1 - u0 > 1 || v1 - v0 > 1;
        ctx.beginPath();
        for (const [u, v] of [[u0, v0], [u1, v0], [u1, v1], [u0, v1]]) ctx.lineTo(...pt(u, v));
        ctx.closePath();
        ctx.strokeStyle = wide ? '#3ce0ff' : 'rgba(255, 255, 160, .7)'; ctx.lineWidth = 1; ctx.stroke();
      }
      if (showOrder) {
        const [x, y] = pt((u0 + u1) / 2, (v0 + v1) / 2);
        ctx.font = '9px system-ui, sans-serif'; ctx.textAlign = 'center';
        ctx.fillStyle = 'rgba(0, 0, 0, .75)'; ctx.fillRect(x - 9, y - 6, 18, 11);
        ctx.fillStyle = '#fff'; ctx.fillText(String(i), x, y + 3);
      }
    });
  }
  // grid outlines every small tile of the map's grid, on its ground or on its raised top
  function grid() {
    ctx.strokeStyle = 'rgba(255, 255, 255, .22)'; ctx.lineWidth = 1;
    ctx.beginPath();
    for (let y = 0; y < map.ground.length; y++) {
      for (let x = 0; x < map.ground[0].length; x++) {
        const U = x + map.origin[0], V = y + map.origin[1];
        const [cx, cy] = propPoint((U - .5) / 2, (V - .5) / 2), up = lift(U, V);
        ctx.moveTo(cx, cy - 8 - up); ctx.lineTo(cx + 16, cy - up); ctx.lineTo(cx, cy + 8 - up); ctx.lineTo(cx - 16, cy - up); ctx.closePath();
      }
    }
    ctx.stroke();
  }
  function render() {
    if (!map) return;
    window.sceneReady = false;
    ctx.fillStyle = '#675d42'; ctx.fillRect(0, 0, canvas.width, canvas.height);
    let drawn, note = '';
    if (map.format !== 2) {
      drawn = renderLegacy(ctx, map, decor);
      note = `${map.id} · 옛 형식 · 바닥 ${map.ground.length.toLocaleString()} · 장식 ${(map.decorations || []).length.toLocaleString()} · 소품 ${map.props.length.toLocaleString()}`;
    } else if (view === 'diff' && baseMap) {
      const r = compare();
      drawn = r.drawn;
      note = `${map.id} · 비교(장식 없이): 기존과 다른 픽셀 ${r.count.toLocaleString()}개 · 묶음 ${window.sceneDiff.boxes.length}개`;
    } else if (view === 'old' && baseMap) {
      drawn = renderLegacy(ctx, baseMap, decor);
      note = `${map.id} · 기존 데이터를 옛 방식으로`;
    } else {
      drawn = renderPlan(ctx);
      if (showFoot || showOrder) overlay();
      if (window.sceneEditor?.grid) grid();
      window.sceneEditor?.decorate(ctx, drawn.hits);
      note = `${map.id} · 형식 2 · 평면 ${plan.flat.length.toLocaleString()} · 입체 ${[...plan.solid, ...(plan.levels || []).flatMap(l => l.solid)].length.toLocaleString()}`;
    }
    hits = drawn.hits; objectIndex = drawn.index;
    for (const button of document.querySelectorAll('[data-step]')) button.setAttribute('aria-pressed', String(Number(button.dataset.step) === step));
    for (const button of document.querySelectorAll('[data-turn]')) button.setAttribute('aria-pressed', String(Number(button.dataset.turn) === turn));
    for (const button of document.querySelectorAll('[data-view]')) {
      button.setAttribute('aria-pressed', String(button.dataset.view === view));
      button.disabled = map.format !== 2;
    }
    for (const [id, on] of [['decor', decor], ['foot', showFoot], ['order', showOrder]]) document.getElementById(id).setAttribute('aria-pressed', String(on));
    info.textContent = note;
    if (pointer) updateHover(pointer);
    window.sceneReady = true;
  }
  function alphaMask(hit) {
    if (alphaMasks.has(hit.actual)) return alphaMasks.get(hit.actual);
    const [sx, sy, w, h] = hit.meta.rect;
    const temp = document.createElement('canvas'); temp.width = w; temp.height = h;
    const c = temp.getContext('2d'); c.drawImage(hit.img, sx, sy, w, h, 0, 0, w, h);
    const alpha = c.getImageData(0, 0, w, h).data;
    alphaMasks.set(hit.actual, alpha);
    return alpha;
  }
  function hitObject(x, y) {
    for (let i = hits.length - 1; i >= 0; i--) {
      const item = hits[i];
      const px = Math.floor(x - item.x), py = Math.floor(y - item.y);
      if (px < 0 || py < 0 || px >= item.w || py >= item.h) continue;
      if (alphaMask(item)[(py * item.w + px) * 4 + 3]) return item;
    }
    return null;
  }
  function whiteMask(hit) {
    if (whiteMasks.has(hit.actual)) return whiteMasks.get(hit.actual);
    const [sx, sy, w, h] = hit.meta.rect;
    const temp = document.createElement('canvas'); temp.width = w; temp.height = h;
    const c = temp.getContext('2d'); c.drawImage(hit.img, sx, sy, w, h, 0, 0, w, h);
    c.globalCompositeOperation = 'source-in'; c.fillStyle = '#fff'; c.fillRect(0, 0, w, h);
    whiteMasks.set(hit.actual, temp);
    return temp;
  }
  function lift(U, V) {
    if (map.format === 2) {
      const s = map.height?.[V - map.origin[1]]?.[U - map.origin[0]];
      return s && s !== '0' ? rules.heights[s].lift : 0;
    }
    return map.highland?.[V - map.visual[1]]?.[U - map.visual[0]] === '1' ? 146 : 0;
  }
  function pulse(now) {
    hoverCtx.clearRect(0, 0, hoverCanvas.width, hoverCanvas.height);
    if (!hovered) { animating = false; return; }
    const { U, V, hit } = hovered;
    const alpha = .75 * (1 + Math.sin(2 * Math.PI * now / 2000)) / 2;
    const [x, y] = propPoint((U - .5) / 2, (V - .5) / 2);
    const up = lift(U, V);
    hoverCtx.globalAlpha = alpha * .65; hoverCtx.fillStyle = '#fff'; hoverCtx.beginPath();
    hoverCtx.moveTo(x, y - 8 - up); hoverCtx.lineTo(x + 16, y - up);
    hoverCtx.lineTo(x, y + 8 - up); hoverCtx.lineTo(x - 16, y - up); hoverCtx.closePath(); hoverCtx.fill();
    hoverCtx.globalAlpha = alpha;
    for (const item of objectIndex.get(`${U},${V}`) || []) if (item.meta) hoverCtx.drawImage(whiteMask(item), item.x, item.y);
    if (hit && (hit.U !== U || hit.V !== V)) hoverCtx.drawImage(whiteMask(hit), hit.x, hit.y);
    hoverCtx.globalAlpha = 1;
    requestAnimationFrame(pulse);
  }
  function updateHover(point) {
    if (!map) return;
    const rect = canvas.getBoundingClientRect();
    const x = (point.clientX - rect.left) * canvas.width / rect.width;
    const y = (point.clientY - rect.top) * canvas.height / rect.height;
    if (x < 0 || y < 0 || x >= canvas.width || y >= canvas.height) { hovered = null; return; }
    const a = (x / 2 + left) / 16, b = (y / 2 + top) / 8;
    let [u, v] = unrotate((a + b - 1) / 2, (b - a - 1) / 2);
    let U = Math.floor(2 * u + .5), V = Math.floor(2 * v + .5), cu, cv;
    if (map.format === 2) [U, V, cu, cv] = groundAt(x + 2 * left, y + 2 * top);
    const hit = hitObject(x, y);
    hovered = { U, V, hit, cu, cv };
    const names = [...new Set((objectIndex.get(`${U},${V}`) || []).map(item => item.side ? `${item.name}(${turnSide(item.side)})` : item.name))];
    const glyph = map.format === 2 ? map.ground[V - map.origin[1]]?.[U - map.origin[0]] : null;
    cursor.textContent = `작은 칸 (${U}, ${V})${glyph ? ` '${glyph}'` : ''} · 논리 (${u.toFixed(2)}, ${v.toFixed(2)})${hit ? ` · 그림 ${hit.name} (${hit.U}, ${hit.V})` : ''}${names.length ? ` · ${names.join(', ')}` : ''}`;
    if (!animating) { animating = true; requestAnimationFrame(pulse); }
  }
  // groundAt is the small tile under plan point (px, py) of a format 2 map and the map's ground point there, in small
  // tiles (the inverse of internal/scenemap's view point): the top of a raised tile where one covers the point
  function groundAt(px, py) {
    const lifts = [...new Set(Object.values(rules.heights).map(h => h.lift))].sort((p, q) => q - p);
    for (const up of [...lifts, 0]) {
      const ru = (px / 16 + (py + up) / 8) / 2, rv = ((py + up) / 8 - px / 16) / 2;
      const [cu, cv] = turn === 1 ? [2 - rv, ru] : turn === 2 ? [2 - ru, 2 - rv] : turn === 3 ? [rv, 2 - ru] : [ru, rv];
      const U = Math.floor(cu), V = Math.floor(cv);
      if (up === 0 || lift(U, V) === up) return [U, V, cu, cv];
    }
  }
  function setZoom(next) {
    const centerX = (viewport.scrollLeft + viewport.clientWidth / 2) / zoom;
    const centerY = (viewport.scrollTop + viewport.clientHeight / 2) / zoom;
    zoom = next;
    const width = `${canvas.width * zoom}px`, height = `${canvas.height * zoom}px`;
    surface.style.width = width; surface.style.height = height;
    canvas.style.width = hoverCanvas.style.width = width;
    canvas.style.height = hoverCanvas.style.height = height;
    viewport.scrollLeft = centerX * zoom - viewport.clientWidth / 2;
    viewport.scrollTop = centerY * zoom - viewport.clientHeight / 2;
    for (const button of document.querySelectorAll('[data-zoom]')) button.setAttribute('aria-pressed', String(Number(button.dataset.zoom) === zoom));
  }
  function center() {
    viewport.scrollLeft = (canvas.width * zoom - viewport.clientWidth) / 2;
    viewport.scrollTop = (canvas.height * zoom - viewport.clientHeight) / 2;
  }
  async function fetchJSON(url, body) {
    const response = await fetch(url, body === undefined ? undefined : { method: 'POST', body });
    if (!response.ok) throw new Error(`${url}: ${(await response.text()).trim() || response.status}`);
    return response.json();
  }
  const planQuery = () => `turn=${turn}&decor=${decor && view !== 'diff' ? 1 : 0}`;
  // preview draws the map being edited by the rules: the server builds its plan from the map sent. A change made
  // while one is on its way waits for it, and only the latest is sent.
  let previewing = null, nextPreview = null;
  async function preview(m) {
    map = m;
    if (previewing) { nextPreview = m; return previewing; }
    previewing = (async () => {
      try {
        plan = await fetchJSON(`/api/preview?${planQuery()}`, JSON.stringify(m));
        render();
      } catch (error) {
        info.textContent = error.message;
      }
    })();
    await previewing;
    previewing = null;
    if (nextPreview) { const n = nextPreview; nextPreview = null; await preview(n); }
  }
  // load reads the map (and for a format 2 map its plan and its comparison revision) and draws it
  async function load(id) {
    window.sceneReady = false;
    const edited = window.sceneEditor?.edited(id);
    map = edited || await fetchJSON(`/api/map/${encodeURIComponent(id)}`);
    plan = baseMap = null;
    if (map.format === 2) {
      plan = edited ? await fetchJSON(`/api/preview?${planQuery()}`, JSON.stringify(map))
        : await fetchJSON(`/api/plan/${encodeURIComponent(id)}?${planQuery()}`);
      const base = await fetch(`/api/base/${encodeURIComponent(id)}`);
      baseMap = base.ok ? legacyScene.normalize(await base.json()) : null;
      if (baseMap?.format === 2) baseMap = null;
      measure(plan.bounds, plan.tall);
    } else {
      legacyScene.normalize(map);
      measure(map.bounds, !!map.highland);
    }
    whiteMasks.clear(); alphaMasks.clear(); hovered = pointer = null;
    cursor.textContent = '커서 좌표: —';
    setZoom(zoom); render(); center();
    const q = new URLSearchParams({ map: id, turn, view, decor: decor ? 1 : 0, step });
    if (showFoot) q.set('foot', 1);
    if (showOrder) q.set('order', 1);
    history.replaceState(null, '', `?${q}`);
    window.sceneEditor?.loaded(map);
  }
  const reload = () => load(map.id).catch(error => { info.textContent = error.message; });
  for (const button of document.querySelectorAll('[data-step]')) button.addEventListener('click', () => { step = Number(button.dataset.step); reload(); });
  for (const button of document.querySelectorAll('[data-zoom]')) button.addEventListener('click', () => setZoom(Number(button.dataset.zoom)));
  for (const button of document.querySelectorAll('[data-turn]')) button.addEventListener('click', () => { turn = Number(button.dataset.turn); reload(); });
  for (const button of document.querySelectorAll('[data-view]')) button.addEventListener('click', () => { view = button.dataset.view; reload(); });
  document.getElementById('decor').addEventListener('click', () => { decor = !decor; reload(); });
  document.getElementById('foot').addEventListener('click', () => { showFoot = !showFoot; reload(); });
  document.getElementById('order').addEventListener('click', () => { showOrder = !showOrder; reload(); });
  surface.addEventListener('mousemove', event => { pointer = { clientX: event.clientX, clientY: event.clientY }; updateHover(pointer); });
  surface.addEventListener('mouseleave', () => { hovered = pointer = null; cursor.textContent = '커서 좌표: —'; });
  select.addEventListener('change', () => {
    if (window.sceneEditor?.dirty && !confirm(`${map.id}의 저장하지 않은 편집을 버릴까요?`)) { select.value = map.id; return; }
    window.sceneEditor?.discard();
    load(select.value).catch(error => { info.textContent = error.message; });
  });
  // what the editor (editor.js) reads and calls
  window.sceneViewer = {
    get map() { return map; },
    get hovered() { return hovered; },
    get rules() { return rules; },
    get graphics() { return graphics; },
    get turn() { return turn; },
    spriteMeta, images,
    surface, preview, render, reload, fetchJSON,
    status(text) { info.textContent = text; },
    // the canvas point of the centre of small tile (U, V), on the top of a raised tile
    tilePoint(U, V) { const [x, y] = propPoint((U - .5) / 2, (V - .5) / 2); return [x, y - lift(U, V)]; },
    // the window point over the centre of small tile (U, V) on its ground, for driving the page from a script
    clientPoint(U, V) {
      const [x, y] = propPoint((U - .5) / 2, (V - .5) / 2), r = canvas.getBoundingClientRect();
      return [r.left + x * r.width / canvas.width, r.top + (y - lift(U, V)) * r.height / canvas.height];
    },
  };
  (async () => {
    const catalog = await fetchJSON('/api/catalog');
    graphics = catalog.graphics; rules = catalog.rules; scenery = catalog.scenery;
    select.replaceChildren(...catalog.scenes.map(scene => new Option(`${scene.name} · ${scene.id}${scene.format === 2 ? ' · 형식 2' : ''}`, scene.id)));
    const sources = { tiles: graphics.tileset.image, props: graphics.props.image, extra: graphics.extra.image, scenery: scenery.image, structures: graphics.structures.image };
    for (const [name, a] of Object.entries(graphics.anims)) sources[`anim:${name}`] = a.image;
    await Promise.all(Object.entries(sources).map(([key, path]) => new Promise((resolve, reject) => {
      const img = new Image(); img.onload = () => { images[key] = img; resolve(); };
      img.onerror = () => reject(new Error(`${path} 그림을 읽지 못했습니다.`));
      img.src = `/assets/graphics/${path}`;
    })));
    const requested = params.get('map');
    select.value = catalog.scenes.some(scene => scene.id === requested) ? requested : 'S_checkpoint';
    if (!select.value) select.selectedIndex = 0;
    await load(select.value);
  })().catch(error => { info.textContent = error.message; });
})();
