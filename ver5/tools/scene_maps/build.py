#!/usr/bin/env python3
"""Make the scenario scene maps of the game from the candidate scenes.

  node ver5/tools/scene_maps/export.mjs > /tmp/scenes_draws.json
  python3 ver5/tools/scene_maps/build.py /tmp/scenes_draws.json

export.mjs lists, for each scene of assets_candidate/tools/viewer_template.html, every sprite it draws and where
(in the viewer's pixels). This script turns that into the maps the game reads: assets/graphics/maps/<id>.json and
their entries in assets/graphics/index.json under "maps".

Scale. The viewer draws everything at 1 pixel per pixel on cells of 32x16. The game draws a battle map's lattice at
that same size for its half cells (small tiles) and its game cells are twice as wide. So a scene map is a "native"
map like a battle map: the viewer's cell is the game's small tile (floors, walls and cliffs are small-tile glyphs),
props and decorations keep their pixels, and a game cell (u, v) is the viewer's cells (2u, 2v)..(2u+1, 2v+1).
Props are placed by game cell with fractions: viewer pixel (x, y) is the game point
((x/32 + y/16 - 1) / 2, (y/16 - 1 - x/32) / 2).

What is left out: units (the scenario's cast places them), flames and smoke (the "fire" effect of a beat), the sky,
the night and dusk tints and the lights, which the game's scene renderer does not have.
"""
import json
import math
import sys
from pathlib import Path

import trees

ROOT = Path(__file__).resolve().parents[3]
GFX = ROOT / 'ver5/assets/graphics'

# (scene, state, map id, name)
EXPORT = [
    ('village', 0, 'S_vil_peace', '마을(울타리)'),
    ('village', 2, 'S_vil_burnt', '불탄 마을'),
    ('tent', 0, 'S_tent', '군막 내부'),
    ('road', 0, 'S_road', '길'),
    ('hill', 0, 'S_hill', '언덕'),
    ('camp', 0, 'S_army_camp', '진영'),
    ('castleOut', 0, 'S_castle_out', '성 밖(성문 닫힘)'),
    ('castleOut', 1, 'S_castle_out_open', '성 밖(성문 열림)'),
    ('courtyard', 0, 'S_courtyard', '성 안'),
    ('rampart', 0, 'S_rampart', '성루'),
    ('office', 0, 'S_office', '관아'),
    ('palace', 0, 'S_palace', '궁궐'),
    ('house', 0, 'S_house', '동승의 집'),
    ('garden', 0, 'S_garden', '조조의 후원'),
]
# the viewer's tile trees and tree props → the scene tree species planted in their place (trees.py)
TREE_KIND = {'tree_0': 'broad', 'tree_1': 'pine', 'tree_2': 'broad'}
TREE_PROPS = (('forest_', 'pine'), ('burnt_forest_', 'dead'), ('burnt_oak_', 'dead'), ('plum_tree', 'plum'), ('peach_tree', 'peach'))
# maps whose viewer houses are the battle map's village tile (a cluster of tiny round huts): the scene houses instead
HOUSES = {'S_vil_peace': ('hut', [('house_thatch', 1.3, .55), ('house_thatch_x', .55, 1.6)]),
          'S_vil_burnt': ('burnt_hut', [('house_burnt', 1.3, .55), ('house_burnt', .55, 1.6)])}


def grid(u0, u1, v0, v1):
    return [(u0 + i / 2, v0 + j / 2) for i in range(int((u1 - u0) * 2) + 1) for j in range(int((v1 - v0) * 2) + 1)]


