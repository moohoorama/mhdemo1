"""A안: 화북 성 컨셉의 성 안 건물을 gen_pixel_castle 방식(2:1 소타일 위 상자 면에 도트 무늬)으로 직접 찍기.

  python3 town_dot.py OUTDIR   -> OUTDIR/<name>.png (원본 크기, pivot = 발자리 중심)
"""
import json
import os
import sys

import numpy as np
from PIL import Image

sys.path.insert(0, "/Users/younwookim/orca/mhdemo1/assets_candidate/tools")
import gen_pixel_castle as pc
from gen_pixel_castle import Body, box, brick

pc.PAL.update({
    # 흙벽(판축): 밝은 면, 그늘 면, 갈라짐, 기둥
    "eL": (210, 168, 112), "el": (186, 144, 92), "eS": (160, 120, 78), "es": (138, 102, 64), "eU": (196, 156, 104),
    # 회벽
    "pL": (230, 216, 184), "pl": (206, 190, 156), "pS": (178, 162, 130), "ps": (156, 140, 112),
    # 나무
    "tW": (132, 84, 48), "tw": (96, 60, 36), "tk": (62, 40, 26), "tE": (168, 118, 72),
    # 초가
    "hA": (222, 182, 100), "hB": (194, 150, 74), "ha": (172, 132, 66), "hb": (140, 104, 50), "hR": (110, 80, 40), "hD": (90, 64, 34),
    # 기와
    "gA": (100, 108, 122), "gB": (76, 82, 96), "ga": (74, 78, 92), "gb": (54, 58, 70), "gE": (156, 162, 172), "gK": (40, 42, 52),
    # 붉은 기둥
    "cR": (178, 56, 40), "cr": (126, 38, 30),
    # 석축
    "sS": (196, 184, 158), "ss": (160, 148, 124), "sR": (150, 136, 112), "sr": (114, 102, 84), "sU": (182, 172, 150),
    # 벽돌(대장간 굴뚝·화덕)
    "bS": (170, 112, 78), "bs": (124, 80, 56), "bR": (130, 86, 60), "br": (94, 62, 44), "bU": (112, 76, 56),
    # 불, 쇠
    "fA": (255, 214, 96), "fB": (244, 128, 44), "fC": (170, 48, 26), "iA": (82, 86, 96), "ia": (50, 52, 60), "iU": (120, 126, 138),
    # 구멍, 창살
    "dH": (46, 34, 28), "dL": (88, 62, 44),
    # 항아리, 천, 편액
    "jA": (128, 64, 42), "ja": (92, 44, 30), "jT": (160, 96, 62),
    "wA": (236, 228, 208), "wa": (196, 186, 166), "rA": (184, 50, 42), "ra": (132, 34, 30),
    "qG": (220, 176, 76), "qD": (56, 40, 32), "lR": (214, 60, 44), "lr": (150, 36, 30),
    "yA": (190, 158, 112),
})


def lit(f):
    return f.endswith("v")


def flat(k_lit, k_shade, k_top=None):
    return lambda f, u, v, z, s, h: np.full(u.shape, (k_top or k_lit) if f == "+z" else k_lit if lit(f) else k_shade)


