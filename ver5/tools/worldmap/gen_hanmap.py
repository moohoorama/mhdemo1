#!/usr/bin/env python3
"""Build the deformed Han map of cmd/_cnmap from the RTK9 overview picture in koei9/.

  python3 tools/worldmap/gen_hanmap.py               # write assets/worldmap/han.json
  python3 tools/worldmap/gen_hanmap.py --preview F   # also a picture of the grid, iso, one colour per kind

RTK9 draws its map as one iso diamond whose logical grid has x east and y south from the north-west corner — the
battle map's u and v. koei9/images/official-map.jpg (3200 x 1650) is that diamond with its top corner at (1600, 42)
and its side corners at (0 / 3200, 842), so a pixel lies on grid point u = N(X/1600 + Y/800)/2, v = N(Y/800 -
X/1600)/2 with X, Y from the top corner. The map is read cell by cell at N = 100 (the picture's 200 halved):

- each cell's mean colour and texture sorts it into water, rock, forest, bare earth or grass; rock thickened over
  its neighbours becomes mountain (hill where thin); the rivers the picture draws as dark green bands and the great
  ranges are traced by hand (RIVERS, RANGES);
- the cities are the castles of the picture, named by matching them to their real places (lon / lat) — a
  reconstruction, not RTK9's own coordinates (koei9/README.md);
- roads join each city to its nearest neighbours along the cheapest way round mountains and wide water; where a road
  crosses water it is a bridge.

han.json: "Rows" one character per cell (legend below), "Roads" the road cells, "Places" the cities, gates, villages
and the story's camps and fields of battle with their cell, kind, province and what the story does there (NOTES)."""
import argparse
import heapq
import json
import math
from pathlib import Path

import numpy as np
from PIL import Image

ROOT = Path(__file__).resolve().parents[2]
SRC = ROOT.parent / 'koei9/images/official-map.jpg'
OUT = ROOT / 'assets/worldmap/han.json'
N = 100

# cell kinds: . grass  , bare earth  ' plain  ~ water  T forest  h hill  M mountain
LEGEND = {'.': 'grass', ',': 'earth', "'": 'plain', '~': 'water', 'T': 'forest', 'h': 'hill', 'M': 'mountain'}

