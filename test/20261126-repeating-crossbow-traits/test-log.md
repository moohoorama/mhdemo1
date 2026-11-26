# 테스트 기록: 20261126-repeating-crossbow-traits

요청 파일: /Users/younwookim/orca/mhdemo1/test/20261126-repeating-crossbow-traits/20261126-repeating-crossbow-traits.md
진행 기록: /Users/younwookim/orca/mhdemo1/process/20261126-repeating-crossbow-traits/progress.md

테스트 단계 진입: 2026-11-26 06:22:33 KST(Asia/Seoul, UTC+09:00)

## [crossbow-data-match] PASS

- 시작: 2026-11-26 06:22:33 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-26 06:22:33 KST(Asia/Seoul, UTC+09:00)
- 명령: `python3 -c 
import json
c=json.load(open('assets/data/classes.json'))['Classes']['repeating_crossbow']
t=json.load(open('assets/data/traits.json'))['Traits']
pool={'exhort':700,'penetrate_art':1000,'rapid_art':1000,'far_sight':1000,'attack_increase':1000}
for k,cost in pool.items():
    assert k in c['Pool'], k
    assert t[k]['Cost']==cost, (k,t[k]['Cost'])
assert c['Pool']==list(pool), c['Pool']
plan={10:'accuracy_up',20:'fire_arrow_art',30:'relief',40:'rapid_art',49:'snipe_prep',50:'penetrate_art',60:'far_sight'}
assert {p['Level']:p['Trait'] for p in c['Plan']}==plan
for k in plan.values(): assert k in t
print('OK pool and plan match summary; costs:', {k:t[k]['Cost'] for k in list(pool)+list(plan.values())})
` (cwd `/Users/younwookim/orca/mhdemo1/ver5`)
- 필수: 예
- 종료 코드: 0

```
OK pool and plan match summary; costs: {'exhort': 700, 'penetrate_art': 1000, 'rapid_art': 1000, 'far_sight': 1000, 'attack_increase': 1000, 'accuracy_up': 500, 'fire_arrow_art': 500, 'relief': 500, 'snipe_prep': 700}
```

### 2026-11-26 06:22:37 KST(Asia/Seoul, UTC+09:00)

완료: 필수 테스트 1개 모두 최근 결과 PASS. 요청 파일을 /Users/younwookim/orca/mhdemo1/result/20261126/20261126-repeating-crossbow-traits/20261126-repeating-crossbow-traits.md 로 옮김

