#!/usr/bin/env python3
"""Seal spell drafts (봉책, 봉쇄, 봉격) on spellsim's world: talismans (yellowed paper plates that tumble and
flutter, red squiggles on them, no characters) and chains (Verlet links held at their lengths, wrapping round the
target's body), drawn straight in screen space; dust from spellsim's Earth.

    uv run --with numpy --with scipy --with pillow python tools/spellsim_seal.py [names]   # reports/spellsim/

The target is a cylinder (radius 9, 40 tall) at the origin. Anything behind its centre is hidden where the
target's sprite stands, so chains and talismans pass behind the body.
"""
import argparse
import math
from pathlib import Path

import numpy as np

import spellsim as S
from spellsim import FPS, G, ORIGIN, SIZE, SUB, screen

PAPER = {"hi": (240, 226, 178), "mid": (214, 192, 138), "lo": (176, 152, 100), "ink": (198, 38, 30),
         "edge": (92, 68, 40)}
IRON = {"l": (196, 200, 206), "m": (128, 132, 142), "d": (78, 80, 90), "o": (22, 22, 30)}
GLOW = (236, 66, 44)
BODY_R, BODY_H = 9.0, 40.0


def unit_mask():
    """Where the target's sprite covers the canvas (as spellsim.compose draws it)."""
    img, foot = S.sheet_unit()(1)
    a = np.array(img)[:, :, 3] > 0
    m = np.zeros((SIZE[1], SIZE[0]), bool)
    x0, y0 = int(ORIGIN[0] - img.width / 2), int(ORIGIN[1] - foot)
    m[y0:y0 + a.shape[0], x0:x0 + a.shape[1]] = a
    return m


MASK = unit_mask()


def put(layer, x, y, col, behind, alpha=255):
    x, y = int(round(x)), int(round(y))
    if 0 <= x < SIZE[0] and 0 <= y < SIZE[1] and not (behind and MASK[y, x]):
        layer[y, x] = (*col, alpha)


def outline(layer, col, alpha=255):
    a = layer[:, :, 3] > 0
    e = S.ring(a)
    layer[e] = (*col, alpha)


def gone(k, x, y, seed=0):
    """A dither that eats a drawing away as k runs past 0.7."""
    if k <= .7:
        return False
    return ((x * 73 + y * 151 + seed * 37) % 97) / 97 < (k - .7) / .28


# --- talismans

def paper_texel(s, t, front):
    """Colour of the plate at (s, t) in -1..1 (t up): a red border band at the top, a red squiggle down the
    middle, a short bar under the band; the back is plain."""
    if not front:
        return "lo"
    ty = (1 - t) * 5          # 0 top .. 10 bottom
    tx = (s + 1) * 3          # 0 .. 6
    if ty < 1.3:
        return "ink"
    if 2.2 < ty < 3.0 and 1 < tx < 5:
        return "ink"
    if ty > 3 and abs(tx - 3 - 1.1 * math.sin(ty * 1.4)) < .8:
        return "ink"
    return "hi" if s < 0 else "mid"


