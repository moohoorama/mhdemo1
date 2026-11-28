"""문으로 드는 햇빛과 마을 길 끝 화살표 이정표 (docs/plan-scene-exits.md).

- door_light_<sw|se>_<day|dusk>: 문 칸에서 방 안쪽으로 뻗는 바닥 빛 (ground). 알파 블렌딩, 디더링 없음(사용자 11-28).
- door_beam_<sw|se>_<day|dusk>: 문에서 비스듬히 내려오는 옅은 빛기둥과 먼지 (object, 사람 위에 겹쳐 빛 속 사람이 밝아진다).
- signpost_arrow_<se|sw|ne|nw>: 나무 기둥에 2:1 대각선 화살표 판 둘, 위 판이 출구 쪽.
피벗은 문 칸 / 기둥 발. 소타일 2:1 격자 (x = 16(u-v), y = 8(u+v)).

  python3 light_props.py OUTDIR      → OUTDIR/<name>.png, OUTDIR/meta.json
"""
import json
import math
import os
import sys

from PIL import Image

COL = {'day': (255, 240, 200), 'dusk': (255, 166, 88)}
INWARD = {'SW': (0, -1), 'SE': (-1, 0)}      # 문 밖으로 나가는 방향의 반대 (방 안쪽)
SIDE = {'SW': (1, 0), 'SE': (0, 1)}


def floor_light(dirn, tone):
    inward, side = INWARD[dirn], SIDE[dirn]
    W, H = 260, 150
    px, py = W // 2, H - 24
    im = Image.new('RGBA', (W, H))
    r, g, b = COL[tone]
    reach = 6.5 if tone == 'day' else 9.0
    peak = 120 if tone == 'day' else 105
    for y in range(H):
        for x in range(W):
            X, Y = x - px, y - py
            u, v = (X / 16 + Y / 8) / 2, (Y / 8 - X / 16) / 2
            depth = u * inward[0] + v * inward[1]
            lat = u * side[0] + v * side[1]
            if depth < -0.5 or depth > reach:
                continue
            half = 1.1 + max(0, depth) * (0.22 if tone == 'day' else 0.3)
            if abs(lat) > half + 0.6:
                continue
            k = (1 - max(0, depth) / reach) ** 1.3
            k *= max(0, min(1, (half + 0.6 - abs(lat)) / 1.0))         # 가장자리 부드럽게
            if depth < -0.35:                                             # 문턱(벽 줄)에서 바로 시작, 바깥은 자른다
                k *= max(0, (depth + 0.5) / 0.15)
            a = round(peak * k)
            if a > 2:
                im.putpixel((x, y), (r, g, b, a))
    return im, [px, py]


def beam(dirn, tone):
    """문 위(벽 줄 바깥 위쪽)에서 방 안 바닥 한 칸 남짓으로 비스듬히 내려오는 빛기둥과 먼지"""
    inward = INWARD[dirn]
    W, H = 140, 130
    px, py = W // 2, H - 14
    im = Image.new('RGBA', (W, H))
    r, g, b = COL[tone]
    tx, ty = 16 * (inward[0] - inward[1]), 8 * (inward[0] + inward[1])     # 안쪽 한 소타일의 화면 걸음
    floor = (px + tx * 1.4, py + ty * 1.4)                                 # 바닥에 닿는 곳: 문턱에서 1.4칸 안
    top = (px - tx * 0.5, py - ty * 0.5 - 78)                              # 문 위, 벽 줄 바깥
    for y in range(H):
        for x in range(W):
            # 이 픽셀이 기둥 축의 어디쯤인지 (0 바닥 … 1 위)
            t = (floor[1] - y) / (floor[1] - top[1])
            if not 0 <= t <= 1:
                continue
            cx = floor[0] + (top[0] - floor[0]) * t
            w = 7 + 9 * (1 - t)
            d = abs(x - cx)
            if d > w:
                continue
            k = (1 - d / w) ** 1.5 * min(1, t * 6) * (1 - t * 0.55)
            a = round(62 * k)
            if a > 2:
                im.putpixel((x, y), (r, g, b, a))
    for f, dx in ((0.25, -4), (0.45, 5), (0.6, -2), (0.75, 3), (0.35, 8)):    # 먼지
        x = round(floor[0] + (top[0] - floor[0]) * f + dx)
        y = round(floor[1] + (top[1] - floor[1]) * f)
        if 0 <= x < W and 0 <= y < H:
            im.putpixel((x, y), (255, 250, 228, 190))
    return im, [px, py]


INK, WOOD, WOODL, WOODD = (46, 30, 20, 255), (190, 142, 88, 255), (214, 170, 112, 255), (140, 96, 56, 255)
POST, POSTD, STONE = (122, 86, 52, 255), (92, 62, 38, 255), (128, 122, 108, 255)


