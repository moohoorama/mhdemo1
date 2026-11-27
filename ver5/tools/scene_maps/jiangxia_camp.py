#!/usr/bin/env python3
"""Generate S_jiangxia_camp: the Jiangxia camp, the main army on the north bank of a river and Guan Yu's
detachment on the south bank. The river is 36 small tiles wide, so a hero standing at one pier sees only water
and never the far pier: he presses the ferry moored by the pier and stands at the other. The two piers sit at the
same u, each with a ferry (boat_l_v) beside it for the exit, and nothing in between can be walked. The north bank
is S_luzhi_camp's three compounds; the south bank is the detachment's bare camp, with no fields (the army is short
of food)."""
import json
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
W, H = 64, 80
RIVER = (32, 67)
PIER = (range(32, 35), range(65, 68))
GAP = range(11, 17)


def fence(objects, left, top, right, bottom, gap_north=None, gap_south=None):
    """A palisade along the compound's edge; a gap is a range of u whose ends are posts."""
    def side(v, gap):
        for u in range(left + 1, right):
            if gap and u in range(gap.start - 1, gap.stop + 1):
                continue
            objects.append(['sfence_05', u, v])
        if gap:
            objects.append(['sfence_01', gap.start - 1, v])
            objects.append(['sfence_04', gap.stop, v])
    side(top, gap_north)
    side(bottom, gap_south)
    for v in range(top + 1, bottom):
        objects.append(['sfence_10', left, v])
        objects.append(['sfence_10', right, v])
    objects += [['sfence_12', left, top], ['sfence_09', right, top], ['sfence_06', left, bottom], ['sfence_03', right, bottom]]


def build():
    old = json.loads((GFX / 'maps/S_luzhi_camp.json').read_text())
    ground = [list(r) for r in old['ground'][:32]]
    for v in (0, 1, 2, 5, 6):
        ground[v] = ['.'] * W
    ground += [['.'] * W for _ in range(H - 32)]

    def paint(u0, v0, u1, v1, tile):
        for v in range(v0, v1 + 1):
            for u in range(u0, u1 + 1):
                ground[v][u] = tile

    paint(0, RIVER[0], W - 1, RIVER[1], '~')
    for rows in PIER:
        paint(12, rows.start, 15, rows[-1], 'b')
    paint(3, 69, 26, 78, 'd')

    objects = []
    for ob in old['objects']:
        name, u, v = ob[0], ob[1], ob[2]
        if name == 'sgate_closed' or name.startswith('stree') and v >= 31:
            continue
        if name == 'sfence_05' and v == 30 and u in range(GAP.start - 1, GAP.stop + 1):
            continue
        if name == 'tent' and u == 13 and v >= 22:
            continue
        objects.append(ob)
    objects += [['sfence_01', GAP.start - 1, 30], ['sfence_04', GAP.stop, 30]]
    objects += [['stree_pine_0', 6, 4], ['stree_pine_1', 22, 3], ['stree_old', 40, 4], ['stree_pine_0', 57, 5],
                ['stree_bush_0', 24, 31], ['stree_bush_1', 38, 31], ['stree_bush_0', 63, 31]]
    objects += [['sbridge_v', u, v] for u in range(12, 16) for rows in PIER for v in rows]
    objects += [['boat_v', 10, 34], ['boat_l_v', 17, 34], ['boat_v', 10, 65], ['boat_l_v', 17, 65]]

    # the detachment's camp on the south bank
    fence(objects, 2, 68, 27, 79, gap_north=GAP)
    objects += [['watchtower', 4, 70], ['watchtower', 25, 70],
                ['brazier', 13, 70], ['command_flag', 9, 71], ['command_flag', 17, 71],
                ['tent', 7, 74], ['tent', 12, 76], ['tent', 18, 74], ['tent', 22, 77], ['tent', 6, 77],
                ['cauldron', 22, 72], ['grain_sack', 24, 74]]
    return dict(id='S_jiangxia_camp', name='강하 진지', format=2, size=[W // 2, H // 2], origin=[0, 0],
                grass=old['grass'], ground=[''.join(r) for r in ground], objects=objects,
                walls=[])


if __name__ == '__main__':
    relative = 'maps/S_jiangxia_camp.json'
    (GFX / relative).write_text(json.dumps(build(), ensure_ascii=False, indent=1) + '\n')
    path = GFX / 'index.json'
    index = json.loads(path.read_text())
    if relative not in index['maps']:
        index['maps'].append(relative)
        path.write_text(json.dumps(index, ensure_ascii=False, indent=1) + '\n')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', 'S_jiangxia_camp'], cwd=GFX.parents[1], check=True)
