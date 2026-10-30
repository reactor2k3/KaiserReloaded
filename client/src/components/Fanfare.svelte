<script lang="ts">
  // Heroldsfanfare mit Banner für die Krönungsurkunde. Die Trompete hebt sich zum Stoß,
  // das Banner (Farben und Wappen des Kaisers, Schwalbenschwanz, Goldfransen) weht leicht.
  // side = 'links' zeigt mit dem Schalltrichter nach links oben, 'rechts' gespiegelt.
  import type { ArmsSpec } from '../lib/types';
  import { COUNTRY_COLORS } from '../lib/heraldry';
  import Wappen from './Wappen.svelte';

  let { side, arms, country }: { side: 'links' | 'rechts'; arms: ArmsSpec; country: number } = $props();

  const uid = $props.id();
  const farben = $derived(COUNTRY_COLORS[country] ?? COUNTRY_COLORS[5]);
</script>

<div class="fanfare {side}" aria-hidden="true">
  <div class="stoss">
    <svg viewBox="0 0 200 200">
      <defs>
        <linearGradient id="{uid}-g" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stop-color="#fbe7a1" /><stop offset=".5" stop-color="#e5b93f" /><stop offset="1" stop-color="#8f6a22" />
        </linearGradient>
      </defs>
      <!-- Banner: hängt am Rohr, unten Schwalbenschwanz; weht über .banner-Animation -->
      <g class="banner">
        <path d="M66 60H164V156L140 140L115 166L90 140L66 156Z" fill={farben.field1} stroke="#2b2118" stroke-width="1.6" stroke-linejoin="round" />
        <path d="M66 60H164V82H66Z" fill={farben.field2} stroke="#2b2118" stroke-width="1.2" />
        <path d="M70 154L90 138L115 162L140 138L160 154" fill="none" stroke="#e5b93f" stroke-width="3" stroke-dasharray="1.5 3" />
        <!-- Kordeln mit Quasten -->
        <path d="M68 60Q60 90 58 112M162 60Q170 90 172 112" fill="none" stroke="#e5b93f" stroke-width="1.6" />
        <path d="M54 112h8l-2 12h-4ZM168 112h8l-2 12h-4Z" fill="#e5b93f" stroke="#8f6a22" stroke-width=".8" />
      </g>
      <!-- Trompete: Mundstück rechts, Rohr mit Knauf, Schalltrichter links -->
      <path d="M40 54H188V62H40Z" fill="url(#{uid}-g)" stroke="#6b5023" stroke-width="1.2" />
      <rect x="120" y="50" width="10" height="16" rx="3" fill="url(#{uid}-g)" stroke="#6b5023" stroke-width="1" />
      <path d="M188 52H196V64H188Z" fill="url(#{uid}-g)" stroke="#6b5023" stroke-width="1" />
      <path d="M44 54Q30 50 4 34V82Q30 66 44 62Z" fill="url(#{uid}-g)" stroke="#6b5023" stroke-width="1.4" stroke-linejoin="round" />
      <ellipse cx="4" cy="58" rx="4" ry="24" fill="#8f6a22" stroke="#6b5023" stroke-width="1" />
      <!-- Schallstrahlen beim Stoß -->
      <g class="schall" stroke="#e5b93f" stroke-width="2.5" stroke-linecap="round">
        <path d="M-6 40L-20 30M-8 58H-24M-6 76L-20 86" />
      </g>
    </svg>
    <div class="wappen"><Wappen {arms} {country} size={34} label="" /></div>
  </div>
</div>

<style>
  .fanfare { width: 170px; height: 170px; flex-shrink: 0; overflow: visible; }
  .stoss { position: relative; width: 100%; height: 100%; transform-origin: 95% 30%; animation: heben 1s ease-out .3s both; }
  /* Linke Fanfare zeigt nach links oben, rechte gespiegelt nach rechts oben */
  .links .stoss { rotate: -18deg; }
  .rechts .stoss { rotate: 18deg; transform-origin: 5% 30%; }
  .rechts svg { transform: scaleX(-1); }
  svg { width: 100%; height: 100%; overflow: visible; filter: drop-shadow(0 4px 6px #2b211855); }
  /* Wappen auf der Bannermitte (gespiegelt positioniert, selbst nicht gespiegelt) */
  .wappen { position: absolute; top: 46%; left: 49%; transform: translate(-50%, -50%); }
  .rechts .wappen { left: 51%; }
  .banner { transform-box: fill-box; transform-origin: 50% 0; animation: wehen 3.2s ease-in-out infinite; }
  .schall { opacity: 0; animation: schall 1.6s ease-out 1.3s 3; }
  @keyframes heben { from { transform: translateY(30px) rotate(25deg); opacity: 0; } to { transform: none; opacity: 1; } }
  .rechts .stoss { animation-name: heben-r; }
  @keyframes heben-r { from { transform: translateY(30px) rotate(-25deg); opacity: 0; } to { transform: none; opacity: 1; } }
  @keyframes wehen { 0%, 100% { transform: skewY(0deg) scaleY(1); } 50% { transform: skewY(3deg) scaleY(.97); } }
  @keyframes schall { 0% { opacity: 0; transform: translateX(6px); } 30% { opacity: 1; } 100% { opacity: 0; transform: translateX(-10px); } }
  @media (prefers-reduced-motion: reduce) { .stoss, .banner, .schall { animation: none; } }
</style>
