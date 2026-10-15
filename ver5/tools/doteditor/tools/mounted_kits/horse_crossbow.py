"""노기병: the 궁기병 (faction hood rider on the 'dot' horse) with the 노병's crossbow instead of the bow.

Everything but the weapon follows mounted_kits/horse_archer.py (rider_pose override, quiver position, horse steady with a
small pitch in the shot). The weapon is mounted_kits/crossbow.py's mesh: held LEVEL like a rifle (stock along the aim, prod
across it), always spanned (prod bent back, string on the latch, bolt on the stock) except in the shot's release and
follow-through; the rear hand is on the trigger lever under the stock, the front hand under the fore-end.

idle/walk/hit: loaded, held at the chest; attack: low ready, raised aim, trigger pulled (string springs forward, bolt
flies with a streak), empty crossbow lowered; exhausted: loaded crossbow across the lap; block: loaded crossbow held
across the chest. A short bolt case replaces the quiver.
"""
import numpy as np

import grid3d_mounted as M
import mounted_kits.crossbow as X

U, A, B, Bl, Hd = M.U, M.A, M.B, M.Bl, M.Hd
m, V = U.m, U.V

ORIG_CAVALRY = U.cavalry  # captured before any kit's make() patches units.cavalry
OFF = V([0, .12, 1.2])    # units.cavalry places the rider (scaled .8) at this offset
SIZE = 1.25               # the crossbow is enlarged around its centre so it reads at the rider's .8 scale

# rider-space hands: rear = right hand (hands[0]) on the trigger, front = left hand (hands[1]) under the fore-end;
# one height each frame so the stock lies level; shifted a little to the left so it clears the horse's neck and ears
REAR, FRONT = [.12, -.3, 1.24], [.3, -.86, 1.24]
SHOT = [dict(z=1.02, lean=.04),                                         # low ready
        dict(z=1.42, lean=.16),                                         # raised, aiming
        dict(z=1.46, lean=.22, rear=.06, flight=1.9),                   # trigger pulled
        dict(z=1.0, lean=.08, front_z=.84, loaded=False)]               # empty, lowering
A.SPEAR['xbow_shot'] = dict(name='기마 석궁', effect='none', frames=[
    dict(hand=[REAR[0], REAR[1], s['z']], dir=[0, -1, 0], lean=s['lean'], rear=s.get('rear', 0)) for s in SHOT])


def pose_for(row, f, style):
    """Hand positions and state of one frame, rider space."""
    if style and style.startswith('block_'):
        k = A.SPEAR[style]['frames'][f]
        c = V(k['hand']) + V([.55, -.6, -.3])                          # in front of the chest, clear of the head
        d = V(k['dir'])
        axis = V([d[0], d[1] - .3, 0])                                  # held level across the body
        axis = V([1, 0, 0]) if np.linalg.norm(axis) < .1 else axis/np.linalg.norm(axis)
        return dict(rear=list(c - axis*.3), front=list(c + axis*.3), lean=k.get('lean', 0), loaded=True)
    if row == U.ATTACK:
        s = SHOT[f]
        return dict(rear=[REAR[0], REAR[1], s['z']], front=[FRONT[0], FRONT[1], s.get('front_z', s['z'])],
                    lean=s['lean'], loaded=s.get('loaded', True), flight=s.get('flight'))
    if row == 4:  # exhausted: loaded crossbow across the lap
        return dict(rear=[-.4, -.35, .85], front=[.35, -.6, .85], lean=None, loaded=True)
    return dict(rear=list(REAR), front=list(FRONT), lean=None, loaded=True)


def crossbow_cavalry(orig):
    def cavalry(row, f, style=None, kit='rider', coat='dot', weapon='none'):
        s = pose_for(row, f, style)
        captured = {}
        orig_pose = U.rider_pose

        def rider_pose(r, ff, k=None):
            p = orig_pose(r, ff, k)
            if s.get('lean') is not None:
                p['lean'] = s['lean']
            p['hands'] = [list(s['rear']), list(s['front'])]
            p['elbows'] = [list((X.SHOULDERS[0] + V(s['rear']))/2 + [-.22, .08, 0]),
                           list((X.SHOULDERS[1] + V(s['front']))/2 + [.12, 0, 0])]
            captured['p'] = p
            return p

        U.rider_pose = rider_pose
        try:
            res = orig(row, f, 'xbow_shot' if row == U.ATTACK and style is None else style, kit=kit, coat=coat,
                       weapon='none')
        finally:
            U.rider_pose = orig_pose
        p = captured['p']
        T = U.transform(p)
        bounce = [.08, 0, .14, .03][f] if row == 1 else 0
        off = OFF + V([0, 0, bounce])
        k = A.SPEAR[style or 'xbow_shot']['frames'][f] if row == U.ATTACK else None
        move = U.horse_move(k.get('rear', 0), k.get('shift', 0)) if k else (lambda v: v)
        world = lambda v: move(np.asarray(v)*.8 + off)  # noqa: E731
        m.meshes = []
        rear, front = T(p['hands'][0]), T(p['hands'][1])
        flight = s.get('flight')
        aim = X.crossbow(rear, front, loaded=s['loaded'] and flight is None)
        centre = (rear + front)/2
        m.meshes = [(centre + (v - centre)*SIZE, c) for v, c in m.meshes]
        effects = []
        if flight is not None:
            tail = front + aim*flight
            X.flying_bolt(tail, aim)
            hw, tw = world(front), world(tail)
            effects = U.bow_effects('crossbow', 2, hw, A.unit(tw - hw), tw)
        X.bolt_case(T)
        res['faces'] = list(res['faces']) + [(world(v), c) for v, c in m.meshes]
        res['effects'] = effects
        return res
    return cavalry


def make():
    cavalry = crossbow_cavalry(ORIG_CAVALRY)
    U.cavalry = cavalry  # blocks.rider calls units.cavalry
    block = Bl.rider(kit='rider', coat='dot', weapon='none')

    def build(row, f, style=None):
        if row == B.BLOCK:
            return block(f, Bl.SELECTED)
        return cavalry(row, f, style)
    return build, Hd.heads(Hd.HOOD, Hd.HOOD_CLOTH)


M.register('horse_crossbow', make)
