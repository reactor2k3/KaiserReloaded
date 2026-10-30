"""Lädt die Landeswappen von Wikimedia Commons, säubert sie und schreibt die Lizenzangaben.

Ergebnis in game/client/public/heraldik/laender/:
  <land>-<n>.svg   Wappen-Varianten je Land (ohne Metadaten, Skripte und Ereignis-Attribute)
  lizenzen.json    Datei, Seite, Urheber und Lizenz je Variante (für Auswahl und Bildnachweis im Spiel)
Die Dateien stehen unter ihrer jeweiligen Lizenz (CC BY-SA, CC BY, CC0, gemeinfrei), nicht unter
der Lizenz des Spielcodes. Bei Änderungen an einer Datei gilt deren Lizenz weiter (Share-Alike).
Aufruf (in der Projektwurzel): python scripts/heraldik/commons_laender.py
"""
import json
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]  # Projektwurzel (Ordner mit client/ und server/)
OUT = ROOT / "client" / "public" / "heraldik" / "laender"
UA = {"User-Agent": "KaiserReloaded/1.0 (Open-Source-Spiel; Wappen-Import) python-urllib"}

# Land (Schlüssel wie COUNTRY_FILES im Client) -> Auswahl an Wappen (Datei auf Commons, Urheber,
# wo Commons keinen brauchbaren Autor-Eintrag liefert). Die erste Variante ist die Vorgabe.
# Deutsche Territorien bevorzugt aus „SVG historical coats of arms of Germany“ (Micha-Vorgabe);
# Böhmen, Mähren, Tirol und Flandern stehen dort nicht, sie kommen aus anderen Commons-Kategorien.
LAENDER = {
    "preussen": [("Arms of East Prussia.svg", None), ("Arms of Brandenburg.svg", None),
                 ("Brandenburg Arms.svg", None), ("Small Arms of the Kingdom of Prussia 1790.svg", None)],
    "hessen": [("Arms of the Grand Duchy of Hesse 1806-1918.svg", None), ("Arms of Hesse-Nassau.svg", None),
               ("Coat of arms of Landgraviate Hesse - small.svg", "Gerhard Matzat, ohne Krone bearbeitet von Perhelion")],
    "bayern": [("Arms of Bavaria.svg", "Balmung0731"), ("Coat of arms of the Kingdom of Bavaria.svg", None),
               ("Coat of Arms of the Kingdom of Bavaria 1835-1918.svg", None)],
    "boehmen": [("Bohemia Arms.svg", None), ("Arms of the Kingdom of Bohemia.svg", None),
                ("Lesser coat of arms of the Kingdom of Bohemia (Wenceslaus II of Bohemia).svg", None)],
    "sachsen": [("Arms of Saxony.svg", None), ("Arms of the Kingdom of Saxony 1806.svg", None),
                ("Shield of Saxe-Coburg and Gotha.svg", None), ("Small Arms of the Kingdom of Saxony 1806-1918.svg", None)],
    "maehren": [("Znak Moravy.svg", "TilmannR (Vektorisierung, Vorlage unbekannt)"), ("Moravia Arms.svg", None),
                ("Moravian Eagle.svg", None)],
    "tirol": [("Arms of Tyrol.svg", None), ("Arms of the County of Tyrol.svg", None), ("Tyrol Arms.svg", None)],
    "pfalz": [("Wappen Rheinpfalz.svg", None), ("Wappen Kurpfalz.svg", "Bayernraute")],
    "flandern": [("Arms of Flanders.svg", "Tom-L"), ("CoA Flanders County.svg", None),
                 ("Coat of Arms of the Count of Flanders (according to the Gelre Armorial).svg", None)],
}


def api(**params) -> dict:
    params["format"] = "json"
    url = "https://commons.wikimedia.org/w/api.php?" + urllib.parse.urlencode(params)
    with urllib.request.urlopen(urllib.request.Request(url, headers=UA), timeout=30) as r:
        return json.load(r)


def fetch(url: str) -> bytes:
    """Lädt eine Datei; bei Ratenbegrenzung (429) mit wachsender Pause erneut."""
    for wait in (0, 30, 60, 120, 240):
        time.sleep(wait)
        try:
            with urllib.request.urlopen(urllib.request.Request(url, headers=UA), timeout=60) as r:
                return r.read()
        except urllib.error.HTTPError as e:
            if e.code != 429:
                raise
    raise RuntimeError("Commons lehnt weiter ab (429): " + url)


