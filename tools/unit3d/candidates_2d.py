#!/usr/bin/env python3
"""First-stage ver2 unit candidates (SW view) in the knight v6 pixel style. Requires Pillow and numpy.

Archived generator, kept unchanged so tools/spritetool/assets/ver2-unit-candidates can be
reproduced (python3 tools/unit3d/candidates_2d.py). The current pipeline is build.py.

Infantry and Yellow Turban bandits reuse the knight SW idle frame body and swap
head, weapon and shield parts drawn as palette-index grids. Cavalry pixelizes the
legacy cavalry render (same 3D model family as the knight source) at the scale
where the rider helmet matches the knight helmet, then swaps the rider head.
"""
import json
from collections import Counter, deque
from pathlib import Path
import numpy as np
from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parents[2]
KNIGHT = ROOT / 'tools/spritetool/assets/knights-pixel-v6'
CAVALRY_SRC = ROOT / 'imageset/legacy/dist/units/cavalry.webp'
OUT = ROOT / 'tools/spritetool/assets/ver2-unit-candidates'
SHEET = ROOT / 'output/ver2-unit-candidates.png'
FONT = '/System/Library/Fonts/AppleSDGothicNeo.ttc'

KEYS = '0123456789abcdefghijklmnopqrstuvwxyz'
PAL = {KEYS[i]: tuple(int(h[j:j+2], 16) for j in (1, 3, 5))
       for i, h in enumerate(json.loads((KNIGHT / 'frames.json').read_text())['palette'])}
KNIGHT_COLORS = dict(PAL)
PAL.update({'S': (0xd9, 0xc6, 0x98), 'T': (0xa8, 0x92, 0x66), 'U': (0x6c, 0x58, 0x3c)})  # hemp cloth
INV = {v: k for k, v in PAL.items()}


def from_img(im):
    return [['.' if im.getpixel((x, y))[3] == 0 else INV[im.getpixel((x, y))[:3]]
             for x in range(im.width)] for y in range(im.height)]


def to_img(g):
    im = Image.new('RGBA', (len(g[0]), len(g)))
    for y, row in enumerate(g):
        for x, c in enumerate(row):
            if c != '.':
                im.putpixel((x, y), PAL[c] + (255,))
    return im


def blank(w, h):
    return [['.'] * w for _ in range(h)]


def overlay(g, part, x0, y0):
    """Paint a grid part; '.' keeps the pixel, ',' clears it."""
    for dy, row in enumerate(part):
        for dx, c in enumerate(row):
            if c != '.':
                g[y0+dy][x0+dx] = '.' if c == ',' else c


def recolor(g, mapping, box):
    x0, y0, x1, y1 = box
    for y in range(y0, y1):
        for x in range(x0, x1):
            g[y][x] = mapping.get(g[y][x], g[y][x])


def erase(g, box):
    x0, y0, x1, y1 = box
    for y in range(y0, y1):
        for x in range(x0, x1):
            g[y][x] = '.'


def grid(*rows, w=13):
    for r in rows:
        assert len(r) == w, (r, len(r))
    return list(rows)


# Heads are 13x12 and replace the knight helmet at (18,11). The helmet rim sits on a
# one-row shadow so the face starts below the rim instead of under a bright band.
FACE = ('.0j0jj0ji0210', '.0j0jj0ji0210', '..0jjjjii0210', '...0iii00210.')
HEAD_IRON = grid('.............', '.......0d0...', '......0dc0...', '....000bb000.',
                 '...0344543320', '..03455443210', '.023333332210', '.0iiiiiii0210', *FACE)
HEAD_PLUME = grid('......0dd0...', '.....0ddcc0..', '....0ddcbbc0.', '....00bb0000.',
                  '...0344543320', '..03455443210', '.023333332210', '.0iiiiiii0210', *FACE)
HEAD_CONE = grid('......0......', '.....040.....', '.....030.....', '....03430....',
                 '...0345320...', '..034554320..', '.023333332870', '.0iiiiiii7870',
                 '.0j0jj0ji7870', '.0j0jj0ji7870', '..0jjjjii7870', '...0iii00870.')
