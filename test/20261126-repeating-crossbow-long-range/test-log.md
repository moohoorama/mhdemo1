# 테스트 기록: 20261126-repeating-crossbow-long-range

요청 파일: /Users/younwookim/orca/mhdemo1/test/20261126-repeating-crossbow-long-range/20261126-repeating-crossbow-long-range.md
진행 기록: /Users/younwookim/orca/mhdemo1/process/20261126-repeating-crossbow-long-range/progress.md

테스트 단계 진입: 2026-11-26 06:45:21 KST(Asia/Seoul, UTC+09:00)

## [crossbow-pool-current] PASS

- 시작: 2026-11-26 06:45:21 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-26 06:45:21 KST(Asia/Seoul, UTC+09:00)
- 명령: `python3 -c 
import json
c=json.load(open('assets/data/classes.json'))['Classes']['repeating_crossbow']
t=json.load(open('assets/data/traits.json'))['Traits']
assert 'exhort' in c['Pool'] and t['exhort']['Cost']==700
assert 'long' not in ' '.join(c['Pool'])
print('OK current pool still has exhort(700); no new distance trait in data (data unchanged as summary states)', c['Pool'])
` (cwd `/Users/younwookim/orca/mhdemo1/ver5`)
- 필수: 예
- 종료 코드: 0

```
OK current pool still has exhort(700); no new distance trait in data (data unchanged as summary states) ['exhort', 'penetrate_art', 'rapid_art', 'far_sight', 'attack_increase']
```

### 2026-11-26 06:45:31 KST(Asia/Seoul, UTC+09:00)

완료: 필수 테스트 1개 모두 최근 결과 PASS. 요청 파일을 /Users/younwookim/orca/mhdemo1/result/20261126/20261126-repeating-crossbow-long-range/20261126-repeating-crossbow-long-range.md 로 옮김

