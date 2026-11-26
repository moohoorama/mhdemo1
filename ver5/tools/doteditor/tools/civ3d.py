#!/usr/bin/env python3
"""Non-combat (<id>_noncombat) dot sheets from the 3D civilian models of tools/unit3d (scripts DO generate the pixels).

  python3 civ3d.py caocao sunjian ...        # build units/<key>_noncombat.yaml; --list shows the keys
  python3 civ3d.py --list
  python3 civ3d.py --poses liubei guanyu     # add the scene poses (sit …) to existing sheets, other rows untouched

People and poses beyond standing (officials, eunuch, wounded, helper pair, stretcher, prone crossbowman; sit,
throne, bow, prostrate, lie, sitground) come from tools/unit3d/scene_people.py.

Per unit: `grid.py new light_infantry <id> --noncombat`, the tag colours of the battle sheet (BATTLE) copied in, the
3D scene model rendered for NW/SW (idle, two walk steps, one action = the held salute), keys mapped to tags, the
grid written and `grid.py import` run. Head pixels (2D heads of tools/unit3d/heads.py) and body pixels are mapped
separately because hair and leather share render keys. The 3D walk swings the arms against the legs (civilian.walk_arms).
Add a look: a `scene(...)` entry in LOOKS (kit parts and head like tools/unit3d/civilian.py), its battle sheet in
BATTLE (whose tag colours it takes) and a NAME. Then `python3 ../spritebuild/build.py units/<id>_noncombat.yaml`
and set `Look.Scene` in assets/data/characters.json (docs/presentation-sprite.md). A mounted class without a scene
sprite of its own stands in `Scene` of its class (classes.json).
"""
import copy
import math
import re
import subprocess
import sys
from pathlib import Path

import numpy as np

HERE = Path(__file__).resolve().parent
DE = HERE.parent
sys.path.insert(0, str(DE.parents[2] / 'tools/unit3d'))
sys.path.append(str(HERE))  # after unit3d: both folders have a render.py
import build as B  # noqa: E402
import civilian as C  # noqa: E402
import dots  # noqa: E402
import heads as Hd  # noqa: E402
import looks as L  # noqa: E402
import render as R  # noqa: E402
import units as U  # noqa: E402
import scene_people as SP  # noqa: E402
from story_heads import DARK_TOPKNOT, bearded, cap, helmet  # noqa: E402

W, H = 56, 46
LEATHER = U.BROWN
HAIR = [f'hair{i + 1}={c}' for i, c in enumerate(['#5a4638', '#352920', '#1c1614'])]  # base sheet hair is maroon


def scene(key, n, head, **kit):
    walk = C.walker(key, n, **kit)
    build = SP.posed(walk, walk.kit)  # the civilian rows, then the scene poses (sit …)
    build.kit = walk.kit
    return build, head


def chosen(key, look):
    """The user's civilian pick (civilian.LOOKS / SELECTED) with the scene poses."""
    _, walk, head = C.LOOKS[look][C.SELECTED[look]]
    kit = walk.kit
    if key in SP.POSE_KITS:  # the sheet colours the robe in other tags than the civilian kit
        kit = f'{kit}_pose'
        U.KITS[kit] = dict(U.KITS[walk.kit], **SP.POSE_KITS[key])
    build = SP.posed(walk, kit)
    build.kit = kit
    return build, head


LOOKS = {
    'caocao': chosen('caocao', 'caocao3'),
    'sunjian': scene('sunjian', 1, Hd.heads(L.bearded(Hd.TOPKNOT, 'bristle'), Hd.BAND), cloth=U.BLUE, guard=LEATHER, tabard=LEATHER),
    'yuanshao': scene('yuanshao', 9, Hd.heads(L.bearded(L.GOLD_IRON, 'goatee'), Hd.PLUME), cloth=U.BLUE, robe=U.BLUE, guard=U.GOLD, tabard=U.GOLD),
    'zhaoyun': scene('zhaoyun', 9, Hd.heads(Hd.IRON), cloth=U.BLUE, guard=C.STEEL),
    'dongzhuo': scene('dongzhuo', 9, Hd.heads(L.bearded(L.CAP, 'bushy')), cloth=LEATHER, robe=LEATHER, guard=U.GOLD, tabard=U.GOLD, belly=True),
    'infantry': scene('infantry', 1, Hd.heads(L.CAP), robe=U.BLUE, guard=U.BLUE),
}
for _key in ('yuanshu', 'gongsunzan', 'liubei', 'guanyu', 'zhangfei'):  # the user's civilian picks (civilian.SELECTED)
    LOOKS[_key] = chosen(_key, _key)
LOOKS.update(SP.LOOKS)


# ---- story part 1 cast (2026-10-23) ---------------------------------------------------------------------------

def shades(hex_color):
    """light, base, dark like the game's sprite.Shades: a character's Look.Palette colour as the sheet's own tags."""
    rgb = [int(hex_color[i:i + 2], 16) for i in (1, 3, 5)]
    mix = lambda to, n: '#' + ''.join(f'{(v*(100 - n) + to*n)//100:02x}' for v in rgb)  # noqa: E731
    return [mix(255, 30), hex_color, mix(0, 30)]


def tags(group, colors):
    return [f'{group}{i + 1}={c}' for i, c in enumerate(colors)]


NO_PLUME = {d: Hd.recolor(r, {'b': '2', 'c': '3', 'd': '3'}) for d, r in Hd.IRON.items()}  # a plain knob on top
WHITE_BAND = {'k': '4', 'l': '5', 'm': '6'}  # the topknot band in plain white cloth
_soldier = U.soldier


