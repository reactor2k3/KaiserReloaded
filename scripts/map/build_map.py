# Baut die Reichskarte (SVG-Fragment) aus Natural-Earth-Daten (gemeinfrei).
# Aufruf: python -I -X utf8 build_map.py <geo-data-dir> <pylib-dir> <out.svgfrag>
import json, math, sys
sys.path.insert(0, sys.argv[2])
from shapely.geometry import shape, box, mapping
from shapely.ops import unary_union, polylabel, transform

DATA, OUT = sys.argv[1], sys.argv[3]
LON0, LAT0 = 11.0, 50.5
BBOX = box(1.8, 45.4, 20.2, 55.6)
W = 900  # Breite der viewBox

def load(name):
    return json.load(open(f"{DATA}\\{name}", encoding="utf-8"))["features"]

# Zuordnung moderner Verwaltungsgebiete zu den neun Spiel-Ländern
LAENDER = {
    "Preußen": [("DEU", n) for n in ["Schleswig-Holstein", "Hamburg", "Bremen", "Niedersachsen", "Nordrhein-Westfalen", "Sachsen-Anhalt", "Brandenburg", "Berlin", "Mecklenburg-Vorpommern"]]
              + [("POL", n) for n in ["West Pomeranian", "Pomeranian", "Lubusz", "Lower Silesian", "Opole", "Silesian"]],
    "Hessen": [("DEU", "Hessen")],
    "Sachsen": [("DEU", "Sachsen"), ("DEU", "Thüringen")],
    "Bayern": [("DEU", "Bayern")],
    "Pfalz": [("DEU", "Rheinland-Pfalz"), ("DEU", "Saarland"), ("DEU", "Baden-Württemberg")],
    "Böhmen": [("CZE", n) for n in ["Ústecký", "Liberecký", "Karlovarský", "Plzeňský", "Jihočeský", "Královéhradecký", "Pardubický", "Středočeský", "Prague"]],
    "Mähren": [("CZE", n) for n in ["Olomoucký", "Moravskoslezský", "Jihomoravský", "Zlínský", "Vysočina"]],
    "Tirol": [("AUT", "*")],
    "Flandern": [("BEL", "*"), ("LUX", "*")],
}
HAUPTSTADT = {"Preußen": (13.40, 52.52, "Berlin"), "Hessen": (9.48, 51.31, "Kassel"), "Sachsen": (13.74, 51.05, "Dresden"),
              "Bayern": (11.58, 48.14, "München"), "Pfalz": (8.47, 49.49, "Mannheim"), "Böhmen": (14.42, 50.08, "Prag"),
              "Mähren": (16.61, 49.20, "Brünn"), "Tirol": (11.40, 47.27, "Innsbruck"), "Flandern": (3.72, 51.05, "Gent")}

# Kegelprojektion (Lambert, flächenarm verzerrt für Mitteleuropa)
p1, p2 = math.radians(48), math.radians(54)
n = math.log(math.cos(p1) / math.cos(p2)) / math.log(math.tan(math.pi / 4 + p2 / 2) / math.tan(math.pi / 4 + p1 / 2))
F = math.cos(p1) * math.tan(math.pi / 4 + p1 / 2) ** n / n
rho0 = F / math.tan(math.pi / 4 + math.radians(LAT0) / 2) ** n
def proj(lon, lat):
    rho = F / math.tan(math.pi / 4 + math.radians(lat) / 2) ** n
    th = n * math.radians(lon - LON0)
    return rho * math.sin(th), rho0 - rho * math.cos(th)

corners = [proj(x, y) for x in (2.3, 19.7) for y in (45.6, 55.3)] + [proj(11, 55.3), proj(11, 45.6)]
minx = min(c[0] for c in corners); maxx = max(c[0] for c in corners)
miny = min(c[1] for c in corners); maxy = max(c[1] for c in corners)
S = W / (maxx - minx)
H = round((maxy - miny) * S)
def xy(lon, lat):
    x, y = proj(lon, lat)
    return (x - minx) * S, (maxy - y) * S
def tf(x, y, z=None):
    import numpy as np
    xs, ys = zip(*[xy(a, b) for a, b in zip(x, y)])
    return xs, ys

