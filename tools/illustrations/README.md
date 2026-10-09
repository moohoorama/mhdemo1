# 일러스트

게임에 쓰는 일러스트의 원본을 모은다. 도트 스프라이트 원본은 `tools/spritetool/assets/`에 따로 있다.

| 폴더 | 내용 |
|---|---|
| `style-samples/` | 화풍 후보 8종과 비교 시트(선택: 4번 수묵 담채) |
| `portraits/` | 장수 초상 `<장수키>.png`, `sheets/`에 6인 시트 원본과 `prompts.md` |
| `events/` | 이벤트 CG (예정: 도원결의, 삼고초려, 장판교 등) |
| `backgrounds/` | 시나리오·회의장 배경 (예정: 탁현, 연합군 진영, 평원현청 등) |
| `ui/` | 타이틀, 장 제목 화면, 지도 등 (예정) |
| `portrait_sheet.sh` | 6인 초상 시트 생성 스크립트 |

원본은 크게 두고, 게임용은 줄이고 묶어 `ver4/assets/graphics/`에 둔다. 새 일러스트도 같은 화풍 기준 이미지(`style-samples/4-ink-wash.png`)를 참조로 넘기고, 실제 프롬프트를 그 폴더의 `prompts.md`에 남긴다.

## 장수 초상

대화창·편성·일기토에 쓰는 장수 초상이다. 도트 스프라이트가 아니라 일러스트이므로 Codex 이미지 생성(`codex-image` 스킬)으로 만든다.
도트 스프라이트에는 이 방식을 쓰지 않는다(`../UNIT_ART_GUIDE.md`).

## 선택한 스타일: 4번 수묵 담채 (2026-10-05 사용자 결정)

- 동양화 수묵에 옅은 채색을 더한다.
- 붓 터치가 살아 있고, 배경은 한지 질감에 옅은 산수 번짐을 둔다.
- 기준 이미지는 [`style-samples/4-ink-wash.png`](style-samples/4-ink-wash.png)다. 새 초상을 만들 때 항상 이 파일을 참조 이미지로 넘겨 화풍을 맞춘다.

## 스타일 후보 8종 (`style-samples/`)

같은 인물(유비)을 같은 공통 프롬프트로 그리고, 끝에 붙는 스타일 문장만 바꿨다. 비교 시트는 [`style-samples/sheet.png`](style-samples/sheet.png)다.

| 번호 | 파일 | 스타일 | 스타일 문장(공통 프롬프트 뒤에 붙임) |
|---|---|---|---|
| 1 | `1-painted-pixel.png` | 회화 + 도트 (기존 초상 계열) | Style: painterly realistic portrait in the tradition of classic Koei Romance of the Three Kingdoms officer portraits (rich oil-painting shading), rendered with a visible coarse pixel grid of about 128x128 effective pixels. |
| 2 | `2-sfc-pixel.png` | SFC 16비트 도트 (영걸전풍) | Style: 16-bit Super Famicom era pixel art portrait like the 1995 Koei game Sangokushi Eiketsuden, about 64x64 effective pixels, limited 16-color palette, crisp hard pixel edges, no anti-aliasing, no blur. |
| 3 | `3-hd-pixel.png` | 고해상 정밀 도트 | Style: high-detail modern pixel art portrait, about 128x128 effective pixels, careful pixel clusters and light dithering, crisp cel-shaded highlights, like HD-2D game character portraits; clean pixel grid, no blur. |
| **4** | `4-ink-wash.png` | **수묵 담채 (선택)** | Style: East Asian ink wash painting with light watercolor tints, expressive calligraphic brush strokes, soft rice-paper texture background instead of navy. |
| 5 | `5-anime-cel.png` | 애니 셀화 | Style: Japanese anime cel-shaded illustration, clean confident line art, two-tone shading, bright clean colors, like a modern Three Kingdoms mobile game character portrait. |
| 6 | `6-woodblock.png` | 목판화 (연화) | Style: traditional Chinese woodblock print (nianhua), bold carved black outlines, flat traditional mineral colors, slight paper grain, warm aged paper background. |
| 7 | `7-digital-painting.png` | 현대 디지털 페인팅 | Style: polished modern digital painting, dramatic rim lighting, detailed fabric and skin, cinematic heroic mood like a modern action game key art portrait; smooth painting, no pixels. |
| 8 | `8-sd-chibi.png` | SD 치비 | Style: super-deformed chibi illustration, large head and small shoulders, cute yet dignified, simple soft cel shading and clean outlines, matching small 2.5-heads-tall game sprites. |

**공통 프롬프트** (인물 설명 + 구도 + 배경)

```text
Liu Bei (Xuande) of the Chinese Three Kingdoms era: a kind, dignified man around thirty, long earlobes,
thin mustache and small goatee, black scholar's cap with a small gold hairpin on his topknot, green robe
with gold trim. Bust portrait from the chest up, three-quarter view facing left, calm resolute expression.
Square 1:1 image, plain dark navy background, no text, no watermark, no border.
```

- 4번(수묵)과 6번(목판화)은 배경을 각각 한지·누런 종이로 바꾸려고 공통 프롬프트에서 "plain dark navy background"를 뺐다.
- 출력은 1254×1254 PNG다.

## 만드는 방법

[`codex-image`](~/.claude/skills/codex-image/) 스킬의 스크립트를 쓴다. Codex CLI의 내장 `image_generation` 도구가 그린다. `codex login`이 필요하고, 1장에 1~3분 걸린다.

