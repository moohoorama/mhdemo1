"""A안 2차: 바닥을 넓혀 납작하게 (초가집 4×4, 대장간·주점·도구점 6×4, 관청 8×3), 초가집 4종, 저택 4종.
벽 높이(사람 38px 기준 문 25px)는 그대로, 지붕 경사를 낮춰 넓은 바닥에서 지붕이 솟지 않게 한다.

  python3 town_dot2.py OUTDIR
"""
import json
import os
import sys

import numpy as np
from PIL import Image

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import town_dot as T
from town_dot import Body, box, flat, earth, plaster, thatch, tiles, stone, bricks, wood, trough, logs, jar_body, post, lit
import gen_pixel_castle as pc
import shade

pc.PAL.update({
    "vG": (120, 146, 62), "vg": (84, 108, 44), "vS": (118, 86, 54), "vs": (96, 68, 42),
    "nW": (92, 124, 132), "nw": (70, 98, 108), "nL": (150, 178, 182),
    "mL": (236, 226, 204), "mS": (196, 184, 160), "mF": (118, 58, 40), "mf": (86, 42, 30),
    "kG": (96, 132, 70), "kg": (66, 96, 50), "kL": (136, 168, 88),
})

U, Vv = pc.Line("u"), pc.Line("v")


# ---------------------------------------------------------------- 지역 설정

ST = {}


def region(name):
    """지역 하나를 고른다: 팔레트를 바꾸고 모양 설정을 ST에 넣는다"""
    pc.PAL.update(BASE_PAL)
    r = REGIONS[name]
    pc.PAL.update(r["pal"])
    ST.clear()
    ST.update(r["st"])


# ---------------------------------------------------------------- 공용 부품

def gable(L, a0, a1, c0, c1, zw, s, wall, roof, ov=.16, ova=.14, t=None):
    """맞배 (기와는 지역의 기와 경사·처마 길이를 따른다)"""
    k = ST["tslope"] if roof == "tiles" else ST["slope"]
    e = ST["eave"]
    return T.gable_house(L, a0, a1, c0, c1, zw, s * k, wall, roof, ov=ov * e, ova=ova * e, t=t or (7 if roof == "thatch" else 4))


def folk_roof(L, a0, a1, c0, c1, zw, s, inset, m=.18):
    """서민 가옥 지붕: 벽 [a0,a1]x[c0,c1] 높이 zw 위에. 지역 설정에 따라 초가 우진각 / 맞배, 짚·갈대 / 기와"""
    k, e = ST["slope"], ST["eave"]
    s = s * k
    m = m * e
    if ST["folk_roof"] == "hip":
        if ST["folk_mat"] == "tiles":
            return tile_hip(L, a0 - m, a1 + m, c0 - m, c1 + m, zw - 2, s / k, ridge=True)
        return thatch_hip(L, a0 - m, a1 + m, c0 - m, c1 + m, zw - 4, s, inset)
    out = T.gable_house(L, a0, a1, c0, c1, zw, s, folk(), ST["folk_mat"], ov=m, ova=m, t=7 if ST["folk_mat"] == "thatch" else 4,
                        z0=zw - 1)
    return out


def thatch_hip(L, a0, a1, c0, c1, ze, s, inset):
    """두툼한 초가 우진각: 처마 짚 단면 띠, 사방 경사, 평평한 꼭대기와 짚 마루 (inset: 꼭대기까지 들어가는 칸)"""
    zt = ze + 5 + s * inset
    out = [Body(L.box(a0, a1, c0, c1, ze, ze + 5), T.thatch_band),
           Body(L.hip(a0 + .02, a1 - .02, c0 + .02, c1 - .02, ze + 5, s, top=zt), thatch(ze + 3, rim=0))]
    cm = (c0 + c1) / 2
    if a1 - a0 - 2 * inset > .1:
        out.append(Body(L.box(a0 + inset, a1 - inset, cm - .1, cm + .1, zt - 1, zt + 4), flat("hB", "hb", "hb")))
    return out


def tile_hip(L, a0, a1, c0, c1, ze, s, ridge=True, skirt=None):
    """기와 우진각 (skirt: 처마 쪽 완만한 자락 경사). 지역의 기와 경사·처마 길이를 따른다"""
    s = s * ST["tslope"]
    x = .15 * (ST["eave"] - 1)
    a0, a1, c0, c1, ze = a0 - x, a1 + x, c0 - x, c1 + x, ze - 6 * x
    out = []
    p = tiles(ze)
    if skirt:
        out.append(Body(L.hip(a0, a1, c0, c1, ze, skirt, top=ze + 8), p))
        d = 8 / skirt
        a0, a1, c0, c1, ze = a0 + d * .6, a1 - d * .6, c0 + d * .6, c1 - d * .6, ze + 5
    half = (c1 - c0) / 2
    zr = ze + s * half
    out.append(Body(L.hip(a0, a1, c0, c1, ze, s), p))
    if ridge:
        cm = (c0 + c1) / 2
        r0, r1 = a0 + half, a1 - half
        out.append(Body(L.box(r0 - .05, r1 + .05, cm - .07, cm + .07, zr - 3, zr + 2), flat("gK", "gK", "gB")))
        for a in (r0 - .1, r1 - .02):
            out.append(Body(L.box(a, a + .12, cm - .08, cm + .08, zr - 3, zr + 4), flat("gK", "gK", "gB")))
    return out


