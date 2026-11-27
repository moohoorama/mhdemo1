#!/usr/bin/env python3
"""Writes S_title_gyeqiao, the title reel's field for 계교: the open plain (S_title_long_field) with plain high
ground beyond one straight line running from the west of the screen to the north (every cell with u below EDGE),
so a rider fleeing north-west finds a cliff and nowhere to go.

    python3 tools/title_reel/cliff_map.py
"""
import json
import subprocess
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from extend_map import MAPS, ROOT, register  # noqa: E402

EDGE = -20  # game cells: the cliff's foot runs along u = EDGE
HIGH = '2'  # a tall cliff (scene maps' height 2)


def main():
    art = json.loads((MAPS / 'S_title_long_field.json').read_text())
    art['id'] = 'S_title_gyeqiao'
    ox = art['origin'][0]
    cut = 2 * EDGE - ox  # small-tile column where the high ground ends
    art['height'] = [HIGH * cut + row[cut:] for row in art['height']]
    (MAPS / 'S_title_gyeqiao.json').write_text(json.dumps(art, ensure_ascii=False, separators=(',', ':')) + '\n')
    register(['S_title_gyeqiao'])
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', 'S_title_gyeqiao'], cwd=ROOT, check=True)
    print('S_title_gyeqiao')


if __name__ == '__main__':
    main()
