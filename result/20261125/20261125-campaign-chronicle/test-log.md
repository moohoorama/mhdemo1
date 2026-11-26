# 테스트 기록: 20261125-campaign-chronicle

요청 파일: /Users/younwookim/orca/mhdemo1/test/20261125-campaign-chronicle/20261125-campaign-chronicle.md
진행 기록: /Users/younwookim/orca/mhdemo1/process/20261125-campaign-chronicle/progress.md

테스트 단계 진입: 2026-11-25 10:17:51 KST(Asia/Seoul, UTC+09:00)

## [build-vet] PASS

- 시작: 2026-11-25 10:18:00 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 10:18:04 KST(Asia/Seoul, UTC+09:00)
- 명령: `sh -c go build ./... && go vet ./internal/core ./internal/storage ./internal/session ./internal/gui` (cwd `ver5`)
- 필수: 예
- 종료 코드: 0

```

```

## [chronicle-core] PASS

- 시작: 2026-11-25 10:18:04 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 10:18:26 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test -p 1 ./internal/core -run TestEveryWaypointIsReachedAndKept|TestB12PointReplaysTheModeBranch -count=1` (cwd `ver5`)
- 필수: 예
- 종료 코드: 0

```
ok  	srpg/internal/core	20.302s
```

## [session-storage] PASS

- 시작: 2026-11-25 10:18:26 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 10:18:31 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test -p 1 ./internal/session ./internal/storage -count=1` (cwd `ver5`)
- 필수: 예
- 종료 코드: 0

```
ok  	srpg/internal/session	2.014s
ok  	srpg/internal/storage	0.694s
```

## [core-all] FAIL

- 시작: 2026-11-25 10:18:39 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 10:20:17 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test -p 1 ./internal/core ./internal/content ./internal/cli -count=1` (cwd `ver5`)
- 필수: 예
- 종료 코드: 1

```
ok  	srpg/internal/core	72.029s
--- FAIL: TestEngineSourceNamesNoGameContent (0.78s)
    purity_test.go:104: ../../internal/gui/uidemo_chronicle.go: "B12" is a stage of the data bundle
FAIL
FAIL	srpg/internal/content	21.958s
ok  	srpg/internal/cli	0.503s
FAIL
```

## [core-content-cli-rerun] PASS

- 시작: 2026-11-25 10:20:36 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 10:22:01 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test -p 1 ./internal/core ./internal/content ./internal/cli ./internal/storage ./internal/session -count=1` (cwd `ver5`)
- 필수: 예
- 종료 코드: 0

```
ok  	srpg/internal/core	63.550s
ok  	srpg/internal/content	17.722s
ok  	srpg/internal/cli	0.301s
ok  	srpg/internal/storage	0.565s
ok  	srpg/internal/session	1.987s
```

## [core-all] PASS

- 시작: 2026-11-25 10:22:12 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 10:23:35 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test -p 1 ./internal/core ./internal/content ./internal/cli -count=1` (cwd `ver5`)
- 필수: 예
- 종료 코드: 0

```
ok  	srpg/internal/core	62.813s
ok  	srpg/internal/content	17.498s
ok  	srpg/internal/cli	0.299s
```

## [gui-all] FAIL

- 시작: 2026-11-25 10:23:35 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 10:28:48 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test -p 1 ./internal/gui -count=1` (cwd `ver5`)
- 필수: 예
- 종료 코드: 1

```
signal: killed
FAIL	srpg/internal/gui	306.121s
FAIL
```

## [gui-targeted] PASS

- 시작: 2026-11-25 10:28:55 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 10:29:27 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test -p 1 ./internal/gui -run Waypoint|Save|Autosave|Chronicle|Overlay|Menu|Title|Window|Scroll -count=1` (cwd `ver5`)
- 필수: 예
- 종료 코드: 0

```
ok  	srpg/internal/gui	24.711s
```

### 2026-11-25 10:29:34 KST(Asia/Seoul, UTC+09:00)

실패 상태: 최근 결과가 실패인 필수 테스트: gui-all
다음 조치: 실패 원인 수정 후 재시험

## [gui-all] PASS

- 시작: 2026-11-25 10:29:49 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-25 10:33:27 KST(Asia/Seoul, UTC+09:00)
- 명령: `/private/tmp/claude-501/-Users-younwookim-orca-mhdemo1/d7f018bd-0ca7-420e-8845-abe182a8d7f6/scratchpad/gui_batches.sh` (cwd `ver5`)
- 필수: 예
- 종료 코드: 0

```

```

### 2026-11-25 10:33:33 KST(Asia/Seoul, UTC+09:00)

완료: 필수 테스트 7개 모두 최근 결과 PASS. 요청 파일을 /Users/younwookim/orca/mhdemo1/result/20261125/20261125-campaign-chronicle/20261125-campaign-chronicle.md 로 옮김

