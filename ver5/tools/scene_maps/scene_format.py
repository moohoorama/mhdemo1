#!/usr/bin/env python3
"""Lay out assets/scenes/*.json the house way: one line per cast member and per beat group, so a diff shows
which beat changed.

  python3 ver5/tools/scene_maps/scene_format.py            # rewrite every chapter file
  from scene_format import write; write(path, scenes)       # from a staging script
"""
import json
import sys
from pathlib import Path

SCENES = Path(__file__).resolve().parents[2] / 'assets/scenes'

D = lambda x: json.dumps(x, ensure_ascii=False)


def write(path, scenes):
    out = ['{', ' "Scenes": {']
    for si, (sid, sc) in enumerate(scenes.items()):
        out.append(f'  {D(sid)}: {{')
        keys = [k for k in ('Map', 'Time', 'Backdrop', 'Glow', 'Music', 'Grass', 'Steps', 'Nodes', 'Ambient') if k in sc]
        unknown = set(sc) - set(keys) - {'Cast', 'Beats'}
        if unknown:
            raise SystemExit(f'{sid}: scene_format does not know {sorted(unknown)}')
        for k in keys:
            out.append(f'   {D(k)}: {D(sc[k])},')
        out.append('   "Cast": {')
        cast = list(sc['Cast'].items())
        for i, (k, a) in enumerate(cast):
            out.append(f'    {D(k)}: {D(a)}' + (',' if i < len(cast) - 1 else ''))
        out.append('   },')
        out.append('   "Beats": {')
        beats = [(n, sc['Beats'][n]) for n in sc['Nodes'] if n in sc.get('Beats', {})]
        for i, (n, lines) in enumerate(beats):
            out.append(f'    {D(n)}: [')
            for li, line in enumerate(lines):
                out.append('     [')
                for gi, g in enumerate(line):
                    out.append(f'      {D(g)}' + (',' if gi < len(line) - 1 else ''))
                out.append('     ]' + (',' if li < len(lines) - 1 else ''))
            out.append('    ]' + (',' if i < len(beats) - 1 else ''))
        out.append('   }')
        out.append('  }' + (',' if si < len(scenes) - 1 else ''))
    out += [' }', '}', '']
    open(path, 'w').write('\n'.join(out))


if __name__ == '__main__':
    for p in sorted(SCENES.glob('*.json')) if len(sys.argv) < 2 else map(Path, sys.argv[1:]):
        write(p, json.load(open(p))['Scenes'])
