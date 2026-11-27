"""B안: 같은 다섯 채를 3D 모형(상자·막대·타원체·곡면 지붕)으로 만들고 장면 소품과 같은 셀 렌더러로 도트화.

좌표는 A안과 같이 소타일 (u, v)와 픽셀 높이 z로 적고, 모형 단위로 바꾼다:
x = u - cu, y = cv - v (+x 오른쪽 아래 = u, -y 왼쪽 아래 = v), 높이 1단위 = PXU 픽셀.

  python3 town_3d.py OUTDIR
"""
import json
import math
import os
import sys

import numpy as np

sys.path.insert(0, "/Users/younwookim/orca/mhdemo1/tools/unit3d")
import base_model as m
import props as P
import render as R
import scene_props3d as SP
import structures as S

PXU = 19.6
EARTH, EARTH_DARK = "#cfa46c", "#a07a4c"
STONEW, BRICK, IRON = "#c8bc9e", "#a8603e", "#3a3d44"
FIRE, EMBER = "#ffd24a", "#e2662a"
JAR, CLOTH, REDC = "#7e3e26", "#ece4d0", "#b0302a"
THATCH, THATCH_DARK, THATCH_EDGE = "#c89a52", "#a8803e", "#7a5a2c"
TILE, TILE_DARK = S.TILE, S.TILE_DARK
WOOD, PLASTER, LACQ = S.WOOD, S.PLASTER, P.LACQUER
GOLD, INK = "#d5a123", "#211f22"
R.add_ramps({EARTH: "TjjS", EARTH_DARK: "UTTj", STONEW: "TSS5", BRICK: "KLLM", FIRE: "lmmn", EMBER: "kLlm",
             JAR: "KKLL", CLOTH: "4566", REDC: "bccd", THATCH_EDGE: "fEEF", THATCH: "EFVV", THATCH_DARK: "EEFV", "#211f22": "0000"})
R.COLOR.update({"E": (128, 92, 46), "F": (190, 146, 74), "V": (222, 184, 104)})

C = (0.0, 0.0)


def P3(u, v, z):
    return np.array([u - C[0], C[1] - v, z / PXU], float)


def box(u0, u1, v0, v1, z0, z1, col):
    m.box(P3((u0 + u1) / 2, (v0 + v1) / 2, (z0 + z1) / 2), [u1 - u0, v1 - v0, (z1 - z0) / PXU], col)


def rod(a, b, r, col):
    m.rod(P3(*a), P3(*b), r, col)


def quad(pts, col):
    m.poly([P3(*p) for p in pts], col)


def surface(f, na, nb, col):
    """f(a, b) -> (u, v, z), a, b in [0,1]; col(i, j) -> colour"""
    for i in range(na):
        for j in range(nb):
            a0, a1, b0, b1 = i / na, (i + 1) / na, j / nb, (j + 1) / nb
            m.poly([P3(*f(a0, b0)), P3(*f(a1, b0)), P3(*f(a1, b1)), P3(*f(a0, b1))], col(i, j))


def lerp(p, q, t):
    return tuple(p[k] + (q[k] - p[k]) * t for k in range(3))


def walls(u0, u1, v0, v1, zw, col, frame=WOOD, gable=None):
    """벽 몸통, 굽, 모서리 기둥, 박공 삼각(gable = 용마루 높이, 용마루는 u 방향)"""
    box(u0, u1, v0, v1, 0, zw, col)
    box(u0 - .03, u1 + .03, v0 - .03, v1 + .03, 0, 3, STONEW)
    for u in (u0, u1):
        for v in (v0, v1):
            box(u - .04, u + .04, v - .04, v + .04, 0, zw + 1, frame)
    box(u0, u1, v1 - .01, v1 + .03, zw - 3, zw, frame)
    if gable:
        cm = (v0 + v1) / 2
        for u in (u0, u1):
            quad([(u, v0, zw), (u, cm, gable), (u, v1, zw)], col)


def door(u0, u1, v, h, col=P.PLANK, face="v"):
    if face == "v":
        box(u0, u1, v - .01, v + .02, 3, h, col)
        box((u0 + u1) / 2 - .012, (u0 + u1) / 2 + .012, v - .01, v + .03, 3, h, INK)
    else:
        box(v - .01, v + .02, u0, u1, 3, h, col)


