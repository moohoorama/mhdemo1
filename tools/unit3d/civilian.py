"""Civilian (평복) looks for scenario scenes: officers on foot without weapons.

Battle sprites stay as they are; these are a second sprite per officer, used where
the story is told (dismounted, unarmed). Four candidates per officer; the user picks
one (SELECTED) and the rest stay as backups. Kits register into units.KITS as
'<key>_civ<n>'.

  python3 tools/unit3d/civilian_candidates.py  -> output/ver4-civilian-candidates.png
"""
import copy
import math

import numpy as np

import heads as Hd
import looks as L
import units as U

GREEN, DARK_GREEN, BLACK, DARK, HEMP = U.GREEN, U.DARK_GREEN, U.BLACK, U.DARK_STEEL, U.HEMP
WHITE, GOLD, BLUE, STEEL, RED = L.WHITE, U.GOLD, U.BLUE, U.STEEL, U.RED


def walker(key, n, **kit):
    name = f'{key}_civ{n}'
    U.KITS[name] = L.kit(**kit)
    build = U.foot_unit(name)
    build.kit = name  # chosen() renders the scene sheet with this kit
    return build


H = Hd.heads
DARK_HAIR = {d: Hd.beard(Hd.recolor(r, {'j': 'h', 'i': 'g'}), 'gh', 'bristle') for d, r in Hd.HAIR.items()}
# chosen look: swarthy (one step darker than plain skin) with the full bushy beard
BEARDED_HAIR = {d: Hd.beard(Hd.recolor(r, {'j': 'i', 'i': 'h'}), 'hi', 'bushy') for d, r in Hd.HAIR.items()}
DARK_HOOD = {d: Hd.beard(Hd.recolor(r, {'7': '0', '8': '1', '9': '2', 'a': '3', 'j': 'h', 'i': 'g'}), 'gh', 'bristle')
             for d, r in Hd.HOOD.items()}
GUANYU = H(Hd.GUANYU, skin='KL', lid='K')
ZHANGFEI_HELM = H(Hd.ZHANGFEI, Hd.PLUME, skin='hi', lid='h')
LIUBEI = H(L.eared(L.GOLD_CAP))

