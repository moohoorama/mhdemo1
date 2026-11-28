"""장면 무대: 스토리 장면이 이 맵에서 사람을 세우고 걷게 하는 칸, 그 사람을 가리는 물건, 장면이 기대던 옛 건물.

좌표: 장면은 게임 칸(u, v), 사람이 선 소타일은 floor(2u+1) (internal/gui smallTileAt), 걷기는 u 먼저 그다음 v
(internal/gui scenePath).
"""
import glob
import json
import math
import os

PERSON = (-10, -44, 10, 0)          # 사람 그림 상자 (발 기준 화면 픽셀)
OLD_HOUSE = ('house', 'hall', 'tavern', 'granary')


def tile(u, v):
    return math.floor(2 * u + 1), math.floor(2 * v + 1)


def scene_path(u, v, tu, tv):
    out = []
    while u != tu:
        u += math.copysign(min(1, abs(tu - u)), tu - u)
        out.append((u, v))
    while v != tv:
        v += math.copysign(min(1, abs(tv - v)), tv - v)
        out.append((u, v))
    return out


def path_tiles(p, path):
    out = set()
    for q in path:
        n = int(math.ceil(max(abs(q[0] - p[0]), abs(q[1] - p[1])) * 4)) or 1
        for i in range(1, n + 1):
            k = i / n
            out.add(tile(p[0] + (q[0] - p[0]) * k, p[1] + (q[1] - p[1]) * k))
        p = q
    return out


def scenes_on(scenes_dir, map_id):
    for f in sorted(glob.glob(os.path.join(scenes_dir, 'ch*.json'))):
        for sid, sc in (json.load(open(f)).get('Scenes') or {}).items():
            if sc.get('Map') == map_id:
                yield sid, sc


def scene_stage(sc):
    """(선 자리, 걷는 칸): 처음 자리·걸음이 멈춘 자리·순찰 끝이 선 자리, 걸음 사이 칸이 걷는 칸"""
    pos, spots, walks = {}, set(), set()
    for key, c in (sc.get('Cast') or {}).items():
        if isinstance(c, dict) and not c.get('Off') and isinstance(c.get('At'), list) and len(c['At']) == 2:
            pos[key] = tuple(c['At'])
            spots.add(tile(*c['At']))

    def play(st):
        a = st.get('Actor')
        if not a:
            return
        if isinstance(st.get('Enter'), list) and len(st['Enter']) == 2:
            pos[a] = tuple(st['Enter'])
        p = pos.get(a)
        if p is None:
            return
        if isinstance(st.get('Patrol'), list) and len(st['Patrol']) == 2:
            q = tuple(st['Patrol'])
            walks.update(path_tiles(p, scene_path(*p, *q)) | path_tiles(q, scene_path(*q, *p)))
            spots.add(tile(*q))
        if isinstance(st.get('Move'), list) and len(st['Move']) == 2:
            path, u, v = [], p[0], p[1]
            for c in [*(st.get('Via') or []), st['Move']]:
                path += scene_path(u, v, c[0], c[1])
                u, v = c[0], c[1]
            walks.update(path_tiles(p, path))
            pos[a] = (u, v)
            if not st.get('Exit'):
                spots.add(tile(u, v))
        if st.get('Exit'):
            pos.pop(a, None)

    for st in sc.get('Ambient') or []:
        play(st)
    for node in sc.get('Nodes') or []:
        for line in (sc.get('Beats') or {}).get(node) or []:
            for group in line:
                for st in group:
                    play(st)
    return spots, walks - spots


def gather_stage(scenes_dir, map_id):
    spots, walks, by = set(), set(), {}
    for sid, sc in scenes_on(scenes_dir, map_id):
        s, w = scene_stage(sc)
        spots |= s
        walks |= w
        by[sid] = s
    return spots, walks, by


# ---------------------------------------------------------------- 화면에서 가리기

class Sprites:
    """index.json props·extra의 그림 크기·피벗"""

    def __init__(self, graphics_dir):
        idx = json.load(open(os.path.join(graphics_dir, 'index.json')))
        self.s = {}
        for k in ('props', 'extra'):
            self.s.update((idx.get(k) or {}).get('sprites') or {})

    def box(self, o):
        """물건의 화면 상자와 깊이(발 y)"""
        sp = self.s.get(o[0])
        if not sp or len(o) < 3 or isinstance(o[1], (list, dict)):
            return None
        dx, dy = (o[3], o[4]) if len(o) >= 5 and not isinstance(o[3], dict) else (0, 0)
        x = 16 * (o[1] - o[2]) + dx
        y = 8 * (o[1] + o[2]) + 8 + dy
        w, h = sp['rect'][2], sp['rect'][3]
        px, py = sp['pivot']
        if sp.get('layer') == 'ground':
            return None
        return (x - px, y - py, x - px + w, y - py + h), y


def person_box(t):
    x, y = 16 * (t[0] - t[1]), 8 * (t[0] + t[1]) + 8
    return (x + PERSON[0], y + PERSON[1], x + PERSON[2], y + PERSON[3]), y


def overlap(a, b, pad=0):
    return a[0] < b[2] + pad and b[0] < a[2] + pad and a[1] < b[3] + pad and b[1] < a[3] + pad


def occluders(objs, spots, sprites, skip=lambda o: False):
    """선 자리의 사람을 앞에서 가리는 물건들의 번호 (물건 발이 사람 발보다 화면 아래, 상자가 겹침)"""
    pb = [person_box(t) for t in spots]
    out = []
    for i, o in enumerate(objs):
        if skip(o):
            continue
        b = sprites.box(o)
        if not b:
            continue
        (box, y) = b
        for (p, py) in pb:
            if y > py and overlap(box, p):
                out.append(i)
                break
    return out


def anchors(old_objs, spots, sprites):
    """장면이 기대던 옛 건물: 선 자리 바로 뒤(화면 위)에서 3칸 안에 바닥이 있는 집"""
    out = []
    for o in old_objs:
        if not o[0].startswith(OLD_HOUSE):
            continue
        if any(abs(o[1] - t[0]) <= 4 and abs(o[2] - t[1]) <= 4 and o[1] + o[2] < t[0] + t[1] for t in spots):
            out.append(o)
    return out


def plazas(by, link=5, pad=1, most=16):
    """장면마다 선 자리를 가까운 것끼리 묶어 네모(u0, v0, w, h)로: 사람들이 모여 선 곳의 광장"""
    out = []
    for spots in by.values():
        groups = []
        for t in sorted(spots):
            near = [g for g in groups if any(max(abs(t[0] - a), abs(t[1] - b)) <= link for a, b in g)]
            merged = {t}.union(*near) if near else {t}
            groups = [g for g in groups if g not in near] + [merged]
        for g in groups:
            us, vs = [a for a, _ in g], [b for _, b in g]
            u0, v0, u1, v1 = min(us) - pad, min(vs) - pad, max(us) + pad, max(vs) + pad
            if u1 - u0 < most and v1 - v0 < most:
                out.append((u0, v0, u1 - u0 + 1, v1 - v0 + 1))
    return out
