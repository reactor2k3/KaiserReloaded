# KaiserReloaded

Ein Browser-Remake des C64-Klassikers **„Kaiser“**: Als Herr oder Herrin eines von neun Ländern im
Heiligen Römischen Reich wirtschaften Sie mit Korn, Land, Steuern und Bauten, führen Kriege und
Bündnisse und steigen bis zur Kaiserkrone auf – allein gegen Computer-Herrscher oder mit bis zu
sechs Spielern je Partie, live oder asynchron über Tage.

Ergänzt um Bausteine aus **Kaiser II** (1987, Public Domain): Zufallsereignisse, Volksstimmung und
Aufstände, Bündnisse, Kredite und Anleihen, Flotten und Kolonien.

- Server: Go (autoritativ, WebSocket, SQLite) · Client: Svelte 5, TypeScript, Vite
- Ein einziges Docker-Image (siehe unten)
- Lizenz: MIT (Code), Bestandteile von Dritten siehe `DRITTANBIETER.md`
- Regeln: in der Spielhilfe (Knopf „?“ im Spiel)

## Aufbau

| Ordner | Inhalt |
|---|---|
| `data/` | `world.json` (Länder, Titel, Angriffswege), `balance.json` (alle einstellbaren Zahlen, `classic` = Original-Regeln) |
| `server/` | Go-Modul. `internal/engine` = reine Spielregeln ohne I/O, deterministisch über `Rng` (Seed) |
| `client/` | TypeScript, Vite 8, Svelte 5 (Runes). Lobby, Hauptbildschirm in drei Spalten (Reichskarte, Aktionsspalte der Phase, eigenes Reich), Modals für Hinweise, Profil und Aufstellung |

## Spielen (lokal)

```powershell
.\dev.ps1    # startet kaiserd (8080) und Vite (5173) in eigenen Fenstern
```

Dann `http://localhost:5173` öffnen, Spiel anlegen, Einladungslink in einem zweiten Tab öffnen und die Anfrage im ersten Tab zulassen. Jeder Tab ist ein eigener Herrscher (Sitzung im sessionStorage, übersteht Neuladen).

## Betrieb mit Docker

Ein einziges, komplettes Image (Spielserver + gebauter Client, keine weiteren Dienste). Fertig gebaut
liegt es in der GitHub Container Registry (gebaut von `.github/workflows/docker.yml` bei jedem Push auf `main`):

```bash
docker run -d --name kaiser --restart unless-stopped -p 8080:8080 -v kaiser-data:/var/lib/kaiser ghcr.io/reactor2k3/kaiserreloaded:latest
```

oder `docker compose up -d` mit der beiliegenden `docker-compose.yml`. Selbst bauen: `docker build -t kaiserreloaded .`.
Danach läuft das Spiel unter `http://<server>:8080`.

- `Dockerfile`: baut Client (Node 24) und Server (Go 1.27, ohne cgo) und packt beides in ein distroless-Image, das als Nicht-root läuft. Healthcheck über `/healthz`.
- kaiserd setzt ohne Reverse-Proxy selbst Sicherheits-Header, gzip für Textdateien und Cache-Regeln (`internal/server/web.go`) und beendet sich bei `docker stop` sauber.
- Spielstände, Ruhmeshalle und Uploads liegen im Volume `kaiser-data` (`/var/lib/kaiser`).
- HTTPS: Port 8080 hinter einen vorhandenen Reverse-Proxy oder Load-Balancer legen; der WebSocket `/ws` muss durchgereicht werden.
### Grenzen und Schutz

Voreingestellt, per Kommandozeile änderbar (in Docker: an `command:` bzw. den `docker run`-Aufruf anhängen):

| Flag | Vorgabe | Wirkung |
|---|---|---|
| `-max-games` | 10 | gleichzeitig laufende Spiele; danach zeigt die Lobby „keine weiteren Spiele möglich“ (Beitreten geht weiter) |
| `-max-players` | 6 | Herrscher je Spiel (inkl. Computer-Gegner) |
| `-max-conns-ip` | 8 | gleichzeitige WebSocket-Verbindungen je IP |
| `-max-conns` | 400 | gleichzeitige WebSocket-Verbindungen insgesamt |
| `-trust-proxy` | aus | Client-IP aus `X-Forwarded-For` lesen – **nur** hinter dem eigenen Reverse-Proxy einschalten, sonst sind die IP-Grenzen fälschbar |