LOOKS = {
    'guanyu': [
        ('녹포 무장 · 도보', walker('guanyu', 1, cloth=GREEN, guard=DARK_GREEN), GUANYU),
        ('녹색 도포', walker('guanyu', 2, cloth=GREEN, robe=GREEN, guard=GREEN), GUANYU),
        ('녹색 도포 · 흰 속옷', walker('guanyu', 3, cloth=WHITE, robe=GREEN, guard=GREEN), GUANYU),
        ('녹포 · 철 갑옷 · 녹색 도포', walker('guanyu', 4, cloth=GREEN, robe=DARK_GREEN, guard=STEEL), GUANYU),
    ],
    'zhangfei': [
        ('흑갑 · 투구 · 도보', walker('zhangfei', 1, cloth=BLACK, guard=DARK), ZHANGFEI_HELM),
        ('진영색 도포 · 붉은 띠 · 맨상투 · 덥수룩한 수염', walker('zhangfei', 2, robe=BLUE, guard=BLUE, tabard=U.RED),
         H(BEARDED_HAIR, skin='hi', lid='h')),
        ('삼베 평복 · 맨상투', walker('zhangfei', 3, cloth=HEMP, guard=U.HEMP_DARK), H(DARK_HAIR, skin='gh', lid='g')),
        ('흑갑 · 검은 두건', walker('zhangfei', 4, cloth=BLACK, guard=DARK), H(DARK_HOOD, skin='gh', lid='g')),
    ],
    'liubei': [
        ('금빛 관모 · 진영색 도포 · 금빛 갑옷', walker('liubei', 1, robe=BLUE, guard=GOLD), LIUBEI),
        ('금빛 관모 · 진영색 도포', walker('liubei', 2, robe=BLUE, guard=BLUE), LIUBEI),
        ('금빛 관모 · 흰 도포', walker('liubei', 3, cloth=WHITE, robe=WHITE, guard=WHITE), LIUBEI),
        ('맨상투 · 삼베 평복 (돗자리 장수)', walker('liubei', 4, cloth=HEMP, guard=U.HEMP_DARK),
         H(L.bearded(Hd.HAIR, 'goatee'))),
    ],
}
# Scene candidates for the prologue and chapter 1 cast (2026-10-04). Team-key parts take each actor's
# faction in the scene (Cao Cao: wei); officers with a strong signature color use fixed colors.
BRICK, MAROON, LEATHER, PURPLE, SILVER, PALE = '#a14a35', '#7a2f22', '#704022', L.PURPLE, '#c9cdd0', '#dde1e3'
B = L.bearded
WHITE_PLUME = {'b': '4', 'c': '5', 'd': '6'}  # fixed white plume instead of the team key
LOOKS.update({
    'zhangshiping': [
        ('검은 관모 · 염소수염 · 벽돌색 비단 도포', walker('zhangshiping', 1, cloth=BRICK, robe=BRICK, guard=GOLD, tabard=GOLD), H(B(L.CAP, 'goatee'))),
        ('맨상투 · 염소수염 · 흰 도포', walker('zhangshiping', 2, cloth=WHITE, robe=WHITE, guard=LEATHER, tabard=LEATHER), H(B(Hd.HAIR, 'goatee'))),
        ('금빛 관모 · 긴 수염 · 적갈색 도포', walker('zhangshiping', 3, cloth=MAROON, robe=MAROON, guard=GOLD, tabard=GOLD), H(B(L.GOLD_CAP, 'long'))),
        ('검은 관모 · 녹색 도포', walker('zhangshiping', 4, cloth=GREEN, robe=GREEN, guard=LEATHER, tabard=GOLD), H(L.CAP)),
    ],
    'sushuang': [
        ('맨상투 · 벽돌색 옷', walker('sushuang', 1, cloth=BRICK, guard=LEATHER, tabard=GOLD), H(Hd.HAIR)),
        ('검은 관모 · 흰 도포', walker('sushuang', 2, cloth=WHITE, robe=WHITE, guard=BRICK, tabard=BRICK), H(L.CAP)),
        ('흰 윤건 · 적갈색 도포', walker('sushuang', 3, cloth=MAROON, robe=MAROON, guard=LEATHER, tabard=GOLD), H(L.WHITE_HOOD)),
        ('맨상투 · 녹색 옷 · 등짐', walker('sushuang', 4, cloth=GREEN, guard=LEATHER, tabard=LEATHER, pack=HEMP), H(Hd.HAIR)),
    ],
    'caocao': [
        ('검은 관모 · 염소수염 · 검은 도포 · 붉은 띠', walker('caocao', 1, cloth=BLACK, robe=BLACK, guard=RED, tabard=RED), H(B(L.CAP, 'goatee'))),
        ('금빛 관모 · 염소수염 · 적갈색 도포', walker('caocao', 2, cloth=MAROON, robe=MAROON, guard=GOLD, tabard=GOLD), H(B(L.GOLD_CAP, 'goatee'))),
        ('검은 관모 · 철 갑옷 · 검은 도포', walker('caocao', 3, cloth=BLACK, robe=BLACK, guard=STEEL, tabard=GOLD), H(B(L.CAP, 'goatee'))),
        ('검은 투구 · 검은 갑옷 · 붉은 띠', walker('caocao', 4, cloth=BLACK, guard=DARK, tabard=RED), H(B(L.DARK_IRON, 'goatee'), Hd.PLUME)),
    ],
    'caocao2': [  # second round: a red look (user, 2026-10-04)
        ('검은 관모 · 붉은 도포 · 검은 띠', walker('caocao', 5, cloth=RED, robe=RED, guard=BLACK, tabard=BLACK), H(B(L.CAP, 'goatee'))),
        ('금빛 관모 · 붉은 도포 · 금빛 어깨', walker('caocao', 6, cloth=RED, robe=RED, guard=GOLD, tabard=GOLD), H(B(L.GOLD_CAP, 'goatee'))),
        ('검은 관모 · 적갈색 옷 · 붉은 도포 · 철 어깨', walker('caocao', 7, cloth=MAROON, robe=RED, guard=STEEL, tabard=GOLD),
         H(B(L.CAP, 'goatee'))),
        ('검은 투구 · 붉은 갑옷 · 붉은 술', walker('caocao', 8, cloth=RED, guard=DARK, tabard=RED), H(B(L.DARK_IRON, 'goatee'))),
    ],
    'caocao3': [  # third round: Wei blue (team key; scenes draw him with the wei faction)
        ('검은 관모 · 청색 도포 · 검은 띠', walker('caocao', 9, cloth=BLUE, robe=BLUE, guard=BLACK, tabard=BLACK), H(B(L.CAP, 'goatee'))),
        ('검은 관모 · 청색 도포 · 금빛 어깨', walker('caocao', 10, cloth=BLUE, robe=BLUE, guard=GOLD, tabard=GOLD), H(B(L.CAP, 'goatee'))),
        ('검은 관모 · 청색 도포 · 철 어깨 · 붉은 띠', walker('caocao', 11, cloth=BLUE, robe=BLUE, guard=STEEL, tabard=RED),
         H(B(L.CAP, 'goatee'))),
        ('검은 투구 · 청색 옷 · 검은 갑옷', walker('caocao', 12, cloth=BLUE, guard=DARK, tabard=BLACK), H(B(L.DARK_IRON, 'goatee'), Hd.PLUME)),
    ],
    'yuanshao': [
        ('금빛 관모 · 흰 도포 · 금빛 어깨', walker('yuanshao', 1, cloth=WHITE, robe=WHITE, guard=GOLD, tabard=GOLD), H(B(L.GOLD_CAP, 'goatee'))),
        ('금빛 투구 · 흰 옷 · 금빛 갑옷', walker('yuanshao', 2, cloth=WHITE, guard=GOLD, tabard=GOLD), H(B(L.GOLD_IRON, 'goatee'), Hd.PLUME)),
        ('금빛 관모 · 금빛 도포 · 흰 띠', walker('yuanshao', 3, cloth=GOLD, robe=GOLD, guard=WHITE, tabard=WHITE), H(B(L.GOLD_CAP, 'goatee'))),
        ('금관 · 흰 도포 · 긴 수염', walker('yuanshao', 4, cloth=WHITE, robe=WHITE, guard=GOLD, tabard=GOLD), H(B(Hd.CROWN, 'long'))),
    ],
    'yuanshu': [
        ('금빛 관모 · 금빛 도포 · 붉은 띠', walker('yuanshu', 1, cloth=GOLD, robe=GOLD, guard=RED, tabard=RED), H(L.GOLD_CAP)),
        ('금관 · 금빛 도포 · 뻣뻣한 수염', walker('yuanshu', 2, cloth=GOLD, robe=GOLD, guard=MAROON, tabard=RED), H(B(Hd.CROWN, 'bristle'))),
        ('금빛 관모 · 적갈색 도포 · 금빛 어깨', walker('yuanshu', 3, cloth=MAROON, robe=MAROON, guard=GOLD, tabard=GOLD), H(B(L.GOLD_CAP, 'goatee'))),
        ('금빛 관모 · 붉은 도포 · 금빛 어깨', walker('yuanshu', 4, cloth=RED, robe=RED, guard=GOLD, tabard=GOLD), H(B(L.GOLD_CAP, 'goatee'))),
    ],
    'gongsunzan': [
        ('철 투구 · 흰 옷 · 은빛 갑옷 (백마장군)', walker('gongsunzan', 1, cloth=WHITE, guard=SILVER, tabard=PALE), H(B(Hd.IRON, 'goatee'), Hd.PLUME)),
        ('흰 윤건 · 흰 도포', walker('gongsunzan', 2, cloth=WHITE, robe=WHITE, guard=PALE, tabard=SILVER), H(B(L.WHITE_HOOD, 'goatee'))),
        ('검은 관모 · 흰 도포 · 철 어깨', walker('gongsunzan', 3, cloth=WHITE, robe=WHITE, guard=STEEL, tabard=STEEL), H(B(L.CAP, 'goatee'))),
        ('철 투구 · 은빛 옷 · 흰 갑옷', walker('gongsunzan', 4, cloth=SILVER, guard=PALE, tabard=WHITE), H(Hd.IRON, WHITE_PLUME)),
    ],
    'dongzhuo': [
        ('검은 관모 · 덥수룩한 수염 · 검은 도포 · 금빛 어깨 · 비대', walker('dongzhuo', 1, cloth=BLACK, robe=BLACK, guard=GOLD, tabard=GOLD, belly=True),
         H(B(L.CAP, 'bushy'))),
        ('금빛 관모 · 덥수룩한 수염 · 붉은 도포 · 비대', walker('dongzhuo', 2, cloth=RED, robe=RED, guard=GOLD, tabard=GOLD, belly=True),
         H(B(L.GOLD_CAP, 'bushy'))),
        ('맨상투 · 뻣뻣한 수염 · 적갈색 도포 · 비대', walker('dongzhuo', 3, cloth=MAROON, robe=MAROON, guard=GOLD, tabard=GOLD, belly=True),
         H(B(Hd.HAIR, 'bristle'))),
        ('검은 투구 · 덥수룩한 수염 · 검은 갑옷 · 비대', walker('dongzhuo', 4, cloth=BLACK, guard=DARK, tabard=GOLD, belly=True),
         H(B(L.DARK_IRON, 'bushy'), Hd.PLUME)),
    ],
    'liru': [
        ('검은 관모 · 염소수염 · 검은 도포', walker('liru', 1, cloth=BLACK, robe=BLACK, guard=PURPLE, tabard=PURPLE), H(B(L.CAP, 'goatee'))),
        ('검은 관모 · 적갈색 도포', walker('liru', 2, cloth=MAROON, robe=MAROON, guard=BLACK, tabard=BLACK), H(L.CAP)),
        ('흰 윤건 · 검은 도포', walker('liru', 3, cloth=BLACK, robe=BLACK, guard=STEEL, tabard=PURPLE), H(L.WHITE_HOOD)),
        ('검은 관모 · 긴 수염 · 회색 도포', walker('liru', 4, cloth='#5a5f66', robe='#5a5f66', guard=BLACK, tabard=PURPLE), H(B(L.CAP, 'long'))),
    ],
})
TITLES = {'guanyu': '관우', 'zhangfei': '장비', 'liubei': '유비', 'zhangshiping': '장세평', 'sushuang': '소쌍', 'caocao': '조조', 'caocao2': '조조', 'caocao3': '조조', 'yuanshao': '원소', 'yuanshu': '원술',
          'gongsunzan': '공손찬', 'dongzhuo': '동탁', 'liru': '이유'}
