# 맵 에디터 · 스프라이트 이미지 모음

이 폴더는 저장소의 이미지 파일을 용도별로 모은 열람·전달용 스냅샷이다. 원본을 이동하거나 수정하지 않았다. 실행 시에는 기존 `assets/`, 제작 시에는 `tools/spritetool/assets/`를 계속 사용한다. 이후 원본 변경이 여기에 자동 반영되지는 않는다.

## 먼저 볼 이미지

- [맵 지형](current/terrain.png): 물·초원·황무지와 자동 연결 경계.
- [맵 장식](current/objects.png): 바위·잔디·나무.
- [현재 기사 v6](current/knights.png): 8방향, 대기·걷기·공격·피격·탈진, 각 4프레임.
- [기사 매핑](current/knights.json), [맵 시트 카탈로그](current/catalog.json): 셀 좌표·기준점·애니메이션 설정.

기사 방향 약어: N 북, NE 북동, E 동, SE 남동, S 남, SW 남서, W 서, NW 북서. 현재 기사 개별 방향 시트는 `sources/knights-pixel-v6/`에 있다. v7은 OpenCV 실험 제안이며 현재 적용본이 아니다. 과거 시트는 각 버전의 설정을 사용하며 현재 셀 크기를 일괄 적용하지 않는다.

지형 시트의 셀 순서는 물 8장 → 황무지 바닥 1장·경계 14장 → 초원 바닥 1장·경계 14장이다. 장식 시트는 셀 0–3 바위, 4–7 잔디 첫 프레임, 8–10 나무, 11–22 잔디 나머지 프레임이다. 남은 셀은 빈칸이다.

## 폴더 구분

| 폴더 | 내용 |
|---|---|
| `current/` | 현재 맵·기사에서 사용하는 최종 아틀라스와 JSON |
| `sources/` | 투명 원본, 방향별 기사 v2–v7, 피격 후보, 바위·잔디·나무 원본과 설정 |
| `history/runtime/` | 이전 기사 런타임 아틀라스와 매핑 |
| `reviews/` | 비교 이미지·검수 GIF |
| `legacy/` | temp에 있던 초기 렌더링·WebP·웹 미리보기 이미지 |
| `references/` | 루트의 보병·나무 참고 이미지 |
| `variants/` | ver1 삼국지 이미지와 GUI 캡처, ver2 후보 A/B/C/C2/D 원화·시트·화면 목업 |

같은 바이트의 이미지도 출처가 다르면 각각 보존했다. `manifest.json`에는 모든 이미지의 원래 경로, 크기, 설명, SHA-256을 기록했다. 복사한 README는 당시 제작 기록이며 그 안의 프로젝트 상대 경로나 실행 명령은 원래 프로젝트 위치 기준이다. 아래 목록의 이미지 링크는 이 폴더 기준으로 열 수 있다.

## 맵 에디터 실행

이 폴더는 이미지 자료 모음이며 에디터 실행 코드는 상위 저장소에 있다.

```sh
cd ~/orca/mhdemo1
go run . -map example-map.json
```

1–6으로 강·황무지·초원·바위·나무 브러시 선택, 왼쪽 드래그로 칠하기, 휠로 확대·축소, Space+드래그로 이동한다. S 저장, L 불러오기, Z/Y 되돌리기·다시 실행, P PNG 내보내기. 종료 시 자동 저장하지 않는다. 기사는 이 에디터의 배치 기능에 아직 연결되지 않았다.

기사 타일 배치·애니메이션 미리보기:

```sh
cd ~/orca/mhdemo1
make preview-knights
# http://127.0.0.1:8765/knight-preview.html
```

## 전체 이미지 목록

크기는 실제 파일의 가로×세로 픽셀이다. GIF 등 여러 장이 들어 있는 파일은 내부 프레임 수도 표시한다. 원래 경로는 저장소 루트 기준이다.

총 **222개 이미지**.

### current

| 이미지 | 크기 / 내부 프레임 | 설명 | 원래 경로 |
|---|---|---|---|
| [current/knights.png](current/knights.png) | 768×480 | 현재 기사 v6: 8방향×5동작×4프레임. 셀 48×48, 16열×10행, 피벗 (24,38). | `assets/knights.png` |
| [current/objects.png](current/objects.png) | 320×144 | 현재 맵 장식: 바위 4종, 잔디 4종×4프레임, 나무 3종. 셀 40×48, 8열×3행, 피벗 (20,44). | `assets/objects.png` |
| [current/terrain.png](current/terrain.png) | 128×40 | 현재 맵 지형 아틀라스: 물 8프레임, 황무지·초원 바닥과 경계. 셀 16×8, 8열×5행. | `assets/terrain.png` |

### sources

