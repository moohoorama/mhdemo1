# 맵·에셋·렌더링 아키텍처

이 문서는 현재 구현을 설명한다. 개발 시 에셋을 가공하는 도구와 게임 실행 경로를 분리하고, 화면과 PNG 출력은 같은 `DrawItem` 목록을 사용한다.

## 1. 전체 구조

```text
개발할 때 실행                         게임을 실행할 때
==============                         ==================

Makefile                                main.go: newGame()
  |                                            |
  +--> tools/tilegenerator                     v
  |      +-- 지형 픽셀 생성             graphics.Load()
  |                                            |
  +--> tools/spritetool                         +-- 시트 PNG 디코딩
         +-- 원본 프레임 분리                   +-- catalog.json 검증
         +-- crop / resize / outline            +-- GPU 시트 업로드
         +-- 고정 셀 atlas 조립                 |
  |                                            v
  v                                     graphics.Catalog
assets/                                        |
  +-- terrain.png  ----------------------------+
  +-- objects.png  ----------------------------+
  +-- catalog.json ----------------------------+

맵 JSON ---> terrain.Load() ---> World
                                  |
                          SceneCache.Ensure()
                                  |
                     맵이 바뀌었을 때만 계산
                                  |
                          BuildMapScene()
                                  |
                                  v
                               MapScene
                                  |
       Catalog + 애니메이션 시각 --+
                                  |
                           BuildDrawList()
                                  |
                                  v
                             []DrawItem
                                  |
                    +-------------+-------------+
                    |                           |
                    v                           v
            screenRenderer.Draw()        graphics.RenderPNG()
                    |                           |
             카메라 이동 / 확대            출력 범위 / 확대
                    |                           |
                    v                           v
                게임 화면                  map-export.png
                    |
              drawEditorUI()
```

Makefile이 프로젝트별 입력 순서, 목표 크기와 atlas 위치를 소유한다. `spritetool`은 이미지가 나무인지 잔디인지 알지 못하며, 애니메이션의 모양도 생성하지 않는다. 애니메이션 원본은 `tools/spritetool/assets/`에 완성된 가로 프레임으로 둔다.

```sh
# 두 런타임 시트 생성
make assets

# 개별 생성기 실행
go run ./tools/tilegenerator assets/terrain.png
go run ./tools/spritetool --size 8x6 --outline 1 input.png output.png
```

`catalog.json`은 게임 고유의 셀 번호·피벗·애니메이션 계약이므로 직접 관리한다. 두 범용/전용 이미지 도구는 맵 JSON이나 카탈로그를 수정하지 않는다.

## 2. 자료구조

```text
game (main.go)
 |
 +-- w: *terrain.World                   저장할 맵 상태
 |    +-- Version = 5
 |    +-- Width, Height
 |    +-- cells: []Cell                   비공개, 행 우선
 |    |    +-- Ground: Kind              River / Grass / Wasteland
 |    |    +-- Decoration                NoDecoration / Rocks / Trees
 |    +-- revision: uint64               실제 셀 변경 때 증가
 |
 +-- sceneCache: terrain.SceneCache      맵 해석 결과를 재사용
 |    +-- world: *World                  이전에 계산한 맵의 정체성
 |    +-- revision                       이전에 계산한 변경 번호
 |    +-- scene: *MapScene
 |         +-- Ground: []GroundTile
 |         |    +-- Position            투영된 서브타일 위 꼭짓점
 |         |    +-- Layers: []int       물/초원/황무지 이미지 참조
 |         +-- Decorations: []DecorationPlacement
 |         |    +-- Position            투영된 발밑 위치
 |         |    +-- Object              오브젝트 종류
 |         |    +-- Phase               좌표로 고정된 애니메이션 위상
 |         +-- Bounds                   PNG 출력 범위, 16px 여백 포함
 |
 +-- catalog: *graphics.Catalog          게임용 에셋
 |    +-- Manifest
 |    |    +-- Version = 1
 |    |    +-- Sheets[]                  파일명, 셀 크기, 열/행 수
 |    |    +-- Sprites[]                 시트 번호, 셀 번호, Pivot
 |    |    +-- Animations[name]          프레임 ID, FrameTicks, Loop
 |    +-- Images[]                      CPU 이미지: PNG 출력용
 |
 +-- renderer: *screenRenderer
 |    +-- catalog                       위 카탈로그 참조
 |    +-- sprites[]                     업로드한 시트의 GPU 셀 뷰
 |
 +-- camera: Camera                     Zoom, PanX, PanY
 +-- waterFrame / treeTick              물 / 식물의 현재 재생 시각
 +-- drawList: []graphics.DrawItem       재사용하는 출력 목록 버퍼
 |    +-- Sprite                        카탈로그의 이미지 ID
 |    +-- Position                      투영된 기준점
 |    +-- Layer                         0=바닥, 1=장식·이동 스프라이트
 |    +-- Depth                         발밑의 화면 Y좌표
 |
 +-- undo / redo / stroke               World 스냅샷을 사용하는 편집 이력
```