SELECTED = {'guanyu': 1, 'zhangfei': 1, 'liubei': 1,
            'zhangshiping': 3, 'sushuang': 3, 'yuanshao': 0, 'yuanshu': 1, 'gongsunzan': 3, 'dongzhuo': 0, 'liru': 3,
            'caocao3': 2}  # chosen by the user (2026-10-04; Cao Cao: Wei blue, steel shoulders, red sash); others are backups
# look key -> sprite key, for officers whose pick came from a later candidate round
SPRITE_KEYS = {'caocao3': 'caocao'}

# Officers who speak in scenes but had no candidate round: their battle look without the weapon.
EXTRA = {
    'jianyong': ('간옹', walker('jianyong', 1), H(L.WHITE_HOOD)),
    'sunqian': ('손건', walker('sunqian', 1, robe=BLUE, guard=BLUE), H(L.CAP)),
}


def from_battle(kit):
    """The chosen battle kit on foot, without weapon, shield or saddle bags."""
    k = {key: v for key, v in U.KITS[kit].items() if key not in ('weapon', 'shield', 'packs')}
    return dict(k, weapon=None, shield=False)


# Chapter 1 officers (2026-10-05): the picked battle look, dismounted and unarmed.
for key, title, kit, head in [
        ('zhaoyun', '조운', 'zhaoyun_4', H(Hd.IRON, Hd.PLUME)),
        ('wenchou', '문추', 'wenchou_2', H(L.bearded(L.DARK_IRON, 'bushy'), Hd.PLUME)),
        ('tianyu', '전예', 'tianyu_5', H(Hd.IRON, L.WHITE_PLUME)),
        ('fangong', '번궁', 'fangong_jab', H(L.GREEN_HOOD)),
        ('gengwu', '경무', 'gengwu_1', H(L.CAP)),
        ('guanchun', '관순', 'guanchun_5', H(L.bearded(L.CAP, 'goatee')))]:
    U.KITS[f'{key}_civ1'] = from_battle(kit)
    _walk = U.foot_unit(f'{key}_civ1')
    _walk.kit = f'{key}_civ1'
    EXTRA[key] = (title, _walk, head)

