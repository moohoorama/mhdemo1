#!/usr/bin/env python3
"""Make the dragon pattern behind the title and in the dialogue box: assets/graphics/ui/dragon.png.

The source is a seamless 1254x1254 painting of dragons in clouds (dark teal). It is shrunk to a
quarter and turned to grey with its luminance stretched so the brightest stroke is white; the game
tints it with a colour scale (the default teal, or the speaker's faction colour).

    python3 tools/ui/dragon_tile.py ~/Downloads/shot0.png
"""
import pathlib
import sys

from PIL import Image

OUT = pathlib.Path(__file__).resolve().parents[2] / "assets" / "graphics" / "ui" / "dragon.png"


def main(src):
    im = Image.open(src).convert("RGB")
    im = im.resize((im.width // 4, im.height // 4), Image.LANCZOS)
    grey = im.convert("L")
    top = sorted(grey.getdata())[int(grey.width * grey.height * .999)]
    grey = grey.point(lambda v: min(255, round(v * 255 / top)))
    grey.save(OUT, optimize=True)
    print(OUT, grey.size)


if __name__ == "__main__":
    main(sys.argv[1])
