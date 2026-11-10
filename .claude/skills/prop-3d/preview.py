#!/usr/bin/env python3
"""Preview 3D scene props with their model-cast shadow softened the way the game softens it, on a floor colour, 5x.

  python3 .claude/skills/prop-3d/preview.py OUT.png name [name ...]

Names are keys of tools/unit3d/scene_props3d.py PROPS. A prop without cast_shadow is drawn without a shadow."""
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'tools/unit3d'))
from PIL import Image, ImageFilter, ImageChops  # noqa: E402
import scene_props3d as SP  # noqa: E402

CORE, HALO, BLUR, PAD = 130, 110, 3, 8  # ver5/internal/sprite/render.go (grow 1)


def soft(mask):
    a = mask.getchannel('A').point(lambda v: 255 if v else 0)
    flat = Image.new('L', (a.width + 2 * PAD, a.height + 2 * PAD))
    flat.paste(a, (PAD, PAD))
    g = flat.filter(ImageFilter.MaxFilter(3))
    halo = g.filter(ImageFilter.BoxBlur(BLUR)).filter(ImageFilter.BoxBlur(BLUR))
    return ImageChops.lighter(g.point(lambda v: CORE if v else 0), halo.point(lambda v: v * HALO // 255))


def main():
    out, names = sys.argv[1], sys.argv[2:]
    tiles = []
    for n in names:
        im, pv = SP.PROPS[n]()
        bg = Image.new('RGBA', (max(110, im.width + 60), max(80, im.height + 40)), (178, 140, 96, 255))
        gx, gy = bg.width // 2 - 10, bg.height - 25
        if hasattr(im, 'shadow'):
            mask, mpv = im.shadow
            sh = Image.new("RGBA", (mask.width + 2 * PAD, mask.height + 2 * PAD), (50, 34, 20, 255))
            sh.putalpha(soft(mask))
            bg.alpha_composite(sh, (gx - mpv[0] - PAD, gy - mpv[1] - PAD))
        bg.alpha_composite(im, (gx - pv[0], gy - pv[1]))
        tiles.append(bg.resize((bg.width * 5, bg.height * 5), Image.NEAREST))
    sheet = Image.new('RGBA', (sum(t.width for t in tiles), max(t.height for t in tiles)))
    x = 0
    for t in tiles:
        sheet.paste(t, (x, 0))
        x += t.width
    sheet.save(out)
    print(out)


if __name__ == '__main__':
    main()
