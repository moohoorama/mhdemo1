#!/usr/bin/env python3
"""Chapter 21 (여남): map21 layout, stage21.json, story21.json, two new characters, scenario entry.
Run from ver5/. Rewrites its outputs from scratch every time."""
import json
import re
import subprocess
from pathlib import Path

ROOT = Path('.')
HERO = ROOT / 'assets/scenario/hero'
W, H = 20, 32

# ---------------------------------------------------------------- map
g = [['g--'] * W for _ in range(H)]


def put(x, y, c):
    g[y][x] = c


for y in range(1, 16):
    for x in range(2, 18):
        edge = x in (2, 17) or y in (1, 15)
        put(x, y, 'c--' if edge else 'l--')
for x in range(3, 17):          # the two main streets
    put(x, 8, 'i--')
for y in range(2, 15):
    put(10, y, 'i--')
for x, y in ((10, 1), (2, 8), (17, 8), (10, 15)):
    put(x, y, 'm-O')
for x, y in ((5, 4), (7, 5), (13, 4), (15, 6), (5, 11), (7, 12), (13, 11), (15, 12), (4, 6), (14, 13)):
    put(x, y, 'lv-')
for y in range(16, 31):          # the road south
    put(10, y, 'd--')
for x in range(W):
    put(x, 24, 'w--')
    put(x, 25, 'w--')
put(10, 24, 'wb-')
put(10, 25, 'wb-')
put(16, 24, 's--')
put(16, 25, 's--')
put(10, 30, 'go-')
for x, y in ((0, 0), (5, 0), (15, 0), (19, 3), (0, 11), (19, 13), (3, 18), (6, 21), (14, 19), (18, 22), (1, 27),
             (5, 29), (14, 28), (18, 30), (7, 31), (13, 22), (4, 23), (17, 27)):
    put(x, y, 'gt-')
for x, y in ((7, 17), (8, 18), (12, 17), (13, 19), (9, 21), (11, 22), (9, 27), (11, 28), (8, 29), (12, 29)):
    put(x, y, 'l--')

layout = ROOT / 'tools/mapgen/layouts/hero/map21.txt'
layout.write_text(
    '# map21 여남 전투 (20x32, night). Grid compass: north = row 0. 여남성 fills the top half (walls x 2..17, y 1..15,\n'
    '# gates north (10,1), west (2,8), east (17,8), south (10,15), all open). South of it the field, the 여수 across\n'
    '# rows 24-25 with one bridge (10) and one ford (16), and the small castle on the road to 형주 at (10,30).\n'
    + '\n'.join(' '.join(r) for r in g) + '\n', encoding='utf-8')
subprocess.run(['python3', 'tools/mapgen/mapgen.py', str(layout), str(HERO / 'maps/map21.json')], check=True)

# ---------------------------------------------------------------- characters
chars = {
    'liupi': {
        'Name': '유벽', 'Biography': '여남의 옛 황건 두령. 관도 싸움 때 원소 편에 서서 허도 뒤를 쳤고, 예주목 유비를 여남으로 맞아들였다.',
        'Class': 'outlaw', 'Stats': [68, 70, 42, 60, 56, 60], 'Level': 20, 'Playable': True, 'Traits': [], 'Learn': [],
        'Equipment': ['copper_sword', 'cotton_clothes'], 'Look': {'Faction': 'shu'}},
    'gongdu': {
        'Name': '공도', 'Biography': '여남의 옛 황건 두령. 유벽과 함께 여남에서 군사를 일으켰다.',
        'Class': 'outlaw', 'Stats': [60, 68, 36, 62, 50, 52], 'Level': 1, 'Traits': [], 'Learn': [],
        'Equipment': ['copper_sword', 'cotton_clothes'], 'Look': {'Faction': 'shu'}},
}
cpath = ROOT / 'assets/data/characters_story.json'
text = cpath.read_text(encoding='utf-8')
anchor = '  "villager": {'
assert text.count(anchor) == 1
for cid, body in chars.items():
    old = re.search(r'  "%s": \{.*?\n  \},\n' % cid, text, re.S)
    block = '  "%s": %s,\n' % (cid, json.dumps(body, ensure_ascii=False, indent=1).replace('\n', '\n  '))
    text = text.replace(old.group(0), block) if old else text.replace(anchor, block + anchor)
cpath.write_text(text, encoding='utf-8')


# ---------------------------------------------------------------- stage
def foe(c, x, y, group, facing='S', policy='assault', **kw):
    e = {'Character': c, 'X': x, 'Y': y, 'Facing': facing, 'Group': group, 'Policy': policy, 'Faction': 'wei'}
    if c.startswith('troop_'):
        e['Shift'] = -0.07
    e.update(kw)
    return e