def soldier(p, kit, weapon=True):
    """units.soldier, plus an apron (kit 'apron': its colour) hanging from the sash to the knees with a bib, a cape
    (kit 'cape') hanging down the back from the shoulders to the calves, and a feather fan (kit 'fan': its colour)
    held up in the right hand in front of the chest, below the face."""
    faces, head = _soldier(p, kit, weapon)
    T = U.transform(p)
    apron = U.KITS[kit].get('apron')
    if apron:
        U.m.box(T([0, -.3, .7]), [.58, .05, .62], apron)
        U.m.box(T([0, -.29, 1.2]), [.36, .04, .32], apron)
    cape = U.KITS[kit].get('cape')
    if cape:
        U.m.box(T([0, .29, 1.03]), [.7, .05, .84], cape)
        U.m.box(T([0, .22, 1.42]), [.7, .12, .1], cape)
    fan = U.KITS[kit].get('fan')
    if fan:
        hand = T(p['hands'][1])  # the hand on the near side in SW/SE: in the other the body hides the fan
        d = U.V([-.4, -.3, .85]) / math.hypot(.4, .3, .85)  # up across the chest
        side = U.V([.9, 0, .43]) / math.hypot(.9, .43)  # across d, so the fan faces front and back
        U.m.rod(hand - d*.08, hand + d*.16, .04, LEATHER)
        base = hand + d*.15 + U.V([0, -.25, 0])  # held out before the chest and beard
        U.m.poly([base] + [base + (d*math.cos(a) + side*math.sin(a))*.55 for a in np.linspace(-.9, .9, 7)], fan)
        U.m.ell(base, [.06]*3, U.GOLD)
    return U.m.meshes, head


U.soldier = soldier


def work(look, name, pose):
    """A look with one more row, name: a work motion drawn by pose(kit) -> (pose, extra meshes(T, p))."""
    build, head = look
    rows = build.rows + [name]

    def b(row, f, style=None):
        if rows[row] != name:
            return build(row, f, style)
        p, extra = pose()
        faces, h = U.soldier(p, build.kit, weapon=False)
        extra(U.transform(p), p)
        return U.result(U.m.meshes, h, 0, f)
    b.rows, b.kit = rows, build.kit
    return b, head


def rest_pose(**hands):
    p = U.pose(0, 0)
    p.update(copy.deepcopy(C.REST))
    p.update(hands)
    return p


def hammer():
    """The smith's hammer raised beside the head, the other hand holding the tongs forward."""
    p = rest_pose(hands=[[-.62, -.22, 1.82], [.3, -.5, 1.05]], elbows=[[-.66, -.05, 1.5], [.5, -.2, 1.15]])
    p['lean'] = .06

    def extra(T, p):
        h = T(p['hands'][0])
        U.m.rod(h + [0, .02, -.1], h + [-.04, .1, .36], .05, U.SHAFT)
        U.m.box(h + [-.04, .12, .4], [.22, .36, .2], U.STEEL)
        t = T(p['hands'][1])
        U.m.rod(t, t + [0, -.42, -.12], .035, C.DARK)
    return p, extra


def serve():
    """A bowl of soup held out in both hands."""
    p = rest_pose(hands=[[-.17, -.55, 1.2], [.17, -.55, 1.2]], elbows=[[-.46, -.25, 1.15], [.46, -.25, 1.15]])

    def extra(T, p):
        c = (T(p['hands'][0]) + T(p['hands'][1])) / 2 + U.V([0, -.04, .08])
        U.m.ell(c, [.24, .24, .11], SP.STOOL)
        U.m.ell(c + U.V([0, 0, .07]), [.19, .19, .03], SP.IVORY)
    return p, extra


# 삿갓: a low, wide straw cone in the themeC keys (m sunlit half, l shaded half, k the rim's near edge), drawn from
# its outline so the sides run straight from the apex to the rim; the head grid grows to 23 columns and 3 rows on top
SATGAT_W, SATGAT_PAD, SATGAT_TOP = 23, 5, 3


def satgat_mask(cx, cy, tilt, a=10.5, b=2.4, h=6.0):
    """The hat seen from above: rim ellipse (a, b) round (cx, cy), the apex h above it, the sides the apex's tangents
    to the rim, the whole turned by tilt (negative lowers the left side). {(x, y): key}, outline included."""
    c, s = math.cos(tilt), math.sin(tilt)
    tv, tu = -b*b/h, a*math.sqrt(1 - (b/h)**2)
    out = {}
    for y in range(-SATGAT_TOP, 20):
        for x in range(SATGAT_W):
            dx, dy = x + .5 - cx, y + .5 - cy
            u, v = dx*c + dy*s, -dx*s + dy*c
            inside = (u/a)**2 + (v/b)**2 <= 1 or -h <= v <= tv and abs(u) <= tu*(v + h)/(tv + h)
            if inside:
                near = v > 0 and (u/a)**2 + ((v + 1)/b)**2 > 1
                out[x, y] = 'k' if near else ('m' if u < -.3 else 'l')
    for x, y in [p for p in out if any((p[0] + dx, p[1] + dy) not in out for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)))]:
        out[x, y] = '0'
    return out


