"""Regenerate every candidate sprite and the manifest in one process (the manifest is shared), then the viewer."""
import json
import gen_all
import gen_interior  # registers brazier and map_table on import
import gen_gate
import gen_scene_props
import gen_3d_props
import gen_3d_castle
import gen_cover
from common import out

gen_gate.run()
gen_scene_props.run()
gen_3d_props.run()
gen_3d_castle.run()
gen_cover.run()
json.dump(gen_all.manifest, open(out("tiles", "tiles.json"), "w"), indent=1)
print(len(gen_all.manifest["sprites"]), "sprites")
import build_viewer  # noqa: F401  (writes viewer.html)