def spawn(c, x, y, group, facing, **kw):
    return dict(foe(c, x, y, group, facing, **kw), Side='enemy')


def say(actor, text, **kw):
    return {'Kind': 'say', 'Lines': [dict({'Actor': actor, 'Text': text}, **kw)]}


stage = {
    'ID': 'B21', 'Name': '여남 전투', 'Width': 0, 'Height': 0, 'Max': 6, 'Required': ['liubei', 'zhaoyun'],
    'Victory': [{'If': ['at:liubei:1'], 'Text': '유비가 남문을 나선다'}],
    'Sub': [{'ID': 'liupi', 'Text': '유벽이 끝까지 살아남는다', 'If': ['alive:liupi'],
             'Flag': 'flag-b21-liupi', 'Reward': {'insu_high': 1}},
            {'ID': 'yuejin', 'Text': '악진을 물리친다', 'If': ['defeated:yuejin'], 'Flag': 'flag-b21-yuejin',
             'Reward': {'mirror_armor': 1}}],
    'TurnLimit': 25, 'Map': 'map21.json', 'Night': True,
    'Lights': [{'X': 10, 'Y': 1}, {'X': 2, 'Y': 8}, {'X': 17, 'Y': 8}, {'X': 10, 'Y': 15}, {'X': 10, 'Y': 8},
               {'X': 10, 'Y': 23}, {'X': 10, 'Y': 30}],
    'Allies': [{'X': x, 'Y': y} for x, y in ((10, 4), (10, 6), (9, 5), (11, 5), (9, 6), (11, 6), (9, 7), (11, 7))],
    'Recommend': {'zhaoyun': {'X': 10, 'Y': 4}, 'liubei': {'X': 10, 'Y': 6}},
    'Party': [
        {'AI': True, 'Policy': 'hold', 'Facing': 'N', 'Group': 'liupi', 'Character': 'liupi', 'X': 10, 'Y': 13, 'Faction': 'shu'},
        {'AI': True, 'Policy': 'hold', 'Facing': 'N', 'Group': 'pigate', 'Character': 'troop_footman', 'X': 9, 'Y': 14,
         'Faction': 'shu', 'Shift': -0.07},
        {'AI': True, 'Policy': 'hold', 'Facing': 'N', 'Group': 'pigate', 'Character': 'troop_footman', 'X': 11, 'Y': 14,
         'Faction': 'shu', 'Shift': -0.07},
    ],
    'Enemies': [
        foe('yuejin', 10, 3, 'yue', Rank='commander', Deputy='gaolan', DuelFirst=True),
        foe('troop_cavalry', 9, 3, 'yue'), foe('troop_cavalry', 11, 3, 'yue'),
        foe('troop_cavalry', 10, 2, 'yue'),
        # 조조군은 남쪽 길에도 미리 와 있다: 유벽이 뚫으러 나가면 여기에 걸린다
        foe('xuchu', 10, 20, 'xu', 'N', 'react'), foe('troop_footman', 9, 20, 'xu', 'N', 'react'),
        foe('troop_footman', 11, 20, 'xu', 'N', 'react'),
    ],
    'Reward': {'money': 1000},
    'Areas': [
        {'ID': 1, 'Rect': {'X': 8, 'Y': 16, 'W': 5, 'H': 3}, 'Name': '남문 밖',
         'Description': '유벽의 군사가 쥐고 있는 남문 바로 밖. 유비가 여기까지 나오면 성을 빠져나온 것이다'},
        {'ID': 2, 'Rect': {'X': 10, 'Y': 30, 'W': 1, 'H': 1}, 'Name': '남쪽 성',
         'Description': '형주로 가는 길목의 작은 성. 손건이 돌아온 뒤 유비가 닿으면 승리'},
        {'ID': 4, 'Rect': {'X': 0, 'Y': 18, 'W': 20, 'H': 2}, 'Name': '들판 가운데',
         'Description': '남문과 강 사이의 들판. 유비가 여기 닿으면 동쪽 숲에서 조조의 복병이 나온다'},
        {'ID': 5, 'Rect': {'X': 0, 'Y': 26, 'W': 20, 'H': 2}, 'Name': '강 남쪽 기슭',
         'Description': '다리를 건넌 바로 남쪽. 유비가 여기 닿으면 양쪽에서 조조의 복병이 나온다'},
        {'ID': 3, 'Rect': {'X': 0, 'Y': 20, 'W': 20, 'H': 4}, 'Name': '여수 강가',
         'Description': '다리와 여울 앞 강가. 유비가 여기 닿으면 다음 차례에 관우와 장비가 맵 끝에 나타난다'},
    ],
    'Events': [
        {'ID': 'opening', 'If': ['round>=1'], 'Do': [
            say('yuejin', '성문은 넷 다 우리 것이다! 유비를 놓치지 마라!'),
            say('zhaoyun', '공, 남문까지 멈추지 마십시오. 뒤는 제가 막겠습니다.'),
            say('liupi', '남문 밖 길은 내가 먼저 뚫어 놓겠소! 유 예주는 뒤따라오시오!'),
            say('liubei', '유 장군, 혼자는 위험하오! ……모두 남문으로 간다. 유 장군을 따라붙어라!'),
            {'Kind': 'ai', 'Targets': ['liupi', 'pigate'], 'Policy': 'march', 'GoalArea': 2}]},
        {'ID': 'south-blocked', 'If': ['round>=2', 'turnstart:ally'], 'Do': [
            say('xuchu', '남문으로 나올 줄 알았다. 승상께서 예까지 내다보셨다!'),
            say('jianyong', '남쪽 길에도 벌써 조조군이네! 유 장군이 저 속으로 들어가고 있네!')]},
        {'ID': 'flanks', 'If': ['round>=2', 'turnstart:enemy'], 'Do': [
            {'Kind': 'spawn', 'Spawns': [
                spawn('troop_footman', 3, 8, 'west', 'E'), spawn('troop_footman', 4, 8, 'west', 'E'),
                spawn('troop_footman', 16, 8, 'east', 'W'), spawn('troop_archer', 15, 8, 'east', 'W')]},
            say('jianyong', '현덕, 서문과 동문으로도 들어오네! 가운데 길이 끊기기 전에 지나가야 하네!')]},
        {'ID': 'north-3', 'If': ['round>=3', 'turnstart:enemy'], 'Do': [
            {'Kind': 'spawn', 'Spawns': [spawn('troop_cavalry', 9, 2, 'more', 'S'), spawn('troop_cavalry', 11, 2, 'more', 'S')]},
            say('yuejin', '뒤의 기병도 다 들여라! 성 안에서 끝낸다!')]},
        {'ID': 'north-5', 'If': ['round>=5', 'turnstart:enemy'], 'Do': [
            {'Kind': 'spawn', 'Spawns': [spawn('troop_cavalry', 9, 2, 'more', 'S'), spawn('troop_cavalry', 11, 2, 'more', 'S')]}]},
        {'ID': 'north-7', 'If': ['round>=7', 'turnstart:enemy'], 'Do': [
            {'Kind': 'spawn', 'Spawns': [spawn('troop_cavalry', 9, 2, 'more', 'S'), spawn('troop_cavalry', 11, 2, 'more', 'S')]}]},
        {'ID': 'north-9', 'If': ['round>=9', 'turnstart:enemy'], 'Do': [
            {'Kind': 'spawn', 'Spawns': [spawn('troop_cavalry', 9, 2, 'more', 'S'), spawn('troop_cavalry', 11, 2, 'more', 'S')]}]},
        {'ID': 'out', 'If': ['at:liubei:1'], 'Do': [
            {'Kind': 'join', 'Spawns': [{'Character': 'sunqian', 'X': 10, 'Y': 28, 'Facing': 'N'}]},
            say('sunqian', '황숙! 유경승께서 받아 주시겠답니다! 강 건너 남쪽 성으로 오십시오. 형주로 가는 길입니다!'),
            say('liubei', '손 공이 돌아왔다! 다리를 건너 남쪽 성으로 간다!'),
            {'Kind': 'spawn', 'Spawns': [
                spawn('zhanghe', 1, 19, 'zh', 'E'), spawn('troop_cavalry', 0, 18, 'zh', 'E'),
                spawn('troop_cavalry', 0, 20, 'zh', 'E'),
                ]},
            say('zhanghe', '기병은 나를 따르라. 다리목을 먼저 잡는다!'),
            {'Kind': 'ai', 'Targets': ['xu'], 'Policy': 'assault'},
            {'Kind': 'goals', 'Victory': [{'If': ['at:liubei:2'], 'Text': '유비가 남쪽 성에 닿는다'}]},
            {'Kind': 'flag', 'Flags': {'flag-b21-out': 1}}]},
        {'ID': 'ambush-field', 'If': ['flag:flag-b21-out', 'at:liubei:4', 'turnstart:enemy'], 'Do': [
            {'Kind': 'spawn', 'Spawns': [
                spawn('troop_footman', 18, 19, 'amb1', 'W'), spawn('troop_footman', 19, 20, 'amb1', 'W'),
                spawn('troop_archer', 19, 19, 'amb1', 'W')]},
            say('jianyong', '동쪽 숲에서도 나오네! 들판에까지 군사를 묻어 뒀구먼!')]},
        {'ID': 'ambush-south', 'If': ['flag:flag-b21-out', 'at:liubei:5', 'turnstart:enemy'], 'Do': [
            {'Kind': 'spawn', 'Spawns': [
                spawn('troop_cavalry', 1, 27, 'amb2', 'E'),
                spawn('troop_footman', 18, 28, 'amb2', 'W'), spawn('troop_archer', 19, 29, 'amb2', 'W')]},
            say('liubei', '강 건너에도 복병인가……. 멈추지 마라! 성이 눈앞이다!')]},
        {'ID': 'brothers', 'If': ['flag:flag-b21-out', 'at:liubei:3', 'turnstart:ally'], 'Do': [
            {'Kind': 'join', 'Spawns': [
                {'Character': 'guanyu', 'X': 1, 'Y': 22, 'Facing': 'E'}, {'Character': 'guanping', 'X': 0, 'Y': 22, 'Facing': 'E'},
                {'Character': 'zhoucang', 'X': 0, 'Y': 23, 'Facing': 'E'},
                {'Character': 'zhangfei', 'X': 18, 'Y': 21, 'Facing': 'W'}, {'Character': 'chendao', 'X': 19, 'Y': 21, 'Facing': 'W'},
                {'Character': 'fangong', 'X': 19, 'Y': 22, 'Facing': 'W'}]},
            {'Kind': 'spawn', 'Spawns': [
                {'Character': 'gongdu', 'X': 19, 'Y': 19, 'Facing': 'N', 'Group': 'gongdu', 'AI': True, 'Policy': 'run',
                 'GoalArea': 2, 'Faction': 'shu', 'Side': 'ally'},
                spawn('xiahouyuan', 19, 18, 'yuan', 'S'), spawn('troop_cavalry', 18, 16, 'yuan', 'S'),
                spawn('troop_cavalry', 18, 17, 'yuan', 'S')]},
            say('guanyu', '형님! 하후돈은 허깨비였습니다. 속았습니다!'),
            say('zhangfei', '형! 공도는 데려왔슈! 근디 하후연이 꼬리에 붙었슈!'),
            say('xiahouyuan', '도적놈이 어디까지 달아나느냐! 군량 도둑의 목은 내가 벤다!'),
            {'Kind': 'duel', 'Duel': 'runan-gongdu-xiahouyuan'},
            say('zhangfei', '공도! ……하후연 이놈!'),
            say('liubei', '……운장, 익덕! 뒤를 막아 다오. 모두 남쪽 성으로 간다!')]},
        {'ID': 'liupi-falls', 'If': ['defeated:liupi'], 'Do': [say('liupi', '도적이 아니라…… 예주의 장수로…… 죽는 거요…….')]},
    ],
    'Faction': 'wei',
    'Duels': [{
        'ID': 'runan-gongdu-xiahouyuan', 'Ally': 'gongdu', 'Enemy': 'xiahouyuan', 'Outcome': 'defeat', 'AllyResult': 'death',
        'EnemyResult': 'stay', 'Text': '공도가 하후연의 칼에 쓰러졌다.', 'Melee': {'xiahouyuan': 'light_cavalry'},
        'Unlisted': True,
        'Beats': [
            {'Kind': 'talk', 'Title': '하후연이 공도를 따라잡다', 'Speech': '거기 서라!', 'Side': 1, 'Duration': 2.0},
            {'Kind': 'talk', 'Title': '공도가 돌아서다', 'Speech': '에잇, 더는 못 뛰겠수. 덤벼 보슈!', 'Side': 0, 'Duration': 2.6},
            {'Kind': 'approach', 'Title': '하후연이 말을 몰아 들이치다', 'Speech': '', 'Side': 1, 'Duration': 1.2},
            {'Kind': 'block', 'Title': '공도가 가까스로 막다', 'Speech': '', 'Side': 1, 'Duration': 1.05},
            {'Kind': 'rush-block', 'Title': '하후연이 거푸 내리치다', 'Speech': '', 'Side': 1, 'Duration': 1.1},
            {'Kind': 'talk', 'Title': '공도가 이를 악물다', 'Speech': '유벽 형님…… 먼저 가우…….', 'Side': 0, 'Duration': 2.4},
            {'Kind': 'critical', 'Title': '하후연이 공도를 베다', 'Speech': '', 'Side': 1, 'Duration': 1.3},
            {'Kind': 'win', 'Title': '공도가 쓰러지다', 'Speech': '', 'Side': 1, 'Duration': 2.0}]}, {
        'ID': 'runan-zhaoyun-yuejin', 'Ally': 'zhaoyun', 'Enemy': 'yuejin', 'Outcome': 'victory', 'AllyResult': 'stay',
        'EnemyResult': 'retreat', 'Text': '조운의 창에 밀린 악진이 북문 쪽으로 물러났다.',
        'Beats': [
            {'Kind': 'talk', 'Title': '악진이 조운을 막아서다', 'Speech': '비켜라! 내 창은 유비의 목만 찾는다!', 'Side': 1, 'Duration': 2.6},
            {'Kind': 'talk', 'Title': '조운이 답하다', 'Speech': '상산의 조자룡이다. 이 창을 넘어야 갈 수 있다!', 'Side': 0, 'Duration': 2.6},
            {'Kind': 'approach', 'Title': '두 장수가 좁은 길에서 맞붙다', 'Speech': '', 'Side': 0, 'Duration': 1.2},
            {'Kind': 'block', 'Title': '조운이 악진의 창을 받다', 'Speech': '', 'Side': 1, 'Duration': 1.05},
            {'Kind': 'rush-block', 'Title': '악진이 거푸 찔러 들어오다', 'Speech': '', 'Side': 1, 'Duration': 1.1},
            {'Kind': 'dodge', 'Title': '조운이 몸을 틀어 피하다', 'Speech': '', 'Side': 1, 'Duration': 1.05},
            {'Kind': 'rush-block', 'Title': '조운이 되받아 몰아치다', 'Speech': '', 'Side': 0, 'Duration': 1.1},
            {'Kind': 'yielding-block', 'Title': '악진이 크게 밀리다', 'Speech': '(이놈, 창끝이 보이지 않는다……!)', 'Side': 0, 'Duration': 1.2},
            {'Kind': 'critical-block', 'Title': '조운이 창을 내리찍다', 'Speech': '하앗!', 'Side': 0, 'Duration': 1.2},
            {'Kind': 'talk', 'Title': '악진이 이를 갈다', 'Speech': '……오늘은 네 창에 막혔다. 허나 유비는 못 빠져나간다!', 'Side': 1, 'Duration': 3.0},
            {'Kind': 'retreat', 'Title': '악진이 북문 쪽으로 물러나다', 'Speech': '', 'Side': 1, 'Duration': 3.0},
            {'Kind': 'withdrawn', 'Title': '악진이 퇴각하다', 'Speech': '', 'Side': 1, 'Duration': 1.1}]}],
    'Armies': [{'Faction': 'shu', 'Name': '유비군', 'Ally': True}, {'Faction': 'wei', 'Name': '조조군'}],
}
def fix(o):
    if isinstance(o, dict):
        if 'Facing' in o:
            o['Facing'] = {'N': 'NE', 'S': 'SW', 'E': 'SE', 'W': 'NW'}[o['Facing']]
        for v in o.values():
            fix(v)
    elif isinstance(o, list):
        for v in o:
            fix(v)


