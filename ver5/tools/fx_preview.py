#!/usr/bin/env python3
"""Contact sheets, GIFs and checks for the hand-pixel effect sprites (the isofx skill's eyes).

    uv run --with pillow python tools/fx_preview.py heal water          # reports/fx/<name>.png
    uv run --with pillow python tools/fx_preview.py heal --gif --dur 1.1 # + reports/fx/<name>.gif
    uv run --with pillow python tools/fx_preview.py --check              # every sprite of the source
    uv run --with pillow python tools/fx_preview.py 'promote*' --source ver5-fx

A contact sheet has one row per ground (grass, sand, night) and one column per frame, each frame drawn
the way the game draws it (the normal sheet, then the light sheet added on top) at --scale times, over
a faint 64x32 battle cell diamond whose centre is the anchor. Sprites only need to be in settings.json's
"sprites" to be previewed, so drafts can stay out of "frame_order" (and out of the built sheet).
"""
import argparse
import fnmatch
import sys
from pathlib import Path

from PIL import Image, ImageDraw

sys.dont_write_bytecode = True
sys.path.insert(0, str(Path(__file__).resolve().parent))
from build_effects import ROOT, load_source, sprite_frames  # noqa: E402

GROUNDS = (("grass", (86, 118, 60)), ("sand", (184, 156, 104)), ("night", (24, 28, 38)))
CELL_W, CELL_H = 64, 32  # a battle cell: 2x2 small 32x16 tiles
PAD = 8


def composite(bg, normal, light):
    """normal over a flat ground, then light added the way the game's additive pass does."""
    out = Image.new("RGBA", normal.size, bg + (255,))
    out.alpha_composite(normal)
    if light is None:
        return out
    px, lp = out.load(), light.load()
    for y in range(out.height):
        for x in range(out.width):
            r, g, b, a = lp[x, y]
            if a:
                k = a / 255
                o = px[x, y]
                px[x, y] = (min(255, o[0] + round(r * k)), min(255, o[1] + round(g * k)), min(255, o[2] + round(b * k)), 255)
    return out


def guide(draw, ax, ay, scale, ox, oy, colour):
    pts = [(ax, ay - CELL_H / 2), (ax + CELL_W / 2, ay), (ax, ay + CELL_H / 2), (ax - CELL_W / 2, ay)]
    pts = [(ox + x * scale, oy + y * scale) for x, y in pts]
    draw.line(pts + pts[:1], fill=colour, width=1)


def contact(s, frames, scale):
    w, h = s["cell"]
    ax, ay = s["anchor"]
    fw, fh = w * scale + PAD, h * scale + PAD
    sheet = Image.new("RGBA", (PAD + fw * len(frames), PAD + 12 + fh * len(GROUNDS)), (40, 40, 40, 255))
    d = ImageDraw.Draw(sheet)
    for i in range(len(frames)):
        d.text((PAD + i * fw, 2), str(i), fill=(220, 220, 220))
    for row, (_, bg) in enumerate(GROUNDS):
        for i, (normal, light) in enumerate(frames):
            ox, oy = PAD + i * fw, PAD + 12 + row * fh
            tile = Image.new("RGBA", (w, h), bg + (255,))
            g = Image.new("RGBA", (w * scale, h * scale), (0, 0, 0, 0))
            guide(ImageDraw.Draw(g), ax, ay, scale, 0, 0, (255, 255, 255, 60))
            frame = composite(bg, normal, light).resize((w * scale, h * scale), Image.NEAREST)
            tile = tile.resize((w * scale, h * scale), Image.NEAREST)
            tile.alpha_composite(g)
            # the effect over the guide: keep only pixels the effect changed
            mask = Image.new("L", frame.size)
            mask.putdata([0 if p[:3] == bg else 255 for p in frame.getdata()])
            tile.paste(frame, (0, 0), mask)
            sheet.paste(tile, (ox, oy))
            d.rectangle([ox + ax * scale, oy + ay * scale, ox + ax * scale + scale - 1, oy + ay * scale + scale - 1],
                        fill=(255, 40, 40))
    return sheet


def gif(s, frames, scale, dur, path):
    bg = GROUNDS[0][1]
    w, h = s["cell"]
    ims = [composite(bg, n, l).resize((w * scale, h * scale), Image.NEAREST).convert("RGB") for n, l in frames]
    ms = s.get("ms") or [round(dur * 1000 / len(ims))] * len(ims)
    ims[0].save(path, save_all=True, append_images=ims[1:], duration=ms, loop=0)


