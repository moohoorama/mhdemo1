"""Scene people and poses beyond standing (2026-10-22), for the story scenes' sheets (civ3d.py converts them).

People: court officials in wide-sleeved robes holding a tablet (홀) in the faction colour or in black, a eunuch, a
wounded soldier, and two stretcher bearers with a man on the stretcher. Poses
added to scene sheets: sit (on a stool), throne (on the imperial chair), bow (읍),
prostrate (부복), lie (on the back), sitground (slumped on the ground), and two threats (2026-10-29): fist (a fist
thrown up as after an uppercut) and point (an arm thrust out, jabbing a finger).

The model is the civilian soldier of units.py. Bending and lying rotate the finished meshes about the hip (or the
ground) instead of adding joints; a figure lying down stamps its 2D head turned on its side (turned()), and a
sprite with two or three people carries every head in fr['heads'] for frame() to stamp.
"""
import copy
import math

import numpy as np

import blocks as Bl  # noqa: F401  (build imports it; keeps import order like civilian users)
import build as B
import civilian as C
import heads as Hd
import looks as L
import render as R
import units as U

m, V = U.m, U.V
IVORY, WOOD, DARK_WOOD, CLOTH = '#ede1bd', '#976039', '#5e3a22', '#e8e2d0'
STOOL, THRONE_RED, THRONE_GOLD = '#6e4126', '#a8402e', U.GOLD  # the throne: red lacquered wood, gold trim
R.add_ramps({THRONE_RED: 'KLLM'})
R.add_ramps({CLOTH: '5566', DARK_WOOD: '0ffg', STOOL: 'fggh'})
# the quilt of the sickbed (bed) in keys of its own, dark to light; civ3d puts them in the saddle tags (BED_COLORS)
BLANKET = '#4e5a7e'
BLANKET_KEYS = '«»¬'
R.COLOR.update({'«': (40, 48, 74), '»': (78, 90, 126), '¬': (112, 126, 164)})
R.add_ramps({BLANKET: '«»¬¬'})
SLEEP_HAIR, SLEEP_CAP, SLEEP_EYE = '#d4d8d8', '#1e2832', '#15191b'
R.add_ramps({SLEEP_HAIR: '3456', SLEEP_CAP: '0112'})
# the sleeper's own look by kit (default: dark hair, bare head, no beard)
SLEEPERS = {'taoqian_civ1': dict(hair=SLEEP_HAIR, cap=True, beard=True)}

POSES = ['sit', 'throne', 'bow', 'prostrate', 'lie', 'sitground', 'fist', 'point', 'bed']
ROWS = C.ANIMATIONS + [p for p in POSES if p not in C.ANIMATIONS]  # scene sheet rows: civilian's, then these
TURNED_HEAD = {'lie'}  # the 2D head turned on its side (crown toward the head end)
MODELLED_HEAD = {'bed'}  # no 2D head: the head is modelled with the pose (sleeper)


def turned(head_set):
    """The head set lying on its side: every view's grid turned a quarter clockwise, the anchor with it."""
    grids = {d: [''.join(r[x] for r in reversed(g)) for x in range(len(g[0]))] for d, g in head_set['grids'].items()}
    ax, ay = head_set.get('anchor', Hd.ANCHOR)
    h = len(next(iter(head_set['grids'].values())))
    return dict(head_set, grids=grids, anchor=(h - 1 - ay, ax))


def rotate(meshes, angle, pivot_z, shift=(0, 0, 0), above=None):
    """Rotate vertices about the x axis through (y=0, z=pivot_z): +angle tips the top forward (-y). With above,
    only what lies above that height turns, eased over 0.2 so a robe bends instead of tearing."""
    c, s = math.cos(angle), math.sin(angle)
    out = []
    for verts, color in meshes:
        vs = []
        for x, y, z in verts:
            a = angle
            if above is not None:
                a = angle * min(1, max(0, (z - above + .1) / .2))
            ca, sa = (c, s) if above is None else (math.cos(a), math.sin(a))
            dz = z - pivot_z
            vs.append([x + shift[0], y * ca - dz * sa + shift[1], pivot_z + y * sa + dz * ca + shift[2]])
        out.append((np.array(vs), color))
    return out


def turn_point(p, angle, pivot_z, shift=(0, 0, 0)):
    return rotate([(np.array([p]), None)], angle, pivot_z, shift)[0][0][0]


def moved(meshes, dx=0, dy=0, dz=0):
    return [(verts + V([dx, dy, dz]), color) for verts, color in meshes]


