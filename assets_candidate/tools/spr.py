"""Export a game .spr (see ver5/docs/presentation-sprite.md) to a PNG sheet + JSON, in base colours."""
import json, struct, sys, zlib
from PIL import Image
from common import *


class R:
    def __init__(s, b): s.b, s.i = b, 0
    def u(s, f):
        n = struct.calcsize(f); v = struct.unpack_from("<" + f, s.b, s.i); s.i += n; return v if len(v) > 1 else v[0]
    def str(s):
        n = s.u("H"); t = s.b[s.i:s.i + n].decode(); s.i += n; return t
    def raw(s, n): v = s.b[s.i:s.i + n]; s.i += n; return v


def export(uid, tint=None):
    path = os.path.join(os.path.dirname(GAME), "sprites", uid + ".spr")
    data = open(path, "rb").read()
    assert data[:4] == b"MHSP"
    r = R(zlib.decompress(data[6:]))
    r.str(); r.str()
    cw, ch, px_, py_ = r.u("HHHH")
    rows = [r.str() for _ in range(r.u("B"))]
    anims = []
    for _ in range(r.u("B")):
        name = r.str(); first, frames, loop, n = r.u("HHBB")
        anims.append(dict(name=name, first=first, frames=frames, loop=loop, ms=[r.u("I") for _ in range(n)]))
    tags = {}
    for _ in range(r.u("B")):
        idx = r.u("B"); grp = r.str(); shade = r.u("B"); col = r.u("BBBB"); tags[idx] = (grp, shade, col)
    pal = [tuple(r.raw(4)) for _ in range(r.u("H"))]
    cols = r.u("H")
    sheet = Image.new("RGBA", (cw * cols, ch * len(rows)))
    p = sheet.load()
    for ri in range(len(rows)):
        for ci in range(cols):
            idx = r.raw(cw * ch); tg = r.raw(cw * ch)
            for k in range(cw * ch):
                if idx[k] == 0:
                    continue
                c = pal[idx[k]]
                if tg[k] and tg[k] in tags:
                    t = tags[tg[k]][2]
                    c = (t[0], t[1], t[2], c[3])
                p[ci * cw + k % cw, ri * ch + k // cw] = c
    sheet.save(out("units", uid + ".png"))
    json.dump(dict(id=uid, cell=[cw, ch], pivot=[px_, py_], rows=rows, columns=cols, animations=anims), open(out("units", uid + ".json"), "w"), indent=1)
    return rows, [a["name"] for a in anims], (cw, ch)


if __name__ == "__main__":
    for u in sys.argv[1:]:
        print(u, export(u))
