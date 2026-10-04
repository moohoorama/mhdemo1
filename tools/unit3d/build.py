#!/usr/bin/env python3
"""Build the ver2 unit sprites: 8 directions x 5 actions x 4 frames. Requires Pillow and numpy.

Every unit is rendered from the legacy 3D model with cel shading (render.py), then
a hand-drawn 2D head for that direction is stamped on the projected head centre
(heads.py) and translucent attack effects are composited (effects.py). Output
follows the knight v6 layout: {D}-pixel.png (4 frames x 5 actions) and frames.json
per unit, with the cell cropped to the union of the unit's frames, plus
factions.json and a review sheet.

  python3 tools/unit3d/build.py                  # idle frame 0, all directions (review only)
  python3 tools/unit3d/build.py --full           # every frame, sheets + frames.json
  python3 tools/unit3d/build.py --full --unit guanyu
"""
import argparse
import json
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

import effects as E
import factions as F
import heads as Hd
import render as R
import units as U
import looks

ROOT = Path(__file__).resolve().parents[2]
OUT = ROOT / 'tools/spritetool/assets/ver2-units'
SHEET = ROOT / 'output/ver2-units.png'
FONT = '/System/Library/Fonts/AppleSDGothicNeo.ttc'
DIRECTIONS = R.DIRECTIONS
ANIMATIONS = ['idle', 'walk', 'attack', 'hit', 'exhausted']
# ms per frame; attack holds the gathered power (frame 2) and the strike (frame 3).
DURATIONS = {'idle': 180, 'walk': 120, 'attack': [180, 280, 240, 200], 'hit': 120, 'exhausted': 220}

UNITS = [
    ('infantry', '경보병', U.infantry, Hd.heads(Hd.IRON, Hd.PLUME)),
    ('bandit', '황건 적병', U.bandit, Hd.heads(Hd.TOPKNOT, Hd.BAND)),
    ('spearman', '창병', U.spearman, Hd.heads(Hd.CONE, Hd.PLUME)),
    ('archer', '궁병', U.archer, Hd.heads(Hd.HOOD, Hd.HOOD_CLOTH)),
    ('cavalry', '경기병', U.cavalry, Hd.heads(Hd.IRON, Hd.PLUME)),
    ('guanyu', '관우', U.guanyu, Hd.heads(Hd.GUANYU, skin='KL', lid='K')),
    ('zhangfei', '장비', U.zhangfei, Hd.heads(Hd.ZHANGFEI, Hd.PLUME, skin='gh', lid='g')),
]
UNITS += looks.chosen()  # ver4 additions: strategist class and heroes (candidates in looks.py)
# Working canvas; each unit is cropped afterwards to the union of all its frames.
SPEC = dict(cell=(192, 160), pivot=(96, 116), scale=11.5)
R.add_ramps(U.RAMPS)


def frame(build, head_set, spec, d, row, col, lift=1, style=None):
    fr = build(row, col, style=style) if style else build(row, col)
    pivot = (spec['pivot'][0], spec['pivot'][1] - lift)
    gr = R.render(fr['faces'], d, spec['scale'], cell=spec['cell'], pivot=pivot)

    def to_px(p):
        x, y = R.project(p, d, spec['scale'], pivot)
        return x - .5, y - .5

    Hd.stamp(gr, head_set, d, fr['face'], *to_px(fr['head']))
    sprite = Hd.to_img(gr)
    if not fr['effects']:
        return sprite
    under, over = E.draw(fr['effects'], to_px, spec['cell'])
    under.alpha_composite(sprite)
    under.alpha_composite(over)
    return under