fix(stage)
(HERO / 'stage21.json').write_text(json.dumps(stage, ensure_ascii=False, indent=1) + '\n', encoding='utf-8')

# ---------------------------------------------------------------- story
nodes = []


def scene(sid, place):
    nodes.append({'ID': sid, 'Kind': 'scene', 'Place': place})


def lines(prefix, rows):
    for i, (who, text) in enumerate(rows, 1):
        nodes.append({'ID': f'{prefix}{i:02d}', 'Kind': 'say', 'Speaker': who, 'Text': text})


def leave(who):
    nodes.append({'ID': f'c21-leave-{who}', 'Kind': 'leave', 'Character': who, 'Event': f'leave-{who}-21'})


scene('c21-runan', '여남 · 관청')
lines('c21-r', [
    ('liupi', '유 예주께서 오셨구려. 여남의 유벽이오. 이쪽은 공도요.'),
    ('gongdu', '공도올시다. 한때 누런 두건을 썼던 몸이라 예법은 모르오.'),
    ('liubei', '유비요. 성문을 열어 주어 고맙소.'),
    ('zhangfei', '형, 누런 두건이면…… 우리가 탁군서부터 때려잡던 그 황건적 아니유?'),
    ('liupi', '들리오, 장 장군. 맞소. 그 두건 벗은 지 십수 년인데, 아직도 그 소리를 듣소.'),
    ('liupi', '관도 싸움 때 우리는 원소 공 편에 서서 허도 뒤를 쳤소. 조조가 이기면 우리는 토벌당할 도적 떼요.'),
    ('liupi', '예주목의 깃발 아래 서면 예주의 군사가 되오. 그래서 유 예주를 모신 거요.'),
    ('messenger', '급보요! 관도에서 원소 공의 대군이 무너졌습니다!'),
    ('messenger', '오소의 군량이 불타고, 원소 공은 팔백 기만 데리고 강을 건너 달아났다 합니다!'),
    ('jianyong', '……그 큰 군사가 무너졌다고?'),
    ('guanyu', '조조라면 그럴 수 있습니다. 곁에서 보았습니다. 군사가 적을 때일수록 과감해지는 사람입니다.'),
    ('gongdu', '북쪽이 정리됐으니, 다음은 남쪽이겠구려.'),
    ('liubei', '조조는 허도 등 뒤에 적을 두고 지내는 사람이 아니오. 머지않아 직접 올 것이오.'),
    ('liupi', '여남에도 군사가 있고 성이 있소.'),
    ('guanyu', '관도의 군사가 그대로 내려온다면, 성 하나로 오래 버티지는 못합니다.'),
    ('sunqian', '황숙, 남쪽 형주의 유경승은 황숙과 같은 한실의 종친입니다. 군사가 십만이고 땅이 넉넉합니다.'),
    ('sunqian', '여남이 버티든 못 버티든, 등 뒤에 기댈 곳을 마련해 두어야 합니다. 제가 다녀오겠습니다.'),
    ('liubei', '유경승과는 얼굴 한 번 본 적이 없소.'),
    ('sunqian', '그러니 싸움이 벌어지기 전에 가야 합니다. 쫓겨서 문을 두드리는 것과, 미리 사람을 보내 예를 갖추는 것은 다릅니다.'),
    ('liubei', '손 공, 부탁하오.'),
    ('zhangfei', '손 선생, 빨리 다녀오슈. 조조가 손 선생 기다려 줄 것 같진 않으니께.'),
    ('liupi', '우리는 성벽을 손보겠소. 공도, 자네는 고을을 돌며 군량을 거둬 오게.'),
    ('gongdu', '알았수.'),
])
leave('sunqian')
nodes.append({'ID': 'c21-walk', 'Kind': 'explore', 'Explore': 'explore21'})
scene('c21-alarm', '여남 · 관청')
lines('c21-a', [
    ('messenger', '급보요! 조조의 장수 하후돈이 샛길로 돌아, 성 서쪽 고을들을 불태우며 오고 있습니다!'),
    ('liupi', '서쪽은 성벽이 낮소. 거기로 붙으면 오래 못 버티오.'),
    ('messenger', '급보요! 군량을 싣고 오던 공 장군이 동쪽 고갯길에서 하후연에게 에워싸였습니다!'),
    ('liupi', '공도가……!'),
    ('jianyong', '서쪽과 동쪽이 한날에 터졌구먼. 현덕, 둘 다 막으려면 운장과 익덕을 다 내보내야 하네. 그러면 성이 비네.'),
    ('liubei', '공도를 버리면 군량도 사람도 잃네. 서쪽을 버리면 성을 잃고. 어느 쪽도 버릴 수 없네.'),
    ('liubei', '운장은 서쪽으로 가서 하후돈을 막아라. 익덕은 동쪽으로 가서 공도를 데려와라.'),
    ('zhangfei', '맡겨 두슈! 금방 데려올 테니께 밥이나 넉넉히 해 놓으슈.'),
    ('guanyu', '형님, 하후돈을 물리치는 대로 밤을 새워서라도 돌아오겠습니다.'),
    ('zhaoyun', '성은 제가 지키겠습니다. 두 분은 마음 놓고 다녀오십시오.'),
])
for who in ('guanyu', 'guanping', 'zhoucang', 'zhangfei', 'chendao', 'fangong'):
    leave(who)
