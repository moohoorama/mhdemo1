"""Attack keyframes and effect geometry (model space) for the ver2 units.

Every style follows the same beat: frames 1-2 gather power (wind up, crouch, pull
back), frame 3 releases it explosively (extended pose + effect), frame 4 follows
through. Effects are plain geometry dicts drawn later as translucent 2D layers by
effects.py: arc (swept weapon band), lines (speed lines), cone (thrust burst),
ring (shock ring), wedge (arrow streak), ghost (afterimages).
"""
import numpy as np

V = lambda a: np.array(a, dtype=float)  # noqa: E731


def unit(v):
    v = V(v)
    return v / np.linalg.norm(v)


# Sword (infantry, bandit). hand/elbow = sword arm, blade = blade direction,
# crouch/lunge = stance weights (see stance()).
SWORD = {
    'overhead': dict(name='내려베기', effect='arc', frames=[
        dict(hand=[-.5, .05, 1.8], elbow=[-.62, .05, 1.55], blade=[-.15, .45, .88], lean=.08),
        dict(hand=[-.55, .2, 2.05], elbow=[-.68, .12, 1.72], blade=[-.35, .75, .55], lean=.26, bob=-.1, crouch=1),
        dict(hand=[-.35, -.9, 1.0], elbow=[-.5, -.5, 1.25], blade=[0, -.7, -.7], lean=-.44, bob=-.11, lunge=1),
        dict(hand=[-.4, -.75, .85], elbow=[-.5, -.45, 1.1], blade=[.1, -.45, -.88], lean=-.28, bob=-.08, lunge=1)],
        via=[dict(hand=[-.45, -.1, 2.1], blade=[-.1, .1, 1]), dict(hand=[-.4, -.6, 1.8], blade=[0, -.6, .8]),
             dict(hand=[-.37, -.85, 1.35], blade=[0, -1, .05])]),
    'sweep': dict(name='횡베기', effect='arc', frames=[
        dict(hand=[-.75, .1, 1.3], elbow=[-.65, 0, 1.25], blade=[-.6, .6, .15], lean=.05),
        dict(hand=[-.9, .4, 1.6], elbow=[-.8, .2, 1.45], blade=[-.55, .55, .62], lean=.24, bob=-.09, crouch=.8),
        dict(hand=[.15, -.85, 1.25], elbow=[-.25, -.5, 1.3], blade=[.8, -.55, 0], lean=-.36, bob=-.06, lunge=.8),
        dict(hand=[.4, -.55, 1.2], elbow=[0, -.4, 1.25], blade=[.9, .25, -.1], lean=-.2, bob=-.04, lunge=.8)],
        via=[dict(hand=[-.85, -.2, 1.4], blade=[-.95, -.15, .25]), dict(hand=[-.45, -.85, 1.28], blade=[-.45, -.9, 0]),
             dict(hand=[-.1, -.95, 1.25], blade=[.3, -.95, 0])]),
    'rising': dict(name='올려베기', effect='arc', frames=[
        dict(hand=[-.55, .15, .95], elbow=[-.62, .05, 1.15], blade=[-.2, .6, -.75], lean=.1, bob=-.05, crouch=.5),
        dict(hand=[-.8, .3, .85], elbow=[-.75, .12, 1.1], blade=[-.75, .45, -.5], lean=.26, bob=-.15, crouch=1),
        dict(hand=[-.35, -.6, 1.9], elbow=[-.5, -.35, 1.62], blade=[0, -.35, .95], lean=-.24, bob=.06, lunge=.6),
        dict(hand=[-.35, -.3, 2.0], elbow=[-.5, -.15, 1.7], blade=[0, .1, 1], lean=-.1, bob=.03, lunge=.6)],
        via=[dict(hand=[-.6, -.3, .85], blade=[-.35, -.45, -.82]), dict(hand=[-.45, -.75, 1.25], blade=[0, -1, -.05]),
             dict(hand=[-.4, -.75, 1.65], blade=[0, -.75, .65])]),
    'thrust': dict(name='찌르기', effect='pierce', frames=[
        dict(hand=[-.5, .25, 1.1], elbow=[-.62, .2, 1.15], blade=[0, -1, .15], lean=.08),
        dict(hand=[-.8, .5, 1.4], elbow=[-.78, .25, 1.35], blade=[.15, -1, .12], lean=.28, bob=-.12, crouch=1),
        dict(hand=[-.25, -1.15, 1.2], elbow=[-.38, -.6, 1.25], blade=[0, -1, 0], lean=-.46, bob=-.08, lunge=1.2),
        dict(hand=[-.3, -.8, 1.15], elbow=[-.42, -.45, 1.2], blade=[0, -1, -.1], lean=-.26, bob=-.06, lunge=1)]),
    # Bare hands (martial artist, weapon 'fist'): blade = forearm direction, the effect sits on the fist.
    'jab': dict(name='정권 지르기', effect='pierce', frames=[
        dict(hand=[-.4, .15, 1.2], elbow=[-.6, .1, 1.15], blade=[0, -1, 0], lean=.05),
        dict(hand=[-.45, .38, 1.15], elbow=[-.62, .25, 1.2], blade=[0, -1, .05], lean=.22, bob=-.1, crouch=1),
        dict(hand=[-.25, -1.05, 1.4], elbow=[-.35, -.55, 1.4], blade=[0, -1, 0], lean=-.4, bob=-.06, lunge=1.1),
        dict(hand=[-.3, -.8, 1.35], elbow=[-.4, -.45, 1.35], blade=[0, -1, 0], lean=-.22, bob=-.04, lunge=.9)]),
    'hook': dict(name='돌려치기', effect='arc', frames=[
        dict(hand=[-.7, .1, 1.35], elbow=[-.62, 0, 1.3], blade=[-.5, -.8, 0], lean=.05),
        dict(hand=[-.95, .3, 1.4], elbow=[-.8, .15, 1.4], blade=[-.6, -.6, 0], lean=.2, bob=-.08, crouch=.8),
        dict(hand=[.05, -.85, 1.4], elbow=[-.3, -.55, 1.4], blade=[.9, -.4, 0], lean=-.35, bob=-.05, lunge=.8),
        dict(hand=[.3, -.6, 1.35], elbow=[-.05, -.45, 1.35], blade=[1, 0, 0], lean=-.2, bob=-.03, lunge=.8)],
        via=[dict(hand=[-.85, -.4, 1.4], blade=[-.3, -.95, 0]), dict(hand=[-.4, -.9, 1.4], blade=[.2, -1, 0])]),
    'upper': dict(name='올려치기', effect='arc', frames=[
        dict(hand=[-.5, .05, 1.05], elbow=[-.62, .05, 1.15], blade=[0, -.4, -.9], lean=.08, bob=-.04, crouch=.5),
        dict(hand=[-.6, .2, .85], elbow=[-.65, .1, 1.05], blade=[0, -.4, -.9], lean=.22, bob=-.15, crouch=1),
        dict(hand=[-.3, -.6, 2.0], elbow=[-.45, -.4, 1.6], blade=[0, -.3, .95], lean=-.22, bob=.06, lunge=.6),
        dict(hand=[-.3, -.4, 2.05], elbow=[-.45, -.25, 1.65], blade=[0, 0, 1], lean=-.1, bob=.03, lunge=.5)],
        via=[dict(hand=[-.5, -.5, 1.1], blade=[0, -1, -.2]), dict(hand=[-.4, -.7, 1.55], blade=[0, -.7, .7])]),
    'rush': dict(name='돌진 지르기', effect='pierce', frames=[
        dict(hand=[-.4, .2, 1.2], elbow=[-.6, .15, 1.15], blade=[0, -1, 0], lean=.1, bob=-.04, crouch=.5),
        dict(hand=[-.5, .45, 1.1], elbow=[-.65, .3, 1.15], blade=[0, -1, .05], lean=.3, bob=-.14, crouch=1.2),
        dict(hand=[-.2, -1.35, 1.35], elbow=[-.3, -.75, 1.38], blade=[0, -1, 0], lean=-.58, bob=-.1, lunge=1.5),
        dict(hand=[-.25, -1.0, 1.3], elbow=[-.35, -.6, 1.33], blade=[0, -1, 0], lean=-.35, bob=-.07, lunge=1.3)]),
}