# where people stand on the viewer maps the story has not staged yet (trees keep clear of it and do not hide it); the
# staged maps' spots come from their scenes (trees.scene_spots). The playing area is not kept clear as a whole:
# these maps' trees stand in it
TAVERN_FRONT = grid(3.0, 4.0, .4, 1.2)  # the village's wine shop stays in view
OPEN = {'S_vil_peace': grid(1, 5, 2.5, 5.5) + TAVERN_FRONT, 'S_vil_burnt': grid(1, 5, 2.5, 5.5) + TAVERN_FRONT, 'S_castle_out': grid(2.4, 3.6, 2.2, 7),
        'S_castle_out_open': grid(2.4, 3.6, 2.2, 7), 'S_garden': grid(2.0, 3.4, 3.0, 4.2)}
SURFACE = {'.': 1, 'd': 2, '~': 0, 'c': 1, 'h': 2, 'i': 2, 'b': 0, 'v': 2, 'k': 2, 'f': 1, 'o': 1, 'g': 1}


def gate_slices():
    """The scene gatehouse's upright slices (assets_candidate/tools/gen_3d_castle.py), by (state, axis): (name,
    cells along the wall from the gate's point)."""
    meta = json.load(open(ROOT / 'assets_candidate/tiles/tiles.json'))['sprites']
    out = {}
    for n, e in meta.items():
        if 'gate_offset' in e:
            _, state, axis, _ = n.split('_')
            out.setdefault((state, axis), []).append((n, e['gate_offset'], e['size'][0] < 16))
    return out


def seat_gates(props):
    """Set the gates into the scene walls. A wall line is laid half a cell at a time from its first piece to its last;
    a gate (sgate_<state>_<axis> at a point on the line, or the viewer's whole-gate halves) is a run of half-cell
    slices, each taking the place of the wall piece where it stands, so the gate sorts by depth with the wall, the
    people and the props exactly as the wall does, and a man walking through the passage stands between its back
    half and its front half. The gate snaps to the line's half cells."""
    slices = gate_slices()
    gates, rest, lines = [], [], {}
    for p in props:
        name, u, v = p
        parts = name.split('_')
        if name.startswith('swall_'):
            axis = name[-1]
            lines.setdefault((axis, v if axis == 'u' else u), set()).add(u if axis == 'u' else v)
        elif name.startswith('sgate_') and len(parts) == 3 or name.startswith('sgate_') and parts[2] == 'first':
            gates.append((parts[1], parts[-1], u, v))
        elif not name.startswith('sgate_'):
            rest.append(p)
    out = rest
    for (axis, line), ts in lines.items():
        ts = sorted(ts)
        on = [(st, u if axis == 'u' else v) for st, gaxis, u, v in gates if gaxis == axis and (v if axis == 'u' else u) == line]
        # the wall's half cells line up on the (first) gate, so the cut down the middle of its arch falls between two
        # of them and the gate stays where the map put it (on its road, its bridge)
        phase = on[0][1] % .5 if on else ts[0] % .5
        t0 = math.floor((ts[0] - phase) * 2 + 1e-6) / 2 + phase
        t1 = math.ceil((ts[-1] - phase) * 2 - 1e-6) / 2 + phase
        laid = {round(t0 + k * .5, 3): [f'swall_{axis}'] for k in range(round((t1 - t0) * 2) + 1)}
        for state, a in on:
            a = math.floor((a - phase) * 2 + .5) / 2 + phase
            for n, off, edge in slices[(state, axis)]:
                t = round(a + off, 3)
                # a slice fills its half cell and takes the wall piece's place; the narrow edge of a post spilling into
                # the next half cell is drawn just after the wall piece standing there
                laid[t] = laid.get(t, [])[:1] + [n] if edge else [n]
        for t, names in sorted(laid.items()):
            for n in names:
                out.append([n, t, line] if axis == 'u' else [n, line, t])
    return out


def floor_half(x):
    return x // 2


def ground_layers(dry, soil):
    layers = [8]
    if soil != 15:
        layers = [23]
        if dry != 15:
            layers = [0]
            if dry != 0:
                layers.append(23 + dry)
        if soil != 0:
            layers.append(8 + soil)
    return layers


def cell_of(x, y):
    """The viewer cell whose centre is the pixel (x, y)."""
    return round((x / 16 + y / 8 - 1) / 2), round((y / 8 - 1 - x / 16) / 2)


