# 기사 스프라이트 도구

## 현재 사용하는 경로

`project.json` → 완성된 v6 원본 + `frames.json` → `../build_knights.py`
→ 범용 spritetool crop/pack → `assets/knights.png`, `assets/knights.json`
→ `knight-preview.html`.

`project.json`은 현재 빌드 원본, 아틀라스 열 수, 미리보기 버전과 표시 문구를 관리한다.
각 원본의 `frames.json`은 프레임 좌표·방향·동작·기준점·시간을 관리한다.
미리보기는 출력 JSON을 읽으며, 과거 버전은 선택할 때만 로드한다.
현재 PNG의 해시를 출력 메타데이터에 기록해 이미지 캐시를 갱신한다.

```sh
make knights
make check-knights
make preview-knights
```

일반 패킹에는 Python 표준 라이브러리와 Go만 필요하다. `--check`는 Go 빌드 없이
원본 PNG 크기, crop 범위, 프레임 누락·중복, 재생 시간을 검사한다.

```sh
python3 tools/build_knights.py --manifest tools/spritetool/assets/knights-pixel-v6/frames.json --output /tmp/knights-check
```

상대 경로 옵션은 `--root`(기본값: 저장소 루트)를 기준으로 해석한다.
파일 시스템 위치와 상관없이 같은 결과를 만든다. import 시에는 빌드를 실행하지 않는다.

## 제작 이력과 선택적 도구

현재 이미지를 수정하지 않고 재패킹하려면 위의 일반 빌드만 사용한다.
다음 스크립트들은 기존 원본을 다시 가공하는 제작 레시피이며 일반 빌드가 호출하지 않는다.
재작업 전에 [공통 가이드](../spritetool/assets/SPRITE_ART_GUIDE.md)와 대상 원본 README를 읽는다.

| 스크립트 (`tools/` 기준) | 입력 → 출력 | 용도 |
|---|---|---|
| `prepare_knight_references.py` | 투명 원본 → v2 참고 시트 | 최초 머리 크기 정규화 |
| `finalize_knight_pixels.py` | 보관된 ImageGen 시트 → v2 도트 | 최초 팔레트·크기 정리 |
| `animate_knight_pixels.py` | v2 → v3 | 초기 동작 레시피와 방향별 rig |
| `knight_hit_backarch.py` | v3 → v4 | 피격 후보 1번 |
| `retouch_knight_frames.py` | v4 → v5 | 국소 픽셀 보정 이력 |
| `rebuild_knight_motion.py` | v2 + v5 → v6 | 현재 동작 레시피 |
| `repair_knights_cv.py` | v6 + v2 → v7 제안 | 선택적 OpenCV 실험, 런타임 미적용 |
| `build_knights_v1.py` | 이전 투명 원본 → 초기 축소본 | 초기 제작 방식 보관 |

일반 제작 레시피에는 Pillow와 NumPy가 필요하다. OpenCV 실험만 별도의 환경을 사용한다.

```sh
python3 -m venv /tmp/knight-opencv-env
/tmp/knight-opencv-env/bin/pip install -r tools/requirements-knight-cv.txt
/tmp/knight-opencv-env/bin/python3 tools/repair_knights_cv.py
```

`knight-cv-review.html`은 v6와 v7의 진단 비교, `knight-hit-candidates.html`은 피격 선택 이력이다.
`output/knight-frame-review/`는 검수 이미지·분석 기록이다. 이들은 현재 빌드의 입력이 아니다.
기존 경로와 staged 파일은 유지해 이전 링크와 롤백 기준을 보존한다.

## 검증 범위

`check_preview.cjs`는 실제 JSON·PNG를 읽는 Node 기반 미리보기 계약 검사다.
버전 전환, 동작별 프레임 선택, 과거 동작 이름 변환, 이미지 좌표와 발밑 정렬을 확인한다.
실제 브라우저 렌더링이나 도트의 시각적 품질 검사를 대신하지 않는다.
