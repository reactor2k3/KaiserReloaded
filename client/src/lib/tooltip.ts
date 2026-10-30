// Ein gemeinsamer Tooltip im Spieldesign für die ganze Oberfläche.
// Jedes Element mit title (oder data-tip) bekommt ihn automatisch: Beim ersten Überfahren wird
// title nach data-tip verschoben, damit der graue System-Tooltip nicht erscheint. Der Tooltip
// liegt per Popover-API in der obersten Ebene (auch über offenen Modals) und sitzt unter dem
// Element, bei Platzmangel darüber. Tastatur: erscheint auch bei Fokus, Esc schließt.

let tip: HTMLDivElement | null = null;
let current: Element | null = null;
let timer: ReturnType<typeof setTimeout> | undefined;

/** Text des Tooltips; title wird dabei übernommen und entfernt. */
function textOf(el: Element): string {
  const t = el.getAttribute('title');
  if (t !== null) {
    el.removeAttribute('title');
    if (t) el.setAttribute('data-tip', t);
    // Ohne sichtbaren Text bleibt der Hinweis für Vorleseprogramme als Name erhalten
    if (t && !el.getAttribute('aria-label') && !el.textContent?.trim()) el.setAttribute('aria-label', t);
  }
  return el.getAttribute('data-tip') ?? '';
}

function place(el: Element) {
  if (!tip) return;
  const r = el.getBoundingClientRect();
  const t = tip.getBoundingClientRect();
  const below = r.bottom + 8 + t.height <= innerHeight;
  const left = Math.min(Math.max(6, r.left + r.width / 2 - t.width / 2), innerWidth - t.width - 6);
  tip.style.left = `${left}px`;
  tip.style.top = `${below ? r.bottom + 8 : r.top - 8 - t.height}px`;
  tip.classList.toggle('oben', !below);
  // Zeiger auf die Mitte des Elements ausrichten
  tip.style.setProperty('--zeiger', `${Math.min(Math.max(12, r.left + r.width / 2 - left), t.width - 12)}px`);
}

function show(el: Element, delay: number) {
  clearTimeout(timer);
  const text = textOf(el);
  if (!text || !tip) return hide();
  current = el;
  timer = setTimeout(() => {
    if (!tip || current !== el || !el.isConnected) return;
    tip.textContent = text;
    if (!tip.matches(':popover-open')) tip.showPopover();
    place(el);
    tip.classList.add('sichtbar');
  }, delay);
}

function hide() {
  clearTimeout(timer);
  current = null;
  if (tip?.matches(':popover-open')) {
    tip.classList.remove('sichtbar');
    tip.hidePopover();
  }
}

const target = (e: Event) => (e.target instanceof Element ? e.target.closest('[title], [data-tip]') : null);

/** Einmal beim Start aufrufen (main.ts). */
export function initTooltips() {
  tip = document.createElement('div');
  tip.className = 'tooltip';
  tip.setAttribute('role', 'tooltip');
  tip.setAttribute('popover', 'manual');
  document.body.append(tip);

  document.addEventListener('pointerover', (e) => {
    const el = target(e);
    if (el && el !== current) show(el, 350);
    else if (!el && current) hide();
  });
  document.addEventListener('pointerout', (e) => {
    if (current && !(e.relatedTarget instanceof Node && current.contains(e.relatedTarget))) hide();
  });
  document.addEventListener('focusin', (e) => {
    const el = target(e);
    if (el) show(el, 0);
  });
  document.addEventListener('focusout', hide);
  document.addEventListener('pointerdown', hide);
  document.addEventListener('keydown', (e) => { if (e.key === 'Escape') hide(); });
  addEventListener('scroll', hide, true);
}