def window(u0, u1, v, z0, z1, face="v", col=INK, bar=P.PLANK):
    if face == "v":
        box(u0, u1, v - .01, v + .02, z0, z1, col)
        for k in np.linspace(u0, u1, 4)[1:-1]:
            box(k - .012, k + .012, v, v + .03, z0, z1, bar)
        box(u0, u1, v, v + .03, (z0 + z1) / 2 - .5, (z0 + z1) / 2 + .5, bar)
    else:
        box(v - .01, v + .02, u0, u1, z0, z1, col)
        for k in np.linspace(u0, u1, 4)[1:-1]:
            box(v, v + .03, k - .012, k + .012, z0, z1, bar)


def gable_roof(u0, u1, v0, v1, zw, rise, ov=.16, ova=.14, kind="thatch"):
    """u 방향 용마루 맞배. 초가는 볼록하게 부풀리고 처마에 두툼한 짚 끝(막대), 기와는 골마다 짙은 줄과 처마 곡선"""
    cm = (v0 + v1) / 2
    zr = zw + rise * (v1 - v0) / 2
    U0, U1 = u0 - ova, u1 + ova
    ze = zw - rise * ov

    def slope(sign):
        ve = v1 + ov if sign > 0 else v0 - ov

        def f(a, b):
            u = U0 + (U1 - U0) * a
            v = ve + (cm - ve) * b
            z = ze + (zr - ze) * b
            if kind == "thatch":
                z += 4 * math.sin(math.pi * b) * (1 - b * .3)
            else:
                z -= 2.5 * math.sin(math.pi * b)          # 기와 지붕은 오목하게
                z += 3 * abs(2 * a - 1) ** 4 * (1 - b)    # 처마 양끝이 살짝 들림
            return (u, v, z)
        return f

    if kind == "thatch":
        col = lambda i, j: THATCH_DARK if j == 0 or (i * 7 + j * 3) % 11 == 0 else THATCH
        n = 14
    else:
        col = lambda i, j: TILE_DARK if i % 2 else TILE
        n = int((U1 - U0) * 12)
    for sgn in (1, -1):
        surface(slope(sgn), n, 5, col)
    # 처마 끝 두께
    edge_col = THATCH_EDGE if kind == "thatch" else TILE_DARK
    r = .07 if kind == "thatch" else .04
    for sgn in (1, -1):
        f = slope(sgn)
        pts = [f(a, 0) for a in np.linspace(0, 1, 9)]
        for p, q in zip(pts, pts[1:]):
            rod((p[0], p[1], p[2] - 1), (q[0], q[1], q[2] - 1), r, edge_col)
        for a in (0, 1):  # 박공 쪽 끝선
            g = [f(a, b) for b in np.linspace(0, 1, 6)]
            for p, q in zip(g, g[1:]):
                rod((p[0], p[1], p[2] - 1), (q[0], q[1], q[2] - 1), r * .8, edge_col)
    if kind == "thatch":
        rod((U0 - .02, cm, zr + 4), (U1 + .02, cm, zr + 4), .09, THATCH_DARK)
        for a in np.linspace(U0 + .2, U1 - .2, 4):           # 용마루 새끼 매듭
            m.ell(P3(a, cm, zr + 4), [.05, .13, .08], WOOD)
    else:
        box(U0, U1, cm - .07, cm + .07, zr - 3, zr + 3, TILE_DARK)
        for a in (U0, U1 - .12):
            box(a, a + .12, cm - .08, cm + .08, zr - 1, zr + 7, TILE_DARK)
    return zr


