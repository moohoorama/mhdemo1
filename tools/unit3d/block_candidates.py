#!/usr/bin/env python3
"""Block motion candidates (SW view): four styles x four weapon families, frames 2-3.

  python3 tools/unit3d/block_candidates.py  -> output/ver4-block-candidates.png
"""
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

import blocks as B
import build as V
import heads as Hd
import units as U

ROOT = Path(__file__).resolve().parents[2]
CELL, PIVOT = (96, 80), (48, 60)
FAMILIES = [
    ('경보병 (칼·방패)', U.infantry, B.foot('infantry'), Hd.heads(Hd.IRON, Hd.PLUME)),
    ('창병', U.spearman, B.foot('spearman'), Hd.heads(Hd.CONE, Hd.PLUME)),
    ('궁병', U.archer, B.bowman('archer'), Hd.heads(Hd.HOOD, Hd.HOOD_CLOTH)),
    ('관우 (기마)', U.guanyu, B.rider(kit='guanyu', coat='red', weapon='glaive'), Hd.heads(Hd.GUANYU, skin='KL', lid='K')),
]


def main():
    sc, pad, label = 4, 10, 28
    bw, bh = CELL[0]*sc, CELL[1]*sc
    cols = len(FAMILIES)*2
    sheet = Image.new('RGBA', (pad + cols*(bw + pad), pad + len(B.STYLES)*(bh + label + pad)), (236, 232, 220, 255))
    draw = ImageDraw.Draw(sheet)
    font = ImageFont.truetype(V.FONT, 18)
    spec = dict(V.SPEC, cell=CELL, pivot=PIVOT)
    for r, style in enumerate(B.STYLES):
        for c, (title, idle, block, head) in enumerate(FAMILIES):
            build = B.as_unit(idle, block, style)
            lift = V.frame(build, head, spec, 'SW', 0, 0, lift=0).getbbox()[3] - 1 - PIVOT[1]
            for k, f in enumerate([1, 2]):
                im = V.frame(build, head, spec, 'SW', 5, f, lift)
                x0, y0 = pad + (2*c + k)*(bw + pad), pad + r*(bh + label + pad)
                name = B.SWORD[style]['name']
                draw.text((x0, y0), f'{r + 1}. {name} · {title} · {f + 1}프레임', fill=(40, 36, 30), font=font)
                draw.rectangle((x0, y0 + label, x0 + bw - 1, y0 + label + bh - 1), fill=(148, 179, 110, 255))
                sheet.alpha_composite(im.resize((bw, bh), Image.NEAREST), (x0, y0 + label))
            print(style, title, flush=True)
    out = ROOT / 'output/ver4-block-candidates.png'
    sheet.convert('RGB').save(out)
    print(out)


if __name__ == '__main__':
    main()
