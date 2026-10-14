#!/usr/bin/env python3
"""Live progress of hero dot work for progress.html: renders and counts, judges nothing.

  live.py liubei guanyu liubei_noncombat guanyu_noncombat [--every 30] [--once] [--out live-x7k2]

Each round reads units/<id>.yaml with the grid working copy laid over it one file at a time (a file that does
not parse yet is counted as 편집 중 and keeps the yaml pixels), then writes heroes/live/<id>-<anim>.png
(rows = directions, columns = frames, 1x) and heroes/live/status.json. Changed frames are compared with the
base unit (combat) or with the unit's own yaml (noncombat, which still holds the starting copy).
"""
import argparse
import json
import os
import time
from datetime import datetime

from PIL import Image

import dots
from render import NAMES, image

LIVE = dots.HEROES / 'live'
REVIEW = 0.3  # extra time on top of drawing for review and fixes


def done_files(unit_id, anim):
    if unit_id.endswith('_noncombat'):
        return ['notes.md']
    if anim == 'idle':
        return ['model-sheet.md']
    if anim in ('hit', 'exhausted'):
        return ['notes-hit_exhausted.md', f'notes-{anim}.md']
    return [f'notes-{anim}.md']


def write_atomic(path, write):
    tmp = path.with_name(path.name + '.tmp')
    write(tmp)
    os.replace(tmp, path)


_yaml, _start = {}, {}


def working(unit_id):
    """The unit yaml (re-read only when it changes) with a fresh frame list for the grid overlay."""
    mtime = (dots.UNITS / f'{unit_id}.yaml').stat().st_mtime
    if _yaml.get(unit_id, (None,))[0] != mtime:
        _yaml[unit_id] = mtime, dots.with_alpha(dots.load(unit_id))
    doc = _yaml[unit_id][1]
    if unit_id not in _start:  # starting frames: the base unit, or the noncombat yaml as first read
        src = doc if unit_id.endswith('_noncombat') else dots.load(doc['unit']['source_base'])
        _start[unit_id] = {(f['animation'], f['direction'], f['frame']): f['tags'] for f in src['frames']}
    return {**doc, 'frames': [dict(f) for f in doc['frames']]}  # apply_grid replaces tags/alpha, never edits them


def scan(unit_id):
    doc = working(unit_id)
    u, before, editing = doc['unit'], _start[unit_id], {}
    for path in dots.grid_files(unit_id):
        try:
            dots.apply_grid(doc, path)
        except (SystemExit, ValueError, OSError):
            editing[path.parent.name] = editing.get(path.parent.name, 0) + 1
    folder = dots.HEROES / unit_id
    anims = {}
    for anim, a in u['animations'].items():
        frames = [[dots.frame(doc, anim, d, n) for n in range(a['frames'])] for d in u['rows']]
        changed = sum(f['tags'] != before.get((anim, f['direction'], f['frame'])) for row in frames for f in row)
        name = f'{unit_id}-{anim}.png'
        w, h = u['cell']
        strip = Image.new('RGBA', (w * a['frames'], h * len(u['rows'])))
        for r, row in enumerate(frames):
            for c, f in enumerate(row):
                strip.paste(image(doc, f), (c * w, r * h))
        write_atomic(LIVE / name, lambda p: strip.save(p, format='PNG'))
        anims[anim] = dict(frames=a['frames'], ms=a['ms'], loop=a['loop'], changed=changed,
                           total=len(u['rows']) * a['frames'], editing=editing.get(anim, 0),
                           done=any((folder / n).exists() for n in done_files(unit_id, anim)), image=name)
    return dict(name=u['name'], cell=u['cell'], pivot=u['pivot'], rows=u['rows'], anims=anims,
                reviews=sorted(p.name for p in folder.glob('review-*.md')))


def clock(sec):
    sec = int(sec)
    return f'{sec // 3600}:{sec // 60 % 60:02d}:{sec % 60:02d}'


def main():
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument('ids', nargs='+')
    p.add_argument('--every', type=int, default=30, help='seconds between rounds')
    p.add_argument('--once', action='store_true')
    p.add_argument('--out', default='live', help='folder under heroes/ (progress.html?live=<out>); one per session')
    a = p.parse_args()
    global LIVE
    LIVE = dots.HEROES / a.out
    LIVE.mkdir(exist_ok=True)
    st = (dots.HEROES / a.ids[0].removesuffix('_noncombat') / 'grid').stat()
    started = getattr(st, 'st_birthtime', st.st_ctime)
    units, first = {}, None
    while True:
        now = time.time()
        for uid in a.ids:
            try:
                units[uid] = scan(uid)
            except Exception as e:  # yaml mid-import and the like: keep the previous round
                print(f'{uid}: 건너뜀 ({e})')
        anims = [x for u in units.values() for x in u['anims'].values()]
        changed, total = sum(x['changed'] for x in anims), sum(x['total'] for x in anims)
        first = first or (now, changed)
        eta, rate = None, (changed - first[1]) / (now - first[0]) if now - first[0] >= 60 else 0
        if rate > 0:
            eta = round((total - changed) / rate * (1 + REVIEW))
        status = dict(updated=now, started=started, changed=changed, total=total, eta=eta, units=units)
        write_atomic(LIVE / 'status.json', lambda p: p.write_text(json.dumps(status, ensure_ascii=False)))
        parts = [f"{u['name']}{'(비전투)' if uid.endswith('_noncombat') else ''} " +
                 ' '.join(f"{NAMES.get(n, n)} {x['changed']}/{x['total']}{'✓' if x['done'] else ''}"
                          f"{'~' + str(x['editing']) if x['editing'] else ''}" for n, x in u['anims'].items())
                 for uid, u in units.items()]
        print(f"[{datetime.fromtimestamp(now):%H:%M:%S}] 경과 {clock(now - started)} · {changed}/{total} · "
              f"남은 {'약 ' + clock(eta) if eta else '계산 중'} | " + ' | '.join(parts), flush=True)
        if a.once:
            return
        time.sleep(max(1, a.every - (time.time() - now)))


if __name__ == '__main__':
    main()
