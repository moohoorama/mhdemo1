# 조운 작별 B안 및 정비 탐험 결과

- `ver5/assets/scenario/hero/story13.json`: 선택한 B안 대본을 적용하고 조운의 북평 귀환 선언 직후 정비 탐험으로 전환한다. 조운에게 말을 걸면 후속 송별 대화와 다음 날 성문 작별이 이어진다. 작별 흐름의 여포 소식은 뺐다.
- `ver5/assets/scenario/hero/explore13_farewell.json`, `ver5/assets/scenes/ch13.json`: 관청 장면과 같은 위치에 관우·장비·조운·간옹·손건을 두고 조운과 상호작용한다. 부대 메뉴에서 정비 가능하다. 조운 출발 시 칼 공격 동작은 제거했다.
- `ver5/internal/content/content.go`, `ver5/internal/session/session.go`, `ver5/internal/session/autosave_test.go`: 선택된 탐험 노드의 시나리오 자동저장과 복원 테스트를 추가했다. 탐험 중 이동과 불러오기는 자동저장을 중복 생성하지 않는다.
- `ver5/docs/fullstory.md`, `ver5/docs/data-speech-style.md`, `ver5/docs/system-save.md`, `ver5/docs/presentation-explore.md`: 대본과 저장·탐험 규칙을 동기화했다.
- 검증: `go test ./internal/session ./internal/content ./internal/scenes`, `go test ./internal/session -run TestAutosave -count=1`, 변경 파일 `git diff --check`, jargon 0줄 통과. `scene/`과 `capture/`에서 귀환 선언과 탐험·대화 화면을 확인했다. `_balance -stage B13 -seeds 1`: rush 0/1, 1라운드 (변경 전 비교 없음).
- 미완료: 이야기 색인은 Ollama 연결 거부로 갱신하지 못했다. srpg-cli 슬롯 9로 전투 전체 플레이는 실행하지 못했다. 전체 워크스페이스의 diff check는 기존 `internal/core/battle_events_test.go` 말미 공백 때문에 실패하여 해당 사용자 변경은 건드리지 않았다.