| 이미지 | 크기 / 내부 프레임 | 설명 | 원래 경로 |
|---|---|---|---|
| [sources/E-transparent.png](sources/E-transparent.png) | 1121×1403 | 기사 동쪽 방향 배경 투명화 원본 시트. | `tools/spritetool/assets/E-transparent.png` |
| [sources/N-transparent.png](sources/N-transparent.png) | 1121×1403 | 기사 북쪽 방향 배경 투명화 원본 시트. | `tools/spritetool/assets/N-transparent.png` |
| [sources/NE-transparent.png](sources/NE-transparent.png) | 1120×1404 | 기사 북동 방향 배경 투명화 원본 시트. | `tools/spritetool/assets/NE-transparent.png` |
| [sources/NW-transparent.png](sources/NW-transparent.png) | 1121×1403 | 기사 북서 방향 배경 투명화 원본 시트. | `tools/spritetool/assets/NW-transparent.png` |
| [sources/S-transparent.png](sources/S-transparent.png) | 1121×1403 | 기사 남쪽 방향 배경 투명화 원본 시트. | `tools/spritetool/assets/S-transparent.png` |
| [sources/SE-transparent.png](sources/SE-transparent.png) | 1120×1404 | 기사 남동 방향 배경 투명화 원본 시트. | `tools/spritetool/assets/SE-transparent.png` |
| [sources/SW-transparent.png](sources/SW-transparent.png) | 1121×1403 | 기사 남서 방향 배경 투명화 원본 시트. | `tools/spritetool/assets/SW-transparent.png` |
| [sources/W-transparent.png](sources/W-transparent.png) | 1120×1404 | 기사 서쪽 방향 배경 투명화 원본 시트. | `tools/spritetool/assets/W-transparent.png` |
| [sources/grass_0.png](sources/grass_0.png) | 40×5 | 잔디 0번 원본: 가로 4프레임 애니메이션. | `tools/spritetool/assets/grass_0.png` |
| [sources/grass_1.png](sources/grass_1.png) | 40×4 | 잔디 1번 원본: 가로 4프레임 애니메이션. | `tools/spritetool/assets/grass_1.png` |
| [sources/grass_2.png](sources/grass_2.png) | 36×5 | 잔디 2번 원본: 가로 4프레임 애니메이션. | `tools/spritetool/assets/grass_2.png` |
| [sources/grass_3.png](sources/grass_3.png) | 40×5 | 잔디 3번 원본: 가로 4프레임 애니메이션. | `tools/spritetool/assets/grass_3.png` |
| [sources/knight-hit-candidates/SW-hit-options.png](sources/knight-hit-candidates/SW-hit-options.png) | 1227×1282 | 기사 남서 방향 피격 후보 4개 비교 시트. 후보 1번을 후속 버전에 적용. | `tools/spritetool/assets/knight-hit-candidates/SW-hit-options.png` |
| [sources/knights-pixel-v2/E-normalized-reference.png](sources/knights-pixel-v2/E-normalized-reference.png) | 640×960 | 기사 v2 (최초 재도트), 동쪽 보기 머리 크기 정규화 참고 시트. | `tools/spritetool/assets/knights-pixel-v2/E-normalized-reference.png` |
| [sources/knights-pixel-v2/E-pixel.png](sources/knights-pixel-v2/E-pixel.png) | 192×240 | 기사 v2 (최초 재도트), 동쪽 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v2/E-pixel.png` |
| [sources/knights-pixel-v2/E-redrawn.png](sources/knights-pixel-v2/E-redrawn.png) | 1024×1536 | 기사 v2 (최초 재도트), 동쪽 보기 재도트 생성 원화. | `tools/spritetool/assets/knights-pixel-v2/E-redrawn.png` |
| [sources/knights-pixel-v2/N-normalized-reference.png](sources/knights-pixel-v2/N-normalized-reference.png) | 640×960 | 기사 v2 (최초 재도트), 북쪽 보기 머리 크기 정규화 참고 시트. | `tools/spritetool/assets/knights-pixel-v2/N-normalized-reference.png` |
| [sources/knights-pixel-v2/N-pixel.png](sources/knights-pixel-v2/N-pixel.png) | 192×240 | 기사 v2 (최초 재도트), 북쪽 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v2/N-pixel.png` |
| [sources/knights-pixel-v2/N-redrawn.png](sources/knights-pixel-v2/N-redrawn.png) | 1024×1536 | 기사 v2 (최초 재도트), 북쪽 보기 재도트 생성 원화. | `tools/spritetool/assets/knights-pixel-v2/N-redrawn.png` |
| [sources/knights-pixel-v2/NE-normalized-reference.png](sources/knights-pixel-v2/NE-normalized-reference.png) | 640×960 | 기사 v2 (최초 재도트), 북동 보기 머리 크기 정규화 참고 시트. | `tools/spritetool/assets/knights-pixel-v2/NE-normalized-reference.png` |
| [sources/knights-pixel-v2/NE-pixel.png](sources/knights-pixel-v2/NE-pixel.png) | 192×240 | 기사 v2 (최초 재도트), 북동 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v2/NE-pixel.png` |
| [sources/knights-pixel-v2/NE-redrawn.png](sources/knights-pixel-v2/NE-redrawn.png) | 1024×1536 | 기사 v2 (최초 재도트), 북동 보기 재도트 생성 원화. | `tools/spritetool/assets/knights-pixel-v2/NE-redrawn.png` |
| [sources/knights-pixel-v2/NW-normalized-reference.png](sources/knights-pixel-v2/NW-normalized-reference.png) | 640×960 | 기사 v2 (최초 재도트), 북서 보기 머리 크기 정규화 참고 시트. | `tools/spritetool/assets/knights-pixel-v2/NW-normalized-reference.png` |
| [sources/knights-pixel-v2/NW-pixel.png](sources/knights-pixel-v2/NW-pixel.png) | 192×240 | 기사 v2 (최초 재도트), 북서 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v2/NW-pixel.png` |
| [sources/knights-pixel-v2/NW-redrawn.png](sources/knights-pixel-v2/NW-redrawn.png) | 1024×1536 | 기사 v2 (최초 재도트), 북서 보기 재도트 생성 원화. | `tools/spritetool/assets/knights-pixel-v2/NW-redrawn.png` |
| [sources/knights-pixel-v2/S-normalized-reference.png](sources/knights-pixel-v2/S-normalized-reference.png) | 640×960 | 기사 v2 (최초 재도트), 남쪽 보기 머리 크기 정규화 참고 시트. | `tools/spritetool/assets/knights-pixel-v2/S-normalized-reference.png` |
| [sources/knights-pixel-v2/S-pixel.png](sources/knights-pixel-v2/S-pixel.png) | 192×240 | 기사 v2 (최초 재도트), 남쪽 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v2/S-pixel.png` |
| [sources/knights-pixel-v2/S-redrawn.png](sources/knights-pixel-v2/S-redrawn.png) | 1024×1536 | 기사 v2 (최초 재도트), 남쪽 보기 재도트 생성 원화. | `tools/spritetool/assets/knights-pixel-v2/S-redrawn.png` |
| [sources/knights-pixel-v2/SE-normalized-reference.png](sources/knights-pixel-v2/SE-normalized-reference.png) | 640×960 | 기사 v2 (최초 재도트), 남동 보기 머리 크기 정규화 참고 시트. | `tools/spritetool/assets/knights-pixel-v2/SE-normalized-reference.png` |
| [sources/knights-pixel-v2/SE-pixel.png](sources/knights-pixel-v2/SE-pixel.png) | 192×240 | 기사 v2 (최초 재도트), 남동 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v2/SE-pixel.png` |
| [sources/knights-pixel-v2/SE-redrawn.png](sources/knights-pixel-v2/SE-redrawn.png) | 1024×1536 | 기사 v2 (최초 재도트), 남동 보기 재도트 생성 원화. | `tools/spritetool/assets/knights-pixel-v2/SE-redrawn.png` |
| [sources/knights-pixel-v2/SW-normalized-reference.png](sources/knights-pixel-v2/SW-normalized-reference.png) | 640×960 | 기사 v2 (최초 재도트), 남서 보기 머리 크기 정규화 참고 시트. | `tools/spritetool/assets/knights-pixel-v2/SW-normalized-reference.png` |
| [sources/knights-pixel-v2/SW-pixel.png](sources/knights-pixel-v2/SW-pixel.png) | 192×240 | 기사 v2 (최초 재도트), 남서 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v2/SW-pixel.png` |
| [sources/knights-pixel-v2/SW-redrawn.png](sources/knights-pixel-v2/SW-redrawn.png) | 1024×1536 | 기사 v2 (최초 재도트), 남서 보기 재도트 생성 원화. | `tools/spritetool/assets/knights-pixel-v2/SW-redrawn.png` |
| [sources/knights-pixel-v2/W-normalized-reference.png](sources/knights-pixel-v2/W-normalized-reference.png) | 640×960 | 기사 v2 (최초 재도트), 서쪽 보기 머리 크기 정규화 참고 시트. | `tools/spritetool/assets/knights-pixel-v2/W-normalized-reference.png` |
| [sources/knights-pixel-v2/W-pixel.png](sources/knights-pixel-v2/W-pixel.png) | 192×240 | 기사 v2 (최초 재도트), 서쪽 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v2/W-pixel.png` |
| [sources/knights-pixel-v2/W-redrawn.png](sources/knights-pixel-v2/W-redrawn.png) | 1024×1536 | 기사 v2 (최초 재도트), 서쪽 보기 재도트 생성 원화. | `tools/spritetool/assets/knights-pixel-v2/W-redrawn.png` |
| [sources/knights-pixel-v3/E-pixel.png](sources/knights-pixel-v3/E-pixel.png) | 192×240 | 기사 v3 (초기 동작 구성), 동쪽 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v3/E-pixel.png` |
| [sources/knights-pixel-v3/N-pixel.png](sources/knights-pixel-v3/N-pixel.png) | 192×240 | 기사 v3 (초기 동작 구성), 북쪽 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v3/N-pixel.png` |
| [sources/knights-pixel-v3/NE-pixel.png](sources/knights-pixel-v3/NE-pixel.png) | 192×240 | 기사 v3 (초기 동작 구성), 북동 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v3/NE-pixel.png` |
| [sources/knights-pixel-v3/NW-pixel.png](sources/knights-pixel-v3/NW-pixel.png) | 192×240 | 기사 v3 (초기 동작 구성), 북서 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v3/NW-pixel.png` |
| [sources/knights-pixel-v3/S-pixel.png](sources/knights-pixel-v3/S-pixel.png) | 192×240 | 기사 v3 (초기 동작 구성), 남쪽 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v3/S-pixel.png` |
| [sources/knights-pixel-v3/SE-pixel.png](sources/knights-pixel-v3/SE-pixel.png) | 192×240 | 기사 v3 (초기 동작 구성), 남동 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v3/SE-pixel.png` |
| [sources/knights-pixel-v3/SW-pixel.png](sources/knights-pixel-v3/SW-pixel.png) | 192×240 | 기사 v3 (초기 동작 구성), 남서 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v3/SW-pixel.png` |
| [sources/knights-pixel-v3/W-pixel.png](sources/knights-pixel-v3/W-pixel.png) | 192×240 | 기사 v3 (초기 동작 구성), 서쪽 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v3/W-pixel.png` |
| [sources/knights-pixel-v4/E-pixel.png](sources/knights-pixel-v4/E-pixel.png) | 192×240 | 기사 v4 (피격 후보 1 적용), 동쪽 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v4/E-pixel.png` |
| [sources/knights-pixel-v4/N-pixel.png](sources/knights-pixel-v4/N-pixel.png) | 192×240 | 기사 v4 (피격 후보 1 적용), 북쪽 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v4/N-pixel.png` |
| [sources/knights-pixel-v4/NE-pixel.png](sources/knights-pixel-v4/NE-pixel.png) | 192×240 | 기사 v4 (피격 후보 1 적용), 북동 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v4/NE-pixel.png` |
| [sources/knights-pixel-v4/NW-pixel.png](sources/knights-pixel-v4/NW-pixel.png) | 192×240 | 기사 v4 (피격 후보 1 적용), 북서 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v4/NW-pixel.png` |
| [sources/knights-pixel-v4/S-pixel.png](sources/knights-pixel-v4/S-pixel.png) | 192×240 | 기사 v4 (피격 후보 1 적용), 남쪽 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v4/S-pixel.png` |
| [sources/knights-pixel-v4/SE-pixel.png](sources/knights-pixel-v4/SE-pixel.png) | 192×240 | 기사 v4 (피격 후보 1 적용), 남동 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v4/SE-pixel.png` |
| [sources/knights-pixel-v4/SW-pixel.png](sources/knights-pixel-v4/SW-pixel.png) | 192×240 | 기사 v4 (피격 후보 1 적용), 남서 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v4/SW-pixel.png` |
| [sources/knights-pixel-v4/W-pixel.png](sources/knights-pixel-v4/W-pixel.png) | 192×240 | 기사 v4 (피격 후보 1 적용), 서쪽 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v4/W-pixel.png` |
| [sources/knights-pixel-v5/E-pixel.png](sources/knights-pixel-v5/E-pixel.png) | 192×240 | 기사 v5 (국소 조각 보정), 동쪽 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v5/E-pixel.png` |
| [sources/knights-pixel-v5/N-pixel.png](sources/knights-pixel-v5/N-pixel.png) | 192×240 | 기사 v5 (국소 조각 보정), 북쪽 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v5/N-pixel.png` |
| [sources/knights-pixel-v5/NE-pixel.png](sources/knights-pixel-v5/NE-pixel.png) | 192×240 | 기사 v5 (국소 조각 보정), 북동 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v5/NE-pixel.png` |
| [sources/knights-pixel-v5/NW-pixel.png](sources/knights-pixel-v5/NW-pixel.png) | 192×240 | 기사 v5 (국소 조각 보정), 북서 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v5/NW-pixel.png` |
| [sources/knights-pixel-v5/S-pixel.png](sources/knights-pixel-v5/S-pixel.png) | 192×240 | 기사 v5 (국소 조각 보정), 남쪽 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v5/S-pixel.png` |
| [sources/knights-pixel-v5/SE-pixel.png](sources/knights-pixel-v5/SE-pixel.png) | 192×240 | 기사 v5 (국소 조각 보정), 남동 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v5/SE-pixel.png` |
| [sources/knights-pixel-v5/SW-pixel.png](sources/knights-pixel-v5/SW-pixel.png) | 192×240 | 기사 v5 (국소 조각 보정), 남서 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v5/SW-pixel.png` |
| [sources/knights-pixel-v5/W-pixel.png](sources/knights-pixel-v5/W-pixel.png) | 192×240 | 기사 v5 (국소 조각 보정), 서쪽 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v5/W-pixel.png` |
| [sources/knights-pixel-v6/E-pixel.png](sources/knights-pixel-v6/E-pixel.png) | 192×240 | 기사 v6 (현재 채택한 연결부·동작 재구성), 동쪽 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v6/E-pixel.png` |
| [sources/knights-pixel-v6/N-pixel.png](sources/knights-pixel-v6/N-pixel.png) | 192×240 | 기사 v6 (현재 채택한 연결부·동작 재구성), 북쪽 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v6/N-pixel.png` |
| [sources/knights-pixel-v6/NE-pixel.png](sources/knights-pixel-v6/NE-pixel.png) | 192×240 | 기사 v6 (현재 채택한 연결부·동작 재구성), 북동 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v6/NE-pixel.png` |
| [sources/knights-pixel-v6/NW-pixel.png](sources/knights-pixel-v6/NW-pixel.png) | 192×240 | 기사 v6 (현재 채택한 연결부·동작 재구성), 북서 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v6/NW-pixel.png` |
| [sources/knights-pixel-v6/S-pixel.png](sources/knights-pixel-v6/S-pixel.png) | 192×240 | 기사 v6 (현재 채택한 연결부·동작 재구성), 남쪽 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v6/S-pixel.png` |
| [sources/knights-pixel-v6/SE-pixel.png](sources/knights-pixel-v6/SE-pixel.png) | 192×240 | 기사 v6 (현재 채택한 연결부·동작 재구성), 남동 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v6/SE-pixel.png` |
| [sources/knights-pixel-v6/SW-pixel.png](sources/knights-pixel-v6/SW-pixel.png) | 192×240 | 기사 v6 (현재 채택한 연결부·동작 재구성), 남서 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v6/SW-pixel.png` |
| [sources/knights-pixel-v6/W-pixel.png](sources/knights-pixel-v6/W-pixel.png) | 192×240 | 기사 v6 (현재 채택한 연결부·동작 재구성), 서쪽 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v6/W-pixel.png` |
| [sources/knights-pixel-v7/E-pixel.png](sources/knights-pixel-v7/E-pixel.png) | 192×240 | 기사 v7 (OpenCV 자동 복원 제안 · 미적용), 동쪽 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v7/E-pixel.png` |
| [sources/knights-pixel-v7/N-pixel.png](sources/knights-pixel-v7/N-pixel.png) | 192×240 | 기사 v7 (OpenCV 자동 복원 제안 · 미적용), 북쪽 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v7/N-pixel.png` |
| [sources/knights-pixel-v7/NE-pixel.png](sources/knights-pixel-v7/NE-pixel.png) | 192×240 | 기사 v7 (OpenCV 자동 복원 제안 · 미적용), 북동 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v7/NE-pixel.png` |
| [sources/knights-pixel-v7/NW-pixel.png](sources/knights-pixel-v7/NW-pixel.png) | 192×240 | 기사 v7 (OpenCV 자동 복원 제안 · 미적용), 북서 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v7/NW-pixel.png` |
| [sources/knights-pixel-v7/S-pixel.png](sources/knights-pixel-v7/S-pixel.png) | 192×240 | 기사 v7 (OpenCV 자동 복원 제안 · 미적용), 남쪽 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v7/S-pixel.png` |
| [sources/knights-pixel-v7/SE-pixel.png](sources/knights-pixel-v7/SE-pixel.png) | 192×240 | 기사 v7 (OpenCV 자동 복원 제안 · 미적용), 남동 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v7/SE-pixel.png` |
| [sources/knights-pixel-v7/SW-pixel.png](sources/knights-pixel-v7/SW-pixel.png) | 192×240 | 기사 v7 (OpenCV 자동 복원 제안 · 미적용), 남서 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v7/SW-pixel.png` |
| [sources/knights-pixel-v7/W-pixel.png](sources/knights-pixel-v7/W-pixel.png) | 192×240 | 기사 v7 (OpenCV 자동 복원 제안 · 미적용), 서쪽 보기 방향별 완성 도트 시트. | `tools/spritetool/assets/knights-pixel-v7/W-pixel.png` |
| [sources/rock_large.png](sources/rock_large.png) | 1354×1161 | 바위 대형 원본. | `tools/spritetool/assets/rock_large.png` |
| [sources/rock_medium.png](sources/rock_medium.png) | 1470×1070 | 바위 중형 원본. | `tools/spritetool/assets/rock_medium.png` |
| [sources/rock_medium2.png](sources/rock_medium2.png) | 1322×1190 | 바위 중형 변형 원본. | `tools/spritetool/assets/rock_medium2.png` |
| [sources/rock_small.png](sources/rock_small.png) | 1536×1024 | 바위 소형 원본. | `tools/spritetool/assets/rock_small.png` |
| [sources/tree1.png](sources/tree1.png) | 1028×940 | 나무 1번 투명 PNG 원본. | `tools/spritetool/assets/tree1.png` |
| [sources/tree2.png](sources/tree2.png) | 1315×1196 | 나무 2번 투명 PNG 원본. | `tools/spritetool/assets/tree2.png` |
| [sources/tree3.png](sources/tree3.png) | 1315×1196 | 나무 3번 투명 PNG 원본. | `tools/spritetool/assets/tree3.png` |
| [sources/tree_0.png](sources/tree_0.png) | 1774×887 | 이전 나무 생성 시트 0; 현재 tree1~3 교체 전 참고 원본. | `tools/spritetool/assets/tree_0.png` |
| [sources/tree_1.png](sources/tree_1.png) | 2172×724 | 이전 나무 생성 시트 1; 현재 tree1~3 교체 전 참고 원본. | `tools/spritetool/assets/tree_1.png` |

### history

| 이미지 | 크기 / 내부 프레임 | 설명 | 원래 경로 |
|---|---|---|---|
| [history/runtime/knights-v1.png](history/runtime/knights-v1.png) | 640×480 | 기사 v1 과거 런타임 아틀라스; 현재 적용본과 비교용. | `assets/knights-v1.png` |
| [history/runtime/knights-v2.png](history/runtime/knights-v2.png) | 768×480 | 기사 v2 과거 런타임 아틀라스; 현재 적용본과 비교용. | `assets/knights-v2.png` |
| [history/runtime/knights-v3.png](history/runtime/knights-v3.png) | 768×480 | 기사 v3 과거 런타임 아틀라스; 현재 적용본과 비교용. | `assets/knights-v3.png` |
| [history/runtime/knights-v4.png](history/runtime/knights-v4.png) | 768×480 | 기사 v4 과거 런타임 아틀라스; 현재 적용본과 비교용. | `assets/knights-v4.png` |
| [history/runtime/knights-v5.png](history/runtime/knights-v5.png) | 768×480 | 기사 v5 과거 런타임 아틀라스; 현재 적용본과 비교용. | `assets/knights-v5.png` |

### reviews

| 이미지 | 크기 / 내부 프레임 | 설명 | 원래 경로 |
|---|---|---|---|
| [reviews/knight-frame-review/E-after.png](reviews/knight-frame-review/E-after.png) | 1152×1540 | 기사 동쪽 방향 프레임 보정 후 검수 이미지. | `output/knight-frame-review/E-after.png` |
| [reviews/knight-frame-review/E-before.png](reviews/knight-frame-review/E-before.png) | 1152×1540 | 기사 동쪽 방향 프레임 보정 전 검수 이미지. | `output/knight-frame-review/E-before.png` |
| [reviews/knight-frame-review/N-after.png](reviews/knight-frame-review/N-after.png) | 1152×1540 | 기사 북쪽 방향 프레임 보정 후 검수 이미지. | `output/knight-frame-review/N-after.png` |
| [reviews/knight-frame-review/N-before.png](reviews/knight-frame-review/N-before.png) | 1152×1540 | 기사 북쪽 방향 프레임 보정 전 검수 이미지. | `output/knight-frame-review/N-before.png` |
| [reviews/knight-frame-review/NE-after.png](reviews/knight-frame-review/NE-after.png) | 1152×1540 | 기사 북동 방향 프레임 보정 후 검수 이미지. | `output/knight-frame-review/NE-after.png` |
| [reviews/knight-frame-review/NE-before.png](reviews/knight-frame-review/NE-before.png) | 1152×1540 | 기사 북동 방향 프레임 보정 전 검수 이미지. | `output/knight-frame-review/NE-before.png` |
| [reviews/knight-frame-review/NW-after.png](reviews/knight-frame-review/NW-after.png) | 1152×1540 | 기사 북서 방향 프레임 보정 후 검수 이미지. | `output/knight-frame-review/NW-after.png` |
| [reviews/knight-frame-review/NW-before.png](reviews/knight-frame-review/NW-before.png) | 1152×1540 | 기사 북서 방향 프레임 보정 전 검수 이미지. | `output/knight-frame-review/NW-before.png` |
| [reviews/knight-frame-review/S-after.png](reviews/knight-frame-review/S-after.png) | 1152×1540 | 기사 남쪽 방향 프레임 보정 후 검수 이미지. | `output/knight-frame-review/S-after.png` |
| [reviews/knight-frame-review/S-before.png](reviews/knight-frame-review/S-before.png) | 1152×1540 | 기사 남쪽 방향 프레임 보정 전 검수 이미지. | `output/knight-frame-review/S-before.png` |
| [reviews/knight-frame-review/SE-after.png](reviews/knight-frame-review/SE-after.png) | 1152×1540 | 기사 남동 방향 프레임 보정 후 검수 이미지. | `output/knight-frame-review/SE-after.png` |
| [reviews/knight-frame-review/SE-before.png](reviews/knight-frame-review/SE-before.png) | 1152×1540 | 기사 남동 방향 프레임 보정 전 검수 이미지. | `output/knight-frame-review/SE-before.png` |
| [reviews/knight-frame-review/SW-after.png](reviews/knight-frame-review/SW-after.png) | 1152×1540 | 기사 남서 방향 프레임 보정 후 검수 이미지. | `output/knight-frame-review/SW-after.png` |
| [reviews/knight-frame-review/SW-before.png](reviews/knight-frame-review/SW-before.png) | 1152×1540 | 기사 남서 방향 프레임 보정 전 검수 이미지. | `output/knight-frame-review/SW-before.png` |
| [reviews/knight-frame-review/W-after.png](reviews/knight-frame-review/W-after.png) | 1152×1540 | 기사 서쪽 방향 프레임 보정 후 검수 이미지. | `output/knight-frame-review/W-after.png` |
| [reviews/knight-frame-review/W-before.png](reviews/knight-frame-review/W-before.png) | 1152×1540 | 기사 서쪽 방향 프레임 보정 전 검수 이미지. | `output/knight-frame-review/W-before.png` |
| [reviews/knight-frame-review/retouch-comparison.png](reviews/knight-frame-review/retouch-comparison.png) | 1152×2156 | 기사 프레임 보정 비교. | `output/knight-frame-review/retouch-comparison.png` |
| [reviews/knight-frame-review/v6-exhausted.gif](reviews/knight-frame-review/v6-exhausted.gif) | 768×424 / 4장 | 기사 v6 탈진 애니메이션 검수 GIF. | `output/knight-frame-review/v6-exhausted.gif` |
| [reviews/knight-frame-review/v6-hit.gif](reviews/knight-frame-review/v6-hit.gif) | 768×424 / 3장 | 기사 v6 피격 애니메이션 검수 GIF. | `output/knight-frame-review/v6-hit.gif` |
| [reviews/knight-frame-review/v6-walk.gif](reviews/knight-frame-review/v6-walk.gif) | 768×424 / 4장 | 기사 v6 걷기 애니메이션 검수 GIF. | `output/knight-frame-review/v6-walk.gif` |
| [reviews/knights-v2-comparison.png](reviews/knights-v2-comparison.png) | 1536×440 | 기사 초기 축소본과 v2 재도트 비교. | `output/knights-v2-comparison.png` |
| [reviews/trees-preview.png](reviews/trees-preview.png) | 2496×1296 | 나무 3종 배치 검수 이미지. | `output/trees-preview.png` |

### legacy

| 이미지 | 크기 / 내부 프레임 | 설명 | 원래 경로 |
|---|---|---|---|
| [legacy/dist/retouched/E-transparent.png](legacy/dist/retouched/E-transparent.png) | 1121×1403 | 기사 동쪽 방향 배경 투명화 원본 시트. | `temp/dist/retouched/E-transparent.png` |
| [legacy/dist/retouched/E.webp](legacy/dist/retouched/E.webp) | 1121×1403 | 기사 동쪽 방향 초기 WebP 원본. | `temp/dist/retouched/E.webp` |
| [legacy/dist/retouched/N-transparent.png](legacy/dist/retouched/N-transparent.png) | 1121×1403 | 기사 북쪽 방향 배경 투명화 원본 시트. | `temp/dist/retouched/N-transparent.png` |
| [legacy/dist/retouched/N.webp](legacy/dist/retouched/N.webp) | 1121×1403 | 기사 북쪽 방향 초기 WebP 원본. | `temp/dist/retouched/N.webp` |
| [legacy/dist/retouched/NE-transparent.png](legacy/dist/retouched/NE-transparent.png) | 1120×1404 | 기사 북동 방향 배경 투명화 원본 시트. | `temp/dist/retouched/NE-transparent.png` |
| [legacy/dist/retouched/NE.webp](legacy/dist/retouched/NE.webp) | 1120×1404 | 기사 북동 방향 초기 WebP 원본. | `temp/dist/retouched/NE.webp` |
| [legacy/dist/retouched/NW-transparent.png](legacy/dist/retouched/NW-transparent.png) | 1121×1403 | 기사 북서 방향 배경 투명화 원본 시트. | `temp/dist/retouched/NW-transparent.png` |
| [legacy/dist/retouched/NW.webp](legacy/dist/retouched/NW.webp) | 1121×1403 | 기사 북서 방향 초기 WebP 원본. | `temp/dist/retouched/NW.webp` |
| [legacy/dist/retouched/S-transparent.png](legacy/dist/retouched/S-transparent.png) | 1121×1403 | 기사 남쪽 방향 배경 투명화 원본 시트. | `temp/dist/retouched/S-transparent.png` |
| [legacy/dist/retouched/S.webp](legacy/dist/retouched/S.webp) | 1121×1403 | 기사 남쪽 방향 초기 WebP 원본. | `temp/dist/retouched/S.webp` |
| [legacy/dist/retouched/SE-transparent.png](legacy/dist/retouched/SE-transparent.png) | 1120×1404 | 기사 남동 방향 배경 투명화 원본 시트. | `temp/dist/retouched/SE-transparent.png` |
| [legacy/dist/retouched/SE.webp](legacy/dist/retouched/SE.webp) | 1121×1403 | 기사 남동 방향 초기 WebP 원본. | `temp/dist/retouched/SE.webp` |
| [legacy/dist/retouched/SW-transparent.png](legacy/dist/retouched/SW-transparent.png) | 1121×1403 | 기사 남서 방향 배경 투명화 원본 시트. | `temp/dist/retouched/SW-transparent.png` |
| [legacy/dist/retouched/SW.webp](legacy/dist/retouched/SW.webp) | 1121×1403 | 기사 남서 방향 초기 WebP 원본. | `temp/dist/retouched/SW.webp` |
| [legacy/dist/retouched/W-transparent.png](legacy/dist/retouched/W-transparent.png) | 1120×1404 | 기사 서쪽 방향 배경 투명화 원본 시트. | `temp/dist/retouched/W-transparent.png` |
| [legacy/dist/retouched/W.webp](legacy/dist/retouched/W.webp) | 1120×1404 | 기사 서쪽 방향 초기 WebP 원본. | `temp/dist/retouched/W.webp` |
| [legacy/dist/retouched/knight-preview/assets/knights.png](legacy/dist/retouched/knight-preview/assets/knights.png) | 640×480 | 초기 웹 미리보기에 포함된 기사 아틀라스 복사본. | `temp/dist/retouched/knight-preview/assets/knights.png` |
| [legacy/dist/retouched/knight-preview/assets/terrain.png](legacy/dist/retouched/knight-preview/assets/terrain.png) | 128×40 | 초기 웹 미리보기에 포함된 지형 아틀라스 복사본. | `temp/dist/retouched/knight-preview/assets/terrain.png` |
| [legacy/dist/sprites/E.png](legacy/dist/sprites/E.png) | 2048×640 | 기사 동쪽 방향 초기 렌더링 시트. | `temp/dist/sprites/E.png` |
| [legacy/dist/sprites/N.png](legacy/dist/sprites/N.png) | 2048×640 | 기사 북쪽 방향 초기 렌더링 시트. | `temp/dist/sprites/N.png` |
| [legacy/dist/sprites/NE.png](legacy/dist/sprites/NE.png) | 2048×640 | 기사 북동 방향 초기 렌더링 시트. | `temp/dist/sprites/NE.png` |
| [legacy/dist/sprites/NW.png](legacy/dist/sprites/NW.png) | 2048×640 | 기사 북서 방향 초기 렌더링 시트. | `temp/dist/sprites/NW.png` |
| [legacy/dist/sprites/S.png](legacy/dist/sprites/S.png) | 2048×640 | 기사 남쪽 방향 초기 렌더링 시트. | `temp/dist/sprites/S.png` |
| [legacy/dist/sprites/SE.png](legacy/dist/sprites/SE.png) | 2048×640 | 기사 남동 방향 초기 렌더링 시트. | `temp/dist/sprites/SE.png` |
| [legacy/dist/sprites/SW.png](legacy/dist/sprites/SW.png) | 2048×640 | 기사 남서 방향 초기 렌더링 시트. | `temp/dist/sprites/SW.png` |
| [legacy/dist/sprites/W.png](legacy/dist/sprites/W.png) | 2048×640 | 기사 서쪽 방향 초기 렌더링 시트. | `temp/dist/sprites/W.png` |
| [legacy/dist/units/archer.webp](legacy/dist/units/archer.webp) | 1121×1403 | 궁병 초기 WebP 원본 (temp 제작 자료). | `temp/dist/units/archer.webp` |
| [legacy/dist/units/cavalry.webp](legacy/dist/units/cavalry.webp) | 1121×1403 | 기병 초기 WebP 원본 (temp 제작 자료). | `temp/dist/units/cavalry.webp` |

### references

| 이미지 | 크기 / 내부 프레임 | 설명 | 원래 경로 |
|---|---|---|---|
| [references/footman.png](references/footman.png) | 284×308 | footman 이름의 보병 참고 이미지; 현재 맵 에디터 런타임 미연결. | `footman.png` |
| [references/tree1.jpeg](references/tree1.jpeg) | 1536×1397 | 나무 1번 JPEG 참고 원화. | `tree1.jpeg` |
| [references/tree2.jpeg](references/tree2.jpeg) | 1536×1397 | 나무 2번 JPEG 참고 원화. | `tree2.jpeg` |
| [references/tree3.jpeg](references/tree3.jpeg) | 1536×1397 | 나무 3번 JPEG 참고 원화. | `tree3.jpeg` |

### variants

| 이미지 | 크기 / 내부 프레임 | 설명 | 원래 경로 |
|---|---|---|---|
| [variants/ver1/assets/graphics/enemies.png](variants/ver1/assets/graphics/enemies.png) | 2172×724 | ver1 삼국지 여포·화웅·장각·황건 장수 초상; 런타임 시트. | `ver1/assets/graphics/enemies.png` |
| [variants/ver1/assets/graphics/portraits.png](variants/ver1/assets/graphics/portraits.png) | 2172×724 | ver1 삼국지 유비·관우·장비·간옹 초상; 런타임 시트. | `ver1/assets/graphics/portraits.png` |
| [variants/ver1/assets/graphics/terrain.png](variants/ver1/assets/graphics/terrain.png) | 1774×887 | ver1 삼국지 초원·사막·숲·산·성채·요새·마을·물 지형; 런타임 시트. | `ver1/assets/graphics/terrain.png` |
| [variants/ver1/assets/graphics/units.png](variants/ver1/assets/graphics/units.png) | 1402×1122 | ver1 삼국지 보병·기병·궁병·문관 × 대기·보행A·보행B·공격·피격; 런타임 시트. | `ver1/assets/graphics/units.png` |
| [variants/ver1/bin/SRPG.app/Contents/Resources/assets/graphics/enemies.png](variants/ver1/bin/SRPG.app/Contents/Resources/assets/graphics/enemies.png) | 2172×724 | ver1 삼국지 여포·화웅·장각·황건 장수 초상; 앱 패키지 복사본. | `ver1/bin/SRPG.app/Contents/Resources/assets/graphics/enemies.png` |
| [variants/ver1/bin/SRPG.app/Contents/Resources/assets/graphics/portraits.png](variants/ver1/bin/SRPG.app/Contents/Resources/assets/graphics/portraits.png) | 2172×724 | ver1 삼국지 유비·관우·장비·간옹 초상; 앱 패키지 복사본. | `ver1/bin/SRPG.app/Contents/Resources/assets/graphics/portraits.png` |
| [variants/ver1/bin/SRPG.app/Contents/Resources/assets/graphics/terrain.png](variants/ver1/bin/SRPG.app/Contents/Resources/assets/graphics/terrain.png) | 1774×887 | ver1 삼국지 초원·사막·숲·산·성채·요새·마을·물 지형; 앱 패키지 복사본. | `ver1/bin/SRPG.app/Contents/Resources/assets/graphics/terrain.png` |
| [variants/ver1/bin/SRPG.app/Contents/Resources/assets/graphics/units.png](variants/ver1/bin/SRPG.app/Contents/Resources/assets/graphics/units.png) | 1402×1122 | ver1 삼국지 보병·기병·궁병·문관 × 대기·보행A·보행B·공격·피격; 앱 패키지 복사본. | `ver1/bin/SRPG.app/Contents/Resources/assets/graphics/units.png` |
| [variants/ver1/reports/gui-verify/screen-000005.png](variants/ver1/reports/gui-verify/screen-000005.png) | 1280×850 | ver1 삼국지 GUI 동작 검증 화면 캡처 000005. | `ver1/reports/gui-verify/screen-000005.png` |
| [variants/ver1/reports/gui-verify/screen-000010.png](variants/ver1/reports/gui-verify/screen-000010.png) | 1280×850 | ver1 삼국지 GUI 동작 검증 화면 캡처 000010. | `ver1/reports/gui-verify/screen-000010.png` |
| [variants/ver1/reports/gui-verify/screen-000015.png](variants/ver1/reports/gui-verify/screen-000015.png) | 1280×850 | ver1 삼국지 GUI 동작 검증 화면 캡처 000015. | `ver1/reports/gui-verify/screen-000015.png` |
| [variants/ver1/reports/gui-verify/screen-000027.png](variants/ver1/reports/gui-verify/screen-000027.png) | 1280×850 | ver1 삼국지 GUI 동작 검증 화면 캡처 000027. | `ver1/reports/gui-verify/screen-000027.png` |
| [variants/ver1/reports/gui-verify/screen-000269.png](variants/ver1/reports/gui-verify/screen-000269.png) | 1280×850 | ver1 삼국지 GUI 동작 검증 화면 캡처 000269. | `ver1/reports/gui-verify/screen-000269.png` |
| [variants/ver1/reports/gui-verify/screen-000271.png](variants/ver1/reports/gui-verify/screen-000271.png) | 1280×850 | ver1 삼국지 GUI 동작 검증 화면 캡처 000271. | `ver1/reports/gui-verify/screen-000271.png` |
| [variants/ver1/reports/gui-verify/screen-000273.png](variants/ver1/reports/gui-verify/screen-000273.png) | 1280×850 | ver1 삼국지 GUI 동작 검증 화면 캡처 000273. | `ver1/reports/gui-verify/screen-000273.png` |
| [variants/ver1/reports/gui-verify/screen-000275.png](variants/ver1/reports/gui-verify/screen-000275.png) | 1280×850 | ver1 삼국지 GUI 동작 검증 화면 캡처 000275. | `ver1/reports/gui-verify/screen-000275.png` |
| [variants/ver1/reports/gui-verify/screen-000277.png](variants/ver1/reports/gui-verify/screen-000277.png) | 1280×850 | ver1 삼국지 GUI 동작 검증 화면 캡처 000277. | `ver1/reports/gui-verify/screen-000277.png` |
| [variants/ver1/reports/gui-verify/screen-000279.png](variants/ver1/reports/gui-verify/screen-000279.png) | 1280×850 | ver1 삼국지 GUI 동작 검증 화면 캡처 000279. | `ver1/reports/gui-verify/screen-000279.png` |
| [variants/ver1/reports/gui-verify/screen-000299.png](variants/ver1/reports/gui-verify/screen-000299.png) | 1280×850 | ver1 삼국지 GUI 동작 검증 화면 캡처 000299. | `ver1/reports/gui-verify/screen-000299.png` |
| [variants/ver1/reports/gui-verify/screen-000623.png](variants/ver1/reports/gui-verify/screen-000623.png) | 1280×850 | ver1 삼국지 GUI 동작 검증 화면 캡처 000623. | `ver1/reports/gui-verify/screen-000623.png` |
| [variants/ver1/reports/gui-verify/screen-000625.png](variants/ver1/reports/gui-verify/screen-000625.png) | 1280×850 | ver1 삼국지 GUI 동작 검증 화면 캡처 000625. | `ver1/reports/gui-verify/screen-000625.png` |
| [variants/ver1/reports/gui-verify/screen-000627.png](variants/ver1/reports/gui-verify/screen-000627.png) | 1280×850 | ver1 삼국지 GUI 동작 검증 화면 캡처 000627. | `ver1/reports/gui-verify/screen-000627.png` |
| [variants/ver1/reports/gui-verify/screen-000631.png](variants/ver1/reports/gui-verify/screen-000631.png) | 1280×850 | ver1 삼국지 GUI 동작 검증 화면 캡처 000631. | `ver1/reports/gui-verify/screen-000631.png` |
| [variants/ver1/reports/gui-verify/screen-000633.png](variants/ver1/reports/gui-verify/screen-000633.png) | 1280×850 | ver1 삼국지 GUI 동작 검증 화면 캡처 000633. | `ver1/reports/gui-verify/screen-000633.png` |
| [variants/ver1/reports/gui-verify/screen-000653.png](variants/ver1/reports/gui-verify/screen-000653.png) | 1280×850 | ver1 삼국지 GUI 동작 검증 화면 캡처 000653. | `ver1/reports/gui-verify/screen-000653.png` |
| [variants/ver1/reports/gui-verify/screen-001377.png](variants/ver1/reports/gui-verify/screen-001377.png) | 1280×850 | ver1 삼국지 GUI 동작 검증 화면 캡처 001377. | `ver1/reports/gui-verify/screen-001377.png` |
| [variants/ver1/reports/gui-verify/screen-001379.png](variants/ver1/reports/gui-verify/screen-001379.png) | 1280×850 | ver1 삼국지 GUI 동작 검증 화면 캡처 001379. | `ver1/reports/gui-verify/screen-001379.png` |
| [variants/ver1/reports/gui-verify/screen-001381.png](variants/ver1/reports/gui-verify/screen-001381.png) | 1280×850 | ver1 삼국지 GUI 동작 검증 화면 캡처 001381. | `ver1/reports/gui-verify/screen-001381.png` |
| [variants/ver1/tools/spritetool/assets/three-kingdoms/enemies-source.png](variants/ver1/tools/spritetool/assets/three-kingdoms/enemies-source.png) | 2172×724 | ver1 삼국지 여포·화웅·장각·황건 장수 초상; 원화. | `ver1/tools/spritetool/assets/three-kingdoms/enemies-source.png` |
| [variants/ver1/tools/spritetool/assets/three-kingdoms/portraits-source.png](variants/ver1/tools/spritetool/assets/three-kingdoms/portraits-source.png) | 2172×724 | ver1 삼국지 유비·관우·장비·간옹 초상; 원화. | `ver1/tools/spritetool/assets/three-kingdoms/portraits-source.png` |
| [variants/ver1/tools/spritetool/assets/three-kingdoms/terrain-source.png](variants/ver1/tools/spritetool/assets/three-kingdoms/terrain-source.png) | 1774×887 | ver1 삼국지 초원·사막·숲·산·성채·요새·마을·물 지형; 원화. | `ver1/tools/spritetool/assets/three-kingdoms/terrain-source.png` |
| [variants/ver1/tools/spritetool/assets/three-kingdoms/units-source.png](variants/ver1/tools/spritetool/assets/three-kingdoms/units-source.png) | 1402×1122 | ver1 삼국지 보병·기병·궁병·문관 × 대기·보행A·보행B·공격·피격; 원화. | `ver1/tools/spritetool/assets/three-kingdoms/units-source.png` |
| [variants/ver1/tools/spritetool/assets/three-kingdoms/units-v2-source.png](variants/ver1/tools/spritetool/assets/three-kingdoms/units-v2-source.png) | 1402×1122 | ver1 삼국지 보병·기병·궁병·문관 × 대기·보행A·보행B·공격·피격; 원화. | `ver1/tools/spritetool/assets/three-kingdoms/units-v2-source.png` |
| [variants/ver2/art-candidates/A/out/portrait.png](variants/ver2/art-candidates/A/out/portrait.png) | 32×36 | ver2 후보 A (클래식): 관우 초상. | `ver2/art-candidates/A/out/portrait.png` |
| [variants/ver2/art-candidates/A/out/screen.png](variants/ver2/art-candidates/A/out/screen.png) | 1280×850 | ver2 후보 A (클래식): 전장·UI 화면 목업. | `ver2/art-candidates/A/out/screen.png` |
| [variants/ver2/art-candidates/A/out/sheet-x4.png](variants/ver2/art-candidates/A/out/sheet-x4.png) | 1536×336 | ver2 후보 A (클래식): 보병·기병·황건 적병 애니메이션 시트 (4배 확대 검수본). | `ver2/art-candidates/A/out/sheet-x4.png` |
| [variants/ver2/art-candidates/A/out/sheet.png](variants/ver2/art-candidates/A/out/sheet.png) | 384×84 | ver2 후보 A (클래식): 보병·기병·황건 적병 애니메이션 시트. | `ver2/art-candidates/A/out/sheet.png` |
| [variants/ver2/art-candidates/B/out/portrait-x4.png](variants/ver2/art-candidates/B/out/portrait-x4.png) | 256×288 | ver2 후보 B (정밀): 관우 초상 (4배 확대 검수본). | `ver2/art-candidates/B/out/portrait-x4.png` |
| [variants/ver2/art-candidates/B/out/portrait.png](variants/ver2/art-candidates/B/out/portrait.png) | 64×72 | ver2 후보 B (정밀): 관우 초상. | `ver2/art-candidates/B/out/portrait.png` |
| [variants/ver2/art-candidates/B/out/screen.png](variants/ver2/art-candidates/B/out/screen.png) | 1280×850 | ver2 후보 B (정밀): 전장·UI 화면 목업. | `ver2/art-candidates/B/out/screen.png` |
| [variants/ver2/art-candidates/B/out/sheet-x4.png](variants/ver2/art-candidates/B/out/sheet-x4.png) | 3584×672 | ver2 후보 B (정밀): 보병·기병·황건 적병 애니메이션 시트 (4배 확대 검수본). | `ver2/art-candidates/B/out/sheet-x4.png` |
| [variants/ver2/art-candidates/B/out/sheet.png](variants/ver2/art-candidates/B/out/sheet.png) | 896×168 | ver2 후보 B (정밀): 보병·기병·황건 적병 애니메이션 시트. | `ver2/art-candidates/B/out/sheet.png` |
| [variants/ver2/art-candidates/C/out/portrait.png](variants/ver2/art-candidates/C/out/portrait.png) | 48×48 | ver2 후보 C (아이소메트릭): 관우 초상. | `ver2/art-candidates/C/out/portrait.png` |
| [variants/ver2/art-candidates/C/out/screen.png](variants/ver2/art-candidates/C/out/screen.png) | 1280×850 | ver2 후보 C (아이소메트릭): 전장·UI 화면 목업. | `ver2/art-candidates/C/out/screen.png` |
| [variants/ver2/art-candidates/C/out/sheet-x4.png](variants/ver2/art-candidates/C/out/sheet-x4.png) | 2560×800 | ver2 후보 C (아이소메트릭): 보병·기병·황건 적병 애니메이션 시트 (4배 확대 검수본). | `ver2/art-candidates/C/out/sheet-x4.png` |
| [variants/ver2/art-candidates/C/out/sheet.png](variants/ver2/art-candidates/C/out/sheet.png) | 640×200 | ver2 후보 C (아이소메트릭): 보병·기병·황건 적병 애니메이션 시트. | `ver2/art-candidates/C/out/sheet.png` |
| [variants/ver2/art-candidates/C2/out/portrait.png](variants/ver2/art-candidates/C2/out/portrait.png) | 96×96 | ver2 후보 C2 (아이소메트릭 생성 원화): 관우 초상. | `ver2/art-candidates/C2/out/portrait.png` |
| [variants/ver2/art-candidates/C2/out/portraits/chengyuanzhi.png](variants/ver2/art-candidates/C2/out/portraits/chengyuanzhi.png) | 96×96 | ver2 후보 C2 (아이소메트릭 생성 원화): 정원지 초상. | `ver2/art-candidates/C2/out/portraits/chengyuanzhi.png` |
| [variants/ver2/art-candidates/C2/out/portraits/guanyu.png](variants/ver2/art-candidates/C2/out/portraits/guanyu.png) | 96×96 | ver2 후보 C2 (아이소메트릭 생성 원화): 관우 초상. | `ver2/art-candidates/C2/out/portraits/guanyu.png` |
| [variants/ver2/art-candidates/C2/out/portraits/liubei.png](variants/ver2/art-candidates/C2/out/portraits/liubei.png) | 96×96 | ver2 후보 C2 (아이소메트릭 생성 원화): 유비 초상. | `ver2/art-candidates/C2/out/portraits/liubei.png` |
| [variants/ver2/art-candidates/C2/out/portraits/zhangfei.png](variants/ver2/art-candidates/C2/out/portraits/zhangfei.png) | 96×96 | ver2 후보 C2 (아이소메트릭 생성 원화): 장비 초상. | `ver2/art-candidates/C2/out/portraits/zhangfei.png` |
| [variants/ver2/art-candidates/C2/out/portraits/zhangjue.png](variants/ver2/art-candidates/C2/out/portraits/zhangjue.png) | 96×96 | ver2 후보 C2 (아이소메트릭 생성 원화): 장각 초상. | `ver2/art-candidates/C2/out/portraits/zhangjue.png` |
| [variants/ver2/art-candidates/C2/out/screen.png](variants/ver2/art-candidates/C2/out/screen.png) | 1280×850 | ver2 후보 C2 (아이소메트릭 생성 원화): 전장·UI 화면 목업. | `ver2/art-candidates/C2/out/screen.png` |
| [variants/ver2/art-candidates/C2/out/sheet-x4.png](variants/ver2/art-candidates/C2/out/sheet-x4.png) | 4096×1024 | ver2 후보 C2 (아이소메트릭 생성 원화): 보병·기병·황건 적병 애니메이션 시트 (4배 확대 검수본). | `ver2/art-candidates/C2/out/sheet-x4.png` |
| [variants/ver2/art-candidates/C2/out/sheet.png](variants/ver2/art-candidates/C2/out/sheet.png) | 1024×256 | ver2 후보 C2 (아이소메트릭 생성 원화): 보병·기병·황건 적병 애니메이션 시트. | `ver2/art-candidates/C2/out/sheet.png` |
| [variants/ver2/art-candidates/C2/src/guanyu-test.png](variants/ver2/art-candidates/C2/src/guanyu-test.png) | 1254×1254 | ver2 후보 C2 (아이소메트릭 생성 원화): 관우 초상 시험 원화. | `ver2/art-candidates/C2/src/guanyu-test.png` |
| [variants/ver2/art-candidates/C2/src/portrait-chengyuanzhi.png](variants/ver2/art-candidates/C2/src/portrait-chengyuanzhi.png) | 1254×1254 | ver2 후보 C2 (아이소메트릭 생성 원화): 정원지 초상 원화. | `ver2/art-candidates/C2/src/portrait-chengyuanzhi.png` |
| [variants/ver2/art-candidates/C2/src/portrait-guanyu.png](variants/ver2/art-candidates/C2/src/portrait-guanyu.png) | 1254×1254 | ver2 후보 C2 (아이소메트릭 생성 원화): 관우 초상 원화. | `ver2/art-candidates/C2/src/portrait-guanyu.png` |
| [variants/ver2/art-candidates/C2/src/portrait-liubei.png](variants/ver2/art-candidates/C2/src/portrait-liubei.png) | 1254×1254 | ver2 후보 C2 (아이소메트릭 생성 원화): 유비 초상 원화. | `ver2/art-candidates/C2/src/portrait-liubei.png` |
| [variants/ver2/art-candidates/C2/src/portrait-zhangfei.png](variants/ver2/art-candidates/C2/src/portrait-zhangfei.png) | 1254×1254 | ver2 후보 C2 (아이소메트릭 생성 원화): 장비 초상 원화. | `ver2/art-candidates/C2/src/portrait-zhangfei.png` |
| [variants/ver2/art-candidates/C2/src/portrait-zhangjue.png](variants/ver2/art-candidates/C2/src/portrait-zhangjue.png) | 1254×1254 | ver2 후보 C2 (아이소메트릭 생성 원화): 장각 초상 원화. | `ver2/art-candidates/C2/src/portrait-zhangjue.png` |
| [variants/ver2/art-candidates/C2/src/unit-bandit-idle.png](variants/ver2/art-candidates/C2/src/unit-bandit-idle.png) | 2172×724 | ver2 후보 C2 (아이소메트릭 생성 원화): 황건 적병 대기 생성 원화. | `ver2/art-candidates/C2/src/unit-bandit-idle.png` |
| [variants/ver2/art-candidates/C2/src/unit-bandit-walk.png](variants/ver2/art-candidates/C2/src/unit-bandit-walk.png) | 2172×724 | ver2 후보 C2 (아이소메트릭 생성 원화): 황건 적병 걷기 생성 원화. | `ver2/art-candidates/C2/src/unit-bandit-walk.png` |
| [variants/ver2/art-candidates/C2/src/unit-cavalry-idle.png](variants/ver2/art-candidates/C2/src/unit-cavalry-idle.png) | 2172×724 | ver2 후보 C2 (아이소메트릭 생성 원화): 기병 대기 생성 원화. | `ver2/art-candidates/C2/src/unit-cavalry-idle.png` |
| [variants/ver2/art-candidates/C2/src/unit-cavalry-walk.png](variants/ver2/art-candidates/C2/src/unit-cavalry-walk.png) | 2172×724 | ver2 후보 C2 (아이소메트릭 생성 원화): 기병 걷기 생성 원화. | `ver2/art-candidates/C2/src/unit-cavalry-walk.png` |
| [variants/ver2/art-candidates/C2/src/unit-infantry-idle.png](variants/ver2/art-candidates/C2/src/unit-infantry-idle.png) | 2172×724 | ver2 후보 C2 (아이소메트릭 생성 원화): 보병 대기 생성 원화. | `ver2/art-candidates/C2/src/unit-infantry-idle.png` |
| [variants/ver2/art-candidates/C2/src/unit-infantry-walk.png](variants/ver2/art-candidates/C2/src/unit-infantry-walk.png) | 2172×724 | ver2 후보 C2 (아이소메트릭 생성 원화): 보병 걷기 생성 원화. | `ver2/art-candidates/C2/src/unit-infantry-walk.png` |
| [variants/ver2/art-candidates/D/out/portrait-x4.png](variants/ver2/art-candidates/D/out/portrait-x4.png) | 288×288 | ver2 후보 D (수묵 UI): 관우 초상 (4배 확대 검수본). | `ver2/art-candidates/D/out/portrait-x4.png` |
| [variants/ver2/art-candidates/D/out/portrait.png](variants/ver2/art-candidates/D/out/portrait.png) | 72×72 | ver2 후보 D (수묵 UI): 관우 초상. | `ver2/art-candidates/D/out/portrait.png` |
| [variants/ver2/art-candidates/D/out/screen.png](variants/ver2/art-candidates/D/out/screen.png) | 1280×850 | ver2 후보 D (수묵 UI): 전장·UI 화면 목업. | `ver2/art-candidates/D/out/screen.png` |
| [variants/ver2/art-candidates/D/out/sheet-x4.png](variants/ver2/art-candidates/D/out/sheet-x4.png) | 4096×768 | ver2 후보 D (수묵 UI): 보병·기병·황건 적병 애니메이션 시트 (4배 확대 검수본). | `ver2/art-candidates/D/out/sheet-x4.png` |
| [variants/ver2/art-candidates/D/out/sheet.png](variants/ver2/art-candidates/D/out/sheet.png) | 1024×192 | ver2 후보 D (수묵 UI): 보병·기병·황건 적병 애니메이션 시트. | `ver2/art-candidates/D/out/sheet.png` |
| [variants/ver2/art-candidates/D/out/terrain-x3.png](variants/ver2/art-candidates/D/out/terrain-x3.png) | 1344×192 | ver2 후보 D (수묵 UI): 지형 타일 시트 (3배 확대 검수본). | `ver2/art-candidates/D/out/terrain-x3.png` |
| [variants/ver2/art-candidates/D/out/terrain.png](variants/ver2/art-candidates/D/out/terrain.png) | 448×64 | ver2 후보 D (수묵 UI): 지형 타일 시트. | `ver2/art-candidates/D/out/terrain.png` |
