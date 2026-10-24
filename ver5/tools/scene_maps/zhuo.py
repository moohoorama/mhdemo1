#!/usr/bin/env python3
"""Make the maps of the 탁군 walking demo (assets/towns/zhuo.json): the town inside the walls and the rooms behind
its doors. Uses compose.py's room and outdoor builders and writes only these maps, so the other scene maps are left
as they are.

  python3 ver5/tools/scene_maps/zhuo.py

Coordinates are game cells: u runs down to the right on screen, v down to the left. A house's door is on its lower
left face (house_*: +v) or lower right face (house_*_x: +u), so every enterable house turns its door to a street.
"""
import json

import build
from compose import GFX, avenue, blobs, gate, in_blob, outdoor, room, wall_u, wall_v

W, H, M = 12, 10, 6
WASTE = blobs(31, 10, 0, W, 0, H)


def zhuo_glyph(cu, cv):
    u, v = cu / 2, cv / 2
    if u < -1 or v < -1:
        return '.'  # outside the walls
    if 4.5 <= u < 7.5 and 1.5 <= v < 2.5:
        return 'i'  # the paved yard before the governor's hall
    if 4.5 <= v < 5.5 or 5.5 <= u < 6.5 and v >= 2.5 or 6.5 <= u and 8.5 <= v < 9.5:
        return 'd'  # the main street across, the way down from the hall, the lane before the store
    return 'd' if in_blob(cu, cv, WASTE) else '.'


# 탁군 성 안 (12×10 tiles): the scene city wall along the back two sides with the west gate on the main street; the
# governor's hall (관사) at the back behind a paved yard, the paved way down from it (u = 6), the main street across
# (v = 5). The mansion (저택) is the hall-like house in the plum garden at the back left, its door on its right
# face; the smithy (무기점) faces the main street from the back right, weapon racks out front; the tavern (주점)
# faces the paved way from the near left; the general store (도구점) faces the lane (v = 9) at the near right. The
# recruiting notice stands by the gate. Every building and standing prop sits on a tile's centre, since the walk
# is tile by tile: a house covers one tile, a hall three.
ZHUO = outdoor(W, H, zhuo_glyph, props=[
    *wall_u(-.75, -.75, W + M), *wall_v(-.75, -.25, H + M), *gate('v', -.75, 5.0),
    ('hall_u', 6, 1), ('banner', 5, 2), ('banner', 7, 2),
    *avenue('v', [(u, v / 2 + .25) for u in (5.75, 6.25) for v in range(5, 2 * H + 6)]),
    # 저택
    ('hall_v', 2, 2), ('stree_plum_0', 0, 0), ('stree_plum_1', 3, 0), ('taihu_rock', 4, 3),
    ('lantern', 3, 1), ('lantern', 3, 3), ('flagstones_0', 3.4, 2.0),
    # 무기점
    ('house_tile', 9, 3), ('weapon_rack', 8, 4), ('weapon_rack', 10, 4), ('armor_stand', 11, 4),
    ('house_thatch', 11, 1),
    # 주점
    ('house_thatch_x', 4, 7), ('wine_jar', 5, 6), ('wine_jar', 5, 8), ('bench', 3, 9), ('table_wine', 4, 9),
    ('house_tile_x', 1, 8),
    # 도구점
    ('house_tile', 10, 8), ('market_stall_jar', 11, 7), ('grain_sack', 9, 8), ('water_jar', 11, 6),
    # the street
    ('notice_post', 1, 4), ('well', 8, 7), ('market_stall', 3, 4),
    ('market_stall_cloth', 1, 6), ('flagstones_1', 7.4, 7.4), ('flagstones_0', 2.4, 6.2), ('straw_mat', 11, 5.6),
    ('stree_bush_0', 11, 9), ('stree_bush_1', 0, 9), ('stree_willow', 7, 8),
], trees=[(-2.2, 1.0, 'pine'), (-2.4, 7.0, 'broad'), (2.0, -2.4, 'pine'), (6.5, -2.4, 'broad'), (11.5, -2.0, 'old'),
          (13.5, 4.0, 'broad'), (13.0, 8.0, 'pine'), (4.0, 11.5, 'broad'), (10.5, 11.4, 'old'), (-2.0, 10.5, 'pine')],
    seed=19, grass=.06, margin=M)

