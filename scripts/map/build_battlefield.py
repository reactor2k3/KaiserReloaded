# Erzeugt das Schlachtfeld-SVG im Stil einer barocken Gefechtskarte und setzt es in Schlacht.dc.html ein.
# Aufruf: python -I -X utf8 build_battlefield.py <Schlacht.dc.html>
import math, random, sys
random.seed(1724)
W, H = 900, 640
out = []
def a(s): out.append(s)

def catmull(pts):
    # Catmull-Rom-Kurve als kubische Bézier-Pfadangabe
    d = f"M{pts[0][0]:.1f} {pts[0][1]:.1f}"
    for i in range(len(pts) - 1):
        p0 = pts[max(i - 1, 0)]; p1 = pts[i]; p2 = pts[i + 1]; p3 = pts[min(i + 2, len(pts) - 1)]
        c1 = (p1[0] + (p2[0] - p0[0]) / 6, p1[1] + (p2[1] - p0[1]) / 6)
        c2 = (p2[0] - (p3[0] - p1[0]) / 6, p2[1] - (p3[1] - p1[1]) / 6)
        d += f"C{c1[0]:.1f} {c1[1]:.1f} {c2[0]:.1f} {c2[1]:.1f} {p2[0]:.1f} {p2[1]:.1f}"
    return d

RIVER = [(565, -10), (545, 80), (578, 170), (535, 260), (552, 350), (518, 440), (548, 530), (528, 650)]
def river_x(y):
    for (x1, y1), (x2, y2) in zip(RIVER, RIVER[1:]):
        if y1 <= y <= y2:
            return x1 + (x2 - x1) * (y - y1) / (y2 - y1)
    return RIVER[-1][0]
FORESTS = [(120, 120, 95, 60), (300, 560, 120, 55), (775, 120, 100, 70), (770, 520, 95, 62), (440, 60, 60, 35)]
HILLS = [(330, 185, "Galgenberg"), (700, 400, "Kapellenhöhe")]
VILLAGES = [(205, 330, "Wiesenthal"), (725, 285, "Neukirch")]
def in_forest(x, y, pad=0):
    return any(((x - fx) / (rx + pad)) ** 2 + ((y - fy) / (ry + pad)) ** 2 < 1 for fx, fy, rx, ry in FORESTS)

a(f'<svg role="img" aria-label="Schlachtfeld bei Neukirch: links Ihre Aufstellung, rechts hinter dem Fluss die verdeckte Aufstellung Bayerns" viewBox="0 0 {W} {H}" style="width: 100%; height: 100%">')
a('''<defs>
<filter id="wolke" x="-10%" y="-10%" width="120%" height="120%"><feTurbulence type="fractalNoise" baseFrequency="0.018" numOctaves="3" seed="7"></feTurbulence><feDisplacementMap in="SourceGraphic" scale="34" xChannelSelector="R" yChannelSelector="G"></feDisplacementMap></filter>
<filter id="papier"><feTurbulence type="fractalNoise" baseFrequency="0.9" numOctaves="2" seed="3" result="n"></feTurbulence><feColorMatrix in="n" type="matrix" values="0 0 0 0 0.42  0 0 0 0 0.33  0 0 0 0 0.2  0 0 0 0.07 0"></feColorMatrix></filter>
<pattern id="wasser" width="12" height="6" patternUnits="userSpaceOnUse"><rect width="12" height="6" fill="#A9C1CF"></rect><path d="M0 3Q3 1 6 3T12 3" fill="none" stroke="#7F9DB3" stroke-width="0.7"></path></pattern>
</defs>''')
a(f'<rect width="{W}" height="{H}" fill="#EAE0C6"></rect>')

