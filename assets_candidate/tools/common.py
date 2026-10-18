"""Shared helpers: read the game's graphics index and cut sprites out of its sheets."""
import json, os
from PIL import Image

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
GAME = os.path.join(os.path.dirname(ROOT), "ver5", "assets", "graphics")
INDEX = json.load(open(os.path.join(GAME, "index.json")))
_sheets = {}


def sheet(name):
    if name not in _sheets:
        _sheets[name] = Image.open(os.path.join(GAME, "map", name + ".png")).convert("RGBA")
    return _sheets[name]


def prop(name):
    """(image, pivot) of a sprite from props.png."""
    s = INDEX["props"]["sprites"][name]
    x, y, w, h = s["rect"]
    return sheet("props").crop((x, y, x + w, y + h)), tuple(s["pivot"])


def structure(name):
    """(image, pivot) of a cell from structures.png."""
    i = INDEX["structures"]["sprites"][name]
    w, h = INDEX["structures"]["cell"]
    return sheet("structures").crop((i * w, 0, i * w + w, h)), tuple(INDEX["structures"]["pivot"])


def site(name):
    i = INDEX["sites"]["sprites"][name]
    w, h = INDEX["sites"]["cell"]
    return sheet("sites").crop((i * w, 0, i * w + w, h)), tuple(INDEX["sites"]["pivot"])


def tile(i):
    x, y, w, h, px, py = INDEX["tileset"]["sprites"][str(i)]
    return sheet("tileset").crop((x, y, x + w, y + h)), (px, py)


def out(*p):
    path = os.path.join(ROOT, *p)
    os.makedirs(os.path.dirname(path), exist_ok=True)
    return path
