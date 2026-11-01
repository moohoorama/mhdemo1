#!/usr/bin/env python3
"""12장 불탄 마을 시안 장면 맵(형식 2): 조조군이 지나간 서주 길가 마을. S_xuzhou_ruin.json을 여기에 쓴다."""
import json
from pathlib import Path

W, H, PAD = 14, 10, 12          # 게임 칸, 여백(작은 칸)
SW, SH = W * 2, H * 2           # 작은 칸
cols, rows = SW + 2 * PAD, SH + 2 * PAD
g = [['.'] * cols for _ in range(rows)]


def dirt(u0, v0, u1, v1):
    for v in range(v0, v1 + 1):
        for u in range(u0, u1 + 1):
            if -PAD <= u < SW + PAD and -PAD <= v < SH + PAD:
                g[v + PAD][u + PAD] = 'd'


dirt(-PAD, 9, SW + PAD - 1, 10)        # 동서로 지나는 길(서쪽에서 들어온다)
dirt(13, 10, 14, SH + PAD - 1)         # 남쪽 서주로 가는 갈림길
for u0, v0, u1, v1 in [(3, 2, 9, 7), (10, 1, 15, 7), (17, 2, 23, 7), (5, 12, 11, 17), (15, 12, 20, 17)]:
    dirt(u0, v0, u1, v1)               # 집터 둘레의 그을린 흙

obj = []
# 불탄 집과 오두막, 무너진 잔해
obj += [['house_burnt', 6, 5], ['house_burnt', 13, 4], ['house_burnt', 20, 5], ['burnt_hut', 8, 15],
        ['house_burnt', 18, 15], ['burnt_hut', 24, 3]]
obj += [['rubble_0', 9, 6], ['rubble_2', 15, 6], ['rubble_3', 4, 3], ['rubble_1', 22, 7], ['rubble_4', 11, 15],
        ['rubble_2', 16, 13]]
obj += [['charred_0', 10, 3], ['charred_2', 17, 3], ['charred_1', 6, 13], ['charred_3', 20, 13], ['charred_4', 3, 7]]
obj += [['ash_2_0', 7, 7], ['ash_3_1', 12, 6], ['ash_1_1', 19, 7], ['ash_2_2', 9, 12], ['ash_3_0', 17, 12],
        ['ash_1_2', 22, 9]]
# 짓밟힌 밭(북서·남동), 부서진 울타리
obj += [['field_bare', 1, 1], ['field_bare', 3, 0], ['field_trampled', 0, 4],
        ['field_bare', 23, 13], ['field_bare', 25, 15], ['field_trampled', 22, 17], ['field_bare', 26, 18],
        ['field_bare', 2, 15], ['field_bare', 1, 18]]
# 불탄 밭: 밭 위의 재와 숯
obj += [['ash_3_2', 1, 2], ['charred_1', 0, 4], ['ash_2_1', 23, 14], ['charred_2', 22, 17], ['ash_3_0', 25, 16],
        ['ash_2_0', 2, 16], ['charred_0', 1, 19]]
obj += [['sfence_05', u, 8] for u in (0, 1, 2, 5)] + [['sfence_05', u, 12] for u in (22, 23, 26, 27)]
# 죽은 나무, 탄 나무
obj += [['stree_dead', 1, 12], ['stree_dead', 26, 6], ['stree_dead', 11, 18], ['burnt_oak_0', 27, 10],
        ['burnt_oak_1', 3, 19], ['burnt_oak_2', 25, 1]]
# 싸움이 아니라 학살의 흔적: 주검을 덮은 피 묻은 천, 꺾인 창, 그을린 깃발, 빈 우물, 뒤집힌 물동이
obj += [['bloody_cloth', 5, 9], ['bloody_cloth', 24, 10], ['bloody_cloth', 10, 13], ['broken_spear', 4, 12],
        ['broken_spear', 25, 8], ['rubble_1', 12, 17], ['bloody_cloth', 11, 8], ['bloody_cloth', 16, 8], ['bloody_cloth', 7, 11], ['bloody_cloth', 20, 11],
        ['bloody_cloth', 13, 12], ['broken_spear', 9, 8], ['broken_spear', 18, 11], ['fallen_flag', 14, 8],
        ['well', 21, 9], ['water_bucket', 22, 10], ['water_jar', 5, 11]]

doc = {'id': 'S_xuzhou_ruin', 'name': '불탄 마을(서주)', 'format': 2, 'size': [W, H], 'origin': [-PAD, -PAD],
       'ground': [''.join(r) for r in g], 'objects': obj}
Path(__file__).with_name('S_xuzhou_ruin.json').write_text(json.dumps(doc, ensure_ascii=False) + '\n', encoding='utf-8')
