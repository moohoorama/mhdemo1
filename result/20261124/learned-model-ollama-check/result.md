# 학습 자동실행 및 Ollama 확인

- `ver5/assets/autoplay/battle.json`은 존재하며 크기 약 3.3 MiB, SHA-256 `076d447362168b0c46fd9cbda75585d55123f461cbceaa7a434eec59d06adc8b`이다. B01·B07·B09·B10·B12·B13·B14·B17·B18·B19 전투별 모델을 포함한다. 학습 가중치는 남아 있다.
- 실제 실행 `go run ./cmd/_autoplay run -player neural -battle-model assets/autoplay/battle.json -seeds 310004 -tries 1 -stages 1 -out /tmp/mhdemo-model-check.json`은 `battle model content hash differs; retrain against current game data`로 실패했다. GUI도 `campaign_auto.go`에서 같은 콘텐츠 해시 불일치를 거부한다. 이는 학습 데이터 소실이 아니라 학습 당시 모델과 현재 게임 데이터의 불일치다.
- `reports/autoplay/goal10/model.json` 및 이전 배포 모델도 남아 있다. goal10 문서에 있는 모델 해시는 현재 배포 모델 해시와 다르므로 해당 문서의 옛 모델을 현재 모델과 혼동하면 안 된다.
- 모델 가중치·해시를 변경하지 않았다. 해시만 덮어쓰는 것은 바뀐 데이터에서 모델의 적합성을 검증하지 못하므로 하지 않았다. 재학습 또는 명시적인 전이와 현재 데이터 검증이 필요하다. 일시적으로 GUI에서 중지 중 `학습(시험)`을 클릭해 `전투: 탐색`으로 바꾸면 규칙 기반 자동실행을 사용할 수 있다.
- Ollama 실행 파일이 설치되어 있으나 서버는 꺼져 있었다. `ollama serve`로 로컬 서버를 시작했고, 설치된 모델은 없었다. storydb에 필요한 `bge-m3`를 설치하고 색인을 갱신했다(1,059개 청크 중 305개 재생성, 754개 재사용, 79개 삭제). 소패 조운 작별 의미 검색이 현재 대본을 반환하는 것도 확인했다.
