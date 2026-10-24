// The old scene maps' drawing, kept as it was (with its exceptions) to draw the maps not yet in format 2 and, for the
// maps that are, the same map at the comparison revision. Remove with the last old map (plan 2026_10_24, step 4).
window.legacyScene = (() => {
  const faceEdges = { SW: [-8, 4], SE: [8, 4], NW: [-8, -4], NE: [8, -4] };

  // normalize turns the checkpoint's small-grid entries into props, as the old viewer did on loading
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

  // render draws map on env.ctx: env = { ctx, map, graphics, images, turn, left, top, step, decor }. It returns the
  // drawn pictures (hits) and the objects by small tile (index) for the hover.
  function render(env) {
    const { ctx, map, graphics, images, turn, left, top, step, decor } = env;
    const hits = [], index = new Map();
    function rotate(u, v) {
      if (turn === 1) return [v, -u];
      if (turn === 2) return [-u, -v];
      if (turn === 3) return [-v, u];
      return [u, v];
    }
    function rotateMask(mask) {
      let result = 0;
      for (let bit = 0; bit < 4; bit++) if (mask & (1 << bit)) result |= 1 << ((bit + 4 - turn) % 4);
      return result;
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
      hits.push({ name, actual, U: U ?? Math.floor(2 * u + .5), V: V ?? Math.floor(2 * v + .5), x: px, y: py, w, h, meta, img });
    }
    function rim(U, V, mask) {
      const [x, y] = propPoint((U - .5) / 2, (V - .5) / 2);
      ctx.beginPath();
      if (mask & 1) { ctx.moveTo(x, y - 154); ctx.lineTo(x - 16, y - 146); }
      if (mask & 2) { ctx.moveTo(x, y - 154); ctx.lineTo(x + 16, y - 146); }
      ctx.strokeStyle = '#4a3a25'; ctx.lineWidth = 1; ctx.stroke();
    }
    function addProp(entry, items) {
      eachProp(entry, (name, u, v, U, V, side) => {
        if (hidden(name)) return;
        const smallU = U ?? Math.floor(2 * u + .5), smallV = V ?? Math.floor(2 * v + .5);
        const key = `${smallU},${smallV}`;
        if (!index.has(key)) index.set(key, []);
        index.get(key).push({ name, side, ...pivotAt(name, u, v) });
        items.push({ y: propPoint(u, v)[1], draw: () => drawProp(name, u, v, smallU, smallV) });
      });
    }
    function pivotAt(name, u, v) {
      const found = sprite(name);
      if (!found) return {};
      const [x, y] = propPoint(u, v);
      return { x: Math.round(x - found[1].pivot[0]), y: Math.round(y - found[1].pivot[1]), meta: found[1], img: found[2], actual: found[0] };
    }
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
      if (decor) for (const [x, y, name] of map.decorations || []) {
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
    return { hits, index };
  }

  return { normalize, render };
})();
