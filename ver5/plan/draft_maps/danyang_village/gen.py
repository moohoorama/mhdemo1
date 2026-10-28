"""The Danyang village drafts of 2026-10-28 (단양 고순의 마을 시안): the scene after B03, three framings of one place.

  python3 ver5/plan/draft_maps/danyang_village/gen.py      -> ver5/plan/draft_maps/danyang_village/<draft>.json

The place follows the B03 battle map (plan/draft_maps/README.md §2): the village sits in a corner under the mountain
(cliffs on the NW and NE sides), a palisade closes its SE face with one gap (어귀) where 고순 stood, and the valley road
Tao Qian came by runs east from the gap along a stream. Grain is looted and some houses are broken (c04-a11).
Coordinates are small tiles; the playing area starts at (0, 0) and the margin around it is drawn but not walked."""
import json, random
from pathlib import Path

OUT = Path(__file__).resolve().parent

TREES = ['stree_broad_0', 'stree_broad_1', 'stree_pine_0', 'stree_pine_1', 'stree_bush_0', 'stree_bush_1']
FENCE_BITS = {(-1, 0): 1, (0, -1): 2, (1, 0): 4, (0, 1): 8}
M = 12  # margin in small tiles


class Map:
    def __init__(self, mid, name, gw, gh, seed=1):
        self.mid, self.name, self.gw, self.gh = mid, name, gw, gh
        self.W, self.H = gw * 2, gh * 2
        self.g = [['.'] * (self.W + 2 * M) for _ in range(self.H + 2 * M)]
        self.h = [['0'] * (self.W + 2 * M) for _ in range(self.H + 2 * M)]
        self.obj, self.fence, self.used = [], set(), set()
        self.rng = random.Random(seed)

    def _cells(self, u0, v0, u1, v1):
        for v in range(max(-M, v0), min(self.H + M - 1, v1) + 1):
            for u in range(max(-M, u0), min(self.W + M - 1, u1) + 1):
                yield u, v

    def paint(self, u0, v0, u1, v1, ch='d'):
        for u, v in self._cells(u0, v0, u1, v1):
            self.g[v + M][u + M] = ch

    def cliff(self, u0, v0, u1, v1):
        for u, v in self._cells(u0, v0, u1, v1):
            self.h[v + M][u + M] = '2'

    def high(self, u, v):
        return 0 <= u + M < len(self.h[0]) and 0 <= v + M < len(self.h) and self.h[v + M][u + M] == '2'

    def put(self, name, u, v, dx=0, dy=0):
        self.obj.append([name, u, v, dx, dy] if dx or dy else [name, u, v])
        self.used.add((u, v))

    def fence_line(self, u0, v0, u1, v1, gaps=()):
        if u0 == u1:
            cells = [(u0, v) for v in range(min(v0, v1), max(v0, v1) + 1)]
        else:
            cells = [(u, v0) for u in range(min(u0, u1), max(u0, u1) + 1)]
        self.fence.update(c for c in cells if c not in gaps)

    def trees(self, n, region, avoid=lambda u, v: False, kinds=TREES, gap=2):
        u0, v0, u1, v1 = region
        placed = tries = 0
        while placed < n and tries < n * 80:
            tries += 1
            u, v = self.rng.randint(u0, u1), self.rng.randint(v0, v1)
            if avoid(u, v) or self.g[v + M][u + M] != '.':
                continue
            if any(abs(u - a) < gap and abs(v - b) < gap for a, b in self.used):
                continue
            self.put(self.rng.choice(kinds), u, v)
            placed += 1

    def save(self):
        for (u, v) in sorted(self.fence):
            m = sum(b for (du, dv), b in FENCE_BITS.items() if (u + du, v + dv) in self.fence)
            self.obj.append(['sfence_%02d' % m, u, v])
        d = {"id": self.mid, "name": self.name, "format": 2, "size": [self.gw, self.gh], "origin": [-M, -M],
             "ground": [''.join(r) for r in self.g], "height": [''.join(r) for r in self.h], "objects": self.obj}
        json.dump(d, open(OUT / ('%s.json' % self.mid.replace('S_draft_', '')), 'w'), ensure_ascii=False, indent=1)


def mountain(c, back_v=-1, back_u=-1):
    """Cliffs on the NE (v <= back_v) and NW (u <= back_u) sides with pines on top."""
    c.cliff(-M, -M, c.W + M, back_v)
    c.cliff(-M, -M, back_u, c.H + M)
    for u in range(-M + 2, c.W + M - 2, 3):
        c.put(c.rng.choice(['stree_pine_0', 'stree_pine_1', 'stree_broad_1']), u, back_v - 1 - c.rng.randint(0, 1))
    for v in range(-M + 4, c.H + M - 2, 3):
        c.put(c.rng.choice(['stree_pine_0', 'stree_pine_1', 'stree_broad_0']), back_u - 1 - c.rng.randint(0, 1), v)


def battle_marks(c, cells):
    kinds = ['broken_spear', 'fallen_flag', 'broken_spear', 'rubble_1', 'broken_spear']
    for i, (u, v) in enumerate(cells):
        c.put(kinds[i % len(kinds)], u, v)


