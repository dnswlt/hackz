
import math


def html(svg_elems, view_box=(400, 400)):
    svg_elems_str = '\n'.join(svg_elems)

    return f"""<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>SVG</title>
</head>

<body>
    <svg viewBox="0 0 {view_box[0]} {view_box[1]}" xmlns="http://www.w3.org/2000/svg">
        {svg_elems_str}
    </svg>
</body>

</html>"""


def hexagon(a, x, y):
    b = math.sqrt(3) * a
    h = a / 2
    # points in
    ps = [
        a + 0j,
        a + b/2 + (h) * 1j,
        a + b/2 + (h + a) * 1j,
        a + (2 * a) * 1j,
        a - b/2 + (h + a) * 1j,
        a - b/2 + (h) * 1j,
        a + 0j,
    ]
    return [p + (x + y * 1j) for p in ps]


def hexagon_svg(hex_points, stroke="black", fill="none"):
    points = " ".join(f"{p.real:.1f},{p.imag:.1f}" for p in hex_points)
    return f'<polyline points="{points}" fill="{fill}" stroke="{stroke}" />'


def bbox(hs):
    min_x = min(min(c.real for c in h) for h in hs)
    min_y = min(min(c.imag for c in h) for h in hs)
    max_x = max(max(c.real for c in h) for h in hs)
    max_y = max(max(c.imag for c in h) for h in hs)
    return ((min_x, min_y), (max_x, max_y))


def main():
    filename = "./polyline.html"
    padding = 10
    hexes = []
    a = 100
    b = math.sqrt(3) * a
    ylim = 32
    for dy in range(ylim):
        xlim = ylim if dy & 1 == 0 else ylim - 1
        for dx in range(xlim):
            x_off = 0 if dy & 1 == 0 else b / 2
            hexes.append(hexagon(a, x_off + dx * b, dy * a * 3/2))
    ((_, _), (max_x, max_y)) = bbox(hexes)
    with open(filename, 'w') as f_out:
        hex_svgs = []
        # fills = ['#dfffdf', '#dfdfff', '#ffdfdf']
        # fills = ['#cad2c5', '#84a98c', '#52796f']
        fills = ['#ef8354', '#4f5d75', '#bfc0c0']
        for i, h in enumerate(hexes):
            fill = fills[i % len(fills)]
            # f'#{i:02x}{i:02x}ff'
            hex_svgs.append(hexagon_svg(h, fill=fill))
        f_out.write(html(hex_svgs, view_box=(max_x + padding, max_y + padding)))
        print(f"Wrote {filename}")


if __name__ == "__main__":
    main()