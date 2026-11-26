# 수공 물·판자·파편 해상도 2배 (2026-11-23)

웹 보고: https://claude.ai/artifact/UcjPuaNCdix7Xo9XvEPfjy · 파일: `report/index.html`
영상: `new/suigong_main_battle.mp4` (효과음 포함, 960x638 60fps 19.2초) · 비교 캡처: `zoom_*.png`, `frames/`

## 한 일
- 병사 쓸림 연출: 물이 닿은 병사가 강 아래로 2.2~3.4칸 떠내려가며 0.7초에 걸쳐 투명해짐(`field_surge_show.go` `suigongWash`, `unitVis.fade`).
- 텍스처: 쌓인 둑과 날아가는 파편에 판자 나뭇결·흙 알갱이(`texture_cube`, `Dam.grain`). 흙 색은 맵 절벽 타일의 모래색(윗면 187,158,106 / 옆면 152,121,75, 120,93,57). 팔레트 17→29색(+스프레이 8).
- `ver5/tools/floodsim.py`: `FLOOD_RES`(기본 2). 물 격자 `DX` 2→1, 둑 복셀 `VS` 4→2(몸 하나는 `VS0=4` 그대로, 복셀 8개로 나뉘어 그려짐, 물리·난수 불변). 높이·폴리곤·거품 무늬는 도트 크기로 나눠 같은 모양 유지. `FLOOD_OUT`으로 출력 폴더 지정. 굽기 진행 로그 추가.
- 저장 형식 변경: 옛 시트는 4096x17048(RGBA 279MB)이라 ebiten이 거부(`atlas ... too big`). 프레임마다 팔레트 번호를 deflate로 눌러 `suigong.idx`에 담고(스프레이 124프레임도 같은 방식), 게임 `internal/gui/field_surge.go`가 보이는 프레임의 조각만 그림으로 만들고 지난 프레임은 `Dispose`. `suigong.png`는 없어짐.
- 문서: `docs/tools-build.md`, `docs/history.md`. 측정 시험: `internal/gui/measure_render_test.go`의 `TestMeasureSurge` (`SRPG_RENDER_TEST=1`).

## 수치
| | RES=1 | RES=2 |
|---|---|---|
| 굽는 시간 / 최대 메모리 | 225초 / 800MB | 3,063초(텍스처 포함) / 820MB |
| 점 합계 | 14.3M | 57.9M (4.04배) |
| 옛 방식 시트 PNG | 4096x6459, 2.8MB | 4096x17048, 7.1MB (게임 거부) |
| 새 방식 `suigong.idx` | 1.7MB | 4.4MB |
| 게임 재생 | — | GPU 최대 +16MB, 프레임 그림 만들기+그리기 중앙 0.84~1.4ms, p90 1.7~2.5ms, 최대 5.8~37ms (조용한 실행; 부하 높을 때 중앙 10ms, 최대 221ms) |

## 검증
- `go build ./...`, `go vet ./internal/gui`, gui `-run 'Suigong|Surge|Cutscene|Ch25'`, `./internal/content`, `./internal/cli` 통과. gofmt는 바꾼 파일 통과.
- 새 `bake()`를 `RES=1`로 다시 굽고(241초) 변환본과 비교: 시간표·밴드·스프레이·깊이·팔레트 같음, 점 2,503만 개 중 0.047% 차이(흙·판자 끝결 알갱이 난수). 이 검증에서 저장 단계 int8 오버플로 버그를 찾아 고침.
- 실제 GUI(`-ui suigong-battle`)로 본편 흐름 재생·캡처. 오디오 AAC 평균 -28.1dB / 최대 -5.4dB (이전 -28.5 / -5.5).

## 한계
- 소리를 귀로 듣지 못했다. 폰 실기 측정 못 함.
- 물 계산 격자가 바뀌어 파도·판자 개별 위치는 옛 영상과 정확히 같지 않다(승인된 타이밍은 같다).
- 병사·말 그림과 스프레이(이미 1px)는 그대로.
- GPU 값은 ebiten이 세는 이미지 메모리이고 아틀라스 페이지 단위로 오른다. 이 맥 한 대의 값.
- `new/main_silent.mp4`(17.5MB)는 소리 입히기 전 원본이라 지워도 된다.
- 임시로 쓴 변환 스크립트 `tools/_flood_repack.py`는 지웠다. 옛 `RES=1` 데이터 백업은 임시 폴더에만 있다.
