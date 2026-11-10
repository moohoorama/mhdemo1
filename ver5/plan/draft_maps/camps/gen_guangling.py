"""광릉 진영 시안 (2026-11-09): 하비를 잃고 물러난 유비군이 굶주리는 진영, 15장 S#1 `c15-grain`의 무대.

  python3 ver5/plan/draft_maps/camps/gen_guangling.py   -> guangling_a.json, guangling_b.json (S_draft_guangling_*)

Uses the camp kit of gen_camps.py (the Camp class). Coordinates are small tiles from origin 0, 0; the top (small u+v)
is the far side of the screen."""
from pathlib import Path

src = (Path(__file__).resolve().parent / 'gen_camps.py').read_text()
exec(src.split('# ---------------- militia ----------------')[0])

BURNT = ['burnt_oak_0', 'burnt_oak_1', 'burnt_oak_2']

# ---------- A: 강가의 패잔 진영 ----------
# the 한구 canal behind, a palisade only half put up, the grain cart nearly bare at the near-left, the mouth at the
# right where the road comes in and 미방's merchants park their carts outside
c = Camp('S_draft_guangling_a', '광릉 진영 A', 36, 28, seed=41)
c.paint(0, 0, 35, 3, '~')
c.label(18, 1, '한구 물길 (진영 등 뒤)', 'far')
c.put('boat_v', 7, 2); c.put('boat_v', 14, 1)
c.label(10, 2, '타고 온 배', 'zone')
c.road_u(13, 3, 35, w=2)
c.label(34, 13, '광릉으로 가는 길 → 진영 들머리', 'exit')
# palisade: only the side toward the road, logs missing
c.fence_line(26, 6, 26, 25, gaps={(26, 12), (26, 13), (26, 14), (26, 15), (26, 9), (26, 20), (26, 21)})
c.fence_line(3, 25, 26, 25, gaps={(9, 25), (10, 25), (17, 25)})
c.put('banner', 26, 11); c.put('banner', 26, 16)
c.put('watchtower', 25, 6)
c.label(26, 14, '들머리 (미방이 들어오는 곳)', 'gate')
# grain line: the cart nearly empty, sacks folded flat, the line of soldiers along the road
c.paint(3, 10, 9, 16)
c.put('grain_cart', 5, 11); c.put('grain_sack', 7, 10); c.put('water_jar', 3, 15)
c.label(6, 12, '군량 수레 (미축이 한 되씩)', 'enter')
c.label(14, 14, '줄 선 병사들', 'zone')
# tents along the bank, few and torn
c.row_u('tent', 4, 16, 6, step=4)
c.put('tent_round', 21, 8); c.put('banner', 23, 9); c.put('lantern', 19, 9)
c.label(21, 8, '유비의 군막 (S#2 장면)', 'zone')
# men sleeping in the open
c.grid('straw_mat', 6, 18, 4, 2, 3, 3)
c.put('campfire', 10, 21)
c.put('medicine_pot', 18, 22); c.put('bloody_cloth', 19, 20); c.put('fallen_flag', 22, 22); c.put('broken_spear', 23, 19)
c.label(12, 20, '천막 없이 거적에서 자는 병사', 'zone')
# kitchen with nothing in it
c.put('cauldron', 15, 10); c.put('soup_stove', 17, 11)
c.label(16, 10, '빈 솥', 'zone')
# merchants outside the mouth
c.put('grain_cart_v', 30, 11); c.put('market_stall_cloth', 31, 17); c.put('market_stall_jar', 34, 17); c.put('grain_sack', 29, 16)
c.label(31, 15, '미방이 부른 상인들 (첩자가 섞임)', 'zone')
c.trees(8, (0, 26, 35, 27), kinds=['stree_bush_0', 'stree_bush_1'], gap=3)
c.trees(7, (28, 4, 35, 27), avoid=lambda u, v: 8 <= v <= 20, gap=3)
c.trees(4, (0, 4, 2, 25), gap=3)
c.save()

# ---------- B: 불탄 마을 터 ----------
# the army camps in a village burnt in the fighting: tents between the shells of houses, the grain handed out at the
# well in the square, the merchants setting up where the road comes in
c = Camp('S_draft_guangling_b', '광릉 진영 B', 36, 28, seed=43)
c.road_u(13, 0, 35, w=2)
c.label(34, 13, '광릉으로 가는 길 →', 'exit')
c.label(1, 13, '← 하비 쪽 (패잔병이 들어오는 길)', 'gate')
c.paint(10, 9, 21, 18)
c.put('well', 15, 11); c.put('water_bucket', 16, 12)
c.put('grain_cart', 12, 11); c.put('grain_sack', 13, 10)
c.label(13, 11, '우물가 군량 수레 (미축이 한 되씩)', 'enter')
c.label(16, 16, '마을 광장 · 줄 선 병사들', 'zone')
for u, v, n in [(5, 6, 'house_burnt'), (24, 6, 'house_burnt'), (5, 21, 'house_burnt'), (25, 21, 'burnt_hut'), (11, 22, 'burnt_hut')]:
    c.put(n, u, v)
c.label(5, 6, '불탄 집', 'zone')
for i, (u, v) in enumerate([(8, 9), (21, 8), (9, 19), (20, 20), (28, 9)]):
    c.put('rubble_%d' % i, u, v)
for i, (u, v) in enumerate([(3, 10), (18, 4), (29, 18), (15, 24)]):
    c.put('charred_%d' % i, u, v)
# tents pitched between the ruins
for u, v in [(12, 5), (16, 5), (18, 21), (22, 24), (30, 23)]:
    c.put('tent', u, v)
c.put('tent_round', 25, 15); c.put('banner', 23, 16); c.put('lantern', 27, 17)
c.label(25, 15, '유비의 군막 (S#2 장면)', 'zone')
c.grid('straw_mat', 2, 16, 2, 2, 3, 3)
c.put('campfire', 8, 15); c.put('cauldron', 19, 10)
c.put('medicine_pot', 3, 22); c.put('bloody_cloth', 4, 24)
c.label(4, 18, '거적 · 부상병', 'zone')
# merchants setting up at the road's mouth
c.put('market_stall', 31, 10); c.put('market_stall_jar', 33, 17); c.put('grain_cart_v', 30, 16); c.put('wine_jar', 34, 10)
c.label(32, 14, '미방이 부른 상인들 (첩자가 섞임)', 'zone')
c.trees(10, (0, 0, 35, 3), kinds=BURNT + TREES[:3], gap=3)
c.trees(10, (0, 25, 35, 27), kinds=BURNT + TREES, gap=3)
c.save()