HEAD_HOOD = grid('.............', '.............', '....000000...', '...08999880..',
                 '..089aaa98870', '.089aaa998870', '.077777788870', '.0iiiiii8870.',
                 '.0j0jj0i8870.', '.0j0jj0i8870.', '..0jjjii8870.', '...0iii08870.')
HEAD_YHOOD = [r.translate(str.maketrans('789a', 'klmn')) for r in HEAD_HOOD]
HEAD_TOPKNOT = grid('......000....', '.....01110...', '....0011100..', '...001111100.',
                    '..0mmmmmmml0.', '.0mmmmmllkk0m', '.0lllllkkk0ml', '.0iiiiiii110k',
                    '.0j0jj0ij110.', '.0j0jj0ii110.', '..0jjjjii10..', '...0iiii00...')
HEAD_STRAW = grid('.............', '.............', '.....000.....', '...00lmm00...',
                  '.00klllmmll00', '0kkkkkllllkk0', '.0mmmmllkkk0.', '.0iiiiii110..',
                  '.0j0jj0i110..', '.0j0jj0i110..', '..0jjjii10...', '...0iiii0....')

# Torso and legs without the shield, 18 wide at (13,25).
BODY_OPEN = grid('.fllh7899a998780..', '.0klh789aa9987980.', '..0f07kllllkk7980.',
                 '.....0789a9887980.', '.....078998870jj0.', '.....0788078870ii0',
                 '.....0787078870...', '.....0870078870...', w=18)
SPEAR_TIP = ['..0..', '.050.', '.060.', '05640', '04530', '.040.', '0ddd0', '.0d0.']
HALBERD_TIP = ['..0....', '.050...', '.0600..', '056440.', '0453350', '.0400..', '0ddd0..', '.0d0...']
SHIELD_RECT = grid('00000000', '0a9999a0', '0lkkkkl0', '0999a990', '09a99a90',
                   '0999a990', '0lkkkkl0', '0a9999a0', '08888880', '00000000', w=8)
SHIELD_PLANK = grid('.000000.', '0hghhgh0', '0ighigh0', '0hgh3gh0', '0hgh4gh0',
                    '0ighigh0', '0hghhgh0', '.000000.', w=8)

BLUE = {'b': '8', 'c': '9', 'd': 'a'}  # red tabard -> blue
HEMP = {'7': 'U', '8': 'T', '9': 'S', 'a': 'S', 'b': 'k', 'c': 'l', 'd': 'm',
        '1': 'U', '2': 'T', '3': 'T', '4': 'S'}
KNIGHT_GRID = from_img(Image.open(KNIGHT / 'SW-pixel.png').convert('RGBA').crop((0, 0, 48, 48)))


def knight():
    return [r[:] for r in KNIGHT_GRID]


def put_head(g, head, x=18, y=11):
    erase(g, (x, y, x+13, y+12))
    overlay(g, head, x, y)


def skin_hand(g):
    recolor(g, {'l': 'j', 'k': 'i'}, (13, 21, 18, 27))


def polearm(g, tip):
    erase(g, (13, 17, 18, 23))
    overlay(g, ['.000.', '0jjj0'], 13, 21)
    for y in range(8, 36):
        if g[y][15] in '.0' or not 21 <= y <= 26:
            g[y][15] = 'g'
    overlay(g, tip, 13, 1)


def swap_shield(g, shield, y):
    erase(g, (23, 25, 31, 33))
    overlay(g, shield, 22, y)


def infantry(i):
    g = knight()
    if i == 1:
        recolor(g, BLUE, (13, 23, 31, 33)); put_head(g, HEAD_IRON); skin_hand(g)
    elif i == 2:
        overlay(g, [r.replace('.', ',') for r in BODY_OPEN], 13, 25)
        put_head(g, HEAD_HOOD); polearm(g, SPEAR_TIP)
    elif i == 3:
        recolor(g, BLUE, (13, 23, 31, 33)); put_head(g, HEAD_CONE); skin_hand(g)
        swap_shield(g, SHIELD_RECT, 24)
    elif i == 4:
        put_head(g, HEAD_PLUME); polearm(g, HALBERD_TIP)
        recolor(g, {'h': '9', 'g': '8', 'i': 'a', 'f': '7'}, (22, 25, 31, 33))
    return g


