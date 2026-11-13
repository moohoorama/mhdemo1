# 빌드 도구

이 문서가 맡는 범위: 에셋·앱 빌드 명령(스프라이트, 상태 아이콘, 거점 그림, Android APK)과 Android 빌드의 설계.
값이 다르면 `Makefile`, `tools/build_android.py`, `tools/spritebuild/build.py`를 따른다.

## 에셋 빌드

```sh
python3 tools/spritebuild/build.py     # tools/doteditor/units/*.yaml → assets/sprites/*.spr (형식: presentation-sprite.md)
uv run --with pillow python tools/build_status_icons.py   # 상태 아이콘 수제 픽셀 원본 → assets/graphics/status/
uv run --with pillow python tools/build_effects.py        # 수제 픽셀 원본 → assets/graphics/{effects/projectiles,effects/fx(+_light),map/outside,ui/ui}.png, .json
uv run --with numpy --with scipy --with pillow python tools/spellsim.py --bake   # 물·불·땅 책략 시뮬레이션 → assets/graphics/effects/spells.png, .json
```

```sh
python3 tools/build_extra.py                                   # ../assets_candidate 그림 → assets/graphics/map/extra.png, effects/*.png (index.json의 extra, anims의 불·연기·향로 줄)
node tools/scene_maps/export.mjs > /tmp/scenes_draws.json      # 후보 viewer의 장면을 돌려 어떤 그림이 어디에 놓이는지 뽑는다
python3 tools/scene_maps/build.py /tmp/scenes_draws.json       # → assets/graphics/maps/S_*.json 와 index.json의 maps
python3 tools/scene_maps/compose.py                            # viewer에 없는 방(S_council, S_hall, S_tent_mid, S_tent_lux, S_tavern, S_inn), 야외(S_bridge, S_pass, S_peach_grove, S_castle_town, S_jail_street, S_market, S_checkpoint), 변형(S_turban_camp, S_rampart_siege, S_army_camp_hungry, S_castle_town_burnt)
python3 tools/scene_maps/scene_v2.py [--rev REV] S_id ...     # 옛 장면 맵 → 형식 2(땅 글자·높이·물체·성벽 줄). compose.py는 이미 형식 2인 맵을 덮어쓰지 않는다
python3 tools/scene_maps/backdrops.py                          # 장면 원경 → assets/graphics/backdrops/*.png (index.json의 backdrops)
python3 tools/scene_maps/scene_props.py                        # 색만 바꾼 장면 그림(누런 깃발, 금빛 병풍)과 새·반짝임 → ../assets_candidate/tiles
python3 tools/scene_maps/join_overlay.py                       # 알파만 쓴 그림자: 모서리 삼각형 jshade_*, 절벽 밑동 cfoot_* → ../assets_candidate/tiles
python3 tools/scene_maps/tall_cliff.py                         # 높은 장면 절벽 tcliff_*: 전투 맵 절벽 블록의 바위 면 행을 쌓음 → ../assets_candidate/tiles (그 뒤 build_extra.py, compose.py)
python3 ../assets_candidate/tools/gen_3d_castle.py --write      # 옛 형식 장면용 성벽·성문(성문은 아치 가운데에서 갈라 반 칸 조각) → ../assets_candidate/tiles
python3 ../assets_candidate/tools/gen_pixel_castle.py --write    # 형식 2 성벽 칸·성가퀴·밑동 그림자·성문 조각(도트로 직접) → ../assets_candidate/tiles (--preview 경로: 확인 그림; 그 뒤 build_extra.py)
python3 ../assets_candidate/tools/gen_pixel_kit.py --write       # 전투 성벽 wall_clean_*·성문 gatehouse_{closed,open}_{u,v}_{0..3}, 울타리 fence_*·마을 village_0..3·주둔지 camp와 불탄 판, 실내 벽 wall_*_{nw,ne}·바닥 floor_*(같은 방식) → ../assets_candidate/tiles (--compare 경로: 지금 그림과 비교; 그 뒤 build_extra.py. 스킬 pixel-iso-kit)
python3 tools/build_ground_tiles.py                             # 여울·평지 땅 경계 조각 두 벌(물가·흙길 경계 모양과 음영을 빌려 새 질감) → assets/graphics/map/tileset.png, index.json의 tileset (원본 tileset.png에 한 번만)
python3 ../assets_candidate/tools/gen_scenery.py --install     # 땅 장식(풀포기·자갈·여울의 늪 수풀 …)·소나무·거점 깃발 → assets/graphics/map/scenery.png, scenery.json
python3 ../tools/unit3d/scene_props3d.py                        # 3D 모델 장면 소품 → ../assets_candidate/tiles (--review: 옛 그림과 비교, 이름을 주면 그 소품만, --out=경로; 그 뒤 build_extra.py)
python3 tools/pixelize_tree.py [이름 …]                         # 장면 나무: codex 그림(../assets_candidate/tiles/sources/*.png) → 3D 렌더와 같은 크기·피벗의 도트 (값은 sources/trees.json; 그 뒤 build_extra.py)
python3 tools/doteditor/tools/civ3d.py official wounded …       # 장면 사람 시트(3D 변환, tools/unit3d/scene_people.py)
python3 tools/doteditor/tools/civ3d.py --poses liubei …        # 기존 비전투 시트에 sit·throne 같은 자세 칸만 덧붙임(다른 칸은 그대로)
python3 tools/scene_maps/wire_story.py                         # 장면이 없는 이야기 구간에 assets/scenes/chNN.json의 빈 장면(맵·노드)을 만든다
python3 tools/scene_maps/scene_format.py                       # assets/scenes/*.json 줄 배치를 맞춘다
```

