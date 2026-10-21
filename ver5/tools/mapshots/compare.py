#!/usr/bin/env python3
"""Build reports/scenery/compare.html from the before/ and after/ pictures tools/mapshots/shots.py made.

  python3 tools/mapshots/compare.py

Writes the page only; it does not open it. Every shot is a slider (drag to wipe between before and after);
clicking a picture opens the original. The asset previews of assets_candidate/scenery are copied beside it."""
import html
import shutil
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
REPORT = ROOT / 'reports/scenery'
PREVIEWS = ROOT.parent / 'assets_candidate/scenery'

KINDS = [
    ('overview', '전체 (배율 1)'),
    ('grass', '풀밭'),
    ('forest', '숲'),
    ('waste', '황무지'),
    ('cliff', '절벽 윗면'),
    ('camp', '주둔지 (깃발)'),
    ('village', '마을 (연기)'),
    ('keep', '성 (깃발)'),
    ('castle', '성내'),
    ('water', '물가'),
    ('wind1', '숲 가까이 (배율 3)'),
]
STAGE_NAMES = {1: '1장', 2: '2장', 3: '3장', 4: '4장', 5: '5장', 6: '6장', 7: '7장'}

CSS = """
:root{--bg:#14181a;--panel:#1d2326;--ink:#e6ebe8;--mute:#9aa7a2;--line:#2f3a3d;--acc:#b7d26a}
@media (prefers-color-scheme:light){:root{--bg:#f4f5f2;--panel:#fff;--ink:#1c2420;--mute:#5d6b65;--line:#d6dbd5;--acc:#4f7a1a}}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--ink);font:15px/1.6 -apple-system,'Apple SD Gothic Neo','Noto Sans KR',sans-serif}
main{max-width:1180px;margin:0 auto;padding:28px 16px 80px}
h1{font-size:26px;margin:0 0 4px}h2{font-size:20px;margin:44px 0 12px;padding-top:8px;border-top:1px solid var(--line)}
h3{font-size:15px;margin:22px 0 6px;color:var(--mute);font-weight:600}
p,li{color:var(--mute)}b{color:var(--ink)}
nav{display:flex;flex-wrap:wrap;gap:8px;margin:16px 0}
nav a{color:var(--acc);text-decoration:none;border:1px solid var(--line);border-radius:6px;padding:2px 10px}
.cmp{position:relative;background:#000;border:1px solid var(--line);border-radius:6px;overflow:hidden;line-height:0;user-select:none}
.cmp img{width:100%;height:auto;display:block;image-rendering:pixelated}
.cmp .top{position:absolute;inset:0;clip-path:inset(0 0 0 var(--cut,50%))}
.cmp .tag{position:absolute;top:8px;padding:2px 8px;border-radius:4px;background:rgba(0,0,0,.6);color:#fff;font:600 12px/1.6 sans-serif}
.cmp .tag.l{left:8px}.cmp .tag.r{right:8px}
.cmp .bar{position:absolute;top:0;bottom:0;width:2px;background:#fff;left:var(--cut,50%);pointer-events:none;box-shadow:0 0 4px #000}
.cmp input{position:absolute;inset:0;width:100%;height:100%;opacity:0;cursor:ew-resize;margin:0}
.pair{display:grid;grid-template-columns:1fr 1fr;gap:10px}
.pair figure{margin:0}.pair img{width:100%;height:auto;display:block;border:1px solid var(--line);border-radius:6px;image-rendering:pixelated}
figcaption{font-size:13px;color:var(--mute);margin:4px 0 0}
.assets img{max-width:100%;image-rendering:pixelated;border:1px solid var(--line);border-radius:6px;margin:4px 0}
code{background:var(--panel);padding:1px 5px;border-radius:4px;color:var(--ink)}
@media(max-width:700px){.pair{grid-template-columns:1fr}}
"""

JS = """
document.querySelectorAll('.cmp').forEach(c=>{
  const r=c.querySelector('input');
  const set=()=>c.style.setProperty('--cut',r.value+'%');
  r.addEventListener('input',set);set();
});
"""


def slider(stage, kind):
    name = f'B{stage:02d}_{kind}.png'
    if not (REPORT / 'before' / name).exists() or not (REPORT / 'after' / name).exists():
        return ''
    return (f'<div class="cmp"><img src="before/{name}" alt="before"><a class="top" href="after/{name}" target="_blank">'
            f'<img src="after/{name}" alt="after"></a><span class="tag l">이전</span><span class="tag r">이후</span>'
            f'<span class="bar"></span><input type="range" min="0" max="100" value="50" aria-label="이전과 이후 비교"></div>')