Als „laufend“ zählt eine offene Lobby 6 Stunden, ein gestartetes Spiel bis 14 Tage nach dem letzten Zug; beendete Spiele nie.
Dazu kommen feste Bremsen je IP: 20 HTTP-Anfragen/s, 20 Spielnachrichten/s (Dauerflut trennt die Verbindung), höchstens 3 neue
Spiele auf einmal und danach eines je 5 Minuten, Uploads gedrosselt. Außerdem: Content-Security-Policy, Header-/Leerlauf-Timeouts,
Größenlimits für Nachrichten und Uploads, Origin-Prüfung am WebSocket und `/api/status` für die Auslastung.

Gegen Injektion und Spam: Datenbankzugriffe nur mit Platzhaltern, Texte werden im Client nie als HTML gesetzt,
hochgeladene SVG-Wappen laufen durch eine Positivliste und werden mit `sandbox`-CSP ausgeliefert. Namen und
Wahlsprüche prüft ein gemeinsamer Filter (`engine/text.go`): keine unsichtbaren oder richtungsdrehenden Zeichen,
keine gestapelten Akzente, keine Links. Der Gastgeber kann vor dem Start Herrscher aus der Lobby entfernen;
entfernte Spieler sind für dieses Spiel per IP gesperrt.

- Backup: `docker run --rm -v kaiser-data:/d -v "$PWD":/b alpine tar czf /b/kaiser-backup.tgz -C /d .`

## Protokoll (WebSocket `/ws`)

Client → Server: `create`, `join`, `rejoin` (Spiel-ID + Token), `games` (Stand mehrerer gemerkter Plätze, höchstens 20), `start` (nur Gastgeber), `act` (Spielzug).
Server → Client: `welcome`, `lobby`, `public` (nach dem ersten Vollstand nur geänderte Herrscher), `self` (nur an Betroffene), `events` (Chronik), `games` (Stand für „Meine Spiele“, Details nur bei passendem Token), `error` (Regelverstoß im Klartext).

## Entwicklung

Voraussetzungen: Go 1.27 und Node 24 (für `dev.ps1` optional ein portables Go über `$env:KAISER_GO`).

```powershell
cd server
gofmt -l .        # muss leer bleiben
go vet ./...
go test ./...
```

## Stand Engine

Fertig mit Tests: Ernte, Markt (Korn/Land), Volk versorgen, Einkünfte und Sold, Steuern und Justiz, Prüfungen nach den Steuern (Bankrott, Amtsenthebung, Landmangel), Bauen, Rekrutieren, Manöver, Jahresende (Titel, Zeitstrafe, Tod, Zins, Moral).
Spielablauf mit gleichzeitigen Phasen, Zeitlimit je Spiel und Jahreswechsel steht (`engine/game.go`), Server und Client spielen ein komplettes Jahr mit zwei Herrschern durch.
Spielstände liegen in `server/kaiser.db` (SQLite, reiner Go-Treiber `modernc.org/sqlite`) und werden nach jedem Zug geschrieben; nach einem Neustart laufen alle Spiele weiter, laufende Phasen bekommen ihre Zugzeit neu.
Lobby zeigt vergebene Länder schon vor dem Beitritt (`peek`), Söldner sind im Bau-Dialog wählbar.
Krieg (`engine/war.go`, `engine/battle.go`): Erklärung in der Bauphase (ab Baron), am Jahresende Diplomatie der Unbeteiligten, Wegsuche über die Original-Routen, Leihtruppen, Schlachtfeld-Generator, Gefecht nach Original-Regeln, Abrechnung. Taktischer Modus mit Aufstellung im Browser, schneller Modus automatisch.
Zum Testen von Kriegen: eigene `balance.json` mit `"title": 2` per `-data` übergeben (Herrscher starten als Baron).
Schlachtbericht spielt die Schlacht als Animation ab (Pause, Überspringen, Nochmal).
Profil (`engine/profile.go`, `internal/media`): Wahlspruch, Wappen: Landeswappen oder eigenes hochgeladenes Wappen (den früheren Baukasten gibt es nicht mehr; alte Baukasten-Wappen zeigen das Landeswappen). Uploads per `POST /upload` (Spiel-ID + Token), Spielerbild nur PNG/JPEG/WebP, Wappen zusätzlich SVG. Rasterbilder werden quadratisch auf 512 px neu als PNG kodiert (Metadaten weg), SVG über Allowlist neu geschrieben (keine Skripte, Event-Attribute, externen Verweise). Auslieferung unter `/media/` mit `nosniff` und strenger CSP, Dateien in `server/media/`.
Markt: Neben der freien Menge gibt es Direktkauf-Knöpfe (Korn 5.000 / 10.000 / 20.000 Maß, Land 1.000 / 2.000 / 5.000 ha); sie sind gesperrt, wenn die Menge über dem Jahreslimit bzw. dem freien Land liegt.

