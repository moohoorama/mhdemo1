"""Cel-shaded native-resolution renderer for unit3d meshes. Requires numpy and Pillow.

Faces are rasterized at SS x resolution with a per-pixel plane z-buffer. Shading is
toon style: one key light from the viewer's upper left (camera space, so every
direction is lit alike), hard bands shadow / base / light / specular, and cast
shadows from a light-space depth map. Each output pixel takes the most frequent
(material, tone) sample and maps to a knight v6 palette ramp, so no new colors
appear. Lines: 1px outline around the silhouette and on the far side of every
occluding edge.
"""
import json
import math
from pathlib import Path
import numpy as np
from PIL import Image

ROOT = Path(__file__).resolve().parents[2]
PALETTE = json.loads((ROOT / 'tools/spritetool/assets/knights-pixel-v6/frames.json').read_text())['palette']
KEYS = '0123456789abcdefghijklmnopqrstuvwxyz'
COLOR = {KEYS[i]: tuple(int(h[j:j+2], 16) for j in (1, 3, 5)) for i, h in enumerate(PALETTE)}
# Additions to the knight palette: hemp cloth, green robe, ruddy face, team keys.
# W X Y Z are the team key ramp (factions.TEAM_KEYS), recolored per faction at runtime.
EXTRA = {'S': '#d9c698', 'T': '#a89266', 'U': '#6c583c', 'G': '#183a24', 'H': '#2a5e35', 'I': '#3f8a4a',
         'J': '#79b866', 'K': '#8e3326', 'L': '#c0503a', 'M': '#df7b5c',
         'W': '#1b3358', 'X': '#264d80', 'Y': '#3567a6', 'Z': '#5a8fd0',
         'A': '#963e5c', 'B': '#d77790', 'C': '#f2a7b8', 'D': '#ffd6de',  # A-D: peach blossom pink (props)
         'N': '#22301a', 'O': '#405826', 'P': '#688630', 'Q': '#98b242', 'R': '#ccd668'}  # N-R: olive foliage (map forest)
COLOR.update({k: tuple(int(h[j:j+2], 16) for j in (1, 3, 5)) for k, h in EXTRA.items()})

DIRECTIONS = ['N', 'NE', 'E', 'SE', 'S', 'SW', 'W', 'NW']
AZIMUTH = {'N': math.pi/2, 'NE': math.pi*.75, 'E': math.pi, 'SE': -math.pi*.75,
           'S': -math.pi/2, 'SW': -math.pi/4, 'W': 0, 'NW': math.pi/4}
ELEVATION = .48
LIGHT = (-.6, .75, .45)  # camera space (right, up, toward viewer)
SHADOW_BIAS = .12        # model units
CONTOUR = .3             # depth gap drawn as an inner line

# Material color -> palette keys: shadow, base, light, specular.
RAMPS = {
    '#28486e': 'WXYZ',   # team cloth (faction key ramp)
    '#3974a0': 'XYZZ',   # team sash, light
    '#636971': '2345',   # steel
    '#b91828': 'bccd',   # red cloth
    '#754323': 'fghh',   # leather
    '#efb15c': 'ijjj',   # skin
    '#30343a': '0112',   # dark steel
    '#211f22': '0000',   # eye slit
    '#d5a123': 'kllm',   # gold
    '#bad9f1': '4566',   # blade
    '#e5f4ff': '5666',   # blade edge
    '#825032': 'ghii',   # horse coat
    '#9a6845': 'hiij',   # horse muzzle
    '#352820': '0ffg',   # mane, hooves
    '#a91e2c': 'WWXY',   # team saddle cloth, kept dark
    '#38261d': '0fff',   # reins
    '#15191b': '0000',   # horse eye
    '#966131': 'fghh',   # spear shaft
    '#d6e7ec': '4566',   # spear tip
}


def add_ramps(ramps):
    RAMPS.update(ramps)


