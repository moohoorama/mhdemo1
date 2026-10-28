"""The camp drafts of 2026-10-28 (진영 맵 시안): a small layout kit and the six layouts.

  python3 ver5/plan/draft_maps/camps/gen_camps.py      -> ver5/plan/draft_maps/camps/<draft>.json

The chosen three were copied into the game and edited there (assets/graphics/maps/S_militia_camp, S_luzhi_camp,
S_coalition_camp: big tents and grain carts put in by hand), so those files are the source now; this script keeps
the other three (militia_b, luzhi_a, coalition_b) reproducible. Coordinates are small tiles from origin 0, 0."""
import json, random
from pathlib import Path

OUT = Path(__file__).resolve().parent

TREES = ['stree_broad_0', 'stree_broad_1', 'stree_pine_0', 'stree_pine_1', 'stree_old', 'stree_bush_0', 'stree_bush_1']
FENCE_BITS = {(-1, 0): 1, (0, -1): 2, (1, 0): 4, (0, 1): 8}


class Camp:
    def __init__(self, mid, name, W, H, seed=1):
        self.mid, self.name, self.W, self.H = mid, name, W, H
        self.g = [['.'] * W for _ in range(H)]
        self.obj, self.fence, self.walls = [], set(), []
        self.used = set()
        self.labels = []  # (u, v, text, kind)
        self.rng = random.Random(seed)

    # ground
    def paint(self, u0, v0, u1, v1, ch='d'):
        for v in range(max(0, v0), min(self.H, v1 + 1)):
            for u in range(max(0, u0), min(self.W, u1 + 1)):
                self.g[v][u] = ch

    def road_u(self, v, u0, u1, w=2, ch='d'):
        self.paint(u0, v, u1, v + w - 1, ch)

    def road_v(self, u, v0, v1, w=2, ch='d'):
        self.paint(u, v0, u + w - 1, v1, ch)

    # objects
    def put(self, name, u, v, dx=0, dy=0):
        self.obj.append([name, u, v, dx, dy] if dx or dy else [name, u, v])
        self.used.add((u, v))

    def row_u(self, name, u0, u1, v, step=3):
        for u in range(u0, u1 + 1, step):
            self.put(name, u, v)

    def row_v(self, name, u, v0, v1, step=3):
        for v in range(v0, v1 + 1, step):
            self.put(name, u, v)

    def grid(self, name, u0, v0, nu, nv, su=3, sv=3):
        for j in range(nv):
            for i in range(nu):
                self.put(name, u0 + i * su, v0 + j * sv)

    def fence_line(self, u0, v0, u1, v1, gaps=()):
        if u0 == u1:
            cells = [(u0, v) for v in range(min(v0, v1), max(v0, v1) + 1)]
        else:
            cells = [(u, v0) for u in range(min(u0, u1), max(u0, u1) + 1)]
        for c in cells:
            if c not in gaps:
                self.fence.add(c)

    def fence_box(self, u0, v0, u1, v1, gaps=()):
        self.fence_line(u0, v0, u1, v0, gaps)
        self.fence_line(u0, v1, u1, v1, gaps)
        self.fence_line(u0, v0, u0, v1, gaps)
        self.fence_line(u1, v0, u1, v1, gaps)

    def wall_u(self, v, u0, u1, gate=None, closed=False):
        self.walls.append(['swall', [u0, v], [u1, v]])
        if gate is not None:
            self.put('sgate_closed' if closed else 'sgate_open', gate, v)

    def wall_v(self, u, v0, v1, gate=None, closed=False):
        self.walls.append(['swall', [u, v0], [u, v1]])
        if gate is not None:
            self.put('sgate_closed' if closed else 'sgate_open', u, gate)

    def trees(self, n, region, avoid=lambda u, v: False, kinds=TREES, gap=2):
        u0, v0, u1, v1 = region
        placed, tries = 0, 0
        while placed < n and tries < n * 60:
            tries += 1
            u, v = self.rng.randint(u0, u1), self.rng.randint(v0, v1)
            if not (0 <= u < self.W and 0 <= v < self.H) or avoid(u, v) or self.g[v][u] != '.':
                continue
            if any(abs(u - a) < gap and abs(v - b) < gap for a, b in self.used):
                continue
            self.put(self.rng.choice(kinds), u, v)
            placed += 1

    def label(self, u, v, text, kind='zone'):
        self.labels.append((u, v, text, kind))

    def save(self):
        for (u, v) in self.fence:
            m = sum(b for (du, dv), b in FENCE_BITS.items() if (u + du, v + dv) in self.fence)
            self.obj.append(['sfence_%02d' % m, u, v])
        d = {"id": self.mid, "name": self.name, "format": 2, "size": [(self.W + 1) // 2, (self.H + 1) // 2],
             "origin": [0, 0], "ground": [''.join(r) for r in self.g], "objects": self.obj}
        if self.walls:
            d["walls"] = self.walls
        json.dump(d, open(OUT / ('%s.json' % self.mid.replace('S_draft_', '')), 'w'), ensure_ascii=False, indent=1)


# ---------------- militia ----------------
# ---------- A: under the Zhuo wall ----------
c = Camp('S_draft_militia_a', '의병 주둔지 A', 38, 34, seed=5)
c.paint(0, 0, 37, 2, 'i')
c.wall_u(3, 0, 37, gate=10)
c.label(10, 2, '탁현 성문 → 탁현으로 (출구)', 'exit')
c.road_v(9, 4, 33, w=3)
c.label(10, 31, '대흥산 가는 길 (출진은 관우에게)', 'gate')
# half-built palisade: only the outer sides, logs missing in places
c.fence_line(14, 6, 35, 6, gaps={(20, 6), (21, 6), (28, 6)})
c.fence_line(35, 6, 35, 31, gaps={(35, 14), (35, 22), (35, 23)})
c.fence_line(14, 31, 35, 31, gaps={(17, 31), (26, 31), (27, 31)})
c.label(35, 22, '짓다 만 목책', 'zone')
c.put('watchtower', 34, 7)
# recruiting table by the road
c.put('desk', 12, 9); c.put('bench', 12, 11); c.put('notice_post', 13, 7); c.put('banner', 13, 12)
c.label(12, 9, '모병소 (술꾼이 이름을 올린 곳)', 'zone')
# drill ground: dummies, racks
c.paint(15, 8, 25, 17)
for v in (10, 13, 16): c.put('armor_stand', 24, v)
c.put('weapon_rack', 16, 9); c.put('weapon_rack', 18, 9); c.put('weapon_rack', 16, 16)
c.put('command_flag', 20, 12)
c.label(19, 12, '훈련장 (장비가 의병을 조련)', 'zone')
# barracks: tents and mats
c.grid('tent', 28, 9, 3, 4, 2, 3)
c.grid('straw_mat', 28, 22, 3, 2, 2, 3)
c.put('campfire', 31, 19)
c.label(30, 13, '의병 막사', 'zone')
c.label(30, 24, '천막이 모자라 거적에서 자는 자리', 'zone')
# brothers' tent
c.put('tent_round', 19, 24); c.put('lantern', 21, 25); c.put('banner', 17, 22)
c.label(19, 24, '삼형제의 막사 ★ 들어감 (S_tent)', 'enter')
# kitchen
c.put('cauldron', 14, 21); c.put('soup_stove', 14, 24); c.put('water_jar', 13, 27); c.put('grain_sack', 15, 28); c.put('grain_sack', 16, 29)
c.label(14, 25, '취사 (숲 판 돈으로 산 쌀)', 'zone')
c.trees(26, (0, 9, 7, 33), gap=3)
c.trees(10, (12, 32, 37, 33), gap=3)

c.save()

# ---------- B: a borrowed hamlet ----------
c = Camp('S_draft_militia_b', '의병 주둔지 B', 38, 34, seed=8)
c.road_u(3, 0, 37, w=3)
c.put('milestone', 2, 7)
c.label(1, 4, '← 탁현으로 (출구)', 'exit')
c.road_v(18, 6, 33, w=2)
c.label(18, 31, '대흥산 가는 길', 'gate')
# houses lent by the villagers
for u, v, n in [(8, 10, 'house_thatch'), (8, 16, 'house_thatch_x'), (27, 10, 'house_thatch'), (27, 17, 'house_thatch_x'), (8, 23, 'hut')]:
    c.put(n, u, v)
c.label(8, 10, '마을이 내준 초가 ★ 들어감 (S_house) — 삼형제 숙소', 'enter')
c.label(27, 14, '초가 (의병 숙소)', 'zone')
c.put('well', 14, 12); c.put('water_bucket', 15, 13)
c.label(14, 12, '우물', 'zone')
# low fences of the yards
c.fence_box(4, 7, 12, 27, gaps={(12, 12), (12, 13), (12, 19), (12, 20)})
c.fence_box(23, 7, 31, 21, gaps={(23, 12), (23, 13)})
# drill field behind the hamlet
c.paint(21, 23, 35, 31)
for u in (24, 27, 30, 33): c.put('armor_stand', u, 30)
c.put('weapon_rack', 22, 24); c.put('weapon_rack', 22, 27); c.put('command_flag', 28, 25)
c.label(28, 27, '마을 뒤 공터 훈련장', 'zone')
# tents squeezed between
c.grid('tent', 33, 8, 2, 4, 2, 3)
c.label(34, 13, '모자라서 친 천막', 'zone')
c.put('cauldron', 15, 20); c.put('soup_stove', 15, 23); c.put('grain_sack', 13, 25); c.put('campfire', 20, 20)
c.label(15, 22, '마당 취사', 'zone')
c.put('desk', 16, 6); c.put('bench', 15, 7); c.put('notice_post', 21, 6); c.put('banner', 19, 6)
c.label(16, 6, '모병소', 'zone')
inner = lambda u, v: 3 <= u <= 36 and 5 <= v <= 32
c.trees(26, (0, 5, 37, 33), avoid=inner, gap=3)
c.trees(14, (0, 0, 37, 2), gap=3)
c.trees(6, (13, 14, 17, 18), gap=3)
c.save()


# ---------------- luzhi_a ----------------
c = Camp('S_draft_luzhi_a', '노식 진영 A', 48, 42, seed=3)
# 광종성: the wall along the top-right edge, Yellow Turban flags behind it
c.paint(0, 0, 47, 2, 'i')
c.wall_u(3, 0, 47, gate=24, closed=True)
for u in range(4, 46, 6): c.put('banner_yellow', u, 1)
c.label(24, 1, '광종성 (장각이 틀어박힌 성)', 'far')
# no-man's land and the antler barrier
c.paint(0, 4, 47, 10, 'd')
for u in range(3, 45, 2):
    if u not in (23, 25): c.put('antler_barrier', u, 9)
c.label(10, 9, '녹각 (성 앞 방어선)', 'zone')
# camp palisade and gates
c.fence_box(4, 13, 43, 38, gaps={(23, 13), (24, 13), (25, 13), (23, 38), (24, 38), (25, 38)})
for u, v in [(5, 14), (42, 14), (5, 37), (42, 37)]: c.put('watchtower', u, v)
c.put('watchtower', 21, 11); c.put('watchtower', 27, 11)
c.put('watchtower', 21, 40); c.put('watchtower', 27, 40)
c.label(24, 12, '북문 (성 쪽)', 'gate')
c.label(24, 40, '남문 → 탁군으로 (출구)', 'exit')
c.road_v(23, 10, 41, w=3)
c.road_u(25, 5, 42, w=2)
# 본영: 노식's own enclosure, west of the avenue
c.paint(10, 15, 20, 22)
c.fence_box(9, 15, 21, 23, gaps={(21, 18), (21, 19), (21, 20)})
c.put('tent_round', 15, 18)
for u, v in [(19, 17), (19, 21)]: c.put('command_flag', u, v)
c.put('brazier', 20, 17); c.put('brazier', 20, 21)
c.put('weapon_rack', 11, 21); c.put('armor_stand', 13, 21); c.put('banner', 11, 16); c.put('banner', 18, 16)
c.label(15, 18, '노식의 본영 ★ 들어감 (S_tent_mid)', 'enter')
# 군량: east of the avenue, carts to be drawn
c.paint(28, 15, 40, 22)
for u in range(29, 40, 3):
    for v in (16, 18, 20): c.put('grain_sack', u, v)
c.put('water_jar', 39, 22); c.put('wine_jar', 28, 22)
c.put('tent', 37, 15); c.put('tent', 33, 15)
c.label(34, 18, '군량 더미 · 사신이 세는 수레 자리', 'zone')
# barracks
c.grid('tent', 7, 28, 5, 3, 3, 3)
c.grid('tent', 28, 28, 5, 3, 3, 3)
c.put('campfire', 18, 31); c.put('campfire', 31, 34)
c.label(12, 31, '병사 막사', 'zone'); c.label(34, 31, '병사 막사', 'zone')
# training ground and kitchen
c.put('weapon_rack', 37, 24); c.put('weapon_rack', 40, 24)
c.put('cauldron', 8, 24); c.put('soup_stove', 11, 24); c.put('water_bucket', 13, 24)
c.label(10, 25, '취사', 'zone')
c.put('prison_cart', 26, 35)
c.label(26, 35, '함거 (3장 노식 압송 때만)', 'zone')
c.put('notice_post', 22, 36)
# trees beyond the camp
c.trees(26, (0, 39, 47, 41), gap=3)
c.trees(18, (0, 11, 3, 41), gap=3)
c.trees(18, (44, 11, 47, 41), gap=3)
c.save()


# ---------------- luzhi_b ----------------
# B: three camps in a line (전영·중군·후영), 광종성 on the left across a stream
c = Camp('S_draft_luzhi_b', '노식 진영 B', 64, 34, seed=11)
c.paint(0, 0, 63, 2, 'i')
c.wall_u(3, 0, 63, gate=12, closed=True)
c.label(12, 1, '광종성', 'far')
c.paint(0, 5, 63, 6, '~')
c.label(40, 5, '해자 삼은 개울', 'zone')
for u in range(2, 62, 2):
    c.put('antler_barrier', u, 8)
c.road_u(19, 0, 63, w=2)
c.label(61, 19, '→ 탁군으로 (출구)', 'exit')
# 전영 (front, toward the gate)
c.fence_box(2, 11, 18, 30, gaps={(18, 19), (18, 20), (10, 11)})
c.paint(3, 12, 17, 18); c.put('watchtower', 3, 12); c.put('watchtower', 17, 12)
c.put('shield_wall', 6, 12); c.put('shield_wall', 9, 12); c.put('shield_wall', 13, 12)
c.grid('tent', 4, 22, 5, 3, 3, 3)
c.put('brazier', 10, 14)
c.label(10, 15, '전영 — 성문을 노려보는 선봉', 'zone')
# 중군
c.fence_box(22, 11, 40, 30, gaps={(22, 19), (22, 20), (40, 19), (40, 20)})
c.paint(25, 12, 37, 18)
c.put('tent_round', 31, 14)
c.put('command_flag', 28, 17); c.put('command_flag', 34, 17); c.put('banner', 25, 13); c.put('banner', 37, 13)
c.put('brazier', 29, 18); c.put('brazier', 33, 18); c.put('map_table', 35, 14)
c.label(31, 14, '중군 · 노식의 본영 ★ 들어감', 'enter')
c.grid('tent', 24, 23, 6, 2, 3, 3)
c.put('weapon_rack', 24, 13); c.put('armor_stand', 26, 13)
c.label(31, 25, '중군 막사', 'zone')
# 후영: supplies, the envoy
c.fence_box(44, 11, 60, 30, gaps={(44, 19), (44, 20), (60, 19), (60, 20)})
c.paint(46, 12, 58, 18)
for u in range(46, 58, 3):
    for v in (13, 15, 17): c.put('grain_sack', u, v)
c.put('cauldron', 47, 23); c.put('soup_stove', 50, 23); c.put('water_jar', 52, 23)
c.grid('tent', 47, 26, 4, 1, 3, 3)
c.put('tent', 56, 23); c.put('lantern', 55, 24)
c.label(52, 15, '후영 · 군량 (사신이 수레를 세는 곳)', 'zone')
c.label(56, 23, '사신의 막사', 'zone')
for u in (20, 42, 62): c.put('watchtower', u, 22)
c.trees(26, (0, 31, 63, 33), gap=3)
c.save()


# ---------------- coalition ----------------
def lord(c, u0, v0, u1, v1, name, gate, enter=None, tents=True, extra=None, note=None):
    """One lord's quarter: low fence with a gate, two banners at the gate, a lord's tent, soldiers' tents."""
    gu, gv = gate
    gaps = {(gu, gv), (gu + (1 if gv in (v0, v1) else 0), gv + (1 if gu in (u0, u1) else 0))}
    c.fence_box(u0, v0, u1, v1, gaps=gaps)
    bu, bv = (1, 0) if gv in (v0, v1) else (0, 1)
    c.put('banner', gu - bu, gv - bv); c.put('banner', gu + 2 * bu, gv + 2 * bv)
    mu, mv = (u0 + u1) // 2, (v0 + v1) // 2
    c.paint(mu - 3, mv - 3, mu + 3, mv + 2)
    c.put('tent_round', mu, mv - 1)
    c.put('command_flag', mu - 2, mv + 1)
    c.put('brazier', mu + 2, mv + 1)
    c.label(mu, mv - 1, name + (' ★ 들어감 (%s)' % enter if enter else ''), 'enter' if enter else 'zone')
    if tents:
        for v in range(v0 + 2, v1 - 1, 3):
            for u in range(u0 + 2, u1 - 1, 3):
                if abs(u - mu) <= 4 and abs(v - mv) <= 4:
                    continue
                if (u, v) in c.used or (abs(u - gu) <= 1 and abs(v - gv) <= 1):
                    continue
                c.put('tent', u, v)
    if extra:
        extra(c, mu, mv)

# ---------- A: 십자 대로, 맹주 본영이 한가운데 ----------
c = Camp('S_draft_coalition_a', '연합 진영 A', 80, 72, seed=21)
c.fence_box(3, 3, 76, 68, gaps={(3, 35), (3, 36), (76, 35), (76, 36), (39, 3), (40, 3), (39, 68), (40, 68)})
for u, v in [(4, 4), (75, 4), (4, 67), (75, 67), (4, 32), (4, 39), (73, 68), (36, 4), (43, 4), (36, 67), (43, 67), (75, 32), (75, 39)]:
    c.put('watchtower', u, v)
c.road_u(35, 0, 79, w=2); c.road_v(39, 0, 71, w=2)
c.label(1, 35, '서문 → 사수관 · 호로관 (출진 방향)', 'gate')
c.label(40, 70, '남문 · 진영 들머리 (출구 자리)', 'exit')
# center: 맹주
c.paint(31, 27, 48, 44)
c.put('tent_round', 39, 34, 16, 0)
for u, v in [(35, 31), (44, 31), (35, 40), (44, 40)]: c.put('command_flag', u, v)
for u, v in [(37, 38), (42, 38)]: c.put('brazier', u, v)
c.put('banner', 38, 43); c.put('banner', 42, 43)
c.label(39, 34, '맹주 원소의 대막사 ★ 들어감 (S_council)', 'enter')
c.label(40, 44, '맹주 앞 광장', 'zone')
# quarters
lord(c, 6, 6, 20, 31, '원소군 (맹주 직속)', (20, 18))
def granary(c, mu, mv):
    for u in range(26, 34, 2):
        for v in (8, 10, 12): c.put('grain_sack', u, v)
    c.label(29, 10, '원술 군량 더미 (수레가 떠나지 않음)', 'zone')
lord(c, 24, 6, 36, 31, '원술 진영', (36, 24), enter='S_tent_lux', tents=False, extra=granary)
c.put('wine_jar', 26, 20); c.put('wine_jar', 27, 22); c.put('table_wine', 31, 18)
lord(c, 43, 6, 56, 31, '조조의 영채', (43, 24), enter='S_tent_mid')
c.paint(60, 8, 73, 18)
for u in (61, 64, 67, 70, 73): c.put('armor_stand', u, 9)
c.put('weapon_rack', 61, 16); c.put('weapon_rack', 64, 16)
c.label(67, 13, '조조군 조련장', 'zone')
lord(c, 60, 21, 73, 31, '공융 · 북해 진영', (66, 31))
lord(c, 6, 39, 20, 51, '손견 진영 (사수관 쪽)', (13, 39))
c.put('medicine_pot', 9, 49); c.put('bloody_cloth', 11, 49)
lord(c, 6, 55, 20, 65, '하내 왕광 진영', (20, 60))
lord(c, 24, 39, 36, 65, '상당 장양 진영', (36, 56))
lord(c, 43, 39, 57, 65, '공손찬 진영 · 유비 삼형제 막사', (43, 56), enter='S_tent')
# market by the south gate
c.paint(60, 42, 74, 64)
for u, v, n in [(62, 48, 'market_stall'), (66, 48, 'market_stall_cloth'), (70, 48, 'market_stall_jar'), (62, 54, 'market_stall_jar'), (66, 54, 'market_stall'), (70, 54, 'market_stall_cloth')]:
    c.put(n, u, v)
c.put('wine_jar', 63, 60); c.put('grain_sack', 67, 60); c.put('bench', 71, 60); c.put('campfire', 68, 58)
c.label(66, 51, '종군 상인 거리 (도구상)', 'zone')
c.trees(40, (0, 0, 79, 71), avoid=lambda u, v: 2 <= u <= 77 and 2 <= v <= 69, gap=3)
c.save()

# ---------- B: 앞줄은 싸우는 자, 뒷줄은 노는 자 ----------
c = Camp('S_draft_coalition_b', '연합 진영 B', 104, 40, seed=31)
c.paint(0, 0, 103, 3, '~')
c.label(60, 1, '강 (진영 등 뒤를 막음)', 'far')
c.road_u(19, 0, 103, w=3)
c.label(1, 20, '← 사수관 · 호로관 (출진 방향)', 'gate')
c.label(102, 20, '진영 들머리 →', 'exit')
for v in range(6, 36, 2): c.put('antler_barrier', 1, v)
c.put('watchtower', 3, 16); c.put('watchtower', 3, 23)
# front: those who fight
lord(c, 5, 6, 19, 17, '손견 진영 (맨 앞, 비어 감)', (12, 17), tents=True)
c.put('medicine_pot', 8, 15); c.put('bloody_cloth', 10, 15)
lord(c, 5, 23, 19, 36, '공손찬 진영 · 유비 삼형제 막사', (12, 23), enter='S_tent')
lord(c, 22, 6, 36, 17, '조조의 영채', (29, 17), enter='S_tent_mid')
lord(c, 22, 23, 36, 36, '공융 · 북해 진영', (29, 23))
lord(c, 39, 6, 51, 17, '상당 장양 진영', (45, 17))
lord(c, 39, 23, 51, 36, '하내 왕광 진영', (45, 23))
# rear: those who feast
def granary(c, mu, mv):
    for u in range(56, 66, 2):
        for v in (28, 30, 32, 34): c.put('grain_sack', u, v)
    c.label(61, 31, '원술 군량 (산처럼 쌓임)', 'zone')
lord(c, 54, 23, 72, 37, '원술 진영', (63, 23), enter='S_tent_lux', tents=False, extra=granary)
lord(c, 54, 5, 72, 17, '원소군 (맹주 직속)', (63, 17))
c.paint(75, 5, 98, 37)
c.fence_box(76, 6, 97, 36, gaps={(76, 20), (76, 21), (76, 19)})
c.put('tent_round', 88, 20, 16, 0)
for u, v in [(82, 15), (82, 26), (94, 15), (94, 26)]: c.put('command_flag', u, v)
for u, v in [(80, 18), (80, 23)]: c.put('brazier', u, v)
for u, v in [(85, 10), (89, 10), (93, 10)]: c.put('table_wine', u, v)
c.put('wine_jar', 86, 31); c.put('wine_jar', 90, 31); c.put('lantern', 78, 17); c.put('lantern', 78, 24)
for u, v in [(77, 7), (96, 7), (77, 35), (96, 35)]: c.put('watchtower', u, v)
c.label(88, 20, '맹주 원소의 대막사 ★ 들어감 (S_council) — 가장 안쪽', 'enter')
c.label(88, 10, '술상', 'zone')
for u in range(80, 96, 3):
    c.put('tent', u, 33)
for v in range(9, 31, 3):
    c.put('tent', 95, v)
c.label(95, 22, '원소 친위대 막사', 'zone')
c.paint(77, 19, 85, 21, 'C')
for u, v in [(84, 12), (92, 12), (84, 28), (92, 28)]: c.put('banner', u, v)
c.put('weapon_rack', 79, 30); c.put('armor_stand', 79, 32); c.put('lantern', 86, 24); c.put('lantern', 90, 24)
c.label(81, 20, '붉은 깔개 길', 'zone')
c.paint(56, 19, 72, 21)
for u, v, n in [(99, 26, 'market_stall'), (99, 30, 'market_stall_cloth'), (101, 34, 'market_stall_jar')]:
    c.put(n, u, v)
c.label(100, 30, '종군 상인', 'zone')
c.trees(20, (0, 37, 103, 39), gap=3)
c.save()