# ---- poses ---------------------------------------------------------------------------------------------------

def standing(kit_pose=None):
    p = U.pose(0, 0)
    p.update(copy.deepcopy(C.REST))
    return p


def sit_pose():
    """On a stool: hips at knee height, thighs forward, hands on the knees. Hands and elbows are given before the
    bob (the model adds it), knees and feet are where they are."""
    p = standing()
    p['bob'] = -.33
    p['knees'] = [[-.24, -.42, .46], [.24, -.42, .46]]
    p['feet'] = [[-.25, -.46, .13], [.25, -.46, .13]]
    p['hands'] = [[-.3, -.38, .97], [.3, -.38, .97]]
    p['elbows'] = [[-.5, -.12, 1.2], [.5, -.12, 1.2]]
    return p


def hold_pose(p):
    """Both hands together in front of the chest (holding the tablet, or fist in palm)."""
    p['hands'] = [[-.05, -.42, 1.22], [.07, -.4, 1.24]]
    p['elbows'] = [[-.46, -.22, 1.12], [.46, -.22, 1.12]]
    return p


def kneel_pose():
    """Kneeling upright: knees on the ground ahead of the hips, shins flat behind."""
    p = standing()
    p['bob'] = -.3
    p['knees'] = [[-.24, -.22, .12], [.24, -.22, .12]]
    p['feet'] = [[-.24, .3, .08], [.24, .3, .08]]
    return p


def stool():
    m.meshes = []
    m.box(V([0, .02, .2]), [.52, .46, .4], STOOL)
    return m.meshes


def throne():
    """The imperial chair: a step under it, a broad seat, arms ending in gold knobs and a tall back rising well above
    the sitter's head with a gold crest, so it shows round him from both views."""
    m.meshes = []
    m.box(V([0, .05, .06]), [1.3, 1.1, .12], THRONE_RED)  # the step
    m.box(V([0, .08, .26]), [1.02, .86, .3], THRONE_RED)  # seat
    m.box(V([0, .08, .43]), [1.06, .9, .05], THRONE_GOLD)
    m.box(V([0, .5, 1.25]), [1.22, .14, 1.7], THRONE_RED)  # tall back
    m.box(V([0, .5, 2.14]), [1.4, .2, .14], THRONE_GOLD)  # crest
    for s in (-1, 1):
        m.box(V([s*.56, .1, .66]), [.12, .82, .46], THRONE_RED)  # arms
        m.box(V([s*.56, .1, .9]), [.16, .86, .06], THRONE_GOLD)
        m.ell(V([s*.56, -.33, .95]), [.09, .09, .09], THRONE_GOLD)
        m.ell(V([s*.7, .5, 2.14]), [.1, .1, .1], THRONE_GOLD)
    return m.meshes


BED_SIZE = 1.5  # the sickbed against a life-size one, every way: the 2D heads are big, so the bed is too


def sickbed(lift):
    """침상 for the sick: a low wooden platform on short legs with a low board at the head end (+y), a pillow under
    the head and a quilt drawn up to the shoulders; the sleeper lies along y, head at +y, on its top at lift (all
    measures times BED_SIZE)."""
    m.meshes = []
    k = BED_SIZE
    L0, L1, W = -1.35*k, 1.45*k, .6*k
    yc, ln = (L0 + L1)/2, L1 - L0
    m.box(V([0, yc, lift - .08*k]), [W*2, ln, .16*k], STOOL)  # the platform
    for sx in (-1, 1):
        for y in (L0 + .1*k, L1 - .1*k):
            m.box(V([sx*(W - .08*k), y, (lift - .16*k)/2]), [.14*k, .14*k, lift - .16*k], DARK_WOOD)
    m.box(V([0, L1 - .04*k, lift + .2*k]), [W*2, .08*k, .4*k], DARK_WOOD)  # head board
    m.ell(V([0, 1.2*k, lift + .1*k]), [.34*k, .2*k, .1*k], CLOTH)  # pillow
    m.box(V([0, -.45*k, lift + .25*k]), [W*2 - .08*k, 1.75*k, .36*k], BLANKET)  # the quilt over the bed
    m.ell(V([0, -.4*k, lift + .44*k]), [.46*k, 1.0*k, .26*k], BLANKET)  # and over the body
    return m.meshes