# Spear (cavalry). hand/elbow in rider space, dir = spear direction,
# rear = horse pitch (+ lifts the forehand), shift = whole body along y (- forward).
SPEAR = {
    'charge': dict(name='돌진 찌르기', effect='pierce', frames=[
        dict(hand=[-.48, .3, 1.2], dir=[0, -1, .12], lean=.12),
        dict(hand=[-.45, .6, 1.3], dir=[0, -1, .2], lean=.32, rear=.12),
        dict(hand=[-.48, -1.3, 1.15], dir=[0, -1, 0], lean=-.42, shift=-.3),
        dict(hand=[-.48, -.85, 1.15], dir=[0, -1, -.05], lean=-.22, shift=-.18)]),
    'downstab': dict(name='내려찍기', effect='arc', frames=[
        dict(hand=[-.45, .1, 1.6], dir=[0, -.45, .9], lean=.1, bob=.04),
        dict(hand=[-.4, .35, 1.95], dir=[0, .2, 1], lean=.28, bob=.1, rear=.25),
        dict(hand=[-.45, -.75, 1.2], dir=[0, -.55, -.83], lean=-.38, shift=-.15),
        dict(hand=[-.45, -.6, 1.05], dir=[0, -.4, -.92], lean=-.22, shift=-.1)],
        via=[dict(hand=[-.42, -.2, 1.75], dir=[0, -.6, .8]), dict(hand=[-.44, -.55, 1.5], dir=[0, -1, .1])]),
    'swing': dict(name='휘두르기', effect='arc', frames=[
        dict(hand=[-.6, .2, 1.25], dir=[-.7, .7, .05], lean=.06),
        dict(hand=[-.7, .45, 1.3], dir=[-.35, .93, .1], lean=.22, rear=.06),
        dict(hand=[0, -.75, 1.2], dir=[.85, -.5, 0], lean=-.32, shift=-.12),
        dict(hand=[.3, -.4, 1.2], dir=[.95, .25, -.05], lean=-.16, shift=-.08)],
        via=[dict(hand=[-.7, -.1, 1.3], dir=[-.98, -.2, .05]), dict(hand=[-.45, -.6, 1.25], dir=[-.4, -.92, 0]),
             dict(hand=[-.15, -.75, 1.2], dir=[.3, -.95, 0])]),
    'glaive_sweep': dict(name='언월도 횡베기', effect='arc', frames=[
        dict(hand=[-.55, .15, 1.45], dir=[-.5, .6, .62], lean=.08),
        dict(hand=[-.75, .45, 1.75], dir=[-.35, .75, .55], lean=.3, bob=.05, rear=.1),
        dict(hand=[.2, -.7, 1.25], dir=[.88, -.45, -.05], lean=-.38, shift=-.15),
        dict(hand=[.4, -.35, 1.2], dir=[.95, .25, -.1], lean=-.2, shift=-.1)],
        via=[dict(hand=[-.8, -.15, 1.5], dir=[-.95, -.2, .15]), dict(hand=[-.45, -.7, 1.35], dir=[-.35, -.93, .05]),
             dict(hand=[-.05, -.8, 1.3], dir=[.4, -.9, 0])]),
    'leap': dict(name='도약 찌르기', effect='pierce', frames=[
        dict(hand=[-.48, .3, 1.2], dir=[0, -1, .1], lean=.1, rear=.12),
        dict(hand=[-.45, .5, 1.4], dir=[0, -1, .35], lean=.3, rear=.45),
        dict(hand=[-.48, -1.25, 1.1], dir=[0, -1, -.08], lean=-.46, shift=-.35),
        dict(hand=[-.48, -.8, 1.12], dir=[0, -1, -.03], lean=-.2, shift=-.18)]),
}

