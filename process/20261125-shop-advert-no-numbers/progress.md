# 진행 기록: 20261125-shop-advert-no-numbers

시각은 기록한 순간의 실제 시계(Asia/Seoul)다. `heartbeat`는 1분마다 자동 기록하고,
`status`/`단계`/`테스트` 행은 작업자·처리기가 사건이 생길 때 남긴다.

| 시각 | 구분 | 내용 |
|---|---|---|
| 2026-11-25 19:07:05 KST(Asia/Seoul, UTC+09:00) | 단계 | request/20261125-shop-advert-no-numbers.md -> process/20261125-shop-advert-no-numbers/20261125-shop-advert-no-numbers.md 이동, 작업 시작 |
| 2026-11-25 19:07:05 KST(Asia/Seoul, UTC+09:00) | status:working | 수행: 요청 접수 / 다음: 요청 읽기 |
| 2026-11-25 19:07:05 KST(Asia/Seoul, UTC+09:00) | heartbeat | 시작 pid=14674 간격=60초 |
| 2026-11-25 19:08:05 KST(Asia/Seoul, UTC+09:00) | heartbeat | 단계=process / 상태=working / 수행: 요청 접수 / 다음: 요청 읽기 |
| 2026-11-25 19:08:49 KST(Asia/Seoul, UTC+09:00) | status:working | 수행: 구현 완료, 시험 단계 / 다음: begin-test, run-test |
| 2026-11-25 19:08:49 KST(Asia/Seoul, UTC+09:00) | 단계 | process/20261125-shop-advert-no-numbers/ -> test/20261125-shop-advert-no-numbers/20261125-shop-advert-no-numbers.md 이동, 테스트 시작 |
| 2026-11-25 19:08:49 KST(Asia/Seoul, UTC+09:00) | status:testing | 수행: 테스트 실행 / 다음: 필수 테스트 전부 통과 확인 |
| 2026-11-25 19:08:50 KST(Asia/Seoul, UTC+09:00) | 테스트 | 시작 [scenegraph-shop] go test -p 1 -count=1 ./internal/scenegraph |
| 2026-11-25 19:09:05 KST(Asia/Seoul, UTC+09:00) | heartbeat | 단계=test / 상태=testing / 수행: 테스트 실행 / 다음: 필수 테스트 전부 통과 확인 |
| 2026-11-25 19:09:21 KST(Asia/Seoul, UTC+09:00) | 테스트 | 종료 [scenegraph-shop] FAIL (exit 1) |
| 2026-11-25 19:09:36 KST(Asia/Seoul, UTC+09:00) | 테스트 | 시작 [scenegraph-shop] go test -p 1 -count=1 ./internal/scenegraph |
| 2026-11-25 19:10:05 KST(Asia/Seoul, UTC+09:00) | heartbeat | 단계=test / 상태=testing / 수행: 테스트 실행 / 다음: 필수 테스트 전부 통과 확인 |
| 2026-11-25 19:20:59 KST(Asia/Seoul, UTC+09:00) | 테스트 | 종료 [scenegraph-shop] PASS (exit 0) |
| 2026-11-25 19:21:02 KST(Asia/Seoul, UTC+09:00) | 테스트메모 | cause: regex unit 치 matched '만만치' in 특급주 line (false positive idiom). fix: units limited to 냥/푼/퍼센트/%, idiom added to test; rerun same label |
| 2026-11-25 19:21:54 KST(Asia/Seoul, UTC+09:00) | heartbeat | 단계=test / 상태=testing / 수행: 테스트 실행 / 다음: 필수 테스트 전부 통과 확인 / 작업자 마지막 보고 2026-11-25 19:08:49 KST (13분 전) 이후 새 보고 없음 |
| 2026-11-25 19:22:48 KST(Asia/Seoul, UTC+09:00) | 테스트 | 시작 [core-advert] go test -p 1 -count=1 -run TestNewStockIsToldOutdoorsAndSoldInside/TestAShopSeenAgainIsNotCriedAgain ./internal/core |
| 2026-11-25 19:22:52 KST(Asia/Seoul, UTC+09:00) | 테스트 | 종료 [core-advert] PASS (exit 0) |
| 2026-11-25 19:22:54 KST(Asia/Seoul, UTC+09:00) | heartbeat | 단계=test / 상태=testing / 수행: 테스트 실행 / 다음: 필수 테스트 전부 통과 확인 / 작업자 마지막 보고 2026-11-25 19:08:49 KST (14분 전) 이후 새 보고 없음 |
| 2026-11-25 19:22:56 KST(Asia/Seoul, UTC+09:00) | 테스트 | 시작 [content-stock] go test -p 1 -count=1 ./internal/content |
| 2026-11-25 19:23:17 KST(Asia/Seoul, UTC+09:00) | 테스트 | 종료 [content-stock] PASS (exit 0) |
| 2026-11-25 19:23:17 KST(Asia/Seoul, UTC+09:00) | 테스트 | 시작 [go-build] go build ./... |
| 2026-11-25 19:23:20 KST(Asia/Seoul, UTC+09:00) | 테스트 | 종료 [go-build] PASS (exit 0) |
| 2026-11-25 19:23:24 KST(Asia/Seoul, UTC+09:00) | 테스트 | 시작 [core-shop-explore] go test -p 1 -count=1 -run Shop/Explore/Notice/Talk ./internal/core |
| 2026-11-25 19:23:27 KST(Asia/Seoul, UTC+09:00) | 테스트 | 종료 [core-shop-explore] PASS (exit 0) |
| 2026-11-25 19:23:54 KST(Asia/Seoul, UTC+09:00) | heartbeat | 단계=test / 상태=testing / 수행: 테스트 실행 / 다음: 필수 테스트 전부 통과 확인 / 작업자 마지막 보고 2026-11-25 19:08:49 KST (15분 전) 이후 새 보고 없음 |
| 2026-11-25 19:26:11 KST(Asia/Seoul, UTC+09:00) | 테스트메모 | 완료: 필수 테스트 5개 모두 최근 결과 PASS. 요청 파일을 /Users/younwookim/orca/mhdemo1/result/20261125/20261125-shop-advert-no-numbers/20261125-shop-advert-no-numbers.md 로 옮김 |
| 2026-11-25 19:26:11 KST(Asia/Seoul, UTC+09:00) | status:done | 수행: 완료, result로 이동 |
| 2026-11-25 19:26:29 KST(Asia/Seoul, UTC+09:00) | heartbeat | 종료: 요청이 result 단계 |
