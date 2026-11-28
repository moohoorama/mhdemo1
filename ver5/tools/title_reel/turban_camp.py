#!/usr/bin/env python3
"""Writes S_title_turban_camp, the title reel's 황건 camp where 장각 cries out to his host: the open plain
(S_title_long_field, made by extend_map.py) with an altar and two censers behind the brothers' place, yellow banners and
braziers to either side, and behind them a palisade with a gate, two watchtowers and the tents. They are the map's own
objects, so the dusk darkens them like the ground and the braziers light it.

    python3 tools/title_reel/extend_map.py S_long_field --pad 46
    python3 tools/title_reel/turban_camp.py

Cells here are game cells (half cells allowed); the map lists objects on small tiles, two to a cell.
"""
import json
import subprocess
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from extend_map import MAPS, ROOT, register  # noqa: E402

SRC, DST = 'S_title_long_field', 'S_title_turban_camp'
FENCE_U = -3                 # the palisade runs along this column behind the brothers
GATE = (8.5, 11.5)           # its opening, behind 장각 (0, 10)

PIECES = [('altar', -2, 10), ('incense', -1, 9), ('incense', -1, 11), ('brazier', 1, 6), ('brazier', 1, 14),
          ('banner_yellow', -2, 6.5), ('banner_yellow', -2, 13.5), ('banner_yellow', -4, 3), ('banner_yellow', -4, 17),
          ('banner_yellow', -5, 8), ('banner_yellow', -5, 12), ('watchtower', FENCE_U, 1), ('watchtower', FENCE_U, 19),
          ('tent', -6, 5), ('tent', -6, 15), ('tent_round', -7, 10), ('antler_barrier', -2, 4), ('antler_barrier', -2, 16)]


def objects():
    out = [[n, round(2 * u), round(2 * v)] for n, u, v in PIECES]
    out += [['sfence_10', 2 * FENCE_U, sv] for sv in range(4, 37) if not 2 * GATE[0] <= sv <= 2 * GATE[1]]
    return out


def main():
    art = json.loads((MAPS / f'{SRC}.json').read_text())
    art['id'] = DST
    art['objects'] = art['objects'] + objects()
    (MAPS / f'{DST}.json').write_text(json.dumps(art, ensure_ascii=False, separators=(',', ':')) + '\n')
    register([DST])
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', DST], cwd=ROOT, check=True)
    print(DST)


if __name__ == '__main__':
    main()