# Felder als leicht gedrehte Parzellen mit Furchen
tints = ["#DED2AA", "#D5C99B", "#E4DAB8", "#D9D0A4", "#CFC596"]
for gy in range(0, H, 64):
    for gx in range(0, W, 84):
        cx, cy = gx + 42 + random.uniform(-10, 10), gy + 32 + random.uniform(-8, 8)
        if in_forest(cx, cy, 25) or abs(cx - river_x(cy)) < 50 or random.random() < 0.18:
            continue
        if any(math.hypot(cx - vx, cy - vy) < 55 for vx, vy, _ in VILLAGES):
            continue
        w, h = random.uniform(56, 76), random.uniform(38, 52)
        ang = random.uniform(-9, 9)
        horiz = random.random() < 0.5
        lines = []
        step = 5
        if horiz:
            for yy in range(int(-h / 2) + 4, int(h / 2) - 2, step):
                lines.append(f"M{-w/2+3:.1f} {yy}H{w/2-3:.1f}")
        else:
            for xx in range(int(-w / 2) + 4, int(w / 2) - 2, step):
                lines.append(f"M{xx} {-h/2+3:.1f}V{h/2-3:.1f}")
        a(f'<g transform="translate({cx:.1f} {cy:.1f}) rotate({ang:.1f})"><rect x="{-w/2:.1f}" y="{-h/2:.1f}" width="{w:.1f}" height="{h:.1f}" fill="{random.choice(tints)}" stroke="#BFAF86" stroke-width="0.8"></rect><path d="{"".join(lines)}" stroke="#BFAF86" stroke-width="0.5" opacity="0.8"></path></g>')

# Hügel als Höhenlinien mit Schraffur
for hx, hy, name in HILLS:
    for i, (rx, ry) in enumerate([(78, 46), (60, 35), (42, 24), (24, 13)]):
        a(f'<ellipse cx="{hx}" cy="{hy}" rx="{rx}" ry="{ry}" fill="{"#D8C9A0" if i == 3 else "none"}" stroke="#8A6A44" stroke-width="{0.8 + i * 0.2:.1f}" stroke-dasharray="{"4 3" if i == 0 else "none"}" opacity="0.85"></ellipse>')
    ticks = []
    for k in range(36):
        t = 2 * math.pi * k / 36
        x1, y1 = hx + 60 * math.cos(t), hy + 35 * math.sin(t)
        x2, y2 = hx + 72 * math.cos(t), hy + 42 * math.sin(t)
        ticks.append(f"M{x1:.1f} {y1:.1f}L{x2:.1f} {y2:.1f}")
    a(f'<path d="{"".join(ticks)}" stroke="#8A6A44" stroke-width="0.7" opacity="0.7"></path>')
    a(f'<text x="{hx}" y="{hy + 4}" text-anchor="middle" font-family="EB Garamond, serif" font-style="italic" font-size="13" fill="#5E4E3D">{name}</text>')

# Straßen
roads = [[(-10, 335), (110, 330), (205, 330), (330, 360), (450, 350), (552, 345), (640, 310), (725, 285), (910, 275)],
         [(250, -10), (230, 120), (205, 330), (240, 470), (215, 650)],
         [(725, 285), (760, 420), (700, 650)]]
for r in roads:
    d = catmull(r)
    a(f'<path d="{d}" fill="none" stroke="#9B8360" stroke-width="7" stroke-linecap="round"></path><path d="{d}" fill="none" stroke="#E8DCBC" stroke-width="4.5" stroke-linecap="round"></path>')

# Wälder: Bäume mit Schatten
trees = []
for fx, fy, rx, ry in FORESTS:
    n = int(rx * ry / 55)
    for _ in range(n):
        t = random.uniform(0, 2 * math.pi); r = math.sqrt(random.random())
        trees.append((fx + rx * r * math.cos(t), fy + ry * r * math.sin(t), random.uniform(4.5, 7)))
for x, y in [(random.uniform(20, 880), random.uniform(20, 620)) for _ in range(70)]:
    if not in_forest(x, y, 10) and abs(x - river_x(y)) > 30:
        trees.append((x, y, random.uniform(3.5, 5)))
trees.sort(key=lambda t: t[1])
a('<g>' + "".join(f'<circle cx="{x+1.5:.1f}" cy="{y+2:.1f}" r="{r:.1f}" fill="#5F7346" opacity="0.55"></circle>' for x, y, r in trees) + '</g>')
a('<g stroke="#3E4F2C" stroke-width="0.8">' + "".join(f'<circle cx="{x:.1f}" cy="{y:.1f}" r="{r:.1f}" fill="{random.choice(["#7E9660", "#8AA06A", "#738B55"])}"></circle>' for x, y, r in trees) + '</g>')

