#!/usr/bin/env python3
"""Builds the title screen's backdrop reel, assets/scenes/title/reel.json.

    python3 tools/title_reel/build.py                 # check every shot, write the reel when all pass
    python3 tools/title_reel/build.py --force         # write it even when a check fails
    python3 tools/title_reel/build.py --only grain    # check (and write) these shots alone, for a quick capture

The reel is a row of shots the title plays one after another under its menu (internal/gui/title_reel.go), each
seen at zoom 1 in a remembered circle: a scene stage (map, time of day, cast, ambient effects) and its beat groups,
in the step format of assets/scenes (docs/presentation-scene.md). Crowds, ranks and fights are laid out here.

Every shot is checked before it is written (Shot.check): people stand and walk only on open ground of the map
(cmd/_reelgrid, scenemap.Walkable: no walls, houses, trees or water), never two on a cell, strike only at arm's
length, leave by walking out of the circle rather than fading, and no edge of the map shows inside the circle.
Walks go round what blocks them (Shot.route).
"""
import heapq
import json
import math
import random
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
OUT = ROOT / 'assets/scenes/title/reel.json'

SCREEN_H = 850       # world px of the reference screen's height: a shot's radius is a share of it
CELL_W, CELL_H = 64, 32  # a scene map cell on the screen at zoom 1 (two small tiles each way)
STEP = .28           # seconds a cell at walking speed 1
IMPACT = .2          # seconds from a swing's start to the blow landing (half the attack frames)
RELEASE = .2         # seconds from an archer's draw to the arrow leaving
COMMONERS = ['commoner_man', 'commoner_woman', 'commoner_elder', 'commoner_oldwoman']
FOOT = ['light_infantry', 'pikeman']
STEPS = {'SE': (1, 0), 'SW': (0, 1), 'NW': (-1, 0), 'NE': (0, -1)}
CHEERS = ['!', '!!', '!!!']

_grids = {}


def grid(map_id):
    if map_id not in _grids:
        out = subprocess.run(['go', 'run', './cmd/_reelgrid', map_id], cwd=ROOT, capture_output=True, text=True, check=True)
        _grids[map_id] = json.loads(out.stdout)[map_id]
    return _grids[map_id]


def facing(du, dv):
    """The diagonal a person faces to look along (du, dv)."""
    if abs(du) >= abs(dv):
        return 'SE' if du > 0 else 'NW'
    return 'SW' if dv > 0 else 'NE'


def screen(du, dv):
    return (du - dv) * CELL_W / 2, (du + dv) * CELL_H / 2


def r2(x):
    return round(x, 2)


def walk_time(cells, speed=1.0):
    return cells * STEP / speed


def arrow_time(src, dst):
    """Seconds from an archer's draw to the arrow coming down on dst (scene_arrow.go)."""
    return RELEASE + max(.55, math.hypot(dst[0] - src[0], dst[1] - src[1]) * .07)


class Shot:
    def __init__(self, sid, map_id, cam, radius=.42, time='', hold=.8, **extra):
        self.id, self.map, self.cam, self.radius = sid, map_id, list(cam), radius
        self.time, self.hold, self.extra = time, hold, extra
        self.cast, self.ambient, self.groups = {}, [], []
        self.g = grid(map_id)
        self.cams = [tuple(cam)]
        self.pos, self.away, self.down = {}, set(), set()  # where each stands as the groups are written
        self.blocks = set()
        self.sides = {}  # which side each is on: people of two sides never pass through each other
        self.interior = extra.pop('interior', False)  # a room: its walls are its edge, the circle may show past them

    # --- the ground -------------------------------------------------------------------------------------------

    def cell(self, u, v):
        g = self.g
        y, x = round(v) - g['V0'], round(u) - g['U0']
        if 0 <= y < len(g['Rows']) and 0 <= x < len(g['Rows'][0]):
            return g['Rows'][y][x]
        return ' '

    def open(self, u, v):
        return self.cell(u, v) in '.:' and (round(u), round(v)) not in self.blocks

    def block(self, *cells):
        """Cells a beat's prop (a tree, a cart) stands on: nobody walks there."""
        self.blocks.update((round(u), round(v)) for u, v in cells)

    def outside(self, u, v, cam=None, margin=30):
        """Whether cell (u, v) is past the circle's rim, out of sight."""
        cu, cv = cam or self.cam
        return math.hypot(*screen(u - cu, v - cv)) > self.radius * SCREEN_H + margin

    def taken(self, but=None):
        return {at for k, at in self.pos.items() if k != but and k not in self.away}

    def free(self, near, n=1, avoid=()):
        """n open cells nobody stands on, nearest to near first."""
        out, r, taken = [], 0, self.taken() | set(avoid)
        level = self.cell(*near) if self.open(*near) else '.'  # on the same ground: not up a cliff
        while len(out) < n and r < 60:
            for du in range(-r, r + 1):
                for dv in range(-r, r + 1):
                    c = (near[0] + du, near[1] + dv)
                    if max(abs(du), abs(dv)) == r and self.open(*c) and self.cell(*c) == level and c not in taken and c not in out:
                        out.append(c)
            r += 1
        return out[:n]

    def away_cell(self, frm, toward):
        """The first open cell out of the circle from frm along the direction toward (du, dv)."""
        u, v = frm
        for i in range(1, 120):
            c = (round(u + toward[0] * i), round(v + toward[1] * i))
            if self.outside(*c, margin=60) and self.open(*c):
                return c
        raise ValueError(f'{self.id}: no way out from {frm} toward {toward}')

    def route(self, src, dst, blocked=()):
        """Corners (Via) and the end of a walk from src to dst on open ground round blocked cells, few turns."""
        src, dst = tuple(map(round, src)), tuple(map(round, dst))
        best, n = {}, 0
        heap = [(0, 0, 0, src, None, (src,))]
        while heap:
            _, cost, _, at, d, path = heapq.heappop(heap)
            if at == dst:
                corners = [path[i] for i in range(1, len(path) - 1)
                           if (path[i][0] - path[i - 1][0], path[i][1] - path[i - 1][1]) !=
                           (path[i + 1][0] - path[i][0], path[i + 1][1] - path[i][1])]
                return [list(c) for c in corners], list(dst)
            if best.get((at, d), 1e9) <= cost:
                continue
            best[(at, d)] = cost
            for nd in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                nxt = (at[0] + nd[0], at[1] + nd[1])
                if not self.open(*nxt) or (nxt in blocked and nxt != dst) or self.open(*at) and self.cell(*nxt) != self.cell(*at):
                    continue
                c = cost + 1 + (4 if d and nd != d else 0)
                n += 1
                h = abs(dst[0] - nxt[0]) + abs(dst[1] - nxt[1])
                heapq.heappush(heap, (c + h, c, n, nxt, nd, path + (nxt,)))
        raise ValueError(f'{self.id}: no way from {src} to {dst}')

    # --- the cast and the beats ---------------------------------------------------------------------------------

    def actor(self, key, at, face, sprite='', faction='', side=None, **kw):
        a = dict(At=list(at), Face=face, **kw)
        self.sides[key] = side or faction or 'named'
        if sprite:
            a['Sprite'] = sprite
        if faction:
            a['Faction'] = faction
        self.cast[key] = a
        self.pos[key] = tuple(at)
        if kw.get('Off'):
            self.away.add(key)
        return key

    def group(self, *steps):
        flat = []
        for s in steps:
            flat.extend(s if isinstance(s, list) else [s])
        # props stay where they are put: one carried along by a hand that holds nothing slides through the air
        # (a boat carrying its man over the water, _ride, is the one exception)
        self.groups.append([st for st in flat if st.get('_ride') or not (isinstance(st.get('Prop'), list) and len(st['Prop']) == 2)])

    def move(self, key, dst, end=None, leave=False, straight=False, avoid=(), past_own=False, **kw):
        """A step walking key to dst round whatever blocks the way (people standing included)."""
        src = self.pos[key]
        st = dict(Actor=key, **kw)
        if key in self.away and 'Enter' not in kw:
            raise ValueError(f'{self.id}: {key} is off the stage')
        if 'Enter' in kw:
            src = tuple(kw['Enter'])
            self.away.discard(key)
        dst = tuple(dst)
        if not self.open(*dst) or dst in self.taken(key):  # the nearest free cell to it instead
            dst = self.free(dst)[0]
        if straight:
            st['Move'], st['Ease'] = list(dst), st.get('Ease', 'line')
        else:
            standing = {at for k, at in self.pos.items() if k != key and k not in self.away and k not in self.down
                        and not (past_own and self.sides[k] == self.sides[key])}  # a crowd running together slips past its own
            via, end_at = self.route(src, dst, standing | set(avoid))  # over the fallen, round the standing
            st['Move'] = end_at
            if via:
                st['Via'] = via
        if end:
            st['End'] = end
        if leave:  # walking out of sight holds nothing up
            st['_leave'], st['Free'] = True, True
        self.pos[key] = tuple(dst)
        return st

    def fell(self, key):
        self.down.add(key)

    # --- the check ------------------------------------------------------------------------------------------------

    def check(self):
        problems = []
        r = self.radius * SCREEN_H
        for cam in ([] if self.interior else self.cams):
            reach = int(r / 8) + 2
            off = [(cam[0] + du, cam[1] + dv) for du in range(-reach, reach + 1) for dv in range(-reach, reach + 1)
                   if math.hypot(*screen(du, dv)) <= r and self.cell(cam[0] + du, cam[1] + dv) == ' ']
            if off:
                problems.append(f'map edge inside the circle round {cam}: {len(off)} cells, e.g. {off[:3]}')
        pos = {k: tuple(a['At']) for k, a in self.cast.items()}
        gone = {k for k, a in self.cast.items() if a.get('Off')}
        down = set()
        seen = {}
        for k, at in pos.items():
            if k in gone:
                continue
            if not self.open(*at) and self.cast[k].get('Pose') not in ('sit', 'throne') and not k.startswith('cam#'):
                problems.append(f'{k} stands on {self.cell(*at)!r} at {at}')
            if at in seen:
                problems.append(f'{k} and {seen[at]} share {at}')
            seen[at] = k
        named = [k for k, a in self.cast.items() if a.get('Off') and '#' not in k]  # a named one waiting off stage must come on
        came = {st.get('Actor') for g in self.groups for st in g if 'Enter' in st or 'Place' in st} | \
               {st.get('Drop') for g in self.groups for st in g}
        problems += [f'{k} never comes on' for k in named if k not in came]
        for gi, group in enumerate(self.groups):
            problems += self.crossings(gi, group, pos, gone, down)
            # anyone who walks or is driven back in the group is checked by time in crossings, not here
            moved = {st.get('Actor') for st in group if 'Move' in st} | {st.get('Strike') for st in group if st.get('Push')}
            for st in group:
                k = st.get('Actor')
                if 'Enter' in st:
                    pos[k] = tuple(st['Enter'])
                    gone.discard(k)
                if 'Place' in st:  # stands there at once
                    pos[k] = tuple(st['Place'])
                    gone.discard(k)
                if 'Move' in st:
                    at = pos[k]
                    for c in st.get('Via', []) + [st['Move']]:
                        prev = at
                        for cell in walk_cells(at, c, straight='Ease' in st):
                            if k.startswith('cam#'):
                                break
                            if st.get('_boat') and self.cell(*cell) in '~.':  # rowed in on a boat, then ashore
                                prev = cell
                                continue
                            if not self.open(*cell) or self.open(*prev) and self.cell(*cell) != self.cell(*prev):
                                problems.append(f'group {gi}: {k} walks over {self.cell(*cell)!r} at {cell}')
                                break
                            prev = cell
                        at = tuple(c)
                    pos[k] = at
                    moved.add(k)
                    if st.get('_leave') and not self.outside(*at):
                        problems.append(f'group {gi}: {k} stops at {at}, inside the circle')
                if st.get('Push') and st.get('Strike') in pos:  # driven back along the blow, to a whole cell
                    (bu, bv), (tu, tv) = pos[k], pos[st['Strike']]
                    d = math.hypot(tu - bu, tv - bv) or 1
                    pos[st['Strike']] = (round(tu + (tu - bu) / d * st['Push']), round(tv + (tv - bv) / d * st['Push']))
                if st.get('Through') and st.get('Strike') in pos:  # cut down in passing: the rider comes out beyond
                    (bu, bv), (tu, tv) = pos[k], pos[st['Strike']]
                    d = math.hypot(tu - bu, tv - bv) or 1
                    pos[k] = (round(tu + (tu - bu) / d * st['Through']), round(tv + (tv - bv) / d * st['Through']))
                if st.get('Drop'):
                    gone.discard(st['Drop'])
                    down.add(st['Drop'])
                    pos[st['Drop']] = pos[k]
                if st.get('Fall') or st.get('Dead'):
                    down.add(k)
                if st.get('Kill'):
                    down.add(st['Strike'])
                for field in ('Strike', 'Brawl'):
                    o = st.get(field)
                    if o and o != '-' and k not in moved and o not in moved:
                        d = max(abs(pos[k][0] - pos[o][0]), abs(pos[k][1] - pos[o][1]))
                        if d != 1:
                            problems.append(f'group {gi}: {k} {field.lower()}s {o} from {d} cells')
            standing = {}
            for k, at in pos.items():
                if k in gone or k in down or self.outside(*at):
                    continue
                if at in standing:
                    problems.append(f'after group {gi}: {k} and {standing[at]} share {at}')
                standing[at] = k
        return problems

    def crossings(self, gi, group, pos, gone, down, step=.1, near=.45):
        """People of different sides who come within near of a cell of each other while one of them walks."""
        tracks = {}  # key -> [(t, u, v) ...]: every walk and every push of the group, in time
        cur = {}
        entering = {st['Actor'] for st in group if 'Enter' in st}

        def where(k, t):
            pts = tracks.get(k)
            if not pts:
                return cur.get(k, pos[k]) if k in cur else pos[k]
            if t <= pts[0][0]:
                return pts[0][1:] if k in entering else pos[k]
            for (ta, ua, va), (tb, ub, vb) in zip(pts, pts[1:]):
                if t <= tb:
                    f = (t - ta) / max(1e-6, tb - ta)
                    return ua + (ub - ua) * f, va + (vb - va) * f
            return pts[-1][1:]

        def when(x):
            return x.get('After', 0) + (x.get('Gather', 0) + IMPACT if 'Move' not in x else 0)

        events = [x for x in group if ('Move' in x and not x.get('Actor', '').startswith('cam#')) or (x.get('Push') and x.get('Strike'))]
        for st in sorted(events, key=when):
            if 'Move' in st:
                k = st['Actor']
                t0, speed = st.get('After', 0), st.get('Speed', 1) or 1
                at = tuple(st['Enter']) if 'Enter' in st else tuple(map(float, where(k, t0)))
                pts = tracks.setdefault(k, [])
                pts[:] = [p for p in pts if p[0] <= t0]  # a new walk takes over from wherever he has got to
                pts.append((t0, *at))
                if 'Ease' in st:
                    d = math.hypot(st['Move'][0] - at[0], st['Move'][1] - at[1])
                    pts.append((t0 + d * STEP / speed, *st['Move']))
                else:
                    t = t0
                    for c in st.get('Via', []) + [st['Move']]:
                        for cell in walk_cells((round(at[0]), round(at[1])), c):
                            t += STEP / speed
                            pts.append((t, *cell))
                        at = tuple(c)
                cur[st['Actor']] = tuple(st['Move'])
            else:  # a blow that drives its man back: he slides along it as it lands
                t = when(st)
                (bu, bv), (tu, tv) = where(st['Actor'], t), where(st['Strike'], t)
                d = math.hypot(tu - bu, tv - bv) or 1
                to = (round(tu + (tu - bu) / d * st['Push']), round(tv + (tv - bv) / d * st['Push']))
                pts = tracks.setdefault(st['Strike'], [(0, *pos[st['Strike']])])
                pts[:] = [p for p in pts if p[0] <= t]
                pts += [(t, tu, tv), (t + .2 + .12 * st['Push'], *to)]
                cur[st['Strike']] = to
        out = []
        for st in group:  # a blow lands where the striker and his man both are at that moment
            if st.get('Strike') and (st['Actor'] in tracks or st['Strike'] in tracks):
                t = st.get('After', 0) + IMPACT
                (au, av), (bu, bv) = where(st['Actor'], t), where(st['Strike'], t)
                if max(abs(au - bu), abs(av - bv)) > 1.5:
                    out.append(f'group {gi}: {st["Actor"]}\'s blow lands {max(abs(au - bu), abs(av - bv)):.1f} cells from {st["Strike"]} at {t:.1f}s')
        falling = {st['Actor'] for st in group if st.get('Fall')} | {st['Strike'] for st in group if st.get('Kill')}
        live = [k for k in pos if k not in down and k not in falling and (k not in gone or k in entering) and not k.startswith('cam#')]
        if not tracks:
            return out
        end = max(p[-1][0] for p in tracks.values())
        seen = set()
        t = 0.0
        while t <= end + step:
            cells = {}
            for k in live:
                if k in entering and k in tracks and t < tracks[k][0][0]:
                    continue
                u, v = where(k, t)
                if all(self.outside(u, v, cam=c, margin=0) for c in self.cams):  # out of sight: nobody sees them brush past
                    continue
                cells.setdefault((round(u), round(v)), []).append((k, u, v))
            for (cu, cv), here in cells.items():
                others = [x for du in (-1, 0, 1) for dv in (-1, 0, 1) for x in cells.get((cu + du, cv + dv), [])]
                for k, u, v in here:
                    if k not in tracks:
                        continue
                    for o, ou, ov in others:
                        if o != k and self.sides[o] != self.sides[k] and math.hypot(u - ou, v - ov) < near and (k, o) not in seen:
                            seen.add((k, o))
                            out.append(f'group {gi}: {k} passes through {o} near ({u:.1f}, {v:.1f}) at {t:.1f}s')
            t += step
        return out

    def json(self):
        def clean(x):
            if isinstance(x, dict):
                return {k: clean(v) for k, v in x.items() if not k.startswith('_') and v not in ('', None, [], {})}
            if isinstance(x, list):
                return [clean(v) for v in x]
            if isinstance(x, float):
                return round(x, 2)
            return x
        out = dict(ID=self.id, Map=self.map, Time=self.time, Radius=self.radius, Cast=self.cast,
                   Ambient=[dict(CamAt=self.cam)] + self.ambient, Groups=self.groups, Hold=self.hold, **self.extra)
        return clean(out)


