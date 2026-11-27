#!/usr/bin/env python3
"""The three-second flood (수공 3초): a dam of wooden planks and earth bursts in a gorge of the map's own cliff
tiles and the water runs down the river onto the army wading its ford. The water is a height field (the shallow
water equations on the battle map's ground), drawn as iso pixel art and baked for the game
(internal/gui/field_surge.go, demo `-ui suigong`). The cliffs are the game's own tiles and are not drawn here (the
sim only knows them as walls); the dam is built of voxels, and the same voxels fly apart when it bursts.

It plays as two shots: the dam up to T_CUT, then the ford. The whole river from the gorge to below the ford is
simulated at the game map's own distances (GAP 0), so the ford shot at the game's 2x zoom shows the flood coming
down the river from its upper right edge instead of a slab cut off where the simulation used to end.

    uv run --with numpy --with scipy --with pillow python tools/floodsim.py            # preview: reports/floodsim/
    uv run --with numpy --with scipy --with pillow python tools/floodsim.py --bake     # into the game

World units are spellsim.py's: x, y run along the map's cell axes u, v, 32 units to a cell, h is screen pixels
up, so a point lands on the field's canvas at (x - y, (x + y) / 2 - h) from the map's origin. The layout below is
the demo's map (uidemo_suigong.go): keep the two the same.

- sim: h, hu, hv on a 2-unit grid over a bed (river channel, ford, banks, the cliffs as walls, the dam), first
  order finite volumes with hydrostatic reconstruction (Audusse et al. 2004) and Rusanov fluxes, so the still
  river stays still and the banks flood and dry. The dam is bed that sinks from its middle outwards when it bursts.
- dam: 4-unit voxels, earth inside and boards (courses of planks, posts) on its faces. At the burst the voxels
  of a plank (a strip of up to two cells' length) let go together as one rigid box, each clod and post alone, with a
  velocity of its own. The water carries them (drag toward the flow) and turns them: yaw from the shear across a
  plank's length, a pull of its long way along the stream and the water's wobble; pitch and roll settle on the
  surface slope with wobble while afloat and spin as thrown while airborne. Boards float on and ride the flow,
  earth sinks and is swallowed. Drawn as rotated boxes (flat faces by their normal, planks outlined); the limit is
  that bodies do not collide with each other or the banks and a plank's turning is a model, not a rigid-body solve.
- look: one dot (2x2 canvas pixels) to a grid cell, painted far to near as columns from the surface down to what
  the water stands on, so a wave front and the fall over the breach show their faces. Tones are flat steps of the
  river's colour; foam is a field the flow carries, made where the water rises fast or runs shallow and fast.
- depth order: each frame is cut into bands, one per diagonal of cells (u + v), which the game draws among the
  units by that depth. Still river that the flood has not reached is left to the game's own water.
- with it: how deep each spot is under water frame by frame (a unit standing there is drawn that deep in it), and
  the spray where the front breaks on the ford (spellsim's particle water, drawn over everything).
"""
import argparse
import json
import math
import os
import sys
import time
import zlib
from pathlib import Path

import numpy as np
from PIL import Image, ImageDraw
from scipy.ndimage import gaussian_filter, laplace, map_coordinates
from scipy.spatial import ConvexHull

sys.path.insert(0, str(Path(__file__).resolve().parent))
import spellsim  # noqa: E402

T0 = time.time()
ROOT = Path(__file__).resolve().parents[1]
OUT = Path(os.environ.get("FLOOD_OUT", ROOT / "assets/graphics/effects/flood"))

# the demo's map (uidemo_suigong.go)
RIVER, WIDTH, DAM, FORD = 12, 6, 8, 30   # the river's first column and its width, the dam's row, the ford's first row
CELL = 32.0
U0, U1, V1 = 8, 24, 50
V0 = int(os.environ.get("FLOOD_V0", -20))  # the cells simulated: the pool behind the dam reaches up to V0, long enough
                                            # that the front still stands up as a wall when it has come the 22 cells to the ford
RES = float(os.environ.get("FLOOD_RES", 2))
RES = int(RES) if RES == int(RES) else RES   # a fraction (0.5) only for --probe, which builds no dam   # dots per old dot, along each side: 1 is the first bake (2 canvas pixels a dot), 2 halves the dot
DX = 2.0 / RES                      # grid step: one dot, which is also DX canvas pixels wide on the map's canvas
NX, NY = int((U1 - U0) * CELL / DX), int((V1 - V0) * CELL / DX)

GRAV = 1500.0                       # tuned so the front runs from the dam to the ford in about 1.3 s
REST = -9.0                         # the river's surface (field_water.go sunkDepth)
POOL = float(os.environ.get("FLOOD_POOL", 22.0))   # the surface behind the dam
BED, FORD_BED, FORD_TOP = -15.0, -6.0, -5.0
CLIFF, DAM_TOP = 26.0, 25.0         # the cliff tiles' height (assets/graphics/index.json cliff_rise) and the dam's crest
T_BREAK, FPS = 1.0, 30              # simulated seconds: the dam bursts at T_BREAK
T_END = float(os.environ.get("FLOOD_END", 5.8))   # the last frame: the front reaches the ford at about 4.7 s
T_CUT, GAP = 1.9, 0                 # the shot changes from the dam to the ford, GAP cells down the game's map
# slow motion: the sim is played slower than it runs, so each shot is baked with more frames than FPS gives, and each
# frame shows for longer. (shot's simulated seconds, frames per 1/FPS second, seconds it plays for) per shot
FORD_SLOW = 4.4 / 1.88 * 1.0        # seconds the ford shot is baked to play for, per simulated second (the game plays it 2x)
SLOW = ((T_CUT - T_BREAK, 4, 2.5), (T_END - T_CUT, 2.4, FORD_SLOW * (T_END - T_CUT)))
SHOTS = ((V0, V1), (V0, V1))        # the rows each shot can show: all of them, at 2x zoom a shot sees most of the river
CAMS = ((RIVER + WIDTH / 2, DAM), (RIVER + WIDTH / 2, FORD + .8))   # preview: the cell each shot looks at
EPS = .05
CREST = 10.0                        # pixels the foaming front is drawn above the height field

VS0 = 4                             # a dam body's side: what a clod is and how thick a plank is, world units
VS = VS0 / RES                      # a dam voxel's side: a body is RES voxels thick each way
TS = 4                              # a voxel's picture is TS x TS dots
DAM_X0, DAM_Y0 = RIVER * CELL, (DAM + .125) * CELL
DAM_NI = int(WIDTH * CELL / VS0)
DAM_T = (10, 9, 8, 7, 6, 5, 4, 3, 3, 3)   # the dam's thickness in voxels layer by layer: straight on the pool, stepped below
DAM_NK = len(DAM_T)

PAL = np.array([  # deep, body, light, foam, white; then the same body tones for thin water over ground
    (24, 74, 92, 255), (44, 108, 119, 255), (88, 150, 160, 255), (176, 216, 222, 255), (248, 252, 255, 255),
    (44, 108, 119, 120), (60, 124, 134, 185),
    (222, 178, 100, 255), (190, 138, 70, 255), (138, 94, 48, 255), (84, 54, 28, 255),    # boards: top, left, right, edge
    (187, 158, 106, 255), (152, 121, 75, 255), (120, 93, 57, 255), (72, 54, 34, 255),   # earth: top, left, right, edge (the cliff tiles' sand)
    (244, 214, 140, 255), (112, 72, 38, 255),                                             # board grain: end, post
], np.uint8)
DEEP, BODY, LIGHT, FOAM, WHITE, FILM, THIN, W_TOP, W_LEFT, W_RIGHT, W_EDGE, E_TOP, E_LEFT, E_RIGHT, E_EDGE, W_END, W_POST = range(17)
GRAINED = (W_TOP, W_LEFT, W_RIGHT, E_TOP, E_LEFT, E_RIGHT)
SHADE = {}      # face tone -> (lighter, darker) variants, appended to PAL
for _c in GRAINED:
    _rgb = PAL[_c][:3].astype(float)
    SHADE[_c] = tuple(len(PAL) + k for k in range(2))
    PAL = np.vstack([PAL, [(*np.clip(_rgb * 1.16 + 6, 0, 255).astype(np.uint8), 255), (*np.clip(_rgb * .8, 0, 255).astype(np.uint8), 255)]]).astype(np.uint8)


