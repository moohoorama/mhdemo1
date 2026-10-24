# 빌드 도구

이 문서가 맡는 범위: 에셋·앱 빌드 명령(스프라이트, 상태 아이콘, 거점 그림, Android APK)과 Android 빌드의 설계.
값이 다르면 `Makefile`, `tools/build_android.py`, `tools/spritebuild/build.py`를 따른다.

## 에셋 빌드

```sh
python3 tools/spritebuild/build.py     # tools/doteditor/units/*.yaml → assets/sprites/*.spr (형식: presentation-sprite.md)
uv run --with pillow python tools/build_status_icons.py   # 상태 아이콘 수제 픽셀 원본 → assets/graphics/status/
uv run --with pillow python tools/build_effects.py        # 수제 픽셀 원본 → assets/graphics/{effects/projectiles,effects/fx(+_light),map/outside,ui/ui}.png, .json
```

```sh
python3 tools/build_extra.py                                   # ../assets_candidate 그림 → assets/graphics/map/extra.png, effects/*.png (index.json의 extra, effects)
node tools/scene_maps/export.mjs > /tmp/scenes_draws.json      # 후보 viewer의 장면을 돌려 어떤 그림이 어디에 놓이는지 뽑는다
python3 tools/scene_maps/build.py /tmp/scenes_draws.json       # → assets/graphics/maps/S_*.json 와 index.json의 maps
python3 tools/scene_maps/compose.py                            # viewer에 없는 방(S_council, S_hall, S_tent_mid, S_tent_lux, S_tavern, S_inn), 야외(S_bridge, S_pass, S_peach_grove, S_castle_town, S_jail_street, S_market, S_checkpoint), 변형(S_turban_camp, S_rampart_siege, S_army_camp_hungry, S_castle_town_burnt)
python3 tools/scene_maps/scene_v2.py [--rev REV] S_id ...     # 옛 장면 맵 → 형식 2(땅 글자·높이·물체·성벽 줄). compose.py는 이미 형식 2인 맵을 덮어쓰지 않는다
python3 tools/scene_maps/backdrops.py                          # 장면 원경 → assets/graphics/backdrops/*.png (index.json의 backdrops)
python3 tools/scene_maps/scene_props.py                        # 색만 바꾼 장면 그림(누런 깃발, 금빛 병풍)과 새·반짝임 → ../assets_candidate/tiles
python3 tools/scene_maps/join_overlay.py                       # 알파만 쓴 그림자: 모서리 삼각형 jshade_*, 절벽 밑동 cfoot_* → ../assets_candidate/tiles
python3 tools/scene_maps/tall_cliff.py                         # 높은 장면 절벽 tcliff_*: 전투 맵 절벽 블록의 바위 면 행을 쌓음 → ../assets_candidate/tiles (그 뒤 build_extra.py, compose.py)
python3 ../assets_candidate/tools/gen_3d_castle.py --write      # 장면용 성벽·성문(성문은 아치 가운데에서 갈라 반 칸 조각) → ../assets_candidate/tiles
python3 ../tools/unit3d/scene_props3d.py                        # 3D 모델 장면 소품 → ../assets_candidate/tiles (--review: 옛 그림과 비교, 이름을 주면 그 소품만, --out=경로; 그 뒤 build_extra.py)
python3 tools/pixelize_tree.py [이름 …]                         # 장면 나무: codex 그림(../assets_candidate/tiles/sources/*.png) → 3D 렌더와 같은 크기·피벗의 도트 (값은 sources/trees.json; 그 뒤 build_extra.py)
python3 tools/doteditor/tools/civ3d.py official wounded …       # 장면 사람 시트(3D 변환, tools/unit3d/scene_people.py)
python3 tools/doteditor/tools/civ3d.py --poses liubei …        # 기존 비전투 시트에 sit·throne 같은 자세 칸만 덧붙임(다른 칸은 그대로)
python3 tools/scene_maps/wire_story.py                         # 장면이 없는 이야기 구간에 assets/scenes/chNN.json의 빈 장면(맵·노드)을 만든다
python3 tools/scene_maps/scene_format.py                       # assets/scenes/*.json 줄 배치를 맞춘다
```