```sh
# 형식: codex-image.sh <출력.png> "<프롬프트>" [참조 이미지 ...]
~/.claude/skills/codex-image/codex-image.sh tools/illustrations/style-samples/4-ink-wash.png \
  "<공통 프롬프트> Style: East Asian ink wash painting with light watercolor tints, expressive calligraphic brush strokes, soft rice-paper texture background instead of navy."
```

- 여러 장은 각각 따로 실행해 병렬로 돌린다.
- 비교 시트는 PIL로 440px로 줄여 4열 × 2행으로 붙였다.

### 새 인물 초상을 만들 때 (수묵 담채)

1. **인물 설명**: 나이, 체격, 얼굴 특징, 수염, 머리 장식, 옷 색을 적는다. 옷과 관모는 그 인물의 도트 스프라이트·평복과 맞춘다(예: 조조 = 위 청색 도포·검은 관모·철 어깨·붉은 띠).
2. **구도**: 유비와 같게 한다. 가슴 위 흉상, 왼쪽을 향한 3/4 측면, 정사각형, 글자·워터마크·테두리 없음.
3. **화풍**: 끝에 4번 스타일 문장을 붙인다.
4. **참조 이미지**: `style-samples/4-ink-wash.png`를 넘겨 붓 터치와 한지 배경을 맞춘다.
5. **확인**: 결과를 열어 화풍, 구도, 인물 특징을 확인하고, 어긋나면 프롬프트를 고쳐 다시 만든다.

```sh
~/.claude/skills/codex-image/codex-image.sh tools/illustrations/portraits/caocao.png \
  "Cao Cao of the Chinese Three Kingdoms era: ... (인물 설명). Bust portrait from the chest up, three-quarter view facing left. Square 1:1 image, no text, no watermark, no border. Style: East Asian ink wash painting with light watercolor tints, expressive calligraphic brush strokes, soft rice-paper texture background. Match the brushwork, palette and paper of the reference image." \
  tools/illustrations/style-samples/4-ink-wash.png
```

## 진영별 6인 시트 (2026-10-05 채택)

같은 진영 6명을 한 장에 그리면 붓 터치, 색감, 배경이 저절로 맞는다. 칸 하나는 약 512px로, 대화창(140~180px)에 충분하다.

- **만드는 법**

  ```sh
  tools/illustrations/portrait_sheet.sh <시트 이름> "<여섯 칸의 인물 문장>"
  ```

  - 스크립트가 앞에 공통 문장을 붙인다: 3×2 격자, 칸마다 한 사람, 칸 사이 한지 여백, 기준 이미지와 같은 수묵 담채, 칸마다 다른 포즈·손동작·얼굴 방향과 배경, 같은 산 배경 반복 금지.
  - 끝에는 "글자·이름·번호·낙관·워터마크·테두리 없음"을 붙인다.
  - 참조 이미지로 `style-samples/4-ink-wash.png`를 넘긴다.
- **인물 문장**: "Top row, left to right: (1) … Bottom row, left to right: (4) …" 형식이다. 칸마다 다음 세 가지를 적는다.
  - **외형**: 도트 외형과 맞춘 옷 색, 관모·투구, 수염
  - **포즈**: 손동작, 표정, 얼굴 방향(왼쪽·오른쪽·정면을 섞음)
  - **배경 소재**: 인물과 어울리는 것. 예) 관우 대나무, 조조 청매(푸른 매실)와 술잔, 장각 부적과 구름
- **처음 실패한 방식**: 6명 모두에게 "흉상, 왼쪽 3/4 측면"만 지정하면 포즈와 산 배경이 전부 같아진다(`portraits/sheets/A-shu.png`).
- **실제 프롬프트**: 6장 모두 [`portraits/sheets/prompts.md`](portraits/sheets/prompts.md)에 있다.
- **결과**: 1536×1024 PNG를 `portraits/sheets/<시트>.png`에 둔다. 3×2로 나눠 가장자리를 8px 안쪽으로 잘라 `portraits/<장수키>.png`(496×496)로 저장한다.
- **묶음과 장수 키** (2026-10-05, 36명)
  - A-shu-v2 유비군: guanyu, zhangfei, jianyong, sunqian, tianyu, fangong
  - B-allies 합류·우군: gengwu, guanchun, zhangshiping, sushuang, hanying, guoshi
  - C-yuan1 원소군 1: yuanshao, wenchou, yanliang, zhanghe, tianfeng, shenpei
  - D-yuan2-lords 원소군 2 + 연합 제후: chunyuqiong, quyi, gaolan, caocao, yuanshu, gongsunzan
  - E-dong 동탁군: dongzhuo, liru, lubu, huaxiong, lijue, huzhen
  - F-gongsun-turban 공손찬군 + 황건: zhaoyun, yangang, zhangjiao, chengyuanzhi, dengmao, turban_soldier
- 전체 비교: `output/portraits-c1-all.png`
- 참조 이미지를 넘길 때 `codex exec -i`가 프롬프트까지 이미지 경로로 먹는 문제가 있었다. 스킬 스크립트에서 프롬프트 앞에 `--`를 넣어 고쳤다(2026-10-05).

## 남은 일

- 유비는 `style-samples/4-ink-wash.png`를 그대로 쓴다(포즈가 다른 새 버전이 필요하면 따로 만든다).
- 런타임 크기·자르기 규칙과 `ver4/assets/graphics`로 묶는 방식을 정한다. 지금 GUI는 `portraits.png`·`enemies.png` 4열 시트에서 잘라 쓴다.
