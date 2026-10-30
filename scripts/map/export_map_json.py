# Wandelt reichskarte.svgfrag + reichskarte.json in game/client/src/assets/map.json für den Client.
# Aufruf: python -I -X utf8 export_map_json.py <svgfrag> <out.json>
import json, re, sys
frag_path, out = sys.argv[1], sys.argv[2]
frag = open(frag_path, encoding="utf-8").read()
meta = json.load(open(frag_path.replace(".svgfrag", ".json"), encoding="utf-8"))
ids = {"Preußen": 1, "Hessen": 2, "Bayern": 3, "Böhmen": 4, "Sachsen": 5, "Mähren": 6, "Tirol": 7, "Pfalz": 8, "Flandern": 9}
def path(cls):
    m = re.search(r'<path class="' + cls + r'" d="([^"]*)"', frag)
    return m.group(1) if m else ""
lands = [{"id": ids[m.group(1)], "name": m.group(1), "d": m.group(2),
          "label": [round(v) for v in meta["labels"][m.group(1)]],
          "capital": [round(meta["hauptstadt"][m.group(1)][0]), round(meta["hauptstadt"][m.group(1)][1]), meta["hauptstadt"][m.group(1)][2]]}
         for m in re.finditer(r'<path data-land="([^"]+)" d="([^"]+)"', frag)]
data = {"width": meta["W"], "height": meta["H"], "neighbours": path("nachbar"), "rivers": path("fluss"),
        "majorRivers": path("strom"), "lakes": path("see"), "lands": sorted(lands, key=lambda l: l["id"]),
        "riverLabels": {k: [round(v[0]), round(v[1])] for k, v in meta["fluss_labels"].items()}}
# Sachsen-Beschriftung liegt sonst in Thüringen und kollidiert mit Hessen
for l in data["lands"]:
    if l["name"] == "Sachsen":
        l["label"] = [545, 325]
json.dump(data, open(out, "w", encoding="utf-8"), ensure_ascii=False, separators=(",", ":"))
print("ok", len(lands))
