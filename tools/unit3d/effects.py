"""Translucent 2D effect layers for unit3d attack effects. Requires numpy and Pillow.

Shapes are rasterized at SS x and kept where coverage is high enough, so effect
pixels stay crisp; each band gets a fixed palette color and alpha (translucent but
not anti-aliased). draw() returns (under, over) RGBA layers to composite behind
and in front of the sprite.
"""
import math
import numpy as np
from PIL import Image, ImageDraw

SS = 4
COLOR = {'p': (73, 173, 240), 'q': (140, 213, 255), 'r': (212, 243, 255)}
# (color, alpha) per band, from the body of a shape to its brightest edge.
ARC = [('p', 90), ('q', 150), ('r', 215)]
LINES = ('r', 170)
CONE = [('q', 120), ('r', 190)]
RING = ('r', 205)
WEDGE = [('q', 110), ('r', 170)]
GHOST = 'r'


class Layer:
    def __init__(self, cell):
        self.cell = cell
        self.rgba = np.zeros((cell[1], cell[0], 4), np.uint8)

    def mask(self, paint):
        im = Image.new('L', (self.cell[0]*SS, self.cell[1]*SS))
        paint(ImageDraw.Draw(im), SS)
        return np.array(im.resize(self.cell, Image.BOX)) >= 110

    def fill(self, mask, key, alpha):
        self.rgba[mask] = COLOR[key] + (int(alpha),)

    def image(self):
        return Image.fromarray(self.rgba, 'RGBA')


def scaled(points, s):
    return [(x*s, y*s) for x, y in points]


def arc(layer, fx, to_px):
    pts = [(to_px(a), to_px(b)) for a, b in fx['pairs']]
    for frac, (key, alpha) in zip((0, .45), ARC[:2]):
        def paint(dr, s, frac=frac):
            for (a0, b0), (a1, b1) in zip(pts, pts[1:]):
                i0 = (a0[0] + (b0[0]-a0[0])*frac, a0[1] + (b0[1]-a0[1])*frac)
                i1 = (a1[0] + (b1[0]-a1[0])*frac, a1[1] + (b1[1]-a1[1])*frac)
                dr.polygon(scaled([i0, b0, b1, i1], s), fill=255)
        layer.fill(layer.mask(paint), key, alpha*fx['alpha'])
    key, alpha = ARC[2]
    layer.fill(layer.mask(lambda dr, s: dr.line(scaled([b for _, b in pts], s), fill=255, width=s)), key,
               alpha*fx['alpha'])


def lines(layer, fx, to_px, key=LINES[0], alpha=LINES[1]):
    def paint(dr, s):
        for a, b in fx['segments']:
            dr.line(scaled([to_px(a), to_px(b)], s), fill=255, width=s)
    layer.fill(layer.mask(paint), key, alpha*fx.get('alpha', 1))


def cone(layer, fx, to_px):
    tip, apex = np.array(to_px(fx['tip'])), np.array(to_px(fx['tip'] + fx['dir']*fx['length']))
    axis = apex - tip
    length = np.linalg.norm(axis)
    if length < 1e-6:
        return
    normal = np.array([-axis[1], axis[0]]) / length
    half = fx['width'] / fx['length'] * length / 2
    for scale, (key, alpha) in zip((1, .55), CONE):
        base, end = tip, tip + axis*scale
        poly = [tuple(base + normal*half*scale), tuple(end), tuple(base - normal*half*scale)]
        layer.fill(layer.mask(lambda dr, s, poly=poly: dr.polygon(scaled(poly, s), fill=255)), key,
                   alpha*fx['alpha'])


def ring(layer, fx, to_px):
    d = fx['axis']
    u = np.cross(d, [0, 0, 1]) if abs(d[2]) < .9 else np.cross(d, [1, 0, 0])
    u /= np.linalg.norm(u)
    v = np.cross(d, u)
    pts = [to_px(fx['center'] + fx['radius']*(u*math.cos(t) + v*math.sin(t)))
           for t in np.linspace(0, 2*math.pi, 25)]
    key, alpha = RING
    layer.fill(layer.mask(lambda dr, s: dr.line(scaled(pts, s), fill=255, width=s)), key, alpha*fx['alpha'])


def wedge(layer, fx, to_px):
    a, b = np.array(to_px(fx['start'])), np.array(to_px(fx['end']))
    axis = b - a
    length = np.linalg.norm(axis)
    if length < 1e-6:
        return
    normal = np.array([-axis[1], axis[0]]) / length
    half = max(1.5, fx['width']*length / (np.linalg.norm(fx['end'] - fx['start']) + 1e-6) / 2)
    key, alpha = WEDGE[0]
    poly = [tuple(a), tuple(b + normal*half), tuple(b - normal*half)]
    layer.fill(layer.mask(lambda dr, s: dr.polygon(scaled(poly, s), fill=255)), key, alpha*fx['alpha'])
    key, alpha = WEDGE[1]
    layer.fill(layer.mask(lambda dr, s: dr.line(scaled([tuple(a), tuple(b)], s), fill=255, width=s)), key,
               alpha*fx['alpha'])


def ghost(layer, fx, to_px):
    for (a, b), alpha in zip(fx['segments'], fx['alphas']):
        lines(layer, dict(segments=[(a, b)]), to_px, GHOST, 255*alpha)


DRAW = dict(arc=arc, lines=lines, cone=cone, ring=ring, wedge=wedge, ghost=ghost)


def draw(effects, to_px, cell):
    layers = {'under': Layer(cell), 'over': Layer(cell)}
    for fx in effects:
        DRAW[fx['kind']](layers[fx['layer']], fx, to_px)
    return layers['under'].image(), layers['over'].image()