class Talisman:
    def __init__(self, path, t0, stick_t, stick, spin, size=(2.2, 3.6)):
        """path(tau) -> position while flying; at stick_t it slaps onto stick (x, y, h) facing outward."""
        self.path, self.t0, self.stick_t, self.stick, self.spin, self.size = path, t0, stick_t, np.array(stick, float), spin, size
        self.p = None

    def frame(self, t):
        """Centre, width axis, height axis at time t (None before it appears)."""
        tau = t - self.t0
        if tau < 0:
            return None
        if self.stick_t is not None and t >= self.stick_t:
            c = self.stick
            out = np.array([c[0], c[1], 0.0])
            n = out / (np.linalg.norm(out) or 1)
            a = np.cross([0, 0, 1], n)
            a /= np.linalg.norm(a) or 1
            # a slap: a little lean that settles
            k = min(1, (t - self.stick_t) / .12)
            lean = (1 - k) * .5
            b = np.array([0, 0, 1.0]) * math.cos(lean) + n * math.sin(lean)
            return c, a, b
        c = self.path(tau)
        if self.stick_t is not None:  # swoop into the stick point over the last 0.12 s
            k = 1 - min(1, (self.stick_t - t) / .12)
            c = c * (1 - k ** 2) + self.stick * k ** 2
        a0 = np.array([math.cos(tau * self.spin), math.sin(tau * self.spin), 0.0])
        flutter = .6 * math.sin(tau * 17 + self.spin)
        b = np.array([0, 0, 1.0]) * math.cos(flutter) + np.cross(a0, [0, 0, 1]) * math.sin(flutter)
        return c, a0, b

    def draw(self, layer, t, k, seed):
        f = self.frame(t)
        if f is None:
            return
        c, a, b = f
        w, hh = self.size
        cx, cy = screen(c[None])
        lin = lambda v: np.array([v[0] - v[1], (v[0] + v[1]) / 2 - v[2]])
        A, B = lin(a) * w, lin(b) * hh
        M = np.array([A, B]).T
        if abs(np.linalg.det(M)) < .5:  # edge-on: a thin line
            for j in range(-2, 3):
                put(layer, cx[0] + B[0] * j / 2, cy[0] + B[1] * j / 2, PAPER["mid"], c[0] + c[1] < 0)
            return
        inv = np.linalg.inv(M)
        n = np.cross(a, b)
        front = n @ S.VIEW > 0
        r = int(abs(A).max() + abs(B).max()) + 1
        behind = c[0] + c[1] < -2
        for py in range(int(cy[0]) - r, int(cy[0]) + r + 1):
            for px in range(int(cx[0]) - r, int(cx[0]) + r + 1):
                s, tt = inv @ np.array([px + .5 - cx[0], py + .5 - cy[0]])
                if abs(s) <= 1 and abs(tt) <= 1 and not gone(k, px, py, seed):
                    put(layer, px, py, PAPER[paper_texel(s, tt, front)], behind)


# --- chains

class Chain:
    """Links held at length L by Verlet constraints. Each end is pinned to a path (a function of t, or None to
    hang free); the body cylinder pushes links out, the ground holds them up."""

    def __init__(self, n, L, root, head, t0=0.0, root_at=None):
        self.n, self.L, self.root, self.head, self.t0 = n, L, root, head, t0
        start = np.array(root_at if root_at is not None else root(0.0), float)
        self.p = np.repeat(start[None], n, 0)
        self.prev = self.p.copy()
        self.drop = None   # time the chain lets go and falls apart

    def step(self, t, dt):
        if t < self.t0:
            return
        v = (self.p - self.prev) * .985
        self.prev = self.p.copy()
        self.p = self.p + v
        self.p[:, 2] -= G * dt * dt
        falling = self.drop is not None and t > self.drop
        for _ in range(12):
            if not falling:
                if self.root:
                    self.p[0] = self.root(t - self.t0)
                if self.head:
                    self.p[-1] = self.head(t - self.t0)
            d = self.p[1:] - self.p[:-1]
            r = np.maximum(np.linalg.norm(d, axis=1, keepdims=True), 1e-6)
            corr = (r - self.L) / r * d * .5
            self.p[:-1] += corr
            self.p[1:] -= corr
            rel = self.p[:, :2]
            dist = np.linalg.norm(rel, axis=1)
            inside = (dist < BODY_R + 1.2) & (self.p[:, 2] < BODY_H) & (self.p[:, 2] > -1)
            if inside.any() and not falling:
                self.p[inside, :2] = rel[inside] / np.maximum(dist[inside, None], 1e-6) * (BODY_R + 1.2)
            if not falling:  # only the two links at each end may run into the ground
                self.p[2:-2, 2] = np.maximum(self.p[2:-2, 2], .5)
                self.p[:, 2] = np.maximum(self.p[:, 2], -8)
            else:
                self.p[:, 2] = np.maximum(self.p[:, 2], 0)

    def draw(self, layer, t, k, seed):
        if t < self.t0:
            return
        for i in range(self.n - 1):
            a, b = self.p[i], self.p[i + 1]
            if a[2] < 0 and b[2] < 0:
                continue
            m = (a + b) / 2
            if m[2] < -.5:
                continue
            sx, sy = screen(m[None])
            ex, ey = screen(np.array([b]))[0] - screen(np.array([a]))[0], screen(np.array([b]))[1] - screen(np.array([a]))[1]
            horiz = abs(ex[0] if np.ndim(ex) else ex) >= abs(ey[0] if np.ndim(ey) else ey)
            behind = m[0] + m[1] < -3
            x, y = sx[0], sy[0]
            if gone(k, int(x), int(y), seed + i):
                continue
            if i % 2 == 0:  # a link seen open
                pts = ([(-2, 0, "l"), (-1, -1, "l"), (0, -1, "l"), (1, -1, "m"), (2, 0, "m"), (1, 1, "d"), (0, 1, "d"),
                        (-1, 1, "d")] if horiz else
                       [(0, -2, "l"), (-1, -1, "l"), (-1, 0, "l"), (-1, 1, "m"), (0, 2, "d"), (1, 1, "d"), (1, 0, "d"),
                        (1, -1, "m")])
            else:  # a link seen edge-on
                pts = [(-1, 0, "m"), (0, 0, "l"), (1, 0, "m")] if horiz else [(0, -1, "l"), (0, 0, "m"), (0, 1, "d")]
            for dx, dy, c in pts:
                put(layer, x + dx, y + dy, IRON[c], behind)


