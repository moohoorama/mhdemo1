# 진행 기록: npc-trait-prerequisites-2

시각은 기록한 순간의 실제 시계(Asia/Seoul)다. `heartbeat`는 1분마다 자동 기록하고,
`status`/`단계`/`테스트` 행은 작업자·처리기가 사건이 생길 때 남긴다.

| 시각 | 구분 | 내용 |
|---|---|---|
| 2026-11-26 07:11:49 KST(Asia/Seoul, UTC+09:00) | 단계 | request/npc-trait-prerequisites.md -> process/npc-trait-prerequisites-2/npc-trait-prerequisites-2.md 이동, 작업 시작 |
| 2026-11-26 07:11:49 KST(Asia/Seoul, UTC+09:00) | status:working | 수행: 요청 접수 / 다음: 요청 읽기 |
| 2026-11-26 07:11:49 KST(Asia/Seoul, UTC+09:00) | heartbeat | 시작 pid=14372 간격=60초 |
| 2026-11-26 07:12:49 KST(Asia/Seoul, UTC+09:00) | heartbeat | 단계=process / 상태=working / 수행: 요청 접수 / 다음: 요청 읽기 |
| 2026-11-26 07:13:37 KST(Asia/Seoul, UTC+09:00) | 단계 | process/npc-trait-prerequisites-2/ -> test/npc-trait-prerequisites-2/npc-trait-prerequisites-2.md 이동, 테스트 시작 |
| 2026-11-26 07:13:37 KST(Asia/Seoul, UTC+09:00) | status:testing | 수행: 테스트 실행 / 다음: 필수 테스트 전부 통과 확인 |
| 2026-11-26 07:13:37 KST(Asia/Seoul, UTC+09:00) | 테스트 | 시작 [npc-prerequisites] go test ./internal/content ./internal/core -run TestResolveNPCPlannedPrerequisites/TestNPCPlannedPrerequisitesAtSpawn/TestOfficerPlannedTraitPrerequisites/TestOfficerOwnedTraitPrerequisites/TestClassPlannedTraitPrerequisites/TestPrerequisitesGateLearning/TestPoisonAndStrictOrderPrerequisites/TestRepeatingCrossbowPlanOrder -count=1 |
| 2026-11-26 07:13:45 KST(Asia/Seoul, UTC+09:00) | 테스트 | 종료 [npc-prerequisites] PASS (exit 0) |
| 2026-11-26 07:13:49 KST(Asia/Seoul, UTC+09:00) | 테스트 | 시작 [content-core-packages] go test ./internal/content ./internal/core -count=1 |
| 2026-11-26 07:13:49 KST(Asia/Seoul, UTC+09:00) | heartbeat | 단계=test / 상태=testing / 수행: 테스트 실행 / 다음: 필수 테스트 전부 통과 확인 |
| 2026-11-26 07:14:39 KST(Asia/Seoul, UTC+09:00) | 테스트 | 시작 [spawn-equipment-skills] go test ./internal/core -run ^TestNPCPlannedPrerequisitesAtSpawn$ -count=1 |
| 2026-11-26 07:14:44 KST(Asia/Seoul, UTC+09:00) | 테스트 | 종료 [spawn-equipment-skills] PASS (exit 0) |
| 2026-11-26 07:14:49 KST(Asia/Seoul, UTC+09:00) | heartbeat | 단계=test / 상태=testing / 수행: 테스트 실행 / 다음: 필수 테스트 전부 통과 확인 |
| 2026-11-26 07:15:30 KST(Asia/Seoul, UTC+09:00) | 테스트 | 종료 [content-core-packages] FAIL (exit 1) |
| 2026-11-26 07:15:34 KST(Asia/Seoul, UTC+09:00) | 테스트메모 | 기존 중복 금지 검사는 유지했다. 기존 데이터의 순유·법정·유비·허사·유벽·황보숭·하후연·번궁 중복 8건은 이번 요청 범위 밖이며 별도 수정이 필요하다. |
| 2026-11-26 07:15:46 KST(Asia/Seoul, UTC+09:00) | 테스트메모 | 실패 상태: content/core 전체 검사에서 기존 고유·병종 특성 중복 8건이 실패한다. 다음 조치: 장수별 특성 배치 수정안 결정 후 데이터를 수정하고 전체 검사를 재실행한다. |
| 2026-11-26 07:15:46 KST(Asia/Seoul, UTC+09:00) | status:failed | 수행: 필수 검증 미완료 / 다음: 장수별 특성 배치 수정안 결정 후 데이터를 수정하고 전체 검사를 재실행한다. / 장애: content/core 전체 검사에서 기존 고유·병종 특성 중복 8건이 실패한다. |
| 2026-11-26 07:15:49 KST(Asia/Seoul, UTC+09:00) | heartbeat | 단계=test / 상태=failed / 수행: 필수 검증 미완료 / 다음: 장수별 특성 배치 수정안 결정 후 데이터를 수정하고 전체 검사를 재실행한다. / 장애: content/core 전체 검사에서 기존 고유·병종 특성 중복 8건이 실패한다. |
| 2026-11-26 07:15:49 KST(Asia/Seoul, UTC+09:00) | heartbeat | 종료: 상태 failed |