def sleeper(look, c):
    """A big (2D-head sized) head lying face up (+z) with its crown toward +y, centred on c: face, hair round the
    back and crown, shut eyes, brows, and the look's court cap and beard."""
    hair = look.get('hair', '#2a2220')
    m.ell(c + V([0, .2, -.12]), [.6, .56, .42], hair)  # back of the head and crown on the pillow
    m.ell(c, [.54, .5, .44], U.SKIN)  # face
    for sx in (-1, 1):
        m.box(c + V([sx*.22, .1, .43]), [.2, .07, .06], SLEEP_EYE)  # shut eyes
        m.box(c + V([sx*.22, .25, .4]), [.22, .08, .06], hair)  # brows
    if look.get('cap'):  # 진현관 lying flat toward the head board
        m.box(c + V([0, .72, 0]), [.62, .36, .62], SLEEP_CAP)
    if look.get('beard'):  # a long beard laid on the chest
        m.ell(c + V([0, -.55, .22]), [.42, .5, .2], hair)


# ---- sheet builders ---------------------------------------------------------------------------------------------

def posed(walk_build, kit, held=False, extra=None):
    """A scene sheet builder: the civilian rows (idle, walk, scene motions), then the POSES rows. held: both hands
    stay together in front (an official with his tablet), also while walking. extra(p, meshes) adds parts that
    follow the body (sleeves, the tablet) before any bending."""
    base = C.scene_unit(walk_build, kit)

    def body(p):
        faces, head = U.soldier(p, kit, weapon=False)
        faces = list(faces)
        if extra:
            faces += extra(p)
        return faces, head

    def build(row, f, style=None):
        name = ROWS[row]
        if held and name in ('idle', 'walk'):
            p = U.pose(1 if name == 'walk' else 0, f)
            p.update({k: v for k, v in copy.deepcopy(C.REST).items()})
            if name == 'idle':
                p['bob'] = [0, .02, 0, -.01][f]
            hold_pose(p)
            faces, head = body(p)
            return U.result(faces, head, 0, f)
        if name not in POSES:
            return base(row, f, style) if style else base(row, f)
        if name == 'sit':
            p = sit_pose()
            if held:
                hold_pose(p)
                p['hands'] = [[x, y, z - .05] for x, y, z in p['hands']]
            faces, head = body(p)
            return U.result(faces + stool(), head, 0, f)
        if name == 'throne':
            p = sit_pose()
            p['hands'] = [[-.42, -.2, .98], [.42, -.2, .98]]  # on the chair's arms
            faces, head = body(p)
            return U.result(throne() + faces, head, 0, f)
        if name == 'bow':
            p = hold_pose(standing())
            faces, head = body(p)
            a = .85
            return U.result(rotate(faces, a, .8, above=.8), turn_point(head, a, .8), 0, f)
        if name == 'prostrate':
            p = kneel_pose()
            p['hands'] = [[-.3, -.5, 1.5], [.3, -.5, 1.5]]
            p['elbows'] = [[-.5, -.4, 1.45], [.5, -.4, 1.45]]
            faces, head = body(p)
            a, hip = 1.15, .47
            return U.result(rotate(faces, a, hip, above=hip), turn_point(head, a, hip), 0, f)
        if name == 'sitground':
            p = U.pose(C.EXHAUSTED, 0)
            p['lean'] = -.12
            p['hands'][0] = [-.12, -.3, .5]  # a hand pressed to the chest (seated pose: before its bob)
            faces, head = body(p)
            return U.result(faces, head, C.EXHAUSTED, f)
        if name == 'lie':
            p = standing()
            p['hands'] = [[-.52, -.05, .95], [.5, -.1, .95]]
            p['elbows'] = [[-.55, 0, 1.18], [.55, 0, 1.18]]
            p['bob'] = [0, .015][f % 2]  # breathing
            faces, head = body(p)
            shift = (0, -.95, 0)
            lying = rotate(faces, -math.pi/2, .3, shift)
            h = turn_point(head, -math.pi/2, .3, shift)
            return U.result(lying, h, 0, f)
        if name == 'bed':  # the body is all under the quilt: a modelled head lies face up on the pillow
            k, lift = BED_SIZE, .45*BED_SIZE
            kit_ = U.KITS[kit]
            m.meshes = []
            if not SLEEPERS.get(kit, {}).get('beard'):
                m.ell(V([0, .62*k, lift + .3*k]), [.36*k, .14*k, .16*k], kit_.get('robe') or kit_['cloth'])
            sleeper(SLEEPERS.get(kit, {}), V([0, 1.25, lift + .55]))
            return U.result(m.meshes + sickbed(lift + .02*k), V([0, 1.3, lift + .5]), 0, f)
        if name == 'fist':
            p = standing()
            p['hands'][0], p['elbows'][0] = [-.32, -.55, 2.25], [-.46, -.42, 1.7]
            p['hands'][1], p['elbows'][1] = [.4, -.25, 1.05], [.52, -.05, 1.18]
            p['lean'] = -.08
            faces, head = body(p)
            return U.result(faces, head, 0, f)
        if name == 'point':
            p = standing()
            p['hands'][0], p['elbows'][0] = [-.18, -.88, 1.42], [-.38, -.45, 1.38]
            p['hands'][1], p['elbows'][1] = [.4, .08, 1.02], [.62, .1, 1.2]  # the other fist on the hip
            p['lean'] = -.1
            faces, head = body(p)
            return U.result(faces, head, 0, f)
        raise KeyError(name)
    build.rows = ROWS
    return build


