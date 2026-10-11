#!/usr/bin/env python3
"""Downscale screenshots for review: view.py in.png [out.png] [width] — or a 2x2 sheet: view.py --sheet out.png a b c d"""
import sys
from PIL import Image
a = sys.argv[1:]
if a[0] == '--sheet':
    out, files = a[1], a[2:]
    ims = [Image.open(f).convert('RGB') for f in files]
    w = 1200; ims = [im.resize((w, round(im.height * w / im.width)), Image.LANCZOS) for im in ims]
    cols = 2; rows = (len(ims) + 1) // 2; h = max(im.height for im in ims)
    sheet = Image.new('RGB', (cols * w + 8, rows * h + 8 * (rows - 1)), 'black')
    for i, im in enumerate(ims): sheet.paste(im, ((i % 2) * (w + 8), (i // 2) * (h + 8)))
    sheet.save(out)
else:
    src = a[0]; out = a[1] if len(a) > 1 else src.replace('.png', '.s.png'); w = int(a[2]) if len(a) > 2 else 1440
    im = Image.open(src).convert('RGB'); im.resize((w, round(im.height * w / im.width)), Image.LANCZOS).save(out)
print(out)
