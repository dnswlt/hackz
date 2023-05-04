"""Angle of view."""

from collections import namedtuple
from math import atan, ceil, cos, pi, sin, sqrt
import sys


html_tmpl = """<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="UTF-8">
    <title>Angle of view</title>
    <meta name="viewport" content="width=device-width,initial-scale=1">
    <style>
        html {{
            font-family: sans-serif;
        }}
        .annot {{
            font-size: 10pt;
        }}
    </style>
</head>

<body>
    <h1>Angle of view</h1>

    <div id="svgs">
        <svg viewBox="{view_box}" width="{width}" height="{height}" xmlns="http://www.w3.org/2000/svg">
{elements}
        </svg>
    </div>

</body>
</html>
"""

def aovh(f):
    return 2 * atan(36/(2*f))


def aovv(f):
    return 2 * atan(24/(2*f))


Arc = namedtuple('Arc', ['path', 'outline', 'x0', 'y0', 'x1', 'y1', 'r', 'phi'])


def svgpath(cx, cy, r, phi, fill="#7f7f7f"):
    x0 = cx + r * cos(-pi/2 - phi/2)
    y0 = cy + r * sin(-pi/2 - phi/2)
    x1 = cx + r * cos(-pi/2 + phi/2)
    y1 = cy + r * sin(-pi/2 + phi/2)
    large_arc_flag = int(phi > pi)
    sweep_flag = 1
    return Arc(path=f'<path d="M {x0} {y0} A {r} {r} 0 {large_arc_flag} {sweep_flag} {x1} {y1} L {cx} {cy} Z" fill="{fill}" />', 
               outline=f'<path d="M {x0} {y0} A {r} {r} 0 {large_arc_flag} 1 {x1} {y1}" stroke="black" fill="none"/>',
               x0=x0, y0=y0, x1=x1, y1=y1, r=r, phi=phi)


def col(x):
    """Returns a #aabbcc color code for the given x in [0, 1]."""
    h = int(255 * (0.4 + 0.5 * (1-x)))
    return "#{0:02x}{0:02x}{0:02x}".format(h)


def main():
    r = 400
    cx = 0
    cy = 0
    l = len(sys.argv)-1
    cols = [col(float(i)/(l-1)) for i in range(l)]
    elems = []
    arcs = []
    focal_lengths = sorted([int(f) for f in sys.argv[1:]])
    for i, f in enumerate(focal_lengths):
        a = aovh(int(f))
        arc = svgpath(cx, cy, r, a, fill=cols[i])
        arcs.append(arc)
        elems.append(arc.path)
        elems.append(f'<text x="{arc.x1}" y="{arc.y1}" class="annot">{f}</text>')
        r *= 1.05
    # Draw horizontal line to visualize field of view as well.
    elems.append(f'<path d="M {arcs[0].x0} {arcs[0].y0} L {arcs[0].x1} {arcs[0].y1}" stroke="black"/>')
    # ... and ticks to estimate the relative sizes.
    # Each tick represents 1m, assuming that the subject (i.e. the line) is 10m away.
    a = arcs[0]
    meter = abs(a.y1) / 10  # or a.y0, which should be equal.
    assert(meter> 0)
    tx = 0
    n = 0
    while tx < a.x1 - meter/2:
        ticklen = 5 if n % 5 == 0 else 3
        elems.append(f'<path d="M {tx} {a.y1-ticklen} L {tx} {a.y1+ticklen}" stroke="black"/>')
        if tx > 0:
            elems.append(f'<path d="M {-tx} {a.y1-ticklen} L {-tx} {a.y1+ticklen}" stroke="black"/>')
        n += 1
        tx += meter
    # Draw arc stroke for widest angle arc.
    elems.append(arcs[0].outline)
    # And ticks every step_degrees.
    step_degrees = 5/180 * pi
    phi = 0
    n = 0
    while phi < a.phi/2 - step_degrees/5:
        ticklen = 10 if n % 2 == 0 else 6
        x0 = cx + a.r * cos(-pi/2 - phi)
        y0 = cy + a.r * sin(-pi/2 - phi)
        x1 = cx + (a.r - ticklen) * cos(-pi/2 - phi)
        y1 = cy + (a.r - ticklen) * sin(-pi/2 - phi)
        elems.append(f'<path d="M {x0} {y0} L {x1} {y1}" stroke="black"/>')
        if phi > 0:
            elems.append(f'<path d="M {-x0} {y0} L {-x1} {y1}" stroke="black"/>')
        phi += step_degrees
        n += 1
    # Write HTML.
    with open('aov.html', 'w') as f_out:
        vb = f"{-a.r} {-r} {2*a.r} {r}"
        f_out.write(html_tmpl.format(elements='\n'.join(elems), view_box=vb, width=ceil(2*a.r), height=ceil(r)))


if __name__ == "__main__":
    main()