def nearest_key(hex_color):
    rgb = np.array([int(hex_color[i:i+2], 16) for i in (1, 3, 5)])
    return min(COLOR, key=lambda k: ((np.array(COLOR[k]) - rgb) ** 2).sum())


def view(az):
    t = np.array([math.cos(az)*math.cos(ELEVATION), math.sin(az)*math.cos(ELEVATION), math.sin(ELEVATION)])
    r = np.array([-math.sin(az), math.cos(az), 0.])
    return t, r, np.cross(t, r)


def project(point, direction, scale, pivot):
    """Pixel position of a model-space point in the output cell."""
    t, r, u = view(AZIMUTH[direction])
    p = np.asarray(point, float)
    return pivot[0] + .5 + p @ r * scale, pivot[1] + 1 - p @ u * scale


def prepare(faces):
    out = []
    for verts, color in faces:
        v = np.asarray(verts, float)
        n = np.cross(v[1]-v[0], v[2]-v[0])
        ln = np.linalg.norm(n)
        if ln > 1e-9:
            out.append((v, n/ln, color))
    return out


def rasterize(faces, t, r, u, s, origin, size, attr=None):
    """Plane z-buffer; returns depth (toward t) and the winning face index per sample."""
    W, H = size
    zbuf = np.full((H, W), -np.inf)
    ids = np.full((H, W), -1, np.int32)
    ys, xs = np.mgrid[0:H, 0:W]
    sx, sy = (xs + .5 - origin[0]) / s, (origin[1] - ys - .5) / s
    for fi, (v, n, _) in enumerate(faces):
        nt = np.dot(n, t)
        if abs(nt) < 1e-6:
            continue
        qx, qy = v @ r * s + origin[0], origin[1] - v @ u * s
        x0, x1 = max(0, int(qx.min())), min(W, int(math.ceil(qx.max())) + 1)
        y0, y1 = max(0, int(qy.min())), min(H, int(math.ceil(qy.max())) + 1)
        if x0 >= x1 or y0 >= y1:
            continue
        bx, by = xs[y0:y1, x0:x1] + .5, ys[y0:y1, x0:x1] + .5
        k = len(v)
        area = sum(qx[i]*qy[(i+1) % k] - qx[(i+1) % k]*qy[i] for i in range(k))
        sign = 1 if area > 0 else -1
        inside = np.ones(bx.shape, bool)
        for i in range(k):
            ax, ay, cx, cy = qx[i], qy[i], qx[(i+1) % k], qy[(i+1) % k]
            inside &= sign*((cx-ax)*(by-ay) - (cy-ay)*(bx-ax)) >= 0
        if not inside.any():
            continue
        depth = (np.dot(n, v[0]) - sx[y0:y1, x0:x1]*np.dot(n, r) - sy[y0:y1, x0:x1]*np.dot(n, u)) / nt
        zb = zbuf[y0:y1, x0:x1]
        win = inside & (depth > zb)
        zb[win] = depth[win]
        ids[y0:y1, x0:x1][win] = fi
    return zbuf, ids, sx, sy


def light_basis(light):
    up = np.array([0, 0, 1.]) if abs(light[2]) < .95 else np.array([1., 0, 0])
    r = np.cross(up, light)
    r /= np.linalg.norm(r)
    return light, r, np.cross(light, r)


