#!/usr/bin/env python3
"""Snow-covered versions of the story-scene trees and bushes, packed beside (never into) the originals.

  uv run --with numpy --with pillow python ver5/tools/snow_props.py [--amount 1.0]

Reads every looping tree strip of assets/graphics/index.json "anims" (the sway sheet) and the static pictures of the
same names (stree_*, plus the sprouting field) and turns each pixel by its own colour:

  leaf green (hue 60-170)     keeps its light and shade but turns white: the lit side is almost pure white, the
                              shade stays a blue grey, so a crown reads as snow on leaves, not as a flat white blob
  bark brown (hue 0-55)       loses its colour to a cool grey of the same light
  outline (value <= 0.15)     a dark blue grey
  everything else, alpha      untouched

amount (0..1) blends the result with the original colour; 1.0 is full snow. The output is the same pictures at the
same size and pivot under the name <name>_snow (bamboo: <name>_snow_a, _b, _c, see BAMBOO_LEAVES), in assets/graphics/map/props_snow.png with props_snow.json (static
"sprites" and looping "anims" in the same shape as index.json's), which the game reads when a scene or area has
"Snow". The originals are only read, so running it again after the sheets change (a new tree) makes its snow version.
Same input, same output."""
import argparse
import json
from pathlib import Path

import numpy as np
from PIL import Image

ROOT = Path(__file__).resolve().parents[2]
GFX = ROOT / 'ver5/assets/graphics'
OUT_IMAGE, OUT_INDEX = 'map/props_snow.png', 'map/props_snow.json'
WIDTH = 1024
SUFFIX = '_snow'
STATIC_EXTRA = ('field_sprout',)  # ground pictures with green sprouts
TREE_PREFIX = 'stree_'

OUTLINE_V = 0.15  # the 1px outline (11,16,27) is the darkest value of the sheets
BARK_MIN_SAT = 0.18


def rule(hue, min_sat, dark, light, span, gamma=1.0, leaf=False):
    return {'leaf': leaf, 'hue': hue, 'sat': min_sat, 'dark': np.array(dark, float), 'light': np.array(light, float), 'span': span, 'gamma': gamma}


# Each family of props has its own rules, tried in order (the first that matches a pixel wins): the pixel's hue range
# and least saturation pick it, its value (span: the darkest and lightest of the palette) sets where it lies between
# the dark and the light colour (gamma below 1 lifts the middle toward the light). The outline is the family's own.
PROFILES = {
    # crowns: leaf green turns white (shade in blue grey), bark brown turns cool grey
    'tree': {'outline': (30, 38, 54), 'rules': [
        rule((60, 170), 0.30, (92, 108, 136), (252, 253, 255), (0.19, 0.84), 0.5),
        rule((0, 55), BARK_MIN_SAT, (40, 44, 54), (188, 194, 206), (0.2, 0.9)),
    ]},
    # bamboo: leaves nearly white with a faint green cast, stems a pale grey green; its palettes share yellow greens, so a
    # stem is told by shape: yellow green in a vertical run of stem_run pixels or more (rules: leaf, then stem)
    'bamboo': {'outline': (34, 52, 52), 'stem_run': 5, 'rules': [
        rule((0, 360), 0.30, (96, 128, 124), (238, 252, 246), (0.30, 0.75), 0.6, leaf=True),
        rule((0, 360), 0.30, (112, 134, 112), (206, 222, 196), (0.19, 0.84), 0.8),
    ]},
}
# the bamboo's leaves, three ways (a name <prop>_snow_<letter> each): none leaves the bare stems with snow on their
# tips, sparse keeps about a third of the leaf clumps, full keeps them all
BAMBOO_LEAVES = {'a': 'none', 'b': 'sparse', 'c': 'full'}
SPARSE_BLOCK, SPARSE_KEEP = 3, 3  # a clump is 3x3 px; one clump in 3 stays
CAP = (240, 248, 244)  # the white tip of a stem
CAP_ROWS = 3
BAMBOO = {'stree_bamboo_0', 'stree_bamboo_1', 'stree_bamboo_thin', 'stree_sasa'}


