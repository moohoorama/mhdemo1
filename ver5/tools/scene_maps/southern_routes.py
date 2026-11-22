#!/usr/bin/env python3
"""Build the chapter 25–27 scene maps from small-tile terrain and props."""

import json
import math
import subprocess
from pathlib import Path

GRAPHICS = Path(__file__).resolve().parents[2] / 'assets/graphics'


class SceneMap:
    def __init__(self, name, label, size, origin, extent, grass=None):
        self.meta = {'id': name, 'name': label, 'format': 2, 'size': size, 'origin': origin}
        if grass is not None:
            self.meta['grass'] = grass
        self.origin = origin
        self.width, self.depth = extent
        self.ground = [['.'] * self.width for _ in range(self.depth)]
        self.height = [['0'] * self.width for _ in range(self.depth)]
        self.objects = []

    def terrain(self, choose):
        for y, row in enumerate(self.ground):
            for x in range(len(row)):
                row[x] = choose(x + self.origin[0], y + self.origin[1])

    def patch(self, u0, v0, u1, v1, tile, height=None):
        for v in range(v0, v1 + 1):
            for u in range(u0, u1 + 1):
                x, y = u - self.origin[0], v - self.origin[1]
                self.ground[y][x] = tile
                if height is not None:
                    self.height[y][x] = height

    def props(self, *items):
        self.objects.extend(map(list, items))

    def deck(self, u0, v0, u1, v1, axis):
        self.patch(u0, v0, u1, v1, 'b')
        self.props(*((f'sbridge_{axis}', u, v)
                     for v in range(v0, v1 + 1) for u in range(u0, u1 + 1)))

    def write(self):
        data = {**self.meta, 'ground': [''.join(row) for row in self.ground]}
        if any('0' * self.width != ''.join(row) for row in self.height):
            data['height'] = [''.join(row) for row in self.height]
        data['objects'] = self.objects
        target = f'maps/{self.meta["id"]}.json'
        (GRAPHICS / target).write_text(json.dumps(data, ensure_ascii=False, indent=1) + '\n')
        return target


def changban(broken=False):
    map_id = 'S_changban_broken' if broken else 'S_changban'
    scene = SceneMap(map_id, '장판교 · 동쪽 둑', [20, 15], [-12, -18], [65, 59],
                     {'hue': 22, 'sat': 0.75, 'val': 0.87})

    def terrain(u, v):
        center = math.floor(1.6 * math.sin((u - 16) / 11))
        if center - 5 <= v <= center + 5:
            return '~'
        if abs(u - 17) <= 2 or (9 <= v <= 14 and u >= 12):
            return 'd'
        if 5 <= u <= 26 and 10 <= v <= 25 and math.sin(u / 4 + v / 5) > -0.25:
            return 'd'
        return '.'

    scene.terrain(terrain)
    if broken:
        scene.deck(16, -6, 17, -3, 'v')
        scene.deck(16, 3, 17, 7, 'v')
        scene.props(('broken_spear', 17, 4), ('rubble_1', 18, 4))
    else:
        scene.deck(16, -6, 17, 7, 'v')
    scene.props(
        ('stree_pine_1', 10, -8), ('stree_old', 23, -9), ('stree_broad_0', 3, -6),
        ('stree_willow', 28, 6), ('stree_bush_0', 22, 8), ('stree_bush_1', -1, 7),
        ('stree_pine_0', 34, 3), ('stree_pine_1', 41, 5), ('stree_broad_1', -5, 15),
        ('stree_broad_0', 8, 25), ('stree_bush_1', 1, 18), ('stree_bush_0', 38, 18),
        ('tent', 6, 15), ('tent', 9, 19), ('tent_round', 20, 19),
        ('campfire', 12, 21), ('straw_mat', 13, 19), ('straw_mat', 16, 21),
        ('medicine_pot', 14, 20), ('grain_cart', 25, 16), ('grain_sack', 24, 18),
        ('broken_spear', 20, 9), ('fallen_flag', 12, 7), ('water_jar', 4, 17),
        ('milestone', 31, 9), ('rock_pile_0', 21, 7), ('rock_pile_1', 5, 8),
    )
    return scene


