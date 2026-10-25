# 중국 전통 문양 타일 시안

## 제작 계획

- 목적: 기존 먹빛 청록 용무늬 이미지와 어울리는 은은한 반복 배경 3종을 비교한다.
- 결과물: 여의운문, 뇌문, 수자문 생성 원본과 반복용 PNG 각 1장, `preview.html`.
- 화풍: 어두운 청록 바탕, 저대비 수묵·직물 질감. 글자나 테두리 없이 정사각형으로 만든다.
- 확인: `preview.html`에서 각 이미지를 가로세로 3회 이상 반복하고 경계와 밀도를 확인한다.
- 범위: 게임 런타임 자산에는 연결하지 않고 시안으로 둔다.

## 결과와 재생성

- `ruyi-cloud.png`, `leiwen.png`, `shou.png`: 내장 imagegen으로 만든 1254×1254 원본.
- `*-tile.png`: 반복용 이미지. 여의운문·뇌문은 양쪽 가장자리 32픽셀을 섞어 색을 맞춘다. 수자문은 반복 주기 314×315픽셀을 거울식으로 배열한다.
- 최종 대비: 여의운문·수자문은 이전 시안의 80%(생성 원본 편차의 56%), 뇌문은 이전과 같은 17%로 둔다. 뇌문 평균색은 먹빛 청록으로 맞춘다. 반복용 PNG의 흑백 명암 표준편차는 여의운문 0.0273, 뇌문 0.0149, 수자문 0.0208이다.
- 반복용 PNG 재생성: 이 폴더에서 `go run build_tiles.go`.
- HTML은 이미지 파일과 같은 폴더에 둔 채 `preview.html`을 열면 된다. 문양 전환, 반복 크기, 원본 비교, 경계선 표시, PNG 저장을 지원한다.
- 대비 슬라이더는 현재 반복용 PNG를 100%로 하여 문양별로 미리보기만 조절한다. 선택한 값은 브라우저에 저장하고 화면에 세 값을 함께 보여준다. 값을 확정하면 `build_tiles.go`에 반영해 PNG를 다시 만든다.

## 생성 프롬프트 (내장 imagegen)

### 여의운문

> Use case: stylized-concept. Asset type: seamless repeating square wallpaper texture for a historical Chinese game UI, first of a coherent three-pattern set. Create a refined, understated traditional Chinese ruyi cloud (如意云纹) pattern only: flowing curled cloud lobes, varied scale and spacing, delicate ink-wash and silk-brocade texture. Palette: very dark charcoal teal and muted grey jade, extremely low contrast like embossed damask, closely matching the subdued mood of a dark teal Chinese dragon-and-cloud brocade background. Important: this is a true TWO-DIMENSIONAL SEAMLESS REPEAT TILE; left and right boundaries must join exactly, and top and bottom boundaries must join exactly, with no border, framing, vignetting, central medallion, isolated hero motif, text, dragon, watermark, or bright gold. Flat straight-on surface, square composition, fine but legible detail.

### 뇌문

> Use case: stylized-concept. Asset type: seamless repeating square wallpaper texture for a historical Chinese game UI, second of a coherent three-pattern set. Create an authentic ancient Chinese leiwen (雷纹) ground pattern inspired by Shang and Zhou bronze vessels: fine connected squared spirals and angular meanders, shallow engraved relief with irregular aged bronze and ink-wash silk texture. Repeated geometry should feel handmade, sophisticated, quiet, not a modern maze. Palette: very dark charcoal teal and muted grey jade, low contrast, matching a subdued dark teal Chinese dragon-and-cloud brocade background. Important: true TWO-DIMENSIONAL SEAMLESS REPEAT TILE; left and right boundaries join exactly, top and bottom boundaries join exactly. Full bleed flat straight-on texture, no border, no vignette, no central emblem, no clouds, no dragon, no text, no watermark, no bright metallic colors.

### 수자문

> Use case: stylized-concept. Asset type: seamless repeating square wallpaper texture for a historical Chinese game UI, third of a coherent three-pattern set. Create an understated traditional Chinese longevity brocade pattern: small correctly drawn 壽 (shou, longevity) seal-script medallions, widely spaced in a staggered grid, connected by fine restrained cloud-scroll and silk-woven ornament. The medallions must be secondary and subtle, leaving ample dark negative space. Historical textile feeling, brush-ink and aged damask texture, flat straight-on surface. Palette: very dark charcoal teal and muted grey jade, very low contrast, matching a subdued dark teal Chinese dragon-and-cloud brocade background. Important: true TWO-DIMENSIONAL SEAMLESS REPEAT TILE; left and right boundaries join exactly and top and bottom boundaries join exactly. No border, no vignette, no large central medallion, no dragon, no other text, no watermark, no bright gold.
