#!/usr/bin/env python3
"""Look candidates (SW view) for the ver4 units: idle frame 0 and the attack strike frame.

  python3 tools/unit3d/look_candidates.py [--unit KEY]  -> output/ver4-look-candidates[-KEY].png
"""
import argparse
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

import build as V
import looks as L

ROOT = Path(__file__).resolve().parents[2]
CELL, PIVOT = (84, 72), (42, 54)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--unit', choices=list(L.LOOKS))
    args = ap.parse_args()
    keys = [args.unit] if args.unit else list(L.LOOKS)
    sc, pad, label = 6, 12, 30
    bw, bh = CELL[0]*sc, CELL[1]*sc
    width = pad + 4*(bw + pad)
    height = pad + len(keys)*2*(bh + label + pad)
    sheet = Image.new('RGBA', (width, height), (236, 232, 220, 255))
    draw = ImageDraw.Draw(sheet)
    font = ImageFont.truetype(V.FONT, 20)
    big = dict(V.SPEC, cell=CELL, pivot=PIVOT)
    for r, key in enumerate(keys):
        for c, (title, build, head) in enumerate(L.LOOKS[key]):
            lift = V.frame(build, head, big, 'SW', 0, 0, lift=0).getbbox()[3] - 1 - PIVOT[1]
            for k, (row, col) in enumerate([(0, 0), (2, 2)]):
                im = V.frame(build, head, big, 'SW', row, col, lift)
                x0, y0 = pad + c*(bw + pad), pad + (2*r + k)*(bh + label + pad)
                text = f'{L.TITLES[key]} {c + 1} · {title}' if k == 0 else f'{L.TITLES[key]} {c + 1} · 공격'
                draw.text((x0, y0), text, fill=(40, 36, 30), font=font)
                draw.rectangle((x0, y0 + label, x0 + bw - 1, y0 + label + bh - 1), fill=(148, 179, 110, 255))
                sheet.alpha_composite(im.resize((bw, bh), Image.NEAREST), (x0, y0 + label))
            print(key, c + 1, flush=True)
    out = ROOT / f"output/ver4-look-candidates{'-' + args.unit if args.unit else ''}.png"
    out.parent.mkdir(parents=True, exist_ok=True)
    sheet.convert('RGB').save(out)
    print(out)


if __name__ == '__main__':
    main()
