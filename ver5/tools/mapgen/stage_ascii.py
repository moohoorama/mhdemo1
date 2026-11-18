#!/usr/bin/env python3
"""A battle stage's placement drawn and edited as ASCII, one cell to three characters.

    python3 tools/mapgen/stage_ascii.py dump B01 [B02 ...]     # write tools/mapgen/layouts/hero/stageNN.txt
    python3 tools/mapgen/stage_ascii.py load B01 [B02 ...]     # read edited pictures back into the stage file

The picture covers the map's play area, rows along Y and columns along X, (0, 0) at the top left:

    # B01  stage01.json  map01.json  origin 0 0
    TRE...............,,,,,,........TRE
    ......tu1...Den...tu2.............
    ...............Zha...Gua..........
    ##########################[ ]#####

Terrain: `...` grass, `,,,` wasteland, `~~~` water, `###` wall, `___` castle floor, `^^^` cliff, `[ ]` gate wall,
and the structures and states as srpg-cli draws them (`TRE` forest, `VIL` village, `CMP` camp, `CST` castle,
`PAS` pass, `BRG` bridge, `FNC` fence, `FIR` flames, `EMB` embers). Units: named officers in title case
(`Cao`, `Liu`), template troops and extras in lower case with a number (`tu1` = troop_turban), the player's
deployment slots `pl1`, `pl2` … `p10` … in the order of `Allies`, torches `Trc`. The legend under the picture says which
stage entry each code is (`Enemies[3]`), its side, group and policy; areas are listed with their rectangles.
`load` sets each listed entry's X and Y to the cell its code stands on. Torches, areas and the terrain are not
moved by `load` (edit the map in tools/mapgen/layouts or the map editor).
"""
import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
HERO = ROOT / 'assets/scenario/hero'
OUT = ROOT / 'tools/mapgen/layouts/hero'
sys.path.insert(0, str(ROOT / 'tools/scene_maps'))
from scene_ascii import OFFICER  # noqa: E402  the same officer codes as the scene pictures

GROUND = {'g': '...', 'd': ',,,', 'l': "'''", 's': ';;;', 'w': '~~~', 'c': '###', 'i': '___', 'h': '^^^', 'm': '[ ]', 'u': ' n ', 'U': ' N ', 'n': 'MMM'}
STRUCTURE = {'t': 'TRE', 'v': 'VIL', 'k': 'CMP', 'o': 'CST', 'p': 'PAS', 'b': 'BRG', 'z': 'FNC'}
STATE = {'e': 'EMB', 'F': 'FIR', 'x': 'xxx', 'O': '[O]'}
SIDES = (('Allies', 'player'), ('Party', 'ally'), ('Enemies', 'enemy'))


def terrain(code):
    ground, structure, state = (code + '---')[:3]
    if state in STATE:
        return STATE[state]
    if structure in STRUCTURE:
        g = STRUCTURE[structure]
        return g.lower() if state in 'sd' else g
    return GROUND.get(ground, f' {ground} ')


def stage_paths(sid):
    n = re.fullmatch(r'B(\d+)', sid)
    if not n:
        raise SystemExit(f'not a stage id: {sid}')
    path = HERO / f'stage{int(n[1]):02d}.json'
    return path, OUT / f'{path.stem}.txt'


def codes(stage):
    """(list, index) of every placed unit → a unique three-letter code."""
    out, used = {}, set()
    for key, _ in SIDES:
        for i, u in enumerate(stage.get(key, [])):
            cid = u.get('Character', '')
            if key == 'Allies':
                want = [f'pl{n}' for n in range(1, 10)] + [f'p{n}' for n in range(10, 100)]
            elif cid and not cid.startswith('troop_') and cid != 'soldier':
                want = [OFFICER.get(cid, cid[:3].title()), cid[0].upper() + cid[-2:]] + [cid[:2].title() + str(n) for n in range(1, 10)]
            else:
                stem = cid.removeprefix('troop_')[:2] or 'xx'
                want = [f'{stem}{n}' for n in range(1, 10)] + [f'{stem[0]}{n:02d}' for n in range(10, 100)]
            code = next(c for c in want if c not in used)
            used.add(code)
            out[(key, i)] = code
    return out


