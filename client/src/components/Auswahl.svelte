<script lang="ts" generics="T extends string | number">
  // Auswahlfeld im Spieldesign statt der System-Liste (die sich nicht gestalten lässt).
  // Barrierefrei nach ARIA-Muster „Select-only Combobox“: Knopf mit role=combobox,
  // Liste mit role=listbox; Tastatur: Pfeile, Pos1/Ende, Enter/Leertaste, Esc, Tab.
    let { value = $bindable(), options, label, onchange, dark = false, disabled = false, float = '' }: {
    value: T;
    options: { value: T; label: string; disabled?: boolean }[];
    label: string;
    /** sichtbare schwebende Beschriftung (Kartusche wie bei .feld in app.css) */
    float?: string;
    onchange?: (v: T) => void;
    dark?: boolean;
    disabled?: boolean;
  } = $props();

  const uid = $props.id();
  let open = $state(false);
  let active = $state(0);
  let root: HTMLDivElement;
  let list = $state<HTMLUListElement>();
  // Position der Liste aus dem Knopf (Fensterkoordinaten); reicht der Platz unten nicht, klappt sie nach oben auf.
  let place = $state({ left: 0, top: 0, width: 0, up: false, max: 320 });
  const current = $derived(options.find((o) => o.value === value));

  function show() {
    if (disabled) return;
    active = Math.max(0, options.findIndex((o) => o.value === value));
    const r = root.getBoundingClientRect();
    const below = innerHeight - r.bottom - 12;
    const above = r.top - 12;
    const up = below < 220 && above > below;
    place = { left: r.left, top: up ? r.top - 6 : r.bottom + 6, width: r.width, up, max: Math.min(320, up ? above : below) };
    open = true;
  }

  function choose(i: number) {
    const o = options[i];
    if (!o || o.disabled) return;
    open = false;
    if (o.value !== value) {
      value = o.value;
      onchange?.(o.value);
    }
  }

  // nächster wählbarer Eintrag in Richtung dir
  function step(dir: number) {
    let i = active;
    for (let n = 0; n < options.length; n++) {
      i = (i + dir + options.length) % options.length;
      if (!options[i].disabled) break;
    }
    active = i;
  }

  function keydown(e: KeyboardEvent) {
    if (!open && ['ArrowDown', 'ArrowUp', 'Enter', ' '].includes(e.key)) {
      e.preventDefault();
      show();
      return;
    }
    if (!open) return;
    switch (e.key) {
      case 'ArrowDown': e.preventDefault(); step(1); break;
      case 'ArrowUp': e.preventDefault(); step(-1); break;
      case 'Home': e.preventDefault(); active = 0; break;
      case 'End': e.preventDefault(); active = options.length - 1; break;
      case 'Enter': case ' ': e.preventDefault(); choose(active); break;
      case 'Escape': e.preventDefault(); open = false; break;
      case 'Tab': open = false; break;
    }
  }

  // Liste in die oberste Ebene (Popover-API): so schneiden weder scrollende Karten noch
  // Dialoge (overflow, filter) sie ab, und sie liegt auch über offenen Modals.
  $effect(() => {
    if (open && list && !list.matches(':popover-open')) list.showPopover();
  });

  // Klick außerhalb schließt die Liste
  $effect(() => {
    if (!open) return;
    const close = (e: PointerEvent) => { if (!root.contains(e.target as Node)) open = false; };
    // Scrollt der Container darunter oder ändert sich das Fenster, passt die feste Position nicht mehr
    const away = (e: Event) => { if (e.target !== list) open = false; };
    document.addEventListener('pointerdown', close);
    addEventListener('scroll', away, true);
    addEventListener('resize', away);
    return () => {
      document.removeEventListener('pointerdown', close);
      removeEventListener('scroll', away, true);
      removeEventListener('resize', away);
    };
  });
</script>

