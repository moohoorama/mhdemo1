"""Civilian (평복) looks for scenario scenes: officers on foot without weapons.

Battle sprites stay as they are; these are a second sprite per officer, used where
the story is told (dismounted, unarmed). Four candidates per officer; the user picks
one (SELECTED) and the rest stay as backups. Kits register into units.KITS as
'<key>_civ<n>'.

  python3 tools/unit3d/civilian_candidates.py  -> output/ver4-civilian-candidates.png
"""
import copy
import heads as Hd
import looks as L
import units as U

GREEN, DARK_GREEN, BLACK, DARK, HEMP = U.GREEN, U.DARK_GREEN, U.BLACK, U.DARK_STEEL, U.HEMP
WHITE, GOLD, BLUE, STEEL = L.WHITE, U.GOLD, U.BLUE, U.STEEL


def walker(key, n, **kit):
    name = f'{key}_civ{n}'
    U.KITS[name] = L.kit(**kit)
    return U.foot_unit(name)


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
TITLES = {'guanyu': '관우', 'zhangfei': '장비', 'liubei': '유비'}
SELECTED = {'guanyu': 1, 'zhangfei': 1, 'liubei': 1}  # chosen by the user (2026-10-04); others are backups

# Officers who speak in scenes but had no candidate round: their battle look without the weapon.
EXTRA = {
    'jianyong': ('간옹', walker('jianyong', 1), H(L.WHITE_HOOD)),
    'sunqian': ('손건', walker('sunqian', 1, robe=BLUE, guard=BLUE), H(L.CAP)),
}

# Scene motions (rows). Poses use the soldier keys; hand 0 is the gesturing hand.
ANIMATIONS = ['idle', 'walk', 'talk', 'salute', 'toast', 'surprise', 'nod']
DURATIONS = {'idle': 180, 'walk': 120, 'talk': 160, 'salute': [160, 200, 420, 300], 'toast': [160, 200, 420, 300],
             'surprise': [80, 120, 320, 220], 'nod': [140, 160, 220, 160]}
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
        else:
            p, k = scene_pose(name, f)
        faces, head = U.soldier(p, kit, weapon=False)
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
        out.append((key, TITLES[key], scene_unit(walk, f'{key}_civ{i + 1}'), head))
    for key, (title, walk, head) in EXTRA.items():
        out.append((key, title, scene_unit(walk, f'{key}_civ1'), head))
    return out
