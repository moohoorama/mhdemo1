"""The pieces of the scene maps' diorama board edge (docs/presentation-scene.md, 판 단면), drawn as pixel art on the
2:1 small-tile grid with the castle kit's renderer and face coordinates (s across, h up/down), so strata and brick
courses sit on the slope crisply. The scene map rules (internal/scenemap, scene.json "board", wall families' and
heights' "cut", "cuts" for walls laid as objects) lay them tile by tile; nothing here knows any map.

  board_<family>_pu_<k>, board_<family>_pv_<k>   the ground's cross-section under one small tile of the +u / +v
                 front edge, k = 0..3 variants; families: grass, road, water, tent, hall (a room's floor on its stone
                 plinth), under (below a raised tile, whose own faces stand above). Pivot at the tile's centre.
  board_end_pu, board_end_pv   the outline down the board's side corner past the last tile of the +u / +v edge
  swall2_cut_pu/pv              the castle wall's cut where a line runs into a front edge: brick facing round a
                                rubble core, parapet strips on top
  cwall_cut_pu/pv (+ _roof)     the compound wall's cut: stone footing, plaster skin round rammed earth, the coping's
                                tiles (greys, tinted like the wall: body and roof layers)
  pwall_u_cut_pu                the palace wall's cut (vermilion skin, granite footing, tiled coping)
  rwall_cut_pu/pv_<a|m|b|ab>    a city wall top's cut, one small tile of it: a (the run's start), m (middle), b (end),
                                ab (a run one tile wide); brick facing on the run's outer sides and along the top

  python3 tools/scene_maps/diorama.py --write      # the pieces into assets_candidate/tiles and tiles.json
  python3 tools/scene_maps/diorama.py --preview D  # the pieces at 4x and a laid-out corner into D
then `python3 tools/build_extra.py` from the repo root packs them into map/extra.png.
"""
import json
import os
import sys
from pathlib import Path

import numpy as np
from PIL import Image

VER5 = Path(__file__).resolve().parents[2]
TOOLS = VER5.parent / 'assets_candidate/tools'
TILES = VER5.parent / 'assets_candidate/tiles'
sys.path.insert(0, str(TOOLS))
import gen_pixel_castle as pc  # noqa: E402

DEPTH = 56          # the slab under the board, pixels
INK = pc.INK
VARIANTS = 4
MADE = "diorama board edge, pixel art on the small-tile grid (tools/scene_maps/diorama.py)"

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
    'plank': ((126, 74, 44), (96, 56, 34)),
    'plank_l': ((166, 106, 66), (128, 82, 52)),
}
def hsh(*a):
    """a stable small hash of integers (numpy arrays welcome)"""
    h = np.uint32(2166136261)
    for x in a:
        h = (np.asarray(x, np.int64).astype(np.uint32) ^ h) * np.uint32(16777619)
        h ^= h >> np.uint32(13)
    return h


def rgb(name, lit):
    return P[name][0 if lit else 1]


