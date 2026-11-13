"""Bakes each scene map's diorama board edge: the cross-section of the ground slab under the board's two front edges
(+u, the screen's lower right, and +v, its lower left) and the cut faces of the city walls that run into them, as
pixel art on the 2:1 small-tile grid (the pixel iso kit's projection: x = 16(u - v), y = 8(u + v) - z, patterns laid
in face coordinates s across, h up/down). The game only draws the pictures (docs/presentation-scene.md, 판 단면).

The board is the ground the map draws: a format 2 map's grid (origin, ground), a format 1 map's visual rectangle.
Per map it writes assets/graphics/maps/board/<ID>.png (an atlas of pieces) and <ID>.json:

  {"map", "source": sha1 of the map file, "board": [U0, V0, U1, V1] small tiles, "image",
   "section": [[x, y, w, h, ax, ay], ...]   atlas rect, top left on the board canvas (ground point (cu, cv) at
                                             (16(cu - cv), 8(cu + cv)), the canvas of the map's origin)
   "cuts": [{"rect", "at", "tile": [U, V], "kind": "wall" | "rampart"}]}   drawn with the wall piece on that tile

  python3 tools/scene_maps/diorama.py --all          # every scene map in index.json
  python3 tools/scene_maps/diorama.py S_zhuo S_x     # these maps
  python3 tools/scene_maps/diorama.py --check        # list maps whose board is missing or older than the map

`go run ./cmd/_walkmap -fmt <ID>` (which the map build scripts run after writing a map) runs it for that map.
"""
import hashlib
import json
import sys
from pathlib import Path

import numpy as np
from PIL import Image

VER5 = Path(__file__).resolve().parents[2]
GFX = VER5 / 'assets/graphics'
OUT = GFX / 'maps/board'
sys.path.insert(0, str(VER5.parent / 'assets_candidate/tools'))
import gen_pixel_castle as pc  # noqa: E402

DEPTH = 56          # the earth slab under an outdoor board, pixels
DEPTH_HALL = 40     # the stone plinth under a floored room
CHUNK = 8           # small tiles of edge per section piece
INK = pc.INK

# (lit +v face, shaded +u face)
P = {
    'grass': ((70, 90, 48), (54, 68, 37)),
    'top': ((95, 66, 49), (66, 46, 34)),
    'soil': ((128, 95, 60), (98, 73, 46)),
    'soil_d': ((112, 82, 52), (86, 63, 40)),
    'speck': ((166, 138, 98), (128, 106, 76)),
    'clay': ((104, 76, 50), (80, 59, 38)),
    'pebble': ((150, 140, 124), (112, 104, 92)),
    'pebble_d': ((92, 84, 74), (70, 64, 57)),
    'rock': ((118, 111, 105), (86, 81, 77)),
    'rock_l': ((134, 127, 120), (98, 93, 88)),
    'seam': ((90, 85, 80), (64, 60, 57)),
    'crust': ((70, 64, 60), (52, 48, 45)),
    'road': ((150, 116, 76), (116, 90, 58)),
    'water': ((70, 112, 146), (52, 86, 114)),
    'water_l': ((120, 160, 186), (90, 124, 148)),
    'mud': ((84, 74, 60), (64, 56, 46)),
    'canvas': ((190, 172, 132), (146, 132, 100)),
    'ashlar': ((178, 170, 154), (132, 126, 114)),
    'ashlar_j': ((122, 114, 102), (90, 84, 75)),
    'ashlar_f': ((146, 138, 124), (108, 102, 92)),
}
PLINTH_FLOORS = {'W', 'P', 'C', 'M'}
FLOOR_PROPS = {'floor_wood': 'W', 'floor_palace': 'P', 'floor_carpet': 'C', 'floor_mat': 'M', 'floor_tent': 'T'}


def hsh(*a):
    """a stable small hash of integers (numpy arrays welcome)"""
    h = np.uint32(2166136261)
    for x in a:
        h = (np.asarray(x, np.int64).astype(np.uint32) ^ h) * np.uint32(16777619)
        h ^= h >> np.uint32(13)
    return h


def rgb(name, lit):
    return P[name][0 if lit else 1]


