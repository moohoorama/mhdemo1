# S_bridge 사용 장면 확인 결과

## 결론

현재 게임 장면(`assets/scenes/`, `assets/scenario/`)에서 `S_bridge`를 쓰는 곳은 없다. 맵 파일과 그래픽 등록, 기획·문서에만 남아 있다.

## 근거

- 참조 보고서 `result/20261123/s-bridge-usage/summary.md`: 장면과 시나리오에 현재 참조가 없고, 25장 백하 둑 장면은 `S_baihe_dike`를 쓴다고 적혀 있다.
- 직접 `grep -rl "S_bridge"`로 확인했다.
  - `assets/scenes`, `assets/scenario`: 일치 없음.
  - 그 밖의 참조:
    - 도구: `tools/scene_maps/baihe_dike.py`
    - 그래픽: `assets/graphics/index.json`, `assets/graphics/maps/S_bridge.json`
    - 기획·문서: `plan/_decision_22_27.md`, `output/chapters/ch25.md`, `output/chapters/ch27.md`, `output/chapters/ch25v.md`, `output/scenario.md`, `docs/history.md`, `docs/presentation-scene.md`, `docs/plan-small-tile-maps.md`

## 참고

`tools/scene_maps/baihe_dike.py`가 `S_bridge`를 언급하므로, 장면 생성 도구 쪽에는 아직 남아 있다. 게임 데이터에서 실제로 불러 쓰는지는 이번 확인에서 보지 않았다.
