#!/usr/bin/env python3
"""Make S_castle_out_siege, the field in front of 하비's open gate for chapter 25's memory of the siege (S25 d10~d13):
S_castle_out_open with the trees taken out of the ground where 조조's squares and the cavalry that has just ridden
out of the gate stand (scene cells u -13..19, v 3..17), so a few hundred people fit on the grass in whole cells.

  python3 ver5/tools/scene_maps/castle_siege.py

Format 2 (internal/scenemap). Everything else — the wall, the moat, the bridge, the open gate, the banners and the
trees further out — is S_castle_out_open's. The scenes that use that map are not touched.
"""
import json
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
SRC, ID = 'S_castle_out_open', 'S_castle_out_siege'
CLEAR = (-13, 19, 3, 17)     # scene cells u0, u1, v0, v1

art = json.load(open(GFX / f'maps/{SRC}.json'))
art['id'], art['name'] = ID, '성 밖(포위)'
kept = []
for o in art['objects']:
    name, U, V = o[0], o[1], o[2]
    if name.startswith('stree') and CLEAR[0] <= U // 2 <= CLEAR[1] and CLEAR[2] <= V // 2 <= CLEAR[3]:
        continue
    kept.append(o)
art['objects'] = kept

if __name__ == '__main__':
    json.dump(art, open(GFX / f'maps/{ID}.json', 'w'), ensure_ascii=False, indent=1)
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', ID], cwd=GFX.parents[1], check=True)
    print(ID, art['size'], len(kept), 'objects')