def satgat(grids, rim, tilt=0.0):
    """The head under a 삿갓 whose rim centre lies on head row rim: the head above the hat's outline is cleared."""
    out = {}
    for d, rows in grids.items():
        n = len(rows) + SATGAT_TOP
        g = [list('.' * SATGAT_W) for _ in range(n)]
        for y, r in enumerate(rows):
            g[y + SATGAT_TOP][SATGAT_PAD:SATGAT_PAD + len(r)] = r
        mask = satgat_mask(SATGAT_PAD + 6.5, rim + SATGAT_TOP + .5, tilt)
        for x in range(SATGAT_W):
            under = max((y for (mx, y) in mask if mx == x), default=-1)
            for y in range(min(under, rim + SATGAT_TOP)):
                g[y][x] = '.'
        for (x, y), k in mask.items():
            if 0 <= y < n:
                g[y][x] = k
        out[d] = [''.join(r) for r in g]
    return out


def satgat_heads(rim, tilt=0.0):
    head = Hd.heads(satgat(L.bearded(L.CAP, 'goatee'), rim, tilt))
    return dict(head, anchor=(Hd.ANCHOR[0] + SATGAT_PAD, Hd.ANCHOR[1] + SATGAT_TOP))


GREY = tags('hair', ['#d6d4ce', '#9e9c96', '#605e5a'])
WHITE_HAIR = tags('hair', ['#f0eee8', '#c2c0ba', '#7c7a76'])
STORY = {
    # 노식: the light infantry helmet (its themeA leather, black by his palette) with the team plume, a long white
    # beard; the battle sheet's faction tunic under a black (themeA) robe
    'luzhi': ('노식', scene('luzhi', 1, Hd.heads(bearded(Hd.IRON, 'long', 'h'), Hd.PLUME),
                          cloth=U.BLUE, robe=LEATHER, guard=LEATHER, tabard=U.BLUE),
              'light_infantry', WHITE_HAIR + tags('themeA', shades('#2b2b33')) + tags('iron', shades('#e0c060'))),
    # 황보숭: the same helmet (dark red by his palette), black goatee, dark red coat, silver shoulders, gold belt
    'huangfusong': ('황보숭', scene('huangfusong', 1, Hd.heads(bearded(Hd.IRON, 'goatee', 'g'), Hd.PLUME),
                                  cloth=LEATHER, guard=U.STEEL, tabard=U.RED),
                    'light_infantry', HAIR + tags('themeA', shades('#7a1f1f')) + tags('themeB', shades('#d0a030'))
                    + tags('iron', shades('#e8e8f0'))),
    # 하후돈: the light cavalry hood in the team colour, bristling beard, team tunic, gilded shoulders (his iron),
    # red sash (his saddle red)
    'xiahoudun': ('하후돈', scene('xiahoudun', 1, Hd.heads(bearded(Hd.HOOD, 'bristle', 'g'), Hd.HOOD_CLOTH),
                                cloth=U.BLUE, guard=U.STEEL, tabard=U.RED),
                  'light_cavalry', HAIR + tags('iron', shades('#e0c060')) + tags('themeB', shades('#d03030'))),
    # lords and officials, told apart by the robe (themeB) colour: 유언 purple with a gold belt (imperial clansman,
    # governor), 도겸 crimson, 공융 dark green with a white collar
    'liuyan': ('유언', scene('liuyan', 1, Hd.heads(bearded(L.CAP, 'long', 'h')),
                           cloth=U.RED, robe=U.RED, guard=U.RED, tabard=U.GOLD),
               'light_infantry', WHITE_HAIR + tags('themeB', ['#9a72b4', '#6a4386', '#3e2452'])),
    'taoqian': ('도겸', scene('taoqian', 1, Hd.heads(bearded(SP.JINXIAN, 'long', 'g'), crest=2),
                            cloth=U.RED, robe=U.RED, guard=U.RED, tabard=U.GOLD),
                'light_infantry', GREY + tags('themeB', ['#d0605a', '#a03434', '#621e22'])),
    'kongrong': ('공융', scene('kongrong', 1, Hd.heads(bearded(SP.JINXIAN, 'long', 'g'), crest=2),
                             cloth=U.RED, robe=U.RED, guard=L.WHITE, tabard=U.BLACK),
                 'light_infantry', HAIR + tags('themeB', ['#5f9a68', '#356b42', '#1e4228'])),
    # 간옹: Liu Bei's friend from Zhuo, a scholar at ease — a white cloth kerchief over the hair, a short beard, a
    # loose green robe (themeB) over a white inner robe, a dark sash; his action raises a cup (the portrait)
    'jianyong': ('간옹', scene('jianyong', 1, Hd.heads(bearded(Hd.TOPKNOT, 'goatee', 'g'), WHITE_BAND),
                             cloth=U.RED, robe=U.RED, guard=L.WHITE, tabard=LEATHER),
                 'light_infantry', HAIR + tags('themeB', ['#7fa77a', '#4f7a52', '#2e4d34'])),
    # 추정: a Han commandant, iron helmet with the team plume, team tunic, steel shoulders, leather belt
    'zoujing': ('추정', scene('zoujing', 1, Hd.heads(bearded(Hd.IRON, 'goatee', 'g'), Hd.PLUME),
                            cloth=U.BLUE, guard=U.STEEL, tabard=LEATHER),
                'light_infantry', HAIR),
    # 고순: plain militia armour, an iron helmet without plume, leather over a drab coat, no beard
    'gaoshun': ('고순', scene('gaoshun', 1, Hd.heads(NO_PLUME), cloth=U.RED, guard=LEATHER, tabard=LEATHER),
                'light_infantry', HAIR + tags('themeB', ['#8c8272', '#5e574c', '#3a352e'])
                + tags('themeA', ['#9a6a42', '#6c4528', '#442a18'])),
    # 조조 in disguise (5장 주막, 관문): the 삿갓 pulled low over his goatee face, a plain grey-brown hemp robe
    # (themeB) and a leather belt instead of the Wei blue
    'caocao_hat': ('조조', scene('caocao_hat', 1, satgat_heads(4),
                                 cloth=U.RED, robe=U.RED, guard=U.RED, tabard=LEATHER),
                   'light_infantry', HAIR + tags('themeB', ['#9a917c', '#6c6555', '#454034'])
                   + tags('themeC', ['#e0c98a', '#b8995a', '#7a6236'])),
    # the same with the 삿갓 taken off (주막, after he is recognised): his scholar's cap over the hemp robe
    'caocao_plain': ('조조', scene('caocao_plain', 1, Hd.heads(L.bearded(L.CAP, 'goatee')),
                                   cloth=U.RED, robe=U.RED, guard=U.RED, tabard=LEATHER),
                     'light_infantry', HAIR + tags('themeB', ['#9a917c', '#6c6555', '#454034'])),
    # extras (aprons in a theme colour of their own: themeC)
    'jailer': ('옥졸', scene('jailer', 1, Hd.heads(L.CAP), cloth=U.RED, guard=LEATHER, tabard=U.GOLD),
               'light_infantry', HAIR + tags('themeB', ['#76808c', '#4e5660', '#30353c'])
               + tags('themeC', ['#d0503c', '#a0302a', '#6a1e1e'])),
    # 국밥집 주인: an unbleached cotton apron over a brown-grey coat
    'innkeeper': ('국밥집 주인', work(scene('innkeeper', 1, Hd.heads(Hd.HAIR), cloth=U.RED, guard=U.RED, tabard=LEATHER,
                                       apron=U.GOLD), 'serve', serve),
                  'light_infantry', HAIR + tags('themeB', ['#a08a6c', '#76624a', '#4e4030'])
                  + tags('themeC', ['#efe3bf', '#d6c495', '#a8936a'])),
    # 대장장이: a tan leather apron, darker leather shoulders and boots, bare arms
    'blacksmith': ('대장장이', work(scene('blacksmith', 1, Hd.heads(bearded(Hd.TOPKNOT, 'bristle', 'g'), WHITE_BAND),
                                      cloth=U.RED, guard=LEATHER, tabard=LEATHER, arms=U.SKIN, apron=U.GOLD),
                                'hammer', hammer),
                   'light_infantry', HAIR + tags('themeB', ['#7e7468', '#544c44', '#34302c'])
                   + tags('themeC', ['#c49a64', '#946c40', '#624426'])),
}
# ---- later cast (2026-10-30): no battle sheet of their own yet, so the look is set here ------------------------
# 제갈량·사마의·육손·주유 follow their 삼국지 14 portraits (2026-10-30). Colours beyond the five sheet groups go to the
# horse tags, unused on a sheet without a horse: XTRA renders in N/O/P (-> horseSkin), MANE in K/L/M (-> horseMane)
# and SADDLE in Q/R/S (-> saddle); KEY_TAGS maps those render keys for these units only.
XTRA, MANE, SADDLE = '#5a3a7a', '#a0382a', '#4ab0b0'
FAN = '#e8e8f0'  # feathers, in the MANE keys; a weapon colour, so it is drawn back over the 2D head (the beard)
R.add_ramps({XTRA: 'NOPP', MANE: 'KLMM', SADDLE: 'QRSS', FAN: 'KLMM'})
U.WEAPON_COLORS.add(FAN)
EXTRA_KEYS = {'N': 'x', 'O': 'w', 'P': 'v', 'K': 'A', 'L': 'z', 'M': 'y', 'Q': 'D', 'R': 'C', 'S': 'B'}