맵의 셀 배열은 외부에서 직접 바꿀 수 없다. `SetCell()` 또는 브러시용 `SetTerrain()`을 사용해야 변경 번호와 캐시가 함께 동작한다. `Cell()`은 값 복사본을 반환한다. 크기는 `New()` 또는 `Load()`에서 정하며, 실행 중 `Width`·`Height`를 직접 수정하지 않는다.

`Kind`의 기존 6종은 편집기 브러시 프리셋으로 유지된다. 예를 들어 `RockGrass`는 `Cell{Ground: Grass, Decoration: Rocks}`로 변환된다. 경계 계산은 오직 `Ground`, 장식 규칙은 `Decoration`과 바닥을 참조한다. `NoDecoration`은 별도 바위/숲 프리셋이 없다는 의미이며, 초원에서는 기존대로 확률적으로 잔디를 배치한다.

## 3. 함수 호출 흐름

```text
main()
 |
 +--> newGame()
 |     |
 |     +--> graphics.Load(embedded runtime assets)
 |     |     +--> Manifest.Validate()
 |     |     +--> PNG 크기와 셀 격자 일치 검사
 |     |
 |     +--> terrain.ValidateAssets()
 |     |     +--> 지형 38종 / 물 8프레임 / 장식 애니메이션 검사
 |     |
 |     +--> newScreenRenderer(catalog)
 |     |     +--> 시트마다 NewImageFromImage() 한 번
 |     |     +--> 각 셀의 SubImage() 뷰 보관
 |     |
 |     +--> terrain.Load(path), 실패 시 샘플 맵
 |     +--> fit()
 |     +--> sceneCache.Ensure(w)
 |           +--> BuildMapScene(w)
 |                 +--> MasksFor() / Masks() / SurfaceLayers()
 |                 +--> Placements() / Position() / 위상 계산
 |
 +--> ebiten.RunGame(g)
       |
       +--> Update()
       |     +--> animate() / advanceSway()
       |     +--> 입력 및 action() 처리
       |     |     +--> paint() -> SetTerrain() -> SetCell()
       |     |     |                 +--> 실제 변경 시 revision++
       |     |     +--> undo/redo/load/demo -> w 교체
       |     |     +--> save -> World.Save()
       |     |     +--> export -> exportImage(4) -> writePNG()
       |     +--> 카메라 이동 / 확대
       |
       +--> Draw(screen)
             +--> buildDrawList()
             |     +--> sceneCache.Ensure(w)
             |     |     +--> 같은 포인터 + 같은 revision: 캐시 반환
             |     |     +--> 변경/교체됨: BuildMapScene() 후 반환
             |     +--> MapScene.BuildDrawList(catalog, 시간, extra, 버퍼)
             |           +--> 물 프레임 선택
             |           +--> 장식 프레임 선택: 현재 tick + 고정 위상
             |           +--> extra가 있으면 공통 깊이 정렬
             +--> screenRenderer.Draw(screen, 목록, camera)
             +--> drawEditorUI(screen)

exportImage(scale)
 +--> buildDrawList()                  화면과 같은 장면 생성 함수
 +--> graphics.RenderPNG(catalog, 목록, SceneBounds(catalog, 목록, MapScene.Bounds), scale)
       +--> image/draw로 셀 합성
       +--> 정수배 최근접 확대
```