def hanjin():
    scene = SceneMap('S_hanjin_ferry', '한진 나루', [19, 14], [-12, -18], [63, 60],
                     {'hue': 35, 'sat': 0.62, 'val': 0.9})

    def terrain(u, v):
        bank = round(11 + 3 * math.sin((u - 12) / 7) + 1.5 * math.sin(u / 3.5))
        if v < bank:
            return '~'
        if v < bank + 4 or abs(u - (13 + (v - 15) // 4)) <= 2:
            return 'd'
        return '.'

    scene.terrain(terrain)
    scene.deck(11, 7, 12, 14, 'v')
    scene.props(
        ('boat_l_v', 9, 9), ('boat_v', 15, 9), ('boat_u', 1, 5),
        ('stree_willow', 4, 13), ('stree_willow', 23, 16), ('stree_bush_1', 3, 16),
        ('rock_pile_0', 6, 13), ('rock_pile_2', 19, 13), ('rock_pile_1', -4, 11),
        ('grain_sack', 15, 14), ('water_jar', 16, 16), ('hut', 23, 20),
        ('campfire', 19, 20), ('straw_mat', 21, 20), ('milestone', 9, 18),
        ('stree_broad_1', 35, 19), ('stree_broad_0', -1, 27), ('stree_bush_0', 32, 21),
    )
    return scene


def jiangxia():
    scene = SceneMap('S_jiangxia_harbor', '강하 나루', [23, 16], [-16, -20], [75, 65],
                     {'hue': -12, 'sat': 0.79, 'val': 0.91})

    def terrain(u, v):
        shore = 10 if -3 <= u <= 31 else 12 + round(2 * math.sin(u / 9))
        if v < shore:
            return '~'
        if 0 <= u <= 30 and v <= 20 or 12 <= u <= 17:
            return 'i'
        return 'd' if (u + v) % 11 < 4 else '.'

    scene.terrain(terrain)
    scene.deck(10, 1, 13, 10, 'v')
    scene.deck(23, 3, 25, 10, 'v')
    scene.deck(1, 5, 3, 10, 'v')
    scene.props(
        ('boat_l_v', 7, 6), ('boat_v', 4, 5), ('boat_l_v', 16, 4),
        ('boat_l_v', 21, 5), ('boat_l_v', 28, 4), ('boat_u', -2, 3),
        ('boat_u', 33, 0), ('boat_v', -8, 5),
        ('lantern', 8, 11), ('lantern', 16, 11), ('lantern', 22, 10),
        ('brazier', 5, 14), ('command_flag', 15, 13), ('banner', 26, 11),
        ('granary_raised', 1, 20), ('granary_round', 6, 23), ('hall_v', 25, 18),
        ('grain_cart_v', 22, 17), ('grain_sack', 23, 18), ('grain_sack', 2, 14),
        ('grain_sack', 4, 19), ('water_jar', 18, 16), ('watchtower', 29, 15),
        ('stree_willow', -5, 15), ('stree_broad_0', 33, 19), ('stree_bush_1', 31, 24),
        ('rock_pile_1', -8, 14), ('rock_pile_0', 35, 12),
    )
    scene.props(*(('sfence_05', u, 20) for u in (*range(8, 12), *range(18, 23))))
    return scene


def huarong():
    scene = SceneMap('S_huarong_marsh', '화용도 · 늪길', [21, 15], [-14, -17], [73, 58],
                     {'hue': 38, 'sat': 0.55, 'val': 0.77})

    def terrain(u, v):
        path = 7 + round(3.4 * math.sin((u - 10) / 9))
        delta = abs(v - path)
        noise = math.sin(u * .63 + v * .38) + math.sin(u * .22 - v * .66)
        if delta <= 1:
            return 'd'
        if delta == 2:
            return 'R' if noise > 0.65 else '.'
        if delta < 6:
            return 'F' if noise > -0.65 else '~'
        if noise > 0.7:
            return '.' if delta > 8 else 'F'
        return '~'

    scene.terrain(terrain)
    scene.props(
        ('stree_willow', 0, 10), ('stree_old', 15, -2), ('stree_willow', 25, 16),
        ('stree_willow', -6, -1), ('stree_willow', 37, 11),
        ('stree_bush_0', 7, 2), ('stree_bush_1', 9, 8), ('stree_bush_1', 13, 4),
        ('stree_bush_0', 16, 14), ('stree_bush_1', 20, 16), ('stree_bush_0', 27, 6),
        ('stree_bush_1', 30, 13), ('stree_bush_0', 34, 4), ('stree_bush_1', -3, 2),
        ('rock_pile_1', 6, 8), ('rock_pile_0', 18, 5), ('rock_pile_2', 29, 7),
        ('rubble_1', 11, 1), ('broken_spear', 17, 9), ('fallen_flag', 23, 10),
        ('broken_spear', 28, 11), ('milestone', 33, 7),
    )
    return scene


def chibi():
    scene = SceneMap('S_chibi_fleet', '적벽 · 강 위', [24, 17], [-22, -26], [91, 74],
                     {'hue': 31, 'sat': 0.64, 'val': 0.67})
    scene.terrain(lambda u, v: '~')
    for u in range(-22, 69):
        edge = -9 + round(3 * math.sin(u / 8)) + (2 if u < 5 else 0)
        scene.patch(u, -26, u, edge, 'd', '2')
        scene.patch(u, edge + 1, u, edge + 1, 'd')

    # The tapered foreground deck is long enough for the officers and the prow beat.
    scene.patch(5, 15, 27, 21, 'W')
    scene.patch(2, 17, 4, 20, 'W')
    scene.patch(28, 16, 31, 20, 'W')
    scene.patch(32, 17, 33, 19, 'W')
    scene.props(*(('sfence_05', u, 15) for u in range(5, 28)),
                *(('sfence_05', u, 21) for u in range(5, 28)))
    scene.props(('sfence_12', 5, 15), ('sfence_09', 27, 15),
                ('sfence_06', 5, 21), ('sfence_03', 27, 21))
    scene.props(
        ('command_flag', 9, 16), ('banner', 24, 16), ('lantern', 5, 19),
        ('lantern', 29, 16), ('grain_sack', 6, 16), ('water_jar', 26, 20),
        ('map_table', 17, 16), ('grain_sack', 3, 19),
        ('boat_l_u', -1, 12), ('boat_u', 4, 27), ('boat_l_u', 29, 28),
        ('boat_l_u', 36, 15), ('boat_l_u', 44, 17),
        ('boat_l_u', 0, 2), ('boat_l_u', 7, 1), ('boat_l_v', 14, 4),
        ('boat_l_u', 20, 0), ('boat_l_u', 28, 3), ('boat_l_u', 36, 3),
        ('boat_l_u', 5, -4), ('boat_l_u', 17, -4), ('boat_l_u', 27, -3),
        ('boat_l_u', 38, -3), ('boat_v', 24, 9), ('boat_u', 33, 8),
        ('stree_pine_0', 5, -20), ('stree_pine_1', 15, -22),
        ('rock_pile_0', 27, -20), ('stree_pine_0', 41, -20),
        ('stree_broad_0', -5, -17), ('stree_pine_1', 52, -21),
    )
    return scene


def main():
    index_path = GRAPHICS / 'index.json'
    index = index_path.read_text()
    targets = [build().write() for build in (changban, hanjin, jiangxia, huarong, chibi)]
    targets.append(changban(broken=True).write())
    new_paths = []
    for target in targets:
        if target not in index:
            new_paths.append(target)
    if new_paths:
        list_end = index.index('\n ],', index.index(' "maps": ['))
        entries = ''.join(f',\n  "{path}"' for path in new_paths)
        index_path.write_text(index[:list_end] + entries + index[list_end:])
    for target in targets:
        subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', Path(target).stem],
                       cwd=GRAPHICS.parents[1], check=True)


if __name__ == '__main__':
    main()
