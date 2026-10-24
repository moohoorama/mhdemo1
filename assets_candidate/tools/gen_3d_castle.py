"""A scenario-scale castle kit modelled in 3D (unit3d cel renderer): the battle map's walls are deformed and low, so the scenes get their own:
  swall_{u,v}        a tall plain wall with a crenellated top, one small cell long (seamless along its run)
  sgate_{closed,open}_{u,v}_{k}  a gate tower four cells wide with a two-cell passage a column can walk through, doors, an upper storey
                     with a tiled roof, and on each side a short run of the plain wall (two cells) closed by a thick post; cut into upright
                     slices one small cell wide like the wall pieces, cut down the middle of the arch, slice k standing gate_offset cells along
                     the wall from the gate's point (tiles.json), so in a wall each slice takes a wall piece's place and sorts by depth like it,
                     and a man in the passage stands between the back half and the front half
_u: the piece runs along u (the wall's long face looks SW); _v: along v (looks SE). Pivot: the ground centre of the piece."""
import math, os, sys
sys.path.insert(0, os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))), "tools", "unit3d"))
import numpy as np
from PIL import Image
import base_model as m
import render as R
import props as P
import structures as S
import gen_3d_props as G3

V = S.V
BRICK, BRICK_D, BRICK_L = '#b4bcc2', '#767f87', '#d2d8dc'
GATE_STONE = '#9aa3aa'
R.add_ramps({BRICK: '3455', BRICK_D: '2344', BRICK_L: '4566', GATE_STONE: '3445'})
H, T = 3.2, 1.0    # wall height and thickness (a whole cell, flush with the gate), in model units (one unit = one small cell edge; an officer is 1.5 high)
# the block kit (models2) keeps every part inside its own small cell: the scene rules sort a wall by its cells, so a
# part standing out over the next cell would be drawn in that cell's order, in front of or behind the wrong things
FLUSH = False


def ledge(w):
    """how far a ledge stands out past the wall's face: none in the block kit"""
    return 0 if FLUSH else w


def courses(x0, x1, y, z0, z1, along="x"):
    """mortar lines on a wall face: horizontal courses with staggered vertical joints, as thin protruding boxes (a rod this thin vanishes
    at one pixel in some views)"""
    k = 0
    z = z0 + .3
    while z < z1 - .05:
        m.box(V((x0 + x1) / 2, y, z), V(x1 - x0, .05, .05), BRICK_D)
        zz = z - .3
        joints = [0.0] if k % 2 == 0 else [-.25, .25]
        for jx in joints:
            m.box(V(jx, y, (zz + z) / 2), V(.05, .05, .3), BRICK_D)
        z += .3
        k += 1


def merlons(y, z0, thick=.2, base=.26, h=.6):
    """one parapet along a wall: a base slab and one merlon per cell, whole inside the cell with a dip at both cell edges, an arrow slit and a cap"""
    m.box(V(0, y, z0 + base / 2), V(1.0, thick, base), BRICK)
    m.box(V(0, y, z0 + base + h / 2), V(.56, thick, h), BRICK_L)
    m.box(V(0, y, z0 + base + h + .03), V(.56, thick, .06), BRICK_D)
    m.box(V(0, y - thick / 2 - .005, z0 + base + h * .55), V(.07, .02, .28), P.INK)


def wall_body(tall=H):
    m.box(V(0, 0, tall / 2), V(1.0, T, tall), BRICK)
    m.box(V(0, 0, .12), V(1.0, T + ledge(.08), .24), BRICK_D)         # plinth
    courses(-.5, .5, -T / 2 - .025, .24, tall - .2)
    m.box(V(0, 0, tall - .08), V(1.0, T + ledge(.14), .16), BRICK_D)   # the corbel course under the parapets


def parapets():
    """the block kit's parapets: low merlons along both edges of the walkway, so its top shows the whole small cell"""
    for y in (-T / 2 + .08, T / 2 - .08):
        merlons(y, H, thick=.16, base=.18, h=.34)


def strip():
    """a wall piece is rendered as three pieces in a row and only the middle one is kept (cropped to 16 px, one cell's run), so its ends
    carry no outline and neighbours join seamlessly"""
    base = list(m.meshes)
    m.meshes = [(np.asarray(v) + [dx, 0, 0], c) for dx in (-1.0, 0.0, 1.0) for v, c in base]


