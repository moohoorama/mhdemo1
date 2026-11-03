#!/usr/bin/env python3
"""Gas spell drafts (혼란, 대혼란, 독연, 독무) on top of spellsim: haze and poison gas as particles that rise or
sink, swirl round vortices and slow by drag; drawn as see-through clouds in a gas's own colours with a dark
silhouette edge, plus sparkles (혼란) and bubbles (독).

    uv run --with numpy --with scipy --with pillow python tools/spellsim_gas.py              # every draft
    uv run --with numpy --with scipy --with pillow python tools/spellsim_gas.py honran_a dokmu_b

Each draft writes reports/spellsim/<name>.gif and <name>.png like spellsim.py.
"""
import argparse
import math
import sys
from pathlib import Path

import numpy as np
from scipy.ndimage import gaussian_filter

sys.path.insert(0, str(Path(__file__).resolve().parent))
from spellsim import (CROSS, FPS, G, ORIGIN, ROOT, SIZE, SUB, VOX, Particles, compose, ring,  # noqa: E402
                      screen, zbuffer)

HAZE = {"light": (255, 196, 244), "mid": (232, 122, 222), "deep": (168, 70, 204), "edge": (82, 26, 116),
        "spark": [(255, 255, 255), (255, 150, 236), (255, 236, 120)]}
POISON = {"light": (190, 214, 96), "mid": (146, 92, 168), "deep": (98, 52, 124), "edge": (40, 14, 56),
          "spark": [(236, 255, 150), (176, 220, 80), (110, 60, 130)], "wisp": 20}
HEAD = 40.0  # head height of a standing unit


