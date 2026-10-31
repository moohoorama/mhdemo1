#!/usr/bin/env python3
"""Chapter 11-12 proposal battle maps: writes map11.txt/map12.txt and map11.json/map12.json here."""
import json, sys
from pathlib import Path
HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE.parents[1] / 'tools/mapgen'))
import mapgen


def grid(w, h, base='g--'):
    return [[base] * w for _ in range(h)]


def put(g, cells, code):
    for x, y in cells:
        g[y][x] = code


def path(points):
    """Cells along straight runs between the points (each run moves along one axis)."""
    out = []
    for (x0, y0), (x1, y1) in zip(points, points[1:]):
        if x0 == x1:
            s = 1 if y1 >= y0 else -1
            out += [(x0, y) for y in range(y0, y1 + s, s)]
        else:
            s = 1 if x1 >= x0 else -1
            out += [(x, y0) for x in range(x0, x1 + s, s)]
    return out


def save(name, g, note):
    txt = HERE / f'{name}.txt'
    txt.write_text('\n'.join(note) + '\n' + '\n'.join(' '.join(r) for r in g) + '\n', encoding='utf-8')
    doc = mapgen.make(mapgen.read(txt))
    (HERE / f'{name}.json').write_text(json.dumps(doc, ensure_ascii=False, indent=1) + '\n', encoding='utf-8')


# ---- B11 북해 (16 x 15) -------------------------------------------------------------------------------
# 1 길(남쪽 모퉁이) -> 2 성 한 칸 -> 3 길 -> 4 여울 -> 5 길 -> 6 황건 -> 7 북해성(북동쪽, 성벽과 닫힌 성문)
W, H = 16, 15
g = grid(W, H)
# 가운데 강(동서로), 여울
put(g, [(x, y) for x in range(W) for y in (7, 8)], 'w--')
put(g, [(10, 7), (11, 7), (10, 8), (11, 8)], 's--')
# 북해성: 성벽 줄 y=3, 닫힌 성문 (7,3), 안쪽 성내
put(g, [(x, y) for x in range(4, 10) for y in range(0, 3)], 'i--')
put(g, [(3, y) for y in range(0, 4)] + [(10, y) for y in range(0, 4)] + [(x, 3) for x in range(3, 11)], 'c--')
put(g, [(7, 3)], 'm--')
# 길: 남쪽 모퉁이 -> 성 한 칸 -> 여울 -> 성문 앞
put(g, path([(15, 14), (13, 14), (13, 12), (10, 12), (10, 9)]), 'l--')
put(g, path([(11, 6), (11, 5), (7, 5), (7, 4)]), 'l--')
put(g, [(10, 11)], 'go-')
# 남서쪽 구릉(증원이 나타나는 곳)
put(g, [(4, 13), (5, 13), (6, 13), (7, 13), (4, 14), (5, 14), (6, 14), (7, 14), (5, 12), (6, 12)], 'u--')
# 성문 앞 짓밟힌 땅과 관해의 진채
put(g, [(5, 4), (6, 4), (8, 4), (9, 4), (6, 6), (9, 6), (4, 5)], 'd--')
put(g, [(9, 5)], 'dk-')
# 숲과 황무지
put(g, [(1, 5), (2, 6), (0, 4), (13, 4), (14, 2), (12, 1), (15, 5), (1, 10), (3, 12), (8, 10), (0, 13),
        (5, 10), (9, 13), (2, 1), (1, 2)], 'gt-')
put(g, [(4, 11), (8, 12), (2, 9), (13, 5)], 'd--')
save('map11', g, ['# B11 북해 (16x15): 남쪽 모퉁이 길 -> 성(10,11) -> 여울(10~11,7~8) -> 길 -> 황건 -> 북해성(성벽 y=3, 닫힌 성문 7,3).',
                  '# 남서쪽 구릉(4~7,12~14)에서 증원 다섯이 나온다.'])

# ---- B12 서주 (24 x 22) -------------------------------------------------------------------------------
W, H = 24, 22
g = grid(W, H)
# 북쪽 모퉁이에서 남쪽 서주성까지 내려가는 길
road = path([(0, 1), (4, 1), (4, 6), (9, 6), (9, 11), (13, 11), (13, 16), (18, 16), (18, 19), (20, 19)])
put(g, road, 'l--')
# 개울: 서쪽 가장자리에서 동쪽으로, 길에 다리, 서쪽에 여울
creek = [(x, 14) for x in range(0, 6)] + [(x, 15) for x in range(5, 13)] + [(x, 14) for x in range(12, 18)] + \
        [(17, 13), (18, 13), (19, 13)]
put(g, creek, 'w--')
put(g, [(13, 15)], 'wb-')
put(g, [(2, 14), (3, 14)], 's--')
put(g, [(13, 14)], 'wb-')
# 불탄 마을과 재
put(g, [(6, 3)], 'dve')
put(g, [(11, 8)], 'dve')
put(g, [(5, 4), (7, 3), (6, 2), (10, 8), (12, 8), (11, 9)], 'd-s')
# 고순이 지키는 동쪽 마을(목책 반쯤)
put(g, [(20, 9)], 'gv-')
put(g, [(19, 8), (19, 10), (21, 8)], 'gz-')
# 선비가 숨은 서쪽 대숲
put(g, [(1, 10), (2, 10), (1, 11), (3, 11), (2, 12), (0, 12), (1, 13), (4, 10)], 'gt-')
# 그 밖의 나무와 황무지
put(g, [(8, 1), (12, 2), (16, 1), (21, 3), (22, 14), (20, 17), (16, 20), (9, 19), (5, 18), (2, 20), (15, 7), (7, 9),
        (22, 6), (17, 4), (11, 18), (7, 16)], 'gt-')
put(g, [(14, 4), (15, 4), (16, 5), (18, 2), (8, 12), (6, 17), (11, 20), (21, 12)], 'd--')
# 서주성 하나
put(g, [(20, 19)], 'go-')
save('map12', g, ['# B12 서주 (24x22): 북쪽 모퉁이에서 남쪽 서주성(20,19)까지 내려가는 피난길. 개울 다리(13,14~15), 서쪽 여울(2~3,14).',
                  '# 고순의 마을(20,9) 동쪽, 선비가 숨은 대숲(0~4,10~13) 서쪽, 불탄 마을 둘.'])
