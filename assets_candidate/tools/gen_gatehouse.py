"""성문 C: a gate house in the wall line, built ON the game's own full wall block (wall_15) so its stone, bricks and outline match the
walls beside it: the faces are kept, the crenellated top is cleared, a dome-topped arch with a shut door is cut into one face, and
a low hip roof sits on the wall top. When the gate opens it simply disappears (gatehouse_floor_* stays on the ground).
_u: the wall runs along u, the door is on the SW (left) face;  _v: along v, on the SE (right) face"""
import json, math
from common import *
from PIL import Image, ImageDraw

OL = (11, 16, 27, 255)
ST = [(196, 210, 217, 255), (147, 163, 175, 255), (99, 114, 128, 255), (63, 74, 86, 255)]
WOOD = [(150, 98, 56, 255), (120, 76, 44, 255), (88, 54, 32, 255)]
IRON = (176, 182, 194, 255)
EAVE = (102, 55, 32, 255)
W, HH = 40, 48
PIV = (21, 39)
AW, AH = 10, 13          # the arch: 10 wide, 13 high on a 17px face


def face_top(x):
    """y of the top edge of the front faces of the wall block"""
    return 23 + (x - 4) * 0.5 if x < 21 else 33 - (x - 21) * 0.5 + 0.5


def base():
    im, _ = structure("wall_15")
    im = im.copy()
    px = im.load()
    for x in range(im.width):
        for y in range(im.height):
            if px[x, y][3] and y < face_top(x) - 1:
                px[x, y] = (0, 0, 0, 0)
    return im


def arch_cells(face):
    """pixels of a dome-topped arch in face coordinates: straight sides, a semicircular head"""
    out = {}
    r0 = AW / 2
    h0 = AH - r0
    x0 = 8 if face == "L" else 24
    for i in range(AW):
        x = x0 + i
        yb = 40 + (x - 6) * 0.5 if face == "L" else 47.5 - (x - 21) * 0.5
        dx = (i + 0.5 - AW / 2) / r0
        top = int(h0 + r0 * math.sqrt(max(0.0, 1 - dx * dx)) + 0.5)
        for r in range(top):
            out[(x, round(yb) - 1 - r)] = (r, i, top)
    return out


def roof(d):
    cx, cy = 21, 24
    d.polygon([(cx - 17, cy), (cx, cy + 8), (cx + 17, cy), (cx + 17, cy - 3), (cx, cy + 5), (cx - 17, cy - 3)], fill=EAVE, outline=OL)
    peak = (cx, cy - 10)
    d.polygon([(cx - 17, cy - 3), (cx, cy + 5), peak], fill=ST[2], outline=OL)
    d.polygon([(cx, cy + 5), (cx + 17, cy - 3), peak], fill=ST[3], outline=OL)
    d.line([(cx, cy + 5), peak], fill=OL)
    d.line([(cx - 13, cy - 3), (cx - 3, cy + 2)], fill=ST[1])


def make(orient):
    im = base()
    px = im.load()
    for (x, y), (r, i, top) in arch_cells("L" if orient == "u" else "R").items():
        edge = r == top - 1 or i in (0, AW - 1)
        c = WOOD[0] if (i // 2) % 2 == 0 else WOOD[1]
        if i in (AW // 2 - 1, AW // 2):
            c = WOOD[2] if i == AW // 2 - 1 else OL
        if r in (3, 8):
            c = IRON if i % 4 == 1 else WOOD[2]
        if edge:
            c = WOOD[2]
        px[x, y] = c
    # light stone voussoirs along the head
    for (x, y), (r, i, top) in arch_cells("L" if orient == "u" else "R").items():
        if r == top - 1 and px[x, y - 1][3] and px[x, y - 1] not in (OL,):
            px[x, y - 1] = ST[0]
    roof(ImageDraw.Draw(im))
    return im


def floor():
    im = Image.new("RGBA", (W, HH))
    d = ImageDraw.Draw(im)
    d.polygon([(5, 39), (21, 31), (37, 39), (21, 47)], fill=ST[1], outline=OL)
    px = im.load()
    for y in range(31, 48):
        for x in range(6, 37):
            if px[x, y] == ST[1] and ((x + 2 * y) % 8 == 0 or (x - 2 * y) % 8 == 3):
                px[x, y] = ST[2]
    return im


def run():
    import gen_all
    for o in ("u", "v"):
        gen_all.save("tiles", "gatehouse_closed_" + o, make(o), PIV, "object", scale="battle", blocks=True,
                     note="one wall cell, as high as the walls (it is the wall block); when the gate opens it simply disappears (gatehouse_floor_* stays)")
        gen_all.save("tiles", "gatehouse_floor_" + o, floor(), PIV, "ground", scale="battle")