STORY.update({
    # 제갈량: the teal cap with a gold band, a goatee, the white robe with blue lapels (themeA, the shoes and the fan's
    # handle too), a teal ribbon at the neck (iron), a purple sash (XTRA), the crane-feather fan in a whiter white
    # with grey shading (FAN) so it stands off the robe
    'zhugeliang': ('제갈량', scene('zhugeliang', 1, Hd.heads(bearded(cap('45', 'lm'), 'goatee', 'g')),
                                 cloth=U.RED, robe=U.RED, guard=LEATHER, shoulder=U.RED, tabard=XTRA, scarf=U.STEEL,
                                 fan=FAN),
                   'light_infantry', HAIR + tags('themeB', ['#f6f2e6', '#d6cfbc', '#9a927e'])
                   + tags('themeA', ['#6a8cc8', '#3a5a96', '#1e3060']) + tags('iron', ['#86dcd2', '#3aa09a', '#1e6260'])
                   + tags('themeC', ['#f0d070', '#c8a040', '#806020'])
                   + tags('horseSkin', ['#8a7aa0', '#5e4e78', '#3a2e50'])
                   + tags('horseMane', ['#ffffff', '#c8ccd8', '#7a7e90'])),
    # 사마의: the dark red-purple cap (SADDLE) down to the brow (a gold band read as bare forehead), grey hair, a long
    # black beard (in the black robe's themeB, the hair tags being grey), a black robe under a dark bronze
    # coat (sleeves, shoulders, skirt: themeA), the blue-purple collar band down the front (iron), a maroon belt (XTRA)
    'simayi': ('사마의', scene('simayi', 1, Hd.heads(bearded(cap('RS', 'RR'), 'long', 'd')),
                             cloth=U.RED, robe=LEATHER, arms=LEATHER, shoulder=LEATHER, guard=U.STEEL, tabard=XTRA,
                             scarf=U.STEEL),
               'light_infantry', GREY + tags('themeB', ['#4a4a52', '#2a2a30', '#16161a'])
               + tags('themeA', ['#8a7a5a', '#5e5038', '#3a3022']) + tags('iron', ['#a4a4e4', '#6262b4', '#38387c'])
               + tags('themeC', ['#f0d070', '#c8a040', '#806020'])
               + tags('horseSkin', ['#a04a5a', '#70303e', '#481e28'])
               + tags('saddle', ['#7a4466', '#58304c', '#381c30'])),
    # 육손: the blue-domed helmet with a yellow plume, beardless; silver scale chest and shoulders (iron), brown sleeves
    # and trousers (themeA), navy skirt and belt (themeB), a green cape and collar (XTRA)
    'luxun': ('육손', scene('luxun', 1, Hd.heads(helmet('bcd', 'klm')), cloth=LEATHER, arms=LEATHER,
                          guard=U.STEEL, shoulder=U.STEEL, robe=U.RED, tabard=U.RED, cape=XTRA, scarf=XTRA, scarf_tail=.1),
              'light_infantry', HAIR + tags('themeA', ['#a08058', '#74583a', '#4a3622'])
              + tags('themeB', ['#5a6aa8', '#34447a', '#1e284a']) + tags('iron', ['#e0e4ea', '#a0a8b4', '#5e6670'])
              + tags('themeC', ['#ffe680', '#e8c030', '#a07818'])
              + tags('horseSkin', ['#7ab85a', '#4a8a3a', '#2a5a24'])),
    # 주유: the red-domed helmet (MANE) with a teal plume (SADDLE), beardless; white robe (themeB), silver chest
    # (iron), gold shoulders, a dark belt and boots (themeA), a purple cape and collar (XTRA)
    'zhouyu': ('주유', scene('zhouyu', 1, Hd.heads(helmet('KLM', 'QRS')), cloth=U.RED, robe=U.RED,
                           guard=U.STEEL, shoulder=U.GOLD, tabard=LEATHER, cape=XTRA, scarf=XTRA, scarf_tail=.1),
               'light_infantry', HAIR + tags('themeB', ['#f6f2e6', '#d6cfbc', '#9a927e'])
               + tags('themeA', ['#7a5a40', '#523a28', '#30221a']) + tags('iron', ['#e0e4ea', '#a0a8b4', '#5e6670'])
               + tags('themeC', ['#f0d070', '#c8a040', '#806020'])
               + tags('horseSkin', ['#9a78c8', '#6a4a9a', '#40286a'])
               + tags('horseMane', ['#d0604a', '#a0382a', '#682018']) + tags('saddle', ['#8ae0dc', '#4ab0b0', '#2a7276'])),
    # 황충: the old general off duty — white topknot, long white beard, team tunic, steel shoulders, red sash
    'huangzhong': ('황충', scene('huangzhong', 1, Hd.heads(bearded(Hd.HAIR, 'long', 'h')),
                               cloth=U.BLUE, guard=U.STEEL, tabard=U.RED),
                   'light_infantry', WHITE_HAIR + tags('themeB', ['#c8604a', '#963a2c', '#5e221c'])),
    # 장료: dark iron helmet with the team plume (his portrait), a moustache, team tunic under his navy coat
    'zhangliao': ('장료', scene('zhangliao', 1, Hd.heads(bearded(L.DARK_IRON, 'goatee', 'g'), Hd.PLUME),
                              cloth=U.BLUE, robe=U.RED, guard=U.STEEL, tabard=LEATHER),
                  'light_infantry', HAIR + tags('themeB', ['#5a6a8c', '#3a4a6a', '#222c44'])),
    # 여몽: a soldier risen from the ranks — bare topknot, bristling beard, team tunic, leather shoulders and belt
    'lumeng': ('여몽', scene('lumeng', 1, Hd.heads(bearded(DARK_TOPKNOT, 'bristle', 'g')), cloth=U.BLUE, guard=LEATHER,
                           tabard=LEATHER),
               'light_infantry', HAIR + tags('themeA', ['#9a6a42', '#6c4528', '#442a18'])),
    # 손권: 紫髯 — dark purple hair and beard, the gold crown, a crimson robe over the team tunic, gold belt
    'sunquan': ('손권', scene('sunquan', 1, Hd.heads(bearded(Hd.CROWN, 'goatee', 'g')), cloth=U.BLUE, robe=U.RED,
                            guard=U.RED, tabard=U.GOLD),
                'light_infantry', tags('hair', ['#7a4e86', '#4e2e5a', '#2c1834'])
                + tags('themeB', ['#b8484a', '#86282e', '#521a1e'])),
    # 여포 (2026-11-08): his battle sheet off the horse — the gold helmet with the two pheasant tail feathers (in his
    # bright red themeB, as on the sheet), beardless; team tunic under dark iron armour, gold shoulders and belt, a
    # red robe (themeB) and the red cape (themeA) down his back. His tags are the battle sheet's own.
    'lubu': ('여포', scene('lubu', 1, Hd.heads(Hd.crested(L.GOLD_IRON, {d: Hd.recolor(r, {'h': 'd', 'i': 'c'})
                                                                      for d, r in Hd.FEATHERS.items()}),
                                           crest=Hd.FEATHER_ROWS),
                         cloth=U.STEEL, robe=U.RED, guard=U.BLUE, shoulder=U.GOLD, tabard=U.GOLD, cape=LEATHER),
             'lubu', HAIR),
})
# 제갈현's nephews and niece (13장, 2026-11-04): children of a gentry house on the road, plain but decent and a little
# worn (tools/unit3d/scene_people.py child): the boy in a slate-blue coat and trousers, the girl in a dusty rose robe
# with red ribbons (themeC) in her buns; a dark brown belt and shoes (themeA), a white inner collar (iron)
KID_COLORS = HAIR + tags('themeA', ['#8a6a4a', '#5e4630', '#3a2a1c']) + tags('iron', ['#f2ede0', '#d2c9b4', '#958b76'])
STORY.update({
    'child_boy': ('소년', SP.child('child_boy', SP.BOY_HEAD, cloth=U.RED, guard=U.RED, tabard=LEATHER, scarf=C.WHITE,
                                 scarf_tail=.1),
                  'light_infantry', KID_COLORS + tags('themeB', ['#8494a8', '#5a687e', '#384254'])),
    'child_girl': ('소녀', SP.child('child_girl', SP.GIRL_HEAD, long_robe=True, cloth=U.RED, robe=U.RED, guard=U.RED,
                                  tabard=LEATHER, scarf=C.WHITE, scarf_tail=.1),
                   'light_infantry', KID_COLORS + tags('themeB', ['#c8a09a', '#9a706a', '#644644'])
                   + tags('themeC', ['#d86a5a', '#b04438', '#742a24'])),
})
STORY_POSES = {k: ['sit'] for k in ('luzhi', 'huangfusong', 'xiahoudun', 'liuyan', 'taoqian', 'kongrong', 'zoujing',
                                     'gaoshun', 'caocao_hat', 'caocao_plain', 'zhugeliang', 'simayi', 'huangzhong',
                                     'zhangliao', 'zhouyu', 'luxun', 'lumeng', 'sunquan', 'lubu')}
