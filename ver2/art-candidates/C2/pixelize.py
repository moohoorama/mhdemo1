"""Convert Codex-generated source artwork in src/ into native-resolution pixel assets.

Portraits: quantize the painted source to a fixed palette, then take the most
frequent palette color of each source block (no averaging, so no new colors).
Units: key out the magenta background, split the strip into 8 frames, use ONE
scale per unit, place every frame on the same ground anchor, and add a 1px
dark outline.
"""
import json
import os
from collections import Counter

from PIL import Image

HERE = os.path.dirname(os.path.abspath(__file__))
SRC = os.path.join(HERE, "src")
OUT = os.path.join(HERE, "out")

PORTRAIT_SIZE = 96
PORTRAIT_COLORS = 48
CELL = (64, 64)
ANCHOR = (32, 58)
UNIT_COLORS = 32
UNIT_HEIGHT = {"infantry": 44, "bandit": 44, "cavalry": 54}
OUTLINE = (20, 16, 22, 255)


def mode_downscale(img, size):
    """Most frequent color per block; img must already be palette-quantized RGBA."""
    w, h = img.size
    tw, th = size
    px = img.load()
    out = Image.new("RGBA", size, (0, 0, 0, 0))
    po = out.load()
    for ty in range(th):
        y0, y1 = ty * h // th, max(ty * h // th + 1, (ty + 1) * h // th)
        for tx in range(tw):
            x0, x1 = tx * w // tw, max(tx * w // tw + 1, (tx + 1) * w // tw)
            c = Counter(px[x, y] for y in range(y0, y1) for x in range(x0, x1))
            opaque = sum(n for col, n in c.items() if col[3])
            if opaque * 2 < (x1 - x0) * (y1 - y0):
                continue
            po[tx, ty] = max((n, col) for col, n in c.items() if col[3])[1]
    return out


def quantize(img, colors):
    rgb = img.convert("RGB").quantize(colors, method=Image.Quantize.MEDIANCUT, dither=Image.Dither.NONE)
    q = rgb.convert("RGBA")
    if img.mode == "RGBA":
        q.putalpha(img.getchannel("A").point(lambda a: 255 if a >= 128 else 0))
    return q


def portrait(name):
    src = Image.open(os.path.join(SRC, f"portrait-{name}.png")).convert("RGB")
    return mode_downscale(quantize(src, PORTRAIT_COLORS), (PORTRAIT_SIZE, PORTRAIT_SIZE))


def key_magenta(img):
    img = img.convert("RGBA")
    px = img.load()
    for y in range(img.height):
        for x in range(img.width):
            r, g, b, _ = px[x, y]
            # magenta and its anti-aliased fringe: strong red+blue, weak green
            if r > 150 and b > 150 and g < 110 or (r - g > 70 and b - g > 70 and abs(r - b) < 60):
                px[x, y] = (0, 0, 0, 0)
    return img


def split_frames(img, n=8):
    """Split by empty columns when possible; fall back to equal-width cells."""
    a = img.getchannel("A")
    w, h = img.size
    cols = [any(a.getpixel((x, y)) for y in range(0, h, 2)) for x in range(w)]
    runs, start = [], None
    for x, on in enumerate(cols + [False]):
        if on and start is None:
            start = x
        elif not on and start is not None:
            if x - start > w // (n * 6):
                runs.append((start, x))
            start = None
    if len(runs) != n:
        runs = [(i * w // n, (i + 1) * w // n) for i in range(n)]
    frames = []
    for x0, x1 in runs:
        f = img.crop((x0, 0, x1, h))
        frames.append(f.crop(f.getbbox()) if f.getbbox() else f)
    return frames, len(runs) == n


def head_center_x(f):
    """x-centroid of the top 30% of opaque pixels: keeps the head/torso registered."""
    a = f.getchannel("A")
    bb = a.getbbox()
    top = bb[1] + (bb[3] - bb[1]) * 3 // 10
    xs = [x for y in range(bb[1], top) for x in range(f.width) if a.getpixel((x, y))]
    return sum(xs) / len(xs)


def add_outline(f):
    a = f.getchannel("A").load()
    out = f.copy()
    po = out.load()
    w, h = f.size
    for y in range(h):
        for x in range(w):
            if a[x, y]:
                continue
            if any(0 <= x + dx < w and 0 <= y + dy < h and a[x + dx, y + dy]
                   for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1))):
                po[x, y] = OUTLINE
    return out