def board(im, x0, y0, sx, sy, length=22, h=6):
    """(x0, y0)에서 화면 (sx, sy/2) 쪽 2:1 대각선으로 뻗은 판, 끝은 화살촉"""
    pts = {}
    for i in range(length):
        x = x0 + sx * i
        ytop = y0 + sy * (i // 2)
        tip = length - i
        hh = h if tip > 5 else max(1, h + 2 - (5 - tip) * 2)        # 화살촉: 넓게 시작해 끝으로 좁아진다
        off = 0 if tip > 5 else (5 - tip) - 2
        for j in range(0 if tip > 5 else -1, hh + (0 if tip > 5 else 1)):
            y = ytop + j + off
            pts[(x, y)] = WOODL if j <= 1 else WOOD if j < hh - 1 else WOODD
        continue
        for j in []:
            pts[(x, ytop + j)] = WOODL if j <= 1 else WOOD if j < h - 1 else WOODD
    for (x, y), c in pts.items():
        if 0 <= x < im.width and 0 <= y < im.height:
            im.putpixel((x, y), c)
    for (x, y) in pts:
        for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
            q = (x + dx, y + dy)
            if q not in pts and 0 <= q[0] < im.width and 0 <= q[1] < im.height and im.getpixel(q)[3] == 0:
                im.putpixel(q, INK)


def rising(sx, length, h):
    """오르는 판: 내려가는 판을 위아래로 뒤집고 밝은 윗면·어두운 아랫면을 바로잡는다"""
    tmp = Image.new('RGBA', (40, 30))
    board(tmp, 20 - 2 * sx if sx > 0 else 20 + 2, 4, sx, 1, length, h)
    tmp = tmp.transpose(Image.FLIP_TOP_BOTTOM)
    px = tmp.load()
    for y in range(tmp.height):
        for x in range(tmp.width):
            if px[x, y] == WOODL:
                px[x, y] = WOODD
            elif px[x, y] == WOODD:
                px[x, y] = WOODL
    return tmp


def signpost(dirn):
    """위 판이 dirn 쪽: SE 오른쪽 아래, SW 왼쪽 아래, NE 오른쪽 위, NW 왼쪽 위. 아래 판은 반대 옆 길"""
    sx, sy = {'SE': (1, 1), 'SW': (-1, 1), 'NE': (1, -1), 'NW': (-1, -1)}[dirn]
    W, H = 56, 70
    im = Image.new('RGBA', (W, H))
    px = W // 2
    for y in range(18, H - 6):
        for x in (px - 1, px, px + 1):
            im.putpixel((x, y), POST if x < px + 1 else POSTD)
        im.putpixel((px - 2, y), INK)
        im.putpixel((px + 2, y), INK)
    for x in range(px - 3, px + 4):
        im.putpixel((x, 17), INK)
    if sy > 0:
        board(im, px - 2 * sx, 20, sx, 1)
        board(im, px + 2 * sx, 36, -sx, 1, length=16, h=5)
    else:
        top = rising(sx, 22, 6)
        low = rising(-sx, 16, 5)
        # 뒤집은 판의 뿌리(기둥 쪽 끝)를 기둥에 맞춘다
        def root(layer, s_):
            xs = [x for x in range(layer.width) for y in range(layer.height) if layer.getpixel((x, y))[3]]
            ys = [y for x in range(layer.width) for y in range(layer.height) if layer.getpixel((x, y))[3]]
            return (min(xs) if s_ > 0 else max(xs)), max(ys)
        rx, ry = root(top, sx)
        im.alpha_composite(top, (px - 2 * sx - rx if sx > 0 else px + 2 - rx, 32 - ry))
        rx, ry = root(low, -sx)
        im.alpha_composite(low, (px + 2 * sx - rx if -sx > 0 else px + 2 * sx - rx, 50 - ry))
    for x in range(px - 6, px + 7):
        im.putpixel((x, H - 6), STONE)
        im.putpixel((x, H - 5), (0, 0, 0, 70))
    return im, [px, H - 6]


def main(out):
    os.makedirs(out, exist_ok=True)
    meta = {}
    for d in ('SW', 'SE'):
        for tone in ('day', 'dusk'):
            for kind, fn, layer in (('door_light', floor_light, 'ground'), ('door_beam', beam, 'object')):
                im, pv = fn(d, tone)
                n = f'{kind}_{d.lower()}_{tone}'
                im.save(os.path.join(out, n + '.png'))
                meta[n] = {'pivot': pv, 'layer': layer, 'walk': 'open'}
    for d in ('SE', 'SW', 'NE', 'NW'):
        im, pv = signpost(d)
        n = f'signpost_arrow_{d.lower()}'
        im.save(os.path.join(out, n + '.png'))
        meta[n] = {'pivot': pv, 'layer': 'object'}
    json.dump(meta, open(os.path.join(out, 'meta.json'), 'w'), indent=1)
    return meta


if __name__ == '__main__':
    print(len(main(sys.argv[1])), 'sprites')