STORY_POSES['xiahoudun'] = ['sit', 'fist']
STORY_POSES['jianyong'] = ['sit']
STORY_POSES['taoqian'] = ['sit', 'bed']  # 도겸 on his sickbed (서주 양도, his last words)
STORY_POSES['liuyan'] = ['sit', 'bed']  # 유표 (who wears 유언's look) on his sickbed (25장, 형주를 맡기려 할 때)
STORY_POSES.update(child_boy=['bow'], child_girl=['bow'])
ACTION = {'jianyong': ('toast', 2), 'jailer': ('talk', 1), 'innkeeper': ('serve', 0), 'blacksmith': ('hammer', 0)}  # else the salute
for _key, (_name, _look, _battle, _colors) in STORY.items():
    LOOKS[_key] = _look
# 유비: in shu his green coat (themeB) and the team robe and sash were one green; gold shoulders and a leather belt as
# on his battle sheet (2026-10-23), and the white inner collar of his battle sheet (2026-10-27)
U.KITS[LOOKS['liubei'][0].kit].update(shoulder=U.GOLD, tabard=LEATHER, scarf=C.WHITE, scarf_tail=.12)
BATTLE = {'caocao': 'caocao', 'sunjian': 'sunjian', 'yuanshao': 'yuanshao', 'zhaoyun': 'zhaoyun', 'dongzhuo': 'light_cavalry', 'infantry': 'light_infantry',
          'yuanshu': 'yuanshu', 'gongsunzan': 'light_cavalry', 'liubei': 'liubei', 'guanyu': 'guanyu', 'zhangfei': 'zhangfei'}
