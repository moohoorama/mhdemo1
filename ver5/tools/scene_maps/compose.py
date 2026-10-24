#!/usr/bin/env python3
"""Make the scene maps that the candidate viewer has no scene for: rooms laid out here prop by prop, and variants of
the viewer-made maps with props swapped or added. Writes assets/graphics/maps/<id>.json and lists them in index.json.

  python3 ver5/tools/scene_maps/compose.py

Rooms go through build.py's builder as if the viewer had drawn them (props only, on plain ground), so they come out
in the same native format. Coordinates are game cells: u runs down to the right on screen, v down to the left;
the back walls stand on u = -0.5 (left) and v = -0.5 (right).
"""
import copy
import json
import math
import random
from pathlib import Path

import build

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'


def room(W, H, floor, wall, extra_walls=(), props=(), pillar='pillar_red'):
    """A W x H room: a floor prop on every cell, walls along the two back sides with pillars at the ends."""
    draws = []

    def put(name, u, v):
        draws.append({'n': f'pr:{name}', 'x': 32 * (u - v), 'y': 16 * (u + v + 1)})

    for v in range(H):
        for u in range(W):
            put(floor, u, v)
    for v in range(H):
        put(f'{wall}_nw', -.5, v - .5)
        for w in extra_walls:
            put(f'{w}_nw', -.5, v - .5)
    for u in range(W):
        put(f'{wall}_ne', u - .5, -.5)
        for w in extra_walls:
            put(f'{w}_ne', u - .5, -.5)
    for u, v in ((-.5, -.5), (W - .5, -.5), (-.5, H - .5)):
        put(pillar, u, v)
    for name, u, v in props:
        put(name, u, v)
    return {'w': 2 * W, 'h': 2 * H, 'noGround': True, 'states': [{'visible': [], 'draws': draws, 'grass': []}]}


# 맹주의 막사: the host's dais against the left wall, guests in two rows facing across the hall, the way out at the
# lower right
COUNCIL = room(9, 6, 'floor_tent', 'wall_canvas', props=[
    ('banner', -.2, 1.2), ('banner', -.2, 3.8), ('banner', 2.5, -.2), ('banner', 5.5, -.2),
    ('command_flag', .3, 1.4), ('command_flag', .3, 3.6),
    ('dais', .55, 2.5), ('desk', 1.6, 2.5),
    ('table_wine', 2.4, 1.75), ('table_wine', 3.9, 1.75), ('table_wine', 5.4, 1.75),
    ('table_wine', 2.4, 3.25), ('table_wine', 3.9, 3.25), ('table_wine', 5.4, 3.25),
    ('brazier', 1.2, .5), ('brazier', 1.2, 4.6), ('brazier', 6.8, .5), ('brazier', 6.8, 4.6),
])
# 궁궐 옥좌 앞: the emperor's seat (his sprite brings the throne) before a gold screen at the right wall, the red way down the middle, room for two rows of officials
HALL = room(7, 9, 'floor_palace', 'wall_lacquer', extra_walls=['curtain'], props=[
    *[('floor_carpet', 3, v) for v in range(1, 9)],
    ('screen_gold', 3.0, .1), ('candle_stand', 1.9, .4), ('candle_stand', 4.1, .4), ('incense', 2.2, 1.2), ('incense', 3.8, 1.2),
    *[(p, u, v) for v in (1.3, 3.6, 5.9) for u, p in ((.7, 'pillar_red'), (5.3, 'pillar_red'))],
    ('lantern', 1.2, 7.6), ('lantern', 4.8, 7.6),
])

# 중형 군막: a lord's own tent (공손찬, 노식): bed, weapon rack and armour stand along the walls
TENT_MID = room(6, 5, 'floor_tent', 'wall_canvas', props=[
    ('banner', -.2, 1.0), ('banner', 2.6, -.2), ('bed', .9, 3.6), ('weapon_rack', 4.2, -.1), ('armor_stand', 5.2, -.05),
    ('map_table', 2.6, 1.9), ('brazier', .6, .6), ('brazier', 5.2, 4.2), ('desk', .7, 1.9),
])
# 원술의 막사: the same size, dressed for a rich man: gold screen behind his seat, wine tables, incense, candles
TENT_LUX = room(6, 5, 'floor_tent', 'wall_canvas', props=[
    *[('floor_carpet', u, v) for u in range(1, 5) for v in range(1, 4)],
    ('screen_gold', .25, 2.0), ('dais', .6, 2.0), ('table_wine', 1.6, 2.0), ('table_wine', 2.8, 1.0), ('table_wine', 2.8, 3.0),
    ('incense', 1.4, .5), ('candle_stand', .3, .7), ('candle_stand', .3, 3.3), ('wine_jar', 4.6, .5), ('wine_jar', 4.9, .8),
    ('banner_yellow', -.2, 4.0), ('banner_yellow', 4.0, -.2),
])
# 맹주의 막사 (12×8): the host's dais at the left wall, four seats down each side behind low tables, room for each
# lord's officer to stand behind him, the way out at the lower right
COUNCIL_BIG = room(12, 8, 'floor_tent', 'wall_canvas', props=[
    ('banner', -.2, 1.5), ('banner', -.2, 5.5), ('banner', 3.5, -.2), ('banner', 7.5, -.2),
    ('command_flag', .3, 2.3), ('command_flag', .3, 4.7), ('dais', .55, 3.5), ('desk', 1.6, 3.5),
    ('candle_stand', .3, 2.9), ('candle_stand', .3, 4.1),
    *[('table_wine', u, v) for u in (2.6, 4.4, 6.2, 8.0) for v in (2.25, 4.75)],
    ('brazier', 1.4, .5), ('brazier', 1.4, 6.6), ('brazier', 9.6, .5), ('brazier', 9.6, 6.6),
    ('weapon_rack', 5.4, -.1), ('weapon_rack', -.1, 6.4), ('armor_stand', 9.0, -.05),
])

