# 나무 흔들림 원본

`tree1.png`–`tree3.png`는 Makefile이 읽는 4프레임 원본이다. 순서는 기본 → 오른쪽으로 살짝 →
거의 기본 → 왼쪽으로 살짝이며, 바람은 강풍이 아니라 잎 덩어리가 바스락거리는 정도다.

| 파일 | 역할 |
|---|---|
| `treeN-generated.png` | Codex 이미지 생성 결과(마젠타 배경, 1774×887) |
| `treeN.png` | `tools/prepare_tree_sway.py`로 배경 제거·뿌리 기준 정렬·줄기 고정한 투명 원본 |

```sh
python3 tools/prepare_tree_sway.py tools/spritetool/assets/tree-sway/tree1-generated.png \
    tools/spritetool/assets/tree-sway/tree1.png
make objects
python3 ver3/tools/build_assets.py   # ver3 tileset.png·assets.yaml 갱신
```

정렬 스크립트는 프레임마다 아래 6% 줄의 중심을 뿌리 기준점으로 맞추고, 모든 프레임에서 잎이 없는
아래쪽 줄(줄기·뿌리)을 프레임 0에서 복사한다. 게임 해상도에서 줄기·뿌리 픽셀은 네 프레임이 같다.

## 프롬프트

참조 이미지: `../treeN.png`(정지 나무). `tree2`는 "broad round", `tree3`은 "tall"로 바꿔 사용했다.

```text
A sprite sheet of exactly four animation frames of the SAME deciduous tree shown in the reference
image, arranged in one horizontal row, evenly spaced, on a 1536x768 canvas. Keep the reference's
exact art style, olive/yellow-green leaf palette, round layered foliage clumps, upper-left lighting,
brown forked trunk and roots, proportions and size. Gentle breeze animation, NOT strong wind:
frame 1 neutral (identical to reference), frame 2 the upper crown leans very slightly to the right
and a few leaf clumps on the right rim lift and flutter, frame 3 settled back near neutral with leaf
highlights shifted subtly, frame 4 the upper crown leans very slightly to the left with left rim
clumps fluttering. The trunk, roots and lower canopy stay pixel-identical and in the same position
in all four frames; only foliage clusters move a tiny amount. Individual leaf clumps should visibly
rustle (shape and highlight changes), not the whole tree sliding. Flat solid magenta #FF00FF
background everywhere (for chroma key), no ground, no shadow, no outline added, no text, no
watermark, safe margins between frames.
```
