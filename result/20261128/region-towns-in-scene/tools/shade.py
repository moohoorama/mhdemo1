"""gen_pixel_castle.render에 그림자를 더한 사본: 픽셀이 맞은 점에서 빛 쪽으로 광선을 쏴 다른 몸체(처마, 윗지붕,
옆채, 담장)에 막히면 그늘 색으로 칠한다. 원래 그늘 면(+u 쪽)은 이미 어두워 다시 어둡게 하지 않는다.

빛: 땅 그림자는 게임처럼 높이 1px당 화면 (0.6, 0.3)px, 곧 +u로 0.0375칸 흐른다. 밝은 +v 면이 해를 받도록
v 성분을 조금 더해, 빛으로 가는 방향은 높이 1px당 (u -0.0375, v +0.03).
"""
import numpy as np

import gen_pixel_castle as pc

LIGHT = (-0.0375, 0.03, 1.0)      # 빛 쪽으로: u 칸, v 칸, z px (높이 1px당)
SHADE = np.array([0.6, 0.62, 0.72])   # 그늘 곱 (약간 푸른 그늘)
SHADE_U = np.array([0.74, 0.75, 0.82])  # 이미 그늘인 면에 한 번 더
EAVE = 0.016                           # 처마 밑 그늘 광선: 높이 1px당 바깥쪽으로 가는 칸


def lit_face(f):
    return f == "+z" or f.endswith("v")


def render(bodies, bounds, own=None):
    x0, y0, x1, y1 = bounds
    X, Y = np.meshgrid(np.arange(x0, x1) + .5, np.arange(y0, y1) + .5)
    au, av = X / 32 + Y / 16, Y / 16 - X / 32
    best = np.full(X.shape, -np.inf)
    rgb = np.zeros(X.shape + (3,), np.uint8)
    depth = np.full(X.shape, -np.inf)
    face_win = np.full(X.shape, "", object)
    body_win = np.full(X.shape, -1)
    for bi, b in enumerate(bodies):
        lo, hi = np.full(X.shape, -np.inf), np.full(X.shape, np.inf)
        face = np.full(X.shape, "", object)
        ok = np.ones(X.shape, bool)
        for (nu, nv, nz), d, f in b.planes:
            c = nu / 16 + nv / 16 + nz
            rhs = d - nu * au - nv * av
            if c > 0:
                bound = rhs / c
                face = np.where(bound < hi, f, face)
                hi = np.minimum(hi, bound)
            elif c < 0:
                lo = np.maximum(lo, rhs / c)
            else:
                ok &= rhs >= 0
        hit = ok & (lo <= hi + 1e-9) & (hi > best + 1e-9)
        if not hit.any():
            continue
        z = hi
        u, v = au + z / 16, av + z / 16
        for f in set(face[hit]):
            m = hit & (face == f)
            s = np.floor(16 * u[m] + 1e-6).astype(int) if f.endswith("v") else np.floor(-16 * v[m] + 1e-6).astype(int)
            h = np.floor(z[m] + 1e-6).astype(int)
            keys = b.paint(f, u[m], v[m], z[m], s, h)
            rgb[m] = np.array([pc.PAL[k] for k in np.atleast_1d(keys)], np.uint8).reshape(-1, 3)
        best = np.where(hit, z, best)
        depth = np.where(hit, u + v, depth)
        face_win = np.where(hit, face, face_win)
        body_win = np.where(hit, bi, body_win)
    solid = np.isfinite(best)
    # 해 그림자: 밝은 면의 점에서 빛 쪽으로. 처마 밑 그늘: 벽면의 점에서 바깥쪽 위로 비스듬히(높이 1px당
    # EAVE 칸) 쏴서 처마·윗층에 막히면 그늘 (밝은 면은 SHADE, 원래 그늘 면은 SHADE_U)
    zz = np.where(solid, best, 0)
    Pall = np.stack([au + zz / 16, av + zz / 16, zz], -1)
    shadow = np.zeros(X.shape, bool)
    shadow_u = np.zeros(X.shape, bool)
    fw = face_win
    lit_m = solid & np.vectorize(lit_face)(fw)
    if lit_m.any():
        shadow[lit_m] = occluded(Pall[lit_m], np.array(LIGHT), bodies)
    for f, n in (("+v", (0, 1)), ("-v", (0, -1)), ("+u", (1, 0)), ("-u", (-1, 0))):
        m = solid & (fw == f)
        if not m.any():
            continue
        D = np.array([n[0] * EAVE, n[1] * EAVE, 1.0])
        occ = occluded(Pall[m], D, bodies)
        if f.endswith("v"):
            shadow[m] |= occ
        else:
            shadow_u[m] = occ
    ink = np.zeros(X.shape, bool)
    owned = solid & own(Pall[..., 0], Pall[..., 1]) if own is not None else solid
    ink_own = np.zeros(X.shape, bool)
    ink_depth = np.full(X.shape, -np.inf)
    H, W = X.shape
    for dy, dx in ((0, 1), (0, -1), (1, 0), (-1, 0)):
        nd = np.full(X.shape, -np.inf)
        no = np.zeros(X.shape, bool)
        ys, yd = slice(max(dy, 0), H + min(dy, 0)), slice(max(-dy, 0), H + min(-dy, 0))
        xs, xd = slice(max(dx, 0), W + min(dx, 0)), slice(max(-dx, 0), W + min(-dx, 0))
        nd[yd, xd] = depth[ys, xs]
        no[yd, xd] = owned[ys, xs]
        nearer = np.isfinite(nd) & (~solid | (nd > depth + .25))
        take = nearer & (nd > ink_depth)
        ink |= nearer
        ink_own = np.where(take, no, ink_own)
        ink_depth = np.where(take, nd, ink_depth)
    out = rgb.astype(float)
    out[shadow] *= SHADE
    out[shadow_u & ~shadow] *= SHADE_U
    a = np.zeros(X.shape + (4,), np.uint8)
    a[..., :3] = np.clip(out, 0, 255).astype(np.uint8)
    a[solid, 3] = 255
    a[ink] = pc.INK + (255,)
    if own is None:
        return a, (x0, y0)
    return a, (x0, y0), np.where(ink, ink_own, owned)


def occluded(P, D, bodies):
    """점들 P에서 D 방향(조금 띄워서) 광선이 어느 몸체든 지나면 True"""
    P = P + D * 0.6
    hit = np.zeros(len(P), bool)
    for b in bodies:
        t0, t1 = np.zeros(len(P)), np.full(len(P), 400.0)
        for (nu, nv, nz), d, f in b.planes:
            n = np.array([nu, nv, nz], float)
            nd = n @ D
            rhs = d - P @ n
            if abs(nd) < 1e-12:
                t1 = np.where(rhs < 0, -1, t1)
            elif nd > 0:
                t1 = np.minimum(t1, rhs / nd)
            else:
                t0 = np.maximum(t0, rhs / nd)
        hit |= t0 < t1
    return hit
