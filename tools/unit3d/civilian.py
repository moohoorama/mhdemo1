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
import render as R
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
# Scene candidates for the prologue and chapter 1 cast (2026-10-04). Non-Shu officers use fixed
# colors, not the team keys: scenes draw every actor with one faction.
BRICK, MAROON, LEATHER, PURPLE, SILVER, PALE = '#a14a35', '#7a2f22', '#704022', L.PURPLE, '#c9cdd0', '#dde1e3'
B = L.bearded
WHITE_PLUME = {'b': '4', 'c': '5', 'd': '6'}  # fixed white plume instead of the team key
WEI_BLUE, WEI_SKY = '#193b65', '#245784'  # knight v6 blue ramp 7 8 9 a (+ o), no new colors
R.add_ramps({WEI_BLUE: '789a', WEI_SKY: '89ao'})
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
    'caocao3': [  # third round: blue battle robe and cap (user, 2026-10-04); fixed knight-v6 blue, not the team keys
        ('검은 관모 · 푸른 전포 · 검은 띠', walker('caocao', 9, cloth=WEI_BLUE, robe=WEI_BLUE, guard=BLACK, tabard=BLACK),
         H(B(L.CAP, 'goatee'))),
        ('금빛 관모 · 푸른 전포 · 금빛 어깨', walker('caocao', 10, cloth=WEI_BLUE, robe=WEI_BLUE, guard=GOLD, tabard=GOLD),
         H(B(L.GOLD_CAP, 'goatee'))),
        ('검은 관모 · 푸른 전포 · 철 어깨 · 금빛 띠', walker('caocao', 11, cloth=WEI_BLUE, robe=WEI_BLUE, guard=STEEL, tabard=GOLD),
         H(B(L.CAP, 'goatee'))),
        ('검은 관모 · 흰 속옷 · 밝은 푸른 전포 · 금빛 띠', walker('caocao', 12, cloth=WHITE, robe=WEI_SKY, guard=WEI_SKY, tabard=GOLD),
         H(B(L.CAP, 'goatee'))),
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
TITLES = {'guanyu': '관우', 'zhangfei': '장비', 'liubei': '유비', 'zhangshiping': '장세평', 'sushuang': '소쌍', 'caocao': '조조', 'caocao2': '조조', 'caocao3': '조조','yuanshao': '원소', 'yuanshu': '원술',
          'gongsunzan': '공손찬', 'dongzhuo': '동탁', 'liru': '이유'}
SELECTED = {'guanyu': 1, 'zhangfei': 1, 'liubei': 1,
            'zhangshiping': 3, 'sushuang': 3, 'yuanshao': 0, 'yuanshu': 1, 'gongsunzan': 3, 'dongzhuo': 0, 'liru': 3,
            'caocao3': 3}  # chosen by the user (2026-10-04); others are backups
# look key -> sprite key, for officers whose pick came from a later candidate round
SPRITE_KEYS = {'caocao3': 'caocao'}

# Officers who speak in scenes but had no candidate round: their battle look without the weapon.
EXTRA = {
    'jianyong': ('간옹', walker('jianyong', 1), H(L.WHITE_HOOD)),
    'sunqian': ('손건', walker('sunqian', 1, robe=BLUE, guard=BLUE), H(L.CAP)),
}

# Scene sprites face only the four diagonals and are drawn for NW and SW; the game mirrors
# them horizontally for NE and SE (user rule, 2026-10-04). Battle sprites keep all eight.
DIRECTIONS = ['NW', 'SW']
MIRRORED = {'NE': 'NW', 'SE': 'SW'}

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
        out.append((SPRITE_KEYS.get(key, key), TITLES[key], scene_unit(walk, walk.kit), head))
    for key, (title, walk, head) in EXTRA.items():
        out.append((key, title, scene_unit(walk, walk.kit), head))
    return out
