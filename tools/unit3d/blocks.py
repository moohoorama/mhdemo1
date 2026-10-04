"""Block (막기) motion: the defender's reaction when an attack misses. Four candidate
styles per weapon family; the user picks one (SELECTED), the others stay as backups.

Frames follow the hit row's beat: 1 ready, 2 brace, 3 the blow meets the guard
(or slips past), 4 recover. Keyframes use the attack keys (hand, elbow, blade/dir,
lean, bob, crouch, lunge; rider: rear, shift) plus:
  off   - off hand (shield / bow hand) position
  sway  - whole body leans to the side (+x), for dodging
  back  - whole body slides back (+y, away from the attacker)
No effect layers: the clash spark is a runtime effect.
"""
import copy
import numpy as np
import attacks as A
import units as U

V = U.V

# foot units with a one-handed weapon (sword, fan, staff; the shield sits on the off hand)
SWORD = {
    'guard': dict(name='막아서기', frames=[
        dict(hand=[-.45, -.45, 1.55], blade=[1, -.1, .15], off=[.4, -.55, 1.25], lean=.02),
        dict(hand=[-.42, -.5, 1.6], blade=[1, -.05, .2], off=[.3, -.62, 1.32], crouch=.6, lean=.06, bob=-.06),
        dict(hand=[-.38, -.42, 1.62], blade=[1, 0, .25], off=[.28, -.52, 1.34], crouch=.8, lean=.22, bob=-.08, back=.12),
        dict(hand=[-.45, -.45, 1.45], blade=[.9, -.2, .3], off=[.4, -.5, 1.2], crouch=.3, lean=.1)]),
    'parry': dict(name='쳐내기', frames=[
        dict(hand=[-.6, -.3, 1.2], blade=[-.3, -.6, .75], lean=.04),
        dict(hand=[-.2, -.65, 1.3], blade=[.5, -.6, .6], crouch=.5, lean=.1),
        dict(hand=[-.85, -.3, 1.5], blade=[-.85, -.25, .45], crouch=.4, lean=.18, back=.06),
        dict(hand=[-.65, -.3, 1.25], blade=[-.4, -.5, .75], lean=.08)]),
    'sidestep': dict(name='몸 비켜 피하기', frames=[
        dict(hand=[-.62, -.35, 1.08], blade=[-.05, -.32, .94], lean=.02),
        dict(hand=[-.66, -.25, 1.1], blade=[-.2, -.3, .9], crouch=.4, lean=.06, sway=.12),
        dict(hand=[-.75, -.05, 1.0], blade=[-.5, -.2, .8], off=[.75, -.05, 1.25], crouch=.6, lean=.12, sway=.34),
        dict(hand=[-.66, -.25, 1.08], blade=[-.2, -.3, .9], crouch=.2, lean=.04, sway=.1)]),
    'backstep': dict(name='뒤로 물러서기', frames=[
        dict(hand=[-.6, -.4, 1.2], blade=[0, -.5, .85], lean=.06),
        dict(hand=[-.55, -.45, 1.35], blade=[.2, -.6, .75], off=[.42, -.5, 1.3], crouch=.5, lean=.2),
        dict(hand=[-.5, -.4, 1.45], blade=[.3, -.6, .7], off=[.4, -.5, 1.38], lunge=-.6, lean=.38, bob=.04, back=.3),
        dict(hand=[-.58, -.4, 1.25], blade=[.1, -.55, .8], lunge=-.3, lean=.15, back=.2)]),
}