# Scene sprites face only the four diagonals and are drawn for NW and SW; the game mirrors
# them horizontally for NE and SE (user rule, 2026-10-04). Battle sprites keep all eight.
DIRECTIONS = ['NW', 'SW']
MIRRORED = {'NE': 'NW', 'SE': 'SW'}

# Scene motions (rows). Poses use the soldier keys; hand 0 is the gesturing hand.
# exhausted: the battle sprites' winded pose, also used for kneeling and pleading (user, 2026-10-05)
ANIMATIONS = ['idle', 'walk', 'talk', 'salute', 'toast', 'surprise', 'nod', 'exhausted']
DURATIONS = {'idle': 180, 'walk': 120, 'talk': 160, 'salute': [160, 200, 420, 300], 'toast': [160, 200, 420, 300],
             'surprise': [80, 120, 320, 220], 'nod': [140, 160, 220, 160], 'exhausted': 220}
EXHAUSTED = 4  # battle row of the winded pose (units.result shows the tired face for it)
REST = dict(hands=[[-.5, -.2, 1.0], [.5, -.2, 1.0]], elbows=[[-.52, 0, 1.18], [.52, 0, 1.18]])
MOTIONS = {
    'talk': [dict(hand=[-.5, -.45, 1.25]), dict(hand=[-.45, -.6, 1.45], bob=.01), dict(hand=[-.55, -.5, 1.32]),
             dict(hand=[-.42, -.6, 1.48], bob=.01)],
    'salute': [dict(both=[-.1, -.5, 1.3]), dict(both=[-.06, -.68, 1.5]), dict(both=[-.06, -.7, 1.46], lean=-.2, bob=-.03),
               dict(both=[-.06, -.68, 1.48], lean=-.12, bob=-.02)],
    'toast': [dict(hand=[-.4, -.42, 1.3], cup=True), dict(hand=[-.4, -.45, 1.62], cup=True),
              dict(hand=[-.32, -.4, 2.0], cup=True, lean=.06), dict(hand=[-.33, -.42, 1.95], cup=True, lean=.04)],
    'surprise': [dict(), dict(hand=[-.6, -.4, 1.5], off=[.6, -.4, 1.5], lean=.18, bob=.05),
                 dict(hand=[-.62, -.38, 1.62], off=[.62, -.38, 1.62], lean=.24, bob=.03),
                 dict(hand=[-.55, -.3, 1.3], off=[.55, -.3, 1.3], lean=.08)],
    'nod': [dict(), dict(lean=-.08, bob=-.02), dict(lean=-.15, bob=-.045), dict(lean=-.05, bob=-.01)],
}