def profile_of(name):
    return 'bamboo' if name in BAMBOO else 'tree'


def hsv(rgb):
    """Hue (degrees), saturation and value (0..1) of float RGB 0..255, shape (..., 3)."""
    c = rgb / 255.0
    mx, mn = c.max(-1), c.min(-1)
    d = mx - mn
    r, g, b = c[..., 0], c[..., 1], c[..., 2]
    safe = np.where(d == 0, 1, d)
    h = np.where(mx == r, ((g - b) / safe) % 6, np.where(mx == g, (b - r) / safe + 2, (r - g) / safe + 4)) * 60
    h = np.where(d == 0, 0, h)
    return h, np.where(mx == 0, 0, d / np.where(mx == 0, 1, mx)), mx


def ramp(v, span, dark, light, gamma=1.0):
    t = np.clip((v - span[0]) / (span[1] - span[0]), 0, 1) ** gamma
    return dark + (light - dark) * t[..., None]


def stem_mask(h, s, v, alpha, run):
    """Yellow-green pixels that stand in a vertical run of at least run of them: a stem, not a leaf's lit tip."""
    cand = alpha & (v > OUTLINE_V) & (h < 100) & (s >= 0.30)
    keep = np.zeros_like(cand)
    for x in range(cand.shape[1]):
        y = 0
        col = cand[:, x]
        while y < len(col):
            if not col[y]:
                y += 1
                continue
            e = y
            while e < len(col) and col[e]:
                e += 1
            if e - y >= run:
                keep[y:e, x] = True
            y = e
    return keep


def rule_masks(prof, h, s, v, alpha):
    """The pixels each rule of the profile claims (before the first-wins order)."""
    if 'stem_run' in prof:
        stem = stem_mask(h, s, v, alpha, prof['stem_run'])
        body = alpha & (v > OUTLINE_V)
        return [body & ~stem, stem]
    return [(h >= r['hue'][0]) & (h <= r['hue'][1]) & (s >= r['sat']) for r in prof['rules']]


def snow(image, amount=1.0, profile='tree', leaves='full'):
    """The RGBA array image turned by the profile's rules; alpha is kept."""
    prof = PROFILES[profile]
    a = np.asarray(image, np.uint8)
    rgb = a[..., :3].astype(float)
    h, s, v = hsv(rgb)
    out = rgb.copy()
    done = (v <= OUTLINE_V)
    out[done & (a[..., 3] > 0)] = prof['outline']
    for r, hit in zip(prof['rules'], rule_masks(prof, h, s, v, a[..., 3] > 0)):
        hit = ~done & hit
        out[hit] = ramp(v, r['span'], r['dark'], r['light'], r['gamma'])[hit]
        done |= hit
    out = rgb + (out - rgb) * float(amount)
    res = a.copy()
    res[..., :3] = np.clip(np.rint(out), 0, 255).astype(np.uint8)
    if leaves != 'full':
        res = thin_leaves(res, a, prof, leaves, h, s, v)
    return res


def thin_leaves(res, src, prof, leaves, h, s, v):
    """res (the snow picture of one frame) with its leaves removed ('none') or thinned ('sparse'), the outline
    redrawn round what is left and, with no leaves, the stems' tips whitened."""
    alpha = src[..., 3] > 0
    outline = alpha & (v <= OUTLINE_V)
    leaf = np.zeros_like(alpha)
    for r, hit in zip(prof['rules'], rule_masks(prof, h, s, v, alpha)):
        if r['leaf']:
            leaf |= hit & ~outline
    ys, xs = np.mgrid[0:alpha.shape[0], 0:alpha.shape[1]]
    if leaves == 'sparse':
        bx, by = xs // SPARSE_BLOCK, ys // SPARSE_BLOCK
        keep = ((bx * 73856093) ^ (by * 19349663)) % SPARSE_KEEP == 0
        gone = leaf & ~keep
    else:
        gone = leaf
    body = alpha & ~outline & ~gone
    pad = np.pad(body, 1)
    rim = (pad[:-2, 1:-1] | pad[2:, 1:-1] | pad[1:-1, :-2] | pad[1:-1, 2:]) & ~body
    res = res.copy()
    res[gone | outline] = 0
    res[rim, :3] = prof['outline']
    res[rim, 3] = 255
    if leaves == 'none':
        above = np.cumsum(body, axis=0) - body  # stem pixels above in the column
        tip = body & (above < CAP_ROWS)
        res[tip, :3] = CAP
    return res


