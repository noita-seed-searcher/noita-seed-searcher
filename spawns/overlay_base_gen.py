#!/usr/bin/env python3
"""Decode the coalmine overlay PNG (RGBA) for the coalmine hack.

Independent decoder (PIL); validates Go's loadOverlay and feeds the JS
reference via a stubbed loadPNG. Output: spawns/overlay_base.json
"""
import json
import os
from PIL import Image

HERE = os.path.dirname(os.path.abspath(__file__))
img = Image.open(os.path.join(HERE, "data", "wang_tiles", "extra_layers", "coalmine.png")).convert("RGBA")
w, h = img.size
rgba = []
for (r, g, b, a) in img.getdata():
    rgba.extend([r, g, b, a])

with open(os.path.join(HERE, "overlay_base.json"), "w") as f:
    json.dump({"coalmine": {"w": w, "h": h, "rgba": rgba}}, f)
print(f"wrote overlay_base.json: coalmine {w}x{h}")
