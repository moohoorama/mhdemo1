# 지형 회복 안내와 상태 표시 수정

- 성내의 RestHP=0, RestMP=2 규칙은 유지한다. 지형 설명에서 병력 회복과 병법치 회복 배율을 따로 표시한다.
- 상태 표시는 `internal/gui/status_visual.go`의 벡터 도형이며 별도 이미지 자산이 없다.
- Field.Canvas는 지도 좌표로 그린 뒤 화면 배율을 적용한다. 상태 표시의 선에도 화면용 strokeLine을 쓰면 배율이 중복 적용된다.
  Canvas 안에서는 vector.StrokeLine을 직접 사용해 선·바탕·원에 동일한 지도 배율만 적용한다.
- 성내 회복 안내와 실제 턴 회복, 화면 배율 1/2에서 아이콘 픽셀이 같은지 검증한다. 실제 렌더링 캡처도 확인한다.

검증 완료:
- `go test ./internal/gui ./internal/core`, `go vet ./internal/gui ./internal/core` 통과.
- `SRPG_RENDER_TEST=1 go test ./internal/gui -run TestRenderedEditorAndStatuses -count=1 -v` 통과.
- 1배/2배 화면 배율에서 아이콘의 Canvas 픽셀 일치 및 색상 선 표시를 확인했다.
- `reports/editor-regression/status-icons.png`, `statuses-hidpi.png`에서 강화·약화 아이콘과 맵 위 상태 표시를 육안 확인했다. 이 파일은 검증용 캡처이며 런타임 자산이 아니다.