Rat des Kanzlers: Zu Beginn jedes Schritts (Markt, Volk, Steuern, Bauen) öffnet sich der Rat automatisch, sobald Ereignis-, Risiko- und Schlachtfenster geschlossen sind. Bei aktivem Autopilot öffnen sich weder Rat noch Ereignis-, Risiko- oder Schlachtfenster (alles steht in der Chronik; die Krönung erscheint weiter). Abschaltbar im Rat-Fenster („Bei jedem Schritt automatisch zeigen“, Vorgabe an, gemerkt im Browser).

Spieleliste und Beitritt (`server/internal/server/beitritt.go`, `GET /api/games`): Die Startseite listet alle laufenden Spiele, beitretbare zuerst. Beitreten (aus der Liste, per Code oder Einladungslink) ist immer eine Anfrage: Der Gastgeber bekommt ein Fenster mit „Zulassen“ / „Ablehnen“ (auch mitten im Spiel, mit Ton und bei verstecktem Tab Browser-Meldung). Beitreten geht vor dem Start und im laufenden Spiel, bis der erste Herrscher befördert wurde; wer mitten im Jahr dazukommt, steigt zum nächsten Jahr ein und wird beim laufenden Jahresende nicht abgerechnet. Höchstens 8 offene Anfragen je Spiel, drei Ablehnungen sperren die IP für das Spiel. Anfragen liegen nur im Speicher (nach Server-Neustart neu anfragen).

Spielmodus (Lobby, `Settings.Mode`): **Live** (Zugzeit unbegrenzt oder 60–300 s je Phase, Partie ruht ohne verbundene Spieler) oder **Asynchron** (1 h, 6 h, 12 h, 24 h oder 3 Tage je Phase, ohne Lebenszeit-Strafe; Partie läuft auch ohne Spieler weiter). Im asynchronen Spiel zeigt der Tab-Titel „● Sie sind dran“, und auf Wunsch (Knopf „Benachrichtigen“) kommt eine Browser-Benachrichtigung (`lib/benachrichtigung.ts`). Restzeiten erscheinen ab einer Stunde als „5 h 07 min“ bzw. „2 T 3 h“.

Favicon-Set (`client/public`): `favicon.svg` (Krone auf dunklem Siegel), `favicon.ico`, PNG 32/180/192/512, `site.webmanifest`, Theme-Farbe.

Lizenz: Code unter MIT (`LICENSE`); Bestandteile von Dritten (Wappen, Kartendaten, Schriften) in `DRITTANBIETER.md`. Nur dieser Ordner (`game/`) ist für das öffentliche Repository gedacht – die Analysedateien zu den Originalspielen daneben nicht.

Sterblichkeit (Lobby, `Settings.Mortality`): unsterblich, niedrig, normal (Original) oder hoch; verschiebt die Lebenserwartung und die jährliche Sterbechance danach. Exil und Attentat gibt es weiterhin.

Computer-Gegner (Lobby, `Settings.Difficulty`, `engine/botplan.go` botSkills): leicht (Korn bis 110 %, die Hälfte der Jahre ohne Investitionen, höchstens zwei Aufträge je Jahr, ohne Titelplanung), mittel (bis 125 %, 30 % Trödeljahre, zwei Aufträge, mit Planung), schwer (volle Planung). Der Autopilot menschlicher Spieler spielt immer voll. Simulation (Kaufmann gegen Feldherr, je 40 Partien): Kaiser im Median um 1768 / 1750 / 1744. `go run ./cmd/simulate -difficulty leicht`.