def pieces(index):
    """(name, kind, image rect source, entry) of everything to turn, sorted by name."""
    sheets = {}

    def sheet(path):
        if path not in sheets:
            sheets[path] = Image.open(GFX / path).convert('RGBA')
        return sheets[path]

    items = []
    for name, e in index['anims'].items():
        if name.startswith(TREE_PREFIX):
            x, y, w, h = e['rect']
            im = sheet(e['image']).crop((x, y, x + w * e['frames'], y + h))
            items.append((name, 'anim', im, e))
    for name, e in index['extra']['sprites'].items():
        if name.startswith(TREE_PREFIX) or name in STATIC_EXTRA:
            x, y, w, h = e['rect']
            items.append((name, 'sprite', sheet(index['extra']['image']).crop((x, y, x + w, y + h)), e))
    return sorted(items, key=lambda i: (i[1], i[0]))


def variants(name):
    """(name suffix, profile, leaves) of each picture made from a prop."""
    if name in BAMBOO:
        return [(SUFFIX + '_' + k, 'bamboo', v) for k, v in BAMBOO_LEAVES.items()]
    return [(SUFFIX, 'tree', 'full')]


def build(amount=1.0):
    index = json.load(open(GFX / 'index.json'))
    jobs = [(name + suffix, kind, im, e, profile, leaves) for name, kind, im, e in pieces(index)
            for suffix, profile, leaves in variants(name)]
    placed, x, y, row = [], 0, 0, 0
    for name, kind, im, e, profile, leaves in jobs:
        w, h = im.size
        if x + w > WIDTH:
            x, y, row = 0, y + row, 0
        placed.append((x, y, name, kind, im, e, profile, leaves))
        x, row = x + w, max(row, h)
    sheet = Image.new('RGBA', (WIDTH, y + row), (0, 0, 0, 0))
    out = {'version': 1, 'image': OUT_IMAGE, 'sprites': {}, 'anims': {}}
    for x, y, name, kind, im, e, profile, leaves in placed:
        w, h = im.size
        n = e['frames'] if kind == 'anim' else 1
        for i in range(n):  # each frame alone: the leaves of one never lend a neighbour its outline
            frame = np.asarray(im.crop((i * (w // n), 0, (i + 1) * (w // n), h)))
            sheet.paste(Image.fromarray(snow(frame, amount, profile, leaves), 'RGBA'), (x + i * (w // n), y))
        if kind == 'anim':
            out['anims'][name] = {k: v for k, v in e.items() if k not in ('image', 'rect')} | {
                'image': OUT_IMAGE, 'rect': [x, y, w // n, h]}
        else:
            out['sprites'][name] = {k: v for k, v in e.items() if k != 'rect'} | {'rect': [x, y, w, h]}
    sheet.save(GFX / OUT_IMAGE)
    with open(GFX / OUT_INDEX, 'w') as f:
        json.dump(out, f, indent=1, sort_keys=True)
        f.write('\n')
    print(f'{len(out["anims"])} strips, {len(out["sprites"])} pictures -> {OUT_IMAGE} {sheet.size}')


if __name__ == '__main__':
    ap = argparse.ArgumentParser()
    ap.add_argument('--amount', type=float, default=1.0)
    build(ap.parse_args().amount)