def hip_roof(u0, u1, v0, v1, ze, zr, r0, r1, lift=6, sag=4, kind="tile"):
    """우진각 기와: 처마 선이 양 끝으로 들리고 면이 오목한 곡면, 골마다 짙은 줄"""
    cm = (v0 + v1) / 2
    corners = {"fl": (u0, v1), "fr": (u1, v1), "br": (u1, v0), "bl": (u0, v0)}

    def eave(p, q, a):
        u = p[0] + (q[0] - p[0]) * a
        v = p[1] + (q[1] - p[1]) * a
        return (u, v, ze + lift * abs(2 * a - 1) ** 3)

    def face(p, q, rp, rq, n):
        def f(a, b):
            e = eave(p, q, a)
            rr = lerp((rp[0], rp[1], zr), (rq[0], rq[1], zr), a)
            pt = lerp(e, rr, b)
            return (pt[0], pt[1], pt[2] - sag * math.sin(math.pi * b))
        if kind == 'tile':
            surface(f, n, 6, lambda i, j: TILE_DARK if i % 2 else TILE)
        else:
            surface(f, n, 6, lambda i, j: THATCH_DARK if j == 0 or (i * 7 + j * 3) % 11 == 0 else THATCH)
        pts = [eave(p, q, a) for a in np.linspace(0, 1, 10)]
        for a, b in zip(pts, pts[1:]):
            if kind == 'tile':
                rod((a[0], a[1], a[2] - 1), (b[0], b[1], b[2] - 1), .045, TILE_DARK)
            else:
                rod((a[0], a[1], a[2] - 1.5), (b[0], b[1], b[2] - 1.5), .08, THATCH_EDGE)

    R0, R1 = (r0, cm), (r1, cm)
    face(corners["fl"], corners["fr"], R0, R1, int((u1 - u0) * 12))   # 앞
    face(corners["fr"], corners["br"], R1, R1, int((v1 - v0) * 12))   # 오른쪽
    face(corners["br"], corners["bl"], R1, R0, int((u1 - u0) * 12))
    face(corners["bl"], corners["fl"], R0, R0, int((v1 - v0) * 12))
    if kind != 'tile':
        rod((r0 - .05, cm, zr + 1), (r1 + .05, cm, zr + 1), .1, THATCH_DARK)
        for a in np.linspace(r0, r1, 3):
            m.ell(P3(a, cm, zr + 2), [.05, .15, .09], WOOD)
        return
    box(r0 - .05, r1 + .05, cm - .07, cm + .07, zr - 3, zr + 3, TILE_DARK)
    for a in (r0 - .1, r1 - .02):
        box(a, a + .12, cm - .08, cm + .08, zr - 1, zr + 10, TILE_DARK)
    for (u, v) in corners.values():   # 추녀마루
        rod((u, v, ze + lift), (r0 if u == u0 else r1, cm, zr), .05, TILE_DARK)


def cyl(u, v, z0, z1, r, col, n=10, top=False):
    """뚜껑 없는 원통 (m.rod는 끝에 공을 붙인다)"""
    for i in range(n):
        a0, a1 = i * math.tau / n, (i + 1) * math.tau / n
        quad([(u + r * math.cos(a0), v + r * math.sin(a0), z0), (u + r * math.cos(a1), v + r * math.sin(a1), z0),
              (u + r * math.cos(a1), v + r * math.sin(a1), z1), (u + r * math.cos(a0), v + r * math.sin(a0), z1)], col)
    if top:
        quad([(u + r * math.cos(i * math.tau / n), v + r * math.sin(i * math.tau / n), z1) for i in range(n)], "#211f22")


def jar(u, v, r=.1, h=10):
    m.ell(P3(u, v, h * .45), [r, r, h * .5 / PXU], JAR)
    box(u - r * .55, u + r * .55, v - r * .55, v + r * .55, h * .8, h, JAR)


# ---------------------------------------------------------------- 건물들 (A안과 같은 배치)

def cottage():
    walls(.3, 1.6, .5, 1.45, 30, EARTH)
    door(.58, .86, 1.45, 25)
    window(1.1, 1.42, 1.45, 13, 21)
    window(.8, 1.15, 1.6, 13, 21, face="u")
    cyl(.43, .62, 0, 56, .08, BRICK, top=True)
    hip_roof(.14, 1.76, .34, 1.61, 26, 46, .75, 1.15, lift=0, sag=-4, kind="thatch")
    for k, z in enumerate((1, 4, 7)):                       # 장작더미
        for v in np.linspace(.6, .9, 3):
            rod((1.67, v, z + 1.5), (1.92, v, z + 1.5), .05, WOOD)
    jar(1.78, 1.25, .09, 9)
    return (2, 2)