def strata(mat, s, d, lit, depth):
    """the section's colour at dots s along the edge, d pixels below the ground, as an (n, 3) array"""
    n = s.shape[0]
    out = np.zeros((n, 3), np.uint8)
    di = np.floor(d).astype(int)
    hs = hsh(s // 2, di)
    col = lambda name: np.array(rgb(name, lit), np.uint8)
    if mat == 'hall':
        floor = col('plank')
        out[:] = col('ashlar')
        # dressed stone courses of 7 rows and a joint, blocks 14 dots with a joint dot, every other course shifted
        c = np.maximum(di - 4, 0)
        course = c // 8
        joint = (c % 8 == 7) | ((s + course * 7) % 14 == 13)
        out[joint] = col('ashlar_j')
        out[(c // 8) == (depth - 4) // 8] = col('ashlar_f')
        out[di < 4] = floor
        out[di < 1] = col('plank_l')
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
    if mat == 'under':
        out[di >= depth - 1] = col('crust')
        return out
    lip = {'grass': 'grass', 'road': 'road', 'tent': 'canvas', 'water': 'water'}[mat]
    lip_rows = 3 if mat == 'water' else 2
    out[di < lip_rows] = col(lip)
    if mat == 'water':
        out[(di == 0) & (s % 4 == 1)] = col('water_l')
        out[(di >= lip_rows) & (di < b_top)] = col('mud')
    out[di >= depth - 1] = col('crust')
    return out



FAMILIES = ('grass', 'road', 'water', 'tent', 'hall', 'under')


def ground_piece(fam, face, k):
    """one small tile of a front edge's cross-section: tile (0, 0), its +u side (the plane u = 1) or +v side (v = 1),
    the strata laid in face coordinates continuing those of variant k - 1 (s counts dots along the edge)"""
    lit = face == 'v'
    xs = (-16, 0) if lit else (0, 16)
    X, Y = np.meshgrid(np.arange(xs[0] - 1, xs[1] + 1) + .5, np.arange(0, 16 + DEPTH + 2) + .5)
    if lit:
        u = X / 16 + 1
        d = Y - 8 * (u + 1)
        s = np.floor(16 * u + 1e-6).astype(int)
    else:
        v = 1 - X / 16
        d = Y - 8 * (1 + v)
        s = np.floor(16 * (1 - v) + 1e-6).astype(int)
    s = np.clip(s, 0, 15) + 16 * k
    inside = (X >= xs[0]) & (X < xs[1])
    body = inside & (d > .3) & (d < DEPTH)
    a = np.zeros(X.shape + (4,), np.uint8)
    a[body, :3] = strata(fam, s[body], d[body], lit, DEPTH)
    a[body, 3] = 255
    ink = inside & (d >= DEPTH) & (d < DEPTH + 1)
    a[ink] = INK + (255,)
    return a, (xs[0] - 1, 0)


def end_piece(face):
    """the outline down the side corner: left of the +v edge's first tile, right of the +u edge's first tile"""
    x = -17 if face == 'v' else 16
    y0 = 8 if face == 'v' else 0     # the corner's ground point: (0, 1) on the +v edge, (1, 0) on the +u edge
    a = np.zeros((DEPTH + 2, 1, 4), np.uint8)
    a[1:DEPTH + 1, 0] = INK + (255,)
    return a, (x, y0 + 8)


WALLS = {
    'swall2': dict(height=pc.WALL_H, pal=((214, 202, 178), (156, 144, 124), (150, 136, 114), (104, 92, 76), (190, 180, 160))),
}
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



class Pal(dict):
    """the renderer's palette, taking any "#rrggbb" key as its colour"""
    def __missing__(self, k):
        v = hexrgb(k)
        self[k] = v
        return v


def cut_render(bodies, axis, bounds):
    """the bodies drawn, and which pixels lie on their cut face (the plane at 1 across the run's end, toward +axis)
    with the outline round it"""
    L = pc.Line(axis)
    cut_face = '+' + axis
    saved = pc.PAL
    pc.PAL = Pal(saved)
    try:
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
    keep = cut | (ink & grown)
    out = np.zeros_like(a)
    out[keep] = a[keep]
    return out, (x0, y0)


def swall2_cut(axis):
    """the castle wall block's cut: the block of tile (0, 0) and its parapet strips, cut at 1 toward +axis"""
    spec = WALLS['swall2']
    H, pal = spec['height'], spec['pal']
    lit = axis == 'v'
    cut_face = '+' + axis
    width = 16

    def paint(f, u, v, z, s, h):
        if f == cut_face:
            q = np.clip(np.floor(16 * (v if axis == 'u' else u) + 1e-6).astype(int), 0, width - 1)
            if axis == 'u':
                q = width - 1 - q
            c = section_paint(q, np.floor(z + 1e-6).astype(int), width, H + pc.MERLON[0], lit, pal, top_band=4 + pc.MERLON[0])
            return ['#%02x%02x%02x' % tuple(x) for x in c]
        return ['#%02x%02x%02x' % pal[4]] * len(u)

    bodies = [pc.Body(pc.box(0, 1, 0, 1, 0, H), paint, 0)]
    for side in ('+', '-'):
        a0, a1 = (1 - pc.EDGE, 1) if side == '+' else (0, pc.EDGE)
        box = pc.box(0, 1, a0, a1, H, H + pc.MERLON[0]) if axis == 'u' else pc.box(a0, a1, 0, 1, H, H + pc.MERLON[0])
        bodies.append(pc.Body(box, paint, 0))
    return cut_render(bodies, axis, (-40, -110, 40, 40))


def rampart_cut(axis, place, lift=120):
    """one small tile of a city wall top's cut (tile (0, 0), the plane at 1 toward +axis), place in its run along
    the edge: a its first tile (toward -v for a +u cut, -u for +v), b its last, m between, ab both"""
    lit = axis == 'v'
    width = 16 * 3
    start = {'a': 0, 'm': 16, 'b': 32, 'ab': 0}[place]
    if place == 'ab':
        width = 16
    xs = (0, 16) if axis == 'u' else (-16, 0)
    X, Y = np.meshgrid(np.arange(xs[0], xs[1]) + .5, np.arange(-lift - 4, 24) + .5)
    if axis == 'u':
        v = 1 - X / 16
        z = 8 * (1 + v) - Y
        g = np.floor(16 * v + 1e-6).astype(int)
    else:
        u = X / 16 + 1
        z = 8 * (u + 1) - Y
        g = np.floor(16 * u + 1e-6).astype(int)
    body = (z > .3) & (z <= lift + .25)
    h = np.clip(np.floor(z).astype(int), 0, lift - 1)
    colours = section_paint(np.clip(g, 0, 15) + start, h, width, lift, lit, RAMPART_PAL, top_band=6, facing=4)
    a = np.zeros(X.shape + (4,), np.uint8)
    a[body, :3] = colours[body]
    a[body, 3] = 255
    return a, (xs[0], -lift - 4)


EARTH = ((160, 128, 92), (132, 104, 74))     # rammed earth inside a plastered wall, and its layer lines


def hexrgb(h, default=(255, 255, 255)):
    return tuple(int(h[i:i + 2], 16) for i in (1, 3, 5)) if h else default


def kit_bodies(kit, L):
    """a wall run of a kit along L through tile (0, 0) (its line through the tile's middle), past both sides; each
    body with its material: stone footing, plastered wall, tiled coping and ridge"""
    if kit == 'cwall':
        import gen_pixel_compound as ck
        return ck.wall_line(L, -3, 4, c=.5), ck.T, dict(
            stone=((166,) * 3, (134,) * 3), skin=((236,) * 3, (194,) * 3), tile=((84,) * 3, (112,) * 3))
    import gen_pixel_palace as pk
    return pk.wall_bodies(-3, 4, c=.5), pk.WALL[2], dict(
        stone=((170, 166, 156), (120, 116, 108)), skin=((164, 58, 44), (122, 40, 32)), tile=((70, 78, 82), (104, 112, 116)))


def kit_cut(axis, kit):
    """a kit wall's run cut flat at 1 toward +axis through tile (0, 0), the cut painted by material: stone courses,
    the plaster skin round a rammed-earth core, the coping's tiles. Returns the picture and its roof part apart
    (the game tints them as the wall's body and roof layers)"""
    L = pc.Line(axis)
    bodies, half, pal = kit_bodies(kit, L)
    mats = ['stone', 'wall', 'tile', 'tile']
    shade = 1 if axis == 'v' else .82
    hexc = lambda rgb: '#%02x%02x%02x' % tuple(int(min(255, c * shade)) for c in rgb)
    cut_face = '+' + axis
    plane = ((1, 0, 0), 1, '+u') if axis == 'u' else ((0, 1, 0), 1, '+v')

    def cut_paint(mat):
        def paint(f, u, v, z, s, h):
            if f != cut_face:
                return ['#%02x%02x%02x' % pal['skin'][0]] * len(u)
            across = (v if axis == 'u' else u) - .5
            q = np.floor(16 * across + 1e-6).astype(int)
            hz = np.floor(z + 1e-6).astype(int)
            if mat == 'stone':
                joint = (hz % 3 == 2) | ((q + (hz // 3) * 3) % 6 == 5)
                return [hexc(pal['stone'][1 if j else 0]) for j in joint]
            if mat == 'tile':
                return [hexc(pal['tile'][1 if (a + b) % 3 == 0 else 0]) for a, b in zip(q, hz)]
            skin = np.abs(across) > half - 1 / 16 - 1e-6
            return [hexc(pal['skin'][0]) if k else hexc(EARTH[1 if l else 0]) for k, l in zip(skin, hz % 4 == 0)]
        return paint

    cut_bodies = [pc.Body(b.planes + [plane], cut_paint(m), 0) for b, m in zip(bodies, mats)]
    a, origin = cut_render(cut_bodies, axis, (-48, -60, 48, 32))
    # the roof part: the coping's bodies alone, cut the same way
    roof, _ = cut_render([b if m == 'tile' else pc.Body(b.planes, lambda f, u, v, z, s, h: ['#010101'] * len(u), 0)
                          for b, m in zip(cut_bodies, mats)], axis, (-48, -60, 48, 32))
    is_roof = (roof[..., 3] > 0) & ~np.all(roof[..., :3] == (1, 1, 1), axis=-1) & ~np.all(roof[..., :3] == INK, axis=-1)
    body = a.copy()
    body[is_roof] = 0
    top = np.zeros_like(a)
    top[is_roof] = a[is_roof]
    return body, top, origin


def piece(a, origin, pivot=(0, 8)):
    """trimmed picture and its pivot (the tile's centre by default)"""
    ys, xs = np.nonzero(a[..., 3])
    x0, x1, y0, y1 = xs.min(), xs.max() + 1, ys.min(), ys.max() + 1
    return Image.fromarray(a[y0:y1, x0:x1]), (int(pivot[0] - origin[0] - x0), int(pivot[1] - origin[1] - y0))


def kit():
    """every piece by name: (picture, pivot, layer)"""
    out = {}
    for fam in FAMILIES:
        for face in ('u', 'v'):
            for k in range(VARIANTS):
                out[f'board_{fam}_p{face}_{k}'] = (*piece(*ground_piece(fam, face, k)), 'ground')
    for face in ('u', 'v'):
        out[f'board_end_p{face}'] = (*piece(*end_piece(face)), 'ground')
        out[f'swall2_cut_p{face}'] = (*piece(*swall2_cut(face)), 'object')
        body, roof, origin = kit_cut(face, 'cwall')
        out[f'cwall_cut_p{face}'] = (*piece(body, origin), 'object')
        out[f'cwall_cut_p{face}_roof'] = (*piece(roof, origin), 'object')
        for place in ('a', 'm', 'b', 'ab'):
            out[f'rwall_cut_p{face}_{place}'] = (*piece(*rampart_cut(face, place)), 'object')
    body, roof, origin = kit_cut('u', 'pwall')
    a = body.copy()
    a[roof[..., 3] > 0] = roof[roof[..., 3] > 0]
    out['pwall_u_cut_pu'] = (*piece(a, origin), 'object')
    return out


def write():
    """the pieces into assets_candidate/tiles and their entries of tiles.json, replacing this kit's earlier ones"""
    path = TILES / 'tiles.json'
    man = json.load(open(path))
    made = kit()
    for name in [n for n, e in man['sprites'].items() if e.get('made') == MADE and n not in made]:
        man['sprites'].pop(name)
        if (TILES / f'{name}.png').exists():
            os.remove(TILES / f'{name}.png')
    for name, (im, pivot, layer) in made.items():
        im.save(TILES / f'{name}.png')
        man['sprites'][name] = dict(file=f'tiles/{name}.png', size=list(im.size), pivot=list(pivot), layer=layer,
                                    scale='scene', made=MADE, shadow=False)
    json.dump(man, open(path, 'w'), ensure_ascii=False, indent=1)
    open(path, 'a').write('\n')
    print(len(made), 'pieces')


def preview(out):
    """the pieces at 4x, and a board corner laid out of them: a grass +v edge with a road tile, a +u edge with water
    and a raised tile, a castle wall line running into each edge"""
    os.makedirs(out, exist_ok=True)
    k = kit()
    for name, (im, _, _) in k.items():
        im.resize((im.width * 4, im.height * 4), Image.NEAREST).save(os.path.join(out, name + '.png'))
    n = 8
    cv = Image.new('RGBA', (32 * n + 80, 16 * n + 200), (18, 18, 22, 255))
    ox, oy = 16 * n + 40, 100

    def lay(name, u, v):
        im, (px, py), _ = k[name]
        x, y = ox + 16 * (u - v), oy + 8 * (u + v + 1)
        cv.alpha_composite(im, (x - px, y - py))
    from PIL import ImageDraw
    d = ImageDraw.Draw(cv)
    P = lambda u, v: (ox + 16 * (u - v), oy + 8 * (u + v))
    d.polygon([P(0, 0), P(n, 0), P(n, n), P(0, n)], fill=(96, 132, 70, 255))
    for u in range(n):
        lay(f'board_{"road" if u == 3 else "grass"}_pv_{(u * 7) % 4}', u, n - 1)
    for v in range(n):
        lay(f'board_{"water" if v == 2 else "grass"}_pu_{(v * 5) % 4}', n - 1, v)
    lay('board_end_pv', 0, n - 1)
    lay('board_end_pu', n - 1, 0)
    lay('swall2_cut_pu', n - 1, 4)
    lay('swall2_cut_pv', 5, n - 1)
    cv.resize((cv.width * 3, cv.height * 3), Image.NEAREST).save(os.path.join(out, 'corner.png'))


if __name__ == '__main__':
    if sys.argv[1:] == ['--write']:
        write()
    elif sys.argv[1:2] == ['--preview']:
        preview(sys.argv[2])
    else:
        print(__doc__)