class MapInfo:
    """what the bake needs of a map, both formats"""

    def __init__(self, path):
        self.path = path
        raw = path.read_bytes()
        self.source = hashlib.sha1(raw).hexdigest()
        d = json.loads(raw)
        self.id = d['id']
        self.fmt = d.get('format', 1)
        rules = json.load(open(GFX / 'scene.json'))
        self.rules = rules
        self.floors = {}
        self.cuts = []  # (kind, axis, U, V, extra)
        if self.fmt == 2:
            self.U0, self.V0 = d['origin']
            self.ground = d['ground']
            self.height = d.get('height') or []
            self.U1, self.V1 = self.U0 + len(self.ground[0]), self.V0 + len(self.ground)
            self.wall_cuts_fmt2(d)
            self.object_wall_cuts(d)
        else:
            if 'visual' not in d or not d.get('native'):
                raise SkipMap('format 1 map without a visual rectangle (an old non-native map)')
            self.U0, self.V0, w, h = d['visual']
            self.U1, self.V1 = self.U0 + w, self.V0 + h
            self.ground = d['small_tiles']
            self.height = []
            props = d.get('props') or []
            for p in props:
                if p[0] in FLOOR_PROPS and len(p) >= 3:
                    us = [p[1], p[3]] if len(p) >= 5 else [p[1], p[1]]
                    vs = [p[2], p[4]] if len(p) >= 5 else [p[2], p[2]]
                    for gu in range(int(us[0]), int(us[1]) + 1):
                        for gv in range(int(vs[0]), int(vs[1]) + 1):
                            for i in (0, 1):
                                for j in (0, 1):
                                    self.floors[(2 * gu + i, 2 * gv + j)] = FLOOR_PROPS[p[0]]
            self.wall_cuts_fmt1(props)
        self.rampart_cuts()

    def glyph(self, u, v):
        x = min(max(u - self.U0, 0), self.U1 - self.U0 - 1)
        y = min(max(v - self.V0, 0), self.V1 - self.V0 - 1)
        return self.floors.get((u, v), self.ground[y][x])

    def step(self, u, v):
        if not self.height or not (self.U0 <= u < self.U1 and self.V0 <= v < self.V1):
            return 0
        return int(self.height[v - self.V0][u - self.U0])

    def material(self, u, v):
        g = self.glyph(u, v)
        if g in PLINTH_FLOORS:
            return 'plinth'
        if g == 'T':
            return 'tent'
        if g in self.rules['flood']['glyphs'] or self.rules['surfaces'].get(g, 1) == 0:
            return 'water'
        if self.rules['surfaces'].get(g, 1) == 2:
            return 'road'
        return 'grass'

    def wall_cuts_fmt2(self, d):
        fams = self.rules['walls']
        for fam, a, b, *tint in d.get('walls') or []:   # a row may end in a tint ({"body": ...})
            if fam not in fams:
                continue
            if fams[fam].get('piece') in KITS:   # a piece family drawn by its kit: cut through the kit's own bodies
                extra = {'kit': fams[fam]['piece'], 'tint': tint[0] if tint else {}}
            else:
                block = fams[fam].get('block') or fams[fam]['piece'] + '_05'
                body = (tint[0] if tint else {}).get('body')
                extra = (block, body) if body else block
            if a[1] == b[1]:   # along u
                v = a[1]
                if self.V0 <= v < self.V1 and max(a[0], b[0]) >= self.U1 - 1 and min(a[0], b[0]) <= self.U1 - 1:
                    self.cuts.append(('wall', 'u', self.U1 - 1, v, extra))
            else:
                u = a[0]
                if self.U0 <= u < self.U1 and max(a[1], b[1]) >= self.V1 - 1 and min(a[1], b[1]) <= self.V1 - 1:
                    self.cuts.append(('wall', 'v', u, self.V1 - 1, extra))

    def object_wall_cuts(self, d):
        """walls laid as objects, one per small tile (허도's pwall_u run, cwall_<mask> pieces): a tile of the run on
        a front edge is cut there (the plan lays the straight piece there, scenemap runsOn)"""
        for ob in d.get('objects') or []:
            name, U, V = ob[0], ob[1], ob[2]
            tint = ob[-1] if isinstance(ob[-1], dict) else {}
            base = name
            for end in ('_end_m', '_end_p', '_gate_m', '_gate_p'):
                base = base.removesuffix(end)
            if base in OBJECT_KITS and U == self.U1 - 1 and self.V0 <= V < self.V1:
                self.cuts.append(('wall', 'u', U, V, {'kit': OBJECT_KITS[base], 'tint': tint}))
            if name.startswith('cwall_') and name[6:].isdigit():
                mask = int(name[6:])
                if U == self.U1 - 1 and mask & 1:
                    self.cuts.append(('wall', 'u', U, V, {'kit': 'cwall', 'tint': tint}))
                if V == self.V1 - 1 and mask & 2:
                    self.cuts.append(('wall', 'v', U, V, {'kit': 'cwall', 'tint': tint}))

    def wall_cuts_fmt1(self, props):
        """the scene castle's thin wall columns (swall_u, swall_v): a piece covers the small tile round its anchor
        (2g + 1); the last one left on the board touching a front edge is cut there"""
        for p in props:
            if p[0] not in ('swall_u', 'swall_v') or len(p) != 3:
                continue
            # a column covers [2g + .5, 2g + 1.5] along its run: the last one on the board ends on the edge, or one
            # straddles it (swall_v's columns sit half a tile off the small tiles): the game leaves that one out and
            # its picture's part on the board goes into the cut
            U, V = int(np.floor(2 * p[1] + .5 + 1e-6)), int(np.floor(2 * p[2] + .5 + 1e-6))
            along, edge, across, lo, hi = (('u', self.U1, V, self.V0, self.V1) if p[0] == 'swall_u'
                                           else ('v', self.V1, U, self.U0, self.U1))
            start = 2 * (p[1] if along == 'u' else p[2]) + .5
            if not lo <= across < hi:
                continue
            tile = (edge - 1, V) if along == 'u' else (U, edge - 1)
            if abs(start + 1 - edge) < 1e-6:
                self.cuts.append(('wall', along, *tile, p[0]))
            elif start < edge < start + 1:
                self.cuts.append(('wall', along, *tile, {'column': p[0], 'at': (p[1], p[2])}))

    def rampart_cuts(self):
        """a city wall top (a height with a rim) running into a front edge: cut there, one piece per edge tile"""
        heights = self.rules['heights']
        for u in range(self.U0, self.U1):
            s = self.step(u, self.V1 - 1)
            if s and heights[str(s)].get('rim'):
                self.cuts.append(('rampart', 'v', u, self.V1 - 1, s))
        for v in range(self.V0, self.V1):
            s = self.step(self.U1 - 1, v)
            if s and heights[str(s)].get('rim'):
                self.cuts.append(('rampart', 'u', self.U1 - 1, v, s))

    def depth(self):
        mats = [self.material(u, self.V1 - 1) for u in range(self.U0, self.U1)]
        mats += [self.material(self.U1 - 1, v) for v in range(self.V0, self.V1)]
        return DEPTH_HALL if mats.count('plinth') * 2 > len(mats) else DEPTH


