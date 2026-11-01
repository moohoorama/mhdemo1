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
# 화면 좌표 x = u - v(오른쪽 +), y = u + v(아래 +). 사수가 화면에서 ㄱ자 바깥(위쪽 띠와 오른쪽 띠)을 맵의 꼭 절반으로
# 갈라낸다: y 17~18의 가로 물줄기, x 6~7의 세로 물줄기. 안쪽(왼쪽 아래)에 서주성.
W, H = 24, 24
Y0, X0 = 17, 6
g = grid(W, H)


def at(x, y):
    return ((x + y) // 2, (y - x) // 2)


def inside(u, v):
    return 0 <= u < W and 0 <= v < H


for u in range(W):
    for v in range(H):
        x, y = u - v, u + v
        if (y in (Y0, Y0 + 1) and x <= X0 + 1) or (x in (X0, X0 + 1) and y >= Y0):
            g[v][u] = 'w--'
# 다리 둘: 동쪽(세로 물줄기, 고순 쪽)과 서쪽(가로 물줄기, 제갈현 쪽)
put(g, [(16, 10), (17, 10)], 'wb-')
put(g, [(4, 13), (4, 14)], 'wb-')
# 북쪽 꼭대기의 아군과 북서쪽 조인 사이를 가르는 산줄기(x -6~-5, 강까지)
put(g, [(u, v) for u in range(W) for v in range(H) if u - v in (-6, -5) and u + v <= 16], 'n--')
# 길: 아군 출발 -> 북동쪽 -> 오른쪽 띠를 내려와 동쪽 다리 -> 서주성 / 조인 -> 서쪽 다리 -> 서주성
put(g, path([(4, 1), (10, 1), (10, 5), (16, 5), (16, 9), (18, 9), (18, 10)]), 'l--')
put(g, path([(15, 10), (15, 16), (17, 16), (17, 19)]), 'l--')
put(g, path([(2, 9), (4, 9), (4, 12)]), 'l--')
put(g, path([(4, 15), (4, 17), (10, 17), (10, 20), (17, 20)]), 'l--')
# 서주성: 북서·북동 성벽(성곽), 관문 둘, 성 안 성내와 성(관아)
put(g, [(u, v) for u in range(18, 24) for v in range(18, 24) if inside(u, v) and u - v < X0], 'i--')
put(g, [(17, v) for v in range(17, 24)] + [(u, 17) for u in range(17, 23)], 'c--')
put(g, [(17, 20)], 'ip-')
put(g, [(20, 17)], 'ip-')
put(g, [(21, 21)], 'io-')
put(g, [(19, 22), (22, 19)], 'iv-')
# 고순이 기대는 마을(동쪽 다리 머리), 제갈현이 기대는 마을(서쪽 다리 건너)
put(g, [(15, 10)], 'gv-')
put(g, [(5, 16)], 'gv-')
# 짓밟힌 바깥 고을과 그 밖
put(g, [(8, 3)], 'dve')
put(g, [(13, 6)], 'dve')
put(g, [(9, 2), (12, 7), (14, 5), (7, 4)], 'd-s')
put(g, [(6, 18), (7, 17)], 'u--')
put(g, [(2, 5), (0, 9), (1, 3), (12, 2), (15, 1), (20, 4), (22, 7), (21, 9), (23, 13), (19, 13), (11, 9), (13, 11),
        (8, 21), (11, 22), (14, 23), (9, 15), (12, 14), (2, 21), (5, 20), (14, 18), (6, 10), (0, 15)], 'gt-')
put(g, [(11, 3), (18, 6), (20, 12), (10, 19), (13, 16), (3, 18)], 'd--')
save('map12', g, ['# B12 서주 (24x24): 사수가 화면의 ㄱ자 바깥(위·오른쪽 띠)을 맵의 절반으로 가른다.',
                  '# 동쪽 다리 (16~17,10) 건너 고순의 마을 (15,10), 서쪽 다리 (4,13~14) 건너 제갈현의 마을 (5,16), 남쪽 서주성(성곽).'])