def build_unit(key, build, head_set, full):
    """Render every frame on the working canvas, then crop all to one tight cell."""
    rows = ANIMATIONS if full else ANIMATIONS[:1]
    cols = 4 if full else 1
    images = {}
    for d in DIRECTIONS:
        # Per direction, put the lowest idle pixel (foot outline) on the pivot row.
        lift = frame(build, head_set, SPEC, d, 0, 0, lift=0).getbbox()[3] - 1 - SPEC['pivot'][1]
        for row in range(len(rows)):
            for col in range(cols):
                images[d, row, col] = frame(build, head_set, SPEC, d, row, col, lift)
        print(key, d, flush=True)
    boxes = [im.getbbox() for im in images.values()]
    x0, y0 = min(b[0] for b in boxes) - 1, min(b[1] for b in boxes) - 1
    x1, y1 = max(b[2] for b in boxes) + 1, max(b[3] for b in boxes) + 1
    W, H = SPEC['cell']
    assert x0 >= 0 and y0 >= 0 and x1 <= W and y1 <= H, (key, 'frames reach the working canvas edge')
    w, h = x1 - x0, y1 - y0
    pivot = [SPEC['pivot'][0] - x0, SPEC['pivot'][1] - y0]
    images = {k: im.crop((x0, y0, x1, y1)) for k, im in images.items()}
    for d in DIRECTIONS:
        assert images[d, 0, 0].getbbox()[3] - 1 == pivot[1], (key, d, 'idle ground row')
    firsts = {d: images[d, 0, 0] for d in DIRECTIONS}
    if not full:
        return firsts, pivot
    out = OUT / key
    out.mkdir(parents=True, exist_ok=True)
    frames = []
    for di, d in enumerate(DIRECTIONS):
        sheet = Image.new('RGBA', (w*4, h*len(ANIMATIONS)))
        for row, action in enumerate(ANIMATIONS):
            for col in range(4):
                sheet.alpha_composite(images[d, row, col], (col*w, row*h))
                frames.append({'direction': d, 'animation': action, 'frame': col,
                               'index': di*20 + row*4 + col, 'crop': [col*w, row*h, w, h]})
        sheet.save(out / f'{d}-pixel.png')
    manifest = {'version': 1, 'unit': key, 'directions': DIRECTIONS, 'animations': ANIMATIONS,
                'cell': [w, h], 'pivot': pivot, 'frame_durations_ms': DURATIONS,
                'palette': R.PALETTE + list(R.EXTRA.values()), 'frames': frames}
    (out / 'frames.json').write_text(json.dumps(manifest, indent=1, ensure_ascii=False) + '\n')
    return firsts, pivot


def review_sheet(results):
    sc, pad, label = 4, 12, 34
    bw = max(im.width for _, firsts, _ in results for im in firsts.values())
    above = max(p[1] for _, _, p in results)
    below = max(im.height - p[1] for _, firsts, p in results for im in firsts.values())
    bh = above + below
    sheet = Image.new('RGBA', (pad + 8*(bw*sc + pad), pad + len(results)*(bh*sc + label + pad)), (236, 232, 220, 255))
    draw = ImageDraw.Draw(sheet)
    font = ImageFont.truetype(FONT, 18)
    for r, (title, firsts, pivot) in enumerate(results):
        y0 = pad + r*(bh*sc + label + pad)
        for c, d in enumerate(DIRECTIONS):
            im = firsts[d]
            x0 = pad + c*(bw*sc + pad)
            draw.rectangle((x0, y0, x0 + bw*sc - 1, y0 + bh*sc - 1), fill=(150, 180, 110, 255))
            ox, oy = (bw//2 - pivot[0])*sc, (above - pivot[1])*sc
            sheet.alpha_composite(im.resize((im.width*sc, im.height*sc), Image.NEAREST), (x0 + ox, y0 + oy))
            draw.text((x0, y0 + bh*sc + 6), f'{title} · {d}', fill=(40, 36, 30), font=font)
    SHEET.parent.mkdir(parents=True, exist_ok=True)
    sheet.convert('RGB').save(SHEET)
    print(SHEET)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--full', action='store_true', help='render every action frame')
    ap.add_argument('--unit', choices=[u[0] for u in UNITS])
    args = ap.parse_args()
    OUT.mkdir(parents=True, exist_ok=True)
    (OUT / 'factions.json').write_text(json.dumps(F.table(), indent=1, ensure_ascii=False) + '\n')
    results = []
    for key, title, build, head_set in UNITS:
        if args.unit and key != args.unit:
            continue
        firsts, pivot = build_unit(key, build, head_set, args.full)
        results.append((title, firsts, pivot))
    review_sheet(results)


if __name__ == '__main__':
    main()