def stake(layer, at, depth_in, k, seed=0):
    """An iron spike driven into the ground, depth_in 0 (just touching) .. 1 (driven home)."""
    x, y = screen(np.array([[at[0], at[1], 0.0]]))
    top = 9 - 5 * depth_in
    for j in range(int(top) + 1):
        if not gone(k, int(x[0]), int(y[0] - j), seed):
            put(layer, x[0], y[0] - j, IRON["l"], False)
            put(layer, x[0] + 1, y[0] - j, IRON["d"], False)
    for dx in (-1, 0, 1, 2):
        put(layer, x[0] + dx, y[0] - top - 1, IRON["m"], False)


# --- glows

def ground_ring(img, r, alpha, col=GLOW, thick=1.2, at=(0, 0)):
    cx, cy = screen(np.array([[at[0], at[1], 0.0]]))
    for a in np.linspace(0, 2 * math.pi, int(r * 8) + 8, endpoint=False):
        for dr in np.arange(-thick / 2, thick / 2 + .01, .6):
            x = cx[0] + math.cos(a) * (r + dr) * 1.0
            y = cy[0] + math.sin(a) * (r + dr) * .5
            xi, yi = int(round(x)), int(round(y))
            if 0 <= xi < SIZE[0] and 0 <= yi < SIZE[1]:
                img[yi, xi] = (*col, alpha)


def halo(img, at, r, alpha, col=GLOW):
    """A soft glow in three hard alpha steps."""
    cx, cy = screen(np.array([at], float))
    for y in range(int(cy[0] - r), int(cy[0] + r) + 1):
        for x in range(int(cx[0] - r), int(cx[0] + r) + 1):
            d = math.hypot(x - cx[0], (y - cy[0]) * 1.2) / r
            if d < 1 and 0 <= x < SIZE[0] and 0 <= y < SIZE[1]:
                a = alpha * (1 if d < .45 else .6 if d < .75 else .3)
                if img[y, x, 3] == 0:
                    img[y, x] = (*col, int(a))


# --- the sim holding them