# mounted (rider space; dir = weapon direction)
MOUNT = {
    'guard': dict(name='막아서기', frames=[
        dict(hand=[-.45, -.2, 1.6], dir=[1, 0, .15]),
        dict(hand=[-.4, -.25, 1.7], dir=[1, 0, .2], lean=.1, rear=.05),
        dict(hand=[-.35, -.2, 1.75], dir=[1, 0, .25], lean=.25, rear=.15, shift=.15),
        dict(hand=[-.45, -.2, 1.55], dir=[.95, 0, .3], lean=.1)]),
    'parry': dict(name='쳐내기', frames=[
        dict(hand=[-.5, 0, 1.3], dir=[0, -.6, .8]),
        dict(hand=[-.2, -.5, 1.35], dir=[.6, -.6, .5], lean=.1),
        dict(hand=[-.8, -.1, 1.5], dir=[-.9, -.2, .4], lean=.2, rear=.1),
        dict(hand=[-.55, 0, 1.35], dir=[-.3, -.5, .8], lean=.08)]),
    'sidestep': dict(name='몸 비켜 피하기', frames=[
        dict(hand=[-.48, .1, 1.3], dir=[0, -.35, .94]),
        dict(hand=[-.48, .1, 1.3], dir=[0, -.3, .95], lean=.08, sway=.12),
        dict(hand=[-.5, .15, 1.25], dir=[-.2, -.2, .95], lean=.15, sway=.34),
        dict(hand=[-.48, .1, 1.3], dir=[0, -.3, .95], lean=.05, sway=.1)]),
    'backstep': dict(name='뒤로 물러서기', frames=[
        dict(hand=[-.48, .1, 1.35], dir=[0, -.4, .9], lean=.06),
        dict(hand=[-.45, .2, 1.45], dir=[0, -.2, 1], lean=.2, rear=.15, shift=.15),
        dict(hand=[-.42, .3, 1.55], dir=[0, .1, 1], lean=.35, rear=.35, shift=.35),
        dict(hand=[-.48, .15, 1.4], dir=[0, -.3, .95], lean=.15, rear=.1, shift=.2)]),
}

# bow: draw = string hand, off = bow hand, axis = bow limb direction
BOW = {
    'guard': dict(name='막아서기', frames=[
        dict(hand=[-.3, -.7, 1.3], off=[.02, -.82, 1.32], axis=[.3, -.15, .95]),
        dict(hand=[-.28, -.74, 1.38], off=[0, -.86, 1.4], axis=[.32, -.1, .95], crouch=.6, lean=.06, bob=-.06),
        dict(hand=[-.26, -.7, 1.42], off=[-.02, -.82, 1.45], axis=[.35, -.05, .94], crouch=.8, lean=.22, bob=-.08, back=.12),
        dict(hand=[-.3, -.7, 1.3], off=[.02, -.82, 1.32], axis=[.3, -.15, .95], crouch=.3, lean=.1)]),
    'parry': dict(name='쳐내기', frames=[
        dict(hand=[.05, -.22, 1.18], off=[.12, -.62, .97], axis=[0, -.5, .87]),
        dict(hand=[-.1, -.4, 1.3], off=[-.25, -.6, 1.3], axis=[.7, -.3, .65], crouch=.5, lean=.1),
        dict(hand=[.1, -.3, 1.3], off=[.75, -.35, 1.45], axis=[-.4, -.5, .75], crouch=.4, lean=.18, back=.06),
        dict(hand=[.05, -.25, 1.2], off=[.3, -.6, 1.1], axis=[0, -.5, .87], lean=.08)]),
    'sidestep': dict(name='몸 비켜 피하기', frames=[
        dict(hand=[.05, -.22, 1.18], off=[.12, -.62, .97], axis=[0, -.5, .87]),
        dict(hand=[.05, -.2, 1.18], off=[.15, -.6, 1.0], axis=[0, -.5, .87], crouch=.4, lean=.06, sway=.12),
        dict(hand=[-.2, -.1, 1.1], off=[.6, -.3, 1.2], axis=[.3, -.5, .8], crouch=.6, lean=.12, sway=.34),
        dict(hand=[.05, -.2, 1.18], off=[.15, -.6, 1.0], axis=[0, -.5, .87], crouch=.2, lean=.04, sway=.1)]),
    'backstep': dict(name='뒤로 물러서기', frames=[
        dict(hand=[.05, -.22, 1.18], off=[.12, -.62, .97], axis=[0, -.5, .87], lean=.06),
        dict(hand=[-.1, -.3, 1.35], off=[.25, -.55, 1.3], axis=[.4, -.4, .8], crouch=.5, lean=.2),
        dict(hand=[-.1, -.25, 1.45], off=[.25, -.5, 1.4], axis=[.4, -.4, .8], lunge=-.6, lean=.38, bob=.04, back=.3),
        dict(hand=[0, -.25, 1.25], off=[.2, -.55, 1.15], axis=[.2, -.45, .85], lunge=-.3, lean=.15, back=.2)]),
}