def wall():
    m.meshes = []
    wall_body()
    merlons(-T / 2 + .1, H)                                         # the high outer parapet, toward the camera
    merlons(T / 2 - .08, H, thick=.14, base=.22, h=0)               # a low inner kerb behind the walkway
    strip()
    return G3.finish((80, 130), (40, 112), 1.0)


W2, D, PIER, PH = 4.0, 1.0, 1.0, 2.5                                # gate: width, depth, pier width, passage height
WING, POST = 2.0, 1.0   # each side of the tower: a run of wall (one game cell) and a thick post (half a cell) closing it
OPEN = W2 - 2 * PIER
HALF = {}


def gb(cx, y0, y1, z0, z1, wx, color):
    """a box over y in [y0, y1], clipped to the half being drawn along x"""
    x0, x1 = max(cx - wx / 2, HALF["x"][0]), min(cx + wx / 2, HALF["x"][1])
    if x1 - x0 > 1e-6:
        m.box(V((x0 + x1) / 2, (y0 + y1) / 2, (z0 + z1) / 2), V(x1 - x0, y1 - y0, z1 - z0), color)


def gate(closed):
    """a gate tower along x: a passage along y with a stepped arch, a plastered upper storey with dark windows and a tiled hip roof, a
    wing of plain wall and a thick post on either side"""
    m.meshes = []
    HALF["x"] = (-99.0, 99.0)
    for sx in (-1, 1):
        cx = sx * (W2 / 2 - PIER / 2)
        gb(cx, -D / 2, D / 2, 0, H, PIER, GATE_STONE)
        gb(cx, -D / 2 - ledge(.04), D / 2 + ledge(.04), 0, .24, PIER + ledge(.06), BRICK_D)
        for z in (.5, .8, 1.1, 1.4, 1.7, 2.0, 2.3, 2.6, 2.9):
            gb(cx, -D / 2 - .05, -D / 2, z - .025, z + .025, PIER, BRICK_D)
    gb(0, -D / 2, D / 2, PH, H, OPEN + .02, GATE_STONE)             # the lintel block over the passage
    for i, w in enumerate((.45, .3, .15)):                          # the arch steps down into the corners
        for sx in (-1, 1):
            gb(sx * (OPEN / 2 - w / 2), -D / 2, D / 2, PH - .15 * (i + 1), PH - .15 * i, w, GATE_STONE)
    gb(0, -D / 2 - .03, -D / 2, PH - .06, PH + .06, OPEN - .2, P.GOLD)   # a gilded band along the arch
    if closed:
        for sx in (-1, 1):
            gb(sx * OPEN / 4, -D / 2 + .06, -D / 2 + .22, 0, PH - .3, OPEN / 2 - .03, P.PLANK)
            for z in (.45, 1.0, 1.55, 2.0):
                gb(sx * OPEN / 4, -D / 2 + .02, -D / 2 + .06, z - .04, z + .04, OPEN / 2 - .03, BRICK_D)
        gb(0, -D / 2 + .02, -D / 2 + .06, 0, PH - .3, .05, P.INK)
    else:
        for sx in (-1, 1):                                          # opened inward, folded flat against the piers
            gb(sx * (OPEN / 2 - .06), -.42, .42, 0, PH - .35, .08, P.PLANK)
    sz = H + .02                                                    # upper storey
    yf = -(D - .15) / 2
    gb(0, yf, -yf, sz, sz + 1.3, W2 - .2, S.PLASTER)
    for sx in (-1.2, 0, 1.2):
        gb(sx, yf - .015, yf, sz + .35, sz + .95, .55, P.INK)
    for sx in (-2, -1, 0, 1, 2):
        gb(sx * .9, yf - .03, yf, sz, sz + 1.3, .09, P.LACQUER)
    gb(0, -D / 2, D / 2, sz - .03, sz + .03, W2, BRICK_D)
    for sx in (-1, 1):
        wing(sx)
        post(sx * (W2 / 2 + WING + POST / 2))
    G3.hip_roof(-W2 / 2 - .25, W2 / 2 + .25, -D / 2 - ledge(.3), D / 2 + ledge(.25), sz + 1.32, .85, G3.TILE_R, G3.TILE_D)
    return list(m.meshes)