def official_parts(p):
    """Wide sleeves over the hands and the ivory tablet held upright between them."""
    T = U.transform(p)
    m.meshes = []
    kit = U.KITS[OFFICIAL_KIT['now']]
    for h in p['hands']:
        m.ell(T(h) + [0, .1, .02], [.15, .17, .19], kit['cloth'])
    c = (T(p['hands'][0]) + T(p['hands'][1])) / 2
    m.box(c + [0, -.16, .24], [.13, .03, .55], IVORY)
    return m.meshes


OFFICIAL_KIT = {'now': None}


def official(key, n, robe, sash, head):
    walk = C.walker(key, n, cloth=robe, robe=robe, guard=sash, tabard=sash)

    def parts(p):
        OFFICIAL_KIT['now'] = walk.kit
        return official_parts(p)
    return posed(walk, walk.kit, held=True, extra=parts), head


JINXIAN = {d: Hd.recolor(Hd.gwanmo(rows, 2), {'2': '1', '3': '1'}) for d, rows in Hd.HAIR.items()}  # tall black court cap (crest=2)
EUNUCH_CAP = {d: Hd.recolor(Hd.gwanmo(rows, 1), {'2': '1', '3': '2'}) for d, rows in Hd.HAIR.items()}

LOOKS = {
    # robe in the faction colour (a Han court reads red, a lord's hall his own colour), black sash
    'official': official('official', 1, U.BLUE, U.BLACK, Hd.heads(L.bearded(JINXIAN, 'goatee'), crest=2)),
    # senior minister in black with a red sash
    'official_black': official('official', 2, U.BLACK, U.RED, Hd.heads(L.bearded(JINXIAN, 'long'), crest=2)),
    'eunuch': official('eunuch', 1, U.BROWN, U.BLACK, Hd.heads(EUNUCH_CAP)),
}


# ---- people in twos and threes ---------------------------------------------------------------------------------

def frame(build, head_set, spec, d, row, col, lift=1, style=None):
    """build.frame for sprites with several 2D heads (fr['heads']: (point, head set, face))."""
    U.VIEW['t'] = R.view(R.AZIMUTH[d])[0]
    fr = build(row, col)
    pivot = (spec['pivot'][0], spec['pivot'][1] - lift)
    gr = R.render(fr['faces'], d, spec['scale'], cell=spec['cell'], pivot=pivot, depths=[])
    for point, heads, face in sorted(fr.get('heads', [(fr['head'], head_set, fr['face'])]),
                                     key=lambda h: -float(np.dot(h[0], U.VIEW['t']))):  # far head first
        x, y = R.project(point, d, spec['scale'], pivot)
        Hd.stamp(gr, heads or head_set, d, face, x - .5, y - .5)
    return Hd.to_img(gr)


