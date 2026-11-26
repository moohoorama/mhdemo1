# 계교 전투 여울 복병과 시작 대사

- `ver5/assets/scenario/hero/stage10.json`: 영역 1을 (14,12)~(19,17)로 옮기고 이름을 여울 건너편으로 바꿨다. 유비·관우·장비·조운 조건, 적군 차례 시작 조건과 4라운드 대체 조건은 유지했다. 원소·전풍의 시작 대사에서 문추와 전술 정보를 빼고 다리 수비 명령과 격려로 고쳤다.
- `ver5/docs/fullstory.md`: 계교 복병의 실제 배치·발동과 바뀐 시작 대사를 반영했다.
- `ver5/docs/data-speech-style.md`: 원소·전풍의 새 말투 예시를 기록했다.
- `ver5/docs/history.md`: 변경 내용과 의도를 기록했다.
- 설계는 `design.md`, 시작 화면은 `capture/`에 저장했다. 시작 화면에서 대사와 방향을 확인했다.
- 영역 및 조건의 JSON 검사, `git diff --check`, `storydb.py jargon`(0줄), `_facecheck B10`, `go test ./internal/content` 통과.
- `go test ./internal/core ./internal/content`: core의 강하 일기토 InvalidDeployment, 손건 잔류, c27v-start UnknownWaypoint 세 테스트 실패. 계교 일기토 개별 검사는 전투가 먼저 끝나 SKIP됐다. 실제 문추 발동을 게임 플레이로 재현하지 못했다.
- `_balance -stage B10 -seeds 5`: rush 0/5 승리, 평균 3.0라운드. 변경 전과 비교하지 않았으므로 원인은 판단할 수 없다.
- storydb 색인 갱신은 Ollama 연결 거부로 실패했다. `srpg-cli -slot 9` 전체 진행은 수행하지 못했다.
- 기존 변경을 보존했으며, 기존 감시 프로세스의 중복 수신을 피하기 위해 요청 기록을 process/에 직접 남겼다.