class Seal:
    def __init__(self, seed):
        self.rng = np.random.default_rng(seed)
        self.t, self.k = 0.0, 0.0
        self.talismans, self.chains, self.stakes, self.glows = [], [], [], []
        self.earth = S.Earth(seed + 5)
        self.hit = None

    def fade(self, k):
        self.k = k
        self.earth.fade(k)

    def step(self, dt):
        self.t += dt
        for c in self.chains:
            c.step(self.t, dt)
        self.earth.step(dt)

    def render(self):
        img = S.render_earth(self.earth)
        under = np.zeros_like(img)
        for g in self.glows:
            g(under, self.t, self.k)
        m = (under[:, :, 3] > 0) & (img[:, :, 3] == 0)
        img[m] = under[m]
        layer = np.zeros_like(img)
        for i, c in enumerate(self.chains):
            c.draw(layer, self.t, self.k, i * 11)
        for at, t_in, i in self.stakes:
            if self.t >= t_in:
                stake(layer, at, min(1, (self.t - t_in) / .08), self.k, i)
        if (layer[:, :, 3] > 0).any():
            outline(layer, IRON["o"])
        tal = np.zeros_like(img)
        for i, tl in enumerate(sorted(self.talismans, key=lambda q: -(q.frame(self.t)[0][:2].sum() if q.frame(self.t) else 0))):
            tl.draw(tal, self.t, self.k, i * 7)
        if (tal[:, :, 3] > 0).any():
            outline(tal, PAPER["edge"])
        for lay in (layer, tal):
            m = lay[:, :, 3] > 0
            img[m] = lay[m]
        return img


def flash(t0, peak=(0, 0, 24), r0=10, out=36, dur=.5):
    """A red seal glow that blooms at the chest and a ring that runs out on the ground."""
    def g(img, t, k):
        u = (t - t0) / dur
        if 0 <= u < 1:
            halo(img, peak, r0 * (1 + .4 * u), int(200 * (1 - u)))
            ground_ring(img, 8 + out * (1 - (1 - u) ** 2), int(230 * (1 - u)), thick=1.6)
    return g


def orbit(r, h0, h1, turns, dur, phase=0.0, start=None):
    def path(tau):
        u = min(1.0, tau / dur)
        a = phase + u * turns * 2 * math.pi
        p = np.array([math.cos(a) * r, math.sin(a) * r, h0 + (h1 - h0) * u])
        if start is not None and u < .3:  # flying in from start
            w = u / .3
            p = np.asarray(start) * (1 - w) + p * w
        return p
    return path


def chest_point(a, h=24.0):
    """A point on the body's front surface, a radians round from the viewer (0 = straight at the camera)."""
    base = math.atan2(1, 1)  # the camera looks from +x +y
    return (math.cos(base + a) * (BODY_R + .5), math.sin(base + a) * (BODY_R + .5), h)


# --- 봉책 (seal_spell)

def bongchaek_a():
    """봉책 A: five talismans fly in from the caster, spiral twice round the body and slap onto the chest one after
    another; a red seal glow blooms and a ring runs out."""
    s = Seal(41)
    caster = (-96.0, 0.0, 30.0)
    for i in range(5):
        stick = chest_point((i - 2) * .5, 18 + (i % 2) * 10)
        s.talismans.append(Talisman(orbit(18, 34, 22, 1.6, .7, phase=i * 1.25, start=caster), i * .05,
                                    .62 + i * .06, stick, 6 + i))
    s.glows.append(flash(.92))
    s.hit = .92
    return s, 1.6, "single"


def bongchaek_b():
    """봉책 B: talismans flutter down from above, gather into a spinning band round the chest, then the band shrinks
    and they all slap on at once with a red flash."""
    s = Seal(42)
    for i in range(6):
        ph = i * 2 * math.pi / 6

        def path(tau, ph=ph, i=i):
            if tau < .4:  # fluttering down, swaying
                return np.array([math.cos(ph) * 26 + 6 * math.sin(tau * 9 + i), math.sin(ph) * 26,
                                 110 - 200 * tau + 4 * math.sin(tau * 13 + i)])
            u = tau - .4
            r = 26 - 12 * min(1, u / .35)
            a = ph + u * 9
            return np.array([math.cos(a) * r, math.sin(a) * r, 30 - 4 * min(1, u / .35)])
        s.talismans.append(Talisman(path, i * .03, .9, chest_point((i - 2.5) * .36, 12 + i * 4), 5 + i))
    s.glows.append(flash(.92, out=40))
    s.hit = .92
    return s, 1.6, "single"


