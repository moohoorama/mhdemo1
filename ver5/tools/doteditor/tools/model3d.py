#!/usr/bin/env python3
"""Export the 3D scene models of civ3d.py for model3d.html (a three.js viewer; the heads are the per-direction 2D heads).

  python3 model3d.py caocao sunjian ...   -> heroes/model3d/<key>.json  (open model3d.html?unit=<key> on `make serve`)
"""
import base64
import copy
import io
import json
import sys

import civ3d  # noqa: F401  (puts tools/unit3d on the path and builds LOOKS)
import build as B
import civilian as C
import heads as Hd
import render as R
import units as U

OUT = civ3d.DE / 'heroes/model3d'


def export(key):
    build, head = civ3d.LOOKS[key]
    pose = U.pose(0, 0)
    pose.update(copy.deepcopy(C.REST))
    faces, center = U.soldier(pose, build.kit, weapon=False)
    face = build(0, 0)['face']
    pos, col = [], []
    for quad, color in faces:
        quad = [list(map(float, v)) for v in quad]
        rgb = [int(color[i:i + 2], 16)/255 for i in (1, 3, 5)]
        for tri in ([0, 1, 2], [0, 2, 3]):
            for i in tri:
                pos += quad[i]
                col += rgb
    heads = {}
    for d in R.AZIMUTH:
        im = Hd.to_img(Hd.expression(Hd.head_for(head['grids'], d), face, head['skin'], head['lid']))
        buf = io.BytesIO()
        im.save(buf, 'PNG')
        heads[d] = dict(png=base64.b64encode(buf.getvalue()).decode(), w=im.width, h=im.height,
                        t=[float(x) for x in R.view(R.AZIMUTH[d])[0]])
    OUT.mkdir(parents=True, exist_ok=True)
    (OUT / f'{key}.json').write_text(json.dumps(dict(
        pos=[round(x, 4) for x in pos], col=[round(x, 3) for x in col], head=[float(x) for x in center],
        heads=heads, anchor=list(head.get('anchor', Hd.ANCHOR)), scale=B.SPEC['scale'])))
    print(OUT / f'{key}.json')


if __name__ == '__main__':
    for key in sys.argv[1:] or sys.exit('keys: ' + ' '.join(civ3d.LOOKS)):
        export(key)
