#!/usr/bin/env python3
"""Render battle-map shots headlessly, with the real game renderer, for before/after comparison.

  python3 tools/mapshots/shots.py <label> [--stages 1,2,3] [--size 1600x1000] [--gif 4,6]

For each stage it picks camera spots from the map's terrain (open grass, forest, wasteland, cliff top, camp, village, keep, castle
floor, cliff top, water, plus the forest again three times for wind), plays the `scenery` demo through
cmd/_uidemo-vm (no window) and writes reports/scenery/<label>/B0x_<shot>.png (--gif adds a 3 s forest loop). Run it from ver5/ under
`caffeinate -d -i`. The comparison page is built by tools/mapshots/compare.py. Requires Pillow."""
import argparse
import json
import os
import subprocess
import sys
import tempfile
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[2]
SCENARIO = ROOT / 'assets/scenario/hero'
FIRST, SPACING = 10, 30  # internal/gui/uidemo_scenery.go
WINDOW = 2  # a shot looks at the best (2*WINDOW+1)^2 cells


def load_cells(n):
    m = json.load(open(SCENARIO / f'maps/map{n:02d}.json'))
    play = m['Play']
    cells = {}
    for y in range(play['H']):
        for x in range(play['W']):
            cells[x, y] = m['Cells'].get(f"{x + play['X']},{y + play['Y']}", m.get('Default', 'g--'))
    return play['W'], play['H'], cells


def best_window(w, h, cells, pred, need):
    best = (0, None)
    for cy in range(WINDOW, h - WINDOW):
        for cx in range(WINDOW, w - WINDOW):
            n = sum(1 for dy in range(-WINDOW, WINDOW + 1) for dx in range(-WINDOW, WINDOW + 1)
                    if pred(cells[cx + dx, cy + dy]))
            if n > best[0]:
                best = (n, (cx, cy))
    return best[1] if best[0] >= need else None


def tour(n):
    w, h, cells = load_cells(n)
    kinds = [
        ('grass', 2, lambda c: c[0] == 'g' and c[1] == '-', 14),
        ('forest', 2, lambda c: c[1] == 't', 5),
        ('waste', 2, lambda c: c[0] == 'd' and c[1] == '-', 5),
        ('cliff', 2, lambda c: c[0] == 'h', 5),
        ('camp', 3, lambda c: c[1] == 'k', 3),
        ('village', 3, lambda c: c[1] == 'v', 3),
        ('keep', 3, lambda c: c[1] == 'o', 1),
        ('castle', 2, lambda c: c[0] == 'i', 5),
        ('water', 2, lambda c: c[0] == 'w', 5),
    ]
    shots = [('overview', w / 2 - .5, h / 2 - .5, 1)]
    forest = None
    for name, zoom, pred, need in kinds:
        spot = best_window(w, h, cells, pred, need)
        if spot:
            shots.append((name, spot[0], spot[1], zoom))
            if name == 'forest':
                forest = spot
    if forest:
        shots += [(f'wind{k}', forest[0], forest[1], 3) for k in (1, 2, 3, 4)]
    return shots


def run_stage(n, outdir, size):
    shots = tour(n)
    frames = [FIRST + SPACING * k for k in range(len(shots))]
    env = dict(os.environ, SCENERY_TOUR=';'.join(f'{u},{v},{z}' for _, u, v, z in shots))
    with tempfile.TemporaryDirectory(prefix='mapshots') as tmp:
        cmd = ['caffeinate', '-d', '-i', 'go', 'run', './cmd/_uidemo-vm', '-ui', 'scenery', '-stage', f'B{n:02d}',
               '-frames', str(frames[-1] + 1), '-at', ','.join(map(str, frames)), '-out', tmp, '-size', size]
        r = subprocess.run(cmd, cwd=ROOT, env=env, capture_output=True, text=True)
        if r.returncode:
            sys.exit(f'stage {n}: {r.stderr[-800:]}')
        sw, sh = map(int, size.split('x'))
        for (name, *_), f in zip(shots, frames):
            im = Image.open(Path(tmp) / f'frame-{f:04d}.png').convert('RGB')
            if name != 'overview':  # the centre is the target; keep the panels out of the picture
                cw, ch = int(sw * .72), int(sh * .66)
                im = im.crop(((sw - cw) // 2, (sh - ch) // 2, (sw + cw) // 2, (sh + ch) // 2))
            im.save(outdir / f'B{n:02d}_{name}.png')
    print(f'B{n:02d}: {len(shots)} shots')


def run_gif(n, outdir, size, kind):
    """a short loop of the forest spot at 60 fps (recorded by the host), cropped around the camera target"""
    shots = tour(n)
    forest = next((sh for sh in shots if sh[0] == kind), None)
    if not forest:
        return
    name = 'sway' if kind == 'forest' else kind
    sw, sh_ = map(int, size.split('x'))
    cw, ch = 640, 400
    env = dict(os.environ, SCENERY_TOUR=f'{forest[1]},{forest[2]},3')
    with tempfile.TemporaryDirectory(prefix='mapshots') as tmp:
        mp4 = Path(tmp) / 'loop.mp4'
        cmd = ['caffeinate', '-d', '-i', 'go', 'run', './cmd/_uidemo-vm', '-ui', 'scenery', '-stage', f'B{n:02d}',
               '-frames', '200', '-mp4', str(mp4), '-size', size]
        r = subprocess.run(cmd, cwd=ROOT, env=env, capture_output=True, text=True)
        if r.returncode:
            sys.exit(f'gif B{n:02d}: {r.stderr[-800:]}')
        vf = (f'crop={cw}:{ch}:{(sw - cw) // 2}:{(sh_ - ch) // 2},fps=20,split[a][b];'
              '[a]palettegen=max_colors=96[p];[b][p]paletteuse=dither=none')
        subprocess.run(['ffmpeg', '-y', '-loglevel', 'error', '-ss', '0.4', '-t', '3', '-i', str(mp4), '-vf', vf,
                        '-loop', '0', str(outdir / f'B{n:02d}_{name}.gif')], check=True)
    print(f'B{n:02d}: {name}.gif')


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('label')
    ap.add_argument('--stages', default='1,2,3,4,5,6,7')
    ap.add_argument('--size', default='1600x1000')
    ap.add_argument('--gif', default='', help='stages to record a 3 s loop for, e.g. 4,6')
    ap.add_argument('--gifkind', default='forest', help='which spot the loop shows: forest, camp or village')
    a = ap.parse_args()
    outdir = ROOT / 'reports/scenery' / a.label
    outdir.mkdir(parents=True, exist_ok=True)
    for n in map(int, filter(None, a.stages.split(','))):
        run_stage(n, outdir, a.size)
    for n in filter(None, a.gif.split(',')):
        run_gif(int(n), outdir, a.size, a.gifkind)


if __name__ == '__main__':
    main()
