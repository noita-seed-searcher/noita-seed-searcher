#!/usr/bin/env python3
"""Decode the base biome-map PNGs to ground-truth RGBA byte arrays.

Independent third decoder (PIL) used to validate both the Go PNG decode and to
feed the JS reference generator. Output: spawns/biome_base.json
"""
import json
import os
from PIL import Image

HERE = os.path.dirname(os.path.abspath(__file__))
MAPS = {
    "normal": "biome_map.png",
    "ngp": "biome_map_newgame_plus.png",
    "nightmare": "biome_map_nightmare.png",
}

out = {}
for key, fname in MAPS.items():
    img = Image.open(os.path.join(HERE, "data", "biome_maps", fname)).convert("RGBA")
    w, h = img.size
    px = list(img.getdata())  # list of (r,g,b,a)
    rgba = []
    argb = []  # 0xFFRRGGBB, what Go's loadBiomeBase should produce
    for (r, g, b, a) in px:
        rgba.extend([r, g, b, 255])
        argb.append((0xFF000000 | (r << 16) | (g << 8) | b))
    out[key] = {"w": w, "h": h, "rgba": rgba, "argb": argb}

with open(os.path.join(HERE, "biome_base.json"), "w") as f:
    json.dump(out, f)
print("wrote biome_base.json:", {k: f'{v["w"]}x{v["h"]}' for k, v in out.items()})