BATTLE.update({k: 'light_infantry' for k in SP.LOOKS})
NAME = {'caocao': '조조', 'sunjian': '손견', 'yuanshao': '원소', 'zhaoyun': '조운', 'dongzhuo': '동탁', 'infantry': '일반보병',
        'yuanshu': '원술', 'gongsunzan': '공손찬', 'liubei': '유비', 'guanyu': '관우', 'zhangfei': '장비', **SP.NAMES}
EXTRA_COLORS = {'dongzhuo': HAIR + ['themeA1=#7a5a9a', 'themeA2=#523470', 'themeA3=#2e1a44'], 'infantry': HAIR,
                **{k: HAIR for k in SP.LOOKS}}
for _key, (_name, _look, _battle, _colors) in STORY.items():
    BATTLE[_key], NAME[_key], EXTRA_COLORS[_key] = _battle, _name, _colors
# 노식·황보숭: their light infantry helmet is the battle sheet's leather (themeA), not iron
HEAD_TAGS = {k: {'1': 'o', '2': 'o', '3': 'n', '4': 'm', '5': 'm'} for k in ('luzhi', 'huangfusong')}
KEY_TAGS = {k: EXTRA_KEYS for k in ('zhugeliang', 'simayi', 'luxun', 'zhouyu')}
# hand retouches laid over a rendered pose frame: (unit, pose, direction) -> [(row, first column, pixels)] in grid
# characters ('.' clears); 도겸's sleeping head (the modelled head is too small for its face to read unaided)
RETOUCH = {
    ('taoqian', 'bed', 'SW'): [
        (9, 29, '......................'),
        (10, 29, '................######..'),
        (11, 29, '..............##cccccc#.'),
        (12, 29, '............##aacccccccc#'),
        (13, 29, '.........###aaaaaccccccc#'),
        (14, 29, '......####ddaaaaaacccc####'),
        (15, 35, 'a###dde#daaaaccc'),  # the face as on the standing sheet: mid skin, a light band by the hair
        (16, 36, 'a#aeee#edaaabb#'),
        (17, 35, 'aa#aaeeee#ddeab#'),
        (18, 33, 'BBaaaaaeeee#deab#'),
        (19, 32, 'BBaaaaaaabeeeeeb#'),
        (20, 31, 'BBaaaaaaabbbeeeb#'),
        (21, 31, 'B#aaaaaabbbb###'),
        (22, 32, '##aabbb##BBDD'),
        (23, 34, '###'),
        # the eyes: two dots on a diagonal each (lower left, upper right), one above the other across the face
        (15, 41, 'e#'),
        (16, 39, 'ee#edd'),
        (17, 40, 'eeeeedde'),
        (18, 40, 'eeee#de'),
        (19, 42, 'e#eee')],
    ('taoqian', 'bed', 'NW'): [
        (21, 34, 'Baaaa##'),
        (22, 33, 'Baaaaaaa#b'),
        (23, 33, 'Caaaaaaaeed#'),
        (24, 34, 'baaaeee#dda#'),
        (25, 35, 'Dbaeee#ddaacccc#'),
        (26, 36, '#ee#eddaacccccc##'),
        (27, 36, '#ee#eddaaccccc#'),
        (28, 36, '#eeeedaaacccc#'),
        (29, 36, '#eeeaaabbcc#'),
        (30, 37, '#aabbbccc#'),
        (31, 37, '#bbb###'),
        # the eyes: two dots on a diagonal each, mirroring SW's (upper left, lower right)
        (24, 38, 'ee#edd'),
        (25, 38, 'eee#dd'),
        (26, 37, 'eeeedd'),
        (27, 37, 'e#eedd'),
        (28, 37, 'ee#ed')],
}
# 유표's sleeping head: the same cap, beard and face as 도겸's
RETOUCH[('liuyan', 'bed', 'SW')] = RETOUCH[('taoqian', 'bed', 'SW')]
RETOUCH[('liuyan', 'bed', 'NW')] = RETOUCH[('taoqian', 'bed', 'NW')]


