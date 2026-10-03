"""Bundle every art candidate into one self-contained comparison page."""
import base64
import json
import pathlib

ROOT = pathlib.Path(__file__).resolve().parent
NAMES = {
    "A": "영걸전 클래식",
    "B": "조조전 정밀",
    "C": "아이소메트릭",
    "C2": "아이소메트릭 + Codex 원화",
    "D": "수묵 UI + 고해상도 도트",
}


def data_uri(path):
    return "data:image/png;base64," + base64.b64encode(path.read_bytes()).decode()


def load(cid):
    out = ROOT / cid / "out"
    readme = (ROOT / cid / "README.md").read_text(encoding="utf-8")
    return {
        "id": cid,
        "name": NAMES[cid],
        "screen": data_uri(out / "screen.png"),
        "sheet": data_uri(out / "sheet.png"),
        "sheetMeta": json.loads((out / "sheet.json").read_text()),
        "portrait": data_uri(out / "portrait.png"),
        "portraitMeta": json.loads((out / "portrait.json").read_text()),
        "readme": readme,
        "gallery": [data_uri(p) for p in sorted((out / "portraits").glob("*.png"))] if (out / "portraits").is_dir() else [],
    }


TEMPLATE = r"""<!doctype html>
<html lang="ko"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>SRPG 그래픽 후보</title>
<style>
:root{--bg:#f4f1ea;--fg:#1d2430;--muted:#5d6675;--card:#fffdf8;--line:#d9d2c3;--accent:#a8781e}
@media (prefers-color-scheme:dark){:root:not([data-theme="light"]){--bg:#141922;--fg:#e8e4da;--muted:#9aa3b2;--card:#1c2330;--line:#2e3747;--accent:#d9a640}}
:root[data-theme="dark"]{--bg:#141922;--fg:#e8e4da;--muted:#9aa3b2;--card:#1c2330;--line:#2e3747;--accent:#d9a640}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--fg);font:15px/1.6 -apple-system,"Apple SD Gothic Neo","Noto Sans KR",sans-serif}
main{max-width:1340px;margin:0 auto;padding:24px 16px 64px}
h1{font-size:26px;margin:0 0 4px}
.lead{color:var(--muted);margin:0 0 20px}
nav{display:flex;gap:8px;flex-wrap:wrap;margin-bottom:20px;position:sticky;top:0;background:var(--bg);padding:8px 0;z-index:2}
nav button{border:1px solid var(--line);background:var(--card);color:var(--fg);padding:8px 14px;border-radius:8px;cursor:pointer;font:inherit}
nav button.on{border-color:var(--accent);box-shadow:inset 0 -3px 0 var(--accent)}
section{background:var(--card);border:1px solid var(--line);border-radius:12px;padding:16px;margin-bottom:28px}
section h2{margin:0 0 12px;font-size:20px}
.screen{width:100%;max-width:1280px;image-rendering:pixelated;display:block;border-radius:6px;border:1px solid var(--line)}
.row{display:flex;gap:24px;flex-wrap:wrap;margin-top:16px;align-items:flex-start}
.anim{display:flex;flex-direction:column;gap:6px;align-items:center}
.anim canvas{image-rendering:pixelated;background:repeating-conic-gradient(#8883 0 25%,#0000 0 50%) 0 0/16px 16px;border:1px solid var(--line);border-radius:4px}
.anim small{color:var(--muted)}
.portrait img{image-rendering:pixelated;border:1px solid var(--line);border-radius:4px}
details{margin-top:12px}
pre{white-space:pre-wrap;font:13px/1.55 ui-monospace,Menlo,monospace;background:var(--bg);padding:12px;border-radius:8px;overflow-x:auto}
.tools{display:flex;gap:12px;align-items:center;flex-wrap:wrap;color:var(--muted);font-size:13px;margin-top:8px}
.screen-wrap{overflow-x:auto}
</style></head><body><main>
<h1>SRPG 그래픽 후보 4종</h1>
<p class="lead">실제 크기 1280×850 전장 화면과 8프레임 대기·걷기 반복 재생. 체크무늬는 미리보기 배경이며 원화는 실제 알파 투명입니다. 하나를 고르거나, 섞을 요소(시점·밀도·UI·팔레트)를 알려 주세요.</p>
<nav id="nav"></nav>
<div id="list"></div>
<div class="tools"><label><input type="checkbox" id="anchor" checked> 기준점 표시</label><label>배속 <select id="speed"><option value="0.5">0.5×</option><option value="1" selected>1×</option><option value="2">2×</option></select></label></div>
</main>
<script>
const C = __DATA__;
const list = document.getElementById('list'), nav = document.getElementById('nav');
const players = [];
for (const c of C) {
  const b = document.createElement('button'); b.textContent = c.id + ' · ' + c.name;
  b.onclick = () => document.getElementById('c' + c.id).scrollIntoView({behavior:'smooth'});
  nav.appendChild(b);
  const s = document.createElement('section'); s.id = 'c' + c.id;
  s.innerHTML = `<h2>${c.id} · ${c.name}</h2><div class="screen-wrap"><img class="screen" src="${c.screen}" alt="${c.name} 전장 화면"></div><div class="row"></div><details><summary>설정과 자체 평가 (README)</summary><pre></pre></details>`;
  s.querySelector('pre').textContent = c.readme;
  const row = s.querySelector('.row');
  const img = new Image(); img.src = c.sheet;
  const m = c.sheetMeta, k = Math.max(2, Math.round(96 / m.cell[1]));
  for (const sp of m.sprites) for (const an of sp.anims) {
    const w = document.createElement('div'); w.className = 'anim';
    const cv = document.createElement('canvas'); cv.width = m.cell[0] * k; cv.height = m.cell[1] * k;
    w.appendChild(cv); const lb = document.createElement('small'); lb.textContent = `${sp.id} · ${an.name} ${an.frames}f`; w.appendChild(lb);
    row.appendChild(w);
    players.push({cv, img, m, sp, an, k});
  }
  const p = document.createElement('div'); p.className = 'anim portrait';
  const pk = Math.max(1, Math.round(160 / c.portraitMeta.size[1]));
  p.innerHTML = `<img src="${c.portrait}" width="${c.portraitMeta.size[0]*pk}" height="${c.portraitMeta.size[1]*pk}" alt="관우 초상"><small>관우 초상 ${c.portraitMeta.size.join('×')}</small>`;
  row.appendChild(p);
  for (const g of c.gallery) { const e = document.createElement('div'); e.className = 'anim portrait'; e.innerHTML = `<img src="${g}" width="192" height="192" alt="초상">`; row.appendChild(e); }
  list.appendChild(s);
}
const anchorBox = document.getElementById('anchor'), speedSel = document.getElementById('speed');
function tick(t) {
  const sp = +speedSel.value;
  for (const P of players) {
    if (!P.img.complete) continue;
    const g = P.cv.getContext('2d'); g.imageSmoothingEnabled = false;
    g.clearRect(0, 0, P.cv.width, P.cv.height);
    const f = P.an.start + Math.floor(t * sp / P.an.ms) % P.an.frames;
    const [cw, ch] = P.m.cell;
    g.drawImage(P.img, f * cw, P.sp.row * ch, cw, ch, 0, 0, cw * P.k, ch * P.k);
    if (anchorBox.checked) {
      const [ax, ay] = P.m.anchor;
      g.fillStyle = '#ff2d55'; g.fillRect(0, ay * P.k, P.cv.width, 1);
      g.fillRect(ax * P.k, ay * P.k - 4, 1, 8);
    }
  }
  requestAnimationFrame(tick);
}
requestAnimationFrame(tick);
</script></body></html>
"""


def main():
    ready = [cid for cid in NAMES if (ROOT / cid / "out" / "screen.png").exists()]
    data = [load(cid) for cid in ready]
    html = TEMPLATE.replace("__DATA__", json.dumps(data, ensure_ascii=False))
    (ROOT / "index.html").write_text(html, encoding="utf-8")
    print("bundled:", ", ".join(ready))


if __name__ == "__main__":
    main()
