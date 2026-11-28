"""지역별 줄 타일(담장·흙담·울타리)과 대문. 줄 타일은 엔진의 wall line 'piece' 방식: 소타일 하나에 그림 하나,
이어진 변 마스크(1 -u, 2 -v, 4 +u, 8 +v)로 <piece>_00..15. 이웃 칸의 몸체를 문맥으로 같이 그려 이음매 없이
붙이고, 자기 칸(0..1)에 맞은 픽셀만 남긴다. 기준점은 칸 중심.

  python3 town_lines.py OUTDIR [지역...]   -> OUTDIR/<지역>/<이름>.png 와 lines.json (크기·기준점·바닥)
"""
import json
import os
import sys

import numpy as np
from PIL import Image

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import town_dot2 as T2
from town_dot2 import Body, box, flat, post, wood, stone, U, Vv, ST, lit
import town_dot as T
import shade

C = .5


# ---------------------------------------------------------------- 줄의 몸체 (L 방향 a0..a1, 가운데 c)

def twall_line(L, a0, a1, end0=False, end1=False, ridge_to=None):
    """기와 담장: 돌 굽, 회벽(나무 기둥 띠), 기와 갓(낮은 경사)과 용마루"""
    t, over, top = .13, .16, 18
    w = t + over
    r0 = a0 + (w if end0 else 0)
    r1 = a1 - (w if end1 else 0)
    if ridge_to is not None:
        r0, r1 = (max(r0, ridge_to), r1) if a0 > ridge_to - 1 else (r0, min(r1, ridge_to))
    def wall(f, u, v, z, s, h):
        if f == "+z":
            return np.full(u.shape, "mS")
        col = np.where(lit(f), "mL", "mS")
        col = np.where(np.mod(s, 16) == 8, "mF" if lit(f) else "mf", col)
        return np.where(h >= top - 2, "mF" if lit(f) else "mf", col)
    return [Body(L.box(a0, a1, C - t - .03, C + t + .03, 0, 4), stone),
            Body(L.box(a0, a1, C - t, C + t, 4, top), wall),
            Body(L.hip(a0, a1, C - w, C + w, top - 1, 12, top=top + 4), T.tiles(top - 1)),
            Body(L.box(r0, r1, C - .06, C + .06, top + 2, top + 5), flat("gK", "gK", "gB"))]


def ewall_line(L, a0, a1, end0=False, end1=False, ridge_to=None):
    """흙담: 지역 서민 벽 재료, 짚(강남은 기와) 갓"""
    t, over, top = .15, .12, 15
    w = t + over
    r0 = a0 + (w if end0 else 0)
    r1 = a1 - (w if end1 else 0)
    if ridge_to is not None:
        r0, r1 = (max(r0, ridge_to), r1) if a0 > ridge_to - 1 else (r0, min(r1, ridge_to))
    out = [Body(L.box(a0, a1, C - t, C + t, 0, top), T2.folk())]
    if ST["folk_mat"] == "tiles":
        out += [Body(L.hip(a0, a1, C - w, C + w, top - 1, 12, top=top + 4), T.tiles(top - 1)),
                Body(L.box(r0, r1, C - .06, C + .06, top + 2, top + 5), flat("gK", "gK", "gB"))]
    else:
        out += [Body(L.box(a0, a1, C - w, C + w, top - 1, top + 2), T.thatch_band),
                Body(L.hip(a0, a1, C - w + .02, C + w - .02, top + 2, 16, top=top + 6), T.thatch(top, rim=0)),
                Body(L.box(r0, r1, C - .05, C + .05, top + 5, top + 7), flat("hB", "hb", "hb"))]
    return out


def fence_style():
    return {"huabei": "rail", "zhongyuan": "rail", "huainan": "reed", "jiangnan": "reed", "bashu": "bamboo"}[REGION[0]]