class Gas(Particles):
    """Gas particles (kind 0) that rise or sink by their lift, swirl round the vortices and slow by drag;
    sparkles (kind 1) held on orbits; bubbles (kind 2) that rise a little from a pool and pop."""

    def __init__(self, seed, pal):
        super().__init__(seed)
        self.age = np.zeros(0)
        self.life = np.zeros(0)
        self.lift = np.zeros(0)
        self.drag = np.zeros(0)
        self.kind = np.zeros(0, int)
        self.pal = pal
        self.vortices = []   # (x, y, h, radius, spin, pull, t0, t1): swirl round (x, y) near height h
        self.emitters = []   # (t0, t1, fn(sim, t, dt)) called every step while active
        self.timed = []      # [t, fn(sim, t), fired]
        self.hit = None

    def add(self, t, p, v, life=(.6, 1.0), lift=0.0, drag=2.0, kind=0):
        n = len(p)
        return super().add(t, p, v, age=np.zeros(n), life=self.rng.uniform(*life, n), lift=np.full(n, lift),
                           drag=np.full(n, drag), kind=np.full(n, kind))

    def cloud(self, t, at, h, n, r, out=20, up=0, life=(.6, 1.0), lift=0.0, drag=2.0, flat=1.0):
        a = self.rng.uniform(0, 2 * math.pi, n)
        rr = r * np.sqrt(self.rng.random(n))
        p = np.column_stack([at[0] + np.cos(a) * rr, at[1] + np.sin(a) * rr, h + self.rng.normal(0, r * .3 * flat, n)])
        sp = self.rng.uniform(.3, 1, n) * out
        v = np.column_stack([np.cos(a) * sp, np.sin(a) * sp, up + self.rng.normal(0, 10, n)])
        p[:, 2] = np.maximum(p[:, 2], 0)
        return self.add(t, p, v, life, lift, drag)

    def at(self, t, fn):
        self.timed.append([t, fn, False])

    def orbit(self, t, centre, h, r, n, speed, life, kind=1, wobble=3.0):
        """n sparkles held on a circle round centre at height h, spaced evenly, for life seconds."""
        for k in range(n):
            ph = 2 * math.pi * k / n

            def path(tau, ph=ph):
                tau = np.asarray(tau)
                a = ph + tau * speed
                rr = r * np.minimum(1, tau / .15 + .3)
                return np.column_stack([centre[0] + np.cos(a) * rr, centre[1] + np.sin(a) * rr,
                                        h + np.sin(tau * 9 + ph * 3) * wobble])
            i = self.add(t, np.zeros((1, 3)), np.zeros((1, 3)), (life, life), kind=kind)
            self.guide(i, path, life, np.zeros((1, 3)), np.zeros((1, 3)))

    def step(self, dt):
        self.t += dt
        for tr in self.timed:
            if not tr[2] and self.t >= tr[0]:
                tr[2] = True
                tr[1](self, self.t)
        for t0, t1, fn in self.emitters:
            if t0 <= self.t < t1:
                fn(self, self.t, dt)
        live = (self.born <= self.t) & ~self.dead
        held = self.steer(dt)
        on = np.flatnonzero(live)
        self.age[on] += dt
        free = np.flatnonzero(live & ~held)
        if len(free):
            p = self.p[free]
            for x, y, h, rad, spin, pull, t0, t1 in self.vortices:
                if not t0 <= self.t < t1:
                    continue
                rel = p[:, :2] - (x, y)
                d = np.maximum(np.linalg.norm(rel, axis=1), 1e-3)
                fall = np.exp(-(d / rad) ** 2) * np.exp(-((p[:, 2] - h) / max(rad, 12)) ** 2)
                tang = np.column_stack([-rel[:, 1], rel[:, 0]]) / d[:, None]
                want = tang * spin - rel / d[:, None] * pull
                self.v[free, :2] += (want - self.v[free, :2]) * np.minimum(1, 8 * dt * fall)[:, None]
            t = self.t
            noise = np.column_stack([np.sin(p[:, 1] * .13 + t * 5.1) + np.sin(p[:, 2] * .2 - t * 3.7),
                                     np.sin(p[:, 0] * .11 - t * 4.3) + np.sin(p[:, 2] * .17 + t * 6.1),
                                     np.sin(p[:, 0] * .09 + p[:, 1] * .07 + t * 3.3)]) * 60 * dt
            bub = self.kind[free] == 2
            self.v[free] += np.where(bub[:, None], 0, noise)
            self.v[free, 2] += np.where(bub, 40, self.lift[free]) * dt
            self.v[free] *= np.exp(-self.drag[free] * dt)[:, None]
            self.p[free] += self.v[free] * dt
            low = free[self.p[free, 2] < 0]
            self.p[low, 2] = 0
            self.v[low, 2] = np.maximum(self.v[low, 2], 0)
        self.dead |= self.age > self.life

    def render(self):
        on = (self.born <= self.t) & ~self.dead
        return render_gas(self.p[on], self.v[on], self.kind[on], 1 - self.age[on] / self.life[on], self.age[on],
                          self.pal)


