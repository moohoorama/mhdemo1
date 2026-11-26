# 계교 전투 B안 적용 결과

- `ver5/assets/scenario/hero/stage10.json`: 선택한 시작 대사 B안과 전투 중 공통 분기 대사를 적용하고 시작 대화 순서에 맞춰 기존 방향 전환을 재배치했다. 안량 출현 대사는 “하북의 안량이 왔다! 장익덕은 어디 있느냐!”로 고쳐 거록에서 퇴각했다는 암시를 제거했다. 전투 조건·효과는 유지했다.
- `ver5/docs/fullstory.md`, `ver5/docs/data-speech-style.md`: 시작 대사와 인물 말투 예시를 동기화했다.
- `ver5/docs/history.md`: 변경 내용과 이유를 기록했다. 구현 설계는 `design.md`에 있다.
- 검증: JSON 조건·대사 확인, `git diff --check`, `go test ./internal/content`, 계교 일기토 선별 테스트(상위 테스트 통과이나 해당 일기토는 전투가 먼저 끝나 SKIP), `_facecheck B10`, jargon 검사 0줄. B10 rush 측정은 0/5 승리·평균 3.0라운드로 이전 측정과 같다.
- 미완료: `_uidemo-vm`은 GLFW가 모니터를 찾지 못해 캡처하지 못했다. 이야기 색인 갱신은 Ollama 연결 거부로 실패했다. `srpg-cli -slot 9` 전체 전투 플레이도 수행하지 못했다.
- 기존 감시 프로세스의 중복 수신을 피하기 위해 요청 기록은 process/에 직접 남겼다.
