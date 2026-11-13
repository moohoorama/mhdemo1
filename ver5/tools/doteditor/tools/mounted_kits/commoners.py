"""민중: unarmed civilians on the 서생 rig. Four looks - 남자 (leads a child by the hand), 여자 (long skirt, child on
her back), 노인 (white hair, long beard, a cane), 노파 (skirt, white hair in a low bun, a
cane, a deep stoop and a hand on the small of her back).

- body: hemp tunic (themeC) over brown trousers or a floor-length skirt (themeA), dark cord sash and shoes (iron).
  No faction colour at all, so a stage's faction never tints them.
- heads: the 서생's topknot slimmed in the S view; the 망건 band row and the faction tie become hair. Women: no band.
- child: simple 3D parts (head, tunic, legs) scaled to about 55%; held by the man's right hand, carried on the
  woman's back with a cloth sling (its head peeks over her shoulder). Both are part of the one sprite.
- no attack row: only idle, walk, hit, exhausted and block are converted.
- prone (2026-11-01): a scene pose, dead face down on the ground (`grid.py poses <id> prone`, export, then
  `grid3d_mounted.py <id> --unit <id> --base scholar --anims prone --dirs N,S,W,NW,SW --mirror`). The finished meshes tip forward onto the ground and the back-of-head grid is
  stamped turned on its side, crown away from the body (prone_heads), as scene_people's 'lie' does.
"""
import math

import grid3d_mounted as M
from mounted_kits import scholar as SC

G, U, Hd, B, Bl = M.G, M.U, M.Hd, M.B, M.Bl
m, V = G.m, G.m.V
HEMP, BROWN, SKIN, CORD, COLLAR, KID = U.HEMP, U.BROWN, U.SKIN, SC.CORD, SC.COLLAR, SC.SCROLL
HAIR_KEYS = {'f': '8', 'g': '9'}


HAIR_OF = {'m': '7', 'l': '8', 'k': '9', 'f': '8', 'g': '9'}


def hair_keys(row):
    return ''.join(HAIR_OF.get(c, c) for c in row)


def put(row, x, c):
    return row[:x] + c + row[x + 1:]


def man_rows(rows):
    """Topknot over short hair with a cloth headband (themeC) across the brow and hair at the temples."""
    out = []
    for y, r in enumerate(rows):
        out.append(r if y == 5 else hair_keys(r))
    if out[6][2] == 'i':  # S view: sideburns
        out[6] = put(put(out[6], 2, '8'), 10, '8')
        out[7] = put(put(out[7], 2, '8'), 10, '8')
    return out


def woman_rows(rows, d):
    """No bun: the dome is hair down to the brow, and the hair hangs past the jaw (2-3 extra rows)."""
    out = [hair_keys(r) for r in rows]
    w = len(out[0])
    top = ''.join('0' if c != '.' else '.' for c in rows[3])
    out[0], out[1], out[2] = '.'*w, '.'*w, top
    out[3] = out[3].replace('0', '0')
    if d == 'S':
        for y in range(6, 12):
            out[y] = put(put(out[y], 1, '8'), 11, '8')
        out += [put(put('.'*w, 1, '8'), 11, '8')]*1
        out += [put(put(put(put('.'*w, 1, '9'), 2, '9'), 10, '9'), 11, '9')]*2
    else:
        ext = ''.join(c if c in '89' else ('0' if c == '0' and False else '.') for c in out[10])
        mid = ''.join(c if c in '89' else '.' for c in out[9])
        out += [mid, ext, ext]
    return out


def elder_rows(rows):
    """Bare gray hair streaked with all three hair shades, no band; the beard is the lightest shade with a few strands."""
    out = []
    for y, r in enumerate(rows):
        row = hair_keys(r)
        row = ''.join(
            ('8' if (i + y) % 4 == 0 else '7') if c == '1' else
            ('7' if (i*7 + y*13) % 5 == 0 else '9' if (i*7 + y*13) % 5 == 1 else '8') if c in '789' and y < 9 else c
            for i, c in enumerate(row))
        out.append(row)
    return out


# 쪽: the old woman's hair knotted low at the back of the head, per view (x, y of the pattern's top left)
BUN = ['.78.', '7889', '.89.']  # no inner outline: over the hair a ring would read as an eye
BUN_AT = {'SW': (9, 6), 'W': (9, 6), 'NW': (6, 7), 'N': (5, 7)}


