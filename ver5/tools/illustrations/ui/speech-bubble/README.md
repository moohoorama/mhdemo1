# 흰색 10조각 말풍선

`build.py`가 정확한 치수로 도형을 그린 원본이며 런타임 PNG 10개를 생성한다. 수묵 초상과 함께 쓰는 UI로 흰 바탕·먹색 윤곽을 사용한다. 초상 생성이나 유닛 스프라이트 파이프라인과 분리한다.

```sh
python3 tools/illustrations/ui/speech-bubble/build.py
go run ./cmd/srpg-gui -duel B01-duel-1 -duel-capture reports/speech-bubble
```

모서리 16×16, 상하 변 1×16, 좌우 변 16×1, 중앙 1×1, 꼬리 24×20이다. 외곽선은 2px이며 모서리만 4배 해상도에서 그린 뒤 축소해 부드럽게 만든다. 직선과 중앙은 단색이라 길이만 늘려도 두께와 색이 유지된다. 꼬리의 첫 2px가 하단 변을 덮어 연결부의 가로 외곽선을 지운다. 꼬리와 모서리는 늘리지 않는다.