def scene_pose(name, f):
    p = U.pose(0, 0)
    p.update(copy.deepcopy(REST))
    k = MOTIONS[name][f]
    if 'hand' in k:
        p['hands'][0] = list(k['hand'])
        p['elbows'][0] = ((U.SHOULDER + U.V(k['hand']))/2 + [-.18, .05, -.08]).tolist()
    if 'off' in k:
        p['hands'][1] = list(k['off'])
        p['elbows'][1] = ((U.V([.4, 0, 1.42]) + U.V(k['off']))/2 + [.18, .05, -.08]).tolist()
    if 'both' in k:  # fist in palm in front of the chest
        c = U.V(k['both'])
        p['hands'] = [(c + [-.06, 0, 0]).tolist(), (c + [.08, .02, .02]).tolist()]
        p['elbows'] = [[-.48, -.25, 1.2], [.48, -.25, 1.2]]
    p['lean'], p['bob'] = k.get('lean', 0), k.get('bob', 0)
    return p, k


def scene_unit(walk_build, kit):
    """Builder for the scene sheet: idle and walk from the soldier, then the scene motions."""
    def build(row, f, style=None):
        name = ANIMATIONS[row]
        if name in ('idle', 'walk'):
            p = U.pose(row, f)
            p.update(copy.deepcopy(REST)) if name == 'idle' else None
            p['bob'] = [0, .02, 0, -.01][f] if name == 'idle' else p['bob']
        elif name == 'exhausted':
            p = U.pose(EXHAUSTED, f)
        else:
            p, k = scene_pose(name, f)
        faces, head = U.soldier(p, kit, weapon=False)
        if name == 'exhausted':
            return U.result(faces, head, EXHAUSTED, f)
        if name == 'toast':  # a small cup in the raised hand
            T = U.transform(p)
            h = T(p['hands'][0])
            U.m.rod(h + [0, 0, .06], h + [0, 0, .22], .085, U.GOLD)
            U.m.ell(h + [0, 0, .23], [.09, .09, .03], '#bad9f1')
            faces = U.m.meshes
        return U.result(faces, head, 0, f)
    return build


