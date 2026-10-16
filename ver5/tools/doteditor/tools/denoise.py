#!/usr/bin/env python3
"""Optional clean-up of 3D-converted tag grids: the cel-shaded render leaves single pixels of another tone inside a
flat area and 1-px bumps on tone boundaries. Only tones of ONE material ramp (iron a b c, skin d e f, faction j k l,
...) are merged into each other, so material borders, outlines, the silhouette, eyes / mouth and effects never change.

  python3 denoise.py <id> [<id> ...] [--dry-run]     # units/<id>.yaml in place (prints the changed pixel count)
  python3 denoise.py <id> --plume-head              # a grid3d.py sheet: its pasted head box is left alone
  python3 grid3d_mounted.py <id> --unit K --denoise  # the same pass on every converted frame before it is written
  python3 grid3d_mounted.py <id> --unit K --denoise-only
                                                     # clean heroes/<id>/grid in place, the stamped 2D head kept
Without a head mask (plain denoise.py on a mounted-kit sheet) the 2D head is cleaned too: use --denoise-only there.

Rules (a pixel's neighbours are its 4 side neighbours unless said otherwise):
  speck — one pixel, or two touching ones, of a tag no other pixel around them has (8-neighbours), whose side
          neighbours are all tones of the same ramp, apart from at most one outline side for a single pixel
          (none for a pair; more is a small part such as a fist, or a rim light): it takes the tone most of
          those neighbours have (ties: the 8-neighbour count, then the ramp's middle tone);
  bump  — a pixel with one side neighbour of its own tag, three of one other tone B of the same ramp, and its own tag
          again on a diagonal (it sticks out of its tone region by one pixel: a 1-px zig-zag step) becomes B.
          With no diagonal partner it is the tip of a 1-px line and stays.
Never changed: outline, background, effects (anything outside the ten ramps protects its neighbours), a pixel next to
another material (a rivet, knuckle plate, mouth), iron1 glints, pixels in the protect mask (the 2D head).
Candidates come from the original grid; one next to an already changed pixel is skipped, so a change never cascades
along a line and two pixels never swap. Hand-drawn sheets must not be run through this.
"""
import argparse
import re
import sys

import dots

RAMPS = ['abc', 'def', 'ghi', 'jkl', 'mno', 'pqr', 'stu', 'vwx', 'yzA', 'BCD']
RAMP_OF = {c: r for r in RAMPS for c in r}
PLUME = re.compile(r'#(?:j|jk|kj|jj)#')
N4 = ((0, -1), (-1, 0), (1, 0), (0, 1))
N8 = N4 + ((-1, -1), (1, -1), (-1, 1), (1, 1))
MAX_OUTLINE = {1: 1, 2: 0}  # outline sides a 1-px / 2-px speck may touch (more: a small part such as a fist, or a rim light)
GLINT = 'a'  # iron1: a 1-2 px metal highlight is a deliberate glint (bolt tip, prod end, rivet) and is never merged


def _speck(g, x, y, t):
    """The pixels of tag t 8-connected to (x, y), or None when there are more than two."""
    cells, todo = {(x, y)}, [(x, y)]
    while todo:
        cx, cy = todo.pop()
        for dx, dy in N8:
            nx, ny = cx + dx, cy + dy
            if (nx, ny) not in cells and 0 <= ny < len(g) and 0 <= nx < len(g[0]) and g[ny][nx] == t:
                if len(cells) == max(MAX_OUTLINE):
                    return None
                cells.add((nx, ny))
                todo.append((nx, ny))
    return cells


def _ring(g, cells, nbrs):
    out = []
    for cx, cy in cells:
        for dx, dy in nbrs:
            p = (cx + dx, cy + dy)
            if p in cells:
                continue
            if not (0 <= p[1] < len(g) and 0 <= p[0] < len(g[0])):
                return None
            out.append(g[p[1]][p[0]])
    return out


