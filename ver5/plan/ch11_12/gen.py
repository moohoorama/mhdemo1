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

# ---- B12 서주 (24 x 24) -------------------------------------------------------------------------------
# 사수가 화면의 ㄱ자 바깥(위쪽 띠와 오른쪽 띠)과 안쪽(왼쪽 아래, 서주성)을 가른다. 물길은 1:1 계단으로 곧게 긋지 않고,
# 서쪽은 맵 끝에서 서쪽 다리까지 곧게, 가운데는 2~3칸으로 두껍게 굽이치고, 동쪽 다리 아래에서 꺾여 동쪽 끝까지 곧게 간다.
W, H = 24, 24
g = grid(W, H)


def inside(u, v):
    return 0 <= u < W and 0 <= v < H


def brush(u, v, n=2):
    return [(u + i, v + j) for i in range(n) for j in range(n) if inside(u + i, v + j)]


river = []
river += [(u, v) for u in range(0, 7) for v in (13, 14)]                  # 서쪽: 맵 끝에서 서쪽 다리까지 곧게
for u, v, n in ((6, 12, 2), (7, 11, 2), (8, 10, 3), (9, 9, 3), (10, 8, 2), (10, 7, 2), (11, 6, 2), (11, 5, 2),
                (12, 4, 2), (13, 4, 2), (14, 5, 2), (14, 6, 2), (15, 7, 2)):
    river += brush(u, v, n)                                                 # 가운데: 두껍게 굽이침
river += [(u, v) for u in (16, 17) for v in range(8, 14)]                  # 동쪽 다리 둘레: 세로로 곧게
river += [(u, v) for u in range(16, W) for v in (12, 13)]                  # 동쪽 다리 아래에서 동쪽 끝까지 곧게

# 북쪽 꼭대기에 나타나는 조인과 유비군 사이를 가르는 산줄기(맵 끝에서 강 바로 위까지), 강에 닿는 끝은 구릉으로 튼다
put(g, [(u, v) for u in range(W) for v in range(H) if u - v in (2, 3) and u + v <= 14], 'n--')  # 맵 끝부터: 조인은 서쪽으로만 내려간다
put(g, [(8, 6)], 'u--')
# 길: 아군 출발 -> 동쪽 다리 / 위쪽 띠를 따라 서쪽 다리 / 다리들 -> 서주성 두 문
put(g, path([(16, 2), (19, 2), (19, 10), (18, 10)]), 'l--')
put(g, path([(8, 3), (8, 4), (9, 4), (9, 5), (5, 5), (5, 11), (4, 11), (4, 12)]), 'l--')
put(g, path([(15, 10), (14, 10), (14, 15), (20, 15), (20, 16)]), 'l--')
put(g, path([(4, 15), (4, 17), (10, 17), (10, 20), (16, 20)]), 'l--')
# 서주성: 북서·북동 성벽(성곽), 관문 둘, 성 안 성내와 성(관아)
put(g, [(u, v) for u in range(18, 24) for v in range(18, 24)], 'i--')
put(g, [(17, v) for v in range(17, 24)] + [(u, 17) for u in range(17, 24)], 'c--')
put(g, [(17, 20)], 'ip-')
put(g, [(20, 17)], 'ip-')
put(g, [(21, 21)], 'io-')
put(g, [(19, 22), (22, 19)], 'iv-')
# 고순이 기대는 마을(동쪽 다리 머리), 제갈현이 기대는 마을(서쪽 다리 건너)
put(g, [(15, 10)], 'gv-')
put(g, [(5, 16)], 'gv-')
# 짓밟힌 바깥 고을과 그 밖
put(g, [(8, 2)], 'dve')
put(g, [(13, 2)], 'dve')
put(g, [(9, 1), (12, 1), (14, 3), (7, 3)], 'd-s')
put(g, [(6, 18), (7, 17)], 'u--')
put(g, [(0, 9), (12, 0), (16, 0), (20, 4), (23, 7), (21, 10), (19, 15), (23, 15), (11, 12), (13, 14), (8, 21),
        (11, 22), (14, 23), (9, 15), (2, 21), (5, 20), (14, 18), (2, 10), (0, 16), (22, 1), (1, 6)], 'gt-')
put(g, [(11, 2), (18, 6), (21, 8), (10, 19), (12, 16), (3, 18)], 'd--')
# 물은 맨 나중에 깐다(나무가 물길을 끊지 않게), 다리는 그 위에
put(g, river, 'w--')
put(g, [(16, 10), (17, 10)], 'wb-')
put(g, [(4, 13), (4, 14)], 'wb-')
save('map12', g, ['# B12 서주 (24x24): 사수가 ㄱ자 바깥(위·오른쪽 띠)과 서주성 쪽을 가른다. 서쪽은 곧게, 가운데는 두껍게, 동쪽 다리 아래는 동쪽 끝까지 곧게.',
                  '# 동쪽 다리 (16~17,10) 건너 고순의 마을 (15,10), 서쪽 다리 (4,13~14) 건너 제갈현의 마을 (5,16), 남쪽 서주성(성곽).'])