# ---------- A: the gap in the palisade, inside and outside ----------
c = Map('S_draft_dy_gate', '단양 마을 어귀', 14, 10, seed=4)
mountain(c)
# palisade on the village's SE face; the gap (어귀) at v 9..10, two posts knocked down at v 5 and 14
c.fence_line(10, 0, 10, 19, gaps={(10, 9), (10, 10), (10, 5), (10, 14)})
c.put('rubble_2', 10, 5); c.put('rubble_0', 10, 14)
# inside: a lane from the gap, houses on both sides, one broken in
c.paint(0, 9, 9, 10)
c.paint(4, 7, 8, 12)
c.put('house_thatch', 2, 2); c.put('house_thatch_x', 6, 2); c.put('hut', 9, 3); c.put('house_thatch', 2, 6)
c.put('house_burnt', 2, 14); c.put('house_thatch_x', 6, 15); c.put('hut', 2, 18); c.put('burnt_hut', 8, 18)
c.put('bench', 5, 5); c.put('wine_jar', 8, 5); c.put('water_jar', 4, 17); c.put('rubble_4', 4, 13)
c.put('well', 6, 7); c.put('water_bucket', 7, 7)
c.put('straw_mat', 2, 9); c.put('straw_mat', 2, 11); c.put('grain_sack', 3, 12)
# outside: trampled ground before the gap, the valley road east by the stream, the mountain road south
c.paint(11, 6, 17, 13)
c.paint(11, 9, 27, 10)
c.paint(12, 11, 13, 19)
c.paint(12, 3, 27, 4, '~')
battle_marks(c, [(12, 7), (14, 8), (16, 12), (13, 13), (15, 6), (18, 11), (11, 12)])
c.put('stree_willow', 20, 6); c.put('stree_willow', 25, 6)
c.trees(5, (18, 16, 30, 26), avoid=lambda u, v: 11 <= u <= 14, gap=4)
c.trees(3, (0, 0, 9, 19), avoid=lambda u, v: 6 <= v <= 13, kinds=['stree_bush_0', 'stree_bush_1'], gap=3)
c.save()

# ---------- B: the village square ----------
c = Map('S_draft_dy_square', '단양 마을 마당', 12, 10, seed=7)
mountain(c)
c.fence_line(22, 0, 22, 19, gaps={(22, 9), (22, 10)})
c.fence_line(0, 19, 22, 19)
# square: the old village tree and the well in the middle, houses all round
c.paint(6, 6, 16, 13)
c.paint(17, 9, 27, 10)
c.put('stree_old', 9, 7); c.put('well', 13, 11); c.put('water_jar', 14, 12)
c.put('house_thatch', 3, 2); c.put('house_thatch_x', 9, 2); c.put('house_burnt', 15, 2); c.put('hut', 20, 3)
c.put('house_thatch_x', 2, 9); c.put('hut', 2, 15)
c.put('house_thatch', 8, 16); c.put('burnt_hut', 13, 16); c.put('rubble_3', 14, 14)
# the threshing floor by the gap: mats left bare, a torn sack or two
c.paint(16, 13, 20, 17)
c.put('straw_mat', 17, 14); c.put('straw_mat', 19, 14); c.put('straw_mat', 17, 16); c.put('straw_mat', 19, 16)
c.put('grain_sack', 20, 13); c.put('grain_sack', 16, 17)
battle_marks(c, [(21, 8), (20, 11), (18, 10), (21, 12)])
c.trees(4, (25, 0, 33, 19), avoid=lambda u, v: 7 <= v <= 12, gap=4)
c.trees(3, (0, 23, 33, 30), gap=4)
c.put('bench', 6, 5); c.put('wine_jar', 18, 4); c.put('house_thatch', 5, 6); c.put('hut', 18, 7); c.put('water_jar', 4, 12); c.put('hut', 5, 18)
c.save()

# ---------- C: the whole corner under the mountain ----------
c = Map('S_draft_dy_whole', '단양 산자락 마을', 16, 12, seed=11)
mountain(c)
c.fence_line(18, 0, 18, 18, gaps={(18, 8), (18, 9)})
c.fence_line(0, 18, 18, 18)
# lanes and a small plaza
c.paint(0, 8, 17, 9)
c.paint(8, 0, 9, 17)
c.paint(6, 6, 12, 11)
c.put('well', 11, 7); c.put('stree_old', 7, 7)
for name, u, v in [('house_thatch', 3, 2), ('house_thatch_x', 13, 2), ('hut', 16, 4), ('house_burnt', 3, 12),
                   ('hut', 6, 15), ('burnt_hut', 16, 14), ('house_thatch_x', 12, 16)]:
    c.put(name, u, v)
# fields: two bare plots fenced low inside the palisade
c.paint(1, 13, 5, 16); c.paint(10, 13, 11, 14)
c.put('straw_mat', 14, 7); c.put('straw_mat', 15, 9); c.put('grain_sack', 14, 13)
# outside: the valley road east by the stream, the mountain road south
c.paint(19, 7, 23, 10)
c.paint(19, 8, 31, 9)
c.paint(20, 10, 21, 23)
c.paint(20, 3, 31, 4, '~')
battle_marks(c, [(19, 6), (21, 7), (22, 11), (19, 11), (23, 9)])
c.put('stree_willow', 24, 5); c.put('stree_willow', 29, 5)
c.trees(4, (24, 14, 35, 30), gap=4)
c.trees(3, (0, 22, 17, 30), gap=4)
c.put('hut', 5, 7); c.put('water_jar', 2, 5)
c.save()
