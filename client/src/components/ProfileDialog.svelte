<script lang="ts">
  // Profil-Dialog: Spielerbild, Wahlspruch und Wappen (Landeswappen oder eigenes hochgeladenes).
  // Wahlspruch und Wechsel zum Landeswappen werden erst nach einer echten Änderung speicherbar;
  // Uploads übernimmt der Server sofort nach Prüfung.
  import { untrack } from 'svelte';
  import Modal from './Modal.svelte';
  import Wappen from './Wappen.svelte';
  import { app } from '../lib/state.svelte';
  import { act, upload } from '../lib/net';
  import type { ArmsSpec } from '../lib/types';
  import { loadArmsCredits, variantCount, COUNTRY_FILES, type ArmsCredit } from '../lib/heraldry';
  import { setSkin, skin } from '../lib/skin.svelte';

  let { open = $bindable(false) }: { open?: boolean } = $props();

  const p = $derived(app.self!.player);
  let motto = $state('');
  let tab = $state<'country' | 'upload'>('country');
  // Gewählte Variante des Landeswappens und Bildnachweise für die Auswahl
  let variant = $state(0);
  let credits = $state<Record<string, ArmsCredit>>({});
  const count = $derived(Math.max(1, variantCount(credits, p.country)));
  const credit = $derived(credits[`${COUNTRY_FILES[p.country]}-${variant}`]);
  let busy = $state(false);

  // Nur beim Öffnen den gespeicherten Stand übernehmen (spätere Server-Updates überschreiben keine Eingaben).
  $effect(() => {
    if (open) untrack(() => {
      motto = p.motto;
      tab = p.arms.kind === 'upload' ? 'upload' : 'country';
      variant = p.arms.kind === 'country' ? p.arms.variant ?? 0 : 0;
      loadArmsCredits().then((c) => (credits = c));
    });
  });

  // Uploads speichert der Server sofort; hier bleibt nur der Wechsel zurück zum Landeswappen
  const armsChanged = $derived(tab === 'country' && (p.arms.kind !== 'country' || (p.arms.variant ?? 0) !== variant));
  const changed = $derived(motto.trim() !== p.motto || armsChanged);
  const mottoValid = $derived(motto.trim().length <= 40);

  function save() {
    const arms: ArmsSpec | undefined = armsChanged ? { kind: 'country', variant, media: '' } : undefined;
    act({ type: 'setProfile', motto: motto.trim(), arms });
  }

  async function onFile(kind: 'avatar' | 'arms', e: Event) {
    const input = e.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    busy = true;
    await upload(kind, file);
    busy = false;
    input.value = '';
  }
</script>