# Bow (archer). bow/draw = bow hand / string hand, aim = shot direction, bend = limb
# curvature, flight = how far the released arrow has flown (None = still on string).
BOW = {
    'power': dict(name='강궁', effect='streak', frames=[
        dict(bow=[.2, -.85, 1.55], draw=[.05, -.5, 1.55], aim=[0, -1, 0], bend=1, lean=.04),
        dict(bow=[.22, -.95, 1.6], draw=[-.25, .5, 1.62], aim=[0, -1, 0], bend=1.8, lean=.22, bob=-.04, crouch=.5),
        dict(bow=[.2, -.9, 1.55], draw=[-.4, .55, 1.72], aim=[0, -1, 0], bend=1, lean=.26, flight=2.2),
        dict(bow=[.2, -.85, 1.5], draw=[-.3, .3, 1.45], aim=[0, -1, 0], bend=1, lean=.1)]),
    'low': dict(name='낮게 쏘기', effect='ring', frames=[
        dict(bow=[.2, -.8, 1.4], draw=[.05, -.45, 1.4], aim=[0, -1, 0], bend=1, bob=-.08, crouch=.6),
        dict(bow=[.22, -.9, 1.35], draw=[-.2, .45, 1.4], aim=[0, -1, 0], bend=1.7, lean=.15, bob=-.2, crouch=1),
        dict(bow=[.2, -.9, 1.35], draw=[-.35, .5, 1.5], aim=[0, -1, 0], bend=1, lean=.2, bob=-.2, crouch=1,
             flight=2.0),
        dict(bow=[.2, -.85, 1.35], draw=[-.25, .3, 1.35], aim=[0, -1, 0], bend=1, lean=.08, bob=-.16, crouch=1)]),
    'sky': dict(name='하늘로 쏘기', effect='streak', frames=[
        dict(bow=[.2, -.7, 1.85], draw=[.05, -.4, 1.7], aim=[0, -.8, .6], bend=1, lean=.12),
        dict(bow=[.22, -.75, 2.0], draw=[-.2, .35, 1.5], aim=[0, -.8, .6], bend=1.8, lean=.3, bob=-.06, crouch=.6),
        dict(bow=[.2, -.7, 1.95], draw=[-.35, .45, 1.45], aim=[0, -.8, .6], bend=1, lean=.34, flight=1.9),
        dict(bow=[.2, -.65, 1.8], draw=[-.25, .25, 1.4], aim=[0, -.8, .6], bend=1, lean=.16)]),
    'rapid': dict(name='속사', effect='ghost', frames=[
        dict(bow=[.2, -.85, 1.5], draw=[.05, -.5, 1.5], aim=[0, -1, 0], bend=1, lean=.02),
        dict(bow=[.2, -.9, 1.52], draw=[-.12, .25, 1.52], aim=[0, -1, 0], bend=1.5, lean=.12, bob=-.03),
        dict(bow=[.2, -.88, 1.5], draw=[-.3, .45, 1.6], aim=[0, -1, 0], bend=1, lean=.16, flight=2.4),
        dict(bow=[.2, -.85, 1.5], draw=[.05, -.5, 1.5], aim=[0, -1, 0], bend=1, lean=.04)]),
}

