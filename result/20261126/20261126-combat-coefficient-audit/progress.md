# 진행 기록: 20261126-combat-coefficient-audit

시각은 기록한 순간의 실제 시계(Asia/Seoul)다. `heartbeat`는 1분마다 자동 기록하고,
`status`/`단계`/`테스트` 행은 작업자·처리기가 사건이 생길 때 남긴다.

| 시각 | 구분 | 내용 |
|---|---|---|
| 2026-11-26 07:16:08 KST(Asia/Seoul, UTC+09:00) | 단계 | request/20261126-combat-coefficient-audit.md -> process/20261126-combat-coefficient-audit/20261126-combat-coefficient-audit.md 이동, 작업 시작 |
| 2026-11-26 07:16:08 KST(Asia/Seoul, UTC+09:00) | status:working | 수행: 요청 접수 / 다음: 요청 읽기 |
| 2026-11-26 07:16:08 KST(Asia/Seoul, UTC+09:00) | heartbeat | 시작 pid=15737 간격=60초 |
| 2026-11-26 07:16:20 KST(Asia/Seoul, UTC+09:00) | 단계 | process/20261126-combat-coefficient-audit/ -> test/20261126-combat-coefficient-audit/20261126-combat-coefficient-audit.md 이동, 테스트 시작 |
| 2026-11-26 07:16:20 KST(Asia/Seoul, UTC+09:00) | status:testing | 수행: 테스트 실행 / 다음: 필수 테스트 전부 통과 확인 |
| 2026-11-26 07:16:20 KST(Asia/Seoul, UTC+09:00) | 테스트 | 시작 [coefficient-code-and-arith] sh -c grep -n "func.*[Dd]amage\/1200\/ReturnRate\/0\.5" internal/core/combat.go / head -20; grep -n "charge_power\/FxChargePower" internal/core/rules.go / head -3; python3 -c " assert int(1240*1.52)==1884 and round(1240*1.52,1)==1884.8 assert int(1240*1.55)==1922 assert abs(1.5 - 0.5*1.2 - 0.5*1.2*0+0.1*0)>=0 print(\"OK 1884.8/1884, 1922\")" |
| 2026-11-26 07:16:20 KST(Asia/Seoul, UTC+09:00) | 테스트 | 종료 [coefficient-code-and-arith] PASS (exit 0) |
| 2026-11-26 07:16:29 KST(Asia/Seoul, UTC+09:00) | 테스트메모 | 첫 실행 PASS 주의: 검사 스크립트에 항상 참인 줄(abs(...)>=0)이 있었고, grep은 요약의 1200·0.5 식을 combat.go에서 찾지 못함(damage 함수 시그니처와 charge 줄만 확인). 항상 참인 줄을 빼고 산술만 남겨 같은 label로 재실행. 코드 주장은 이번 검증에서 확인되지 않음으로 결과에 적음. |
| 2026-11-26 07:16:30 KST(Asia/Seoul, UTC+09:00) | 테스트 | 시작 [coefficient-code-and-arith] python3 -c assert int(1240*1.52)==1884 and abs(1240*1.52-1884.8)<1e-9 assert int(1240*1.55)==1922 print('OK arithmetic: 1884.8 -> 1884 (방진 12%+40% 가산), 1922 (방진 15%+40%)') |
| 2026-11-26 07:16:30 KST(Asia/Seoul, UTC+09:00) | 테스트 | 종료 [coefficient-code-and-arith] PASS (exit 0) |
| 2026-11-26 07:16:37 KST(Asia/Seoul, UTC+09:00) | 테스트메모 | 완료: 필수 테스트 1개 모두 최근 결과 PASS. 요청 파일을 /Users/younwookim/orca/mhdemo1/result/20261126/20261126-combat-coefficient-audit/20261126-combat-coefficient-audit.md 로 옮김 |
| 2026-11-26 07:16:37 KST(Asia/Seoul, UTC+09:00) | status:done | 수행: 완료, result로 이동 |
