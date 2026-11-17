"""관우 (sheet guanyu, 2026-11-16): the 청룡언월도 rider the sheet was converted from (units.guanyu on the dot coat), with
a scene row `cleave` for 19장 연진 (관우 rides past 문추 at a gallop and hews him): a two-handed riding cleave.

The horse keeps galloping through the whole row (keyframe `gait`) while the glaive goes round far more than the
horse tilts: at the wind-up the forehand is up about 30 degrees and the blade hangs behind and below the rider
(about 210 degrees from forward through straight up), then it sweeps about 270 degrees over the top and down past
the horse's right shoulder as the forehand lands.

  0-1 wind-up: the glaive goes back over the shoulder, both hands on the shaft, the horse gathers
  2   peak (held): arms reaching back, blade behind and below, forehand up
  3-5 swing: over the top and forward, the trail grows from the peak (bright leading edge)
  6   impact (held): blade low in front past the shoulder, rider bent far forward, forehand landed
  7   follow-through: the trail fading
  8-11 recovery to the galloping carry
Twelve frames put the impact frame at the row's halfway mark, where the scene's Strike lands the blow
(gui.impactTime sums the first half of the frame times).

The sheet keeps its own hand-cleaned head: heads['sheet_head'] makes grid3d_mounted paste the idle frame's head
pixels (where the 2D head stamp lands) instead of the stamp, and heads['tags'] maps the guanyu keys (green cloth,
red skin, shaft, red tassel) onto the sheet's tags (calibrated against its idle and walk frames: robe = faction,
shaft = themeB, collar = themeC).

  grid.py rows guanyu cleave 12 <MS>  (once; the row's grid files are then rewritten by the converter)
  grid3d_mounted.py guanyu --unit guanyu --base none --anims cleave && grid.py import guanyu
Impact (frame 6) starts IMPACT_MS into the row.
"""
import math

import numpy as np

import grid3d_mounted as M

U, A, Hd = M.U, M.A, M.Hd
E = M.B.E
STYLE = 'glaive_cleave'
MS = [70, 70, 120, 50, 40, 40, 120, 70, 60, 60, 60, 70]
IMPACT = 6
IMPACT_MS = sum(MS[:IMPACT])


def key(theta, hand, lean, rear=0., shift=0., bob=0., side=-.12, gait=0., grip=.42):
    """theta: the glaive's angle (degrees) in the horse's frame from forward through straight up; the world angle
    adds the horse's rear (radians). grip: where the left hand holds the shaft, behind the right hand."""
    t = math.radians(theta)
    return dict(hand=hand, dir=[side, -math.cos(t), math.sin(t)], lean=lean, rear=rear, shift=shift, bob=bob,
                gait=gait, grip=grip)


Q = math.pi/2
FRAMES = [
    key(115, [-.5, .1, 1.75], .05, rear=.12, gait=0),
    key(160, [-.4, .35, 1.95], 0, rear=.55, gait=Q*.7, side=.35),
    key(180, [-.25, .5, 2.0], -.05, rear=.85, bob=.03, gait=Q*1.2, side=.75),
    key(125, [-.38, .2, 2.2], -.2, rear=.65, gait=Q*1.7, side=.25),
    key(60, [-.42, -.25, 2.2], -.25, rear=.35, shift=-.06, gait=Q*2.2),
    key(5, [-.42, -.7, 1.85], -.38, rear=.1, shift=-.14, gait=Q*2.7),
    key(-58, [-.42, -.9, 1.35], -.58, rear=-.06, shift=-.22, bob=-.06, gait=Q*3.2, side=-.18),
    key(-78, [-.4, -.75, 1.15], -.52, rear=-.04, shift=-.2, bob=-.05, gait=Q*4, side=-.25),
    key(-30, [-.45, -.5, 1.28], -.36, shift=-.14, gait=Q*5),
    key(15, [-.47, -.25, 1.4], -.2, shift=-.07, gait=Q*6),
    key(50, [-.48, -.1, 1.42], -.1, shift=-.02, gait=Q*7),
    key(70, [-.48, -.05, 1.4], -.06, gait=Q*8),
]
A.SPEAR[STYLE] = dict(name='언월도 말 위 내려베기', effect='arc', frames=FRAMES)