def smithy():
    a0, a1, c0, c1, zw, s = .25, 2.35, .35, 1.45, 32, 20
    box(a0, a1, c0, c0 + .1, 0, zw, EARTH)
    box(a0, a0 + .1, c0, c1, 0, zw, EARTH)
    cm = (c0 + c1) / 2
    zr = zw + s * (c1 - c0) / 2
    for u in (a0 + .05, 1.3, a1 - .05):
        rod((u, c1 - .05, 0), (u, c1 - .05, zw), .045, WOOD)
    box(a0, a1, c1 - .1, c1, zw - 3, zw, WOOD)
    box(.4, 1.1, .5, 1.0, 0, 16, BRICK)                    # 화덕
    box(.6, .9, .99, 1.01, 3, 10, EMBER)
    box(.65, .85, 1.0, 1.02, 3, 6, FIRE)
    box(.6, .9, .62, .88, 16, 16.5, FIRE)
    box(.55, .95, .55, .85, 16, 40, BRICK)
    hip_roof(a0 - .16, a1 + .16, c0 - .16, c1 + .16, zw - 4, zw + 16, .95, 1.65, lift=0, sag=-4, kind="thatch")
    m.ell(P3(2.6, .6, 30), [.18, .18, 32 / PXU], BRICK) if False else None
    for k in range(8):                                     # 둥근 굴뚝: 벽돌 고리 쌓기
        cyl(2.6, .6, k * 8, k * 8 + 8, .17 - k * .006, BRICK if k % 2 == 0 else "#9a5636")
    cyl(2.6, .6, 64, 67, .19, "#7e3e26", top=True)
    rod((1.53, 1.79, 0), (1.53, 1.79, 6), .08, WOOD)       # 모루
    box(1.38, 1.7, 1.74, 1.84, 6, 9, IRON)
    m.ell(P3(1.33, 1.79, 7.5), [.07, .04, .06], IRON)
    box(1.95, 2.45, 1.62, 1.86, 0, 6, WOOD)                # 물통
    box(2.0, 2.4, 1.66, 1.82, 5.5, 6, "#3974a0")
    rod((1.85, 1.6, 0), (1.95, 1.4, 18), .025, WOOD)       # 기대 놓은 망치
    return (3, 2)


def yamen():
    box(.15, 3.85, .25, 2.4, 0, 9, STONEW)
    for k in range(3):
        box(1.55, 2.45, 2.4, 2.4 + .17 * (3 - k), 0, 3 * (3 - k), STONEW)
    zb, zw = 9, 47
    box(.6, 3.4, .75, 1.85, zb, zw, PLASTER)
    box(.6, 3.4, 1.84, 1.87, zw - 9, zw - 7, WOOD)
    for (a, b, h) in ((1.0, 1.45, 22), (1.65, 2.35, 24), (2.55, 3.0, 22)):
        box(a, b, 1.85, 1.88, zb + 1, zb + h, LACQ)
        for k in np.linspace(a, b, 6)[1:-1]:
            box(k - .01, k + .01, 1.86, 1.9, zb + 1, zb + h, P.LACQUER_DARK)
        for z in np.linspace(zb + 4, zb + h - 2, 4):
            box(a, b, 1.86, 1.9, z - .4, z + .4, P.LACQUER_DARK)
    window(.8, 1.6, 3.4, zb + 10, zb + 18, face="u")
    for u in (.6, 1.25, 1.65, 2.35, 2.75, 3.4):
        rod((u, 2.2, zb), (u, 2.2, zw), .055, LACQ)
        box(u - .07, u + .07, 2.13, 2.27, zb, zb + 1.5, STONEW)
    box(.55, 3.45, 2.15, 2.25, zw - 4, zw, P.LACQUER_DARK)
    box(1.78, 2.22, 2.25, 2.28, zw - 14, zw - 5, GOLD)      # 편액 (글자 없이)
    box(1.82, 2.18, 2.27, 2.3, zw - 13, zw - 6, INK)
    hip_roof(.35, 3.65, .5, 2.4, zw - 1, zw + 20, 1.3, 2.7, lift=4, sag=3)
    for u in (.1, 3.9):
        rod((u, 2.7, 0), (u, 2.7, 66), .03, WOOD)
        m.ell(P3(u, 2.7, 67), [.04, .04, .06], GOLD)
        surface(lambda a, b, u=u: (u + .04 + .02 * math.sin(a * 6), 2.45 + .23 * a, 38 + 26 * b), 6, 4,
                lambda i, j: REDC)
    return (4, 3)


