# 미사용 장면맵 S_bridge 삭제

현재 사용하지 않는 `S_bridge` 장면맵을 **실제로 삭제**하고 관련 등록과 재생성 경로를 정리해 주세요.

- `ver5/assets/graphics/maps/S_bridge.json`과 `ver5/assets/graphics/index.json`의 등록을 확인한다.
- `ver5/assets/scenes/`, 시나리오, 코드와 생성 도구의 사용·생성 관계를 다시 확인하고, S_bridge 전용 항목만 제거한다. 공유 다리 타일, 다른 맵, 다른 미커밋 변경은 보존한다.
- 현재 문서에서 S_bridge가 사용 가능한 맵처럼 소개되는 내용은 맞게 정리한다. 과거 결정·이력은 역사 기록으로서 보존한다.
- 관련 테스트 및 콘텐츠 로딩을 확인하고 변경·검증·한계를 `result/YYYYMMDD/<task-name>/`에 기록한다. 프로젝트 지침과 `ver5/docs/history.md` 기록 규칙을 따른다.
