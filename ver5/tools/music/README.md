# 삼국지 여섯 테마

## 제작 설계

궁·상·우 세 오음 선법을 바탕으로 서로 다른 여섯 곡을 작곡한다. 선율은
직접 작성한 8마디 악구를 도입·응답·대조·재현으로 전개하고, 단순한 음계
나열 대신 중심음, 종지, 쉼표와 악기별 역할로 각 분위기를 구별한다.
모든 유음 악기는 C·D·E·G·A 음군 안에서 연주한다. 궁의 중심은 C, 상은 D,
우는 A이며 템포와 타악 편성을 따로 둔다.

| 파일 | 테마 / 곡명 | 선법 | 편성 방향 |
| --- | --- | --- | --- |
| 01-village | 마을과 정겨움 / 버들 아래 장터 | 궁 | 가벼운 피리, 뜯는 현, 낮은 현 |
| 02-court | 관청과 높은 사람들 / 높은 처마의 조회 | 궁 | 느린 관악, 종, 절제한 북 |
| 03-intrigue | 음모를 꾸미는 사람들 / 발 너머의 밀담 | 우 | 낮은 현, 짧은 발현, 드문 종 |
| 04-battle | 전투 / 진군하는 깃발 | 상 | 힘 있는 관악, 현의 반복, 군고 |
| 05-urgency | 다급한 상황 / 봉화가 오르는 밤 | 우 | 빠른 발현, 짧은 관악, 긴박한 북 |
| 06-victory | 성취와 환희 / 돌아오는 깃발 | 궁 | 밝은 관악과 발현 합주, 종과 북 |

## 산출물과 범위

- `compose.py`: Python 표준 라이브러리로 MIDI Type 1 악보를 생성하고
  FluidSynth와 로컬 GM SoundFont로 렌더링, FFmpeg로 MP3를 만든다.
- `player.html`: 감상 페이지 원본. 외부 라이브러리·폰트·네트워크 요청 없이
  재생/일시정지/탐색, 곡 반복, 전체 연속 재생, 음량, 개별 저장을 제공한다.
- `../../assets/music/themes/`: 여섯 곡의 `.mid`, `.mp3`, 전체 MIDI ZIP.
- `../../docs/music-themes.html`: MP3·MIDI·ZIP을 내장한 독립 HTML.
  이 파일 하나만 복사해도 오프라인으로 감상하고 저장할 수 있다.

GM 악기로 전통 악기의 역할을 근사한 MIDI풍 게임 음악이다. 특정 중국
전통 악기의 실연이나 삼국시대 음악의 복원으로 표기하지 않는다.
게임 엔진에는 아직 연결하지 않으며 기존 코드·음악 재생에는 영향을 주지 않는다.

## 검증 계획

MIDI 음높이·길이·이벤트 짝을 검사하고 여섯 MP3의 길이·음량·클리핑을
확인한다. 브라우저에서 여섯 곡 재생, 곡 전환, 일시정지, 반복·연속 재생
선택과 다운로드, 좁은 화면 배치를 확인하고 실제 화면을 캡처한다.

## 재생성

```sh
python3 ver5/tools/music/compose.py --soundfont /path/to/soundfont.sf2
```

`fluidsynth`와 `ffmpeg`가 PATH에 있어야 한다. SoundFont는 감상 파일에
샘플 음원으로 포함되므로 배포할 때 선택한 SoundFont의 라이선스를 따른다.
사용한 SoundFont 및 제작·검증 결과는 아래에 기록한다.

## 사용한 음색

- TimGM6mb, Debian `timgm6mb-soundfont` 1.3-5.
- SoundFont 저작자: Tim Brechbill (2004), David Bolton (2010), GPL-2.0.
- 입수: https://deb.debian.org/debian/pool/main/t/timgm6mb-soundfont/timgm6mb-soundfont_1.3-5_all.deb
- 원본 이력: https://github.com/musescore/musescore-old/commit/90c33ef9d87b3f5ff92efd3b07d89eb455fb1fef
- SoundFont 자체는 저장소에 추가하지 않았다. 감상용 MP3는 이 음색으로 MIDI를
  렌더링했다. 악보에는 GM 프로그램 번호가 있어 다른 GM SoundFont로도 연주할 수 있다.
- 길이: 마을 1:18, 관청 1:28, 음모 1:21, 전투 0:59, 다급함 0:49, 환희 1:07.
- MIDI Type 1, 480 PPQ, 4/4, 각 32마디. 7개 악기 트랙과 템포 트랙.
- MP3: 44.1 kHz, 스테레오, 128 kbps. 반복은 감상용 파일 전체를 다시
  재생하므로 짧은 종지·잔향을 포함하며 게임용 무봉합 루프 마스터는 아니다.

## 제작 검증

- 독립 MIDI 파서로 여섯 파일의 헤더·트랙 수·32마디 길이·노트 on/off 짝·
  중복 발음·오음 음군·각 선법 중심음으로의 종지를 검사했다.
- FFprobe/FFmpeg로 여섯 MP3의 디코딩·길이·음량·클리핑·3초 이상 무음을 검사했다.
- Chrome에서 여섯 곡 실제 재생·전환, 일시정지·정지·음량, 자동 다음 곡,
  반복 경계, MIDI/MP3/ZIP 저장 후 원본 바이트 일치, 파일 직접 열기와
  390px 화면의 가로 넘침을 검사했다. JavaScript 오류는 없었다.
- 앱 내 Browser 연결이 도구 메타데이터 오류로 시작되지 않아 기존 Playwright와
  별도 headless Chrome으로 검증했다. 캡처: `../../reports/music/desktop.png`,
  `../../reports/music/mobile.png`.

## 십면매복 (전투)

`assets/music/themes/sipmyeon-maebok.mid`는 사용자가 준 musics.zip의 MIDI 스케치다. 전투 곡
`sipmyeon-maebok.mp3`는 이 MIDI를 `sipmyeon.py`로 다시 편성해(선율 고토, 받침 현 피치카토, 첼로 베이스,
태고+팀파니) GeneralUser GS 음색으로 렌더링하고 -15.8 LUFS로 맞춘 것이다(2026-11-09, 시안 B).

```sh
python3 ver5/tools/music/sipmyeon.py /path/to/GeneralUser-GS.sf2 ver5/assets/music/themes/sipmyeon-maebok.mp3
```

GeneralUser GS: https://github.com/mrbumpy409/GeneralUser-GS (S. Christian Collins, 어떤 용도로든 무료). 저장소에는
넣지 않았다. `smf.py`는 표준 라이브러리만 쓰는 최소 MIDI 읽기·쓰기다.
