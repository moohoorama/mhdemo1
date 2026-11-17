#!/usr/bin/env python3
"""Spell effect drafts for water, fire and earth: particles and rocks simulated in the battle map's iso space,
then drawn as iso pixel art by a renderer made for each element.

    uv run --with numpy --with scipy --with pillow python tools/spellsim.py              # every spell
    uv run --with numpy --with scipy --with pillow python tools/spellsim.py hwaryong geoam --out reports/spellsim
    uv run --with numpy --with scipy --with pillow python tools/spellsim.py --bake          # into the game

World units: x, y run along the map's two cell axes, 32 units to a cell (a cell is the 64x32 diamond on screen),
h is height in screen pixels, so screen x = x - y and screen y = (x + y) / 2 - h. The target stands on the cell at
the origin. Each spell writes <name>.gif (over grass, its cells outlined, infantrymen for scale) and <name>.png
(every frame, five to a row).

- water: Clavet et al. 2005's double density relaxation (incompressible, with surface tension) and render_water
  (flat see-through puddles, deep body with a narrow lit band and a glint, streams stretched along their motion).
- fire: hot particles that rise by their heat, sway in a moving noise field and cool; render_fire colours the
  blurred heat from white through yellow, orange and red, with smoke where it has cooled and a glow on the ground.
- earth: rocks (voxel lumps that tumble) that shatter when they land hard into smaller rocks, grit and dust;
  render_earth shades the rocks by face and draws the dust as soft see-through clouds.
"""
import argparse
import json
import math
from pathlib import Path

import numpy as np
from PIL import Image, ImageDraw
from scipy.ndimage import gaussian_filter
from scipy.spatial import cKDTree

ROOT = Path(__file__).resolve().parents[1]
G = 900.0          # gravity, units/s^2
FPS = 15           # frames written a second
SUB = 8            # simulation steps a frame
VOX = 2.0          # voxel edge in world units: 1/16 of a cell, a 4x4 px dot on screen
SIZE, ORIGIN = (420, 360), (210, 260)   # the working canvas and where the world origin falls on it
CROSS = [(0, 0), (32, 0), (-32, 0), (0, 32), (0, -32)]
LINE = [(32 * i, 0) for i in range(6)]  # 폭염's line (skills.json Length 6) from the cell in front of the caster
LAYOUTS = {  # cells outlined and where the caster stands
    "single": ([(0, 0)], (-96, 0)), "cross": (CROSS, (-96, 0)), "line": (LINE, (-32, 0)),
    "line_back": ([(-x, y) for x, y in LINE], (32, 0)),
}
LIGHT = np.array([.2, .7, 1.0]) / np.linalg.norm([.2, .7, 1.0])   # from up and screen-left
VIEW = np.ones(3) / math.sqrt(3)


def screen(p, origin=ORIGIN):
    return p[:, 0] - p[:, 1] + origin[0], (p[:, 0] + p[:, 1]) / 2 - p[:, 2] + origin[1]


def stencil(s):
    """Pixels of one voxel cube (edge s units) relative to the screen point of its (x0, y0, h0) corner, with
    the face each shows: 0 top, 1 left (+y face), 2 right (+x face)."""
    out = []
    for py in range(-int(2 * s) - 2, int(2 * s) + 2):
        for px in range(-int(2 * s) - 2, int(2 * s) + 2):
            cx, cy = px + .5, py + .5
            x, y = cy + s + cx / 2, cy + s - cx / 2
            if 0 <= x < s and 0 <= y < s:
                out.append((px, py, 0))
                continue
            y = s - cx
            h = (s + y) / 2 - cy
            if 0 <= y < s and 0 <= h < s:
                out.append((px, py, 2))
                continue
            x = cx + s
            h = (x + s) / 2 - cy
            if 0 <= x < s and 0 <= h < s:
                out.append((px, py, 1))
    return np.array(out)


ST = stencil(VOX)


def zbuffer(keys, tone_of, size):
    """Voxels (integer voxel indices) drawn front-most per pixel. tone_of(face (n*m,), voxel (n*m,)) gives each
    pixel's tone; returns (mask, tone, thickness) arrays."""
    w, h = size
    x0, y0, z0 = keys[:, 0] * VOX, keys[:, 1] * VOX, keys[:, 2] * VOX
    vx = np.round(x0 - y0 + ORIGIN[0]).astype(int)
    vy = np.round((x0 + y0) / 2 - z0 + ORIGIN[1]).astype(int)
    n, m = len(keys), len(ST)
    px = (vx[:, None] + ST[None, :, 0]).ravel()
    py = (vy[:, None] + ST[None, :, 1]).ravel()
    face = np.tile(ST[:, 2], n)
    vox = np.repeat(np.arange(n), m)
    dep = np.repeat(x0 + y0 + z0, m) + (face == 0) * .01
    ins = (px >= 0) & (px < w) & (py >= 0) & (py < h)
    px, py, face, vox, dep = px[ins], py[ins], face[ins], vox[ins], dep[ins]
    flat = py * w + px
    thick = np.bincount(flat, minlength=w * h).reshape(h, w)
    order = np.lexsort((dep, flat))
    last = np.r_[flat[order][1:] != flat[order][:-1], True]
    sel = order[last]
    mask = np.zeros((h, w), bool)
    mask[py[sel], px[sel]] = True
    tone = np.zeros((h, w), int)
    tone[py[sel], px[sel]] = tone_of(face[sel], vox[sel])
    return mask, tone, thick


def ring(mask):
    """Pixels just outside mask."""
    grow = mask.copy()
    grow[1:] |= mask[:-1]
    grow[:-1] |= mask[1:]
    grow[:, 1:] |= mask[:, :-1]
    grow[:, :-1] |= mask[:, 1:]
    return grow & ~mask


def shadows(img, p, min_h=4):
    h, w = img.shape[:2]
    sx, sy = screen(p * (1, 1, 0))
    sh = np.zeros((h, w), bool)
    for x, y, hh in zip(sx.astype(int), sy.astype(int), p[:, 2]):
        if hh > min_h and 0 <= y < h - 1 and 1 <= x < w - 3:
            sh[y:y + 2, x - 1:x + 3] = True
    sh &= img[:, :, 3] == 0
    img[sh] = (20, 40, 20, 90)


def streaks(img, p, v, k, cols, max_len=5):
    """Short lines from each point back along its motion, head first in cols[0]."""
    h, w = img.shape[:2]
    hx, hy = screen(p)
    tx, ty = screen(p - v * k)
    for ax, ay, bx, by in zip(hx, hy, tx, ty):
        ln = max(1.0, math.hypot(bx - ax, by - ay))
        for j in range(int(min(max_len, ln)) + 1):
            x, y = int(round(ax + (bx - ax) * j / ln)), int(round(ay + (by - ay) * j / ln))
            if 0 <= x < w and 0 <= y < h:
                img[y, x] = (*cols[min(j, len(cols) - 1)], 255)


def aim(src, dst, t):
    """Launch velocity to fly from src to dst (x, y, h) in t seconds."""
    src, dst = np.asarray(src, float), np.asarray(dst, float)
    v = (dst - src) / t
    v[2] += .5 * G * t
    return v


def over_cells(rng, n, cells=CROSS, reach=2.2):
    """Velocities that land n falling things spread over the cells."""
    c = np.array(cells, float)[rng.integers(0, len(cells), n)]
    return np.column_stack([c[:, 0] * reach, c[:, 1] * reach, np.zeros(n)]) + rng.normal(0, 18, (n, 3))


