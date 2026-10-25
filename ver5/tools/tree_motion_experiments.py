#!/usr/bin/env python3
"""Compare changing sunlight from a normal map with one-third-strength foliage sway."""

import json
import math
import sys
from pathlib import Path

import numpy as np
from PIL import Image

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "tools/unit3d"))
sys.path.insert(0, str(ROOT / "ver5/tools"))

import base_model as model  # noqa: E402
import pixelize_tree  # noqa: E402
import render as renderer  # noqa: E402
import scene_props3d as props  # noqa: E402
import structures  # noqa: E402
from prepare_scene_tree_sway import fix_wood, root_point  # noqa: E402
from render_scene_tree_sway import model_frame  # noqa: E402

OUT = ROOT / "tools/spritetool/assets/tree-motion-experiments"
TILES = json.load(open(ROOT / "assets_candidate/tiles/tiles.json"))["sprites"]
SETTINGS = json.load(open(ROOT / "assets_candidate/tiles/sources/trees.json"))["trees"]


def normal_map(meshes, target_size, target_alpha):
    old_elevation = renderer.ELEVATION
    renderer.ELEVATION = math.radians(30)
    try:
        faces = renderer.prepare(meshes)
        toward, right, up = renderer.view(renderer.AZIMUTH["SW"])
        ss = 2
        width, height = 300, 340
        _, ids, _, _ = renderer.rasterize(
            faces, toward, right, up, structures.SCALE * ss,
            ((150 + .5) * ss, (290 + 1) * ss), (width * ss, height * ss))
        camera_normals = []
        for _, normal, _ in faces:
            normal = normal if np.dot(normal, toward) >= 0 else -normal
            camera_normals.append([np.dot(normal, axis) for axis in (right, up, toward)])
        camera_normals = np.asarray(camera_normals)
        hit = ids >= 0
        samples = np.zeros((*ids.shape, 3), float)
        samples[hit] = camera_normals[ids[hit]]
        vectors = samples.reshape(height, ss, width, ss, 3).sum(axis=(1, 3))
        count = hit.reshape(height, ss, width, ss).sum(axis=(1, 3))
        vectors /= np.maximum(count[..., None], 1)
        lengths = np.linalg.norm(vectors, axis=2)
        vectors /= np.maximum(lengths[..., None], 1e-6)
        rgba = np.empty((height, width, 4), np.uint8)
        rgba[..., :3] = np.clip((vectors + 1) * 127.5, 0, 255).astype(np.uint8)
        rgba[..., 3] = (count > 0) * 255
    finally:
        renderer.ELEVATION = old_elevation

    full = Image.fromarray(rgba, "RGBA")
    rendered = structures.draw(meshes, cell=(300, 340), pivot=(150, 290))
    box = rendered.getbbox()
    full = full.crop(box).resize(target_size, Image.Resampling.NEAREST)
    pixels = full.load()
    alpha = target_alpha.load()
    for y in range(target_size[1]):
        for x in range(target_size[0]):
            if alpha[x, y] == 0:
                pixels[x, y] = (0, 0, 0, 0)
            elif pixels[x, y][3] == 0:
                pixels[x, y] = (127, 127, 255, 255)
    return full


def moving_sun():
    name = "stree_broad_0"
    props.PROPS[name]()
    original = Image.open(ROOT / "assets_candidate/tiles" / f"{name}.png").convert("RGBA")
    normals = normal_map(model.meshes, original.size, original.getchannel("A"))
    directory = OUT / name
    directory.mkdir(parents=True, exist_ok=True)
    normals.save(directory / "normal-map.png")

    greens = ["#22301a", "#405826", "#688630", "#98b242", "#ccd668"]
    bark = ["#3d211d", "#663720"]
    ramps = [[tuple(bytes.fromhex(color[1:])) for color in ramp] for ramp in (greens, bark)]
    base = np.array([-.6, .75, .45], float)
    base /= np.linalg.norm(base)
    offsets = [(-.12, 0), (0, .08), (.12, 0), (0, -.08)]
    sheet = Image.new("RGBA", (original.width * 2, original.height * 2))
    for frame, (right, up) in enumerate(offsets):
        light = base + np.array([right, up, 0])
        light /= np.linalg.norm(light)
        image = original.copy()
        pixels, normal_pixels = image.load(), normals.load()
        for y in range(image.height):
            for x in range(image.width):
                color = pixels[x, y]
                if color[3] == 0:
                    continue
                normal = np.array(normal_pixels[x, y][:3], float) / 127.5 - 1
                delta = float(np.dot(normal, light - base))
                for ramp in ramps:
                    if color[:3] not in ramp:
                        continue
                    step = 1 if delta > .04 else -1 if delta < -.04 else 0
                    if ramp is ramps[1] and abs(delta) < .07:
                        step = 0
                    index = min(len(ramp) - 1, max(0, ramp.index(color[:3]) + step))
                    pixels[x, y] = (*ramp[index], 255)
                    break
        sheet.alpha_composite(image, ((frame % 2) * image.width, (frame // 2) * image.height))
    sheet.save(directory / "sunlight-sheet.png")
    print(name, "normal map", normals.size, "sunlight", sheet.size)


def softer_willow():
    name = "stree_willow"
    props.PROPS[name]()
    original = [(vertices.copy(), color) for vertices, color in model.meshes]
    renders = [model_frame(original, 2 * math.pi * frame / 4, 0.04) for frame in range(4)]
    side, scale = 112, 4
    x0, y0 = 150 - side // 2, 290 - (side - 12)
    reference = Image.new("RGB", (2 * side * scale, 2 * side * scale), "#ff00ff")
    for frame, render in enumerate(renders):
        cell = render.crop((x0, y0, x0 + side, y0 + side)).resize(
            (side * scale, side * scale), Image.Resampling.NEAREST)
        background = Image.new("RGBA", cell.size, "#ff00ff")
        background.alpha_composite(cell)
        reference.paste(background.convert("RGB"), ((frame % 2) * side * scale,
                                                   (frame // 2) * side * scale))

    directory = OUT / name
    directory.mkdir(parents=True, exist_ok=True)
    reference.save(directory / "reference-3d.png")
    source = [pixelize_tree.keyed(render.crop((x0, y0, x0 + side, y0 + side)), trim=False)
              for render in renders]
    boxes = [image.getbbox() for image in source]
    crop = (min(box[0] for box in boxes) - 3, min(box[1] for box in boxes) - 3,
            max(box[2] for box in boxes) + 3, max(box[3] for box in boxes) + 3)
    width, height = TILES[name]["size"]
    target = root_point(Image.open(ROOT / "assets_candidate/tiles" / f"{name}.png").convert("RGBA"))
    frames = []
    for image in source:
        dot = pixelize_tree.pixelize(SETTINGS[name], width, height, source=image.crop(crop), trim=False)
        root = root_point(dot)
        aligned = Image.new("RGBA", (width, height))
        aligned.paste(dot, (target[0] - root[0], target[1] - root[1]))
        frames.append(aligned)
    fix_wood(frames, SETTINGS[name], name)
    sheet = Image.new("RGBA", (width * 2, height * 2))
    for frame, image in enumerate(frames):
        sheet.alpha_composite(image, ((frame % 2) * width, (frame // 2) * height))
    sheet.save(directory / "dot-sheet.png")
    print(name, "one-third sway", sheet.size)


def main():
    moving_sun()
    softer_willow()


if __name__ == "__main__":
    main()