# 장비의 주점 (7×6): a plank-floored drinking room, the counter and the wine jars along the back right wall, Guan Yu's
# corner table at the far left, four more tables for the noisy customers, the door at the lower right
TAVERN = room(7, 6, 'floor_wood', 'wall_plaster', pillar='pillar_wood', props=[
    ('counter', 4.6, .15), ('wine_jar', 5.9, .2), ('wine_jar', 6.3, .45), ('wine_jar', 6.25, .05), ('wine_jar', 3.3, .1),
    ('bench', 1.0, .1), ('lantern', -.1, 2.6), ('lantern', 2.6, -.1), ('wine_jar', -.05, 4.3), ('wine_jar', .2, 4.5),
    ('table_wine', .9, 1.0),
    ('table_wine', 3.2, 2.4), ('table_wine', 1.4, 3.6), ('table_wine', 4.6, 3.6), ('table_wine', 3.0, 4.8),
    ('wine_jar', .15, 5.3), ('grain_sack', .2, 4.9), ('water_jar', 6.4, 1.2), ('bench', 6.3, 3.0),
])
# 주막 (5×4): a small rural inn, earth-poor: straw mats on the boards, two low tables, a hearth pot, a jar or two;
# the quiet corner at the far left, the way in at the lower right
INN = room(5, 4, 'floor_wood', 'wall_plaster', pillar='pillar_wood', props=[
    ('straw_mat', .9, 1.1), ('straw_mat', 2.6, 2.3), ('straw_mat', 1.0, 2.9),
    ('table_wine', .8, .9), ('table_wine', 2.6, 2.2), ('lantern', -.1, 1.9), ('candle_stand', 3.4, -.05),
    ('cauldron', 4.3, .5), ('wine_jar', 3.0, .1), ('wine_jar', 2.7, .2), ('water_jar', .1, 3.4), ('grain_sack', 4.5, 1.1),
    ('bench', 1.8, .1),
])

ROOMS = [(COUNCIL_BIG, 'S_council', '맹주의 막사(넓은 군막)'), (HALL, 'S_hall', '궁궐 옥좌 앞(넓은 전각)'),
         (TENT_MID, 'S_tent_mid', '장수의 군막'), (TENT_LUX, 'S_tent_lux', '원술의 막사'),
         (TAVERN, 'S_tavern', '장비의 주점'), (INN, 'S_inn', '주막')]


def outdoor(W, H, glyph_at, props=(), trees=(), seed=1, grass=.12, margin=0):
    """An outdoor map of W x H game cells for build.build: glyph_at(cu, cv) gives each small tile ('.' grass, 'd'
    soil, '~' water, 'b' bridge, 'h' cliff, 'c' wall, 'i' paving); trees are (game u, v, kind); grass tufts are
    scattered on grass. margin: game cells of picture drawn round the playing area on every side, so the ground runs
    on past the screen's edge instead of ending in a diamond."""
    rnd = random.Random(seed)
    visible, draws, tufts = [], [], []
    for cv in range(-2 * margin, 2 * (H + margin)):
        for cu in range(-2 * margin, 2 * (W + margin)):
            ch = glyph_at(cu, cv)
            visible.append((cu, cv, ch))
            if ch == '.' and rnd.random() < grass:
                tufts.append({'n': f'ts:grass_{rnd.randrange(4)}', 'x': 16 * (cu - cv) + rnd.randrange(-6, 7),
                              'y': 8 * (cu + cv + 1) + rnd.randrange(-3, 4)})
    for u, v, kind in trees:  # an old tile tree number, or a scene tree species (trees.py); build.py plants them
        name = f'ts:tree_{kind}' if isinstance(kind, int) else f'tree:{kind}'
        draws.append({'n': name, 'x': 32 * (u - v), 'y': 16 * (u + v + 1)})
    for name, u, v in props:
        draws.append({'n': f'pr:{name}', 'x': 32 * (u - v), 'y': 16 * (u + v + 1)})
    return {'w': 2 * W, 'h': 2 * H, 'noGround': False, 'states': [{'visible': visible, 'draws': draws, 'grass': tufts}]}


def bridge_glyph(cu, cv):
    if 9 <= cv <= 11:
        return 'b' if 11 <= cu <= 12 else '~'
    if 11 <= cu <= 12:
        return 'd'
    return '.'