def grid():
    x = U0 * CELL + (np.arange(NX) + .5) * DX
    y = V0 * CELL + (np.arange(NY) + .5) * DX
    return np.meshgrid(x, y)


LUMPS = gaussian_filter(np.random.default_rng(2).random((256, 256)), 2.2, mode="wrap")
LUMPS = (LUMPS - LUMPS.min()) / (LUMPS.max() - LUMPS.min())


def dam_layers(y):
    """How many voxel layers of the dam stand over world y (0 off its footprint): it is thick low down and
    steps back layer by layer on the downstream side."""
    j = np.floor((y - DAM_Y0) / VS0).astype(int)
    return sum(((j >= 0) & (j < t)).astype(int) for t in DAM_T)


def terrain():
    """The bed without the dam, the dam's footprint with how far each spot is from its middle (0..1) and its
    crest, what water stands on for drawing (the river's own surface, the ford's stones, the ground), and the
    walls the cliff tiles make on both sides of the gorge."""
    x, y = grid()
    cu, cv = x / CELL, y / CELL
    river = (cu >= RIVER) & (cu < RIVER + WIDTH)
    ford = river & (cv >= FORD)
    bed = np.where(river, BED, 0.0)
    bed[ford] = FORD_BED
    wall = ~river & (cv < DAM + 2)
    bed[wall] = CLIFF
    layers = np.where(river, dam_layers(y), 0)
    dam = layers > 0
    across = np.abs(cu - (RIVER + WIDTH / 2)) / (WIDTH / 2)
    crest = BED + VS0 * layers
    base = np.where(river, REST, bed)
    base[ford] = FORD_TOP
    return bed, dam, across, crest, base, river, ford, wall


def rotmat(a):
    """Rotation matrices of yaw, pitch, roll (a: n x 3): Rz(yaw) Ry(pitch) Rx(roll). Column 0 is a body's own x axis, the
    long way of a plank; positive pitch dips that end, positive roll lifts the body's +y side."""
    cy, sy, cp, sp, cr, sr = np.cos(a[:, 0]), np.sin(a[:, 0]), np.cos(a[:, 1]), np.sin(a[:, 1]), np.cos(a[:, 2]), np.sin(a[:, 2])
    R = np.empty((len(a), 3, 3))
    R[:, 0] = np.stack([cy * cp, cy * sp * sr - sy * cr, cy * sp * cr + sy * sr], 1)
    R[:, 1] = np.stack([sy * cp, sy * sp * sr + cy * cr, sy * sp * cr - cy * sr], 1)
    R[:, 2] = np.stack([-sp, cp * sr, cp * cr], 1)
    return R


