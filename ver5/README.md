# ver5

삼국지 SRPG. 엔진(`internal/`)에는 인물·병종·아이템·스킬·특성 이름이 하나도 없고, 게임 내용은 모두 `assets/data/`의 데이터 묶음이다.
스프라이트는 도트 에디터 YAML을 빌드한 `.spr`만 쓰며, 진영색·옷색·말색은 캐릭터 데이터가 정한다.

```sh
cd ver5
go run ./cmd/srpg-gui                      # 그래픽 (1280×850)
go run ./cmd/srpg-cli                      # 사람용 CLI, -json 으로 JSON Lines
go test ./... && go vet ./...
python3 tools/spritebuild/build.py         # tools/doteditor/units/*.yaml → assets/sprites/*.spr
(cd tools/doteditor && python3 -m http.server)  # 도트 에디터: http://localhost:8000
make android                               # APK (자세한 빌드: docs/tools-build.md)
```

문서는 [docs/README.md](docs/README.md)에서 시작한다. 정리 전 옛 문서는 `docs_old/`에 있다.