# Chosen attack per unit. The other styles stay defined as backups
# (compare them with attack_candidates.py).
SELECTED = {'infantry': 'sweep', 'bandit': 'overhead', 'archer': 'power', 'cavalry': 'charge',
            'spearman': 'thrust', 'guanyu': 'glaive_sweep', 'zhangfei': 'swing'}


def stance(p, k):
    """Crouch bends both knees forward; lunge steps the sword-side foot forward."""
    crouch, lunge = k.get('crouch', 0), k.get('lunge', 0)
    knees, feet = np.array(p['knees'], float), np.array(p['feet'], float)
    knees += crouch*np.array([[0, -.14, -.07], [0, -.14, -.07]])
    feet += lunge*np.array([[-.05, -.38, 0], [.05, .28, 0]])
    knees += lunge*np.array([[-.05, -.34, -.04], [.05, .14, -.04]])
    p['knees'], p['feet'] = knees.tolist(), feet.tolist()


TRAIL_REACH = 1.3  # swept band reaches past the blade tip for a cartoon swing


def swing_path(style):
    """Keyframes from the gathered pose (frame 2) through the swing waypoints to the strike (frame 3)."""
    a, b, vias = style['frames'][1], style['frames'][2], style.get('via', [])
    path = [a]
    for i, v in enumerate(vias):
        t = (i + 1)/(len(vias) + 1)
        k = {key: np.array(a.get(key, 0), float)*(1 - t) + np.array(b.get(key, 0), float)*t for key in set(a) | set(b)}
        k.update({key: np.array(val, float) for key, val in v.items()})
        path.append(k)
    return path + [b]


def sample(path, n_per_leg, lerp):
    out = []
    for a, b in zip(path, path[1:]):
        out += [lerp(a, b, t) for t in np.linspace(0, 1, n_per_leg, endpoint=False)]
    return out + [path[-1]]


def arc(pairs, alpha=1.0):
    return dict(kind='arc', pairs=pairs, alpha=alpha, layer='under')


def pierce(tip, direction, hand, alpha=1.0, burst=True, reach=1.0):
    """Thrust: burst cone and shock ring ahead of the tip, speed lines behind the arm."""
    d = unit(direction)
    side = unit(np.cross(d, [0, 0, 1])) if abs(d[2]) < .9 else V([1, 0, 0])
    up = np.cross(side, d)
    offsets = (side*.3 + up*.1, -side*.3 + up*.16, up*.36, -up*.14, side*.12 - up*.3)
    out = [dict(kind='lines', alpha=alpha, layer='under',
                segments=[(hand - d*2.1*reach + o, hand - d*.45 + o) for o in offsets])]
    if burst:
        out += [dict(kind='cone', tip=tip - d*.15, dir=d, length=1.4, width=.75, alpha=alpha, layer='over'),
                dict(kind='ring', center=tip + d*.2, axis=d, radius=.36, alpha=alpha, layer='over')]
    else:
        out.append(dict(kind='ring', center=tip + d*.45, axis=d, radius=.5, alpha=alpha*.5, layer='over'))
    return out