- `build_extra.py`는 후보의 성벽(이음새 없는 `wall_clean_*`, 게임의 `wall_*`를 대신한다)·성문 C·울타리·그을음·불탄 구조물·다리 손상·장면 소품과 불·연기 줄을 게임 그림 묶음으로 옮긴다. 옛 기둥 성문(`gate_*`)과 숲 속 덤불 시안은 넣지 않는다. 불·연기 줄은 `Assets.Flames`, 그림은 `Assets.Props`로 읽힌다.
- 장면 나무(`stree_*`·`peach_bloom_*`)는 3D 렌더를 그대로 쓰지 않는다. 다시 그리려면: `python3 ../tools/unit3d/scene_props3d.py <이름> --3d`로 렌더를 `../output/scene-props3d/`에 받고(인자 없이 돌려도 칠한 나무는 덮어쓰지 않는다), 정수배 최근접 확대해 1024 마젠타 판에 놓아 codex-image에 참고로 준다. 요청문은 `trees.json`의 `prompt`를 따른다 — "참고는 실루엣·크기·줄기 기울기·가지 배치·빛 방향(왼쪽 위)만, 공 모양 덩이·덩이 위 구멍은 베끼지 말 것, 16비트 손도트처럼 큰 잎 덩이 몇 개·3톤 셀 음영·1px 외곽선"(참고를 그대로 따르라고만 하면 3D 덩이를 그대로 베낀다). 후보 둘 중 고른 그림을 512px로 `sources/<이름>.png`에 두고 `pixelize_tree.py`를 돌린다. 변환 값: `penalty`(그 색을 덜 쓰게, 소나무 밝은 잎), `dark`(가닥·꽃 덩이 사이 어두운 틈을 살림), `bark`+`foot`/`speck`+`inner`(꽃 둘레에 칠한 적갈색 테두리가 껍질색으로 잡히는 것을 테두리·꽃 그늘로).
- 장면 맵은 viewer의 JS 장면 정의에서 뽑아 만든다. 원리와 빠지는 것(유닛·불꽃·하늘·밤 색)은 `tools/scene_maps/build.py` 위쪽 설명과 [presentation-scene.md](presentation-scene.md)에 있다.
- 상태 아이콘 원본·설정은 `../tools/spritetool/assets/ver5-status/`, 런타임은 `assets/graphics/status/`다.
- 수제 픽셀 원본은 `../tools/spritetool/assets/`의 `ver5-projectiles/`(돌·그림자·흙먼지), `ver5-fx/`(책략·전투·피격·장면 효과, 일기토 섬광, 입자), `ver5-outside/`(맵 밖 소나무·돌), `ver5-ui/`(창 테두리, 표시·아이콘, HP바 부품, 두루마리 무늬)다. 스프라이트는 레이어 묶음이고, 레이어는 아스키 격자(`pixels.json`, 팔레트 색) 또는 `gen.py` 함수가 그리는 부드러운 반투명 그림이다. 레이어마다 투명도(하나 또는 프레임별)와 합성(`normal`/`lighter`)이 있어 빌더가 보통 시트와 빛 시트(`_light.png`, 게임이 더하기 합성으로 겹침)로 나눠 굽는다. 각 원본의 `seed.py`는 격자 첫 판을 뽑는 도구(공용 도구 `tools/pixelgrid.py`)로, 다시 돌리면 손질이 지워진다. 런타임은 `Assets.Projectiles`·`FX`·`Outside`·`UI`(`internal/gui/shots.go`).
- UI 그림은 그림 4px이 UI 1px이고 부드럽게 축소한다(아이콘·삼각 표시, 이웃 그림이 섞이지 않게 둘레 8px 투명 여백). 단, 창 테두리는 그림 1px이 UI 1px이고 최근접으로 9조각 늘려 단색 사각형처럼 또렷하다. HP바 부품은 게임 화소로 맵 위에 그린다.
- `cmd/_uidemo-vm`은 기본 크기(1280×850)에서 UI를 절반 해상도 버퍼에 그린 뒤 2배로 키운다. 1px 선을 비교할 때는 `-size 2560x1700`으로 찍는다.
- 거점 그림은 `../tools/unit3d/sites.py`가 `assets/graphics/map/sites.png`로 굽는다(게임 칸 하나에 한 장, 관문은 성벽 줄 방향별 두 장).

## Android APK

Go, Python 3, JDK 17, Android SDK Platform 33·Build-Tools·NDK를 설치한 뒤 실행한다.
Android Studio/Gradle은 필요하지 않다. macOS의 기본 SDK 경로는 `~/Library/Android/sdk`이고 다른 경로는 `--sdk` 또는 `ANDROID_HOME`으로 지정한다.

```sh
cd ver5
make android                 # APK 빌드
make android-arm64           # ARM64 휴대폰 전용으로 작게
make android-install         # 연결된 기기에 설치
make android ANDROID_SDK=/path/to/Android/sdk
```

