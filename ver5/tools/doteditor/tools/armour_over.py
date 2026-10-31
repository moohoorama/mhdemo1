#!/usr/bin/env python3
"""A hero's 경기병 sheet in a heavier cavalry class: the class's armour pixels laid over every frame of the hero.

  python3 armour_over.py guanyu heavy_cavalry guanyu_heavy_cavalry [--light _ref_light]

Armour = the pixels where the class sheet differs from the plain rider converted the same way (--light: a working copy
made with `grid3d_mounted.py <id> --unit light_cavalry --base heavy_cavalry --denoise`; the light_cavalry sheet itself
was retouched and no longer matches the heavier classes frame by frame, e.g. its exhausted S), kept only from the
neck down: the class helmet's plume or crest would stand on the hero's own head. Pixels where the hero differs from
cavalry (the sheet it was drawn over: head, beard, weapon, trails) stay the hero's. Armour pixels take the class
sheet's tag and alpha and the hero's tag colours (a leather belt takes the hero's themeA). Writes units/<new>.yaml
with all frames; this script generates pixels, so look at the result (render.py) before using it.
"""
import argparse

import dots

HELMET_ROWS = 9  # rows of the light_cavalry head, from the helmet top, that are left to the hero


def helmet_top(tags):
    """First row of a block of five rows each 5+ pixels wide: the helmet (the spear and the plume are thinner)."""
    runs = [max((len(r) for r in ''.join('x' if t else ' ' for t in row).split()), default=0) for row in tags]
    return next(y for y in range(len(runs) - 4) if min(runs[y:y + 5]) >= 5)


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument('hero')
    ap.add_argument('cls')
    ap.add_argument('id')
    ap.add_argument('--light', default='light_cavalry')
    a = ap.parse_args()
    hero, base, cls = (dots.with_alpha(dots.load(u)) for u in (a.hero, 'cavalry', a.cls))
    light = dots.load_working(a.light)
    index = {u: {(f['animation'], f['direction'], f['frame']): f for f in d['frames']} for u, d in
             (('base', base), ('light', light), ('cls', cls))}
    py = hero['unit']['pivot'][1]
    dl, dk = light['unit']['pivot'][1] - py, cls['unit']['pivot'][1] - py  # hero rows -> light / class rows

    def at(f, key, x, y):
        rows = f[key]
        return rows[y][x] if 0 <= y < len(rows) else (0 if key == 'tags' else '00')

    for f in hero['frames']:
        k = (f['animation'], f['direction'], f['frame'])
        bf, lf, kf = index['base'][k], index['light'][k], index['cls'][k]
        neck = helmet_top(lf['tags']) - dl + HELMET_ROWS
        for y, row in enumerate(f['tags']):
            for x, t in enumerate(row):
                if t and t != bf['tags'][y][x]:
                    continue
                kt = at(kf, 'tags', x, y + dk)
                if y >= neck and kt != at(lf, 'tags', x, y + dl):
                    row[x], f['alpha'][y][x] = kt, at(kf, 'alpha', x, y + dk)
    hero['unit']['id'] = a.id
    print('wrote', dots.save(hero))


if __name__ == '__main__':
    main()
