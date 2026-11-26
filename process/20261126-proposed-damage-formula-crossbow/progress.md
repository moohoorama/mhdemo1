# 진행 기록: 20261126-proposed-damage-formula-crossbow

시각은 기록한 순간의 실제 시계(Asia/Seoul)다. `heartbeat`는 1분마다 자동 기록하고,
`status`/`단계`/`테스트` 행은 작업자·처리기가 사건이 생길 때 남긴다.

| 시각 | 구분 | 내용 |
|---|---|---|
| 2026-11-26 07:21:05 KST(Asia/Seoul, UTC+09:00) | 단계 | request/20261126-proposed-damage-formula-crossbow.md -> process/20261126-proposed-damage-formula-crossbow/20261126-proposed-damage-formula-crossbow.md 이동, 작업 시작 |
| 2026-11-26 07:21:05 KST(Asia/Seoul, UTC+09:00) | status:working | 수행: 요청 접수 / 다음: 요청 읽기 |
| 2026-11-26 07:21:05 KST(Asia/Seoul, UTC+09:00) | heartbeat | 시작 pid=16932 간격=60초 |
| 2026-11-26 07:21:13 KST(Asia/Seoul, UTC+09:00) | 단계 | process/20261126-proposed-damage-formula-crossbow/ -> test/20261126-proposed-damage-formula-crossbow/20261126-proposed-damage-formula-crossbow.md 이동, 테스트 시작 |
| 2026-11-26 07:21:13 KST(Asia/Seoul, UTC+09:00) | status:testing | 수행: 테스트 실행 / 다음: 필수 테스트 전부 통과 확인 |
| 2026-11-26 07:21:14 KST(Asia/Seoul, UTC+09:00) | 테스트 | 시작 [damage-formula-arith] python3 -c A,D=1300,round(1120*1.05) assert D==1176 d=A-D assert d==124 assert abs(d*1+1200-1324)<1e-9 # 일반 assert abs(d*0.5+1200-1262)<1e-9 # 반격 제안 (규칙계수 1, 반격 -50%) assert abs(d*1.5+1200-1386)<1e-9 # 치명 제안 assert abs(d*1.1+1200-1336.4)<1e-9 # 거리4 원사강화 +10% 제안 assert abs(1324*0.5-662)<1e-9 # 현행 반격 assert abs(1324*1.5-1986)<1e-9 # 현행 치명 assert abs(1324*1.1-1456.4)<1e-9 # 현행 원사강화 print('OK 1324 1262 1386 1336.4 / current 662 1986 1456.4 (A=1300, D=1176 assumed)') |
| 2026-11-26 07:21:14 KST(Asia/Seoul, UTC+09:00) | 테스트 | 종료 [damage-formula-arith] PASS (exit 0) |
| 2026-11-26 07:21:19 KST(Asia/Seoul, UTC+09:00) | 테스트메모 | 완료: 필수 테스트 1개 모두 최근 결과 PASS. 요청 파일을 /Users/younwookim/orca/mhdemo1/result/20261126/20261126-proposed-damage-formula-crossbow/20261126-proposed-damage-formula-crossbow.md 로 옮김 |
| 2026-11-26 07:21:19 KST(Asia/Seoul, UTC+09:00) | status:done | 수행: 완료, result로 이동 |
| 2026-11-26 07:22:05 KST(Asia/Seoul, UTC+09:00) | heartbeat | 종료: 요청이 result 단계 |
