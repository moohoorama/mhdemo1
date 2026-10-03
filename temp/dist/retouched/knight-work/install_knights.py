from pathlib import Path
import shutil

stage = Path(__file__).resolve().parents[1]
root = stage.parents[2]
files = {
    stage / 'knight-work/build_knights.py': root / 'tools/build_knights.py',
    stage / 'knight-work/knight-frames.json': root / 'tools/spritetool/assets/knight-frames.json',
    stage / 'knight-preview/assets/knights.png': root / 'assets/knights.png',
    stage / 'knight-preview/assets/knights.json': root / 'assets/knights.json',
    stage / 'knight-preview/app.js': root / 'knight-preview.js',
}
html_target = root / 'knight-preview.html'
for target in [*files.values(), html_target]:
    if target.exists():
        raise SystemExit(f'Refusing to overwrite existing file: {target}')
for source, target in files.items():
    shutil.copy2(source, target)
html_target.write_text((stage / 'knight-preview/index.html').read_text().replace('src="app.js"', 'src="knight-preview.js"'))
print(f'Installed 160 frames and preview: {html_target}')