def flat(a, e, hand, twist, lean=0., rear=0., shift=0., bob=0., gait=0., stride=.3, grip=.42):
    """A key of the flat sweeps. a: the glaive's heading (degrees) from forward toward the rider's LEFT (+x; his right
    is -x), e: its elevation; hand: the right hand after the twist; twist: the shoulders turned (degrees, + to the
    left) on the hips, all of the turn from the chest up; stride: the planted horse's legs at this fraction of a stride."""
    t, h = math.radians(a), math.radians(e)
    return dict(hand=hand, dir=[math.sin(t)*math.cos(h), -math.cos(t)*math.cos(h), math.sin(h)], twist=math.radians(twist),
                lean=lean, rear=rear, shift=shift, bob=bob, gait=gait, stride=stride, grip=grip)


# slash_l: a flat two-handed sweep from his left to his right at a mounted man's chest. Wound up far back over the
# left shoulder (shoulders turned 45 degrees left, the forehand lifting a little as the horse gathers), the blade goes
# round about 200 degrees across the front at chest height, hits ahead and a little right of the horse's head and
# runs out past the right side (shoulders turned right, the horse's weight forward), then comes back to the carry.
# slash_r is the same blow from the other side. Frame 6 (the impact) starts at the row's halfway mark.
SLASH_MS = [70, 80, 80, 50, 40, 40, 120, 70, 60, 60, 70, 80]
CARRY = dict(key(70, [-.48, -.05, 1.4], -.06, gait=Q*4), stride=.1)
SLASH_L = [
    flat(25, 62, [.05, -.25, 1.6], 12, gait=Q*.6),
    flat(115, 30, [.3, .0, 1.68], 32, lean=.05, rear=.06, bob=.02, gait=Q*.9),
    flat(165, 18, [.38, .22, 1.62], 48, lean=.1, rear=.12, bob=.03, gait=Q*1.1),
    flat(125, 4, [.36, -.08, 1.52], 38, lean=.02, rear=.08, gait=Q*1.2),
    flat(75, -4, [.25, -.42, 1.46], 20, lean=-.1, rear=.02, shift=-.03, gait=Q*1.5),
    flat(25, -7, [.04, -.62, 1.42], 4, lean=-.2, shift=-.07, gait=Q*1.9),
    flat(-30, -9, [-.22, -.66, 1.4], -16, lean=-.28, rear=-.05, shift=-.12, bob=-.03, gait=Q*2.4),
    flat(-85, -6, [-.48, -.4, 1.42], -36, lean=-.2, rear=-.04, shift=-.1, bob=-.02, gait=Q*2.8),
    flat(-105, -4, [-.58, -.12, 1.46], -46, lean=-.1, rear=-.02, shift=-.06, gait=Q*3.1),
    flat(-80, 26, [-.52, -.15, 1.52], -30, lean=-.06, shift=-.03, gait=Q*3.4),
    flat(-50, 52, [-.48, -.12, 1.5], -12, gait=Q*3.7),
    CARRY,
]
SLASH_R = [
    flat(-45, 55, [-.5, -.2, 1.62], -12, gait=Q*.6),
    flat(-75, 40, [-.6, -.02, 1.68], -32, lean=.05, rear=.06, bob=.02, gait=Q*.9),
    flat(-108, 8, [-.68, .1, 1.58], -50, lean=.1, rear=.12, bob=.03, gait=Q*1.1),
    flat(-85, 2, [-.58, -.2, 1.5], -38, lean=.02, rear=.08, gait=Q*1.2),
    flat(-45, -5, [-.4, -.5, 1.45], -20, lean=-.1, rear=.02, shift=-.03, gait=Q*1.5),
    flat(0, -7, [-.16, -.68, 1.42], -4, lean=-.2, shift=-.07, gait=Q*1.9),
    flat(48, -9, [.12, -.66, 1.42], 16, lean=-.28, rear=-.05, shift=-.12, bob=-.03, gait=Q*2.4),
    flat(95, -6, [.34, -.42, 1.46], 36, lean=-.2, rear=-.04, shift=-.1, bob=-.02, gait=Q*2.8),
    flat(135, 2, [.4, -.14, 1.5], 46, lean=-.1, rear=-.02, shift=-.06, gait=Q*3.1),
    flat(110, 28, [.25, -.2, 1.55], 30, lean=-.06, shift=-.03, gait=Q*3.4),
    flat(15, 60, [-.2, -.2, 1.55], 6, gait=Q*3.7),
    CARRY,
]
ROWS = {
    'cleave': dict(style=STYLE, frames=FRAMES, swing=(2, 6), trail={3: 1.0, 4: 1.0, 5: 1.0, 6: 1.0, 7: .45}),
    # edge: the blade's edge leads the sweep (+1 toward his left, -1 toward his right)
    'slash_l': dict(style='glaive_slash_l', frames=SLASH_L, swing=(2, 6), trail={3: 1.0, 4: 1.0, 5: 1.0, 6: 1.0, 7: .5},
                    edge=-1, inner=.2, drop=.4, reach=2.4),
    'slash_r': dict(style='glaive_slash_r', frames=SLASH_R, swing=(2, 6), trail={3: 1.0, 4: 1.0, 5: 1.0, 6: 1.0, 7: .5},
                    edge=1, inner=.2, drop=.4, reach=2.4),
}
A.SPEAR['glaive_slash_l'] = dict(name='언월도 왼쪽에서 가로베기', effect='arc', frames=SLASH_L)
A.SPEAR['glaive_slash_r'] = dict(name='언월도 오른쪽에서 가로베기', effect='arc', frames=SLASH_R)
STYLE_ROW = {spec['style']: row for row, spec in ROWS.items()}
INNER, REACH = .45, 2.15  # the band from the middle of the shaft to past the blade tip: a wide crescent
# the sheet's attack trail is effect1 only: a 66 body and an aa edge; the leading edge is drawn opaque
TRAIL_SHADES = {'effect3': ('effect1', '66'), 'effect2': ('effect1', 'aa'), 'effect1': ('effect1', 'aa')}
TAGS = {'G': 'l', 'H': 'k', 'I': 'k', 'J': 'j', 'K': 'e', 'L': 'd', 'X': 'k', 'Y': 'n', 'Z': 'j',
        'f': 'r', 'g': 'q', 'h': 'q', 'b': 'p', 'c': 'p', 'k': 't', 'l': 's', 'm': 's', 't': 'A',
        'u': 'k', 'v': 'k', 'w': 'j'}


