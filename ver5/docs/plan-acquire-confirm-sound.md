# 획득 확인 효과음 수정

획득 창의 확인이 공통 `closeModal`을 호출하면서 취소음 원본인 `close`를 재생한다. 첫 전투 뒤 대장간의 청룡언월도·장팔사모처럼 연이어 받는 보상에서도 같은 문제가 생긴다.

- `internal/gui/windows.go`: 획득 카드 전체를 확인하면 `ok`, 카드 위의 장비·특성 설명을 닫으면 기존 `close`를 재생한다. 획득 이외 모달의 소리는 유지한다.
- `internal/gui/acquire_test.go`: 실제 대장간 노드를 진행하며 두 무기의 지급, 확인 효과음, 중첩 설명 창의 닫기 효과음을 검사한다.
- `internal/gui/uidemo_acquire.go`: 실제 대장간 이야기와 획득 창을 진행하는 `smith-gifts` 데모를 추가한다. 이야기·장면 데이터는 수정하지 않는다.
- `docs/history.md`: 수정과 검증 결과를 기록한다.

검증은 GUI 패키지 시험과 vet, `smith-gifts` 게임 캡처와 `SRPG_SFX_TRACE`의 효과음 순서를 사용한다.