def folk(holes=()):
    """서민 가옥 벽: 지역에 따라 두꺼운 흙벽 / 흙벽+나무 뼈대 / 판자벽 / 대나무벽"""
    kind = ST["folk_wall"]
    if kind == "earth":
        return earth(holes)
    def p(f, u, v, z, s, h):
        if f == "+z" or f.startswith("r"):
            return np.full(u.shape, "eU")
        L = lit(f)
        if kind == "timber":
            col = earth()(f, u, v, z, s, h)
            frame = (np.mod(s, 12) == 0) | (h == 15) | (h == 16) | (h >= 27)
            col = np.where(frame & (h >= 3), "tW" if L else "tw", col)
        elif kind == "plank":
            col = np.where(np.mod(h, 4) == 0, "tw" if L else "tk", np.where((s * 7 + (h // 4) * 5) % 9 == 0, "tw", "tW" if L else "tw"))
            col = np.where(np.mod(s, 16) == 0, "tk", col)
            col = np.where(h < 3, "sr", col)
        else:   # bamboo: 세로 대나무 줄, 마디, 나무 기둥과 띠
            stripe = np.mod(s, 3)
            col = np.where(stripe == 0, "bm" if L else "bn", "bM" if L else "bm")
            col = np.where(np.mod(h + (s // 3) * 3, 9) == 0, "bn", col)
            col = np.where((np.mod(s, 24) == 0) | (h == 14) | (h == 15), "tw" if L else "tk", col)
            col = np.where(h < 3, "sr", col)
        return T.openings(col, f, u, v, s, h, holes)
    return p


def stilt(bodies, u0, u1, v0, v1):
    """강남 고상 가옥: 집을 ST["stilts"] 픽셀 띄우고 기둥 줄과 마루, 앞 디딤 계단을 단다"""
    L = ST.get("stilts", 0)
    if not L:
        return bodies
    out = []
    for b in bodies:
        planes = [(n, d + n[2] * L, f) for n, d, f in b.planes]
        out.append(Body(planes, (lambda p: lambda f, u, v, z, s, h: p(f, u, v, z - L, s, h - L))(b.paint), b.tile))
    out.append(Body(box(u0 - .12, u1 + .12, v0 - .12, v1 + .12, L - 2, L), flat("tE", "tw", "tE")))
    nu, nv = max(2, int(round((u1 - u0) / .5)) + 1), max(2, int(round((v1 - v0) / .5)) + 1)
    for u in np.linspace(u0 - .05, u1 + .05, nu):
        for v in np.linspace(v0 - .05, v1 + .05, nv):
            if u in (u0 - .05, u1 + .05) or v in (v0 - .05, v1 + .05):
                out.append(post(u, v, 0, L - 2, .04))
    um = u0 + .55
    for k in range(3):
        out.append(Body(box(um - .2, um + .2, v1 + .12 + .1 * k, v1 + .22 + .1 * k, 0, L - 2 - 2.5 * k), wood))
    return out


def lean_to(u0, u1, v0, v1, zt, zf, kind="thatch", t=5):
    """+u 쪽으로 낮아지는 외쪽 지붕 (헛간·차양)"""
    s = (zt - zf) / (u1 - u0)
    paint = thatch(zf, rim=3) if kind == "thatch" else tiles(zf)
    return [Body(box(u0, u1, v0, v1, 0, 1e9) + [((s, 0, 1), zt + s * u0 + t, "r+u"), ((-s, 0, -1), -(zt + s * u0), "b+u")],
                 paint)]


def lean_v(u0, u1, v0, v1, zt, zf, paint, t=2):
    """+v 쪽으로 낮아지는 천 차양"""
    s = (zt - zf) / (v1 - v0)
    return [Body(box(u0, u1, v0, v1, 0, 1e9) + [((0, s, 1), zt + s * v0 + t, "r+v"), ((0, -s, -1), -(zt + s * v0), "b+v")],
                 paint)]


def fence(pts, h=13):
    """울타리: 점들을 잇는 가로대 두 줄과 기둥 (축에 나란한 선분만)"""
    out = []
    for (u0, v0), (u1, v1) in zip(pts, pts[1:]):
        n = max(1, int(round(max(abs(u1 - u0), abs(v1 - v0)) / .5)))
        for k in range(n + 1):
            u, v = u0 + (u1 - u0) * k / n, v0 + (v1 - v0) * k / n
            out.append(post(u, v, 0, h, .035))
        for z in (4, 9):
            if u0 == u1:
                out.append(Body(box(u0 - .02, u0 + .02, min(v0, v1), max(v0, v1), z, z + 1.5), wood))
            else:
                out.append(Body(box(min(u0, u1), max(u0, u1), v0 - .02, v0 + .02, z, z + 1.5), wood))
    return out


def plot(u0, u1, v0, v1):
    """텃밭: u 방향 고랑, 이랑마다 푸른 싹"""
    def p(f, u, v, z, s, h):
        if f != "+z":
            return np.full(u.shape, "vs")
        i, j = np.floor(16 * u).astype(int), np.floor(16 * v).astype(int)
        crop = (np.mod(j, 4) < 2) & ((i * 5 + j) % 3 != 0)
        return np.where(np.mod(j, 4) == 3, "vs", np.where(crop, np.where(i % 2 == 0, "vG", "vg"), "vS"))
    return [Body(box(u0, u1, v0, v1, 0, 2), p)]


def paddy(u0, u1, v0, v1):
    """논: 낮은 둑 안에 물, 줄지어 선 벼 포기"""
    def bund(f, u, v, z, s, h):
        return np.full(u.shape, "vS" if f == "+z" else "vs")
    def p(f, u, v, z, s, h):
        if f != "+z":
            return np.full(u.shape, "vs")
        i, j = np.floor(16 * u).astype(int), np.floor(16 * v).astype(int)
        tuft = (np.mod(i, 4) < 2) & (np.mod(j, 4) < 2)
        return np.where(tuft, np.where((i + j) % 4 == 0, "vg", "vG"), np.where((i * 3 + j * 5) % 17 == 0, "nL", "nW"))
    w = .08
    return [Body(box(u0, u1, v0, v0 + w, 0, 3), bund), Body(box(u0, u1, v1 - w, v1, 0, 3), bund),
            Body(box(u0, u0 + w, v0, v1, 0, 3), bund), Body(box(u1 - w, u1, v0, v1, 0, 3), bund),
            Body(box(u0 + w, u1 - w, v0 + w, v1 - w, 0, 1.5), p)]


def field(u0, u1, v0, v1, kind=None):
    return (paddy if (kind or ST["fields"][0]) == "paddy" else plot)(u0, u1, v0, v1)


def field_dry():
    return plot(.2, 3.8, .2, 3.8), (4, 4)


def field_paddy():
    out = []
    for (a, b) in ((.2, 1.95), (2.05, 3.8)):
        out += paddy(.2, 3.8, a, b)
    return out, (4, 4)


def haystack(u, v, r=.35, h=20):
    return [Body(box(u - r, u + r, v - r, v + r, 0, 4), T.thatch_band),
            Body(U.hip(u - r + .03, u + r - .03, v - r + .03, v + r - .03, 4, h / r), thatch(2, rim=0))]


def shrub(u, v, r=.22, h=10):
    def p(f, uu, vv, z, s, hh):
        if f == "+z":
            i, j = np.floor(16 * uu).astype(int), np.floor(16 * vv).astype(int)
            return np.where((i * 3 + j * 7) % 5 == 0, "kL", "kG")
        return np.where((s * 5 + hh * 3) % 4 == 0, "kg", "kG" if lit(f) else "kg")
    return [Body(box(u - r, u + r, v - r, v + r, 0, h - 3), p),
            Body(U.hip(u - r + .04, u + r - .04, v - r + .04, v + r - .04, h - 3, 20), p)]


def pond(u0, u1, v0, v1):
    def p(f, u, v, z, s, h):
        if f != "+z":
            return np.full(u.shape, "sr")
        i, j = np.floor(16 * u).astype(int), np.floor(16 * v).astype(int)
        edge = (u < u0 + .12) | (u > u1 - .12) | (v < v0 + .12) | (v > v1 - .12)
        return np.where(edge, "sU", np.where((i * 3 + j * 5) % 13 == 0, "nL", np.where(v < v0 + .25, "nw", "nW")))
    return [Body(box(u0, u1, v0, v1, 0, 1.5), p)]


def cwall(u0, u1, v0, v1, h=14):
    """담장: 회벽 몸, 돌 굽, 기와 갓 (긴 쪽 방향 맞배)"""
    along_u = (u1 - u0) >= (v1 - v0)
    L = U if along_u else Vv
    a0, a1, c0, c1 = (u0, u1, v0, v1) if along_u else (v0, v1, u0, u1)
    return T.gable_house(L, a0, a1, c0, c1, h, 10, plaster(beam=None), "tiles", ov=.06, ova=.0, t=2)


def gate(u0, u1, v, h=26, roof_h=None):
    """+v 면을 향한 작은 대문: 기둥 둘, 붉은 문짝, 맞배 기와 지붕 (v는 담장 선)"""
    out = [post(u0, v, 0, h, .06, flat("mF", "mf")), post(u1, v, 0, h, .06, flat("mF", "mf"))]
    out.append(Body(box(u0 + .06, u1 - .06, v - .03, v + .03, 0, h - 4),
                    lambda f, u, vv, z, s, hh: np.where(np.mod(s, 4) == 0, "mf", np.where((hh % 8 == 4) & (np.mod(s, 4) == 2), "qG", "mF"))))
    out += T.gable_house(U, u0 - .15, u1 + .15, v - .22, v + .22, h, 14, flat("mF", "mf"), "tiles", ov=.1, ova=.0, t=3)[1:]
    return out


def chimney(u, v, h, r=.08):
    return [Body(box(u - r, u + r, v - r, v + r, 0, h), bricks), Body(box(u - r - .02, u + r + .02, v - r - .02, v + r + .02, h, h + 2), flat("bU", "br", "bU"))]


def mansion_wall(holes=()):
    """저택 벽: 회벽, 아래 판벽(목재 띠), 위 창방"""
    def p(f, u, v, z, s, h):
        if f == "+z" or f.startswith("r"):
            return np.full(u.shape, "mS")
        a, t = ("mL", "mF") if lit(f) else ("mS", "mf")
        col = np.full(u.shape, a, object)
        col = np.where(np.mod(s, 16) == 0, t, col)
        col = np.where((h >= 4) & (h < 10), np.where(np.mod(s, 4) == 0, t, "tW" if lit(f) else "tw"), col)
        col = np.where(h < 4, "sR" if not lit(f) else "sS", col)
        return T.openings(col, f, u, v, s, h, holes)
    return p


def yard_from(out, n):
    """out[n:]은 마당 것(담장·울타리·소품·밭): 집만 그릴 때 뺀다"""
    for b in out[n:]:
        b.yard = True
    return out


def jar_at(u, v, r, h, z0=0):
    return [Body(box(u - r, u + r, v - r, v + r, z0, z0 + h - 2), T.jar),
            Body(box(u - r * .6, u + r * .6, v - r * .6, v + r * .6, z0 + h - 2, z0 + h), flat("jT", "ja", "ja"))]


def stone_base(u0, u1, v0, v1, h=5):
    return Body(box(u0, u1, v0, v1, 0, h), stone)


# ---------------------------------------------------------------- 초가집 4종 (4×4)

def cottage_1():
    """기본: 한 채, 굴뚝, 장작더미·항아리, 앞마당 울타리 한 줄"""
    holes = [("+v", 1.0, 1.3, 3, 25, "door"), ("+v", 1.8, 2.3, 13, 21, "lattice"), ("+u", 1.3, 1.8, 13, 21, "lattice")]
    h = [Body(box(.6, 3.2, .7, 2.4, 0, 30), folk(holes))] + folk_roof(U, .6, 3.2, .7, 2.4, 30, 13, .95)
    out = stilt(h, .6, 3.2, .7, 2.4)
    if not ST.get("stilts"):
        out += chimney(.8, .85, 56 + 8 * (ST["slope"] - 1) * 4)
    n = len(out)
    out.append(Body(box(3.35, 3.7, .9, 1.6, 0, 9), logs))
    out.append(Body(box(3.35, 3.7, .9, 1.6, 9, 10), flat("tE", "tw")))
    out += jar_body(3.5, 2.0, .1, 9) + jar_body(3.62, 2.3, .08, 7)
    out += fence([(.4, 3.5), (2.4, 3.5)])
    return yard_from(out, n), (4, 4)


def cottage_2():
    """ㄱ자집: 안채(u 방향)와 오른쪽 날개(v 방향), 앞마당 장독대"""
    holes = [("+v", .9, 1.2, 3, 25, "door"), ("+v", 1.5, 1.9, 13, 21, "lattice"),
             ("+v", 2.75, 3.05, 3, 25, "door"), ("+u", 1.5, 1.9, 13, 21, "lattice"), ("+u", 2.4, 2.8, 13, 21, "lattice")]
    h = [Body(box(.4, 2.5, .5, 1.9, 0, 30), folk(holes)), Body(box(2.4, 3.6, .5, 3.2, 0, 30), folk(holes))]
    h += folk_roof(U, .4, 3.6, .5, 1.9, 30, 13, .88) + folk_roof(Vv, .5, 3.2, 2.4, 3.6, 30, 13, .78)
    out = stilt(h, .4, 3.6, .5, 1.9) if ST.get("stilts") else h
    if not ST.get("stilts"):
        out += chimney(.6, .62, 54 + 8 * (ST["slope"] - 1) * 4)
    n = len(out)
    out.append(Body(box(.6, 1.6, 2.4, 3.0, 0, 3), stone))
    for u, v, r, hh in ((.8, 2.6, .11, 10), (1.1, 2.55, .12, 11), (1.4, 2.65, .09, 8), (.95, 2.85, .08, 7), (1.3, 2.85, .1, 9)):
        out += jar_at(u, v, r, hh, 3)
    return yard_from(out, n), (4, 4)


def cottage_3():
    """맞배 + 오른쪽 헛간(외쪽 지붕, 짚단·농기구), 앞 울타리"""
    holes = [("+v", .9, 1.2, 3, 25, "door"), ("+v", 1.5, 2.0, 13, 21, "lattice")]
    k = ST["slope"]
    mat = ST["folk_mat"]
    h = gable(U, .45, 2.55, .7, 2.3, 30, 12, folk(holes), mat, ov=.18, ova=.16)
    out = stilt(h, .45, 2.55, .7, 2.3)
    out += lean_to(2.55, 3.65, .6, 2.45, 30, 22, mat if mat == "thatch" else "tiles")
    out.append(Body(box(2.55, 2.65, .7, 2.3, 0, 30), folk()))
    for v in (.75, 2.3):
        out.append(post(3.55, v, 0, 22, .045))
    out += haystack(3.05, 1.2, .3, 14)
    out.append(Body(box(2.8, 3.3, 1.75, 2.15, 0, 7), wood))
    out.append(Body(box(3.38, 3.42, 1.6, 1.64, 0, 22), flat("tw", "tk")))
    n = len(out)
    out += fence([(.3, 3.6), (3.7, 3.6)], 12)
    out += fence([(.3, 2.6), (.3, 3.6)], 12)
    return yard_from(out, n), (4, 4)


def cottage_4():
    """작은 집 + 앞 밭(또는 논), 오른쪽 낟가리, 빨랫줄"""
    holes = [("+v", .85, 1.15, 3, 25, "door"), ("+u", .8, 1.2, 13, 21, "lattice"), ("+v", 1.5, 1.9, 13, 21, "lattice")]
    h = [Body(box(.45, 2.35, .45, 1.9, 0, 29), folk(holes))] + folk_roof(U, .45, 2.35, .45, 1.9, 29, 14, .7)
    out = stilt(h, .45, 2.35, .45, 1.9)
    if not ST.get("stilts"):
        out += chimney(2.2, .6, 50 + 8 * (ST["slope"] - 1) * 4)
    n = len(out)
    out += field(.45, 2.6, 2.4, 3.7, ST["fields"][-1])
    out += haystack(3.2, .9, .4, 22)
    out += haystack(3.25, 1.95, .32, 18)
    for u in (2.75, 3.75):
        out.append(post(u, 2.8, 0, 20, .03))
    out.append(Body(box(2.75, 3.75, 2.79, 2.81, 18, 18.6), flat("tk", "tk")))
    for u0, c in ((2.9, "wA"), (3.2, "rA"), (3.45, "wa")):
        out.append(Body(box(u0, u0 + .2, 2.79, 2.82, 10, 18), flat(c, c)))
    return yard_from(out, n), (4, 4)


# ---------------------------------------------------------------- 6×4

def smithy():
    """대장간: 앞이 트인 넓은 초가 헛간, 안쪽 벽돌 화덕 둘과 풀무, 오른쪽 둥근 굴뚝, 앞마당 모루·물통·숫돌, 쇠 무더기"""
    a0, a1, c0, c1, zw = .4, 4.4, .5, 2.5, 30
    out = folk_roof(U, a0, a1, c0, c1, zw, 13, .9)
    out.append(Body(box(a0, a1, c0, c0 + .1, 0, zw), folk()))
    out.append(Body(box(a0, a0 + .1, c0, c1, 0, zw), folk()))
    for u in (a0 + .05, 1.75, 3.05, a1 - .05):
        out.append(post(u, c1 - .05, 0, zw, .05))
    out.append(post(a1 - .05, 1.5, 0, zw, .05))
    out.append(Body(box(a0, a1, c1 - .1, c1, zw - 3, zw), wood))
    for u0 in (.6, 2.3):
        out.append(Body(box(u0, u0 + .8, .65, 1.25, 0, 16), forge_at(u0)))
        out.append(Body(box(u0 + .2, u0 + .6, .7, 1.0, 16, 44), bricks))
    out.append(Body(box(1.5, 2.0, .7, 1.1, 0, 10), wood))                # 풀무
    out.append(Body(box(1.55, 1.95, .75, 1.05, 10, 12), flat("tk", "tk", "dL")))
    out += chimney(4.85, .9, 72, .17)
    n = len(out)
    for u, v in ((1.6, 3.0), (3.3, 3.1)):                                # 모루 둘
        out.append(Body(box(u - .08, u + .08, v - .07, v + .07, 0, 6), wood))
        out.append(Body(box(u - .16, u + .16, v - .05, v + .05, 6, 9), flat("iA", "ia", "iU")))
    out.append(Body(box(4.6, 5.5, 2.4, 2.7, 0, 6), trough))
    out.append(Body(box(4.65, 5.45, 2.44, 2.66, 6, 6.5), flat("nw", "nw", "nW")))
    out.append(Body(box(.6, 1.1, 2.9, 3.3, 0, 5), flat("iA", "ia", "iU")))  # 쇠 무더기
    out.append(Body(box(.7, 1.0, 3.0, 3.2, 5, 8), flat("iA", "ia", "iU")))
    out.append(Body(box(5.0, 5.6, 1.2, 1.9, 0, 9), logs))                # 숯 장작
    return yard_from(out, n), (6, 4)


def forge_at(u0):
    def p(f, u, v, z, s, h):
        if f == "+z":
            i, j = np.floor(16 * u).astype(int), np.floor(16 * v).astype(int)
            return np.where((u > u0 + .2) & (u < u0 + .6) & (v > .8) & (v < 1.1), np.where((i + j) % 2 == 0, "fA", "fB"), "bU")
        col = bricks(f, u, v, z, s, h)
        if f == "+v":
            m = (u > u0 + .2) & (u < u0 + .6) & (h >= 3) & (h < 11)
            col = np.where(m, np.where(h < 5, "fA", np.where(h < 8, "fB", "fC")), col)
        return col
    return p


def tavern():
    """주점: 넓은 흙벽 기와집(앞 들창 셋), 오른쪽 외쪽 차양 아래 평상 둘, 술항아리 무더기, 붉은 등 둘, 술깃발"""
    holes = [("+v", .7, 1.0, 3, 26, "door"), ("+v", 1.3, 2.1, 13, 22, "lattice"), ("+v", 2.4, 3.2, 13, 22, "lattice"),
             ("+v", 3.5, 3.8, 3, 26, "door"), ("+u", 1.0, 1.6, 13, 21, "lattice")]
    out = gable(U, .4, 4.2, .5, 2.2, 31, 11, folk(holes), "tiles", ov=.16, ova=.14)
    for u0 in (1.3, 2.4):
        out.append(Body(box(u0 - .02, u0 + .82, 2.2, 2.3, 11, 13), wood))
    out += lean_to(4.2, 5.7, .6, 2.3, 27, 20, "tiles", t=3)
    for v in (.65, 2.25):
        out.append(post(5.62, v, 0, 20, .04))
    for u in (1.2, 3.3):
        out.append(Body(box(u - .05, u + .05, 2.38, 2.48, 20, 26), flat("lR", "lr")))
        out.append(Body(box(u - .01, u + .01, 2.42, 2.44, 26, 29), flat("tk", "tk")))
    n = None
    for (u, v) in ((4.5, 1.2), (4.6, 2.9)):
        if n is None and v > 2.5:
            n = len(out)
        out.append(Body(box(u, u + .8, v, v + .45, 5, 7), flat("tE", "tw")))
        for du in (.05, .72):
            for dv in (.05, .38):
                out.append(Body(box(u + du, u + du + .06, v + dv, v + dv + .06, 0, 5), wood))
    for u, v, r, h in ((.5, 2.6, .12, 11), (.75, 2.75, .12, 12), (.6, 3.0, .1, 9), (.95, 3.05, .13, 12), (.45, 3.3, .1, 9), (.8, 3.4, .11, 10)):
        out += jar_body(u, v, r, h)
    out.append(post(5.8, 3.4, 0, 56, .035))
    out.append(Body(box(5.4, 5.77, 3.38, 3.42, 32, 54), flag_white(5.4, 5.77)))
    return yard_from(out, n), (6, 4)


def flag_white(u0, u1):
    def p(f, u, v, z, s, h):
        edge = (h < 34) | (h > 51) | (u < u0 + .04) | (u > u1 - .04)
        return np.where(edge, "rA", "wA" if lit(f) else "wa")
    return p


def shop():
    """도구점: 회벽 기와집, 앞 전체에 홍백 천 차양과 진열대 셋(연장·바구니·항아리), 왼쪽 창 걸이, 오른쪽 통 무더기"""
    holes = [("+v", .6, .9, 3, 26, "door"), ("+v", 1.2, 4.2, 6, 24, "dark"), ("+u", 1.0, 1.5, 13, 21, "lattice")]
    out = gable(U, .4, 4.6, .4, 1.9, 31, 11, plaster(holes, beam=28), "tiles", ov=.14, ova=.14)
    out += lean_v(1.1, 4.3, 1.9, 2.75, 26, 19, T.awning)
    for u in (1.13, 2.7, 4.27):
        out.append(post(u, 2.72, 0, 19, .03))
    for k, (u0, goods) in enumerate(((1.25, "iron"), (2.25, "basket"), (3.25, "iron"))):
        out.append(Body(box(u0, u0 + .85, 2.05, 2.5, 0, 8), wood))
        for i in range(3):
            u = u0 + .15 + i * .28
            out.append(Body(box(u - .07, u + .07, 2.15, 2.35, 8, 10),
                            flat("iA", "ia", "iU") if goods == "iron" else flat("hA", "hb", "hB")))
    n = len(out)
    out.append(Body(box(.35, 1.0, 2.75, 2.85, 0, 3), wood))
    for u in (.38, .97):
        out.append(post(u, 2.8, 0, 22, .035))
    out.append(Body(box(.35, 1.0, 2.76, 2.84, 17, 19), wood))
    for u in np.linspace(.48, .88, 5):
        out.append(Body(box(u - .015, u + .015, 2.85, 2.88, 2, 26), flat("tw", "tk")))
        out.append(Body(box(u - .03, u + .03, 2.85, 2.88, 26, 31), flat("iU", "iA")))
    for u, v in ((4.9, .8), (5.25, .9), (5.0, 1.25), (5.35, 1.3)):
        out.append(Body(box(u - .13, u + .13, v - .13, v + .13, 0, 11), barrel))
    out += jar_body(5.1, 2.6, .12, 11) + jar_body(5.4, 2.9, .1, 9)
    return yard_from(out, n), (6, 4)


def barrel(f, u, v, z, s, h):
    if f == "+z":
        return np.full(u.shape, "tE")
    return np.where((h == 2) | (h == 8), "ia", np.where(np.mod(s, 3) == 0, "tw" if lit(f) else "tk", "tW" if lit(f) else "tw"))


# ---------------------------------------------------------------- 관청 8×3

def yamen():
    """관청: 긴 석축 기단, 가운데 높은 정청(붉은 기둥 툇간·창살문·빈 편액·이중 우진각), 양쪽 낮은 곁채, 가운데 계단, 깃발 넷"""
    out = [Body(box(.2, 7.8, .2, 2.35, 0, 8), stone)]
    for k in range(3):
        out.append(Body(box(3.2, 4.8, 2.35, 2.35 + .18 * (3 - k), 0, 8 - 2.6 * (k + 1) + 2.6), stone))
    zb, zw = 8, 8 + 38
    holes = [("+v", 2.5, 3.3, 3, 24, "redlattice"), ("+v", 3.6, 4.4, 3, 26, "redlattice"), ("+v", 4.7, 5.5, 3, 24, "redlattice")]
    out.append(Body(box(2.2, 5.8, .5, 1.6, zb, zw), plaster(holes, beam=31)))
    for u in (2.2, 2.75, 3.45, 4.55, 5.25, 5.8):
        out.append(post(u, 1.9, zb, zw, .055, flat("cR", "cr")))
    out.append(Body(box(2.15, 5.85, 1.85, 1.95, zw - 4, zw), flat("cr", "cr", "cR")))
    out.append(Body(box(3.7, 4.3, 1.95, 1.98, zw - 14, zw - 5), plaque(3.7, 4.3)))
    out += tile_hip(U, 1.9, 6.1, .2, 2.2, zw - 2, 13, skirt=7)
    # 곁채
    for (a0, a1) in ((.45, 2.15), (5.85, 7.55)):
        h2 = [("+v", a0 + .35, a1 - .35, 3, 20, "redlattice"), ("+u", .8, 1.4, 10, 18, "redlattice")]
        out.append(Body(box(a0, a1, .55, 1.65, zb, zb + 28), plaster(h2, beam=23)))
        out += tile_hip(U, a0 - .2, a1 + .2, .3, 1.95, zb + 26, 11)
    for u in (.35, 7.65):
        out.append(post(u, 2.55, 0, 64, .03))
        out.append(Body(box(u + .03, u + .06, 2.3, 2.52, 36, 62), banner_at(36, 62)))
    return out, (8, 3)


def plaque(u0, u1):
    def p(f, u, v, z, s, h):
        hm, hx = (h.min(), h.max()) if h.size else (0, 0)
        edge = (u < u0 + .05) | (u > u1 - .05) | (h <= hm) | (h >= hx)
        return np.where(edge, "qG", "qD")
    return p


def banner_at(z0, z1):
    return lambda f, u, v, z, s, h: np.where((h < z0 + 2) | (h > z1 - 3), "ra", "rA" if f != "+u" else "ra")


# ---------------------------------------------------------------- 저택 4종

def mansion_1():
    """담장 두른 저택 (6×4): 뒤 본채(기단·툇마루·우진각 기와), 앞 담장 가운데 대문, 마당 나무·석등"""
    out = [stone_base(1.0, 5.0, .4, 1.9)]
    holes = [("+v", 2.0, 2.4, 5, 26, "redlattice"), ("+v", 2.8, 3.2, 5, 28, "redlattice"), ("+v", 3.6, 4.0, 5, 26, "redlattice"),
             ("+u", .8, 1.3, 13, 22, "lattice")]
    out.append(Body(box(1.3, 4.7, .55, 1.55, 5, 36), mansion_wall(holes)))
    for u in (1.3, 2.15, 3.0, 3.85, 4.7):
        out.append(post(u, 1.75, 5, 36, .05, flat("mF", "mf")))
    out += tile_hip(U, 1.05, 4.95, .3, 2.0, 33, 12)
    n = len(out)
    out += cwall(.2, .38, .2, 3.8) + cwall(5.62, 5.8, .2, 3.8)
    out += cwall(.2, 2.55, 3.62, 3.8) + cwall(3.45, 5.8, 3.62, 3.8)
    out += gate(2.6, 3.4, 3.71)
    out += shrub(1.0, 2.7, .3, 22) + shrub(4.8, 2.6, .25, 14) + shrub(5.1, 3.1, .2, 10)
    out.append(Body(box(3.9, 4.1, 2.5, 2.7, 0, 9), stone))         # 석등
    out.append(Body(box(3.85, 4.15, 2.45, 2.75, 9, 11), flat("sS", "sR", "sU")))
    return yard_from(out, n), (6, 4)


def mansion_2():
    """ㄱ자 저택 (6×4): 안채(u)와 사랑채(v, 박공이 앞을 봄), 낮은 담과 연못"""
    out = [stone_base(.4, 4.2, .4, 1.8), stone_base(4.0, 5.5, .4, 3.4)]
    h1 = [("+v", 1.0, 1.4, 5, 26, "redlattice"), ("+v", 1.9, 2.3, 5, 26, "redlattice"), ("+v", 2.8, 3.2, 5, 26, "redlattice")]
    out.append(Body(box(.6, 4.1, .55, 1.6, 5, 35), mansion_wall(h1)))
    out += tile_hip(U, .35, 4.35, .3, 1.85, 32, 12)
    h2 = [("+u", 1.0, 1.5, 13, 24, "lattice"), ("+u", 2.0, 2.6, 5, 27, "redlattice"), ("+v", 4.45, 5.05, 13, 24, "lattice")]
    out += gable(Vv, .6, 3.2, 4.15, 5.35, 35, 12, mansion_wall(h2), "tiles", ov=.14, ova=.14)
    n = len(out)
    out += pond(.7, 2.4, 2.3, 3.3)
    out += shrub(2.8, 2.5, .25, 14) + shrub(.55, 2.2, .2, 10) + shrub(3.3, 3.2, .3, 22)
    out += cwall(.2, 3.85, 3.62, 3.8, 12) + cwall(.2, .38, 1.9, 3.8, 12)
    return yard_from(out, n), (6, 4)


def mansion_3():
    """긴 저택 (8×3): 높은 기단 위 긴 본채, 오른쪽 이층 누각, 앞 낮은 담장과 가운데 솟을대문"""
    out = [stone_base(.4, 6.0, .3, 1.6, 6)]
    holes = [("+v", a, a + .4, 6, 27, "redlattice") for a in (1.0, 1.8, 2.6, 3.4, 4.2, 5.0)]
    out.append(Body(box(.6, 5.8, .45, 1.35, 6, 36), mansion_wall(holes)))
    for u in np.linspace(.6, 5.8, 7):
        out.append(post(u, 1.5, 6, 36, .045, flat("mF", "mf")))
    out += gable(U, .6, 5.8, .45, 1.6, 36, 12, flat("mS", "mS"), "tiles", ov=.16, ova=.2)[1:]
    # 누각: 아래 기둥층, 위 난간 방
    out.append(stone_base(6.1, 7.7, .3, 1.9, 6))
    for u in (6.25, 7.55):
        for v in (.45, 1.75):
            out.append(post(u, v, 6, 30, .06, flat("cR", "cr")))
    out.append(Body(box(6.15, 7.65, .35, 1.85, 30, 34), flat("mF", "mf", "tE")))
    out.append(Body(box(6.3, 7.5, .5, 1.7, 34, 54), mansion_wall([("+v", 6.5, 7.3, 8, 18, "redlattice"), ("+u", .7, 1.5, 8, 18, "redlattice")])))
    out += tile_hip(U, 5.95, 7.85, .15, 2.05, 52, 13, skirt=7)
    n = len(out)
    out += cwall(.2, 3.45, 2.62, 2.8, 12) + cwall(4.55, 7.8, 2.62, 2.8, 12)
    out += gate(3.5, 4.5, 2.71, 30)
    out += shrub(1.2, 2.1, .22, 12) + shrub(6.6, 2.2, .25, 16)
    return yard_from(out, n), (8, 3)


def mansion_4():
    """작은 저택 (4×4): 기와 우진각 한 채와 대청, 앞 낮은 담장(왼쪽 쪽문), 나무 한 그루"""
    out = [stone_base(.5, 3.5, .4, 2.0)]
    holes = [("+v", 1.1, 1.5, 5, 27, "redlattice"), ("+v", 1.9, 2.5, 5, 27, "dark"), ("+u", .8, 1.3, 13, 22, "lattice")]
    out.append(Body(box(.7, 3.3, .55, 1.8, 5, 35), mansion_wall(holes)))
    for u in (.7, 1.6, 2.4, 3.3):
        out.append(post(u, 1.95, 5, 35, .045, flat("mF", "mf")))
    out += tile_hip(U, .45, 3.55, .3, 2.2, 32, 12)
    n = len(out)
    out += cwall(1.4, 3.8, 3.62, 3.8, 12) + cwall(3.62, 3.8, 1.5, 3.8, 12) + cwall(.2, .5, 3.62, 3.8, 12)
    out += gate(.55, 1.35, 3.71, 22)
    out += shrub(2.9, 2.8, .32, 26) + shrub(1.6, 2.9, .18, 9)
    return yard_from(out, n), (4, 4)


BUILDINGS = {
    "cottage_1": cottage_1, "cottage_2": cottage_2, "cottage_3": cottage_3, "cottage_4": cottage_4,
    "smithy": smithy, "tavern": tavern, "shop": shop, "yamen": yamen,
    "mansion_1": mansion_1, "mansion_2": mansion_2, "mansion_3": mansion_3, "mansion_4": mansion_4,
    "field_dry": field_dry, "field_paddy": field_paddy,
}

BASE_PAL = dict(pc.PAL)
BASE_PAL.update({"bM": (196, 186, 112), "bm": (160, 152, 86), "bn": (112, 108, 60)})

REGIONS = {
    "huabei": {"name": "화북", "ground": ((184, 148, 96), (160, 126, 80)), "pal": {},
               "st": dict(slope=1, tslope=1, eave=1, folk_roof="hip", folk_mat="thatch", folk_wall="earth", fields=["dry"])},
    "zhongyuan": {"name": "중원", "ground": ((176, 132, 92), (150, 108, 74)),
                  "pal": {"eL": (206, 150, 100), "el": (184, 128, 84), "eS": (160, 108, 70), "es": (134, 88, 56), "eU": (192, 140, 94),
                          "hA": (204, 162, 96), "hB": (178, 136, 76), "ha": (160, 122, 66), "hb": (132, 96, 52), "hR": (104, 74, 40),
                          "gA": (92, 94, 104), "gB": (70, 72, 82), "ga": (68, 70, 80), "gb": (50, 52, 60), "gE": (148, 150, 158),
                          "pL": (232, 208, 172), "pS": (184, 158, 126), "mL": (232, 212, 178), "mS": (194, 172, 140),
                          "mF": (146, 50, 38), "mf": (106, 36, 28), "kG": (110, 128, 64), "kg": (80, 94, 46)},
                  "st": dict(slope=1.1, tslope=1.05, eave=1, folk_roof="gable", folk_mat="thatch", folk_wall="timber", fields=["dry"])},
    "huainan": {"name": "회남", "ground": ((150, 140, 100), (126, 118, 82)),
                "pal": {"eL": (180, 156, 118), "el": (160, 136, 100), "eS": (138, 116, 84), "es": (116, 96, 68), "eU": (168, 144, 108),
                        "hA": (186, 178, 124), "hB": (160, 154, 102), "ha": (144, 138, 94), "hb": (116, 112, 74), "hR": (92, 90, 58), "hD": (72, 70, 46),
                        "gA": (96, 112, 114), "gB": (74, 88, 90), "ga": (72, 84, 88), "gb": (52, 62, 66), "gE": (150, 166, 166), "gK": (40, 48, 52),
                        "mL": (238, 236, 224), "mS": (200, 200, 188), "kG": (124, 164, 82), "kg": (88, 124, 60), "kL": (164, 196, 108),
                        "vG": (136, 172, 76), "vg": (96, 132, 58), "nW": (96, 136, 132), "nw": (74, 110, 108)},
                "st": dict(slope=1.35, tslope=1.15, eave=1, folk_roof="hip", folk_mat="thatch", folk_wall="earth", fields=["paddy", "dry"])},
    "jiangnan": {"name": "강남", "ground": ((128, 136, 118), (104, 112, 98)),
                 "pal": {"tW": (124, 88, 58), "tw": (92, 64, 42), "tk": (60, 42, 30), "tE": (156, 118, 80),
                         "gA": (86, 100, 102), "gB": (64, 78, 82), "ga": (62, 74, 78), "gb": (46, 56, 60), "gE": (140, 158, 160), "gK": (34, 42, 46),
                         "pL": (238, 238, 230), "pS": (196, 198, 190), "mL": (240, 240, 232), "mS": (202, 204, 196), "mF": (62, 54, 50), "mf": (44, 38, 36),
                         "sS": (168, 172, 166), "ss": (134, 138, 132), "sR": (126, 130, 124), "sr": (96, 100, 96), "sU": (156, 160, 154),
                         "kG": (72, 124, 98), "kg": (50, 92, 72), "kL": (110, 160, 128), "nW": (84, 128, 136), "nw": (64, 100, 110)},
                 "st": dict(slope=1.1, tslope=1.1, eave=1.8, folk_roof="gable", folk_mat="tiles", folk_wall="plank", stilts=9,
                            fields=["paddy"])},
    "bashu": {"name": "파촉", "ground": ((112, 116, 90), (90, 94, 72)),
              "pal": {"hA": (156, 128, 80), "hB": (130, 104, 62), "ha": (120, 98, 58), "hb": (98, 78, 46), "hR": (78, 60, 36), "hD": (60, 46, 28),
                      "gA": (78, 88, 84), "gB": (58, 68, 64), "ga": (56, 64, 62), "gb": (42, 48, 46), "gE": (128, 140, 134), "gK": (32, 38, 36),
                      "sS": (150, 154, 136), "ss": (112, 122, 100), "sR": (118, 122, 106), "sr": (84, 92, 74), "sU": (140, 146, 126),
                      "mL": (226, 220, 200), "mS": (186, 180, 160), "mF": (96, 62, 40), "mf": (70, 44, 30),
                      "kG": (54, 98, 54), "kg": (36, 70, 40), "kL": (84, 130, 72), "vG": (80, 128, 60), "vg": (56, 96, 44)},
              "st": dict(slope=1.5, tslope=1.2, eave=1.1, folk_roof="gable", folk_mat="thatch", folk_wall="bamboo", fields=["paddy"])},
}


def house_only(bodies):
    """마당 것을 빼고, 남은 몸체에 맞춰 바닥을 다시 잡아 (0, 0)으로 옮긴다 (처마는 0.3칸까지 바닥 밖으로 나가도 됨)"""
    bodies = [b for b in bodies if not getattr(b, "yard", False)]
    u0 = min(-d for b in bodies for n, d, f in b.planes if n == (-1, 0, 0))
    u1 = max(d for b in bodies for n, d, f in b.planes if n == (1, 0, 0))
    v0 = min(-d for b in bodies for n, d, f in b.planes if n == (0, -1, 0))
    v1 = max(d for b in bodies for n, d, f in b.planes if n == (0, 1, 0))
    fu0, fv0 = int(np.floor(u0 + .3)), int(np.floor(v0 + .3))
    fu1, fv1 = int(np.ceil(u1 - .3)), int(np.ceil(v1 - .3))
    out = []
    for b in bodies:
        planes = [(n, d - n[0] * fu0 - n[1] * fv0, f) for n, d, f in b.planes]
        paint = (lambda p: lambda f, u, v, z, s, h: p(f, u + fu0, v + fv0, z, s, h))(b.paint)
        out.append(Body(planes, paint, b.tile))
    return out, (fu1 - fu0, fv1 - fv0)


def make(name):
    bodies, (nu, nv) = BUILDINGS[name]()
    if not name.startswith("field_"):
        bodies, (nu, nv) = house_only(bodies)
    a, o = shade.render(bodies, (-16 * nv - 30, -140, 16 * nu + 30, 8 * (nu + nv) + 12))
    im = Image.fromarray(a)
    bb = im.getbbox()
    im = im.crop(bb)
    cx, cy = 16 * (nu - nv) / 2, 8 * (nu + nv) / 2
    return im, (int(cx - o[0] - bb[0]), int(cy - o[1] - bb[1])), (nu, nv)


if __name__ == "__main__":
    out = sys.argv[1]
    regions = [a for a in sys.argv[2:] if a in REGIONS] or list(REGIONS)
    names = [a for a in sys.argv[2:] if a in BUILDINGS] or list(BUILDINGS)
    for rg in regions:
        region(rg)
        d = os.path.join(out, rg)
        os.makedirs(d, exist_ok=True)
        path = os.path.join(d, "meta.json")
        meta = json.load(open(path)) if os.path.exists(path) else {}
        for n in names:
            if n.startswith("field_") and n[6:] not in ST["fields"]:
                continue
            im, pv, fp = make(n)
            im.save(os.path.join(d, n + ".png"))
            meta[n] = {"size": im.size, "pivot": pv, "footprint": fp}
        json.dump(meta, open(path, "w"), indent=1)
        print(rg, len(meta))