def tavern():
    walls(.3, 2.3, .45, 1.45, 31, EARTH, gable=31 + 18 * .5)
    door(.55, .85, 1.45, 26)
    window(1.1, 1.95, 1.45, 13, 22)
    window(.75, 1.15, 2.3, 13, 21, face="u")
    gable_roof(.3, 2.3, .45, 1.45, 31, 18, ov=.16, kind="tile")
    box(1.08, 1.98, 1.45, 1.55, 11, 13, WOOD)
    box(1.0, 1.9, 1.62, 1.95, 5, 7, P.PLANK)                # 평상
    for u in (1.05, 1.85):
        for v in (1.66, 1.9):
            box(u - .03, u + .03, v - .03, v + .03, 0, 5, WOOD)
    for u, v in ((2.45, .7), (2.62, .95), (2.45, 1.2), (2.68, 1.35), (2.4, 1.65)):
        jar(u, v, .1, 11)
    rod((2.75, 1.75, 0), (2.75, 1.75, 50), .03, WOOD)
    surface(lambda a, b: (2.73 - .31 * a, 1.75 + .015 * math.sin(a * 7), 26 + 22 * b), 6, 5,
            lambda i, j: REDC if j in (0, 4) or i in (0, 5) else CLOTH)
    m.ell(P3(.91, 1.6, 23), [.07, .07, 3.5 / PXU], REDC)      # 붉은 등
    rod((.91, 1.6, 26), (.91, 1.6, 30), .01, INK)
    return (3, 2)


def shop():
    walls(.3, 2.3, .4, 1.3, 31, PLASTER, gable=31 + 18 * .45)
    door(.5, .78, 1.3, 26)
    box(1.0, 2.1, 1.29, 1.32, 6, 24, INK)
    window(.65, 1.05, 2.3, 13, 21, face="u")
    gable_roof(.3, 2.3, .4, 1.3, 31, 18, ov=.14, kind="tile")
    surface(lambda a, b: (.95 + 1.2 * a, 1.3 + .55 * b, 26 - 7 * b - 1.2 * math.sin(math.pi * b)), 12, 3,
            lambda i, j: CLOTH if (i // 2) % 2 == 0 else REDC)
    for u in (.98, 2.12):
        rod((u, 1.82, 0), (u, 1.82, 19), .03, WOOD)
    box(1.05, 2.05, 1.45, 1.7, 0, 8, P.PLANK)
    for k, u in enumerate((1.15, 1.35, 1.6, 1.85)):
        if k < 2:
            box(u - .06, u + .06, 1.5, 1.62, 8, 9.5, IRON)
            rod((u - .05, 1.56, 9.5), (u + .07, 1.56, 11), .015, WOOD)
        else:
            m.ell(P3(u, 1.56, 10), [.08, .08, .1], THATCH)
    box(.3, .85, 1.75, 1.85, 0, 3, WOOD)
    for u in (.33, .82):
        rod((u, 1.8, 0), (u, 1.8, 22), .03, WOOD)
    box(.3, .85, 1.76, 1.84, 17, 19, WOOD)
    for u in np.linspace(.42, .74, 4):
        rod((u, 1.86, 2), (u, 1.86, 26), .015, WOOD)
        m.ell(P3(u, 1.86, 28.5), [.025, .025, .14], "#636971")
    return (3, 2)


BUILDINGS = {"cottage": cottage, "smithy": smithy, "yamen": yamen, "tavern": tavern, "shop": shop}


def make(name):
    global C
    m.meshes = []
    nu, nv = {"cottage": (2, 2), "smithy": (3, 2), "yamen": (4, 3), "tavern": (3, 2), "shop": (3, 2)}[name]
    C = (nu / 2, nv / 2)
    BUILDINGS[name]()
    return SP.draw(cell=(220, 220), pivot=(110, 160))


if __name__ == "__main__":
    out = sys.argv[1]
    os.makedirs(out, exist_ok=True)
    meta = {}
    for n in BUILDINGS:
        im, pv = make(n)
        im.save(os.path.join(out, n + ".png"))
        meta[n] = {"size": im.size, "pivot": pv}
        print(n, im.size, pv)
    json.dump(meta, open(os.path.join(out, "meta.json"), "w"), indent=1)