# 계교: a river across the map, a wooden bridge in the middle, the road running to it from both banks; the shield
# wall and the hidden crossbowmen stand on the near bank (large v), the white horses come from the far bank
BRIDGE = outdoor(12, 10, bridge_glyph, props=[
    ('shield_wall', 2.6, 6.4), ('shield_wall', 4.6, 6.4), ('shield_wall', 8.1, 6.4), ('shield_wall', 10.1, 6.4),
    ('watchtower', 9.6, 8.6), ('command_flag_yellow', 8.2, 8.2), ('banner_yellow', 4.0, 8.6), ('banner_yellow', 2.0, 8.6),
    ('tent', 1.2, 9.3), ('tent', 10.8, 9.4),
], trees=[(0.5, 1.0, 0), (1.6, .4, 2), (3.2, 1.2, 1), (8.6, .6, 0), (10.4, 1.4, 2), (11.4, .5, 1), (.4, 3.2, 1),
          (11.5, 3.1, 0), (.4, 8.2, 2), (11.6, 7.6, 1)], seed=3)


def pass_glyph(cu, cv):
    if cv <= 2 or cv >= 13:
        return 'h'
    if 7 <= cv <= 8:
        return 'd'
    return '.'


# 거록 길목: a narrow road between two steep hillsides, 안량's camp across it: antler barriers, a gate, a watchtower
PASS = outdoor(12, 8, pass_glyph, props=[
    ('antler_barrier', 8.0, 2.3), ('antler_barrier', 8.0, 5.3), ('gate_u', 8.6, 3.75), ('watchtower', 9.8, 2.2),
    ('banner_yellow', 9.0, 2.6), ('banner_yellow', 9.0, 5.0), ('tent', 10.8, 2.6), ('tent', 10.8, 5.0), ('tent_round', 11.2, 3.8),
], trees=[(u, v, k) for u, v, k in ((1.0, 1.8, 0), (2.4, 1.7, 2), (3.9, 1.8, 1), (5.4, 1.7, 0), (6.8, 1.8, 2),
                                     (1.6, 6.2, 1), (3.0, 6.3, 0), (4.6, 6.2, 2), (6.2, 6.3, 1), (7.6, 6.2, 0))], seed=5)


def scatter(u0, u1, v0, v1, step, seed, kinds, keep_out=lambda u, v: False):
    """Props on a jittered grid over u0..u1 × v0..v1 (an orchard, a wood), skipping where keep_out(u, v)."""
    rnd = random.Random(seed)
    out = []
    v = v0
    while v <= v1:
        u = u0 + (step / 2 if round((v - v0) / step) % 2 else 0)
        while u <= u1:
            pu, pv = u + rnd.uniform(-.3, .3) * step, v + rnd.uniform(-.3, .3) * step
            if not keep_out(pu, pv):
                out.append((rnd.choice(kinds), round(pu, 2), round(pv, 2)))
            u += step
        v += step
    return out


def wall_u(v, u0, u1):
    """A scene city wall along u at v from u0 to u1 (the castle_out maps' swall_u pieces every half cell), laid
    straight through any gate on it: build.seat_gates sets the gates in."""
    return [('swall_u', u0 + k / 2, v) for k in range(int(round((u1 - u0) * 2)) + 1)]


def wall_v(u, v0, v1):
    return [('swall_v', u, v0 + k / 2) for k in range(int(round((v1 - v0) * 2)) + 1)]


def gate(axis, u, v, state='open'):
    """The scene gatehouse at a point on a wall along axis 'u' or 'v'; build.seat_gates sets its slices into the wall."""
    return [(f'sgate_{state}_{axis}', u, v)]


def tall_cliffs(art, tall, kind='tcliff'):
    """Raise a built map's cliff tiles where tall(cu, cv) says so into the tall scene cliffs (tall_cliff.py): each such
    small tile's glyph becomes plain soil and a tcliff_<mask>_<variant> stands on its centre, the mask and the variant
    picked like the engine picks a battle cliff block's (blockSprite / blockVariant), so the tall pieces sort by depth
    exactly as the blocks would and join the low cliff where tall says no. kind 'lcliff': the blocks at their own
    height but sorted half a cell later (a low cliff's lip in front of a gatehouse post, which it must cover)."""
    vx, vy, vw, vh = art['visual']
    rows = [list(r) for r in art['small_tiles']]
    if 'highland' not in art:
        art['highland'] = [''.join('1' if c == 'h' else '0' for c in row) for row in rows]

    def cliff(x, y):
        if not (vx <= x < vx + vw and vy <= y < vy + vh):
            return False  # all four picture edges expose their outward faces
        return rows[y - vy][x - vx] == 'h'

    put = []
    for y in range(vy, vy + vh):
        for x in range(vx, vx + vw):
            if not cliff(x, y) or not tall(x, y):
                continue
            mask = sum(1 << i for i, (dx, dy) in enumerate(((-1, 0), (0, -1), (1, 0), (0, 1))) if not cliff(x + dx, y + dy))
            put.append((x, y, mask))
    for x, y, mask in put:
        rows[y - vy][x - vx] = 'd'
        if mask == 0:
            continue  # the independent raised ceiling covers this interior tile
        late = .5 if kind == 'lcliff' else 0  # lcliff's pivot is 16 px low: its anchor half a cell on keeps it in place
        art['props'].append([f'{kind}_{mask:02d}_{((x * 7 + y * 3) % 3 + 3) % 3}', x / 2 - .25 + late, y / 2 - .25 + late])
    art['small_tiles'] = [''.join(r) for r in rows]
    art['tiles'] = [''.join(rows[2 * j - vy][2 * i - vx] for i in range(art['size'][0])) for j in range(art['size'][1])]
    return art


