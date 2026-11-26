# 테스트 기록: 20261125-shop-advert-no-numbers

요청 파일: /Users/younwookim/orca/mhdemo1/test/20261125-shop-advert-no-numbers/20261125-shop-advert-no-numbers.md
진행 기록: /Users/younwookim/orca/mhdemo1/process/20261125-shop-advert-no-numbers/progress.md

테스트 단계 진입: 2026-11-25 19:08:49 KST(Asia/Seoul, UTC+09:00)

## [scenegraph-shop] FAIL

- 시작: 2026-11-25 19:08:50 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 19:09:21 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test -p 1 -count=1 ./internal/scenegraph` (cwd `.`)
- 필수: 예
- 종료 코드: 1

```
--- FAIL: TestShopsWithNewStockAreAnnouncedNearTheirDoor (2.21s)
    graph_test.go:170: 
        	Error Trace:	/Users/younwookim/orca/mhdemo1/ver5/internal/scenegraph/graph_test.go:151
        	            				/Users/younwookim/orca/mhdemo1/ver5/internal/scenegraph/graph_test.go:170
        	Error:      	Should be empty, but was [area:explore03/street luoyang_tavern]
        	Test:       	TestShopsWithNewStockAreAnnouncedNearTheirDoor
        	Messages:   	a shop with new stock has nobody near its door telling so (Notice "shop:<id>")
FAIL
FAIL	srpg/internal/scenegraph	27.897s
FAIL
```

## [scenegraph-shop] PASS

- 시작: 2026-11-25 19:09:36 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 19:20:59 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test -p 1 -count=1 ./internal/scenegraph` (cwd `ver5`)
- 필수: 예
- 종료 코드: 0

```
ok  	srpg/internal/scenegraph	30.493s
```

### 2026-11-25 19:21:02 KST(Asia/Seoul, UTC+09:00)

cause: regex unit 치 matched '만만치' in 특급주 line (false positive idiom). fix: units limited to 냥/푼/퍼센트/%, idiom added to test; rerun same label

## [core-advert] PASS

- 시작: 2026-11-25 19:22:48 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 19:22:52 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test -p 1 -count=1 -run TestNewStockIsToldOutdoorsAndSoldInside|TestAShopSeenAgainIsNotCriedAgain ./internal/core` (cwd `ver5`)
- 필수: 예
- 종료 코드: 0

```
ok  	srpg/internal/core	1.871s
```

## [content-stock] PASS

- 시작: 2026-11-25 19:22:56 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 19:23:17 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test -p 1 -count=1 ./internal/content` (cwd `ver5`)
- 필수: 예
- 종료 코드: 0

```
ok  	srpg/internal/content	19.405s
```

## [go-build] PASS

- 시작: 2026-11-25 19:23:17 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 19:23:20 KST(Asia/Seoul, UTC+09:00)
- 명령: `go build ./...` (cwd `ver5`)
- 필수: 예
- 종료 코드: 0

```

```

## [core-shop-explore] PASS

- 시작: 2026-11-25 19:23:24 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 19:23:27 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test -p 1 -count=1 -run Shop|Explore|Notice|Talk ./internal/core` (cwd `ver5`)
- 필수: 예
- 종료 코드: 0

```
ok  	srpg/internal/core	1.632s
```

### 2026-11-25 19:26:11 KST(Asia/Seoul, UTC+09:00)

완료: 필수 테스트 5개 모두 최근 결과 PASS. 요청 파일을 /Users/younwookim/orca/mhdemo1/result/20261125/20261125-shop-advert-no-numbers/20261125-shop-advert-no-numbers.md 로 옮김