def box_downscale(f, size):
    """Area-average downscale on premultiplied color, then hard 0/255 alpha."""
    r, g, b, a = f.split()
    pm = Image.merge("RGB", [Image.composite(ch, Image.new("L", f.size, 0), a) for ch in (r, g, b)])
    pm = pm.resize(size, Image.BOX)
    a2 = a.resize(size, Image.BOX)
    px, pa = pm.load(), a2.load()
    out = Image.new("RGBA", size, (0, 0, 0, 0))
    po = out.load()
    for y in range(size[1]):
        for x in range(size[0]):
            al = pa[x, y]
            if al >= 128:
                c = px[x, y]
                po[x, y] = tuple(min(255, v * 255 // al) for v in c) + (255,)
    return out


def unit_frames(kind, anim):
    src = key_magenta(Image.open(os.path.join(SRC, f"unit-{kind}-{anim}.png")))
    frames, split_ok = split_frames(src)
    heights = sorted(f.height for f in frames)
    scale = (UNIT_HEIGHT[kind] - 2) / heights[len(heights) // 2]  # one scale for the whole strip
    centers = [head_center_x(f) for f in frames]
    ref = sum(centers) / len(centers)
    cells = []
    for f, cx in zip(frames, centers):
        small = box_downscale(f, (max(1, round(f.width * scale)), max(1, round(f.height * scale))))
        cell = Image.new("RGBA", CELL, (0, 0, 0, 0))
        # register the head on the strip mean, keeping half of the authored sway
        x = round(ANCHOR[0] - (cx - (cx - ref) * 0.5) * scale)
        y = ANCHOR[1] - small.height
        cell.alpha_composite(small, (max(1, min(CELL[0] - small.width - 1, x)), max(1, y)))
        cells.append(cell)
    return cells, {"splitByGaps": split_ok, "scale": round(scale, 4)}


def shared_palette(cells, colors):
    """Quantize every frame of a unit with ONE palette so idle and walk match."""
    w, h = cells[0].size
    strip = Image.new("RGBA", (w * len(cells), h), (0, 0, 0, 0))
    for i, c in enumerate(cells):
        strip.alpha_composite(c, (i * w, 0))
    q = quantize(strip, colors)
    return [add_outline(q.crop((i * w, 0, (i + 1) * w, h))) for i in range(len(cells))]


def unit(kind):
    idle, ri = unit_frames(kind, "idle")
    walk, rw = unit_frames(kind, "walk")
    return shared_palette(idle + walk, UNIT_COLORS), {"idle": ri, "walk": rw}


def recolor_enemy(f):
    """Blue armor -> red for enemy variants of allied sprites."""
    out = f.copy()
    px = out.load()
    for y in range(out.height):
        for x in range(out.width):
            r, g, b, a = px[x, y]
            if a and b > r + 25 and b > g:
                px[x, y] = (b, int(g * 0.55), int(r * 0.6), 255)
    return out


def check(frames):
    probs = []
    for i, f in enumerate(frames):
        a = f.getchannel("A")
        if set(a.getdata()) - {0, 255}:
            probs.append(f"{i}: alpha")
        bb = a.getbbox()
        if bb[3] - 1 != ANCHOR[1]:  # outline adds the bottom row at the anchor
            probs.append(f"{i}: bottom {bb[3] - 1} != {ANCHOR[1]}")
        if bb[0] == 0 or bb[1] == 0 or bb[2] == CELL[0] or bb[3] == CELL[1]:
            probs.append(f"{i}: touches edge")
    return probs


def main():
    os.makedirs(os.path.join(OUT, "portraits"), exist_ok=True)
    for name in ["liubei", "guanyu", "zhangfei", "chengyuanzhi", "zhangjue"]:
        portrait(name).save(os.path.join(OUT, "portraits", f"{name}.png"))

    rows, report = [], {}
    for kind in ["infantry", "cavalry", "bandit"]:
        frames, rep = unit(kind)
        rep["problems"] = check(frames)
        report[kind] = rep
        rows.append((kind, frames))
    rows.append(("cavalry-enemy", [recolor_enemy(f) for f in rows[1][1]]))
    ids = {"infantry": "infantry-ally", "cavalry": "cavalry-ally", "bandit": "bandit-enemy",
           "cavalry-enemy": "cavalry-enemy"}

    sheet = Image.new("RGBA", (16 * CELL[0], len(rows) * CELL[1]), (0, 0, 0, 0))
    meta = {"cell": list(CELL), "anchor": list(ANCHOR), "displayScale": 3, "mapDisplayScale": 2, "sprites": []}
    for r, (kind, frames) in enumerate(rows):
        for c, f in enumerate(frames):
            sheet.alpha_composite(f, (c * CELL[0], r * CELL[1]))
        meta["sprites"].append({"id": ids[kind], "row": r, "anims": [
            {"name": "idle", "start": 0, "frames": 8, "ms": 150},
            {"name": "walk", "start": 8, "frames": 8, "ms": 110}]})
    sheet.save(os.path.join(OUT, "sheet.png"))
    with open(os.path.join(OUT, "sheet.json"), "w") as fp:
        json.dump(meta, fp, indent=1)
    bg = Image.new("RGBA", sheet.size, (92, 138, 80, 255))
    bg.alpha_composite(sheet)
    bg.resize((sheet.width * 4, sheet.height * 4), Image.NEAREST).save(os.path.join(OUT, "sheet-x4.png"))
    print(json.dumps(report, indent=1))


if __name__ == "__main__":
    main()
