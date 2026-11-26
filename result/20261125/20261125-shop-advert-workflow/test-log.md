# 테스트 기록: 20261125-shop-advert-workflow

요청 파일: /Users/younwookim/orca/mhdemo1/test/20261125-shop-advert-workflow/20261125-shop-advert-workflow.md
진행 기록: /Users/younwookim/orca/mhdemo1/process/20261125-shop-advert-workflow/progress.md

테스트 단계 진입: 2026-11-25 08:27:06 KST(Asia/Seoul, UTC+09:00)

## [requestflow-unit] PASS

- 시작: 2026-11-25 08:27:10 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 08:27:23 KST(Asia/Seoul, UTC+09:00)
- 명령: `python3 -m unittest tools/request_mcp/test_requestflow.py` (cwd `/Users/younwookim/orca/mhdemo1`)
- 필수: 예
- 종료 코드: 0

```
..............
----------------------------------------------------------------------
Ran 14 tests in 13.319s

OK
```

## [go-build] PASS

- 시작: 2026-11-25 08:27:23 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 08:27:26 KST(Asia/Seoul, UTC+09:00)
- 명령: `go build ./...` (cwd `/Users/younwookim/orca/mhdemo1/ver5`)
- 필수: 예
- 종료 코드: 0

```

```

## [go-vet] PASS

- 시작: 2026-11-25 08:27:26 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 08:27:27 KST(Asia/Seoul, UTC+09:00)
- 명령: `go vet ./...` (cwd `/Users/younwookim/orca/mhdemo1/ver5`)
- 필수: 예
- 종료 코드: 0

```

```

## [go-test-content] PASS

- 시작: 2026-11-25 08:27:30 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 08:27:48 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test -p 1 -count=1 ./internal/content` (cwd `/Users/younwookim/orca/mhdemo1/ver5`)
- 필수: 예
- 종료 코드: 0

```
ok  	srpg/internal/content	17.648s
```

## [go-test-scenegraph] PASS

- 시작: 2026-11-25 08:27:49 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 08:28:16 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test -p 1 -count=1 ./internal/scenegraph` (cwd `/Users/younwookim/orca/mhdemo1/ver5`)
- 필수: 예
- 종료 코드: 0

```
ok  	srpg/internal/scenegraph	24.976s
```

## [go-test-core] PASS

- 시작: 2026-11-25 08:28:19 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 08:29:08 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test -p 1 -count=1 ./internal/core` (cwd `/Users/younwookim/orca/mhdemo1/ver5`)
- 필수: 예
- 종료 코드: 0

```
ok  	srpg/internal/core	47.714s
```

### 2026-11-25 08:32:11 KST(Asia/Seoul, UTC+09:00)

임시 덤프 시험 파일(zz_tmp_shopdump_test.go) 삭제와 gofmt 정리 뒤라 content·scenegraph·build·vet을 같은 label로 재실행한다.

## [go-build] PASS

- 시작: 2026-11-25 08:32:11 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 08:32:14 KST(Asia/Seoul, UTC+09:00)
- 명령: `go build ./...` (cwd `/Users/younwookim/orca/mhdemo1/ver5`)
- 필수: 예
- 종료 코드: 0

```

```

## [go-vet] PASS

- 시작: 2026-11-25 08:32:14 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 08:32:16 KST(Asia/Seoul, UTC+09:00)
- 명령: `go vet ./...` (cwd `/Users/younwookim/orca/mhdemo1/ver5`)
- 필수: 예
- 종료 코드: 0

```

```

## [go-test-content] PASS

- 시작: 2026-11-25 08:32:16 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 08:32:36 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test -p 1 -count=1 ./internal/content` (cwd `/Users/younwookim/orca/mhdemo1/ver5`)
- 필수: 예
- 종료 코드: 0

```
ok  	srpg/internal/content	17.783s
```

## [go-test-scenegraph] PASS

- 시작: 2026-11-25 08:32:36 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 08:33:03 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test -p 1 -count=1 ./internal/scenegraph` (cwd `/Users/younwookim/orca/mhdemo1/ver5`)
- 필수: 예
- 종료 코드: 0

```
ok  	srpg/internal/scenegraph	23.960s
```

## [go-test-gui-explore] PASS

- 시작: 2026-11-25 08:33:03 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 08:33:33 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test -p 1 -count=1 ./internal/gui -run Explore|Shop|Guide|Walk` (cwd `/Users/younwookim/orca/mhdemo1/ver5`)
- 필수: 예
- 종료 코드: 0

```
ok  	srpg/internal/gui	26.735s
```

## [go-test-cli-session] PASS

- 시작: 2026-11-25 08:33:34 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 08:33:39 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test -p 1 -count=1 ./internal/cli ./internal/session` (cwd `/Users/younwookim/orca/mhdemo1/ver5`)
- 필수: 예
- 종료 코드: 0

```
ok  	srpg/internal/cli	0.477s
ok  	srpg/internal/session	2.173s
```

### 2026-11-25 08:34:04 KST(Asia/Seoul, UTC+09:00)

완료: 필수 테스트 8개 모두 최근 결과 PASS. 요청 파일을 /Users/younwookim/orca/mhdemo1/result/20261125/20261125-shop-advert-workflow/20261125-shop-advert-workflow.md 로 옮김

