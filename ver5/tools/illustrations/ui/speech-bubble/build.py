"""Build ten precisely matching PNG pieces for a white speech bubble."""

from pathlib import Path

from PIL import Image, ImageDraw


def main() -> None:
    root = Path(__file__).resolve().parents[4]
    output = root / "assets/graphics/ui/speech-bubble"
    output.mkdir(parents=True, exist_ok=True)
    fill = (255, 255, 255, 255)
    ink = (49, 45, 42, 255)
    factor = 4
    sheet = Image.new("RGBA", (64 * factor, 64 * factor))
    draw = ImageDraw.Draw(sheet)
    draw.rounded_rectangle(
        (0, 0, 64 * factor - 1, 64 * factor - 1),
        radius=16 * factor, fill=fill, outline=ink, width=2 * factor,
    )
    sheet = sheet.resize((64, 64), Image.Resampling.LANCZOS)
    crops = {
        "top-left": (0, 0, 16, 16),
        "top-right": (48, 0, 64, 16),
        "bottom-left": (0, 48, 16, 64),
        "bottom-right": (48, 48, 64, 64),
    }
    for name, bounds in crops.items():
        sheet.crop(bounds).save(output / f"{name}.png")
    for name, size, line in (
        ("top", (1, 16), (0, 0, 0, 1)),
        ("bottom", (1, 16), (0, 14, 0, 15)),
        ("left", (16, 1), (0, 0, 1, 0)),
        ("right", (16, 1), (14, 0, 15, 0)),
    ):
        piece = Image.new("RGBA", size, fill)
        ImageDraw.Draw(piece).rectangle(line, fill=ink)
        piece.save(output / f"{name}.png")
    Image.new("RGBA", (1, 1), fill).save(output / "center.png")
    tail = Image.new("RGBA", (24 * factor, 20 * factor))
    draw = ImageDraw.Draw(tail)
    points = [(0, 0), (23 * factor, 0), (6 * factor, 19 * factor)]
    draw.polygon(points, fill=fill)
    draw.line([points[0], points[2], points[1]], fill=ink, width=2 * factor)
    tail.resize((24, 20), Image.Resampling.LANCZOS).save(output / "tail.png")


if __name__ == "__main__":
    main()