def world_points(k, inner=INNER, drop=0., reach=REACH):
    """(hand, inner, tip) of the glaive in world space, the band reaching from mid-shaft to past the tip. drop lowers
    the band's inner edge: a flat sweep's band then slopes like a shallow cone and keeps its width from above."""
    hand, _, _ = U.spear_points(k, 1, 'glaive')
    move = U.horse_move(k.get('rear', 0), 0)
    d = move(A.unit(k['dir'])) - move(np.zeros(3))
    return hand, hand + d*inner - [0, 0, drop], hand + d*reach


def smear(layer, fx, to_px):
    """Crescent smear: a translucent body over the whole arc, the outer rim brighter, the newest part solid."""
    pts = [(to_px(a), to_px(b)) for a, b in fx['pairs']]
    lead = fx['lead']
    alpha = fx['alpha']

    def band(sub, frac):
        def paint(dr, s):
            for (a0, b0), (a1, b1) in zip(sub, sub[1:]):
                i0 = (a0[0] + (b0[0] - a0[0])*frac, a0[1] + (b0[1] - a0[1])*frac)
                i1 = (a1[0] + (b1[0] - a1[0])*frac, a1[1] + (b1[1] - a1[1])*frac)
                dr.polygon(E.scaled([i0, b0, b1, i1], s), fill=255)
        return paint

    def rim(sub, width):
        return lambda dr, s: dr.line(E.scaled([b for _, b in sub], s), fill=255, width=s*width)
    layer.fill(layer.mask(band(pts, 0)), 'p', 90*alpha)
    layer.fill(layer.mask(band(pts, .55)), 'q', 150*alpha)
    layer.fill(layer.mask(rim(pts, 2)), 'q', 150*alpha)
    if lead and alpha >= 1:
        sub = pts[-lead:]
        layer.fill(layer.mask(band(sub, .3)), 'q', 150)
        layer.fill(layer.mask(rim(sub, 2)), 'r', 255)


E.DRAW['smear'] = smear