# 관사 (7×6): the governor's office: a screen and a raised seat at the back right wall, his desk before it, two
# guest tables, the banners of the Han, the way out at the lower right
OFFICE = room(7, 6, 'floor_wood', 'wall_plaster', pillar='pillar_wood', props=[
    ('screen_u', 3.0, -.05), ('dais', 3.0, .55), ('desk', 3.0, 1.5), ('banner', 1.2, -.2), ('banner', 4.8, -.2),
    ('table_wine', 1.4, 3.0), ('table_wine', 4.6, 3.0), ('lantern', -.1, 1.6), ('lantern', 6.2, -.1),
    ('weapon_rack', -.1, 3.8), ('candle_stand', 1.9, .2), ('candle_stand', 4.1, .2),
])
# 무기점 (6×5): the smith's shop: racks of spears and swords along the back walls, armour on stands, the forge
# brazier and the quench tub in the corner, the counter by the way out at the lower right
WEAPON_SHOP = room(6, 5, 'floor_wood', 'wall_plaster', pillar='pillar_wood', props=[
    ('weapon_rack', .9, -.1), ('weapon_rack', 1.9, -.1), ('weapon_rack', 2.9, -.1), ('weapon_rack', -.1, 1.2),
    ('weapon_rack', -.1, 2.3), ('armor_stand', 4.2, -.05), ('armor_stand', 5.1, -.05), ('armor_stand', -.05, 3.6),
    ('brazier', .6, .6), ('water_bucket', 1.2, .5), ('cauldron', .5, 1.5), ('counter', 3.6, 2.4), ('grain_sack', 5.3, .6),
])
# 도구점 (6×5): a general store: jars, sacks and baskets along the walls, the medicine pot on the brazier, the
# counter in the middle, a bench for the customers
TOOL_SHOP = room(6, 5, 'floor_wood', 'wall_plaster', pillar='pillar_wood', props=[
    ('counter', 2.6, 1.0), ('wine_jar', .2, .2), ('wine_jar', .5, .1), ('water_jar', 1.2, .05), ('wine_jar', 3.4, .1),
    ('water_jar', 4.0, .05), ('grain_sack', -.05, 1.4), ('grain_sack', .1, 1.8), ('grain_sack', -.05, 2.2),
    ('medicine_pot', 5.2, .3), ('brazier', 5.3, .8), ('bench', .2, 3.6), ('water_bucket', 4.6, .2), ('lantern', -.1, 3.0),
    ('desk', 4.4, 2.6),
])
# 저택 (8×6): a rich man's hall: a gold screen and a raised seat at the back right wall, a red carpet down to it,
# wine tables for guests either side, candle stands and incense, curtains on the walls
MANSION = room(8, 6, 'floor_palace', 'wall_lacquer', extra_walls=['curtain'], props=[
    *[('floor_carpet', 3.5, v) for v in range(1, 6)],
    ('screen_gold', 3.5, .1), ('dais', 3.5, .6), ('candle_stand', 2.3, .3), ('candle_stand', 4.7, .3), ('incense', 2.6, 1.3),
    ('table_wine', 1.8, 2.4), ('table_wine', 5.2, 2.4), ('table_wine', 1.8, 4.0), ('table_wine', 5.2, 4.0),
    ('lantern', -.1, 2.0), ('lantern', 7.2, -.1), ('wine_jar', -.05, 4.8), ('wine_jar', .2, 5.1),
])

MAPS = [(ZHUO, 'S_zhuo', '탁군 성 안'), (OFFICE, 'S_zhuo_office', '탁군 관사'), (WEAPON_SHOP, 'S_weapon_shop', '무기점'),
        (TOOL_SHOP, 'S_tool_shop', '도구점'), (MANSION, 'S_mansion', '저택')]


def main():
    index_path = GFX / 'index.json'
    index = json.load(open(index_path))
    known = set(index['props']['sprites']) | set(index['extra']['sprites'])
    warnings = []
    for scene, map_id, name in MAPS:
        art = build.build(scene, 0, map_id, name, known, warnings.append)
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
