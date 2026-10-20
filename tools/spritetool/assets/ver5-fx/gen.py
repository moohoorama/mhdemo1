"""Soft layers the effect builder draws by code (straight alpha), one function per layer name."""
import math
from pathlib import Path

from PIL import Image


def water_mist(i, n, w, h):
    """A low spray of fine drops hanging over the base: swells in the first half, thins out after."""
    k = (i + .5) / n
    im = Image.new("RGBA", (w, h))
    px = im.load()
    cx, cy = w / 2, h - 8  # the anchor
    rx, ry = 6 + 14 * k, 3 + 4 * k
    top = .85 * math.sin(math.pi * min(1, k * 1.3))
    for y in range(h):
        for x in range(w):
            d = math.hypot((x + .5 - cx) / rx, (y + .5 - cy + ry * .6) / ry)
            if d < 1:
                a = top * (1 - d) ** 1.5
                px[x, y] = (225, 242, 255, round(255 * a))
    return im


FLAMES = Path(__file__).resolve().parents[4] / "ver5/assets/graphics/effects"


def _strip(name, w):
    im = Image.open(FLAMES / f"{name}.png").convert("RGBA")
    return [im.crop((i * w, 0, (i + 1) * w, im.height)) for i in range(im.width // w)]


def fire_tongues(i, n, w, h):
    """Eight flame tongues (the burning-map strips) round the target's feet in an iso ellipse: small
    while the fire catches and dies, full height in between with the scene's big tongues in the middle."""
    k = (i + .5) / n
    small, mid, big = _strip("flame_t_small", 7), _strip("flame_t", 10), _strip("flame", 14)
    im = Image.new("RGBA", (w, h))
    cx, cy = w / 2, h - 10  # the anchor
    # (dx, dy, central): back ones first so the front ones cover them
    spots = [(-11, -3, False), (10, -3, False), (-4, -5, True), (4, -4, True), (-14, 1, False),
             (13, 1, False), (0, -1, True), (-7, 4, False), (7, 4, False)]
    for j, (dx, dy, central) in enumerate(sorted(spots, key=lambda s: s[1])):
        if k < .15 or k > .85:
            strip = small
        elif .3 < k < .75:
            strip = big if central else mid
        else:
            strip = mid
        t = strip[(i + j * 2) % len(strip)]
        im.alpha_composite(t, (round(cx + dx - t.width / 2), round(cy + dy - t.height + 2)))
    return im


def fire_glow(i, n, w, h):
    """A soft orange heat glow over the fire that swells and fades."""
    k = (i + .5) / n
    im = Image.new("RGBA", (w, h))
    px = im.load()
    cx, cy, r = w / 2, h - 20, 15 + 9 * k
    for y in range(h):
        for x in range(w):
            d = math.hypot(x + .5 - cx, (y + .5 - cy) * 1.15) / r
            if d < 1:
                px[x, y] = (255, 150, 50, round(255 * .8 * (1 - d) ** 1.2))
    return im


def fire_flash(i, n, w, h):
    """The burst as the fire strikes: a white-hot core that blooms and is gone within a few frames."""
    k = (i + .5) / n
    im = Image.new("RGBA", (w, h))
    if k > .35:
        return im
    px = im.load()
    cx, cy, r = w / 2, h - 16, 10 + 40 * k
    top = 1 - k / .35
    for y in range(h):
        for x in range(w):
            d = math.hypot(x + .5 - cx, (y + .5 - cy) * 1.3) / r
            if d < 1:
                px[x, y] = (255, 240, 190, round(255 * top * (1 - d) ** .8))
    return im


def _glow(w, h, cx, cy, r, colour, top, squash=1.0):
    """A soft disc fading from top opacity at its centre to nothing at radius r."""
    im = Image.new("RGBA", (w, h))
    if r <= 0 or top <= 0:
        return im
    px = im.load()
    for y in range(h):
        for x in range(w):
            d = math.hypot(x + .5 - cx, (y + .5 - cy) * squash) / r
            if d < 1:
                px[x, y] = colour + (round(255 * min(1, top) * (1 - d) ** 1.2),)
    return im


def slash_glow(i, n, w, h):
    k = (i + .5) / n
    return _glow(w, h, w / 2, h / 2, 6 + k * 18, (255, 248, 220), .6 * (1 - k))


def critical_glow(i, n, w, h):
    k = (i + .5) / n
    return _glow(w, h, w / 2, h / 2, 12 + k * 90, (255, 248, 220), 1.4 * (1 - k / .35) if k < .35 else 0)


def block_flash(i, n, w, h):
    """The flash where the blow meets the shield (the shield faces right)."""
    k = (i + .5) / n
    ax, ay = 20, 24  # the anchor; keep in step with settings.json
    pop = .6 + k / .12 * .5 if k < .12 else 1.1 - (k - .12) / .1 * .1 if k < .22 else 1
    return _glow(w, h, ax + 10 + 7 * pop * .4, ay, 4 + k * 30, (255, 248, 220), 1 - k / .3 if k < .3 else 0)


def campfire_glow(i, n, w, h):
    return _glow(w, h, w / 2, h / 2, 22, (255, 160, 50), 1)