<Modal bind:open title="Profil und Wappen" wide>
  <div class="cols">
    <section class="col" aria-labelledby="pf-ruler">
      <h3 id="pf-ruler" class="label">Herrscher</h3>
      <div class="avatar-row">
        {#if p.avatar}
          <img class="avatar" src="/media/{p.avatar}" alt="Ihr Spielerbild" width="104" height="104" />
        {:else}
          <span class="avatar initial" aria-hidden="true">{p.name.slice(0, 1).toUpperCase()}</span>
        {/if}
        <div class="stack">
          <label class="btn file">Bild hochladen
            <input type="file" accept="image/png,image/jpeg,image/webp" disabled={busy} onchange={(e) => onFile('avatar', e)} />
          </label>
          <span class="hint">PNG, JPEG oder WebP, max. 1 MB. Wird quadratisch auf 512 px zugeschnitten.</span>
        </div>
      </div>
      <label class="feld">
        <input bind:value={motto} maxlength="40" placeholder="z. B. Mit Korn und Krone" />
        <span class="feld-label">Wahlspruch</span>
      </label>
      {#if !mottoValid}<p class="error">Höchstens 40 Zeichen.</p>{/if}
      <fieldset class="skin">
        <legend class="label">Darstellung (nur in diesem Browser)</legend>
        <label><input type="radio" name="skin" checked={skin.value === 'barock'} onchange={() => setSkin('barock')} /> Barock</label>
        <label><input type="radio" name="skin" checked={skin.value === 'c64'} onchange={() => setSkin('c64')} /> C64 (Retro)</label>
      </fieldset>
    </section>

    <section class="col wide" aria-labelledby="pf-arms">
      <div class="head">
        <h3 id="pf-arms" class="label">Wappen</h3>
        <div class="tabs" role="tablist" aria-label="Wappen erstellen">
          <button type="button" role="tab" aria-selected={tab === 'country'} class:active={tab === 'country'} onclick={() => (tab = 'country')}>Landeswappen</button>
          <button type="button" role="tab" aria-selected={tab === 'upload'} class:active={tab === 'upload'} onclick={() => (tab = 'upload')}>Hochladen</button>
        </div>
      </div>
      <div class="arms">
        <div class="preview">
          {#if tab === 'country'}<Wappen country={p.country} {variant} size={150} label="Landeswappen" />
          {:else}<Wappen arms={p.arms.kind === 'upload' ? p.arms : undefined} country={p.country} size={150} label="Hochgeladenes Wappen" />{/if}
        </div>
        <div class="options">
          {#if tab === 'country'}
            <p>Wählen Sie eines der überlieferten Wappen Ihres Landes. Ein eigenes Wappen können Sie unter „Hochladen“ hinzufügen.</p>
            <!-- Auswahl der Varianten; Bildnachweis der gewählten darunter (Pflicht der freien Lizenzen) -->
            <div class="varianten" role="radiogroup" aria-label="Landeswappen wählen">
              {#each Array.from({ length: count }, (_, i) => i) as i (i)}
                <button type="button" role="radio" aria-checked={variant === i} class:on={variant === i}
                  title={credits[`${COUNTRY_FILES[p.country]}-${i}`]?.datei ?? ''} onclick={() => (variant = i)}>
                  <Wappen country={p.country} variant={i} size={56} label="Variante {i + 1}" />
                </button>
              {/each}
            </div>
            {#if credit}
              <p class="hint">Quelle: <a href={credit.seite} target="_blank" rel="noopener">{credit.datei}</a> von {credit.urheber},
                {#if credit.lizenzUrl}<a href={credit.lizenzUrl} target="_blank" rel="noopener">{credit.lizenz}</a>{:else}{credit.lizenz}{/if}</p>
            {/if}
          {:else}
            <label class="btn file">Wappen hochladen
              <input type="file" accept="image/svg+xml,image/png,image/jpeg,image/webp" disabled={busy} onchange={(e) => onFile('arms', e)} />
            </label>
            <p class="hint">SVG (wird gesäubert: keine Skripte, keine externen Verweise) oder PNG, JPEG, WebP bis 1 MB. Wird sofort übernommen.</p>
          {/if}
        </div>
      </div>
    </section>
  </div>

  {#snippet actions()}
    <button class="btn" type="button" onclick={() => (open = false)}>Schließen</button>
    <button class="btn btn-primary" type="button" disabled={!changed || !mottoValid} onclick={save}>Profil speichern</button>
  {/snippet}
</Modal>

<style>
  .cols { display: flex; flex-wrap: wrap; gap: 24px; }
  .col { flex: 1 1 280px; display: flex; flex-direction: column; gap: 12px; }
  .col.wide { flex: 2 1 460px; }
  .avatar-row { display: flex; gap: 14px; align-items: center; }
  .avatar { width: 104px; height: 104px; border-radius: 50%; border: 3px solid var(--gold); box-shadow: 0 0 0 2px var(--ink); object-fit: cover; flex-shrink: 0; }
  .initial { display: flex; align-items: center; justify-content: center; background: #56733a; color: var(--parchment); font-family: var(--font-display); font-size: 52px; }
  .stack { display: flex; flex-direction: column; gap: 6px; }
  .file { display: inline-flex; align-items: center; position: relative; overflow: hidden; align-self: flex-start; }
  .file input { position: absolute; inset: 0; opacity: 0; cursor: pointer; }
  .file:focus-within { outline: 3px solid var(--gold); outline-offset: 2px; }
  .hint { color: var(--sepia); font-size: 14px; margin: 0; }
  .skin { display: flex; gap: 16px; flex-wrap: wrap; align-items: center; }
  .skin label { display: flex; align-items: center; gap: 6px; min-height: 40px; }
  .error { color: var(--warn); margin: 0; }
  .head { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 8px; }
  .tabs { display: flex; border: 1px solid var(--line-strong); border-radius: 4px; overflow: hidden; }
  .tabs button { min-height: 44px; padding: 0 14px; border: none; background: transparent; font-family: var(--font-label); font-size: 13px; font-weight: 600; cursor: pointer; }
  .tabs button.active { background: var(--ink); color: var(--parchment); }
  .arms { display: flex; gap: 20px; flex-wrap: wrap; }
  .preview { flex-shrink: 0; }
  .options { flex: 1 1 260px; display: flex; flex-direction: column; gap: 12px; }
  .varianten { display: flex; flex-wrap: wrap; gap: 6px; }
  .varianten button { padding: 4px; border: 1px solid var(--line); border-radius: 6px; background: #fbf6ea; cursor: pointer; }
  .varianten button.on { border: 2px solid var(--blood); background: #efe3c8; }
  .varianten button:focus-visible { outline: 3px solid var(--gold); outline-offset: 2px; }
</style>
