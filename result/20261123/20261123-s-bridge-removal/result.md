# 미사용 장면맵 S_bridge 삭제 결과

## 한 일

- 삭제: `ver5/assets/graphics/maps/S_bridge.json`(git 추적 파일, 복구 가능).
- 등록 제거: `ver5/assets/graphics/index.json`의 `"maps/S_bridge.json"` 한 줄만 지웠다. 이 파일에는 다른 미커밋 변경이 많아서 그 줄만 바꿨다.
- 문서 정리: `docs/plan-small-tile-maps.md`와 `docs/presentation-scene.md`에서 S_bridge를 사용 가능한 맵처럼 소개하던 행을 지웠다.
- 이력: `docs/history.md`에 삭제 항목을 추가했다. 옛 결정·이력(`plan/_decision_22_27.md`, `docs/history.md`의 과거 항목)은 그대로 두었다.

## 확인

- 장면·시나리오·Go 코드·테스트에서 `S_bridge` 참조가 없음을 확인했다(`grep`).
- `index.json`은 JSON으로 정상 파싱되고 맵 82개, `S_bridge` 없음.
- `go test -p 1 -count=1 ./internal/scenemap/` 통과.
- `internal/gui` 테스트(`TestSceneMapsLoadAndDraw`)는 실행하지 못했다. 다른 작업 중인 `internal/gui/sortie_unified.go`에 빌드 오류가 있다(`slices` 미정의, 상수 변환 오류). 내 변경과 무관하다.

## 남긴 것 (의도적으로 건드리지 않음)

- `tools/scene_maps/baihe_dike.py` 7행 주석: 옛 `S_bridge` 크기를 언급하는 설명. 동작과 무관.
- `output/` 아래 장 대본 문서의 `맵 S_bridge` 표기: 생성된 산출물. 현재 장면 참조가 없으므로 재생성 때 정리되는 항목이다.
- `plan/_decision_22_27.md`: 과거 결정 기록.

## 한계

- 게임 화면(GUI)에서 장면 로딩을 직접 확인하지 않았다. 빌드 오류 때문에 GUI 테스트를 돌리지 못했다.
- `index.json`은 원래 `tools/build_assets.py`가 만든다. 이번에는 한 줄만 손으로 지웠다. 다음에 생성 도구를 돌리면 결과가 같은지 확인이 필요하다.