# (name, province, u, v, kind): the castles of the picture; gates and villages placed by their place in history
PLACES = [
    ('양평', '유주', 82, 10, 'city'), ('북평', '유주', 67, 2, 'city'), ('계', '유주', 56, 3, 'city'),
    ('남피', '기주', 73, 12, 'city'), ('평원', '기주', 70, 6, 'city'), ('업', '기주', 54, 20, 'city'),
    ('진양', '병주', 42, 8, 'city'), ('상당', '병주', 41, 19, 'city'), ('북해', '청주', 83, 31, 'city'),
    ('복양', '연주', 68, 27, 'city'), ('진류', '연주', 57, 30, 'city'), ('하비', '서주', 76, 43, 'city'),
    ('소패', '서주', 68, 39, 'city'), ('허도', '예주', 56, 38, 'city'), ('여남', '예주', 57, 50, 'city'),
    ('낙양', '사례', 45, 33, 'capital'), ('장안', '옹주', 30, 33, 'city'), ('안정', '옹주', 27, 22, 'city'),
    ('천수', '옹주', 14, 29, 'city'), ('서평', '양주', 3, 16, 'city'), ('무위', '양주', 9, 8, 'city'),
    ('수춘', '양주', 70, 53, 'city'), ('여강', '양주', 70, 66, 'city'), ('말릉', '양주', 85, 60, 'city'),
    ('오', '양주', 97, 63, 'city'), ('회계', '양주', 93, 73, 'city'), ('시상', '양주', 70, 76, 'city'),
    ('완', '형주', 40, 41, 'city'), ('신야', '형주', 40, 53, 'city'), ('양양', '형주', 45, 57, 'city'),
    ('강릉', '형주', 45, 72, 'city'), ('강하', '형주', 57, 59, 'city'), ('장사', '형주', 60, 88, 'city'),
    ('무릉', '형주', 45, 85, 'city'), ('계양', '형주', 64, 95, 'city'), ('영릉', '형주', 48, 94, 'city'),
    ('서성', '형주', 27, 48, 'city'), ('영안', '익주', 30, 69, 'city'), ('한중', '익주', 18, 44, 'city'),
    ('자동', '익주', 18, 62, 'city'), ('성도', '익주', 9, 71, 'city'), ('강주', '익주', 20, 79, 'city'),
    ('건녕', '익주', 18, 91, 'city'), ('운남', '익주', 5, 90, 'city'), ('교지', '교주', 35, 97, 'city'),
    ('오환', '이민족', 76, 2, 'tribe'), ('강', '이민족', 5, 27, 'tribe'), ('산월', '이민족', 90, 90, 'tribe'),
    ('남만', '이민족', 13, 98, 'tribe'),
    ('호로관', '사례', 50, 32, 'gate'), ('동관', '사례', 37, 33, 'gate'), ('양평관', '익주', 15, 49, 'gate'),
    ('검각', '익주', 17, 56, 'gate'),
    ('탁현', '유주', 54, 7, 'village'),
    # the story's places (docs/fullstory.md): camps, gates and fields of battle of the common part 1~24
    ('탁군 들판', '유주', 51, 9, 'field'), ('광종', '기주', 60, 20, 'camp'), ('장사성', '예주', 52, 41, 'city'),
    ('단양', '서주', 78, 48, 'village'), ('팽성', '서주', 71, 41, 'city'), ('산조', '연주', 60, 31, 'camp'),
    ('사수관', '사례', 50, 36, 'gate'), ('신안', '사례', 41, 33, 'field'), ('신도', '기주', 63, 10, 'city'),
    ('거록', '기주', 57, 13, 'field'), ('계교', '기주', 66, 14, 'camp'), ('서주', '서주', 80, 38, 'city'),
    ('태산', '연주', 76, 30, 'field'), ('광릉', '서주', 78, 50, 'city'), ('고성', '예주', 60, 45, 'city'),
    ('백마', '연주', 64, 31, 'field'), ('여양', '기주', 63, 27, 'camp'), ('관도', '연주', 55, 34, 'field'),
    ('융중', '형주', 42, 59, 'village'), ('박망파', '형주', 42, 47, 'field'), ('장판', '형주', 46, 65, 'field'),
    # ... and of the two modes after Changban
    ('적벽', '형주', 56, 68, 'field'), ('화용도', '형주', 50, 70, 'field'), ('공안', '형주', 45, 80, 'city'),
    ('맥성', '형주', 43, 68, 'city'), ('이릉', '형주', 38, 72, 'field'), ('번성', '형주', 45, 55, 'city'),
    ('하구', '형주', 53, 60, 'city'), ('상용', '형주', 35, 54, 'city'), ('부수관', '익주', 14, 66, 'gate'),
    ('낙성', '익주', 12, 68, 'city'), ('가맹관', '익주', 16, 53, 'gate'), ('와구관', '익주', 24, 60, 'gate'),
    ('정군산', '익주', 14, 46, 'field'),
    # not in the story, but places of the age
    ('진창', '옹주', 21, 34, 'city'), ('합비', '양주', 73, 58, 'city'),
]
# what each place of the story is for, by chapter
NOTES = {
    '탁현': '1장 도원결의', '탁군 들판': '1장 탁군 전투', '광종': '2장 노식의 진영', '장사성': '2장 영천 전투',
    '낙양': '3장 조조의 저택', '단양': '3장 단양 전투', '팽성': '4장 팽성 전투', '산조': '5~8장 반동탁 연합 진영',
    '사수관': '5장 사수관 전투', '호로관': '6장 호로관 전투', '신안': '7장 동탁 추격전', '평원': '8장 평원현',
    '신도': '8장 신도성 전투', '거록': '9장 거록 전투', '계교': '10장 공손찬의 진영 · 계교 전투',
    '북평': '11장 공손찬의 관저', '북해': '11장 북해 전투', '서주': '12장 서주 전투', '태산': '13장 태산 전투',
    '광릉': '14장 광릉 전투', '소패': '15장 소패 전투', '하비': '16장 하비 전투 · 백문루',
    '허도': '17장 허도 · 가상 5부 허창 전투', '업': '18장 원소 진영 탈출', '백마': '18장 안량·문추가 베인 곳',
    '여양': '18장 원소군의 강가 진지', '고성': '19장 고성 전투', '관도': '20장 관도의 패배 · 가상 5부 관도 최종결전',
    '여남': '20장 여남 전투', '양양': '21·24장 유표의 관저 · 가상 1부 양양 공략전', '강하': '21장 강하 전투',
    '신야': '22장 신야 전투', '융중': '23장 삼고초려', '박망파': '23장 박망파 전투', '장판': '24장 장판파 전투',
    '완': '24장 송충이 가던 곳',
    '적벽': '연의 1부 적벽 전투', '화용도': '연의 1부 화용도', '강릉': '연의 2부 남군 · 가상 1부 강릉 전투',
    '장사': '연의 2부 형남 4군', '무릉': '연의 2부 형남 4군', '영릉': '연의 2부 형남 4군',
    '계양': '연의 2부 형남 4군', '공안': '연의 2부 공안 전투', '부수관': '연의 3부 부수관 전투',
    '낙성': '연의 3부 낙성 전투', '가맹관': '연의 3부 · 가상 2부 가맹관 전투', '와구관': '연의 4부 와구관 전투',
    '정군산': '연의 4부 · 가상 3부 정군산 전투', '양평관': '연의 4부 · 가상 3부 양평관 전투',
    '번성': '연의 5부 · 가상 4부 번성 전투', '맥성': '연의 5부 맥성 전투', '이릉': '연의 5부 이릉 전투',
    '영안': '연의 엔딩 백제성', '상용': '가상 2부 상용 전투', '하구': '가상 4부 하구 전투',
    '말릉': '가상 4부 말릉 전투', '장안': '가상 5부 장안 전투', '성도': '익주의 치소', '한중': '익주의 북문',
    '진창': '관중의 서쪽 문', '합비': '회남의 요충',
}
LINKS = 2  # roads from each city to this many nearest cities
# roads the story walks, laid first so the others join them: the Hebei arc goes 평원, 신도, 거록, 계교 and over its
# bridge to 업, 평원 goes down past 남피 to 북해, and Yuan Shao's army from 업 to the river at 여양
STORY_ROADS = [('평원', '신도'), ('신도', '거록'), ('거록', '광종'), ('거록', '계교'), ('계교', '업'), ('평원', '남피'),
               ('남피', '북해'), ('업', '여양')]

