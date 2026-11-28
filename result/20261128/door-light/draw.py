"""문으로 드는 햇빛 시안: 바닥 빛 웅덩이(ground)와 공중 빛기둥·먼지(object), 문 방향 SW/SE, 낮·해질녘.
소타일 2:1 격자(x=16(u-v), y=8(u+v)) 위에 4x4 Bayer 디더로 단계 알파. 피벗은 문 칸 가운데."""
import math, sys, json
from PIL import Image

B = [[0, 8, 2, 10], [12, 4, 14, 6], [3, 11, 1, 9], [15, 7, 13, 5]]
COL = {'day': (255, 238, 196), 'dusk': (255, 168, 92)}
OUT = sys.argv[1]
meta = {}


def floor_patch(dirn, tone):
    # 방 안쪽 방향 (문 밖으로 나가는 dir의 반대) 소타일 걸음
    inward = {'SW': (0, -1), 'SE': (-1, 0)}[dirn]
    side = {'SW': (1, 0), 'SE': (0, 1)}[dirn]
    W, H = 260, 150
    px, py = W // 2, H - 24
    im = Image.new('RGBA', (W, H))
    col = COL[tone]
    for y in range(H):
        for x in range(W):
            # 화면 픽셀 → 소타일 좌표 (문 칸 가운데가 0,0)
            X, Y = x - px, y - py
            u = (X / 16 + Y / 8) / 2
            v = (Y / 8 - X / 16) / 2
            depth = u * inward[0] + v * inward[1]          # 안쪽으로 몇 소타일
            lat = u * side[0] + v * side[1]
            if depth < -0.5:
                continue
            reach = 6.5 if tone == 'day' else 9.0
            half = 1.1 + depth * (0.22 if tone == 'day' else 0.3)  # 안으로 갈수록 퍼짐
            if depth > reach or abs(lat) > half:
                continue
            k = (1 - depth / reach) ** 1.2                  # 깊이에 따라 흐려짐
            k *= min(1, (half - abs(lat)) / 0.7)             # 가장자리 부드럽게
            k = max(0, min(1, k)) * (0.8 if tone == "day" else 0.72)
            level = math.floor(k * 4) / 4                     # 4단계
            if level <= 0:
                continue
            if B[y % 4][x % 4] / 16 < level:
                im.putpixel((x, y), col + (150,))
    return im, [px, py]


def beam(dirn, tone):
    """문에서 비스듬히 들어오는 옅은 빛기둥 + 먼지 점 (사람 위에 겹친다)"""
    inward = {'SW': (0, -1), 'SE': (-1, 0)}[dirn]
    W, H = 120, 120
    px, py = W // 2, H - 10
    im = Image.new('RGBA', (W, H))
    col = COL[tone]
    sx = 16 * (inward[0] - inward[1])     # 안쪽 한 소타일의 화면 x 걸음
    for y in range(H):
        for x in range(W):
            t = (H - 10 - y) / 90             # 위로 갈수록 0→1 (문 위쪽에서 바닥으로 내려오는 빛)
            if not 0 <= t <= 1:
                continue
            cx = px + sx * 2.2 * (1 - t) - sx * 0.3   # 바닥에서는 안쪽, 위로 갈수록 문 쪽
            w = 9 + 5 * (1 - t)
            d = abs(x - cx)
            if d > w:
                continue
            k = 0.3 * (1 - d / w) * (1 - abs(t - .45) * 1.2)
            if k <= 0:
                continue
            if B[y % 4][x % 4] / 16 < k * 2.2:
                im.putpixel((x, y), col + (95,))
    for (dx, dy) in [(-6, -30), (4, -52), (10, -24), (-2, -70), (14, -44)]:   # 먼지
        x, y = px + int(sx * 1.2) + dx, py + dy
        if 0 <= x < W and 0 <= y < H:
            im.putpixel((x, y), (255, 250, 228, 200))
    return im, [px, py]


for d in ('SW', 'SE'):
    for tone in ('day', 'dusk'):
        im, pv = floor_patch(d, tone)
        n = f'door_light_{d.lower()}_{tone}'
        im.save(f'{OUT}/{n}.png'); meta[n] = {'pivot': pv, 'layer': 'ground'}
        im, pv = beam(d, tone)
        n = f'door_beam_{d.lower()}_{tone}'
        im.save(f'{OUT}/{n}.png'); meta[n] = {'pivot': pv, 'layer': 'object'}

# 화살표 이정표: 나무 기둥 + 화살표 판 둘 (dir 쪽을 가리킴). 아스키 도트
POST = [
    "......ddd...........",
    ".....dWWWd..........",
    "...ddWWWWWdd........",
    "..dWWWWWWWWWAd......",
    ".dWWWWWWWWWWWAd.....",
    "..dWWWWWWWWWAd......",
    "...ddWWWWWdd........",
    ".....dPPd...........",
    ".....dPPd...........",
    "...ddWWWWWdd........",
    "..dAWWWWWWWWWd......",
    ".dAWWWWWWWWWWWd.....",
    "..dAWWWWWWWWWd......",
    "...ddWWWWWdd........",
    ".....dPPd...........",
    ".....dPPd...........",
    ".....dPPd...........",
    ".....dPPd...........",
    ".....dPPd...........",
    ".....dPPd...........",
    ".....dPPd...........",
    "....ddPPdd..........",
    "...sGGGGGGs.........",
    "....ssssss..........",
]
PAL = {'d': (52, 34, 22, 255), 'W': (196, 150, 98, 255), 'A': (120, 70, 40, 255), 'P': (128, 90, 56, 255),
       'G': (120, 116, 104, 255), 's': (0, 0, 0, 70)}
for d, flip in (('se', False), ('sw', True)):
    h, w = len(POST), len(POST[0])
    im = Image.new('RGBA', (w, h))
    for y, row in enumerate(POST):
        for x, c in enumerate(row):
            if c != '.':
                im.putpixel((w - 1 - x if flip else x, y), PAL[c])
    n = f'signpost_arrow_{d}'
    im.save(f'{OUT}/{n}.png'); meta[n] = {'pivot': [w - 1 - 7 if flip else 7, h - 2], 'layer': 'object'}
json.dump(meta, open(f'{OUT}/meta.json', 'w'), indent=1)
print(len(meta))