def old_woman_rows(rows, d):
    """White hair combed flat to the brow with no topknot or band, knotted in a low bun at the back (hidden from S)."""
    out = [hair_keys(r) for r in rows]
    w = len(out[0])
    out[0], out[1], out[2] = '.'*w, '.'*w, ''.join('0' if c != '.' else '.' for c in rows[3])
    out = [''.join('7' if c == '8' and y < 5 else c for c in r) for y, r in enumerate(out)]
    if d in BUN_AT:
        bx, by = BUN_AT[d]
        for y, line in enumerate(BUN):
            for x, c in enumerate(line):
                if c != '.' and bx + x < w:
                    out[by + y] = put(out[by + y], bx + x, c)
    return out


def grids(kind, beard=None):
    g = Hd.TOPKNOT
    if beard:
        g = {d: Hd.beard(r, 'ij', beard) for d, r in g.items()}
    if kind == 'man':
        return {d: man_rows(r) for d, r in g.items()}
    if kind == 'woman':
        return {d: woman_rows(r, d) for d, r in g.items()}
    if kind == 'old_woman':
        return {d: old_woman_rows(r, d) for d, r in g.items()}
    return {d: elder_rows(r) for d, r in g.items()}


PRONE_SKEW = .64  # share of the turn that would lay the body flat across the screen


def prone_yaw():
    """Turn the lying body part of the way across the screen in the diagonal views (head still toward the facing
    side), so it is not foreshortened along the view: the frame's view comes from units.VIEW (set by build.frame)."""
    t = U.VIEW['t']
    az = math.atan2(t[1], t[0])
    if abs(abs(az) - math.pi/2) < .01:
        return 0.0
    if abs(az) > math.pi/2:
        az -= math.copysign(math.pi, az)
    return PRONE_SKEW*az


def prone_heads(head_set):
    """Per view, the head set for lying face down: the back-of-head (NW) grid with the crown turned away from the
    body - left in the W-side views (the E side mirrors them), up in N, down in S."""
    g, (ax, ay) = head_set['grids']['NW'], head_set['anchor']
    h, w = len(g), len(g[0])
    left = [''.join(r[w - 1 - x] for r in g) for x in range(w)]
    down = [r[::-1] for r in reversed(g)]

    def view(d):
        if d == 'N':
            return dict(head_set, grids={'N': g})
        if d == 'S':
            return dict(head_set, grids={'S': down}, anchor=(w - 1 - ax, h - 1 - ay))
        return dict(head_set, grids={k: left for k in ('NW', 'W', 'SW')}, anchor=(ay, w - 1 - ax))
    return view


def tip_forward(meshes, pivot_z, dy, yaw=0.0):
    """Rotate a standing figure a quarter turn about the x axis through (y=0, z=pivot_z) so it lies face down
    (the head toward -y), move it dy along y, then turn it yaw about the vertical axis."""
    return turn([(V([[x, dy - (z - pivot_z), pivot_z + y] for x, y, z in verts]), color) for verts, color in meshes], yaw)


def turn(meshes, yaw):
    c, s = math.cos(yaw), math.sin(yaw)
    return [(V([[x*c - y*s, x*s + y*c, z] for x, y, z in verts]), color) for verts, color in meshes]


# ---- body ----------------------------------------------------------------------------------
def body(p, skirt):
    m.meshes = []
    T = U.transform(p)
    for i in range(2):
        hip = T([(-.22 if i == 0 else .22), 0, .77])
        knee, foot = V(p['knees'][i]), V(p['feet'][i])
        if not skirt:
            m.rod(hip, knee, .14, BROWN)
            m.rod(knee, foot + [0, 0, .1], .11, BROWN)
        m.rod(foot + [0, 0, .1], foot, .09, BROWN)
        m.ell(foot + [0, -.035, 0], [.13, .2, .1], CORD)
    if skirt:
        m.ell(T([0, -.01, .42]), [.47, .35, .44], BROWN)
        m.ell(T([0, .0, .16]), [.54, .42, .2], BROWN)
        m.ell(T([0, -.02, .78]), [.4, .3, .3], BROWN)
    m.ell(T([0, 0, 1.0]), [.37, .26, .32], HEMP)
    m.ell(T([0, 0, 1.26]), [.35, .24, .36], HEMP)
    m.box(T([0, 0, .98]), [.7, .5, .07], CORD)
    m.rod(T([0, 0, 1.48]), T([0, 0, 1.68]), .13, SKIN)
    for i in range(2):
        sh, el, ha = T([(-.38 if i == 0 else .38), 0, 1.4]), T(p['elbows'][i]), T(p['hands'][i])
        m.ell(sh, [.16, .19, .16], HEMP)
        m.rod(sh, el, .12, HEMP)
        m.rod(el, ha, .115, SKIN)
        m.ell(ha, [.11]*3, SKIN)
    return T