def helix(height, turn, r0=48, r1=6):
    """A whirl drawing in from a wide ring into a twisting column over the centre, reached at tau = 0.6."""
    def path(tau):
        tau = np.asarray(tau)
        u = np.clip(tau / .6, 0, 1)
        r = r0 * (1 - u) ** 1.5 + r1 + 4 * u
        a = tau * turn
        return np.column_stack([np.cos(a) * r, np.sin(a) * r, height * u ** 1.3])
    return path


class Particles:
    """Positions, velocities and birth times, with guided stretches: until `release` after its birth a guided
    particle follows path(tau) + its offset, then it is let go with its own fling velocity."""

    def __init__(self, seed):
        self.rng = np.random.default_rng(seed)
        self.p = np.zeros((0, 3))
        self.v = np.zeros((0, 3))
        self.born = np.zeros(0)
        self.dead = np.zeros(0, bool)
        self.guides = []  # (indices, path, release, offsets, fling)
        self.triggers = []  # [x, y, radius, fn(sim, t, (x, y)), fired]: set off by the first guided one landing near
        self.t = 0.0
        self.hit = None     # when the blow first lands (falling things hitting the ground), for the game's timing

    def _grow(self, n, **cols):
        for name, val in cols.items():
            setattr(self, name, np.concatenate([getattr(self, name), val]))

    def add(self, t, p, v, **extra):
        n = len(p)
        self.p = np.vstack([self.p, p])
        self.v = np.vstack([self.v, v])
        self._grow(n, born=t if np.ndim(t) else np.full(n, float(t)), dead=np.zeros(n, bool), **extra)
        return np.arange(len(self.p) - n, len(self.p))

    def guide(self, idx, path, release, offsets, fling):
        self.guides.append((idx, path, release, offsets, fling))

    def on_landing(self, cells, fn, radius=22):
        """fn(sim, t, cell) runs once in each cell, when the first guided particle lands near it."""
        for x, y in cells:
            self.triggers.append([x, y, radius, fn, False])

    def landed(self, idx):
        if not len(idx) or not self.triggers:
            return
        guided = np.zeros(len(self.p), bool)
        for g in self.guides:
            guided[g[0]] = True
        q = self.p[idx[guided[idx]]]
        if not len(q):
            return
        for tr in self.triggers:
            if not tr[4] and (np.hypot(q[:, 0] - tr[0], q[:, 1] - tr[1]) < tr[2]).any():
                tr[4] = True
                tr[3](self, self.t, (tr[0], tr[1]))

    def steer(self, dt):
        """Moves the guided particles along their paths; returns the mask of those held."""
        held = np.zeros(len(self.p), bool)
        for idx, path, release, off, fling in self.guides:
            tau = self.t - self.born[idx]
            on = (tau >= 0) & (tau < release)
            if on.any():
                i = idx[on]
                new = path(tau[on]) + off[on]
                self.v[i] = (new - path(tau[on] - dt) - off[on]) / dt
                self.p[i] = new
                held[i] = True
            let = (tau >= release) & (tau < release + dt + 1e-9)
            if let.any():
                self.v[idx[let]] = fling[let]
        return held

    def fade(self, k):
        """Past 60% of the effect, a growing share of what is left goes, so it ends empty."""
        if not hasattr(self, "_gone"):
            self._gone = self.rng.random(len(self.p))
        if len(self._gone) < len(self.p):
            self._gone = np.concatenate([self._gone, self.rng.random(len(self.p) - len(self._gone))])
        if k > .6:
            self.dead |= self._gone[:len(self.p)] < (k - .6) / .4


# --- water

H = 5.0            # interaction radius, world units (particles sit about 2 apart)
RHO0 = 6.5         # rest density
K, K_NEAR = .006, .02
VISC = .04
WATER = {"deep": (19, 48, 106), "body": (36, 87, 168), "light": (77, 143, 219), "hi": (168, 220, 255),
         "w": (255, 255, 255), "edge": (11, 31, 69)}


class Water(Particles):
    def __init__(self, seed, colliders=()):
        super().__init__(seed)
        self.life = np.zeros(0)        # seconds it may lie on the ground before draining away
        self.ground = np.zeros(0)
        self.colliders = list(colliders)  # (x, y, radius, height): units that water parts round

    def add(self, t, p, v, life=(.15, .5)):
        n = len(p)
        return super().add(t, p, v, life=self.rng.uniform(*life, n), ground=np.zeros(n))

    def blob(self, t, centre, r, n, vel, jitter=20.0, life=(.15, .5)):
        d = self.rng.normal(size=(n, 3))
        d /= np.linalg.norm(d, axis=1, keepdims=True)
        p = np.asarray(centre) + d * (r * self.rng.random(n) ** (1 / 3))[:, None]
        return self.add(t, p, np.asarray(vel) + self.rng.normal(0, jitter, (n, 3)), life)

    def step(self, dt):
        self.t += dt
        live = (self.born <= self.t) & ~self.dead
        held = self.steer(dt)
        free = np.flatnonzero(live & ~held)
        if not len(free):
            return
        self.v[free, 2] -= G * dt
        q = self.p[free]
        pairs = cKDTree(q).query_pairs(H, output_type="ndarray") if len(free) > 1 else np.zeros((0, 2), int)
        if len(pairs):
            i, j = pairs.T
            d = q[j] - q[i]
            r = np.maximum(np.linalg.norm(d, axis=1), 1e-6)
            n = d / r[:, None]
            w = 1 - r / H
            u = ((self.v[free[i]] - self.v[free[j]]) * n).sum(1)  # closing speed
            imp = np.where(u > 0, dt * w * VISC * u * u / 60, 0)[:, None] * n
            np.add.at(self.v, free[i], -imp / 2)
            np.add.at(self.v, free[j], imp / 2)
        prev = self.p[free].copy()
        self.p[free] += self.v[free] * dt
        if len(pairs):
            q = self.p[free]
            d = q[j] - q[i]
            r = np.maximum(np.linalg.norm(d, axis=1), 1e-6)
            w = np.clip(1 - r / H, 0, None)
            rho = np.bincount(i, w ** 2, len(free)) + np.bincount(j, w ** 2, len(free))
            near = np.bincount(i, w ** 3, len(free)) + np.bincount(j, w ** 3, len(free))
            P, Pn = K * (rho - RHO0), K_NEAR * near
            D = ((P[i] + P[j]) / 2 * w + (Pn[i] + Pn[j]) / 2 * w * w)[:, None] * d / r[:, None]
            delta = np.zeros_like(q)
            np.add.at(delta, i, -D / 2)
            np.add.at(delta, j, D / 2)
            self.p[free] += delta
        hit_any = np.zeros(len(free), bool)
        for cx, cy, rad, hgt in self.colliders:
            rel = self.p[free, :2] - (cx, cy)
            dist = np.linalg.norm(rel, axis=1)
            hit = (dist < rad) & (self.p[free, 2] < hgt)
            if hit.any():
                self.p[free[hit], :2] = (cx, cy) + rel[hit] / np.maximum(dist[hit, None], 1e-6) * rad
                hit_any |= hit
        vz = self.v[free, 2].copy()
        touch = free[self.p[free, 2] < 0]
        if self.hit is None and (vz[self.p[free, 2] < 0] < -150).any():
            self.hit = self.t
        landed = (self.p[free, 2] < 0) & (vz < -280) & (self.rng.random(len(free)) < .25)
        self.p[free, 2] = np.maximum(self.p[free, 2], 0)
        self.v[free] = (self.p[free] - prev) / dt
        if landed.any():  # water that hits the ground hard throws up a crown of splash
            k = free[landed]
            speed = -vz[landed]
            ang = self.rng.uniform(0, 2 * math.pi, len(k))
            self.v[k, 0] += np.cos(ang) * speed * self.rng.uniform(.1, .28, len(k))
            self.v[k, 1] += np.sin(ang) * speed * self.rng.uniform(.1, .28, len(k))
            self.v[k, 2] = speed * self.rng.uniform(.15, .4, len(k))
        if hit_any.any():
            k = free[hit_any]
            self.v[k, 2] += self.rng.uniform(0, 40, len(k))
        flat = free[self.p[free, 2] <= .5]
        self.v[flat, :2] *= math.exp(-3 * dt)
        self.ground[flat] += dt
        self.dead |= self.ground > self.life
        self.landed(touch)

    def render(self):
        on = (self.born <= self.t) & ~self.dead
        return render_water(self.p[on], self.v[on])


