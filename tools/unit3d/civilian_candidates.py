#!/usr/bin/env python3
"""Civilian look candidates: SW idle, SW walk and NW idle per candidate.

  python3 tools/unit3d/civilian_candidates.py  -> output/ver4-civilian-candidates.png
"""
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

import argparse

import build as V
import civilian as C

ROOT = Path(__file__).resolve().parents[2]
CELL, PIVOT = (60, 64), (30, 50)
VIEWS = [('SW', 0, 0, '대기'), ('SW', 1, 1, '걷기'), ('NW', 0, 0, '뒤(NW)')]  # scenes use NW and SW only


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--unit', nargs='*', choices=list(C.LOOKS))
    args = ap.parse_args()
    sc, pad, label = 5, 10, 26
    bw, bh = CELL[0]*sc, CELL[1]*sc
    keys = args.unit or list(C.LOOKS)
    cols = 4*len(VIEWS)
    sheet = Image.new('RGBA', (pad + cols*(bw + pad), pad + len(keys)*(bh + label + pad)), (236, 232, 220, 255))
    draw = ImageDraw.Draw(sheet)
    font = ImageFont.truetype(V.FONT, 17)
    spec = dict(V.SPEC, cell=CELL, pivot=PIVOT)
    for r, key in enumerate(keys):
        for c, (title, build, head) in enumerate(C.LOOKS[key]):
            for k, (d, row, col, view) in enumerate(VIEWS):
                lift = V.frame(build, head, spec, d, 0, 0, lift=0).getbbox()[3] - 1 - PIVOT[1]
                im = V.frame(build, head, spec, d, row, col, lift)
                x0, y0 = pad + (c*len(VIEWS) + k)*(bw + pad), pad + r*(bh + label + pad)
                text = f'{C.TITLES[key]} {c + 1} · {title}' if k == 0 else view
                draw.text((x0, y0), text, fill=(40, 36, 30), font=font)
                draw.rectangle((x0, y0 + label, x0 + bw - 1, y0 + label + bh - 1), fill=(148, 179, 110, 255))
                sheet.alpha_composite(im.resize((bw, bh), Image.NEAREST), (x0, y0 + label))
            print(key, c + 1, flush=True)
    out = ROOT / f"output/ver4-civilian-candidates{'-' + '-'.join(args.unit) if args.unit else ''}.png"
    sheet.convert('RGB').save(out)
    print(out)


if __name__ == '__main__':
    main()