Computer-Herrscher (`engine/bot.go`): Spielweisen „Kaufmann“ und „Feldherr“, Schritt für Schritt über normale Aktionen. In der Lobby per „Computer-Herrscher hinzufügen“, im Spiel als „Autopilot“ für den eigenen Herrscher (Takt 0,9 s je Zug, `server.BotStep`). Offline-Auswertung: `go run ./cmd/simulate -seed 42 > verlauf.json`. Planung in `engine/botplan.go`: Rücklage nach Titel und laufenden Kosten, kein Kauf, der den Aufstieg dieses Jahres kostet, Zielfläche für Bauland (ab Graf 14 000 ha, ab Fürst 26 000 ha), Bedingungen der hohen Titel zuerst (Palast, Kathedrale, 25 Märkte, 15 Mühlen), Märkte und Mühlen notfalls auf Kredit (Rendite über Dispozins), Korn großzügig für Einwanderer, wenn der Preis passt, Verteidigung nur gegen Gegner ab Baron, Bündnisse (Kaufmann bis zwei, Feldherr einen Partner), Feldherren rüsten nur auf, wenn ein Gegner samt Verbündeten in Reichweite ist, und greifen den landreichsten an, der samt Verbündeten deutlich schwächer ist. Simulation (je 50 Partien): Feldherr gegen Kaufmann 42 : 58 %, zu viert beide Feldherren zusammen 24 %, keine Bankrotte; Kaiser meist um 1740–1748.
Landeswappen: freie SVGs von Wikimedia Commons, je Land 2–4 Varianten zur Auswahl im Profil (`public/heraldik/laender/<land>-<n>.svg`; deutsche Territorien aus „SVG historical coats of arms of Germany“, Böhmen/Mähren/Tirol/Flandern aus anderen Commons-Kategorien; keine eigenen oder nachgezeichneten Bilder). Urheber und Lizenzen in `LIZENZEN.md`/`lizenzen.json`, Quelle der gewählten Variante im Profil und alle im Bildnachweis der Hilfe. Import, Säuberung und Lizenzübersicht: `python scripts/heraldik/commons_laender.py` (fortsetzbar bei Ratenbegrenzung). Gewählte Variante: `ArmsSpec.variant` (Server prüft 0–7).

Vorschau der Landeswappen im Dev-Server: `http://localhost:5173/wappen.html`.
Startseite „Meine Spiele“: der Browser merkt sich alle eigenen Plätze (localStorage `kaiser.games`), zeigt Stand und „Sie sind am Zug“ (alle 20 s aktualisiert), Fortsetzen und Entfernen aus der Liste.
Ratgeber „Der Kanzler rät“ (`client/src/lib/ratgeber.ts`): wichtigster Rat in der Fußleiste, alle Ratschläge im Modal „Rat des Kanzlers“; warnt vor Landmangel (Amtsenthebung), Kornmangel und Bankrott, rät zu Bauten und nennt die schwächsten Bereiche für den nächsten Titel.
Eingabefelder: schwebende Beschriftung als Kartusche mit Fleurons (`.feld` / `.feld-label` in `app.css`, Auswahllisten mit `float`), auch im C64-Skin.
Geteilte Länder (Remake): bis zu drei Herrscher je Land, höchstens 12 je Partie. Jedes Land hat `countryArea` Hektar (balance.json, 50.000); Land kaufen geht nur aus der freien Fläche des eigenen Landes, Eroberungen sind davon ausgenommen. Herrscher im selben Land führen Krieg ohne Durchmarsch; ein geteiltes Transitland ist nur passierbar, wenn alle dort zustimmen. Die Karte teilt das Land in Streifen nach Hektaranteil (`client/src/lib/teilung.ts`), die bei Änderungen weich wandern; Farben je Herrscher in Abstufungen der Landesfarbe. Original-Regeln: weiter ein Herrscher je Land.
Tooltips: ein gemeinsamer Tooltip im Spieldesign (`client/src/lib/tooltip.ts`) für alle Elemente mit `title` oder `data-tip`, auch auf der Karte; per Popover über Modals, mit Tastaturfokus.
Bauten auf der Karte: Palast und Kathedrale (Größe nach Baufortschritt) sowie bis zu je 10 Märkte und Mühlen als Symbole im Gebiet des Herrschers, mit Tooltip; neue Bauten ploppen auf. Daten über `PublicPlayer` (markets, mills, palace, cathedral).
Kartenzoom: Mausrad (zum Zeiger), Ziehen zum Verschieben, Doppelklick, Knöpfe +/−/Mein Gebiet/Ganze Karte; 1- bis 6-fach über die viewBox.
Krönungsurkunde (`Kroenung.svelte`): prunkvolles Modal beim Spielende mit Reichskrone, Strahlenkranz, Bildnis, Wappen, Wahlspruch, Bilanz und huldigenden Fürsten; öffnet einmal je Partie mit Fanfare, danach über die Fußleiste.
Ruhende Partien: Ist kein Spieler verbunden, ruht die Partie (keine Zugzeit, keine Computerzüge); beim Wiederverbinden wird die Zugzeit um die Ruhezeit verlängert.
Kaiser-II-Ereignisse (`engine/events.go`, Client `Ereignis.svelte`): je Herrscher und Jahr 50 % Chance; davon ein Viertel Risiko-Entscheidungen (Ja/Nein in der Marktphase, unbeantwortet = Nein), sonst 26 feste Ereignisse um Taler, Land, Korn und Soldaten. Beträge wachsen mit dem Titel. Nicht in den Original-Regeln.
Volksstimmung (`engine/stimmung.go`): Stimmung 0–10 (Start 5) nach Versorgung, Steuern, Justiz, Krieg und Zufall; wirkt auf die Steuern (85–115 %). Innenpolitik in der Steuerphase: Volksfest (+1), Opposition verhaften (+3, einmal im Jahr), Gendarmen (100 Taler, Sold 20, höchstens einer je 40 Einwohner). Umstürze: Revolution bei Stimmung 0–1, Militärputsch bei Moral unter 25 %, Attentat 1,5 % im Jahr; Gendarmen schützen. Scheitert man, geht es ins Exil (Spielende für diesen Herrscher). Balance-Test `mood_balance_test.go`.
Heer (`engine/heer.go`): Ausbildung und Material je 0–10 (Bauphase), Schlagkraft-Faktor 1 + 3 % × Ausbildung + 5 % × Material im Gefecht; Rekruten verdünnen beides, Ausbildung verfällt jährlich mit 30 %.

