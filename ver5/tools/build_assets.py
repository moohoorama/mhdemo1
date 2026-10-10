#!/usr/bin/env python3
"""Rebuild the generated pieces of assets/graphics that have a source in the repository.

  python3 ver5/tools/build_assets.py

Portraits: tools/illustrations/portraits/<key>.png (and the style reference as liubei) scaled to 256x256 under
assets/graphics/portraits/. Sprites are built by tools/spritebuild/build.py. The tileset, structures, props and
the map draw data under assets/graphics/map and maps come from ver3/ver4's terrain tools and are kept as built;
rebuild them with ver4/tools/build_assets.py (ver3/tools/exportmap) when a map changes. Requires Pillow."""
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[2]
OUT = ROOT / 'ver5/assets/graphics/portraits'
SIZE = 256


def main():
    OUT.mkdir(parents=True, exist_ok=True)
    sources = {p.stem: p for p in sorted((ROOT / 'tools/illustrations/portraits').glob('*.png'))}
    sources['liubei'] = ROOT / 'tools/illustrations/style-samples/4-ink-wash.png'  # the style reference is his portrait
    for key, src in sorted(sources.items()):
        Image.open(src).convert('RGB').resize((SIZE, SIZE), Image.LANCZOS).save(OUT / f'{key}.png')
    print('portraits', len(sources))


if __name__ == '__main__':
    main()