def walk_cells(src, dst, straight=False):
    fu, fv = src
    tu, tv = dst
    if straight:
        n = max(1, int(math.ceil(math.hypot(tu - fu, tv - fv) * 2)))
        return [(round(fu + (tu - fu) * i / n), round(fv + (tv - fv) * i / n)) for i in range(1, n + 1)]
    cells, u, v = [], fu, fv
    while round(u) != round(tu):
        u += 1 if tu > u else -1
        cells.append((round(u), round(v)))
    while round(v) != round(tv):
        v += 1 if tv > v else -1
        cells.append((round(u), round(v)))
    return cells


def brawl(a, b, rnd, start=0.0):
    """A pair that keeps trading blows: each swings at the other now and then."""
    return [dict(Actor=a, Brawl=b, After=r2(start + rnd.uniform(0, .9))),
            dict(Actor=b, Brawl=a, After=r2(start + rnd.uniform(0, .9)))]


def cheer(key, rnd, after=0.0, emote=.4):
    """A soldier carried away: a jump with the weapon up, a jump alone, or the weapon raised, now and then a shout."""
    t = r2(after + rnd.uniform(0, 1.2))
    kind = rnd.random()
    if kind < .45:
        steps = [dict(Actor=key, Play='attack', Hop=2, After=t)]
    elif kind < .7:
        steps = [dict(Actor=key, Hop=3, After=t)]
    elif kind < .85:
        steps = [dict(Actor=key, Play='block', Hop=1, After=t)]
    else:
        steps = [dict(Actor=key, Play='attack', After=t), dict(Actor=key, Play='attack', After=r2(t + .45))]
    if rnd.random() < emote:
        steps.append(dict(Actor=key, Emote=rnd.choice(CHEERS), After=r2(t + rnd.uniform(0, .3))))
    return steps


# --- 1. 황건적이 백성에게 곡식을 나눠 준다 ----------------------------------------------------------------------