# the rivers the picture draws as dark green bands, traced by hand over it (the sea, the Yangtze and the lakes are
# read from the picture's water)
RIVERS = {
    '황하': [(14, 2), (20, 6), (28, 8), (32, 14), (33, 20), (33, 27), (37, 30), (44, 30), (50, 29), (56, 27), (62, 29),
           (68, 29), (73, 26), (78, 21)],
    '위수': [(13, 30), (20, 32), (26, 31), (33, 28)],
    '한수': [(17, 46), (26, 50), (34, 52), (42, 55), (48, 57), (54, 58), (58, 62)],
    '회수': [(48, 46), (58, 47), (66, 49), (74, 48), (84, 45)],
    '청하': [(67, 3), (65, 10), (64, 14), (63, 20), (62, 27)],
}

# the great ranges, drawn by hand two cells thick so the passes (the gates) are the ways through them
RANGES = {
    '진령': [(10, 38), (18, 39), (26, 40), (34, 41)],
    '대파산': [(4, 55), (12, 56), (20, 57), (28, 59), (34, 62)],
    '태항산': [(47, 3), (48, 11), (47, 18), (45, 26)],
    '효산': [(37, 29), (37, 37)],
    '숭산': [(50, 30), (50, 37)],
    '거록 벼랑': [(57, 8), (57, 18)],
}


def trace(points):
    """The cells of a polyline, stepping one axis at a time so the water joins edge to edge."""
    out = []
    for (u0, v0), (u1, v1) in zip(points, points[1:]):
        n = max(abs(u1 - u0), abs(v1 - v0)) * 2
        for k in range(n + 1):
            c = (round(u0 + (u1 - u0) * k / n), round(v0 + (v1 - v0) * k / n))
            if out and c[0] != out[-1][0] and c[1] != out[-1][1]:
                out.append((c[0], out[-1][1]))
            if not out or out[-1] != c:
                out.append(c)
    return out


