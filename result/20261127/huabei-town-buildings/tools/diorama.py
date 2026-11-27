"""두 안의 건물을 같은 자리에 놓은 성 안 마을: 흙 바닥, 돌길, 뒤 두 변에 장면 성벽(swall2), 사람 크기 표시(38px)"""
import json, os, sys
import numpy as np
from PIL import Image
CAND = '/Users/younwookim/orca/mhdemo1/assets_candidate/tiles/'
N = 13
LAYOUT = [("yamen", 6, 1), ("tavern", 1, 1), ("cottage", 1, 5), ("smithy", 9, 6), ("shop", 4, 8), ("cottage", 1, 9),
          ("cottage", 9, 10)]
FOOT = {"cottage": (2, 2), "smithy": (3, 2), "yamen": (4, 3), "tavern": (3, 2), "shop": (3, 2)}
PEOPLE = [(4.5, 5.5), (7.6, 4.8), (8.2, 9.0)]
OX, OY = 16 * (N + 1) + 8, 100

def ground():
    W, H = 32 * (N + 2) + 16, 16 * (N + 1) + 140
    im = np.zeros((H, W, 4), np.uint8)
    rnd = np.random.default_rng(3)
    for y in range(H):
        for x in range(W):
            X, Y = x - OX + .5, y - OY + .5
            u, v = X / 32 + Y / 16, Y / 16 - X / 32
            if 0 <= u < N and 0 <= v < N:
                i, j = int(u * 16), int(v * 16)
                road = (5.2 <= u < 6.0) or (4.6 <= v < 5.4) and u > 2
                if road:
                    c = (178, 160, 128) if (i % 8 and j % 8) else (140, 124, 98)
                else:
                    n = (i * 7 + j * 13) % 29
                    c = (176, 140, 92) if n else (150, 116, 74)
                    if (i * 3 + j * 5) % 11 == 0: c = (188, 154, 104)
                im[y, x] = c + (255,)
    return Image.fromarray(im)

def tile_c(u, v):  # 소타일 (u,v)의 중심 화면 좌표
    return OX + 16 * (u - v), OY + 8 * (u + v) + 8

def build(d):
    meta = json.load(open(os.path.join(d, 'meta.json')))
    g = ground()
    items = []
    wall = Image.open(CAND + 'swall2.png')
    for k in range(-1, N):
        items.append((tile_c(k, -1), wall, (16, 75)))
        if k >= 0: items.append((tile_c(-1, k), wall, (16, 75)))
    for name, u, v in LAYOUT:
        fu, fv = FOOT[name]
        im = Image.open(os.path.join(d, name + '.png'))
        cx, cy = OX + 16 * ((u + fu / 2) - (v + fv / 2)), OY + 8 * ((u + fu / 2) + (v + fv / 2))
        items.append(((cx, cy), im, tuple(meta[name]['pivot'])))
    man = Image.new('RGBA', (10, 38), (40, 60, 110, 255))
    man.paste((230, 200, 160, 255), (2, 0, 8, 8))
    for u, v in PEOPLE:
        items.append((tile_c(u - .5, v - .5), man, (5, 37)))
    for (x, y), im, (px, py) in sorted(items, key=lambda t: t[0][1]):
        g.alpha_composite(im, (int(x - px), int(y - py)))
    return g

if __name__ == '__main__':
    for d in sys.argv[1:]:
        g = build(d)
        g = g.crop(g.getbbox())
        g.save(d + '_town.png')
        g.resize((g.width * 2, g.height * 2), Image.NEAREST).save(d + '_town2x.png')
        print(d, g.size)