STYLES = ['guard', 'parry', 'sidestep', 'backstep']
SELECTED = 'guard'  # chosen by the user (2026-10-04); the other styles are backups


def displace(res, sway, back, mounted=False):
    """Lean the whole figure sideways (+x, about the feet) and slide it back (+y)."""
    if not sway and not back:
        return res

    def move(v):
        v = np.atleast_2d(np.array(v, float))
        if mounted:  # the horse slides sideways instead of tilting
            d = np.c_[np.full(len(v), sway*.9), np.full(len(v), back*.6), np.zeros(len(v))]
        else:
            d = np.c_[sway*(v[:, 2]/1.6 + .25), np.full(len(v), back), np.zeros(len(v))]
        return v + d
    res['faces'] = [(move(v), c) for v, c in res['faces']]
    res['head'] = move(res['head'])[0]
    return res


def foot_pose(k):
    p = U.sword_pose(k)
    if 'off' in k:
        p['hands'][1] = list(k['off'])
        p['elbows'][1] = ((V([.4, 0, 1.42]) + V(k['off']))/2 + [.15, .05, -.05]).tolist()
    return p


def foot(kit):
    def build(f, style):
        k = SWORD[style]['frames'][f]
        faces, head = U.soldier(foot_pose(k), kit)
        return displace(U.result(faces, head, 0, f), k.get('sway', 0), k.get('back', 0))
    return build


def bowman(kit):
    def build(f, style):
        k = BOW[style]['frames'][f]
        p = U.pose(0, 0)
        p['hands'] = [list(k['hand']), list(k['off'])]
        p['elbows'] = [((V([-.4, 0, 1.42]) + V(k['hand']))/2 + [-.2, .08, 0]).tolist(),
                       ((V([.4, 0, 1.42]) + V(k['off']))/2 + [.12, 0, 0]).tolist()]
        p['lean'], p['bob'] = k.get('lean', 0), k.get('bob', 0)
        A.stance(p, k)
        _, head = U.soldier(p, kit, weapon=False)
        T = U.transform(p)
        h = T(p['hands'][1])
        axis = A.unit(k['axis'])
        belly = A.unit(np.cross(axis, [1, 0, 0])) if abs(axis[0]) < .9 else V([0, -1, 0])
        U.bow(h, h + belly*.04, axis, belly, width=.075 if style == 'guard' else .045)
        U.m.rod(T([-.21, .3, .93]), T([-.21, .3, 1.53]), .12, '#704022')
        for x in [-.28, -.2, -.12]:
            U.m.rod(T([x, .3, 1.3]), T([x, .3, 1.82]), .03, '#d7bc7c')
        return displace(U.result(U.m.meshes, head, 0, f), k.get('sway', 0), k.get('back', 0))
    return build


def rider(**cav):
    def build(f, style):
        k = MOUNT[style]['frames'][f]
        name = 'block_' + style
        A.SPEAR[name] = dict(name=MOUNT[style]['name'], effect='none', frames=MOUNT[style]['frames'])
        res = U.cavalry(U.ATTACK, f, style=name, **cav)
        res['effects'] = []
        res['face'] = 'normal'
        return displace(res, k.get('sway', 0), 0, mounted=True)
    return build


def as_unit(idle, block, style):
    """build.UNITS-style builder that renders the block row for every row but idle."""
    return lambda row, f, style_=None: idle(row, f) if row == 0 else block(f, style)
