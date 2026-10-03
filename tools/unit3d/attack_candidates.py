#!/usr/bin/env python3
"""Attack style candidates (SW view) for picking one style per ver2 unit.

Renders every style of each weapon family for the listed units, so backups in
attacks.py can be compared again before changing attacks.SELECTED.

  python3 tools/unit3d/attack_candidates.py
  -> output/ver2-attack-candidates.png (frames) and .gif (playback)
"""
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

import attacks as A
import build as V

ROOT = Path(__file__).resolve().parents[2]

OUT = ROOT / 'output/ver2-attack-candidates'
DIRECTION = 'SW'
STYLES = {'infantry': A.SWORD, 'bandit': A.SWORD, 'archer': A.BOW, 'cavalry': A.SPEAR}  # backup comparison
DURATIONS = [180, 280, 240, 200]  # gather, hold the gathered power, strike, follow through
CELL, PIVOT = (112, 88), (56, 64)


def main():
    sc, pad, label = 3, 10, 30
    units = [u for u in V.UNITS if u[0] in STYLES]
    cols = 4
    box_w, box_h = CELL[0]*sc, CELL[1]*sc
    width = pad + cols*(box_w + pad)
    height = pad + len(units)*(box_h + label + pad)
    font = ImageFont.truetype(V.FONT, 18)
    frames = []
    cache = {}
    for key, title, build, heads in units:
        big = dict(V.SPEC, cell=CELL, pivot=PIVOT)
        lift = V.frame(build, heads, big, DIRECTION, 0, 0, lift=0).getbbox()[3] - 1 - PIVOT[1]
        for style in STYLES[key]:
            for f in range(4):
                cache[key, style, f] = V.frame(build, heads, big, DIRECTION, 2, f, lift, style=style)
        print(key, flush=True)
    for f in range(4):
        sheet = Image.new('RGBA', (width, height), (236, 232, 220, 255))
        draw = ImageDraw.Draw(sheet)
        for r, (key, title, *_rest) in enumerate(units):
            y0 = pad + r*(box_h + label + pad)
            for c, (style, spec) in enumerate(STYLES[key].items()):
                x0 = pad + c*(box_w + pad)
                draw.text((x0, y0), f'{title} {c + 1} · {spec["name"]}', fill=(40, 36, 30), font=font)
                draw.rectangle((x0, y0 + label, x0 + box_w - 1, y0 + label + box_h - 1), fill=(148, 179, 110, 255))
                im = cache[key, style, f]
                sheet.alpha_composite(im.resize((box_w, box_h), Image.NEAREST), (x0, y0 + label))
        frames.append(sheet.convert('RGB'))
    OUT.parent.mkdir(parents=True, exist_ok=True)
    frames[0].save(f'{OUT}.gif', save_all=True, append_images=frames[1:], duration=DURATIONS, loop=0)
    strip = Image.new('RGB', (width*4 + 30, height), (255, 255, 255))
    for i, im in enumerate(frames):
        strip.paste(im, (i*(width + 10), 0))
    strip.save(f'{OUT}.png')
    print(f'{OUT}.gif')


if __name__ == '__main__':
    main()