def bongchaek_c():
    """봉책 C: four talismans fly to the cell's corners and stand in the ground, red lines run between them into a
    square seal on the floor, then a fifth slaps onto the chest."""
    s = Seal(43)
    caster = np.array([-96.0, 0.0, 30.0])
    corners = [(14, -14), (14, 14), (-14, 14), (-14, -14)]
    for i, (x, y) in enumerate(corners):
        dst = np.array([x, y, 6.0])

        def path(tau, dst=dst, i=i):
            u = min(1, tau / .4)
            return caster * (1 - u) + dst * u + np.array([0, 0, 40 * math.sin(math.pi * u)])
        tl = Talisman(path, i * .06, .4 + i * .06, dst, 8 + i)
        tl.stick_dir = (x, y)
        s.talismans.append(tl)

    def square(img, t, k):
        u = (t - .7) / .25
        if u < 0:
            return
        pts = [np.array([x, y, 0.0]) for x, y in corners]
        for j in range(4):
            a, b = pts[j], pts[(j + 1) % 4]
            run = min(1, max(0, u * 4 - j))
            for q in np.linspace(0, run, 24):
                x, y = screen(((a + (b - a) * q))[None])
                if not gone(k, int(x[0]), int(y[0]), j):
                    put(img, x[0], y[0], GLOW, False, 230)
                    put(img, x[0], y[0] - 1, (255, 150, 120), False, 160)
    s.glows.append(square)
    s.talismans.append(Talisman(lambda tau: caster + (np.array(chest_point(0, 24)) - caster) * min(1, tau / .25), 1.0,
                                1.25, chest_point(0, 24), 10))
    s.glows.append(flash(1.25, out=30))
    s.hit = 1.25
    return s, 1.8, "single"


# --- 봉쇄 (root_spell)

def wrap(r0, h0, h1, turns, dur, phase=0.0, end=None, end_t=None):
    """A path that winds round the legs from h0 to h1, then (from end_t) runs to the ground point end."""
    def path(tau):
        u = min(1.0, tau / dur)
        a = phase + u * turns * 2 * math.pi
        p = np.array([math.cos(a) * r0, math.sin(a) * r0, h0 + (h1 - h0) * u])
        if end is not None and tau > end_t:
            w = min(1, (tau - end_t) / .12)
            p = p * (1 - w) + np.array([end[0], end[1], -1.0]) * w
        return p
    return path


def bongswae_a():
    """봉쇄 A: three chains burst out of the ground round the target, coil down round his legs and drive their ends
    into the ground on the far side; dust where they break out."""
    s = Seal(51)
    for i in range(3):
        a = i * 2 * math.pi / 3 + .4
        root = (math.cos(a) * 16, math.sin(a) * 16)
        end = (math.cos(a + 2.6) * 15, math.sin(a + 2.6) * 15)
        s.chains.append(Chain(26, 2.6, lambda tau, r=root: np.array([r[0], r[1], -3.0]),
                              wrap(11, 26, 5, 1.6, .45, a, end, .5), i * .05, root_at=(root[0], root[1], -3)))
        s.earth.puff(i * .05, root, 30, out=40, up=(30, 70), life=(.4, .7))
        s.stakes.append((end, .62 + i * .05, i))
    s.hit = .62
    return s, 1.5, "single"