<div class="auswahl" class:dark class:open class:float={!!float} bind:this={root}>
  <button type="button" role="combobox" aria-label={label} aria-haspopup="listbox" aria-expanded={open}
    aria-controls="{uid}-list" aria-activedescendant={open ? `${uid}-o${active}` : undefined}
    {disabled} onclick={() => (open ? (open = false) : show())} onkeydown={keydown}>
    <span class="text">{current?.label ?? ''}</span>
    <svg class="chev" aria-hidden="true" width="14" height="10" viewBox="0 0 14 10"><path d="M1 2l6 6 6-6" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" /></svg>
  </button>
  {#if float}<span class="feld-label oben" aria-hidden="true">{float}</span>{/if}
  {#if open}
    <ul id="{uid}-list" role="listbox" aria-label={label} bind:this={list} popover="manual" class:up={place.up}
      style:left="{place.left}px" style:top="{place.top}px" style:min-width="{place.width}px" style:max-height="{place.max}px">
      {#each options as o, i}
        <li id="{uid}-o{i}" role="option" aria-selected={o.value === value} aria-disabled={o.disabled}
          class:active={i === active} class:sel={o.value === value} class:dis={o.disabled}
          onpointerenter={() => !o.disabled && (active = i)} onpointerdown={(e) => { e.preventDefault(); choose(i); }}>
          <span class="mark" aria-hidden="true">{o.value === value ? '◆' : ''}</span>{o.label}
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .auswahl { position: relative; min-width: 0; }
  button {
    width: 100%; min-height: 44px; display: flex; align-items: center; gap: 10px; padding: 0 12px;
    border: 1px solid #a8987c; border-radius: 4px; background: #fbf6ea; color: var(--ink);
    font: inherit; text-align: left; cursor: pointer;
    box-shadow: inset 0 0 0 2px #fbf6ea, inset 0 0 0 3px #cdbb9466;
  }
  button:disabled { opacity: .5; cursor: not-allowed; }
  /* Mit Kartusche: gleiche Höhe und Rahmen wie die Textfelder (.feld) */
  .float button { min-height: 54px; padding-top: 10px; font-size: 18px; border-color: #8a7350; border-radius: 3px;
    background: linear-gradient(180deg, #fdf9ef, #f5ead2); box-shadow: inset 0 0 0 3px #fdf9ef, inset 0 0 0 4px #c9a95c99, inset 0 3px 8px #6b553722; }
  .float.open button, .float button:focus-visible { border-color: var(--blood); }
  .float.open .feld-label, .float:focus-within .feld-label { color: var(--blood); }
  button:focus-visible { outline: 3px solid var(--gold); outline-offset: 2px; }
  .text { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .chev { flex-shrink: 0; color: var(--gold); transition: transform .15s; }
  .open .chev { transform: rotate(180deg); }

  /* Dunkle Variante für die Kopfleiste */
  .dark button { min-height: 36px; background: var(--ink); color: var(--gold-light); border-color: var(--gold);
    font-family: var(--font-label); font-size: 13px; font-weight: 600; box-shadow: inset 0 0 0 2px var(--ink), inset 0 0 0 3px #b08a3e66; }

  ul {
    /* Popover-Vorgaben des Browsers (inset, margin, Hintergrund) zurücksetzen */
    position: fixed; inset: auto; margin: 0; padding: 18px 10px; background: transparent; overflow-x: hidden;
    list-style: none; color: var(--ink); font-family: var(--font-body); font-size: 16px; font-weight: 400;
    /* Zierrahmen wie bei Karten und Dialogen (app.css) */
    border: 16px solid transparent; border-image: var(--frame) 35 fill / 26px / 0 stretch;
    filter: drop-shadow(0 10px 18px #00000055);
    overflow-y: auto; overscroll-behavior: contain;
  }
  ul.up { transform: translateY(-100%); }
  li { display: flex; align-items: center; gap: 8px; min-height: 40px; padding: 0 12px 0 6px; border-radius: 3px; cursor: pointer; white-space: nowrap; }
  li.active { background: #efe0c4; box-shadow: inset 3px 0 0 var(--blood); }
  li.sel { font-weight: 600; }
  li.dis { color: #a8987c; cursor: not-allowed; }
  .mark { width: 12px; color: var(--blood); font-size: 10px; }
</style>