- 스크립트를 직접 실행하려면 `python3 tools/build_android.py`. macOS에서는 `JAVA_HOME`이 없으면 설치된 JDK 17을 자동 선택한다. 직접 지정은 `JAVA_HOME=$(/usr/libexec/java_home -v 17) make android`.
- 빌드 스크립트는 Go 바인딩 생성부터 APK 서명까지 같은 JDK를 쓴다. JDK 25로 만든 클래스는 Build-Tools 34의 D8에서 내부 오류가 날 수 있다.
- 결과는 `bin/android/srpg-ver5-debug.apk`. ARM64 휴대폰과 x86_64 에뮬레이터를 지원한다. 최소 Android 6.0(API 23), OpenGL ES 3.0, 가로 화면.
- 테스트 서명 APK이며 Play Store 배포용 서명·AAB는 없다. 서명 키는 `bin/android/debug.keystore`에 있으므로 같은 기기에 업데이트하려면 이 파일을 유지한다.
- 조작: 한 손가락은 버튼·유닛 클릭과 전장 드래그 이동, 두 손가락은 지도 이동·핀치 확대/축소. 에디터에서는 한 손가락 드래그로 칠하고 두 손가락으로 이동한다. 뒤로 가기는 Escape와 같다.
- 세이브는 앱 내부 저장소에 유지된다. 에셋이 변경된 APK 업데이트는 배포 에셋을 다시 풀기 때문에 앱에서 편집한 맵이 덮어써질 수 있다.

## Android 설계

기존 데스크톱 GUI와 규칙 엔진을 재사용한다. `mobile` 패키지는 Android의
앱 전용 경로로 콘텐츠·폰트·세이브를 초기화하고 Ebitengine `mobile.SetGame`에
GUI를 전달한다. Android Activity는 APK에 포함된 런타임 에셋을 앱 내부에
풀고, EbitenView의 시작·중단·재개와 뒤로 가기를 담당한다. 세이브는 에셋과
별도의 디렉터리에 저장한다. 에셋 묶음이 변경된 업데이트에서는 배포 에셋을
교체하므로 앱 내부 맵 편집은 다음 업데이트에서 덮어쓸 수 있다.

입력은 기존 GUI 버튼·카메라·에디터가 공유하는 포인터 함수를 통해 연결한다.
한 손가락은 클릭/드래그, 두 손가락은 이동/핀치 확대이며 Android 뒤로 가기는
Escape와 같다. 데스크톱 마우스와 키보드 입력은 기존 동작을 유지한다.

`tools/build_android.py`는 프로젝트에 고정된 Ebitengine 버전으로 AAR을
만들고, Android SDK의 aapt2·javac·d8·zipalign·apksigner로 테스트 APK를
패키징한다. Gradle/Android Studio 프로젝트 없이 실행할 수 있다.
기본 ABI는 arm64-v8a와 x86_64, 최소 Android 버전은 API 23이다.
앱은 가로 방향으로 실행하며, 인터넷/외부 저장소 권한을 요구하지 않는다.
빌드 산출물과 테스트 서명 키는 `bin/android/`에 두고 Git에서 제외한다.

검증: 터치 상태 전환 단위 테스트, 관련 Go 패키지 테스트, APK 서명 검사,
에뮬레이터에서 시작 메뉴·새 게임·일기토 실행 및 터치/뒤로 가기 확인.
실제 휴대폰의 성능과 모든 캠페인 진행은 별도 검수가 필요하다.

## 검증 결과 (2026-10-13)

- `go test ./internal/gui ./internal/content ./internal/storage ./internal/session ./mobile`
  및 `go vet ./internal/gui ./mobile` 통과. 데스크톱 GUI 빌드도 통과.
- `python3 tools/build_android.py`로 약 20.2 MiB 테스트 APK 생성.
  APK v1/v2/v3 서명 검사 통과, ARM64/x86_64 ELF LOAD 정렬 16KB 확인.
- Android 16(API 36) ARM64 에뮬레이터에 설치·실행. 한글·초상·스프라이트,
  터치로 새 게임/일기토 감상, Android 뒤로 가기 확인.
- 터치로 슬롯 1 저장 후 같은 서명 APK 재설치. 세이브 유지와 불러오기 확인.
- 화면 증거는 로컬 `reports/android/`에 보관한다(Git 제외).
- 핀치 상태 전환은 단위 테스트로 검증했다. 실제 기기 멀티터치와 전체 캠페인,
  x86_64 실행은 아직 검수하지 않았다.