# a head of its own for a pose: 조조 seated pulls the 삿갓 down and forward over his eyes
POSE_HEADS = {'caocao_hat': {'sit': satgat_heads(6, -.18)}}
# (3D animation, direction, 3D frame) -> (sheet animation, frame); walk steps are the two stride frames
FRAMES = [('idle', 'NW', 0, 'idle', 0), ('idle', 'SW', 0, 'idle', 0),
          ('walk', 'NW', 0, 'walk', 0), ('walk', 'NW', 2, 'walk', 1), ('walk', 'SW', 0, 'walk', 0), ('walk', 'SW', 2, 'walk', 1),
          ('salute', 'NW', 2, 'action', 0), ('salute', 'SW', 2, 'action', 0)]
BODY = {'0': '#', '1': 'c', '2': 'c', '3': 'b', '4': 'b', '5': 'a', '6': 'a', 'i': 'd', 'j': 'e', 'f': 'o', 'g': 'n', 'h': 'm',
        'k': 'u', 'l': 't', 'm': 's', 'W': 'l', 'X': 'l', 'Y': 'k', 'Z': 'j', 'A': 'r', 'B': 'q', 'C': 'p', 'b': 'r', 'c': 'q', 'd': 'p'}
BODY.update(zip(SP.BLANKET_KEYS, 'DCB'))  # the sickbed quilt (scene_people.sickbed) dark/mid/light -> saddle3/2/1
HEADMAP = dict(BODY, f='i', g='h', h='g')  # head hair dark/mid/light -> hair3/2/1
BED_COLORS = tags('saddle', ['#707ea4', '#4e5a7e', '#28304a'])  # the quilt, set on a sheet that gets the bed pose
BED_CELL = (64, 50, 32, 36)  # w, h, pivot x, y: a sheet with the bed pose is widened to hold the bed
LETTERS = 'abcdefghijklmnopqrstuvwxyzABCD#EFG'  # tag order of tag_colors


def nearest(key, colors):
    """Tag whose colour is closest to an unmapped render key (never faction, horse or effect tags); reported."""
    rgb = R.COLOR[key]
    return LETTERS[min((i for i in range(len(colors)) if LETTERS[i] in 'abcdefghimnopqrstu'),
                       key=lambda i: sum((a - int(colors[i][1 + 2*j:3 + 2*j], 16))**2 for j, a in enumerate(rgb)))]


def grid(*args):
    subprocess.run(['python3', 'grid.py', *args], cwd=HERE, check=True, capture_output=True)


def head_box(rows):
    """(top row, centre column) of a frame's head: the first 4 rows of drawn pixels."""
    top = next(y for y, r in enumerate(rows) if any(c != '.' for c in r))
    xs = [x for r in rows[top:top + 4] for x, c in enumerate(r) if c != '.']
    return top, round(sum(xs) / len(xs))


def sheet_head(uid, d, out, under):
    """Put the sheet's own idle head (12 rows, 15 columns) over a pose frame's rendered head. under: the frame drawn
    without its head, so the rendered head's pixels (and only those) go back to whatever lies behind them, and the
    new head lands on the old one's top even when a chair back rises above it."""
    tags = dots.frame(dots.load(uid), 'idle', d, 0)['tags']
    idle = [''.join(dots.TAG_CHARS[t - 1] if t else '.' for t in r) for r in tags]
    spots = [(y, x) for y in range(H) for x in range(W) if out[y][x] != under[y][x]]
    if not spots:  # hidden behind something
        return
    ty, cx = head_box(idle)
    ny = min(y for y, _ in spots)
    top = [x for y, x in spots if y < ny + 4]
    nx = round(sum(top) / len(top))
    for y, x in spots:
        out[y][x] = under[y][x]
    for y in range(12):
        for x in range(-7, 8):
            c = idle[ty + y][cx + x] if 0 <= ty + y < len(idle) and 0 <= cx + x < len(idle[0]) else '.'
            if c != '.' and 0 <= ny + y < H and 0 <= nx + x < W:
                out[ny + y][nx + x] = c