def render_water(p, v):
    """Puddles flat on the ground (see-through, a bright rim, glints); the moving body splatted stretched along
    its motion so fast water runs in streams and sheets, toned deep with a narrow lit band and a white glint, a
    bright rim on its upper-left edge; free drops as short streaks."""
    w, h = SIZE
    img = np.zeros((h, w, 4), np.uint8)
    if not len(p):
        return img
    ground = p[:, 2] < 1.5
    gp = p[ground]
    if len(gp):
        idx = np.floor(gp[:, :2] / VOX).astype(int)
        lo = idx.min(0) - 3
        g2 = np.zeros(idx.max(0) - lo + 4, np.float32)
        np.add.at(g2, tuple((idx - lo).T), 1)
        wet = gaussian_filter(g2, 1.3) > .22
        at = np.argwhere(wet)
        rim = np.zeros_like(wet)
        rim[1:-1, 1:-1] = wet[1:-1, 1:-1] & ~(wet[:-2, 1:-1] & wet[2:, 1:-1] & wet[1:-1, :-2] & wet[1:-1, 2:])
        for (a, b), r in zip(at, rim[tuple(at.T)]):
            x0, y0 = (a + lo[0]) * VOX, (b + lo[1]) * VOX
            sx, sy = int(round(x0 - y0 + ORIGIN[0])), int(round((x0 + y0) / 2 + ORIGIN[1]))
            glint = (a * 7 + b * 13) % 23 == 0
            col, al = (WATER["hi"], 230) if r else (WATER["w"], 255) if glint else (WATER["body"], 150)
            for dx, dy in ((-1, 0), (0, 0), (1, 0), (2, 0), (0, 1), (1, 1)):
                x, y = sx + dx, sy + dy
                if 0 <= x < w and 0 <= y < h:
                    img[y, x] = (*col, al)
    bp, bv = p[~ground], v[~ground]
    if not len(bp):
        return img
    sp = np.vstack([bp] + [bp - bv * .012 * k for k in (1, 2, 3)])
    sp = sp[sp[:, 2] > 0]
    idx = np.floor(sp / VOX).astype(int)
    lo = idx.min(0) - 4
    grid = np.zeros(idx.max(0) - lo + 5, np.float32)
    np.add.at(grid, tuple((idx - lo).T), 1)
    dens = gaussian_filter(grid, .85)
    occ = dens > .55
    occ[:, :, :max(0, -lo[2]) + 1] = False
    cell = tuple((np.floor(bp / VOX).astype(int) - lo).T)
    drops = ~occ[cell] & (dens[cell] <= .55)
    shadows(img, bp)
    at = np.argwhere(occ)
    if len(at):
        gx, gy, gz = np.gradient(dens)
        nrm = -np.column_stack([gx[tuple(at.T)], gy[tuple(at.T)], gz[tuple(at.T)]])
        nrm /= np.maximum(np.linalg.norm(nrm, axis=1, keepdims=True), 1e-6)
        dif = nrm @ LIGHT
        refl = 2 * dif[:, None] * nrm - LIGHT
        tone = np.where(refl @ VIEW > .975, 4, np.where(dif > .8, 3, np.where(dif > .3, 2, np.where(dif > -.2, 1, 0))))
        body, col, thick = zbuffer(at + lo, lambda face, vox: tone[vox], SIZE)
        col = np.where((thick > 40) & (col == 2), 1, col)
        ramp = np.array([WATER[c] for c in ("deep", "body", "light", "hi", "w")])
        img[body, :3] = ramp[col[body]]
        img[body, 3] = 240
        up = np.zeros_like(body)
        up[1:, 1:] = body[1:, 1:] & ~body[:-1, :-1]
        img[up & (col < 3), :3] = WATER["hi"]
        edge = ring(body)
        img[edge, :3] = WATER["edge"]
        img[edge, 3] = 255
    if drops.any():
        streaks(img, bp[drops], bv[drops], .02, [WATER["w"], WATER["hi"], WATER["light"]])
    return img


def unit(at=(0, 0)):
    return (at[0], at[1], 9.0, 40.0)


def tak():
    """탁류: a lump of water drops straight on the target's head and bursts, spreading in a crown."""
    s = Water(3, [unit()])
    s.blob(0, (0, 0, 120), 9, 480, (0, 0, -260), 10)
    return s, 1.0, "single"


def geyser(s, t0, at, n=30, up=330):
    """A column of water thrown up out of the ground at a cell."""
    for i in range(30):
        p = np.column_stack([s.rng.normal(at[0], 4, n), s.rng.normal(at[1], 4, n), np.zeros(n)])
        v = np.column_stack([s.rng.normal(0, 30, n), s.rng.normal(0, 30, n), s.rng.normal(up, up * .085, n)])
        s.add(t0 + i * .01, p, v)


def gyeok():
    """격류: the ground bursts under the target and a geyser throws a column of water up, raining back down."""
    s = Water(6)
    geyser(s, 0, (0, 0))
    s.hit = .05  # the blow is the burst, not the water falling back
    return s, 1.0, "single"


def sujin():
    """수진: five lumps of water rain down, one on each cell of the cross, one after another."""
    s = Water(8, [unit(c) for c in CROSS])
    for k, (cx, cy) in enumerate(CROSS):
        s.blob(k * .06, (cx, cy, 90), 5.5, 150, (0, 0, -150), 8)
    return s, 1.1, "cross"


def suryong():
    """수룡: a whirl draws water in from a wide ring into a twisting spout over the centre, which breaks and
    falls on the five cells, a smaller 격류 bursting up in each where it lands."""
    s = Water(12)
    n = 1100
    idx = s.add(np.linspace(0, .3, n), np.zeros((n, 3)), np.zeros((n, 3)))
    phase = s.rng.uniform(0, 2 * math.pi, n)
    off = np.column_stack([np.cos(phase) * 3, np.sin(phase) * 3, s.rng.normal(0, 3, n)])
    s.guide(idx, helix(85, 32), .6, off, over_cells(s.rng, n) + (0, 0, -120))
    s.on_landing(CROSS, lambda sim, t, c: geyser(sim, t, c, 14, 240))  # a smaller 격류 bursts where it lands
    return s, 2.0, "cross"