def pack(T, w, h, color):
    """A bundle on the back (the +y side) held by two shoulder straps."""
    m.ell(T([0, .34, 1.1]), [w, .2, h], color)
    m.ell(T([0, .3, 1.4]), [w*.8, .16, .1], color)
    for sx in (-.27, .27):
        m.rod(T([sx, .0, 1.5]), T([sx*.8, .32, 1.0]), .05, COLLAR)
        m.rod(T([sx, .0, 1.5]), T([sx*1.05, -.24, 1.0]), .05, COLLAR)


_transform = U.transform


def transform(p):
    """units.transform, and with p['hunch'] (radians) everything above the waist bent forward about it: a bowed back
    with the skirt and legs still upright (lean shears the whole figure from the knees)."""
    T, a = _transform(p), p.get('hunch')
    if not a:
        return T
    c, s, w = math.cos(a), math.sin(a), 1.0

    def bent(q):
        x, y, z = q
        if z > w:
            y, z = y*c - (z - w)*s, w + (z - w)*c + y*s
        return T([x, y, z])
    return bent


U.transform = transform


def make(skirt, bundle, stoop=0.0, cane=False, behind=False, hunch=0.0, **head):
    SC.make()
    heads = Hd.heads(grids(**head))

    def figure(p):
        p['lean'] = p.get('lean', 0) + stoop
        p['hunch'] = hunch
        if cane:
            p['hands'][0], p['elbows'][0] = [-.52, -.24, .86], [-.45, -.04, 1.1]
        if behind:  # the other hand rests on the small of the back
            p['hands'][1], p['elbows'][1] = [.12, .34, 1.0], [.44, .2, 1.12]
        T = body(p, skirt)
        if behind:  # a rounded upper back
            m.ell(T([0, .16, 1.32]), [.33, .2, .3], HEMP)
        if bundle:
            pack(T, *bundle)
        if cane:  # from the hand straight down to the ground
            top = T([-.52, -.24, 1.0])
            m.rod(top, V([top[0] - .08, top[1] - .12, 0]), .045, BROWN)
        return T

    def pose_for(row, f):
        return U.hit_pose(f, U.pose(0, 0)) if row == U.HIT else U.pose(row, f)

    def prone(f):
        """Dead face down: arms loose along the sides, legs a little apart, no breathing. A cane lies on the
        ground beside the body."""
        p = U.pose(0, 0)
        p['hands'] = [[-.62, -.1, .82], [.66, -.1, .86]]
        p['elbows'] = [[-.58, -.06, 1.12], [.62, -.06, 1.14]]
        p['knees'] = [[-.27, -.04, .46], [.29, -.04, .46]]
        p['feet'] = [[-.3, 0, .13], [.34, 0, .13]] if not skirt else [[-.2, 0, -.12], [.24, 0, -.1]]  # out past the hem
        T = body(p, skirt)
        if bundle:
            pack(T, *bundle)
        pivot, dy = .3, .9
        yaw = prone_yaw()
        faces = tip_forward(m.meshes, pivot, dy, yaw)
        if cane:
            m.meshes = []
            m.rod(V([.85, -.9, .05]), V([.7, .55, .05]), .045, BROWN)
            faces += turn(m.meshes, yaw)
        head = tip_forward([(V([T(U.HEAD)]), None)], pivot, dy, yaw)[0][0][0]
        return U.result(faces, head, 0, f)

    def build(row, f, style=None):
        if row == 'prone':
            return prone(f)
        if row == B.BLOCK:
            k = Bl.SWORD[Bl.SELECTED]['frames'][f]
            T = figure(Bl.foot_pose(k))
            return Bl.displace(U.result(m.meshes, T(U.HEAD), 0, f), k.get('sway', 0), k.get('back', 0))
        T = figure(pose_for(row, f))
        return U.result(m.meshes, T(U.HEAD), row, f)
    return build, dict(heads, poses={'prone': prone_heads(heads)})


M.register('commoner_man', lambda: make(False, (.34, .36, BROWN), kind='man'))
M.register('commoner_woman', lambda: make(True, (.3, .3, KID), kind='woman'))
M.register('commoner_oldwoman', lambda: make(True, None, cane=True, hunch=.6, behind=True, kind='old_woman'))
M.register('commoner_elder', lambda: make(False, (.26, .26, KID), cane=True, stoop=.12, kind='elder', beard='long'))
