"""대장간 마당: 대장간이 성 안의 앞이 트인 헛간이 되면서(11-28), 들어가는 곳도 실내가 아니라 바깥 마당이다.
지역마다 하나(S_smithy_yard_<지역>). 옛 실내 대장간 S_weapon_shop과 같은 좌표틀이라 대장장이(9,3)·견습공(5,7)·
들어선 칸(6,6)과 2장 장면 c02-e-smith(유비 → [4,3])가 그대로 맞는다. 마당은 앞(+u)으로 넓혀 길이 나간다.

  python3 smithy_yard.py ROOT REGION...   → 맵을 쓰고 index.json에 올리고, 탐험 대장간 지역·장면을 이 맵으로
"""
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
from towngen import place_obj  # noqa: E402

TOWN_REGION = {'S_zhuo': 'huabei', 'S_pingyuan': 'huabei', 'S_beihai': 'huabei', 'S_ye': 'huabei', 'S_xindu': 'huabei',
               'S_xudu': 'zhongyuan', 'S_luoyang': 'zhongyuan', 'S_xuzhou_town': 'huainan', 'S_xiapi': 'huainan'}
EXIT = [15, 7]          # 마당 앞(+u) 끝으로 나가는 길


def yard(region):
    ox, oy, W, H = -3, -3, 22, 18          # 둘레 3칸은 울타리 밖 풀밭·나무
    g = [['.'] * W for _ in range(H)]

    def put(cells, c):
        for u, v in cells:
            if 0 <= u - ox < W and 0 <= v - oy < H:
                g[v - oy][u - ox] = c
    put({(u, v) for u in range(0, 14) for v in range(3, 11)}, 'd')        # 마당 흙바닥
    put({(u, v) for u in range(6, 14) for v in range(0, 3)}, 'd')         # 헛간 옆 일터
    put({(u, v) for u in range(13, W + ox) for v in (7, 8)}, 'd')         # 나가는 길
    objs = [place_obj(f'{region}_smithy', 0, 0, 5, 3),
            ['brazier', 8, 2, 0, -6], ['water_bucket', 10, 2, 0, -6], ['weapon_rack', 7, 0, 0, 0],
            ['weapon_rack', 9, 0, 0, 0], ['weapon_rack', 11, 0, 0, 0], ['armor_stand', 12, 2, 0, -4],
            ['grain_sack', 1, 4, 0, -8], ['grain_sack', 2, 4, 4, -6], ['cauldron', 7, 4, 0, -8],
            ['grain_cart', 12, 10, 0, -8], place_obj(f'{region}_cottage_4_r', 0, 6, 2, 3),
            ['weapon_rack', 13, 4, 0, 0], ['water_bucket', 3, 10, 0, -6]]
    trees = [['stree_broad_0', -2, 2, 0, -8], ['stree_pine_0', -2, 7, 0, -8], ['stree_broad_1', 4, -2, 0, -8],
             ['stree_pine_1', 9, -2, 0, -8], ['stree_broad_0', 16, -1, 0, -8], ['stree_bush_0', 16, 3, 0, -8],
             ['stree_bush_1', 16, 11, 0, -8], ['stree_broad_1', 6, 13, 0, -8], ['stree_bush_0', -2, 12, 0, -8]]
    fam = f'{region}_fence'
    walls = [[fam, [-1, -1], [14, -1]], [fam, [-1, 0], [-1, 11]], [fam, [0, 11], [14, 11]],
             [fam, [14, 0], [14, 6]], [fam, [14, 9], [14, 10]]]
    return {'id': f'S_smithy_yard_{region}', 'name': '대장간 마당', 'format': 2, 'size': [8, 6], 'origin': [ox, oy],
            'ground': [''.join(r) for r in g], 'objects': objs + trees, 'walls': walls}


def main(root, regions):
    g = os.path.join(root, 'assets/graphics')
    idx = json.load(open(os.path.join(g, 'index.json')))
    for r in regions:
        m = yard(r)
        json.dump(m, open(os.path.join(g, 'maps', m['id'] + '.json'), 'w'), ensure_ascii=False, indent=1)
        if f'maps/{m["id"]}.json' not in idx['maps']:
            idx['maps'].append(f'maps/{m["id"]}.json')
    json.dump(idx, open(os.path.join(g, 'index.json'), 'w'), ensure_ascii=False, indent=1)
    notes = []
    scn = os.path.join(root, 'assets/scenario/hero')
    for f in sorted(os.listdir(scn)):
        if not f.startswith('explore'):
            continue
        p = os.path.join(scn, f)
        d = json.load(open(p))
        A = d.get('Areas') or {}
        hit = False
        for k, a in A.items():
            if a.get('Map') != 'S_weapon_shop':
                continue
            town = next((t for t in A.values() if any(e.get('To') == k for e in t.get('Exits') or [])), None)
            r = TOWN_REGION.get((town or {}).get('Map'), 'huabei')
            a['Map'] = f'S_smithy_yard_{r}'
            a['Name'] = '대장간 마당'
            for e in a.get('Exits') or []:
                e['At'] = list(EXIT)
            hit = True
            notes.append(f'{f}/{k} → S_smithy_yard_{r}')
        if hit:
            json.dump(d, open(p, 'w'), ensure_ascii=False, indent=1)
    for f in sorted(os.listdir(os.path.join(root, 'assets/scenes'))):
        p = os.path.join(root, 'assets/scenes', f)
        if not f.endswith('.json'):
            continue
        d = json.load(open(p))
        hit = False
        for sid, sc in (d.get('Scenes') or {}).items():
            if sc.get('Map') == 'S_weapon_shop':
                sc['Map'] = 'S_smithy_yard_huabei'
                hit = True
                notes.append(f'scene {sid} → S_smithy_yard_huabei')
        if hit:
            json.dump(d, open(p, 'w'), ensure_ascii=False, indent=1)
    return notes


if __name__ == '__main__':
    for n in main(sys.argv[1], sys.argv[2:] or ['huabei', 'zhongyuan', 'huainan']):
        print(n)