def chosen():
    """(key, title, build, head) of every scene sprite."""
    out = []
    for key, i in SELECTED.items():
        _, walk, head = LOOKS[key][i]
        out.append((SPRITE_KEYS.get(key, key), TITLES[key], scene_unit(walk, walk.kit), head))
    for key, (title, walk, head) in EXTRA.items():
        out.append((key, title, scene_unit(walk, walk.kit), head))
    for key, (title, build, head) in SCENE_EXTRAS.items():
        out.append((key, title, build, head))
    return out


# ---------------------------------------------------------------- extras (2026-10-05)
# Unnamed people for scenes. Soldiers and messengers use the battle infantry and light cavalry
# sprites; villagers and the caravan's cart puller are drawn here (user, 2026-10-05).
GREY_HAIR = {d: Hd.recolor(r, {'f': '5', 'g': '6'}) for d, r in Hd.HAIR.items()}
WHITE_BEARD = {d: Hd.beard(r, '56', 'long') for d, r in GREY_HAIR.items()}
OCHRE, BROWN_CLOTH = '#a0763a', '#6b4a30'
U.RAMPS.update({OCHRE: 'hiij', BROWN_CLOTH: 'fggh'})
LOOKS.update({
    'commoner': [
        ('맨상투 · 삼베 옷 (마을 젊은이)', walker('commoner', 1, cloth=HEMP, guard=U.HEMP_DARK, tabard=LEATHER), H(Hd.HAIR)),
        ('흰 두건 · 삼베 옷', walker('commoner', 2, cloth=HEMP, guard=U.HEMP_DARK, tabard=U.HEMP_DARK), H(L.WHITE_HOOD)),
        ('흰머리 · 흰 수염 · 갈색 옷 (촌로)', walker('commoner', 3, cloth=BROWN_CLOTH, robe=BROWN_CLOTH, guard=U.HEMP_DARK, tabard=HEMP),
         H(WHITE_BEARD)),
        ('맨상투 · 황토색 옷 · 등짐', walker('commoner', 4, cloth=OCHRE, guard=LEATHER, tabard=LEATHER, pack=HEMP), H(Hd.HAIR)),
    ],
})
TITLES.update({'commoner': '백성', 'cart': '짐수레 상인'})

# Yellow turban soldier off the battlefield: the battle look without sword and shield.
U.KITS['turban_civ1'] = from_battle('bandit')
_walk = U.foot_unit('turban_civ1')
_walk.kit = 'turban_civ1'
EXTRA['turban'] = ('황건병', _walk, H(Hd.TOPKNOT, Hd.BAND))


def disc(c, r, axis, color, n=12):
    """Flat wheel facing along a model axis ('x' or 'y')."""
    pts = []
    for a in np.linspace(0, 2*math.pi, n, endpoint=False):
        u, v = r*math.cos(a), r*math.sin(a)
        pts.append(c + (U.V([0, u, v]) if axis == 'x' else U.V([u, 0, v])))
    U.m.poly(pts, color)