Bündnisse (`engine/buendnis.go`): Verteidigungsabkommen über „Außenpolitik“ in der Bauphase anbieten oder kündigen, Angebote jederzeit annehmen; Angebote verfallen zum Jahresende. Verbündete stehen einem Angegriffenen automatisch bei, die Kriegserklärung warnt vor den Verbündeten des Ziels. Bots antworten sofort und lehnen nur den stärksten Rivalen ab.

Finanzen (`engine/finanzen.go`, Modal „Finanzen“ in Markt- und Bauphase, nur Remake-Regeln): Bankkredit bis zur Bonität ((Jahr − Start + 10) × 1000 + Hektar / 2) zu 8 %, mit offenem Kredit kein Titelaufstieg. Reichsanleihen 250 Taler, 18 Taler Zins im Jahr. Herrscheranleihen anderer Spieler zahlen je Stück 1 ‰ ihrer Staatseinnahmen (Steuern, Märkte, Mühlen), höchstens 100 Stück je Herrscher, Rückgabe zu 90 %, wertlos bei Tod oder Exil. Spenden und Kredite an Mitspieler (0–15 % Zins, 1–10 Jahre, gleiche Raten), vorzeitige Rückzahlung jederzeit. Geldgeschäfte nutzen nur echtes Guthaben, keinen Dispo. Die Jahresabrechnung steht in der Chronik.

Flotte und Kolonien (`engine/kolonien.go`, Modal „Kolonien“ in der Bauphase, nur Remake-Regeln): Handelsschiffe (8000 Taler, 400 Unterhalt) und Kriegsschiffe (4000, 200), je Jahr höchstens Einwohner / 200 Schiffe. Je Kontinent eine Kolonialflotte; ein Handelsschiff bedient 4000 km² Kolonialland (1200 Taler je 1000 km²), Ertrag bei der Steuer etwa die Hälfte der bedienten Fläche in Talern (zählt auch für Anleihen). Seekrieg: einmal im Jahr Kriegsschiffe aus dem Heimathafen gegen die Kolonialflotte eines Nicht-Verbündeten; siegt der Angreifer, gehen Handelsschiffe verloren und die Flotte segelt heim. Schiffsereignisse (Prisen, Freibeuter, Sturm, Piraten) nur mit passenden Schiffen. Kaufmann-Bots bauen Handelsflotten und Kolonien, Feldherren Kriegsschiffe für Kaperfahrten. Flotte und Kolonialland stehen in der Reichsübersicht und im Spieler-Tooltip; der Ratgeber meldet ungenutzte Handelsschiffe, freie Kolonialdeckung, schwach bewachte Flotten und volle Kassen (Reichsanleihen).
Offen: stärkere KI (Diplomatie, Verteidigung), Deployment mit Docker.
