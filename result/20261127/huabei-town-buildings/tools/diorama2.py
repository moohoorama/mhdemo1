"""지역별 성 안 마을: 그 지역 바닥색, 길, 뒤 두 변 장면 성벽(swall2), 건물 12채와 밭/논, 사람 키 표시(38px)"""
import json, os, sys
import numpy as np
from PIL import Image
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from town_dot2 import REGIONS
CAND = '/Users/younwookim/orca/mhdemo1/assets_candidate/tiles/'
NU, NV = 27, 22
LAYOUT = [("yamen", 1, 1), ("mansion_3", 10, 1), ("mansion_1", 19.5, .5),
          ("tavern", 1, 6.5), ("shop", 8, 6.5), ("mansion_2", 15, 6), ("mansion_4", 22, 6),
          ("cottage_1", .5, 12.5), ("cottage_2", 5, 12.5), ("smithy", 10, 12.5), ("cottage_3", 17.5, 12.5), ("cottage_4", 22, 12.5),
          ("field_a", 1, 17.5), ("field_b", 5.5, 17.5)]
PEOPLE = [(7.5, 5.8), (13.2, 11.5), (17.0, 10.2), (4.6, 16.8)]
OX, OY = 16 * NV + 8, 110

def ground(c):
    W, H = 16 * (NU + NV) + 16, 8 * (NU + NV) + 150
    X, Y = np.meshgrid(np.arange(W) - OX + .5, np.arange(H) - OY + .5)
    u, v = X / 32 + Y / 16, Y / 16 - X / 32
    i, j = np.floor(u * 16).astype(int), np.floor(v * 16).astype(int)
    inside = (u >= 0) & (u < NU) & (v >= 0) & (v < NV)
    road = ((v >= 4.9) & (v < 6.1)) | ((v >= 10.9) & (v < 12.1)) | ((u >= 15.5) & (u < 16.7) & (v > 11))
    a, b = np.array(c[0]), np.array(c[1])
    pave = (a * .5 + np.array([200, 190, 170]) * .5).astype(int)
    img = np.zeros((H, W, 4), np.uint8)
    n = (i * 7 + j * 13) % 29
    col = np.where((n == 0)[..., None], b, a)
    col = np.where((((i * 3 + j * 5) % 11) == 0)[..., None], np.minimum(a + 12, 255), col)
    rc = np.where(((i % 8 == 0) | (j % 8 == 0))[..., None], (pave * .8).astype(int), pave)
    col = np.where(road[..., None], rc, col)
    img[..., :3] = col
    img[..., 3] = np.where(inside, 255, 0)
    return Image.fromarray(img)

def build(region, d):
    meta = json.load(open(os.path.join(d, 'meta.json')))
    fields = [k for k in ('field_paddy', 'field_dry') if k in meta]
    g = ground(REGIONS[region]['ground'])
    items = []
    wall = Image.open(CAND + 'swall2.png')
    tc = lambda u, v: (OX + 16 * (u - v), OY + 8 * (u + v) + 8)
    for k in range(-1, NU):
        items.append((tc(k, -1), wall, (16, 75)))
    for k in range(0, NV):
        items.append((tc(-1, k), wall, (16, 75)))
    for name, u, v in LAYOUT:
        if name.startswith('field_'):
            name = fields[0] if name == 'field_a' else fields[-1]
        m = meta[name]
        fu, fv = m['footprint']
        im = Image.open(os.path.join(d, name + '.png'))
        cu, cv = u + fu / 2, v + fv / 2
        items.append(((OX + 16 * (cu - cv), OY + 8 * (cu + cv)), im, tuple(m['pivot']), name.startswith('field')))
    man = Image.new('RGBA', (10, 38), (40, 60, 110, 255)); man.paste((230, 200, 160, 255), (2, 0, 8, 8))
    for u, v in PEOPLE:
        items.append((tc(u - .5, v - .5), man, (5, 37)))
    items = [t if len(t) == 4 else t + (False,) for t in items]
    for (x, y), im, (px, py), flat in sorted(items, key=lambda t: (not t[3], t[0][1])):
        g.alpha_composite(im, (int(x - px), int(y - py)))
    return g.crop(g.getbbox())

if __name__ == '__main__':
    root = sys.argv[1]
    for r in REGIONS:
        g = build(r, os.path.join(root, r))
        g.save(os.path.join(root, r + '_town.png'))
        print(r, g.size)