캐시는 변경 직후 즉시 매번 만드는 대신 **다음 Draw 또는 export에서 한 번** 갱신한다. 한 번의 드래그 갱신에서 여러 셀을 바꿔도 그 사이에 렌더 요청이 없다면 한 번만 재계산한다. 카메라와 애니메이션 시각이 바뀌어도 캐시는 유지된다. 저장 필요 표시 `dirty`와 캐시 변경 번호 `revision`은 별개의 개념이다.

## 4. 시트와 좌표 규약

| 시트 | 셀 | 격자 | 전체 크기 | 사용 셀 | Pivot |
|---|---|---|---|---|---|
| terrain.png | 16×8 | 8열×5행 | 128×40 | 38 / 40 | (8, 0) |
| objects.png | 24×32 | 8열×4행 | 192×128 | 28 / 32 | (12, 28) |

지형 ID 0–37은 기존 타일 순서 그대로다. 장식 ID는 38부터 시작한다. 바위 4프레임(각 종류 1장), 나무 8프레임(2종×4), 잔디 16프레임(4종×4)을 포함한다. 장식은 원래 그림의 크기를 유지하고 투명 여백으로 공통 셀 크기와 피벗을 맞춘다. 숲 장식은 서브타일 4칸에 분산하지 않고 큰 타일 중앙에 하나만 배치한다.

```text
카탈로그의 셀 선택:
  column = sprite.Cell % sheet.Columns
  row    = sprite.Cell / sheet.Columns
  source = (column * CellWidth, row * CellHeight, CellWidth, CellHeight)

맵에서 장면 좌표로:
  x = (u - v) * 16
  y = (u + v) * 8

화면에서 그릴 왼쪽 위:
  x = Camera.PanX + (Position.X - Pivot.X) * Camera.Zoom
  y = Camera.PanY + (Position.Y - Pivot.Y) * Camera.Zoom

PNG에서 그릴 왼쪽 위 (확대 전):
  point = Position - Pivot - exportBounds.Min
```

피벗은 이미지 높이에서 추정하지 않고 카탈로그에 명시한다. 장식의 발밑 위치는 유지한다. 나무는 기존 4프레임 `tree_0.png`, `tree_1.png`를 사용하며 게임 해상도는 큰 나무 22×20, 작은 나무 20×18이다. PNG 출력은 `graphics.SceneBounds()`로 나무가 잘리지 않도록 범위를 확장한다. 원본 PNG의 프레임 분리·축소·외곽선은 Makefile이 `spritetool` 옵션으로 명시한다.

시간의 기본 단위는 125ms다. 물은 1 tick마다 8프레임을 순환한다. 나무·잔디는 기본 16 ticks마다 4프레임을 순환하며 좌표별 위상이 있다. 편집기의 식물 속도 기본값은 2배이고 1/2/4/8배를 선택할 수 있다. 카탈로그의 `Animation.Frame()`은 반복과 비반복의 마지막 프레임 유지 모두 지원한다.

## 5. 저장 및 호환성

현재 저장은 버전 5의 `cells` 배열이다. 지형·장식 배치 캐시, 이미지 번호, 애니메이션 시각, revision은 저장하지 않는다. 로드는 버전 1–4를 현재 `Cell` 구조로 변환하며, 원본 파일은 명시적으로 저장할 때만 버전 5로 바뀐다. `example-map.json`은 이전 버전 호환성을 보여주는 버전 4 샘플로 보존했다.

