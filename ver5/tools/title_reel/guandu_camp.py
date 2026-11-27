#!/usr/bin/env python3
"""Lays the title reel's camp mouth on S_title_guandu_field (made by extend_map.py from S_guandu_field): a row of
antler barriers out from either end of the pike files, a palisade running back into the camp from each end and a
watchtower in each corner, all near the rim of the shot's circle and clear of where the riders meet the pikes, and two
braziers before the files. They are the map's own objects so the night darkens them and the braziers light it.

    python3 tools/title_reel/extend_map.py S_guandu_field --pad 20
    python3 tools/title_reel/guandu_camp.py

Cells here are game cells; the map lists objects on small tiles, two to a cell.
"""
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
MAP = ROOT / 'assets/graphics/maps/S_title_guandu_field.json'

F = 24                        # the front rank (build.py guandu)
UL, UR = -2, 40               # the palisade's two arms
BACK = 39                     # how far back into the camp they run
BRAZIERS = [(12, F - 1), (24, F - 1)]  # before the files, one toward each end


def objects():
    out = [('antler_barrier', 2 * u, 2 * F) for u in list(range(UL, 3)) + list(range(36, UR + 1))]
    out += [('sfence_10', 2 * u, sv) for u in (UL, UR) for sv in range(2 * F + 1, 2 * BACK + 2)]
    out += [('watchtower', 2 * (UL + 2), 2 * (F + 3)), ('watchtower', 2 * (UR - 2), 2 * (F + 3))]
    out += [('brazier', 2 * u, 2 * v) for u, v in BRAZIERS]
    return [list(o) for o in out]


def main():
    art = json.loads(MAP.read_text())
    mine = objects()
    art['objects'] = [o for o in art['objects'] if o not in mine] + mine
    MAP.write_text(json.dumps(art, ensure_ascii=False, separators=(',', ':')) + '\n')
    print(MAP.relative_to(ROOT), len(mine), 'objects')


if __name__ == '__main__':
    main()
