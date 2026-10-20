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
SURFACE = {'.': 1, 'd': 2, '~': 0, 'c': 1, 'h': 2, 'i': 2, 'b': 0, 'v': 2, 'k': 2, 'f': 1, 'o': 1, 'g': 1}


def floor_half(x):
    return (x - 1) // 2 if x < 0 else x // 2


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


def build(scene, state, map_id, name, props_known, warn):
    st = scene['states'][state]
    w, h = scene['w'], scene['h']
    glyph = {}
    for u, v, ch in st['visible']:
        glyph[(u, v)] = ch
    props, decorations = [], []
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
        elif kind == 'ts' and n.startswith(('tree_', 'rock_')):
            decorations.append([round(x / 2), round(y / 2), n, (len(decorations) * 37) % 60])
        elif kind in ('pr', 'c'):
            if n not in props_known:
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
        art = build(scenes[key], state, map_id, name, known, warnings.append)
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
