// Darstellung umschalten: „barock“ (Standard) oder „c64“ (Retro-Hommage).
// Die Wahl gilt nur für diesen Browser (localStorage) und setzt data-skin am <html>-Element;
// alle Farben und Schriften dazu stehen in app.css.

export type Skin = 'barock' | 'c64';
const KEY = 'kaiser.skin';

function load(): Skin {
  try {
    return localStorage.getItem(KEY) === 'c64' ? 'c64' : 'barock';
  } catch {
    return 'barock';
  }
}

export const skin = $state({ value: load() as Skin });

/** Wendet die Darstellung an und merkt sie sich. */
export function setSkin(s: Skin) {
  skin.value = s;
  if (typeof document !== 'undefined') document.documentElement.dataset.skin = s;
  try { localStorage.setItem(KEY, s); } catch { /* privater Modus */ }
}

/** Beim Start die gemerkte Darstellung setzen. */
export function initSkin() {
  if (typeof document !== 'undefined') document.documentElement.dataset.skin = skin.value;
}
