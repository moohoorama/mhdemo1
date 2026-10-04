#!/usr/bin/env python3
"""Build the scene (평복) sprites: NW and SW x 7 motions x 4 frames (NE, SE are mirrored at runtime).

  python3 tools/unit3d/civilian_build.py [--unit KEY]   (the review sheet always shows every built unit)
  -> tools/spritetool/assets/ver4-civilians/<key>/{D}-pixel.png + frames.json
     output/ver4-civilians.png (review: SW/S of every motion's key frame)
"""
import argparse
import json
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

import build as V
import civilian as C

ROOT = Path(__file__).resolve().parents[2]
V.OUT = ROOT / 'tools/spritetool/assets/ver4-civilians'
V.ANIMATIONS = C.ANIMATIONS
V.DURATIONS = C.DURATIONS
V.DIRECTIONS = C.DIRECTIONS


def review(results):
    sc, pad, label = 4, 10, 24
    keys = list(results)
    cw = max(im.width for frames in results.values() for im in frames.values())
    ch = max(im.height for frames in results.values() for im in frames.values())
    cols = len(C.ANIMATIONS)
    sheet = Image.new('RGBA', (pad + cols*(cw*sc + pad), pad + len(keys)*(ch*sc + label + pad)), (236, 232, 220, 255))
    draw = ImageDraw.Draw(sheet)
    font = ImageFont.truetype(V.FONT, 16)
    for r, key in enumerate(keys):
        for c, name in enumerate(C.ANIMATIONS):
            im = results[key][name]
            x0, y0 = pad + c*(cw*sc + pad), pad + r*(ch*sc + label + pad)
            draw.text((x0, y0), f'{key} · {name}', fill=(40, 36, 30), font=font)
            draw.rectangle((x0, y0 + label, x0 + cw*sc - 1, y0 + label + ch*sc - 1), fill=(148, 179, 110, 255))
            sheet.alpha_composite(im.resize((im.width*sc, im.height*sc), Image.NEAREST), (x0, y0 + label))
    out = ROOT / 'output/ver4-civilians.png'
    sheet.convert('RGB').save(out)
    print(out)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--unit')
    args = ap.parse_args()
    results = {}
    for key, title, build, head in C.chosen():
        if not args.unit or key == args.unit:
            V.build_unit(key, build, head, True)
        d = V.OUT / key
        if not (d / 'frames.json').exists():
            continue
        meta = json.loads((d / 'frames.json').read_text())
        sw = Image.open(d / 'SW-pixel.png')
        w, h = meta['cell']
        results[key] = {name: sw.crop((2*w, r*h, 3*w, (r + 1)*h)) for r, name in enumerate(C.ANIMATIONS)}
    review(results)


if __name__ == '__main__':
    main()