- `build_extra.py`는 후보의 성벽(도트 `wall_clean_*`, 게임의 `wall_*`를 대신한다)·성문(`gatehouse_{closed,open}_*`)·실내 벽·바닥(같은 이름의 props.png 그림을 덮는다)·울타리·그을음·불탄 구조물·다리 손상·장면 소품과 불·연기 줄을 게임 그림 묶음으로 옮긴다. 옛 기둥 성문(`gate_*`)과 숲 속 덤불 시안은 넣지 않는다. `../assets_candidate/anims/anims.json`의 줄(정지 그림 위에 아스키 프레임, 향로 연기)도 `effects/`에 그린다. 불·연기·향로 줄은 `index.json`의 `"anims"`로 `Assets.Anims`, 그림은 `Assets.Props`로 읽힌다(형식은 [presentation-scene.md](presentation-scene.md)).
- 장면 나무(`stree_*`·`peach_bloom_*`)는 3D 렌더를 그대로 쓰지 않는다. 다시 그리려면: `python3 ../tools/unit3d/scene_props3d.py <이름> --3d`로 렌더를 `../output/scene-props3d/`에 받고(인자 없이 돌려도 칠한 나무는 덮어쓰지 않는다), 정수배 최근접 확대해 1024 마젠타 판에 놓아 codex-image에 참고로 준다. 요청문은 `trees.json`의 `prompt`를 따른다 — "참고는 실루엣·크기·줄기 기울기·가지 배치·빛 방향(왼쪽 위)만, 공 모양 덩이·덩이 위 구멍은 베끼지 말 것, 16비트 손도트처럼 큰 잎 덩이 몇 개·3톤 셀 음영·1px 외곽선"(참고를 그대로 따르라고만 하면 3D 덩이를 그대로 베낀다). 후보 둘 중 고른 그림을 512px로 `sources/<이름>.png`에 두고 `pixelize_tree.py`를 돌린다. 변환 값: `penalty`(그 색을 덜 쓰게, 소나무 밝은 잎), `dark`(가닥·꽃 덩이 사이 어두운 틈을 살림), `bark`+`foot`/`speck`+`inner`(꽃 둘레에 칠한 적갈색 테두리가 껍질색으로 잡히는 것을 테두리·꽃 그늘로).
- 장면 맵은 viewer의 JS 장면 정의에서 뽑아 만든다. 원리와 빠지는 것(유닛·불꽃·하늘·밤 색)은 `tools/scene_maps/build.py` 위쪽 설명과 [presentation-scene.md](presentation-scene.md)에 있다.
- 상태 아이콘 원본·설정은 `../tools/spritetool/assets/ver5-status/`, 런타임은 `assets/graphics/status/`다.
- 수제 픽셀 원본은 `../tools/spritetool/assets/`의 `ver5-projectiles/`(돌·그림자·흙먼지), `ver5-fx/`(책략·전투·피격·장면 효과, 일기토 섬광, 입자), `ver5-outside/`(맵 밖 소나무·돌), `ver5-ui/`(창 테두리, 표시·아이콘, HP바 부품, 두루마리 무늬)다. 스프라이트는 레이어 묶음이고, 레이어는 아스키 격자(`pixels.json`, 팔레트 색) 또는 `gen.py` 함수가 그리는 부드러운 반투명 그림이다. 레이어마다 투명도(하나 또는 프레임별)와 합성(`normal`/`lighter`)이 있어 빌더가 보통 시트와 빛 시트(`_light.png`, 게임이 더하기 합성으로 겹침)로 나눠 굽는다. 각 원본의 `seed.py`는 격자 첫 판을 뽑는 도구(공용 도구 `tools/pixelgrid.py`)로, 다시 돌리면 손질이 지워진다. 런타임은 `Assets.Projectiles`·`FX`·`Outside`·`UI`(`internal/gui/shots.go`).
- 물·불·땅 책략 열다섯(탁류~지룡)의 모양은 손 그림이 아니라 `tools/spellsim.py`의 시뮬레이션이다. 물은 유체(SPH), 불은 열기로 떠오르며 식는 입자, 땅은 쪼개지는 바위 복셀과 흙먼지를 전투맵 공간에서 굴려 원소마다 다른 방식으로 아이소 도트를 찍는다. `--bake`가 프레임마다 그린 부분만 잘라 `effects/spells` 한 시트에 담고(프레임마다 기준점 `anchors`), 효과가 시작된 뒤 몇 초에 맞는지(`hit`)를 적는다. 물·불·땅 열다섯은 제 속도로는 너무 빨리 끝나서 2배 슬로모션(`ELEMENT_SLOW`, 프레임 두 배·`hit`도 두 배)으로 굽는다. 미리보기도 `--slow 2`를 줘야 게임과 같은 길이다. 게임은 시트를 `FX`에 합쳐 `spell_<기술 id>`가 있는 기술을 그 효과 하나로 범위 전체에 그리고, 피해는 `hit`까지 미룬다(`effects.go`의 `drawSpell`, `player.go`의 `swing`). 상태 책략 여덟(팔진도·혼란·대혼란·독연·독무·봉책·봉쇄·봉격)도 같은 시트로, 원본은 `tools/spellsim_gas.py`(안개·독)·`spellsim_seal.py`(부적·사슬)·`spellsim_formation.py`(팔진도)이고 `spellsim.py`의 `baked()`가 고른 안을 모은다. 팔진도의 진형(`spell_eightfold`)은 유닛 아래 땅 층(`groundMarks`)에 깔고, 닿은 유닛마다 같은 편이면 `_ally`, 아니면 `_enemy` 조각을 얹는다. 폭염은 일렬 첫 칸에서 그리며 `spell_inferno`(오른쪽 아래)·`spell_inferno_back`(왼쪽 위)을 좌우로 뒤집어 네 방향을 덮는다. 미리보기는 같은 도구를 이름만 주고 돌리면 `reports/spellsim/<이름>.gif`로 나온다. 만드는 방법은 `.claude/skills/isofx`.
- UI 그림은 그림 4px이 UI 1px이고 부드럽게 축소한다(아이콘·삼각 표시, 이웃 그림이 섞이지 않게 둘레 8px 투명 여백). 단, 창 테두리는 그림 1px이 UI 1px이고 최근접으로 9조각 늘려 단색 사각형처럼 또렷하다. HP바 부품은 게임 화소로 맵 위에 그린다.
- `cmd/_uidemo-vm`은 기본 크기(1280×850)에서 UI를 절반 해상도 버퍼에 그린 뒤 2배로 키운다. 1px 선을 비교할 때는 `-size 2560x1700`으로 찍는다.
- 거점 그림은 `../tools/unit3d/sites.py`가 `assets/graphics/map/sites.png`로 굽는다(게임 칸 하나에 한 장, 관문은 성벽 줄 방향별 두 장). 마을·주둔지는 `gen_pixel_kit.py`의 같은 이름 그림(`extra`)이 있으면 그것을 그린다. 마을은 네 판(`village_0..3`)을 칸 좌표 (x+2y) mod 4로 골라 이웃 칸과 겹치지 않게 하고, 판마다 굴뚝 연기 자리가 `scenery.json`의 `village_<n>`에 있다(`gen_scenery.py`가 `gen_pixel_kit.smoke_points`에서 읽음). 주둔지 깃발 천은 아직 sites.png 주둔지에서 오려 낸다(`gen_scenery.py camp_parts`).

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
- 앱 아이콘은 `android/res/mipmap-*/ic_launcher.png`(48~192px)다. 데스크톱 창 아이콘은 `assets/graphics/ui/app_icon.png`(256px)이고 Windows·Linux에서만 보인다(macOS 창에는 아이콘이 없다). 두 곳 모두 같은 그림을 줄인 것이다.
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