def clean(svg: str) -> str:
    """Entfernt Metadaten, Kommentare, Skripte, Ereignis-Attribute und externe Verweise."""
    svg = re.sub(r"<\?xml[^>]*\?>|<!DOCTYPE[^>]*>|<!--.*?-->", "", svg, flags=re.S)
    svg = re.sub(r"<(metadata|script|foreignObject)\b.*?</\1>", "", svg, flags=re.S | re.I)
    svg = re.sub(r"<(sodipodi|inkscape):[^>]*/>", "", svg)
    svg = re.sub(r'\s(on\w+|sodipodi:\w+|inkscape:\w+)="[^"]*"', "", svg)
    svg = re.sub(r'\s(xlink:)?href="(?!#)[^"]*"', "", svg)  # nur interne Verweise erlaubt
    return svg.strip() + "\n"


def text(meta: dict, key: str) -> str:
    return re.sub(r"<[^>]+>", "", meta.get(key, {}).get("value", "")).strip()


def main() -> int:
    OUT.mkdir(parents=True, exist_ok=True)
    files = [f for variants in LAENDER.values() for f, _ in variants]
    by_title = {}
    for i in range(0, len(files), 40):  # API erlaubt 50 Titel je Abfrage
        pages = api(action="query", titles="|".join("File:" + f for f in files[i:i + 40]), prop="imageinfo",
                    iiprop="url|extmetadata", iiextmetadatafilter="LicenseShortName|LicenseUrl|Artist")["query"]["pages"]
        by_title.update({p["title"]: p for p in pages.values()})
    credits = {}
    for key, variants in LAENDER.items():
        for n, (file, author) in enumerate(variants):
            info = by_title["File:" + file]["imageinfo"][0]
            meta = info["extmetadata"]
            target = OUT / f"{key}-{n}.svg"
            if not target.exists():  # fortsetzbar: schon geladene Dateien bleiben (bei Ratenbegrenzung)
                time.sleep(6)  # Commons freundlich behandeln
                target.write_text(clean(fetch(info["url"]).decode("utf-8")), encoding="utf-8")
            credits[f"{key}-{n}"] = {
                "datei": file,
                "seite": info["descriptionurl"],
                "urheber": author or re.sub(r"\s+", " ", text(meta, "Artist")) or "siehe Dateiseite",
                "lizenz": text(meta, "LicenseShortName").replace("Public domain", "gemeinfrei"),
                "lizenzUrl": text(meta, "LicenseUrl"),
            }
            print(f"{key}-{n}", credits[f"{key}-{n}"]["lizenz"], "|", credits[f"{key}-{n}"]["urheber"][:60])
    # Dateien ohne Eintrag (alte Benennung, entfernte Varianten) löschen: nichts ohne Nachweis
    for old in OUT.glob("*.svg"):
        if old.stem not in credits:
            old.unlink()
    (OUT / "lizenzen.json").write_text(json.dumps(credits, ensure_ascii=False, indent=1), encoding="utf-8")
    (OUT / "LIZENZEN.md").write_text(markdown(credits), encoding="utf-8")
    return 0


def markdown(credits: dict) -> str:
    """Lizenzübersicht für Menschen (gleiche Angaben wie lizenzen.json)."""
    rows = [f"| {k} | [{c['datei']}]({c['seite']}) | {c['urheber']} | "
            + (f"[{c['lizenz']}]({c['lizenzUrl']})" if c["lizenzUrl"] else c["lizenz"]) + " |"
            for k, c in credits.items()]
    head = [
        "# Landeswappen – Herkunft und Lizenzen",
        "",
        "Die Wappen in diesem Ordner stammen von Wikimedia Commons und stehen unter ihren eigenen Lizenzen,",
        "nicht unter der Lizenz des Spielcodes. Sie werden unverändert verwendet (nur Metadaten entfernt).",
        "Deutsche Territorien aus der Kategorie „SVG historical coats of arms of Germany“, Böhmen, Mähren,",
        "Tirol und Flandern aus anderen Commons-Kategorien. Maschinenlesbar: `lizenzen.json`.",
        "Neu laden: `python scripts/heraldik/commons_laender.py` (in der Projektwurzel).",
        "",
        "| Variante | Datei | Urheber | Lizenz |",
        "|---|---|---|---|",
    ]
    foot = [
        "",
        "Hinweis: Wappen bestehender Gebietskörperschaften können unabhängig vom Urheberrecht als",
        "Hoheitszeichen geschützt sein. Das Spiel zeigt historische Landeswappen in einem Spiel über das",
        "17. Jahrhundert; es erweckt keinen amtlichen Eindruck.",
    ]
    return "\n".join(head + rows + foot) + "\n"


if __name__ == "__main__":
    sys.exit(main())