def cart(kind):
    """The cart, in world space (it does not lean or bob with the puller). The puller faces -y
    and holds the shafts at his sides; the cart follows behind (+y). 'barrow' is pushed in front."""
    W = U.V
    if kind == 'barrow':
        for s in (-1, 1):
            U.m.rod(W([s*.28, -.25, .95]), W([s*.26, -1.15, .55]), .045, U.SHAFT)
        U.m.box(W([0, -1.05, .62]), [.66, .8, .2], U.SHAFT)
        U.m.ell(W([0, -1.05, .86]), [.32, .34, .24], HEMP)
        disc(W([0, -1.6, .34]), .34, 'x', '#6e4126')
        U.m.ell(W([0, -1.6, .34]), [.06, .08, .08], U.STEEL)
        return
    y0, L, B, Z = .5, 1.5, 1.3, .8  # front of the bed, bed length and width, bed height
    yc = y0 + L/2
    for s in (-1, 1):
        U.m.rod(W([s*.42, -.1, .95]), W([s*.5, y0 + .1, Z]), .045, U.SHAFT)
        disc(W([s*(B/2 + .08), yc, .46]), .46, 'x', '#6e4126')
        U.m.ell(W([s*(B/2 + .12), yc, .46]), [.07, .09, .09], U.STEEL)
    U.m.rod(W([-B/2 - .08, yc, .46]), W([B/2 + .08, yc, .46]), .035, U.SHAFT)
    U.m.box(W([0, yc, Z]), [B, L, .1], '#976039')
    for s in (-1, 1):
        U.m.box(W([s*(B/2 - .03), yc, Z + .13]), [.06, L, .2], '#976039')
    U.m.box(W([0, y0 + L - .03, Z + .13]), [B, .06, .2], '#976039')
    if kind == 'sacks':
        for x, y, z in ((-.3, y0 + .45, 1.05), (.3, y0 + .5, 1.05), (-.15, y0 + 1.1, 1.05), (.3, y0 + 1.1, 1.05),
                        (0, y0 + .75, 1.38)):
            U.m.ell(W([x, y, z]), [.3, .27, .25], HEMP)
            U.m.rod(W([x, y, z + .18]), W([x, y, z + .3]), .06, U.HEMP_DARK)
    elif kind == 'crates':
        U.m.box(W([0, yc, Z + .35]), [B - .15, L - .15, .6], '#6e4126')
        U.m.box(W([0, yc, Z + .68]), [B - .1, L - .1, .07], HEMP)
        for y in (y0 + .4, y0 + L - .4):
            U.m.box(W([0, y, Z + .38]), [B - .08, .05, .68], U.HEMP_DARK)
    elif kind == 'covered':
        n, r = 8, B/2
        for i in range(n):
            a0, a1 = math.pi*i/n, math.pi*(i + 1)/n
            p = lambda a, y: W([r*math.cos(a), y, Z + .05 + r*math.sin(a)])  # noqa: E731
            U.m.poly([p(a0, y0 + .05), p(a1, y0 + .05), p(a1, y0 + L - .05), p(a0, y0 + L - .05)], HEMP)


def cart_unit(kit, kind):
    """Scene sheet of a cart puller: hands stay on the shafts while idle and walking."""
    grips = [[-.42, -.12, .95], [.42, -.12, .95]] if kind != 'barrow' else [[-.28, -.3, .95], [.28, -.3, .95]]

    def build(row, f, style=None):
        name = ANIMATIONS[row]
        if name in ('idle', 'walk'):
            p = U.pose(row, f)
            p.update(copy.deepcopy(REST))
            p['hands'] = copy.deepcopy(grips)
            p['elbows'] = [[-.5, -.02, 1.18], [.5, -.02, 1.18]]
            p['bob'] = [0, .02, 0, -.01][f] if name == 'idle' else U.pose(row, f)['bob']
            p['lean'] = .1 if kind != 'barrow' else .14  # leaning into the load
        elif name == 'exhausted':
            p = U.pose(EXHAUSTED, f)
        else:
            p, _ = scene_pose(name, f)
        faces, head = U.soldier(p, kit, weapon=False)
        cart(kind)
        return U.result(U.m.meshes, head, EXHAUSTED if name == 'exhausted' else 0, f)
    return build


U.KITS['cart_civ'] = L.kit(cloth=HEMP, guard=LEATHER, tabard=LEATHER)
_cart_walk = U.foot_unit('cart_civ')
_cart_walk.kit = 'cart_civ'
LOOKS['cart'] = [(title, cart_unit('cart_civ', kind), H(L.WHITE_HOOD))
                 for title, kind in (('손수레 · 짐 자루', 'sacks'), ('손수레 · 나무 상자', 'crates'),
                                     ('손수레 · 천 덮개', 'covered'), ('외바퀴 수레 (밀기)', 'barrow'))]
for _title, _build, _head in LOOKS['cart']:
    _build.kit = 'cart_civ'

# Chosen extras (user, 2026-10-05): the village youth and the elder, the cart puller with sacks.
# Scene builders, packed like the officers (civ_<key>).
SCENE_EXTRAS = {
    'villager': ('마을 젊은이', scene_unit(LOOKS['commoner'][0][1], LOOKS['commoner'][0][1].kit), LOOKS['commoner'][0][2]),
    'elder': ('촌로', scene_unit(LOOKS['commoner'][2][1], LOOKS['commoner'][2][1].kit), LOOKS['commoner'][2][2]),
    'cart': ('짐수레 상인', LOOKS['cart'][0][1], LOOKS['cart'][0][2]),
}