scene('c21-gate', '여남 · 북문 안')
lines('c21-g', [
    ('gatekeeper', '관 장군께서 돌아오셨습니다! 문을 엽니다!'),
    ('liubei', '운장이 벌써? 하후돈을 이리 빨리 물리쳤단 말인가.'),
    ('gatekeeper', '조, 조조군이다! 조조군이 어떻게……!'),
    ('yuejin', '승상의 은혜를 저버린 놈, 유비! 여기 숨어 있었구나!'),
    ('liubei', '……악진. 원소 밑에 있던 고람까지.'),
    ('gaolan', '허도에 머문 며칠 동안 너희가 성문으로 들인 깃발 하나, 밤마다 바꾼 군호 하나까지 승상의 장부에 다 적혀 있다!'),
    ('gaolan', '너희 깃발도 군호도 승상의 손바닥 안이다! 거록에서 입 조심하라는 말을 들었다만, 이 계책만은 자랑해도 좋겠지!'),
    ('jianyong', '그럼 서쪽의 하후돈도, 동쪽의 공도도…….'),
    ('gaolan', '관우와 장비가 곁에 있는 유비를 누가 치겠느냐? 둘을 떼어 놓으라는 것이 승상의 계책이시다. 서쪽의 하후돈은 미끼일 뿐, 진짜는……'),
    ('yuejin', '고 장군, 말은 그쯤 하시오! 유비의 목은 이 악진이 가져간다!'),
    ('liupi', '유 예주, 남문은 아직 우리 군사가 쥐고 있소! 그리로 빠지시오!'),
    ('zhaoyun', '공, 제 뒤에 서십시오. 길은 제가 열겠습니다.'),
])
nodes.append({'ID': 'c21-battle', 'Kind': 'battle', 'Stage': 'B21', 'Text': '여남 전투 — 출전 준비'})
# whoever the battle ended without (a quick escape) is back here all the same
for who in ('sunqian', 'guanyu', 'guanping', 'zhoucang', 'zhangfei', 'chendao', 'fangong'):
    nodes.append({'ID': 'c21-back-' + who, 'Kind': 'join', 'Character': who, 'Event': 'back-' + who + '-21'})