def swing_effects(row, f):
    """The smear of a row's swing: from its peak to its impact, then only the end of the arc lingers."""
    spec = ROWS[row]
    trail, (a, b), frames = spec['trail'], spec['swing'], spec['frames']
    if f not in trail:
        return []
    keys = A.sample(frames[a:min(f, b) + 1], 6, U.lerp_key)
    if f > b:
        keys = keys[-8:] + A.sample(frames[b:f + 1], 4, U.lerp_key)[1:]
    pairs = [world_points(k, spec.get('inner', INNER), spec.get('drop', 0.), spec.get('reach', REACH))[1:] for k in keys]
    return [dict(kind='smear', pairs=pairs, alpha=trail[f], lead=7, layer='under')]


def twist_weight(z):
    """How much of the shoulder turn a point of the rider at height z (pose space) takes: none at the hips."""
    return min(1., max(0., (z - .8)/.6))


def turn(x, y, a):
    c, s = math.cos(a), math.sin(a)
    return x*c - y*s, x*s + y*c


def twisted(transform):
    """transform(p) with the chest and arms turned by p['twist'] about the spine (the sweeps' wind-up and follow-through)."""
    def T(p):
        inner, tw = transform(p), float(p.get('twist', 0))
        if not tw:
            return inner

        def at(q):
            x, y, z = q
            x, y = turn(x, y, tw*twist_weight(z + p['bob']))
            return inner([x, y, z])
        return at
    return T


def two_handed(rider_pose):
    """The left hand leaves the reins for the shaft, `grip` behind the right hand (pose space from world space).
    A twisting key gives the right hand where it ends up after the twist."""
    def pose(row, f, k=None):
        p = rider_pose(row, f, k)
        if k is None or 'grip' not in k:
            return p
        tw = float(k.get('twist', 0))
        if tw:
            p['twist'] = tw
            hx, hy, hz = (float(v) for v in k['hand'])
            x, y = turn(hx, hy, -tw*twist_weight(hz))
            p['hands'][0] = [x, y, hz - p['bob']]
            p['elbows'][0] = ((U.SHOULDER + np.array(p['hands'][0]))/2 + [-.18, 0, -.06]).tolist()
        hand = U.transform(p)(p['hands'][0])*.8 + U.RIDER_OFFSET
        w = (hand - A.unit(k['dir'])*float(k['grip']) - U.RIDER_OFFSET)/.8
        x, y = float(w[0]), float(w[1] - p['lean']*(w[2] - .45))
        x, y = turn(x, y, -tw*twist_weight(w[2]))
        q = [x, y, float(w[2] - p['bob'])]
        p['hands'][1] = q
        p['elbows'][1] = ((np.array([.4, 0, 1.42]) + q)/2 + [.16, 0, -.06]).tolist()
        return p
    return pose


SWEEP = {'row': None}  # the row being built: its blade turns its edge into the swing


def edge_leading(glaive_at):
    """In the cleave the blade is turned halfway toward its edge leading the swing (forward and down over the top), so
    it keeps some width in every view instead of going edge-on behind the rider. In the flat sweeps the edge leads
    round the rider, the blade tipped up a little so the iso camera still sees its face."""
    def at(hand, d, side):
        row = SWEEP['row']
        if row == 'cleave':
            side = A.unit(A.unit(np.cross([1, 0, 0], d)) + A.unit(np.cross(d, [0, 0, 1])))
        elif row:
            side = A.unit(A.unit(np.cross([0, 0, 1], d))*ROWS[row]['edge'] + np.array([0, 0, .45]))
        return glaive_at(hand, d, side)
    return at


def make():
    U.glaive_at = edge_leading(U.glaive_at)
    U.transform = twisted(U.transform)
    effects, pose = U.spear_effects, U.pose
    U.pose = lambda row, f: pose(row, min(f, 3))  # the scene rows have 12 frames; the base pose table has four per row
    U.rider_pose = two_handed(U.rider_pose)
    U.spear_effects = lambda style, f, weapon: (swing_effects(STYLE_ROW[style], f) if style in STYLE_ROW
                                                else effects(style, f, weapon))
    inner = M.rider_build('guanyu', weapon='glaive')

    def build(row, f, style=None):
        if row not in ROWS:
            SWEEP['row'] = None
            return inner(row, f, style)
        SWEEP['row'] = row if row != 'cleave' or 1 <= f <= 8 else None
        return inner(U.ATTACK, f, ROWS[row]['style'])
    heads = Hd.heads(Hd.GUANYU, skin='KL', lid='K')
    heads.update(sheet_head=True, tags=TAGS, trail=TRAIL_SHADES)
    return build, heads


M.register('guanyu', make)
