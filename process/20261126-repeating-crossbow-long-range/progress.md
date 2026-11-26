# 진행 기록: 20261126-repeating-crossbow-long-range

시각은 기록한 순간의 실제 시계(Asia/Seoul)다. `heartbeat`는 1분마다 자동 기록하고,
`status`/`단계`/`테스트` 행은 작업자·처리기가 사건이 생길 때 남긴다.

| 시각 | 구분 | 내용 |
|---|---|---|
| 2026-11-26 06:45:08 KST(Asia/Seoul, UTC+09:00) | 단계 | request/20261126-repeating-crossbow-long-range.md -> process/20261126-repeating-crossbow-long-range/20261126-repeating-crossbow-long-range.md 이동, 작업 시작 |
| 2026-11-26 06:45:08 KST(Asia/Seoul, UTC+09:00) | status:working | 수행: 요청 접수 / 다음: 요청 읽기 |
| 2026-11-26 06:45:08 KST(Asia/Seoul, UTC+09:00) | heartbeat | 시작 pid=6546 간격=60초 |
| 2026-11-26 06:45:21 KST(Asia/Seoul, UTC+09:00) | 단계 | process/20261126-repeating-crossbow-long-range/ -> test/20261126-repeating-crossbow-long-range/20261126-repeating-crossbow-long-range.md 이동, 테스트 시작 |
| 2026-11-26 06:45:21 KST(Asia/Seoul, UTC+09:00) | status:testing | 수행: 테스트 실행 / 다음: 필수 테스트 전부 통과 확인 |
| 2026-11-26 06:45:21 KST(Asia/Seoul, UTC+09:00) | 테스트 | 시작 [crossbow-pool-current] python3 -c import json c=json.load(open('assets/data/classes.json'))['Classes']['repeating_crossbow'] t=json.load(open('assets/data/traits.json'))['Traits'] assert 'exhort' in c['Pool'] and t['exhort']['Cost']==700 assert 'long' not in ' '.join(c['Pool']) print('OK current pool still has exhort(700); no new distance trait in data (data unchanged as summary states)', c['Pool']) |
| 2026-11-26 06:45:21 KST(Asia/Seoul, UTC+09:00) | 테스트 | 종료 [crossbow-pool-current] PASS (exit 0) |
| 2026-11-26 06:45:31 KST(Asia/Seoul, UTC+09:00) | 테스트메모 | 완료: 필수 테스트 1개 모두 최근 결과 PASS. 요청 파일을 /Users/younwookim/orca/mhdemo1/result/20261126/20261126-repeating-crossbow-long-range/20261126-repeating-crossbow-long-range.md 로 옮김 |
| 2026-11-26 06:45:31 KST(Asia/Seoul, UTC+09:00) | status:done | 수행: 완료, result로 이동 |
| 2026-11-26 06:46:08 KST(Asia/Seoul, UTC+09:00) | heartbeat | 종료: 요청이 result 단계 |
