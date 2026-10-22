#!/usr/bin/env python3
"""Compile hand-authored pixel sheets (tools/spritetool/assets/ver5-projectiles, ver5-fx, ver5-outside, ver5-ui) without resampling.

A sprite is a stack of layers. A layer is either an ASCII grid per frame in pixels.json, coloured by
the layer's palette ("." is clear), or, for soft translucent stuff, an image a function in the
source's gen.py draws per frame. Each layer has an opacity (one number, or one per frame) and a
blend: "normal" layers are flattened into <name>.png, "lighter" (additive) ones into
<name>_light.png, which the game draws with additive blending over the first. A sprite with no
"layers" is one opaque normal layer whose frames are plain grids (the projectile sheet's form).
A sprite with "dot": k has grids k times smaller than its cell, each letter a k×k block (dot art
for the UI's four-pixels-per-UI-pixel scale).
"""
import importlib.util
import json
import sys
from pathlib import Path

from PIL import Image, ImageColor

sys.dont_write_bytecode = True  # importing a source's gen.py leaves no __pycache__ beside it
ROOT = Path(__file__).resolve().parents[1]
ASSETS = ROOT.parent / "tools/spritetool/assets"
# source directory -> sheet path under assets/graphics
SOURCES = {"ver5-projectiles": "effects/projectiles", "ver5-fx": "effects/fx", "ver5-outside": "map/outside", "ver5-ui": "ui/ui"}


def layers_of(s):
    if "layers" in s:
        return s["layers"]
    return {"main": {"palette": s["palette"]}}


def opacity(layer, i):
    a = layer.get("alpha", 1)
    return a[i] if isinstance(a, list) else a


def grid_image(grid, palette, w, h, name):
    assert len(grid) == h and all(len(r) == w for r in grid), f"{name}: not {w}x{h}"
    im = Image.new("RGBA", (w, h))
    for y, row in enumerate(grid):
        for x, pixel in enumerate(row):
            if pixel != ".":
                im.putpixel((x, y), ImageColor.getrgb(palette[pixel]) + (255,))
    return im


def faded(im, a):
    if a >= 1:
        return im
    r, g, b, alpha = im.split()
    return Image.merge("RGBA", (r, g, b, alpha.point(lambda v: round(v * a))))


def add(acc, im):
    """Adds im's premultiplied colour into acc (a w*h list of [r, g, b] in 0..1)."""
    for i, (r, g, b, a) in enumerate(im.getdata()):
        if a:
            k = a / 255 / 255
            acc[i][0] += r * k
            acc[i][1] += g * k
            acc[i][2] += b * k


def from_sum(acc, w, h):
    """An image that adds acc when drawn additively: alpha is the brightest channel."""
    im = Image.new("RGBA", (w, h))
    out = []
    for r, g, b in acc:
        r, g, b = min(1, r), min(1, g), min(1, b)
        a = max(r, g, b)
        out.append((0, 0, 0, 0) if a == 0 else (round(r / a * 255), round(g / a * 255), round(b / a * 255), round(a * 255)))
    im.putdata(out)
    return im


def build(source, name):
    src = ASSETS / source
    settings = json.loads((src / "settings.json").read_text())
    pixels = json.loads((src / "pixels.json").read_text())
    gen = None
    if (src / "gen.py").exists():
        spec = importlib.util.spec_from_file_location("gen", src / "gen.py")
        gen = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(gen)
    rows = []  # one sheet row per sprite: (name, settings, [(normal, light) per frame])
    for sprite in settings["frame_order"]:
        s = settings["sprites"][sprite]
        w, h = s["cell"]
        layers = layers_of(s)
        frames = pixels.get(sprite) or [{} for _ in range(s["frames"])]
        out = []
        for i, frame in enumerate(frames):
            normal, light, lit = Image.new("RGBA", (w, h)), [[0.0, 0.0, 0.0] for _ in range(w * h)], False
            for lname, layer in layers.items():
                if "gen" in layer:
                    im = getattr(gen, layer["gen"])(i, len(frames), w, h)
                else:
                    grid = frame if isinstance(frame, list) else frame[lname]
                    palette = {"o": settings["outline"], **layer["palette"]}
                    k = s.get("dot", 1)
                    im = grid_image(grid, palette, w // k, h // k, f"{sprite} frame {i} {lname}").resize((w, h), Image.NEAREST)
                im = faded(im, opacity(layer, i))
                if layer.get("blend", "normal") == "lighter":
                    add(light, im)
                    lit = True
                else:
                    normal.alpha_composite(im)
            out.append((normal, from_sum(light, w, h) if lit else None))
        rows.append((sprite, s, out))
    width = max(s["cell"][0] * len(frames) for _, s, frames in rows)
    height = sum(s["cell"][1] for _, s, _ in rows)
    sheet, light_sheet = Image.new("RGBA", (width, height)), Image.new("RGBA", (width, height))
    index, top, any_light = {}, 0, False
    for sprite, s, frames in rows:
        w, h = s["cell"]
        rects = []
        for i, (normal, light) in enumerate(frames):
            sheet.paste(normal, (i * w, top))
            if light:
                light_sheet.paste(light, (i * w, top))
            rects.append([i * w, top, w, h])
        index[sprite] = {"anchor": s["anchor"], "frames": rects}
        if any(light for _, light in frames):
            index[sprite]["light"] = True
            any_light = True
        if "ms" in s:
            assert len(s["ms"]) == len(frames), f"{sprite}: ms and frames differ"
            index[sprite]["ms"] = s["ms"]
        top += h
    out = ROOT / "assets/graphics" / name
    out.parent.mkdir(parents=True, exist_ok=True)
    sheet.save(out.with_suffix(".png"))
    if any_light:
        light_sheet.save(out.parent / f"{out.name}_light.png")
    out.with_suffix(".json").write_text(json.dumps(index, indent=1) + "\n")


for source, name in SOURCES.items():
    build(source, name)