def check(name, s, settings, pixels, frames):
    """Problems a picture won't show at a glance."""
    out = []
    w, h = s["cell"]
    k = s.get("dot", 1)
    layers = s.get("layers") or {"main": {"palette": s.get("palette", {})}}
    n = len(frames)
    if n != s["frames"]:
        out.append(f"{n} frames drawn, settings say {s['frames']}")
    if "ms" in s and len(s["ms"]) != n:
        out.append(f"ms has {len(s['ms'])} entries for {n} frames")
    for lname, layer in layers.items():
        a = layer.get("alpha")
        if isinstance(a, list) and len(a) != n:
            out.append(f"layer {lname}: alpha has {len(a)} entries for {n} frames")
    for i, frame in enumerate(pixels.get(name) or []):
        for lname, layer in layers.items():
            if "gen" in layer:
                continue
            grid = frame if isinstance(frame, list) else frame.get(lname)
            if grid is None:
                out.append(f"frame {i}: no grid for layer {lname}")
                continue
            if len(grid) != h // k or any(len(r) != w // k for r in grid):
                out.append(f"frame {i} {lname}: grid is not {w // k}x{h // k}")
                continue
            known = set(layer["palette"]) | {".", "o"}
            if bad := {c for r in grid for c in r} - known:
                out.append(f"frame {i} {lname}: letters with no colour {''.join(sorted(bad))}")
    if w > 8 and h > 8:
        sides, hit = {"top": [], "bottom": [], "left": [], "right": []}, set()
        for i, pair in enumerate(frames):
            for im in pair:
                if im is None:
                    continue
                a = im.getchannel("A")
                for side, pts in (("top", [(x, 0) for x in range(w)]), ("bottom", [(x, h - 1) for x in range(w)]),
                                  ("left", [(0, y) for y in range(h)]), ("right", [(w - 1, y) for y in range(h)])):
                    if (side, i) not in hit and max(a.getpixel(p) for p in pts) > 40:
                        sides[side].append(i)
                        hit.add((side, i))
        for side, idx in sides.items():
            if idx:
                out.append(f"touches the {side} edge in frames {','.join(map(str, idx))} (clipped? grow the cell)")
    if n > 1 and "played once" in s.get("timing", ""):
        normal, light = frames[-1]
        peak = max(normal.getchannel("A").getextrema()[1], light.getchannel("A").getextrema()[1] if light else 0)
        if peak > 128:
            out.append("a one-shot whose last frame is still strong (alpha %d): fade, shrink or dither it out" % peak)
    return out


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("names", nargs="*", help="sprite names or globs (default: all with --check)")
    ap.add_argument("--source", default="ver5-fx")
    ap.add_argument("--scale", type=int, default=4)
    ap.add_argument("--gif", action="store_true")
    ap.add_argument("--dur", type=float, default=1.0, help="GIF length in seconds when the sprite has no ms")
    ap.add_argument("--check", action="store_true")
    ap.add_argument("--out", default=str(ROOT / "reports/fx"))
    args = ap.parse_args()

    settings, pixels, gen = load_source(args.source)
    every = list(settings["sprites"])
    names = [n for pat in (args.names or (every if args.check else [])) for n in fnmatch.filter(every, pat)]
    if not names:
        sys.exit("no sprite matches")
    out = Path(args.out)
    out.mkdir(parents=True, exist_ok=True)
    problems = 0
    for name in names:
        s = settings["sprites"][name]
        frames = sprite_frames(settings, pixels, gen, name)
        if args.check:
            for p in check(name, s, settings, pixels, frames):
                print(f"{name}: {p}")
                problems += 1
            if name not in settings["frame_order"]:
                print(f"{name}: not in frame_order (a draft? it won't be built)")
        if args.names:
            contact(s, frames, args.scale).save(out / f"{name}.png")
            print(out / f"{name}.png")
            if args.gif:
                gif(s, frames, args.scale, args.dur, out / f"{name}.gif")
                print(out / f"{name}.gif")
    if args.check:
        print("clean" if not problems else f"{problems} problems")


if __name__ == "__main__":
    main()
