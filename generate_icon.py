from PIL import Image, ImageDraw


def main() -> None:
    W = 1024
    img = Image.new("RGBA", (W, W), (0, 0, 0, 0))

    # rounded square background with vertical gradient
    bg = Image.new("RGBA", (W, W), (0, 0, 0, 0))
    d = ImageDraw.Draw(bg)
    r = 184
    mask = Image.new("L", (W, W), 0)
    md = ImageDraw.Draw(mask)
    md.rounded_rectangle([96, 96, W - 96, W - 96], radius=r, fill=255)

    g1 = (0x0B, 0x10, 0x20)
    g2 = (0x1E, 0x2A, 0x5A)
    for y in range(W):
        t = y / (W - 1)
        col = (
            int(g1[0] * (1 - t) + g2[0] * t),
            int(g1[1] * (1 - t) + g2[1] * t),
            int(g1[2] * (1 - t) + g2[2] * t),
            255,
        )
        d.line([(0, y), (W, y)], fill=col)

    img = Image.composite(bg, img, mask)
    draw = ImageDraw.Draw(img)

    accent = (0x5E, 0xEA, 0xD4, 255)
    accent2 = (0x22, 0xD3, 0xEE, 255)

    # simple glow underlay
    for w, alpha in [(70, 70), (52, 90)]:
        c = (accent[0], accent[1], accent[2], alpha)
        draw.line([(512, 250), (512, 760)], fill=c, width=w)

    stroke = 34
    stroke2 = 26

    def bezier(p0, p1, p2, p3, steps=60):
        pts = []
        for i in range(steps + 1):
            t = i / steps
            x = (
                (1 - t) ** 3 * p0[0]
                + 3 * (1 - t) ** 2 * t * p1[0]
                + 3 * (1 - t) * t**2 * p2[0]
                + t**3 * p3[0]
            )
            y = (
                (1 - t) ** 3 * p0[1]
                + 3 * (1 - t) ** 2 * t * p1[1]
                + 3 * (1 - t) * t**2 * p2[1]
                + t**3 * p3[1]
            )
            pts.append((x, y))
        return pts

    # staff
    draw.line([(512, 250), (512, 760)], fill=accent, width=stroke)
    draw.line([(512, 250), (512, 760)], fill=accent2, width=stroke - 10)

    # left prong
    pts = bezier((512, 290), (460, 290), (418, 330), (418, 384), 50) + bezier(
        (418, 384), (418, 432), (452, 470), (496, 470), 50
    )
    draw.line(pts, fill=accent, width=stroke, joint="curve")
    pts = bezier((418, 384), (392, 360), (374, 334), (362, 306), 40)
    draw.line(pts, fill=accent, width=stroke2)

    # right prong
    pts = bezier((512, 290), (564, 290), (606, 330), (606, 384), 50) + bezier(
        (606, 384), (606, 432), (572, 470), (528, 470), 50
    )
    draw.line(pts, fill=accent, width=stroke, joint="curve")
    pts = bezier((606, 384), (632, 360), (650, 334), (662, 306), 40)
    draw.line(pts, fill=accent, width=stroke2)

    # spear head
    spear = [
        (512, 250),
        (490, 276),
        (462, 324),
        (486, 312),
        (512, 306),
        (538, 312),
        (562, 324),
        (534, 276),
    ]
    draw.polygon(spear, fill=accent2)

    # chain links
    link_w = 28
    draw.rounded_rectangle([420, 560, 610, 680], radius=60, outline=accent, width=link_w)
    draw.rounded_rectangle(
        [472, 520, 662, 640], radius=60, outline=(accent2[0], accent2[1], accent2[2], 200), width=link_w
    )

    # inner border
    draw.rounded_rectangle(
        [126, 126, W - 126, W - 126], radius=168, outline=(0x2A, 0x3A, 0x66, 150), width=8
    )

    img.save("onex-icon.png")

    sizes = [16, 24, 32, 48, 64, 128, 256]
    icons = [img.resize((s, s), Image.LANCZOS) for s in sizes]
    icons[0].save("onex-icon.ico", format="ICO", sizes=[(s, s) for s in sizes])


if __name__ == "__main__":
    main()