# Fluss mit Ufer und Brücke
rd = catmull(RIVER)
a(f'<path d="{rd}" fill="none" stroke="#5E7F99" stroke-width="22" stroke-linecap="round"></path>')
a(f'<path d="{rd}" fill="none" stroke="url(#wasser)" stroke-width="18" stroke-linecap="round"></path>')
a('<g transform="translate(552 345) rotate(-12)"><rect x="-18" y="-6" width="36" height="12" fill="#C9B48A" stroke="#5E4E3D" stroke-width="1.2"></rect><path d="M-14 -6V6M-7 -6V6M0 -6V6M7 -6V6M14 -6V6" stroke="#8A6A44" stroke-width="0.8"></path></g>')
a('<text x="575" y="200" font-family="EB Garamond, serif" font-style="italic" font-size="15" fill="#1F3A5F" transform="rotate(78 575 200)" stroke="#EAE0C6" stroke-width="3" paint-order="stroke">die Pleiße</text>')

# Dörfer: Häuser mit Ziegeldach und Kirche
for vx, vy, name in VILLAGES:
    houses = []
    for _ in range(12):
        t = random.uniform(0, 2 * math.pi); r = random.uniform(12, 38)
        hx, hy = vx + r * math.cos(t), vy + r * 0.7 * math.sin(t)
        houses.append(f'<g transform="translate({hx:.1f} {hy:.1f}) rotate({random.uniform(-20, 20):.0f})"><rect x="-6" y="-4" width="12" height="8" fill="#A0452F" stroke="#4A2A1C" stroke-width="0.8"></rect><path d="M-6 0H6" stroke="#4A2A1C" stroke-width="0.6"></path></g>')
    a("".join(houses))
    a(f'<g transform="translate({vx} {vy})"><rect x="-9" y="-6" width="18" height="12" fill="#D9CBA6" stroke="#4A2A1C" stroke-width="1"></rect><rect x="5" y="-12" width="7" height="18" fill="#D9CBA6" stroke="#4A2A1C" stroke-width="1"></rect><path d="M8.5 -18V-12M6 -15.5H11" stroke="#4A2A1C" stroke-width="1.2"></path></g>')
    a(f'<text x="{vx}" y="{vy + 52}" text-anchor="middle" font-family="Cinzel, serif" font-weight="600" font-size="13" letter-spacing="1.5" fill="#2B2118" stroke="#EAE0C6" stroke-width="3" paint-order="stroke">{name.upper()}</text>')

a(f'<rect width="{W}" height="{H}" filter="url(#papier)"></rect>')

# Truppen als Regimentsblöcke (Stil der Gefechtspläne des 18. Jahrhunderts)
GRUEN, PREUSS = "#4E6B33", "#4F5560"
def inf(x, y, c, rot=0):
    a(f'<g transform="translate({x} {y}) rotate({rot})"><rect x="-22" y="-7" width="44" height="14" fill="{c}" stroke="#1E1710" stroke-width="1.2"></rect><path d="M-22 -2.5H22" stroke="#F6EEDB" stroke-width="1"></path><path d="M-18 -7V-15L-10 -12.5L-18 -10" fill="#F6EEDB" stroke="#1E1710" stroke-width="0.8"></path></g>')
def kav(x, y, c, rot=0):
    a(f'<g transform="translate({x} {y}) rotate({rot})"><rect x="-18" y="-7" width="36" height="14" fill="{c}" stroke="#1E1710" stroke-width="1.2"></rect><path d="M-18 7L18 -7" stroke="#F6EEDB" stroke-width="1.6"></path></g>')
def art(x, y, c):
    a(f'<g transform="translate({x} {y})"><rect x="-11" y="-6" width="22" height="12" fill="{c}" stroke="#1E1710" stroke-width="1.2"></rect><circle cx="-3" cy="2" r="3" fill="none" stroke="#F6EEDB" stroke-width="1.2"></circle><path d="M-2 0L14 -5" stroke="#1E1710" stroke-width="3" stroke-linecap="round"></path><path d="M-2 0L14 -5" stroke="#F6EEDB" stroke-width="1.2" stroke-linecap="round"></path></g>')
def miliz(x, y, c):
    a(f'<g transform="translate({x} {y})"><rect x="-16" y="-6" width="32" height="12" fill="#EAE0C6" stroke="{c}" stroke-width="2" stroke-dasharray="4 2"></rect><path d="M-10 0H10" stroke="{c}" stroke-width="2"></path></g>')