def to_tags(full, body, dx, dy, colors, unmapped, posed_unit, unit=None):
    """A rendered frame as tag characters, moved by (dx, dy): head pixels (where it differs from the headless body
    render) by HEADMAP (unit's HEAD_TAGS first), the rest by BODY. posed_unit: that unit's pose key tags come first."""
    out = [['.']*W for _ in range(H)]
    for y, row in enumerate(full):
        for x, c in enumerate(row):
            if c == '.':
                continue
            table = BODY if body[y][x] == c else HEADMAP
            tag = SP.POSE_KEY_TAGS.get(posed_unit, {}).get(c) or KEY_TAGS.get(unit, {}).get(c) or table.get(c)
            if table is HEADMAP:
                tag = HEAD_TAGS.get(unit, {}).get(c, tag)
            if tag is None:
                tag = unmapped[c, table is HEADMAP] = nearest(c, colors)
            if 0 <= x + dx < W and 0 <= y + dy < H:
                out[y + dy][x + dx] = tag
    return out


def make(unit, only_poses=False):
    """The whole sheet from the 3D look, or with only_poses just the scene poses added to an existing sheet."""
    uid = f'{unit}_noncombat'
    build, head = LOOKS[unit]
    poses = SP.SHEET_POSES.get(unit) or SP.OFFICER_POSES.get(unit) or STORY_POSES.get(unit, [])
    if not only_poses:
        grid('new', 'light_infantry', uid, NAME[unit], '--noncombat')
        doc = dots.with_alpha(dots.load(uid))
        doc['tag_colors'] = list(dots.load(BATTLE[unit])['tag_colors'])
        dots.save(doc)
        if unit in EXTRA_COLORS:
            grid('colors', uid, *EXTRA_COLORS[unit])
    if poses:
        grid('poses', uid, *poses)
    if unit in SP.POSE_COLORS:
        grid('colors', uid, *SP.POSE_COLORS[unit])
    resized = False
    if 'bed' in poses:
        grid('colors', uid, *BED_COLORS)
        if dots.load(uid)['unit']['cell'] != list(BED_CELL[:2]):
            grid('cell', uid, *map(str, BED_CELL))
            resized = True
    global W, H
    (W, H), (px, py) = (u := dots.load(uid)['unit'])['cell'], u['pivot']
    if only_poses and not resized:
        for pose in poses:
            grid('export', uid, '--anim', pose, '--force')
    else:
        grid('export', uid, '--force')
    colors = dots.load(uid)['tag_colors']
    rows = getattr(build, 'rows', C.ANIMATIONS)
    draw = getattr(build, 'frame', B.frame)
    frames = [] if only_poses else list(FRAMES)
    if unit in ACTION:  # a work motion instead of the salute
        frames = [(ACTION[unit][0], d, ACTION[unit][1], oa, on) if oa == 'action' else (a, d, n, oa, on)
                  for a, d, n, oa, on in frames]
    frames += [(pose, d, 0, pose, 0) for pose in poses for d in ('NW', 'SW')]
    blank = Hd.heads({d: ['.'*13]*12 for d in Hd.IRON})
    captured, to_img = [], Hd.to_img
    Hd.to_img = lambda g: (captured.append([r[:] for r in g]), to_img(g))[1]
    unmapped = {}

    def render(anim, d, n, heads):
        captured.clear()
        draw(build, heads, B.SPEC, d, rows.index(anim), n, lift=0)
        return captured[-1]

    ground = {d: max(y for y, r in enumerate(render('idle', d, 0, head)) for c in r if c != '.') for d in ('NW', 'SW')}
    for anim, d, n, out_anim, out_n in frames:
        own = SP.turned(head) if anim in SP.TURNED_HEAD else POSE_HEADS.get(unit, {}).get(anim, head)
        if anim in SP.MODELLED_HEAD:
            own = blank
        full, body = render(anim, d, n, own), render(anim, d, n, blank)
        dx, dy = px - B.SPEC['pivot'][0], py - ground[d]
        out, under = to_tags(full, body, dx, dy, colors, unmapped, unit if out_anim in poses else None, unit), None
        if out_anim in poses and unit in SP.POSE_SHEET_HEAD:
            under = to_tags(body, body, dx, dy, colors, unmapped, unit)
            sheet_head(uid, d, out, under)
        for y, x, run in RETOUCH.get((unit, out_anim, d), []):
            out[y][x:x + len(run)] = run
        path = DE / f'heroes/{uid}/grid/{out_anim}/{d}-{out_n}.txt'
        lines = path.read_text().split('\n')
        for i, line in enumerate(lines):
            if re.match(r'^\d\d\|', line):
                lines[i] = line[:3] + ''.join(out[int(line[:2])])
        path.write_text('\n'.join(lines))
    Hd.to_img = to_img
    grid('import', uid)
    print(unit, 'done', f'(unmapped render keys, nearest tag used: {unmapped})' if unmapped else '', flush=True)


if __name__ == '__main__':
    if '--list' in sys.argv or len(sys.argv) < 2:
        sys.exit('keys: ' + ' '.join(LOOKS))
    only = '--poses' in sys.argv
    for key in [a for a in sys.argv[1:] if not a.startswith('--')]:
        make(key, only_poses=only)