def bongswae_b():
    """봉쇄 B: four iron stakes slam into the cell's corners, chains whip from them to the target's ankles and pull
    taut, binding his feet to the ground."""
    s = Seal(52)
    corners = [(16, -16), (16, 16), (-16, 16), (-16, -16)]
    for i, (x, y) in enumerate(corners):
        s.stakes.append(((x, y), i * .06, i))
        s.earth.puff(i * .06, (x, y), 18, out=30, up=(20, 50), life=(.3, .6))
        a = math.atan2(-y, -x)
        ankle = (-math.cos(a) * (BODY_R + 1.5), -math.sin(a) * (BODY_R + 1.5), 6.0)
        start = (x, y, 2.0)

        def head(tau, ankle=np.array(ankle), start=np.array(start)):
            u = min(1, max(0, (tau - .15) / .2))
            return start * (1 - u) + ankle * u + np.array([0, 0, 18 * math.sin(math.pi * u)])
        s.chains.append(Chain(12, 2.6, lambda tau, st=start: np.array(st), head, i * .06))
    s.hit = .4
    return s, 1.4, "single"


def bongswae_c():
    """봉쇄 C: one heavy chain spirals up out of the ground round the legs like a snake, three turns, and its end is
    pinned by a big stake slammed down in front with a burst of dust."""
    s = Seal(53)
    s.chains.append(Chain(44, 2.6, lambda tau: np.array([12.0, 4.0, -3.0]),
                          wrap(11, 0, 22, 3, .55, 0, (17, 10), .6), 0, root_at=(12, 4, -3)))
    s.earth.puff(0, (12, 4), 30, out=40, up=(30, 80), life=(.4, .8))
    s.stakes.append(((17, 10), .72, 0))
    s.earth.puff(.72, (17, 10), 40, out=70, up=(20, 60), life=(.5, .9))
    s.hit = .72
    return s, 1.5, "single"


# --- 봉격 (disarm_spell)

def bonggyeok_a():
    """봉격 A: two chains whip in from either side at arm height, wrap the body twice and cinch tight; a talisman
    flies in and sticks on the knot."""
    s = Seal(61)
    for i, side in enumerate((-1, 1)):
        a0 = math.atan2(1, 1) + side * 1.6
        root = (math.cos(a0) * 60, math.sin(a0) * 60, 26.0)

        def head(tau, a0=a0, side=side):
            u = min(1, tau / .5)
            r = 30 * (1 - u) + (BODY_R + 1.2) * u
            a = a0 + side * u * 2 * 2 * math.pi
            return np.array([math.cos(a) * r, math.sin(a) * r, 24 - 4 * u])

        def rootp(tau, root=np.array(root), side=side):
            u = min(1, tau / .5)
            a = math.atan2(root[1], root[0]) + side * u * 2.5
            r = 60 * (1 - u) + (BODY_R + 1.2) * u
            return np.array([math.cos(a) * r, math.sin(a) * r, 26 - 6 * u])
        s.chains.append(Chain(22, 3.0, rootp, head, i * .05, root_at=root))
    s.talismans.append(Talisman(lambda tau: np.array([-96.0, 0, 30]) + (np.array(chest_point(0, 22)) - [-96.0, 0, 30])
                                * min(1, tau / .25), .55, .8, chest_point(0, 22), 9))
    s.glows.append(flash(.8, peak=(0, 0, 22), r0=8, out=26, dur=.4))
    s.hit = .8
    return s, 1.5, "single"


