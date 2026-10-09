#!/usr/bin/env python3
"""Text-grid pen for hero dots: frames go out as one character per pixel and come back into units/<id>.yaml.

  grid.py new cavalry guanyu 관우                      # units/guanyu.yaml from the base (source_base = cavalry)
  grid.py new infantry liubei_noncombat 유비 --noncombat --steps 1,3
                                                       # NW·SW only: idle 1, walk 2 (base walk frames 1 and 3), action 1
  grid.py export liubei [--anim idle] [--dir SW] [--frame 0] [--force]
                                                       # -> heroes/liubei/grid/<anim>/<dir>-<n>.txt (keeps existing files)
  grid.py import liubei                                # every grid file back into units/liubei.yaml
  grid.py colors guanyu themeA1=#5fae5a themeA2=#3d8a45 themeA3=#24573a

Each grid file starts with ';' comment lines (frame, cell, pivot, legend, column ruler); pixel lines are 'NN|<row>'.
"""
import argparse
import copy
import re
import sys

import dots

FACTION_TAGS = {'faction1', 'faction2', 'faction3'}  # game team keys: recolored per faction at runtime


def legend(doc):
    """(char -> (tag, alpha)) and printable lines for a unit."""
    names = {t['index']: (t['id'], t['name']) for t in doc['tag_definitions']}
    table, lines = {'.': (0, '00')}, ['; . = 투명']
    for i, ch in enumerate(dots.TAG_CHARS, 1):
        table[ch] = (i, 'ff')
        lines.append(f'; {ch} = {names[i][0]} {names[i][1]} {doc["tag_colors"][i - 1][:7]}')
    pairs = sorted({(t, a) for f in doc['frames'] for trow, arow in zip(f['tags'], f['alpha'])
                    for t, a in zip(trow, arow) if t and a not in ('ff', '00')})
    for ch, (t, a) in zip(dots.TRANSLUCENT_CHARS, pairs):
        table[ch] = (t, a)
        lines.append(f'; {ch} = {names[t][0]} {names[t][1]} 반투명 alpha {a}')
    return table, lines


def cmd_new(a):
    doc = dots.load(a.base)
    u = doc['unit']
    u.update(source_base=a.base, id=a.id, name=a.name)
    doc['comparison_layers'] = []
    if a.noncombat:
        if not a.id.endswith('_noncombat'):
            raise SystemExit('noncombat id must end with _noncombat')
        left, right = (int(n) for n in a.steps.split(','))
        pick = [('idle', 'idle', 0, 0), ('walk', 'walk', left, 0), ('walk', 'walk', right, 1), ('action', 'idle', 0, 0)]
        frames = []
        for d in ('NW', 'SW'):
            for anim, src, n, k in pick:
                f = copy.deepcopy(dots.frame(doc, src, d, n))
                f.update(animation=anim, frame=k)
                frames.append(f)
        doc['frames'] = frames
        u['rows'] = ['NW', 'SW']
        u['animations'] = {'idle': dict(first_column=0, frames=1, ms=[400], loop=True),
                           'walk': dict(first_column=1, frames=2, ms=[180, 180], loop=True),
                           'action': dict(first_column=3, frames=1, ms=[800], loop=False)}
    dots.with_alpha(doc)
    print('wrote', dots.save(doc))


def cmd_export(a):
    doc = dots.with_alpha(dots.load(a.id))
    table, lines = legend(doc)
    chars = {v: k for k, v in table.items()}
    (w, h), (px, py) = doc['unit']['cell'], doc['unit']['pivot']
    out = dots.HEROES / a.id / 'grid'
    count = 0
    for f in doc['frames']:
        if (a.anim and f['animation'] != a.anim) or (a.dir and f['direction'] != a.dir) or \
                (a.frame is not None and f['frame'] != a.frame):
            continue
        path = out / f['animation'] / f"{f['direction']}-{f['frame']}.txt"
        if path.exists() and not a.force:
            continue  # the grid is the working copy; --force replaces it with the yaml frame
        path.parent.mkdir(parents=True, exist_ok=True)
        head = [f"; {a.id} · {f['animation']} · {f['direction']} · frame {f['frame']}",
                f'; cell {w}x{h} · pivot (x={px}, y={py}) = 발밑 접지점', *lines,
                ';    ' + ''.join(str(x // 10) for x in range(w)),
                ';    ' + ''.join(str(x % 10) for x in range(w))]
        rows = [f'{y:02d}|' + ''.join(chars[(t, al if t else '00')] for t, al in zip(trow, arow))
                for y, (trow, arow) in enumerate(zip(f['tags'], f['alpha']))]
        path.write_text('\n'.join(head + rows) + '\n')
        count += 1
    print(f'exported {count} frames to {out} (existing files kept unless --force)')


def cmd_import(a):
    doc = dots.with_alpha(dots.load(a.id))
    count = dots.apply_grids(doc)
    if not count:
        raise SystemExit(f'no grid files under {dots.HEROES / a.id / "grid"}')
    print(f'imported {count} frames ->', dots.save(doc))


def cmd_colors(a):
    doc = dots.with_alpha(dots.load(a.id))
    ids = dots.tag_ids(doc)
    for pair in a.pairs:
        tag, color = pair.split('=')
        if tag in FACTION_TAGS:
            raise SystemExit('faction colors are the game team keys; keep them at the defaults')
        if not re.fullmatch(r'#[0-9a-fA-F]{6}', color):
            raise SystemExit(f'bad color {color}')
        doc['tag_colors'][ids.index(tag)] = color.lower() + 'ff'
    print('wrote', dots.save(doc))


def main():
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = p.add_subparsers(dest='cmd', required=True)
    n = sub.add_parser('new')
    n.add_argument('base'), n.add_argument('id'), n.add_argument('name')
    n.add_argument('--noncombat', action='store_true')
    n.add_argument('--steps', default='1,3', help='base walk frames used as left-foot and right-foot steps')
    e = sub.add_parser('export')
    e.add_argument('id'), e.add_argument('--anim'), e.add_argument('--dir'), e.add_argument('--frame', type=int)
    e.add_argument('--force', action='store_true', help='overwrite existing grid files')
    sub.add_parser('import').add_argument('id')
    c = sub.add_parser('colors')
    c.add_argument('id'), c.add_argument('pairs', nargs='+')
    a = p.parse_args()
    {'new': cmd_new, 'export': cmd_export, 'import': cmd_import, 'colors': cmd_colors}[a.cmd](a)


if __name__ == '__main__':
    sys.exit(main())
