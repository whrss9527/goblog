#!/usr/bin/env python3
"""Regenerate the PWA / home-screen icons in static/icons from static/logo.png.

Usage: make icons   (needs Python 3 with Pillow: pip install pillow)

The logo is a full-bleed square picture, so the same artwork works for the
"any" and the "maskable" purpose. Icons are saved with an optimised 256 colour
palette, which keeps the 512px one around 120 KB.
"""
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parent.parent
SIZES = {"icon-192.png": 192, "icon-512.png": 512, "apple-touch-icon.png": 180}


def main() -> None:
    logo = Image.open(ROOT / "static" / "logo.png").convert("RGB")
    out = ROOT / "static" / "icons"
    out.mkdir(parents=True, exist_ok=True)
    for name, size in SIZES.items():
        icon = logo.resize((size, size), Image.LANCZOS)
        icon = icon.quantize(colors=256, method=Image.Quantize.MEDIANCUT, dither=Image.Dither.FLOYDSTEINBERG)
        icon.save(out / name, optimize=True)
        print(f"{name}: {(out / name).stat().st_size} bytes")


if __name__ == "__main__":
    main()