def haeil():
    """해일: a huge lump of water crashes down on the target from up-left, one hard slap, a wide splash."""
    s = Water(14, [unit()])
    src = (-70, -10, 150)
    s.blob(0, src, 16, 1400, aim(src, (0, 0, 30), .26), 14)
    return s, 1.1, "single"


# --- fire

FIRE = [(255, 250, 220), (255, 222, 96), (255, 156, 40), (226, 78, 26), (150, 36, 18), (84, 22, 14)]
SMOKE = (52, 46, 44)
BUOY, SWAY = 520.0, 420.0


class Fire(Particles):
    """Hot particles: heat runs from 1 at birth (or on landing, for a falling fireball) down to 0 over its life;
    they rise by their heat, sway in a moving noise field and slow by their drag. Heavy ones fall like burning
    lumps and catch alight where they land."""

    def __init__(self, seed):
        super().__init__(seed)
        self.age = np.zeros(0)
        self.life = np.zeros(0)
        self.drag = np.zeros(0)
        self.heavy = np.zeros(0, bool)
        self.ember = np.zeros(0, bool)
        self.hot = np.zeros(0)
        self.soft_embers = False  # sparks cool through the ramp and add light over the flame instead of painting on it

    def add(self, t, p, v, life=(.35, .7), drag=2.5, heavy=False, ember=False, hot=(.65, 1.0)):
        n = len(p)
        return super().add(t, p, v, age=np.zeros(n), life=self.rng.uniform(*life, n), drag=np.full(n, drag),
                           heavy=np.full(n, heavy), ember=np.full(n, ember), hot=self.rng.uniform(*hot, n))

    def burst(self, t, at, radius, n, up=(60, 140), spread=40, life=(.35, .7), drag=2.5):
        a = self.rng.uniform(0, 2 * math.pi, n)
        r = radius * np.sqrt(self.rng.random(n))
        p = np.column_stack([at[0] + np.cos(a) * r, at[1] + np.sin(a) * r, self.rng.uniform(0, 4, n)])
        v = np.column_stack([self.rng.normal(0, spread, n), self.rng.normal(0, spread, n), self.rng.uniform(*up, n)])
        return self.add(t, p, v, life, drag)

    def embers(self, t, at, n, speed=160, radius=0.0, h=(4, 20), life=(.5, .9)):
        """Sparks thrown up and out; radius and h scatter where they start, through the flame rather than from one
        point."""
        a = self.rng.uniform(0, 2 * math.pi, n)
        b, r = self.rng.uniform(0, 2 * math.pi, n), radius * np.sqrt(self.rng.random(n))
        p = np.column_stack([at[0] + np.cos(b) * r, at[1] + np.sin(b) * r, self.rng.uniform(*h, n)])
        v = np.column_stack([np.cos(a) * speed * .5, np.sin(a) * speed * .5, self.rng.uniform(.6, 1.4, n) * speed])
        return self.add(t, p, v, life, 1.0, ember=True)

    def heat(self):
        """How hot each one burns, 0..1: its starting heat cooling over its life. A burning lump in flight
        flickers between orange and yellow."""
        h = self.hot * np.clip(1 - self.age / self.life, 0, 1) ** 1.4
        flick = .55 + .3 * np.sin(np.arange(len(h)) * 1.7 + self.t * 11)
        return np.where(self.heavy, flick, h)

    def fade(self, k):
        if k > .85:  # lumps still in the air at the end go out
            self.dead |= self.heavy

    def step(self, dt):
        self.t += dt
        live = (self.born <= self.t) & ~self.dead
        held = self.steer(dt)
        free = np.flatnonzero(live & ~held)
        if not len(free):
            return
        self.age[free] += np.where(self.heavy[free], 0, dt)
        p, ht = self.p[free], self.heat()[free]
        t = self.t
        sway = np.column_stack([np.sin(p[:, 1] * .11 + p[:, 2] * .07 + t * 7.1) + np.sin(p[:, 2] * .19 - t * 5.3),
                                np.sin(p[:, 0] * .13 - p[:, 2] * .09 + t * 6.3) + np.sin(p[:, 2] * .17 + t * 4.7),
                                np.zeros(len(p))]) * SWAY * ht[:, None] * dt
        hv, em = self.heavy[free], self.ember[free]
        rise = np.where(hv, -G, np.where(em, -G * .35, BUOY * ht)) * dt
        self.v[free, 2] += rise
        self.v[free] += np.where((hv | em)[:, None], 0, sway)
        self.v[free] *= np.exp(-np.where(hv, 0, self.drag[free]) * dt)[:, None]
        self.p[free] += self.v[free] * dt
        low = self.p[free, 2] < 0
        self.p[free[low], 2] = 0
        land = free[low & hv]
        if len(land) and self.hit is None:
            self.hit = self.t
        if len(land) and self.triggers:  # the lumps feed the flames their cells burst into
            self.dead[land] = True
            self.landed(land)
            land = land[:0]
        if len(land):  # a burning lump hits the ground and bursts into flame there
            self.heavy[land] = False
            self.age[land] = 0
            self.life[land] = self.rng.uniform(.45, .8, len(land))
            self.v[land] = np.column_stack([self.rng.normal(0, 60, len(land)), self.rng.normal(0, 60, len(land)),
                                            self.rng.uniform(80, 200, len(land))])
        self.v[free[low & ~hv], 2] = np.abs(self.v[free[low & ~hv], 2]) * .2
        self.dead |= self.age > self.life

    def render(self):
        on = (self.born <= self.t) & ~self.dead
        return render_fire(self.p[on], self.v[on], self.heat()[on], self.ember[on], self.heavy[on], self.soft_embers)