def game_point(x, y):
    return round((x / 32 + y / 16 - 1) / 2, 3), round((y / 16 - 1 - x / 32) / 2, 3)


def build(scene, state, map_id, name, props_known, warn, **plant_opts):
    st = scene['states'][state]
    w, h = scene['w'], scene['h']
    glyph = {}
    for u, v, ch in st['visible']:
        glyph[(u, v)] = ch
    props, decorations, tree_cands = [], [], []
    block_cells = set()
    for d in st['draws']:
        kind, _, n = d['n'].partition(':')
        x, y = d['x'], d['y']
        if kind == 'st' and n == 'floor':
            glyph[cell_of(x, y)] = 'i'
            block_cells.add(cell_of(x, y))
        elif kind == 'st' and n.startswith('wall_') or kind == 'c' and n.startswith('wall_clean_'):
            glyph[cell_of(x, y)] = 'c'
            block_cells.add(cell_of(x, y))
        elif kind == 'st' and n.startswith('cliff_'):
            glyph[cell_of(x, y)] = 'h'
            block_cells.add(cell_of(x, y))
        elif kind == 'st' and n == 'village':
            props.append(['hut', *game_point(x, y)])
        elif kind == 'ts' and n in TREE_KIND:
            u, v = game_point(x, y)
            tree_cands.append((u, v, TREE_KIND[n] if (int(x) * 7 + int(y) * 3) % 5 else 'old'))
        elif kind == 'tree':
            tree_cands.append((*game_point(x, y), n))
        elif kind in ('pr', 'c') and n.startswith(tuple(p for p, _ in TREE_PROPS)):
            u, v = game_point(x, y)
            sp = next(k for p, k in TREE_PROPS if n.startswith(p))
            tree_cands += [(u - .35, v + .2, sp), (u + .35, v - .2, sp)] if n.startswith('forest_') else [(u, v, sp)]
        elif kind == 'ts' and n.startswith(('tree_', 'rock_')):
            decorations.append([round(x / 2), round(y / 2), n, (len(decorations) * 37) % 60])
        elif kind in ('pr', 'c'):
            if n not in props_known and not n.startswith('sgate_'):  # a gate is set into its wall by seat_gates
                warn(f'{map_id}: no picture {n}')
                continue
            props.append([n, *game_point(x, y)])
        else:
            warn(f'{map_id}: left out {d["n"]}')
    for g in st['grass']:
        n = g['n'][3:]
        u2, v2 = (g['x'] / 16 + g['y'] / 8) / 2, (g['y'] / 8 - g['x'] / 16) / 2
        if (math.floor(u2), math.floor(v2)) in block_cells:
            continue  # the viewer paves over the grass; a decoration would stand on the floor
        decorations.append([round(g['x'] / 2), round(g['y'] / 2), n, (len(decorations) * 37) % 60])

    # the part of the picture that is drawn, in viewer cells
    if scene['noGround']:
        cells = list(block_cells) or [(u, v) for v in range(h) for u in range(w)]
    else:
        cells = [(u, v) for u, v, _ in st['visible']]
    vx, vy = min(u for u, v in cells), min(v for u, v in cells)
    vw, vh = max(u for u, v in cells) - vx + 1, max(v for u, v in cells) - vy + 1
    if vx % 2:  # a game cell is two viewer cells wide: keep the lattice aligned
        vx, vw = vx - 1, vw + 1
    if vy % 2:
        vy, vh = vy - 1, vh + 1
    vw += vw % 2
    vh += vh % 2
    # every cell of the lattice has a glyph; the floors of a roofed scene stand on plain ground
    for v in range(vy, vy + vh):
        for u in range(vx, vx + vw):
            glyph.setdefault((u, v), '.')

    for n, u, v in [p[:3] for p in props if p[0] in ('bridge_u', 'bridge_v')]:
        for j in range(2):  # a bridge's water cells say so, so the game draws the bridge over the water, not under it
            for i in range(2):
                if glyph.get((math.floor(2 * u) + i, math.floor(2 * v) + j)) == '~':
                    glyph[(math.floor(2 * u) + i, math.floor(2 * v) + j)] = 'b'
    small = [''.join(glyph[(u, v)] for u in range(vx, vx + vw)) for v in range(vy, vy + vh)]
    W, H = math.ceil(w / 2), math.ceil(h / 2)
    tiles = [''.join(glyph.get((2 * i, 2 * j), '.') for i in range(W)) for j in range(H)]

    def surface(u, v):
        return SURFACE.get(glyph.get((u, v), '.'), 1)

    def vertex(sx, sy, kind):
        for y in range(floor_half(sy - 1), floor_half(sy) + 1):
            for x in range(floor_half(sx - 1), floor_half(sx) + 1):
                if vx <= x < vx + vw and vy <= y < vy + vh and surface(x, y) >= kind:
                    return True
        return False

    def mask(x, y, kind):
        n = 0
        for bit, (dx, dy) in enumerate([(0, 0), (1, 0), (1, 1), (0, 1)]):
            if vertex(x + dx, y + dy, kind):
                n |= 1 << bit
        return n

    ground = []
    for y in range(vy, vy + vh):
        for x in range(vx, vx + vw):
            for j in range(2):
                for i in range(2):
                    dry, soil = mask(x * 2 + i, y * 2 + j, 1), mask(x * 2 + i, y * 2 + j, 2)
                    ground.append([(x - y) * 8 + (i - j) * 4, (x + y) * 4 + (i + j) * 2, ground_layers(dry, soil)])
    def wet(u, v):  # a tree by water grows as a willow
        return any(glyph.get((round(2 * u) + i, round(2 * v) + j)) == '~' for i in range(-3, 4) for j in range(-3, 4))

    tree_cands = [(u, v, 'willow' if k in ('broad', 'old') and wet(u, v) else k) for u, v, k in tree_cands]
    props += [[n, u, v] for n, u, v in trees.plant(tree_cands, map_id, W, H, **plant_opts)]
    props = seat_gates(props)
    if map_id in HOUSES:
        old_name, houses = HOUSES[map_id]
        props = [p for p in props if p[0] != old_name] + [list(h) for h in houses]
    bounds = [(vx - (vy + vh)) * 8 - 24, (vx + vy) * 4 - 48, ((vx + vw) - vy) * 8 + 24, (vx + vy + vw + vh) * 4 + 24]
    return {'id': map_id, 'name': name, 'size': [W, H], 'tiles': tiles, 'bounds': bounds, 'ground': ground,
            'decorations': decorations, 'props': props, 'deformed': False, 'native': True, 'scene': True,
            'visual': [vx, vy, vw, vh], 'small_tiles': small}


def main():
    scenes = json.load(open(sys.argv[1]))
    index_path = GFX / 'index.json'
    index = json.load(open(index_path))
    known = set(index['props']['sprites']) | set(index['extra']['sprites'])
    warnings = []
    for key, state, map_id, name in EXPORT:
        art = build(scenes[key], state, map_id, name, known, warnings.append, protect=False, extra=OPEN.get(map_id, ()))
        path = GFX / 'maps' / f'{map_id}.json'
        json.dump(art, open(path, 'w'), ensure_ascii=False, separators=(',', ':'))
        rel = f'maps/{map_id}.json'
        if rel not in index['maps']:
            index['maps'].append(rel)
        print(map_id, art['size'], 'visual', art['visual'], len(art['props']), 'props', len(art['decorations']), 'decorations')
    json.dump(index, open(index_path, 'w'), ensure_ascii=False, indent=1)
    open(index_path, 'a').write('\n')
    for w in sorted(set(warnings)):
        print('warning:', w)


if __name__ == '__main__':
    main()
