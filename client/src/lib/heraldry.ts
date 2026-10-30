// Landeswappen: freie SVG-Dateien von Wikimedia Commons (public/heraldik/laender, Lizenzen in
// lizenzen.json und LIZENZEN.md, Import über scripts/heraldik/commons_laender.py).

/** Dateischlüssel je Land (Index wie COUNTRIES); Varianten liegen als <schlüssel>-<n>.svg vor */
export const COUNTRY_FILES = ['', 'preussen', 'hessen', 'bayern', 'boehmen', 'sachsen', 'maehren', 'tirol', 'pfalz', 'flandern'];

/** Pfad eines Landeswappens (Variante 0 = Vorgabe) */
export function countryArmsSrc(country: number, variant = 0): string {
  return `/heraldik/laender/${COUNTRY_FILES[country] || 'sachsen'}-${Math.max(0, variant)}.svg`;
}

let creditsCache: Promise<Record<string, ArmsCredit>> | null = null;

/** Bildnachweise aller Varianten (lizenzen.json), einmal geladen; Schlüssel "<land>-<n>" */
export function loadArmsCredits(): Promise<Record<string, ArmsCredit>> {
  creditsCache ??= fetch('/heraldik/laender/lizenzen.json')
    .then((r) => (r.ok ? r.json() : {}))
    .catch(() => { creditsCache = null; return {}; });
  return creditsCache;
}

/** Anzahl der Varianten eines Landes laut Bildnachweis */
export function variantCount(credits: Record<string, ArmsCredit>, country: number): number {
  const key = COUNTRY_FILES[country];
  return Object.keys(credits).filter((k) => k.startsWith(key + '-')).length;
}

/** Bildnachweis eines Landeswappens (vom Import-Skript) */
export interface ArmsCredit { datei: string; seite: string; urheber: string; lizenz: string; lizenzUrl: string }

/** Hausfarben je Land (Banner der Krönungsfanfaren): Feld und Kopfleiste, nach den Landeswappen */
export const COUNTRY_COLORS: Record<number, { field1: string; field2: string }> = {
  1: { field1: '#f4f1e8', field2: '#1d1a17' }, // Preußen: Schwarz auf Silber
  2: { field1: '#22508f', field2: '#b3262b' }, // Hessen: Rot-silberner Löwe auf Blau
  3: { field1: '#f4f1e8', field2: '#22508f' }, // Bayern: Weiß-blaue Rauten
  4: { field1: '#b3262b', field2: '#f4f1e8' }, // Böhmen: Silberner Löwe auf Rot
  5: { field1: '#e5b93f', field2: '#1d1a17' }, // Sachsen: Schwarz-Gold mit Rautenkranz
  6: { field1: '#22508f', field2: '#b3262b' }, // Mähren: Rot-silbern geschachter Adler auf Blau
  7: { field1: '#f4f1e8', field2: '#b3262b' }, // Tirol: Roter Adler auf Silber
  8: { field1: '#1d1a17', field2: '#e5b93f' }, // Pfalz: Goldener Löwe auf Schwarz
  9: { field1: '#e5b93f', field2: '#1d1a17' }, // Flandern: Schwarzer Löwe auf Gold
};