def path(geom, tol=0.6):
    g = transform(lambda x, y, z=None: tf(x, y), geom).simplify(tol, preserve_topology=True)
    out = []
    polys = getattr(g, "geoms", [g])
    for p in polys:
        if p.geom_type == "Polygon":
            for ring in [p.exterior, *p.interiors]:
                pts = list(ring.coords)
                out.append("M" + "L".join(f"{x:.1f} {y:.1f}" for x, y in pts) + "Z")
        elif p.geom_type in ("LineString",):
            pts = list(p.coords)
            out.append("M" + "L".join(f"{x:.1f} {y:.1f}" for x, y in pts))
        elif p.geom_type == "MultiLineString":
            for l in p.geoms:
                out.append("M" + "L".join(f"{x:.1f} {y:.1f}" for x, y in l.coords))
    return "".join(out)

adm = load("ne_10m_admin_1_states_provinces.geojson")
teile = {k: [] for k in LAENDER}
nachbarn = []
for f in adm:
    p = f["properties"]; g = shape(f["geometry"])
    if not g.intersects(BBOX):
        continue
    hit = None
    for land, keys in LAENDER.items():
        for a3, nm in keys:
            if p["adm0_a3"] == a3 and (nm == "*" or p["name"] == nm):
                hit = land
    if hit:
        teile[hit].append(g.buffer(0))
    else:
        nachbarn.append(g.buffer(0).intersection(BBOX))

frag = [f'<!-- viewBox 0 0 {W} {H} -->']
frag.append(f'<path class="nachbar" d="{path(unary_union(nachbarn), 0.8)}"></path>')
labels = {}
for land, gs in teile.items():
    u = unary_union(gs).buffer(0.002).buffer(-0.002)
    frag.append(f'<path data-land="{land}" d="{path(u)}"></path>')
    big = max(getattr(u, "geoms", [u]), key=lambda q: q.area)
    lp = polylabel(big, tolerance=0.01)
    labels[land] = xy(lp.x, lp.y)

# Flüsse: Hauptflüsse aus beiden Natural-Earth-Dateien
FLUESSE = {"Rhein", "Rhine", "Elbe", "Donau", "Danube", "Oder", "Main", "Vltava", "Weser", "Inn", "Morava", "Saale", "Neckar", "Spree", "Isar", "Moselle", "Mosel", "Meuse", "Maas", "Havel", "Lech", "Salzach", "Enns", "Dyje", "Svratka", "Luzicka Nisa", "Fulda", "Regnitz", "Ruhr", "Lippe", "Schwarze Elster", "Unstrut", "Sázava", "Ohře", "Eger"}
riv = []
for fn in ("ne_10m_rivers_lake_centerlines.geojson", "ne_10m_rivers_europe.geojson"):
    for f in load(fn):
        nm = f["properties"].get("name") or ""
        if nm in FLUESSE and f["geometry"]:
            g = shape(f["geometry"]).intersection(BBOX)
            if not g.is_empty:
                major = nm in {"Rhein", "Rhine", "Elbe", "Donau", "Danube", "Oder"}
                riv.append((major, g))
frag.append('<path class="fluss" d="' + "".join(path(g, 0.5) for m, g in riv if not m) + '"></path>')
frag.append('<path class="strom" d="' + "".join(path(g, 0.5) for m, g in riv if m) + '"></path>')
seen = set(); lk = []
for fn in ("ne_10m_lakes.geojson", "ne_10m_lakes_europe.geojson"):
    for f in load(fn):
        g = shape(f["geometry"])
        if g.intersects(BBOX) and g.area > 0.004:
            lk.append(g.buffer(0).intersection(BBOX))
frag.append(f'<path class="see" d="{path(unary_union(lk), 0.4)}"></path>')

open(OUT, "w", encoding="utf-8").write("\n".join(frag))
json.dump({"W": W, "H": H, "labels": labels, "hauptstadt": {k: [*xy(v[0], v[1]), v[2]] for k, v in HAUPTSTADT.items()},
           "fluss_labels": {"Rhein": xy(7.6, 50.4), "Elbe": xy(11.9, 52.9), "Donau": xy(14.0, 48.45), "Oder": xy(14.6, 52.6), "Moldau": xy(14.35, 49.5), "Main": xy(9.6, 50.0)}},
          open(OUT + ".json", "w", encoding="utf-8"), ensure_ascii=False, indent=1)
print("H", H, "bytes", sum(len(s) for s in frag))