def earth(holes=(), posts=8, footing=3):
    """흙벽: 밝은 바탕에 흙 결(가로로 뭉친 점), 모서리·간격마다 나무 기둥, 아래 돌 굽. holes = (face, t0, t1, h0, h1, kind)"""
    def p(f, u, v, z, s, h):
        if f == "+z" or f.startswith("r"):
            return np.full(u.shape, "eU")
        a, b, c = ("eL", "el", "es") if lit(f) else ("eS", "es", "tw")
        n = (s * 7 + (h // 2) * 13) % 23
        col = np.where(n == 0, b, np.where(n == 11, c, a))
        col = np.where((h % 7 == 3) & ((s + h) % 5 == 0), b, col)
        if posts:
            col = np.where(np.mod(s, posts * 4) == 0, "tw" if lit(f) else "tk", col)
        col = np.where(h < footing, np.where((s + (h % 2) * 2) % 4 == 0, "sr", "ss" if lit(f) else "sr"), col)
        return openings(col, f, u, v, s, h, holes)
    return p


def plaster(holes=(), beam=None):
    def p(f, u, v, z, s, h):
        if f == "+z" or f.startswith("r"):
            return np.full(u.shape, "pS")
        a, t = ("pL", "tW") if lit(f) else ("pS", "tw")
        col = np.full(u.shape, a, object)
        col = np.where(np.mod(s, 16) == 0, t, col)
        if beam is not None:
            col = np.where((h >= beam) & (h < beam + 2), t, col)
        col = np.where(h < 3, "sr" if not lit(f) else "ss", col)
        return openings(col, f, u, v, s, h, holes)
    return p


def openings(col, f, u, v, s, h, holes):
    at = u if f.endswith("v") else v
    for fc, t0, t1, h0, h1, kind in holes:
        m = (f == fc) & (at >= t0) & (at < t1) & (h >= h0) & (h < h1)
        edge = m & ((at < t0 + 1 / 16) | (at >= t1 - 1 / 16) | (h == h1 - 1))
        if kind == "door":
            fill = np.where(np.mod(s, 4) == 0, "tk", "tw")
        elif kind == "dark":
            fill = np.full(u.shape, "dH", object)
        elif kind == "lattice":
            fill = np.where((np.mod(s, 3) == 0) | (np.mod(h, 3) == 0), "tW", "dH")
        elif kind == "redlattice":
            fill = np.where((np.mod(s, 3) == 0) | (np.mod(h, 3) == 0), "cr", "dL")
        col = np.where(m, fill, col)
        col = np.where(edge & (kind != "dark"), "tk", col)
    return col


def thatch(ze, rim=3):
    """짚: 세 줄마다 짙은 골, 점점이 짚 끝, 처마 끝(아래 rim 픽셀)은 두껍고 어둡게"""
    def p(f, u, v, z, s, h):
        if f == "+z":
            return np.full(u.shape, "hR")
        if f in ("+u", "-u", "+v", "-v"):          # 지붕 판 옆 단면(박공 끝)
            return np.where(np.mod(h, 2) == 0, "hb", "hR")
        top = lit(f)
        a, b, c = ("hA", "hB", "hb") if top else ("ha", "hb", "hR")
        col = np.where(np.mod(h, 3) == 0, b, a)
        col = np.where((s * 7 + h * 5) % 13 == 0, c, col)
        col = np.where((s * 3 + h * 11) % 17 == 0, "hA" if top else "hB", col)
        return np.where(z < ze + rim, np.where(np.mod(s, 2) == 0, "hR", "hD"), col)
    return p


def tiles(ze):
    """기와: 네 점마다 골, 처마 끝에 둥근 막새 줄"""
    def p(f, u, v, z, s, h):
        if f == "+z":
            return np.full(u.shape, "gK")
        if f in ("+u", "-u", "+v", "-v"):
            return np.full(u.shape, "gK")
        a, b, e = ("gA", "gB", "gE") if lit(f) else ("ga", "gb", "gB")
        col = np.where(np.mod(s, 4) == 3, b, a)
        col = np.where((np.mod(s, 4) == 1) & (np.mod(h, 4) == 0), b, col)
        return np.where(z < ze + 2, np.where(np.mod(s, 4) == 3, "gK", e), col)
    return p


def ridge(kind):
    return flat("hR", "hD") if kind == "thatch" else flat("gK", "gK", "gB")


def stone(f, u, v, z, s, h):
    if f == "+z":
        i, j = np.floor(16 * u).astype(int), np.floor(16 * v).astype(int)
        return np.where((np.mod(i, 8) == 0) | (np.mod(j, 8) == 0), "ss", "sU")
    return brick(s, h, "sS", "ss", w=8, rows=3) if lit(f) else brick(s, h, "sR", "sr", w=8, rows=3)


def bricks(f, u, v, z, s, h):
    if f == "+z":
        return np.full(u.shape, "bU")
    return brick(s, h, "bS", "bs", w=6, rows=2) if lit(f) else brick(s, h, "bR", "br", w=6, rows=2)


def wood(f, u, v, z, s, h):
    if f == "+z":
        return np.full(u.shape, "tE")
    a, b = ("tW", "tw") if lit(f) else ("tw", "tk")
    return np.where((s * 5 + h * 3) % 11 == 0, b, a)


def jar(f, u, v, z, s, h):
    if f == "+z":
        return np.full(u.shape, "ja")
    a = "jA" if lit(f) else "ja"
    return np.where(h >= z.max() - 1 if z.size else False, "jT", np.where(np.mod(h, 5) == 2, "ja", a))


# ---------------------------------------------------------------- 몸체

def gable_house(L, a0, a1, c0, c1, zw, s, wall_paint, roof="thatch", ov=.16, ova=.14, t=5, z0=0):
    """벽(+박공 삼각) 하나와 지붕 판 두 장(앞·뒤 경사), 용마루. 처마는 벽보다 ov 만큼 나온다."""
    cm = (c0 + c1) / 2
    out = [Body(L.box(a0, a1, c0, c1, z0, 1e9) + [L.plane(0, s, 1, zw + s * c1, "r+c"), L.plane(0, -s, 1, zw - s * c0, "r-c")],
                wall_paint)]
    paint = thatch(zw - s * ov) if roof == "thatch" else tiles(zw - s * ov)
    A0, A1 = a0 - ova, a1 + ova
    out.append(Body(L.box(A0, A1, cm, c1 + ov, 0, 1e9) + [L.plane(0, s, 1, zw + s * c1 + t, "r+c"),
                                                         L.plane(0, -s, -1, -(zw + s * c1), "b+c")], paint))
    out.append(Body(L.box(A0, A1, c0 - ov, cm, 0, 1e9) + [L.plane(0, -s, 1, zw - s * c0 + t, "r-c"),
                                                         L.plane(0, s, -1, -(zw - s * c0), "b-c")], paint))
    zr = zw + s * (c1 - cm) + t
    rw = .07 if roof == "thatch" else .06
    out.append(Body(L.box(A0 - .02, A1 + .02, cm - rw, cm + rw, zr - 2, zr + 2), ridge(roof)))
    if roof != "thatch":
        for a in (A0 - .02, A1 - .1):
            out.append(Body(L.box(a, a + .12, cm - .08, cm + .08, zr - 2, zr + 3), ridge(roof)))
    return out


def thatch_hip(L, a0, a1, c0, c1, ze, s, top):
    """두툼한 초가 우진각: 처마 쪽 짚 단면 띠(5px)를 두르고 그 위에 사방 경사, 꼭대기는 평평하게 잘라 짚 마루"""
    zt = ze + s * top
    out = [Body(L.box(a0, a1, c0, c1, ze, ze + 5), thatch_band),
           Body(L.hip(a0 + .02, a1 - .02, c0 + .02, c1 - .02, ze + 5, s, top=zt + 5), thatch(ze + 3, rim=0))]
    cm = (c0 + c1) / 2
    ra0, ra1 = a0 + top + .02, a1 - top - .02
    out.append(Body(L.box(ra0, ra1, cm - .08, cm + .08, zt + 4, zt + 8), flat("hB", "hb", "hb")))
    return out


def thatch_band(f, u, v, z, s, h):
    if f == "+z":
        return np.full(u.shape, "hB")
    return np.where(np.mod(s, 2) == 0, "hb" if lit(f) else "hR", "hR" if lit(f) else "hD")


def post(u, v, z0, z1, r=.05, paint=wood):
    return Body(box(u - r, u + r, v - r, v + r, z0, z1), paint)


def jar_body(u, v, r=.1, hgt=9):
    return [Body(box(u - r, u + r, v - r, v + r, 0, hgt - 2), jar),
            Body(box(u - r * .6, u + r * .6, v - r * .6, v + r * .6, hgt - 2, hgt), flat("jT", "ja", "ja"))]


# ---------------------------------------------------------------- 건물들 (u, v 소타일; 발자리 원점 = 뒤 모서리)

def cottage():
    """초가집 2×2: 흙벽, 두툼한 초가 맞배, 앞 흙벽에 문과 창, 박공에 작은 창, 오른쪽에 장작더미·항아리"""
    L = pc.Line("u")
    holes = [("+v", .58, .86, 3, 25, "door"), ("+v", 1.1, 1.42, 13, 21, "lattice"), ("+u", .8, 1.15, 13, 21, "lattice")]
    out = [Body(box(.3, 1.6, .5, 1.45, 0, 30), earth(holes))] + thatch_hip(L, .14, 1.76, .34, 1.61, 26, 34, .2)
    out.append(Body(box(1.66, 1.92, .55, .95, 0, 9), logs))
    out.append(Body(box(1.66, 1.92, .55, .95, 9, 10), flat("tE", "tw")))
    out += jar_body(1.78, 1.25, .09, 8)
    out.append(Body(box(.35, .5, .55, .68, 0, 56), bricks))     # 뒤 흙 굴뚝
    return out, (2, 2)


def logs(f, u, v, z, s, h):
    if f == "+z":
        return np.full(u.shape, "tw")
    if f == "+u":
        return np.where((np.mod(s, 3) == 0) | (h % 3 == 0), "tk", "tE")
    return np.where(h % 3 == 0, "tk", "tw")


def smithy():
    """대장간 3×2: 뒤·왼쪽만 흙벽, 앞이 트인 초가 헛간, 안쪽에 벽돌 화덕(불빛), 오른쪽 끝에 높은 굴뚝, 앞에 모루와 물통"""
    L = pc.Line("u")
    out = []
    a0, a1, c0, c1, zw, s = .25, 2.35, .35, 1.45, 32, 20
    out += thatch_hip(L, a0 - .16, a1 + .16, c0 - .16, c1 + .16, zw - 4, 30, .35)
    out.append(Body(box(a0, a1, c0, c0 + .1, 0, zw), earth()))             # 뒤 흙벽
    out.append(Body(box(a0, a0 + .1, c0, c1, 0, zw), earth()))             # 왼쪽 흙벽
    for u in (a0 + .05, 1.3, a1 - .05):
        out.append(post(u, c1 - .05, 0, zw, .05))
    out.append(Body(box(a0, a1, c1 - .1, c1, zw - 3, zw), wood))          # 앞 도리
    out.append(Body(box(.4, 1.1, .5, 1.0, 0, 16), forge_paint))            # 화덕
    out.append(Body(box(.55, .95, .55, .85, 16, 40), bricks))              # 화덕 위 연도
    out.append(Body(box(2.45, 2.75, .45, .75, 0, 64), bricks))                # 굴뚝
    out.append(Body(box(2.42, 2.78, .42, .78, 64, 67), flat("bU", "br", "bU")))
    out.append(Body(box(1.45, 1.62, 1.72, 1.86, 0, 6), wood))              # 모루 받침
    out.append(Body(box(1.38, 1.7, 1.74, 1.84, 6, 9), flat("iA", "ia", "iU")))
    out.append(Body(box(1.95, 2.45, 1.62, 1.86, 0, 6), trough))            # 물통
    out.append(Body(box(2.0, 2.4, 1.66, 1.82, 6, 6.5), flat("ia", "ia", "iA")))
    return out, (3, 2)


def forge_paint(f, u, v, z, s, h):
    if f == "+z":
        i, j = np.floor(16 * u).astype(int), np.floor(16 * v).astype(int)
        return np.where((u > .6) & (u < .9) & (v > .62) & (v < .88), np.where((i + j) % 2 == 0, "fA", "fB"), "bU")
    col = bricks(f, u, v, z, s, h)
    if f == "+v":     # 화구
        m = (u > .6) & (u < .9) & (h >= 3) & (h < 10)
        glow = np.where(h < 5, "fA", np.where(h < 8, "fB", "fC"))
        col = np.where(m, glow, col)
    return col


def trough(f, u, v, z, s, h):
    if f == "+z":
        return np.full(u.shape, "tw")
    return np.where(h % 3 == 0, "tk", "tW" if lit(f) else "tw")


def yamen():
    """관청 4×3: 석축 기단과 가운데 계단, 회벽에 붉은 기둥 툇간, 붉은 창살문, 가운데 빈 편액, 우진각 기와 지붕, 양쪽 붉은 깃발"""
    L = pc.Line("u")
    out = [Body(box(.15, 3.85, .25, 2.4, 0, 9), stone)]
    for k in range(3):                                         # 계단
        out.append(Body(box(1.55 - k * .0, 2.45, 2.4, 2.4 + .17 * (3 - k), 0, 9 - 3 * (k + 1) + 3), stone))
    holes = [("+v", 1.0, 1.45, 3, 22, "redlattice"), ("+v", 1.65, 2.35, 3, 24, "redlattice"),
             ("+v", 2.55, 3.0, 3, 22, "redlattice"), ("+u", .8, 1.6, 10, 18, "redlattice")]
    hall = plaster(holes, beam=30)
    zb, zw = 9, 9 + 38
    out.append(Body(box(.6, 3.4, .75, 1.85, zb, zw), hall))
    for u in (.6, 1.25, 1.65, 2.35, 2.75, 3.4):                # 툇간 붉은 기둥
        out.append(post(u, 2.2, zb, zw, .055, flat("cR", "cr")))
    out.append(Body(box(.55, 3.45, 2.15, 2.25, zw - 4, zw), flat("cr", "cr", "cR")))   # 창방
    out.append(Body(box(1.78, 2.22, 2.25, 2.28, zw - 14, zw - 5), plaque))             # 편액 (글자 없이)
    # 지붕: 처마 쪽 완만한 자락 + 위 우진각, 용마루와 치미
    roof = tiles(zw - 2)
    out.append(Body(L.hip(.3, 3.7, .45, 2.45, zw - 2, 12, top=zw + 6), roof))
    out.append(Body(L.hip(.7, 3.3, .8, 2.1, zw + 4, 26), roof))
    zr = zw + 4 + 26 * .65
    out.append(Body(box(1.4, 2.6, 1.38, 1.52, zr - 3, zr + 2), flat("gK", "gK", "gB")))
    for a in (1.33, 2.55):
        out.append(Body(box(a, a + .12, 1.37, 1.53, zr - 3, zr + 9), flat("gK", "gK", "gB")))
    # 깃발
    for u in (.3, 3.7):
        out.append(post(u, 2.7, 0, 66, .03))
        out.append(Body(box(u + .03, u + .06, 2.45, 2.68, 38, 64), banner))
    return out, (4, 3)


def plaque(f, u, v, z, s, h):
    edge = (u < 1.81) | (u > 2.19) | (h <= z.min() + 1 if z.size else False) | (h >= z.max() - 1 if z.size else False)
    return np.where(edge, "qG", "qD")


def banner(f, u, v, z, s, h):
    return np.where((h < 40) | (h > 61), "ra", "rA" if f != "+u" else "ra")


def tavern():
    """주점 3×2: 흙벽 기와집, 앞 벽 왼쪽은 문, 가운데는 창살 들창, 처마 밑 붉은 등, 오른쪽에 술항아리와 빈 술깃발, 앞 평상"""
    L = pc.Line("u")
    holes = [("+v", .55, .85, 3, 26, "door"), ("+v", 1.1, 1.95, 13, 22, "lattice"), ("+u", .75, 1.15, 13, 21, "lattice")]
    out = gable_house(L, .3, 2.3, .45, 1.45, 31, 18, earth(holes), "tiles", ov=.14, t=4)
    out.append(Body(box(1.08, 1.98, 1.45, 1.55, 11, 13), wood))                   # 들창 받침
    out.append(Body(box(1.0, 1.9, 1.62, 1.95, 0, 7), wood))                      # 평상
    for u in (1.05, 1.85):
        out.append(post(u, 1.9, 0, 4, .04))
    for u, v in ((2.45, .7), (2.62, .95), (2.45, 1.2), (2.68, 1.35), (2.4, 1.65)):
        out += jar_body(u, v, .1, 10)
    out.append(post(2.75, 1.75, 0, 50, .03))                                     # 술깃발 장대
    out.append(Body(box(2.42, 2.73, 1.73, 1.77, 26, 48), flag_white))
    out.append(Body(box(.86, .96, 1.55, 1.65, 20, 26), flat("lR", "lr")))          # 붉은 등
    out.append(Body(box(.9, .92, 1.59, 1.61, 26, 30), flat("tk", "tk")))
    return out, (3, 2)


def flag_white(f, u, v, z, s, h):
    edge = (h < 28) | (h > 45) | (u < 2.46) | (u > 2.69)
    return np.where(edge, "rA", "wA" if lit(f) else "wa")


def shop():
    """도구점 3×2: 회벽 기와집, 앞에 천 차양과 진열대(연장·바구니), 오른쪽에 창 걸이"""
    L = pc.Line("u")
    holes = [("+v", .5, .78, 3, 26, "door"), ("+v", 1.0, 2.1, 6, 24, "dark"), ("+u", .65, 1.05, 13, 21, "lattice")]
    out = gable_house(L, .3, 2.3, .4, 1.3, 31, 18, plaster(holes, beam=28), "tiles", ov=.14, t=4)
    # 차양: 처마에서 앞으로 낮게 기울어지는 천
    zt, zf, c0, c1 = 26, 19, 1.3, 1.85
    sl = (zt - zf) / (c1 - c0)
    out.append(Body(L.box(.95, 2.15, c0, c1, 0, 1e9) + [L.plane(0, sl, 1, zt + sl * c0 + 1, "r+c"),
                                                       L.plane(0, -sl, -1, -(zt + sl * c0), "b+c")], awning))
    for u in (.98, 2.12):
        out.append(post(u, 1.82, 0, zf, .03))
    out.append(Body(box(1.05, 2.05, 1.45, 1.7, 0, 8), wood))                    # 진열대
    for u in (1.15, 1.35, 1.6, 1.85):
        out.append(Body(box(u - .06, u + .06, 1.5, 1.62, 8, 10), flat("iA", "ia", "iU") if u < 1.5 else flat("hA", "hb", "hB")))
    out.append(Body(box(.3, .85, 1.75, 1.85, 0, 3), wood))                       # 창 걸이 (왼쪽 앞)
    for u in (.33, .82):
        out.append(post(u, 1.8, 0, 22, .035))
    out.append(Body(box(.3, .85, 1.76, 1.84, 17, 19), wood))
    for u in np.linspace(.42, .74, 4):
        out.append(Body(box(u - .015, u + .015, 1.85, 1.88, 2, 26), flat("tw", "tk")))
        out.append(Body(box(u - .03, u + .03, 1.85, 1.88, 26, 31), flat("iU", "iA")))
    return out, (3, 2)


def awning(f, u, v, z, s, h):
    if not f.startswith("r"):
        return np.full(u.shape, "wa")
    stripe = np.mod(np.floor(16 * u), 8) < 4
    return np.where(stripe, "wA", "rA")


BUILDINGS = {"cottage": cottage, "smithy": smithy, "yamen": yamen, "tavern": tavern, "shop": shop}


def make(name):
    bodies, (nu, nv) = BUILDINGS[name]()
    x0, x1 = -16 * nv - 24, 16 * nu + 24
    y0, y1 = -120, 8 * (nu + nv) + 12
    a, own, o = pc.render(bodies, bounds=(x0, y0, x1, y1))
    im = Image.fromarray(a)
    bb = im.getbbox()
    im = im.crop(bb)
    cx, cy = 16 * (nu - nv) / 2, 8 * (nu + nv) / 2      # 발자리 중심
    return im, (int(cx - o[0] - bb[0]), int(cy - o[1] - bb[1]))


if __name__ == "__main__":
    out = sys.argv[1]
    os.makedirs(out, exist_ok=True)
    meta = {}
    for n in BUILDINGS:
        im, pv = make(n)
        im.save(os.path.join(out, n + ".png"))
        meta[n] = {"size": im.size, "pivot": pv, "footprint": BUILDINGS[n]()[1]}
        print(n, im.size, pv)
    json.dump(meta, open(os.path.join(out, "meta.json"), "w"), indent=1)