scene('c21-south', '여남 남쪽 · 작은 성 밖')
lines('c21-s', [
    ('guanyu', '형님, 제 이름으로 문이 열렸습니다. 이 운장의 죄입니다.'),
    ('liubei', '조조가 운장을 잘 알기에 운장의 이름을 쓴 것일세. 그대의 죄가 아니네.'),
    ('zhangfei', '하후돈이고 하후연이고 죄다 미끼였슈. 공도는 내 눈앞에서 당했슈. 한 발만 빨랐어도…….'),
])
nodes.append({'ID': 'c21-pi-br', 'Kind': 'branch', 'Cases': [{'If': ['flag:flag-b21-liupi'], 'Next': 'c21-p01'}], 'Else': 'c21-pi-br2'})
lines('c21-p', [
    ('liupi', '공도는 도적으로 죽지 않았소. 예주의 군사로 죽었소.'),
    ('liupi', '성은 잃었지만 깃발은 남았소. 나도 형주까지 따르겠소.'),
])
nodes.append({'ID': 'c21-join-liupi', 'Kind': 'join', 'Character': 'liupi', 'Event': 'join-liupi'})
nodes.append({'ID': 'c21-pi-br2', 'Kind': 'branch', 'Cases': [{'If': ['flag:flag-b21-liupi'], 'Next': 'c21-e01'}], 'Else': 'c21-d01'})
lines('c21-d', [
    ('liubei', '유 장군도, 공도도…….'),
    ('liubei', '……도적으로 죽지 않았네. 예주의 장수로 죽었네.'),
])
lines('c21-e', [
    ('sunqian', '황숙, 유경승께서 양양 밖까지 마중을 나오시겠답니다.'),
    ('jianyong', '가세, 현덕. 조조가 강을 건너기 전에.'),
])
nodes.append({'ID': 'c21-done', 'Kind': 'set', 'Flags': {'c21-runan': 1}})
(HERO / 'story21.json').write_text(json.dumps({'Nodes': nodes, 'Events': []}, ensure_ascii=False, indent=1) + '\n', encoding='utf-8')

