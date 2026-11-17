# 3장 저택 대화 종료 오류음 수정

3장 하진 회상 뒤 저택의 마지막 대사에서 유비가 장비를 향해 `E`로 돈다. 화면은 8방향을 쓰지만 `internal/core/explore.go`의 `walk` 검사는 대각선 네 방향만 허용한다. 탐험 복귀 후 `reportSpot`이 화면의 방향을 저장하려다 `InvalidFace`와 `ui-denied`를 낸다. 기존 `explore` 데모의 정상 3장 경로에서도 4980프레임에 재현했다.

- `internal/core/explore.go`: 탐험 명령에서 8방향을 허용해 화면의 실제 방향을 위치와 함께 보존한다. 스테이지·전투 방향 및 원본 탐험 데이터의 배치 검사는 바꾸지 않는다.
- `internal/core/explore_test.go`: 8방향 위치 기록·저장 복원·이벤트 출구 통과와 잘못된 방향 거부를 검사한다.
- `internal/gui/explore_test.go`: 실제 3장 마지막 대사의 자동 회전, 암전, 탐험 복귀, 위치 기록까지 실행해 오류음과 오류 알림이 없고 방향이 보존되는지 검사한다.
- `docs/history.md`: 원인·수정 파일·검증을 기록한다.

관련 GUI·core 패키지 시험과 vet를 실행한다. 기존 `explore` 데모로 같은 탐험 경로를 다시 캡처하고 `SRPG_SFX_TRACE`에서 `ui-denied`가 사라졌는지 확인한다.
