"""Candidate tiles and effects for fire, ruin, fences and bridges. Run: python3 gen_all.py"""
import json, math, random
from common import *
from PIL import Image, ImageDraw

OUTLINE = (26, 18, 22, 255)
manifest = {"note": "pivot = ground point the sprite stands on; layer = ground (under units) or object (sorted with units)", "sprites": {}}


def save(group, name, im, pivot, layer, **extra):
    im.save(out(group, name + ".png"))
    manifest["sprites"][name] = dict(file="%s/%s.png" % (group, name), size=list(im.size), pivot=list(pivot), layer=layer, **extra)


# ---- burnt things: recolour the game's own sprites ----
CHAR = [(34, 30, 32), (58, 50, 48), (84, 72, 62), (110, 92, 76)]


def lum(c):
    return (c[0] * 30 + c[1] * 59 + c[2] * 11) / 100


def charred(im, seed, drop, keep_top=0.0, embers=6):
    """Dark outline stays; fills become charcoal by brightness; some fill pixels vanish (more near the top)."""
    rnd = random.Random(seed)
    im = im.copy()
    px = im.load()
    w, h = im.size
    solid = [(x, y) for y in range(h) for x in range(w) if px[x, y][3]]
    ys = [y for _, y in solid]
    top, bot = min(ys), max(ys)
    for x, y in solid:
        r, g, b, a = px[x, y]
        if lum((r, g, b)) < 40:
            continue
        k = (y - top) / max(1, bot - top)  # 0 top .. 1 bottom
        if rnd.random() < drop * (1 - k) ** 0.6 * (1 - keep_top):
            px[x, y] = (0, 0, 0, 0)
            continue
        l = lum((r, g, b))
        i = 0 if l < 70 else 1 if l < 110 else 2 if l < 150 else 3
        c = CHAR[i]
        # a warm tint in the brown parts (wood), cold grey otherwise
        if r > g + 25:
            c = (c[0] + 22, c[1] + 6, c[2] - 4)
        px[x, y] = c + (a,)
    for _ in range(embers):
        x, y = rnd.choice(solid)
        if px[x, y][3]:
            px[x, y] = rnd.choice([(255, 150, 40, 255), (230, 80, 30, 255), (255, 210, 90, 255)])
    return im


for i in range(4):
    im, pv = prop("forest_%d" % i)
    save("tiles", "burnt_forest_%d" % i, charred(im, 10 + i, 0.5, embers=7), pv, "object")