def picture(sid, path, stage):
    doc = json.loads((HERO / 'maps' / stage['Map']).read_text())
    play = doc['Play']
    code = codes(stage)
    at = {(u['X'], u['Y']): code[(key, i)] for key, _ in SIDES for i, u in enumerate(stage.get(key, []))}
    lights = {(p['X'], p['Y']) for p in stage.get('Lights', [])}
    lines = [f'# {sid}  {path.name}  {stage["Map"]}  origin {play["X"]} {play["Y"]}']
    for y in range(play['Y'], play['Y'] + play['H']):
        row = ''
        for x in range(play['X'], play['X'] + play['W']):
            if (x, y) in at:
                row += at[(x, y)]
            elif (x, y) in lights:
                row += 'Trc'
            else:
                row += terrain(doc['Cells'].get(f'{x},{y}', doc['Default']))
        lines.append(row)
    lines.append('')
    for key, side in SIDES:
        for i, u in enumerate(stage.get(key, [])):
            notes = [side]
            if u.get('Character'):
                notes.insert(0, u['Character'])
            for k in ('Group', 'Policy', 'Rank', 'Faction'):
                if u.get(k):
                    notes.append(f'{k.lower()} {u[k]}')
            lines.append(f'{code[(key, i)]} {key}[{i}]  ' + ', '.join(notes))
    if stage.get('Areas'):
        lines.append('')
        for a in stage['Areas']:
            r = a['Rect']
            lines.append(f'# area {a["ID"]} {a.get("Name", "")}: x {r["X"]}..{r["X"] + r["W"] - 1}, y {r["Y"]}..{r["Y"] + r["H"] - 1}')
    return '\n'.join(lines) + '\n'


def cmd_dump(ids):
    OUT.mkdir(parents=True, exist_ok=True)
    for sid in ids:
        path, txt = stage_paths(sid)
        txt.write_text(picture(sid, path, json.loads(path.read_text())))
        print(txt.relative_to(ROOT))


def cmd_load(ids):
    for sid in ids:
        path, txt = stage_paths(sid)
        text = path.read_text()
        stage = json.loads(text)
        pic = txt.read_text().split('\n')
        head = re.search(r'origin (-?\d+) (-?\d+)', pic[0])
        x0, y0 = int(head[1]), int(head[2])
        spots = {}
        for r, row in enumerate(pic[1:pic.index('')]):
            for c in range(0, len(row), 3):
                spots.setdefault(row[c:c + 3], []).append((x0 + c // 3, y0 + r))
        changed = False
        for (key, i), c in codes(stage).items():
            found = spots.get(c, [])
            if len(found) != 1:
                raise SystemExit(f'{sid}: {c} ({key}[{i}]) is in the picture {len(found)} times')
            u = stage[key][i]
            if (u['X'], u['Y']) != found[0]:
                print(f'{sid}: {c} {key}[{i}] ({u["X"]}, {u["Y"]}) -> {found[0]}')
                u['X'], u['Y'] = found[0]
                changed = True
        if changed:
            if json.dumps(json.loads(text), ensure_ascii=False, indent=1) + '\n' != text:
                raise SystemExit(f'{path.name} is not in the plain indented layout; not rewriting it')
            path.write_text(json.dumps(stage, ensure_ascii=False, indent=1) + '\n')


if __name__ == '__main__':
    if len(sys.argv) < 3 or sys.argv[1] not in ('dump', 'load'):
        raise SystemExit(__doc__)
    {'dump': cmd_dump, 'load': cmd_load}[sys.argv[1]](sys.argv[2:])
