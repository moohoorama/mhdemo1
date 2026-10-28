#!/usr/bin/env python3
"""Add the ford and plain ground sets to assets/graphics/map/tileset.png and index.json.

Each set is 15 pictures: the whole tile at its first id and the edge where the level lies over the levels below at
id + corner mask (1..14), the layering of content/map_graphics.go. A set borrows its edge shapes from an existing set
-- the ford the grass-over-water edges (it meets the water), the plain the soil-over-grass edges -- and their shading:
each pixel of an edge keeps its shape's alpha and the ratio of its shape's colour to that set's whole tile.

  python3 ver5/tools/build_ground_tiles.py
"""
import json
from pathlib import Path

import numpy as np
from PIL import Image

GFX = Path(__file__).resolve().parents[1] / 'assets/graphics'
FORD, PLAIN = 70, 86


def main():
    index = json.load(open(GFX / 'index.json'))
    ts = index['tileset']
    sheet = np.array(Image.open(GFX / ts['image']).convert('RGBA')).astype(float)

    def tile(i):
        x, y, w, h, _, _ = ts['sprites'][str(i)]
        return sheet[y:y + h, x:x + w].copy()

    grass, soil = tile(23), tile(8)
    # the ford: shallow water over mud, blue-grey with the grass's grain; the plain: grass and soil half and half
    water = tile(0)
    opaque = grass[..., 3] > 0
    lum = (grass[..., :3].mean(axis=2, keepdims=True) / grass[opaque, :3].mean()).clip(.85, 1.15)
    ford = grass.copy()
    ford[..., :3] = (np.array([98, 120, 128]) * .7 + water[..., :3] * .3) * lum   # shallow water over mud: blue-grey
    rng = np.random.default_rng(3)
    spots = rng.random(ford.shape[:2])
    ford[(spots < .05) & opaque, :3] = [148, 174, 182]   # glints of standing water
    ford[(spots > .05) & (spots < .11) & opaque, :3] = [84, 92, 86]   # mud
    plain = grass.copy()
    plain[..., :3] = grass[..., :3] * .5 + soil[..., :3] * .5

    def edges(tex, shape_full, shape_base):
        whole = tile(shape_full)[..., :3].clip(1)
        out = [tex]
        for m in range(1, 15):
            s = tile(shape_base + m)
            e = tex.copy()
            e[..., :3] = (tex[..., :3] * (s[..., :3] / whole)).clip(0, 255)
            e[..., 3] = s[..., 3]
            out.append(e)
        return out

    sets = [(FORD, edges(ford, 23, 23)), (PLAIN, edges(plain, 8, 8))]
    h0 = sheet.shape[0]
    grown = np.zeros((h0 + 8 * len(sets), sheet.shape[1], 4))
    grown[:h0] = sheet
    for row, (first, pics) in enumerate(sets):
        for k, p in enumerate(pics):
            x, y = 16 * k, h0 + 8 * row
            grown[y:y + 8, x:x + 16] = p
            ts['sprites'][str(first + k)] = [x, y, 16, 8, 8, 0]
    Image.fromarray(grown.round().astype(np.uint8)).save(GFX / ts['image'])
    ts['ground']['ford'] = f'{FORD}-{FORD + 14} ({FORD} = full, {FORD}+mask = edge)'
    ts['ground']['plain'] = f'{PLAIN}-{PLAIN + 14} ({PLAIN} = full, {PLAIN}+mask = edge)'
    json.dump(index, open(GFX / 'index.json', 'w'), ensure_ascii=False, indent=1)
    open(GFX / 'index.json', 'a').write('\n')


if __name__ == '__main__':
    main()