def render(faces, direction, scale, cell=(48, 48), pivot=(24, 38), ss=4, depths=None):
    """Return a cell-sized grid of palette keys ('.' transparent).
    depths: if a list, the per-pixel depth toward the viewer (model units, NaN where empty) is appended."""
    faces = prepare(faces)
    t, r, u = view(AZIMUTH[direction])
    light = LIGHT[0]*r + LIGHT[1]*u + LIGHT[2]*t
    light /= np.linalg.norm(light)
    W, H = cell[0]*ss, cell[1]*ss
    s = scale*ss
    zbuf, ids, sx, sy = rasterize(faces, t, r, u, s, ((pivot[0]+.5)*ss, (pivot[1]+1)*ss), (W, H))

    # Cast shadows: light-space depth map over the model bounds.
    lt, lr, lu = light_basis(light)
    pts = np.concatenate([v for v, _, _ in faces])
    px, py = pts @ lr, pts @ lu
    pad = 2
    origin = (-(px.min())*s + pad, py.max()*s + pad)
    size = (int((px.max() - px.min())*s) + 2*pad + 1, int((py.max() - py.min())*s) + 2*pad + 1)
    lz, _, _, _ = rasterize(faces, lt, lr, lu, s, origin, size)

    hit = ids >= 0
    world = sx[..., None]*r + sy[..., None]*u + np.where(hit, zbuf, 0)[..., None]*t
    lx = np.clip((world @ lr*s + origin[0]).astype(int), 0, size[0]-1)
    ly = np.clip((origin[1] - world @ lu*s).astype(int), 0, size[1]-1)
    shadowed = hit & (lz[ly, lx] - world @ lt > SHADOW_BIAS)

    tone_of = []
    for v, n, color in faces:
        nn = n if np.dot(n, t) > 0 else -n
        lit = np.dot(nn, light)
        tone_of.append(0 if lit < 0 else 1 if lit < .6 else 2 if lit < .92 else 3)
    tone = np.where(hit, np.array(tone_of + [0])[ids], -1)
    tone[shadowed] = 0
    material = {c: i for i, c in enumerate(sorted({c for _, _, c in faces}))}
    names = list(material)
    mat = np.where(hit, np.array([material[c] for _, _, c in faces] + [0])[ids], -1)
    key = np.where(hit, mat*4 + tone, -1)

    out = [['.'] * cell[0] for _ in range(cell[1])]
    depth = np.full((cell[1], cell[0]), np.nan)
    blocks = hit.reshape(cell[1], ss, cell[0], ss).any(axis=(1, 3))
    rows, cols = np.nonzero(blocks)
    if depths is not None:
        depths.append(depth)
    if not len(rows):
        return out
    box = (max(0, cols.min() - 1), max(0, rows.min() - 1), min(cell[0], cols.max() + 2), min(cell[1], rows.max() + 2))
    for y in range(box[1], box[3]):
        for x in range(box[0], box[2]):
            block = key[y*ss:(y+1)*ss, x*ss:(x+1)*ss].ravel()
            got = block[block >= 0]
            if len(got)*2 < len(block):
                continue
            best = np.bincount(got).argmax()
            color = names[best // 4]
            ramp = RAMPS.get(color) or nearest_key(color)*4
            out[y][x] = ramp[best % 4]
            depth[y, x] = zbuf[y*ss:(y+1)*ss, x*ss:(x+1)*ss].ravel()[block == best].mean()
    return outline(contour(out, depth, box), box)


N4 = ((1, 0), (-1, 0), (0, 1), (0, -1))


def contour(g, depth, box):
    """Draw a line on pixels lying just behind an occluding edge."""
    h, w = len(g), len(g[0])
    out = [row[:] for row in g]
    for y in range(box[1], box[3]):
        for x in range(box[0], box[2]):
            if g[y][x] == '.':
                continue
            for dy, dx in N4:
                ny, nx = y+dy, x+dx
                if 0 <= ny < h and 0 <= nx < w and g[ny][nx] != '.' and depth[ny, nx] - depth[y, x] > CONTOUR:
                    out[y][x] = '0'
                    break
    return out


def outline(g, box):
    h, w = len(g), len(g[0])
    out = [row[:] for row in g]
    for y in range(box[1], box[3]):
        for x in range(box[0], box[2]):
            if g[y][x] == '.' and any(0 <= y+dy < h and 0 <= x+dx < w and g[y+dy][x+dx] != '.' for dy, dx in N4):
                out[y][x] = '0'
    return out


def to_image(g):
    im = Image.new('RGBA', (len(g[0]), len(g)))
    for y, row in enumerate(g):
        for x, c in enumerate(row):
            if c != '.':
                im.putpixel((x, y), COLOR[c] + (255,))
    return im
