"""Soft UI layers the builder draws by code (straight alpha), one function per layer name."""
from PIL import Image


def window_glow(i, n, w, h):
    """The window's top-lit wash: eight bands fading downward (the game stretches it over the
    upper half of the window's body)."""
    im = Image.new("RGBA", (w, h))
    for y in range(h):
        for x in range(w):
            im.putpixel((x, y), (255, 210, 160, round(14 - y * 1.7)))
    return im


def scroll_curl(i, n, w, h):
    """The paper darkening as it curls toward the left roller, one column per art pixel (the game
    mirrors it for the right)."""
    im = Image.new("RGBA", (w, h))
    for y in range(h):
        for x in range(w):
            im.putpixel((x, y), (110, 80, 35, round(255 * (.28 - x * .045))))
    return im
