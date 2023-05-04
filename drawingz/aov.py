"""Angle of view."""

from math import atan, cos, pi, sin
import sys


html_tmpl = """<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="UTF-8">
    <title>Angle of view</title>
    <meta name="viewport" content="width=device-width,initial-scale=1">
</head>

<body>
    <h1>Angle of view</h1>

    <div id="svgs">
        <svg width="800" height="800" xmlns="http://www.w3.org/2000/svg">
{paths}
        </svg>
    </div>

</body>
</html>
"""

def aovh(f):
    return 2 * atan(36/(2*f))


def aovv(f):
    return 2 * atan(24/(2*f))


def svgpath(cx, cy, r, phi, fill="#7f7f7f"):
    x0 = cx + r * cos(-pi/2 - phi/2)
    y0 = cy + r * sin(-pi/2 - phi/2)
    x1 = cx + r * cos(-pi/2 + phi/2)
    y1 = cy + r * sin(-pi/2 + phi/2)
    large_arc_flag = int(phi > pi)
    sweep_flag = 1
    return f'<path d="M {x0} {y0} A {r} {r} 0 {large_arc_flag} {sweep_flag} {x1} {y1} L {cx} {cy} Z" fill="{fill}" />'


def main():
    r = 300
    cx = 400
    cy = 800
    l = len(sys.argv)-1
    cols = ["#{0:02x}{0:02x}{0:02x}".format(int(255 * (0.2 + 0.5 * float(l-i) / l))) for i in range(l)]
    paths = []
    for i, f in enumerate(sys.argv[1:]):
        a = aovh(int(f))
        paths.append(svgpath(cx, cy, r, a, fill=cols[i]))
        r *= 1.05
    with open('aov.html', 'w') as f_out:
        f_out.write(html_tmpl.format(paths='\n'.join(paths)))

if __name__ == "__main__":
    main()