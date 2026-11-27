#!/usr/bin/env python3
"""Writes S_title_chibi_shore, the title reel's burning camp on the Red Cliffs shore: S_title_jiangxia_harbor (made by
extend_map.py) turned half round, so that its river and landing stages lie south-west of the camp, where 손권's boats
come in from, and the open shore runs north-west, where 조조's men run.

    python3 tools/title_reel/extend_map.py S_jiangxia_harbor --pad 66
    python3 tools/title_reel/turn_map.py

Ground characters are the same either way round; an object keeps its picture and moves to the turned small tile.
"""
import json
import subprocess
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from extend_map import MAPS, ROOT, register  # noqa: E402

SRC, DST = 'S_title_jiangxia_harbor', 'S_title_chibi_shore'


def main():
    art = json.loads((MAPS / f'{SRC}.json').read_text())
    w, h = len(art['ground'][0]), len(art['ground'])
    ox, oy = art['origin']
    art['id'] = DST
    art['ground'] = [row[::-1] for row in art['ground'][::-1]]
    if 'height' in art:
        art['height'] = [row[::-1] for row in art['height'][::-1]]
    # small tile x of [ox, ox+w) goes to -x-1: the turned map spans [-ox-w, -ox)
    art['origin'] = [-ox - w, -oy - h]
    art['objects'] = [[o[0], -o[1] - 1, -o[2] - 1, *o[3:]] for o in art['objects']]
    (MAPS / f'{DST}.json').write_text(json.dumps(art, ensure_ascii=False, separators=(',', ':')) + '\n')
    register([DST])
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', DST], cwd=ROOT, check=True)
    print(DST)


if __name__ == '__main__':
    main()
