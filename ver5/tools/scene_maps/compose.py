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
import random
from pathlib import Path

import build

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'


def room(W, H, floor, wall, extra_walls=(), props=()):
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
        put('pillar_red', u, v)
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

ROOMS = [(COUNCIL_BIG, 'S_council', '맹주의 막사(넓은 군막)'), (HALL, 'S_hall', '궁궐 옥좌 앞(넓은 전각)'),
         (TENT_MID, 'S_tent_mid', '장수의 군막'), (TENT_LUX, 'S_tent_lux', '원술의 막사')]


def outdoor(W, H, glyph_at, props=(), trees=(), seed=1, grass=.12):
    """An outdoor map of W x H game cells for build.build: glyph_at(cu, cv) gives each small tile ('.' grass, 'd'
    soil, '~' water, 'b' bridge, 'h' cliff); trees are (game u, v, kind); grass tufts are scattered on grass."""
    rnd = random.Random(seed)
    visible, draws, tufts = [], [], []
    for cv in range(2 * H):
        for cu in range(2 * W):
            ch = glyph_at(cu, cv)
            visible.append((cu, cv, ch))
            if ch == '.' and rnd.random() < grass:
                tufts.append({'n': f'ts:grass_{rnd.randrange(4)}', 'x': 16 * (cu - cv) + rnd.randrange(-6, 7),
                              'y': 8 * (cu + cv + 1) + rnd.randrange(-3, 4)})
    for u, v, kind in trees:
        draws.append({'n': f'ts:tree_{kind}', 'x': 32 * (u - v), 'y': 16 * (u + v + 1)})
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

OUTDOOR = [(BRIDGE, 'S_bridge', '계교 강과 다리'), (PASS, 'S_pass', '거록 협곡 길목')]

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
]


def main():
    index_path = GFX / 'index.json'
    index = json.load(open(index_path))
    known = set(index['props']['sprites']) | set(index['extra']['sprites'])
    warnings = []
    out = []
    for scene, map_id, name in ROOMS + OUTDOOR:
        out.append((map_id, build.build(scene, 0, map_id, name, known, warnings.append)))
    for map_id, name, base, renames, adds in VARIANTS:
        art = copy.deepcopy(json.load(open(GFX / 'maps' / f'{base}.json')))
        art['id'], art['name'] = map_id, name
        for p in art['props']:
            p[0] = renames.get(p[0], p[0])
        art['props'] += [[n, u, v] for n, u, v in adds]
        for p in art['props']:
            if p[0] not in known:
                warnings.append(f'{map_id}: no picture {p[0]}')
        out.append((map_id, art))
    for map_id, art in out:
        json.dump(art, open(GFX / 'maps' / f'{map_id}.json', 'w'), ensure_ascii=False, separators=(',', ':'))
        rel = f'maps/{map_id}.json'
        if rel not in index['maps']:
            index['maps'].append(rel)
        print(map_id, art['size'], 'visual', art['visual'], len(art['props']), 'props')
    json.dump(index, open(index_path, 'w'), ensure_ascii=False, indent=1)
    open(index_path, 'a').write('\n')
    for w in sorted(set(warnings)):
        print('warning:', w)


if __name__ == '__main__':
    main()
