"""화살표 이정표: 나무 기둥에 2:1 대각선으로 뻗은 화살표 판 둘. 위 판은 출구 방향, 아래 판은 다른 길(반대 대각)"""
import json
from PIL import Image
INK, WOOD, WOODL, WOODD, POST, POSTD, STONE = (46, 30, 20, 255), (190, 142, 88, 255), (214, 170, 112, 255), (140, 96, 56, 255), (122, 86, 52, 255), (92, 62, 38, 255), (128, 122, 108, 255)


def board(im, x0, y0, sx, length=22, h=6):
    """(x0,y0)에서 sx(+1 오른쪽 아래, -1 왼쪽 아래) 쪽 2:1로 뻗은 판, 끝은 화살촉"""
    pts = {}
    for i in range(length):
        x = x0 + sx * i
        ytop = y0 + i // 2
        tip = length - i
        hh = h if tip > 5 else max(1, h + 2 - (5 - tip) * 2)
        off = 0 if tip > 5 else -1 + (5 - tip)
        for j in range(-1 if tip <= 5 else 0, hh + (1 if tip <= 5 else 0)):
            y = ytop + j + (off if tip <= 5 else 0) - (1 if tip <= 5 else 0)
            pts[(x, y)] = WOODL if j <= 1 else WOOD if j < hh - 1 else WOODD
    for (x, y), c in pts.items():
        im.putpixel((x, y), c)
    for (x, y) in pts:                                   # 1px 외곽선
        for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
            if (x + dx, y + dy) not in pts and 0 <= x + dx < im.width and 0 <= y + dy < im.height and im.getpixel((x + dx, y + dy))[3] == 0:
                im.putpixel((x + dx, y + dy), INK)


def sign(sx):
    W, H = 56, 64
    im = Image.new('RGBA', (W, H))
    px = W // 2
    for y in range(14, H - 6):                           # 기둥
        for x in (px - 1, px, px + 1):
            im.putpixel((x, y), POST if x < px + 1 else POSTD)
        im.putpixel((px - 2, y), INK); im.putpixel((px + 2, y), INK)
    for x in range(px - 3, px + 4):
        im.putpixel((x, 13), INK)
    board(im, px - 2 * sx, 16, sx)                        # 출구 쪽 판
    board(im, px + 2 * sx, 30, -sx, length=16, h=5)      # 다른 길 판 (짧게)
    for x in range(px - 6, px + 7):                       # 받침돌·그림자
        for y in (H - 6, H - 5):
            im.putpixel((x, y), STONE if y == H - 6 else (0, 0, 0, 70))
    return im, [px, H - 6]


meta = json.load(open('meta.json'))
for d, sx in (('se', 1), ('sw', -1)):
    im, pv = sign(sx)
    im.save(f'signpost_arrow_{d}.png'); meta[f'signpost_arrow_{d}'] = {'pivot': pv, 'layer': 'object'}
json.dump(meta, open('meta.json', 'w'), indent=1)
