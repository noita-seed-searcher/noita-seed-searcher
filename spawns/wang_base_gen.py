#!/usr/bin/env python3
"""Decode the coalmine Wang-tile PNG to ground-truth RGB bytes.

Independent decoder (PIL) to validate Go's loadWangRGB and to feed the JS
reference tileset builder. Output: spawns/wang_base.json
"""
import json
import os
from PIL import Image

HERE = os.path.dirname(os.path.abspath(__file__))
TILES = {"coalmine": "coalmine.png"}

out = {}
for key, fname in TILES.items():
    img = Image.open(os.path.join(HERE, "data", "wang_tiles", fname)).convert("RGB")
    w, h = img.size
    rgb = []
    for (r, g, b) in img.getdata():
        rgb.extend([r, g, b])
    out[key] = {"w": w, "h": h, "rgb": rgb}

with open(os.path.join(HERE, "wang_base.json"), "w") as f:
    json.dump(out, f)
print("wrote wang_base.json:", {k: f'{v["w"]}x{v["h"]}' for k, v in out.items()})
