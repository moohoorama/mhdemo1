"""Compare proposed bonuses using current class data and the documented damage formula."""
import json
import math
from pathlib import Path

DATA = Path(__file__).resolve().parents[3] / 'ver5' / 'assets' / 'data'
classes = json.loads((DATA / 'classes.json').read_text())['Classes']
terrain = json.loads((DATA / 'terrain.json').read_text())['Terrain']
level, attribute = 30, 80
stat = lambda bonus: math.floor((level + 10) * (400 / (140 - attribute) + bonus))
attack = stat(classes['repeating_crossbow']['Bonus'][0])
print(f'연노병 공격력 {attack}, 레벨 {level}, 무력·통솔 {attribute}, 장비·부관 없음')
for key in ['guards', 'royal_guard', 'heavy_infantry', 'heavy_cavalry', 'repeating_crossbow']:
    unit = classes[key]
    defense = stat(unit['Bonus'][1])
    factor = terrain[unit['Terrain']]['Ground']['g']['Factor']
    base = max(1, attack - defense * factor + (level + 10) * 20)
    print(unit['Name'], f'방어 {defense}, 지형 {factor}, 기본 {base:.2f}', end='')
    for allies in (1, 3, 5):
        buffed_attack = int(attack * (1 + .04 * allies))
        attack_damage = max(1, buffed_attack - defense * factor + 800)
        damage_bonus = base * (1 + .04 * allies)
        print(f' | {allies}기 공격력 {int(attack_damage)} / 피해 {int(damage_bonus)}', end='')
    print()
for label, defense in [('근위병', 826), ('방어 강화', int(826 * 1.2)),
                       ('밀집대형 5기 + 방어 강화', int(int(826 * 1.2) * 1.2))]:
    base = attack - defense + 800
    buffed = int(attack * 1.2) - defense + 800
    print(f'{label}: 방어 {defense}, 기본 {base}, 공격력 +20% {buffed}, 피해 +20% {int(base * 1.2)}')