kav(395, 140, GRUEN, -8); kav(400, 168, GRUEN, -8)
inf(470, 255, GRUEN, 8); inf(478, 285, GRUEN, 8); inf(455, 400, GRUEN, -6)
inf(420, 330, PREUSS, 4); inf(415, 360, PREUSS, 4)
art(345, 160, GRUEN); art(490, 225, GRUEN)
miliz(190, 300, GRUEN); miliz(225, 365, GRUEN)
a('<rect x="355" y="440" width="54" height="26" fill="none" stroke="#7A1F1F" stroke-width="2" stroke-dasharray="5 3"></rect>')
a('<g opacity="0.55">'); inf(382, 453, GRUEN); a('</g>')
a('<path d="M392 470L392 488L397 483L401 492L404 490L400 481L407 481Z" fill="#F6EEDB" stroke="#1E1710" stroke-width="1"></path>')

# Verdeckte Feindseite: Nebel jenseits des Flusses
fog = [(river_x(y) + 26, y) for y in range(-40, H + 41, 40)]
fd = catmull(fog) + f"L{W + 60} {H + 40}L{W + 60} -40Z"
a(f'<path d="{fd}" fill="#F1E9D4" opacity="0.88" filter="url(#wolke)"></path>')
a('<g transform="translate(735 330)"><circle r="54" fill="#F6EEDB" stroke="#1F3A5F" stroke-width="1.5"></circle><circle r="48" fill="none" stroke="#1F3A5F" stroke-width="0.8" stroke-dasharray="2 3"></circle>'
  '<text y="-8" text-anchor="middle" font-family="Cinzel, serif" font-weight="700" font-size="13" letter-spacing="1.5" fill="#1F3A5F">BAYERN</text>'
  '<text y="9" text-anchor="middle" font-family="EB Garamond, serif" font-size="13" fill="#2B2118">stellt verdeckt auf</text>'
  '<text y="25" text-anchor="middle" font-family="EB Garamond, serif" font-style="italic" font-size="12" fill="#5E4E3D">≈ 200 Mann</text></g>')

# Kartusche, Kompass, Maßstab, Rahmen
a('<g transform="translate(24 22)"><rect width="250" height="62" fill="#F6EEDB" stroke="#2B2118" stroke-width="1.2"></rect><rect x="4" y="4" width="242" height="54" fill="none" stroke="#B08A3E" stroke-width="0.8"></rect>'
  '<text x="125" y="34" text-anchor="middle" font-family="UnifrakturMaguntia, serif" font-size="24" fill="#2B2118">Schlacht bei Neukirch</text>'
  '<text x="125" y="51" text-anchor="middle" font-family="Cinzel, serif" font-size="11" letter-spacing="2" fill="#5E4E3D">ANNO 1724</text></g>')
a('<g transform="translate(62 575)"><circle r="26" fill="#F6EEDB" stroke="#2B2118" stroke-width="1"></circle><path d="M0 -24L5 0L0 24L-5 0Z" fill="#2B2118"></path><path d="M-24 0L0 4L24 0L0 -4Z" fill="#8A6A44"></path><path d="M0 -24L5 0L0 0Z" fill="#7A1F1F"></path><text y="-30" text-anchor="middle" font-family="Cinzel, serif" font-size="11" font-weight="700" fill="#2B2118">N</text></g>')
a('<g transform="translate(110 600)"><rect width="40" height="6" fill="#2B2118"></rect><rect x="40" width="40" height="6" fill="#F6EEDB" stroke="#2B2118" stroke-width="1"></rect><rect x="80" width="40" height="6" fill="#2B2118"></rect><text x="60" y="20" text-anchor="middle" font-family="EB Garamond, serif" font-size="12" fill="#2B2118">1 500 Schritt</text></g>')
a(f'<rect x="4" y="4" width="{W-8}" height="{H-8}" fill="none" stroke="#2B2118" stroke-width="2"></rect><rect x="9" y="9" width="{W-18}" height="{H-18}" fill="none" stroke="#2B2118" stroke-width="0.7"></rect>')
a('</svg>')
svg = "\n".join(out)

path = sys.argv[1]
src = open(path, encoding="utf-8").read()
s = src.index('<svg role="img" aria-label="Schlachtfeld')
e = src.index('</svg>', src.index('<text x="680" y="30"', s)) + len('</svg>') if '<text x="680" y="30"' in src[s:] else src.index('</svg>\n    </section>', s) + len('</svg>')
src = src[:s] + svg + src[e:]
open(path, "w", encoding="utf-8", newline="\n").write(src)
print("ok", len(svg))
