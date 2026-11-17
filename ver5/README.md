# ver5

삼국지 SRPG. 엔진(`internal/`)에는 인물·병종·아이템·스킬·특성 이름이 하나도 없고, 게임 내용은 모두 `assets/data/`의 데이터 묶음이다.
스프라이트는 도트 에디터 YAML을 빌드한 `.spr`만 쓰며, 진영색·옷색·말색은 캐릭터 데이터가 정한다.

```sh
cd ver5
go run ./cmd/srpg-gui                      # debug GUI (1280×850)
go run ./cmd/srpg-cli                      # 사람용 CLI, -json 으로 JSON Lines
go test ./... && go vet ./...
python3 tools/spritebuild/build.py         # tools/doteditor/units/*.yaml → assets/sprites/*.spr
(cd tools/doteditor && python3 -m http.server)  # 도트 에디터: http://localhost:8000
make                                       # release 데스크톱 빌드
make android                               # release APK (자세한 빌드: docs/tools-build.md)
make android android-install ANDROID_MODE=debug  # Android에서 디버그 도구·자동실행 사용
```

문서는 [docs/README.md](docs/README.md)에서 시작한다. 정리 전 옛 문서는 `docs_old/`에 있다.

개발 실행은 debug가 기본이다(`-debug=false`로 도구 숨김). 새 게임을 시작한 뒤 왼쪽 위 진행판 맨 오른쪽의 **자동실행**을 누르면 이야기·구매·장비·승급·특성 학습·전투를 실제 자원으로 진행한다. **학습(시험)**이 기본이며 장비·특성·전투 명령을 학습된 신경망이 고른다. 승급·편성·상점·이야기는 기존 규칙을 쓴다. 현재 모델의 정상 새 게임 최고 기록은 11탄 무패(12탄 첫 패배)이며, 미사용 8시드 평균은 5.5탄이다([학습·평가 기록](docs/tools-autoplay.md#9-첫-패배까지의-특성장비전투-공동-학습)). 중지한 상태에서 이 버튼을 누르면 기존 **전투: 탐색**으로 전환한다. 모델은 아직 실험용이며 캠페인 완주를 보장하지 않는다. 다시 누르거나 Esc로 중지한다. 기본 Make 빌드는 `release` 태그를 사용하며 이 버튼이 없다. Android에서는 `ANDROID_MODE=debug`로 디버그 도구를 포함할 수 있다.
