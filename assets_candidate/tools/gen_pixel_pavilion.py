"""An open pavilion deck for the scenes (조조's back garden, S_pond_deck, chapter 17): a 정자 with no roof and no
pillars, drawn as pixel art on the small-tile grid with the castle kit's renderer (gen_pixel_castle.render). The deck
stands one step up (scene.json "heights" "5"): a granite plinth under a wooden fascia, planks on top (ground glyph W),
a wooden railing round its outer edges; one flight of two stone steps (heights "6", half the lift) leads up from the
path, and the railing is open there.

  pdeck_<mask>_<k>   the plinth face of a deck tile, mask 4 (+u drops) / 8 (+v) / 12 (both), k = 0..2, from the
                     ground (z = 0) to the deck (LIFT), pivot at the tile's centre on the ground.
  prail_<side>_<k>   the railing on a deck tile's edge over lower ground (heights "rim"), side sw/se/ne/nw, k = 0..3
                     (k 1..3 only differ in the grain), pivot at the tile's centre on the deck. Posts at both ends of
                     the edge, so tiles meet on one post; a top rail and a middle rail between them.
  pstep_<mask>_<k>   a step tile's face (heights "6", STEP px), as pdeck, with the upper riser (STEP to LIFT) on its back
                     edge where the deck rises behind it: a face, so it stands on the ground among the faces and the
                     raised tops never cover it (the deck's own face is left out beside a step, the step being higher
                     by level).

  python3 gen_pixel_pavilion.py --preview DIR
  python3 gen_pixel_pavilion.py --write
"""
import json
import os
import sys

import numpy as np
from PIL import Image

import gen_pixel_castle as pc

LIFT = 16                  # the deck over the ground, pixels
STEP = 8                   # the lower step
FASCIA = 3                 # the wooden band along the plinth's top
RAIL = (14, .1, .08)       # railing: height, inset from the edge, thickness (tiles)
POST = .1                  # post width along the edge (tiles)
DECK, RAILS, STEPS = "pdeck", "prail", "pstep"

pc.PAL.update({
    "pG0": (150, 146, 136), "pG1": (140, 136, 126), "pG2": (160, 156, 144), "pGj": (98, 94, 88),     # granite lit
    "pg0": (110, 106, 100), "pg1": (102, 98, 92), "pg2": (118, 114, 106), "pgj": (72, 70, 66),       # granite shaded
    "pGt": (176, 172, 160),                                                                           # step tread
    "pF": (124, 72, 40), "pf": (90, 52, 30), "pFt": (146, 92, 54),                                    # fascia
    "pR": (132, 46, 34), "pr": (96, 34, 26), "pRt": (166, 70, 50), "pRk": (70, 26, 20),               # lacquered rail
    "pM": (60, 84, 52), "pm": (48, 68, 42),                                                           # moss
})


def granite(s, h, lit_face, k=None):
    """blocks 12 dots long, courses 5 px (4 and a joint), every other course shifted; tones by the block's place"""
    course = np.floor_divide(h, 5)
    off = np.mod(course, 2) * 6
    b = np.floor_divide(s + off, 12)
    joint = (np.mod(h, 5) == 4) | (np.mod(s + off, 12) == 11)
    t = np.mod(course * 5 + b * 7, 6)
    tone = np.where(t < 3, "0", np.where(t < 5, "1", "2"))
    pre = "pG" if lit_face else "pg"
    keys = np.where(joint, pre + "j", np.char.add(pre, tone))
    if k == 1:      # moss at the foot, inside the tile's middle dots
        m = np.mod(s, 16)
        keys = np.where((m >= 5) & (m <= 10) & (h <= 1 + np.mod(m, 2)), "pM" if lit_face else "pm", keys)
    if k == 2:      # a chipped block
        m = np.mod(s, 16)
        keys = np.where((m >= 7) & (m <= 9) & (h >= 6) & (h <= 7), pre + "j", keys)
    return keys