def bonggyeok_b():
    """봉격 B: a chain lashes from the caster round the target's weapon arm, coils it and pulls taut back toward
    the caster; a talisman slaps on where it binds."""
    s = Seal(62)
    caster = np.array([-96.0, 0, 28.0])
    arm = np.array(chest_point(-.9, 22))

    def head(tau):
        u = min(1, tau / .25)
        if u < 1:
            return caster * (1 - u) + arm * u + np.array([0, 0, 20 * math.sin(math.pi * u)])
        w = min(1, (tau - .25) / .3)  # coils twice round the arm
        a = math.atan2(arm[1], arm[0]) + w * 4 * math.pi
        return np.array([math.cos(a) * (BODY_R + 1.2), math.sin(a) * (BODY_R + 1.2), 22 - 3 * w])

    def root(tau):
        pull = min(1, max(0, (tau - .55) / .15))  # yanked back taut
        return caster + np.array([10 - 20 * pull, 0, 0])
    s.chains.append(Chain(34, 3.0, root, head, 0, root_at=caster))
    s.talismans.append(Talisman(lambda tau: caster + (arm - caster) * min(1, tau / .25), .6, .85, arm + [0, 0, 1], 7))
    s.glows.append(flash(.85, peak=tuple(arm), r0=7, out=22, dur=.4))
    s.hit = .7
    return s, 1.5, "single"


def bonggyeok_c():
    """봉격 C: an iron shackle ring drops onto the chest and clamps, short chains from it stake into the ground on
    both sides; a talisman sticks on the ring."""
    s = Seal(63)

    def ring_head(side):
        def f(tau):
            drop = max(0, 1 - tau / .25)
            u = min(1, tau / .25)
            r = 16 - 5 * min(1, max(0, (tau - .25) / .1))
            a = math.atan2(1, 1) + side * math.pi / 2
            return np.array([math.cos(a) * r, math.sin(a) * r, 22.0])
        return f
    # the ring itself: a closed loop of links following a shrinking circle
    loop = []
    for i in range(14):
        def f(tau, i=i):
            drop = max(0, 1 - tau / .25)
            r = 16 - 5 * min(1, max(0, (tau - .25) / .1))
            a = i * 2 * math.pi / 14
            return np.array([math.cos(a) * r, math.sin(a) * r, 22 + 60 * drop ** 2])
        loop.append(f)

    class Ring(Chain):
        def step(self, t, dt):
            if t < self.t0:
                return
            for j, f in enumerate(loop):
                self.p[j] = f(t - self.t0)
    ring = Ring(14, 3.0, None, None, 0, root_at=(0, 0, 82))
    ring.p = np.array([f(0) for f in loop])
    s.chains.append(ring)
    for i, side in enumerate((-1, 1)):
        a = math.atan2(1, 1) + side * math.pi / 2
        gp = (math.cos(a) * 22, math.sin(a) * 22)
        s.chains.append(Chain(10, 3.0, lambda tau, g=gp: np.array([g[0], g[1], 0.0]), ring_head(side), .3,
                              root_at=(gp[0], gp[1], 0)))
        s.stakes.append((gp, .3, i))
        s.earth.puff(.3, gp, 16, out=30, up=(20, 50), life=(.3, .6))
    s.talismans.append(Talisman(lambda tau: np.array([-96.0, 0, 30]) + (np.array(chest_point(0, 23)) - [-96.0, 0, 30])
                                * min(1, tau / .25), .4, .65, chest_point(0, 23), 9))
    s.glows.append(flash(.65, peak=(0, 0, 22), r0=8, out=24, dur=.4))
    s.hit = .35
    return s, 1.4, "single"


SPELLS = {f.__name__: f for f in (bongchaek_a, bongchaek_b, bongchaek_c, bongswae_a, bongswae_b, bongswae_c,
                                   bonggyeok_a, bonggyeok_b, bonggyeok_c)}


def run(name):
    sim, secs, layout = SPELLS[name]()
    frames = []
    n = int(secs * FPS)
    for f in range(n):
        sim.fade(f / n)
        for _ in range(SUB):
            sim.step(1 / (FPS * SUB))
        frames.append(sim.render())
    return frames, layout


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("names", nargs="*")
    ap.add_argument("--out", default=str(S.ROOT / "reports/spellsim"))
    args = ap.parse_args()
    out = Path(args.out)
    out.mkdir(parents=True, exist_ok=True)
    for name in args.names or SPELLS:
        frames, layout = run(name)
        S.compose(name, frames, layout, out)
        print(out / f"{name}.gif")


if __name__ == "__main__":
    main()
