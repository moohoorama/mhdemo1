// Runs the scenario scenes of assets_candidate/tools/viewer_template.html without a browser and writes what each
// scene puts where: node ver5/tools/scene_maps/export.mjs > scenes_draws.json (build.py reads it).
// The scenes are JavaScript that composes sprites; here every sprite lookup remembers its name and `put` records
// the call, so the output is the list of (sprite, x, y) of each scene in each state, in the viewer's map pixels.
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');
const cand = path.join(root, 'assets_candidate');
const gfx = JSON.parse(fs.readFileSync(path.join(root, 'ver5/assets/graphics/index.json'), 'utf8'));
// a picture the viewer names but the candidate set no longer has (the old whole-gate halves, now cut into slices)
// still gets recorded by name; build.py turns it into what the game draws
const tiles = new Proxy(JSON.parse(fs.readFileSync(path.join(cand, 'tiles/tiles.json'), 'utf8')).sprites,
  { get: (t, n) => t[n] ?? (String(n).startsWith('sgate_') ? { size: [0, 0], pivot: [0, 0] } : undefined) });
const fx = JSON.parse(fs.readFileSync(path.join(cand, 'effects/effects.json'), 'utf8'));

const DATA = {
  imgs: {}, tileset: gfx.tileset.sprites, water: gfx.tileset.animations.water,
  anims: Object.fromEntries(Object.entries(gfx.tileset.animations).filter(([k]) => /^(grass|tree|rock)/.test(k))),
  structures: gfx.structures, props: gfx.props.sprites, cand: tiles, units: {}, fx, sites: gfx.sites,
};
// which tileset animation a sprite id is the first frame of
const animOf = {};
for (const [k, v] of Object.entries(DATA.anims)) for (const f of v.frames) animOf[f] = k;

let html = fs.readFileSync(path.join(cand, 'tools/viewer_template.html'), 'utf8');
let code = html.slice(html.indexOf('<script>') + 8, html.indexOf('// ---- player ----'));
code = code.replace('/*DATA*/', '');
const rec = [];
code = code
  .replace('const loaded=Promise.all', 'const loaded=0&&Promise.all')
  // the game's scene has no canvas edge: draw further than the viewer's picture does
  .replace('X>-pad&&X<W+pad&&Y>-pad&&Y<H+pad', 'X>-pad-MORE&&X<W+pad+MORE&&Y>-pad-MORE&&Y<H+pad+MORE')
  .replace(/return\{img:IM\.tileset,/, "return{name:'ts:'+i,img:null,")
  .replace(/return\{img:IM\.structures,/, "return{name:'st:'+n,img:null,")
  .replace(/return\{img:IM\.props,/, "return{name:'pr:'+n,img:null,")
  .replace(/return\{img:IM\.sites,/, "return{name:'site:'+n,img:null,")
  .replace(/return\{img:IM\['c:'\+n\],/, "return{name:'c:'+n,img:null,")
  .replace(/function put\(c,s,x,y,k=1\)\{[^\n]*\}/, 'function put(c,s,x,y,k=1){REC.push({n:s.name,x:+x.toFixed(2),y:+y.toFixed(2),k});}');
code = "const document={querySelector:()=>null};const MORE=120;\n" + code + "\nreturn {SCENES, ground, addBorder, visTile, chAt, EXT};";

const { SCENES: scenes, ground, addBorder, visTile, chAt, EXT } = new Function('DATA', 'REC', 'Image', code)(DATA, rec, class {});
const ctx = new Proxy({}, { get: () => () => {}, set: () => true }); // every canvas call does nothing
const out = {};
for (const [key, sc] of Object.entries(scenes)) {
  const gen = sc.tiles.gen;
  const tilesAt = (u, v) => (u >= 0 && v >= 0 && u < sc.w && v < sc.h) ? sc.tiles[v][u] : (gen ? gen(u, v) : '.');
  const states = [];
  for (let s = 0; s < (sc.states || ['']).length; s++) {
    const m = sc.build.call(sc, s);
    addBorder(m, sc);
    const scene = m.tiles ? { ...sc, tiles: m.tiles } : sc;
    const grass = [];
    if (!sc.noGround) {
      rec.length = 0;
      ground(ctx, scene, 0);
      for (const r of rec) if (animOf[+r.n.slice(3)]?.startsWith('grass')) grass.push({ n: 'ts:' + animOf[+r.n.slice(3)], x: r.x, y: r.y });
    }
    const under = [];
    for (const f of m.under || []) { rec.length = 0; f(ctx); under.push(...rec.map(r => ({ ...r, layer: 'under' }))); }
    const items = [...m.items].sort((a, b) => a.y - b.y);
    const list = [];
    for (const it of items) { rec.length = 0; try { it.draw(ctx); } catch (e) { /* a unit ghost: the viewer's units are not exported */ } list.push(...rec.map(r => ({ ...r, layer: 'item' }))); }
    const visible = [];
    for (let v = -EXT; v < sc.h + EXT; v++) for (let u = -EXT; u < sc.w + EXT; u++) if (visTile(u, v, sc)) visible.push([u, v, (chAt(sc, u, v) || { ch: '.' }).ch]);
    states.push({
      name: sc.states ? sc.states[s] : '', night: !!m.night, sky: m.sky || null, tint: m.tint || null, grass, visible,
      draws: [...under, ...list].map(d => (d.n.startsWith('ts:') ? { ...d, n: 'ts:' + (animOf[+d.n.slice(3)] || d.n.slice(3)) } : d)),
      units: (m.units || []).map(u => ({ id: u.id, u: u.u, v: u.v, face: u.face })),
    });
  }
  out[key] = { name: sc.name, w: sc.w, h: sc.h, noGround: !!(sc.noGround), border: sc.border || 0,
    tiles: Array.from({ length: sc.h }, (_, v) => Array.from({ length: sc.w }, (_, u) => tilesAt(u, v)).join('')), states };
}
process.stdout.write(JSON.stringify(out));