def bandit(i):
    """All bandits carry the sword, their equipped weapon."""
    g = knight()
    if i in (1, 3):
        overlay(g, [r.replace('.', ',') for r in BODY_OPEN], 13, 25)
    recolor(g, HEMP, (13, 23, 31, 33))
    skin_hand(g)
    head = {1: HEAD_TOPKNOT, 2: HEAD_YHOOD, 3: HEAD_STRAW, 4: HEAD_TOPKNOT}[i]
    put_head(g, head)
    if i == 2:
        swap_shield(g, SHIELD_PLANK, 25)
    return g


# ---- cavalry: pixelized legacy render
CAV_SCALE = 0.18      # rider helmet ~10px, same as knight v6
CAV_CELL = (48, 56)
CAV_PIVOT = (24, 50)  # bottom row of the near front hoof
KPAL = np.array(list(KNIGHT_COLORS.values()), np.float32)
OUTLINE = KNIGHT_COLORS['0']


def cut_background(frame):
    """Flood-fill the baked checkerboard from the borders into alpha 0."""
    a = np.array(frame.convert('RGB')).astype(int)
    h, w, _ = a.shape
    bgish = (a.max(-1) - a.min(-1) < 14) & (a.mean(-1) > 185)
    bg = np.zeros((h, w), bool)
    q = deque([(y, x) for x in range(w) for y in (0, h-1)] + [(y, x) for y in range(h) for x in (0, w-1)])
    while q:
        y, x = q.popleft()
        if 0 <= y < h and 0 <= x < w and not bg[y, x] and bgish[y, x]:
            bg[y, x] = True
            q.extend(((y+1, x), (y-1, x), (y, x+1), (y, x-1)))
    return Image.fromarray(np.dstack([a, np.where(bg, 0, 255)]).astype(np.uint8), 'RGBA')


def quantize(im):
    a = np.array(im)
    dist = ((a[:, :, None, :3].astype(np.float32) - KPAL[None, None]) ** 2).sum(-1)
    a[:, :, :3] = KPAL[dist.argmin(-1)].astype(np.uint8)
    a[:, :, 3] = np.where(a[:, :, 3] >= 128, 255, 0)
    a[a[:, :, 3] == 0] = 0
    return a


N4 = ((1, 0), (-1, 0), (0, 1), (0, -1))


def despeckle(a):
    """Replace single pixels that match none of their four opaque neighbours."""
    h, w, _ = a.shape
    b = a.copy()
    for y in range(1, h-1):
        for x in range(1, w-1):
            c = tuple(a[y, x, :3])
            nb = [tuple(a[y+dy, x+dx, :3]) for dy, dx in N4 if a[y+dy, x+dx, 3]]
            if a[y, x, 3] and len(nb) == 4 and c not in nb and c != OUTLINE:
                b[y, x, :3] = Counter(nb).most_common(1)[0][0]
    return b


def outline_edges(a):
    """Darken silhouette pixels of regions at least 3px thick; keep thin lines (spear) intact."""
    h, w, _ = a.shape
    op = lambda y, x: 0 <= y < h and 0 <= x < w and a[y, x, 3] > 0
    b = a.copy()
    for y in range(h):
        for x in range(w):
            if op(y, x) and any(not op(y+dy, x+dx) and op(y-dy, x-dx) and op(y-2*dy, x-2*dx) for dy, dx in N4):
                b[y, x, :3] = OUTLINE
    return b