def fence_line(L, a0, a1, end0=False, end1=False, ridge_to=None):
    """울타리: 칸 가운데와 양 끝 칸 경계에 기둥, 사이에 가로대 둘(rail) / 갈대 엮음 판(reed) / 대나무 판(bamboo)"""
    style = fence_style()
    out = []
    lo, hi = a0, a1
    for k in range(int(np.floor(lo)), int(np.ceil(hi)) + 1):
        for a in (k, k + .5):
            if lo - 1e-6 <= a <= hi + 1e-6:
                out.append(Body(L.box(a - .045, a + .045, C - .045, C + .045, 0, 15), wood))
    if style == "rail":
        for z0 in (5, 10):
            out.append(Body(L.box(lo, hi, C - .025, C + .025, z0, z0 + 2), wood))
    elif style == "reed":
        def reed(f, u, v, z, s, h):
            if f == "+z":
                return np.full(u.shape, "hb")
            weave = (np.mod(s + (h // 3) * 2, 4) < 2)
            return np.where(np.mod(h, 3) == 0, "hR", np.where(weave, "hB" if lit(f) else "hb", "ha" if lit(f) else "hR"))
        out.append(Body(L.box(lo, hi, C - .02, C + .02, 1, 12), reed))
        out.append(Body(L.box(lo, hi, C - .03, C + .03, 7, 8.5), wood))
    else:
        def bamboo(f, u, v, z, s, h):
            if f == "+z":
                return np.full(u.shape, "bm")
            col = np.where(np.mod(s, 2) == 0, "bm" if lit(f) else "bn", "bM" if lit(f) else "bm")
            return np.where(np.mod(h + (s // 2) * 3, 7) == 0, "bn", col)
        out.append(Body(L.box(lo, hi, C - .025, C + .025, 0, 14), bamboo))
        for z0 in (4, 10):
            out.append(Body(L.box(lo, hi, C - .04, C + .04, z0, z0 + 1.5), flat("tw", "tk")))
    return out


LINES = {"twall": twall_line, "ewall": ewall_line, "fence": fence_line}


def mask_bodies(kind, mask):
    """소타일 [0,1]^2의 줄 몸체: 닫힌 변으로 이웃 칸까지 이어 그린다 (cwall과 같은 규칙)"""
    line = LINES[kind]
    arms = {"u": (mask & 1, mask & 4), "v": (mask & 2, mask & 8)}
    out = []
    for L, axis in ((U, "u"), (Vv, "v")):
        lo, hi = arms[axis]
        if not (lo or hi):
            continue
        other = arms["v" if axis == "u" else "u"]
        bend = other[0] or other[1]
        if kind == "fence":     # 울타리는 꺾여도 가운데 기둥에서 만나기만 하면 된다
            out += line(L, -3 if lo else C, 4 if hi else C)
        elif lo and hi:
            out += line(L, -3, 4)
        elif lo:
            out += line(L, -3, C + .3 if bend else 1, end1=not bend, ridge_to=C if bend else None)
        else:
            out += line(L, C - .3 if bend else 0, 4, end0=not bend, ridge_to=C if bend else None)
    if mask == 0:
        out = line(U, C - .2, C + .2, end0=True, end1=True) if kind != "fence" else line(U, C, C)
    return out


def own_tile(u, v):
    return (np.floor(u - 1e-6) == 0) & (np.floor(v - 1e-6) == 0)


def piece(bodies, own, center, bounds=(-60, -70, 60, 40)):
    a, o, owner = shade.render(bodies, bounds, own=own)
    a = a.copy()
    a[~owner] = 0
    im = Image.fromarray(a)
    bb = im.getbbox()
    if bb is None:
        return Image.new("RGBA", (1, 1)), [0, 0]
    im = im.crop(bb)
    return im, [int(center[0] - o[0] - bb[0]), int(center[1] - o[1] - bb[1])]


# ---------------------------------------------------------------- 대문 (2칸, 줄 방향 a 0..2, 가운데 c)

def tgate(L):
    """기와 대문: 붉은 기둥 넷, 붉은 문짝(금빛 장식), 맞배 기와 지붕, 양옆 담장 문맥"""
    out = twall_line(L, -3, 0, end1=True) + twall_line(L, 2, 5, end0=True)   # 맵에서 대문 옆에 오는 줄 끝 조각과 같은 모양
    for a in (.08, 1.92):
        for c in (C - .2, C + .2):
            out.append(Body(L.box(a - .06, a + .06, c - .06, c + .06, 0, 28), flat("mF", "mf")))
    def leaf(f, u, v, z, s, h):
        col = np.where(np.mod(s, 4) == 0, "mf", "mF" if lit(f) else "mf")
        return np.where((h % 8 == 4) & (np.mod(s, 4) == 2), "qG", col)
    out.append(Body(L.box(.14, 1.86, C - .03, C + .03, 0, 24), leaf))
    out.append(Body(L.box(0, 2, C - .24, C + .24, 24, 28), flat("mF", "mf", "mF")))
    out += T.gable_house(L, .06, 1.94, C - .3, C + .3, 28, 12, flat("mF", "mf"), "tiles", ov=.12, ova=.0, t=3, z0=26)   # 박공까지 막힘
    return out


def egate(L):
    """사립문: 나무 기둥 둘, 엮은 문짝, 작은 초가(강남은 기와) 지붕, 양옆 흙담 문맥"""
    out = ewall_line(L, -3, 0, end1=True) + ewall_line(L, 2, 5, end0=True)
    for a in (.1, 1.9):
        out.append(Body(L.box(a - .06, a + .06, C - .06, C + .06, 0, 24), wood))
    def leaf(f, u, v, z, s, h):
        return np.where(np.mod(s, 3) == 0, "tk", np.where(np.mod(h, 6) == 0, "tw", "tW" if lit(f) else "tw"))
    out.append(Body(L.box(.18, 1.82, C - .025, C + .025, 1, 16), leaf))
    mat = "tiles" if ST["folk_mat"] == "tiles" else "thatch"
    out += T.gable_house(L, .06, 1.94, C - .26, C + .26, 24, 14, flat("tW", "tw"), mat, ov=.12, ova=.0,
                         t=4 if mat == "thatch" else 3, z0=22)
    return out


GATES = {"tgate": tgate, "egate": egate}

REGION = [None]


def kit(region):
    REGION[0] = region
    T2.region(region)
    out = {}
    for kind in LINES:
        for m in range(16):
            out[f"{region}_{kind}_{m:02d}"] = piece(mask_bodies(kind, m), own_tile, (0, 8)) + ([1, 1],)
    for g, fn in GATES.items():
        for o in ("u", "v"):
            L = T2.pc.Line(o)
            own = (lambda u, v: (u >= 0) & (u < 2) & (v >= 0) & (v < 1)) if o == "u" else \
                  (lambda u, v: (v >= 0) & (v < 2) & (u >= 0) & (u < 1))
            cu, cv = (1, .5) if o == "u" else (.5, 1)
            im, pv = piece(fn(L), own, (16 * (cu - cv), 8 * (cu + cv)), bounds=(-90, -90, 90, 60))
            out[f"{region}_{g}_{o}"] = (im, pv, [2, 1] if o == "u" else [1, 2])
    return out


if __name__ == "__main__":
    root = sys.argv[1]
    regions = sys.argv[2:] or list(T2.REGIONS)
    for r in regions:
        d = os.path.join(root, r)
        os.makedirs(d, exist_ok=True)
        meta = {}
        for name, (im, pv, fp) in kit(r).items():
            im.save(os.path.join(d, name + ".png"))
            meta[name] = {"size": im.size, "pivot": pv, "footprint": fp}
        json.dump(meta, open(os.path.join(d, "lines.json"), "w"), indent=1)
        print(r, len(meta))