def main():
    for stub in ('pines', 'grass', 'waste', 'floor', 'forest'):
        src = PREVIEWS / f'preview_{stub}.png'
        if src.exists():
            (REPORT / 'assets').mkdir(exist_ok=True)
            shutil.copy(src, REPORT / 'assets' / src.name)
    stages = sorted({int(p.name[1:3]) for p in (REPORT / 'after').glob('B*.png')})
    out = ['<!doctype html><html lang="ko"><head><meta charset="utf-8">',
           '<meta name="viewport" content="width=device-width,initial-scale=1"><title>전투맵 장식 전후 비교</title>',
           f'<style>{CSS}</style></head><body><main>',
           '<h1>전투맵 배경·숲 전후 비교</h1>',
           '<p>같은 맵, 같은 카메라를 실제 게임 렌더러(<code>cmd/_uidemo-vm</code>, 창 없음)로 찍었습니다. '
           '그림을 좌우로 드래그하면 <b>이전</b>(왼쪽)과 <b>이후</b>(오른쪽)가 갈립니다. 클릭하면 원본 크기로 엽니다.</p>',
           '<ul><li><b>땅 타일은 그대로</b>, 그 위에 외곽선 없는 부드러운 장식(풀포기·꽃·자갈·흙과 풀 톤 얼룩 / 황무지: 마른 풀·갈라짐·돌 / 성내: 금·이끼·얼룩)을 얹었습니다.</li>',
           '<li><b>숲</b>은 한 칸에 같은 소나무 묶음 하나가 아니라, 기존 소나무 한 종을 근간으로 만든 10가지 변형(키·너비·단 수·밝은 어린 나무·고사목)을 칸당 4~8그루 섞어 놓았습니다. 외곽선과 팔레트는 그대로입니다.</li>',
           '<li><b>물</b>: 물가 갈대와 거품 줄기, 연잎, 깊이 얼룩, 반짝임과 잔물결(움직임). <b>절벽 윗면</b>: 금·돌·마른 덤불·얼룩(블록과 같은 순서로 그려 깊이가 맞음).</li>',
           '<li><b>전투 영역 밖</b>은 바닥뿐 아니라 나무·절벽·성벽·구조물도 같이 어두워지고, 멀수록 더 어두워집니다(발 밑 칸이 영역 밖인 것만, 통째로).</li>',
           '<li><b>구조물 생기</b>: 주둔지 깃발이 바람 위상으로 펄럭이고, 마을 지붕에서 연기가 오르며, 성 망루에 깃발이 선다.</li>',
           '<li>나무마다 <b>잎 흔들림 4프레임</b>, 위치에 따라 어긋난 위상과 시간 지연, 오른쪽 아래로 떨어지는 <b>그림자</b>, 숲 바닥의 <b>덤불·낙엽·고사리·그늘</b>.</li></ul>',
           '<nav>' + ''.join(f'<a href="#s{n}">{STAGE_NAMES.get(n, n)}</a>' for n in stages) + '<a href="#sway">흔들림</a><a href="#assets">소재</a></nav>']
    for n in stages:
        out.append(f'<h2 id="s{n}">{STAGE_NAMES.get(n, n)} (B{n:02d})</h2>')
        for kind, label in KINDS:
            s = slider(n, kind)
            if s:
                out.append(f'<h3>{label}</h3>{s}')
    out.append('<h2 id="sway">움직임 (3초 반복, 60fps 녹화를 20fps로)</h2>')
    for kind, title, note in (('sway', '숲 잎 흔들림', '나무마다 위상이 달라 바람이 지나가는 것처럼'),
                              ('camp', '주둔지 깃발', '깃발이 바람 위상에 맞춰 펄럭임'),
                              ('village', '마을 연기', '일부 칸의 지붕에서 연기가 피어올라 오른쪽으로 흩어짐')):
        for n in stages:
            b, a = REPORT / 'before' / f'B{n:02d}_{kind}.gif', REPORT / 'after' / f'B{n:02d}_{kind}.gif'
            if a.exists() and b.exists():
                out.append(f'<h3>{title} (B{n:02d})</h3><div class="pair"><figure><img src="before/{b.name}" alt="이전"><figcaption>이전: 정지</figcaption></figure>'
                           f'<figure><img src="after/{a.name}" alt="이후"><figcaption>이후: {note}</figcaption></figure></div>')
    out.append('<h2 id="assets">소재 미리보기 (5배 확대)</h2><div class="assets">')
    for stub, label in (('pines', '소나무 10종 × 흔들림 4프레임'), ('grass', '풀밭 장식'), ('waste', '황무지 장식'),
                        ('floor', '성내 장식'), ('forest', '숲 바닥 장식')):
        if (REPORT / 'assets' / f'preview_{stub}.png').exists():
            out.append(f'<h3>{html.escape(label)}</h3><img src="assets/preview_{stub}.png" alt="{html.escape(label)}">')
    out.append('</div></main><script>' + JS + '</script></body></html>')
    page = REPORT / 'compare.html'
    page.write_text('\n'.join(out), encoding='utf-8')
    print(page)


if __name__ == '__main__':
    main()