def wing(sx):
    """a short run of the plain wall against the tower on side sx: the wall's own body, mortar, corbel course and
    parapets, so the wall laid behind meets it in kind wherever it is cut"""
    x0, x1 = sorted((sx * W2 / 2, sx * (W2 / 2 + WING)))
    cx, L = (x0 + x1) / 2, x1 - x0
    gb(cx, -T / 2, T / 2, 0, H, L, BRICK)
    gb(cx, -T / 2 - ledge(.04), T / 2 + ledge(.04), 0, .24, L, BRICK_D)
    gb(cx, -T / 2 - ledge(.07), T / 2 + ledge(.07), H - .16, H, L, BRICK_D)
    k, z = 0, .54
    while z < H - .25:  # mortar courses with staggered joints
        gb(cx, -T / 2 - .05, -T / 2, z - .025, z + .025, L, BRICK_D)
        for jx in np.arange(x0 + (.25 if k % 2 else .5), x1 - .01, .5):
            gb(jx, -T / 2 - .05, -T / 2, z - .3, z, .05, BRICK_D)
        z += .3
        k += 1
    for c in np.arange(x0 + .5, x1, 1.0):  # one merlon per small cell, like the wall pieces
        if FLUSH:  # the block kit's low merlons on both edges (parapets)
            for y in (-T / 2 + .08, T / 2 - .08):
                gb(c, y - .08, y + .08, H, H + .18, 1.0, BRICK)
                gb(c, y - .08, y + .08, H + .18, H + .52, .56, BRICK_L)
                gb(c, y - .08, y + .08, H + .52, H + .58, .56, BRICK_D)
            continue
        y = -T / 2 + .1
        gb(c, y - .1, y + .1, H, H + .26, 1.0, BRICK)
        gb(c, y - .1, y + .1, H + .26, H + .86, .56, BRICK_L)
        gb(c, y - .1, y + .1, H + .86, H + .92, .56, BRICK_D)
        gb(c, y - .105, y - .1, H + .4, H + .68, .07, P.INK)
        gb(c, T / 2 - .15, T / 2 - .01, H, H + .22, 1.0, BRICK)


def post(cx):
    """the thick post that closes a wing: a stone pier standing proud of the wall front and back and above its
    parapet, plinth and capstone, a lantern slot"""
    t = T + ledge(.4)  # the block kit's post is a tower of its own small cell, as thick as the wall
    top = H + (1.4 if FLUSH else .7)  # in the block kit it rises well over the parapets, so the wall before it hides only its foot
    gb(cx, -t / 2, t / 2, 0, top, POST, GATE_STONE)
    gb(cx, -t / 2 - ledge(.05), t / 2 + ledge(.05), 0, .3, POST + ledge(.1), BRICK_D)
    gb(cx, -t / 2 - ledge(.08), t / 2 + ledge(.08), top, top + .16, POST + ledge(.16), BRICK_D)
    gb(cx, -t / 2 - ledge(.04), t / 2 + ledge(.04), top + .16, top + .3, POST - .1, BRICK_L)
    for z in np.arange(.7, top - .3, .4):
        gb(cx, -t / 2 - .05, -t / 2, z - .025, z + .025, POST, BRICK_D)
    gb(cx, -t / 2 - .06, -t / 2, top - .9, top - .4, .14, P.INK)


def mirror():
    m.meshes = [(np.asarray(v)[:, [1, 0, 2]] * [-1, -1, 1], c) for v, c in m.meshes]


def crop_strip(im, pivot, side):
    """keep only the middle cell's 16 px run of a three-piece strip, centred on the cell's front face (it sits half the wall's thickness toward
    the camera: 8 px left of the pivot for _u, right for _v), so a tile owns its whole face and the merlon above it"""
    x0 = pivot[0] - 16 if side == "u" else pivot[0]
    return im.crop((x0, 0, x0 + 16, im.height)), (pivot[0] - x0, pivot[1])


def models():
    out = {}
    im = wall()
    out["swall_u"] = crop_strip(im, im.pivot, "u")
    mirror()
    im = P.draw((80, 130), (40, 112))
    out["swall_v"] = crop_strip(im, im.pivot, "v")
    for state, closed in (("closed", True), ("open", False)):
        for orient in ("u", "v"):
            gate(closed)
            if orient == "v":
                mirror()
            whole = P.draw((340, 230), (170, 180))
            for k, (im, pv, off) in enumerate(slices(whole, whole.pivot, orient)):
                out["sgate_%s_%s_%d" % (state, orient, k)] = (im, pv, off)
    return out