sp = HERO / 'scenario.json'
s = json.loads(sp.read_text(encoding='utf-8'))
if not any(c['ID'] == '21' for c in s['Chapters']):
    t = sp.read_text(encoding='utf-8')
    s['Chapters'].append({'ID': '21', 'File': 'story21.json'})
    print('scenario.json: add chapter 21 by hand-format check')
    sp.write_text(json.dumps(s, ensure_ascii=False, indent=1) + '\n', encoding='utf-8')
print('ok')


# the walk about 여남 between Sun Qian's leaving and the alarm (S_runan: tools/scene_maps/runan.py)
def person(at, face, sprite, name, text, **more):
    return {'At': at, 'Face': face, 'Sprite': sprite, 'Faction': 'merchant', 'Talk': [[name, text]], **more}


explore = {
    'Hero': 'liubei', 'Start': {'Area': 'town', 'At': [19, 24], 'Face': 'NE'},
    'Areas': {
        'town': {
            'Map': 'S_runan', 'Name': '여남성', 'At': [19, 24],
            'Exits': [{'Name': '관청', 'At': [21, 23], 'Object': [21, 20], 'To': 'office'},
                      {'Name': '도구점', 'At': [9, 17], 'Object': [9, 15], 'To': 'tools'},
                      {'Name': '주점', 'At': [25, 17], 'Object': [25, 15], 'To': 'tavern'}],
            'People': {
                '관우@town': {'At': [17, 22], 'Face': 'SE', 'Sortie': {'Ask': '형님, 날이 저뭅니다. 관청으로 돌아가시겠습니까?'}},
                '장비@town': {'At': [11, 17], 'Face': 'SE', 'Talk': [['zhangfei', '형, 유벽이고 공도고 황건 출신이라더니 군사들 꼴은 제법 갖췄슈. 그래도 영 낯이 설어유.']]},
                '조운@town': {'At': [17, 4], 'Face': 'SW', 'Talk': [['zhaoyun', '성벽이 낮고 문이 넷입니다. 지킬 사람에 비해 문이 많습니다.']]},
                '고순@town': {'At': [13, 23], 'Face': 'SE', 'If': ['party:gaoshun'],
                            'Talk': [['gaoshun', '유벽의 군사를 보았습니다. 수는 되나 진을 짜 본 적이 없는 군사입니다.']]},
                '유벽@town': {'At': [29, 10], 'Face': 'SW', 'Talk': [['liupi', '성벽은 사흘이면 손볼 수 있소. 공도가 군량만 제때 거둬 오면 한 철은 버티오.']]},
                '미축@town': {'At': [23, 10], 'Face': 'SW', 'If': ['party:mizhu'],
                            'Talk': [['mizhu', '여남은 장이 작습니다. 군량을 사들이려 해도 팔 사람이 없습니다.']]},
                '장사꾼': person([20, 11], 'SW', 'commoner_man', '장사꾼', '북쪽이 정리되면 조조는 어디를 보겠소. 허도에서 여남은 지척이오.'),
                '유벽의 군사': {**person([15, 2], 'SE', 'light_infantry', '유벽의 군사',
                                    '조조 밑의 악진이란 장수는 갑옷이 거울처럼 번쩍여서 활로 맞히기가 쉽지 않답디다.'), 'Faction': 'shu'},
                '유벽의 군사#1': {'At': [19, 2], 'Face': 'SW', 'Sprite': 'light_infantry', 'Faction': 'shu'},
                '노인': person([12, 9], 'NE', 'commoner_elder', '노인', '황건 때도, 원술 때도 이 성은 주인이 바뀌었지. 이번엔 얼마나 갈는지.'),
            }},
        'office': {'Map': 'S_zhuo_office', 'Name': '여남성 · 관청', 'At': [9, 5],
                   'Exits': [{'Name': '거리로', 'At': [13, 9], 'To': 'town'}],
                   'People': {'군사#1': {'At': [3, 5], 'Face': 'SE', 'Sprite': 'light_infantry', 'Faction': 'shu'},
                              '군사#2': {'At': [13, 5], 'Face': 'SW', 'Sprite': 'light_infantry', 'Faction': 'shu'}}},
        'tools': {'Map': 'S_tool_shop', 'Name': '도구점', 'At': [6, 7],
                  'Exits': [{'Name': '거리로', 'At': [11, 7], 'To': 'town'}],
                  'People': {'상인': {'At': [9, 3], 'Face': 'SW', 'Sprite': 'innkeeper_noncombat', 'Faction': 'merchant', 'Shop': 'beihai'}}},
        'tavern': {'Map': 'S_tavern', 'Name': '주점', 'At': [7, 7],
                   'Exits': [{'Name': '거리로', 'At': [13, 9], 'To': 'town'}],
                   'People': {
                       '점원': {'At': [11, 3], 'Face': 'SW', 'Sprite': 'commoner_woman', 'Faction': 'merchant', 'Shop': 'beihai_tavern'},
                       '주점 주인': person([8, 4], 'SW', 'innkeeper_noncombat', '주점 주인',
                                       '오소의 군량이 하룻밤에 다 탔다지요. 허유라는 자가 조조에게 넘어가 길을 일러 줬답니다.'),
                       '간옹@tavern': {'At': [9, 7], 'Face': 'NW',
                                     'Talk': [['jianyong', '손건이 형주까지 다녀오자면 달포는 걸릴 걸세. 조조가 그 달포를 줄지 모르겠네.']]},
                       '주점 손님': person([5, 6], 'NE', 'commoner_man', '주점 손님', '들었소? 관도에서 원 대장군이 크게 졌다 하오.'),
                   }},
    }}
(HERO / 'explore21.json').write_text(json.dumps(explore, ensure_ascii=False, indent=1) + '\n', encoding='utf-8')