def soft_sparks(img, p, v, heat):
    """Sparks coloured by how hot they still are, white through yellow and orange to red, and fading as they cool.
    Over the flame a spark adds its light to the pixel under it, so it glints in the fire rather than sitting on it;
    off the flame it is see-through in steps."""
    h, w = img.shape[:2]
    ramp = np.array(FIRE, float)
    hx, hy = screen(p)
    tx, ty = screen(p - v * .02)
    for ax, ay, bx, by, ht in zip(hx, hy, tx, ty, heat):
        if ht < .12:  # out: a dim red spark over grass reads as a speck of dirt
            continue
        tone = int(np.digitize(-ht, [-.8, -.55, -.32]))
        alpha = (60 + 195 * min(1.0, ht * 1.6)) // 50 * 50
        ln = max(1.0, math.hypot(bx - ax, by - ay))
        for j in range(int(min(4, ln)) + 1):
            x, y = int(round(ax + (bx - ax) * j / ln)), int(round(ay + (by - ay) * j / ln))
            if not (0 <= x < w and 0 <= y < h):
                continue
            col, a = ramp[min(tone + j // 2, 3)], alpha * (1 - j * .18)
            under = img[y, x]
            if under[3] == 255:
                under[:3] = np.minimum(255, under[:3] + col * a / 255 * .55).astype(np.uint8)
            elif a > under[3]:
                img[y, x] = (*col.astype(np.uint8), int(a))


def render_fire(p, v, heat, ember, heavy, soft_embers=False):
    """The ground glow under low flames, smoke where the fire has cooled, then the flame body: heat blurred in 3D
    and cut, each pixel coloured by its temperature on the ramp, hotter where the flame is thick, the silhouette
    one step darker; sparks as streaks."""
    w, h = SIZE
    img = np.zeros((h, w, 4), np.uint8)
    if not len(p):
        return img
    lowp = p[(p[:, 2] < 30) & ~ember]
    if len(lowp):  # a warm glow on the ground
        gx, gy = screen(lowp * (1, 1, 0))
        glow = np.zeros((h, w), np.float32)
        ok = (gx >= 0) & (gx < w) & (gy >= 0) & (gy < h)
        np.add.at(glow, (gy[ok].astype(int), gx[ok].astype(int)), 1)
        glow = gaussian_filter(glow, (3, 6))
        a = np.clip(glow * 60, 0, 70)
        m = a > 12
        img[m] = (255, 140, 40, 0)
        img[m, 3] = (a[m] // 20 * 20).astype(np.uint8)
    hot = (heat > .18) & ~ember
    cold = (heat <= .18) & ~ember & ~heavy
    if cold.any():  # smoke
        sp = p[cold]
        idx = np.floor(sp / VOX).astype(int)
        lo = idx.min(0) - 4
        grid = np.zeros(idx.max(0) - lo + 5, np.float32)
        np.add.at(grid, tuple((idx - lo).T), 1)
        dens = gaussian_filter(grid, 1.6)
        at = np.argwhere(dens > .12)
        if len(at):
            mask, _, thick = zbuffer(at + lo, lambda f, vox: 0, SIZE)
            a = np.clip(40 + thick * 2, 0, 130)
            a = (a // 30 * 30).astype(np.uint8)
            img[mask, :3] = SMOKE
            img[mask, 3] = np.maximum(img[mask, 3], a[mask])
    if hot.any():
        hp, hh = p[hot], heat[hot]
        hp = np.vstack([hp] + [hp - v[hot] * .012 * k for k in (1, 2, 3)])
        hh = np.concatenate([hh] * 4)
        idx = np.floor(hp / VOX).astype(int)
        lo = idx.min(0) - 4
        shape = idx.max(0) - lo + 5
        mass, temp = np.zeros(shape, np.float32), np.zeros(shape, np.float32)
        np.add.at(mass, tuple((idx - lo).T), hh)
        np.add.at(temp, tuple((idx - lo).T), hh * hh)
        mass, temp = gaussian_filter(mass, .9), gaussian_filter(temp, .9)
        occ = mass > .3
        occ[:, :, :max(0, -lo[2])] = False
        at = np.argwhere(occ)
        if len(at):
            tt = temp[tuple(at.T)] / np.maximum(mass[tuple(at.T)], 1e-6)
            tone = np.digitize(-tt, [-.85, -.68, -.5, -.32, -.18])
            body, col, thick = zbuffer(at + lo, lambda f, vox: tone[vox], SIZE)
            col = np.where(thick > 28, np.maximum(col - 1, 0), col)
            edge = body & ring(~body)
            col = np.where(edge, np.minimum(col + 1, len(FIRE) - 1), col)
            ramp = np.array(FIRE)
            img[body, :3] = ramp[col[body]]
            img[body, 3] = 255
    if ember.any() and soft_embers:
        soft_sparks(img, p[ember], v[ember], heat[ember])
    elif ember.any():
        streaks(img, p[ember], v[ember], .02, [FIRE[0], FIRE[1], FIRE[2], FIRE[3]], 4)
    return img


def scorch(s, t0, at):
    """Flames catch round a cell's feet and flare up to chest height, then die down."""
    for i in range(18):
        s.burst(t0 + i * .025, at, 9, 22, (70, 150), 30, (.45, .8))
    s.soft_embers = True
    for i in range(16):  # sparks rise out of the whole flame while it burns, thinning and dying with it
        k = 1 - i / 16 * .65
        s.embers(t0 + .04 + i * .03, at, int(9 - i * .35), 150 * k, radius=8, h=(4, 30), life=(.35 * k, .6 * k))


def choyeol():
    """초열: flames catch round the target's feet and flare up to his chest, then die down."""
    s = Fire(21)
    scorch(s, 0, (0, 0))
    return s, 1.0, "single"


def eophwa():
    """업화: a roaring pillar of fire bursts up and swallows the target, throwing sparks."""
    s = Fire(22)
    s.burst(0, (0, 0), 10, 120, (200, 320), 50, (.15, .3))  # the white-hot blast
    for i in range(28):
        s.burst(.02 + i * .025, (0, 0), 15, 50, (150, 280), 40, (.5, .9))
    s.soft_embers = True
    for i in range(24):  # sparks keep bursting out of the whole flame for as long as it roars, thinning and
        k = 1 - i / 24 * .7  # living shorter as it dies so the last go out with it
        s.embers(.03 + i * .032, (0, 0), int(34 - i * .9), 260 * k, radius=13, h=(4, 46), life=(.4 * k, .7 * k))
    return s, 1.3, "single"


def hwajin():
    """화진: a flame flares up in each of the five cells of the cross."""
    s = Fire(23)
    for k, c in enumerate(CROSS):
        for i in range(10):
            s.burst(k * .03 + i * .025, c, 8, 18, (60, 130), 28, (.4, .7))
        s.embers(k * .03 + .08, c, 40, 120)
    return s, 1.1, "cross"


def hwaryong():
    """화룡: a whirl of fire twists up into a burning column over the centre, which breaks and falls on the five
    cells, each bursting into 초열 where it lands."""
    s = Fire(24)
    n = 1500
    idx = s.add(np.linspace(0, .3, n), np.zeros((n, 3)), np.zeros((n, 3)), (.5, .8))
    phase = s.rng.uniform(0, 2 * math.pi, n)
    off = np.column_stack([np.cos(phase) * 3, np.sin(phase) * 3, s.rng.normal(0, 3, n)])
    s.heavy[idx] = True
    s.guide(idx, helix(85, 32), .6, off, over_cells(s.rng, n) + (0, 0, -120))
    s.on_landing(CROSS, scorch)  # each cell the fire falls on bursts into 초열
    return s, 2.0, "cross"


def pokyeom(back=False):
    """폭염: pillars of fire like 업화 burst up one after another along the cells of the line, a wave of flame
    rolling away from the caster, each throwing a fountain of sparks. back: the line runs up-left (-x) instead
    of down-right; mirrored, the two cover all four directions."""
    s = Fire(25)
    for k, (cx, cy) in enumerate(LINE):
        c = (-cx, cy) if back else (cx, cy)
        t0 = k * .08
        s.burst(t0, c, 9, 60, (200, 320), 50, (.15, .3))
        for i in range(16):
            s.burst(t0 + .02 + i * .025, c, 13, 36, (140, 260), 40, (.45, .8))
        s.embers(t0 + .03, c, 120, 240)
    return s, 1.5, "line_back" if back else "line"


def pokyeom_back():
    return pokyeom(back=True)


# --- earth

STONE = [(168, 158, 142), (128, 118, 104), (96, 87, 76), (70, 63, 55), (50, 44, 38)]
STONE_EDGE = (30, 25, 20)
DUST = [(222, 208, 178), (184, 166, 134)]


def chunk(rng, r):
    """Voxel offsets of an angular rock of radius about r (world units): a box-ish block cut by a dozen random
    planes, so it has flat faces and hard edges."""
    n = int(math.ceil(r * 1.2 / VOX)) + 1
    g = np.mgrid[-n:n + 1, -n:n + 1, -n:n + 1].reshape(3, -1).T.astype(float) * VOX
    keep = np.ones(len(g), bool)
    planes = rng.normal(size=(12, 3))
    planes /= np.linalg.norm(planes, axis=1, keepdims=True)
    for nv, d in zip(planes, rng.uniform(.7, 1.0, 12) * r):
        keep &= g @ nv <= d
    keep &= np.abs(g[:, 2]) <= r * .8
    return g[keep] / VOX


def lump(rng, r, parts=1):
    """An angular rock, or for parts > 1 a heap of them pressed together into one rough block."""
    out = [chunk(rng, r)]
    for _ in range(parts - 1):
        d = rng.normal(size=3) * (1, 1, .5)
        d /= np.linalg.norm(d)
        out.append(chunk(rng, r * rng.uniform(.6, .75)) + d * r * rng.uniform(.45, .6) / VOX)
    return np.unique(np.round(np.vstack(out)), axis=0)


def rot(axis, a):
    axis = axis / np.linalg.norm(axis)
    x, y, z = axis
    c, s, C = math.cos(a), math.sin(a), 1 - math.cos(a)
    return np.array([[c + x * x * C, x * y * C - z * s, x * z * C + y * s],
                     [y * x * C + z * s, c + y * y * C, y * z * C - x * s],
                     [z * x * C - y * s, z * y * C + x * s, c + z * z * C]])


class Earth:
    """Rocks (voxel lumps that tumble and shatter when they land hard), grit (tiny stones) and dust."""

    def __init__(self, seed):
        self.rng = np.random.default_rng(seed)
        self.rocks = []        # dicts: p, v, r, shape, axis, ang, spin, born, guide
        self.grit = Particles(seed + 1)
        self.grit.life = np.zeros(0)
        self.dust = Particles(seed + 2)
        self.dust.life = np.zeros(0)
        self.dust.age = np.zeros(0)
        self.t = 0.0
        self.hit = None

    def rock(self, t, p, v, r, guide=None, frag=False, stay=False, parts=1):
        self.rocks.append({"p": np.array(p, float), "v": np.array(v, float), "r": r, "shape": lump(self.rng, r, parts),
                           "axis": self.rng.normal(size=3), "ang": self.rng.uniform(0, 6.3),
                           "spin": self.rng.uniform(4, 10) * (6 / max(r, 3)), "born": t, "guide": guide,
                           "tone": int(self.rng.integers(0, 2)), "rest": 0.0, "frag": frag, "stay": stay})

    def puff(self, t, at, n, out=90, up=(20, 70), life=(.5, 1.0), ring=0.0):
        a = self.rng.uniform(0, 2 * math.pi, n)
        rr = ring + self.rng.uniform(0, 6, n)
        p = np.column_stack([at[0] + np.cos(a) * rr, at[1] + np.sin(a) * rr, self.rng.uniform(0, 6, n)])
        sp = self.rng.uniform(.3, 1, n) * out
        v = np.column_stack([np.cos(a) * sp, np.sin(a) * sp, self.rng.uniform(*up, n)])
        self.dust.add(t, p, v, life=self.rng.uniform(*life, n), age=np.zeros(n))

    def chips(self, t, at, n, speed=150):
        a = self.rng.uniform(0, 2 * math.pi, n)
        sp = self.rng.uniform(.3, 1, n) * speed
        p = np.column_stack([np.full(n, at[0]), np.full(n, at[1]), np.full(n, max(at[2], 1.0))])
        v = np.column_stack([np.cos(a) * sp * .6, np.sin(a) * sp * .6, self.rng.uniform(.5, 1.2, n) * speed])
        self.grit.add(t, p, v, life=self.rng.uniform(.4, .8, n))

    def step(self, dt):
        self.t += dt
        new = []
        for k in self.rocks:
            if k["born"] > self.t:
                new.append(k)
                continue
            g = k["guide"]
            if g is not None:
                path, release, fling, tau0 = g
                tau = self.t - k["born"]
                if tau < release:
                    k["p"] = path(np.array([tau]))[0] + tau0
                    k["ang"] += k["spin"] * dt
                    new.append(k)
                    continue
                k["v"] = fling
                k["guide"] = None
            k["v"][2] -= G * dt
            k["p"] += k["v"] * dt
            k["ang"] += k["spin"] * dt
            bottom = k["p"][2] - k["r"] * .8
            if bottom < 0:
                k["p"][2] -= bottom
                speed = -k["v"][2]
                if self.hit is None and speed > 150:
                    self.hit = self.t
                if speed > 230 and k["r"] >= 3.5 and not k["frag"]:  # grit and a burst of dust; it shatters
                    self.chips(self.t, k["p"] - (0, 0, k["r"] * .8), int(k["r"] * 2), speed * .4)
                    self.puff(self.t, k["p"], int(k["r"] * (90 if k["stay"] else 30)), out=speed * (.45 if k["stay"] else .35),
                              up=(20, 60), life=(.7, 1.2), ring=k["r"] * .8)
                    pieces = int(self.rng.integers(3, 5))
                    for _ in range(pieces):
                        a = self.rng.uniform(0, 2 * math.pi)
                        out = self.rng.uniform(.1, .25) * speed
                        size = k["r"] * self.rng.uniform(.4, .55) * (.5 if k["stay"] else 1)
                        self.rock(self.t, k["p"] + (math.cos(a) * k["r"] * .6, math.sin(a) * k["r"] * .6, 0),
                                  (math.cos(a) * out, math.sin(a) * out, speed * self.rng.uniform(.15, .35)),
                                  size, frag=True)
                    if not k["stay"]:
                        continue
                    k["frag"] = True  # a boulder that stays: it bedded in and will not burst again
                k["v"][2] = speed * .25 if speed > 60 else 0
                k["v"][:2] *= .6
                k["spin"] *= .5
                if speed > 120:
                    self.puff(self.t, k["p"], int(k["r"] * 5), out=60)
            if k["p"][2] - k["r"] * .8 <= .5:
                k["v"][:2] *= math.exp(-6 * dt)
                k["spin"] *= math.exp(-6 * dt)
                k["rest"] += dt
            new.append(k)
        self.rocks = new
        gr = self.grit
        gr.t = self.t
        on = np.flatnonzero((gr.born <= self.t) & ~gr.dead)
        if len(on):
            gr.v[on, 2] -= G * dt
            gr.p[on] += gr.v[on] * dt
            low = on[gr.p[on, 2] < 0]
            gr.p[low, 2] = 0
            gr.v[low, 2] *= -.3
            gr.v[low, :2] *= .6
            gr.life[on] -= dt
            gr.dead |= gr.life < 0
        du = self.dust
        du.t = self.t
        on = np.flatnonzero((du.born <= self.t) & ~du.dead)
        if len(on):
            du.v[on, :2] *= math.exp(-3.5 * dt)
            du.v[on, 2] = du.v[on, 2] * math.exp(-2.5 * dt) + 12 * dt
            du.p[on] += du.v[on] * dt
            du.p[on, 2] = np.maximum(du.p[on, 2], 0)
            du.age[on] += dt
            du.dead |= du.age > du.life

    def fade(self, k):
        self.grit.fade(k)
        if k > .7:  # rocks still lying there sink into the ground
            for r in self.rocks:
                r["p"][2] -= 60 / FPS * (k - .7) / .3 * 2

    def render(self):
        return render_earth(self)


def render_earth(e):
    """Shadows, then the rocks (faces lit top-left, a rough speckle, a dark outline), the grit as single stones,
    and the dust over them as soft see-through clouds in two tones."""
    w, h = SIZE
    img = np.zeros((h, w, 4), np.uint8)
    keys, tones, centers = [], [], []
    for i, k in enumerate(e.rocks):
        if k["born"] > e.t:
            continue
        R = rot(k["axis"], k["ang"])
        off = (k["shape"] @ R.T) * VOX
        vox = np.floor((k["p"] + off) / VOX).astype(int)
        vox = vox[vox[:, 2] >= 0]
        if not len(vox):
            continue
        keys.append(vox)
        tones.append(np.full(len(vox), k["tone"]) + ((vox[:, 0] * 7 + vox[:, 1] * 13 + vox[:, 2] * 5 + i) % 5 == 0))
        centers.append(k["p"])
    gr = e.grit
    gon = (gr.born <= e.t) & ~gr.dead
    if gon.any():
        g = np.floor(gr.p[gon] / VOX).astype(int)
        keys.append(g)
        tones.append(np.ones(len(g), int))
    if centers:
        shadows(img, np.array(centers), 2)
    if keys:
        allk = np.vstack(keys)
        allt = np.concatenate(tones)
        body, col, _ = zbuffer(allk, lambda face, vox: allt[vox] + face, SIZE)
        ramp = np.array(STONE)
        img[body, :3] = ramp[np.clip(col[body], 0, len(STONE) - 1)]
        img[body, 3] = 255
        edge = ring(body)
        img[edge, :3] = STONE_EDGE
        img[edge, 3] = 255
    du = e.dust
    don = (du.born <= e.t) & ~du.dead
    if don.any():
        dp = du.p[don]
        fresh = 1 - du.age[don] / du.life[don]
        idx = np.floor(dp / VOX).astype(int)
        lo = idx.min(0) - 5
        grid = np.zeros(idx.max(0) - lo + 6, np.float32)
        np.add.at(grid, tuple((idx - lo).T), fresh)
        dens = gaussian_filter(grid, 2.0)
        at = np.argwhere(dens > .06)
        if len(at):
            d = dens[tuple(at.T)]
            mask, face, thick = zbuffer(at + lo, lambda f, vox: f, SIZE)
            a = np.clip(90 + thick * 5, 0, 225)
            a = (a // 45 * 45).astype(np.uint8)
            m = mask & (a > 0)
            col = np.where(face == 0, 0, 1)
            cur = img[m, :3].astype(float)
            al = a[m, None] / 255.0
            dc = np.array(DUST)[col[m]]
            under = img[m, 3] > 0
            mix = np.where(under[:, None], cur * (1 - al) + dc * al, dc)
            img[m, :3] = mix.astype(np.uint8)
            img[m, 3] = np.where(under, 255, a[m])
    return img


def nakseok():
    """낙석: a rock drops on the target from above and shatters."""
    e = Earth(31)
    e.rock(0, (0, 0, 150), (0, 0, -120), 9)
    return e, 1.1, "single"


def sansatae():
    """산사태: rocks tumble down on the target one after another from up-left, shattering in a cloud of dust."""
    e = Earth(32)
    for i in range(6):
        src = (-60 + e.rng.normal(0, 8), e.rng.normal(0, 10), 150 + e.rng.normal(0, 15))
        dst = (e.rng.normal(0, 6), e.rng.normal(0, 6), 8)
        e.rock(i * .07, src, aim(src, dst, .32), e.rng.uniform(4.5, 7))
    return e, 1.3, "single"


def jijin():
    """지진: dust bursts up out of the ground in all five cells of the cross while stones pour down on them."""
    e = Earth(33)
    for k, c in enumerate(CROSS):
        for i in range(4):
            e.puff(k * .03 + i * .08, c, 60, out=50, up=(40, 110), life=(.6, 1.1), ring=6)
        for i in range(6):
            src = (c[0] + e.rng.normal(0, 8), c[1] + e.rng.normal(0, 8), 80 + e.rng.uniform(0, 70))
            e.rock(k * .03 + i * .07 + e.rng.uniform(0, .05), src, (0, 0, -80), e.rng.uniform(2.5, 4.2))
    return e, 1.4, "cross"


def geoam():
    """거암: a huge boulder falls out of the sky on the target and lands with a bang in a ring of dust."""
    e = Earth(34)
    e.rock(0, (0, 0, 230), (0, 0, -300), 15, stay=True, parts=3)
    return e, 1.3, "single"


def jiryong():
    """지룡: a whirl of stones and dust twists up into a column over the centre, then the stones crash down on the
    five cells."""
    e = Earth(35)
    path = helix(85, 32)
    for i in range(46):
        a = e.rng.uniform(0, 2 * math.pi)
        tau0 = np.array([math.cos(a) * 3, math.sin(a) * 3, e.rng.normal(0, 3)])
        fling = over_cells(e.rng, 1)[0] + (0, 0, 80)
        e.rock(i * .0065, (0, 0, 0), (0, 0, 0), e.rng.uniform(2.5, 4.5), (path, .6, fling, tau0))
    for i in range(10):
        e.puff(i * .05, (0, 0), 30, out=90, up=(30, 90), life=(.5, .9), ring=30)
    return e, 1.5, "cross"


SPELLS = {f.__name__: f for f in (tak, gyeok, sujin, suryong, haeil, choyeol, eophwa, hwajin, hwaryong, pokyeom,
                                   pokyeom_back, nakseok, sansatae, jijin, geoam, jiryong)}
# the sprite each bakes into (assets/graphics/effects/spells): spell_<skill id in skills.json>
SKILL = {"tak": "torrent", "gyeok": "rapids", "sujin": "water_array", "suryong": "water_dragon", "haeil": "tsunami",
         "choyeol": "scorch", "eophwa": "blaze", "hwajin": "fire_array", "hwaryong": "fire_dragon",
         "pokyeom": "inferno", "pokyeom_back": "inferno_back", "nakseok": "rockfall", "sansatae": "landslide",
         "jijin": "earthquake", "geoam": "boulder", "jiryong": "earth_dragon"}


ELEMENT_SLOW = 2.0  # water, fire and earth spells bake in 2x slow motion: at real speed they were over too soon


def run(name, slow=1.0):
    return simulate(SPELLS[name], slow)


def simulate(spell, slow=1.0):
    """Runs a spell function's sim to its end: (frames, layout); run.hit is when its blow lands. slow plays it in
    slow motion: slow times the frames, each stepping the sim by 1/slow of a frame."""
    sim, secs, layout = spell()
    frames = []
    n = int(secs * FPS * slow)
    for f in range(n):
        sim.fade(f / n)
        for _ in range(SUB):
            sim.step(1 / (FPS * SUB * slow))
        frames.append(sim.render())
    run.hit = (sim.hit if sim.hit is not None else .1) * slow
    return frames, layout


def sheet_unit():
    im = Image.open(ROOT / "internal/sprite/testdata/units/light_infantry.png").convert("RGBA")
    fw, fh = im.width // 24, im.height // 8

    def frame(row):
        f = im.crop((0, row * fh, fw, (row + 1) * fh))
        return f, f.getbbox()[3]
    return frame


def crop_box(frames):
    ys, xs = [], []
    for fr in frames:
        a = np.argwhere(fr[:, :, 3] > 0)
        if len(a):
            ys += [a[:, 0].min(), a[:, 0].max()]
            xs += [a[:, 1].min(), a[:, 1].max()]
    ox, oy = ORIGIN
    x0, x1 = min(xs + [ox - 140]) - 6, max(xs + [ox + 80]) + 6
    y0, y1 = min(ys + [oy - 100]) - 6, max(ys + [oy + 40]) + 6
    return max(x0, ox - 190), max(y0, oy - 240), min(x1, ox + 200), min(y1, oy + 90)


def compose(name, frames, layout, out):
    unit_frame = sheet_unit()
    target, tfoot = unit_frame(1)
    caster, cfoot = unit_frame(4)
    cells, (cx0, cy0) = LAYOUTS[layout]
    x0, y0, x1, y1 = crop_box(frames)
    ox, oy = ORIGIN
    gif = []
    for fr in frames:
        bg = Image.new("RGBA", SIZE, (86, 118, 60, 255))
        d = ImageDraw.Draw(bg)
        for cx, cy in cells:
            px, py = ox + cx - cy, oy + (cx + cy) / 2
            d.polygon([(px, py - 16), (px + 32, py), (px, py + 16), (px - 32, py)], outline=(120, 150, 92, 255))
        csx, csy = ox + cx0 - cy0, oy + (cx0 + cy0) / 2
        bg.alpha_composite(caster, (int(csx - caster.width / 2), int(csy - cfoot)))
        bg.alpha_composite(target, (int(ox - target.width / 2), int(oy - tfoot)))
        bg.alpha_composite(Image.fromarray(fr, "RGBA"))
        bg = bg.crop((x0, y0, x1, y1))
        gif.append(bg.resize((bg.width * 3, bg.height * 3), Image.NEAREST).convert("RGB"))
    gif[0].save(out / f"{name}.gif", save_all=True, append_images=gif[1:], duration=1000 // FPS, loop=0)
    cols = 5
    fw, fh = (x1 - x0) * 2, (y1 - y0) * 2
    sheet = Image.new("RGB", (fw * cols, fh * ((len(gif) + cols - 1) // cols)), (40, 40, 40))
    for i, g in enumerate(gif):
        sheet.paste(g.resize((fw, fh), Image.NEAREST), ((i % cols) * fw, (i // cols) * fh))
    sheet.save(out / f"{name}.png")


def baked():
    """{sprite: () -> (frames, hit, trim)} for every look the game plays: this file's spells, the gas and seal
    spells picked from spellsim_gas / spellsim_seal, 혼란 and 대혼란 from spellsim_confuse with the status marks the
    game turns round the head of a unit under a status (status_mark_<status>), and 팔진도 from spellsim_formation as three pieces (the formation
    drawn on the ground under the units, and an ally and an enemy piece the game puts on each cell it reaches). trim
    drops a piece's empty lead-in, since the game starts it at the formation's hit."""
    import spellsim_confuse as confuse
    import spellsim_formation as formation
    import spellsim_gas as gas
    import spellsim_seal as seal
    out = {}
    for name in SPELLS:
        def element(name=name):
            frames, _ = run(name, ELEMENT_SLOW)
            return frames, run.hit, False
        out["spell_" + SKILL[name]] = element
    picked = {"poison_mist": gas.dokyeon_b, "poison_fog": gas.dokmu_b, "seal_spell": seal.bongchaek_c,
              "root_spell": seal.bongswae_a, "disarm_spell": seal.bonggyeok_c}
    for skill, fn in picked.items():
        def status(fn=fn):
            frames, _ = simulate(fn)
            return frames, run.hit, False
        out["spell_" + skill] = status
    for skill, name in {"confuse": "honran_q", "great_confusion": "daehonran_q"}.items():
        def marks(name=name):
            frames, _ = confuse.run(name)
            return frames, confuse.run.hit, False
        out["spell_" + skill] = marks
    for sprite, frames in confuse.status_marks().items():
        out[sprite] = lambda frames=frames: (frames, 0, False)
    out["spell_eightfold"] = lambda: (formation.ground_frames("b"), .55, False)
    for side in ("ally", "enemy"):
        out[f"spell_eightfold_{side}"] = lambda side=side: (formation.piece_frames("b", side), 0, True)
    return out


def bake(only=()):
    """Writes each look to its own sheet, assets/graphics/effects/spells/<sprite>.{png,json}: the game reads a
    sheet the first time it plays it, so a battle holds only the looks it uses. only limits it to those sprites."""
    WRAP = 1024
    ox, oy = ORIGIN
    out = ROOT / "assets/graphics/effects/spells"
    out.mkdir(parents=True, exist_ok=True)
    looks = baked()
    if only:
        looks = {k: v for k, v in looks.items() if k in only}
    else:
        for old in out.glob("*.*"):
            old.unlink()
    for sprite, make in looks.items():
        frames, hit, trim = make()
        if trim:
            frames = frames[next(i for i, f in enumerate(frames) if f[:, :, 3].any()):]
        crops, anchors = [], []
        for f in frames:
            ys, xs = np.nonzero(f[:, :, 3])
            if len(xs):
                x0, x1 = max(xs.min(), ox - 190), min(xs.max() + 1, ox + 200)
                y0, y1 = max(ys.min(), oy - 240), min(ys.max() + 1, oy + 90)
            else:
                x0, x1, y0, y1 = ox, ox + 1, oy, oy + 1
            crops.append(f[y0:y1, x0:x1])
            anchors.append([int(ox - x0), int(oy - y0)])
        ms = [round((i + 1) * 1000 / FPS) - round(i * 1000 / FPS) for i in range(len(frames))]
        rects = [None] * len(crops)
        x = y = row_h = width = 0
        for i in sorted(range(len(crops)), key=lambda i: -crops[i].shape[0]):  # shelves, tallest first
            h, w = crops[i].shape[:2]
            if x + w > WRAP:
                x, y, row_h = 0, y + row_h, 0
            rects[i] = [int(x), int(y), int(w), int(h)]
            x += w
            row_h = max(row_h, h)
            width = max(width, x)
        sheet = np.zeros((y + row_h, width, 4), np.uint8)
        for img, (px, py, w, h) in zip(crops, rects):
            sheet[py:py + h, px:px + w] = img
        Image.fromarray(sheet, "RGBA").save(out / f"{sprite}.png", optimize=True)
        entry = {"anchor": anchors[0], "anchors": anchors, "frames": rects, "ms": ms, "hit": round(hit, 3)}
        (out / f"{sprite}.json").write_text(json.dumps({sprite: entry}, indent=1) + "\n")
        print(f"{sprite}: {len(frames)} frames, hit {hit:.2f}s, {width}x{y + row_h}")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("names", nargs="*")
    ap.add_argument("--out", default=str(ROOT / "reports/spellsim"))
    ap.add_argument("--slow", type=float, default=1.0, help="slow motion: this many times the frames")
    ap.add_argument("--bake", action="store_true", help="write every look the game plays to assets/graphics/effects/spells/<sprite>.{png,json}, or only the sprites named")
    args = ap.parse_args()
    if args.bake:
        bake(set(args.names))
        return
    out = Path(args.out)
    out.mkdir(parents=True, exist_ok=True)
    for name in args.names or SPELLS:
        frames, layout = run(name, args.slow)
        compose(name, frames, layout, out)
        print(out / f"{name}.gif")


if __name__ == "__main__":
    main()