def cell_stats():
    im = np.asarray(Image.open(SRC).convert('RGB')).astype(float)
    h, w, _ = im.shape
    ys, xs = np.mgrid[0:h, 0:w]
    X, Y = xs - 1600.0, ys - 42.0
    u, v = N * (X / 1600 + Y / 800) / 2, N * (Y / 800 - X / 1600) / 2
    inside = (u >= 0) & (v >= 0) & (u < N) & (v < N)
    idx = (np.floor(v).astype(int) * N + np.floor(u).astype(int))[inside]
    n = np.maximum(np.bincount(idx, minlength=N * N), 1)
    mean = np.stack([np.bincount(idx, im[..., c][inside], N * N) / n for c in range(3)], -1)
    sq = np.stack([np.bincount(idx, im[..., c][inside] ** 2, N * N) / n for c in range(3)], -1)
    std = np.sqrt(np.maximum(sq - mean ** 2, 0)).mean(-1)
    return mean.reshape(N, N, 3), std.reshape(N, N)


def neighbours(grid, u, v, r=1):
    return grid[max(0, v - r):v + r + 1, max(0, u - r):u + r + 1]


def classify():
    mean, std = cell_stats()
    r, g, b = mean[..., 0], mean[..., 1], mean[..., 2]
    lum = mean.mean(-1)
    water = (b - r > 15) & (g - r > 15) & (lum < 75)
    dark = ~water & (lum < 52) & (g >= r)
    rock = ~water & ~dark & (std > 23) & (lum > 60) & (np.abs(r - g) < 22)
    bare = ~water & ~dark & ~rock & (r - g > 8) & (lum > 85)
    forest = ~water & ~dark & ~rock & ~bare & (g - r > 2) & (lum < 72)
    kind = np.full((N, N), '.', '<U1')
    kind[bare] = ','
    kind[forest] = 'T'
    kind[dark] = 'T'
    kind[water] = '~'
    for line in RIVERS.values():
        for u, v in trace(line):
            kind[v, u] = '~'
    for line in RANGES.values():
        for u, v in trace(line):
            for x, y in ((u, v), (u + 1, v), (u, v + 1)):
                if x < N and y < N and kind[y, x] != '~':
                    kind[y, x] = 'M'
    # rock gathered thickly is mountain, thinly hill
    for v in range(N):
        for u in range(N):
            if kind[v, u] == '~':
                continue
            share = neighbours(rock, u, v).mean()
            if share >= .5 and rock[v, u]:
                kind[v, u] = 'M'
            elif share >= .45:
                kind[v, u] = 'h'
    # lone water cells (a dark speck) dry up; lone mountains in open ground become hills
    for v in range(N):
        for u in range(N):
            n = neighbours(kind, u, v)
            if kind[v, u] == '~' and (n == '~').sum() <= 1:
                kind[v, u] = 'T'
            if kind[v, u] == 'M' and (n == 'M').sum() <= 2:
                kind[v, u] = 'h'
    return kind


def clear_places(kind):
    """A place stands on open dry ground: its cell is cleared of mountains, hills, forest and water, the ring round a
    city of mountains and hills."""
    for name, _, u, v, k in PLACES:
        ring = 1 if k in ('city', 'capital') else 0
        for y in range(v - ring, v + ring + 1):
            for x in range(u - ring, u + ring + 1):
                if 0 <= x < N and 0 <= y < N and (x, y) != (u, v) and kind[y, x] in 'Mh':
                    kind[y, x] = '.'
        if kind[v, u] in '~MhT':
            kind[v, u] = '.'


def route(kind, a, b, bridge=9):
    """The cheapest way from cell a to b: mountains cost much, water more (a bridge), forest and hills a little."""
    cost = {'.': 1, ',': 1, "'": 1, 'T': 2, 'h': 3, 'M': 40, '~': bridge}
    dist, prev = {a: 0}, {}
    q = [(0, a)]
    while q:
        d, c = heapq.heappop(q)
        if c == b:
            break
        if d > dist[c]:
            continue
        for du, dv in ((1, 0), (-1, 0), (0, 1), (0, -1)):
            n = (c[0] + du, c[1] + dv)
            if not (0 <= n[0] < N and 0 <= n[1] < N):
                continue
            nd = d + (.3 if n in ROADS else cost[kind[n[1], n[0]]])  # follow the roads already laid, not beside them
            if nd < dist.get(n, 1e18):
                dist[n], prev[n] = nd, c
                heapq.heappush(q, (nd, n))
    path, c = [b], b
    while c != a:
        c = prev[c]
        path.append(c)
    return path


ROADS = set()