def _decide(g, x, y, protect):
    """[(x, y, new tag)] for the speck or bump at (x, y), or []."""
    t = g[y][x]
    ramp = RAMP_OF.get(t)
    if not ramp or t == GLINT:
        return []
    cells = _speck(g, x, y, t)
    if cells:
        if protect and any(protect[cy][cx] for cx, cy in cells):
            return []
        ring = _ring(g, cells, N4)
        if ring is None or any(c != '#' and c not in RAMP_OF for c in ring):
            return []  # background (silhouette), effect or translucent pixel next to the speck
        tones = [c for c in ring if RAMP_OF.get(c) == ramp]
        if len(tones) + ring.count('#') < len(ring) or ring.count('#') > MAX_OUTLINE[len(cells)]:
            return []  # touches another material (may be drawn on purpose), or lies along the outline
        ring8 = _ring(g, cells, N8)
        new = max((c for c in ramp if c != t), key=lambda c: (tones.count(c), ring8.count(c), c == ramp[1]))
        return [(cx, cy, new) for cx, cy in cells]
    if protect and protect[y][x] or not (0 < x < len(g[0]) - 1 and 0 < y < len(g) - 1):
        return []
    n4 = [g[y + dy][x + dx] for dx, dy in N4]
    others = [c for c in n4 if c != t]
    if len(others) == 3 and others.count(others[0]) == 3 and RAMP_OF.get(others[0]) == ramp \
            and sum(g[y + dy][x + dx] == t for dx, dy in N8[4:]) >= 1:
        return [(x, y, others[0])]  # bump: one pixel sticking out of its tone region into a flat tone B
    return []


def clean(grid, protect=None):
    """A cleaned copy of a char grid (rows of tag chars as in the grid files) and the number of changed pixels.
    protect: optional same-sized grid of booleans for pixels that must not change (e.g. a stamped 2D head)."""
    g = [list(r) for r in grid]
    out = [r[:] for r in g]
    changed = set()
    for y in range(len(g)):
        for x in range(len(g[0])):
            if (x, y) in changed:
                continue
            todo = _decide(g, x, y, protect)
            if any((cx + dx, cy + dy) in changed for cx, cy, _ in todo for dx, dy in N4):
                continue  # next to a pixel changed already: no cascades along a line, no swaps
            for cx, cy, new in todo:
                out[cy][cx] = new
                changed.add((cx, cy))
    return out, len(changed)


def plume_head(grid):
    """Protect mask for a grid3d.py sheet: the 12x15 head box it pasted from the base sheet, found by the same plume
    pattern grid3d.py uses (the first '#j#'-like run from the top)."""
    mask = [[False] * len(grid[0]) for _ in grid]
    for top, row in enumerate(grid):
        m = PLUME.search(''.join(row))
        if m:
            px = m.start() + 1
            for y in range(top, min(top + 12, len(grid))):
                for x in range(max(px - 7, 0), min(px + 8, len(row))):
                    mask[y][x] = True
            break
    return mask


def clean_doc(doc, protect=None):
    """Clean every frame of a with_alpha() doc in place; only fully opaque pixels are candidates or neighbours.
    protect(grid) -> boolean grid or None (grid: the frame as tag chars). Returns the changed pixel count."""
    total = 0
    for f in doc['frames']:
        grid = [[dots.TAG_CHARS[t - 1] if t and a == 'ff' else '.' for t, a in zip(trow, arow)]
                for trow, arow in zip(f['tags'], f['alpha'])]
        out, n = clean(grid, protect(grid) if protect else None)
        if n:
            for y, row in enumerate(out):
                for x, c in enumerate(row):
                    if c != grid[y][x]:
                        f['tags'][y][x] = dots.TAG_CHARS.index(c) + 1
        total += n
    return total


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument('ids', nargs='+')
    ap.add_argument('--dry-run', action='store_true')
    ap.add_argument('--plume-head', action='store_true', help='keep the head box a grid3d.py sheet pasted from its base')
    a = ap.parse_args()
    for uid in a.ids:
        doc = dots.with_alpha(dots.load(uid))
        n = clean_doc(doc, plume_head if a.plume_head else None)
        if n and not a.dry_run:
            dots.save(doc)
        print(uid, n, 'pixels changed', '(dry run)' if a.dry_run else '')


if __name__ == '__main__':
    sys.exit(main())