def slices(im, pivot, orient):
    """cut the whole gate into half-cell upright slices the size of the wall pieces, each with its own ground point on the wall line
    (offset cells along the wall from the gate's point), so every slice sorts by depth exactly like the wall piece whose place it takes.
    The cut between slices 0 and 0.5 runs down the middle of the arch: the back half of the gate (smaller u or v, up the screen) sorts
    before a man walking through the passage, the front half (the lintel's near half, the near pier, wing and post) after him."""
    px, py = pivot
    out = []
    for k in range(-7, 9):
        off = k * .5
        x = px + 32 * off if orient == "u" else px - 32 * off  # the slice's ground point on screen (along u right, along v left)
        y = py + 16 * off
        x0 = int(round(x - 16 if orient == "u" else x))
        part = im.crop((x0, 0, x0 + 16, im.height))
        if part.getbbox() is None:
            continue
        out.append((part, (int(round(x)) - x0, int(round(y))), off))
    return out


def run():
    import gen_all
    for name, (im, pv, *_) in models().items():
        t, p = G3.trimmed(im, pv)
        gen_all.save("tiles", name, t, p, "object", scale="scene", made="3D model + unit3d cel renderer")


def write():
    """save the castle kit into ../tiles and update only its own entries of tiles.json (gen_all.py rewrites the whole
    manifest; this leaves everyone else's entries alone)"""
    import json
    tiles = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "tiles")
    made = {}
    for name, (im, pv, *off) in models().items():
        t, p = G3.trimmed(im, pv)
        t.save(os.path.join(tiles, name + ".png"))
        made[name] = dict(file="tiles/%s.png" % name, size=list(t.size), pivot=list(p), layer="object", scale="scene",
                          made="3D model + unit3d cel renderer")
        if off:
            made[name]["gate_offset"] = off[0]  # cells along the wall from the gate's point
    path = os.path.join(tiles, "tiles.json")
    man = json.load(open(path))
    for old in [n for n in man["sprites"] if n.startswith(("sgate_post_", "sgate_open_", "sgate_closed_")) and n not in made]:
        del man["sprites"][old]  # the whole-gate halves and the far post the slices replace
        f = os.path.join(tiles, old + ".png")
        if os.path.exists(f):
            os.remove(f)
    man["sprites"].update(made)
    json.dump(man, open(path, "w"), ensure_ascii=False, indent=1)
    open(path, "a").write("\n")
    for n, e in made.items():
        print(n, e["size"], e["pivot"])


# ---- the block kit: a wall piece is a whole small cell of wall (its front, its walkway and both parapets), a gate
# is ten small cells of it; each picture is cut out of a longer render by which cell owns each pixel (render's owner)

import heads as Hd

BLOCK_CELL, BLOCK_PIVOT = (96, 140), (48, 120)
GATE_CELL, GATE_PIVOT = (360, 240), (180, 190)
GATE_TILES = 10  # the gate model runs over x in [-5, 5): the tower, a wing and a post either side


def owned_draw(cell, pivot, owner):
    """the model drawn whole, and which part owns each pixel (owner of model-space points; -1 for none)"""
    own = []
    im = Hd.to_img(R.render(m.meshes, S.VIEW, S.SCALE, cell=cell, pivot=pivot, owner=owner, owners=own))
    im.pivot = (pivot[0] + 1, pivot[1] + 1)
    return im, own[0]


def keep(im, own, k):
    """the pixels of im owned by k, trimmed; the pivot moves with the crop"""
    a = np.array(im)
    a[own != k] = 0
    part = Image.fromarray(a)
    x0, y0, x1, y1 = part.getbbox()
    return part.crop((x0, y0, x1, y1)), (im.pivot[0] - x0, im.pivot[1] - y0)


def along(orient):
    """the model coordinate along the wall: x for _u, and for _v (the model mirrored by mirror()) -y"""
    return (lambda p: p[:, 0]) if orient == "u" else (lambda p: -p[:, 1])


def block(orient):
    """one small cell of wall: the cell drawn alone (its whole shape, so the end of a line shows its end face), less
    the outline it would cast on the neighbour behind it in a line; where three in a row show it, their colours (no
    outline at the joints)"""
    def model(row):
        m.meshes = []
        wall_body()
        parapets()
        if row:
            strip()
        if orient == "v":
            mirror()
    t = along(orient)
    model(True)
    row, own = owned_draw(BLOCK_CELL, BLOCK_PIVOT, lambda p: np.floor(t(p) + .5).astype(int) + 1)
    model(False)
    alone = P.draw(BLOCK_CELL, BLOCK_PIVOT)
    a, r = np.array(alone), np.array(row)
    ink = np.all(a[..., :3] == R.COLOR['0'], axis=-1)
    a[own == 1] = r[own == 1]
    a[(own == 0) & ink] = 0  # its outline over the neighbour behind, which in a line is drawn first
    im = Image.fromarray(a)
    im.pivot = row.pivot
    x0, y0, x1, y1 = im.getbbox()
    return im.crop((x0, y0, x1, y1)), (im.pivot[0] - x0, im.pivot[1] - y0)