def plinth_paint(top, k=None):
    def paint(f, u, v, z, s, h):
        lit_face = pc.lit(f)
        if f == "+z":
            return np.full(u.shape, "pGt")
        keys = granite(s, h, lit_face, k)
        if top == LIFT:     # the deck's wooden fascia
            band = h >= LIFT - FASCIA
            keys = np.where(band, np.where(h == LIFT - 1, "pFt", "pF" if lit_face else "pf"), keys)
        return keys
    return paint


def block(u, v, tile, top, k=None):
    return pc.Body(pc.box(u, u + 1, v, v + 1, 0, top), plinth_paint(top, k), tile)


BOUNDS = (-48, -LIFT - 40, 48, 40)


def face(mask, k, top):
    du, dv = mask & 4, mask & 8
    near = [(0, -1), (-1, -1)] + ([] if dv else [(-1, 1), (0, 1)]) + ([] if du else [(1, -1), (1, 0)])
    near += [(-1, 0)] if dv or not du else []
    bodies = [block(0, 0, 0, top, k)] + [block(a, b, -1, top, k) for a, b in set(near)]
    if top == STEP:     # the upper riser on the step's back edge, the deck behind it as context
        bodies += [pc.Body(pc.box(0, 1, -.3, 0, STEP, LIFT), plinth_paint(LIFT), 0),
                   pc.Body(pc.box(-1, 2, -1.3, -.3, 0, LIFT), plinth_paint(LIFT), -1)]
    a, own, o = pc.render(bodies, bounds=BOUNDS)
    # the top is the deck's own floor: keep only the faces
    top_px = np.all(a[..., :3] == pc.PAL["pGt"], axis=-1) & (a[..., 3] > 0) & (own == 0)
    a[top_px] = 0
    own = np.where(top_px, -1, own)
    return pc.piece(a, own, o, 0, (0, 8))


def strip(side, u, v, a0, a1, c0, c1, z0, z1):
    if side == "sw":
        return pc.box(u + a0, u + a1, v + 1 - c1, v + 1 - c0, z0, z1)
    if side == "se":
        return pc.box(u + 1 - c1, u + 1 - c0, v + a0, v + a1, z0, z1)
    if side == "ne":
        return pc.box(u + a0, u + a1, v + c0, v + c1, z0, z1)
    return pc.box(u + c0, u + c1, v + a0, v + a1, z0, z1)


def wood_paint(k):
    def paint(f, u, v, z, s, h):
        if f == "+z":
            return np.full(u.shape, "pRt")
        lit_face = pc.lit(f)
        base = "pR" if lit_face else "pr"
        grain = np.mod(s * 3 + h + k, 7) == 0
        return np.where(grain, "pRk" if not lit_face else "pr", base)
    return paint


def rail(side, u, v, tile, k):
    """posts at both ends of the edge, a top rail and a middle rail between them"""
    hgt, inset, t = RAIL
    p = wood_paint(k)
    z0 = LIFT
    out = []
    for a in (0, 1 - POST):
        out.append(pc.Body(strip(side, u, v, a, a + POST, inset, inset + t + .02, z0, z0 + hgt), p, tile))
    out.append(pc.Body(strip(side, u, v, 0, 1, inset + .01, inset + t, z0 + hgt - 3, z0 + hgt - 1), p, tile))
    out.append(pc.Body(strip(side, u, v, 0, 1, inset + .02, inset + t - .01, z0 + 5, z0 + 6), p, tile))
    return out


def rail_piece(side, k):
    along = [(-1, 0), (1, 0)] if side in ("sw", "ne") else [(0, -1), (0, 1)]
    bodies = rail(side, 0, 0, 0, k) + [block(0, 0, -1, LIFT)]
    for a, b in along:
        bodies += rail(side, a, b, -1, 0) + [block(a, b, -1, LIFT)]
    a, own, o = pc.render(bodies, bounds=BOUNDS)
    return pc.piece(a, own, o, 0, (0, 8 - LIFT))


def empty():
    return Image.new("RGBA", (1, 1), (0, 0, 0, 0)), (0, 0)


