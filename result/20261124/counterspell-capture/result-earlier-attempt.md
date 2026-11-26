# 반계 화면 재현 진행 상황

반계 확인용 `ver5/internal/gui/uidemo_counterspell.go`를 추가했다. 원래 전투 엔진에 화계 공격 및 반계 보유 적을 준비하고 원래 GUI 재생 경로를 호출하는 데모다. 테스트는 데모에서 명중률을 100으로, 책략 피해와 비용을 낮추어 재생이 안정되도록 조정한다.

`caffeinate -d -i go run ./cmd/_uidemo-vm -ui counterspell -frames 260 -every 20 -out ../result/20261124/counterspell-capture -mp4 ../result/20261124/counterspell-capture/counterspell.mp4` 실행은 Mac 잠금으로 GLFW `no monitor was found at initializeGLFW` 오류가 발생하여 캡처하지 못했다. GUI 패키지 테스트는 통과했다.

코드 확인: `internal/gui/player.go`는 모든 반격 재생에 `core.Command{Kind: "attack"}`을 전달하여 반계도 일반 공격 연출로 처리한다. `internal/core/combat.go`는 받은 책략으로 반계 피해를 계산한다. 화면에서 확인되기 전까지 시각 결과는 미검증이다. 잠금 해제 후 위 명령을 다시 실행해야 한다.
