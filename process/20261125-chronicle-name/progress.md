# 진행 기록: 20261125-chronicle-name

시각은 기록한 순간의 실제 시계(Asia/Seoul)다. `heartbeat`는 1분마다 자동 기록하고,
`status`/`단계`/`테스트` 행은 작업자·처리기가 사건이 생길 때 남긴다.

| 시각 | 구분 | 내용 |
|---|---|---|
| 2026-11-25 18:05:31 KST(Asia/Seoul, UTC+09:00) | 단계 | request/20261125-chronicle-name.md -> process/20261125-chronicle-name/20261125-chronicle-name.md 이동, 작업 시작 |
| 2026-11-25 18:05:31 KST(Asia/Seoul, UTC+09:00) | status:working | 수행: 요청 접수 / 다음: 요청 읽기 |
| 2026-11-25 18:05:31 KST(Asia/Seoul, UTC+09:00) | heartbeat | 시작 pid=4040 간격=60초 |
| 2026-11-25 18:06:08 KST(Asia/Seoul, UTC+09:00) | status:working | 수행: 문구 교체·history 완료 / 다음: 빌드·테스트·캡처 |
| 2026-11-25 18:06:08 KST(Asia/Seoul, UTC+09:00) | 단계 | process/20261125-chronicle-name/ -> test/20261125-chronicle-name/20261125-chronicle-name.md 이동, 테스트 시작 |
| 2026-11-25 18:06:08 KST(Asia/Seoul, UTC+09:00) | status:testing | 수행: 테스트 실행 / 다음: 필수 테스트 전부 통과 확인 |
| 2026-11-25 18:06:09 KST(Asia/Seoul, UTC+09:00) | 테스트 | 시작 [build] bash -c cd ver5 && go build ./... && go vet ./internal/gui ./internal/session ./internal/storage |
| 2026-11-25 18:06:16 KST(Asia/Seoul, UTC+09:00) | 테스트 | 종료 [build] PASS (exit 0) |
| 2026-11-25 18:06:19 KST(Asia/Seoul, UTC+09:00) | 테스트 | 시작 [build] bash -c cd ver5 && go build ./... && go vet ./internal/gui ./internal/session ./internal/storage |
| 2026-11-25 18:06:21 KST(Asia/Seoul, UTC+09:00) | 테스트 | 종료 [build] PASS (exit 0) |
| 2026-11-25 18:06:24 KST(Asia/Seoul, UTC+09:00) | 테스트 | 시작 [chronicle-tests] bash -c cd /Users/younwookim/orca/mhdemo1/ver5 && go test -p 1 ./internal/session ./internal/storage -run "Chronicle/B12" && go test -p 1 ./internal/gui -run "Chronicle/Menu/Title/Window" |
| 2026-11-25 18:06:31 KST(Asia/Seoul, UTC+09:00) | heartbeat | 단계=test / 상태=testing / 수행: 테스트 실행 / 다음: 필수 테스트 전부 통과 확인 |
| 2026-11-25 18:06:33 KST(Asia/Seoul, UTC+09:00) | 테스트 | 종료 [chronicle-tests] PASS (exit 0) |
| 2026-11-25 18:06:54 KST(Asia/Seoul, UTC+09:00) | 테스트메모 | 완료: 필수 테스트 2개 모두 최근 결과 PASS. 요청 파일을 /Users/younwookim/orca/mhdemo1/result/20261125/20261125-chronicle-name/20261125-chronicle-name.md 로 옮김 |
| 2026-11-25 18:06:54 KST(Asia/Seoul, UTC+09:00) | status:done | 수행: 완료, result로 이동 |
| 2026-11-25 18:07:31 KST(Asia/Seoul, UTC+09:00) | heartbeat | 종료: 요청이 result 단계 |