class SkipMap(Exception):
    pass


# ---- the earth section ---------------------------------------------------------------------------------------------

def strata(mat, s, d, lit, depth):
    """the section's colour at dots s along the edge, d pixels below the ground, as an (n, 3) array"""
    n = s.shape[0]
    out = np.zeros((n, 3), np.uint8)
    di = np.floor(d).astype(int)
    hs = hsh(s // 2, di)
    col = lambda name: np.array(rgb(name, lit), np.uint8)
    if mat == 'plinth':
        floor = np.array(FLOOR_EDGE.get('W'), np.uint8)
        out[:] = col('ashlar')
        # dressed stone courses of 7 rows and a joint, blocks 14 dots with a joint dot, every other course shifted
        c = np.maximum(di - 4, 0)
        course = c // 8
        joint = (c % 8 == 7) | ((s + course * 7) % 14 == 13)
        out[joint] = col('ashlar_j')
        out[(c // 8) == (depth - 4) // 8] = col('ashlar_f')
        out[di < 4] = floor
        out[di < 1] = np.minimum(floor.astype(int) + 40, 255).astype(np.uint8)
        out[di >= depth - 1] = col('crust')
        return out
    wob = lambda seed, k, amp: (hsh(s // k, seed) % (amp + 1)).astype(int)
    b_top = 5 + wob(1, 8, 1)
    b_soil = 22 + wob(2, 12, 2)
    b_clay = b_soil + 3
    b_peb = b_clay + 4 + wob(3, 6, 1)
    out[:] = col('rock')
    rock = di >= b_peb
    # bedding seams of the rock, broken in places, and lighter flecks
    seam = rock & ((di - b_peb) % 9 == 4) & (hsh(s // 6, (di - b_peb) // 9, 5) % 4 != 0)
    out[seam] = col('seam')
    out[rock & ~seam & (hs % 23 == 0)] = col('rock_l')
    out[rock & ~seam & (hs % 31 == 1)] = col('seam')
    peb = (di >= b_clay) & (di < b_peb)
    out[peb] = col('clay')
    stones = peb & (hsh(s // 2, di // 2, 7) % 3 != 0)
    out[stones] = np.where((hsh(s // 2, di // 2, 9) % 2 == 0)[stones, None], col('pebble'), col('pebble_d'))
    clay = (di >= b_soil) & (di < b_clay)
    out[clay] = col('clay')
    soil = (di >= b_top) & (di < b_soil)
    out[soil] = col('soil')
    out[soil & (hs % 13 == 0)] = col('speck')
    out[soil & (hs % 17 == 3)] = col('soil_d')
    top = di < b_top
    out[top] = col('top')
    out[top & (hs % 11 == 0)] = col('soil_d')
    lip = {'grass': 'grass', 'road': 'road', 'tent': 'canvas', 'water': 'water'}[mat]
    lip_rows = 3 if mat == 'water' else 2
    out[di < lip_rows] = col(lip)
    if mat == 'water':
        out[(di == 0) & (s % 4 == 1)] = col('water_l')
        out[(di >= lip_rows) & (di < b_top)] = col('mud')
    out[di >= depth - 1] = col('crust')
    return out


FLOOR_EDGE = {}


def section_piece(m, face, a0, a1, depth):
    """the section under small tiles a0..a1 of one front edge: +v (lit) runs along u at v = V1, +u along v at u = U1.
    Returns (RGBA image, top left on the board canvas)."""
    lit = face == 'v'
    if lit:
        xs = (16 * (a0 - m.V1), 16 * (a1 - m.V1))
        yt = 8 * (a0 + m.V1)
        yb = 8 * (a1 + m.V1)
    else:
        xs = (16 * (m.U1 - a1), 16 * (m.U1 - a0))
        yt = 8 * (m.U1 + a0)
        yb = 8 * (m.U1 + a1)
    left_end = lit and a0 == m.U0
    right_end = not lit and a0 == m.V0
    x0, x1 = xs[0] - (1 if left_end else 0), xs[1] + (1 if right_end else 0)
    y0, y1 = min(yt, yb), max(yt, yb) + depth + 2
    X, Y = np.meshgrid(np.arange(x0, x1) + .5, np.arange(y0, y1) + .5)
    if lit:
        u = X / 16 + m.V1
        d = Y - 8 * (u + m.V1)
        s = np.floor(16 * u + 1e-6).astype(int)
        tile = np.floor(u).astype(int)
        inside = (X >= xs[0]) & (X < xs[1])
    else:
        v = m.U1 - X / 16
        d = Y - 8 * (m.U1 + v)
        s = np.floor(-16 * v + 1e-6).astype(int)
        tile = np.floor(v).astype(int)
        inside = (X >= xs[0]) & (X < xs[1])
    a = np.zeros(X.shape + (4,), np.uint8)
    body = inside & (d > .3) & (d < depth)
    tiles = np.clip(tile, a0, a1 - 1)
    for t in range(a0, a1):
        mat = m.material(t, m.V1 - 1) if lit else m.material(m.U1 - 1, t)
        sel = body & (tiles == t)
        if sel.any():
            a[sel, :3] = strata(mat, s[sel], d[sel], lit, depth)
            a[sel, 3] = 255
    ink = inside & (d >= depth) & (d < depth + 1)
    if left_end:
        ink |= (X < xs[0]) & (d > .3) & (d < depth + 1)
    if right_end:
        ink |= (X >= xs[1]) & (d > .3) & (d < depth + 1)
    a[ink] = INK + (255,)
    return a, (x0, y0)


# ---- cut faces -----------------------------------------------------------------------------------------------------

WALLS = {
    # the scene castle (format 2 wall family swall): one small tile thick, 66px, parapet strips both sides
    'swall2': dict(height=pc.WALL_H, pal=((214, 202, 178), (156, 144, 124), (150, 136, 114), (104, 92, 76), (190, 180, 160))),
    # the old scene wall columns of format 1 maps (swall_u, swall_v): the battle map's blue-grey stone
    'swall_u': dict(height=pc.WALL_H, pal=((147, 163, 175), (99, 114, 128), (116, 130, 142), (72, 84, 96), (196, 210, 217))),
}
WALLS['swall_v'] = WALLS['swall_u']
def sampled_wall(name):
    """another wall family's block: its height from the picture's pivot (the foot's diamond 8px under the top of a
    66px block sits 75px down), its colours from the picture: the face's middle tones for brick, its darker quarter
    for joints, its lightest tenth for the top; the shaded side a step darker"""
    index = json.load(open(GFX / 'index.json'))
    for sheet in ('extra', 'props'):
        sp = index[sheet]['sprites'].get(name)
        if sp:
            break
    else:
        raise SystemExit(f'no picture {name} for a wall family block')
    x, y, w, h = sp['rect']
    a = np.array(Image.open(GFX / index[sheet]['image']).convert('RGBA'))[y:y + h, x:x + w].reshape(-1, 4)
    a = a[(a[:, 3] > 0) & (a[:, :3].sum(1) > 3 * 40)][:, :3].astype(int)
    lum = a.sum(1)
    order = np.argsort(lum)
    q = lambda lo, hi: tuple(int(c) for c in np.median(a[order[int(lo * len(a)):max(int(lo * len(a)) + 1, int(hi * len(a)))]], 0))
    brick, joint, top = q(.45, .75), q(.1, .3), q(.9, 1)
    dim = lambda c, k: tuple(int(x * k) for x in c)
    return dict(height=max(8, sp['pivot'][1] - 9), pal=(brick, joint, dim(brick, .8), dim(joint, .8), top))


RAMPART_PAL = ((150, 142, 126), (112, 104, 92), (118, 112, 100), (84, 78, 70), (166, 160, 146))


def section_paint(q, h, width, height, lit, pal, top_band=4, facing=3):
    """a wall's cut: brick facing round the outside (both faces and the top band), rubble core inside. q dots across
    (0..width-1), h pixels up (0..height-1). pal: lit brick, lit joint, shaded brick, shaded joint, top."""
    brick, joint = np.array(pal[0]), np.array(pal[1])
    if not lit:   # the shaded side's facing a step under the lit one, so the core still reads darker
        brick, joint = (brick + np.array(pal[2])) // 2, (joint + np.array(pal[3])) // 2
    out = np.zeros(q.shape + (3,), np.uint8)
    shell = (q < facing) | (q >= width - facing) | (h >= height - top_band)
    hb = height - 1 - h
    bj = (hb % 3 == 2) | (((q + (hb // 3) * 2) % 4) == 3)
    out[shell] = np.where(bj[shell, None], joint, brick)
    core = ~shell
    # rubble: stones about 5 x 3 in rows that drift, set in dark mortar, three tones darker than the facing
    row = h // 4
    shift = hsh(row, 11) % 5
    k = (q + shift) // 5
    mortar = (h % 4 == 0) | ((q + shift) % 5 == 0)
    tone = hsh(k, row, 3) % 3
    dark = lambda f: np.clip(brick * f + joint * (1 - f) * .5, 0, 255).astype(int)
    stone = np.where((tone == 0)[..., None], dark(.72), np.where((tone == 1)[..., None], dark(.62), dark(.54)))
    mor = (joint * .5).astype(int)
    out[core] = np.where(mortar[core, None], mor, stone[core])
    return out


def wall_cut(kind, axis, U, V, pal_key, m):
    """the cut face of a wall line's last tile on the board (axis u: cut at u = U + 1, toward +u), drawn with the
    kit's renderer from the tile's bodies; format 1's thin columns also take the wall top beside them"""
    tint = None
    if isinstance(pal_key, tuple):   # a tinted wall line: its body colour multiplies the section as it does the wall
        pal_key, tint = pal_key
    spec = WALLS.get(pal_key) or sampled_wall(pal_key)
    if pal_key.startswith('cwall'):   # the compound wall: 5px plinth and 22px plaster under its roof (gen_pixel_compound.py)
        spec = dict(spec, height=27)
    if tint:
        k = [int(tint[i:i + 2], 16) / 255 for i in (1, 3, 5)]
        spec = dict(spec, pal=tuple(tuple(int(c * k[j]) for j, c in enumerate(col)) for col in spec['pal']))
    H, pal = spec['height'], spec['pal']
    lit = axis == 'v'
    L = pc.Line(axis)
    cut_face = '+' + axis
    width = 16

    def paint(f, u, v, z, s, h):
        if f == cut_face:
            q = np.floor(16 * ((v - V) if axis == 'u' else (u - U)) + 1e-6).astype(int)
            q = np.clip(q, 0, width - 1)
            if axis == 'u':
                q = width - 1 - q   # across from the face's left on screen
            c = section_paint(q, np.floor(z + 1e-6).astype(int), width, H + pc.MERLON[0], lit, pal, top_band=4 + pc.MERLON[0])
            return ['#%02x%02x%02x' % tuple(x) for x in c]
        if f == '+z':
            return ['#%02x%02x%02x' % pal[4]] * len(u)
        b, j = (pal[0], pal[1]) if f.endswith('v') else (pal[2], pal[3])
        k = pc.brick(s, h, 'B', 'J')
        return ['#%02x%02x%02x' % (b if x == 'B' else j) for x in k]

    bodies = [pc.Body(pc.box(U, U + 1, V, V + 1, 0, H), paint, 0)]
    for side in (('+v', '-v') if axis == 'u' else ('+u', '-u')):
        if side[1] == 'u':
            a0, a1 = (U + 1 - pc.EDGE, U + 1) if side[0] == '+' else (U, U + pc.EDGE)
            bodies.append(pc.Body(pc.box(a0, a1, V, V + 1, H, H + pc.MERLON[0]), paint, 0))
        else:
            a0, a1 = (V + 1 - pc.EDGE, V + 1) if side[0] == '+' else (V, V + pc.EDGE)
            bodies.append(pc.Body(pc.box(U, U + 1, a0, a1, H, H + pc.MERLON[0]), paint, 0))
    back = (U - 1, V) if axis == 'u' else (U, V - 1)
    bodies.append(pc.Body(pc.box(back[0], back[0] + 1, back[1], back[1] + 1, 0, H), paint, -1))
    saved = dict(pc.PAL)
    keys = set()

    class Pal(dict):
        def __missing__(self, k):
            v = tuple(int(k[i:i + 2], 16) for i in (1, 3, 5))
            self[k] = v
            return v
    pc.PAL = Pal(saved)
    try:
        a, owner, (x0, y0) = pc.render(bodies, along=L.along())
        # which pixels are the cut: render the cut face alone in a marker colour
        def mark(f, u, v, z, s, h):
            return ['#010203' if f == cut_face else '#000000'] * len(u)
        mb = [pc.Body(b.planes, mark, b.tile) for b in bodies]
        ma, _, _ = pc.render(mb, along=L.along(), bounds=(x0, y0, x0 + a.shape[1], y0 + a.shape[0]))
    finally:
        pc.PAL = saved
    cut = np.all(ma[..., :3] == (1, 2, 3), axis=-1) & (ma[..., 3] > 0)
    X = np.arange(a.shape[1])[None, :] + x0 + .5
    cx = 16 * (U - V)   # the tile's centre on screen
    keep = cut.copy()
    if pal_key in ('swall_u', 'swall_v'):
        # the thin column draws the wall's near face only; past it lies what the dropped next column drew
        beside = (X >= cx) if axis == 'u' else (X < cx)
        keep |= beside & (owner == 0) & (a[..., 3] > 0)
    ink = np.all(a[..., :3] == INK, axis=-1) & (a[..., 3] > 0)
    grown = np.zeros_like(keep)
    for dy, dx in ((0, 1), (0, -1), (1, 0), (-1, 0)):
        grown |= np.roll(np.roll(keep, dy, 0), dx, 1)
    keep |= ink & grown & ~cut & (owner >= 0)
    out = np.zeros_like(a)
    out[keep] = a[keep]
    return out, (x0, y0)


def sprite(name):
    """a picture off the extra or props sheet: RGBA array and pivot"""
    index = json.load(open(GFX / 'index.json'))
    for sheet in ('extra', 'props'):
        sp = index[sheet]['sprites'].get(name)
        if sp:
            x, y, w, h = sp['rect']
            a = np.array(Image.open(GFX / index[sheet]['image']).convert('RGBA'))[y:y + h, x:x + w]
            return a, sp['pivot']
    raise SystemExit(f'no picture {name}')


def column_part(name, gu, gv, axis, m):
    """the part on the board of a format 1 wall column standing across a front edge: its picture's pixels on the
    near side of where the edge crosses its face (the face lies half a tile out from the anchor)"""
    a, (px, py) = sprite(name)
    cu, cv = 2 * gu + 1, 2 * gv + 1
    ax, ay = 16 * (cu - cv), 8 * (cu + cv)
    x0, y0 = int(round(ax - px)), int(round(ay - py))
    X = np.arange(a.shape[1])[None, :] + x0 + .5
    keep = (X < 16 * (m.U1 - (cv + .5))) if axis == 'u' else (X > 16 * (cu + .5 - m.V1))
    out = a.copy()
    out[~np.broadcast_to(keep, a.shape[:2])] = 0
    return out, (x0, y0)


def overlay(a, at, b, bt):
    """b laid over a, both RGBA with their top lefts: one picture"""
    x0, y0 = min(at[0], bt[0]), min(at[1], bt[1])
    x1 = max(at[0] + a.shape[1], bt[0] + b.shape[1])
    y1 = max(at[1] + a.shape[0], bt[1] + b.shape[0])
    out = np.zeros((y1 - y0, x1 - x0, 4), np.uint8)
    for img, (x, y) in ((b, bt), (a, at)):   # the cut over the column's part
        sub = out[y - y0:y - y0 + img.shape[0], x - x0:x - x0 + img.shape[1]]
        m = img[..., 3] > 0
        sub[m] = img[m]
    return out, (x0, y0)


# the walls drawn by a kit of their own: the section of the kit's bodies, each body by what it is made of
KITS = {'cwall': 'compound', 'pwall': 'palace'}
OBJECT_KITS = {'pwall_u': 'pwall'}
EARTH = ((160, 128, 92), (132, 104, 74))     # rammed earth inside a plastered wall, and its layer lines


def hexrgb(h, default=(255, 255, 255)):
    return tuple(int(h[i:i + 2], 16) for i in (1, 3, 5)) if h else default


def kit_bodies(kit, L):
    """a wall run of the kit along L over small tiles up to the cut at 1 (local tile [0, 1] x [0, 1], line through
    its middle), each body with its material: stone footing, plastered wall, tiled coping"""
    if kit == 'cwall':
        import gen_pixel_compound as ck
        bodies = ck.wall_line(L, -3, 4, c=.5)
        return bodies, ['stone', 'wall', 'tile', 'tile'], ck.T, dict(
            stone=((166,) * 3, (134,) * 3), skin=((236,) * 3, (194,) * 3), tile=((84,) * 3, (112,) * 3)), ('body', 'body', 'roof')
    import gen_pixel_palace as pk
    bodies = pk.wall_bodies(-3, 4, c=.5)
    return bodies, ['stone', 'wall', 'tile', 'tile'], pk.WALL[2], dict(
        stone=((170, 166, 156), (120, 116, 108)), skin=((164, 58, 44), (122, 40, 32)), tile=((70, 78, 82), (104, 112, 116))), ('body', 'body', 'roof')


def kit_cut(axis, U, V, kit, tint):
    """the cut of a kit wall's run at a front edge, through the tile (U, V): the kit's bodies cut flat at the edge,
    the cut face painted by material — stone courses, plaster skin round a rammed-earth core, the tiles' section"""
    L = pc.Line(axis)
    bodies, mats, half, pal, layers = kit_bodies(kit, L)
    lit = axis == 'v'
    shade = 1 if lit else .82
    mul = {k: [c / 255 for c in hexrgb(tint.get(k))] for k in ('body', 'roof')}
    col = lambda rgb, layer: '#%02x%02x%02x' % tuple(int(min(255, c * mul[layer][i] * shade)) for i, c in enumerate(rgb))
    cut_face = '+' + axis
    plane = ((1, 0, 0), 1, '+u') if axis == 'u' else ((0, 1, 0), 1, '+v')

    def cut_paint(mat, orig):
        def paint(f, u, v, z, s, h):
            if f != cut_face:
                return orig(f, u, v, z, s, h)
            across = (v if axis == 'u' else u) - .5      # from the wall's middle line
            q = np.floor(16 * across + 1e-6).astype(int)
            hz = np.floor(z + 1e-6).astype(int)
            if mat == 'stone':
                joint = (hz % 3 == 2) | ((q + (hz // 3) * 3) % 6 == 5)
                return [col(pal['stone'][1 if j else 0], 'body') for j in joint]
            if mat == 'tile':
                return [col(pal['tile'][1 if (a + b) % 3 == 0 else 0], 'roof') for a, b in zip(q, hz)]
            skin = np.abs(across) > half - 1 / 16 - 1e-6
            layer = hz % 4 == 0
            return [col(pal['skin'][0], 'body') if k else '#%02x%02x%02x' % tuple(int(c * shade) for c in EARTH[1 if l else 0])
                    for k, l in zip(skin, layer)]
        return paint

    bodies = [pc.Body(b.planes + [plane], cut_paint(mat, b.paint), 0) for b, mat in zip(bodies, mats)]

    class Pal(dict):
        def __missing__(self, k):
            v = hexrgb(k)
            self[k] = v
            return v
    saved = pc.PAL
    pc.PAL = Pal(saved)
    try:
        bounds = (-80, -90, 80, 40)
        a, owner, (x0, y0) = pc.render(bodies, along=L.along(), bounds=bounds)
        mark = lambda f, u, v, z, s, h: ['#010203' if f == cut_face else '#000000'] * len(u)
        ma, _, _ = pc.render([pc.Body(b.planes, mark, 0) for b in bodies], along=L.along(), bounds=bounds)
    finally:
        pc.PAL = saved
    cut = np.all(ma[..., :3] == (1, 2, 3), axis=-1) & (ma[..., 3] > 0)
    ink = np.all(a[..., :3] == INK, axis=-1) & (a[..., 3] > 0)
    grown = np.zeros_like(cut)
    for dy, dx in ((0, 1), (0, -1), (1, 0), (-1, 0)):
        grown |= np.roll(np.roll(cut, dy, 0), dx, 1)
    X = np.arange(a.shape[1])[None, :] + x0 + .5
    # ink only on the cut's outer side, where the wall ends
    outer = (X >= 16) if axis == 'u' else (X <= -16)
    keep = cut | (ink & grown & ~cut & np.broadcast_to(outer | True, cut.shape))
    out = np.zeros_like(a)
    out[keep] = a[keep]
    return out, (x0 + 16 * (U - V), y0 + 8 * (U + V))


def rampart_cut(axis, U, V, s, m):
    """the cut of a city wall top running into a front edge, one small tile of it: brick facing on the run's two
    outer sides and along the top, rubble core between. The run is the edge tiles of that height round this one."""
    lift = m.rules['heights'][str(s)]['lift']
    lit = axis == 'v'
    t = V if axis == 'u' else U
    on = (lambda k: m.step(U, k) == s) if axis == 'u' else (lambda k: m.step(k, V) == s)
    lo, hi = t, t
    while on(lo - 1):
        lo -= 1
    while on(hi + 1):
        hi += 1
    if axis == 'u':
        x0, x1 = 16 * (U - V), 16 * (U - V + 1)
    else:
        x0, x1 = 16 * (U - V - 1), 16 * (U - V)
    yt = 8 * (U + V + 1) - lift - 4
    y1 = 8 * (U + V + 2) + 4
    X, Y = np.meshgrid(np.arange(x0, x1) + .5, np.arange(yt, y1) + .5)
    if axis == 'u':
        v = (U + 1) - X / 16
        z = 8 * (U + 1 + v) - Y
        g = np.floor(16 * (v - lo) + 1e-6).astype(int)
    else:
        u = X / 16 + V + 1
        z = 8 * (u + V + 1) - Y
        g = np.floor(16 * (u - lo) + 1e-6).astype(int)
    width = 16 * (hi - lo + 1)
    body = (z > .3) & (z <= lift + .25)
    h = np.clip(np.floor(z).astype(int), 0, lift - 1)
    colours = section_paint(np.clip(g, 0, width - 1), h, width, lift, lit, RAMPART_PAL, top_band=6, facing=4)
    a = np.zeros(X.shape + (4,), np.uint8)
    a[body, :3] = colours[body]
    a[body, 3] = 255
    return a, (x0, yt)


# ---- the board -----------------------------------------------------------------------------------------------------

def pack(pieces):
    """shelf-pack RGBA arrays (trimmed) into one atlas: [(rect, (dx, dy) trim offset)]"""
    trimmed = []
    for a in pieces:
        ys, xs = np.nonzero(a[..., 3])
        if len(xs) == 0:
            trimmed.append((None, (0, 0)))
            continue
        trimmed.append((a[ys.min():ys.max() + 1, xs.min():xs.max() + 1], (int(xs.min()), int(ys.min()))))
    width = max([1024] + [t.shape[1] for t, _ in trimmed if t is not None])
    x = y = shelf = 0
    rects = []
    for t, _ in trimmed:
        if t is None:
            rects.append(None)
            continue
        h, w = t.shape[:2]
        if x + w > width:
            x, y, shelf = 0, y + shelf + 1, 0
        rects.append((x, y, w, h))
        x += w + 1
        shelf = max(shelf, h)
    atlas = np.zeros((max(1, y + shelf), width, 4), np.uint8)
    for (t, _), r in zip(trimmed, rects):
        if r:
            atlas[r[1]:r[1] + r[3], r[0]:r[0] + r[2]] = t
    return atlas, [(r, off) for r, (_, off) in zip(rects, trimmed)]


def bake(path):
    m = MapInfo(path)
    depth = m.depth()
    pieces, meta = [], []
    for face, lo, hi in (('v', m.U0, m.U1), ('u', m.V0, m.V1)):
        for a0 in range(lo, hi, CHUNK):
            a, at = section_piece(m, face, a0, min(hi, a0 + CHUNK), depth)
            pieces.append(a)
            meta.append(('section', at, None))
    for kind, axis, U, V, extra in m.cuts:
        if kind == 'rampart':
            a, at = rampart_cut(axis, U, V, extra, m)
        elif isinstance(extra, dict) and 'kit' in extra:
            a, at = kit_cut(axis, U, V, extra['kit'], extra['tint'])
        elif isinstance(extra, dict):
            a, at = wall_cut(kind, axis, U, V, extra['column'], m)
            a, at = overlay(a, at, *column_part(extra['column'], *extra['at'], axis, m))
        else:
            a, at = wall_cut(kind, axis, U, V, extra, m)
        pieces.append(a)
        meta.append((kind, at, (U, V)))
    atlas, rects = pack(pieces)
    OUT.mkdir(parents=True, exist_ok=True)
    Image.fromarray(atlas).save(OUT / f'{m.id}.png', optimize=True)
    doc = {'map': m.id, 'source': m.source, 'board': [m.U0, m.V0, m.U1, m.V1], 'image': f'{m.id}.png', 'section': [], 'cuts': []}
    for (kind, at, tile), (r, off) in zip(meta, rects):
        if r is None:
            continue
        ax, ay = at[0] + off[0], at[1] + off[1]
        if kind == 'section':
            doc['section'].append([*r, ax, ay])
        else:
            doc['cuts'].append({'rect': list(r), 'at': [ax, ay], 'tile': list(tile), 'kind': kind})
    lines = ['{', f' "map": {json.dumps(doc["map"])},', f' "source": "{doc["source"]}",', f' "board": {json.dumps(doc["board"])},',
             f' "image": {json.dumps(doc["image"])},', ' "section": [']
    lines += ['  ' + json.dumps(s) + (',' if i < len(doc['section']) - 1 else '') for i, s in enumerate(doc['section'])]
    lines += [' ],', ' "cuts": [']
    lines += ['  ' + json.dumps(c) + (',' if i < len(doc['cuts']) - 1 else '') for i, c in enumerate(doc['cuts'])]
    lines += [' ]', '}']
    (OUT / f'{m.id}.json').write_text('\n'.join(lines) + '\n')
    return m, len(doc['section']), len(doc['cuts'])


def scene_maps():
    index = json.load(open(GFX / 'index.json'))
    return [GFX / p for p in index['maps'] if Path(p).name.startswith('S_')]


def main(args):
    for row in json.load(open(GFX / 'scene.json'))['floors'].items():
        if row[1].get('edge'):
            FLOOR_EDGE[row[0]] = tuple(int(row[1]['edge'][i:i + 2], 16) for i in (1, 3, 5))
    if args == ['--check']:
        stale = []
        for p in scene_maps():
            j = OUT / (p.stem + '.json')
            want = hashlib.sha1(p.read_bytes()).hexdigest()
            try:
                MapInfo(p)
            except SkipMap:
                continue
            if not j.exists() or json.load(open(j))['source'] != want:
                stale.append(p.stem)
        print('stale or missing:', ' '.join(stale) or 'none')
        return
    paths = scene_maps() if args == ['--all'] else [GFX / 'maps' / f'{a}.json' for a in args]
    if not paths:
        print(__doc__)
        return
    for p in paths:
        try:
            m, ns, nc = bake(p)
            print(f'{m.id}: board {m.U0},{m.V0}..{m.U1},{m.V1}, {ns} section pieces, {nc} cuts')
        except SkipMap as e:
            print(f'{p.stem}: skipped, {e}')


if __name__ == '__main__':
    main(sys.argv[1:])
