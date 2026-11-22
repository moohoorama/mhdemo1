#!/usr/bin/env python3
"""Generate the walkable city maps for Xinye, Xiangyang and Jiangling."""
import json
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
MARGIN = 12


class Town:
    def __init__(self, key, name, width, height):
        self.key, self.name = key, name
        self.width, self.height = width, height
        self.ground = [['.'] * (width + 2 * MARGIN) for _ in range(height + 2 * MARGIN)]
        self.objects = []
        self.walls = []

    def paint(self, u0, v0, u1, v1, tile):
        for v in range(v0, v1 + 1):
            for u in range(u0, u1 + 1):
                self.ground[v + MARGIN][u + MARGIN] = tile

    def place(self, *objects):
        self.objects.extend(objects)

    def enclosure(self, kind, left, top, right, bottom):
        self.walls.extend([
            [kind, [left, top], [right, top]],
            [kind, [left, top + 1], [left, bottom]],
            [kind, [left + 1, bottom], [right, bottom]],
            [kind, [right, top + 1], [right, bottom - 1]],
        ])

    def city(self):
        self.paint(0, 0, self.width - 1, self.height - 1, 'd')
        self.enclosure('swall', -1, -1, self.width, self.height)

    def write(self):
        relative = f'maps/{self.key}.json'
        art = dict(id=self.key, name=self.name, format=2,
                   size=[self.width // 2, self.height // 2], origin=[-MARGIN, -MARGIN],
                   ground=[''.join(row) for row in self.ground],
                   objects=self.objects, walls=self.walls)
        (GFX / relative).write_text(json.dumps(art, ensure_ascii=False, indent=1) + '\n')
        return relative


def xinye():
    t = Town('S_xinye', '신야성', 30, 26)
    t.city()
    t.paint(18, -12, 20, 25, 'i')
    t.paint(4, 9, 29, 12, 'i')
    t.paint(7, 15, 16, 24, '.')
    t.paint(21, 1, 29, 7, '.')
    t.place(('sgate_open', 19, -1), ('sgate_open', 30, 11),
            ('hall_v', 2, 11), ('banner', 5, 7), ('banner', 5, 15),
            ('notice_board', 7, 9), ('stree_old', 2, 4),
            ('tavern', 11, 6), ('wine_jar', 8, 8), ('lantern', 13, 8),
            ('house_tile', 24, 7), ('market_stall_jar', 27, 8),
            ('house_thatch', 7, 3), ('house_tile_x', 16, 3),
            ('well', 13, 16), ('water_bucket', 14, 17),
            ('house_tile_x', 4, 20), ('house_thatch', 11, 23),
            ('field_sprout', 8, 16), ('field_ripe', 16, 22),
            ('granary_raised', 26, 3), ('grain_sack', 23, 4),
            ('market_stall_cloth', 23, 15), ('market_stall', 27, 15),
            ('grain_cart_v', 27, 20), ('house_tile', 23, 23),
            ('stree_willow', 19, 23), ('stree_broad_0', 26, 24))
    return t


def estate(t, left, wood, trees):
    right, gate = left + 19, left + 10
    t.enclosure('cwall', left, 4, right, 14)
    t.paint(left + 1, 5, right - 1, 7, '.')
    t.paint(left + 1, 8, right - 1, 13, 'i')
    t.paint(gate - 3, 14, gate + 2, 19, 'i')
    t.place(('cgate_open', gate, 14, {'wood': wood}),
            ('hall_u', gate - 1, 8, 8, 4), ('hall_v', gate - 6, 11, 8, 4),
            ('hall_v', gate + 4, 11, 8, 4), ('flagstones_0', gate - 1, 11, 8, 4),
            (trees[0], left + 2, 6), (trees[1], right - 2, 6),
            ('pavilion', right - 3, 8), ('taihu_rock', left + 1, 11),
            ('banner', gate - 3, 16), ('banner', gate + 2, 16))


def xiangyang():
    t = Town('S_xiangyang', '양양성', 48, 40)
    t.city()
    # Han River runs behind the north wall and bends around the east wall.
    for u in range(0, 51):
        bank = -5 + min(3, u // 15)
        t.paint(u, -12, u, bank, '~')
    for v in range(-12, 52):
        bank = 53 - min(3, max(0, v) // 9)
        if v < -2:
            bank = min(bank, 50)
        t.paint(bank, v, 59, v, '~')
    t.paint(-12, 20, 47, 23, 'i')
    t.paint(22, 15, 26, 39, 'i')
    t.paint(7, 26, 18, 35, '.')
    t.paint(9, 34, 16, 36, 'i')
    t.paint(30, 27, 45, 35, 'i')
    estate(t, 2, '#8c5a40', ('stree_pine_0', 'stree_pine_1'))
    estate(t, 26, '#544032', ('stree_plum_0', 'stree_plum_1'))
    t.place(('sgate_open', -1, 22), ('hall_u', 12, 30),
            ('stree_plum_0', 8, 28), ('stree_plum_1', 16, 28),
            ('stree_plum_0', 7, 34), ('lantern', 10, 32), ('lantern', 15, 32),
            ('flagstones_0', 12, 32), ('flagstones_1', 12, 33),
            ('well', 17, 36), ('field_sprout', 19, 33),
            ('hall_v', 4, 27), ('notice_board', 7, 24), ('banner', 8, 25),
            ('tavern', 32, 26), ('wine_jar', 29, 27), ('lantern', 30, 28),
            ('house_tile', 41, 25), ('market_stall_jar', 44, 27),
            ('market_stall_cloth', 33, 34), ('market_stall', 39, 34),
            ('grain_cart', 44, 35), ('house_tile_x', 44, 39),
            ('house_tile', 36, 38), ('house_tile_x', 5, 37), ('house_tile', 23, 37),
            ('stree_willow', 49, 5), ('stree_willow', 49, 17),
            ('boat_v', 54, 22), ('boat_v', 54, 36))
    return t


def jiangling():
    t = Town('S_jiangling', '강릉성', 46, 38)
    t.city()
    for u in range(-12, 58):
        bank = 40 + min(5, max(0, u) // 8)
        t.paint(u, bank, u, 49, '~')
    t.paint(-12, 14, 45, 17, 'i')
    t.paint(11, 14, 15, 41, 'i')
    t.paint(5, 34, 24, 39, 'i')
    t.paint(12, 40, 15, 44, 'b')
    t.place(*(('sbridge_v', u, v) for u in range(12, 16) for v in range(40, 45)))
    t.paint(23, 24, 43, 35, '.')
    t.place(('sgate_open', -1, 16), ('sgate_open', 14, 38),
            ('hall_u', 12, 9), ('banner', 9, 12), ('banner', 16, 12),
            ('notice_board', 18, 13), ('stree_pine_0', 6, 6),
            ('tavern', 25, 10), ('wine_jar', 22, 12), ('lantern', 27, 12),
            ('house_tile', 36, 11), ('market_stall_jar', 39, 13),
            ('market_stall', 22, 20), ('market_stall_cloth', 28, 20),
            ('granary_raised', 6, 28), ('granary_round', 19, 29),
            ('grain_sack', 16, 35), ('grain_cart', 21, 36),
            ('boat_v', 10, 43), ('boat_u', 18, 45), ('boat_u', 28, 46),
            ('well', 26, 26), ('house_tile_x', 31, 29),
            ('house_tile', 39, 26), ('house_tile_x', 43, 33),
            ('field_sprout', 31, 34), ('stree_willow', 2, 38),
            ('stree_willow', 26, 40), ('stree_broad_1', 42, 22))
    return t


if __name__ == '__main__':
    towns = (xinye(), xiangyang(), jiangling())
    path = GFX / 'index.json'
    index = json.loads(path.read_text())
    for town in towns:
        relative = town.write()
        if relative not in index['maps']:
            index['maps'].append(relative)
    path.write_text(json.dumps(index, ensure_ascii=False, indent=1) + '\n')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', *(town.key for town in towns)],
                   cwd=GFX.parents[1], check=True)
