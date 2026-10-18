"""성문 (a roofed gate pavilion), drawn from scratch as a full tile. Four tall stone pillars carry a hip roof high enough
for a horseman to ride under; a door hangs between the pillars at each end of the passage.
  gate_*_u : passage along v - doors on the SW (front) and NE (back) edges;  gate_*_v : along u - SE (front), NW (back)
Layers, so units sort inside the tile: floor (ground), back (object, sort y - 3), front (object, sort y + 3), roof (object, sort y + 5)
open  : both leaves swung inward against the side walls       closed : both leaves shut, iron-banded"""
import json
from common import *
from PIL import Image, ImageDraw

OL = (11, 16, 27, 255)
ST = [(196, 210, 217, 255), (147, 163, 175, 255), (99, 114, 128, 255), (63, 74, 86, 255)]
WOOD = [(150, 98, 56, 255), (120, 76, 44, 255), (88, 54, 32, 255)]
IRON = (176, 182, 194, 255)
EAVE = (102, 55, 32, 255)
W, H = 48, 64
SX, SY = 4, 16                # the 40x48 wall cell sits at (4,16) inside the taller cell
PIV = (21 + SX, 39 + SY)
POST = 32                     # pillar height: a rider (~30px) passes under the beam
LEAF = 24                     # door leaf height
EDGES = {"SW": ((5, 39), (21, 47)), "NE": ((21, 31), (37, 39)), "SE": ((21, 47), (37, 39)), "NW": ((5, 39), (21, 31))}
T_FRONT = {"u": (2, -1), "v": (-2, -1)}
T_SIDE = {"u": (-2, -1), "v": (2, -1)}
SIDE_STEP = {"SW": (8, -4), "NE": (-8, 4), "SE": (-8, -4), "NW": (8, 4)}


def P(x, y):
    return (x + SX, y + SY)


def poly(d, pts, fill):
    d.polygon([P(x, y) for x, y in pts], fill=fill, outline=OL)


def post(d, x, y):
    x, y = P(x, y)
    d.rectangle([x - 1, y - POST, x + 1, y], fill=ST[1], outline=OL)
    d.line([(x + 1, y - POST + 1), (x + 1, y - 1)], fill=ST[2])
    d.line([(x - 1, y - POST), (x + 1, y - POST)], fill=ST[0])


def beam(d, a, b):
    """the wooden beam along a tile edge at the top of the pillars"""
    (ax, ay), (bx, by) = a, b
    poly(d, [(ax, ay - POST), (bx, by - POST), (bx, by - POST + 3), (ax, ay - POST + 3)], EAVE)


def leaf(d, im, a, b, t):
    (ax, ay), (bx, by) = a, b
    up = -LEAF
    tx, ty = t
    poly(d, [(ax, ay + up), (bx, by + up), (bx + tx, by + up + ty), (ax + tx, ay + up + ty)], WOOD[0])
    e = (bx, by) if tx * (bx - ax) >= 0 else (ax, ay)
    poly(d, [e, (e[0] + tx, e[1] + ty), (e[0] + tx, e[1] + ty + up), (e[0], e[1] + up)], WOOD[2])
    poly(d, [(ax, ay), (bx, by), (bx, by + up), (ax, ay + up)], WOOD[1])
    n = max(abs(bx - ax), 1)
    for i in range(1, n):
        x = round(ax + (bx - ax) * i / n)
        y = round(ay + (by - ay) * i / n)
        c = WOOD[0] if (i // 2) % 2 == 0 else WOOD[1]
        for k in range(1, LEAF):
            col = (IRON if i % 3 else WOOD[2]) if k in (4, 5, 12, 13, 20, 21) else c
            px, py = P(x, y - k)
            if 0 <= px < W and 0 <= py < H:
                im.putpixel((px, py), col)


def door(d, im, edge, state, orient):
    a, b = EDGES[edge]
    (ax, ay), (bx, by) = a, b
    mx, my = (ax + bx) // 2, (ay + by) // 2
    if state == "closed":
        leaf(d, im, (ax, ay), (mx, my), T_FRONT[orient])
        leaf(d, im, (mx, my), (bx, by), T_FRONT[orient])
        px, py = P(mx, my)
        d.line([(px, py - LEAF), (px, py)], fill=OL)
    else:
        sx, sy = SIDE_STEP[edge]
        for ex, ey in (a, b):
            leaf(d, im, (ex, ey), (ex + sx, ey + sy), T_SIDE[orient])
    post(d, ax, ay)
    post(d, bx, by)
    beam(d, a, b)


def roof(d):
    cx, cy = 21, 39 - POST + 2          # centre of the roof base, on top of the pillars
    d.polygon([P(cx - 19, cy), P(cx, cy + 9), P(cx + 19, cy), P(cx + 19, cy - 3), P(cx, cy + 6), P(cx - 19, cy - 3)], fill=EAVE, outline=OL)
    peak = P(cx, cy - 17)
    d.polygon([P(cx - 19, cy - 3), P(cx, cy + 6), peak], fill=ST[2], outline=OL)
    d.polygon([P(cx, cy + 6), P(cx + 19, cy - 3), peak], fill=ST[3], outline=OL)
    d.line([P(cx, cy + 6), peak], fill=OL)
    d.line([P(cx - 15, cy - 3), P(cx - 3, cy + 3)], fill=ST[1])


def render(orient, state, part):
    im = Image.new("RGBA", (W, H))
    d = ImageDraw.Draw(im)
    front, back = ("SW", "NE") if orient == "u" else ("SE", "NW")
    if part == "floor":
        poly(d, [(5, 39), (21, 31), (37, 39), (21, 47)], ST[1])
        px = im.load()
        for y in range(31, 48):
            for x in range(5, 38):
                if px[x + SX, y + SY] == ST[1] and ((x + 2 * y) % 8 == 0 or (x - 2 * y) % 8 == 3):
                    px[x + SX, y + SY] = ST[2]
    elif part == "roof":
        roof(d)
    else:
        door(d, im, front if part == "front" else back, state, orient)
        # the beam along the side that belongs to this layer
        side = ("SE" if orient == "u" else "SW") if part == "front" else ("NW" if orient == "u" else "NE")
        a, b = EDGES[side]
        beam(d, a, b)
    return im


def run():
    import gen_all
    for o in ("u", "v"):
        gen_all.save("tiles", "gate_floor_" + o, render(o, "open", "floor"), PIV, "ground", scale="battle")
        gen_all.save("tiles", "gate_roof_" + o, render(o, "open", "roof"), PIV, "object", scale="battle", sort_dy=5,
                     note="draw after units standing in the gate tile")
        for state in ("open", "closed"):
            for part, dy in (("back", -3), ("front", 3)):
                gen_all.save("tiles", "gate_%s_%s_%s" % (state, o, part), render(o, state, part), PIV, "object", scale="battle",
                             sort_dy=dy, blocks=(state == "closed"), note="one of the two door frames; draw with sort y offset sort_dy")
    json.dump(gen_all.manifest, open(out("tiles", "tiles.json"), "w"), indent=1)