def cavalry_base():
    frame = cut_background(Image.open(CAVALRY_SRC).crop((0, 0, 280, 280)))
    frame = frame.crop(frame.getbbox())
    small = frame.resize((round(frame.width*CAV_SCALE), round(frame.height*CAV_SCALE)), Image.Resampling.BOX)
    a = quantize(small)
    for _ in range(2):
        a = despeckle(a)
    px = from_img(Image.fromarray(outline_edges(a)))
    g = blank(*CAV_CELL)
    ox, oy = CAV_PIVOT[0] - 13, CAV_PIVOT[1] - (len(px) - 1)
    overlay(g, px, ox, oy)
    return g, ox, oy


COATS = {
    'bay': {},
    'brown': {'l': 'h', 'i': 'g', 'k': 'g', 'h': 'f', 'g': 'f'},
    'black': {'l': '3', 'i': '2', 'k': '1', 'h': '1', 'g': '0', 'f': '0'},
    'grey': {'l': '6', 'i': '5', 'k': '4', 'h': '3', 'g': '2', 'f': '1'},
}


def cavalry(i):
    g, ox, oy = cavalry_base()
    coat, head = {1: ('bay', None), 2: ('brown', HEAD_IRON), 3: ('black', HEAD_HOOD), 4: ('grey', HEAD_CONE)}[i]
    # Horse occupies everything below the saddle line plus the head left of the rider.
    recolor(g, COATS[coat], (ox, oy+19, ox+27, oy+41))
    recolor(g, COATS[coat], (ox, oy+17, ox+13, oy+19))
    if head:
        put_head(g, head, ox+9, oy+2)
    overlay(g, ['.0.', '050', '060', '040'], ox+4, oy-2)
    return g


UNITS = [
    ('infantry', '경보병', infantry, (24, 38),
     ['철투구·환두도·원형 방패', '청색 두건·창', '원뿔 투구·장방형 방패', '붉은 깃 투구·극']),
    ('bandit', '황건 적병', bandit, (24, 38),
     ['상투+황건·칼', '황색 두건·칼·판자 방패', '삿갓+황건·칼', '상투+황건·칼·원형 방패']),
    ('cavalry', '경기병', cavalry, CAV_PIVOT,
     ['밤색 말·기사 투구·창', '흑갈색 말·철투구·창', '흑마·청색 두건·창', '백마·원뿔 투구·창']),
]


def validate(name, im, pivot):
    assert {p[3] for p in im.getdata()} <= {0, 255}, name
    x0, y0, x1, y1 = im.getbbox()
    assert y1 - 1 == pivot[1], (name, y1 - 1)
    assert y0 > 0 and x0 > 0 and x1 < im.width, name


def main():
    OUT.mkdir(parents=True, exist_ok=True)
    sc, pad, label, box = 5, 16, 40, 64
    rows = []
    for key, title, fn, pivot, notes in UNITS:
        cells = [('기사 v6 (기준)', to_img(knight()), (24, 38))]
        for i in range(1, 5):
            im = to_img(fn(i))
            validate(f'{key}-{i}', im, pivot)
            im.save(OUT / f'{key}-{i}.png')
            cells.append((f'{title} {i} · {notes[i-1]}', im, pivot))
        rows.append(cells)
    cw = box*sc + pad
    sheet = Image.new('RGBA', (pad + cw*5, pad + len(rows)*(box*sc + label + pad)), (236, 232, 220, 255))
    draw = ImageDraw.Draw(sheet)
    font = ImageFont.truetype(FONT, 17)
    for r, cells in enumerate(rows):
        y0 = pad + r*(box*sc + label + pad)
        for c, (text, im, pivot) in enumerate(cells):
            x0 = pad + c*cw
            draw.rectangle((x0, y0, x0 + box*sc - 1, y0 + box*sc - 1), fill=(150, 180, 110, 255))
            big = im.resize((im.width*sc, im.height*sc), Image.NEAREST)
            sheet.alpha_composite(big, (x0 + (32 - pivot[0])*sc, y0 + (58 - pivot[1])*sc))
            draw.text((x0, y0 + box*sc + 8), text, fill=(40, 36, 30), font=font)
    SHEET.parent.mkdir(parents=True, exist_ok=True)
    sheet.convert('RGB').save(SHEET)
    print(SHEET)


if __name__ == '__main__':
    main()