def gate_tiles(closed, orient):
    """the gate's ten slices, slice j the pixels of the small cell x in [j - 5, j - 4): a slice stands on its own cell
    of the wall line, gate_offset game cells from the gate's point (the cell just past the middle of the arch)"""
    gate(closed)
    tower = list(m.meshes)
    m.meshes = []  # a wall block on either side, drawn with the gate so its ends meet them without an outline
    wall_body()
    parapets()
    wall = list(m.meshes)
    m.meshes = tower + [(np.asarray(v) + [dx, 0, 0], c) for dx in (-GATE_TILES / 2 - .5, GATE_TILES / 2 + .5) for v, c in wall]
    if orient == "v":
        mirror()
    t = along(orient)

    def owner(p):
        j = np.floor(t(p)).astype(int) + GATE_TILES // 2
        return np.where((j >= 0) & (j < GATE_TILES), j, GATE_TILES)  # the blocks beside: their own part, left out

    im, own = owned_draw(GATE_CELL, GATE_PIVOT, owner)
    # the outline between the gate's end and a block beside it is drawn by neither the block (its outline over the
    # cell behind it is left out) nor that block's part here: the gate's end cell takes it
    ink = np.all(np.array(im)[..., :3] == R.COLOR['0'], axis=-1)
    for y, x in zip(*np.nonzero(ink & (own == GATE_TILES))):
        near = [own[y + dy, x + dx] for dy, dx in R.N4 if 0 <= y + dy < own.shape[0] and 0 <= x + dx < own.shape[1]]
        mine = [k for k in near if 0 <= k < GATE_TILES]
        if mine:
            own[y, x] = max(set(mine), key=mine.count)
    out = []
    for j in range(GATE_TILES):
        part, (px, py) = keep(im, own, j)
        rel = j - 5
        dx = 16 * (rel + .5) if orient == "u" else -16 * (rel + .5)  # the slice's cell centre from the gate's model origin
        out.append((part, (int(round(px + dx)), int(round(py + 8 * (rel + .5)))), rel / 2))
    return out


def models2():
    global FLUSH
    FLUSH = True
    out = {}
    for orient in ("u", "v"):
        out["swall2_" + orient] = block(orient)
    for state, closed in (("closed", True), ("open", False)):
        for orient in ("u", "v"):
            for j, (im, pv, off) in enumerate(gate_tiles(closed, orient)):
                out["sgate2_%s_%s_%d" % (state, orient, j)] = (im, pv, off)
    FLUSH = False
    return out


def write2():
    """save the block kit into ../tiles and update only its entries of tiles.json"""
    import json
    tiles = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "tiles")
    made = {}
    for name, (im, pv, *off) in models2().items():
        im.save(os.path.join(tiles, name + ".png"))
        made[name] = dict(file="tiles/%s.png" % name, size=list(im.size), pivot=list(pv), layer="object", scale="scene",
                          made="3D model + unit3d cel renderer, cut by owning cell (gen_3d_castle.py)")
        if off:
            made[name]["gate_offset"] = off[0]
    path = os.path.join(tiles, "tiles.json")
    man = json.load(open(path))
    man["sprites"].update(made)
    json.dump(man, open(path, "w"), ensure_ascii=False, indent=1)
    open(path, "a").write("\n")
    for n, e in made.items():
        print(n, e["size"], e["pivot"], e.get("gate_offset", ""))


if __name__ == "__main__" and sys.argv[1:] == ["--write2"]:
    write2()
elif __name__ == "__main__" and sys.argv[1:2] == ["--preview2"]:
    # python3 gen_3d_castle.py --preview2 OUT_DIR: the block kit's pictures, and each gate put back together
    out = sys.argv[2]
    for name, (im, pv, *off) in models2().items():
        im.save(os.path.join(out, name + ".png"))
        print(name, im.size, pv, off)
elif __name__ == "__main__" and sys.argv[1:] == ["--write"]:
    write()
elif __name__ == "__main__":
    # preview: python3 gen_3d_castle.py OUT_DIR writes every model's trimmed picture there
    out = sys.argv[1]
    for name, (im, pv, *off) in models().items():
        t, p = G3.trimmed(im, pv)
        print(name, t.size, p, off)
        t.save(os.path.join(out, "cs_" + name + ".png"))