def grain():
    s = Shot('grain', 'S_title_road_town', (5, 9), radius=.38, time='dusk', hold=.4)
    givers = {9: s.actor('황건#1', (3, 9), 'SE', 'bandit', 'turban'), 10: s.actor('황건#2', (3, 10), 'SE', 'bandit', 'turban')}
    s.actor('황건#3', (1, 8), 'SE', 'ruffian', 'turban')
    s.actor('황건#4', (2, 12), 'NE', 'bandit', 'turban')
    for i in range(6):
        s.ambient.append(dict(Prop=[f'grain_sack#p{i}', 1.6 + i % 2 * .5, 9.2 + i // 2 * .45]))
    rows = {9: [], 10: []}
    for i in range(6):
        rows[9].append(s.actor(f'백성#{i}', (5 + i, 9), 'NW', COMMONERS[i % 4], 'merchant'))
    for i in range(5):
        rows[10].append(s.actor(f'백성#{6 + i}', (6 + i, 10), 'NW', COMMONERS[(i + 2) % 4], 'merchant'))
    s.ambient += [dict(Actor=k, Chatter='…||…|', Lag=r2(1 + .37 * i)) for i, k in enumerate(rows[9][2:] + rows[10][1:])]
    s.group(dict(Wait=.6), s.move(rows[10][0], (5, 10), end='NW'))
    # turns alternate between the two rows: one giver hands over while the other fetches the next sack
    order = [9, 10, 9, 10, 9, 10]
    exits = [(-8, 30), (-6, 31), (-10, 29), (-4, 30), (-9, 31), (-7, 29)]
    for turn, row in enumerate(order):
        giver = givers[row]
        taker = rows[row].pop(0)
        s.group(s.move(giver, (4, row), end='SE'))  # the sacks stay piled where they are: nothing pops up in a hand
        s.group(dict(Actor=taker, Emote='…', After=.2), dict(Actor=giver, Emote='…', After=.5),
                dict(Wait=.5))
        steps = [s.move(taker, exits[turn], leave=True, Speed=.9)]
        s.away.add(taker)
        steps.append(s.move(giver, (3, row), end='SE', After=.3))
        for i, k in enumerate(rows[row]):
            steps.append(s.move(k, (5 + i, row), end='NW', After=r2(.5 + .15 * i)))
        s.group(*steps)
    s.group(dict(Wait=.8))
    return s


# --- 2. 장각이 외치고 황건병이 환호한다 -----------------------------------------------------------------------

def rally():
    """장각 before his camp cries the 황건 watchword to the host: 창천이사, then 황천당립; his brothers take it up, and
    the host after them, the front ranks first and the cry running back through the crowd. The words are shouts
    (Step.Shout), not balloons."""
    rnd = random.Random(2)
    s = Shot('rally', 'S_title_turban_camp', (4, 10), radius=.46, time='dusk', hold=.4)
    s.actor('장각', (0, 10), 'SE', Hold=True)
    s.actor('장량', (0, 8), 'SE', Hold=True)  # his brothers at his sides, Taoists in yellow like him
    s.actor('장보', (0, 12), 'SE', Hold=True)
    crowd = []
    for u in range(3, 11):
        for v in range(3, 18):
            if (u + v) % 2 and u > 3 or not s.open(u, v):
                continue
            key = s.actor(f'황건#{len(crowd)}', (u, v), 'NW', rnd.choice(['bandit', 'bandit', 'ruffian']), 'turban', Hold=True)
            crowd.append(key)
    s.ambient += [dict(Actor=k, Chatter=rnd.choice(['…|…||…', '…?||…|', '…||…!|']), Lag=r2(rnd.uniform(0, 2.6))) for k in crowd]
    hush = [dict(Actor=k, Chatter='-') for k in crowd]
    criers = sorted(rnd.sample(crowd, 10), key=lambda k: s.pos[k][0])  # a few of them cry it out loud, front ranks first

    def roar(t0=0.0, cry=False):
        g = []
        for k in crowd:
            g += cheer(k, rnd, t0, emote=0)
        if cry:
            g += [dict(Actor=k, Shout='황천당립!', ShoutSize=15, After=r2(t0 + .15 * (s.pos[k][0] - 3) + rnd.uniform(0, .25)))
                  for k in criers]
        return g + [dict(Shake=.3, After=r2(t0 + .4)), dict(Wait=r2(t0 + 1.9))]

    s.group([dict(st, After=2.2) for st in hush], dict(Wait=2.5))  # the host falls silent before he cries out
    s.group(dict(Actor='장각', Play='action'), dict(Actor='장각', Shout='창천이사!', ShoutSize=34), dict(Wait=1.6))
    s.group(dict(Actor='장각', Play='action', Hop=1), dict(Actor='장각', Shout='황천당립!', ShoutSize=34), dict(Wait=1.3))
    s.group(dict(Actor='장량', Play='action', Hop=1), dict(Actor='장량', Shout='황천당립!', ShoutSize=24),
            dict(Actor='장보', Play='action', Hop=1, After=.2), dict(Actor='장보', Shout='황천당립!', ShoutSize=24, After=.2),
            roar(.8, cry=True))
    s.group(dict(Actor='장각', Play='action', Hop=1), dict(Actor='장각', Shout='황천당립!', ShoutSize=34), roar(.4, cry=True))
    s.group(roar(), dict(Actor='장량', Play='action', Hop=2), dict(Actor='장보', Play='action', Hop=2, After=.3))
    return s


# --- 3. 황건적과 관군이 맞붙는다 ------------------------------------------------------------------------------

def clash():
    rnd = random.Random(3)
    s = Shot('clash', 'S_title_field_road', (5, 7), radius=.44, hold=.4)
    front_t, front_h, back_t, back_h = [], [], [], []
    for v in range(2, 13):
        if not (s.open(4, v) and s.open(5, v)):
            continue
        front_t.append(s.actor(f'황건#f{v}', (4, v), 'SE', rnd.choice(['bandit', 'ruffian']), 'turban'))
        front_h.append(s.actor(f'관군#f{v}', (5, v), 'NW', rnd.choice(FOOT), 'han'))
        for u in (2, 1):
            if s.open(u, v):
                back_t.append(s.actor(f'황건#b{u}{v}', (u, v), 'SE', rnd.choice(['bandit', 'ruffian', 'light_infantry']), 'turban'))
        if v % 2 == 0 and s.open(7, v):
            back_h.append(s.actor(f'관군#r{v}', (7, v), 'NW', 'pikeman', 'han'))
    s.actor('관군#장수', (9, 7), 'NW', 'light_cavalry', 'han')
    s.group([x for a, b in zip(front_t, front_h) for x in brawl(a, b, rnd)],
            [st for k in back_t for st in cheer(k, rnd, emote=.25)],
            [dict(Actor=k, Play='attack', After=r2(rnd.uniform(0, 1.2))) for k in back_h], dict(Wait=1.6))
    for rnd_no in range(3):
        g = []
        victims = [k for k in rnd.sample(front_t, 3) if k not in s.down][:2]
        for i, k in enumerate(victims):  # a turban cut down; the man who did it turns on the next one along
            foe = front_h[front_t.index(k)]
            g.append(dict(Actor=foe, Strike=k, React='hit', Kill=True, After=r2(.4 + .6 * i)))
            s.fell(k)
        losers = [k for k in rnd.sample(front_h, 2) if k not in s.down][:1]
        for k in losers:  # and a soldier of the han falls to a turban
            foe = front_t[front_h.index(k)]
            if foe not in s.down:
                g.append(dict(Actor=foe, Strike=k, React='hit', Kill=True, After=1.2))
                s.fell(k)
        g += [st for k in back_t if k not in s.down and k not in front_t for st in cheer(k, rnd, emote=.2)]
        g.append(dict(Wait=1.4))
        s.group(g)
        g = []
        for k in victims:
            foe = front_h[front_t.index(k)]
            v = s.pos[k][1]
            near = [t for t in front_t if t not in s.down and abs(s.pos[t][1] - v) == 1]
            if near and foe not in s.down:
                g.append(dict(Actor=foe, Brawl=near[0], After=.3))
            spare = sorted((b for b in back_t if b not in s.down and s.pos[b][0] == 2 and abs(s.pos[b][1] - v) <= 1),
                           key=lambda b: abs(s.pos[b][1] - v))
            if spare and (3, v) not in s.taken():
                g.append(s.move(spare[0], (3, v), end='SE', Speed=1.5))
        for k in losers:
            v = s.pos[k][1]
            spare = [b for b in back_h if b not in s.down and abs(s.pos[b][1] - v) <= 2]
            if spare and (6, v) not in s.taken():
                g.append(s.move(spare[0], (6, v), end='NW', Speed=1.5))
                back_h.remove(spare[0])
        g += [dict(Actor=k, Play='attack', After=r2(rnd.uniform(0, 1))) for k in back_h if k not in s.down]
        g.append(dict(Wait=.8))
        s.group(g)
    s.group(s.move('관군#장수', (6, 8), Speed=1.6, end='NW'), dict(Wait=.4))
    near = [k for k in front_t if k not in s.down and max(abs(s.pos[k][0] - 6), abs(s.pos[k][1] - 8)) == 1]
    if near:
        s.group(dict(Actor='관군#장수', Strike=near[0], React='hit', Kill=True, Push=1, Crit=True), dict(Wait=1.2))
        s.fell(near[0])
    return s


# --- 4. 황건적이 마을을 약탈한다 ------------------------------------------------------------------------------

def raid():
    rnd = random.Random(4)
    s = Shot('raid', 'S_title_village_burnt', (5, 5), radius=.4, time='dusk', hold=.6)
    s.ambient += [dict(Effect='bigfire', At=[1, 1]), dict(Effect='bigfire', At=[8, 2])]
    folk = [(4, 4), (3, 3), (7, 4), (5, 7), (10, 6), (2, 9), (6, 9)]
    for i, at in enumerate(folk):
        s.actor(f'백성#{i}', at, rnd.choice(['SE', 'SW', 'NE']), COMMONERS[i % 4], 'merchant')
    woman = s.actor('아낙', (5, 3), 'NE', 'commoner_woman', 'merchant')
    raiders = []
    for i, at in enumerate([(5, -6), (3, -6), (9, -5), (1, -5), (11, -6)]):
        raiders.append(s.actor(f'황건#{i}', at, 'SW', ['bandit', 'ruffian'][i % 2], 'turban', Off=True))
    s.group(dict(Wait=.6), [dict(Actor=k, Emote='!', After=r2(.2 + .1 * i)) for i, k in enumerate(list(s.cast)[:len(folk)])])
    # the raiders run in from the north-east; the village scatters south-west, out of the circle
    steps = []
    for i, k in enumerate(raiders):
        at = s.pos[k]
        steps.append(s.move(k, (at[0], at[1] + 5), Enter=list(at), Speed=1.7, After=r2(.25 * i)))
    for i, at in enumerate(folk):
        k = f'백성#{i}'
        steps.append(s.move(k, s.away_cell(at, (rnd.uniform(-.5, .3), 1)), leave=True, Speed=r2(rnd.uniform(1.4, 2)),
                            After=r2(.3 + .12 * i)))
        s.away.add(k)
    # the woman is slow: a raider comes running down a clear street from out of sight and cuts her down from behind
    lane = next(u for u in sorted(range(-2, 13), key=lambda u: abs(u - 5))
                if all(s.open(u, v) and (u, v) not in s.taken(woman) for v in range(-18, 29)))
    s.pos[woman] = (lane, 3)
    s.cast[woman]['At'] = [lane, 3]
    s.cast['황건#0']['At'] = [lane, -17]
    s.pos['황건#0'] = (lane, -17)
    steps = [st for st in steps if st.get('Actor') != '황건#0']
    chased, t = chase(s, '황건#0', woman, (0, 1), 20, kspeed=1.9, vspeed=.55, vafter=.5)
    steps += chased
    s.group(steps, dict(Wait=r2(t + 1.2)))
    # one loots a sack from a doorway, the others smash and run on after the folk
    s.ambient.append(dict(Prop=['grain_sack#x', 9.4, 3.6]))
    s.group(s.move('황건#2', (9, 4), end='SW', Speed=1.4), s.move('황건#1', (4, 2), end='NW', Speed=1.4),
            s.move('황건#3', (2, 5), end='SW', Speed=1.4), s.move('황건#4', (11, 3), end='SW', Speed=1.4))
    s.group(dict(Prop=['grain_sack#x', '황건#2']), dict(Actor='황건#1', Play='attack'), dict(Actor='황건#1', Play='attack', After=.5),
            dict(Actor='황건#0', Emote='!', After=.3), dict(Actor='황건#0', Play='attack', Hop=1, After=.4))
    s.group(s.move('황건#2', s.away_cell((9, 4), (-.2, -1)), leave=True, Speed=1.2),
            s.move('황건#3', s.away_cell((2, 5), (0, 1)), leave=True, Speed=1.8, After=.2),
            s.move('황건#4', s.away_cell((11, 3), (0, 1)), leave=True, Speed=1.8, After=.4),
            dict(Actor='황건#1', Play='attack', After=.3), dict(Actor='황건#1', Hop=2, After=1), dict(Wait=1.6))
    return s


# --- 5. 도원결의 ----------------------------------------------------------------------------------------------

def oath():
    s = Shot('oath', 'S_title_peach_grove', (4, 4), radius=.34, hold=1.2)
    s.actor('관우', (3, 4), 'NE')
    s.actor('유비', (4, 4), 'NE')
    s.actor('장비', (6, 4), 'NE')
    s.ambient.append(dict(Effect='petals', At=[4.75, 4.25]))
    s.group(dict(Wait=1.4))
    s.group(dict(Actor='유비', Play='action'), dict(Actor='관우', Play='action', After=.15), dict(Actor='장비', Play='action', After=.3),
            dict(Wait=1.2))
    s.group(dict(Actor='관우', Face='SE'), dict(Actor='장비', Face='NW'), dict(Actor='유비', Face='SE', After=.3))
    s.group(dict(Actor='장비', Play='action'), dict(Actor='관우', Play='action', After=.2), dict(Actor='유비', Emote='…', After=.4),
            dict(Wait=1.2))
    s.group(dict(Actor='유비', Face='NE'), dict(Actor='관우', Face='NE'), dict(Actor='장비', Face='NE'))
    s.group(dict(Actor='유비', Play='action'), dict(Actor='관우', Play='action', After=.15), dict(Actor='장비', Play='action', After=.3),
            dict(Wait=1.6))
    return s


def chase(s, killer, victim, d, gap, kspeed=1.9, vspeed=.6, vafter=.3, kafter=0.0, enter=True):
    """A killer running a fleeing victim down from behind along a straight line d, cutting as he catches up.
    The victim runs on out of sight unless cut; the killer stops where the blow lands."""
    P = s.pos[victim]
    K = (P[0] - d[0] * gap, P[1] - d[1] * gap)
    xv = lambda t: max(0, t - vafter) / (STEP / vspeed)
    xk = lambda t: -gap + max(0, t - kafter) / (STEP / kspeed)
    # he stops on the first cell where, as he gets to it, the victim is just ahead of him, and cuts as he arrives
    kc = -gap
    while True:
        kc += 1
        t = kafter + (kc + gap) * STEP / kspeed
        if xv(t) - kc <= 1.3:
            break
    stop = (P[0] + d[0] * kc, P[1] + d[1] * kc)
    far = s.away_cell(P, d)
    while (far[0] - P[0]) * d[0] + (far[1] - P[1]) * d[1] < xv(t) + 3:  # he is still running when the blow lands
        far = (far[0] + d[0], far[1] + d[1])
    if not all(s.open(*c) for c in walk_cells(K, far)):
        raise ValueError(f'{s.id}: no clear line for {killer} after {victim}')
    steps = [dict(Actor=victim, Move=list(far), Speed=vspeed, After=r2(vafter), Free=True)]
    kw = dict(Speed=kspeed, After=r2(kafter))
    if enter:
        kw['Enter'] = list(K)
        s.pos[killer] = K
    s.away.discard(killer)
    st = dict(Actor=killer, Move=list(stop), Ease='line', **kw)
    if enter:
        st['Enter'] = list(K)
    steps.append(st)
    s.pos[killer] = stop
    steps.append(dict(Actor=killer, Strike=victim, React='hit', Kill=True, After=r2(max(0, t - IMPACT))))
    s.pos[victim] = (P[0] + d[0] * round(xv(t)), P[1] + d[1] * round(xv(t)))
    s.fell(victim)
    return steps, t


def flee(s, keys, rnd, toward, after=0.0, spread=.35, speed=(1.4, 2.0), avoid=()):
    """Folk running off out of sight, each on a slightly different heading."""
    steps = []
    for i, k in enumerate(keys):
        heads = [(toward[0] + rnd.uniform(-spread, spread) * (1 + t / 3), toward[1] + rnd.uniform(-spread, spread) * (1 + t / 3)) for t in range(8)]
        for d in heads + [(1, .3), (0, 1), (.7, .7), (1, -.3), (-.3, 1)]:  # another heading when the way is shut
            try:
                st = s.move(k, s.away_cell(s.pos[k], d), leave=True, Speed=r2(rnd.uniform(*speed)), After=r2(after + rnd.uniform(0, .8)),
                            past_own=True, avoid=avoid)
                break
            except ValueError:
                continue
        else:
            raise ValueError(f'{s.id}: {k} finds no way out')
        steps.append(st)
        s.away.add(k)
    return steps


# --- 6. 반동탁 연합군이 북서쪽으로 진군한다 -------------------------------------------------------------------

def march():
    rnd = random.Random(6)
    s = Shot('march', 'S_title_long_field', (-8, 30), radius=.44, time='dawn', hold=0, Steps=.12)
    # the lords' own troops are seasoned arms; 유비's are volunteers, a smaller square of plain footmen
    units = [('원소', 'yuanshao', 'yuan', 'heavy_infantry', 8), ('조조', 'caocao', 'wei', 'heavy_cavalry', 8),
             ('원술', 'yuanshu', 'yuan', 'heavy_infantry', 8), ('손견', 'sunjian', 'wu', 'pikeman', 8),
             ('도겸', 'guards', 'han', 'heavy_infantry', 8), ('공손찬', 'light_cavalry_white', 'gongsun', 'light_cavalry_white', 8),
             ('유비', 'liubei', 'shu', 'light_infantry', 6)]
    waves = [[0, 1, 2], [3, 4, 5], [6]]
    lanes = {3: [13, 0, -13], 1: [0]}  # left-lower, middle, upper-right
    far, speed = 40, 1.15
    steps, end = [], 0
    for w, wave in enumerate(waves):
        for lane, ui in zip(lanes[len(wave)], wave):
            name, sprite, faction, troop, n = units[ui]
            vc = s.cam[1] + lane
            ug = s.cam[0] + 23 + 12 * w + rnd.randint(0, 3) + (2 if lane == 0 else 0)  # each wave further back
            t0 = 1.5 * w + rnd.uniform(0, .6)
            people = [(name, sprite, ug, vc)]
            if name == '유비':
                people += [('관우', 'guanyu', ug + 1, vc - 2), ('장비', 'zhangfei', ug + 1, vc + 2)]
            for i in range(n):
                for j in range(n):
                    jit = (rnd.uniform(-.05, .05), rnd.uniform(-.05, .05))  # ranks kept, a hair off true
                    people.append((f'{name}군#{i}{j}', troop, ug + 2 + i + jit[0], vc - n // 2 + j + jit[1]))
            go = far
            while not all(s.outside(u - go, v) for _, _, u, v in people):  # on until the whole unit is out of sight
                go += 1
            for key, spr, u, v in people:
                s.actor(key, (u, v), 'NW', spr, faction)
                steps.append(dict(Actor=key, Move=[r2(u - go), r2(v)], Speed=speed, After=r2(t0 + rnd.uniform(0, .06)),
                                  Free=True, _leave=True))
                s.pos[key] = (u - go, v)
            if name == '유비':  # the shot closes once 유비's whole square has come well into sight
                rear = ug + 2 + n - 1
                inside = next(d for d in range(0, 80) if not s.outside(rear - d, vc, margin=-140))
                end = t0 + inside * STEP / speed + .6
    s.group(steps, dict(Wait=r2(end)))
    return s


# --- 5a. 유관장이 황건적에게서 장세평·소쌍을 구한다 ----------------------------------------------------------------

def rescue():
    rnd = random.Random(51)
    s = Shot('rescue', 'S_title_field_road', (5, 8), radius=.4, hold=.8)
    cx, cy = 5, 9
    s.ambient.append(dict(Prop=['grain_cart', cx - 1.5, cy - .5]))
    s.block((cx - 2, cy - 1), (cx - 1, cy - 1), (cx - 2, cy), (cx - 1, cy))
    s.actor('장세평', (cx, cy), 'NE', Pose='pant')
    s.actor('소쌍', (cx, cy + 1), 'NE', Pose='pant')
    s.actor('하인#1', (cx - 1, cy + 1), 'NE', 'commoner_man', 'merchant', Pose='pant')
    ring = [(cx + 2, cy - 1), (cx + 2, cy + 1), (cx + 1, cy + 3), (cx - 2, cy + 2), (cx - 3, cy - 2), (cx + 1, cy - 2)]
    turbans = []
    for i, at in enumerate(ring):
        if not s.open(*at):
            continue
        turbans.append(s.actor(f'황건#{i}', at, facing(cx - at[0], cy - at[1]), rnd.choice(['bandit', 'ruffian']), 'turban'))
    # the turbans close in on the merchants, blades up; the merchants cower
    g = []
    for i, k in enumerate(turbans):
        u, v = s.pos[k]
        step = (u - (u > cx) + (u < cx), v - (v > cy) + (v < cy))
        if s.open(*step) and step not in s.taken(k):
            g.append(s.move(k, step, end=facing(cx - step[0], cy - step[1]), Speed=.7, After=r2(.3 + .25 * i)))
        g.append(dict(Actor=k, Play='block', After=r2(1.4 + rnd.uniform(0, .6))))
    g += [dict(Actor=k, Emote=e, After=r2(.4 + .5 * i)) for i, (k, e) in enumerate((('장세평', '!'), ('소쌍', '!!'), ('하인#1', '…')))]
    g += [dict(Actor=turbans[0], Emote='!', After=.9), dict(Actor=turbans[2], Emote='!', After=1.3), dict(Wait=2.2)]
    s.group(g)
    # the three come running in from the north-east; the turbans turn on them, and go down one after another
    order = [('관우', 'guanyu'), ('장비', 'zhangfei'), ('유비', 'liubei')]
    g = [dict(Actor=k, Emote='!', After=r2(.5 + .1 * i)) for i, k in enumerate(turbans)]
    g += [dict(Actor=k, Look='NE', After=r2(.6 + .1 * i)) for i, k in enumerate(('장세평', '소쌍'))]
    left = list(turbans)
    hero_t = {}
    for i, (name, spr) in enumerate(order):
        foe = min(left, key=lambda f: s.pos[f][1] - s.pos[f][0])
        for toward in ((1, -.3), (1, 0), (1, .3), (.6, -.6), (1, .6)):  # whichever way in is clear
            start = s.away_cell((s.pos[foe][0] + 2, s.pos[foe][1] - 1 + i), toward)
            s.actor(name, start, 'SW', spr, 'shu', Off=True)
            s.away.discard(name)
            s.pos[name] = start
            steps, t = engage(s, name, foe, rnd, speed=2.4, after=.2 + .45 * i, react='hit')
            if steps:
                break
        else:
            raise ValueError(f'rescue: {name} finds no way in')  # all three brothers come, or the shot is not written
        left.remove(foe)
        steps[0]['Enter'] = list(start)
        more, t = fight(s, name, foe, t + .3, rnd.randint(1, 2), foe, rnd, gap=(.55, .75))
        g += steps + more
        hero_t[name] = t
    for name in ('관우', '장비'):  # the two go straight on to the next
        if not left or name not in hero_t:
            continue
        foe = min(left, key=lambda f: abs(s.pos[f][0] - s.pos[name][0]) + abs(s.pos[f][1] - s.pos[name][1]))
        steps, t = engage(s, name, foe, rnd, speed=2.2, after=hero_t[name], react='hit')
        if steps:
            left.remove(foe)
            more, t = fight(s, name, foe, t + .2, 1, foe, rnd, gap=(.55, .7))
            g += steps + more
            hero_t[name] = t
    # the last of them take to their heels
    t_run = max(hero_t.values()) - 1
    g += [dict(Actor=k, Emote='!!', After=r2(t_run - .3)) for k in left]
    g += flee(s, left, rnd, (-.3, 1), after=t_run, spread=.3, speed=(1.6, 2))
    s.group(g)
    # 유비 goes to them; they get up and thank him
    near = min(s.free((cx + 1, cy - 1), n=6), key=lambda c: abs(c[0] - cx - 1) + abs(c[1] - cy + 1))
    g = [s.move('유비', near, end=facing(cx - near[0], cy - near[1]), Speed=1.2)]
    g += [dict(Actor=k, Pose='idle', After=.4) for k in ('장세평', '소쌍', '하인#1')]
    g += [dict(Actor='장세평', Face=facing(near[0] - cx, near[1] - cy), After=.8), dict(Actor='소쌍', Face='NE', After=.9),
          dict(Actor='장세평', Emote='…', After=1.4), dict(Actor='장세평', Emote='!', After=1.9), dict(Actor='소쌍', Emote='…!', After=2.1),
          dict(Actor='유비', Emote='…', After=2.6), dict(Actor='장비', Emote='!', After=2.8), dict(Wait=3.6)]
    s.group(g)
    return s


# --- 6a. 호뢰관: 여포와 겨루는 장비, 달려드는 관우와 유비 -----------------------------------------------------------

def hulao():
    rnd = random.Random(61)
    s = Shot('hulao', 'S_title_long_field', (12, 64), radius=.44, time='dusk', hold=.8)
    lb, zf = (12, 62), (12, 63)
    s.actor('여포', lb, 'SW', 'lubu', 'dong')
    s.actor('장비', zf, 'NE', 'zhangfei', 'shu')
    dong = [s.actor(f'서량기병#{i}', (u, v), 'SW', rnd.choice(['light_cavalry', 'heavy_cavalry']), 'dong')
            for i, (u, v) in enumerate((u, v) for v in (54, 55) for u in range(4, 21, 2))]
    allies = [s.actor(f'연합군#{i}', (u, v), 'NE', rnd.choice(['heavy_infantry', 'pikeman', 'light_infantry']), rnd.choice(['yuan', 'wei', 'wu', 'han']))
              for i, (u, v) in enumerate((u, v) for v in (71, 72) for u in range(4, 21, 2))]
    g = []
    g += [dict(Actor=k, Emote=rnd.choice(['!', '!!']), After=r2(rnd.uniform(0, 14))) for k in rnd.sample(dong + allies, 14)]
    g += [st for k in allies for st in cheer(k, rnd, after=rnd.uniform(2, 12), emote=0)]
    more, t = duel(s, '장비', '여포', .6, rnd, n=8, favor='여포')
    g += more + [dict(Actor='장비', Emote='!', After=r2(t - 1.5))]
    # 관우 comes in from the west, then 유비 from the allies' side; three on one
    arrived = t
    for name, spr, toward, delay in (('관우', 'guanyu', (-1, .2), .2), ('유비', 'liubei', (.2, 1), 1.6)):
        start = s.away_cell(s.pos['여포'], toward)
        s.actor(name, start, 'NE', spr, 'shu', Off=True)
        s.away.discard(name)
        s.pos[name] = start
        steps, ta = engage(s, name, '여포', rnd, speed=2.3, after=t + delay)
        if steps:
            steps[0]['Enter'] = list(start)
            g += steps
            arrived = max(arrived, ta)
    t = arrived + .5
    three = [k for k in ('장비', '관우', '유비') if k in s.pos and k not in s.away]
    for n in range(10):  # three on one, and still 여포 has the better of it: he turns from one to the next, they barely get a blow in
        k = three[n % len(three)]
        if max(abs(s.pos[k][0] - s.pos['여포'][0]), abs(s.pos[k][1] - s.pos['여포'][1])) != 1:
            continue
        if n % 2 == 0:
            st = dict(Actor='여포', Strike=k, React=rnd.choice(['block', 'hit']), Tempo=DUEL_TEMPO, After=r2(t))
            if n % 4 == 0:
                st['Crit'] = True
            g.append(st)
        else:
            g.append(dict(Actor=k, Strike='여포', React='block', Tempo=DUEL_TEMPO, After=r2(t)))
        t += DUEL_IMPACT + .04  # the next comes in as this one lands
    g.append(dict(Wait=r2(t + .8)))
    s.extra['End'] = r2(t - 1.2)  # it darkens in the thick of it
    s.group(g)
    return s


# --- 7. 낙양이 불타고 동탁군이 약탈한다 -----------------------------------------------------------------------

def luoyang():
    rnd = random.Random(7)
    s = Shot('luoyang', 'S_title_luoyang', (14, 13), radius=.46, time='night', hold=.4, Glow='fire_nw')
    s.ambient += [dict(Effect='bigfire', At=list(at)) for at in [(4.5, 15.5), (10.5, 15.5), (7.5, 21.5), (24.5, 10.5)]]
    folk = []
    for i, at in enumerate([(6, 12), (9, 13), (12, 12), (15, 13), (18, 12), (21, 13), (24, 12), (8, 17), (13, 18),
                            (17, 16), (20, 19), (26, 13), (3, 13), (11, 5), (16, 4)]):
        folk.append(s.actor(f'백성#{i}', at, rnd.choice(['SE', 'SW', 'NE']), COMMONERS[i % 4], 'merchant'))
    troops = []
    for i in range(10):
        troops.append(s.actor(f'서량병#{i}', (-14 - i % 3, 12 + i % 2), 'SE', ['heavy_infantry', 'light_infantry'][i % 2], 'dong', Off=True))
    riders = [s.actor(f'서량기병#{i}', (-16, 12 + i), 'SE', 'light_cavalry', 'dong', Off=True) for i in range(2)]
    # the first three on the avenue are run down from behind; the rest scatter south and east
    g = [dict(Actor=k, Emote='!', After=r2(.2 + .07 * i)) for i, k in enumerate(folk)]
    end = 0
    for n, (killer, victim, gap, ka) in enumerate([('서량병#0', '백성#0', 14, 0), ('서량병#1', '백성#1', 17, .3), ('서량병#2', '백성#2', 20, .6)]):
        steps, t = chase(s, killer, victim, (1, 0), gap, kafter=ka, vafter=.5)
        g += steps
        end = max(end, t)
    g += flee(s, folk[3:13], rnd, (1, .5), after=.4)
    g.append(dict(Wait=r2(end + .6)))
    s.group(g)
    # more come up the avenue: some break into houses for loot, the riders sweep through, the killers move on
    s.ambient += [dict(Prop=['wine_jar#a', 5.6, 13.6]), dict(Prop=['grain_sack#b', 11.6, 14]), dict(Prop=['grain_sack#c', 8.5, 14])]
    g = []
    for i, (k, dst) in enumerate([('서량병#3', (6, 13)), ('서량병#4', (11, 13)), ('서량병#5', (8, 13))]):
        g.append(s.move(k, dst, Enter=[-14, 12 + i % 2], end='SW', Speed=1.6, After=r2(.3 * i)))
    for i, k in enumerate(riders):
        g.append(s.move(k, (40, 12 + i), Enter=[-16, 12 + i], straight=True, Speed=2.4, After=r2(1 + .5 * i), leave=True))
        s.away.add(k)
    for k in ('서량병#0', '서량병#1', '서량병#2'):
        g.append(dict(Actor=k, Play='attack', After=r2(rnd.uniform(.2, .8))))
        g.append(dict(Actor=k, Hop=1, After=r2(rnd.uniform(1, 1.6))))
    g += flee(s, folk[13:], rnd, (-.2, -1), after=.6)
    s.group(g)
    s.group(dict(Prop=['wine_jar#a', '서량병#3']), dict(Prop=['grain_sack#b', '서량병#4']), dict(Prop=['grain_sack#c', '서량병#5']),
            dict(Actor='서량병#3', Play='attack'), dict(Actor='서량병#4', Play='attack', After=.3))
    g = []
    for i, k in enumerate(('서량병#3', '서량병#4', '서량병#5')):
        g.append(s.move(k, s.away_cell(s.pos[k], (-1, -.2)), leave=True, Speed=1.1, After=r2(.2 * i)))
        s.away.add(k)
    for i, k in enumerate(('서량병#0', '서량병#1', '서량병#2')):
        g.append(s.move(k, s.away_cell(s.pos[k], (1, .3)), leave=True, Speed=1.5, After=r2(.5 + .3 * i)))
        s.away.add(k)
    for i, k in enumerate(troops[6:]):
        g.append(s.move(k, s.away_cell((-14, 13), (1, .15 * (i - 2))), Enter=[-14 - i, 12 + i % 2], leave=True, Speed=1.5,
                        After=r2(.4 + .35 * i)))
        s.away.add(k)
    g.append(dict(Wait=3))
    s.group(g)
    return s


# --- 8. 계교: 공손찬을 쫓는 문추를 조운이 옆구리에서 받아 내고, 둘이 겨룬다 ------------------------------------

def duel(s, a, b, t0, rnd, n=10, favor=None):
    """Two champions at it like a duel, with no breath between: the man who parries cuts back the moment the blow
    lands on his blade, a blow dodged with a step back still facing the foe has the other coming straight on, a blow
    that goes home drives its man back a cell with the striker going in with him, and now and then one gives ground and
    the other presses after. favor, if given, is the stronger: he attacks two blows in three, and the other's blows never
    go home. Returns the steps and the end."""
    land = DUEL_IMPACT  # seconds from a blow's start to it landing, at the duel's tempo
    whole = DUEL_SWING  # seconds of the whole swing
    order = []
    for i in range(n):
        if favor:
            weak = b if favor == a else a
            order.append((favor, weak) if i % 3 != 2 else (weak, favor))
        else:
            order.append((a, b) if (i + (i // 3)) % 2 == 0 else (b, a))
    steps, t = [], t0

    def back_cell(frm, of):
        n_ = math.hypot(of[0] - frm[0], of[1] - frm[1]) or 1
        return (round(of[0] + (of[0] - frm[0]) / n_), round(of[1] + (of[1] - frm[1]) / n_))

    for i, (by, at) in enumerate(order):
        (bu, bv), (tu, tv) = s.pos[by], s.pos[at]
        if favor and by != favor:
            kind = rnd.choices(['block', 'dodge'], [3, 1])[0]
        elif favor:
            kind = rnd.choices(['block', 'dodge', 'heavy', 'hit'], [2, 1, 2.5, 2.5])[0]
        else:
            kind = rnd.choices(['block', 'dodge', 'heavy', 'hit'], [4, 2, 1.5, 1.5])[0]
        back = back_cell((bu, bv), (tu, tv))
        if kind != 'block' and (not s.open(*back) or back in s.taken(at)):
            kind = 'block'
        nxt = order[i + 1][0] if i + 1 < len(order) else None
        if kind == 'dodge':  # the blow cuts air: he springs back still facing the man, who comes straight on after him
            steps += [dict(Actor=by, Face=facing(tu - bu, tv - bv), After=r2(t)), dict(Actor=by, Play='attack', Tempo=DUEL_TEMPO, After=r2(t)),
                      dict(Actor=at, Face=facing(bu - tu, bv - tv), After=r2(t)),
                      dict(Actor=at, Move=list(back), Ease='out', Speed=3.4, Back=True, After=r2(t + .03)),
                      dict(Actor=by, Move=[tu, tv], Ease='out', Speed=2.6, After=r2(t + land), End=facing(tu - bu, tv - bv))]
            s.pos[at], s.pos[by] = back, (tu, tv)
            t += land + .12
        elif kind == 'block':  # parried: the other cuts back as it lands, or the same man swings again when his is done
            steps.append(dict(Actor=by, Strike=at, React='block', Tempo=DUEL_TEMPO, After=r2(t)))
            t += land if nxt == at else whole
        else:  # it goes home, or a mighty blow is parried and still drives him back: in he goes with it
            st = dict(Actor=by, Strike=at, React='hit' if kind == 'hit' else 'block', Push=1, Tempo=DUEL_TEMPO, After=r2(t))
            if kind == 'heavy':
                st['Crit'] = True
            steps += [st, dict(Actor=by, Move=[tu, tv], Ease='out', Speed=.95, After=r2(t + land))]
            s.pos[at], s.pos[by] = back, (tu, tv)
            t += land + slide_time(1) + .02
        if i % 4 == 3 and nxt:  # footwork: the next man to be struck gives ground a step, the other presses after him
            fb, fa = nxt, (order[i + 1][1])
            (pu, pv), (qu, qv) = s.pos[fb], s.pos[fa]
            give = back_cell((pu, pv), (qu, qv))
            if s.open(*give) and give not in s.taken(fa):
                steps += [dict(Actor=fa, Face=facing(pu - qu, pv - qv), After=r2(t)),
                          dict(Actor=fa, Move=list(give), Ease='out', Speed=2, Back=True, After=r2(t)),
                          dict(Actor=fb, Move=[qu, qv], Ease='out', Speed=2, After=r2(t + .05), End=facing(qu - pu, qv - pv))]
                s.pos[fa], s.pos[fb] = give, (qu, qv)
                t += .2
    return steps, t


DUEL_TEMPO = 2  # a duel's blows and the answers to them play at twice their speed
DUEL_IMPACT = .23  # seconds from a duel's blow to its landing, at that tempo (a 0.9 s swing, its first half the wind-up)
DUEL_SWING = .45  # seconds of a whole duel swing


def slide_time(n):
    return .2 + .12 * n


def gyeqiao():
    rnd = random.Random(8)
    s = Shot('gyeqiao', 'S_title_gyeqiao', (-12, 30), radius=.46, hold=.8)
    s.cams.append((-15, 31))
    s.actor('공손찬', (14, 30), 'NW', 'light_cavalry_white', 'gongsun', Off=True)
    s.actor('문추', (22, 30), 'NW', 'heavy_cavalry', 'yuan', Off=True)
    sp = 2.3
    per = STEP / sp
    # 공손찬 rides north-west until the cliff stops him, and turns
    g = [s.move('공손찬', (-19, 30), Enter=[14, 30], straight=True, Speed=sp, end='SE')]
    t_g = (14 + 19) * per
    g.append(dict(Actor='공손찬', Emote='!', After=r2(t_g + .2)))
    # 문추 on his heels; 조운 comes down from the north-east and takes him in the flank as he passes
    t_m0 = 1.2
    stop_m = (-15, 30)
    t_m = t_m0 + (22 - stop_m[0]) * per
    zsrc = (-15, 8)
    s.actor('조운', zsrc, 'SW', 'zhaoyun', 'gongsun', Off=True)
    zsp = 2.6
    t_z0 = t_m - (30 - 1 - zsrc[1]) * STEP / zsp + .05
    g += [s.move('문추', stop_m, Enter=[22, 30], straight=True, Speed=sp, After=t_m0),
          s.move('조운', (-15, 29), Enter=list(zsrc), straight=True, Speed=zsp, After=r2(t_z0)),
          dict(Actor='조운', Strike='문추', React='hit', Push=3, Crit=True, After=r2(t_m - IMPACT + .05))]
    s.pos['문추'] = (-15, 33)
    s.group(g, dict(Wait=r2(t_m + slide_time(3) + .6)))
    # 조운 closes; they go at each other
    g = [s.move('조운', (-15, 32), end='SW', Speed=1.6), dict(Actor='문추', Face='NE', After=.3),
         dict(Actor='공손찬', Look='SW', After=.5)]
    more, t = duel(s, '조운', '문추', .9, rnd, n=11)
    g += more + [dict(Actor='공손찬', Emote='…', After=r2(t * .6)), dict(Wait=r2(t + .6))]
    s.extra['End'] = r2(t - 1.2)  # it darkens with the two still at it
    s.group(g)
    return s


def path_len(s, st, src):
    """Cells a step's walk covers from src."""
    at, n = tuple(src), 0
    for c in st.get('Via', []) + [st['Move']]:
        n += abs(c[0] - at[0]) + abs(c[1] - at[1]) if 'Ease' not in st else math.hypot(c[0] - at[0], c[1] - at[1])
        at = tuple(c)
    return n


def engage(s, a, foe, rnd, speed=1.8, after=0.0, kill=False, react=None, motion='', avoid=()):
    """a runs to the free cell next to foe nearest to it and strikes as it gets there; [] when no way."""
    fu, fv = s.pos[foe]
    au, av = s.pos[a]
    spots = sorted((c for c in s.free((fu, fv), n=9) if max(abs(c[0] - fu), abs(c[1] - fv)) == 1),
                   key=lambda c: abs(c[0] - au) + abs(c[1] - av))
    for spot in spots:
        src = s.pos[a]
        try:
            st = s.move(a, spot, Speed=speed, After=r2(after), avoid=avoid,
                        end=facing(fu - spot[0], fv - spot[1]))
        except ValueError:
            continue
        t = after + walk_time(path_len(s, st, src), speed)
        hit = dict(Actor=a, Strike=foe, React=react or ('hit' if kill else rnd.choice(['block', 'hit'])), After=r2(max(0, t - IMPACT)))
        if kill:
            hit['Kill'] = True
            s.fell(foe)
        if motion:
            hit['Motion'] = motion
        return [st, hit], t
    return [], 0


def adjacent(s, a, b):
    return max(abs(s.pos[a][0] - s.pos[b][0]), abs(s.pos[a][1] - s.pos[b][1])) == 1


def melee_round(s, attackers, defenders, rnd, pairs, reach=6, counter=2, kills=.25, wait=1.8, speed=1.8, hold=()):
    """One round of a melee: everyone of attackers not fighting goes for the nearest enemy in reach and strikes as
    he gets there; defenders not fighting turn on an attacker close by; the pairs already at it keep trading blows;
    and a share of the pairs end with one of them down. pairs maps a fighter to his foe and is kept up to date.
    Returns the round's steps."""
    g = []
    live = lambda ks: [k for k in ks if k not in s.down and k not in s.away]
    for k in list(pairs):  # pairs whose man fell, or who drifted apart, break up
        f = pairs[k]
        if k in s.down or f in s.down or not adjacent(s, k, f):
            pairs.pop(k, None)
    busy = set(pairs) | set(pairs.values())

    def go(a, pool, rng, spd):
        for foe in sorted((f for f in pool if f not in busy), key=lambda f: abs(s.pos[f][0] - s.pos[a][0]) + abs(s.pos[f][1] - s.pos[a][1])):
            if abs(s.pos[foe][0] - s.pos[a][0]) + abs(s.pos[foe][1] - s.pos[a][1]) > rng:
                return
            if adjacent(s, a, foe):
                g.extend(brawl(a, foe, rnd))
            else:
                steps, t = engage(s, a, foe, rnd, speed=spd, after=rnd.uniform(0, .6))
                if not steps:
                    continue
                g.extend(steps + brawl(a, foe, rnd, t + .3))
            pairs[a], pairs[foe] = foe, a
            busy.update((a, foe))
            return

    for a in live(attackers):
        if a not in busy:
            go(a, live(defenders), reach, speed)
    for d in live(defenders):
        if d not in busy and d not in hold:
            go(d, live(attackers), counter, 1.4)
    done = set()
    for k, f in list(pairs.items()):
        if k in done or f in done or rnd.random() >= kills:
            continue
        done.update((k, f))
        a, b = (k, f) if rnd.random() < .5 else (f, k)
        t = rnd.uniform(.6, wait)
        g += [dict(Actor=a, Play='attack', After=r2(t)), dict(Actor=b, Fall=True, After=r2(t + IMPACT))]
        s.fell(b)
    g.append(dict(Wait=wait))
    return g


def fight(s, a, b, t0, exchanges, loser, rnd, gap=(.75, 1.05)):
    """a and b at arm's length trade exchanges blows from t0, a first: mostly parried, now and then one goes home,
    now and then a heavy blow gathered and driven; then the winner cuts the loser down. Returns the steps and when
    it ends."""
    steps, t = [], t0
    for i in range(exchanges):
        by, at = (a, b) if i % 2 == 0 else (b, a)
        st = dict(Actor=by, Strike=at, After=r2(t))
        roll = rnd.random()
        if roll < .18:  # a heavy blow, gathered first
            st.update(Gather=.35, Crit=True, React='block')
            t += .35
        elif roll < .45:
            st['React'] = 'hit'
        else:
            st['React'] = 'block'
        steps.append(st)
        t += rnd.uniform(*gap)
    winner = b if loser == a else a
    steps.append(dict(Actor=winner, Strike=loser, React='hit', Kill=True, After=r2(t)))
    s.fell(loser)
    return steps, t + .6



# --- 9. 서주: 고을 밖 들판의 학살, 북서에서 뛰어드는 관우, 북동에서 하나둘 오는 유비군 ------------------------------

def xuzhou():
    rnd = random.Random(9)
    s = Shot('xuzhou', 'S_title_xuzhou_ruin', (10, 12), radius=.46, time='dusk', hold=.3)
    s.ambient += [dict(Effect='bigfire', At=list(at)) for at in [(6.5, 1.5), (2.5, 2.5), (9.5, 2.5)]]
    g = []
    # the folk run off south-east across the fields, each along its own row; 조조's men come after them from the
    # north-west, cut down the slowest in the north-west of the fields and run on after the rest
    rows = list(range(11, 22))
    folk, chasers = [], []
    for i, v in enumerate(rows):
        for u in (rnd.randint(-7, -3), rnd.randint(1, 5), rnd.randint(8, 12)):
            folk.append(s.actor(f'백성#{len(folk)}', (u, v), rnd.choice(['SE', 'NE', 'SW']), COMMONERS[len(folk) % 4], 'merchant'))
    victims = []
    for v in rows[::2]:  # in every other row the hindmost is run down
        k = min((f for f in folk if s.pos[f][1] == v), key=lambda f: s.pos[f][0])
        victims.append(k)
    after_kill = {}
    for n, vk in enumerate(victims):
        k = s.actor(f'조조군#c{n}', (s.pos[vk][0] - 18, s.pos[vk][1]), 'SE', rnd.choice(FOOT), 'wei', Off=True)
        steps, t = chase(s, k, vk, (1, 0), 18, kspeed=1.9, vspeed=.5, vafter=r2(.4 + .2 * n), kafter=r2(.15 * n))
        g += steps
        chasers.append(k)
        after_kill[k] = t
    for k in folk:  # the rest run, some late
        if k in victims:
            continue
        u, v = s.pos[k]
        g.append(s.move(k, s.away_cell((u, v), (1, 0)), leave=True, Speed=r2(rnd.uniform(.6, .9)), After=r2(rnd.uniform(.1, 2.5)),
                        past_own=True))
        s.away.add(k)
    # the killers run on after the folk; 관우 comes at them from the east, the way the folk ran, and meets the first
    # head on: one stroke and he is down
    sc = .8
    T = max(after_kill.values()) + 2.2  # 관우's first blow
    plan = {}
    for k in chasers:
        u0, v = s.pos[k]
        t0 = after_kill[k] + .6
        plan[k] = (u0, v, t0)
    lead = max(chasers, key=lambda k: plan[k][0] + (T - plan[k][2]) * sc / STEP)
    u0, v, t0 = plan[lead]
    meet = round(u0 + (T - t0) * sc / STEP)
    T = t0 + (meet - u0) * STEP / sc
    g.append(dict(Actor=lead, Move=[meet, v], Ease='line', Speed=sc, After=r2(t0)))
    s.pos[lead] = (meet, v)
    src = (meet + 18, v - 10)
    s.actor('관우', src, 'SW', 'guanyu', 'shu', Off=True)
    run = math.hypot(src[0] - meet - 1, src[1] - v)
    sp = 3.4
    g.append(dict(Actor='관우', Enter=list(src), Move=[meet + 1, v], Ease='line', Speed=sp, After=r2(T - run * STEP / sp), End='NW'))
    s.pos['관우'] = (meet + 1, v)
    s.away.discard('관우')
    g.append(dict(Actor='관우', Strike=lead, Motion='cleave', React='hit', Kill=True, Crit=True, After=r2(T - IMPACT)))
    s.fell(lead)
    def clean(steps, names):
        """Whether steps added to g keep names clear of everyone (no passing through, every blow in reach)."""
        s.groups.append(g + steps)
        bad = [p for p in s.check() if any(n in p for n in names)]
        s.groups.pop()
        return not bad

    def shifted(steps, by):
        return [dict(st, After=r2(st.get('After', 0) + by)) for st in steps]

    # the other killers, still running on after the folk, and more of 조조's men coming up from the south-west, meet
    # the three brothers one after another: three or four blows each, then down
    pool = [k for k in chasers if k != lead]
    gu, gv = s.pos['관우']
    for k in pool:  # each runs on after the folk, and on seeing 관우 closes round him, three cells off
        u0k, vk, t0k = plan[k]
        ring = sorted((c for c in s.free((gu, gv), n=40) if max(abs(c[0] - gu), abs(c[1] - gv)) == 2),
                      key=lambda c: abs(c[0] - u0k) + abs(c[1] - vk))
        if ring:
            try:
                g.append(s.move(k, ring[0], end=facing(gu - ring[0][0], gv - ring[0][1]), Speed=1.9, After=r2(t0k)))
            except ValueError:
                pass
        g.append(dict(Actor=k, Emote='!', After=r2(T + rnd.uniform(.1, .6))))
    for i in range(5):
        at = (meet - 8 + 3 * i, v + 20 + i % 2 * 2)
        k = s.actor(f'조조군#w{i}', at, 'NE', rnd.choice(FOOT), 'wei', Off=True)
        pool.append(k)
    hero_t = {'관우': T + .5}
    for name, sprite, delay in (('장비', 'zhangfei', 1.0), ('유비', 'liubei', 2.0)):
        s.actor(name, src, 'SW', sprite, 'shu', Off=True)
        hero_t[name] = T + delay
    busy, end = set(), T
    for _ in range(2):
        for h in ('관우', '장비', '유비'):
            live = [c for c in pool if c not in s.down and c not in busy]
            if not live:
                break
            foe = min(live, key=lambda c: abs(s.pos[c][0] - s.pos[h][0]) + abs(s.pos[c][1] - s.pos[h][1]))
            done = False
            for delay in (0, .6, 1.2, 2, 3):
                if h in s.away:  # 장비 and 유비 come in from the east as 관우 did, a little apart
                    for d in ((18, -10), (16, -14), (12, -16), (20, -6)):
                        start = (s.pos[foe][0] + d[0], s.pos[foe][1] + d[1])
                        if not s.open(*start):
                            continue
                        keep = dict(s.pos), set(s.away)
                        s.cast[h]['At'] = list(start)
                        s.pos[h] = start
                        s.away.discard(h)
                        steps, t = engage(s, h, foe, rnd, speed=2.4, after=hero_t[h] + delay)
                        if steps:
                            steps[0]['Enter'] = list(start)
                            if clean(steps, [h]):
                                done = True
                                break
                        s.pos.clear(); s.pos.update(keep[0]); s.away.clear(); s.away.update(keep[1])
                    if done:
                        break
                    continue
                keep = dict(s.pos), set(s.away)
                if foe in s.away:  # a man from the south-west comes on to meet him where he stands
                    s.away.discard(foe)
                    steps, t = engage(s, foe, h, rnd, speed=1.7, react='block')
                    if steps:
                        steps[0]['Enter'] = list(s.cast[foe]['At'])
                        travel = t
                        steps = shifted(steps, max(0, hero_t[h] + delay - travel))
                        t = max(travel, hero_t[h] + delay)
                else:
                    steps, t = engage(s, h, foe, rnd, speed=2.2, after=hero_t[h] + delay)
                if steps and clean(steps, [h, foe]):
                    done = True
                    break
                s.pos.clear(); s.pos.update(keep[0]); s.away.clear(); s.away.update(keep[1])
            if not done:
                continue
            g += steps
            busy.add(foe)
            more, t_end = fight(s, h, foe, t + .35, rnd.randint(3, 4), foe, rnd, gap=(.55, .75))
            g += more
            hero_t[h] = t_end
            end = max(end, t_end)
    # the first of 유비's own men come running from the north-east as the shot closes
    t_close = min(end, max(hero_t['유비'], hero_t['장비'])) + .2
    for i in range(3):
        k = s.actor(f'유비군#{i}', (meet + 14 + 2 * i, v - 16 - i), 'SW', rnd.choice(FOOT), 'shu', Off=True)
        dst = s.free((meet + 6 + 2 * i, v - 6), n=1)[0]
        g.append(s.move(k, dst, Enter=list(s.cast[k]['At']), Speed=1.8, After=r2(t_close - 1.6 + .3 * i)))
    g.append(dict(Wait=r2(t_close + .4)))
    joined = max(min((st['After'] for st in g if st.get('Actor') == h and st.get('Strike')), default=t_close) for h in ('유비', '장비'))
    s.extra['End'] = r2(joined + 1.3)  # it darkens with 유비 and 장비 a blow or two into it, everyone still at it
    s.extra['Lead'] = 1.5  # it opens on the folk already running
    s.group(g)
    return s


# --- 11. 조조 진영에 여포가 왔다는 소식이 닿는다 ---------------------------------------------------------------

def alarm():
    s = Shot('alarm', 'S_council', (5, 4), radius=.3, hold=.6, interior=True)
    s.actor('조조', (1, 3), 'SE', Pose='sit')
    for k, at, f in [('순욱', (3, 1), 'SW'), ('곽가', (6, 1), 'SW'), ('정욱', (8, 1), 'SW'), ('하후돈', (3, 6), 'NE'),
                     ('하후연', (4, 6), 'NE'), ('조인', (6, 6), 'NE'), ('허저', (10, 2), 'SW')]:
        s.actor(k, at, f)
    s.actor('전령#1', (11, 4), 'NW', 'light_infantry', 'wei', Off=True)
    s.group(dict(Actor='곽가', Emote='…'), dict(Actor='조조', Emote='…', After=.8), dict(Actor='순욱', Emote='…', After=1.4), dict(Wait=1.8))
    s.group(s.move('전령#1', (5, 4), Enter=[11, 4], end='NW', Speed=2.2))
    s.group(dict(Actor='전령#1', Say='!!!!'))
    s.group(dict(Actor='조조', Pose='idle'), dict(Actor='조조', Emote='!!!!'), dict(Actor='곽가', Emote='!!!!', After=.2),
            dict(Actor='순욱', Emote='!!', After=.35), dict(Actor='정욱', Emote='!', After=.5),
            dict(Actor='하후돈', Look='SE', After=.2), dict(Actor='허저', Look='SE', After=.3), dict(Wait=1.2))
    s.group(s.move('조조', (2, 4), end='SE', Speed=1.4), s.move('하후돈', (3, 4), end='NW', Speed=2),
            s.move('하후연', (7, 5), end='NW', Speed=2, After=.2), s.move('조인', (9, 5), end='NW', Speed=2, After=.4),
            s.move('허저', (9, 3), end='SW', Speed=2, After=.1), s.move('곽가', (4, 2), end='SW', Speed=1.5, After=.3),
            s.move('순욱', (2, 2), end='SE', Speed=1.5, After=.3), s.move('전령#1', (10, 4), end='SE', Speed=2, After=.5))
    s.group(dict(Actor='조조', Play='action'), dict(Actor='곽가', Emote='…', After=.3),
            s.move('하후돈', (11, 4), end='SE', Speed=2, After=.6), s.move('허저', (11, 3), end='SE', Speed=2, After=.8),
            dict(Actor='정욱', Look='SE', After=.6), dict(Wait=1.6))
    return s


# --- 12. 백문루 ---------------------------------------------------------------------------------------------------

def baimen():
    s = Shot('baimen', 'S_title_baimen', (-2, -3), radius=.36, hold=.8, interior=True)
    s.actor('조조', (5, 3), 'SW', Pose='sit', Hold=True)
    for k, at, f in [('관우', (2, 4), 'SE'), ('유비', (3, 4), 'SE'), ('장비', (3, 5), 'SE'), ('허저', (7, 4), 'SW')]:
        s.actor(k, at, f)
    s.actor('위병#1', (8, 4), 'SW', 'light_infantry', 'wei')
    s.actor('위병#2', (9, 4), 'SW', 'light_infantry', 'wei')
    for k in ('진궁', '여포'):
        s.actor(k, (22, 5), 'NW', Off=True, Bound=True)
    for i in range(2):
        s.actor(f'호송병#{i}', (23 + i, 5), 'NW', 'light_infantry', 'wei', Off=True)

    def bring(who):
        s.group(s.move(who, (5, 5), Enter=[22, 5], end='NE', Speed=.8),
                s.move('호송병#0', (6, 5), Enter=[23, 5], end='NW', Speed=.8, After=.15),
                s.move('호송병#1', (7, 5), Enter=[24, 5], end='NW', Speed=.8, After=.3))

    def away(who):
        out = [s.move('호송병#1', (26, 5), leave=True, Speed=.8), s.move('호송병#0', (25, 5), leave=True, Speed=.8),
               s.move(who, (24, 5), leave=True, Speed=.8)]
        for k in (who, '호송병#0', '호송병#1'):
            s.away.add(k)
        return out

    bring('진궁')
    for who, line in [('조조', '……'), ('진궁', '……'), ('조조', '……?'), ('진궁', '……!'), ('조조', '……'), ('진궁', '……')]:
        s.group(dict(Actor=who, Say=line))
        if who == '진궁' and line == '……!':
            s.group(dict(Actor='진궁', Look='NW'), dict(Actor='조조', Emote='…', After=.4), dict(Wait=.8))
    s.group(dict(Actor='조조', Emote='…'), dict(Wait=1))
    s.group(away('진궁'), dict(Actor='조조', Look='SE', After=.3), dict(Wait=1.6))
    s.group(dict(Actor='조조', Face='SW'))
    s.pos['호송병#0'], s.pos['호송병#1'] = (23, 5), (24, 5)
    bring('여포')
    s.group(dict(Actor='여포', Say='……!'))
    s.group(dict(Actor='조조', Say='……'))
    s.group(dict(Actor='여포', Face='NW'), dict(Actor='여포', Say='……!!'))
    s.group(dict(Actor='유비', Say='……'))
    s.group(dict(Actor='조조', Look='NW'), dict(Actor='조조', Emote='…', After=.3), dict(Wait=.8))
    s.group(dict(Actor='여포', Emote='!!!!'), dict(Actor='여포', Say='!!!!', After=.2), dict(Shake=.3, After=.2),
            dict(Actor='장비', Emote='!', After=.6))
    s.group(away('여포'), dict(Actor='유비', Look='SE', After=.6), dict(Wait=1.8))
    return s


# --- 13a. 조조가 헌제를 맞이한다 ----------------------------------------------------------------------------------

def emperor():
    rnd = random.Random(31)
    s = Shot('emperor', 'S_title_field_road', (4, 7), radius=.44, hold=1)
    v = 7
    # the emperor comes down the road from the north-west with a few ministers and guards
    party = [('헌제', '', '', (-16, v))]
    for i, (du, dv) in enumerate([(-1, -1), (-1, 1), (-2, 0), (-3, -1), (-3, 1)]):
        party.append((f'대신#{i}', rnd.choice(['official_blue_noncombat', 'official_noncombat', 'official_crimson_noncombat']), 'han',
                      (-16 + du, v + dv)))
    for i, (du, dv) in enumerate([(-5, -1), (-5, 1), (-6, 0)]):
        party.append((f'호위#{i}', 'guards', 'han', (-16 + du, v + dv)))
    g = []
    stop = 0
    for k, spr, fac, (u, vv) in party:
        s.actor(k, (u, vv), 'SE', spr, fac, Hold=k == '헌제')
        g.append(s.move(k, (u + 14, vv), Speed=.6, end='SE', After=r2(rnd.uniform(0, .3))))
    # 조조, 하후돈 and their riders come up the road from the south-east
    riders = [('조조말', 'caocao', (24, v), (6, v), '조조', 'caocao_noncombat'),
              ('하후돈말', 'royal_guard', (25, v - 2), (7, v - 2), '하후돈', 'xiahoudun_noncombat')]
    for i, (du, dv) in enumerate([(2, -1), (2, 1), (3, -3), (3, 3), (4, -1), (4, 1), (5, -2), (5, 2)]):
        riders.append((f'기병#{i}', 'light_cavalry', (26 + du, v + dv), (8 + du, v + dv), f'병사#{i}', 'infantry_noncombat'))
    for k, spr, start, end, person, pspr in riders:
        s.actor(k, start, 'NW', spr, 'wei', Off=True)
        g.append(s.move(k, end, Enter=list(start), straight=True, Speed=1.7, After=r2(1.5 + rnd.uniform(0, .5)), end='NW'))
        s.actor(person, (end[0] - 1, end[1]), 'NW', '' if person in ('조조', '하후돈') else pspr, '' if person in ('조조', '하후돈') else 'wei',
                Off=True)
    s.group(g)
    # they get down from their horses and bow; the emperor raises a hand
    g = []
    for i, (k, spr, start, end, person, pspr) in enumerate(riders):
        t = .2 + .12 * i
        g += [dict(Actor=k, Sprite='light_cavalry_horse', After=r2(t)), dict(Actor=person, Place=[end[0] - 1, end[1]], Face='NW', After=r2(t))]
        s.away.discard(person)
        s.pos[person] = (end[0] - 1, end[1])
    g += [dict(Actor='조조', Pose='bow', After=1.6), dict(Actor='하후돈', Play='action', After=1.7)]
    g += [dict(Actor=f'병사#{i}', Pose='prostrate', After=r2(1.7 + .08 * i)) for i in range(8)]
    g += [dict(Actor='헌제', Play='action', After=2.6), dict(Actor='대신#0', Emote='…', After=2.9), dict(Wait=4)]
    s.group(g)
    return s


# --- 13. 관도: 화살 아래로 달려오는 원소 기병, 궁병이 빠진 틈을 창병이 메운 진영 입구, 닿는 대로 싸우는 기병 ---------

def guandu():
    rnd = random.Random(13)
    F = 24  # the front rank, out on the field before the barricade
    s = Shot('guandu', 'S_title_guandu_field', (18, F + 4), radius=.5, time='deepnight', hold=.6)
    s.cams += [(18, -6), (18, 6)]
    even = list(range(4, 35, 2))
    odd = list(range(5, 34, 2))
    # the camp mouth, the palisade near the circle's rim and the two braziers before the files are the map's own
    # (guandu_camp.py), so the night darkens them; nobody stands or rides where a brazier burns
    BRAZIER_U = (12, 24)
    s.ambient += [dict(Effect='bonfire', At=[u, F - 1.2]) for u in BRAZIER_U]  # the braziers blaze high: the brightest light here
    archers = {u: s.actor(f'궁병#{u}', (u, F - 2), 'NE', 'archer', 'wei') for u in range(4, 35)}
    front_rows = {k: {u: s.actor(f'창병#{k}a{u}', (u, F + 4 * k), 'NE', 'pikeman', 'wei') for u in even} for k in range(4)}
    back_rows = {k: {u: s.actor(f'창병#{k}b{u}', (u, F + 4 * k + 2), 'NE', 'pikeman', 'wei') for u in even[:-1]} for k in range(4)}
    s.actor('cam#1', (18, F + 4), 'NE', 'light_infantry', 'wei', Off=True)
    s.ambient.append(dict(Follow='cam#1'))
    sp = 1.0
    per = STEP / sp
    riders = []
    for u in range(4, 35):
        if u in BRAZIER_U:
            continue
        v0 = -20 - rnd.randint(0, 6)
        name = '안량' if u == 19 else f'원소기병#{len(riders)}'
        riders.append(dict(key=s.actor(name, (u, v0), 'SW', 'heavy_cavalry' if name == '안량' else 'light_cavalry', 'yuan', Off=True),
                           u=u, v0=v0, after=rnd.uniform(0, .5)))
    at = lambda r, t: (r['u'], r['v0'] + max(0, t - r['after']) / per)
    # the camera follows the arrows out over the plain, rides along above the charge, then runs ahead back to the line
    g = [dict(Actor='cam#1', Move=[18, -6], Ease='line', Speed=4, After=.5, Free=True),  # out with the arrows to the charge
         dict(Actor='cam#1', Move=[18, 6], Ease='line', Speed=1.0, After=3.4, Free=True),  # along with it
         dict(Actor='cam#1', Move=[18, F + 2], Ease='line', Speed=2.0, After=4.8, Free=True)]  # ahead of it, back to the line
    hits, drop_n = set(), 0
    for volley, t0 in enumerate((.2, 1.4, 2.6)):
        for u, a in archers.items():
            ts = t0 + rnd.uniform(0, .6)
            r = rnd.choice(riders)
            tl = ts + 2
            for _ in range(4):
                tl = ts + arrow_time((u, F - 2), at(r, tl))
            tgt = at(r, tl)
            g.append(dict(Actor=a, Play='attack', Arrow=[r2(tgt[0] + rnd.uniform(-1.5, 1.5)), r2(tgt[1] + rnd.uniform(-2, 2))], FireArrow=True,
                          After=r2(ts)))
            if volley and r['key'] not in hits and len(hits) < 6 and rnd.random() < .1 and -14 < tgt[1] < F - 14 and r['key'] != '안량':
                hits.add(r['key'])
                if drop_n < 4:  # the rider comes off; the horse runs on a few strides, then breaks away to the side
                    fallen = s.actor(f'낙마#{drop_n}', (0, 0), 'SW', 'light_infantry', 'yuan', Off=True)
                    g.append(dict(Actor=r['key'], Drop=fallen, Sprite='light_cavalry_horse', After=r2(tl)))
                    r['riderless'] = (round(tgt[1]) + 5, tl + 5 * per)
                    drop_n += 1
                else:
                    g.append(dict(Actor=r['key'], Fall=True, After=r2(tl)))
                    r['down'] = (r['u'], round(tgt[1]))
    # the archers go back through the empty files to the barricade, then on through its gaps out of sight
    for i, (u, a) in enumerate(archers.items()):
        lane = min((x for x in odd if x not in BRAZIER_U), key=lambda x: (abs(x - u), x))
        t = 6.8 + (0 if lane == u else .5) + .02 * i
        g.append(dict(Actor=a, Move=[lane, F + 15], Via=[[lane, F - 2]] if lane != u else [], Speed=1.8, After=r2(t), Free=True))
        s.pos[a] = (lane, F + 15)
        t += (abs(lane - u) + 17) * STEP / 1.8
        g.append(s.move(a, s.away_cell((lane, F + 15), (0, 1)), leave=True, past_own=True, Speed=1.8, After=r2(t)))
        s.away.add(a)
    # each line's back row steps up half a file across, then into the gaps of its front row: four solid lines, held there
    t1 = 9.4
    lines = {}
    for k in range(4):
        lines[k] = dict(front_rows[k])
        for u, p in back_rows[k].items():
            g += [dict(Actor=p, Move=[u + 1, F + 4 * k + 1], Ease='line', Speed=.9, After=r2(t1), End='NE'),
                  dict(Actor=p, Move=[u + 1, F + 4 * k], Speed=.9, After=r2(t1 + 1.3), End='NE')]
            s.pos[p] = (u + 1, F + 4 * k)
            lines[k][u + 1] = p
    for r in riders:
        k = r['key']
        if 'down' in r:
            end = r['down']
            s.fell(k)
        elif 'riderless' in r:
            end = (r['u'], r['riderless'][0])
        else:
            end = (r['u'], F - 1)
        g.append(s.move(k, end, Enter=[r['u'], r['v0']], straight=True, Speed=sp, After=r2(r['after'])))
        if 'riderless' in r:
            out = s.away_cell(end, (rnd.choice([-1, 1]), 0))
            g.append(dict(Actor=k, Move=list(out), Ease='line', Speed=sp, After=r2(r['riderless'][1] + .05), Free=True, _leave=True))
            s.pos[k] = out
            s.away.add(k)
        r['arrive'] = r['after'] + (F - 1 - r['v0']) * per
    # each rider fights the pike before him the moment he gets there: two or three blows and through to the next line,
    # where now and then a rider goes down instead
    end_all, second = 0, []
    for r in riders:
        k = r['key']
        if 'down' in r or 'riderless' in r:
            continue
        t = r['arrive'] + .1
        for line in range(2):
            p = lines[line].get(r['u'])
            if not p or p in s.down:
                break
            if line:
                g.append(dict(Actor=k, Move=[r['u'], F + 4 * line - 1], Speed=1.4, After=r2(t), End='SW'))
                s.pos[k] = (r['u'], F + 4 * line - 1)
                t += 4 * STEP / 1.4 + .1
                second.append(t)
            loser = k if line and rnd.random() < .2 + .2 * line else p
            more, t = fight(s, k, p, t, rnd.randint(2, 3), loser, rnd, gap=(.6, .9))
            g += more
            if loser == k:
                break
        end_all = max(end_all, t)
    # the shot closes as the first few riders through are at the second line: nothing starts after that
    arrivals = sorted(r['arrive'] for r in riders if 'down' not in r and 'riderless' not in r)
    cut = arrivals[len(arrivals) // 2] + 2.2  # while the riders and the pikes are hard at it at the front line
    s.extra['End'] = r2(cut)  # it darkens while they fight on
    g.append(dict(Wait=r2(cut)))
    s.group(g)
    return s


# --- 14~16. 삼고초려 ------------------------------------------------------------------------------------------------

def visit(n):
    s = Shot(f'visit{n}', 'S_title_vil_peace', (4, 3), radius=.38, hold=.4, **({'Snow': 1} if n == 2 else {}))
    s.actor('동자', (3, 3), 'SE')
    if n == 3:
        s.actor('제갈량', (2, 4), 'NE')
    start = {'유비': None, '관우': None, '장비': None}
    for k, stand, off in [('유비', (4, 3), (0, 0)), ('관우', (5, 2), (0, -1)), ('장비', (5, 4), (0, 1))]:
        far = s.away_cell((stand[0] + off[0], stand[1] + off[1]), (1, .55))
        s.actor(k, far, 'NW', Off=True)
        start[k] = (far, stand)
    s.group(dict(Wait=.5),
            *[s.move(k, stand, Enter=list(far), end='NW', After=r2(.3 * i)) for i, (k, (far, stand)) in enumerate(start.items())])
    if n == 1:
        s.group(dict(Actor='유비', Play='action'), dict(Actor='동자', Emote='…', After=.5), dict(Actor='동자', Look='NW', After=1))
        s.group(dict(Actor='유비', Emote='…'), dict(Actor='관우', Look='NE', After=.3), dict(Wait=.8))
    elif n == 2:
        s.group(dict(Actor='유비', Play='action'), dict(Actor='동자', Emote='…', After=.5))
        s.group(dict(Actor='장비', Emote='!!'), dict(Actor='장비', Look='SE', After=.3), dict(Actor='관우', Emote='…', After=.6),
                dict(Actor='유비', Emote='…', After=.9), dict(Wait=1))
    else:
        s.group(dict(Actor='유비', Play='action'), dict(Actor='동자', Look='NW', After=.4), dict(Actor='제갈량', Face='SE', After=1))
        s.group(dict(Actor='제갈량', Play='action'), dict(Actor='유비', Play='action', After=.4), dict(Actor='관우', Emote='…', After=.8),
                dict(Wait=1.6))
        return s
    s.group(*[s.move(k, far, leave=True, After=r2(.3 * i)) for i, (k, (far, stand)) in enumerate(start.items())],
            dict(Actor='동자', Look='SE', After=.6), dict(Wait=2.4))
    return s


def visit1():
    return visit(1)


def visit2():
    return visit(2)


def visit3():
    return visit(3)


# --- 17. 유비와 제갈량이 이야기한다 --------------------------------------------------------------------------

def talk():
    s = Shot('talk', 'S_house', (2, 2), radius=.2, time='night', hold=1, interior=True)
    s.actor('유비', (2, 2), 'NE')
    s.actor('제갈량', (2, 1), 'SW')
    s.group(dict(Wait=1))
    s.group(dict(Actor='제갈량', Play='action'), dict(Actor='제갈량', Emote='…', After=.4), dict(Wait=1.2))
    s.group(dict(Actor='유비', Emote='…'), dict(Wait=1))
    s.group(dict(Actor='제갈량', Play='action'), dict(Actor='제갈량', Emote='……', After=.3), dict(Wait=1.4))
    s.group(dict(Actor='유비', Emote='!'), dict(Actor='유비', Play='action', After=.6), dict(Wait=1.6))
    return s


# --- 18. 적벽: 부서진 선단이 불탄다 --------------------------------------------------------------------------

def chibi():
    rnd = random.Random(18)
    s = Shot('chibi', 'S_chibi_wreck', (-2, 9), radius=.46, time='night', hold=.3)
    s.cams += [(20, 9)]
    s.actor('cam#1', (-2, 9), 'SE', 'light_infantry', 'wei', Off=True)
    s.ambient.append(dict(Follow='cam#1'))
    fires = [(3.5, .5), (10, 0), (12, 4.5), (18, 7.5), (19, -1.5), (8.5, -2), (2, 9.5), (9, 10), (14.5, 10.5)]
    smoke = [(7, 2), (14, 1.5), (16.5, 4), (22, 8.5), (13.5, -1.5), (0, 6)]
    s.ambient += [dict(Effect='bigfire', At=list(at)) for at in fires] + [dict(Effect='smoke', At=list(at)) for at in smoke]
    deck = [c for c in ((u, v) for u in range(-2, 23) for v in range(6, 14)) if s.open(*c)]
    rnd.shuffle(deck)
    crew = []
    for i, at in enumerate(deck[:min(11, len(deck))]):
        crew.append(s.actor(f'조조군#{i}', at, rnd.choice(['SE', 'SW', 'NW', 'NE']), rnd.choice(FOOT + ['archer']), 'wei'))
    g = [dict(Actor='cam#1', Move=[20, 9], Ease='line', Speed=.42, Free=True)]
    for i, k in enumerate(crew):
        u, v = s.pos[k]
        mates = [c for c in deck if c != (u, v) and abs(c[0] - u) + abs(c[1] - v) == 1 and c not in s.taken()]
        kind = i % 4
        if kind == 0 and mates:
            g.append(dict(Actor=k, Patrol=list(mates[0]), After=r2(rnd.uniform(0, 2))))
        elif kind == 1:
            g.append(dict(Actor=k, Pose='pant', After=r2(rnd.uniform(0, 2))))
        elif kind == 2:
            g += [dict(Actor=k, Look=d, After=r2(t)) for d, t in zip(rnd.sample(['SE', 'SW', 'NW', 'NE'], 3), (1, 3.5, 6))]
        else:
            g.append(dict(Actor=k, Fall=True, After=r2(rnd.uniform(3, 8))))
            s.fell(k)
        g += [dict(Actor=k, Emote='!', After=r2(rnd.uniform(0, 9)))]
    g.append(dict(Wait=11))
    s.group(g)
    return s


# --- 19. 적벽: 불붙은 진영에서 북서로 달아나는 조조군, 남동·남서·북동에서 덮치는 손권군 ----------------------------

def shore():
    """The camp on the shore burning, 조조's men running north-west and 손권's coming on after them: the hindmost are
    the first caught, run down from behind or turning to fight and soon taken two, three, five to one, while the rest
    run on; more of 손권's come in from the south-west and the north-east. The circle runs north-west with the rout."""
    rnd = random.Random(20)
    V, U0, CS, T0 = -20, -2, .56, 3.6  # the rout's middle row, the circle's start (on the landing stages), its speed and when it sets off
    U1 = -42
    SHORE = -6  # the last row of land; the river and its landing stages lie south-west of it
    s = Shot('shore', 'S_title_chibi_shore', (U0, V + 10), radius=.48, time='night', hold=.4)
    per = STEP / CS
    run = math.hypot(U0 - U1, 10)  # the circle runs straight from (U0, V + 10) to (U1, V)
    along = lambda t: min(1, max(0, t - T0) * CS / STEP / run)
    cam_u = lambda t: U0 + (U1 - U0) * along(t)
    cam_at = lambda t: (cam_u(t), V + 10 * (1 - along(t)))
    s.cams += [(u, V) for u in range(U0 - 6, U1 - 1, -6)] + [(U1, V)]
    s.actor('cam#1', (U0, V + 10), 'NW', 'light_infantry', 'wu', Off=True)
    s.ambient.append(dict(Follow='cam#1'))
    g = [dict(Actor='cam#1', Move=[U1, V], Ease='line', Speed=CS, After=T0, Free=True)]
    # every third row a row of 조조's men runs along; the row before it is for 손권's men coming up beside them, the row
    # after it burns: men a row apart would stand one over another on the screen
    lanes = list(range(V - 9, V + 7, 3))
    fire_rows = tuple(v + 1 for v in lanes)
    fires = [(u + rnd.uniform(-2, 2), v) for v in fire_rows for u in range(U0 + 14 - 5 * (v % 3), U1 - 30, -15)]
    huts = [(-13, -11), (-6, -10), (-1, -8), (3, -8), (9, -9)]  # the camp's huts by the water
    huts += [(u, SHORE - 2 - (u // 5) % 2) for u in range(11, 34, 7)]  # and the stores along the shore, lighting the landing
    s.ambient += [dict(Effect='fire', At=[r2(u), v]) for u, v in fires] + [dict(Effect='bigfire', At=list(h)) for h in huts]
    fires += huts
    s.block(*fires)
    far = U1 - 42  # well out of sight to the north-west of where the circle stops
    sp = lambda c: c * STEP  # cells a second -> walking speed
    cao_n, wu_n = [0], [0]

    def cao(at):
        cao_n[0] += 1
        return s.actor(f'조조군#{cao_n[0]}', at, 'NW', rnd.choice(FOOT), 'wei')

    def wu(at, face='NW'):
        wu_n[0] += 1
        k = s.actor(f'오군#{wu_n[0]}', at, face, rnd.choice(FOOT + ['guards', 'light_infantry']), 'wu', Off=True)
        s.away.discard(k)
        return k

    def run_off(k, after, speed, enter=None):
        u, v = s.pos[k] if enter is None else enter
        kw = dict(Enter=list(enter)) if enter else {}
        st = dict(Actor=k, Move=[far, v], Ease='line', Speed=r2(speed), After=r2(after), Free=True, _leave=True, **kw)
        s.pos[k] = (far, v)
        s.away.add(k)
        return st

    stands = []  # (조조군, cell, from, kill time)
    finishers = []  # 오군 free after a kill: (key, when)
    for v in lanes:
        # the front man of the row runs for it and gets away; the hindmost is caught one way or another
        fu = U0 + rnd.randint(-14, -8)
        for _ in range(rnd.choice([2, 3, 3, 4])):  # the men ahead of him run for it, and stay ahead of the men after them
            if fu > U0 + 4:
                break
            front = cao((fu, v))
            g.append(run_off(front, rnd.uniform(1.2, 2.2), rnd.uniform(.59, .66)))
            fu += rnd.randint(3, 5)
        ru = max(fu, U0 + rnd.randint(1, 15))
        rear = cao((ru, v))
        kind = rnd.choices(['cut', 'stand', 'run'], [4, 6, 1])[0]
        r_after, r_speed = rnd.uniform(1.6, 3), rnd.uniform(.54, .6)
        rel = ru - U0
        enter_u = round(ru + (26 - rel) + rnd.randint(0, 6))
        if kind == 'cut':
            k = wu((enter_u, v))
            s.pos[k] = (enter_u, v)
            gap = enter_u - ru
            ks = rnd.uniform(1.3, 1.5)
            steps, t = chase(s, k, rear, (-1, 0), gap, kspeed=ks, vspeed=r_speed, vafter=r2(r_after), kafter=r2(rnd.uniform(0, 9)))
            steps[0]['Move'] = [far, v]
            g += steps
            finishers.append((k, t + .5))
            kmove, stop = steps[1], s.pos[k]
            for n in range(1 if rnd.random() < .4 else 2):  # more on his heels, a row aside so as not to stand on him
                eu = kmove['Enter'][0] + 1 + 2 * n
                p2 = wu((eu, v - 1))
                g.append(dict(Actor=p2, Enter=[eu, v - 1], Move=[stop[0] + 1 + 2 * n, v - 1], Ease='line',
                              Speed=kmove['Speed'], After=kmove['After']))
                s.pos[p2] = (stop[0] + 1 + 2 * n, v - 1)
                finishers.append((p2, t + .3))
        elif kind == 'run':  # he keeps ahead of the man after him, both out of sight
            g.append(run_off(rear, r_after, r_speed))
            k = wu((enter_u, v))
            g.append(run_off(k, r_after + rnd.uniform(.5, 2), r_speed - .03, enter=(enter_u, v)))
        else:  # he runs a while, then turns on the man close behind him
            T = r_after + rnd.uniform(1.5, 15)
            us = round(ru - (T - r_after) * r_speed / STEP)
            if not all(s.open(u, v) for u in range(us, ru + 1)):
                us = ru
            g.append(dict(Actor=rear, Move=[us, v], Ease='line', Speed=r2(r_speed), After=r2(r_after), End='SE'))
            T = r_after + (ru - us) * STEP / r_speed
            s.pos[rear] = (us, v)
            k = wu((enter_u, v))
            ks = rnd.uniform(1.1, 1.4)
            t_in = T + .3 - (enter_u - us - 1) * STEP / ks
            if t_in < 0:
                enter_u = round(us + 1 + (T + .3) * ks / STEP)
                s.cast[k]['At'] = [enter_u, v]
                t_in = 0
            g.append(dict(Actor=k, Enter=[enter_u, v], Move=[us + 1, v], Ease='line', Speed=r2(ks), After=r2(t_in), End='NW'))
            s.pos[k] = (us + 1, v)
            g.append(dict(Actor=rear, Emote=rnd.choice(['!', '!!']), After=r2(T - .2)))
            more, t_end = fight(s, k, rear, T + .45, rnd.randint(4, 7), rear, rnd, gap=(.6, .85))
            g += more
            stands.append(dict(key=rear, at=(us, v), main=k, T=T + .4, kill=t_end - .6, helpers=0))
            finishers.append((k, t_end))

    def gang(k, when_free, src=None, enter=None):
        """Sends 오군 k to one of the men still holding out, to get there before he falls; False if none."""
        for st in sorted(stands, key=lambda d: (d['helpers'], abs(d['at'][0] - (src or s.pos[k])[0]) + abs(d['at'][1] - (src or s.pos[k])[1]))):
            if st['helpers'] >= 3:
                continue
            spots = [c for c in s.free(st['at'], n=8) if max(abs(c[0] - st['at'][0]), abs(c[1] - st['at'][1])) == 1 and c[1] not in fire_rows and c[1] < SHORE - 6]
            for spot in spots:
                try:
                    mv = s.move(k, spot, Speed=1.2, end=facing(st['at'][0] - spot[0], st['at'][1] - spot[1]), **({'Enter': list(enter)} if enter else {}))
                except ValueError:
                    continue
                n = path_len(s, mv, enter or src)
                arrive = max(when_free, st['T'] + .3) + n * STEP / 1.2
                if arrive > st['kill'] - .5:
                    s.pos[k] = tuple(src or enter)
                    continue
                mv['After'] = r2(arrive - n * STEP / 1.2)
                st['helpers'] += 1
                out = [mv]
                tt = arrive - IMPACT
                while tt < st['kill'] - .6:
                    out.append(dict(Actor=k, Strike=st['key'], React=rnd.choice(['block', 'hit']), After=r2(tt)))
                    tt += rnd.uniform(.9, 1.3)
                return out, st['kill'] + .3
        return None, 0

    # those who cut their man down go on to the next fight close by, or on after the rest
    for k, t in finishers:
        src = s.pos[k]
        if src[0] == far:
            continue
        more, t2 = gang(k, t, src=src)
        if more:
            g += more
            t = t2
        g.append(run_off(k, t + rnd.uniform(.2, .6), rnd.uniform(.55, .58)))
    # first the boats: they come in over the river from the south-west, run up on the shore and stay there, and the
    # men leap off and make for the men holding out, or after the rest
    # boats already moored and the landing stages stand in the water: nobody rows in through them
    objs = json.loads((ROOT / 'assets/graphics/maps/S_title_chibi_shore.json').read_text())['objects']
    moored = {o[1] // 2 + d for o in objs if o[0].startswith(('boat', 'sbridge')) and o[2] // 2 > SHORE for d in (-1, 0, 1)}
    shore_u = [u for u in range(U0 - 6, U0 + 16) if s.cell(u, SHORE + 1) == '~' and s.open(u, SHORE) and (u, SHORE) not in s.taken()
               and u not in moored]
    for i, u in enumerate(shore_u[::2]):
        k = wu((u, 10), 'NE')
        t0 = r2(i % 3 * .5 + rnd.uniform(0, .4))
        boat = f'boat_v#{k}'
        g += [dict(Actor=k, Enter=[u, 10], Move=[u, SHORE], Ease='line', Speed=1.6, After=t0, _boat=True),
              dict(Prop=[boat, u, 10], After=t0), dict(Prop=[boat, k], After=r2(t0 + .02), _ride=True)]
        t_land = t0 + (10 - SHORE - 1) * STEP / 1.6
        g.append(dict(Prop=[boat, u, SHORE + 1.2], After=r2(t_land)))
        s.pos[k] = (u, SHORE)
        more, t2 = gang(k, t_land + (SHORE - (SHORE + 1)) * 0 + .4, src=(u, SHORE))
        if more:
            g += more
            g.append(run_off(k, t2 + rnd.uniform(.2, .6), rnd.uniform(.55, .58)))
        else:
            lane = min(lanes, key=lambda v: abs(v - (V + 6)))
            g.append(s.move(k, (far, lane), leave=True, Speed=r2(rnd.uniform(.8, 1)), After=r2(t_land + .3), Free=True))
            s.away.add(k)
    # more of them come running onto each man holding out: from behind, and from the south-west and the north-east
    for st in sorted(stands, key=lambda d: d['T']):
        dirs = [(1, 0)] + ([(0, 1)] if st['at'][1] >= V - 3 else []) + ([(0, -1)] if st['at'][1] <= V + 3 else [])
        for j in range(rnd.choice([3, 4, 4, 5])):
            a_t = st['T'] + .5 + .55 * j + rnd.uniform(0, .3)
            if a_t > st['kill'] - .7 or st['helpers'] >= 5:
                break
            d = rnd.choice(dirs)
            spots = [c for c in s.free(st['at'], n=8) if max(abs(c[0] - st['at'][0]), abs(c[1] - st['at'][1])) == 1 and c[1] not in fire_rows and c[1] < SHORE - 6]
            done = False
            for spot in spots:
                for dist in range(8, 40):
                    src = (spot[0] + d[0] * dist, spot[1] + d[1] * dist)
                    if not s.open(*src) or src in s.taken():
                        continue
                    k = wu(src, facing(-d[0], -d[1]))
                    s.pos[k] = src
                    try:
                        mv = s.move(k, spot, Enter=list(src), Speed=1.2, end=facing(st['at'][0] - spot[0], st['at'][1] - spot[1]))
                    except ValueError:
                        del s.cast[k]; s.pos.pop(k); wu_n[0] -= 1
                        continue
                    t_in = a_t - path_len(s, mv, src) * STEP / 1.2
                    if t_in < 0 or not s.outside(*src, cam=cam_at(t_in), margin=10):
                        del s.cast[k]; s.pos.pop(k); wu_n[0] -= 1
                        if t_in < 0:
                            break
                        continue
                    mv['After'] = r2(t_in)
                    g.append(mv)
                    tt = a_t - IMPACT
                    while tt < st['kill'] - .6:
                        g.append(dict(Actor=k, Strike=st['key'], React=rnd.choice(['block', 'hit']), After=r2(tt)))
                        tt += rnd.uniform(.9, 1.3)
                    g.append(run_off(k, st['kill'] + rnd.uniform(.4, .9), rnd.uniform(.55, .58)))
                    st['helpers'] += 1
                    done = True
                    break
                if done:
                    break
    g.append(dict(Wait=r2(run * per + T0)))
    s.extra['End'] = r2(run * per + T0 - .5)  # it darkens with the rout running on
    s.group(g)
    # anyone whose way takes him through a man of the other side stays out of it
    for _ in range(8):
        bad = {p.split(': ')[1].split(' ')[0].removesuffix("'s") for p in s.check() if 'passes through' in p or 'blow lands' in p}
        bad = {k for k in bad if k.startswith('오군') and any(st.get('Actor') == k and 'Enter' in st for st in s.groups[-1])}
        if not bad:
            break
        tied = lambda st: any(isinstance(x, str) and any(x == k or x.endswith('#' + k) for k in bad) for x in st.get('Prop') or [])
        s.groups[-1] = [st for st in s.groups[-1] if st.get('Actor') not in bad and st.get('Strike') not in bad and not tied(st)]
        for k in bad:  # and the boat he came in, with him
            s.cast.pop(k, None)
            s.pos.pop(k, None)
    return s


# --- 20. 조조가 북서쪽으로 달아난다 ---------------------------------------------------------------------------

def flight():
    rnd = random.Random(20)
    s = Shot('flight', 'S_title_long_field', (0, 60), radius=.44, time='dusk', hold=.3, Steps=.5)
    riders = [('조조', 'caocao', 0), ('장료', 'zhangliao', -1), ('허저', 'light_cavalry', 1), ('하후돈', 'royal_guard', 0)]
    g = []
    for i, (k, spr, dv) in enumerate(riders):
        start = (s.cam[0] + 30 + i * 1.5, s.cam[1] + dv + (2 if k == '하후돈' else 0))
        s.actor(k, start, 'NW', spr, 'wei', Off=True)
        g.append(s.move(k, (start[0] - 64, start[1]), Enter=list(start), straight=True, Speed=2.1, leave=True, After=r2(.4 * i)))
    for i in range(10):
        start = (s.cam[0] + 30 + 3 + i, s.cam[1] - 4 + i % 9)
        k = s.actor(f'패잔병#{i}', start, 'NW', 'light_infantry', 'wei', Off=True)
        if i in (3, 7):
            stop = (start[0] - 30 - i, start[1])
            g.append(s.move(k, stop, Enter=list(start), Speed=.6, After=r2(1.5 + .5 * i)))
            g.append(dict(Actor=k, Fall=True, After=r2(1.5 + .5 * i + walk_time(30 + i, .6))))
            s.fell(k)
        else:
            g.append(s.move(k, (start[0] - 70, start[1]), Enter=list(start), Speed=r2(rnd.uniform(.55, .8)), leave=True,
                            After=r2(1.5 + .5 * i)))
    s.group(g)
    return s


# --- 21. 화용도 ----------------------------------------------------------------------------------------------------

def huarong():
    rnd = random.Random(21)
    s = Shot('huarong', 'S_huarong_pass', (13, 6), radius=.46, time='dusk', hold=1)
    s.actor('관우', (11, 6), 'SE', 'guanyu', 'shu')
    hidden = {(u, v) for u in range(11, 16) for v in range(0, 5)} | {(15, v) for v in range(9, 14)}
    troops = []
    for u in range(2, 11):  # massed across the road, the flat to either side kept clear to step aside into
        for v in range(3, 11):
            if (u, v) in hidden or (u, v) == (11, 6) or not s.open(u, v):
                continue
            troops.append(s.actor(f'유비군#{u}_{v}', (u + rnd.uniform(-.1, .1), v + rnd.uniform(-.1, .1)), 'SE', rnd.choice(FOOT), 'shu'))
            s.pos[troops[-1]] = (u, v)
    cao = [('조조', 'caocao'), ('장료', 'zhangliao'), ('허저', 'light_cavalry'), ('하후돈', 'royal_guard')]
    road = [(17, 6), (19, 5), (19, 7), (21, 6)]
    for (k, spr), (u, v) in zip(cao, road):
        s.actor(k, (u + 26, v), 'NW', spr, 'wei', Off=True)
    g = [s.move(k, at, Enter=[at[0] + 26, at[1]], straight=True, Speed=.8, end='NW', After=r2(.4 * i)) for i, ((k, _), at) in enumerate(zip(cao, road))]
    g += [dict(Actor=k, Look=rnd.choice(['SE', 'NE']), After=r2(rnd.uniform(1, 5))) for k in rnd.sample(troops, 8)]
    s.group(g)
    s.group(dict(Actor='조조', Emote='!'), dict(Actor='장료', Emote='!', After=.2), dict(Actor='허저', Play='block', After=.4))
    for who, line in [('관우', '……'), ('조조', '……'), ('관우', '……'), ('조조', '……!')]:
        s.group(dict(Actor=who, Say=line))
    s.group(dict(Actor='관우', Emote='…'), dict(Wait=1.4))
    # 관우 turns his horse aside and his men clear the road; they stand watching the road and the riders on it
    g = [s.move('관우', (12, 10), end='NE', Speed=.7)]
    road = {(u2, v2) for u2 in range(-10, 40) for v2 in range(5, 9)}
    north = sorted((k for k in troops if s.pos[k][1] <= 6), key=lambda k: s.pos[k][1])
    south = sorted((k for k in troops if s.pos[k][1] > 6), key=lambda k: -s.pos[k][1])
    for group_, edge, face in ((north, 0, 'SW'), (south, 13, 'NE')):
        for i, k in enumerate(group_):
            u, v = s.pos[k]
            if (u, v) not in road and (v <= 1 or v >= 12):
                g.append(dict(Actor=k, Face=face, After=r2(rnd.uniform(.5, 1.5))))
                continue
            for dst in s.free((u, edge), n=12, avoid=hidden | road):
                try:
                    g.append(s.move(k, dst, end=face, Speed=1.2, After=r2(.3 + .06 * i)))
                    break
                except ValueError:
                    continue
    s.group(g)
    g = []
    for i, ((k, _), dv) in enumerate(zip(cao, (6, 5, 7, 6))):
        g.append(s.move(k, (-26, dv), straight=True, Speed=1.3, leave=True, After=r2(.6 * i)))
        s.away.add(k)
    g += [dict(Actor='관우', Look='SW', After=1), dict(Actor='관우', Look='NW', After=3.4), dict(Wait=5)]
    s.group(g)
    return s


SHOTS = [grain, rally, raid, oath, rescue, march, hulao, luoyang, gyeqiao, xuzhou, alarm, emperor, baimen, guandu, visit1, visit2, visit3,
         talk, chibi, shore, huarong]



def main(args):
    force = '--force' in args
    only = args[args.index('--only') + 1].split(',') if '--only' in args else None
    shots, bad = [], 0
    for f in SHOTS:
        if only and f.__name__ not in only:
            continue
        s = f()
        problems = s.check()
        for p in problems[:12]:
            print(f'{s.id}: {p}')
        if len(problems) > 12:
            print(f'{s.id}: … {len(problems) - 12} more')
        bad += len(problems)
        shots.append(s.json())
    if bad and not force:
        raise SystemExit(f'{bad} problems; nothing written (--force writes anyway)')
    OUT.parent.mkdir(parents=True, exist_ok=True)
    OUT.write_text(json.dumps({'Shots': shots}, ensure_ascii=False, indent=1) + '\n')
    print(OUT.relative_to(ROOT), len(shots), 'shots', sum(len(s['Cast']) for s in shots), 'actors')


if __name__ == '__main__':
    main(sys.argv[1:])
