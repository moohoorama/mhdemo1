#!/usr/bin/env python3
"""시안: S_beihai, 포위가 풀린 이튿날의 북해성 안(12장 탐험, 서주 전투 전 정비). plan/ch11_12/S_beihai.json에 쓴다.

형식 2(internal/scenemap). tools/scene_maps/xindu.py와 같은 짜임: 성벽이 북서·남서쪽, 성문은 남서쪽 벽에 있고
해자 다리(12장 S#1)에서 들어온다. 정면 큰길 끝이 공융의 부중, 동쪽에 학당(공자의 후손이 다스리는 고을),
북서쪽 성벽 아래에 포위 동안 들어온 피난민이 거적을 깔고 죽을 받는다. 가로 거리에 저잣거리와 상점.
"""
import json
from pathlib import Path

W, H, M = 32, 28, 12
MAIN = (13, 16)    # 성문에서 부중까지 큰길(u)
CROSS = (14, 16)   # 가로 거리(v)
GATE = 14

g = [['.'] * (W + 2 * M) for _ in range(H + 2 * M)]
obj = []


def paint(u0, v0, u1, v1, ch):
    for v in range(max(-M, v0), min(H + M - 1, v1) + 1):
        for u in range(max(-M, u0), min(W + M - 1, u1) + 1):
            g[v + M][u + M] = ch


def put(name, u, v):
    obj.append([name, u, v])


paint(-M, -M, -2, H + M, 'd')          # 북서쪽 성벽 밖 들판
paint(-M, H + 1, W + M, H + M, 'd')    # 남서쪽 성벽 밖(해자 쪽 길)
paint(MAIN[0], 0, MAIN[1], H + M, 'd')
paint(0, CROSS[0], W + M, CROSS[1], 'd')
paint(9, 3, 20, 8, 'i')                # 부중 마당
paint(21, 3, 29, 9, 'i')               # 학당 뜰

# 공융의 부중
put('hall_u', 14, 2), put('banner', 10, 4), put('banner', 19, 4), put('lantern', 12, 8), put('lantern', 17, 8)
put('notice_board', 18, 10)
# 학당: 정자, 태호석, 버들, 서안
put('pavilion', 25, 5), put('taihu_rock', 22, 7), put('stree_willow', 28, 2), put('desk', 24, 8), put('incense', 27, 8)
# 북서쪽 성벽 아래 피난민 거처: 거적, 오두막, 죽 쑤는 솥
put('hut', 2, 4)
for u, v in ((1, 8), (3, 9), (1, 11), (4, 12), (2, 20), (5, 22)):
    put('straw_mat', u, v)
put('soup_stove', 6, 9), put('cauldron', 7, 11), put('water_jar', 5, 7), put('grain_sack', 8, 8)
# 가로 거리: 주점, 저잣거리, 무기점(대장간 앞 병기 걸이), 도구점
put('tavern', 5, 17), put('wine_jar', 2, 18), put('bench', 9, 18)
put('house_tile', 22, 12), put('market_stall_jar', 19, 13), put('market_stall_cloth', 25, 13)
put('house_tile_x', 28, 18), put('weapon_rack', 26, 17), put('armor_stand', 30, 16)
put('market_stall', 10, 12), put('grain_cart_v', 11, 18)
# 남쪽 민가와 우물
put('house_tile_x', 7, 23), put('house_thatch', 22, 22), put('house_thatch_x', 27, 23), put('well', 19, 21),
put('water_bucket', 20, 22), put('stree_broad_0', 4, 26)
# 성문 안쪽: 포위 때 쌓은 흙부대와 꺾인 화살(가볍게)
put('grain_sack', 11, 26), put('grain_sack', 18, 26), put('broken_spear', 17, 24)
# 성 밖 들판의 나무
for u, v, name in ((6, -7, 'stree_pine_0'), (20, -9, 'stree_broad_1'), (37, 3, 'stree_broad_0'), (39, 17, 'stree_pine_1'),
                   (-7, 6, 'stree_pine_1'), (-6, 22, 'stree_broad_0'), (26, 35, 'stree_old')):
    put(name, u, v)
put('sgate_open', GATE, H)

art = {'id': 'S_beihai', 'name': '북해성 안', 'format': 2, 'size': [W // 2, H // 2], 'origin': [-M, -M],
       'ground': [''.join(r) for r in g], 'objects': obj,
       'walls': [['swall', [-1, -M], [-1, H]], ['swall', [0, H], [W + M - 1, H]]]}
Path(__file__).with_name('S_beihai.json').write_text(json.dumps(art, ensure_ascii=False) + '\n', encoding='utf-8')
print('S_beihai', art['size'], len(obj), 'objects')
