"""Bundle the game's sheets, the candidates and the scene data into one self-contained viewer.html"""
import base64, json, glob
from common import *


def uri(path):
    return "data:image/png;base64," + base64.b64encode(open(path, "rb").read()).decode()


imgs = {}
for n in ("tileset", "structures", "props"):
    imgs[n] = uri(os.path.join(GAME, "map", n + ".png"))
tiles = json.load(open(out("tiles", "tiles.json")))["sprites"]
for n, s in tiles.items():
    imgs["c:" + n] = uri(out(s["file"]))
fx = json.load(open(out("effects", "effects.json")))
for k, v in fx.items():
    imgs["fx:" + k] = uri(out(v["file"]))
imgs["sites"] = uri(os.path.join(GAME, "map", "sites.png"))
units = {}
for f in glob.glob(out("units", "*.json")):
    u = json.load(open(f))
    units[u["id"]] = u
    imgs["u:" + u["id"]] = uri(out("units", u["id"] + ".png"))
data = dict(
    imgs=imgs,
    tileset=INDEX["tileset"]["sprites"],
    water=INDEX["tileset"]["animations"]["water"],
    anims={k: v for k, v in INDEX["tileset"]["animations"].items() if k.startswith(("grass", "tree", "rock"))},
    structures=INDEX["structures"], props=INDEX["props"]["sprites"], cand=tiles, units=units,
    fx=fx, sites=INDEX["sites"],
)
tpl = open(out("tools", "viewer_template.html"), encoding="utf-8").read()
html = tpl.replace("/*DATA*/", "const DATA=" + json.dumps(data, separators=(",", ":")) + ";")
open(out("viewer.html"), "w", encoding="utf-8").write(html)
print("viewer.html", len(html) // 1024, "KB")