class Dam:
    """The dam as voxels: earth inside, boards on its faces (courses of planks laid along the river's width, posts
    up its downstream face). While it stands each voxel is a cube of the picture. As it bursts the boards' voxels
    of one plank (a strip of up to four) let go together as one rigid body, every earth clod and post by itself; the
    water carries each off and turns it: yaw, pitch and roll (see step)."""

    EARTH, BOARD, POST = 0, 1, 2

    def __init__(self, seed=5):
        r = np.random.default_rng(seed)
        ijk = [(i, j, k) for k in range(DAM_NK) for j in range(DAM_T[k]) for i in range(DAM_NI)]
        self.ijk = np.array(ijk)
        i, j, k = self.ijk.T
        thick = np.array(DAM_T)
        top = np.append(thick, 0)[k + 1] <= j            # nothing above it
        down = thick[k] - 1 == j                          # on the downstream face
        mat = np.full(len(i), self.EARTH)
        strip = (i + 3 * ((j + k) % 3)) // 4
        mat[(top & ((strip + k) % 2 == 0)) | (top & (k == DAM_NK - 1))] = self.BOARD      # boards laid over the treads and the crest
        mat[down & (k % 3 == 1)] = self.BOARD                                              # a course of boards on every third riser
        mat[down & (i % 12 == 5) & (k % 3 != 1)] = self.POST                              # posts between them
        self.mat = mat
        self.strip, self.i = strip, i
        self.pos = np.stack([DAM_X0 + (i + .5) * VS0, DAM_Y0 + (j + .5) * VS0, BED + (k + .5) * VS0], 1)   # bodies as they stand
        across = np.abs(self.pos[:, 0] / CELL - (RIVER + WIDTH / 2)) / (WIDTH / 2)
        n = len(i)
        self.n = n
        wait = T_BREAK + .13 * across + .02 * (DAM_NK - 1 - k) + r.uniform(0, .05, n)
        stripkey = self.strip + 100 * k + 10000 * (j % 3)
        _, inv = np.unique(stripkey, return_inverse=True)
        sv = r.uniform(0, 1, (inv.max() + 1, 3))
        lift = k / (DAM_NK - 1)
        side = np.sign(self.pos[:, 0] - (DAM_X0 + WIDTH * CELL / 2))
        toss = np.stack([
            side * (20 + 40 * sv[inv, 0]) + r.normal(0, 16, n),
            50 + 110 * sv[inv, 1] + r.normal(0, 34, n) + 20 * lift,
            20 + 110 * lift * sv[inv, 2] + r.uniform(0, 40, n),
        ], 1)
        rate = r.uniform(3, 7, n)
        self.tone = r.integers(0, 4, n)                  # which of the voxel's few variants it wears
        # bodies: a plank's voxels (one strip of one course) together, everything else alone
        gid = np.arange(n)
        bd = mat == self.BOARD
        course = down & (k % 3 == 1)                       # a whole course of boards: its planks are two strips long
        plank = np.where(course, 1000 + strip // 2, strip)
        _, planks = np.unique(np.stack([plank[bd], j[bd], k[bd]], 1), axis=0, return_inverse=True)
        gid[bd] = n + planks.reshape(-1)
        _, self.vb = np.unique(gid, return_inverse=True)
        self.vb = self.vb.reshape(-1)
        nb = self.vb.max() + 1
        self.nb = nb
        cnt = np.bincount(self.vb, minlength=nb)

        def mean(a):
            return np.stack([np.bincount(self.vb, a[:, c], nb) for c in range(a.shape[1])], 1) / cnt[:, None] if a.ndim > 1 else np.bincount(self.vb, a, nb) / cnt

        self.bpos = mean(self.pos)
        self.off = self.pos - self.bpos[self.vb]          # each voxel from its body's centre, in the body's frame at rest
        self.bmat = np.zeros(nb, int)
        self.bmat[self.vb] = mat
        self.ex = np.zeros(nb)
        np.maximum.at(self.ex, self.vb, np.abs(self.off[:, 0]))
        self.ex += VS0 / 2                                 # half the length of a plank
        self.wait, self.toss, self.rate = mean(wait), mean(toss), mean(rate)
        self.free = np.zeros(nb, bool)
        self.gone = np.zeros(nb, bool)
        self.vel = np.zeros((nb, 3))
        self.age = np.zeros(nb)
        self.ang = np.zeros((nb, 3))                       # yaw, pitch, roll
        wood = self.bmat != self.EARTH
        self.om = r.normal(0, 1, (nb, 3)) * np.where(wood, 1.0, 1.6)[:, None] * np.array([4.0, 4.5, 4.5])
        self.wob = np.stack([r.uniform(4, 9, nb), r.uniform(0, 6.3, nb), r.uniform(0, 6.3, nb), r.uniform(4, 9, nb)], 1)
        self.subdivide(i, r)

    def subdivide(self, i, r):
        """Cut every body's voxel into RES^3 voxels of VS (the physics above knows only bodies, so nothing moves
        differently): each has its own speckle and the seams of the boards stay as wide as they were."""
        n = len(i)
        sub = np.array([(a, b, c) for a in range(RES) for b in range(RES) for c in range(RES)])
        par = np.repeat(np.arange(n), len(sub))
        a, b, c = np.tile(sub, (n, 1)).T
        pos = self.pos[par] + (np.stack([a, b, c], 1) + .5) * VS - VS0 / 2
        self.vb = self.vb[par]
        self.off = pos - self.bpos[self.vb]
        self.vpos0 = pos
        self.mat = self.mat[par]
        self.seam = (i[par] % 4 == 0) & (a == 0)
        self.tone = np.random.default_rng(7).integers(0, 4, len(par))
        self.n = len(par)

    def cell(self, p):
        return (np.clip(((p[:, 1] - V0 * CELL) / DX).astype(int), 0, NY - 1), np.clip(((p[:, 0] - U0 * CELL) / DX).astype(int), 0, NX - 1))

    def step(self, f, dt):
        late = ~self.free & (f.t >= self.wait)
        self.free |= late
        self.vel[late] = self.toss[late]
        idx = np.nonzero(self.free & ~self.gone)[0]
        if not len(idx):
            return
        u, v = f.velocity()
        eta, base = f.h + f.b, f.base
        p, q = self.bpos[idx], self.vel[idx]
        ang, om = self.ang[idx], self.om[idx]
        j, i = self.cell(p)
        wood = self.bmat[idx] != self.EARTH
        q[:, 2] -= spellsim.G * dt
        rate = self.rate[idx] * np.where(wood, 1.0, .5) * dt
        flow = np.stack([u[j, i], v[j, i]], 1)
        q[:, :2] += (flow - q[:, :2]) * np.minimum(1, rate)[:, None]
        # turning: the flow pushes the two ends of a plank by different amounts (shear), turns its long way along the
        # stream (a log in a current) and tumbles it by the water's own wobble; in the air it spins as it was thrown
        R = rotmat(ang)
        axis = R[:, :, 0]
        ex = self.ex[idx]
        ends = []
        for sgn in (1, -1):
            e = p + sgn * axis * ex[:, None]
            je, ie = self.cell(e)
            ends.append(np.stack([u[je, ie], v[je, ie]], 1))
        ah = axis[:, :2] / np.maximum(np.hypot(axis[:, 0], axis[:, 1]), 1e-3)[:, None]
        perp = np.stack([-ah[:, 1], ah[:, 0]], 1)
        shear = ((ends[0] - ends[1]) * perp).sum(1) / (2 * ex)
        speed = np.hypot(flow[:, 0], flow[:, 1])
        fh = flow / np.maximum(speed, 1e-3)[:, None]
        cross = ah[:, 0] * fh[:, 1] - ah[:, 1] * fh[:, 0]
        align = -9.0 * cross * (ah * fh).sum(1) * np.minimum(1, speed / 60)
        surf = eta[j, i]
        wet = p[:, 2] < surf + VS0 * .5
        swirl = np.clip(speed / 120, 0, 1)
        t = f.t
        w = self.wob[idx]
        yaw_target = 2.5 * shear + align + 4.5 * swirl * np.sin(w[:, 0] * t + w[:, 1])
        om[:, 0] += np.where(wet, (yaw_target - om[:, 0]) * np.minimum(1, 6 * dt), 0)
        gy, gx = np.gradient(eta, DX)
        sgx, sgy = gx[j, i], gy[j, i]
        along = sgx * ah[:, 0] + sgy * ah[:, 1]
        across = sgx * perp[:, 0] + sgy * perp[:, 1]
        want = np.stack([-np.arctan(along) + .45 * swirl * np.sin(w[:, 3] * t + w[:, 2]), np.arctan(across) + .45 * swirl * np.cos(w[:, 0] * t + w[:, 1])], 1)
        k = np.minimum(1, 9 * dt)
        rel = (want - ang[:, 1:]) * k
        ang[:, 1:] = np.where(wet[:, None], ang[:, 1:] + rel, ang[:, 1:] + om[:, 1:] * dt)
        om[:, 1:] = np.where(wet[:, None], rel / dt, om[:, 1:])
        ang[:, 0] += om[:, 0] * dt
        p += q * dt
        p[:, 0] = np.clip(p[:, 0], DAM_X0 + VS0, DAM_X0 + WIDTH * CELL - VS0)       # it stays in the channel
        j, i = self.cell(p)
        surf = eta[j, i]
        age = f.t - self.wait[idx]
        self.age[idx] = age
        float_at = surf - VS0 * .5 - 1                                              # boards ride at the surface
        wet = p[:, 2] < float_at
        rise = wood & wet & (surf - base[j, i] > 2)
        p[rise, 2] += (float_at[rise] - p[rise, 2]) * np.minimum(1, 10 * dt)
        q[rise, 2] *= .5
        sink = ~wood & (p[:, 2] < surf)
        q[sink, 2] = np.minimum(q[sink, 2], 0) - 30                                # earth sinks into the silt
        axz = np.abs(rotmat(ang)[:, 2, 0])
        p[:, 2] = np.maximum(p[:, 2], base[j, i] + VS0 * .5 + ex * axz)
        self.bpos[idx], self.vel[idx], self.ang[idx], self.om[idx] = p, q, ang, om
        dead = (~wood & ((age > .4) | (p[:, 2] + VS0 * .5 < surf - 4))) | (p[:, 1] > (V1 - .3) * CELL) | (f.t > T_END + .5)
        self.gone[idx[dead]] = True

    def world(self):
        """Every voxel's centre and the rotation of its body: where each is now."""
        R = rotmat(self.ang)
        free = self.free[self.vb]
        pos = self.bpos[self.vb] + np.einsum("nij,nj->ni", R[self.vb], self.off)
        return np.where(free[:, None], pos, self.vpos0), R[self.vb]

    def paint(self, tone, band, view):
        """Cubes far to near over the frame: first the ones the water has not let go yet, then the ones it carries,
        each turned as its body is."""
        pos, Rv = self.world()
        alive = ~self.gone[self.vb]
        free = self.free[self.vb]
        cv = pos[:, 1] / CELL
        inview = alive & (cv >= view[0]) & (cv < view[1])
        idx = np.nonzero(inview & ~free)[0]
        key = pos[idx, 0] + pos[idx, 1] + .01 * pos[idx, 2]
        for n in idx[np.argsort(key, kind="stable")]:
            x, y, z = pos[n]
            c0 = int(round((x - y) / DX - TS / 2)) - PX0
            r0 = int(round(((x + y) / 2 - z) / DX - TS / 2)) - PY0
            t = cube_tones(self.mat[n], self.tone[n], self.seam[n], n)
            if TEXTURE:
                texture_cube(t, self.mat[n], pos[n])
            ys, xs = np.nonzero(t >= 0)
            ys, xs = ys + r0, xs + c0
            ok = (ys >= 0) & (ys < tone.shape[0]) & (xs >= 0) & (xs < tone.shape[1])
            tone[ys[ok], xs[ok]] = t[t >= 0][ok]
            band[ys[ok], xs[ok]] = int(x // CELL) + int(y // CELL)
        idx = np.nonzero(inview & free)[0]
        if not len(idx):
            return
        # the turned ones: a plank is one box (outlined, so its turning reads), clods and posts a cube each
        planks = idx[self.bmat[self.vb[idx]] == self.BOARD]
        _, first = np.unique(self.vb[planks], return_index=True)
        singles = idx[self.bmat[self.vb[idx]] != self.BOARD]
        items = [(self.vb[planks[f]], True) for f in first] + [(self.vb[n], False) for n in singles]
        owner = {self.vb[n]: n for n in idx}
        bodies = np.array([b for b, _ in items], int)
        centre = self.bpos[bodies]
        order = np.argsort(centre[:, 0] + centre[:, 1] + .01 * centre[:, 2], kind="stable")
        timg, bimg = Image.fromarray(tone.astype(np.int32)), Image.fromarray(band.astype(np.int32))
        dt_, db_ = ImageDraw.Draw(timg), ImageDraw.Draw(bimg)
        R = rotmat(self.ang)
        for n in order:
            b, plank = items[n]
            self.draw_box(dt_, db_, self.bpos[b], R[b], np.array([self.ex[b], VS0 / 2, VS0 / 2]) if plank else np.full(3, VS0 / 2),
                          self.bmat[b], plank, int(self.bpos[b, 0] // CELL) + int(self.bpos[b, 1] // CELL), int(b) % 997)
        tone[...] = np.array(timg).astype(np.int8)
        band[...] = np.array(bimg).astype(np.int16)

    def draw_box(self, dt, db, c, R, half, mat, outline, bd, seed=0):
        """One box of half sizes `half` turned by R at c: its visible faces as flat polygons, each tone from how the face
        lies to the light; with an outline the box silhouette and edges are dark."""
        corners = np.array([(a, b, d) for a in (-1, 1) for b in (-1, 1) for d in (-1, 1)]) * half
        w = c + corners @ R.T
        px = (w[:, 0] - w[:, 1]) / DX - PX0
        py = ((w[:, 0] + w[:, 1]) / 2 - w[:, 2]) / DX - PY0
        if outline:
            hull = ConvexHull(np.stack([px, py], 1))
            pts = [(px[v], py[v]) for v in hull.vertices]
            dt.polygon(pts, fill=int(W_EDGE))
            db.polygon(pts, fill=bd)
        pull = .6 if outline else .15
        for axis in range(3):
            for sign in (1, -1):
                nrm = R[:, axis] * sign
                if nrm.sum() <= .02:
                    continue
                b = .5 * nrm[2] + .3 * nrm[1] - .1 * nrm[0]
                face = 0 if b >= .38 else 1 if b >= .12 else 2
                col = (EARTH_FACE if mat == self.EARTH else BOARD_FACE if mat == self.BOARD else (W_POST, W_RIGHT, W_EDGE))[face]
                if mat == self.BOARD and axis == 0:
                    col = W_END                                                       # the end grain of a plank
                sel = [q for q in range(8) if corners[q, axis] * sign > 0]
                pts = [(px[q], py[q]) for q in sel]
                cx, cy = sum(p[0] for p in pts) / 4, sum(p[1] for p in pts) / 4
                pts.sort(key=lambda p: math.atan2(p[1] - cy, p[0] - cx))
                pts = [(p[0] + (cx - p[0]) * pull / max(1e-6, math.hypot(cx - p[0], cy - p[1])) if outline else p[0] + (cx - p[0]) * pull,
                        p[1] + (cy - p[1]) * pull / max(1e-6, math.hypot(cx - p[0], cy - p[1])) if outline else p[1] + (cy - p[1]) * pull) for p in pts]
                dt.polygon(pts, fill=int(col))
                db.polygon(pts, fill=bd)
                if col in SHADE and mat != self.POST:
                    self.grain(dt, pts, corners, px, py, sel, axis, half, col, mat == self.EARTH, seed)


    def grain(self, dt, pts, corners, px, py, sel, axis, half, col, earth, seed):
        """A face's wood grain (streaks that run along a plank) or earth speckle over its flat tone: where each pixel
        lies on the face comes from the face's four corners, so the pattern turns with the box."""
        xs, ys = [p[0] for p in pts], [p[1] for p in pts]
        x0, y0, x1, y1 = int(min(xs)), int(min(ys)), int(max(xs)) + 1, int(max(ys)) + 1
        if x1 - x0 < 3 or y1 - y0 < 3:
            return
        mask = Image.new("1", (x1 - x0 + 1, y1 - y0 + 1))
        ImageDraw.Draw(mask).polygon([(x - x0, y - y0) for x, y in pts], fill=1)
        gy, gx = np.nonzero(np.array(mask))
        if not len(gx):
            return
        a1, a2 = [k for k in range(3) if k != axis]
        src = np.stack([px[sel], py[sel], np.ones(len(sel))], 1)
        dst = np.stack([corners[sel, a1], corners[sel, a2]], 1)
        m = np.linalg.lstsq(src, dst, rcond=None)[0]
        loc = (np.stack([gx + x0 + .5, gy + y0 + .5, np.ones(len(gx))], 1) @ m) / DX      # world length from the face's centre, in dots
        lighter, darker = SHADE[col]
        if earth:
            n = noise(np.floor(loc[:, 0] * 1.3), np.floor(loc[:, 1] * 1.3), seed)
            low, high = n < .2, n > .88
        else:
            long, across = (loc[:, 0], loc[:, 1]) if a1 == 0 else (loc[:, 1], loc[:, 0])
            streak = np.floor(across / 1.7)
            run = noise(streak, np.floor((long + 7 * noise(streak, np.zeros_like(streak), seed + 1) * 9) / 6), seed)
            low, high = run < .2, run > .86
        for sel_, tone in ((low, darker), (high, lighter)):
            if sel_.any():
                dt.point([(int(x + x0), int(y + y0)) for x, y in zip(gx[sel_], gy[sel_])], fill=int(tone))


def cube_template(s=TS):
    """Which face of a cube of side s each pixel of its iso picture shows (0 top, 1 the face looking down the
    river, 2 the one looking across it), -1 for none; and where on that face the pixel lies, as the two face
    coordinates (a, b) in 0..1 (top: x, y; the face looking down the river: x, height; the other: y, height)."""
    out = np.full((s, s), -1)
    uv = np.zeros((3, s, s, 2))
    n = np.zeros((3, s, s))
    g = (np.arange(s * 8) + .5) / (s * 8)
    a, b = np.meshgrid(g, g)
    for face, (px, py) in enumerate([
        ((a - b) * s / 2 + s / 2, (a + b) * s / 4),                       # top: x = a, y = b at the top
        (a * s / 2, (a * s + s) / 4 + (1 - b) * s / 2),                   # the y = s face: x = a, h = b
        (s - b * s / 2, (s + b * s) / 4 + (1 - a) * s / 2),               # the x = s face: y = b, h = a
    ]):
        c, r = np.floor(px).astype(int).clip(0, s - 1), np.floor(py).astype(int)
        m = (r >= 0) & (r < s)
        out[r[m], c[m]] = face if face == 0 else np.where(out[r[m], c[m]] == 0, 0, face)
        np.add.at(n[face], (r[m], c[m]), 1)
        np.add.at(uv[face], (r[m], c[m]), np.stack([a[m], b[m]], 1))
    coords = np.zeros((s, s, 2))
    for f in range(3):
        sel = (out == f) & (n[f] > 0)
        coords[sel] = uv[f][sel] / n[f][sel][:, None]
    return out, coords


CUBE, CUBE_UV = cube_template()
BOARD_FACE = (W_TOP, W_LEFT, W_RIGHT)
EARTH_FACE = (E_TOP, E_LEFT, E_RIGHT)


TEXTURE = True      # the standing dam's faces get wood grain and earth speckle, laid on the dam as a whole, not voxel by voxel


def texture_cube(t, mat, centre):
    """Grain over a standing voxel's flat tones (as Dam.grain gives a flying box's): the pattern comes from where each
    pixel lies in the world, so the planks' streaks run on from voxel to voxel along the dam."""
    if mat == Dam.POST:
        return
    ys, xs = np.nonzero(t >= 0)
    face = CUBE[ys, xs]
    a, b = CUBE_UV[ys, xs].T
    x0, y0, z0 = centre - VS / 2
    wx = np.where(face == 2, x0 + VS, x0 + a * VS)
    wy = np.where(face == 1, y0 + VS, y0 + b * VS)
    wz = np.where(face == 0, z0 + VS, z0 + np.where(face == 1, b, a) * VS)
    base = t[ys, xs]
    if mat == Dam.EARTH:
        n = noise(np.floor((wx + wy) / DX * 1.3), np.floor(wz / DX * 1.3 + wy / DX), 5)
        low, high = n < .2, n > .88
    else:
        across = np.where(face == 0, wy, wz)
        streak = np.floor(across / DX / 1.7)
        run = noise(streak, np.floor((wx / DX + 63 * noise(streak, np.zeros_like(streak), 4)) / 6), 3)
        low, high = run < .2, run > .86
    for c in GRAINED:
        m = base == c
        t[ys[m & low], xs[m & low]] = SHADE[c][1]
        t[ys[m & high], xs[m & high]] = SHADE[c][0]


def cube_tones(mat, variant, seam, n):
    """A voxel's pixels as tone indices: boards are light with a dark course line under them and a seam where a
    board ends, posts dark, earth in speckled clods."""
    out = np.full(CUBE.shape, -1, np.int8)
    last = CUBE.shape[0] - 1
    if mat == Dam.EARTH:
        out[CUBE == 0], out[CUBE == 1], out[CUBE == 2] = E_TOP, E_LEFT, E_RIGHT
        ys, xs = np.nonzero(CUBE == 1)
        if variant:
            k = (n * 7) % len(ys)
            out[ys[k], xs[k]] = E_TOP if variant == 1 else E_EDGE
        if variant == 3:
            ys, xs = np.nonzero(CUBE == 2)
            out[ys[n % len(ys)], xs[n % len(ys)]] = E_EDGE
        return out
    if mat == Dam.POST:
        out[CUBE == 0], out[CUBE == 1], out[CUBE == 2] = W_POST, W_RIGHT, W_EDGE
        return out
    out[CUBE == 0], out[CUBE == 1], out[CUBE == 2] = W_TOP, W_LEFT, W_RIGHT
    out[last][CUBE[last] == 1] = W_RIGHT                 # the shade under a board
    if seam:
        out[CUBE == 1] = np.where(np.arange(CUBE.shape[1])[None, :].repeat(CUBE.shape[0], 0)[CUBE == 1] == 0, W_EDGE, out[CUBE == 1])
        first = np.nonzero(CUBE[0] == 0)[0]
        out[0, first[0]] = W_END
    elif variant == 0:
        out[CUBE == 0] = W_END
    return out


class Flood:
    def __init__(self):
        self.bed0, self.dam, self.across, self.crest, self.base0, self.river, self.ford, self.wall = terrain()
        x, y = grid()
        self.x, self.y = x, y
        self.t = 0.0
        self.b = self.bed0.copy()
        self.b[self.dam] = self.crest[self.dam]
        pool = self.river & (y / CELL < DAM + .5)
        eta = np.where(pool, POOL, REST)
        self.h = np.maximum(0, eta - self.b)
        self.h[self.dam] = 0
        self.qx = np.zeros_like(self.h)
        self.qy = np.zeros_like(self.h)
        self.foam = np.zeros_like(self.h)
        self.tx, self.ty = x.copy(), y.copy()     # where each spot's water came from: the foam's pattern rides on it
        self.touched = pool.copy()                # water the flood has reached (the rest is the game's river)
        self.pool = pool
        self.eta_prev = self.h + self.b

    def burst(self):
        """The dam sinks from its middle outwards."""
        k = np.clip((self.t - T_BREAK - .13 * self.across) / .1, 0, 1)
        top = self.crest + (self.bed0 - self.crest) * k
        self.b[self.dam] = top[self.dam]

    @property
    def base(self):
        """What water stands on for drawing: as terrain() has it, and the dam while it stands above that."""
        return np.where(self.dam, np.maximum(self.base0, self.b), self.base0)

    def step(self, dt):
        self.burst()
        h, qx, qy, b = self.h, self.qx, self.qy, self.b
        wet = h > EPS
        u = np.where(wet, qx / np.maximum(h, EPS), 0)
        v = np.where(wet, qy / np.maximum(h, EPS), 0)

        def flux(hl, hr, nl, nr, tl, tr, bl, br):
            bs = np.maximum(bl, br)
            hls, hrs = np.maximum(0, hl + bl - bs), np.maximum(0, hr + br - bs)
            a = np.maximum(np.abs(nl) + np.sqrt(GRAV * hls), np.abs(nr) + np.sqrt(GRAV * hrs))
            fh = .5 * (hls * nl + hrs * nr) - .5 * a * (hrs - hls)
            fn = .5 * (hls * nl * nl + hrs * nr * nr + .5 * GRAV * (hls ** 2 + hrs ** 2)) - .5 * a * (hrs * nr - hls * nl)
            ft = .5 * (hls * nl * tl + hrs * nr * tr) - .5 * a * (hrs * tr - hls * tl)
            return fh, fn + .5 * GRAV * (hl ** 2 - hls ** 2), fn + .5 * GRAV * (hr ** 2 - hrs ** 2), ft

        k = dt / DX
        fh, fl, fr, ft = flux(h[:, :-1], h[:, 1:], u[:, :-1], u[:, 1:], v[:, :-1], v[:, 1:], b[:, :-1], b[:, 1:])
        gh, gl, gr, gt = flux(h[:-1], h[1:], v[:-1], v[1:], u[:-1], u[1:], b[:-1], b[1:])
        nh, nqx, nqy = h.copy(), qx.copy(), qy.copy()
        nh[:, :-1] -= k * fh
        nh[:, 1:] += k * fh
        nqx[:, :-1] -= k * fl
        nqx[:, 1:] += k * fr
        nqy[:, :-1] -= k * ft
        nqy[:, 1:] += k * ft
        nh[:-1] -= k * gh
        nh[1:] += k * gh
        nqy[:-1] -= k * gl
        nqy[1:] += k * gr
        nqx[:-1] -= k * gt
        nqx[1:] += k * gt
        wall = .5 * GRAV * h * h   # the domain's top and side edges are walls: they push back as hard as the water presses
        nqy[0] += k * wall[0]
        nqx[:, 0] += k * wall[:, 0]
        nqx[:, -1] -= k * wall[:, -1]
        nh = np.maximum(nh, 0)
        nh[-1], nqx[-1], nqy[-1] = nh[-2], nqx[-2], nqy[-2]   # the river runs on downstream
        speed = np.hypot(nqx, nqy) / np.maximum(nh, EPS)
        drag = 1 / (1 + dt * (.02 + .5 * (nh < 3)) * speed / np.maximum(nh, .5))   # thin sheets over ground drag
        dry = nh < EPS
        self.h = nh
        self.qx = np.where(dry, 0, nqx * drag)
        self.qy = np.where(dry, 0, nqy * drag)
        self.t += dt

    def advance(self, seconds):
        end = self.t + seconds
        while self.t < end - 1e-9:
            c = np.sqrt(GRAV * self.h) + np.hypot(self.qx, self.qy) / np.maximum(self.h, EPS)
            self.step(min(end - self.t, .22 * DX / max(c.max(), 1)))

    def velocity(self):
        wet = self.h > EPS
        return np.where(wet, self.qx / np.maximum(self.h, EPS), 0), np.where(wet, self.qy / np.maximum(self.h, EPS), 0)

    def carry(self, dt):
        """Once a frame: the foam and its pattern ride the flow; foam is made where the water rises fast, falls
        over the breach or runs fast and shallow, and dies away."""
        u, v = self.velocity()
        jj, ii = np.mgrid[0:NY, 0:NX].astype(float)
        src = [jj - v * dt / DX, ii - u * dt / DX]
        self.foam = map_coordinates(self.foam, src, order=1, mode="nearest")
        self.tx = map_coordinates(self.tx, src, order=1, mode="nearest")
        self.ty = map_coordinates(self.ty, src, order=1, mode="nearest")
        eta = self.h + self.b
        rise = np.clip((eta - self.eta_prev) / dt / 90, 0, 1)
        speed = np.hypot(u, v)
        fast = np.clip((speed - 110) / 160, 0, 1) * (self.h < 9) * (self.h > .3)
        fall = np.clip(speed / np.sqrt(GRAV * np.maximum(self.h, .3)) - 1.1, 0, 1) * (self.h > .3)
        self.foam = np.clip(self.foam * math.exp(-dt / .3) + 1.1 * rise + dt * (4 * fast + 5 * fall), 0, 1.5)
        self.foam[self.h < .3] = 0
        self.touched |= (self.h > .3) & ((eta > REST + .6) | (self.bed0 >= REST) | (self.foam > .35))
        self.touched &= (self.h > .3) & ((eta > REST + .35) | (self.bed0 >= REST))
        self.eta_prev = eta


def noise(a, b, seed=0):
    n = (a.astype(np.int64) * 374761393 + b.astype(np.int64) * 668265263 + seed * 1274126177) & 0xffffffff
    n = ((n ^ (n >> 13)) * 1274126177) & 0xffffffff
    return ((n ^ (n >> 16)) & 0xffff) / 65535.0


def tones(f):
    """Each grid cell's tone on its top and on its face."""
    eta = f.h + f.b
    gy, gx = np.gradient(eta, DX)
    lit = -(gx * -.55 + gy * .35)        # the light comes from the screen's upper left
    grain = noise(np.floor(f.tx / 2), np.floor(f.ty / 5))      # streaks that run along the river
    top = np.full(f.h.shape, BODY)
    top[lit > .22] = LIGHT
    top[lit < -.3] = DEEP
    top[(grain > .972) & (f.h > 6)] = LIGHT                     # glints on still deep water
    froth = f.foam * (.55 + .5 * grain + .8 * lumps(f))
    top[froth > .5] = FOAM
    top[froth > 1.2] = WHITE
    top[(froth > .5) & (lumps(f) < .34) & (grain < .6)] = LIGHT    # water showing through the froth
    land = f.bed0 >= REST
    shallow = land & (top <= LIGHT)
    top[shallow & (f.h < 4)] = THIN
    top[shallow & (f.h < 1.6)] = FILM
    edge = land & (f.h < 1.0) & (np.hypot(*f.velocity()) > 25)  # the lip of the sheet running over ground
    top[edge] = FOAM
    face = np.full(f.h.shape, DEEP)
    streak = noise(np.floor((f.x - f.y) / 2), np.floor(f.ty / 9), 3)
    face[f.foam > .2] = BODY
    face[(f.foam > .3) & (streak > .55)] = LIGHT
    face[(f.foam > .6) & (streak > .82)] = FOAM
    return top, face


# the dots of the simulated cells: global dot (gi, gj) has its diamond's left dot at (gi - gj - 1, floor((gi + gj) / 2))
GI = int(U0 * CELL / DX) + np.arange(NX)[None, :].repeat(NY, 0)
GJ = int(V0 * CELL / DX) + np.arange(NY)[:, None].repeat(NX, 1)
SUM = GI + GJ
PX0, PY0 = int((GI - GJ).min()) - 2, int(SUM.min() // 2) - 40 * RES
PW, PH = int((GI - GJ).max()) - PX0 + 3, int(SUM.max() // 2) - PY0 + 12 * RES
BAND = np.floor(GI * DX / CELL).astype(int) + np.floor(GJ * DX / CELL).astype(int)
DIAGS = [np.nonzero(SUM == s) for s in range(int(SUM.min()), int(SUM.max()) + 1)]


def lumps(f):
    """A lumpy 0..1 field that rides the flow and churns in time."""
    at = [f.ty / 2 + 90 * f.t, f.tx / 2 + 50 * f.t]    # the pattern's own grid is the first bake's 2-unit dots
    if RES == 1:
        return LUMPS[at[0].astype(int) % 256, at[1].astype(int) % 256]
    return map_coordinates(LUMPS, at, order=1, mode="grid-wrap")


def surface(f):
    """The surface as drawn: the frothing front rolls up taller than the height field holds it, in lumps."""
    return f.h + f.b + CREST * np.clip(f.foam, 0, 1) * (.35 + 1.1 * lumps(f)) * (f.h > 1)


def paint(f, shot, dam):
    """One frame of a shot as (tone index per dot or -1, band per dot): water columns far to near from its surface
    down to what it stands on; ground and river the flood has not reached wipe what lies behind their own surface;
    the dam's voxels over it all."""
    top, face = tones(f)
    eta, base = surface(f), f.base
    tone = np.full((PH, PW), -1, np.int8)
    band = np.zeros((PH, PW), np.int16)
    cv = GJ * DX / CELL
    view = (cv >= SHOTS[shot][0]) & (cv < SHOTS[shot][1])
    shown = f.touched & view & ~(f.dam & (f.h < 1.5))
    for jj, ii in DIAGS:
        s = int(SUM[jj[0], ii[0]])
        x = (GI[jj, ii] - GJ[jj, ii] - 1) - PX0
        on = shown[jj, ii]
        off = ~on & view[jj, ii]
        if off.any():   # ground, or river the flood has not reached: nothing shows on it
            yo = np.floor(s / 2 - np.where(f.river[jj, ii], base[jj, ii], f.b[jj, ii])[off] / DX).astype(int) - PY0
            for dx in (0, 1):
                tone[yo, x[off] + dx] = -1
        if not on.any():
            continue
        x, j2, i2 = x[on], jj[on], ii[on]
        y0 = np.floor(s / 2 - eta[j2, i2] / DX).astype(int) - PY0
        y1 = np.maximum(y0, np.floor(s / 2 - base[j2, i2] / DX).astype(int) - PY0)
        t0, t1, bd = top[j2, i2], face[j2, i2], BAND[j2, i2]
        for k in range(int((y1 - y0).max()) + 1):
            m = y0 + k <= y1
            if k and not m.any():
                break
            for dx in (0, 1):
                tone[y0[m] + k, x[m] + dx] = t0[m] if k == 0 else t1[m]
                band[y0[m] + k, x[m] + dx] = bd[m]
    dam.paint(tone, band, SHOTS[shot])
    return tone, band


def depth_grid(f, cell=8.0):
    """How deep a unit standing at each spot is under water, in quarter pixels: above the ground, the ford's
    stones or the river's own surface."""
    n = int(cell / DX)
    eta = np.where(f.touched, surface(f), -99)
    d = np.clip(eta - f.base, 0, 60)
    d = d[:NY // n * n, :NX // n * n].reshape(NY // n, n, NX // n, n).mean((1, 3))
    return np.clip(d * 4, 0, 255).astype(np.uint8)


def spray():
    """The front breaking on the ford: water thrown up and on downstream where it reaches the soldiers, row by
    row, simulated as spellsim's particle water round its origin (SPRAY_AT) and drawn in the flood's foam tones.
    Returns its frames at the ford shot's rate from T_SPRAY on."""
    s = spellsim.Water(21)
    r = s.rng
    for k in range(16):
        cu, cv = r.uniform(-1.5, 1.5), r.uniform(-1.3, 1.5)
        at = (cu * CELL, cv * CELL, 8)
        s.blob(.12 * (cv + 1.3) + r.uniform(0, .05), at, 3.5, 60, (r.normal(0, 45), 130 + r.normal(0, 45), r.uniform(230, 340)), 55)
    tones, spellsim.WATER = spellsim.WATER, {"deep": tuple(PAL[LIGHT][:3]), "body": tuple(PAL[FOAM][:3]), "light": tuple(PAL[WHITE][:3]),
                                             "hi": (255, 255, 255), "w": (255, 255, 255), "edge": tuple(PAL[BODY][:3])}
    frames = []
    rate = FPS * SLOW[1][1]
    for _ in range(int(round((T_END - T_SPRAY) * rate))):
        for _ in range(spellsim.SUB):
            s.step(1 / (rate * spellsim.SUB))
        frames.append(s.render())
    spellsim.WATER = tones
    return frames


T_SPRAY = 4.7
SPRAY_AT = ((RIVER + 1.5) * CELL, (FORD + 1.2) * CELL)


def squeeze(tone, band):
    """A painted frame kept small while the rest are simulated: the box its dots fill, deflated."""
    if not (tone >= 0).any():
        return None
    x0, y0, x1, y1 = crop(tone >= 0)
    t, b = tone[y0:y1, x0:x1], band[y0:y1, x0:x1]
    return y0, x0, t.shape, zlib.compress(t.tobytes(), 1), zlib.compress(b.tobytes(), 1)


def unsqueeze(packed):
    """The frame squeeze kept, as the full (tone, band)."""
    tone = np.full((PH, PW), -1, np.int8)
    band = np.zeros((PH, PW), np.int16)
    if packed is not None:
        y0, x0, shape, t, b = packed
        tone[y0:y0 + shape[0], x0:x0 + shape[1]] = np.frombuffer(zlib.decompress(t), np.int8).reshape(shape)
        band[y0:y0 + shape[0], x0:x0 + shape[1]] = np.frombuffer(zlib.decompress(b), np.int16).reshape(shape)
    return tone, band


def probe():
    """The water alone, without the dam's pictures or the frames: where the front is down the river's middle as
    simulated time goes, for setting T_END and T_SPRAY."""
    f = Flood()
    f.t = 0.0
    f.advance(T_BREAK)
    while f.t < T_END - 1e-9:
        f.advance(.1)
        f.carry(.1)
        reached = f.touched & ~f.pool
        rows = np.nonzero(reached.any(1))[0]
        v_front = V0 + (rows.max() + 1) * DX / CELL if len(rows) else 0
        cols = np.nonzero(reached.any(0))[0]
        wide = f"{U0 + cols.min() * DX / CELL:.1f}..{U0 + (cols.max() + 1) * DX / CELL:.1f}" if len(cols) else "-"
        j = int((FORD + 1 - V0) * CELL / DX)
        print(f"t={f.t:.1f} front v={v_front:.1f} flooded u {wide} ford depth {(f.h[j] * f.ford[j]).max():.1f} "
              f"{time.time() - T0:.0f}s", flush=True)


def simulate(log=False, rot=None):
    """Frames [seconds it shows, the painted tone and band squeezed, None, depth, shot, simulated time]: the burst's
    seconds in slow motion."""
    f, dam = Flood(), Dam()
    dry = np.zeros_like(depth_grid(f))
    frames = [[T_BREAK, squeeze(*paint(f, 0, dam)), None, dry, 0, T_BREAK]]   # the dam standing: it shows until the burst
    f.t = T_BREAK
    for shot, (sim_secs, per, real_secs) in enumerate(SLOW):
        n = int(round(sim_secs * FPS * per))
        dt = sim_secs / n
        subs = math.ceil(dt * 120 - 1e-9)
        for k in range(n):
            for _ in range(subs):
                f.advance(dt / subs)
                dam.step(f, dt / subs)
            f.carry(dt)
            if rot is not None and k % 9 == 8:
                live = dam.free & ~dam.gone & (dam.bmat == Dam.BOARD)
                if live.any():
                    deg = np.degrees(np.abs(dam.ang[live]))
                    since = f.t - dam.wait[live]
                    rot.append(f"t={f.t:.2f} free planks={int(live.sum())} median turn yaw/pitch/roll deg "
                               f"{np.median(deg[:, 0]):.0f}/{np.median(deg[:, 1]):.0f}/{np.median(deg[:, 2]):.0f}, max yaw {deg[:, 0].max():.0f}, "
                               f"planks turned >45 deg: {int((deg.max(1) > 45).sum())}")
            frames.append([real_secs / n, squeeze(*paint(f, shot, dam)), None, depth_grid(f) if shot else dry, shot, f.t])
            if k % 10 == 9:
                print(f"shot {shot} frame {k + 1}/{n} t={f.t:.2f} {time.time() - T0:.0f}s", file=sys.stderr, flush=True)
            if log and k % (per * 3) == per * 3 - 1:
                eta = f.h + f.b
                mid = eta[:, NX // 2]
                front = np.nonzero((mid > REST + 1) & (f.bed0[:, NX // 2] < REST))[0]
                v_front = V0 + (front.max() + .5) * DX / CELL if len(front) else 0
                j = int(((FORD + 1.5) - V0) * CELL / DX)
                print(f"t={f.t:.2f} front v={v_front:.1f} at ford eta={mid[j]:.1f} max eta below dam={mid[int((DAM + 1 - V0) * CELL / DX):].max():.1f}"
                      f" pool={mid[int((DAM - 2 - V0) * CELL / DX)]:.1f}")
    return frames


def rgba(tone):
    out = np.zeros((*tone.shape, 4), np.uint8)
    m = tone >= 0
    out[m] = PAL[tone[m]]
    return out


def preview(frames, sprays, out):
    """Each frame as its shot's view at the game's 2x zoom would frame it (640x424 canvas pixels round CAMS) over
    flat grass: for looking at the rock, the water's shape and the timing without the game."""
    shots = []
    for secs, packed, _, _, shot, sim in frames:
        tone, _ = unsqueeze(packed)
        cu, cv = CAMS[shot]
        cx, cy = (cu - cv) * CELL, (cu + cv + 1) * CELL / 2
        w, h = int(640 / DX), int(424 / DX)
        x0, y0 = int(cx / DX) - w // 2 - PX0, int(cy / DX) - h // 2 - PY0
        im = Image.new("RGBA", (w, h), (104, 140, 72, 255))
        im.alpha_composite(Image.fromarray(rgba(tone)).crop((x0, y0, x0 + w, y0 + h)))
        im = im.resize((640, 424), Image.NEAREST)
        i = int(round((sim - T_SPRAY) * FPS * SLOW[1][1])) - 1
        if 0 <= i < len(sprays):
            sp = Image.fromarray(sprays[i])
            ox, oy = SPRAY_AT[0] - SPRAY_AT[1] - spellsim.ORIGIN[0], (SPRAY_AT[0] + SPRAY_AT[1]) / 2 - spellsim.ORIGIN[1]
            im.alpha_composite(sp, (int(ox - cx + 320), int(oy - cy + 212)))
        shots.append(im.convert("RGB"))
    out.mkdir(parents=True, exist_ok=True)
    shots[0].save(out / "suigong.gif", save_all=True, append_images=shots[1:], loop=0,
                  duration=[int(f[0] * 1000) for f in frames])
    picks = [0] + [min(range(len(frames)), key=lambda k: abs(frames[k][5] - t)) for t in (1.1, 1.3, 1.6, 1.9, 2.3, 2.7, 3.1, 3.5)]
    sheet = Image.new("RGB", (640 * 3, 424 * 3))
    for n, k in enumerate(picks):
        sheet.paste(shots[min(k, len(shots) - 1)], (n % 3 * 640, n // 3 * 424))
    sheet.save(out / "suigong.png")
    print("wrote", out / "suigong.gif", out / "suigong.png")


def pack(images, width=4096):
    """Shelf-packs RGBA arrays (equal ones share a place): the sheet and each image's (x, y, w, h)."""
    seen, rects, order = {}, [None] * len(images), sorted(range(len(images)), key=lambda i: -images[i].shape[0])
    x = y = shelf = 0
    for i in order:
        a = images[i]
        key = (a.shape, a.tobytes())
        if key in seen:
            rects[i] = seen[key]
            continue
        hh, ww = a.shape[:2]
        if x + ww > width:
            x, y, shelf = 0, y + shelf + 1, 0
        rects[i] = seen[key] = (x, y, ww, hh)
        x, shelf = x + ww + 1, max(shelf, hh)
    sheet = np.zeros((y + shelf, width, 4), np.uint8)
    for i, a in enumerate(images):
        rx, ry, ww, hh = rects[i]
        sheet[ry:ry + hh, rx:rx + ww] = a
    return sheet, rects


def crop(mask):
    ys, xs = np.nonzero(mask)
    return xs.min(), ys.min(), xs.max() + 1, ys.max() + 1


def bake(frames, sprays):
    """assets/graphics/effects/flood/suigong.{json,idx} and suigong_depth.png. The ford's shot goes GAP cells down
    the map's river. Each frame's bands, and each frame of spray (already in canvas pixels), are kept as palette
    indices (255: clear) deflated one frame at a time, so the game makes pictures of only the frame showing."""
    OUT.mkdir(parents=True, exist_ok=True)
    blobs, bands = [], []
    for k, (_, packed, _, _, shot, _) in enumerate(frames):
        tone, band = unsqueeze(packed)
        g = GAP * shot
        raw, entries = bytearray(), []
        for d in np.unique(band[tone >= 0]):
            m = (tone >= 0) & (band == d)
            x0, y0, x1, y1 = crop(m)
            idx = np.where(m, tone.astype(np.int16), 255)[y0:y1, x0:x1]
            raw += idx.astype(np.uint8).tobytes()
            entries.append([int(d) + g, int(x0 + PX0) * int(DX) - g * int(CELL), int(y0 + PY0) * int(DX) + g * int(CELL) // 2, int(x1 - x0), int(y1 - y0)])
        blobs.append(zlib.compress(bytes(raw), 9))
        bands.append(entries)
    packed, frame_at = bytearray(), []
    for b in blobs:
        frame_at.append([len(packed), len(b)])
        packed += b
    pal = [tuple(c) for c in PAL.tolist()]
    for im in sprays:
        for c in map(tuple, np.unique(im.reshape(-1, 4), axis=0).tolist()):
            if c[3] and c not in pal:
                pal.append(c)
    assert len(pal) < 255
    lut = {c: i for i, c in enumerate(pal)}
    spray_at = []
    for im in sprays:
        if im[..., 3].any():
            x0, y0, x1, y1 = crop(im[..., 3] > 0)
            im = im[y0:y1, x0:x1]
        else:
            x0 = y0 = 0
            im = np.zeros((1, 1, 4), np.uint8)
        idx = np.full(im.shape[:2], 255, np.uint8)
        for c, i in lut.items():
            idx[(im == np.array(c, np.uint8)).all(2)] = i
        blob = zlib.compress(idx.tobytes(), 9)
        frame_at.append([len(packed), len(blob)])
        packed += blob
        spray_at.append((int(x0), int(y0), im.shape[1], im.shape[0]))
    (OUT / "suigong.idx").write_bytes(bytes(packed))
    ms, shown, last = [], 0.0, 0
    for f in frames:
        shown += f[0]
        ms.append(int(round(shown * 1000)) - last)
        last += ms[-1]
    cut_ms = sum(m for m, f in zip(ms, frames) if f[4] == 0)
    spray_ms = sum(m for m, f in zip(ms, frames) if f[5] < T_SPRAY - 1e-6)
    ford_ms = SLOW[1][2] * 1000 / int(round(SLOW[1][0] * FPS * SLOW[1][1]))
    sx, sy = SPRAY_AT[0], SPRAY_AT[1] + GAP * CELL
    ox, oy = sx - sy - spellsim.ORIGIN[0], (sx + sy) / 2 - spellsim.ORIGIN[1]
    dg = np.stack([f[3] for f in frames])
    cols = 16
    rows = -(-len(dg) // cols)
    dsheet = np.zeros((rows * dg.shape[1], cols * dg.shape[2]), np.uint8)
    for k, g in enumerate(dg):
        dsheet[k // cols * g.shape[0]:, k % cols * g.shape[1]:][:g.shape[0], :g.shape[1]] = g
    Image.fromarray(dsheet).save(OUT / "suigong_depth.png", optimize=True)
    doc = {
        "note": "tools/floodsim.py --bake",
        "layout": {"river": RIVER, "width": WIDTH, "dam": DAM, "ford": FORD + GAP, "cliff": CLIFF},
        "break": T_BREAK,
        "cut": cut_ms / 1000,     # seconds in: the shot changes from the dam to the ford
        "scale": int(DX),         # canvas pixels a dot is wide: 2 for the first bake, 1 for twice the resolution
        "ms": ms,         # how long each frame shows: the burst plays in slow motion (SLOW)
        "bands": bands,   # per frame: [u + v of the band's cells, x, y, w, h], its dots in suigong.idx in this order, drawn scale times larger
        "pack": {"frames": frame_at, "pal": pal},   # [offset, length] of each frame's deflated indices, then each spray frame's; the colours they index
        "depth": {"cell": 8, "per": 4, "x": U0 * CELL, "y": (V0 + GAP) * CELL, "w": int(dg.shape[2]), "h": int(dg.shape[1]), "cols": cols},
        "spray": {"at": spray_ms / 1000, "ms": int(round(ford_ms)),
                  "frames": [[int(ox + x), int(oy + y), w, h] for x, y, w, h in spray_at]},
    }
    (OUT / "suigong.json").write_text(json.dumps(doc, separators=(",", ":")))
    dots = sum(w * h for fr in bands for *_, w, h in fr)
    print(f"baked {len(frames)} frames, {sum(map(len, bands))} bands, {dots} dots, idx {len(packed) // 1024} KB, "
          f"{len(sprays)} spray frames, {len(pal)} colours")


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--bake", action="store_true", help="write the game's files instead of the preview")
    ap.add_argument("--out", default=str(ROOT / "reports/floodsim"))
    ap.add_argument("--no-spray", action="store_true")
    ap.add_argument("--end", type=float, help="preview only this many seconds")
    ap.add_argument("--probe", action="store_true", help="print where the front is as time goes, nothing else")
    a = ap.parse_args()
    if a.probe:
        probe()
        return
    if a.end:
        global T_END, SLOW
        T_END = a.end
        SLOW = (SLOW[0], (T_END - T_CUT, SLOW[1][1], FORD_SLOW * (T_END - T_CUT)))
    rot = []
    frames = simulate(log=not a.bake, rot=rot)
    print(*rot, sep="\n")
    sprays = [] if a.no_spray else spray()
    if a.bake:
        bake(frames, sprays)
    else:
        preview(frames, sprays, Path(a.out))


if __name__ == "__main__":
    main()
