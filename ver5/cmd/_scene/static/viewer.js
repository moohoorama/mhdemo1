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
  const faceEdges = { SW: [-8, 4], SE: [8, 4], NW: [-8, -4], NE: [8, -4] };
  let graphics, map, images = {}, left, top, right, bottom;
  let turn = Math.max(0, Math.min(3, Number(new URLSearchParams(location.search).get('turn')) || 0));
  let zoom = 1, step = 2, pointer = null, hovered = null;
  let objectIndex = new Map(), surfaceHits = [], animating = false;
  const whiteMasks = new Map(), alphaMasks = new Map();

  function rotate(u, v) {
    if (turn === 1) return [v, -u];
    if (turn === 2) return [-u, -v];
    if (turn === 3) return [-v, u];
    return [u, v];
  }
  function unrotate(u, v) {
    if (turn === 1) return [-v, u];
    if (turn === 2) return [-u, -v];
    if (turn === 3) return [v, -u];
    return [u, v];
  }
  function rotateMask(mask) {
    let result = 0;
    for (let bit = 0; bit < 4; bit++) if (mask & (1 << bit)) result |= 1 << ((bit + 4 - turn) % 4);
    return result;
  }
  function turnSide(side) {
    const sides = ['NW', 'NE', 'SE', 'SW'];
    return sides[(sides.indexOf(side) + 4 - turn) % 4];
  }
  function turnName(name) {
    const cliff = /^(tcliff|lcliff)_(\d\d)_(\d)$/.exec(name);
    if (cliff) {
      const visible = rotateMask(Number(cliff[2])) & 12;
      return visible ? `${cliff[1]}_${String(visible).padStart(2, '0')}_${cliff[3]}` : null;
    }
    const foot = /^cfoot_(sw|se)$/.exec(name);
    if (foot) {
      const visible = rotateMask(foot[1] === 'sw' ? 8 : 4) & 12;
      return visible === 8 ? 'cfoot_sw' : visible === 4 ? 'cfoot_se' : null;
    }
    if (!turn) return name;
    const wall = /^(swall)_(u|v)(_up)?$/.exec(name);
    if (wall) return `${wall[1]}_${turn % 2 ? (wall[2] === 'u' ? 'v' : 'u') : wall[2]}${wall[3] || ''}`;
    const gate = /^(sgate)_(open|closed)_(u|v)_(\d+)$/.exec(name);
    if (gate) {
      const reverse = gate[3] === 'v' ? turn >= 2 : turn === 1 || turn === 2;
      const slice = Number(gate[4]);
      return `${gate[1]}_${gate[2]}_${turn % 2 ? (gate[3] === 'u' ? 'v' : 'u') : gate[3]}_${reverse && slice <= 10 ? 10 - slice : slice}`;
    }
    return name;
  }
  function hidden(name) {
    return (map.highland && turn >= 2 && (/^sgate_(open|closed)_v_11$/.test(name) || name.startsWith('jshade_')))
      || !turnName(name);
  }
  function groundPoint(x, y) {
    const centerY = y + 2;
    const [u, v] = rotate(x / 32 + centerY / 16 - .5, centerY / 16 - x / 32 - .5);
    return [16 * (u - v), 8 * (u + v + 1) - 2];
  }
  function tilePoint(x, y) {
    const [px, py] = groundPoint(x, y);
    return [(px - left) * 2, (py - top) * 2];
  }
  function propPoint(u, v) {
    [u, v] = rotate(u, v);
    return [((u - v) * 16 - left) * 2, ((u + v + 1) * 8 - top) * 2];
  }
  function measure() {
    const [x0, y0, x1, y1] = map.bounds;
    const corners = [[x0, y0], [x0, y1], [x1, y0], [x1, y1]].map(([x, y]) => groundPoint(x, y));
    left = Math.floor(Math.min(...corners.map(p => p[0])) - 100);
    right = Math.ceil(Math.max(...corners.map(p => p[0])) + 100);
    top = Math.floor(Math.min(...corners.map(p => p[1])) - (map.highland ? 190 : 110));
    bottom = Math.ceil(Math.max(...corners.map(p => p[1])) + 100);
    canvas.width = hoverCanvas.width = (right - left) * 2;
    canvas.height = hoverCanvas.height = (bottom - top) * 2;
    ctx.imageSmoothingEnabled = hoverCtx.imageSmoothingEnabled = false;
  }
  function sprite(name) {
    const actual = turnName(name);
    if (!actual) return null;
    const meta = graphics.props.sprites[actual];
    if (meta) return [actual, meta, images.props];
    const extra = graphics.extra.sprites[actual];
    return extra ? [actual, extra, images.extra] : null;
  }
  function highland(U, V) {
    if (!map.highland || !map.visual) return false;
    return map.highland[V - map.visual[1]]?.[U - map.visual[0]] === '1';
  }
  function drawTile(id, x, y, lift = 0) {
    if (turn && id >= 9 && id <= 22) id = 8 + rotateMask(id - 8);
    else if (turn && id >= 24 && id <= 37) id = 23 + rotateMask(id - 23);
    const r = graphics.tileset.sprites[id];
    if (!r) return;
    const [px, py] = tilePoint(x, y);
    const scale = map.native ? 1 : 2;
    ctx.drawImage(images.tiles, r[0], r[1], r[2], r[3],
      Math.round(px - r[4] * scale), Math.round(py - r[5] * scale - lift), r[2] * scale, r[3] * scale);
  }
  function eachProp(entry, fn) {
    if (entry.length < 5 || entry.smallU !== undefined) { fn(entry[0], entry[1], entry[2], entry.smallU, entry.smallV, entry.side); return; }
    for (let v = entry[2]; v <= entry[4]; v++) for (let u = entry[1]; u <= entry[3]; u++) fn(entry[0], u, v);
  }
  function drawProp(name, u, v, U, V) {
    if (hidden(name)) return;
    const found = sprite(name);
    if (!found) return;
    const [actual, meta, img] = found;
    const [sx, sy, w, h] = meta.rect;
    const [x, y] = propPoint(u, v);
    const px = Math.round(x - meta.pivot[0]), py = Math.round(y - meta.pivot[1]);
    ctx.drawImage(img, sx, sy, w, h, px, py, w, h);
    surfaceHits.push({ name, actual, U: U ?? Math.floor(2 * u + .5), V: V ?? Math.floor(2 * v + .5), x: px, y: py, w, h });
  }
  function rim(U, V, mask) {
    const [x, y] = propPoint((U - .5) / 2, (V - .5) / 2);
    ctx.beginPath();
    if (mask & 1) { ctx.moveTo(x, y - 154); ctx.lineTo(x - 16, y - 146); }
    if (mask & 2) { ctx.moveTo(x, y - 154); ctx.lineTo(x + 16, y - 146); }
    ctx.strokeStyle = '#4a3a25'; ctx.lineWidth = 1; ctx.stroke();
  }
  function render() {
    if (!map) return;
    ctx.fillStyle = '#675d42'; ctx.fillRect(0, 0, canvas.width, canvas.height);
    surfaceHits = []; objectIndex = new Map();
    const items = map.ground.map(([x, y, layers]) => {
      const U = Math.floor(x / 16 + y / 8 + 1e-6), V = Math.floor(y / 8 - x / 16 + 1e-6);
      return { y: tilePoint(x, y)[1], draw: () => {
        if (highland(U, V)) drawTile(8, x, y, 146);
        else for (const id of layers) drawTile(id === 0 ? graphics.tileset.animations.water.frames[0] : id, x, y);
      }};
    });
    if (map.highland) for (let v = 0; v < map.visual[3]; v++) for (let u = 0; u < map.visual[2]; u++) {
      const U = map.visual[0] + u, V = map.visual[1] + v;
      if (!highland(U, V)) continue;
      const exposed = (!highland(U - 1, V) ? 1 : 0) | (!highland(U, V - 1) ? 2 : 0)
        | (!highland(U + 1, V) ? 4 : 0) | (!highland(U, V + 1) ? 8 : 0);
      const mask = rotateMask(exposed) & 3;
      if (mask) items.push({ y: propPoint((U - .5) / 2, (V - .5) / 2)[1], draw: () => rim(U, V, mask) });
    }
    if (step >= 1) {
      for (const entry of map.props) {
        const found = sprite(entry[0]);
        if (found?.[1].layer === 'ground') addProp(entry, items);
      }
      for (const [x, y, name] of map.decorations || []) {
        const id = graphics.tileset.animations[name]?.frames[0];
        if (id !== undefined) items.push({ y: tilePoint(x, y)[1], draw: () => drawTile(id, x, y) });
      }
    }
    if (step >= 2) for (const entry of map.props) {
      const found = sprite(entry[0]);
      if (found && found[1].layer !== 'ground') addProp(entry, items);
    }
    items.sort((a, b) => a.y - b.y);
    for (const item of items) item.draw();
    for (const button of document.querySelectorAll('[data-step]')) button.setAttribute('aria-pressed', String(Number(button.dataset.step) === step));
    for (const button of document.querySelectorAll('[data-turn]')) button.setAttribute('aria-pressed', String(Number(button.dataset.turn) === turn));
    info.textContent = `${map.id} · ${map.size.join('×')} 논리 칸 · 바닥 ${map.ground.length.toLocaleString()} · 장식 ${(map.decorations || []).length.toLocaleString()} · 소품 ${map.props.length.toLocaleString()}`;
    if (pointer) updateHover(pointer);
  }
  function addProp(entry, items) {
    eachProp(entry, (name, u, v, U, V, side) => {
      if (hidden(name)) return;
      const smallU = U ?? Math.floor(2 * u + .5), smallV = V ?? Math.floor(2 * v + .5);
      const key = `${smallU},${smallV}`;
      if (!objectIndex.has(key)) objectIndex.set(key, []);
      objectIndex.get(key).push({ name, u, v, U: smallU, V: smallV, side });
      items.push({ y: propPoint(u, v)[1], draw: () => drawProp(name, u, v, smallU, smallV) });
    });
  }
  function alphaMask(actual, meta, img) {
    if (alphaMasks.has(actual)) return alphaMasks.get(actual);
    const [sx, sy, w, h] = meta.rect;
    const temp = document.createElement('canvas'); temp.width = w; temp.height = h;
    const c = temp.getContext('2d'); c.drawImage(img, sx, sy, w, h, 0, 0, w, h);
    const alpha = c.getImageData(0, 0, w, h).data;
    alphaMasks.set(actual, alpha);
    return alpha;
  }
  function hitObject(x, y) {
    for (let i = surfaceHits.length - 1; i >= 0; i--) {
      const item = surfaceHits[i];
      const px = Math.floor(x - item.x), py = Math.floor(y - item.y);
      if (px < 0 || py < 0 || px >= item.w || py >= item.h) continue;
      const found = sprite(item.name);
      if (found && alphaMask(item.actual, found[1], found[2])[(py * item.w + px) * 4 + 3]) return item;
    }
    return null;
  }
  function whiteMask(name) {
    const found = sprite(name);
    if (!found) return null;
    const [actual, meta, img] = found;
    if (whiteMasks.has(actual)) return whiteMasks.get(actual);
    const [sx, sy, w, h] = meta.rect;
    const temp = document.createElement('canvas'); temp.width = w; temp.height = h;
    const c = temp.getContext('2d'); c.drawImage(img, sx, sy, w, h, 0, 0, w, h);
    c.globalCompositeOperation = 'source-in'; c.fillStyle = '#fff'; c.fillRect(0, 0, w, h);
    whiteMasks.set(actual, temp);
    return temp;
  }
  function pulse(now) {
    hoverCtx.clearRect(0, 0, hoverCanvas.width, hoverCanvas.height);
    if (!hovered) { animating = false; return; }
    const { U, V, hit } = hovered;
    const alpha = .75 * (1 + Math.sin(2 * Math.PI * now / 2000)) / 2;
    const [x, y] = propPoint((U - .5) / 2, (V - .5) / 2);
    const lift = highland(U, V) ? 146 : 0;
    hoverCtx.globalAlpha = alpha * .65; hoverCtx.fillStyle = '#fff'; hoverCtx.beginPath();
    hoverCtx.moveTo(x, y - 8 - lift); hoverCtx.lineTo(x + 16, y - lift);
    hoverCtx.lineTo(x, y + 8 - lift); hoverCtx.lineTo(x - 16, y - lift); hoverCtx.closePath(); hoverCtx.fill();
    hoverCtx.globalAlpha = alpha;
    for (const item of objectIndex.get(`${U},${V}`) || []) {
      const found = sprite(item.name), mask = whiteMask(item.name);
      if (!found || !mask) continue;
      const [px, py] = propPoint(item.u, item.v);
      hoverCtx.drawImage(mask, Math.round(px - found[1].pivot[0]), Math.round(py - found[1].pivot[1]));
    }
    if (hit && (hit.U !== U || hit.V !== V)) {
      const mask = whiteMask(hit.name);
      if (mask) hoverCtx.drawImage(mask, hit.x, hit.y);
    }
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
    const [u, v] = unrotate((a + b - 1) / 2, (b - a - 1) / 2);
    const U = Math.floor(2 * u + .5), V = Math.floor(2 * v + .5);
    const hit = hitObject(x, y);
    hovered = { U, V, hit };
    const names = [...new Set((objectIndex.get(`${U},${V}`) || []).map(item => item.side ? `${item.name}(${turnSide(item.side)})` : item.name))];
    cursor.textContent = `바닥 기준 작은 칸 (${U}, ${V}) · 논리 (${u.toFixed(2)}, ${v.toFixed(2)})${hit ? ` · 그림 ${hit.name} (${hit.U}, ${hit.V})` : ''}${names.length ? ` · ${names.join(', ')}` : ''}`;
    if (!animating) { animating = true; requestAnimationFrame(pulse); }
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
  function normalize(raw) {
    const entries = [];
    for (const [name, U, V, dx = 0, dy = 0] of raw.small_props || []) {
      const snap = !name.startsWith('tcliff_') && !name.startsWith('lcliff_');
      const item = [name, (U - .5) / 2 + (snap ? 0 : dx / 64 + dy / 32),
        (V - .5) / 2 + (snap ? 0 : -dx / 64 + dy / 32)];
      item.smallU = U; item.smallV = V; entries.push(item);
    }
    for (const [name, U, V, side, dx = 0, dy = 0] of raw.small_faces || []) {
      const [ex, ey] = faceEdges[side];
      const item = [name, (U - .5) / 2 + (ex + dx) / 64 + (ey + dy) / 32,
        (V - .5) / 2 - (ex + dx) / 64 + (ey + dy) / 32];
      item.smallU = U; item.smallV = V; item.side = side; entries.push(item);
    }
    raw.props = [...(raw.props || []), ...entries];
    return raw;
  }
  async function loadScene(id) {
    const response = await fetch(`/api/map/${encodeURIComponent(id)}`);
    if (!response.ok) throw new Error(`${id} 맵을 읽지 못했습니다.`);
    map = normalize(await response.json());
    whiteMasks.clear(); alphaMasks.clear(); hovered = pointer = null;
    cursor.textContent = '커서 좌표: —';
    measure(); setZoom(zoom); render(); center();
    history.replaceState(null, '', `?map=${encodeURIComponent(id)}&turn=${turn}`);
  }
  for (const button of document.querySelectorAll('[data-step]')) button.addEventListener('click', () => { step = Number(button.dataset.step); render(); });
  for (const button of document.querySelectorAll('[data-zoom]')) button.addEventListener('click', () => setZoom(Number(button.dataset.zoom)));
  for (const button of document.querySelectorAll('[data-turn]')) button.addEventListener('click', () => {
    turn = Number(button.dataset.turn); hovered = pointer = null; whiteMasks.clear(); alphaMasks.clear();
    measure(); setZoom(zoom); render(); center();
    history.replaceState(null, '', `?map=${encodeURIComponent(map.id)}&turn=${turn}`);
  });
  surface.addEventListener('mousemove', event => { pointer = { clientX: event.clientX, clientY: event.clientY }; updateHover(pointer); });
  surface.addEventListener('mouseleave', () => { hovered = pointer = null; cursor.textContent = '커서 좌표: —'; });
  select.addEventListener('change', () => loadScene(select.value).catch(error => { info.textContent = error.message; }));
  (async () => {
    const response = await fetch('/api/catalog');
    if (!response.ok) throw new Error('장면 목록을 읽지 못했습니다.');
    const catalog = await response.json();
    graphics = catalog.graphics;
    select.replaceChildren(...catalog.scenes.map(scene => new Option(`${scene.name} · ${scene.id}`, scene.id)));
    const sources = { tiles: graphics.tileset.image, props: graphics.props.image, extra: graphics.extra.image };
    await Promise.all(Object.entries(sources).map(([key, path]) => new Promise((resolve, reject) => {
      const img = new Image(); img.onload = () => { images[key] = img; resolve(); };
      img.onerror = () => reject(new Error(`${path} 그림을 읽지 못했습니다.`));
      img.src = `/assets/graphics/${path}`;
    })));
    const requested = new URLSearchParams(location.search).get('map');
    select.value = catalog.scenes.some(scene => scene.id === requested) ? requested : 'S_checkpoint';
    if (!select.value) select.selectedIndex = 0;
    await loadScene(select.value);
  })().catch(error => { info.textContent = error.message; });
})();