```json
{
  "version": 5,
  "width": 2,
  "height": 1,
  "cells": [
    { "ground": 1, "decoration": 1 },
    { "ground": 2, "decoration": 2 }
  ]
}
```

위 예는 바위 초원과 황무지 위 숲이다. 물 위 바위/숲은 허용하지 않는다. 새 조합을 저장할 때 기존의 합쳐진 `Kind` 값으로 역변환하지 않으므로 정보가 손실되지 않는다. UI의 기존 6개 브러시는 그대로 유지했다.

## 6. 코드 경계와 확장 지점

```text
Makefile ---> tools/tilegenerator ---> internal/terrain
        \\--> tools/spritetool

assets ---> main.go ---> internal/graphics <--- internal/terrain
                         ^
render.go / editor_render.go
```

| 파일/패키지 | 책임 |
|---|---|
| Makefile | 프로젝트의 입력 순서, 크기, 프레임, atlas 배치 레시피 |
| tools/tilegenerator | 프로젝트 전용 지형 이미지 생성 |
| tools/spritetool | 의미를 모르는 범용 PNG 프레임 가공과 atlas 조립 |
| tools/spritetool/assets | 완성된 애니메이션 프레임을 포함한 원본 이미지 |
| assets | 실행 파일에 포함되는 최종 시트와 카탈로그 |
| internal/terrain/world.go | 셀 모델, 브러시 변환, 좌표 투영, 경계 계산, 샘플 맵 |
| internal/terrain/world_io.go | 버전 변환, 검증, 저장·불러오기 |
| internal/terrain/objects.go | 장식 확률, 위치, 깊이 순서, 위상 규칙 |
| internal/terrain/scene.go | 맵 캐시, 공통 그릴 목록, 지형 에셋 계약 검증 |
| internal/graphics/catalog.go | 범용 시트 메타데이터, 검증, 로딩, 프레임 선택 |
| internal/graphics/render.go | 그릴 항목, 공통 정렬, CPU PNG 합성 |
| main.go | 편집기 상태, 입력, 게임 루프 |
| render.go | 카메라, GPU 시트 뷰, 화면 렌더, PNG 출력 연결 |
| editor_render.go | 격자, 브러시, 패널 등 편집기 표시 |

유닛의 이동·전투 상태와 유닛 이미지 생성은 이번 리팩터링에서 구현하지 않았다. `BuildDrawList()`의 `extra` 인자로 유닛의 `DrawItem`을 전달하면 장식과 함께 깊이 정렬할 수 있다. 기본 맵 렌더링에서는 이미 정렬된 장식 캐시를 재사용하고 매 프레임 다시 정렬하지 않는다.

향후 병종별 40셀 PNG는 같은 `Sheet`/`Sprite` 구조로 등록하고, `Animations`에 방향·동작별 이름과 프레임 ID를 지정할 수 있다. 공통 로더/렌더러를 재사용하되, 방향·동작 상태 선택과 공격/피격 동기화는 별도의 유닛 로직에서 연결해야 한다. 이미지 파일만 추가해 자동으로 전투 기능이 생기는 구조는 아니다.

## 7. 검증

```sh
go test ./...
go vet ./...
go build .
DEMO1_WINDOW_TEST=1 go test -count=1
```

- `spritetool`의 공유 crop, 최근접 축소, 외곽선과 `tilegenerator`의 지형 경계를 각각 도구 옆에서 검사한다.
- 체크인한 런타임 시트·카탈로그를 로드해 크기, 셀 수와 애니메이션 계약을 검사한다.
- 맵 변경·무변경·교체의 캐시 재사용/갱신, 셀 분리와 구버전 변환, 잘못된 카탈로그, 애니메이션 시간, 이동 스프라이트와 장식의 깊이 정렬을 검사한다.
- 기존 경계 조합·확률·편집 이력·식물 애니메이션 검증을 유지한다.
- 실제 창 검증은 GPU 화면에서 물 8프레임, 식물 4프레임, 고정 지형을 확인한다.