def render_gas(p, v, kind, fresh, age, pal):
    """The gas as a see-through cloud: blurred density cut at a low level, the top faces light, the sides mid,
    thick parts deep and nearly solid, thin parts light and faint, a dark edge round the silhouette; sparkles as
    twinkling crosses, bubbles as rimmed dots."""
    w, h = SIZE
    img = np.zeros((h, w, 4), np.uint8)
    g = kind == 0
    if g.any():
        gp, gf = p[g], fresh[g]
        gp = np.vstack([gp] + [gp - v[g] * .025 * k for k in (1, 2)])  # streaks along the swirl
        gf = np.concatenate([gf] * 3) * .7
        gp[:, 2] = np.maximum(gp[:, 2], 0)
        idx = np.floor(gp / VOX).astype(int)
        lo = idx.min(0) - 5
        grid = np.zeros(idx.max(0) - lo + 6, np.float32)
        np.add.at(grid, tuple((idx - lo).T), np.minimum(1, gf * 1.6))
        dens = gaussian_filter(grid, 1.1)
        occ = dens > .14
        occ[:, :, :max(0, -lo[2])] = False
        at = np.argwhere(occ)
        if len(at):
            d = dens[tuple(at.T)]
            tone = np.where(d > .7, 2, np.where(d > .3, 1, 0))
            mask, col, thick = zbuffer(at + lo, lambda f, vox: np.where(f == 0, tone[vox], np.minimum(tone[vox] + 1, 2)),
                                       SIZE)
            ramp = np.array([pal["light"], pal["mid"], pal["deep"]])
            # thin wisps are fainter, thick cloud nearly solid: a few flat alpha steps, no dither
            a = np.clip(50 + thick * 5, 90, 200)
            a = (a // 36 * 36 + 18).clip(0, 200)
            # no outline: the outer band of the cloud is fainter, so it melts into the ground like a gas
            band1 = mask & ring(~mask)
            band2 = mask & ring(~(mask & ~band1)) & ~band1
            a = np.where(band1, a * .45, np.where(band2, a * .72, a)).astype(np.uint8)
            img[mask, :3] = ramp[col[mask]]
            img[mask, 3] = a[mask]
    s = kind == 1
    if s.any():
        sx, sy = screen(p[s])
        for x, y, a in zip(sx.astype(int), sy.astype(int), age[s]):
            on = math.sin(a * 22 + x * .7) > -.3
            if not on or not (2 <= x < w - 2 and 2 <= y < h - 2):
                continue
            img[y, x] = (*pal["spark"][0], 255)
            for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                img[y + dy, x + dx] = (*pal["spark"][1], 255)
            for dx, dy in ((2, 0), (-2, 0), (0, 2), (0, -2)):
                img[y + dy, x + dx] = (*pal["spark"][2], 230)
            for dx, dy in ((1, 1), (-1, -1), (1, -1), (-1, 1)):
                img[y + dy, x + dx] = (*pal["edge"], 160)
    b = kind == 2
    if b.any():
        bx, by = screen(p[b])
        for x, y in zip(bx.astype(int), by.astype(int)):
            if not (2 <= x < w - 3 and 2 <= y < h - 3):
                continue
            for dx, dy in ((-1, 0), (2, 0), (-1, 1), (2, 1), (0, -1), (1, -1), (0, 2), (1, 2)):
                img[y + dy, x + dx] = (*pal["edge"], 255)
            img[y:y + 2, x:x + 2] = (*pal["spark"][0], 255)
    return img


# --- 혼란 (confuse, single)

def honran_a():
    """혼란 A: a haze ring swirls round the target's head, thickening then thinning, sparkles circling through it."""
    s = Gas(41, HAZE)
    s.vortices.append((0, 0, HEAD, 22, 70, 18, 0, 1.3))

    def emit(sim, t, dt):
        n = 2
        a = sim.rng.uniform(0, 2 * math.pi, n)
        p = np.column_stack([np.cos(a) * 12, np.sin(a) * 12, HEAD + sim.rng.normal(0, 3, n)])
        v = np.column_stack([-np.sin(a) * 60, np.cos(a) * 60, sim.rng.normal(0, 6, n)])
        sim.add(t, p, v, (.45, .75), 4, 1.0)
    s.emitters.append((0, .7, emit))
    s.orbit(.05, (0, 0), HEAD + 8, 13, 6, 7, 1.0)
    return s, 1.4, "single"


def honran_b():
    """혼란 B: a funnel of haze comes down from above onto the head, winds into a tight halo and twinkles out."""
    s = Gas(42, HAZE)
    s.vortices.append((0, 0, HEAD + 20, 40, 90, 40, 0, 1.3))

    def emit(sim, t, dt):
        n = 3
        a = sim.rng.uniform(0, 2 * math.pi, n)
        r = 26 * (1 - min(1, t / .5)) + 10
        p = np.column_stack([np.cos(a) * r, np.sin(a) * r, 95 - min(1, t / .5) * 40 + sim.rng.normal(0, 4, n)])
        v = np.column_stack([-np.sin(a) * 50, np.cos(a) * 50, np.full(n, -70.0)])
        sim.add(t, p, v, (.5, .8), -20, 1.2)
    s.emitters.append((0, .55, emit))
    s.orbit(.35, (0, 0), HEAD + 6, 11, 5, -9, .9)
    return s, 1.4, "single"


def honran_c():
    """혼란 C: three ribbons of haze chase each other round the head, leaving fading trails, with sparkles."""
    s = Gas(43, HAZE)

    def emit(sim, t, dt):
        for k in range(3):
            a = t * 9 + k * 2 * math.pi / 3
            r = 13 + 3 * math.sin(t * 5 + k)
            hh = HEAD + 6 * math.sin(t * 6 + k * 2)
            p = np.array([[math.cos(a) * r, math.sin(a) * r, hh]]).repeat(2, 0) + sim.rng.normal(0, 1, (2, 3))
            sim.add(t, p, np.zeros((2, 3)), (.3, .45), 0, 3.0)
    s.emitters.append((0, .85, emit))
    s.orbit(.1, (0, 0), HEAD + 12, 9, 4, 6, 1.0)
    return s, 1.3, "single"


# --- 대혼란 (great confusion, cross)

def _swirl_at(sim, t, c, n_spark=4):
    sim.vortices.append((c[0], c[1], HEAD, 18, 70, 18, t, t + .8))

    def emit(sim2, tt, dt, c=c):
        k = 2
        a = sim2.rng.uniform(0, 2 * math.pi, k)
        p = np.column_stack([c[0] + np.cos(a) * 10, c[1] + np.sin(a) * 10, HEAD + sim2.rng.normal(0, 3, k)])
        v = np.column_stack([-np.sin(a) * 55, np.cos(a) * 55, sim2.rng.normal(0, 6, k)])
        sim2.add(tt, p, v, (.4, .65), 4, 1.0)
    sim.emitters.append((t, t + .45, emit))
    sim.orbit(t, c, HEAD + 8, 10, n_spark, 7, .8)


def daehonran_a():
    """대혼란 A: a wide swirl of haze opens over the centre and spreads; where it reaches each cell a small swirl
    with sparkles winds round that unit's head."""
    s = Gas(51, HAZE)
    s.vortices.append((0, 0, 30, 50, 110, -30, 0, .5))
    s.cloud(0, (0, 0), 30, 300, 8, out=110, up=10, life=(.4, .6), lift=0, drag=1.5)
    for c in CROSS:
        d = math.hypot(*c)
        s.at(.12 + d / 32 * .18, lambda sim, t, c=c: _swirl_at(sim, t, c))
    return s, 1.6, "cross"


def daehonran_b():
    """대혼란 B: a lilac mist slides low over the five cells, then a swirl rises from it round each head in turn."""
    s = Gas(52, HAZE)
    for c in CROSS:
        s.cloud(0, c, 4, 90, 18, out=25, up=0, life=(.7, 1.1), lift=-10, drag=2.5, flat=.2)
    for k, c in enumerate(CROSS):
        s.at(.2 + k * .07, lambda sim, t, c=c: _swirl_at(sim, t, c, 3))
    return s, 1.6, "cross"


def daehonran_c():
    """대혼란 C: sparkles shower down on the five heads and burst into haze rings that turn and fade."""
    s = Gas(53, HAZE)
    for k, c in enumerate(CROSS):
        t0 = k * .05
        p = np.column_stack([c[0] + s.rng.normal(0, 6, 6), c[1] + s.rng.normal(0, 6, 6), 110 + s.rng.uniform(0, 30, 6)])
        s.add(t0, p, np.column_stack([np.zeros(6), np.zeros(6), np.full(6, -160.0)]), (.38, .42), -300, 0, kind=1)
        s.at(t0 + .35, lambda sim, t, c=c: (sim.cloud(t, c, HEAD, 120, 6, out=50, up=0, life=(.5, .8), lift=0,
                                                      drag=2.0, flat=.3),
                                            sim.vortices.append((c[0], c[1], HEAD, 16, 60, 10, t, t + .8)),
                                            sim.orbit(t, c, HEAD + 8, 10, 3, 8, .7)))
    return s, 1.5, "cross"


# --- 독연 (poison mist, single)

def _poison_pool(sim, t, at, n=260, r=8, out=55):
    sim.cloud(t, at, 22, n, r, out=out, up=30, life=(.8, 1.3), lift=-60, drag=2.2)
    for k in range(12):
        tb = t + .15 + k * .06
        p = np.array([[at[0] + sim.rng.normal(0, 12), at[1] + sim.rng.normal(0, 12), 1.0]])
        sim.add(tb, p, np.zeros((1, 3)), (.25, .45), 0, 0, kind=2)


def dokyeon_a():
    """독연 A: a ball of poison gas lobs from the caster and bursts round the target, the heavy gas sinking into a
    pool round his feet that bubbles."""
    s = Gas(61, POISON)
    src = np.array([-90.0, 0.0, 30.0])
    dst = np.array([0.0, 0.0, 24.0])
    n = 160
    idx = s.add(0, np.zeros((n, 3)), np.zeros((n, 3)), (1.2, 1.5), -40, 1.5)

    def path(tau):
        u = np.clip(np.asarray(tau) / .32, 0, 1)[:, None]
        return src + (dst - src) * u + np.column_stack([np.zeros(len(u)), np.zeros(len(u)), 40 * u[:, 0] * (1 - u[:, 0])])
    off = s.rng.normal(0, 3, (n, 3))
    a = s.rng.uniform(0, 2 * math.pi, n)
    fling = np.column_stack([np.cos(a) * 70, np.sin(a) * 70, s.rng.uniform(-20, 40, n)])
    s.guide(idx, path, .32, off, fling)
    s.at(.32, lambda sim, t: _poison_pool(sim, t, (0, 0)))
    s.hit = .32
    return s, 1.6, "single"


def dokyeon_b():
    """독연 B: a glob splashes at the target's feet and poison gas billows up round him to the head, then sinks."""
    s = Gas(62, POISON)
    s.cloud(.05, (0, 0), 2, 360, 6, out=40, up=150, life=(.8, 1.2), lift=-170, drag=1.8)
    s.vortices.append((0, 0, 20, 20, 35, 8, .1, 1.0))
    for k in range(14):
        p = np.array([[s.rng.normal(0, 12), s.rng.normal(0, 12), 1.0]])
        s.add(.3 + k * .05, p, np.zeros((1, 3)), (.25, .45), 0, 0, kind=2)
    return s, 1.5, "single"


def dokyeon_c():
    """독연 C: a ribbon of poison smoke snakes in from the caster, coils round the target and settles over him."""
    s = Gas(63, POISON)

    def emit(sim, t, dt):
        u = min(1, t / .35)
        if u < 1:
            x, y, hh = -90 + 90 * u, 8 * math.sin(u * 9), 28 + 10 * math.sin(u * 5)
        else:
            a = (t - .35) * 14
            r = max(4, 14 - (t - .35) * 18)
            x, y, hh = math.cos(a) * r, math.sin(a) * r, 34 - (t - .35) * 40
        p = np.array([[x, y, hh]]).repeat(5, 0) + sim.rng.normal(0, 1.5, (5, 3))
        sim.add(t, p, np.zeros((5, 3)), (.5, .8), -25, 3.0)
    s.emitters.append((0, .8, emit))
    s.at(.8, lambda sim, t: _poison_pool(sim, t, (0, 0), 140, 6, 35))
    return s, 1.7, "single"


# --- 독무 (poison fog, cross)

def dokmu_a():
    """독무 A: a low bank of poison fog rolls in from the caster's side across the five cells, puffing up to head
    height over each as it passes."""
    s = Gas(71, POISON)

    def emit(sim, t, dt):
        n = 10
        p = np.column_stack([np.full(n, -70.0) + sim.rng.normal(0, 4, n), sim.rng.uniform(-44, 44, n),
                             sim.rng.uniform(0, 10, n)])
        v = np.column_stack([sim.rng.normal(130, 20, n), sim.rng.normal(0, 15, n), np.zeros(n)])
        sim.add(t, p, v, (.9, 1.3), -30, 1.1)
    s.emitters.append((0, .35, emit))
    for c in CROSS:
        s.at(.25 + (c[0] + 32) / 32 * .14, lambda sim, t, c=c: (
            sim.cloud(t, c, 4, 120, 8, out=30, up=120, life=(.7, 1.0), lift=-140, drag=1.8),
            [sim.add(t + .1 + k * .07, np.array([[c[0] + sim.rng.normal(0, 10), c[1] + sim.rng.normal(0, 10), 1.0]]),
                     np.zeros((1, 3)), (.25, .4), 0, 0, kind=2) for k in range(4)]))
    return s, 1.8, "cross"


def dokmu_b():
    """독무 B: poison fog wells up from the ground in the centre and spreads low and outward over the cross, then
    rises in a sickly column over each cell."""
    s = Gas(72, POISON)
    s.cloud(0, (0, 0), 3, 500, 6, out=150, up=10, life=(.9, 1.3), lift=-30, drag=2.8, flat=.2)
    for c in CROSS:
        d = math.hypot(*c)
        s.at(.15 + d / 32 * .15, lambda sim, t, c=c: (
            sim.cloud(t, c, 4, 130, 7, out=25, up=130, life=(.7, 1.0), lift=-150, drag=1.8),
            sim.vortices.append((c[0], c[1], 25, 16, 30, 6, t, t + .8)),
            [sim.add(t + .15 + k * .07, np.array([[c[0] + sim.rng.normal(0, 10), c[1] + sim.rng.normal(0, 10), 1.0]]),
                     np.zeros((1, 3)), (.25, .4), 0, 0, kind=2) for k in range(3)]))
    return s, 1.8, "cross"


def dokmu_c():
    """독무 C: poison drips from a dark cloud that gathers above the cross; where it falls each cell steams with
    gas that pools and bubbles."""
    s = Gas(73, POISON)
    s.vortices.append((0, 0, 100, 40, 50, 25, 0, .6))
    s.cloud(0, (0, 0), 100, 400, 26, out=20, up=0, life=(.6, .8), lift=0, drag=2.5, flat=.4)
    for c in CROSS:
        p = np.column_stack([c[0] + s.rng.normal(0, 4, 20), c[1] + s.rng.normal(0, 4, 20), 95 + s.rng.normal(0, 4, 20)])
        s.add(.4 + s.rng.uniform(0, .1), p, np.zeros((20, 3)), (.33, .36), -G, 0, kind=0)
        s.at(.75, lambda sim, t, c=c: _poison_pool(sim, t, c, 110, 6, 40))
    return s, 1.9, "cross"


SPELLS = {f.__name__: f for f in (honran_a, honran_b, honran_c, daehonran_a, daehonran_b, daehonran_c,
                                   dokyeon_a, dokyeon_b, dokyeon_c, dokmu_a, dokmu_b, dokmu_c)}


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
    ap.add_argument("--out", default=str(ROOT / "reports/spellsim"))
    args = ap.parse_args()
    out = Path(args.out)
    out.mkdir(parents=True, exist_ok=True)
    for name in args.names or SPELLS:
        frames, layout = run(name)
        compose(name, frames, layout, out)
        print(out / f"{name}.gif")


if __name__ == "__main__":
    main()