def stretcher(kit_bearer, head_bearer, kit_hurt):
    """Two bearers carrying a wounded man on a stretcher, one ahead (-y) and one behind."""
    def build(row, f, style=None):
        name = ROWS[row]
        walking = name == 'walk'
        faces, heads = [], []
        for dy, k in ((-1.05, f), (1.05, (f + 2) % 4)):
            p = U.pose(1 if walking else 0, k if walking else f)
            p.update(copy.deepcopy(C.REST))
            p['hands'] = [[-.34, -.05 if dy > 0 else .05, .92], [.34, -.05 if dy > 0 else .05, .92]]
            p['elbows'] = [[-.5, 0, 1.15], [.5, 0, 1.15]]
            fb, hb = U.soldier(p, kit_bearer, weapon=False)
            faces += moved(list(fb), dy=dy)
            heads.append((hb + V([0, dy, 0]), head_bearer, 'normal'))
        m.meshes = []
        for s in (-1, 1):
            m.rod(V([s*.34, -1.25, .9]), V([s*.34, 1.25, .9]), .04, WOOD)
        m.box(V([0, 0, .88]), [.62, 1.7, .04], CLOTH)
        faces += m.meshes
        pl = standing()
        fl, hl = U.soldier(pl, kit_hurt, weapon=False)
        lying = rotate(list(fl), -math.pi/2, .3, (0, -.95, .62))
        h = turn_point(hl, -math.pi/2, .3, (0, -.95, .62))
        faces += lying
        heads.append((h, turned(BANDAGED), 'normal'))
        out = U.result(faces, heads[0][0], 0, f)
        out['heads'] = heads
        return out
    build.rows = ROWS
    build.frame = frame
    return build


U.KITS['wounded_civ'] = L.kit(cloth=U.BLUE, guard=U.BROWN, tabard=U.HEMP_DARK)
_wounded_walk = U.foot_unit('wounded_civ')
_wounded_walk.kit = 'wounded_civ'
U.KITS['bearer_civ'] = L.kit(cloth=U.BLUE, guard=U.BROWN, tabard=U.BROWN)
# bare topknot with a white cloth bound round the brow (the hairline row)
BANDAGED = Hd.heads({d: [r.replace('g', '6') if y == 4 else r for y, r in enumerate(rows)] for d, rows in Hd.HAIR.items()})
CAP_HEAD = Hd.heads(L.CAP)

LOOKS.update({
    'wounded': (posed(_wounded_walk, 'wounded_civ'), BANDAGED),
    'stretcher': (stretcher('bearer_civ', CAP_HEAD, 'wounded_civ'), CAP_HEAD),
})
_xiandi = C.walker('xiandi', 1, cloth=U.BLUE, robe=U.BLUE, guard=U.BLUE, tabard=U.GOLD)  # robe in the sheet's team key
LOOKS['xiandi'] = (posed(_xiandi, _xiandi.kit), Hd.heads(L.GOLD_CAP))  # only its throne row is used
NAMES = {'xiandi': '헌제', 'official': '문관', 'official_black': '중신', 'eunuch': '환관', 'wounded': '부상병',
         'stretcher': '들것'}
# which rows each sheet carries beyond idle, walk and action (civ3d: (row, frames))
SHEET_POSES = {'official': ['bow', 'prostrate', 'sit'], 'official_black': ['bow', 'prostrate', 'sit'],
               'eunuch': ['bow', 'prostrate'], 'wounded': ['lie', 'sitground'], 'stretcher': []}
# officers whose sheets colour the robe differently from their civilian kit: the kit their poses render with
# (Liu Bei: the green coat, his sheet's themeB reached through the red cloth ramp, over the team-coloured robe)
POSE_KITS = {'guanyu': dict(cloth=U.BLUE, guard=U.BLUE, robe=U.BLUE), 'liubei': dict(cloth=U.RED, guard=U.RED, robe=U.BLUE)}
# pose parts in a colour the sheet has no tag for: render key -> tag, and that tag's colours set in the sheet. The
# throne's red lacquer goes to the horse coat tags, which a non-combat sheet never uses and the game never recolours
# (헌제's own palette turns his themeA black and themeB gold).
POSE_KEY_TAGS = {'xiandi': {'K': 'x', 'L': 'w', 'M': 'v'}}
POSE_COLORS = {'xiandi': ['horseSkin1=#c44a3a', 'horseSkin2=#902c28', 'horseSkin3=#5c1a1e']}
POSE_SHEET_HEAD = {'guanyu', 'xiandi'}  # their sheets' heads differ from the civilian pick: pose frames take the sheet's
# poses added to existing officers' sheets (their other rows are not redrawn)
OFFICER_POSES = {'liubei': ['sit'], 'guanyu': ['sit'], 'zhangfei': ['sit', 'point'], 'caocao': ['sit', 'bow'], 'sunjian': ['sit'],
                 'yuanshao': ['sit'], 'zhaoyun': ['sit'], 'dongzhuo': ['sit'], 'yuanshu': ['sit'], 'gongsunzan': ['sit'],
                 'infantry': ['sit', 'sitground', 'prostrate'], 'xiandi': ['throne']}
