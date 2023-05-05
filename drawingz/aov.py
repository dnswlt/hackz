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
        .center {{
            text-anchor: middle;
        }}
        .vcenter {{
            dominant-baseline: middle;
        }}
        .ralign {{
            text-anchor: end;
        }}
        .light {{
            fill: #b0b0b0;
        }}

    </style>
</head>

<body>
    <h1>Horizontal angle and field of view</h1>
    <p>Ticks are at 5&deg; for angle of view and at 1m for field of view (for an object plane distance of 10m).</p>
    <div id="aov">
        <svg viewBox="{view_box}" width="{width}" height="{height}" xmlns="http://www.w3.org/2000/svg">
{elements}
        </svg>
    </div>
    
    <h1>Hyperfocal distance</h1>
    <p>Hyperfocal distance in meters for typical f-numbers and a 35mm full frame sensor.</p>
    <div id="hfd">
        <svg width="{hfd_width}" height="{hfd_height}" xmlns="http://www.w3.org/2000/svg">
{hfd_elements}
        </svg>
    </div>
</body>
</html>
"""

def aovh(f):
    return 2 * atan(36/(2*f))


def aovv(f):
    return 2 * atan(24/(2*f))


def hfd(focal_length, f_number, coc=0.03):
    """Returns the hyperfocal distance in mm."""
    return focal_length**2/(f_number*coc) + focal_length


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
    assert(meter > 0)
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
    # Add hfd elements.
    x_spacing = 100
    x_off = 30
    hfd_width = 2*x_off + (len(focal_lengths)-1) * x_spacing + 40  # for trailing label text
    hfd_height = 400
    hfd_elems = []
    f_numbers = [1.8, 2.8, 4, 5.6, 8, 11, 16]
    y0, y1 = 10, hfd_height * 0.9
    x = x_off
    tick_length = 5
    for f in focal_lengths:
        hfd_elems.append(f'<path d="M {x} {y0} L {x} {y1}" stroke="black" fill="none"/>')
        hfd_elems.append(f'<text class="annot center" x="{x}" y="{y1+16}">{f}</text>')
        # One tick per f-number.
        hfds = [hfd(f, n)/1000 for n in f_numbers]
        ys = [y0 + (y1-y0) * (1-h/hfds[0]) for h in hfds]
        for f, h, y in zip(f_numbers, hfds, ys):
            hfd_elems.append(f'<path d="M {x} {y} L {x+tick_length} {y}" stroke="black" fill="none"/>')
            hfd_elems.append(f'<text class="annot vcenter" x="{x+tick_length+2}" y="{y}">{h:.2f}</text>')
            hfd_elems.append(f'<text class="annot vcenter ralign light" x="{x-2}" y="{y}">{f:.1f}</text>')
        x += x_spacing
    # Write HTML.
    with open('aov.html', 'w') as f_out:
        vb = f"{-a.r} {-r} {2*a.r} {r}"
        f_out.write(html_tmpl.format(
            elements='\n'.join(elems),
            hfd_elements='\n'.join(hfd_elems),
            view_box=vb, 
            width=ceil(2*a.r), 
            height=ceil(r),
            hfd_width=hfd_width,
            hfd_height=hfd_height))


if __name__ == "__main__":
    main()