def kit():
    out = {f"{DECK}_{m:02d}_{k}": face(m, k, LIFT) for m in (4, 8, 12) for k in range(3)}
    out.update({f"{STEPS}_{m:02d}_{k}": face(m, k, STEP) for m in (4, 8, 12) for k in range(3)})
    out.update({f"{RAILS}_{side}_{k}": rail_piece(side, k) for side in ("sw", "se", "ne", "nw") for k in range(4)})
    return {n: (p if p is not None else empty()) for n, p in out.items()}


MADE = "open pavilion deck, pixel art on the small-tile grid (gen_pixel_pavilion.py)"


def write():
    here = os.path.dirname(os.path.abspath(__file__))
    tiles_dir = os.path.join(here, "..", "tiles")
    path = os.path.join(tiles_dir, "tiles.json")
    tiles = json.load(open(path))
    for name, (img, pivot) in kit().items():
        img.save(os.path.join(tiles_dir, name + ".png"))
        tiles["sprites"][name] = {"file": f"tiles/{name}.png", "size": list(img.size), "pivot": list(pivot),
                                  "layer": "object", "scale": "scene", "made": MADE, "shadow": False}
    json.dump(tiles, open(path, "w"), ensure_ascii=False, indent=1)
    open(path, "a").write("\n")


def lay(canvas, img_pivot, x, y):
    img, (px, py) = img_pivot
    canvas.alpha_composite(img, (int(x - px), int(y - py)))


def preview(out):
    """a 10 × 6 deck with a two-tile flight of steps in its front edge, laid the way the engine lays it"""
    os.makedirs(out, exist_ok=True)
    k = kit()
    W, D, gap = 10, 6, (2, 3)
    cv = Image.new("RGBA", (420, 300), (70, 100, 62, 255))
    ox, oy = 210 - 16 * (W // 2 - D // 2), 150 - 8 * (W // 2 + D // 2)

    def at(u, v, z=0):
        return ox + 16 * (u - v), oy + 8 * (u + v) - z
    wood = Image.new("RGBA", (32, 16), (0, 0, 0, 0))
    X, Y = np.meshgrid(np.arange(32) + .5, np.arange(16) + .5)
    inside = np.abs(X - 16) / 16 + np.abs(Y - 8) / 8 <= 1
    arr = np.zeros((16, 32, 4), np.uint8)
    arr[inside] = (150, 102, 62, 255)
    arr[inside & (np.mod(np.floor(X / 2 + Y), 4) == 0)] = (120, 80, 48, 255)
    wood = Image.fromarray(arr)
    for v in range(D):
        for u in range(W):
            x, y = at(u, v, LIFT)
            cv.alpha_composite(wood, (int(x - 16), int(y)))
    for u in range(W):
        x, y = at(u, 0, LIFT)
        lay(cv, k[f"{RAILS}_ne_{u % 4}"], x, y + 8)
    for v in range(D):
        x, y = at(0, v, LIFT)
        lay(cv, k[f"{RAILS}_nw_{v % 4}"], x, y + 8)
    for v in range(D):
        x, y = at(W - 1, v)
        lay(cv, k[f"{DECK}_{12 if v == D - 1 else 4:02d}_{v % 3}"], x, y + 8)
    for u in range(W):
        if gap[0] <= u <= gap[1]:
            x, y = at(u, D)
            lay(cv, k[f"{STEPS}_{12 if u == gap[1] else 8:02d}_{u % 3}"], x, y + 8)
            continue
        x, y = at(u, D - 1)
        lay(cv, k[f"{DECK}_{12 if u == W - 1 else 8:02d}_{u % 3}"], x, y + 8)
        x, y = at(u, D - 1, LIFT)
        lay(cv, k[f"{RAILS}_sw_{u % 4}"], x, y + 8)
    for v in range(D):
        x, y = at(W - 1, v, LIFT)
        lay(cv, k[f"{RAILS}_se_{v % 4}"], x, y + 8)
    cv.resize((cv.width * 3, cv.height * 3), Image.NEAREST).save(os.path.join(out, "layout.png"))


if __name__ == "__main__":
    if sys.argv[1:2] == ["--write"]:
        write()
    elif sys.argv[1:2] == ["--preview"]:
        preview(sys.argv[2])
    else:
        print(__doc__)