def linked(a, b, without):
    """Whether road cells a and b are joined by roads that do not pass through cell without."""
    seen, todo = {a}, [a]
    while todo:
        c = todo.pop()
        if c == b:
            return True
        for du, dv in ((1, 0), (-1, 0), (0, 1), (0, -1)):
            n = (c[0] + du, c[1] + dv)
            if n in ROADS and n != without and n not in seen:
                seen.add(n)
                todo.append(n)
    return False


def thin(keep):
    """Two roads laid side by side make 2 x 2 blocks of road; drop the cells of a block the roads can go without."""
    for u, v in sorted(ROADS):
        for c in ((u, v), (u + 1, v), (u, v + 1), (u + 1, v + 1)):
            if not {(u, v), (u + 1, v), (u, v + 1), (u + 1, v + 1)} <= ROADS:
                break
            if c in keep:
                continue
            near = [n for n in ((c[0] + 1, c[1]), (c[0] - 1, c[1]), (c[0], c[1] + 1), (c[0], c[1] - 1)) if n in ROADS]
            if all(linked(near[0], n, c) for n in near[1:]):
                ROADS.discard(c)


def roads(kind):
    towns = [(u, v) for _, _, u, v, k in PLACES if k != 'tribe']
    pairs = set()
    for a in towns:
        near = sorted((math.dist(a, b), b) for b in towns if b != a)[:LINKS]
        for d, b in near:
            if d < 20:
                pairs.add(tuple(sorted((a, b))))
    at = {n: (u, v) for n, _, u, v, _ in PLACES}
    for a, b in STORY_ROADS:
        ROADS.update(route(kind, at[a], at[b], bridge=3))  # the story crosses rivers rather than going round
    for a, b in sorted(pairs, key=lambda p: math.dist(*p)):
        ROADS.update(route(kind, a, b))
    thin(set(towns))
    for u, v in ROADS:
        if kind[v, u] in 'Mh':
            kind[v, u] = '.'  # a road cuts its pass
    return sorted(ROADS)


def preview(kind, rd, path):
    col = {'.': (112, 143, 77), ',': (187, 158, 106), "'": (150, 160, 95), '~': (44, 108, 119), 'T': (50, 78, 34),
           'h': (140, 150, 90), 'M': (125, 108, 90)}
    img = Image.new('RGB', (N * 16 + 16, N * 8 + 16))
    px = img.load()
    for v in range(N):
        for u in range(N):
            c = col[kind[v, u]]
            if (u, v) in rd:
                c = (220, 190, 120) if kind[v, u] != '~' else (150, 90, 50)
            cx, cy = 8 + (u - v) * 8 + N * 8, 8 + (u + v) * 4
            for j in range(-4, 4):
                half = int(8 - 2 * abs(j + .5))
                for i in range(-half, half):
                    px[cx + i, cy + j] = c
    for name, _, u, v, k in PLACES:
        cx, cy = 8 + (u - v) * 8 + N * 8, 8 + (u + v) * 4
        for j in range(-3, 4):
            for i in range(-3, 4):
                px[cx + i, cy + j] = {'capital': (255, 215, 0), 'city': (230, 60, 50), 'tribe': (160, 60, 160),
                                      'gate': (40, 40, 40), 'village': (255, 255, 255), 'camp': (60, 200, 255),
                                      'field': (255, 140, 0)}[k]
    img.resize((img.width * 2, img.height * 2), Image.NEAREST).save(path)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--preview')
    a = ap.parse_args()
    kind = classify()
    clear_places(kind)
    rd = roads(kind)
    doc = {
        'Name': '후한 13주',
        'Hero': 'liubei',
        'Source': 'koei9/images/official-map.jpg (RTK9 overview), read at 100 x 100; cities named by real place',
        'Size': N,
        'Legend': LEGEND,
        'Rows': [''.join(kind[v]) for v in range(N)],
        'Roads': [[u, v] for u, v in rd],
        'Places': [{'Name': n, 'Province': p, 'At': [u, v], 'Kind': k, 'Note': NOTES.get(n, '')}
                   for n, p, u, v, k in PLACES],
        'Start': '탁현',
    }
    OUT.parent.mkdir(parents=True, exist_ok=True)
    with open(OUT, 'w') as f:
        json.dump(doc, f, ensure_ascii=False, indent=1)
    if a.preview:
        preview(kind, set(rd), a.preview)
    counts = {LEGEND[k]: int((kind == k).sum()) for k in LEGEND}
    print(OUT, counts, len(rd), 'road cells')


if __name__ == '__main__':
    main()