trees = INDEX["tileset"]["animations"]
def skeleton(im, seed):
    """A charred tree: only the trunk stays, with a few bare branches and embers where the crown was."""
    rnd = random.Random(seed)
    im = charred(im, seed, 0.0, embers=0)
    px = im.load()
    w, h = im.size
    pts = [(x, y) for y in range(h) for x in range(w) if px[x, y][3]]
    bot = max(y for _, y in pts)
    base = sorted(x for x, y in pts if y >= bot - 2)
    tx = base[len(base) // 2]
    ty = bot - 9
    for y in range(h):
        for x in range(w):
            if px[x, y][3] and not (y >= ty and abs(x - tx) <= 6):
                px[x, y] = (0, 0, 0, 0)
    ends = []
    for k in range(5):
        dx = rnd.choice([-1, 1]) * rnd.randint(1, 3)
        x, y = tx, ty + 2
        for step in range(rnd.randint(10, 17)):
            y -= 1
            if step % 2 == 0:
                x += 1 if dx > 0 else -1
            if step % 5 == 4:
                x += 1 if dx > 0 else -1
            if 0 <= x < w and 0 <= y < h:
                px[x, y] = CHAR[0] + (255,)
        ends.append((x, y))
    for x, y in ends:
        if 0 <= x < w and 0 <= y < h and rnd.random() < .8:
            px[x, y] = rnd.choice([(255, 150, 40, 255), (230, 80, 30, 255)])
    return im


for i in range(3):
    im, pv = tile(trees["tree_%d" % i]["frames"][0])
    save("tiles", "burnt_oak_%d" % i, skeleton(im, 20 + i), pv, "object", scale="scene", note="scenario scenes only (the battle map uses pines)")
im, pv = structure("village")
save("tiles", "burnt_hut", charred(im, 33, 0.18, embers=9), pv, "object", scale="scene", note="from the structures.png hay-hut cell; scenario scenes")
im, pv = site("village")
save("tiles", "burnt_village", charred(im, 30, 0.2, embers=12), pv, "object", scale="battle", note="from the battle map's village site (100x110, 2x2 cells)")
im, pv = site("camp")
save("tiles", "burnt_camp", charred(im, 32, 0.2, embers=12), pv, "object", scale="battle", note="from the camp site")
im, pv = prop("tavern")
save("tiles", "burnt_tavern", charred(im, 31, 0.3, embers=14), pv, "object", scale="scene", note="scenario scenes only")


# ---- ash: translucent soot patches laid on the ground (a 32x16 diamond, pivot at its centre) ----
def diamond(x, y, w=32, h=16):
    return abs(x + .5 - w / 2) / (w / 2) + abs(y + .5 - h / 2) / (h / 2) <= 1


for level, (cover, alpha) in enumerate([(0.18, 70), (0.42, 110), (0.72, 150)], 1):
    for v in range(3):
        rnd = random.Random(level * 10 + v)
        im = Image.new("RGBA", (32, 16))
        px = im.load()
        blobs = [(rnd.uniform(4, 28), rnd.uniform(3, 13), rnd.uniform(3, 7)) for _ in range(2 + level * 3)]
        for y in range(16):
            for x in range(32):
                if not diamond(x, y):
                    continue
                near = max((1 - math.hypot((x - bx) / 1.6, y - by) / r) for bx, by, r in blobs)
                if near > 0 and rnd.random() < .55 + near * .45:
                    px[x, y] = (28, 24, 26, alpha + rnd.choice([-20, 0, 20]))
                elif rnd.random() < .04:
                    px[x, y] = (70, 64, 62, alpha // 2)
        save("tiles", "ash_%d_%d" % (level, v), im, (16, 8), "ground", level=level)


# ---- fence autotile: post in the middle, a rail toward each connected side ----
WOOD = [(150, 100, 58, 255), (116, 74, 42, 255), (84, 52, 32, 255)]
SIDES = {"nw": (-8, -4), "ne": (8, -4), "se": (8, 4), "sw": (-8, 4)}  # bit 0..3
W, H, PIV = 40, 48, (21, 39)


def rail(px, x0, y0, x1, y1, body):
    """A 2px thick board with a dark line above and below."""
    n = max(abs(x1 - x0), abs(y1 - y0))
    for i in range(n + 1):
        x = round(x0 + (x1 - x0) * i / max(1, n))
        y = round(y0 + (y1 - y0) * i / max(1, n))
        for dy, c in ((-1, OUTLINE), (0, body), (1, WOOD[2]), (2, OUTLINE)):
            if 0 <= x < W and 0 <= y + dy < H:
                px[x, y + dy] = c


def fence_cell(mask, tall):
    im = Image.new("RGBA", (W, H))
    px = im.load()
    cx, cy = PIV
    sides = [s for i, s in enumerate(SIDES) if mask >> i & 1]
    post = 14 if tall else 6
    rails = ((11, WOOD[0]), (6, WOOD[1])) if tall else ((4, WOOD[0]),)
    for far in (True, False):
        if not far:
            for y in range(cy - post, cy + 1):
                for x in ((cx - 1, cx, cx + 1) if tall else (cx - 1, cx)):
                    edge = (x != cx or y in (cy - post, cy)) if tall else (y in (cy - post, cy) or x == cx - 1)
                    px[x, y] = OUTLINE if edge else (WOOD[0] if y < cy - post // 2 else WOOD[1])
            if tall:
                px[cx, cy - post + 1] = WOOD[0]
                px[cx - 1, cy - post] = px[cx + 1, cy - post] = (0, 0, 0, 0)
        for s in sides:
            dx, dy = SIDES[s]
            if (dy < 0) != far:
                continue
            for hgt, col in rails:
                if tall:
                    rail(px, cx, cy - hgt, cx + dx, cy + dy - hgt, col)
                else:
                    n = max(abs(dx), abs(dy))
                    for i in range(n + 1):
                        x, y = cx + round(dx * i / n), cy + round(dy * i / n) - hgt
                        px[x, y] = col
                        if y + 1 < H:
                            px[x, y + 1] = OUTLINE
    return im


for mask in range(16):
    save("tiles", "fence_%02d" % mask, fence_cell(mask, False), PIV, "object", mask=mask, sides_bits=list(SIDES), scale="battle", note="low fence for the deformed battle map")
    save("tiles", "sfence_%02d" % mask, fence_cell(mask, True), PIV, "object", mask=mask, sides_bits=list(SIDES), scale="scene", note="waist-high fence for scenario scenes")


# ---- bridges: damaged (a hole) and broken (two stumps) ----
def bridge_variant(name, kind):
    im, pv = prop(name)
    px = im.load()
    w, h = im.size
    rnd = random.Random(hash((name, kind)) & 0xffff)
    ax = (2, 1) if name.endswith("_u") else (2, -1)  # the span's long direction on screen
    norm = math.hypot(*ax)
    for y in range(h):
        for x in range(w):
            if not px[x, y][3]:
                continue
            dx, dy = x - pv[0], y - pv[1]
            t = (dx * ax[0] + dy * ax[1]) / norm / 28  # -1..1 along the span
            s = (-dx * ax[1] + dy * ax[0]) / norm / 8  # across
            jag = rnd.uniform(-.07, .07)
            if kind == "broken" and abs(t) < .42 + jag:
                px[x, y] = (0, 0, 0, 0)
            elif kind == "damaged" and abs(t) < .3 + jag and abs(s) < .8 + rnd.uniform(-.2, .2):
                px[x, y] = (0, 0, 0, 0)
    # splintered ends: darken the edge pixels next to the gap
    edge = []
    for y in range(h):
        for x in range(w):
            if px[x, y][3] and any(0 <= x + a < w and 0 <= y + b < h and not px[x + a, y + b][3] for a, b in ((1, 0), (-1, 0), (0, 1), (0, -1))):
                if px[x, y][0] > 60 and kind != "intact":
                    edge.append((x, y))
    for x, y in edge:
        if rnd.random() < .6:
            px[x, y] = OUTLINE
    return im, pv


for n in ("bridge_u", "bridge_v"):
    for kind in ("damaged", "broken"):
        im, pv = bridge_variant(n, kind)
        save("tiles", "%s_%s" % (n, kind), im, pv, "ground")


# ---- flame (6 frames, translucent) and smoke (8 frames) ----
def flame_frame(t, n=6, w=14, h=26, seed=0):
    """One narrow teardrop tongue: pointed tip, widest low, rounded fading base (so a ground-level fire never has a flat edge)."""
    im = Image.new("RGBA", (w, h))
    px = im.load()
    ph = t / n * 2 * math.pi + seed
    base = h - 3
    top = h * (0.70 + 0.30 * math.sin(ph * 1.0 + 0.6) * math.sin(ph * 2 + 1.1)) - 2
    sway = math.sin(ph) * 1.6
    for y in range(h):
        r = (base - y) / max(1.0, top)  # 0 at the base .. 1 at the tip
        if r < -0.08 or r > 1:
            continue
        r = max(0.0, r)
        half = (w / 2 - 1) * math.sin(math.pi * min(1.0, r * 0.82 + 0.12)) ** 0.9
        if r > 0.7:
            half *= (1 - r) / 0.3
        cx = w / 2 + sway * r * r
        for x in range(w):
            d = abs(x + .5 - cx)
            if d > half:
                continue
            core = d / max(0.5, half)
            if r < .05:
                c = (255, 230, 140, 80)
            elif core < .38 and r < .5:
                c = (255, 238, 150, 175)
            elif core < .7 and r < .78:
                c = (255, 160, 40, 165)
            elif r < .9:
                c = (232, 84, 30, 150)
            else:
                c = (190, 44, 28, 95)
            px[x, y] = c
    return im


def flame_sheet(name, w, h, note):
    sheet_im = Image.new("RGBA", (w * 6, h))
    for i in range(6):
        sheet_im.paste(flame_frame(i, 6, w, h, seed=0.0), (i * w, 0))
    sheet_im.save(out("effects", name + ".png"))
    return {"file": "effects/%s.png" % name, "frame": [w, h], "frames": 6, "ms": 110, "pivot": [w // 2, h - 3], "note": note}


FX = {"flame_t": flame_sheet("flame_t", 10, 18, "battle map: one narrow tongue, ~1/2 tile tall; 3-5 per 32x16 cell in an ellipse, 8-12 on a 2x2 site"),
      "flame_t_small": flame_sheet("flame_t_small", 7, 11, "battle map: a small tongue for gaps and for embers on ruins"),
      "flame": flame_sheet("flame", 14, 26, "a narrow translucent tongue; stand 3-6 of them on a tile, spread in an ellipse twice as wide as it is deep, with different frame offsets"),
      "flame_small": flame_sheet("flame_small", 10, 17, "a smaller tongue for the gaps between the big ones and for embers on ruins")}


def smoke_frame(i, n=8, s=28):
    im = Image.new("RGBA", (s, s))
    px = im.load()
    k = i / (n - 1)
    r = 4 + 9 * k
    alpha = int(185 * (1 - k) ** 1.3) + 14
    rnd = random.Random(5)
    cl = [(s / 2 + rnd.uniform(-4, 4) * (.3 + k), s / 2 + rnd.uniform(-3, 3) * (.3 + k), r * rnd.uniform(.5, .95)) for _ in range(5)]
    tones = [(150, 146, 150), (112, 108, 114), (78, 74, 80)]
    for y in range(s):
        for x in range(s):
            hit = None
            for cx, cy, rr in cl:
                d = math.hypot(x - cx, (y - cy) * 1.15) / rr
                if d <= 1:
                    lx, ly = cx - rr * .35, cy - rr * .35
                    sh = math.hypot(x - lx, y - ly) / rr
                    if hit is None or sh < hit:
                        hit = sh
            if hit is not None:
                px[x, y] = tones[0 if hit < .5 else 1 if hit < .95 else 2] + (alpha,)
    return im


sm = Image.new("RGBA", (28 * 8, 28))
for i in range(8):
    sm.paste(smoke_frame(i), (i * 28, 0))
sm.save(out("effects", "smoke.png"))

FX["smoke_t"] = {"file": "effects/smoke_t.png", "frame": [16, 16], "frames": 8, "pivot": [8, 8], "note": "battle map smoke particle, rises ~12px/s"}
tsm = Image.new("RGBA", (16 * 8, 16))
for i in range(8):
    tsm.paste(smoke_frame(i, 8, 28).resize((16, 16), Image.NEAREST), (i * 16, 0))
tsm.save(out("effects", "smoke_t.png"))
FX["smoke"] = {"file": "effects/smoke.png", "frame": [28, 28], "frames": 8, "pivot": [14, 14], "note": "one particle's life, frame 0 -> 7: grows and fades; emitter spawns one every ~250ms and moves it up ~22px/s with a slight drift"}
json.dump(FX, open(out("effects", "effects.json"), "w"), indent=1)
json.dump(manifest, open(out("tiles", "tiles.json"), "w"), indent=1)
print(len(manifest["sprites"]), "sprites")
