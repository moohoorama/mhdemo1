"""궁기병: the 경기병 rider (faction hood, steel shoulders and shins) on the same horse, with a pale-wood bow and a
quiver on the back instead of the spear.

idle/walk/hit hold the bow upright in the left hand with an arrow nocked and the right hand at the chest;
attack is units.archer's 'power' shot (gather, draw, release with the arrow in flight and a streak, follow-through);
exhausted lets the bow hang; block holds the bow across the body in both hands.
"""
import math

import numpy as np

import grid3d_mounted as M
import attacks as A
import base_model as m
import blocks as Bl
import units as U

V = m.V
WOOD, STRING = '#d7bc7c', '#ede1bd'  # pale wood: its ramp keys A B C map to themeB; string: iron highlight
SHOULDER_R, SHOULDER_L = V([-.4, 0, 1.42]), V([.4, 0, 1.42])
OFF = V([0, .12, 1.2])  # units.cavalry places the rider (scaled .8) at this offset

# rider-space hand positions: draw = right hand (hands[0]) at the string, bow = left hand (hands[1]) on the grip
IDLE = dict(draw=[.12, -.3, 1.22], bow=[.5, -.95, 1.25], lean=None, bend=1.2, arrow=True)  # lean None: keep the walk/hit lean
# units.attacks.BOW['power'] on a seated rider: gather, draw to the cheek, release (arrow in flight), follow through
SHOT = [
    dict(draw=[.15, -.35, 1.42], bow=[.5, -.95, 1.42], lean=.04, bend=1.0, arrow=True),
    dict(draw=[.1, .25, 1.5], bow=[.52, -1.0, 1.46], lean=.2, bend=1.7, arrow=True, rear=.04),
    dict(draw=[.05, .3, 1.58], bow=[.5, -1.0, 1.46], lean=.24, bend=1.0, flight=2.2, rear=.06),
    dict(draw=[.1, .1, 1.36], bow=[.5, -.95, 1.4], lean=.1, bend=1.0)]
A.SPEAR['bow_shot'] = dict(name='기마사격', effect='none', frames=[
    dict(hand=s['draw'], dir=[0, -1, 0], lean=s['lean'], rear=s.get('rear', 0)) for s in SHOT])


def bow(c, axis, belly, nock, bend, half=.8, width=.09):
    """Pale-wood bow centred on c: limbs bulge toward belly, the string runs from the tips to the nock point."""
    pts = [c + belly*.22*bend*math.sin(math.pi*t) + axis*half*(1 - 2*t) for t in np.linspace(0, 1, 13)]
    for a, b in zip(pts, pts[1:]):
        m.rod(a, b, width, WOOD)
    for end in (pts[0], pts[-1]):
        m.rod(end, nock, .035, STRING)


def arrow(tail, d):
    head = tail + d*1.1
    m.rod(tail, head, .04, '#b99553')
    m.poly([head + d*.2, head + V([-.09, 0, 0]), head + V([.09, 0, 0])], '#bacbd6')
    m.poly([head + V([0, 0, -.09]), head + V([0, 0, .09]), head + d*.2], '#bacbd6')
    for dx in (-.07, .07):
        m.poly([tail, tail + V([dx, 0, 0]) + d*.12, tail + d*.26], WOOD)


def pose_for(row, f, style):
    """The bow pose of one frame, rider space."""
    if style and style.startswith('block_'):
        k = A.SPEAR[style]['frames'][f]
        axis = A.unit(V(k['dir']) + V([0, -.6, .55]))  # tilted up and forward so it shows from the side views too
        c = V(k['hand']) + V([.55, -.6, -.3])  # held out in front of the chest, clear of the head
        return dict(block=True, c=c, axis=axis, draw=list(c - axis*.35), bow=list(c + axis*.35), bend=1.0,
                    lean=k.get('lean', 0), sway=k.get('sway', 0))
    if row == U.ATTACK:
        return dict(SHOT[f])
    if row == 4:  # exhausted: the bow hangs from the left hand, the right hand slumps
        return dict(draw=[-.48, -.38, .85], bow=[.5, -.4, .8], lean=None, bend=1.0, hang=True)
    return dict(IDLE)


def archer_cavalry(orig):
    def cavalry(row, f, style=None, kit='rider', coat='dot', weapon='none'):
        s = pose_for(row, f, style)
        captured = {}
        orig_pose = U.rider_pose

        def rider_pose(r, ff, k=None):
            p = orig_pose(r, ff, k)
            if s.get('lean') is not None:
                p['lean'] = s['lean']
            p['hands'] = [list(s['draw']), list(s['bow'])]
            p['elbows'] = [list((SHOULDER_R + V(s['draw']))/2 + [-.22, .08, 0]),
                           list((SHOULDER_L + V(s['bow']))/2 + [.12, 0, 0])]
            captured['p'] = p
            return p

        U.rider_pose = rider_pose
        try:
            res = orig(row, f, 'bow_shot' if row == U.ATTACK and not s.get('block') else style, kit=kit, coat=coat,
                       weapon='none')
        finally:
            U.rider_pose = orig_pose
        p = captured['p']
        T = U.transform(p)
        bounce = [.08, 0, .14, .03][f] if row == 1 else 0
        off = OFF + V([0, 0, bounce])
        k = A.SPEAR[style if s.get('block') else 'bow_shot']['frames'][f] if row == U.ATTACK else None
        move = U.horse_move(k.get('rear', 0), k.get('shift', 0)) if k else (lambda v: v)
        world = lambda v: move(np.asarray(v)*.8 + off)  # noqa: E731
        m.meshes = []
        hb, hd = T(p['hands'][1]), T(p['hands'][0])
        belly = V([0, -1, 0])
        effects = []
        if s.get('block'):
            bow(T(s['c']), s['axis'], belly, T(s['c']) + V([0, .5, 0]), s['bend'])
        elif s.get('hang'):
            belly = A.unit([.6, -.8, 0])
            bow(hb, A.unit([0, -.3, -.95]), belly, hb - belly*.3, 1.0)
        else:
            aim = A.unit(hb - hd)
            axis = A.unit([0, -.2, .98])
            bow(hb, axis, belly, hd, s['bend'])
            flight = s.get('flight')
            if s.get('arrow'):
                arrow(hd, aim)
            elif flight:
                arrow(hb + aim*flight, aim)
                hw, tw = world(hb), world(hb + aim*flight)
                effects = U.bow_effects('power', f, hw, A.unit(tw - hw), tw)
        # quiver slung diagonally across the back
        q0, q1 = T([-.3, .34, .85]), T([-.05, .4, 1.7])
        m.rod(q0, q1, .15, '#704022')
        for t in (.15, .5, .85):
            m.rod(q0 + (q1 - q0)*t, q0 + (q1 - q0)*(t + .02), .17, U.BROWN)
        for dx in (-.07, 0, .07):
            m.rod(q1 + V([dx, 0, 0]), q1 + V([dx*1.6, .02, .32]), .035, WOOD)
        res['faces'] = list(res['faces']) + [(world(v), c) for v, c in m.meshes]
        res['effects'] = effects
        return res
    return cavalry


def make():
    cavalry = archer_cavalry(U.cavalry)
    U.cavalry = cavalry  # blocks.rider calls units.cavalry
    block = Bl.rider(kit='rider', coat='dot', weapon='none')

    def build(row, f, style=None):
        if row == M.B.BLOCK:
            return block(f, Bl.SELECTED)
        return cavalry(row, f, style)
    return build, M.Hd.heads(M.Hd.HOOD, M.Hd.HOOD_CLOTH)


M.register('horse_archer', make)