def _edge_column(name, first):
    """the first or last drawn column of a picture: (screen x of it from the pivot, foot dy, top dy from the pivot)"""
    from PIL import Image
    meta = json.load(open(GFX.parents[2] / 'assets_candidate/tiles/tiles.json'))['sprites'][name]
    a = Image.open(GFX.parents[2] / 'assets_candidate' / meta['file']).convert('RGBA').getchannel('A')
    px, py = meta['pivot']
    cols = [x for x in range(a.width) if any(a.getpixel((x, y)) for y in range(a.height))]
    x = cols[0] if first else cols[-1]
    ys = [y for y in range(a.height) if a.getpixel((x, y))]
    return x - px, max(ys) - py, min(ys) - py


def corner_shadow(art, X, Y, H):
    """Lay the triangular corner shadow (join_overlay.py) of an inner corner whose right face (along u) starts at
    canvas point (X, Y) on its foot, H px high: slice j on that face's foot 16 j px to the right."""
    H = max(8, min(160, int(H) // 8 * 8))
    names = json.load(open(GFX.parents[2] / 'assets_candidate/tiles/tiles.json'))['sprites']
    j = 0
    while f'jshade_{H}_{j}' in names:
        x, y = X + 16 * j, Y + 8 * j
        art['props'].append([f'jshade_{H}_{j}', round((x / 32 + y / 16) / 2, 4), round((y / 16 - x / 32) / 2, 4)])
        j += 1
    return art


def wall_corners(art):
    """Where a scene wall line runs out of a cliff behind it, or two wall lines meet in a back corner, the face on the
    left (along v) throws a corner shadow on the face on the right (along u): a v line whose end toward smaller v
    meets a cliff, a u line whose end toward smaller u meets one, or a u line starting where a v line ends."""
    vx, vy, vw, vh = art['visual']
    cliffs = {(round(2 * (u + .25)), round(2 * (v + .25))) for n, u, v in art['props'] if n.startswith('tcliff_')}

    def cliff(u, v):
        x, y = math.floor(2 * u), math.floor(2 * v)
        return (x, y) in cliffs or 0 <= y - vy < vh and 0 <= x - vx < vw and art['small_tiles'][y - vy][x - vx] == 'h'

    lines = {}
    for n, u, v in art['props']:
        if n.startswith(('swall_', 'sgate_')):
            axis = n[-1] if n.startswith('swall_') else n.split('_')[2]
            lines.setdefault((axis, v if axis == 'u' else u), []).append((u if axis == 'u' else v, n))
    for (axis, line), pieces in lines.items():
        t, n = min(pieces)
        u, v = (t, line) if axis == 'u' else (line, t)
        if axis == 'v' and cliff(u, v - .5):  # the wall on the left, the cliff's face on the right
            dx, foot, top = _edge_column(n, first=False)
            X, Y = 32 * (u - v) + dx + 1, 16 * (u + v) + foot + .5
        elif axis == 'u' and (cliff(u - .5, v) or ('v', u - .5) in lines):  # a cliff or a wall on the left
            dx, foot, top = _edge_column(n, first=True)
            X, Y = 32 * (u - v) + dx, 16 * (u + v) + foot
        else:
            continue
        corner_shadow(art, X, Y, foot - top)
    return art


def walls_on_cliffs(art):
    """A scene wall piece whose half cell lies on cliff tiles on both sides of its line stands on the cliff's top:
    swall_<axis>_up, the same picture raised by the cliff's rise (tall_cliff.py), drawn after the blocks under it."""
    vx, vy, vw, vh = art['visual']
    low = {(round(2 * (u - .25)), round(2 * (v - .25))) for n, u, v in art['props'] if n.startswith('lcliff_')}

    def cliff(x, y):
        return (x, y) in low or 0 <= y - vy < vh and 0 <= x - vx < vw and art['small_tiles'][y - vy][x - vx] == 'h'

    for p in art['props']:
        n, u, v = p
        if n in ('swall_u', 'swall_v'):
            x, y = math.floor(2 * u), math.floor(2 * v)
            sides = ((x - 1, y), (x, y)) if n == 'swall_v' else ((x, y - 1), (x, y))
            if all(cliff(*c) for c in sides):
                p[0] = n + '_up'
    return art


def raise_near_cliffs(art):
    """Raise the near ridge without changing which of its ground faces are open."""
    for p in art['props']:
        if not p[0].startswith('lcliff_'):
            continue
        p[0] = p[0].replace('lcliff_', 'tcliff_', 1)
        p[1] -= .5
        p[2] -= .5
    return art


def cliff_feet(art):
    """A soft shadow band on the ground along the foot of every tall cliff's open lower faces (join_overlay.py)."""
    for n, u, v in list(art['props']):
        if n.startswith('tcliff_'):
            mask = int(n.split('_')[1])
            for bit, face in ((8, 'sw'), (4, 'se')):
                if mask & bit:
                    art['props'].append([f'cfoot_{face}', u, v])
    return art


def small_grid_props(art):
    """Store checkpoint props on the integer small-tile grid, retaining exact pixel placement."""
    def screen_round(n):
        return math.floor(n + .5) if n >= 0 else math.ceil(n - .5)

    placed = []
    for name, u, v in art['props']:
        if name.startswith(('tcliff_', 'cfoot_')):
            U, V = round(2 * u + .5), round(2 * v + .5)
        elif name.startswith('lcliff_'):
            U, V = round(2 * u - .5), round(2 * v - .5)
        else:
            U, V = math.floor(2 * u + .5), math.floor(2 * v + .5)
        dx = screen_round(32 * (u - v)) - 16 * (U - V)
        dy = screen_round(16 * (u + v + 1)) - 8 * (U + V + 1)
        placed.append([name, U, V] if dx == 0 and dy == 0 else [name, U, V, dx, dy])
    art['small_props'] = placed
    del art['props']
    return art


def small_tile_faces(art):
    """Attach checkpoint cliff, wall, gate and join pictures to their ground tile's edge."""
    props, faces = [], []
    sides = {1: 'NW', 2: 'NE', 4: 'SE', 8: 'SW'}
    edge = {'NW': (-8, -4), 'NE': (8, -4), 'SE': (8, 4), 'SW': (-8, 4)}
    for entry in art['small_props']:
        name, U, V, *offset = entry
        dx, dy = offset or (0, 0)
        side = None
        if name.startswith(('tcliff_', 'lcliff_')):
            side = sides.get(int(name.split('_')[1]))
        elif name.startswith('cfoot_'):
            side = name.split('_')[1].upper()
        elif name.startswith('swall_'):
            side = 'SW' if name.split('_')[1] == 'v' else 'SE'
        elif name.startswith('sgate_'):
            side = 'SW' if name.split('_')[2] == 'v' else 'SE'
        elif name.startswith('jshade_'):
            V -= 1  # this picture covers the far cliff's SW face, one tile behind the wall
            side = 'SW'
        if side is None:
            props.append(entry)
            continue
        x, y = 16 * (entry[1] - entry[2]) + dx, 8 * (entry[1] + entry[2] + 1) + dy
        ex, ey = edge[side]
        face_dx, face_dy = x - (16 * (U - V) + ex), y - (8 * (U + V + 1) + ey)
        faces.append([name, U, V, side] if face_dx == face_dy == 0 else [name, U, V, side, face_dx, face_dy])
    art['small_props'] = props
    art['small_faces'] = faces
    return art


def avenue(axis, cells):
    """A paved way: the castle_out maps' avenue_u / avenue_v ground pieces on half cells."""
    return [(f'avenue_{axis}', u, v) for u, v in cells]


def blobs(seed, n, u0, u1, v0, v1, r0=.6, r1=1.3):
    """A few soft round patches (centre u, v, radius in small tiles) for waste ground among grass."""
    rnd = random.Random(seed)
    return [(rnd.uniform(u0, u1) * 2, rnd.uniform(v0, v1) * 2, rnd.uniform(r0, r1) * 2) for _ in range(n)]


def in_blob(cu, cv, bl):
    return any((cu - a) ** 2 + (cv - b) ** 2 < r * r for a, b, r in bl)


# 복숭아 숲: a grove in full blossom behind Zhang Fei's house, the clearing in the middle (round (4.6, 3.6), about
# 2.5 cells across) with the wine table, a mat and the incense for the oath; the trees run on past the screen,
# petals on the grass. The trees stand in the playing area, so only the clearing is kept open and in view
GROVE_CLEARING = [(4.6 + r * math.cos(a), 3.6 + r * math.sin(a)) for r in (0, .8, 1.6) for a in
                  [k * math.pi / 4 for k in range(8)]]
PEACH_GROVE = outdoor(10, 8, lambda cu, cv: 'd' if (cu - 9.2)**2 + (cv - 7.2)**2 < 10 else '.', props=[
    ('table_wine', 4.6, 3.6), ('straw_mat', 4.6, 4.3), ('wine_jar', 5.6, 3.0), ('wine_jar', 5.85, 3.25), ('incense', 3.6, 2.9),
    ('water_jar', 5.9, 2.7),
    *[('fallen_petals', u, v) for u, v in ((3.6, 4.4), (5.6, 2.2), (2.4, 2.8), (6.6, 4.6), (4.2, 1.6), (3.0, 5.6), (7.0, 2.0),
                                           (1.4, 4.6), (5.0, 6.2), (8.0, 4.0))],
], trees=[(u, v, 'peach') for _, u, v in scatter(-6, 16, -6, 14, 2.1, 7, ['p'],
                                                  keep_out=lambda u, v: (u - 4.6)**2 + (v - 3.6)**2 < 2.9**2)],
    seed=9, grass=.08, margin=7)

TOWN_WASTE = blobs(21, 9, 0, 10, 0, 9)
TOWN_M = 6  # cells of picture round the town; the walls run on to its edge


def town_glyph(cu, cv):
    if cu < -1 or cv < -1:
        return '.'  # outside the walls
    if 6 <= cu <= 11 and 3 <= cv <= 5:
        return 'i'  # the paved yard before the hall
    if 8 <= cu <= 10 and cv > 5 or 6 <= cv <= 7:
        return 'd'  # the street down from the hall and the lane across to the gate
    return 'd' if in_blob(cu, cv, TOWN_WASTE) else '.'


# 영천 성 안 민가 (10×9): inside the walls (the scene city wall along the back two sides, the gate at the left
# end of the cross lane, (−0.75, 3.25)), the small government hall at the back with a paved yard, the paved main
# street (u≈4.5) down from it, the dirt lane across (v≈3.2), houses in the four blocks, waste ground and grass
# between, a well; the burnt variant swaps the houses for burnt shells
CASTLE_TOWN = outdoor(10, 9, town_glyph, props=[
    *wall_u(-.75, -.25, 10 + TOWN_M), *wall_v(-.75, -.25, 9 + TOWN_M), *gate('v', -.75, 3.25),
    ('hall_u', 4.6, 1.2), ('banner', 3.4, 2.4), ('banner', 5.8, 2.4), ('lantern', 3.7, 2.6), ('lantern', 5.5, 2.6),
    *avenue('v', [(u, v / 2 + .25) for u in (4.25, 4.75) for v in range(5, 22)]),
    ('house_tile', 2.0, 1.2), ('house_tile', 7.6, 1.2), ('house_thatch_x', 2.2, 5.2), ('house_thatch', 1.6, 7.6),
    ('house_thatch', 7.6, 5.0), ('house_tile', 7.6, 7.6),
    ('well', 6.3, 6.3), ('water_jar', 3.2, 4.4), ('water_bucket', 3.4, 4.8), ('grain_sack', 6.2, 4.3), ('grain_sack', 6.4, 4.6),
    ('wine_jar', 3.0, 2.0), ('notice_post', 6.4, 4.0), ('bench', 3.1, 6.6), ('straw_mat', 8.8, 6.6),
    ('flagstones_0', 3.2, 5.6), ('flagstones_1', 6.0, 7.4), ('flagstones_0', 1.0, 3.4),
], trees=[(-2.2, 1.0, 'pine'), (-2.4, 6.0, 'broad'), (2.0, -2.4, 'pine'), (6.5, -2.2, 'broad'), (9.5, -2.0, 'old'),
          (12.0, 4.0, 'broad'), (11.5, 8.0, 'pine'), (4.0, 11.5, 'broad'), (8.5, 11.0, 'old'), (-2.0, 9.5, 'pine')],
    seed=11, grass=.06, margin=TOWN_M)

JAIL_WASTE = blobs(23, 10, -2, 10, 1, 8)
JAIL_M = 5


def street_glyph(cu, cv):
    if cv <= 0:
        return 'i'  # the prison yard behind the wall
    if 6 <= cu <= 13 and cv <= 3:
        return 'i'  # the paved square before the prison gate
    if 8 <= cu <= 11 or 5 <= cu <= 7 and cv <= 15:
        return 'd'  # the street down from the gate, the trodden ground before the houses
    return 'd' if in_blob(cu, cv, JAIL_WASTE) else '.'


# 낙양 정위의 옥 앞 거리 (10×7): the prison's wall (scene city wall, v≈0.25) with its closed gate (4.75, 0.25), a
# paved square before it, the street running down from the gate (u≈4.75, paved); the soup shop is the first house
# down its left side (2.6, 2.7, its door to the street) with its stove, bench, table and jar stall out front, two more
# houses below it; across the street a well, the notice post, waste ground and grass
JAIL_STREET = outdoor(10, 7, street_glyph, props=[
    *wall_u(.25, -JAIL_M, 10 + JAIL_M), *gate('u', 4.75, .25, 'closed'),
    ('lantern', 3.4, 1.0), ('lantern', 6.1, 1.0), ('banner', 3.2, .6), ('banner', 6.3, .6),
    *avenue('v', [(u, v / 2 + .25) for u in (4.5, 5.0) for v in range(3, 18)]),
    ('house_tile_x', 2.4, 2.7), ('soup_stove', 3.5, 2.1), ('water_jar', 3.2, 1.6), ('wine_jar', 3.5, 3.5),
    ('bench', 3.7, 3.0), ('table_wine', 3.8, 2.55), ('market_stall_jar', 3.4, 4.2),
    ('house_thatch_x', 2.4, 5.0), ('house_tile_x', 2.4, 7.0), ('flagstones_1', 3.6, 5.6),
    ('notice_post', 6.4, 1.8), ('well', 7.6, 4.3), ('house_tile', 8.4, 6.0), ('grain_sack', 7.2, 5.4), ('water_bucket', 6.6, 3.9),
    ('lantern', 6.0, 4.4), ('flagstones_0', 6.4, 2.6),
], trees=[(8.6, 3.0, 'broad'), (9.8, 1.5, 'pine'), (0.0, 9.5, 'broad'), (10.5, 8.5, 'old'), (-1.0, 4.0, 'broad')],
    seed=13, grass=.08, margin=JAIL_M)


def market_glyph(cu, cv):
    return 'd' if 6 <= cv <= 9 else '.'


# 장터 (12×8): a country market street along u (v≈3~4.5), stalls along the far side facing it, a wine shop and
# houses behind, sacks and jars on the near side, an open patch at the lower right (u≈9, v≈6.2) for the bonfire
MARKET = outdoor(12, 8, market_glyph, props=[
    ('market_stall', 2.0, 2.4), ('market_stall_cloth', 4.0, 2.4), ('market_stall_jar', 6.2, 2.4), ('market_stall', 8.6, 2.4),
    ('tavern', 1.0, .6), ('house_thatch', 5.2, .5), ('house_tile', 9.8, .5), ('notice_post', 10.6, 2.6),
    ('grain_sack', 2.4, 5.6), ('grain_sack', 2.7, 5.8), ('grain_sack', 2.3, 6.0), ('wine_jar', 5.0, 5.6), ('wine_jar', 5.3, 5.8),
    ('bench', 6.8, 5.6), ('water_bucket', 4.6, 5.9),
    ('campfire', 9.2, 6.2), ('straw_mat', 8.2, 6.0), ('straw_mat', 10.0, 7.0), ('cauldron', 9.9, 5.6),
], trees=[(-1.5, 4.0, 'broad'), (-2.0, -1.0, 'pine'), (12.5, -1.5, 'old'), (3.0, -1.8, 'broad'), (8.0, -2.0, 'pine'),
          (13.8, 3.5, 'broad'), (14.0, 7.5, 'pine'), (3.5, 10.5, 'broad'), (8.0, 10.5, 'old'), (-1.5, 8.5, 'pine')],
    seed=15, grass=.1, margin=4)


CHECK_WASTE = blobs(25, 8, -2, 14, 1.6, 6.4, .4, .9)
CHECK_M = 9


def checkpoint_glyph(cu, cv):
    if cv <= 1 or cv >= 15:  # the hillsides; the gate's near post stands behind the near one's lip
        return 'h'
    if 8 <= cv <= 9:
        return 'd'
    return 'd' if in_blob(cu, cv, CHECK_WASTE) else '.'


# 관문 (12×8): a Han checkpoint across the road (v≈4.5) through a defile: the far hillside (v < 1) is the battle cliff
# raised into a tall scene cliff (tall_cliffs, its top a little above the gatehouse's roof); the near ridge is tall too
# and naturally hides scenery behind it; the scene city wall along v at u≈8.25 ends at the near hillside,
# with its gate on the road (8.25, 4.25; open here, closed in S_checkpoint_closed). The wall's face
# looks east, toward the camera: travellers going west come up the road from the east (u 9.5~12) to the officer at
# the gate (≈9.3, 4.0), the notice post with the wanted face beside the way; the garrison's watchtower and tents stand
# behind the wall
CHECKPOINT = outdoor(12, 8, checkpoint_glyph, props=[
    *wall_v(8.25, .75, 6.75), *gate('v', 8.25, 4.25),
    ('watchtower', 7.0, 2.1), ('banner', 9.0, 1.8), ('banner', 9.0, 6.2), ('command_flag', 6.6, 5.6),
    ('notice_post', 10.2, 5.4), ('lantern', 9.2, 2.6), ('lantern', 9.2, 5.4), ('tent_round', 5.8, 5.4), ('tent', 6.0, 2.6),
    ('weapon_rack', 7.4, 5.4), ('milestone', 11.6, 3.0), ('stree_bush_0', 11.2, 6.3), ('stree_bush_1', 2.4, 5.8),
], trees=[(u, v, k) for u, v, k in ((-1.5, 2.6, 'broad'), (-1.6, 5.6, 'pine'), (13.5, 3.8, 'broad'))], seed=17, margin=CHECK_M)
TALL = {'S_checkpoint': lambda x, y: y <= 1}  # small tiles of each map's cliff raised tall
LATE = {'S_checkpoint': lambda x, y: y >= 15}  # identify the near ridge before replacing its low sorting blocks
CORNERS = {'S_castle_town'}  # maps whose scene wall corners get corner shadows

# how build.py plants each composed map's trees (trees.plant): the grove keeps only its clearing open
PLANT = {'S_peach_grove': dict(protect=False, extra=GROVE_CLEARING, fill_bush=False),
         'S_bridge': dict(protect=False), 'S_pass': dict(protect=False),
         **{m: dict(protect=False, extra=build.grid(0, 12, 2.2, 5.6)) for m in ('S_checkpoint', 'S_checkpoint_closed')}}

OUTDOOR = [(BRIDGE, 'S_bridge', '계교 강과 다리'), (PASS, 'S_pass', '거록 협곡 길목'),
           (PEACH_GROVE, 'S_peach_grove', '복숭아 숲'), (CASTLE_TOWN, 'S_castle_town', '성 안 민가'),
           (JAIL_STREET, 'S_jail_street', '옥문 앞 거리와 국밥집'), (MARKET, 'S_market', '장터'),
           (CHECKPOINT, 'S_checkpoint', '관문')]

# variants of viewer-made maps: (new id, name, base id, renames, added props)
VARIANTS = [
    ('S_turban_camp', '황건 진영', 'S_army_camp', {'banner': 'banner_yellow', 'command_flag': 'command_flag_yellow'},
     [('wine_jar', 2.0, 2.6), ('wine_jar', 2.2, 2.9), ('wine_jar', 3.1, 1.7), ('grain_sack', 1.7, 3.4), ('grain_sack', 1.9, 3.7),
      ('grain_sack', 3.3, 3.8), ('table_wine', 2.95, 2.55)]),
    ('S_rampart_siege', '포위된 성루', 'S_rampart', {},
     [('grain_sack', -1.8, 2.3), ('grain_sack', -1.4, 2.6), ('grain_sack', -2.1, 2.7), ('brazier', -.9, 1.0), ('brazier', 2.6, 1.0),
      ('banner', -2.6, .3), ('banner', 2.4, .3), ('lantern', -5.4, .4), ('straw_mat', -.6, 2.7), ('bloody_cloth', -.1, 3.2),
      ('medicine_pot', -2.6, 3.6)]),
    # a camp that has gone hungry or come back beaten: wounded on mats, a cold cauldron, broken gear
    ('S_army_camp_hungry', '굶주린 진영', 'S_army_camp', {},
     [('straw_mat', 1.0, 4.4), ('straw_mat', 4.0, 4.6), ('straw_mat', .2, 5.6), ('cauldron', 1.7, 5.1), ('broken_spear', 3.4, 5.3),
      ('bloody_cloth', 1.4, 3.9), ('fallen_flag', 4.9, 5.6), ('medicine_pot', .4, 3.3)]),
    # the town after the fire: the houses burnt out, ash in the street, buckets and jars left where the chain stopped
    ('S_castle_town_burnt', '불탄 성 안 민가', 'S_castle_town', {n: 'house_burnt' for n in ('house_tile', 'house_thatch', 'house_tile_x', 'house_thatch_x')},
     [('ash_2_1', 2.6, 3.8), ('ash_1_0', 1.2, 6.4), ('ash_3_2', 8.4, 6.2), ('ash_2_0', 8.2, 2.8), ('ash_1_2', 3.0, 8.4),
      ('water_bucket', 3.6, 5.6), ('water_bucket', 5.9, 6.0), ('water_jar', 6.2, 5.6), ('broken_spear', 4.0, 7.6)]),
    # the checkpoint with its gate shut
    ('S_checkpoint_closed', '관문(문 닫힘)', 'S_checkpoint', {f'sgate_open_v_{k}': f'sgate_closed_v_{k}' for k in range(16)}, []),
]


def main():
    index_path = GFX / 'index.json'
    index = json.load(open(index_path))
    known = set(index['props']['sprites']) | set(index['extra']['sprites'])
    warnings = []
    out = []
    for scene, map_id, name in ROOMS + OUTDOOR:
        art = build.build(scene, 0, map_id, name, known, warnings.append, **PLANT.get(map_id, {}))
        if map_id in TALL:
            art = cliff_feet(wall_corners(tall_cliffs(art, TALL[map_id])))
            if map_id in LATE:
                art = raise_near_cliffs(walls_on_cliffs(tall_cliffs(art, LATE[map_id], 'lcliff')))
        elif map_id in CORNERS:
            art = wall_corners(art)
        out.append((map_id, art))
    for map_id, name, base, renames, adds in VARIANTS:
        made = dict(out)  # a variant of a map made above starts from that, not from last run's file
        art = copy.deepcopy(made[base] if base in made else json.load(open(GFX / 'maps' / f'{base}.json')))
        art['id'], art['name'] = map_id, name
        for p in art['props']:
            p[0] = renames.get(p[0], p[0])
        art['props'] += [[n, u, v] for n, u, v in adds]
        for p in art['props']:
            if p[0] not in known:
                warnings.append(f'{map_id}: no picture {p[0]}')
        out.append((map_id, art))
    for map_id, art in out:
        path = GFX / 'maps' / f'{map_id}.json'
        if path.exists() and json.load(open(path)).get('format') == 2:
            print(map_id, 'is format 2 (scene_v2.py): left as it is')  # variants of it are still made from it above
            continue
        if map_id in ('S_checkpoint', 'S_checkpoint_closed'):
            art = small_tile_faces(small_grid_props(art))
            art['small_snap'] = True
            art['cliff_repeat'] = False
        json.dump(art, open(path, 'w'), ensure_ascii=False, separators=(',', ':'))
        rel = f'maps/{map_id}.json'
        if rel not in index['maps']:
            index['maps'].append(rel)
        print(map_id, art['size'], 'visual', art['visual'], len(art.get('props', art.get('small_props', []))), 'props')
    json.dump(index, open(index_path, 'w'), ensure_ascii=False, indent=1)
    open(index_path, 'a').write('\n')
    for w in sorted(set(warnings)):
        print('warning:', w)


if __name__ == '__main__':
    main()
