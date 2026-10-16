#!/usr/bin/env python3
"""Subset the display fonts to the Hangul the game can show.

Keeps KS X 1001 Hangul (2,350 syllables), ASCII, the punctuation the UI uses and every
character found in assets/data and assets/scenes.json, so story text never falls back.

Sources (OFL, Google Fonts):
  https://github.com/google/fonts/raw/main/ofl/nanumbrushscript/NanumBrushScript-Regular.ttf
  https://github.com/google/fonts/raw/main/ofl/notoserifkr/NotoSerifKR%5Bwght%5D.ttf

    python3 tools/subset_fonts.py <NanumBrushScript-Regular.ttf> <NotoSerifKR[wght].ttf>
"""
import pathlib
import sys

from fontTools import subset
from fontTools.ttLib import TTFont

ROOT = pathlib.Path(__file__).resolve().parent.parent
OUT = ROOT / "assets" / "fonts"


def charset():
    chars = set(chr(c) for c in range(0x20, 0x7F))
    for hi in range(0xB0, 0xC9):
        for lo in range(0xA1, 0xFF):
            try:
                chars.add(bytes([hi, lo]).decode("euc-kr"))
            except UnicodeDecodeError:
                pass
    chars.update("·…‘’“”→←↑↓▸▾▼▲×—–「」『』〈〉《》※○●◆◇■□★☆%")
    for path in list((ROOT / "assets" / "data").rglob("*.json")) + [ROOT / "assets" / "scenes.json"]:
        chars.update(c for c in path.read_text(encoding="utf-8") if ord(c) >= 0x20)
    return "".join(sorted(chars))


def save(font, text, out):
    opts = subset.Options()
    opts.layout_features = ["*"]
    opts.name_IDs = ["*"]
    opts.notdef_outline = True
    s = subset.Subsetter(opts)
    s.populate(text=text)
    s.subset(font)
    font.save(out)
    print(out.relative_to(ROOT), out.stat().st_size // 1024, "KiB")


def main():
    if len(sys.argv) != 3:
        sys.exit(__doc__)
    text = charset()
    save(TTFont(sys.argv[1]), text, OUT / "NanumBrushScript.ttf")
    save(TTFont(sys.argv[2]), text, OUT / "NotoSerifKR.ttf")


if __name__ == "__main__":
    main()
